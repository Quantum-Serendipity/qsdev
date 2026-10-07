package devenv

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/claudesettings"
	"github.com/Quantum-Serendipity/qsdev/internal/doctor"
	"github.com/Quantum-Serendipity/qsdev/internal/sysinfo"
	"github.com/Quantum-Serendipity/qsdev/internal/testutil"
	"github.com/Quantum-Serendipity/qsdev/internal/version"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestDoctorCmd_Flags(t *testing.T) {
	cmd := doctorCmd()

	if cmd.Use != "doctor" {
		t.Errorf("Use = %q, want %q", cmd.Use, "doctor")
	}

	jsonFlag := cmd.Flags().Lookup("json")
	if jsonFlag == nil {
		t.Error("expected --json flag to be registered")
	}
	checkFlag := cmd.Flags().Lookup("check")
	if checkFlag == nil {
		t.Error("expected --check flag to be registered")
	}
}

func TestDoctorCmd_DefaultOutput(t *testing.T) {
	cmd := doctorCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{})

	// Ignore the error since some required tools may not be present.
	_ = cmd.Execute()

	output := buf.String()

	// The report should contain these sections.
	for _, section := range []string{"System", "Shell", "Required Tools"} {
		if !strings.Contains(output, section) {
			t.Errorf("output missing section %q", section)
		}
	}
}

func TestDoctorCmd_JSONOutput(t *testing.T) {
	cmd := doctorCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--json"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("doctor --json failed: %v", err)
	}

	// Verify valid JSON.
	var report doctor.Report
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatalf("invalid JSON output: %v\nOutput: %s", err, buf.String())
	}

	// Verify some expected fields.
	if report.QsdevVersion == "" {
		t.Error("expected non-empty qsdev_version in JSON output")
	}
	if report.System.OS == "" {
		t.Error("expected non-empty system.os in JSON output")
	}
	if report.System.Arch == "" {
		t.Error("expected non-empty system.arch in JSON output")
	}
}

func TestDoctorCmd_CheckMode_Output(t *testing.T) {
	cmd := doctorCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--check"})

	err := cmd.Execute()
	output := buf.String()

	// Either all tools are present (success) or some are missing (error).
	if err == nil {
		if !strings.Contains(output, "All required tools are present") {
			t.Errorf("check mode success should say all tools present, got: %s", output)
		}
	} else {
		if !strings.Contains(output, "Required tools are missing or outdated") {
			t.Errorf("check mode failure should list missing tools, got: %s", output)
		}
	}
}

func TestDoctorCmd_JSONContainsTools(t *testing.T) {
	cmd := doctorCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--json"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("doctor --json failed: %v", err)
	}

	var report doctor.Report
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// There should be at least some required tools listed.
	if len(report.RequiredTools) == 0 {
		t.Error("expected at least one required tool in JSON output")
	}

	// Required tools are the devenv environment prerequisites; language
	// toolchains come from devenv per project and are optional.
	names := make(map[string]bool)
	for _, t := range report.RequiredTools {
		names[t.Name] = true
	}
	for _, expected := range []string{"nix", "devenv", "direnv", "git"} {
		if !names[expected] {
			t.Errorf("expected required tool %q in JSON output", expected)
		}
	}
	for _, optional := range []string{"go", "node", "npm"} {
		if names[optional] {
			t.Errorf("language toolchain %q must not be a required tool", optional)
		}
	}
}

func TestDoctorCmd_JSONReportsBuildVersion(t *testing.T) {
	cmd := doctorCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("doctor --json failed: %v", err)
	}
	var report doctor.Report
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if want := version.Info().Version; report.QsdevVersion != want {
		t.Errorf("qsdev_version = %q, want build version %q", report.QsdevVersion, want)
	}
}

func TestRenderDoctorReport(t *testing.T) {
	t.Parallel()
	missingReport := &doctor.Report{RequiredTools: []doctor.ToolEntry{
		{Name: "git", Found: true, VersionOK: true},
		{Name: "go", Found: false},
	}}
	completeReport := &doctor.Report{RequiredTools: []doctor.ToolEntry{
		{Name: "git", Found: true, VersionOK: true},
	}}

	tests := []struct {
		name     string
		report   *doctor.Report
		json     bool
		check    bool
		wantErr  bool
		wantJSON bool
	}{
		{"json and check with missing tool fails", missingReport, true, true, true, true},
		{"json and check with all tools passes", completeReport, true, true, false, true},
		{"json without check never fails", missingReport, true, false, false, true},
		{"check with missing tool fails", missingReport, false, true, true, false},
		{"check with all tools passes", completeReport, false, true, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			err := renderDoctorReport(&buf, tt.report, tt.json, tt.check)
			if (err != nil) != tt.wantErr {
				t.Errorf("renderDoctorReport() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantJSON {
				var got doctor.Report
				if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
					t.Errorf("output is not valid JSON: %v\n%s", err, buf.String())
				}
			}
		})
	}
}

// TestDefaultChecks_FloorsMatchGenerator: the devenv floor doctor enforces is
// the require_version the generated devenv.yaml declares, and nix has one, so
// a host doctor passes can load what init generates.
func TestDefaultChecks_FloorsMatchGenerator(t *testing.T) {
	t.Parallel()
	floors := map[string]string{}
	for _, c := range doctor.DefaultChecks() {
		floors[c.Name] = c.MinVersion
	}
	gen := strings.TrimPrefix(requireVersion, ">=")
	if floors["devenv"] != gen || gen != types.MinDevenv {
		t.Errorf("devenv floor = %q, generator require_version = %q, types.MinDevenv = %q; want all equal",
			floors["devenv"], requireVersion, types.MinDevenv)
	}
	if floors["nix"] == "" || floors["nix"] != types.MinNix {
		t.Errorf("nix floor = %q, want types.MinNix %q", floors["nix"], types.MinNix)
	}
}

// TestRenderDoctorReport_CheckOutdatedPrintsUpgrade: --check fails on a
// required tool below its floor and says what to upgrade, in text and JSON.
func TestRenderDoctorReport_CheckOutdatedPrintsUpgrade(t *testing.T) {
	t.Parallel()
	osInfo := &sysinfo.OSInfo{OS: "linux", Family: "nixos", Distro: "nixos", PackageManager: "nix", HasNix: true}
	report := doctor.BuildReport(osInfo, []doctor.ToolStatus{
		{Name: "devenv", Required: true, Installed: true, Version: "1.4.1", MinVersion: types.MinDevenv},
		{Name: "nix", Required: true, Installed: true, Version: "2.3.0", MinVersion: types.MinNix},
		{Name: "git", Required: true, Installed: true, Version: "2.47.1", VersionOK: true},
	}, "0.1.0")

	var buf bytes.Buffer
	err := renderDoctorReport(&buf, report, false, true)
	if err == nil {
		t.Fatalf("renderDoctorReport(check) = nil, want an error\n%s", buf.String())
	}
	out := buf.String()
	for _, want := range []string{"Upgrade devenv to >= " + types.MinDevenv, "Upgrade nix to >= " + types.MinNix} {
		if !strings.Contains(out, want) {
			t.Errorf("check output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "All required tools are present") {
		t.Errorf("check output claims success:\n%s", out)
	}

	buf.Reset()
	err = renderDoctorReport(&buf, report, true, true)
	if err == nil || !strings.Contains(err.Error(), "Upgrade devenv to >= "+types.MinDevenv) {
		t.Errorf("renderDoctorReport(json, check) error = %v, want it to name the devenv upgrade", err)
	}
}

// TestHookInterpreterChecks_RequiresPython: a project whose Claude Code hook
// runs the env-python3 package guard needs python3 at the hook floor on
// PATH, so doctor --check fails while it is missing or too old and passes
// once it meets the floor. The other required tools are fakes that meet
// theirs.
func TestHookInterpreterChecks_RequiresPython(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fake tools are POSIX shell scripts")
	}
	project := t.TempDir()
	writeFileMode(t, filepath.Join(project, ".claude", "hooks", "package-guard.py"), "#!/usr/bin/env python3\nprint()\n", 0o755)
	guard := claudesettings.FailClosedCommand("package-guard", `"${CLAUDE_PROJECT_DIR}"/.claude/hooks/package-guard.py`)
	settings, err := json.Marshal(map[string]any{"hooks": map[string]any{"PreToolUse": []any{
		map[string]any{"matcher": "Bash", "hooks": []any{map[string]any{"type": "command", "command": guard}}},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	writeFileMode(t, filepath.Join(project, filepath.FromSlash(claudesettings.ProjectRelPath)), string(settings), 0o644)

	bin := t.TempDir()
	for name, out := range map[string]string{
		"nix":    "nix (Nix) 2.28.0",
		"devenv": "devenv 2.1.2 (x86_64-linux)",
		"direnv": "2.34.0",
		"git":    "git version 2.47.1",
	} {
		writeFileMode(t, filepath.Join(bin, name), "#!/bin/sh\necho '"+out+"'\n", 0o755)
	}
	t.Setenv("PATH", bin)
	osInfo := &sysinfo.OSInfo{OS: "linux", Family: "nixos", Distro: "nixos", PackageManager: "nix", HasNix: true}

	run := func(t *testing.T) (doctor.ToolEntry, string, error) {
		t.Helper()
		checks, err := doctor.ProjectChecks(project)
		if err != nil {
			t.Fatalf("ProjectChecks: %v", err)
		}
		report := doctor.BuildReport(osInfo, doctor.RunChecks(context.Background(), osInfo, checks), "0.1.0")
		var buf bytes.Buffer
		err = renderDoctorReport(&buf, report, false, true)
		for _, e := range report.RequiredTools {
			if e.Name == "python3" {
				return e, buf.String(), err
			}
		}
		t.Fatalf("python3 is not a required tool: %+v", report.RequiredTools)
		return doctor.ToolEntry{}, "", nil
	}

	t.Run("missing", func(t *testing.T) {
		e, out, err := run(t)
		if e.Found || err == nil || !strings.Contains(out, "Install python3") || !strings.Contains(out, doctor.HookRequiredBy) || !strings.Contains(out, "activated devenv shell") {
			t.Errorf("python3 found=%v, err=%v; want a failing check naming the install and the hooks:\n%s", e.Found, err, out)
		}
	})
	py := filepath.Join(bin, "python3")
	t.Run("below floor", func(t *testing.T) {
		writeFileMode(t, py, "#!/bin/sh\necho 'Python 3.8.10'\n", 0o755)
		_, out, err := run(t)
		if err == nil || !strings.Contains(out, "Upgrade python3 to >= "+types.MinHookPython) {
			t.Errorf("err=%v; want a failing check naming the python3 upgrade:\n%s", err, out)
		}
	})
	t.Run("at floor", func(t *testing.T) {
		writeFileMode(t, py, "#!/bin/sh\necho 'Python 3.9.0'\n", 0o755)
		if _, out, err := run(t); err != nil {
			t.Errorf("err=%v; want the check to pass:\n%s", err, out)
		}
	})

	// An activated in-project virtualenv (.venv, devenv's
	// .devenv/state/venv) puts its python3 first on PATH. Doctor never runs
	// a binary inside the project; it reads the version pyvenv.cfg records.
	if err := os.MkdirAll(filepath.Join(project, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", t.TempDir())
	t.Chdir(project)
	venv := filepath.Join(project, ".devenv", "state", "venv")
	marker := filepath.Join(t.TempDir(), "ran")
	writeFileMode(t, filepath.Join(venv, "bin", "python3"), "#!/bin/sh\ntouch '"+marker+"'\necho 'Python 3.12.4'\n", 0o755)
	t.Setenv("PATH", filepath.Join(venv, "bin")+string(os.PathListSeparator)+bin)
	cfg := filepath.Join(venv, "pyvenv.cfg")
	for _, tt := range []struct {
		name, cfg, want string
		pass            bool
	}{
		{"in-project venv at floor", "home = /usr/bin\nversion = 3.12.4\n", "", true},
		{"in-project uv venv at floor", "home = /usr/bin\nversion_info = 3.12.4.final.0\n", "", true},
		{"in-project venv below floor", "version = 3.8.10\n", "Upgrade python3 to >= " + types.MinHookPython, false},
		{"in-project venv of unknown version", "home = /usr/bin\n", "is inside the project, so doctor does not run it; verify it is >= " + types.MinHookPython, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			writeFileMode(t, cfg, tt.cfg, 0o644)
			_, out, err := run(t)
			if (err == nil) != tt.pass || !strings.Contains(out, tt.want) {
				t.Errorf("err=%v, want pass=%v and output containing %q:\n%s", err, tt.pass, tt.want, out)
			}
			if _, statErr := os.Stat(marker); statErr == nil {
				t.Fatal("doctor ran the in-project python3")
			}
		})
	}
}

// writeFileMode writes content to path with mode, creating its directory.
func writeFileMode(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
}

func TestWriterUsesColor_NonTerminalWriter(t *testing.T) {
	t.Parallel()
	if writerUsesColor(&bytes.Buffer{}) {
		t.Error("a buffer must never receive colored output")
	}
}

// TestProjectToolchainWarnings covers the doctor's project toolchain check
// end to end: a Stack snapshot pinning GHC 9.6.7 against a ghc 9.10.3 on
// PATH is reported under the Haskell module.
func TestProjectToolchainWarnings(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake ghc is a shell script")
	}
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "ghc"), []byte("#!/bin/sh\necho 9.10.3\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "stack.yaml"), []byte("snapshot: lts-22.44\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := projectToolchainWarnings(context.Background(), ecosystem.DefaultRegistry(), root)
	if len(got) != 1 || !strings.HasPrefix(got[0], "Haskell: ") || !strings.Contains(got[0], "needs GHC 9.6.7") {
		t.Errorf("projectToolchainWarnings() = %q, want one Haskell GHC 9.6.7 warning", got)
	}
}

// writeDoctorProjectFixture writes a go.mod and a .mcp.json whose one server
// names a command that does not exist into dir.
func writeDoctorProjectFixture(t *testing.T, dir string) {
	t.Helper()
	mcp, err := json.Marshal(map[string]any{"mcpServers": map[string]any{
		"broken": map[string]any{"command": filepath.Join(t.TempDir(), "absent-mcp")},
	}})
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		".mcp.json": string(mcp),
		"go.mod":    "module example.test/doctor\n\ngo 1.22\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// runDoctorFor runs `devenv doctor` with args in the current directory and
// returns its output. Missing required tools on the host are not an error
// here, since the tests look at the project-scoped sections only.
func runDoctorFor(t *testing.T, args ...string) string {
	t.Helper()
	cmd := doctorCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(args)
	_ = cmd.Execute()
	return buf.String()
}

// doctorJSON runs `devenv doctor --json` and decodes the report as a generic
// map, so the tests can tell an absent key from an empty one.
func doctorJSON(t *testing.T) map[string]any {
	t.Helper()
	out := runDoctorFor(t, "--json")
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, out)
	}
	return m
}

// TestDoctor_NoProjectSkipsProjectSections is the U13-V01 regression: outside
// a project doctor ran its project-scoped checks against the working
// directory, reporting its .mcp.json and claiming it had no NFS mounts.
func TestDoctor_NoProjectSkipsProjectSections(t *testing.T) {
	dir := testutil.Project(t, testutil.ProjectOptions{NoGit: true})
	writeDoctorProjectFixture(t, dir)

	human := runDoctorFor(t)
	if !strings.Contains(human, "project checks skipped") {
		t.Errorf("doctor outside a project did not say project checks were skipped:\n%s", human)
	}
	if strings.Contains(human, "MCP Servers") {
		t.Errorf("doctor outside a project reported MCP servers:\n%s", human)
	}
	if strings.Contains(human, "NFS:") {
		t.Errorf("doctor outside a project reported an NFS check:\n%s", human)
	}

	m := doctorJSON(t)
	root, ok := m["project_root"]
	if !ok || root != "" {
		t.Errorf("project_root = %v (present %v), want \"\"", root, ok)
	}
	for _, key := range []string{"mcp_servers", "module_checks"} {
		if _, ok := m[key]; ok {
			t.Errorf("JSON report has %s outside a project: %v", key, m[key])
		}
	}
}

// TestDoctor_InsideProjectRunsProjectSections checks that the U13-V01 gate
// still runs the project-scoped checks from a subdirectory of a project.
func TestDoctor_InsideProjectRunsProjectSections(t *testing.T) {
	proj := testutil.Project(t, testutil.ProjectOptions{NoGit: true})
	writeDoctorProjectFixture(t, proj)
	if err := os.WriteFile(filepath.Join(proj, branding.Get().ConfigFile), []byte("version: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(proj, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(sub)

	m := doctorJSON(t)
	root, _ := m["project_root"].(string)
	got, err := filepath.EvalSymlinks(root)
	if err != nil || got != proj {
		t.Errorf("project_root = %q (resolved %q, err %v), want %q", root, got, err, proj)
	}
	if _, ok := m["mcp_servers"]; !ok {
		t.Error("JSON report inside a project has no mcp_servers section")
	}

	if human := runDoctorFor(t); strings.Contains(human, "project checks skipped") {
		t.Errorf("doctor inside a project said project checks were skipped:\n%s", human)
	}
}

// TestRenderDoctorReport_HookProgramsWarning: when the Claude Code settings
// cannot be read, doctor cannot know the hook programs; every rendering
// (--check text, JSON, the full report) says so instead of a silent pass.
func TestRenderDoctorReport_HookProgramsWarning(t *testing.T) {
	t.Parallel()
	project := t.TempDir()
	writeFileMode(t, filepath.Join(project, filepath.FromSlash(claudesettings.ProjectRelPath)), "{not json", 0o644)
	_, checksErr := doctor.ProjectChecks(project)
	if checksErr == nil {
		t.Fatal("ProjectChecks on malformed settings = nil error")
	}
	osInfo := &sysinfo.OSInfo{OS: "linux"}
	report := doctor.BuildReport(osInfo, nil, "0.1.0")
	report.SetHookProgramsError(checksErr)
	const want = "hook programs were not checked"

	for _, tt := range []struct {
		name        string
		json, check bool
	}{
		{"check text", false, true},
		{"json check", true, true},
		{"full report", false, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			if err := renderDoctorReport(&buf, report, tt.json, tt.check); err != nil {
				t.Fatalf("renderDoctorReport: %v", err)
			}
			if !strings.Contains(buf.String(), want) {
				t.Errorf("output lacks %q:\n%s", want, buf.String())
			}
		})
	}
}
