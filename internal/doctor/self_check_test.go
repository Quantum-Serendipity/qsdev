package doctor

import (
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/sysinfo"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// TestWithSelfCheck: when the hooks run the CLI's own program, its
// lookup-only check becomes a --version probe held to the project's
// qsdev_version, with a PATH hint naming the running binary instead of
// install advice; without such a hook the checks are unchanged.
func TestWithSelfCheck(t *testing.T) {
	t.Parallel()
	app := branding.Get().AppName
	tests := []struct {
		name, program, goos, pathExt string
		wantSelf                     bool
	}{
		{"hooks run the CLI", app, "linux", "", true},
		{"windows spelling", strings.ToUpper(app) + ".EXE", "windows", "", true},
		{"case differs on linux", strings.ToUpper(app), "linux", "", false},
		{"hooks do not run the CLI", "python3", "linux", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			in := requireBinaries(DefaultChecks(), []string{tt.program}, "r", tt.goos, tt.pathExt)
			out := withSelfCheck(in, ">= 0.7.10", "/opt/q/"+app, tt.goos, tt.pathExt)
			i := slices.IndexFunc(out, func(c ToolCheck) bool { return c.Constraint != "" })
			if !tt.wantSelf {
				if i >= 0 {
					t.Errorf("check %+v holds a constraint; want the checks unchanged", out[i])
				}
				return
			}
			if i < 0 {
				t.Fatalf("no check holds the project's constraint: %+v", out)
			}
			c := out[i]
			if c.VersionFlag != "--version" || c.ParseVersion == nil || c.Constraint != ">= 0.7.10" ||
				!strings.Contains(c.PathHint, "/opt/q/"+app) || c.AutoInstall != nil || !c.Required {
				t.Errorf("self check = %+v; want a required --version probe held to the constraint with a PATH hint", c)
			}
			if j := slices.IndexFunc(in, func(c ToolCheck) bool { return c.Constraint != "" || c.PathHint != "" }); j >= 0 {
				t.Error("withSelfCheck modified the checks it was given")
			}
		})
	}
}

func TestParseSelfVersion(t *testing.T) {
	t.Parallel()
	tests := []struct{ raw, want string }{
		{"qsdev version v0.7.10-0.20261006194515-b84cd2f2609b\n", "v0.7.10-0.20261006194515-b84cd2f2609b"},
		{"qsdev version 0.1.0\nBuilt from x\n", "0.1.0"},
		{"acme version 1.2.3", "1.2.3"},
		{"0.1.0", ""},
		{"", ""},
	}
	for _, tt := range tests {
		if got := parseSelfVersion(tt.raw); got != tt.want {
			t.Errorf("parseSelfVersion(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

// TestVersionOK_Constraint: a check with a constraint judges the version by
// it, and an undeterminable version satisfies none.
func TestVersionOK_Constraint(t *testing.T) {
	t.Parallel()
	tc := ToolCheck{Constraint: ">= 0.7.10-0.20261006194515-b84cd2f2609b"}
	tests := []struct {
		version string
		want    bool
	}{
		{"0.1.0", false},
		{"0.7.9+aaf9f08", false},
		{"v0.7.10-0.20261006213341-155123c5777c", true},
		{"0.8.0", true},
		{"", false},
	}
	for _, tt := range tests {
		if got := versionOK(tt.version, tc); got != tt.want {
			t.Errorf("versionOK(%q) = %v, want %v", tt.version, got, tt.want)
		}
	}
	if !versionOK("2.1.0", ToolCheck{MinVersion: "2.1"}) || versionOK("2.0", ToolCheck{MinVersion: "2.1"}) {
		t.Error("versionOK no longer applies MinVersion without a constraint")
	}
}

// TestBuildReport_SelfOnPath: a stale CLI first on PATH fails the required
// tools with a line naming its path, version and the project's constraint,
// and a missing one gets the PATH hint rather than package advice.
func TestBuildReport_SelfOnPath(t *testing.T) {
	t.Parallel()
	app := branding.Get().AppName
	const hint = "put the directory of /opt/q/qsdev first on PATH"
	nixHost := &sysinfo.OSInfo{OS: "linux", Family: "nixos", Distro: "nixos", PackageManager: "nix", HasNix: true}
	stale := BuildReport(nixHost, []ToolStatus{{
		Name: app, Required: true, RequiredBy: HookRequiredBy, Installed: true, Path: "/fake/" + app,
		Version: "0.1.0", Constraint: ">= 0.7.10", PathHint: hint,
	}}, "0.7.10")
	got := stale.RequiredProblems()
	if len(got) != 1 || !strings.Contains(got[0], "/fake/"+app) || !strings.Contains(got[0], "0.1.0") ||
		!strings.Contains(got[0], ">= 0.7.10") || !strings.HasSuffix(got[0], ": "+hint) {
		t.Errorf("RequiredProblems() = %q; want one line naming the path, version, constraint and the PATH hint", got)
	}

	missing := BuildReport(nixHost, []ToolStatus{{Name: app, Required: true, RequiredBy: HookRequiredBy, PathHint: hint}}, "0.7.10")
	got = missing.RequiredProblems()
	if len(got) != 1 || !strings.HasSuffix(got[0], ": "+hint) || strings.Contains(got[0], "no nix package") {
		t.Errorf("RequiredProblems() = %q; want the PATH hint, not package advice", got)
	}
}
