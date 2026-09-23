package canon

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

// TestIsProtected_ControlFilesAnyLocation pins the location-independent
// protection of the files that register or steer enforcement: the project (and
// home) Claude settings that register the qsdev hooks, the command/skill
// definitions, ~/.claude.json, and the project .qsdev/ state (policy, audit).
// Before the fix only .claude/hooks/ and .claude/agents/ were matched outside
// $HOME, so a Write to <project>/.claude/settings.json dropped every hook.
func TestIsProtected_ControlFilesAnyLocation(t *testing.T) {
	t.Parallel()

	resetProtectedPaths(t)

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("getting home dir: %v", err)
	}
	proj := filepath.FromSlash("/work/repo")

	tests := []struct {
		name     string
		path     string
		wantProt bool
		wantCat  string
	}{
		{"project settings.json", filepath.Join(proj, ".claude", "settings.json"), true, "claude-settings"},
		{"project settings.local.json", filepath.Join(proj, ".claude", "settings.local.json"), true, "claude-settings"},
		{"project command", filepath.Join(proj, ".claude", "commands", "deploy.md"), true, "claude-settings"},
		{"project skill", filepath.Join(proj, ".claude", "skills", "x", "SKILL.md"), true, "claude-settings"},
		{"project skills dir itself", filepath.Join(proj, ".claude", "skills"), true, "claude-settings"},
		{"home settings.json", filepath.Join(home, ".claude", "settings.json"), true, "claude-settings"},
		{"home command", filepath.Join(home, ".claude", "commands", "x.md"), true, "claude-settings"},
		{"home claude.json", filepath.Join(home, ".claude.json"), true, "claude-settings"},
		{"project qsdev policy", filepath.Join(proj, ".qsdev", "policy.yaml"), true, "config"},
		{"project qsdev audit", filepath.Join(proj, ".qsdev", "audit", "log.jsonl"), true, "audit"},
		{"project gdev", filepath.Join(proj, ".gdev", "x"), true, "config"},
		{"worktree settings.json", filepath.Join(proj, ".claude", "worktrees", "wt", ".claude", "settings.json"), true, "claude-settings"},

		// Legitimate edit targets stay writable.
		{"worktree source file", filepath.Join(proj, ".claude", "worktrees", "wt", "main.go"), false, ""},
		{"project rules", filepath.Join(proj, ".claude", "rules", "go.md"), false, ""},
		{"project CLAUDE.md", filepath.Join(proj, "CLAUDE.md"), false, ""},
		{"settings backup", filepath.Join(proj, ".claude", "settings.json.bak"), false, ""},
		{"claude.json backup", filepath.Join(home, ".claude.json.bak"), false, ""},
		{"embedded dir name", filepath.Join(proj, "foo.claude", "hooks", "x.sh"), false, ""},
		{"qsdev.yaml is not .qsdev/", filepath.Join(proj, ".qsdev.yaml"), false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotProt, gotCat := IsProtected(tt.path)
			if gotProt != tt.wantProt || gotCat != tt.wantCat {
				t.Errorf("IsProtected(%q) = (%v, %q), want (%v, %q)",
					tt.path, gotProt, gotCat, tt.wantProt, tt.wantCat)
			}
		})
	}
}

// TestProtectedSegmentsCoveredByCommandScan keeps the Write/Edit table and the
// Bash raw-command scan consistent: every location IsProtected guards must also
// be recognized by ContainsProtectedPath, or a Bash command could reach a file
// the Write tool cannot.
func TestProtectedSegmentsCoveredByCommandScan(t *testing.T) {
	t.Parallel()

	for _, seg := range protectedSegments {
		t.Run(seg.segment, func(t *testing.T) {
			t.Parallel()
			p := "/work/repo/" + seg.segment
			if strings.HasSuffix(p, "/") {
				p += "file"
			}
			if !ContainsProtectedPath(p) {
				t.Errorf("ContainsProtectedPath(%q) = false, but IsProtected guards segment %q", p, seg.segment)
			}
		})
	}
}

// TestIsProtected_CaseInsensitiveFilesystems verifies that on a case-folding
// filesystem (macOS/Windows defaults) a differently-cased spelling of a
// protected path is still protected, and that Windows name aliases (trailing
// dots/spaces, alternate data streams) cannot hide one.
func TestIsProtected_CaseInsensitiveFilesystems(t *testing.T) {
	t.Parallel()

	resetProtectedPaths(t)

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("getting home dir: %v", err)
	}
	upperSettings := filepath.Join(home, ".CLAUDE", "Settings.JSON")
	upperHook := filepath.FromSlash("/work/repo/.Claude/Hooks/pre.sh")

	fold := matchOptions{foldCase: true}
	win := matchOptions{foldCase: true, windowsAliases: true}
	exact := matchOptions{}

	tests := []struct {
		name     string
		path     string
		opts     matchOptions
		wantProt bool
	}{
		{"folded home settings", upperSettings, fold, true},
		{"folded project hook", upperHook, fold, true},
		{"folded claude.json", filepath.Join(home, ".Claude.json"), fold, true},
		{"folded qsdev bin", filepath.Join(home, ".QSDEV", "BIN", "qsdev"), fold, true},
		{"case-sensitive fs keeps names distinct", upperSettings, exact, false},
		{"windows ADS suffix", filepath.FromSlash("/work/repo/.claude/settings.json::$DATA"), win, true},
		{"windows trailing dot", filepath.FromSlash("/work/repo/.claude./settings.json."), win, true},
		{"windows trailing space", filepath.FromSlash("/work/repo/.claude/hooks /x.sh"), win, true},
		{"aliases ignored off windows", filepath.FromSlash("/work/repo/.claude/settings.json::$DATA"), fold, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got, _ := isProtected(tt.path, tt.opts); got != tt.wantProt {
				t.Errorf("isProtected(%q, %+v) = %v, want %v", tt.path, tt.opts, got, tt.wantProt)
			}
		})
	}

	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		t.Run("platform default folds case", func(t *testing.T) {
			t.Parallel()
			if got, _ := IsProtected(upperSettings); !got {
				t.Errorf("IsProtected(%q) = false on %s, want true", upperSettings, runtime.GOOS)
			}
			if !ContainsProtectedPath("rm ~/.Claude/settings.json") {
				t.Errorf("ContainsProtectedPath missed a differently-cased path on %s", runtime.GOOS)
			}
		})
	}
}

// TestContainsProtectedPath_ClaudeJSON verifies the raw-command scan agrees
// with IsProtected on ~/.claude.json, which no protected directory fragment
// covers, while a name that merely embeds it stays unmatched.
func TestContainsProtectedPath_ClaudeJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  bool
	}{
		{"echo '{}' > ~/.claude.json", true},
		{"cp /tmp/evil /home/u/.claude.json", true},
		{"python3 -c 'open(\".claude.json\",\"w\")'", true},
		{"cat ~/.claude.json.bak", false},
		{"cat my.claude.json", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			if got := ContainsProtectedPath(tt.input); got != tt.want {
				t.Errorf("ContainsProtectedPath(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestContainsProtectedPath_FoldCase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		foldCase bool
		want     bool
	}{
		{"rm ~/.Claude/settings.json", true, true},
		{"rm -rf .QSDEV", true, true},
		{"cat /ETC/GDEV/policy.yaml", true, true},
		{"rm ~/.Claude/settings.json", false, false},
		{"README.md", true, false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			if got := containsProtectedPath(tt.input, tt.foldCase); got != tt.want {
				t.Errorf("containsProtectedPath(%q, %v) = %v, want %v", tt.input, tt.foldCase, got, tt.want)
			}
		})
	}
}

func TestStripWindowsAliases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in, want string
	}{
		{"C:/Users/u/.claude/settings.json::$DATA", "C:/Users/u/.claude/settings.json"},
		{"C:/Users/u/.claude./settings.json. ", "C:/Users/u/.claude/settings.json"},
		{"C:/a/../b/./c", "C:/a/../b/./c"},
		{"//server/share/.claude/hooks/x.sh:stream", "//server/share/.claude/hooks/x.sh"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			if got := stripWindowsAliases(tt.in); got != tt.want {
				t.Errorf("stripWindowsAliases(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestCanonicalize_SymlinkResolutionForMissingTargets covers paths whose final
// target does not exist yet. A dangling symlink must canonicalize to the file a
// write through it would create, and ".." must be applied after the preceding
// symlink is resolved (as the kernel does), not lexically before it.
func TestCanonicalize_SymlinkResolutionForMissingTargets(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("symlinks require elevated privileges on Windows")
	}

	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("resolving temp dir: %v", err)
	}
	proj := filepath.Join(root, "proj")
	guarded := filepath.Join(root, "guarded", "d")
	for _, d := range []string{proj, guarded} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("creating %s: %v", d, err)
		}
	}
	symlink := func(target, link string) {
		t.Helper()
		if err := os.Symlink(target, link); err != nil {
			t.Fatalf("creating symlink %s -> %s: %v", link, target, err)
		}
	}

	// proj/dangle -> guarded/d/settings.local.json (absent)
	symlink(filepath.Join(guarded, "settings.local.json"), filepath.Join(proj, "dangle"))
	// proj/rel -> ../guarded/d/new.json (relative, absent)
	symlink(filepath.Join("..", "guarded", "d", "new.json"), filepath.Join(proj, "rel"))
	// proj/chain -> proj/rel (dangling chain)
	symlink(filepath.Join(proj, "rel"), filepath.Join(proj, "chain"))
	// proj/qlnk -> guarded/d (existing directory)
	symlink(guarded, filepath.Join(proj, "qlnk"))
	// proj/loop1 <-> proj/loop2
	symlink(filepath.Join(proj, "loop2"), filepath.Join(proj, "loop1"))
	symlink(filepath.Join(proj, "loop1"), filepath.Join(proj, "loop2"))

	tests := []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{"dangling absolute symlink", filepath.Join(proj, "dangle"), filepath.Join(guarded, "settings.local.json"), false},
		{"dangling relative symlink", filepath.Join(proj, "rel"), filepath.Join(guarded, "new.json"), false},
		{"dangling symlink chain", filepath.Join(proj, "chain"), filepath.Join(guarded, "new.json"), false},
		{"dotdot after symlink", proj + "/qlnk/../newfile", filepath.Join(root, "guarded", "newfile"), false},
		// The kernel would fail at "nothere"; a consumer that cleans the path
		// first writes through qlnk, so the symlink after ".." must be followed.
		{"symlink after dotdot out of a missing dir", proj + "/nothere/../qlnk/new.json", filepath.Join(guarded, "new.json"), false},
		{"missing tail under symlink", filepath.Join(proj, "qlnk", "sub", "x"), filepath.Join(guarded, "sub", "x"), false},
		{"symlink loop fails", filepath.Join(proj, "loop1", "x"), "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := Canonicalize(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Canonicalize(%q) = %q, want an error", tt.path, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Canonicalize(%q) returned error: %v", tt.path, err)
			}
			if got != tt.want {
				t.Errorf("Canonicalize(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

// TestResolveMissing_SymlinkLoop exercises the hop limit directly: a dangling
// cycle that EvalSymlinks never reaches (Tier 1 short-circuits on ELOOP) must
// still terminate with an error in the Tier 2 walk.
func TestResolveMissing_SymlinkLoop(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("symlinks require elevated privileges on Windows")
	}

	dir := t.TempDir()
	a, b := filepath.Join(dir, "a"), filepath.Join(dir, "b")
	if err := os.Symlink(b, a); err != nil {
		t.Fatalf("creating symlink: %v", err)
	}
	if err := os.Symlink(a, b); err != nil {
		t.Fatalf("creating symlink: %v", err)
	}
	if got, err := resolveMissing(filepath.Join(a, "x")); err == nil {
		t.Fatalf("resolveMissing through a symlink cycle = %q, want an error", got)
	}
}

// TestInstalledBinaryEntries pins where the guard-hook binary is protected:
// every installer's default directory (install.ps1 uses %LOCALAPPDATA% on
// Windows, not ~/.qsdev/bin) and the running executable wherever it lives.
func TestInstalledBinaryEntries(t *testing.T) {
	t.Parallel()
	sep := string(filepath.Separator)
	home := filepath.FromSlash("/home/alice")
	lad := filepath.FromSlash("/users/alice/appdata/local")
	exe := filepath.FromSlash("/opt/tools/qsdev")
	homeBin := filepath.Join(home, ".qsdev", "bin") + sep
	winBin := filepath.Join(lad, "qsdev", "bin") + sep

	tests := []struct {
		name, goos, lad, exe string
		want                 []string
	}{
		{"linux default", "linux", "", "", []string{homeBin}},
		{"linux ignores LOCALAPPDATA", "linux", lad, "", []string{homeBin}},
		{"windows installer dir", "windows", lad, "", []string{homeBin, winBin}},
		{"custom install dir", "darwin", "", exe, []string{homeBin, exe}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got []string
			for _, e := range installedBinaryEntries(tt.goos, home, tt.lad, tt.exe) {
				if e.category != "binary" {
					t.Errorf("entry %q category = %q, want binary", e.path, e.category)
				}
				got = append(got, e.path)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("entries = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestIsProtected_RunningExecutable checks that the binary running the hooks
// is protected even outside ~/.qsdev/bin (here: the test binary's location).
func TestIsProtected_RunningExecutable(t *testing.T) {
	t.Parallel()
	exe := runningExecutable()
	if exe == "" {
		t.Skip("cannot resolve the running executable")
	}
	if prot, cat := IsProtected(exe); !prot || cat != "binary" {
		t.Errorf("IsProtected(%q) = (%v, %q), want (true, binary)", exe, prot, cat)
	}
	sibling := filepath.Join(filepath.Dir(exe), "unrelated.txt")
	if prot, _ := IsProtected(sibling); prot {
		t.Errorf("IsProtected(%q) = true; only the executable itself should be protected", sibling)
	}
}
