package doctor

import (
	"context"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/sysinfo"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestRunChecksReturns20Results(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	osInfo := &sysinfo.OSInfo{
		OS:             "linux",
		Arch:           "amd64",
		Family:         "linux",
		PackageManager: "nix",
		HasNix:         true,
	}

	results := RunChecks(ctx, osInfo, DefaultChecks())
	if len(results) != 20 {
		t.Errorf("RunChecks returned %d results, want 20", len(results))
	}
}

func TestRunChecksPopulatesToolNames(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	osInfo := &sysinfo.OSInfo{
		OS:     "linux",
		Arch:   "amd64",
		Family: "linux",
	}

	results := RunChecks(ctx, osInfo, DefaultChecks())
	names := make(map[string]bool)
	for _, r := range results {
		if r.Name == "" {
			t.Error("got result with empty Name")
		}
		names[r.Name] = true
	}

	// Verify a few expected names are present
	for _, expected := range []string{"git", "go", "node", "npm", "nix"} {
		if !names[expected] {
			t.Errorf("expected tool %q not in results", expected)
		}
	}
}

func TestRunSingleCheckNotFound(t *testing.T) {
	ctx := context.Background()
	tc := ToolCheck{
		Name:        "nonexistent-tool-xyz",
		Binary:      "nonexistent-tool-xyz-12345",
		VersionFlag: "--version",
		Required:    true,
		ParseVersion: func(raw string) string {
			return raw
		},
		AutoInstall: func(_ *sysinfo.OSInfo) bool { return true },
	}

	osInfo := &sysinfo.OSInfo{OS: "linux", Family: "linux"}
	status := runSingleCheck(ctx, tc, osInfo)

	if status.Installed {
		t.Error("expected nonexistent tool to not be installed")
	}
	if !status.AutoInstallable {
		t.Error("expected AutoInstallable to be true")
	}
	if status.Name != "nonexistent-tool-xyz" {
		t.Errorf("Name = %q, want %q", status.Name, "nonexistent-tool-xyz")
	}
}

// TestRequireBinaries: a program the project needs makes its registry check
// required, keeping the check's floor and version probe; one the registry
// does not know becomes a required lookup-only check.
func TestRequireBinaries(t *testing.T) {
	t.Parallel()
	const why = "needed by hooks"
	checks := RequireBinaries(DefaultChecks(), []string{"python3", "prek", "git", "qsdev-hook-tool"}, why)
	byName := map[string]ToolCheck{}
	for _, c := range checks {
		byName[c.Name] = c
	}
	// prek and qsdev-hook-tool each add a check.
	if got, want := len(checks), len(DefaultChecks())+2; got != want {
		t.Fatalf("len(checks) = %d, want %d", got, want)
	}

	py := byName["python3"]
	if !py.Required || py.MinVersion != types.MinHookPython || py.VersionFlag == "" || py.RequiredBy != why {
		t.Errorf("python3 = Required %v, MinVersion %q, VersionFlag %q, RequiredBy %q; want required with floor %q, its probe and the reason",
			py.Required, py.MinVersion, py.VersionFlag, py.RequiredBy, types.MinHookPython)
	}
	// A program a check names only as an alternative gets its own copy of
	// that check: finding pre-commit would not let a hook running prek run.
	if pc := byName["pre-commit"]; pc.Required {
		t.Errorf("pre-commit = Required %v; only its alternative prek is needed", pc.Required)
	}
	if prek := byName["prek"]; !prek.Required || prek.RequiredBy != why || prek.Binary != "prek" || len(prek.AltBinaries) != 0 ||
		prek.VersionFlag != byName["pre-commit"].VersionFlag || prek.ParseVersion == nil {
		t.Errorf("prek = %+v; want a required copy of the pre-commit check for prek alone", prek)
	}
	// A tool that is already a prerequisite stays one.
	if git := byName["git"]; !git.Required || git.RequiredBy != "" {
		t.Errorf("git = Required %v, RequiredBy %q; want a required prerequisite with no hook reason", git.Required, git.RequiredBy)
	}
	unknown, ok := byName["qsdev-hook-tool"]
	if !ok || !unknown.Required || unknown.Binary != "qsdev-hook-tool" || unknown.VersionFlag != "" || unknown.MinVersion != "" || unknown.RequiredBy != why {
		t.Errorf("unknown program check = %+v (present %v); want a required lookup-only check", unknown, ok)
	}
	for _, c := range DefaultChecks() {
		if c.Name == "python3" && c.Required {
			t.Error("RequireBinaries modified the DefaultChecks it was given")
		}
	}
}

// TestRequireBinaries_PrimaryDropsAlternatives: a program that is a check's
// own Binary is required alone, so an alternative on PATH cannot satisfy it.
func TestRequireBinaries_PrimaryDropsAlternatives(t *testing.T) {
	t.Parallel()
	checks := RequireBinaries(DefaultChecks(), []string{"pre-commit"}, "r")
	if len(checks) != len(DefaultChecks()) {
		t.Fatalf("len(checks) = %d, want %d", len(checks), len(DefaultChecks()))
	}
	for _, c := range checks {
		if c.Name == "pre-commit" && (!c.Required || len(c.AltBinaries) != 0) {
			t.Errorf("pre-commit = Required %v, AltBinaries %q; want required with no alternatives", c.Required, c.AltBinaries)
		}
	}
}

// TestProjectChecks_OutsideProject: outside a project the checks are the
// registry alone.
func TestProjectChecks_OutsideProject(t *testing.T) {
	t.Parallel()
	checks, err := ProjectChecks("")
	if err != nil || len(checks) != len(DefaultChecks()) {
		t.Errorf("ProjectChecks(\"\") = %d checks, %v; want the default registry", len(checks), err)
	}
}

// TestRunSingleCheck_UpgradeHint: a tool installed below its floor whose
// check has an UpgradeHint is not auto-installable (setup would reinstall,
// not upgrade) and carries the hint; a missing one still is installable.
func TestRunSingleCheck_UpgradeHint(t *testing.T) {
	t.Parallel()
	tc := ToolCheck{Name: "x", Binary: "nonexistent-tool-xyz-12345", MinVersion: "2", UpgradeHint: "upgrade x",
		AutoInstall: func(*sysinfo.OSInfo) bool { return true }}
	if st := runSingleCheck(context.Background(), tc, &sysinfo.OSInfo{}); !st.AutoInstallable || st.UpgradeHint != "" {
		t.Errorf("missing tool = %+v; want auto-installable with no upgrade hint", st)
	}
}

func TestRunSingleCheck_LookupOnlyMissing(t *testing.T) {
	t.Parallel()
	status := runSingleCheck(context.Background(), ToolCheck{Name: "x", Binary: "nonexistent-tool-xyz-12345", Required: true, RequiredBy: "r"}, &sysinfo.OSInfo{})
	if status.Installed || !status.Required || status.RequiredBy != "r" {
		t.Errorf("status = %+v, want a required missing tool carrying its reason", status)
	}
}

// TestRequireBinaries_NameSpellings: a hook program spelled as another
// version of a versioned binary (python, python3.11) or, on Windows, in
// another case or with a PATHEXT extension keeps the python3 floor and
// probe instead of becoming a lookup-only check without one.
func TestRequireBinaries_NameSpellings(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, program, goos, pathExt string
		// wantBinary is the binary of the required check with the python3
		// floor; "" wants a lookup-only check with no floor.
		wantBinary string
	}{
		{"exact", "python3", "linux", "", "python3"},
		{"unversioned", "python", "linux", "", "python"},
		{"minor version", "python3.11", "linux", "", "python3.11"},
		{"old minor version", "python3.8", "linux", "", "python3.8"},
		{"case differs on linux", "Python3", "linux", "", ""},
		{"exe on linux", "python3.exe", "linux", "", ""},
		{"other stem", "pythonista", "linux", "", ""},
		{"stem prefix only", "pyth", "linux", "", ""},
		{"case on windows", "Python3", "windows", "", "python3"},
		{"exe on windows", "python3.exe", "windows", ".COM;.EXE;.BAT;.CMD", "python3"},
		{"versioned exe on windows", "Python3.11.EXE", "windows", "", "Python3.11.EXE"},
		{"extension not in PATHEXT", "python3.exe", "windows", ".BAT", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			checks := requireBinaries(DefaultChecks(), []string{tt.program}, "r", tt.goos, tt.pathExt)
			var floored, lookupOnly []ToolCheck
			for _, c := range checks {
				if !c.Required || c.RequiredBy != "r" {
					continue
				}
				if c.MinVersion == types.MinHookPython && c.VersionFlag != "" && c.ParseVersion != nil {
					floored = append(floored, c)
				} else if c.VersionFlag == "" && c.MinVersion == "" {
					lookupOnly = append(lookupOnly, c)
				}
			}
			if tt.wantBinary == "" {
				if len(floored) != 0 || len(lookupOnly) != 1 || lookupOnly[0].Binary != tt.program {
					t.Errorf("floored %+v, lookup-only %+v; want one lookup-only check for %q", floored, lookupOnly, tt.program)
				}
				return
			}
			if len(floored) != 1 || floored[0].Binary != tt.wantBinary || len(lookupOnly) != 0 {
				t.Errorf("floored %+v, lookup-only %+v; want one python3-floored check of binary %q", floored, lookupOnly, tt.wantBinary)
			}
		})
	}
}
