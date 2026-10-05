package shim

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLinkageHint(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		interp string
		want   string
	}{
		{"static", "", ""},
		{"nix store interpreter", "/nix/store/abc-glibc-2.40/lib/ld-linux-x86-64.so.2", ""},
		{"lib64 interpreter", "/lib64/ld-linux-x86-64.so.2",
			"qsdev at /opt/qsdev is dynamically linked against /lib64/ld-linux-x86-64.so.2, which is not visible inside the sandbox; rebuild with CGO_ENABLED=0"},
		{"nix store prefix only by name", "/nix/storefront/ld.so",
			"qsdev at /opt/qsdev is dynamically linked against /nix/storefront/ld.so, which is not visible inside the sandbox; rebuild with CGO_ENABLED=0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := linkageHint("/opt/qsdev", tt.interp); got != tt.want {
				t.Errorf("linkageHint(%q) = %q, want %q", tt.interp, got, tt.want)
			}
		})
	}
}

// TestELFInterp reads the interpreter of real files: the test binary (an ELF
// on Linux, static or not) and a file that is not an executable at all.
func TestELFInterp(t *testing.T) {
	t.Parallel()
	notELF := filepath.Join(t.TempDir(), "not-elf")
	if err := os.WriteFile(notELF, []byte("#!/bin/sh\n"), 0o600); err != nil { //nolint:gosec // test fixture
		t.Fatal(err)
	}
	if _, err := elfInterp(notELF); err == nil {
		t.Error("elfInterp accepted a non-ELF file")
	}
	if got := LinkageHint(notELF); got != "" {
		t.Errorf("LinkageHint(non-ELF) = %q, want empty", got)
	}
	if runtime.GOOS != "linux" {
		return
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	interp, err := elfInterp(exe)
	if err != nil {
		t.Fatalf("elfInterp(test binary): %v", err)
	}
	if interp != "" && !strings.HasPrefix(interp, "/") {
		t.Errorf("elfInterp(test binary) = %q, want empty or absolute", interp)
	}
	if got, want := LinkageHint(exe), linkageHint(exe, interp); got != want {
		t.Errorf("LinkageHint = %q, want %q", got, want)
	}
}

func TestHostExecutable(t *testing.T) {
	t.Parallel()
	got, err := HostExecutable()
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(got) {
		t.Errorf("HostExecutable = %q, want absolute", got)
	}
	if resolved, err := filepath.EvalSymlinks(got); err != nil || resolved != got {
		t.Errorf("HostExecutable = %q is not symlink-free (resolves to %q, %v)", got, resolved, err)
	}
}
