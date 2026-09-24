package toolreg

import (
	"testing"
)

func TestDevExToolsRegistered(t *testing.T) {
	reg := DefaultRegistry()

	tools := []struct {
		name     string
		category ToolCategory
	}{
		{"changelog", CategoryDevEx},
		{"commitlint", CategoryDevEx},
		{"secretspec", CategoryDevEx},
	}

	for _, tt := range tools {
		tool, ok := reg.ByName(tt.name)
		if !ok {
			t.Errorf("devex tool %q not found in registry", tt.name)
			continue
		}
		if tool.Category != tt.category {
			t.Errorf("tool %q has category %v, want %v", tt.name, tool.Category, tt.category)
		}
		if tool.DisplayName == "" {
			t.Errorf("tool %q has empty DisplayName", tt.name)
		}
		if tool.Description == "" {
			t.Errorf("tool %q has empty Description", tt.name)
		}
		if tool.Default != OptIn {
			t.Errorf("tool %q has default %v, want OptIn", tt.name, tool.Default)
		}
	}
}

func TestDevExToolsLifecycleOnly(t *testing.T) {
	assertLifecycleOnly(t, DefaultRegistry(), "changelog", "commitlint", "secretspec")
}
