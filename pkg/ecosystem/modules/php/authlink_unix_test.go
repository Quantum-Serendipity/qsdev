//go:build !windows

package php_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/php"
)

// authLinkEnv is the filesystem a run of the auth-link snippet sees.
type authLinkEnv struct {
	home, xdg, composerHome string
}

func (e authLinkEnv) xdgAuth() string    { return filepath.Join(e.xdg, "composer", "auth.json") }
func (e authLinkEnv) legacyAuth() string { return filepath.Join(e.home, ".composer", "auth.json") }
func (e authLinkEnv) link() string       { return filepath.Join(e.composerHome, "auth.json") }

// writeAuthFile creates an auth.json (and its parent) holding content.
func writeAuthFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

// runAuthLink executes the PHP enterShell snippet with sh in an isolated
// HOME, XDG_CONFIG_HOME and COMPOSER_HOME.
func runAuthLink(t *testing.T, e authLinkEnv) {
	t.Helper()
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Fatalf("sh not found: %v", err)
	}
	script := (&php.Module{}).EnterShellScript(ecosystem.ModuleConfig{})
	cmd := exec.CommandContext(t.Context(), sh, "-c", script)
	cmd.Env = []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + e.home,
		"XDG_CONFIG_HOME=" + e.xdg,
		"COMPOSER_HOME=" + e.composerHome,
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("snippet failed: %v\n%s", err, out)
	}
}

// TestEnterShellScript_ComposerAuthLink executes the auth passthrough
// snippet: it links the first global auth.json into the project Composer
// home and never replaces anything already there.
func TestEnterShellScript_ComposerAuthLink(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		setup    func(t *testing.T, e authLinkEnv)
		wantLink func(e authLinkEnv) string // "" means no link is created
		check    func(t *testing.T, e authLinkEnv)
	}{
		{
			name:     "links the XDG auth.json",
			setup:    func(t *testing.T, e authLinkEnv) { writeAuthFile(t, e.xdgAuth(), "xdg") },
			wantLink: authLinkEnv.xdgAuth,
		},
		{
			name:     "falls back to ~/.composer/auth.json",
			setup:    func(t *testing.T, e authLinkEnv) { writeAuthFile(t, e.legacyAuth(), "legacy") },
			wantLink: authLinkEnv.legacyAuth,
		},
		{
			name: "XDG wins over ~/.composer",
			setup: func(t *testing.T, e authLinkEnv) {
				writeAuthFile(t, e.xdgAuth(), "xdg")
				writeAuthFile(t, e.legacyAuth(), "legacy")
			},
			wantLink: authLinkEnv.xdgAuth,
		},
		{
			name: "existing auth.json is kept",
			setup: func(t *testing.T, e authLinkEnv) {
				writeAuthFile(t, e.xdgAuth(), "xdg")
				writeAuthFile(t, e.link(), "project")
			},
			check: func(t *testing.T, e authLinkEnv) {
				t.Helper()
				info, err := os.Lstat(e.link())
				if err != nil || !info.Mode().IsRegular() {
					t.Fatalf("auth.json was replaced: %v, %v", info, err)
				}
				if got, _ := os.ReadFile(e.link()); string(got) != "project" {
					t.Errorf("auth.json content = %q, want %q", got, "project")
				}
			},
		},
		{
			name: "dangling link is kept",
			setup: func(t *testing.T, e authLinkEnv) {
				writeAuthFile(t, e.xdgAuth(), "xdg")
				if err := os.MkdirAll(e.composerHome, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(e.home, "gone"), e.link()); err != nil {
					t.Fatal(err)
				}
			},
			check: func(t *testing.T, e authLinkEnv) {
				t.Helper()
				if got, err := os.Readlink(e.link()); err != nil || got != filepath.Join(e.home, "gone") {
					t.Errorf("dangling link was replaced: %q, %v", got, err)
				}
			},
		},
		{
			name:  "no global file creates nothing",
			setup: func(*testing.T, authLinkEnv) {},
			check: func(t *testing.T, e authLinkEnv) {
				t.Helper()
				if _, err := os.Lstat(e.composerHome); !os.IsNotExist(err) {
					t.Errorf("COMPOSER_HOME was created with no global auth.json: %v", err)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			e := authLinkEnv{
				home:         filepath.Join(root, "home"),
				xdg:          filepath.Join(root, "xdg"),
				composerHome: filepath.Join(root, "project", ".qsdev-composer"),
			}
			if err := os.MkdirAll(e.home, 0o755); err != nil {
				t.Fatal(err)
			}
			tt.setup(t, e)
			runAuthLink(t, e)
			if tt.wantLink != nil {
				got, err := os.Readlink(e.link())
				if err != nil {
					t.Fatalf("auth.json is not a link: %v", err)
				}
				if want := tt.wantLink(e); got != want {
					t.Errorf("auth.json links to %q, want %q", got, want)
				}
			}
			if tt.check != nil {
				tt.check(t, e)
			}
		})
	}
}

// TestEnterShellScript_XDGDefault links ~/.config/composer/auth.json when
// XDG_CONFIG_HOME is unset, as clean mode leaves it.
func TestEnterShellScript_XDGDefault(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	home := filepath.Join(root, "home")
	src := filepath.Join(home, ".config", "composer", "auth.json")
	writeAuthFile(t, src, "default")
	composerHome := filepath.Join(root, "project", ".qsdev-composer")

	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Fatalf("sh not found: %v", err)
	}
	cmd := exec.CommandContext(t.Context(), sh, "-c", (&php.Module{}).EnterShellScript(ecosystem.ModuleConfig{}))
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + home, "COMPOSER_HOME=" + composerHome}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("snippet failed: %v\n%s", err, out)
	}
	if got, err := os.Readlink(filepath.Join(composerHome, "auth.json")); err != nil || got != src {
		t.Errorf("auth.json links to %q (%v), want %q", got, err, src)
	}
}
