package devinit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Lifecycle round-trip tests (F046): the enable/disable/update/repair bugs
// only show up across a sequence of commands, so these drive the real
// commands against an initialised project.

// initLifecycleProject runs `init --yes --lang go --tier full` in a fresh
// directory with a go.mod and returns the directory.
func initLifecycleProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/lc\n\ngo 1.24\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := executeInitCmd(t, dir, "--yes", "--lang", "go", "--tier", "full"); err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}
	return dir
}

// runLifecycleCmd executes a lifecycle subcommand from dir.
func runLifecycleCmd(t *testing.T, dir string, cmd *cobra.Command, args ...string) (string, error) {
	t.Helper()
	t.Setenv("QSDEV_SKIP_SETUP", "1")
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return buf.String(), err
}

func enableTool(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	return runLifecycleCmd(t, dir, enableCmd(), args...)
}

func disableTool(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	return runLifecycleCmd(t, dir, disableCmd(), args...)
}

func mustEnable(t *testing.T, dir string, args ...string) {
	t.Helper()
	if out, err := enableTool(t, dir, args...); err != nil {
		t.Fatalf("enable %v: %v\n%s", args, err, out)
	}
}

func mustDisable(t *testing.T, dir string, args ...string) {
	t.Helper()
	if out, err := disableTool(t, dir, args...); err != nil {
		t.Fatalf("disable %v: %v\n%s", args, err, out)
	}
}

func readProjectFile(t *testing.T, dir, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, rel))
	if err != nil {
		t.Fatalf("reading %s: %v", rel, err)
	}
	return string(data)
}

func loadProjectState(t *testing.T, dir string) types.GeneratedState {
	t.Helper()
	st, err := state.LoadStateFromFile(filepath.Join(dir, stateFilePath()))
	if err != nil {
		t.Fatalf("loading state: %v", err)
	}
	return st
}

func loadProjectAnswers(t *testing.T, dir string) types.WizardAnswers {
	t.Helper()
	a, err := loadAnswersOrEmpty(dir)
	if err != nil {
		t.Fatalf("loading answers: %v", err)
	}
	return a
}

// assertSharedFilesWellFormed checks every shared file is in its own format:
// JSON files parse, CLAUDE.md carries no raw MCP JSON, and devenv.nix carries
// no Markdown and parses as Nix when nix-instantiate is available.
func assertSharedFilesWellFormed(t *testing.T, dir string) {
	t.Helper()
	for _, rel := range []string{".mcp.json", ".claude/settings.json"} {
		data, err := os.ReadFile(filepath.Join(dir, rel))
		if err != nil {
			continue
		}
		if !json.Valid(data) {
			t.Errorf("%s is not valid JSON:\n%s", rel, data)
		}
	}
	if md, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md")); err == nil {
		if bytes.Contains(md, []byte(`{"command"`)) {
			t.Errorf("CLAUDE.md contains raw MCP server JSON:\n%s", md)
		}
	}
	nix, err := os.ReadFile(filepath.Join(dir, "devenv.nix"))
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(nix), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "- **") {
			t.Errorf("devenv.nix contains a Markdown line: %q", line)
		}
	}
	if nixInstantiate, err := exec.LookPath("nix-instantiate"); err == nil {
		if out, err := exec.Command(nixInstantiate, "--parse", filepath.Join(dir, "devenv.nix")).CombinedOutput(); err != nil {
			t.Errorf("devenv.nix does not parse: %v\n%s", err, out)
		}
	}
}

// TestLifecycle_SharedContentMatchesFileFormat is the F027 regression: tools
// reuse one section ID across files (devenv.nix + CLAUDE.md, .mcp.json +
// CLAUDE.md), so content must be keyed per file. Every registered content
// function must produce its own file's format.
func TestLifecycle_SharedContentMatchesFileFormat(t *testing.T) {
	t.Parallel()
	answers := types.WizardAnswers{ProjectName: "demo"}
	for _, tool := range toolreg.DefaultRegistry().All() {
		for key, fn := range tool.SharedContent {
			content, err := fn(answers)
			if err != nil {
				t.Errorf("%s %s#%s: %v", tool.Name, key.Path, key.SectionID, err)
				continue
			}
			trimmed := strings.TrimSpace(string(content))
			switch {
			case strings.HasSuffix(key.Path, ".json"):
				if !json.Valid(content) {
					t.Errorf("%s: %s#%s content is not JSON: %q", tool.Name, key.Path, key.SectionID, trimmed)
				}
			case strings.HasSuffix(key.Path, ".nix"):
				if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "{\"") {
					t.Errorf("%s: %s#%s content is not Nix: %q", tool.Name, key.Path, key.SectionID, trimmed)
				}
			case strings.HasSuffix(key.Path, ".md"):
				if json.Valid(content) {
					t.Errorf("%s: %s#%s content is JSON, not Markdown: %q", tool.Name, key.Path, key.SectionID, trimmed)
				}
			}
		}
	}
}

// TestLifecycle_EnableProducesWellFormedSharedFiles covers F027 end to end
// and F030's enable half: enabling tools whose section IDs are shared across
// files leaves every shared file well formed, and the tools' contributions
// land in the right file.
func TestLifecycle_EnableProducesWellFormedSharedFiles(t *testing.T) {
	dir := initLifecycleProject(t)

	for _, tool := range []string{"opengrep", "changelog", "postgres-mcp", "mcp-nixos", "commit-ticket", "starship-integration"} {
		mustEnable(t, dir, tool)
		assertSharedFilesWellFormed(t, dir)
	}

	nix := readProjectFile(t, dir, "devenv.nix")
	for _, want := range []string{"env.STARSHIP_CONFIG", "git-hooks.hooks.commit-ticket"} {
		if !devenvDefines(t, nix, want) {
			t.Errorf("devenv.nix missing %q after enable", want)
		}
	}
	mcp := readProjectFile(t, dir, ".mcp.json")
	for _, server := range []string{`"postgres"`, `"mcp-nixos"`} {
		if !strings.Contains(mcp, server) {
			t.Errorf(".mcp.json missing server %s after enable:\n%s", server, mcp)
		}
	}
	md := readProjectFile(t, dir, "CLAUDE.md")
	for _, want := range []string{"OpenGrep SAST", "PostgreSQL MCP"} {
		if !strings.Contains(md, want) {
			t.Errorf("CLAUDE.md missing %q after enable", want)
		}
	}
}

// TestLifecycle_EnableThenUpdateKeepsNixSections is the F030 regression: a
// routine update must not drop the devenv.nix sections of tools that are
// still enabled.
func TestLifecycle_EnableThenUpdateKeepsNixSections(t *testing.T) {
	dir := initLifecycleProject(t)
	mustEnable(t, dir, "starship-integration")
	mustEnable(t, dir, "commit-ticket")

	if out, err := executeInitCmd(t, dir, "--update"); err != nil {
		t.Fatalf("update: %v\n%s", err, out)
	}

	nix := readProjectFile(t, dir, "devenv.nix")
	for _, want := range []string{"env.STARSHIP_CONFIG", "git-hooks.hooks.commit-ticket"} {
		if !devenvDefines(t, nix, want) {
			t.Errorf("devenv.nix lost %q after update while the tool is still enabled", want)
		}
	}
	assertSharedFilesWellFormed(t, dir)

	mustDisable(t, dir, "commit-ticket")
	if devenvDefines(t, readProjectFile(t, dir, "devenv.nix"), "git-hooks.hooks.commit-ticket") {
		t.Error("devenv.nix still has the commit-ticket hook after disable")
	}
}

// TestLifecycle_EnableKeepsSharedFileStrategy is the F028 regression: enable
// and disable must keep each shared file's recorded merge strategy, so a
// later repair treats CLAUDE.md as user-editable and keeps the user's notes.
func TestLifecycle_EnableKeepsSharedFileStrategy(t *testing.T) {
	dir := initLifecycleProject(t)
	before := loadProjectState(t, dir)

	mustEnable(t, dir, "postgres-mcp")
	after := loadProjectState(t, dir)
	for _, rel := range []string{"CLAUDE.md", ".mcp.json"} {
		if got, want := after.Files[rel].Strategy, before.Files[rel].Strategy; got != want {
			t.Errorf("%s strategy after enable = %v, want %v", rel, got, want)
		}
		if after.Files[rel].Owner == "postgres-mcp" {
			t.Errorf("%s owner must not become the last-enabled tool", rel)
		}
	}
	if len(after.Files[".mcp.json"].BaseContent) == 0 {
		t.Error(".mcp.json lost its three-way-merge base content")
	}

	claudeMD := filepath.Join(dir, "CLAUDE.md")
	f, err := os.OpenFile(claudeMD, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	const notes = "\n## Team notes\n\nKeep this.\n"
	if _, err := f.WriteString(notes); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	// Target CLAUDE.md only: other repair items need a real repository. Repair
	// exits non-zero when it skips a file, so judge it by what it did.
	out, _ := runLifecycleCmd(t, dir, repairCmd(), "--file", "CLAUDE.md")
	if strings.Contains(out, "[fix] CLAUDE.md") || !strings.Contains(out, "[skip] CLAUDE.md") {
		t.Errorf("repair must treat CLAUDE.md as user-editable and skip it:\n%s", out)
	}
	if !strings.Contains(readProjectFile(t, dir, "CLAUDE.md"), "Keep this.") {
		t.Fatal("repair overwrote the user's CLAUDE.md notes after an enable")
	}

	mustDisable(t, dir, "postgres-mcp")
	afterDisable := loadProjectState(t, dir)
	if got, want := afterDisable.Files["CLAUDE.md"].Strategy, before.Files["CLAUDE.md"].Strategy; got != want {
		t.Errorf("CLAUDE.md strategy after disable = %v, want %v", got, want)
	}
	if !strings.Contains(readProjectFile(t, dir, "CLAUDE.md"), "Keep this.") {
		t.Error("disable dropped the user's CLAUDE.md notes")
	}
	if strings.Contains(readProjectFile(t, dir, ".mcp.json"), `"postgres"`) {
		t.Error(".mcp.json still lists the postgres server after disable")
	}
}

// TestLifecycle_OpengrepEnableDisable is the F029 regression: opengrep owns a
// whole directory of rule files; disable must remove them all, succeed, and
// leave state and answers consistent.
func TestLifecycle_OpengrepEnableDisable(t *testing.T) {
	dir := initLifecycleProject(t)
	mustEnable(t, dir, "opengrep")

	if len(opengrepRuleFiles(t, dir)) == 0 {
		t.Fatal("enable wrote no opengrep rule files")
	}
	// F217: enabling opengrep must wire a scan, not just deliver rules.
	const scanCmd = "opengrep scan --config .opengrep/rules/core --error"
	if !strings.Contains(readProjectFile(t, dir, "devenv.nix"), scanCmd) {
		t.Errorf("devenv.nix security-scan task does not run %q after enable", scanCmd)
	}

	mustDisable(t, dir, "opengrep")
	if strings.Contains(readProjectFile(t, dir, "devenv.nix"), scanCmd) {
		t.Error("devenv.nix still runs opengrep after disable")
	}
	if _, err := os.Stat(filepath.Join(dir, ".opengrep")); !os.IsNotExist(err) {
		t.Errorf(".opengrep must be removed after disable (stat err=%v)", err)
	}
	for path := range loadProjectState(t, dir).Files {
		if strings.HasPrefix(path, ".opengrep/") {
			t.Errorf("state still tracks %s after disable", path)
		}
	}
	if loadProjectAnswers(t, dir).EnabledTools["opengrep"] {
		t.Error("answers still mark opengrep enabled")
	}
	if strings.Contains(readProjectFile(t, dir, "CLAUDE.md"), "OpenGrep SAST") {
		t.Error("CLAUDE.md still advertises OpenGrep after disable")
	}

	// A modified rule blocks disable before anything is deleted.
	mustEnable(t, dir, "opengrep")
	rules := opengrepRuleFiles(t, dir)
	if len(rules) == 0 {
		t.Fatal("re-enable wrote no opengrep rule files")
	}
	if err := os.WriteFile(rules[0], []byte("# edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := disableTool(t, dir, "opengrep"); err == nil || !strings.Contains(err.Error(), "modified") {
		t.Fatalf("disable with a modified rule: want modification error, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".opengrep", "nix", "default.nix")); err != nil {
		t.Errorf("refused disable must not delete anything: %v", err)
	}
}

// TestLifecycle_OpengrepUpdateRetiresConfig is the F217 migration: projects
// that enabled opengrep before the invented .opengrep/config.yaml was dropped
// still track it as opengrep-owned. Update regenerates opengrep's files, so a
// tracked file it no longer produces is retired: removed when unmodified, left
// in place and untracked when the user edited it.
func TestLifecycle_OpengrepUpdateRetiresConfig(t *testing.T) {
	tests := []struct {
		name     string
		edit     bool
		wantFile bool
	}{
		{name: "unmodified legacy config is removed", edit: false, wantFile: false},
		{name: "edited legacy config is kept and untracked", edit: true, wantFile: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := initLifecycleProject(t)
			mustEnable(t, dir, "opengrep")

			const legacy = ".opengrep/config.yaml"
			st := loadProjectState(t, dir)
			writeTrackedFile(t, &st, dir, legacy, "rules:\n  - .opengrep/rules/core\nseverity: warning\n", "opengrep")
			saveProjectState(t, dir, st)
			if tt.edit {
				if err := os.WriteFile(filepath.Join(dir, legacy), []byte("# user edit\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			if out, err := executeInitCmd(t, dir, "--update"); err != nil {
				t.Fatalf("update failed: %v\n%s", err, out)
			}

			_, statErr := os.Stat(filepath.Join(dir, legacy))
			if exists := statErr == nil; exists != tt.wantFile {
				t.Errorf("%s exists = %v after update, want %v", legacy, exists, tt.wantFile)
			}
			after := loadProjectState(t, dir)
			if _, tracked := after.Files[legacy]; tracked {
				t.Errorf("%s is still tracked after update", legacy)
			}
			if len(opengrepRuleFiles(t, dir)) == 0 {
				t.Error("update must keep the opengrep rule library of the still-enabled tool")
			}
			if _, tracked := after.Files[".opengrep/nix/default.nix"]; !tracked {
				t.Error("update untracked the still-generated opengrep derivation")
			}
		})
	}
}

// TestLifecycle_SemgrepUpdateRetiresLegacyConfig is the F202 migration:
// projects initialized before .semgrep.yml was dropped still track it as
// semgrep-owned. Semgrep could not load it (registry refs under rules: and a
// paths: key) and the security-scan task never read it. Update regenerates
// semgrep's files, so the legacy file is retired (removed when unmodified,
// untracked when edited) while .semgrepignore and the project's own .semgrep/
// rules stay, and the task runs the rule packs plus those rules.
func TestLifecycle_SemgrepUpdateRetiresLegacyConfig(t *testing.T) {
	tests := []struct {
		name     string
		edit     bool
		wantFile bool
	}{
		{name: "unmodified legacy config is removed", edit: false, wantFile: false},
		{name: "edited legacy config is kept and untracked", edit: true, wantFile: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := initLifecycleProject(t)

			const legacy = ".semgrep.yml"
			st := loadProjectState(t, dir)
			writeTrackedFile(t, &st, dir, legacy, "rules:\n  - p/golang\n\npaths:\n  exclude:\n    - vendor/\n", "semgrep")
			saveProjectState(t, dir, st)
			if tt.edit {
				if err := os.WriteFile(filepath.Join(dir, legacy), []byte("# user edit\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			localRule := filepath.Join(dir, ".semgrep", "team.yml")
			if err := os.MkdirAll(filepath.Dir(localRule), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(localRule, []byte("rules: []\n"), 0o644); err != nil {
				t.Fatal(err)
			}

			if out, err := executeInitCmd(t, dir, "--update"); err != nil {
				t.Fatalf("update failed: %v\n%s", err, out)
			}

			_, statErr := os.Stat(filepath.Join(dir, legacy))
			if exists := statErr == nil; exists != tt.wantFile {
				t.Errorf("%s exists = %v after update, want %v", legacy, exists, tt.wantFile)
			}
			after := loadProjectState(t, dir)
			if _, tracked := after.Files[legacy]; tracked {
				t.Errorf("%s is still tracked after update", legacy)
			}
			if _, tracked := after.Files[".semgrepignore"]; !tracked {
				t.Error("update untracked the still-generated .semgrepignore")
			}
			if _, err := os.Stat(localRule); err != nil {
				t.Errorf("update must leave the project's own .semgrep/ rules alone: %v", err)
			}
			nixSrc := readProjectFile(t, dir, "devenv.nix")
			for _, want := range []string{"semgrep --config p/golang", "echo --config .semgrep", "--metrics=off --error ."} {
				if !strings.Contains(nixSrc, want) {
					t.Errorf("devenv.nix security-scan task missing %q", want)
				}
			}
		})
	}
}

// opengrepRuleFiles lists the rule files under .opengrep/rules/core.
func opengrepRuleFiles(t *testing.T, dir string) []string {
	t.Helper()
	var files []string
	root := filepath.Join(dir, ".opengrep", "rules", "core")
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files
}

// TestLifecycle_AttachGuardDisableEnable is the F037 regression: the package
// guard must be restorable with enable after a forced disable, and disable
// must drop its settings.json hook instead of pointing at a deleted script.
func TestLifecycle_AttachGuardDisableEnable(t *testing.T) {
	dir := initLifecycleProject(t)

	mustDisable(t, dir, "attach-guard", "--force")
	if _, err := os.Stat(filepath.Join(dir, ".claude", "hooks", "package-guard.py")); !os.IsNotExist(err) {
		t.Errorf("package-guard.py should be removed by disable (stat err=%v)", err)
	}
	if strings.Contains(readProjectFile(t, dir, ".claude/settings.json"), "package-guard") {
		t.Error("settings.json still references package-guard after disable")
	}

	mustEnable(t, dir, "attach-guard")
	if _, err := os.Stat(filepath.Join(dir, ".claude", "hooks", "package-guard.py")); err != nil {
		t.Errorf("enable must restore package-guard.py: %v", err)
	}
	if !strings.Contains(readProjectFile(t, dir, ".claude/settings.json"), "package-guard") {
		t.Error("enable must restore the package-guard hook in settings.json")
	}
	assertSharedFilesWellFormed(t, dir)
}

// TestLifecycle_EveryToolRoundTrip toggles every registered tool on a copy of
// an initialised project (disable then enable when it starts enabled, enable
// then disable otherwise) and checks each step leaves the shared files well
// formed and answers consistent. A refusal is only acceptable when it is the
// honest "generated none of its files" error.
func TestLifecycle_EveryToolRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("runs enable/disable for every registered tool")
	}
	template := initLifecycleProject(t)
	for _, tool := range toolreg.DefaultRegistry().All() {
		t.Run(tool.Name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.CopyFS(dir, os.DirFS(template)); err != nil {
				t.Fatalf("copying project: %v", err)
			}
			startEnabled := loadProjectAnswers(t, dir).EnabledTools[tool.Name]
			if !startEnabled {
				// Enabling is refused until the tool's prerequisites are on.
				for _, prereq := range tool.Prerequisites {
					if loadProjectAnswers(t, dir).EnabledTools[prereq] {
						continue
					}
					if out, err := enableTool(t, dir, prereq, "--force"); err != nil {
						t.Fatalf("enabling prerequisite %q: %v\n%s", prereq, err, out)
					}
				}
			}
			steps := []func(*testing.T, string, ...string) (string, error){enableTool, disableTool}
			if startEnabled {
				steps = []func(*testing.T, string, ...string) (string, error){disableTool, enableTool}
			}
			for i, step := range steps {
				out, err := step(t, dir, tool.Name, "--force")
				if err != nil {
					if strings.Contains(err.Error(), "generated none of its files") {
						t.Skipf("tool has nothing to generate in this project: %v", err)
					}
					t.Fatalf("step %d: %v\n%s", i, err, out)
				}
				assertSharedFilesWellFormed(t, dir)
			}
			if got := loadProjectAnswers(t, dir).EnabledTools[tool.Name]; got != startEnabled {
				t.Errorf("after the round trip enabled = %v, want %v", got, startEnabled)
			}
		})
	}
}

// TestLifecycle_SembleEnableDisable covers the MCP tool whose .mcp.json and
// CLAUDE.md entries once collided (F027: enable always failed marshaling
// Markdown as JSON). Enabling must turn semble on and emit its files.
func TestLifecycle_SembleEnableDisable(t *testing.T) {
	dir := initLifecycleProject(t)
	if loadProjectAnswers(t, dir).EnabledTools["semble"] {
		mustDisable(t, dir, "semble", "--force")
	}

	mustEnable(t, dir, "semble")
	assertSharedFilesWellFormed(t, dir)
	if !strings.Contains(readProjectFile(t, dir, ".mcp.json"), `"semble"`) {
		t.Errorf(".mcp.json missing the semble server:\n%s", readProjectFile(t, dir, ".mcp.json"))
	}
	if !loadProjectAnswers(t, dir).AgentTools.SembleEnabled {
		t.Error("enable must turn on agent_tools.semble_enabled")
	}

	mustDisable(t, dir, "semble", "--force")
	assertSharedFilesWellFormed(t, dir)
	if strings.Contains(readProjectFile(t, dir, ".mcp.json"), `"semble"`) {
		t.Error(".mcp.json still lists semble after disable")
	}
}

// TestLifecycle_SembleOptInOnly verifies semble (an unpinned uvx server) is
// only configured when opted in: a default init neither lists it in .mcp.json
// nor provisions uv for it, and explicit --agent-*=false opt-outs are kept
// instead of being re-enabled from catalog defaults.
func TestLifecycle_SembleOptInOnly(t *testing.T) {
	t.Run("default init", func(t *testing.T) {
		dir := initLifecycleProject(t)
		if mcp := readProjectFile(t, dir, ".mcp.json"); strings.Contains(mcp, `"semble"`) {
			t.Errorf(".mcp.json lists semble without opt-in:\n%s", mcp)
		}
		if nix := readProjectFile(t, dir, "devenv.nix"); strings.Contains(nix, "pkgs.uv") {
			t.Error("devenv.nix provisions uv for a Go project without semble")
		}
		a := loadProjectAnswers(t, dir)
		if a.AgentTools.SembleEnabled || a.EnabledTools["semble"] || slices.Contains(a.MCPServers, "semble") {
			t.Errorf("semble recorded as configured: agent_tools=%+v mcp=%v", a.AgentTools, a.MCPServers)
		}
	})
	t.Run("explicit opt-outs survive", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/lc\n\ngo 1.24\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if out, err := executeInitCmd(t, dir, "--yes", "--lang", "go", "--tier", "full",
			"--agent-semble=false", "--agent-postmortem=false", "--agent-version-sentinel=false"); err != nil {
			t.Fatalf("init: %v\n%s", err, out)
		}
		a := loadProjectAnswers(t, dir)
		if a.AgentTools.PostmortemEnabled || a.AgentTools.VersionSentinel || a.AgentTools.SembleEnabled {
			t.Errorf("explicit opt-outs were re-enabled: %+v", a.AgentTools)
		}
		if _, err := os.Stat(filepath.Join(dir, ".claude", "skills", "agent-postmortem")); err == nil {
			t.Error("agent-postmortem skill generated despite --agent-postmortem=false")
		}
	})
}

// TestLifecycle_TeardownRemovesGeneratedCore is the W161 regression: teardown
// used to strip only tool sections, leaving the whole generated devenv.nix,
// CLAUDE.md's generated block (advertising the skills it had just deleted),
// the devenv.nix sidecar, the answers copies and empty skill directories.
// User content in a shared file must survive.
func TestLifecycle_TeardownRemovesGeneratedCore(t *testing.T) {
	for _, withUserText := range []bool{false, true} {
		t.Run(fmt.Sprintf("user CLAUDE.md content=%v", withUserText), func(t *testing.T) {
			dir := initLifecycleProject(t)
			claudeMD := filepath.Join(dir, "CLAUDE.md")
			const userText = "# My project\n\nHand-written notes.\n"
			if withUserText {
				// As init leaves a pre-existing CLAUDE.md: the user's text,
				// then the appended generated block.
				generated, err := os.ReadFile(claudeMD)
				if err != nil {
					t.Fatal(err)
				}
				block := generated[strings.Index(string(generated), "<!-- BEGIN GENERATED SECTION"):]
				if err := os.WriteFile(claudeMD, append([]byte(userText+"\n"), block...), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			for _, rel := range []string{"devenv.nix.new", filepath.Join(".claude", ".qsdev-claude-answers.yaml")} {
				if err := os.WriteFile(filepath.Join(dir, rel), []byte("x\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			if out, err := runLifecycleCmd(t, dir, teardownCmd(), "--force"); err != nil {
				t.Fatalf("teardown: %v\n%s", err, out)
			}

			gone := []string{
				"devenv.nix", "devenv.nix.new", "devenv.yaml",
				filepath.Join(".claude", ".qsdev-claude-answers.yaml"),
				filepath.Join(".devenv", ".qsdev-answers.yaml"),
				filepath.Join(".claude", "skills"),
			}
			if !withUserText {
				gone = append(gone, "CLAUDE.md")
			}
			for _, rel := range gone {
				if _, err := os.Stat(filepath.Join(dir, rel)); err == nil {
					t.Errorf("%s survived teardown", rel)
				}
			}
			if withUserText {
				md, err := os.ReadFile(claudeMD)
				if err != nil {
					t.Fatalf("CLAUDE.md with user content was removed: %v", err)
				}
				if string(md) != userText {
					t.Errorf("CLAUDE.md after teardown = %q, want only the user's text", md)
				}
			}
		})
	}
}

// TestLifecycle_NoOutputErrorBlamesTierOnlyBelowFull is the F042 regression:
// a tool with nothing to generate (e.g. secretspec with no declared secrets)
// at the full tier must not be told to raise the tier it already has.
func TestLifecycle_NoOutputErrorBlamesTierOnlyBelowFull(t *testing.T) {
	t.Parallel()
	tool := &toolreg.Tool{
		Name:         "nothing-to-do",
		OwnedFiles:   []toolreg.FileOwnership{{Path: "nothing.toml", Ownership: toolreg.Exclusive}},
		GenerateFunc: func(types.WizardAnswers) ([]types.GeneratedFile, error) { return nil, nil },
	}
	tests := []struct {
		tier       string
		blamesTier bool
	}{
		{"standard", true},
		{"full", false},
	}
	for _, tt := range tests {
		t.Run(tt.tier, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			answers := types.WizardAnswers{Tier: tt.tier, ProjectRoot: root, EnabledTools: map[string]bool{}}
			_, err := planToolEnable(tool, tool.Name, root, answers, types.GeneratedState{Files: map[string]types.FileState{}}, false)
			if err == nil {
				t.Fatal("a tool that generates none of its files must not be enabled")
			}
			if got := strings.Contains(err.Error(), "higher tier"); got != tt.blamesTier {
				t.Errorf("tier %s: blames tier = %v, want %v: %v", tt.tier, got, tt.blamesTier, err)
			}
			if !strings.Contains(err.Error(), "nothing.toml") {
				t.Errorf("error should name the missing files: %v", err)
			}
		})
	}
}

// TestLifecycle_EnableRefusesToClobberUserFiles is the F518 regression: enable
// must not overwrite a pre-existing file qsdev did not generate (unless
// --force), and must never write through a symlink out of the project.
func TestLifecycle_EnableRefusesToClobberUserFiles(t *testing.T) {
	dir := initLifecycleProject(t)
	const tmplPath = ".github/pull_request_template.md"
	mustDisable(t, dir, "pr-templates", "--force")

	userFile := filepath.Join(dir, tmplPath)
	if err := os.MkdirAll(filepath.Dir(userFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(userFile, []byte("MY OWN TEMPLATE\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := enableTool(t, dir, "pr-templates"); err == nil || !strings.Contains(err.Error(), "refusing to overwrite") {
		t.Fatalf("enable over a user file: want refusal, got %v", err)
	}
	if got := readProjectFile(t, dir, tmplPath); got != "MY OWN TEMPLATE\n" {
		t.Fatalf("user template was modified: %q", got)
	}
	if loadProjectAnswers(t, dir).EnabledTools["pr-templates"] {
		t.Error("a refused enable must not mark the tool enabled")
	}

	mustEnable(t, dir, "pr-templates", "--force")
	if got := readProjectFile(t, dir, tmplPath); got == "MY OWN TEMPLATE\n" {
		t.Error("--force should overwrite the existing template")
	}

	if runtime.GOOS == "windows" {
		return
	}
	// Symlink escape: .github points outside the project.
	mustDisable(t, dir, "pr-templates", "--force")
	outside := t.TempDir()
	victim := filepath.Join(outside, "pull_request_template.md")
	if err := os.WriteFile(victim, []byte("OUTSIDE\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(dir, ".github")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, ".github")); err != nil {
		t.Fatal(err)
	}
	if _, err := enableTool(t, dir, "pr-templates", "--force"); err == nil || !strings.Contains(err.Error(), "outside the project") {
		t.Fatalf("enable through an escaping symlink: want refusal, got %v", err)
	}
	if data, _ := os.ReadFile(victim); string(data) != "OUTSIDE\n" {
		t.Errorf("file outside the project was modified: %q", data)
	}
}

// TestExecuteUpdatePlan_RefusesSymlinkEscape covers the update half of F518:
// regenerated files must not be written through a symlink out of the project.
func TestExecuteUpdatePlan_RefusesSymlinkEscape(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires elevated privileges on Windows")
	}
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, ".github")); err != nil {
		t.Fatal(err)
	}
	plan := UpdatePlan{Files: []FileUpdatePlan{{
		Path:       ".github/pull_request_template.md",
		Action:     UpdateActionCreate,
		NewContent: []byte("generated\n"),
	}}}
	if _, err := executeUpdatePlan(plan, root, UpdateOptions{}); err == nil || !strings.Contains(err.Error(), "escapes project root") {
		t.Fatalf("want containment refusal, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(outside, "pull_request_template.md")); !os.IsNotExist(err) {
		t.Errorf("file was written outside the project (stat err=%v)", err)
	}
}

// TestLifecycle_DisableKeepsUntrackedFiles checks disable never deletes a
// declared tool file qsdev did not generate unless --force is given.
func TestLifecycle_DisableKeepsUntrackedFiles(t *testing.T) {
	dir := initLifecycleProject(t)
	mustEnable(t, dir, "changelog")

	// Forget cliff.toml in state, as if the user had written it themselves.
	st := loadProjectState(t, dir)
	delete(st.Files, "cliff.toml")
	if err := state.SaveStateToFile(filepath.Join(dir, stateFilePath()), st); err != nil {
		t.Fatal(err)
	}

	mustDisable(t, dir, "changelog")
	if _, err := os.Stat(filepath.Join(dir, "cliff.toml")); err != nil {
		t.Errorf("untracked cliff.toml must be left in place: %v", err)
	}
}
