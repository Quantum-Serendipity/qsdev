//go:build unix

package catalog

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/projectctx"
)

// writeProjectDefaults writes content as root's project defaults file and
// returns its path.
func writeProjectDefaults(t *testing.T, root, content string) string {
	t.Helper()
	p := ProjectConfigPath(root)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func chmodT(t *testing.T, path string, mode fs.FileMode) {
	t.Helper()
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}

// TestProjectConfigFileRejectsUntrusted: the project defaults file is policy,
// so one that another local user could have written is refused, naming the
// file, rather than applied or silently dropped.
func TestProjectConfigFileRejectsUntrusted(t *testing.T) {
	tests := []struct {
		name string
		// loosen makes the planted file untrusted.
		loosen func(t *testing.T, root, file string)
	}{
		{"world-writable file", func(t *testing.T, _, file string) { chmodT(t, file, 0o666) }},
		{"world-writable state dir", func(t *testing.T, _, file string) { chmodT(t, filepath.Dir(file), 0o777) }},
		{"world-writable sticky root", func(t *testing.T, root, _ string) { chmodT(t, root, 0o777|fs.ModeSticky) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), "proj")
			file := writeProjectDefaults(t, root, "{}\n")
			tt.loosen(t, root, file)

			got, err := ProjectConfigFile(root)
			if got != "" || !errors.Is(err, projectctx.ErrUntrusted) {
				t.Fatalf("ProjectConfigFile = (%q, %v), want (\"\", ErrUntrusted)", got, err)
			}
			if msg := err.Error(); !strings.Contains(msg, "refusing project defaults") || !strings.Contains(msg, file) {
				t.Errorf("error %q, want it to say 'refusing project defaults' and name %s", msg, file)
			}
		})
	}
}

// TestProjectConfigFileAcceptsGroupWritable pins the single projectctx trust
// rule: a user-private-group umask (002) leaves the file group-writable, and
// that is still the user's own project.
func TestProjectConfigFileAcceptsGroupWritable(t *testing.T) {
	root := filepath.Join(t.TempDir(), "proj")
	file := writeProjectDefaults(t, root, "{}\n")
	chmodT(t, root, 0o775)
	chmodT(t, filepath.Dir(file), 0o775)
	chmodT(t, file, 0o664)

	got, err := ProjectConfigFile(root)
	if err != nil || got != file {
		t.Errorf("ProjectConfigFile = (%q, %v), want (%q, nil)", got, err, file)
	}
}

// TestDefaultFailsClosedOnUntrustedProjectDefaults: Default refuses the
// whole catalog rather than applying, or dropping, an untrusted project
// layer.
func TestDefaultFailsClosedOnUntrustedProjectDefaults(t *testing.T) {
	ResetDefault()
	t.Cleanup(ResetDefault)

	root := filepath.Join(t.TempDir(), "proj")
	file := writeProjectDefaults(t, root, "permission_deny_rules:\n  npx:\n    - Bash(project-only-deny *)\n")
	chmodT(t, file, 0o666)
	mustSetProjectRoot(t, root)

	cat, err := Default()
	if cat != nil || !errors.Is(err, projectctx.ErrUntrusted) {
		t.Fatalf("Default() = (%v, %v), want (nil, ErrUntrusted)", cat, err)
	}
	if !strings.Contains(err.Error(), file) {
		t.Errorf("error %q does not name %s", err, file)
	}
}
