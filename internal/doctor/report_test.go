package doctor

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/sysinfo"
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
			"Install node: nix profile install nixpkgs#nodejs",
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
		Timestamp:   "2024-01-15T10:30:00Z",
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
	_ = UseColor(0)
}
