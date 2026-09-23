package claudecode_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
)

// pgJSDriver imports the package-guard hook template as a module with all
// network access replaced by a failing stub, then runs one of:
//   - detect: detect_install_commands, printing each (manager, packages)
//   - flags:  apply_safety_flags for PG_MGR, printing the rewritten command
//   - main:   the full hook on a Bash PreToolUse envelope, printing its exit
//     code and stdout
//
// Any code path that reached the network would fail closed (deny), so a test
// expecting a different decision also proves no registry was contacted.
const pgJSDriver = `
import importlib.util, io, json, os, sys, urllib.request
def _no_network(*args, **kwargs):
    raise OSError("network disabled in package-guard tests")
urllib.request.urlopen = _no_network
spec = importlib.util.spec_from_file_location('pg', os.environ['PG_PATH'])
m = importlib.util.module_from_spec(spec)
spec.loader.exec_module(m)
cmd = os.environ['PG_CMD']
mode = os.environ['PG_MODE']
if mode == 'detect':
    print(json.dumps([{'manager': mgr, 'packages': ps} for _, mgr, _, ps in m.detect_install_commands(cmd)]))
elif mode == 'flags':
    print(json.dumps({'rewritten': m.apply_safety_flags(cmd, os.environ['PG_MGR'])}))
else:
    sys.stdin = io.StringIO(json.dumps({'tool_name': 'Bash', 'tool_input': {'command': cmd}}))
    out = io.StringIO()
    real_stdout, sys.stdout = sys.stdout, out
    code = 0
    try:
        m.main()
    except SystemExit as e:
        code = e.code
    sys.stdout = real_stdout
    print(json.dumps({'code': code, 'out': out.getvalue()}))
`

// runPGJSDriver runs pgJSDriver and decodes its JSON output into v.
func runPGJSDriver(t *testing.T, mode, command, manager string, v any) {
	t.Helper()
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not available; skipping package-guard hook test")
	}
	template, err := filepath.Abs(filepath.Join("templates", "hooks", "package-guard.py"))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(python, "-c", pgJSDriver)
	cmd.Env = append(os.Environ(),
		"PYTHONDONTWRITEBYTECODE=1",
		"PG_PATH="+template,
		"PG_MODE="+mode,
		"PG_CMD="+command,
		"PG_MGR="+manager,
		"CLAUDE_PROJECT_DIR="+t.TempDir(),
		"CLAUDE_AUDIT_DIR="+t.TempDir(),
	)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("driver failed: %v\n%s", err, out)
	}
	if err := json.Unmarshal(out, v); err != nil {
		t.Fatalf("bad driver output %q: %v", out, err)
	}
}

// TestPackageGuard_BunIgnoresScripts verifies W059: Bun runs lifecycle scripts
// for ~370 default-trusted packages, so the guard must add --ignore-scripts to
// bun installs like it does for npm.
func TestPackageGuard_BunIgnoresScripts(t *testing.T) {
	t.Parallel()
	tests := []struct {
		command string
		want    string
	}{
		{"bun add left-pad", "bun add left-pad --ignore-scripts"},
		{"bun install", "bun install --ignore-scripts"},
		{"cd web && bun i", "cd web && bun i --ignore-scripts"},
	}
	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			t.Parallel()
			var res struct {
				Rewritten *string `json:"rewritten"`
			}
			runPGJSDriver(t, "flags", tt.command, "bun", &res)
			if res.Rewritten == nil || *res.Rewritten != tt.want {
				t.Errorf("apply_safety_flags(%q) = %v, want %q", tt.command, res.Rewritten, tt.want)
			}
		})
	}
}

// TestPackageGuard_DenoRegistryPackages verifies W065: deno add/install and
// deno run/x fetch npm and JSR packages exactly like npm install / npx, so the
// guard must extract them, while running a local script is not an install.
func TestPackageGuard_DenoRegistryPackages(t *testing.T) {
	t.Parallel()
	tests := []struct {
		command      string
		wantDetected bool
		wantPackages []string
	}{
		{"deno add npm:left-pad@1.3.0", true, []string{"left-pad@1.3.0"}},
		{"deno add npm:@scope/pkg@2 jsr:@std/path", true, []string{"@scope/pkg@2", "jsr:@std/path"}},
		{"deno install npm:chalk", true, []string{"chalk"}},
		{"deno install", true, []string{}},
		{"deno run -A npm:evil-cli@latest --flag", true, []string{"evil-cli@latest"}},
		{"deno run --config deno.json npm:cowsay/cowthink", true, []string{"cowsay"}},
		{"deno x evil-cli", true, []string{"evil-cli"}},
		{"sudo deno x npm:@evil/cli@1/bin", true, []string{"@evil/cli@1"}},
		{"bash -c 'deno run npm:evil'", true, []string{"evil"}},
		// Deno >= 2.8 treats unprefixed add/install operands as npm packages.
		{"deno add express", true, []string{"express"}},
		{"deno add -D @types/node chart.js", true, []string{"@types/node", "chart.js"}},
		{"deno install -g -N -R -n serve jsr:@std/http/file-server", true, []string{"jsr:@std/http/file-server"}},
		{"deno install -g --root /usr/local/bin greet.ts", true, []string{}},
		{"deno install --entrypoint main.ts", true, []string{}},
		// `deno <module>` is an implicit `deno run`.
		{"deno npm:evil-cli", true, []string{"evil-cli"}},
		{"deno -A npm:evil-cli", true, []string{"evil-cli"}},
		{"deno serve jsr:@std/http/file-server", true, []string{"jsr:@std/http/file-server"}},
		{"deno create npm:vite my-app", true, []string{"vite"}},
		{"deno create --npm create-vite my-app", true, []string{"create-vite"}},
		{"deno x @angular/cli new app", true, []string{"@angular/cli"}},
		// A value flag or `--` must not hide the executed package.
		{"deno run --config npm:evil", true, []string{"evil"}},
		{"deno run -- npm:evil", true, []string{"evil"}},
		{"deno run -A main.ts", false, nil},
		{"deno run -A main.ts npm:not-executed", false, nil},
		{"deno -A main.ts", false, nil},
		{"deno main.ts", false, nil},
		{"deno x ./scripts/tool.ts", false, nil},
		{"deno task build", false, nil},
		{"deno remove npm:left-pad", false, nil},
		{"deno run --allow-net https://example.com/x.ts", false, nil},
		{"deno test", false, nil},
		{"deno fmt", false, nil},
	}
	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			t.Parallel()
			var dets []struct {
				Manager  string   `json:"manager"`
				Packages []string `json:"packages"`
			}
			runPGJSDriver(t, "detect", tt.command, "", &dets)
			if got := len(dets) > 0; got != tt.wantDetected {
				t.Fatalf("detected = %v, want %v (%+v)", got, tt.wantDetected, dets)
			}
			if !tt.wantDetected {
				return
			}
			if dets[0].Manager != "deno" {
				t.Errorf("manager = %q, want deno", dets[0].Manager)
			}
			if !slices.Equal(dets[0].Packages, tt.wantPackages) {
				t.Errorf("packages = %v, want %v", dets[0].Packages, tt.wantPackages)
			}
		})
	}
}

// TestPackageGuard_JSRPackagesAsk verifies W065: JSR packages have no OSV feed
// or age check in the guard, so they are escalated to the user rather than
// silently allowed (or denied by a bogus npm-registry lookup).
func TestPackageGuard_JSRPackagesAsk(t *testing.T) {
	t.Parallel()
	var res struct {
		Code int    `json:"code"`
		Out  string `json:"out"`
	}
	runPGJSDriver(t, "main", "deno add jsr:@std/path", "", &res)
	if res.Code != 0 {
		t.Fatalf("exit code = %d, want 0 with a JSON decision; out: %s", res.Code, res.Out)
	}
	var decision struct {
		HookSpecificOutput struct {
			PermissionDecision string `json:"permissionDecision"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal([]byte(res.Out), &decision); err != nil {
		t.Fatalf("hook output is not a JSON decision: %q: %v", res.Out, err)
	}
	if got := decision.HookSpecificOutput.PermissionDecision; got != "ask" {
		t.Errorf("permissionDecision = %q, want ask; out: %s", got, res.Out)
	}
}

// TestPackageGuard_DenoPackagesAreValidated verifies W065 end to end: npm
// packages fetched by deno (prefixed or, since Deno 2.8, bare) go through the
// registry validation. With the network stubbed out that validation fails
// closed, so a deny proves the package was checked rather than waved through.
func TestPackageGuard_DenoPackagesAreValidated(t *testing.T) {
	t.Parallel()
	for _, command := range []string{"deno add express", "deno -A npm:evil-cli", "deno x @evil/cli"} {
		t.Run(command, func(t *testing.T) {
			t.Parallel()
			var res struct {
				Code int    `json:"code"`
				Out  string `json:"out"`
			}
			runPGJSDriver(t, "main", command, "", &res)
			var decision struct {
				HookSpecificOutput struct {
					PermissionDecision string `json:"permissionDecision"`
				} `json:"hookSpecificOutput"`
			}
			if err := json.Unmarshal([]byte(res.Out), &decision); err != nil {
				t.Fatalf("hook output is not a JSON decision (code %d): %q: %v", res.Code, res.Out, err)
			}
			if got := decision.HookSpecificOutput.PermissionDecision; got != "deny" {
				t.Errorf("permissionDecision = %q, want deny (validation failed closed); out: %s", got, res.Out)
			}
		})
	}
}
