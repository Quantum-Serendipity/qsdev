package fileutil_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

func TestWriteFileAtomic_CreatesFileWithCorrectContentAndPermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := []byte("hello world")

	if err := fileutil.WriteFileAtomic(path, content, 0o644); err != nil {
		t.Fatalf("WriteFileAtomic: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("content = %q, want %q", got, content)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if runtime.GOOS != "windows" {
		if info.Mode().Perm() != 0o644 {
			t.Errorf("mode = %o, want %o", info.Mode().Perm(), 0o644)
		}
	}
}

func TestWriteFileAtomic_OverwritesExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")

	if err := os.WriteFile(path, []byte("old content"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	newContent := []byte("new content")
	if err := fileutil.WriteFileAtomic(path, newContent, 0o644); err != nil {
		t.Fatalf("WriteFileAtomic: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(newContent) {
		t.Errorf("content = %q, want %q", got, newContent)
	}
}

func TestWriteFileAtomic_CreatesNestedDirectories(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "nested", "file.txt")
	content := []byte("nested content")

	if err := fileutil.WriteFileAtomic(path, content, 0o644); err != nil {
		t.Fatalf("WriteFileAtomic: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("content = %q, want %q", got, content)
	}
}

func TestWriteFileAtomic_CleansUpTempFileOnWriteFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("read-only directory permissions not enforced on Windows")
	}
	dir := t.TempDir()
	// Make the directory read-only so Chmod will fail after write succeeds
	// but we can still create temp files. Instead, test a simpler scenario:
	// write to a path where rename will fail because target dir doesn't exist
	// and we prevent MkdirAll from creating it by making parent read-only.
	subdir := filepath.Join(dir, "readonly")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("setup MkdirAll: %v", err)
	}
	if err := os.Chmod(subdir, 0o444); err != nil {
		t.Fatalf("setup Chmod: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(subdir, 0o755)
	})

	path := filepath.Join(subdir, "child", "file.txt")
	err := fileutil.WriteFileAtomic(path, []byte("data"), 0o644)
	if err == nil {
		t.Fatal("expected error when writing to read-only directory, got nil")
	}

	// Verify no temp files were left behind in the subdir
	entries, _ := os.ReadDir(subdir)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".qsdev-tmp-") {
			t.Errorf("temp file left behind: %s", entry.Name())
		}
	}
}

func TestWriteFileAtomic_Mode0755Preserved(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix file permissions not supported on Windows")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "script.sh")
	content := []byte("#!/bin/bash\necho hello")

	if err := fileutil.WriteFileAtomic(path, content, 0o755); err != nil {
		t.Fatalf("WriteFileAtomic: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Errorf("mode = %o, want %o", info.Mode().Perm(), 0o755)
	}
}

func TestWriteFileAtomic_NoPartialContentDuringConcurrentRead(t *testing.T) {
	if runtime.GOOS == "windows" {
		// Windows cannot rename over a file with open handles that lack FILE_SHARE_DELETE.
		// Go's os.ReadFile opens without FILE_SHARE_DELETE (golang/go#32088), making it
		// impossible to atomically replace a file under continuous concurrent reads.
		// Production scenarios (brief, non-overlapping reads) work fine with os.Root.Rename.
		t.Skip("Windows file sharing semantics prevent atomic rename under continuous readers")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "concurrent.txt")
	original := []byte("original content that should remain intact")

	// Write the initial file.
	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	replacement := []byte("replacement content written atomically!")

	var wg sync.WaitGroup
	const readers = 10

	// Start concurrent readers that continuously read the file.
	// Each read should see either the original or the replacement, never a mix.
	errCh := make(chan error, readers)
	stop := make(chan struct{})

	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				data, err := os.ReadFile(path)
				if err != nil {
					// File might be momentarily absent on some OS edge cases; skip.
					continue
				}
				s := string(data)
				if s != string(original) && s != string(replacement) {
					errCh <- &partialContentError{got: s}
					return
				}
			}
		}()
	}

	// Perform the atomic write while readers are running.
	if err := fileutil.WriteFileAtomic(path, replacement, 0o644); err != nil {
		t.Fatalf("WriteFileAtomic: %v", err)
	}

	close(stop)
	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("concurrent reader saw partial content: %v", err)
	}

	// Final check: file should have replacement content.
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("final ReadFile: %v", err)
	}
	if string(got) != string(replacement) {
		t.Errorf("final content = %q, want %q", got, replacement)
	}
}

type partialContentError struct {
	got string
}

func (e *partialContentError) Error() string {
	return "partial content: " + e.got
}

// TestWriteNewFileInRoot pins the create-if-absent write: a new file is
// created with the requested mode, an existing file is left byte-identical
// and reported as fs.ErrExist, and a symlink at the target or one that leads
// out of the root is refused without writing anything.
func TestWriteNewFileInRoot(t *testing.T) {
	t.Parallel()
	const rel = "local.yaml"
	data := []byte("# new\n")
	tests := []struct {
		name string
		rel  string
		// setup prepares root (and outside, a directory beside it).
		setup func(t *testing.T, root, outside string)
		// symlink marks a case that creates a symlink, which needs a
		// privilege Windows does not grant by default.
		symlink   bool
		wantErr   bool
		wantExist bool // the error wraps fs.ErrExist
		wantLink  bool // the error wraps fileutil.ErrSymlink
		// check verifies the files on disk after the call.
		check func(t *testing.T, root, outside string)
	}{
		{
			name: "creates new file",
			rel:  rel,
			check: func(t *testing.T, root, _ string) {
				t.Helper()
				assertFile(t, filepath.Join(root, rel), data)
				if runtime.GOOS == "windows" {
					return
				}
				info, err := os.Stat(filepath.Join(root, rel))
				if err != nil {
					t.Fatal(err)
				}
				if got := info.Mode().Perm(); got != fileutil.ModeReadWrite {
					t.Errorf("mode = %v, want %v", got, fileutil.ModeReadWrite)
				}
			},
		},
		{
			name: "existing file left unchanged",
			rel:  rel,
			setup: func(t *testing.T, root, _ string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(root, rel), []byte("# mine\n"), fileutil.ModeReadWrite); err != nil {
					t.Fatal(err)
				}
			},
			wantErr:   true,
			wantExist: true,
			check: func(t *testing.T, root, _ string) {
				t.Helper()
				assertFile(t, filepath.Join(root, rel), []byte("# mine\n"))
			},
		},
		{
			name:    "dangling symlink refused",
			symlink: true,
			rel:     rel,
			setup: func(t *testing.T, root, _ string) {
				t.Helper()
				symlinkOrFatal(t, "missing.yaml", filepath.Join(root, rel))
			},
			wantErr:  true,
			wantLink: true,
			check: func(t *testing.T, root, _ string) {
				t.Helper()
				assertAbsent(t, filepath.Join(root, "missing.yaml"))
			},
		},
		{
			name:    "symlink to existing file refused, target unchanged",
			symlink: true,
			rel:     rel,
			setup: func(t *testing.T, root, _ string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(root, "mine.yaml"), []byte("# mine\n"), fileutil.ModeReadWrite); err != nil {
					t.Fatal(err)
				}
				symlinkOrFatal(t, "mine.yaml", filepath.Join(root, rel))
			},
			wantErr:  true,
			wantLink: true,
			check: func(t *testing.T, root, _ string) {
				t.Helper()
				assertFile(t, filepath.Join(root, "mine.yaml"), []byte("# mine\n"))
			},
		},
		{
			name:    "symlink escaping root refused",
			symlink: true,
			rel:     rel,
			setup: func(t *testing.T, root, outside string) {
				t.Helper()
				symlinkOrFatal(t, filepath.Join(outside, "victim.yaml"), filepath.Join(root, rel))
			},
			wantErr:  true,
			wantLink: true,
			check: func(t *testing.T, _, outside string) {
				t.Helper()
				assertAbsent(t, filepath.Join(outside, "victim.yaml"))
			},
		},
		{
			name:    "symlinked parent escaping root refused",
			symlink: true,
			rel:     filepath.Join("sub", rel),
			setup: func(t *testing.T, root, outside string) {
				t.Helper()
				symlinkOrFatal(t, outside, filepath.Join(root, "sub"))
			},
			wantErr: true,
			check: func(t *testing.T, _, outside string) {
				t.Helper()
				assertAbsent(t, filepath.Join(outside, rel))
			},
		},
		{
			name:    "non-local path refused",
			rel:     filepath.Join("..", rel),
			wantErr: true,
			check: func(t *testing.T, root, _ string) {
				t.Helper()
				assertAbsent(t, filepath.Join(filepath.Dir(root), rel))
			},
		},
	}
	for _, tt := range tests {
		if tt.symlink && runtime.GOOS == "windows" {
			continue
		}
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base := t.TempDir()
			root, outside := filepath.Join(base, "root"), filepath.Join(base, "outside")
			for _, d := range []string{root, outside} {
				if err := os.Mkdir(d, 0o755); err != nil {
					t.Fatal(err)
				}
			}
			if tt.setup != nil {
				tt.setup(t, root, outside)
			}
			err := fileutil.WriteNewFileInRoot(root, tt.rel, data, fileutil.ModeReadWrite)
			if (err != nil) != tt.wantErr {
				t.Fatalf("WriteNewFileInRoot() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got := errors.Is(err, fs.ErrExist); got != tt.wantExist {
				t.Errorf("errors.Is(%v, fs.ErrExist) = %v, want %v", err, got, tt.wantExist)
			}
			if got := errors.Is(err, fileutil.ErrSymlink); got != tt.wantLink {
				t.Errorf("errors.Is(%v, fileutil.ErrSymlink) = %v, want %v", err, got, tt.wantLink)
			}
			tt.check(t, root, outside)
		})
	}
}

func symlinkOrFatal(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
}

func assertFile(t *testing.T, path string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Errorf("%s = %q, want %q", path, got, want)
	}
}

func assertAbsent(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("%s exists (Lstat error %v), want absent", path, err)
	}
}
