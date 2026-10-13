package nix

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const (
	opengrepJob       = "opengrep-nix"
	packagingScript   = "nix/opengrep/test-packaging.sh"
	hashesOnlyCommand = packagingScript + " --hashes-only"
	hashesOnlyIf      = "${{ !cancelled() && matrix.os == 'ubuntu-latest' }}"
	// installNixScript installs the pinned, checksum-verified official Nix
	// release; the repository's Actions allowlist admits no Nix action.
	installNixScript = ".github/scripts/install-nix.sh"
)

// opengrepLegs is every runner the derivation must build on: one per
// meta.platforms system in nix/opengrep/default.nix.
var opengrepLegs = []string{"ubuntu-latest", "ubuntu-24.04-arm", "macos-latest", "macos-15-intel"}

// bypassRe matches an OpenGrep install that does not go through the
// derivation: a release download URL (any tag form, including
// releases/latest/download) or the env names the old curl job used to carry
// its own duplicated version and digest.
var bypassRe = regexp.MustCompile(`(?i)github\.com/opengrep/[^\s"']*/(releases|download)/|OPENGREP_(SHA256|VERSION)`)

type ciWorkflow struct {
	Jobs map[string]ciJob `yaml:"jobs"`
}

type ciJob struct {
	If              *string `yaml:"if"`
	ContinueOnError any     `yaml:"continue-on-error"`
	RunsOn          string  `yaml:"runs-on"`
	Strategy        struct {
		Matrix map[string]any `yaml:"matrix"`
	} `yaml:"strategy"`
	Steps []ciStep `yaml:"steps"`
}

type ciStep struct {
	If              *string `yaml:"if"`
	ContinueOnError any     `yaml:"continue-on-error"`
	Uses            string  `yaml:"uses"`
	Run             string  `yaml:"run"`
}

// checkOpengrepCI returns every way ci.yml fails to build and check the
// OpenGrep derivation on all platforms. An empty result means it is wired.
func checkOpengrepCI(src []byte) []string {
	var wf ciWorkflow
	if err := yaml.Unmarshal(src, &wf); err != nil {
		return []string{fmt.Sprintf("parsing workflow: %v", err)}
	}

	var problems []string
	if loc := bypassRe.Find(src); loc != nil {
		problems = append(problems, fmt.Sprintf(
			"contains %q: OpenGrep must come from nix/opengrep, not a separate pinned download", loc))
	}

	job, ok := wf.Jobs[opengrepJob]
	if !ok {
		return append(problems, fmt.Sprintf("has no %s job", opengrepJob))
	}
	if job.If != nil {
		problems = append(problems, fmt.Sprintf("job %s has `if: %s`; it must run on every push and PR", opengrepJob, *job.If))
	}
	if job.ContinueOnError != nil {
		problems = append(problems, fmt.Sprintf("job %s sets continue-on-error; a red leg must fail the run", opengrepJob))
	}
	problems = append(problems, checkMatrix(job)...)
	return append(problems, checkSteps(job.Steps)...)
}

func checkMatrix(job ciJob) []string {
	var problems []string
	if job.RunsOn != "${{ matrix.os }}" {
		problems = append(problems, fmt.Sprintf("job %s runs on %q, not ${{ matrix.os }}", opengrepJob, job.RunsOn))
	}
	for key := range job.Strategy.Matrix {
		if key != "os" {
			problems = append(problems, fmt.Sprintf("job %s matrix has %q; only os is expected (include/exclude can drop legs)", opengrepJob, key))
		}
	}
	legs, _ := job.Strategy.Matrix["os"].([]any)
	for _, want := range opengrepLegs {
		if !slices.Contains(legs, any(want)) {
			problems = append(problems, fmt.Sprintf("job %s matrix is missing the %s leg", opengrepJob, want))
		}
	}
	return problems
}

func checkSteps(steps []ciStep) []string {
	var problems []string
	installAt, buildAt, hashesAt := -1, -1, -1
	for i, s := range steps {
		if s.ContinueOnError != nil {
			problems = append(problems, fmt.Sprintf("step %d of %s sets continue-on-error", i, opengrepJob))
		}
		switch run := strings.TrimSpace(s.Run); {
		case run == installNixScript && s.If == nil:
			installAt = i
		case run == packagingScript && s.If == nil:
			buildAt = i
		case run == hashesOnlyCommand && s.If != nil && *s.If == hashesOnlyIf:
			hashesAt = i
		}
	}
	if installAt < 0 {
		problems = append(problems, fmt.Sprintf("job %s has no unconditional step running %s", opengrepJob, installNixScript))
	}
	if buildAt < 0 {
		problems = append(problems, fmt.Sprintf("job %s has no unconditional step running %s", opengrepJob, packagingScript))
	} else if buildAt < installAt {
		problems = append(problems, fmt.Sprintf("job %s runs %s before installing Nix", opengrepJob, packagingScript))
	}
	if hashesAt < 0 {
		problems = append(problems, fmt.Sprintf("job %s has no step running %q with `if: %s`", opengrepJob, hashesOnlyCommand, hashesOnlyIf))
	}
	return problems
}

func readCI(t *testing.T) string {
	t.Helper()
	path := filepath.Join("..", ".github", "workflows", "ci.yml")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return strings.ReplaceAll(string(b), "\r\n", "\n")
}

// TestOpengrepPackagingWiredInCI keeps the OpenGrep derivation built in CI
// (G-02). The derivation is embedded into user projects, so a broken unpack,
// patchelf or hash on any platform ships unnoticed unless CI builds it. A CI
// job does not guard itself: this test fails if the opengrep-nix job loses a
// platform leg, its script steps, or its Nix install; if it is made
// conditional or non-blocking; or if an install that bypasses the derivation
// comes back.
func TestOpengrepPackagingWiredInCI(t *testing.T) {
	t.Parallel()

	for _, p := range checkOpengrepCI([]byte(readCI(t))) {
		t.Errorf(".github/workflows/ci.yml: %s", p)
	}
}

// TestOpengrepPackagingWiredInCI_RejectsMutations proves the guard above is
// not vacuous: each edit that silently stops CI checking the derivation must
// be reported.
func TestOpengrepPackagingWiredInCI_RejectsMutations(t *testing.T) {
	t.Parallel()

	ci := readCI(t)
	const jobHeader = "  " + opengrepJob + ":\n"
	const buildRun = "        run: " + packagingScript + "\n"

	tests := []struct {
		name, old, new string
	}{
		{"build step commented out", buildRun, "        # run: " + packagingScript + "\n"},
		{"hashes step commented out", "        run: " + hashesOnlyCommand, "        # run: " + hashesOnlyCommand},
		{"hashes step only on success", hashesOnlyIf, "matrix.os == 'ubuntu-latest'"},
		{"build step made conditional", buildRun, buildRun + "        if: false\n"},
		{"job disabled", jobHeader, jobHeader + "    if: false\n"},
		{"job non-blocking", jobHeader, jobHeader + "    continue-on-error: true\n"},
		{"step non-blocking", buildRun, buildRun + "        continue-on-error: true\n"},
		{"job renamed", jobHeader, "  opengrep:\n"},
		{"macOS legs dropped", "          - macos-latest\n          - macos-15-intel\n", ""},
		{"arm leg dropped", "          - ubuntu-24.04-arm\n", ""},
		{"leg excluded", "          - macos-15-intel\n", "          - macos-15-intel\n        exclude:\n          - os: macos-15-intel\n"},
		{"fixed runner", "    runs-on: ${{ matrix.os }}\n", "    runs-on: ubuntu-latest\n"},
		{"nix install removed", "        run: " + installNixScript + "\n", "        run: echo skipped\n"},
		{"nix install made conditional", "        run: " + installNixScript + "\n", "        run: " + installNixScript + "\n        if: false\n"},
		{"curl of latest release", buildRun, buildRun + "      - run: curl -fsSLO https://github.com/opengrep/opengrep/releases/latest/download/opengrep_manylinux_x86\n"},
		{"duplicated version env", buildRun, buildRun + "        env:\n          OPENGREP_VERSION: 1.0.0\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Mutate only from the job header on, so a string shared with
			// an earlier job (runs-on, a pinned action) hits this job.
			at := strings.Index(ci, jobHeader)
			if at < 0 {
				t.Fatalf("ci.yml has no %q", jobHeader)
			}
			tail := strings.Replace(ci[at:], tt.old, tt.new, 1)
			if tail == ci[at:] {
				t.Fatalf("mutation did not apply: %q not found in the %s job", tt.old, opengrepJob)
			}
			mutated := ci[:at] + tail
			if problems := checkOpengrepCI([]byte(mutated)); len(problems) == 0 {
				t.Errorf("mutation %q passed the guard", tt.name)
			}
		})
	}
}
