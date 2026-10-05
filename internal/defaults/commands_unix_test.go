//go:build unix

package defaults

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
)

// TestShowRefusesUntrustedProjectDefaults: `defaults show` and `defaults
// validate` apply the same trust rule as the catalog, so a project defaults
// file another local user could have written is refused, never shown as the
// effective policy. Not parallel: it sets the catalog's process-wide project
// root.
func TestShowRefusesUntrustedProjectDefaults(t *testing.T) {
	tests := []struct {
		name  string
		file  os.FileMode // mode of defaults.yaml
		state os.FileMode // mode of the state directory holding it
	}{
		{"world-writable file", 0o666, 0o755},
		{"world-writable state dir", 0o644, 0o777},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			path := catalog.ProjectConfigPath(root)
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			content := "permission_deny_rules:\n  npx:\n    - Bash(evil-hook *)\n"
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(path, tt.file); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(filepath.Dir(path), tt.state); err != nil {
				t.Fatal(err)
			}
			useProjectRoot(t, root)

			for _, sub := range []string{"show", "validate"} {
				cmd := Command()
				var stdout, stderr bytes.Buffer
				cmd.SetOut(&stdout)
				cmd.SetErr(&stderr)
				cmd.SetArgs([]string{sub})
				err := cmd.Execute()
				if err == nil {
					t.Fatalf("defaults %s succeeded, want a refusal; stdout: %s", sub, stdout.String())
				}
				msg := err.Error() + stderr.String()
				if !strings.Contains(msg, "refusing project defaults") || !strings.Contains(msg, path) {
					t.Errorf("defaults %s error %q, want 'refusing project defaults' naming %s", sub, msg, path)
				}
				if strings.Contains(stdout.String(), "evil-hook") {
					t.Errorf("defaults %s printed the untrusted policy: %s", sub, stdout.String())
				}
			}
		})
	}
}
