package claudecode

import (
	"io/fs"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/check"
)

// unexpectedDenyConflicts runs the production deny-conflict check
// (internal/check) over denyRules and skills, returning the failing results.
func unexpectedDenyConflicts(t *testing.T, denyRules []string, skills []SkillDefinition) []check.CheckResult {
	t.Helper()
	ctx := check.CheckContext{DenyRules: denyRules, ExpectedConflictKeys: ExpectedConflicts()}
	for _, s := range skills {
		ctx.SkillOps = append(ctx.SkillOps, check.SkillOps{Name: s.Name, AllowedTools: s.AllowedTools, PreApproved: s.PreApproved})
	}
	return failedResults(check.CheckDenyRuleConflicts(ctx))
}

// askBypasses runs the production pre-approval check (internal/check) over
// askRules and skills, returning the failing results.
func askBypasses(t *testing.T, askRules []string, skills []SkillDefinition) []check.CheckResult {
	t.Helper()
	ctx := check.CheckContext{AskRules: askRules}
	for _, s := range skills {
		ctx.SkillOps = append(ctx.SkillOps, check.SkillOps{Name: s.Name, AllowedTools: s.AllowedTools, PreApproved: s.PreApproved})
	}
	return failedResults(check.CheckSkillPreApprovals(ctx))
}

func failedResults(results []check.CheckResult) []check.CheckResult {
	var failed []check.CheckResult
	for _, r := range results {
		if r.Status == check.StatusFail {
			failed = append(failed, r)
		}
	}
	return failed
}

func TestDenyRuleConflicts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		denyRules []string
		skills    []SkillDefinition
		want      int
	}{
		{
			name:      "safe deny rules do not overlap safe operations",
			denyRules: []string{"Bash(npm install *)", "Bash(pip install *)", "Read(./.env)"},
			skills: []SkillDefinition{
				{Name: "review-pr", AllowedTools: []string{"Bash(git *)", "Bash(gh *)"}},
				{Name: "add-tests", AllowedTools: []string{"Bash(npm test *)", "Bash(go test *)"}},
			},
		},
		{
			name:      "overly broad deny rule blocks a skill operation",
			denyRules: []string{"Bash(npm *)"},
			skills:    []SkillDefinition{{Name: "add-tests", AllowedTools: []string{"Bash(npm test *)"}}},
			want:      1,
		},
		{
			name:      "package installs are in ask, not deny",
			denyRules: AllBaseDenyRules(),
			skills: []SkillDefinition{{Name: "upgrade-dep", AllowedTools: []string{
				"Bash(npm install *)", "Bash(npm uninstall *)", "Bash(pip install *)", "Bash(cargo install *)",
			}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := unexpectedDenyConflicts(t, tt.denyRules, tt.skills)
			if len(got) != tt.want {
				t.Errorf("got %d conflicts, want %d: %+v", len(got), tt.want, got)
			}
		})
	}
}

// TestBuiltinSkillDefinitions_ParseWithoutErrors guards the embedded templates:
// every skill and subagent frontmatter must parse.
func TestBuiltinSkillDefinitions_ParseWithoutErrors(t *testing.T) {
	t.Parallel()
	defs, err := loadBuiltinSkillDefinitions()
	if err != nil {
		t.Fatalf("loadBuiltinSkillDefinitions: %v", err)
	}
	if len(defs) == 0 {
		t.Fatal("no skill definitions parsed from the templates")
	}
}

// TestBuiltinSkillDefinitions_MatchTemplates pins F101: the definitions come
// from the deployed templates' frontmatter, not a hand-maintained list that
// drifts (missing skills, fictional skills, wrong operations).
func TestBuiltinSkillDefinitions_MatchTemplates(t *testing.T) {
	t.Parallel()
	byName := make(map[string][]string)
	for _, d := range BuiltinSkillDefinitions() {
		byName[d.Name] = d.AllowedTools
	}

	tests := []struct {
		skill string
		want  []string
	}{
		{"qsdev-add-dep", []string{"Bash(qsdev *)", "Bash(npm view *)", "Bash(pip index *)", "Read"}},
		{"qsdev-init", []string{"Bash(qsdev *)", "Read", "Grep", "Glob"}},
		{"upgrade-dep", []string{"Bash(git diff *)", "Write", "Edit"}},
		{"write-adr", []string{"Write", "Edit", "Bash(git log *)"}},
		{"lookup-docs", []string{"Bash(qsdev *)", "mcp__local-docs-devdocs(*)", "mcp__context7(*)"}},
		{"security-reviewer", []string{"Read", "Grep", "Glob", "Bash"}},
		// semble-search is a subagent: its template uses the subagent
		// "tools:" key (F085), which lists tool names, not Bash patterns.
		{"semble-search", []string{"Bash", "Read", "Grep", "Glob"}},
	}
	for _, tt := range tests {
		t.Run(tt.skill, func(t *testing.T) {
			t.Parallel()
			got, ok := byName[tt.skill]
			if !ok {
				t.Fatalf("skill %q missing from BuiltinSkillDefinitions", tt.skill)
			}
			for _, op := range tt.want {
				if !slices.Contains(got, op) {
					t.Errorf("skill %q operations %v missing %q", tt.skill, got, op)
				}
			}
		})
	}

	if _, ok := byName["container-migrate"]; ok {
		t.Error("container-migrate has no template but is listed")
	}
}

// TestBuiltinSkillDefinitions_CoverEveryDeclaringTemplate walks the skill
// directories and checks that every SKILL.md declaring allowed-tools yields a
// definition.
func TestBuiltinSkillDefinitions_CoverEveryDeclaringTemplate(t *testing.T) {
	t.Parallel()
	names := make(map[string]bool)
	for _, d := range BuiltinSkillDefinitions() {
		names[d.Name] = true
	}
	err := fs.WalkDir(templateFS, "templates/skills", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.Contains(p, "/SKILL.md") {
			return err
		}
		content, err := templateFS.ReadFile(p)
		if err != nil {
			return err
		}
		if strings.Contains(string(content), "\nallowed-tools:") && !names[templateSkillName(p)] {
			t.Errorf("%s declares allowed-tools but produced no definition", p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestBuiltinSkillDefinitions_NoUnexpectedConflicts is the integration check:
// the real deny rules must not block any operation the real templates request.
func TestBuiltinSkillDefinitions_NoUnexpectedConflicts(t *testing.T) {
	t.Parallel()
	for _, c := range unexpectedDenyConflicts(t, AllBaseDenyRules(), BuiltinSkillDefinitions()) {
		t.Errorf("unexpected conflict: %s", c.Message)
	}
}

// TestNoSkillGrantsBareWildcard pins U15-10: no skill's allowed-tools
// pre-approves a command the catalog gates behind ask (package installs,
// code execution), so a model-invoked skill cannot skip the prompt and the
// package-guard review it leads to.
func TestNoSkillGrantsBareWildcard(t *testing.T) {
	t.Parallel()
	askRules := AllBaseAskRules()
	if len(askRules) == 0 {
		t.Fatal("AllBaseAskRules() is empty")
	}
	for _, c := range askBypasses(t, askRules, BuiltinSkillDefinitions()) {
		t.Errorf("skill pre-approves an ask-gated command: %s", c.Message)
	}

	// The guard must catch every shape that would bypass ask.
	for _, rule := range []string{"Bash(*)", "Bash", "Bash(npm *)", "Bash(go *)", "Bash(npm i*)", "Bash(cargo *)", "Bash(* install *)"} {
		t.Run(rule, func(t *testing.T) {
			t.Parallel()
			skill := SkillDefinition{Name: "probe", AllowedTools: []string{rule}, PreApproved: []string{rule}}
			if len(askBypasses(t, askRules, []SkillDefinition{skill})) == 0 {
				t.Errorf("allowed-tools %s not flagged as bypassing a catalog ask rule", rule)
			}
		})
	}
}

// TestSkillPreApproved_AllowedToolsOnly pins that only a skill's
// allowed-tools are pre-approvals: a subagent's tools list is an
// availability allowlist, so its bare Bash grants nothing.
func TestSkillPreApproved_AllowedToolsOnly(t *testing.T) {
	t.Parallel()
	byName := make(map[string]SkillDefinition)
	for _, d := range BuiltinSkillDefinitions() {
		byName[d.Name] = d
	}
	if d := byName["security-reviewer"]; !slices.Contains(d.AllowedTools, "Bash") || len(d.PreApproved) != 0 {
		t.Errorf("security-reviewer agent: AllowedTools %v, PreApproved %v; want Bash available and nothing pre-approved", d.AllowedTools, d.PreApproved)
	}
	if d := byName["qsdev-add-dep"]; !slices.Equal(d.PreApproved, d.AllowedTools) || len(d.PreApproved) == 0 {
		t.Errorf("qsdev-add-dep skill: PreApproved %v, want its allowed-tools %v", d.PreApproved, d.AllowedTools)
	}
}

// TestNarrowedSkillTools pins the least-privilege allowed-tools of the
// skills that used to pre-approve wildcards.
func TestNarrowedSkillTools(t *testing.T) {
	t.Parallel()
	byName := make(map[string][]string)
	for _, d := range BuiltinSkillDefinitions() {
		byName[d.Name] = d.PreApproved
	}
	readOnlyGit := []string{"Bash(git log *)", "Bash(git diff *)", "Bash(git show *)", "Bash(git status *)", "Bash(git blame *)"}

	want := []string{"Bash(qsdev *)", "Bash(npm view *)", "Bash(npm ls *)", "Bash(pnpm why *)", "Bash(cargo search *)", "Bash(cargo tree *)", "Bash(go list *)", "Bash(pip index *)", "Read", "Grep", "Glob"}
	if got := byName["qsdev-add-dep"]; !slices.Equal(got, want) {
		t.Errorf("qsdev-add-dep allowed-tools = %v, want %v", got, want)
	}
	for _, skill := range []string{"add-tests", "upgrade-dep", "incident-debug"} {
		for _, tool := range byName[skill] {
			if strings.HasPrefix(tool, "Bash") && !slices.Contains(readOnlyGit, tool) {
				t.Errorf("%s allowed-tools grants %s; want only read-only git shapes", skill, tool)
			}
		}
	}
	banned := map[string][]string{
		"review-pr":      {"Bash(gh *)"},
		"write-adr":      {"Bash(find *)", "Bash(ls *)"},
		"migration-plan": {"Bash(find *)", "Bash(ls *)"},
	}
	for skill, tools := range banned {
		if len(byName[skill]) == 0 {
			t.Errorf("skill %q missing", skill)
		}
		for _, tool := range tools {
			if slices.Contains(byName[skill], tool) {
				t.Errorf("%s allowed-tools still grants %s", skill, tool)
			}
		}
	}
}

// TestBuiltinSkillDefinitions_DetectNewDenyConflict pins the F101 failure
// scenario: a deny rule that would block qsdev-add-dep must be reported.
func TestBuiltinSkillDefinitions_DetectNewDenyConflict(t *testing.T) {
	t.Parallel()
	conflicts := unexpectedDenyConflicts(t, []string{"Bash(npm *)"}, BuiltinSkillDefinitions())
	found := false
	for _, c := range conflicts {
		if strings.Contains(c.Message, `"qsdev-add-dep"`) {
			found = true
		}
	}
	if !found {
		t.Errorf("deny rule Bash(npm *) not reported as blocking qsdev-add-dep: %+v", conflicts)
	}
}

func TestExpectedConflicts_Empty(t *testing.T) {
	t.Parallel()
	// Package installs are now in ask, not deny. No expected conflicts remain.
	if ec := ExpectedConflicts(); len(ec) != 0 {
		t.Fatalf("ExpectedConflicts should be empty (package installs moved to ask), got %d entries", len(ec))
	}
}

func TestSplitToolList(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in   string
		want []string
	}{
		{"Bash(git log *) Read Grep", []string{"Bash(git log *)", "Read", "Grep"}},
		{"Read, Grep, Glob, Bash", []string{"Read", "Grep", "Glob", "Bash"}},
		{"  Bash(qsdev *)  mcp__x(*) ", []string{"Bash(qsdev *)", "mcp__x(*)"}},
		{"", nil},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			if got := splitToolList(tt.in); !slices.Equal(got, tt.want) {
				t.Errorf("splitToolList(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
