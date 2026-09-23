package devinit

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// saveProjectState writes the devinit state file of the project in dir.
func saveProjectState(t *testing.T, dir string, st types.GeneratedState) {
	t.Helper()
	if err := state.SaveStateToFile(filepath.Join(dir, stateFilePath()), st); err != nil {
		t.Fatalf("saving state: %v", err)
	}
}

// writeTrackedFile writes content to dir/rel with mode 0o644 and records it
// in st as an unmodified generated file.
func writeTrackedFile(t *testing.T, st *types.GeneratedState, dir, rel, content, owner string) {
	t.Helper()
	abs := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(abs, 0o644); err != nil {
		t.Fatal(err)
	}
	st.Files[rel] = types.FileState{Hash: state.ComputeHash([]byte(content)), Mode: 0o644, Owner: owner}
}

func TestScopeFromAnswers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		opts  InitOptions
		saved string
		want  generationScope
	}{
		{name: "default", want: generationScope{}},
		{name: "claude-only flag", opts: InitOptions{ClaudeOnly: true}, want: generationScope{ClaudeOnly: true}},
		{name: "devenv-only flag", opts: InitOptions{DevenvOnly: true}, want: generationScope{DevenvOnly: true}},
		{name: "persisted claude-only without flag", saved: mergeModeClaudeOnly, want: generationScope{ClaudeOnly: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			answers := types.WizardAnswers{MergeMode: tt.saved}
			applyScopeFlags(tt.opts, &answers)
			if got := scopeFromAnswers(answers); got != tt.want {
				t.Errorf("scopeFromAnswers() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// TestIntegration_ClaudeOnly_UpdateLeavesDevenvAlone reproduces the
// claude-only data-loss bug: the scope must be persisted at init so that a
// later update never generates (and overwrites) the user's devenv files.
func TestIntegration_ClaudeOnly_UpdateLeavesDevenvAlone(t *testing.T) {
	dir := createGoFixture(t)
	const userNix = "# hand-maintained\n{ pkgs, ... }: { packages = [ pkgs.jq ]; }\n"
	if err := os.WriteFile(filepath.Join(dir, "devenv.nix"), []byte(userNix), 0o644); err != nil {
		t.Fatal(err)
	}

	if out, err := executeInitCmd(t, dir, "--claude-only", "--yes", "--force", "--lang", "go"); err != nil {
		t.Fatalf("init --claude-only failed: %v\n%s", err, out)
	}
	answers, err := loadAnswers(dir)
	if err != nil {
		t.Fatal(err)
	}
	if answers.MergeMode != mergeModeClaudeOnly {
		t.Fatalf("persisted merge_mode = %q, want %q", answers.MergeMode, mergeModeClaudeOnly)
	}

	if out, err := executeInitCmd(t, dir, "--update"); err != nil {
		t.Fatalf("update failed: %v\n%s", err, out)
	}

	if got := readFileContent(t, dir, "devenv.nix"); got != userNix {
		t.Errorf("update replaced the user's devenv.nix:\n%s", got)
	}
	requireFileNotExists(t, dir, "devenv.yaml")
	requireFileNotExists(t, dir, ".envrc")
}

func TestBuildUpdatePlan_UntrackedFiles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		strategy   types.MergeStrategy
		onDisk     bool
		diskData   string // on-disk content; defaults to user content
		opts       UpdateOptions
		wantAction UpdateAction
		wantReason string
	}{
		{name: "missing file is created", strategy: types.Overwrite, wantAction: UpdateActionCreate, wantReason: "new file"},
		{name: "existing file is not overwritten", strategy: types.Overwrite, onDisk: true, wantAction: UpdateActionSkip, wantReason: "use --force to overwrite"},
		{
			name: "skip reason names the caller's flag", strategy: types.ManualMerge, onDisk: true,
			opts:       UpdateOptions{OverwriteFlag: "--overwrite-modified"},
			wantAction: UpdateActionSkip, wantReason: "use --overwrite-modified to overwrite",
		},
		{
			name: "existing file identical to generated content is tracked", strategy: types.Overwrite, onDisk: true, diskData: "generated",
			wantAction: UpdateActionRegenerate, wantReason: "tracking",
		},
		{name: "existing file with force is overwritten", strategy: types.Overwrite, onDisk: true, opts: UpdateOptions{Force: true}, wantAction: UpdateActionRegenerate},
		{name: "existing three-way file is merged", strategy: types.ThreeWayMerge, onDisk: true, wantAction: UpdateActionMerge},
		{name: "existing section-marker file is merged", strategy: types.SectionMarker, onDisk: true, wantAction: UpdateActionMerge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if tt.onDisk {
				data := tt.diskData
				if data == "" {
					data = "user content"
				}
				if err := os.WriteFile(filepath.Join(dir, "cfg"), []byte(data), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			files := []types.GeneratedFile{{Path: "cfg", Content: []byte("generated"), Strategy: tt.strategy}}
			plan := buildUpdatePlan(files, map[string]state.FileStatus{}, types.GeneratedState{}, dir, tt.opts)

			fp := plan.Files[0]
			if fp.Action != tt.wantAction {
				t.Errorf("action = %s, want %s (%s)", updateActionString(fp.Action), updateActionString(tt.wantAction), fp.Reason)
			}
			if !strings.Contains(fp.Reason, tt.wantReason) {
				t.Errorf("reason %q does not contain %q", fp.Reason, tt.wantReason)
			}
			if fp.OldContent != nil {
				t.Errorf("untracked file has no recorded base, got OldContent %q", fp.OldContent)
			}
		})
	}
}

// TestIntegration_Update_UntrackedExistingFileKept verifies update no longer
// overwrites a file that exists on disk but is missing from state.
func TestIntegration_Update_UntrackedExistingFileKept(t *testing.T) {
	dir := t.TempDir()
	if _, err := executeInitCmd(t, dir, "--lang", "go", "--yes"); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	st := loadProjectState(t, dir)
	delete(st.Files, "devenv.yaml")
	saveProjectState(t, dir, st)
	const userYAML = "# written by the user\ninputs: {}\n"
	if err := os.WriteFile(filepath.Join(dir, "devenv.yaml"), []byte(userYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := executeInitCmd(t, dir, "--update")
	if err != nil {
		t.Fatalf("update failed: %v\n%s", err, out)
	}
	if got := readFileContent(t, dir, "devenv.yaml"); got != userYAML {
		t.Errorf("untracked devenv.yaml was overwritten:\n%s", got)
	}
	if !strings.Contains(out, "exists but not tracked") {
		t.Errorf("plan does not explain the skip:\n%s", out)
	}
}

func TestExecuteUpdatePlan_MergeFailureIsReported(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	settings := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(settings, []byte(`{"permissions": {"deny": []},}`), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := UpdatePlan{Files: []FileUpdatePlan{
		{Path: "settings.json", Strategy: types.ThreeWayMerge, Action: UpdateActionMerge, NewContent: []byte(`{"permissions": {"deny": ["x"]}}`), OldContent: []byte(`{}`)},
		{Path: "other.txt", Strategy: types.Overwrite, Action: UpdateActionRegenerate, NewContent: []byte("ok")},
	}}

	out, err := executeUpdatePlan(plan, dir, UpdateOptions{})
	if err != nil {
		t.Fatalf("a merge failure must not abort execution: %v", err)
	}
	if len(out.failures) != 1 || out.failures[0].Path != "settings.json" {
		t.Fatalf("failures = %+v, want settings.json", out.failures)
	}
	if len(out.written) != 1 || out.written[0].Path != "other.txt" {
		t.Errorf("written = %+v, want only other.txt", out.written)
	}

	var buf bytes.Buffer
	printUpdateSummary(&buf, plan, out)
	if got := buf.String(); !strings.Contains(got, "1 updated, 0 skipped, 0 removed, 1 failed") {
		t.Errorf("summary counts the failed merge as updated: %q", got)
	}
}

// TestIntegration_Update_MergeFailureFails verifies a failed settings.json
// merge makes update fail instead of reporting success.
func TestIntegration_Update_MergeFailureFails(t *testing.T) {
	dir := t.TempDir()
	if _, err := executeInitCmd(t, dir, "--lang", "go", "--yes"); err != nil {
		t.Fatalf("init failed: %v", err)
	}
	const broken = `{"permissions": {"deny": []},}`
	if err := os.WriteFile(filepath.Join(dir, ".claude", "settings.json"), []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := executeInitCmd(t, dir, "--update")
	if err == nil {
		t.Fatalf("update succeeded despite a failed settings.json merge:\n%s", out)
	}
	if !strings.Contains(out, ".claude/settings.json") || !strings.Contains(out, "1 failed") {
		t.Errorf("output does not report the failed merge:\n%s", out)
	}
	if got := readFileContent(t, dir, ".claude/settings.json"); got != broken {
		t.Errorf("settings.json changed despite failed merge: %s", got)
	}
}

// TestUpdate_PartialFailureRecordsWrittenFiles verifies files rewritten before
// a write failure are still recorded in state, so the next update does not
// treat them as user-modified.
func TestUpdate_PartialFailureRecordsWrittenFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// A regular file where a directory is needed makes the second write fail.
	if err := os.WriteFile(filepath.Join(dir, "blocked"), []byte("file"), 0o644); err != nil {
		t.Fatal(err)
	}
	existing := types.GeneratedState{Files: map[string]types.FileState{
		"a.txt":         {Hash: state.ComputeHash([]byte("old")), Mode: 0o644},
		"blocked/b.txt": {Hash: state.ComputeHash([]byte("old")), Mode: 0o644},
	}}
	plan := UpdatePlan{Files: []FileUpdatePlan{
		{Path: "a.txt", Action: UpdateActionRegenerate, NewContent: []byte("new"), NewMode: 0o644},
		{Path: "blocked/b.txt", Action: UpdateActionRegenerate, NewContent: []byte("new"), NewMode: 0o644},
	}}

	out, execErr := executeUpdatePlan(plan, dir, UpdateOptions{})
	if execErr == nil {
		t.Fatal("expected a write failure")
	}
	stateFile := filepath.Join(dir, stateFilePath())
	if err := saveUpdateResults(plan, out, existing, types.WizardAnswers{}, accumulatorResult{}, stateFile, dir); err != nil {
		t.Fatal(err)
	}

	saved := loadProjectState(t, dir)
	if got := state.CheckModified(saved, dir)["a.txt"].Status; got != types.Unmodified {
		t.Errorf("a.txt status after partial update = %v, want unmodified", got)
	}
	if saved.Files["blocked/b.txt"].Hash != existing.Files["blocked/b.txt"].Hash {
		t.Error("unwritten file lost its previous state entry")
	}
}

func TestPlanOrphans(t *testing.T) {
	t.Parallel()

	stored := types.GeneratedState{Files: map[string]types.FileState{
		"kept.md":                  {},
		"stale.md":                 {},
		"edited.md":                {},
		"gone.md":                  {},
		"tool.yml":                 {Owner: "semgrep"},
		"disabled-tool.yml":        {Owner: "old-tool"},
		"../outside.md":            {},
		branding.Get().LocalConfig: {},
	}}
	modStatus := map[string]state.FileStatus{
		"kept.md":                  {Status: types.Unmodified},
		"stale.md":                 {Status: types.Unmodified},
		"edited.md":                {Status: types.Modified},
		"gone.md":                  {Status: types.Deleted},
		"tool.yml":                 {Status: types.Unmodified},
		"disabled-tool.yml":        {Status: types.Unmodified},
		"../outside.md":            {Status: types.Unmodified},
		branding.Get().LocalConfig: {Status: types.Unmodified},
	}
	answers := types.WizardAnswers{EnabledTools: map[string]bool{"semgrep": true}}
	newFiles := []types.GeneratedFile{{Path: "kept.md"}}

	got := map[string]UpdateAction{}
	for _, fp := range planOrphans(stored, newFiles, modStatus, answers) {
		got[fp.Path] = fp.Action
	}
	want := map[string]UpdateAction{
		"stale.md":          UpdateActionRemove,
		"edited.md":         UpdateActionUntrack,
		"gone.md":           UpdateActionUntrack,
		"disabled-tool.yml": UpdateActionRemove,
		"../outside.md":     UpdateActionUntrack,
	}
	if len(got) != len(want) {
		keys := make([]string, 0, len(got))
		for k := range got {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		t.Fatalf("planned orphans = %v, want %d entries", keys, len(want))
	}
	for path, action := range want {
		if got[path] != action {
			t.Errorf("%s: action = %s, want %s", path, updateActionString(got[path]), updateActionString(action))
		}
	}
}

// TestIntegration_Update_RemovesOrphans verifies files the generators no
// longer produce are cleaned up: unmodified ones are removed, modified ones
// are left in place, and both stop being tracked.
func TestIntegration_Update_RemovesOrphans(t *testing.T) {
	dir := t.TempDir()
	if _, err := executeInitCmd(t, dir, "--lang", "go", "--yes"); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	st := loadProjectState(t, dir)
	writeTrackedFile(t, &st, dir, ".claude/rules/retired-conventions.md", "old rule\n", "")
	writeTrackedFile(t, &st, dir, ".claude/rules/edited-retired.md", "old rule\n", "")
	saveProjectState(t, dir, st)
	edited := filepath.Join(dir, ".claude/rules/edited-retired.md")
	if err := os.WriteFile(edited, []byte("user edit\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := executeInitCmd(t, dir, "--update")
	if err != nil {
		t.Fatalf("update failed: %v\n%s", err, out)
	}

	requireFileNotExists(t, dir, ".claude/rules/retired-conventions.md")
	requireFileExists(t, dir, ".claude/rules/edited-retired.md")
	after := loadProjectState(t, dir)
	for _, p := range []string{".claude/rules/retired-conventions.md", ".claude/rules/edited-retired.md"} {
		if _, tracked := after.Files[p]; tracked {
			t.Errorf("%s is still tracked after update", p)
		}
	}
	if !strings.Contains(out, "1 removed") {
		t.Errorf("summary does not report the removal:\n%s", out)
	}
}

// TestIntegration_TemplateVersionsRecorded verifies init stamps the template
// versions, so the first update does not report a spurious template change,
// and that devenv-only projects never print the Claude template summary.
func TestIntegration_TemplateVersionsRecorded(t *testing.T) {
	tests := []struct {
		name       string
		initArgs   []string
		wantStamps bool
	}{
		{name: "claude project", initArgs: []string{"--lang", "go", "--yes"}, wantStamps: true},
		{name: "devenv-only project", initArgs: []string{"--lang", "go", "--yes", "--devenv-only"}, wantStamps: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if out, err := executeInitCmd(t, dir, tt.initArgs...); err != nil {
				t.Fatalf("init failed: %v\n%s", err, out)
			}

			st := loadProjectState(t, dir)
			if tt.wantStamps {
				if st.TemplateVersion != claudecode.ComputeTemplateVersion() ||
					st.SkillLibraryVersion != claudecode.ComputeSkillLibraryVersion() {
					t.Errorf("init did not record template versions: %q / %q", st.TemplateVersion, st.SkillLibraryVersion)
				}
			}

			out, err := executeInitCmd(t, dir, "--update")
			if err != nil {
				t.Fatalf("update failed: %v\n%s", err, out)
			}
			if strings.Contains(out, "All files up to date.") || strings.Contains(out, "skill(s)") || strings.Contains(out, "rule(s)") || strings.Contains(out, "template(s)") {
				t.Errorf("first update printed a spurious template-change summary:\n%s", out)
			}
		})
	}
}
