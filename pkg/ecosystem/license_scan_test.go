package ecosystem

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// TestAggregateTaskDefinitions_SecurityScanLicense is the F218 regression:
// enabling license-compliance must run a ScanCode scan with the generated
// policy in the security-scan task, not just write a policy file nothing reads.
func TestAggregateTaskDefinitions_SecurityScanLicense(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		enabled map[string]bool
		want    []string
	}{
		{name: "license-compliance only", enabled: map[string]bool{"license-compliance": true}, want: []string{licenseScanCommand()}},
		{
			name:    "with gitleaks",
			enabled: map[string]bool{"license-compliance": true, "gitleaks": true},
			want:    []string{"gitleaks detect --no-banner", licenseScanCommand()},
		},
		{name: "disabled", enabled: map[string]bool{"license-compliance": false}, want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			secTask := findTask(AggregateTaskDefinitions(nil, staticConfig, tt.enabled), SecurityScanTask)
			var got []string
			if secTask != nil {
				got = secTask.Commands
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("security-scan commands = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestLicenseScanCommand_Args checks the ScanCode invocation: license
// detection with the generated policy, JSON on stdout, the policy files and
// build output ignored, and the dependency directories scanned.
func TestLicenseScanCommand_Args(t *testing.T) {
	t.Parallel()
	cmd := licenseScanCommand()

	for _, want := range []string{
		"scancode ",
		" --license ",
		" --license-policy " + LicensePolicyPath + " ",
		" --ignore '" + LicensePolicyPath + "'",
		" --ignore '" + LicenseExceptionsPath + "'",
		" --ignore '.devenv'",
		" --ignore '*.egg-info'",
		" --json - . | jq -r '",
	} {
		if !strings.Contains(cmd, want) {
			t.Errorf("license scan command missing %q:\n%s", want, cmd)
		}
	}
	for _, dep := range []string{"node_modules", "vendor", "third_party", ".venv", "venv"} {
		if strings.Contains(cmd, "--ignore '"+dep+"'") {
			t.Errorf("license scan must scan dependency directory %q, but ignores it", dep)
		}
	}
	if strings.Count(licensePolicyGate, "'") != 0 {
		t.Error("licensePolicyGate must not contain single quotes: it is single-quoted in the task script")
	}
	if strings.Contains(cmd, "${") {
		t.Error("license scan command must not contain ${: it is embedded in a Nix string")
	}
}

// TestLicenseScanCommand_PolicyGate runs the task command under bash (as the
// devenv task script runs it, with errexit, nounset and pipefail) against a
// stub scancode that prints a canned scan, and checks that the real jq gate
// fails the task only on files whose licenses the policy prohibits.
func TestLicenseScanCommand_PolicyGate(t *testing.T) {
	t.Parallel()
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not available")
	}
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq not available")
	}

	const (
		mit      = `{"license_key":"mit","spdx_license_key":"MIT","label":"Approved License","compliance_alert":""}`
		mpl      = `{"license_key":"mpl-2.0","spdx_license_key":"MPL-2.0","label":"Restricted License","compliance_alert":"warning"}`
		agplPlus = `{"license_key":"agpl-3.0-plus","spdx_license_key":"AGPL-3.0-or-later","label":"Prohibited License","compliance_alert":"error"}` // pragma: allowlist secret
	)
	scan := func(files ...string) string {
		return `{"headers":[],"files":[` + strings.Join(files, ",") + `]}`
	}
	file := func(path, policy string) string {
		return `{"path":"` + path + `","type":"file","license_policy":` + policy + `}`
	}

	tests := []struct {
		name       string
		scan       string
		scanExit   int
		wantErr    bool
		wantOutput []string
	}{
		{
			name: "only approved licenses",
			scan: scan(file("LICENSE", "["+mit+"]"), file("main.go", "[]"), `{"path":"src","type":"directory"}`),
		},
		{
			name:       "restricted license is reported but passes",
			scan:       scan(file("vendor/x/LICENSE", "["+mpl+"]")),
			wantOutput: []string{"license needs review (Restricted License): vendor/x/LICENSE: MPL-2.0"},
		},
		{
			name:       "or-later prohibited license fails",
			scan:       scan(file("LICENSE", "["+mit+"]"), file("node_modules/y/package.json", "["+mit+","+agplPlus+"]")),
			wantErr:    true,
			wantOutput: []string{"prohibited license (Prohibited License): node_modules/y/package.json: AGPL-3.0-or-later"},
		},
		{
			name:       "legacy object-shaped policy fails",
			scan:       scan(file("COPYING", agplPlus)),
			wantErr:    true,
			wantOutput: []string{"prohibited license (Prohibited License): COPYING: AGPL-3.0-or-later"},
		},
		{
			name: "no files",
			scan: `{"headers":[],"files":[]}`,
		},
		{
			// ScanCode rejects a policy with a duplicate license_key by
			// recording a header error, applying no policy and exiting 0.
			name:       "policy error in scan headers fails",
			scan:       `{"headers":[{"errors":["ERROR: License Policy file contains duplicate entries"]}],"files":[` + file("COPYING", "[]") + `]}`,
			wantErr:    true,
			wantOutput: []string{"scancode reported errors:", "duplicate entries"},
		},
		{
			name: "headers without errors pass",
			scan: `{"headers":[{"errors":[]}],"files":[` + file("LICENSE", "["+mit+"]") + `]}`,
		},
		{
			name:     "scancode failure fails the task",
			scan:     scan(file("LICENSE", "["+mit+"]")),
			scanExit: 1,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			project := t.TempDir()
			// A file matching an ignore glob must not be expanded by the shell.
			if err := os.Mkdir(filepath.Join(project, "pkg.egg-info"), 0o755); err != nil {
				t.Fatal(err)
			}
			work := t.TempDir()
			scanFile := filepath.Join(work, "scan.json")
			if err := os.WriteFile(scanFile, []byte(tt.scan), 0o644); err != nil {
				t.Fatal(err)
			}
			argsFile := filepath.Join(work, "args")
			bin := t.TempDir()
			stub := "#!" + bash + "\nprintf '%s\\n' \"$@\" > \"$STUB_ARGS\"\ncat \"$STUB_SCAN\"\nexit \"$STUB_EXIT\"\n"
			if err := os.WriteFile(filepath.Join(bin, "scancode"), []byte(stub), 0o755); err != nil {
				t.Fatal(err)
			}

			cmd := licenseScanCommand()
			run := exec.Command(bash, "-c", "set -euo pipefail\n"+cmd)
			run.Dir = project
			run.Env = append(os.Environ(),
				"PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"),
				"STUB_ARGS="+argsFile,
				"STUB_SCAN="+scanFile,
				"STUB_EXIT="+strconv.Itoa(tt.scanExit),
			)
			out, err := run.CombinedOutput()
			if (err != nil) != tt.wantErr {
				t.Fatalf("running license scan: err = %v, wantErr %v\n%s", err, tt.wantErr, out)
			}
			for _, want := range tt.wantOutput {
				if !strings.Contains(string(out), want) {
					t.Errorf("output missing %q:\n%s", want, out)
				}
			}
			if !tt.wantErr && strings.Contains(string(out), "prohibited license") {
				t.Errorf("passing scan reported a prohibited license:\n%s", out)
			}

			args, err := os.ReadFile(argsFile)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(args), "\n*.egg-info\n") {
				t.Errorf("scancode did not receive the literal *.egg-info ignore pattern:\n%s", args)
			}
			if !strings.Contains(string(args), "--license-policy\n"+LicensePolicyPath+"\n") {
				t.Errorf("scancode did not receive --license-policy %s:\n%s", LicensePolicyPath, args)
			}
		})
	}
}
