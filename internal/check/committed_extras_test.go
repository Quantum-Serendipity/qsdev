package check

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/golang"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// writeFiles writes each path (relative to root) with its content.
func writeFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// TestDetectedModuleConfig_CommittedExtras checks the checks see the
// ModuleConfig generation uses: detected extras, with the module's
// committable settings taken from the committed extras alone. A committed
// directory or toolchain is never applied.
func TestDetectedModuleConfig_CommittedExtras(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeFiles(t, root, map[string]string{"go.mod": "module example.com/m\n\ngo 1.26.0\n\ntoolchain go1.26.8\n"})
	mod, ok := ecosystem.DefaultRegistry().ByName(ecosystem.NameGo)
	if !ok {
		t.Fatal("go module not registered")
	}
	tests := []struct {
		name          string
		extras        []string
		wantSource    string
		wantToolchain string
		wantDirectory string
	}{
		{"detected only", nil, "", "1.26.8", ""},
		{"committable setting applied", []string{"go_source=nixpkgs"}, golang.GoSourceNixpkgs, "1.26.8", ""},
		{"other keys ignored", []string{"toolchain=1.26.9", "directory=docs"}, "", "1.26.8", ""},
		{"invalid value ignored", []string{"go_source=nixpgks"}, "", "1.26.8", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			lang := types.LanguageConfig{Name: ecosystem.NameGo, Extras: tt.extras}
			cfg := detectedModuleConfig(root, mod, lang, 0, types.InfraConfig{})
			if got := cfg.Extra(golang.ExtraGoSource, ""); got != tt.wantSource {
				t.Errorf("go_source = %q, want %q", got, tt.wantSource)
			}
			if got := cfg.Extra(golang.ExtraToolchain, ""); got != tt.wantToolchain {
				t.Errorf("toolchain = %q, want %q", got, tt.wantToolchain)
			}
			if got := cfg.Directory(); got != tt.wantDirectory {
				t.Errorf("directory = %q, want %q", got, tt.wantDirectory)
			}
		})
	}
}

// TestCheckLanguageLockFiles_CommittedDirectoryIgnored reproduces a review
// finding: a committed directory=docs with a decoy docs/go.mod that declares
// no dependencies turned the missing go.sum of the real module from a
// failure into a skip.
func TestCheckLanguageLockFiles_CommittedDirectoryIgnored(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"go.mod":      "module example.com/m\n\ngo 1.26.0\n\nrequire golang.org/x/mod v0.30.0\n",
		"docs/go.mod": "module example.com/decoy\n\ngo 1.26.0\n",
	})
	lang := types.LanguageConfig{Name: ecosystem.NameGo, Extras: []string{"directory=docs"}}
	results := checkLanguageLockFiles(root, lang, 0, types.InfraConfig{})
	if len(results) != 1 || results[0].Status != StatusFail {
		t.Fatalf("results = %+v, want one failing go.sum check", results)
	}
}
