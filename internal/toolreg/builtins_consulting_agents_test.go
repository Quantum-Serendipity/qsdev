package toolreg

import (
	"strings"
	"testing"
)

func TestConsultingAgentToolsRegistered(t *testing.T) {
	reg := DefaultRegistry()

	agentNames := []string{
		"consulting-agent-security-reviewer",
		"consulting-agent-codebase-explorer",
		"consulting-agent-test-gap-analyzer",
		"consulting-agent-onboarding-guide",
		"consulting-agent-migration-planner",
		"consulting-agent-handoff-doc-generator",
		"consulting-agent-incident-debugger",
	}

	for _, name := range agentNames {
		tool, ok := reg.ByName(name)
		if !ok {
			t.Errorf("consulting agent tool %q not found in registry", name)
			continue
		}
		if tool.Category != CategoryAIAgent {
			t.Errorf("tool %q has category %v, want CategoryAIAgent", name, tool.Category)
		}
		if tool.DisplayName == "" {
			t.Errorf("tool %q has empty DisplayName", name)
		}
		if tool.Description == "" {
			t.Errorf("tool %q has empty Description", name)
		}
		if len(tool.OwnedFiles) != 1 {
			t.Errorf("tool %q has %d owned files, want 1", name, len(tool.OwnedFiles))
		} else {
			agentName := strings.TrimPrefix(name, "consulting-agent-")
			wantPath := ".claude/agents/" + agentName + ".md"
			if tool.OwnedFiles[0].Path != wantPath {
				t.Errorf("tool %q owned file path = %q, want %q", name, tool.OwnedFiles[0].Path, wantPath)
			}
			if tool.OwnedFiles[0].Ownership != Exclusive {
				t.Errorf("tool %q owned file ownership = %v, want Exclusive", name, tool.OwnedFiles[0].Ownership)
			}
		}
	}
}

func TestConsultingAgentToolsOptIn(t *testing.T) {
	reg := DefaultRegistry()

	for _, tool := range reg.All() {
		if !strings.HasPrefix(tool.Name, "consulting-agent-") {
			continue
		}
		if tool.Default != OptIn {
			t.Errorf("tool %q has default %v, want OptIn", tool.Name, tool.Default)
		}
	}
}

func TestConsultingAgentToolsLifecycleOnly(t *testing.T) {
	reg := DefaultRegistry()
	var names []string
	for _, tool := range reg.All() {
		if strings.HasPrefix(tool.Name, "consulting-agent-") {
			names = append(names, tool.Name)
		}
	}
	if len(names) == 0 {
		t.Fatal("no consulting-agent tools in registry")
	}
	assertLifecycleOnly(t, reg, names...)
}
