package sandbox_test

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/answers"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpconfig"
	"github.com/Quantum-Serendipity/qsdev/internal/projectctx"
	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/canon"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

func TestGuardrailPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		git  func(t *testing.T, project string)
		want []string // project-relative, slash-separated
		not  []string
	}{
		{
			name: "git directory: only its code-executing control files",
			git:  func(t *testing.T, p string) { mkdir(t, filepath.Join(p, ".git", "objects")) },
			want: []string{".git/hooks", ".git/config", ".git/config.worktree", ".git/commondir", ".git/info", ".git/modules"},
			not:  []string{".git", ".git/objects", ".git/index"},
		},
		{
			name: "gitdir pointer file: the file itself",
			git:  func(t *testing.T, p string) { writeFile(t, filepath.Join(p, ".git")) },
			want: []string{".git"},
			not:  []string{".git/hooks", ".git/config"},
		},
		{
			name: "no git: .git, so creating one is caught",
			git:  func(*testing.T, string) {},
			want: []string{".git"},
		},
	}
	// The owned names are derived, never re-spelled.
	owned := []string{
		".claude",
		projectctx.DataDirName(),
		branding.Get().StateDir,
		answers.DevenvCopyFile(),
		mcpconfig.FileName,
		branding.Get().ConfigFile,
		".envrc",
		"devenv.nix", "devenv.local.nix", "devenv.yaml", "devenv.local.yaml", "devenv.lock",
		".npmrc", ".pre-commit-config.yaml",
		branding.Get().LocalConfig, ".devenv/load-exports", ".direnv/flake-profile-a.rc",
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			project := filepath.Join(t.TempDir(), "proj")
			mkdir(t, project)
			tt.git(t, project)
			got := sandbox.GuardrailPaths(project)

			for _, p := range got {
				if !filepath.IsAbs(p) || !strings.HasPrefix(p, project+string(filepath.Separator)) {
					t.Errorf("guardrail path %q is not absolute and inside the project %q", p, project)
				}
				for _, q := range got {
					if strings.HasPrefix(p, q+string(filepath.Separator)) {
						t.Errorf("guardrail %q is listed under guardrail %q", p, q)
					}
				}
			}
			for _, rel := range tt.want {
				if want := filepath.Join(project, filepath.FromSlash(rel)); !slices.Contains(got, want) {
					t.Errorf("GuardrailPaths missing %q; got %v", want, got)
				}
			}
			for _, rel := range owned {
				if p := filepath.Join(project, filepath.FromSlash(rel)); !coveredBy(got, p) {
					t.Errorf("no guardrail covers %q; got %v", p, got)
				}
			}
			for _, rel := range tt.not {
				if p := filepath.Join(project, filepath.FromSlash(rel)); slices.Contains(got, p) {
					t.Errorf("GuardrailPaths lists %q, which must stay writable; got %v", p, got)
				}
			}
		})
	}

	if sandbox.GuardrailPaths("") != nil {
		t.Error("GuardrailPaths(\"\") must be nil: there is no project to protect")
	}
}

// TestGuardrailPaths_CoversCanon keeps the overlay list in step with
// selfprotect's canon: every location canon protects wherever it appears, the
// devenv configuration files it treats as environment sources, the files
// gate-dodge guards and the checkout-local caches and overrides must lie at
// or under a guardrail.
func TestGuardrailPaths_CoversCanon(t *testing.T) {
	t.Parallel()

	project := filepath.Join(t.TempDir(), "proj")
	guards := sandbox.GuardrailPaths(project)
	want := slices.Concat(canon.Segments(), canon.DevenvSourceFiles(), canon.GuardedConfigFiles(), state.LocalOnlyEntries())
	if len(want) == 0 {
		t.Fatal("canon names no project control files")
	}
	for _, rel := range want {
		p := filepath.Join(project, filepath.FromSlash(path.Clean(rel)))
		if !coveredBy(guards, p) {
			t.Errorf("canon protects %q but no guardrail covers %s", rel, p)
		}
	}
}

func TestHookLogDir(t *testing.T) {
	t.Parallel()

	project := filepath.Join(t.TempDir(), "proj")
	got := sandbox.HookLogDir(project)
	if want := filepath.Join(project, filepath.FromSlash(canon.HookLogDir)); got != want {
		t.Errorf("HookLogDir = %q, want %q", got, want)
	}
	// The log directory lies inside a guardrail (so only it is re-opened),
	// never is one.
	guards := sandbox.GuardrailPaths(project)
	if slices.Contains(guards, got) {
		t.Errorf("HookLogDir %q must not itself be a guardrail", got)
	}
	if !slices.ContainsFunc(guards, func(g string) bool { return strings.HasPrefix(got, g+string(filepath.Separator)) }) {
		t.Errorf("HookLogDir %q lies under no guardrail %v", got, guards)
	}
	if sandbox.HookLogDir("") != "" {
		t.Error("HookLogDir(\"\") must be empty")
	}
}

// TestWritableGuardrailDirs: the directories hooks may still write (the hook
// logs and devenv's state, which holds GOPATH and the venv) each lie inside a
// guardrail without being one, so only they are re-opened, and the rest of
// .devenv (its generated shell scripts) stays read-only.
func TestWritableGuardrailDirs(t *testing.T) {
	t.Parallel()

	project := filepath.Join(t.TempDir(), "proj")
	guards := sandbox.GuardrailPaths(project)
	dirs := sandbox.WritableGuardrailDirs(project)
	for _, want := range []string{sandbox.HookLogDir(project), sandbox.DevenvStateDir(project)} {
		if !slices.Contains(dirs, want) {
			t.Errorf("WritableGuardrailDirs = %v, missing %s", dirs, want)
		}
	}
	for _, d := range dirs {
		if slices.Contains(guards, d) {
			t.Errorf("writable dir %s must not itself be a guardrail", d)
		}
		if !slices.ContainsFunc(guards, func(g string) bool { return strings.HasPrefix(d, g+string(filepath.Separator)) }) {
			t.Errorf("writable dir %s lies under no guardrail %v", d, guards)
		}
	}
	if !coveredBy(guards, filepath.Dir(sandbox.DevenvStateDir(project))) {
		t.Errorf("the devenv directory holding %s must stay a guardrail", sandbox.DevenvStateDir(project))
	}
	if sandbox.WritableGuardrailDirs("") != nil || sandbox.DevenvStateDir("") != "" {
		t.Error("an empty project has no writable guardrail dirs")
	}
}

func TestUsesDevenv(t *testing.T) {
	t.Parallel()

	for _, name := range canon.DevenvSourceFiles() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			project := t.TempDir()
			if err := os.WriteFile(filepath.Join(project, name), nil, 0o644); err != nil {
				t.Fatal(err)
			}
			if !sandbox.UsesDevenv(project) {
				t.Errorf("UsesDevenv = false with %s present", name)
			}
		})
	}
	t.Run("none", func(t *testing.T) {
		t.Parallel()
		if sandbox.UsesDevenv(t.TempDir()) || sandbox.UsesDevenv("") {
			t.Error("UsesDevenv = true without a devenv configuration")
		}
	})
}

// symlinkOrSkip creates a symlink at p pointing to dest, skipping the test
// where the platform does not allow it (Windows without the privilege).
func symlinkOrSkip(t *testing.T, dest, p string) {
	t.Helper()
	mkdir(t, filepath.Dir(p))
	if err := os.Symlink(dest, p); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
}

// quarantined returns the quarantined copies of p, as Enforce names them.
func quarantined(t *testing.T, p string) []string {
	t.Helper()
	got, err := filepath.Glob(sandbox.QuarantineName(p, "*"))
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestGuardrailSnapshot_Enforce(t *testing.T) {
	t.Parallel()

	const evil = "repos: [{repo: local, hooks: [{id: x, entry: evil, language: system}]}]"
	precommit := ".pre-commit-config.yaml"
	tests := []struct {
		name    string
		setup   func(t *testing.T, project, outside string)
		mutate  func(t *testing.T, project string)
		wantErr []string
		check   func(t *testing.T, project, outside string)
	}{
		{
			name:   "nothing changes",
			setup:  func(t *testing.T, p, _ string) { writeFile(t, filepath.Join(p, ".envrc")) },
			mutate: func(*testing.T, string) {},
		},
		{
			name:    "absent guardrail created is moved aside",
			mutate:  func(t *testing.T, p string) { writeFile(t, filepath.Join(p, ".envrc")) },
			wantErr: []string{".envrc", "moved to"},
			check: func(t *testing.T, p, _ string) {
				assertAbsent(t, filepath.Join(p, ".envrc"))
				if q := quarantined(t, filepath.Join(p, ".envrc")); len(q) != 1 {
					t.Errorf("quarantined .envrc = %v, want one copy (nothing deleted)", q)
				}
			},
		},
		{
			name:    "absent guardrail directory created",
			mutate:  func(t *testing.T, p string) { mkdir(t, filepath.Join(p, ".git", "hooks")) },
			wantErr: []string{".git"},
			check:   func(t *testing.T, p, _ string) { assertAbsent(t, filepath.Join(p, ".git")) },
		},
		{
			name:    "git hooks created in an existing git directory",
			setup:   func(t *testing.T, p, _ string) { writeFile(t, filepath.Join(p, ".git", "config")) },
			mutate:  func(t *testing.T, p string) { mkdir(t, filepath.Join(p, ".git", "hooks")) },
			wantErr: []string{"hooks"},
			check:   func(t *testing.T, p, _ string) { assertAbsent(t, filepath.Join(p, ".git", "hooks")) },
		},
		{
			name:   "git index and objects written",
			setup:  func(t *testing.T, p, _ string) { writeFile(t, filepath.Join(p, ".git", "config")) },
			mutate: func(t *testing.T, p string) { writeFile(t, filepath.Join(p, ".git", "objects", "ab", "cd")) },
		},
		{
			name:    "devenv.local.nix created",
			mutate:  func(t *testing.T, p string) { writeFile(t, filepath.Join(p, "devenv.local.nix")) },
			wantErr: []string{"devenv.local.nix"},
		},
		{
			name:    "devenv shell cache created",
			mutate:  func(t *testing.T, p string) { writeFile(t, filepath.Join(p, ".devenv", "load-exports")) },
			wantErr: []string{".devenv"},
		},
		{
			name:  "guardrail replaced through a renamed parent",
			setup: func(t *testing.T, p, _ string) { mkdir(t, filepath.Join(p, ".git", "hooks")) },
			mutate: func(t *testing.T, p string) {
				git := filepath.Join(p, ".git")
				if err := os.Rename(git, git+"2"); err != nil {
					t.Fatal(err)
				}
				writeFile(t, filepath.Join(git, "hooks", "post-checkout"))
			},
			wantErr: []string{"hooks", "original was moved or removed"},
			check:   func(t *testing.T, p, _ string) { assertAbsent(t, filepath.Join(p, ".git", "hooks")) },
		},
		{
			name: "dangling symlink target created",
			setup: func(t *testing.T, p, _ string) {
				symlinkOrSkip(t, "envrc.real", filepath.Join(p, ".envrc"))
			},
			mutate:  func(t *testing.T, p string) { writeFile(t, filepath.Join(p, "envrc.real")) },
			wantErr: []string{".envrc", "moved to"},
			check:   func(t *testing.T, p, _ string) { assertAbsent(t, filepath.Join(p, "envrc.real")) },
		},
		{
			name: "symlinked pre-commit config replaced by a regular file",
			setup: func(t *testing.T, p, out string) {
				writeFile(t, filepath.Join(out, "pc.json"))
				symlinkOrSkip(t, filepath.Join(out, "pc.json"), filepath.Join(p, precommit))
			},
			mutate: func(t *testing.T, p string) {
				if err := os.Remove(filepath.Join(p, precommit)); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(p, precommit), []byte(evil), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: []string{precommit, "moved to", "restored"},
			check: func(t *testing.T, p, out string) {
				assertSymlink(t, filepath.Join(p, precommit), filepath.Join(out, "pc.json"))
				q := quarantined(t, filepath.Join(p, precommit))
				if len(q) != 1 {
					t.Fatalf("quarantined config = %v, want one", q)
				}
				if data, err := os.ReadFile(q[0]); err != nil || string(data) != evil {
					t.Errorf("quarantined config = %q (err %v), want the hook's file kept", data, err)
				}
			},
		},
		{
			name: "symlinked pre-commit config removed",
			setup: func(t *testing.T, p, out string) {
				writeFile(t, filepath.Join(out, "pc.json"))
				symlinkOrSkip(t, filepath.Join(out, "pc.json"), filepath.Join(p, precommit))
			},
			mutate: func(t *testing.T, p string) {
				if err := os.Remove(filepath.Join(p, precommit)); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: []string{precommit, "restored"},
			check: func(t *testing.T, p, out string) {
				assertSymlink(t, filepath.Join(p, precommit), filepath.Join(out, "pc.json"))
			},
		},
		{
			name: "every changed guardrail is named",
			mutate: func(t *testing.T, p string) {
				writeFile(t, filepath.Join(p, ".envrc"))
				writeFile(t, filepath.Join(p, "devenv.nix"))
			},
			wantErr: []string{".envrc", "devenv.nix"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			project, outside := t.TempDir(), t.TempDir()
			if tt.setup != nil {
				tt.setup(t, project, outside)
			}
			snap, err := sandbox.SnapshotGuardrails(project)
			if err != nil {
				t.Fatalf("SnapshotGuardrails: %v", err)
			}
			tt.mutate(t, project)
			err = snap.Enforce()
			if len(tt.wantErr) == 0 {
				if err != nil {
					t.Fatalf("Enforce = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, sandbox.ErrGuardrailModified) {
				t.Fatalf("Enforce = %v, want ErrGuardrailModified", err)
			}
			for _, w := range tt.wantErr {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("Enforce = %q, want it to mention %q", err, w)
				}
			}
			if tt.check != nil {
				tt.check(t, project, outside)
			}
		})
	}
}

// coveredBy reports whether p is one of guards or lies under one.
func coveredBy(guards []string, p string) bool {
	return slices.ContainsFunc(guards, func(g string) bool {
		return p == g || strings.HasPrefix(p, g+string(filepath.Separator))
	})
}

func assertAbsent(t *testing.T, p string) {
	t.Helper()
	if _, err := os.Lstat(p); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("%s still in place after Enforce (err %v)", p, err)
	}
}

func assertSymlink(t *testing.T, p, dest string) {
	t.Helper()
	got, err := os.Readlink(p)
	if err != nil || got != dest {
		t.Errorf("%s -> %q (err %v), want the symlink to %q restored", p, got, err, dest)
	}
}

func writeFile(t *testing.T, p string) {
	t.Helper()
	mkdir(t, filepath.Dir(p))
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mkdir(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
}
