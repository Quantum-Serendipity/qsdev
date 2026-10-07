package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// The qsdev processes the guardrail tests start look programs up on PATH,
// so a PATH taken from the host makes their outcome depend on the machine
// running the tests:
//
//   - check requires the program every generated hook runs, qsdev itself,
//     to resolve on PATH (claude_hook_unresolvable). With the host PATH it
//     passes only where a qsdev is installed, and then vouches for that
//     binary instead of the build under test.
//   - project detection probes the container runtimes (container.Detect
//     runs `podman info`, `docker --version` and `docker compose version`).
//     A host podman runs against the isolated HOME, and one killed at the
//     probe timeout leaves a mode-000 overlay work directory there that the
//     test's TempDir cleanup cannot remove.
//
// guardrailBinDir is put first on the PATH of those processes: qsdev in it
// is the test binary, which runs as qsdev (see cliHelperEnv), and the
// container runtimes in it are stubs that always fail, as an unusable
// installation does, so no probe reaches the host's.

// stubbedPrograms are the programs that resolve to a failing stub in
// guardrailBinDir: the container runtimes container.Detect runs.
var stubbedPrograms = []string{"podman", "docker"}

// stubbedProgram returns the stubbed program this process was started as,
// when it was started through a guardrailBinDir stub.
func stubbedProgram() (string, bool) {
	name := strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe")
	return name, slices.Contains(stubbedPrograms, name)
}

// runStubbedProgram is the whole of a stubbed program: it fails without
// doing anything.
func runStubbedProgram(name string) int {
	fmt.Fprintf(os.Stderr, "%s: stubbed out for the qsdev guardrail tests\n", name)
	return 1
}

var (
	guardrailBinOnce sync.Once
	guardrailBin     string
	guardrailBinErr  error
)

// guardrailBinDir returns the directory holding the test binary as qsdev
// and as each stubbed program. It is made once per test process, as on
// Windows each is a copy of the test binary, and removed by
// removeGuardrailBinDir once the tests are done: a file the test binary
// runs from cannot be deleted there while it runs, so no test's TempDir
// can hold it.
func guardrailBinDir(t *testing.T) string {
	t.Helper()
	guardrailBinOnce.Do(func() {
		guardrailBin, guardrailBinErr = makeGuardrailBinDir()
	})
	if guardrailBinErr != nil {
		t.Fatalf("preparing the guardrail PATH: %v", guardrailBinErr)
	}
	return guardrailBin
}

func makeGuardrailBinDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("locating the test binary: %w", err)
	}
	dir, err := os.MkdirTemp("", "qsdev-guardrail-bin-")
	if err != nil {
		return "", fmt.Errorf("creating the bin dir: %w", err)
	}
	for _, name := range append([]string{branding.Get().AppName}, stubbedPrograms...) {
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		if err := linkExecutable(exe, filepath.Join(dir, name)); err != nil {
			return dir, fmt.Errorf("placing %s in %s: %w", name, dir, err)
		}
	}
	return dir, nil
}

// removeGuardrailBinDir deletes guardrailBinDir's directory; TestMain calls
// it once the tests are done.
func removeGuardrailBinDir() {
	if guardrailBin != "" {
		_ = os.RemoveAll(guardrailBin)
	}
}

// linkExecutable makes dst run src: a hard link, else a symbolic link, else
// a copy (a hard link cannot cross devices, and a symbolic link needs a
// privilege on Windows). On Windows it always copies, as a link to the
// running test binary could not be deleted.
func linkExecutable(src, dst string) error {
	if runtime.GOOS == "windows" {
		return copyExecutable(src, dst)
	}
	linkErr := os.Link(src, dst)
	if linkErr == nil {
		return nil
	}
	symlinkErr := os.Symlink(src, dst)
	if symlinkErr == nil {
		return nil
	}
	if err := copyExecutable(src, dst); err != nil {
		return errors.Join(linkErr, symlinkErr, err)
	}
	return nil
}

// copyExecutable copies src to a new executable file dst.
func copyExecutable(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening %s: %w", src, err)
	}
	defer func() { _ = in.Close() }()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o755) //nolint:gosec // a test-only executable in a fresh temp dir
	if err != nil {
		return fmt.Errorf("creating %s: %w", dst, err)
	}
	defer func() {
		if cerr := out.Close(); err == nil && cerr != nil {
			err = fmt.Errorf("closing %s: %w", dst, cerr)
		}
	}()
	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copying %s to %s: %w", src, dst, err)
	}
	return nil
}
