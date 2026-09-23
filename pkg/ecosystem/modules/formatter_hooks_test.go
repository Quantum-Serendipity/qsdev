package modules

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// TestFormatterHooksReceiveFileOperands guards formatter hooks whose tools
// need file operands. `zig fmt` exits with "expected at least one source file
// argument" when given none, and `dart format` / `swiftformat --lint` check
// nothing, so with pass_filenames = false every commit either fails or goes
// unchecked. These hooks must receive the staged files.
func TestFormatterHooksReceiveFileOperands(t *testing.T) {
	t.Parallel()

	tests := []struct {
		module string
		hookID string
	}{
		{module: ecosystem.NameZig, hookID: "zig-fmt"},
		{module: ecosystem.NameDart, hookID: "dart-format"},
		{module: ecosystem.NameSwift, hookID: "swiftformat"},
	}
	for _, tt := range tests {
		t.Run(tt.hookID, func(t *testing.T) {
			t.Parallel()
			mod, ok := ecosystem.DefaultRegistry().ByName(tt.module)
			if !ok {
				t.Fatalf("module %q not registered", tt.module)
			}
			for _, h := range mod.PreCommitHooks(ecosystem.ModuleConfig{}) {
				if h.ID != tt.hookID {
					continue
				}
				if !h.PassFilenames {
					t.Errorf("hook %q (entry %q) has PassFilenames=false and no file operand", h.ID, h.Entry)
				}
				if len(h.Types) == 0 && h.Files == "" {
					t.Errorf("hook %q passes filenames without a Types/Files filter", h.ID)
				}
				return
			}
			t.Fatalf("module %q has no hook %q", tt.module, tt.hookID)
		})
	}
}
