package claudecode_test

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/addons/claudecode"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

// ---------------------------------------------------------------------------
// loadAgentManifest tests
// ---------------------------------------------------------------------------

func TestLoadAgentManifest_Valid(t *testing.T) {
	manifest, err := claudecode.ExportLoadAgentManifest()
	if err != nil {
		t.Fatalf("loadAgentManifest returned error: %v", err)
	}

	if len(manifest.Agents) != 7 {
		t.Errorf("expected 7 agents in manifest, got %d", len(manifest.Agents))
	}

	for i, a := range manifest.Agents {
		if a.Name == "" {
			t.Errorf("agent %d has empty name", i)
		}
		if a.Description == "" {
			t.Errorf("agent %d (%s) has empty description", i, a.Name)
		}
	}
}

func TestLoadAgentManifest_AllFilesExist(t *testing.T) {
	manifest, err := claudecode.ExportLoadAgentManifest()
	if err != nil {
		t.Fatalf("loadAgentManifest returned error: %v", err)
	}

	for _, a := range manifest.Agents {
		// Enable the agent and deploy it to verify the embedded file exists.
		answers := types.WizardAnswers{
			EnabledTools: map[string]bool{
				"consulting-agent-" + a.Name: true,
			},
		}
		files, err := claudecode.ExportDeployAgents(answers)
		if err != nil {
			t.Errorf("agent %q: file not found in embed: %v", a.Name, err)
			continue
		}
		if len(files) != 1 {
			t.Errorf("agent %q: expected 1 file, got %d", a.Name, len(files))
		}
	}
}

// ---------------------------------------------------------------------------
// deployAgents tests
// ---------------------------------------------------------------------------

func TestDeployAgents_SkipsWhenNoEnabledTools(t *testing.T) {
	answers := types.WizardAnswers{
		EnabledTools: nil,
	}

	files, err := claudecode.ExportDeployAgents(answers)
	if err != nil {
		t.Fatalf("deployAgents returned error: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("expected 0 files when EnabledTools is nil, got %d", len(files))
	}
}

func TestDeployAgents_DeploysEnabled(t *testing.T) {
	answers := types.WizardAnswers{
		EnabledTools: map[string]bool{
			"consulting-agent-security-reviewer": true,
			"consulting-agent-incident-debugger": true,
		},
	}

	files, err := claudecode.ExportDeployAgents(answers)
	if err != nil {
		t.Fatalf("deployAgents returned error: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}

	paths := make(map[string]bool)
	for _, f := range files {
		paths[f.Path] = true
	}
	if !paths[".claude/agents/security-reviewer.md"] {
		t.Error("missing .claude/agents/security-reviewer.md")
	}
	if !paths[".claude/agents/incident-debugger.md"] {
		t.Error("missing .claude/agents/incident-debugger.md")
	}
}

func TestDeployAgents_CorrectPaths(t *testing.T) {
	// Enable all agents.
	manifest, err := claudecode.ExportLoadAgentManifest()
	if err != nil {
		t.Fatalf("loadAgentManifest returned error: %v", err)
	}

	enabled := make(map[string]bool)
	for _, a := range manifest.Agents {
		enabled["consulting-agent-"+a.Name] = true
	}

	answers := types.WizardAnswers{EnabledTools: enabled}
	files, err := claudecode.ExportDeployAgents(answers)
	if err != nil {
		t.Fatalf("deployAgents returned error: %v", err)
	}

	for _, f := range files {
		if !strings.HasPrefix(f.Path, ".claude/agents/") {
			t.Errorf("path %q does not start with .claude/agents/", f.Path)
		}
		if !strings.HasSuffix(f.Path, ".md") {
			t.Errorf("path %q does not end with .md", f.Path)
		}
	}
}

func TestDeployAgents_LibraryManagedStrategy(t *testing.T) {
	manifest, err := claudecode.ExportLoadAgentManifest()
	if err != nil {
		t.Fatalf("loadAgentManifest returned error: %v", err)
	}

	enabled := make(map[string]bool)
	for _, a := range manifest.Agents {
		enabled["consulting-agent-"+a.Name] = true
	}

	answers := types.WizardAnswers{EnabledTools: enabled}
	files, err := claudecode.ExportDeployAgents(answers)
	if err != nil {
		t.Fatalf("deployAgents returned error: %v", err)
	}

	for _, f := range files {
		if f.Strategy != types.LibraryManaged {
			t.Errorf("file %q has strategy %v, want LibraryManaged", f.Path, f.Strategy)
		}
	}
}

func TestDeployAgents_OwnerSet(t *testing.T) {
	manifest, err := claudecode.ExportLoadAgentManifest()
	if err != nil {
		t.Fatalf("loadAgentManifest returned error: %v", err)
	}

	enabled := make(map[string]bool)
	for _, a := range manifest.Agents {
		enabled["consulting-agent-"+a.Name] = true
	}

	answers := types.WizardAnswers{EnabledTools: enabled}
	files, err := claudecode.ExportDeployAgents(answers)
	if err != nil {
		t.Fatalf("deployAgents returned error: %v", err)
	}

	for _, f := range files {
		// Extract agent name from path.
		name := strings.TrimPrefix(f.Path, ".claude/agents/")
		name = strings.TrimSuffix(name, ".md")
		expectedOwner := "consulting-agent-" + name
		if f.Owner != expectedOwner {
			t.Errorf("file %q has owner %q, want %q", f.Path, f.Owner, expectedOwner)
		}
	}
}

func TestDeployAgents_ReadOnlyHaveDisallowedTools(t *testing.T) {
	manifest, err := claudecode.ExportLoadAgentManifest()
	if err != nil {
		t.Fatalf("loadAgentManifest returned error: %v", err)
	}

	readOnlyAgents := make(map[string]bool)
	for _, a := range manifest.Agents {
		if a.ReadOnly {
			readOnlyAgents[a.Name] = true
		}
	}

	enabled := make(map[string]bool)
	for _, a := range manifest.Agents {
		enabled["consulting-agent-"+a.Name] = true
	}

	answers := types.WizardAnswers{EnabledTools: enabled}
	files, err := claudecode.ExportDeployAgents(answers)
	if err != nil {
		t.Fatalf("deployAgents returned error: %v", err)
	}

	for _, f := range files {
		name := strings.TrimPrefix(f.Path, ".claude/agents/")
		name = strings.TrimSuffix(name, ".md")

		content := string(f.Content)
		hasDisallowed := strings.Contains(content, "disallowedTools:")

		if readOnlyAgents[name] && !hasDisallowed {
			t.Errorf("read-only agent %q should have disallowedTools in frontmatter", name)
		}
		if !readOnlyAgents[name] && hasDisallowed {
			t.Errorf("non-read-only agent %q should NOT have disallowedTools in frontmatter", name)
		}
	}
}

func TestDeployAgents_CodebaseExplorerUsesHaiku(t *testing.T) {
	answers := types.WizardAnswers{
		EnabledTools: map[string]bool{
			"consulting-agent-codebase-explorer": true,
		},
	}

	files, err := claudecode.ExportDeployAgents(answers)
	if err != nil {
		t.Fatalf("deployAgents returned error: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	content := string(files[0].Content)
	if !strings.Contains(content, "model: haiku") {
		t.Error("codebase-explorer agent should use model: haiku")
	}
}

func TestAvailableAgentNames(t *testing.T) {
	names := claudecode.AvailableAgentNames()
	if len(names) != 7 {
		t.Fatalf("expected 7 agent names, got %d: %v", len(names), names)
	}

	expected := map[string]bool{
		"security-reviewer":    true,
		"codebase-explorer":    true,
		"test-gap-analyzer":    true,
		"onboarding-guide":     true,
		"migration-planner":    true,
		"handoff-doc-generator": true,
		"incident-debugger":    true,
	}

	for _, name := range names {
		if !expected[name] {
			t.Errorf("unexpected agent name: %q", name)
		}
	}
}
