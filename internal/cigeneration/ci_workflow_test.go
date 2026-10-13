package cigeneration

import (
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/testutil"
)

// The CI workflow jobs TestCIRunsToolSuites inspects.
const (
	ciWorkflowFile = "ci.yml"
	nixSandboxJob  = "nix-sandbox"
	skipReportCmd  = "go run ./internal/testutil/cmd/skipreport"
	// installNixScript installs the checksum-pinned official Nix release;
	// the repository's Actions allowlist admits no Nix action.
	installNixScript = ".github/scripts/install-nix.sh"
)

// nixSandboxSwitches are the switches the nix-sandbox job provisions the
// tools for. RequireRuleScanner is the opengrep-nix job's.
var nixSandboxSwitches = []testutil.Switch{
	testutil.RequireNix, testutil.RequireE3, testutil.CheckNixAttrs, testutil.RequireSecTools,
}

// flakeSandboxLdflagRe matches a sandbox-path -X ldflag in flake.nix and
// captures the variable and the path after the store-path interpolation.
var flakeSandboxLdflagRe = regexp.MustCompile(`"-X" "([^"=]+/internal/sandbox\.[A-Za-z]+)=\$\{[^}]+\}([^"]*)"`)

// flakeSandboxLdflags returns, from flake.nix, the sandbox variables the
// qsdev package injects and the file each one names inside its store path.
func flakeSandboxLdflags(t *testing.T) map[string]string {
	t.Helper()

	b, err := os.ReadFile(filepath.Join(repoRoot, "flake.nix"))
	if err != nil {
		t.Fatalf("reading flake.nix: %v", err)
	}
	out := map[string]string{}
	for _, m := range flakeSandboxLdflagRe.FindAllStringSubmatch(string(b), -1) {
		out[m[1]] = m[2]
	}
	if len(out) == 0 {
		t.Fatal("flake.nix injects no internal/sandbox path via -X; the nix-sandbox job would test nothing the package ships")
	}
	return out
}

// stepEnv is the environment a step runs with: the job's, overridden by its own.
func stepEnv(job workflowJob, s workflowStep) map[string]string {
	env := maps.Clone(job.Env)
	if env == nil {
		env = map[string]string{}
	}
	maps.Copy(env, s.Env)
	return env
}

// findStep returns the first step of job whose run script contains every
// one of subs, failing the test when there is none.
func findStep(t *testing.T, id string, job workflowJob, what string, subs ...string) workflowStep {
	t.Helper()
	i := stepIndex(job, subs...)
	if i < 0 {
		t.Fatalf("%s job %s has no step that %s (a run containing %q)", ciWorkflowFile, id, what, subs)
	}
	return job.Steps[i]
}

// TestCIRunsToolSuites is the U26-03 regression test: the suites that guard
// generated output must run somewhere with their tools, and a missing tool
// must fail there rather than skip. The nix-sandbox job builds the flake's
// packages, provisions nix, bubblewrap, ll-restrict and the secret scanners,
// sets every switch governing them, runs the whole suite with the
// ldflags-injected sandbox paths (or the Landlock E3 test cannot run), and
// rejects any skip naming a governed tool. The unit-test leg shuffles test
// order, and pin-audit verifies the MCP packages the catalog launches.
func TestCIRunsToolSuites(t *testing.T) {
	t.Parallel()

	wf, ok := readRepoWorkflows(t)[ciWorkflowFile]
	if !ok {
		t.Fatalf(".github/workflows/%s is missing", ciWorkflowFile)
	}

	t.Run("nix-sandbox", func(t *testing.T) {
		t.Parallel()
		job, ok := wf.Jobs[nixSandboxJob]
		if !ok {
			t.Fatalf("%s has no %s job; the nix and sandbox suites only ever skip", ciWorkflowFile, nixSandboxJob)
		}
		if !slices.Contains(readRequiredChecks(t), nixSandboxJob) {
			t.Errorf("%s does not list %s; a red run would not block merges or releases", requiredChecksPath, nixSandboxJob)
		}
		if job.If != "" || mayContinueOnError(job.ContinueOnError) {
			t.Errorf("%s job has if: %q, continue-on-error: %q; a required check must always run and fail", nixSandboxJob, job.If, job.ContinueOnError)
		}
		install := findStep(t, nixSandboxJob, job, "installs the pinned Nix release", installNixScript)
		if install.If != "" {
			t.Errorf("%s Nix install step has if: %q; it must run whenever the job does", nixSandboxJob, install.If)
		}
		findStep(t, nixSandboxJob, job, "lifts the AppArmor user-namespace restriction",
			"sysctl", "kernel.apparmor_restrict_unprivileged_userns=0")
		findStep(t, nixSandboxJob, job, "builds the flake's packages", "nix build", ".#qsdev", ".#ll-restrict", ".#seccomp-filter")
		if stepIndex(job, installNixScript) > stepIndex(job, "nix build", ".#qsdev") {
			t.Errorf("%s builds the flake before installing Nix", nixSandboxJob)
		}
		findStep(t, nixSandboxJob, job, "checks the flake", "nix flake check")
		findStep(t, nixSandboxJob, job, "pins the nixpkgs registry entry to flake.lock",
			"nix registry add nixpkgs", "flake.lock")

		test := findStep(t, nixSandboxJob, job, "runs the whole suite", "--jsonfile", "./...")
		for sym, file := range flakeSandboxLdflags(t) {
			if !containsAll(test.Run, "-X "+sym+"=", file) {
				t.Errorf("%s test step does not inject %s=<store path>%s as flake.nix does", nixSandboxJob, sym, file)
			}
		}
		// The E3 hooks run cat and sh inside bwrap, which mounts only
		// /nix/store, and nixpkgs bubblewrap provides bwrap itself.
		for _, pkg := range []string{"nix shell", "nixpkgs#bubblewrap", "nixpkgs#coreutils", "nixpkgs#bash"} {
			if !strings.Contains(test.Run, pkg) {
				t.Errorf("%s test step does not provision %s", nixSandboxJob, pkg)
			}
		}
		if test.If != "" {
			t.Errorf("%s test step has if: %q; it must run whenever the job does", nixSandboxJob, test.If)
		}
		report := findStep(t, nixSandboxJob, job, "rejects skips of governed tools", skipReportCmd)
		if !runsAfterFailure(report.If) {
			t.Errorf("%s report step has if: %q; it must run after a failed test step (${{ !cancelled() }} or always())", nixSandboxJob, report.If)
		}
		for _, s := range []workflowStep{test, report} {
			if mayContinueOnError(s.ContinueOnError) {
				t.Errorf("%s step %q has continue-on-error: %q; its failure would not fail the job", nixSandboxJob, s.Name, s.ContinueOnError)
			}
			env := stepEnv(job, s)
			// A cgo test binary links the host loader under /lib64, which the
			// bwrap sandbox does not mount, so the self-invocation E3 test
			// would only skip.
			if env["CGO_ENABLED"] != "0" {
				t.Errorf("%s step %q runs with CGO_ENABLED=%q, want \"0\" as flake.nix builds qsdev", nixSandboxJob, s.Name, env["CGO_ENABLED"])
			}
			for _, sw := range nixSandboxSwitches {
				if env[string(sw)] != "1" {
					t.Errorf("%s step %q runs with %s=%q, want \"1\"", nixSandboxJob, s.Name, sw, env[string(sw)])
				}
			}
		}
	})

	t.Run("test shuffles", func(t *testing.T) {
		t.Parallel()
		step := findStep(t, "test", wf.Jobs["test"], "runs the suite", "gotestsum", "./...")
		if !strings.Contains(step.Run, "-shuffle=on") {
			t.Errorf("test step %q does not pass -shuffle=on; order-dependent tests pass by luck", step.Run)
		}
	})

	t.Run("pin-audit verifies MCP packages", func(t *testing.T) {
		t.Parallel()
		job := wf.Jobs["pin-audit"]
		step := findStep(t, "pin-audit", job, "verifies the MCP server packages", "TestMCPServerPackagesResolve", "./internal/catalog/")
		if stepEnv(job, step)["VERIFY_MCP_PACKAGES"] != "1" {
			t.Errorf("pin-audit step %q does not set VERIFY_MCP_PACKAGES=1, so the test only skips", step.Name)
		}
	})
}

// mayContinueOnError reports whether a continue-on-error value can let a
// failure pass: anything but absent or literal false.
func mayContinueOnError(v string) bool {
	return v != "" && v != "false"
}

// runsAfterFailure reports whether a step condition runs the step after an
// earlier step failed.
func runsAfterFailure(cond string) bool {
	c := strings.TrimSpace(cond)
	if inner, ok := strings.CutPrefix(c, "${{"); ok {
		c = strings.TrimSpace(strings.TrimSuffix(inner, "}}"))
	}
	return c == "!cancelled()" || c == "always()"
}
