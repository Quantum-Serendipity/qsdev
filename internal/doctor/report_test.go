package doctor

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/sysinfo"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

func TestBuildReport(t *testing.T) {
	osInfo := &sysinfo.OSInfo{
		OS:             "linux",
		Arch:           "amd64",
		Family:         "linux",
		Distro:         "NixOS",
		Version:        "24.11",
		PrettyName:     "NixOS 24.11 (Vicuna)",
		Kernel:         "6.12.1",
		Shell:          "zsh",
		ShellPath:      "/usr/bin/zsh",
		ShellRCFile:    "/home/user/.zshrc",
		PackageManager: "nix",
		HasNix:         true,
	}

	checks := []ToolStatus{
		{Name: "git", Required: true, Installed: true, Version: "2.47.1", VersionOK: true, Path: "/usr/bin/git"},
		{Name: "node", Required: true, Installed: false},
		{Name: "shellcheck", Required: false, Installed: true, Version: "0.10.0", VersionOK: true, Path: "/usr/bin/shellcheck"},
	}

	r := BuildReport(osInfo, checks, "0.1.0")

	if r.QsdevVersion != "0.1.0" {
		t.Errorf("QsdevVersion = %q, want %q", r.QsdevVersion, "0.1.0")
	}
	if r.System.OS != "Linux" {
		t.Errorf("System.OS = %q, want %q", r.System.OS, "Linux")
	}
	if r.System.Distro != "NixOS" {
		t.Errorf("System.Distro = %q, want %q", r.System.Distro, "NixOS")
	}
	if r.Shell.Name != "zsh" {
		t.Errorf("Shell.Name = %q, want %q", r.Shell.Name, "zsh")
	}
	if r.AllRequiredPresent {
		t.Error("AllRequiredPresent should be false when node is missing")
	}
	if len(r.RequiredTools) != 2 {
		t.Errorf("len(RequiredTools) = %d, want 2", len(r.RequiredTools))
	}
	if len(r.OptionalTools) != 1 {
		t.Errorf("len(OptionalTools) = %d, want 1", len(r.OptionalTools))
	}
	if len(r.Recommendations) != 1 {
		t.Errorf("len(Recommendations) = %d, want 1", len(r.Recommendations))
	}
}

// TestBuildReport_NixRecommendations verifies the doctor never recommends the
// denied imperative Nix profile install, and gives unmapped tools explicit
// guidance instead of an empty install command.
func TestBuildReport_NixRecommendations(t *testing.T) {
	osInfo := &sysinfo.OSInfo{OS: "linux", Arch: "amd64", Family: "nixos", Distro: "nixos", PackageManager: "nix", HasNix: true}
	checks := []ToolStatus{
		{Name: "direnv", Required: true, Installed: false},
		{Name: "shfmt", Required: false, Installed: false},
		{Name: "no-such-tool-xyz", Required: false, Installed: false},
	}

	r := BuildReport(osInfo, checks, "0.1.0")

	want := []string{
		"Install direnv: qsdev devenv setup",
		"Install shfmt: qsdev devenv add-package shfmt",
		"Install no-such-tool-xyz: no nix package is known for no-such-tool-xyz; install it from its official distribution",
	}
	if !slices.Equal(r.Recommendations, want) {
		t.Errorf("Recommendations = %q, want %q", r.Recommendations, want)
	}
	for _, e := range append(r.RequiredTools, r.OptionalTools...) {
		if strings.Contains(e.FixCommand, "profile") {
			t.Errorf("%s FixCommand = %q recommends a denied imperative install", e.Name, e.FixCommand)
		}
	}
}

func TestBuildReportAllPresent(t *testing.T) {
	osInfo := &sysinfo.OSInfo{
		OS:             "linux",
		Arch:           "amd64",
		Family:         "linux",
		PackageManager: "nix",
	}

	checks := []ToolStatus{
		{Name: "git", Required: true, Installed: true, Version: "2.43.0", VersionOK: true},
		{Name: "go", Required: true, Installed: true, Version: "1.22.3", VersionOK: true},
	}

	r := BuildReport(osInfo, checks, "0.1.0")
	if !r.AllRequiredPresent {
		t.Error("AllRequiredPresent should be true when all required tools are present")
	}
}

func TestFormatReportNoColor(t *testing.T) {
	r := &Report{
		QsdevVersion: "0.1.0",
		System: SystemInfo{
			OS:         "Linux",
			PrettyName: "NixOS 24.11",
			Arch:       "amd64",
			Kernel:     "6.12.1",
		},
		Shell: ShellInfo{
			Name:   "zsh",
			RCFile: "/home/user/.zshrc",
		},
		PackageMgrs: []PkgMgrInfo{
			{Name: "nix", Primary: true},
		},
		RequiredTools: []ToolEntry{
			{Name: "git", Found: true, Version: "2.47.1", VersionOK: true, Path: "/usr/bin/git"},
			{Name: "node", Found: false},
		},
		OptionalTools: []ToolEntry{
			{Name: "shellcheck", Found: true, Version: "0.10.0", VersionOK: true, Path: "/usr/bin/shellcheck"},
		},
		Recommendations: []string{
			"Install node: qsdev devenv setup",
		},
		AllRequiredPresent: false,
	}

	var buf bytes.Buffer
	FormatReport(&buf, r, false)
	output := buf.String()

	// Check key sections are present
	if !strings.Contains(output, "qsdev doctor v0.1.0") {
		t.Error("missing header")
	}
	if !strings.Contains(output, "Linux (NixOS 24.11)") {
		t.Error("missing OS info")
	}
	if !strings.Contains(output, "amd64") {
		t.Error("missing architecture")
	}
	if !strings.Contains(output, "zsh") {
		t.Error("missing shell")
	}
	if !strings.Contains(output, "Required Tools") {
		t.Error("missing Required Tools section")
	}
	if !strings.Contains(output, "[OK]") {
		t.Error("missing [OK] symbol in no-color mode")
	}
	if !strings.Contains(output, "[FAIL]") {
		t.Error("missing [FAIL] symbol in no-color mode")
	}
	if !strings.Contains(output, "Recommendations") {
		t.Error("missing Recommendations section")
	}
}

func TestFormatReportWithColor(t *testing.T) {
	r := &Report{
		QsdevVersion: "0.1.0",
		System: SystemInfo{
			OS:   "Linux",
			Arch: "amd64",
		},
		Shell: ShellInfo{
			Name: "bash",
		},
		RequiredTools: []ToolEntry{
			{Name: "git", Found: true, Version: "2.47.1", VersionOK: true, Path: "/usr/bin/git"},
		},
	}

	var buf bytes.Buffer
	FormatReport(&buf, r, true)
	output := buf.String()

	// Color mode should use ANSI escape sequences
	if !strings.Contains(output, "\033[32m") {
		t.Error("missing green ANSI escape in color mode")
	}
}

func TestReportJSONRoundTrip(t *testing.T) {
	original := &Report{
		QsdevVersion: "0.1.0",
		Timestamp:    "2024-01-15T10:30:00Z",
		System: SystemInfo{
			OS:     "Linux",
			Arch:   "amd64",
			Distro: "NixOS",
		},
		Shell: ShellInfo{
			Name: "zsh",
			Path: "/usr/bin/zsh",
		},
		PackageMgrs: []PkgMgrInfo{
			{Name: "nix", Primary: true},
		},
		RequiredTools: []ToolEntry{
			{Name: "git", Found: true, Version: "2.43.0", VersionOK: true, Path: "/usr/bin/git"},
		},
		OptionalTools: []ToolEntry{
			{Name: "jq", Found: true, Version: "1.7.1", VersionOK: true, Path: "/usr/bin/jq"},
		},
		AllRequiredPresent: true,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var decoded Report
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if decoded.QsdevVersion != original.QsdevVersion {
		t.Errorf("QsdevVersion mismatch: %q vs %q", decoded.QsdevVersion, original.QsdevVersion)
	}
	if decoded.System.Distro != original.System.Distro {
		t.Errorf("Distro mismatch: %q vs %q", decoded.System.Distro, original.System.Distro)
	}
	if decoded.AllRequiredPresent != original.AllRequiredPresent {
		t.Errorf("AllRequiredPresent mismatch: %v vs %v", decoded.AllRequiredPresent, original.AllRequiredPresent)
	}
	if len(decoded.RequiredTools) != 1 {
		t.Fatalf("RequiredTools length = %d, want 1", len(decoded.RequiredTools))
	}
	if decoded.RequiredTools[0].Name != "git" {
		t.Errorf("RequiredTools[0].Name = %q, want %q", decoded.RequiredTools[0].Name, "git")
	}
}

func TestUseColorWithNO_COLOR(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	// Even if fd were a terminal, NO_COLOR should force false
	if UseColor(0) {
		t.Error("UseColor should return false when NO_COLOR is set")
	}
}

func TestUseColorWithDumbTerm(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "dumb")
	if UseColor(0) {
		t.Error("UseColor should return false when TERM=dumb")
	}
}

func TestUseColorNonTerminal(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "xterm-256color")

	// A regular file is never a terminal, whatever the test's own stdio is.
	f, err := os.CreateTemp(t.TempDir(), "not-a-tty")
	if err != nil {
		t.Fatalf("creating temp file: %v", err)
	}
	defer f.Close()

	if UseColor(f.Fd()) {
		t.Error("UseColor should return false for a non-terminal file descriptor")
	}
}

// TestBuildReport_FixCommands is the W162 regression test: on a Nix host the
// doctor must not recommend an imperative profile install (forbidden by the
// generated security rules and deny list), and a tool with no package mapping
// must get guidance rather than an empty command.
func TestBuildReport_FixCommands(t *testing.T) {
	t.Parallel()

	nixHost := &sysinfo.OSInfo{OS: "linux", Family: "nixos", Distro: "nixos", PackageManager: "nix", HasNix: true}
	aptHost := &sysinfo.OSInfo{OS: "linux", Family: "debian", Distro: "ubuntu", PackageManager: "apt"}

	tests := []struct {
		name   string
		osInfo *sysinfo.OSInfo
		tool   ToolStatus
		want   string
	}{
		{"nix optional tool is pinned in the project", nixHost, ToolStatus{Name: "shfmt"}, "qsdev devenv add-package shfmt"},
		{"nix required prerequisite uses setup", nixHost, ToolStatus{Name: "direnv", Required: true}, "qsdev devenv setup"},
		{"nix outdated optional tool is pinned in the project", nixHost, ToolStatus{Name: "shfmt", Installed: true, MinVersion: "9", Version: "3"}, "qsdev devenv add-package shfmt"},
		{"unmapped tool gets guidance", nixHost, ToolStatus{Name: "no-such-tool"}, "no nix package is known for no-such-tool"},
		{"apt host keeps its native install command", aptHost, ToolStatus{Name: "shfmt"}, "apt"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := BuildReport(tt.osInfo, []ToolStatus{tt.tool}, "0.1.0")
			if len(r.Recommendations) != 1 {
				t.Fatalf("Recommendations = %q, want exactly 1", r.Recommendations)
			}
			rec := r.Recommendations[0]
			if !strings.Contains(rec, tt.want) {
				t.Errorf("recommendation = %q, want it to contain %q", rec, tt.want)
			}
			if strings.Contains(rec, "profile install") || strings.HasSuffix(strings.TrimSpace(rec), ":") {
				t.Errorf("recommendation = %q: imperative Nix install or empty command", rec)
			}
		})
	}
}

func TestFormatReport_ProjectToolchains(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		warnings []string
		want     []string
		notWant  []string
	}{
		{
			name:     "warnings shown",
			warnings: []string{"Haskell: stack.yaml needs GHC 9.6.7"},
			want:     []string{"Project Toolchains\n", "  [WARN] Haskell: stack.yaml needs GHC 9.6.7\n"},
		},
		{name: "section omitted without warnings", notWant: []string{"Project Toolchains"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := &Report{QsdevVersion: "0.1.0", System: SystemInfo{OS: "Linux", Arch: "amd64"}}
			r.SetProjectToolchains(tt.warnings)
			var buf bytes.Buffer
			FormatReport(&buf, r, false)
			out := buf.String()
			for _, sub := range tt.want {
				if !strings.Contains(out, sub) {
					t.Errorf("output does not contain %q:\n%s", sub, out)
				}
			}
			for _, sub := range tt.notWant {
				if strings.Contains(out, sub) {
					t.Errorf("output contains %q:\n%s", sub, out)
				}
			}
			data, err := json.Marshal(r)
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.Contains(string(data), `"project_toolchains"`); got != (len(tt.warnings) > 0) {
				t.Errorf("JSON has project_toolchains = %v, want %v: %s", got, len(tt.warnings) > 0, data)
			}
		})
	}
}

// TestFormatReport_ProjectSkippedNote checks that the report says when the
// project-scoped checks were skipped, and that project_root is always in the
// JSON so consumers can tell "skipped" from "clean".
func TestFormatReport_ProjectSkippedNote(t *testing.T) {
	t.Parallel()
	note := "Not inside a " + branding.Get().AppName + " project — project checks skipped."
	tests := []struct {
		name        string
		projectRoot string
		wantNote    bool
	}{
		{name: "outside a project", projectRoot: "", wantNote: true},
		{name: "inside a project", projectRoot: filepath.Join("work", "proj"), wantNote: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := &Report{QsdevVersion: "0.1.0", System: SystemInfo{OS: "Linux", Arch: "amd64"}, ProjectRoot: tt.projectRoot}
			var buf bytes.Buffer
			FormatReport(&buf, r, false)
			if got := strings.Contains(buf.String(), note); got != tt.wantNote {
				t.Errorf("output contains %q = %v, want %v:\n%s", note, got, tt.wantNote, buf.String())
			}
			data, err := json.Marshal(r)
			if err != nil {
				t.Fatal(err)
			}
			var m map[string]any
			if err := json.Unmarshal(data, &m); err != nil {
				t.Fatal(err)
			}
			if got, ok := m["project_root"]; !ok || got != tt.projectRoot {
				t.Errorf("JSON project_root = %v (present %v), want %q", got, ok, tt.projectRoot)
			}
		})
	}
}

// TestRequiredProblems: every required tool that blocks the environment gets
// one actionable line (missing, outdated or of unknown version), optional
// tools never do, and the recommendations reuse the same lines.
func TestRequiredProblems(t *testing.T) {
	t.Parallel()
	nixHost := &sysinfo.OSInfo{OS: "linux", Family: "nixos", Distro: "nixos", PackageManager: "nix", HasNix: true}
	checks := []ToolStatus{
		{Name: "git", Required: true, Installed: true, Version: "2.47.1", VersionOK: true},
		{Name: "nix", Required: true, Installed: false, MinVersion: "2.4"},
		{Name: "devenv", Required: true, Installed: true, Version: "1.4.1", MinVersion: "2.1"},
		{Name: "python3", Required: true, Installed: true, MinVersion: "3.9"},
		{Name: "shfmt", Required: false, Installed: false},
	}

	r := BuildReport(nixHost, checks, "0.1.0")

	want := []string{
		"Install nix: qsdev devenv setup",
		"Upgrade devenv to >= 2.1: qsdev devenv setup",
		"Could not determine python3 version (need >= 3.9)",
	}
	if got := r.RequiredProblems(); !slices.Equal(got, want) {
		t.Errorf("RequiredProblems() = %q, want %q", got, want)
	}
	if r.AllRequiredPresent {
		t.Error("AllRequiredPresent = true with required problems")
	}
	wantRecs := append(slices.Clone(want), "Install shfmt: qsdev devenv add-package shfmt")
	if !slices.Equal(r.Recommendations, wantRecs) {
		t.Errorf("Recommendations = %q, want %q", r.Recommendations, wantRecs)
	}
}

// TestRequiredProblems_None: a report whose required tools all meet their
// floors has no problems.
func TestRequiredProblems_None(t *testing.T) {
	t.Parallel()
	r := BuildReport(&sysinfo.OSInfo{OS: "linux", PackageManager: "nix", HasNix: true}, []ToolStatus{
		{Name: "devenv", Required: true, Installed: true, Version: "2.1.2", MinVersion: "2.1", VersionOK: true},
	}, "0.1.0")
	if got := r.RequiredProblems(); len(got) != 0 {
		t.Errorf("RequiredProblems() = %q, want none", got)
	}
}

// TestBuildReport_RequiredByHooks: a tool required because something other
// than the environment needs it says so, in the problem line and the table,
// and is not sent to `devenv setup`, which installs prerequisites only.
func TestBuildReport_RequiredByHooks(t *testing.T) {
	t.Parallel()
	const why = "needed by Claude Code hooks"
	nixHost := &sysinfo.OSInfo{OS: "linux", Family: "nixos", Distro: "nixos", PackageManager: "nix", HasNix: true}
	r := BuildReport(nixHost, []ToolStatus{
		{Name: "no-such-hook-tool", Required: true, RequiredBy: why},
		{Name: "shfmt", Required: true, RequiredBy: why, Installed: true, Version: "3.0", MinVersion: "3.9"},
	}, "0.1.0")

	want := []string{
		"Install no-such-hook-tool, " + why + ": no nix package is known for no-such-hook-tool; install it from its official distribution",
		"Upgrade shfmt to >= 3.9, " + why + ": qsdev devenv add-package shfmt",
	}
	if got := r.RequiredProblems(); !slices.Equal(got, want) {
		t.Errorf("RequiredProblems() = %q, want %q", got, want)
	}
	var buf bytes.Buffer
	FormatReport(&buf, r, false)
	if !strings.Contains(buf.String(), why) {
		t.Errorf("report does not say why the tool is required:\n%s", buf.String())
	}
}

// TestReport_HookProgramNameTerminalSafe: a program name from repository
// content (a hook command word, a CRLF shebang) cannot put raw control bytes
// into the problem lines or the table.
func TestReport_HookProgramNameTerminalSafe(t *testing.T) {
	t.Parallel()
	const why = "needed by Claude Code hooks"
	nixHost := &sysinfo.OSInfo{OS: "linux", Family: "nixos", Distro: "nixos", PackageManager: "nix", HasNix: true}
	r := BuildReport(nixHost, []ToolStatus{
		{Name: "\x1b[1A\x1b[2K\rAll required tools are present.\x1b[8m selfprotect", Required: true, RequiredBy: why},
		{Name: "python3\r", Required: true, RequiredBy: why},
	}, "0.1.0")

	var buf bytes.Buffer
	FormatReport(&buf, r, false)
	out := strings.Join(r.RequiredProblems(), "\n") + buf.String()
	if i := strings.IndexFunc(out, func(c rune) bool { return c == '\x1b' || c == '\r' }); i >= 0 {
		t.Errorf("output holds a raw control byte at %d:\n%q", i, out)
	}
	if !strings.Contains(out, `"python3\r"`) {
		t.Errorf("output does not show the CRLF name quoted:\n%s", out)
	}
}

// TestBuildReport_InProjectUnknownVersion: a required tool found inside the
// project, which doctor does not run, says so instead of "could not
// determine".
func TestBuildReport_InProjectUnknownVersion(t *testing.T) {
	t.Parallel()
	r := BuildReport(&sysinfo.OSInfo{OS: "linux"}, []ToolStatus{
		{Name: "python3", Required: true, Installed: true, InProject: true, Path: "/p/.venv/bin/python3", MinVersion: "3.9"},
	}, "0.1.0")
	want := []string{"python3 at /p/.venv/bin/python3 is inside the project, so doctor does not run it; verify it is >= 3.9"}
	if got := r.RequiredProblems(); !slices.Equal(got, want) {
		t.Errorf("RequiredProblems() = %q, want %q", got, want)
	}
}

// TestBuildReport_OutdatedNixUpgradeHint: an installed nix below its floor
// is pointed at an in-place upgrade, not at setup, which would run the Nix
// installer over it.
func TestBuildReport_OutdatedNixUpgradeHint(t *testing.T) {
	t.Parallel()
	tc, ok := CheckNamed("nix")
	if !ok || tc.UpgradeHint == "" {
		t.Fatalf("nix check = %+v; want an UpgradeHint", tc)
	}
	nixHost := &sysinfo.OSInfo{OS: "linux", Family: "debian", PackageManager: "nix", HasNix: true}
	r := BuildReport(nixHost, []ToolStatus{
		{Name: "nix", Required: true, Installed: true, Version: "2.3.0", MinVersion: "2.4", UpgradeHint: tc.UpgradeHint},
	}, "0.1.0")
	want := []string{"Upgrade nix to >= 2.4: " + tc.UpgradeHint}
	if got := r.RequiredProblems(); !slices.Equal(got, want) {
		t.Errorf("RequiredProblems() = %q, want %q", got, want)
	}
}
