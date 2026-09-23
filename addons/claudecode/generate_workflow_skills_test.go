package claudecode_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// ---------------------------------------------------------------------------
// loadConsultingSkillManifest tests
// ---------------------------------------------------------------------------

func TestLoadConsultingSkillManifest_Valid(t *testing.T) {
	manifest, err := claudecode.ExportLoadConsultingSkillManifest()
	if err != nil {
		t.Fatalf("loadConsultingSkillManifest returned error: %v", err)
	}

	if len(manifest.Skills) != 8 {
		t.Errorf("expected 8 skills in consulting manifest, got %d", len(manifest.Skills))
	}

	for i, s := range manifest.Skills {
		if s.Name == "" {
			t.Errorf("skill %d has empty name", i)
		}
		if s.Description == "" {
			t.Errorf("skill %d (%s) has empty description", i, s.Name)
		}
	}
}

// ---------------------------------------------------------------------------
// deployWorkflowSkills tests
// ---------------------------------------------------------------------------

func TestDeployWorkflowSkills_SkipsWhenDisabled(t *testing.T) {
	// EnabledTools is nil — all consulting workflow skills should be skipped.
	answers := types.WizardAnswers{
		EnabledTools: nil,
	}

	files, err := claudecode.ExportDeployWorkflowSkills(answers, ecosystem.NewRegistry())
	if err != nil {
		t.Fatalf("deployWorkflowSkills returned error: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("expected 0 files when EnabledTools is nil, got %d", len(files))
	}
}

func TestDeployWorkflowSkills_DeploysEnabled(t *testing.T) {
	answers := types.WizardAnswers{
		EnabledTools: map[string]bool{
			"consulting-workflow-review-pr": true,
			"consulting-workflow-write-adr": true,
			"consulting-workflow-add-tests": false, // explicitly disabled
		},
	}

	files, err := claudecode.ExportDeployWorkflowSkills(answers, ecosystem.NewRegistry())
	if err != nil {
		t.Fatalf("deployWorkflowSkills returned error: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}

	paths := make(map[string]bool)
	for _, f := range files {
		paths[f.Path] = true
	}
	if !paths[".claude/skills/review-pr/SKILL.md"] {
		t.Error("missing .claude/skills/review-pr/SKILL.md")
	}
	if !paths[".claude/skills/write-adr/SKILL.md"] {
		t.Error("missing .claude/skills/write-adr/SKILL.md")
	}
}

func TestDeployWorkflowSkills_AllHaveDisableModelInvocation(t *testing.T) {
	// Enable all skills and verify each has disable-model-invocation in frontmatter.
	manifest, err := claudecode.ExportLoadConsultingSkillManifest()
	if err != nil {
		t.Fatalf("loadConsultingSkillManifest returned error: %v", err)
	}

	enabledTools := make(map[string]bool, len(manifest.Skills))
	for _, s := range manifest.Skills {
		enabledTools["consulting-workflow-"+s.Name] = true
	}

	answers := types.WizardAnswers{
		EnabledTools: enabledTools,
	}

	files, err := claudecode.ExportDeployWorkflowSkills(answers, ecosystem.NewRegistry())
	if err != nil {
		t.Fatalf("deployWorkflowSkills returned error: %v", err)
	}

	for _, f := range files {
		content := string(f.Content)
		if !strings.Contains(content, "disable-model-invocation: true") {
			t.Errorf("skill file %q does not contain 'disable-model-invocation: true'", f.Path)
		}
	}
}

func TestDeployWorkflowSkills_DirectoryFormat(t *testing.T) {
	// Verify paths are in .claude/skills/{name}/SKILL.md format.
	answers := types.WizardAnswers{
		EnabledTools: map[string]bool{
			"consulting-workflow-incident-debug": true,
			"consulting-workflow-migration-plan": true,
		},
	}

	files, err := claudecode.ExportDeployWorkflowSkills(answers, ecosystem.NewRegistry())
	if err != nil {
		t.Fatalf("deployWorkflowSkills returned error: %v", err)
	}

	for _, f := range files {
		if !strings.HasPrefix(f.Path, ".claude/skills/") {
			t.Errorf("path %q does not start with .claude/skills/", f.Path)
		}
		if !strings.HasSuffix(f.Path, "/SKILL.md") {
			t.Errorf("path %q does not end with /SKILL.md", f.Path)
		}
		if f.Mode != 0o644 {
			t.Errorf("file %q has mode %o, want 0o644", f.Path, f.Mode)
		}
		if f.Strategy != types.LibraryManaged {
			t.Errorf("file %q has strategy %v, want LibraryManaged", f.Path, f.Strategy)
		}
	}
}

func TestAvailableConsultingSkillNames(t *testing.T) {
	names := claudecode.AvailableConsultingSkillNames()

	if len(names) != 8 {
		t.Fatalf("expected 8 consulting skill names, got %d: %v", len(names), names)
	}

	// Verify all expected names are present (sorted).
	expected := []string{
		"add-tests",
		"handoff-doc",
		"incident-debug",
		"migration-plan",
		"onboard-me",
		"review-pr",
		"upgrade-dep",
		"write-adr",
	}
	for i, want := range expected {
		if names[i] != want {
			t.Errorf("name[%d] = %q, want %q", i, names[i], want)
		}
	}
}

func TestDeployWorkflowSkills_OwnerSet(t *testing.T) {
	answers := types.WizardAnswers{
		EnabledTools: map[string]bool{
			"consulting-workflow-handoff-doc": true,
		},
	}

	files, err := claudecode.ExportDeployWorkflowSkills(answers, ecosystem.NewRegistry())
	if err != nil {
		t.Fatalf("deployWorkflowSkills returned error: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	if files[0].Owner != "consulting-workflow-handoff-doc" {
		t.Errorf("Owner = %q, want %q", files[0].Owner, "consulting-workflow-handoff-doc")
	}
}

// TestWorkflowSkills_AgentPrerequisites verifies that a consulting workflow
// skill which runs in a named subagent (front-matter "agent: <name>") declares
// that agent's tool as a prerequisite, so it cannot be enabled (and its skill
// deployed) without the agent file it delegates to.
func TestWorkflowSkills_AgentPrerequisites(t *testing.T) {
	t.Parallel()
	reg := toolreg.DefaultRegistry()
	agents := make(map[string]bool)
	for _, a := range claudecode.AvailableAgentNames() {
		agents[a] = true
	}
	for _, name := range claudecode.AvailableConsultingSkillNames() {
		content, err := claudecode.ExportTemplateFS.ReadFile("templates/skills/" + name + "/SKILL.md")
		if err != nil {
			t.Fatalf("reading %s: %v", name, err)
		}
		agent := frontMatterValue(string(content), "agent")
		if agent == "" {
			continue
		}
		if !agents[agent] {
			t.Errorf("workflow %q runs in agent %q, which is not in the agent manifest", name, agent)
			continue
		}
		tool, ok := reg.ByName("consulting-workflow-" + name)
		if !ok {
			t.Errorf("workflow %q has no registered tool", name)
			continue
		}
		if want := "consulting-agent-" + agent; !slices.Contains(tool.Prerequisites, want) {
			t.Errorf("consulting-workflow-%s prerequisites = %v, want to include %q", name, tool.Prerequisites, want)
		}
	}
}

// frontMatterValue returns the value of key in a Markdown file's YAML front
// matter, or "" when absent.
func frontMatterValue(content, key string) string {
	rest, ok := strings.CutPrefix(content, "---\n")
	if !ok {
		return ""
	}
	fm, _, _ := strings.Cut(rest, "\n---")
	for _, line := range strings.Split(fm, "\n") {
		if v, ok := strings.CutPrefix(line, key+":"); ok {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// TestConsultingGenerateFuncs_MatchInitPath verifies the `enable` path (the
// consulting-agent-*/consulting-workflow-* tool GenerateFuncs) applies the
// same tier policy and produces the same file as init/update: nothing below
// Full (so enable's honesty guard reports it), and at Full the identical
// LibraryManaged file deployAgents/deployWorkflowSkills emit.
func TestConsultingGenerateFuncs_MatchInitPath(t *testing.T) {
	t.Parallel()
	reg := toolreg.DefaultRegistry()
	cases := []struct {
		tool   string
		deploy func(types.WizardAnswers) ([]types.GeneratedFile, error)
	}{
		{"consulting-agent-security-reviewer", claudecode.ExportDeployAgents},
		{"consulting-workflow-write-adr", func(a types.WizardAnswers) ([]types.GeneratedFile, error) {
			return claudecode.ExportDeployWorkflowSkills(a, nil)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.tool, func(t *testing.T) {
			t.Parallel()
			tool, ok := reg.ByName(tc.tool)
			if !ok || tool.GenerateFunc == nil {
				t.Fatalf("tool %q not registered with a GenerateFunc", tc.tool)
			}
			enabled := map[string]bool{tc.tool: true}

			below, err := tool.GenerateFunc(types.WizardAnswers{Tier: "standard", EnabledTools: enabled})
			if err != nil {
				t.Fatalf("GenerateFunc(standard): %v", err)
			}
			if len(below) != 0 {
				t.Errorf("GenerateFunc(standard) = %d files, want none (Full-tier artifact)", len(below))
			}

			full := types.WizardAnswers{Tier: "full", EnabledTools: enabled}
			got, err := tool.GenerateFunc(full)
			if err != nil {
				t.Fatalf("GenerateFunc(full): %v", err)
			}
			want, err := tc.deploy(full)
			if err != nil {
				t.Fatalf("deploy: %v", err)
			}
			if len(got) != 1 || len(want) != 1 {
				t.Fatalf("got %d enable files and %d init files, want 1 each", len(got), len(want))
			}
			g, w := got[0], want[0]
			if g.Path != w.Path || g.Strategy != w.Strategy || g.Owner != w.Owner || string(g.Content) != string(w.Content) {
				t.Errorf("enable file %+v differs from init file %+v", g, w)
			}
			if g.Strategy != types.LibraryManaged {
				t.Errorf("strategy = %v, want LibraryManaged", g.Strategy)
			}
		})
	}
}

// TestConsultingToolBehaviours_MatchManifests guards against drift between the
// embedded consulting manifests and the tool registry: every manifest entry
// must be a registered tool with generate behaviour, and
// every registered consulting tool must come from a manifest.
func TestConsultingToolBehaviours_MatchManifests(t *testing.T) {
	t.Parallel()
	reg := toolreg.DefaultRegistry()
	groups := []struct {
		prefix string
		names  []string
	}{
		{"consulting-agent-", claudecode.AvailableAgentNames()},
		{"consulting-workflow-", claudecode.AvailableConsultingSkillNames()},
	}
	for _, g := range groups {
		inManifest := make(map[string]bool, len(g.names))
		for _, n := range g.names {
			inManifest[g.prefix+n] = true
			tool, ok := reg.ByName(g.prefix + n)
			if !ok {
				t.Errorf("manifest entry %q has no registered tool", g.prefix+n)
				continue
			}
			// Enabling and disabling only toggle EnabledTools, which the
			// lifecycle does itself (registry-pkgmgr-1 F476 removed the
			// bookkeeping closures); the generator is the behaviour a tool
			// must carry.
			if tool.GenerateFunc == nil {
				t.Errorf("tool %q is missing its generate behaviour", tool.Name)
			}
		}
		for _, tool := range reg.All() {
			if strings.HasPrefix(tool.Name, g.prefix) && !inManifest[tool.Name] {
				t.Errorf("registered tool %q is not in the embedded manifest", tool.Name)
			}
		}
	}
}
