//go:build unix

package projectctx

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// fakeInfo is a real FileInfo whose Sys() reports a fabricated Stat_t.
type fakeInfo struct {
	fs.FileInfo
	sys *syscall.Stat_t
}

func (f fakeInfo) Sys() any { return f.sys }

// overrideSys replaces the seam *seam so that, for the path target, the real
// result's Stat_t is passed through edit before being returned. Tests using
// it must not run in parallel.
func overrideSys(t *testing.T, seam *func(string) (fs.FileInfo, error), target string, edit func(*syscall.Stat_t)) {
	t.Helper()
	orig := *seam
	t.Cleanup(func() { *seam = orig })
	*seam = func(name string) (fs.FileInfo, error) {
		info, err := orig(name)
		if err != nil || filepath.Clean(name) != target {
			return info, err
		}
		st, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			t.Fatalf("stat %s: Sys() is %T, want *syscall.Stat_t", name, info.Sys())
		}
		cp := *st
		edit(&cp)
		return fakeInfo{FileInfo: info, sys: &cp}, nil
	}
}

// withStatOverride fabricates the Stat_t of target as both the stat and the
// lstat seam report it (the two agree for anything but a symlink).
func withStatOverride(t *testing.T, target string, edit func(*syscall.Stat_t)) {
	t.Helper()
	overrideSys(t, &stat, target, edit)
	overrideSys(t, &lstat, target, edit)
}

// withLstatOverride fabricates the Stat_t of the entry target itself (the
// lstat seam), leaving what stat reports for its symlink target alone.
func withLstatOverride(t *testing.T, target string, edit func(*syscall.Stat_t)) {
	t.Helper()
	overrideSys(t, &lstat, target, edit)
}

func symlink(t *testing.T, oldname, newname string) {
	t.Helper()
	if err := os.Symlink(oldname, newname); err != nil {
		t.Fatalf("symlink %s -> %s: %v", newname, oldname, err)
	}
}

func chmod(t *testing.T, path string, mode fs.FileMode) {
	t.Helper()
	if err := os.Chmod(path, mode); err != nil {
		t.Fatalf("chmod %s: %v", path, err)
	}
}

func TestResolve_UntrustedMarkers(t *testing.T) {
	b := branding.Get()
	tests := []struct {
		name string
		// setup plants the markers and returns the marker paths expected in
		// Ignored, in walk order.
		setup func(t *testing.T, root string) []string
	}{
		{
			name: "marker-in-world-writable-dir-ignored",
			setup: func(t *testing.T, root string) []string {
				mkdirs(t, root, "shared/"+b.StateDir, "shared/victim")
				touch(t, root, "shared/"+b.ConfigFile)
				chmod(t, filepath.Join(root, "shared"), 0o777|fs.ModeSticky)
				return []string{
					filepath.Join(root, "shared", b.ConfigFile),
					filepath.Join(root, "shared", b.StateDir),
				}
			},
		},
		{
			name: "world-writable-marker-ignored",
			setup: func(t *testing.T, root string) []string {
				touch(t, root, "proj/"+b.ConfigFile)
				mkdirs(t, root, "proj/victim")
				chmod(t, filepath.Join(root, "proj", b.ConfigFile), 0o666)
				return []string{filepath.Join(root, "proj", b.ConfigFile)}
			},
		},
		{
			name: "world-writable-state-dir-ignored",
			setup: func(t *testing.T, root string) []string {
				mkdirs(t, root, "proj/"+b.StateDir, "proj/victim")
				chmod(t, filepath.Join(root, "proj", b.StateDir), 0o777)
				return []string{filepath.Join(root, "proj", b.StateDir)}
			},
		},
		{
			name: "foreign-owner-ignored",
			setup: func(t *testing.T, root string) []string {
				touch(t, root, "proj/"+b.ConfigFile)
				mkdirs(t, root, "proj/victim")
				marker := filepath.Join(root, "proj", b.ConfigFile)
				foreign := uint32(os.Geteuid()) + 1 // never the euid, never 0
				withStatOverride(t, marker, func(st *syscall.Stat_t) { st.Uid = foreign })
				return []string{marker}
			},
		},
		{
			// U01-01: a link another user placed must not borrow the trust
			// of a target the current user owns.
			name: "symlink-marker-owned-by-foreign-uid-ignored",
			setup: func(t *testing.T, root string) []string {
				mkdirs(t, root, "mine", "proj/victim")
				link := filepath.Join(root, "proj", b.StateDir)
				symlink(t, filepath.Join(root, "mine"), link)
				foreign := uint32(os.Geteuid()) + 1
				withLstatOverride(t, link, func(st *syscall.Stat_t) { st.Uid = foreign })
				return []string{link}
			},
		},
		{
			name: "symlink-config-owned-by-foreign-uid-ignored",
			setup: func(t *testing.T, root string) []string {
				touch(t, root, "mine.yaml")
				mkdirs(t, root, "proj/victim")
				link := filepath.Join(root, "proj", b.ConfigFile)
				symlink(t, filepath.Join(root, "mine.yaml"), link)
				foreign := uint32(os.Geteuid()) + 1
				withLstatOverride(t, link, func(st *syscall.Stat_t) { st.Uid = foreign })
				return []string{link}
			},
		},
		{
			name: "own-symlink-to-world-writable-target-ignored",
			setup: func(t *testing.T, root string) []string {
				mkdirs(t, root, "drop", "proj/victim")
				chmod(t, filepath.Join(root, "drop"), 0o777)
				link := filepath.Join(root, "proj", b.StateDir)
				symlink(t, filepath.Join(root, "drop"), link)
				return []string{link}
			},
		},
		{
			// git's safe.directory rule: a directory another user owns is
			// not this user's project, even with a marker the user owns.
			name: "marker-in-foreign-owned-dir-ignored",
			setup: func(t *testing.T, root string) []string {
				touch(t, root, "shared/"+b.ConfigFile)
				mkdirs(t, root, "shared/victim")
				foreign := uint32(os.Geteuid()) + 1
				withStatOverride(t, filepath.Join(root, "shared"), func(st *syscall.Stat_t) { st.Uid = foreign })
				return []string{filepath.Join(root, "shared", b.ConfigFile)}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := fixture(t)
			wantIgnored := tt.setup(t, root)
			parent := filepath.Dir(wantIgnored[0])
			start := filepath.Join(parent, "victim")
			got, err := Resolve(start, Enclosing)
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			if got.Found || got.Root != start {
				t.Errorf("Resolve = {Root:%q Found:%v}, want {Root:%q Found:false}", got.Root, got.Found, start)
			}
			if !slices.Equal(got.Ignored, wantIgnored) {
				t.Errorf("Ignored = %v, want %v", got.Ignored, wantIgnored)
			}
		})
	}
}

// TestResolve_UntrustedMarkerWalkContinues: a skipped marker does not end the
// walk; a trusted marker further up is still found, and the skipped one is
// still reported.
func TestResolve_UntrustedMarkerWalkContinues(t *testing.T) {
	b := branding.Get()
	root := fixture(t)
	touch(t, root, "proj/"+b.ConfigFile)
	mkdirs(t, root, "proj/drop/"+b.StateDir, "proj/drop/sub")
	chmod(t, filepath.Join(root, "proj", "drop"), 0o777|fs.ModeSticky)

	got, err := Resolve(filepath.Join(root, "proj", "drop", "sub"), Enclosing)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if want := filepath.Join(root, "proj"); !got.Found || got.Root != want {
		t.Errorf("Resolve = {Root:%q Found:%v}, want {Root:%q Found:true}", got.Root, got.Found, want)
	}
	if want := []string{filepath.Join(root, "proj", "drop", b.StateDir)}; !slices.Equal(got.Ignored, want) {
		t.Errorf("Ignored = %v, want %v", got.Ignored, want)
	}
}

// TestResolve_GroupWritableTrusted: a user-private-group umask (002) leaves
// project files group-writable; they must still be trusted.
func TestResolve_GroupWritableTrusted(t *testing.T) {
	b := branding.Get()
	root := fixture(t)
	touch(t, root, "proj/"+b.ConfigFile)
	mkdirs(t, root, "proj/sub")
	chmod(t, filepath.Join(root, "proj"), 0o775)
	chmod(t, filepath.Join(root, "proj", b.ConfigFile), 0o664)

	got, err := Resolve(filepath.Join(root, "proj", "sub"), Enclosing)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if want := filepath.Join(root, "proj"); !got.Found || got.Root != want || len(got.Ignored) != 0 {
		t.Errorf("Resolve = %+v, want Root %q Found with nothing ignored", got, want)
	}
}

// TestResolve_RootOwnedMarkerTrusted: a marker owned by uid 0 is trusted.
func TestResolve_RootOwnedMarkerTrusted(t *testing.T) {
	b := branding.Get()
	root := fixture(t)
	touch(t, root, "proj/"+b.ConfigFile)
	mkdirs(t, root, "proj/sub")
	marker := filepath.Join(root, "proj", b.ConfigFile)
	withStatOverride(t, marker, func(st *syscall.Stat_t) { st.Uid = 0 })

	got, err := Resolve(filepath.Join(root, "proj", "sub"), Enclosing)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !got.Found || got.Root != filepath.Join(root, "proj") {
		t.Errorf("Resolve = %+v, want the root-owned marker trusted", got)
	}
}

// TestResolve_DeviceCeiling: the walk does not cross onto a parent that lives
// on a different device, as git's GIT_DISCOVERY_ACROSS_FILESYSTEM default.
func TestResolve_DeviceCeiling(t *testing.T) {
	b := branding.Get()
	root := fixture(t)
	touch(t, root, "proj/"+b.ConfigFile)
	mkdirs(t, root, "proj/mnt/sub")
	start := filepath.Join(root, "proj", "mnt", "sub")

	// Control: on one device the marker is found.
	got, err := Resolve(start, Enclosing)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !got.Found {
		t.Fatalf("control Resolve = %+v, want the marker found", got)
	}

	// proj on another device: mnt is a mount point, so the walk stops there.
	withStatOverride(t, filepath.Join(root, "proj"), func(st *syscall.Stat_t) { st.Dev++ })
	got, err = Resolve(start, Enclosing)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Found || got.Root != start || got.GitTop != "" {
		t.Errorf("Resolve across a device change = %+v, want {Root:%q Found:false GitTop:\"\"}", got, start)
	}
}

// TestResolve_UnreadableGitEntryStops covers the fail-closed ceiling: a
// directory whose .git entry cannot be inspected (Lstat fails with an error
// other than not-exist) stops the walk, so an ancestor marker is not adopted.
func TestResolve_UnreadableGitEntryStops(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses directory permissions")
	}
	root := fixture(t)
	touch(t, root, "proj/"+branding.Get().ConfigFile)
	mkdirs(t, root, "proj/locked")
	locked := filepath.Join(root, "proj", "locked")
	// Read without search permission: Lstat(locked/.git) fails with EACCES.
	chmod(t, locked, 0o600)
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	if _, err := os.Lstat(filepath.Join(locked, gitEntryName)); err == nil || errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("precondition: Lstat(.git) error = %v, want a non-not-exist error", err)
	}

	got, err := Resolve(locked, Enclosing)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Found || got.Root != locked || got.GitTop != locked {
		t.Errorf("Resolve = %+v, want no project and GitTop %q: an unreadable .git entry is a ceiling", got, locked)
	}
}

// TestResolve_OwnSymlinkMarkerTrusted: a marker the user linked into place
// (a symlink the user owns, to a directory the user owns) is still a marker.
func TestResolve_OwnSymlinkMarkerTrusted(t *testing.T) {
	b := branding.Get()
	root := fixture(t)
	mkdirs(t, root, "state", "proj/sub")
	symlink(t, filepath.Join(root, "state"), filepath.Join(root, "proj", b.StateDir))

	got, err := Resolve(filepath.Join(root, "proj", "sub"), Enclosing)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if want := filepath.Join(root, "proj"); !got.Found || got.Root != want || len(got.Ignored) != 0 {
		t.Errorf("Resolve = %+v, want Root %q Found with nothing ignored", got, want)
	}
}

// TestProbeBoundary_CrossesDeviceToToplevel: a mount between the working
// directory and the repository toplevel does not shrink the probe boundary
// below the toplevel, which would let a version probe run repository
// binaries it must refuse.
func TestProbeBoundary_CrossesDeviceToToplevel(t *testing.T) {
	root := t.TempDir()
	mkdirs(t, root, "repo/.git", "repo/mnt/sub", "home")
	repo := filepath.Join(root, "repo")
	// repo/mnt is a mount point: its parent, the toplevel, is on another
	// device.
	withStatOverride(t, repo, func(st *syscall.Stat_t) { st.Dev++ })

	wd := filepath.Join(repo, "mnt", "sub")
	if got := probeBoundary(wd, filepath.Join(root, "home")); got != repo {
		t.Errorf("probeBoundary across a mount = %q, want the toplevel %q", got, repo)
	}
}

// TestCheckTrusted pins the one trust rule CheckTrusted shares with the marker
// walk: an entry and its parent directory must be owned by the user or root
// and not world-writable; a symlink must itself be the user's. Group-writable
// is accepted (user-private-group umask 002).
func TestCheckTrusted(t *testing.T) {
	foreign := uint32(os.Geteuid()) + 1 // never the euid, never 0
	tests := []struct {
		name string
		// setup plants the entry below root and returns its path.
		setup      func(t *testing.T, root string) string
		wantReason string // "" means trusted
	}{
		{
			name: "own-file-trusted",
			setup: func(t *testing.T, root string) string {
				touch(t, root, "proj/f")
				return filepath.Join(root, "proj", "f")
			},
		},
		{
			name: "own-dir-trusted",
			setup: func(t *testing.T, root string) string {
				mkdirs(t, root, "proj/d")
				return filepath.Join(root, "proj", "d")
			},
		},
		{
			name: "group-writable-trusted",
			setup: func(t *testing.T, root string) string {
				touch(t, root, "proj/f")
				chmod(t, filepath.Join(root, "proj"), 0o775)
				chmod(t, filepath.Join(root, "proj", "f"), 0o664)
				return filepath.Join(root, "proj", "f")
			},
		},
		{
			name: "world-writable-file-untrusted",
			setup: func(t *testing.T, root string) string {
				touch(t, root, "proj/f")
				chmod(t, filepath.Join(root, "proj", "f"), 0o666)
				return filepath.Join(root, "proj", "f")
			},
			wantReason: "world-writable",
		},
		{
			name: "world-writable-parent-untrusted",
			setup: func(t *testing.T, root string) string {
				touch(t, root, "proj/f")
				chmod(t, filepath.Join(root, "proj"), 0o777|fs.ModeSticky)
				return filepath.Join(root, "proj", "f")
			},
			wantReason: "parent directory is world-writable",
		},
		{
			name: "foreign-owner-untrusted",
			setup: func(t *testing.T, root string) string {
				touch(t, root, "proj/f")
				p := filepath.Join(root, "proj", "f")
				withStatOverride(t, p, func(st *syscall.Stat_t) { st.Uid = foreign })
				return p
			},
			wantReason: "owned by uid",
		},
		{
			name: "foreign-owned-parent-untrusted",
			setup: func(t *testing.T, root string) string {
				touch(t, root, "proj/f")
				withStatOverride(t, filepath.Join(root, "proj"), func(st *syscall.Stat_t) { st.Uid = foreign })
				return filepath.Join(root, "proj", "f")
			},
			wantReason: "parent directory owned by uid",
		},
		{
			name: "foreign-symlink-untrusted",
			setup: func(t *testing.T, root string) string {
				touch(t, root, "mine")
				mkdirs(t, root, "proj")
				link := filepath.Join(root, "proj", "f")
				symlink(t, filepath.Join(root, "mine"), link)
				withLstatOverride(t, link, func(st *syscall.Stat_t) { st.Uid = foreign })
				return link
			},
			wantReason: "symlink owned by uid",
		},
		{
			name: "own-symlink-trusted",
			setup: func(t *testing.T, root string) string {
				touch(t, root, "mine")
				mkdirs(t, root, "proj")
				link := filepath.Join(root, "proj", "f")
				symlink(t, filepath.Join(root, "mine"), link)
				return link
			},
		},
		{
			name: "own-symlink-to-world-writable-target-untrusted",
			setup: func(t *testing.T, root string) string {
				touch(t, root, "mine")
				chmod(t, filepath.Join(root, "mine"), 0o666)
				mkdirs(t, root, "proj")
				link := filepath.Join(root, "proj", "f")
				symlink(t, filepath.Join(root, "mine"), link)
				return link
			},
			wantReason: "world-writable",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t, t.TempDir())
			err := CheckTrusted(path)
			if tt.wantReason == "" {
				if err != nil {
					t.Fatalf("CheckTrusted(%s) = %v, want nil", path, err)
				}
				return
			}
			if !errors.Is(err, ErrUntrusted) {
				t.Fatalf("CheckTrusted(%s) = %v, want ErrUntrusted", path, err)
			}
			var ue *UntrustedError
			if !errors.As(err, &ue) {
				t.Fatalf("CheckTrusted(%s) = %T, want *UntrustedError", path, err)
			}
			if ue.Path != path || !strings.HasPrefix(ue.Reason, tt.wantReason) {
				t.Errorf("UntrustedError = {Path:%q Reason:%q}, want {Path:%q Reason:%q...}", ue.Path, ue.Reason, path, tt.wantReason)
			}
		})
	}
}

// TestCheckTrusted_MissingEntry: an entry that cannot be inspected is an
// error, never trusted, and not reported as ErrUntrusted.
func TestCheckTrusted_MissingEntry(t *testing.T) {
	err := CheckTrusted(filepath.Join(t.TempDir(), "absent"))
	if err == nil || !errors.Is(err, fs.ErrNotExist) || errors.Is(err, ErrUntrusted) {
		t.Errorf("CheckTrusted(absent) = %v, want a not-exist error", err)
	}
}
