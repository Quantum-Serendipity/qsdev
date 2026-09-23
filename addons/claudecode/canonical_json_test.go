package claudecode_test

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/internal/merge"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestGeneratedJSON_MergeIsByteStable is the W166 regression test: init wrote
// settings.json and .mcp.json in struct field order while the three-way merge
// re-serialized them in sorted key order, so the first update (or join) on an
// unchanged project rewrote hundreds of lines. Merging freshly generated
// content with itself must now be a byte-for-byte no-op.
func TestGeneratedJSON_MergeIsByteStable(t *testing.T) {
	t.Parallel()

	answers := types.WizardAnswers{
		ClaudeCode:      true,
		PermissionLevel: "standard",
		MCPServers:      []string{"context7", "github"},
		Hooks:           types.HookChoices{SafetyBlock: true, CredentialScan: true, SelfProtection: true},
	}
	cfg := claudecode.NewConfig()

	settings, err := claudecode.GenerateSettings(answers, ecosystem.NewRegistry(), cfg)
	if err != nil {
		t.Fatalf("GenerateSettings: %v", err)
	}
	mcp, err := claudecode.GenerateMcpJson(answers, cfg)
	if err != nil {
		t.Fatalf("GenerateMcpJson: %v", err)
	}

	tests := []struct {
		name    string
		content []byte
		mergeFn func(base, theirs, ours []byte) ([]byte, error)
		useBase bool
	}{
		{"settings.json with stored base", settings.Content, merge.MergeSettings, true},
		{"settings.json without stored base", settings.Content, merge.MergeSettings, false},
		{".mcp.json with stored base", mcp.Content, merge.MergeMcpJson, true},
		{".mcp.json without stored base", mcp.Content, merge.MergeMcpJson, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var base []byte
			if tt.useBase {
				base = tt.content
			}
			got, err := tt.mergeFn(base, tt.content, tt.content)
			if err != nil {
				t.Fatalf("merge: %v", err)
			}
			if string(got) != string(tt.content) {
				t.Errorf("merge rewrote unchanged content\n--- generated\n%s\n--- merged\n%s", tt.content, got)
			}
		})
	}
}
