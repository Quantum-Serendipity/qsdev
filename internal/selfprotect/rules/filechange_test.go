package rules

import (
	"path/filepath"
	"testing"
)

func TestEvalContext_FileChange(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	existing := writeTestFile(t, dir, "config.yaml", "level: strict\nlevel: strict\n")
	missing := filepath.Join(dir, "new.yaml")

	tests := []struct {
		name       string
		ctx        EvalContext
		wantBefore string
		wantAfter  string
		wantErr    bool
	}{
		{
			name:       "write replaces the whole file",
			ctx:        EvalContext{ToolName: "Write", CanonicalPath: existing, Content: "level: baseline\n"},
			wantBefore: "level: strict\nlevel: strict\n",
			wantAfter:  "level: baseline\n",
		},
		{
			name:      "write creates a missing file",
			ctx:       EvalContext{ToolName: "Write", CanonicalPath: missing, Content: "x"},
			wantAfter: "x",
		},
		{
			name:       "edit replaces the first occurrence",
			ctx:        EvalContext{ToolName: "Edit", CanonicalPath: existing, Edits: []TextEdit{{OldString: "strict", NewString: "baseline"}}},
			wantBefore: "level: strict\nlevel: strict\n",
			wantAfter:  "level: baseline\nlevel: strict\n",
		},
		{
			name:       "edit with replace_all replaces every occurrence",
			ctx:        EvalContext{ToolName: "Edit", CanonicalPath: existing, Edits: []TextEdit{{OldString: "strict", NewString: "baseline", ReplaceAll: true}}},
			wantBefore: "level: strict\nlevel: strict\n",
			wantAfter:  "level: baseline\nlevel: baseline\n",
		},
		{
			name: "multiedit applies edits in order",
			ctx: EvalContext{ToolName: "MultiEdit", CanonicalPath: existing, Edits: []TextEdit{
				{OldString: "strict", NewString: "enhanced"},
				{OldString: "enhanced", NewString: "baseline"},
			}},
			wantBefore: "level: strict\nlevel: strict\n",
			wantAfter:  "level: baseline\nlevel: strict\n",
		},
		{
			name:       "edit whose old string is absent changes nothing",
			ctx:        EvalContext{ToolName: "Edit", CanonicalPath: existing, Edits: []TextEdit{{OldString: "nope", NewString: "x"}}},
			wantBefore: "level: strict\nlevel: strict\n",
			wantAfter:  "level: strict\nlevel: strict\n",
		},
		{
			name:      "edit with empty old string creates a missing file",
			ctx:       EvalContext{ToolName: "Edit", CanonicalPath: missing, Edits: []TextEdit{{NewString: "x"}}},
			wantAfter: "x",
		},
		{
			name:    "edit without replacements cannot be reconstructed",
			ctx:     EvalContext{ToolName: "Edit", CanonicalPath: existing, Content: "x"},
			wantErr: true,
		},
		{
			name:    "unreadable target",
			ctx:     EvalContext{ToolName: "Write", CanonicalPath: dir},
			wantErr: true,
		},
		{
			name:    "no target",
			ctx:     EvalContext{ToolName: "Write"},
			wantErr: true,
		},
		{
			name:    "not a file write",
			ctx:     EvalContext{ToolName: "Bash", CanonicalPath: existing},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			before, after, err := tt.ctx.FileChange()
			if (err != nil) != tt.wantErr {
				t.Fatalf("FileChange() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if before != tt.wantBefore || after != tt.wantAfter {
				t.Errorf("FileChange() = (%q, %q), want (%q, %q)", before, after, tt.wantBefore, tt.wantAfter)
			}
		})
	}
}
