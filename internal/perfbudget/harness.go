package perfbudget

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

	"github.com/Quantum-Serendipity/qsdev/internal/procexec"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

const (
	// BinEnv names a prebuilt qsdev binary for the measurement suites to
	// use instead of building one.
	BinEnv = "QSDEV_PERF_BIN"
	// RequireNetNSEnv, when non-empty, makes a missing network namespace
	// fail the suite instead of measuring with the network reachable. CI
	// sets it, so a runner that cannot isolate fails loudly.
	RequireNetNSEnv = "QSDEV_PERF_REQUIRE_NETNS"

	qsdevPackage = "github.com/Quantum-Serendipity/qsdev/cmd/qsdev"
	noopPackage  = "github.com/Quantum-Serendipity/qsdev/internal/perfbudget/testdata/noop"
)

// Build puts the qsdev binary to measure in dir, under the name generated
// hooks invoke it by, and returns its path: a copy of the one BinEnv names,
// or a fresh build of cmd/qsdev. Put dir first on PATH (see IsolatedEnv) so
// hooks run that binary. The build inherits the caller's Go environment
// (GOFLAGS, GOWORK).
func Build(dir string) (string, error) {
	bin := executable(dir, "qsdev")
	if src := os.Getenv(BinEnv); src != "" {
		if err := copyExecutable(src, bin); err != nil {
			return "", fmt.Errorf("%s: %w", BinEnv, err)
		}
		return bin, nil
	}
	return bin, goBuild(qsdevPackage, bin)
}

// BuildNoop builds the empty Go program Calibrate times into dir and
// returns its path.
func BuildNoop(dir string) (string, error) {
	bin := executable(dir, "perfbudget-noop")
	return bin, goBuild(noopPackage, bin)
}

func executable(dir, name string) string {
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(dir, name)
}

func goBuild(pkg, bin string) error {
	out, err := procexec.Command("go", "build", "-o", bin, pkg).CombinedOutput()
	if err != nil {
		return fmt.Errorf("building %s: %w\n%s", pkg, err, out)
	}
	return nil
}

func copyExecutable(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening the binary: %w", err)
	}
	defer func() { err = errors.Join(err, in.Close()) }()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", filepath.Dir(dst), err)
	}
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return fmt.Errorf("creating %s: %w", dst, err)
	}
	defer func() { err = errors.Join(err, out.Close()) }()
	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copying the binary to %s: %w", dst, err)
	}
	return nil
}

// goProjectFiles is the module WriteGoProject writes: a library package and
// a command using it, enough for qsdev to detect the Go ecosystem.
var goProjectFiles = map[string]string{
	"go.mod":              "module example.com/app\n\ngo 1.23\n",
	"internal/app/app.go": "package app\n\n// Run runs the app.\nfunc Run() int { return 0 }\n",
	"cmd/app/main.go":     "package main\n\nimport \"example.com/app/internal/app\"\n\nfunc main() { app.Run() }\n",
}

// WriteGoProject writes a small Go module into dir, creating it: the
// project the measurement suites initialize. The corpus's file payloads
// name its files.
func WriteGoProject(dir string) error {
	for rel, content := range goProjectFiles {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return fmt.Errorf("creating %s: %w", filepath.Dir(p), err)
		}
		if err := fileutil.WriteFileAtomic(p, []byte(content), 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", p, err)
		}
	}
	return nil
}

// RunIn runs argv in dir with env, as a fixture step rather than a
// measurement, and fails t with its output when it fails.
func RunIn(t testing.TB, dir string, env []string, argv ...string) {
	t.Helper()
	if len(argv) == 0 {
		t.Fatalf("perfbudget: RunIn needs a command")
		return
	}
	cmd := procexec.Command(argv[0], argv[1:]...)
	cmd.Dir, cmd.Env = dir, env
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s: %v\n%s", strings.Join(argv, " "), err, out)
	}
}

// isolatedDir is a per-user directory a measured command gets under its own
// root, and the variables that name it on some OS.
type isolatedDir struct {
	dir  string
	vars []string
}

var isolatedDirs = []isolatedDir{
	{"home", []string{"HOME", "USERPROFILE"}},
	{"tmp", []string{"TMPDIR", "TMP", "TEMP"}},
	{"config", []string{"XDG_CONFIG_HOME"}},
	{"cache", []string{"XDG_CACHE_HOME"}},
	{"data", []string{"XDG_DATA_HOME"}},
	{"state", []string{"XDG_STATE_HOME"}},
}

// IsolatedEnv returns the environment for a hermetic measured command: the
// caller's environment with its home, temp and XDG directories replaced by
// fresh ones under root, binDir first on PATH, GIT_CONFIG_NOSYSTEM set, and
// every Claude Code, qsdev and git session variable removed, so neither the
// caller's configuration nor its project leaks into the measurement. It
// creates the directories. A caller measuring init adds QSDEV_SKIP_SETUP=1.
func IsolatedEnv(root, binDir string) ([]string, error) {
	env := slices.DeleteFunc(os.Environ(), func(kv string) bool {
		k, _, _ := strings.Cut(kv, "=")
		k = strings.ToUpper(k)
		return k == "PATH" || strings.HasPrefix(k, "XDG_") || strings.HasPrefix(k, "CLAUDE_") ||
			strings.HasPrefix(k, "QSDEV_") || strings.HasPrefix(k, "GIT_") ||
			slices.ContainsFunc(isolatedDirs, func(d isolatedDir) bool { return slices.Contains(d.vars, k) })
	})
	for _, d := range isolatedDirs {
		path := filepath.Join(root, d.dir)
		if err := os.MkdirAll(path, 0o755); err != nil {
			return nil, fmt.Errorf("creating the isolated %s directory: %w", d.dir, err)
		}
		for _, v := range d.vars {
			env = append(env, v+"="+path)
		}
	}
	return append(env, "PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"), "GIT_CONFIG_NOSYSTEM=1"), nil
}

// netNSArgs run a command in a fresh, empty network namespace as the
// invoking user: --map-current-user keeps the uid, so the command resolves
// the same home and permissions it would outside (-r would make it root).
// unshare execs the command in place, so its rusage is the command's.
var netNSArgs = []string{"--map-current-user", "--net"}

// probeNetNS resolves unshare and checks that it can make a network
// namespace here, returning the argv prefix that does. It resolves unshare
// to an absolute path, so the prefix works under an environment whose PATH
// lacks it.
var probeNetNS = sync.OnceValues(func() ([]string, error) {
	unshare, err := procexec.LookPath("unshare")
	if err != nil {
		return nil, fmt.Errorf("finding unshare: %w", err)
	}
	argv := append([]string{unshare}, netNSArgs...)
	if out, err := procexec.Command(unshare, append(slices.Clone(netNSArgs), "true")...).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("%s true: %w: %s", strings.Join(argv, " "), err, strings.TrimSpace(string(out)))
	}
	return argv, nil
})

// NetNS returns the argv prefix that runs a measured command without
// network access, so a command that blocks on the network fails fast
// rather than stalling the measurement. Where no network namespace can be
// made (not Linux, no unshare, or unprivileged user namespaces restricted)
// it logs why and returns nil, unless RequireNetNSEnv is set: then it fails
// t.
func NetNS(t testing.TB) []string {
	t.Helper()
	return netNS(t, os.Getenv(RequireNetNSEnv) != "", probeNetNS)
}

func netNS(t testing.TB, require bool, probe func() ([]string, error)) []string {
	t.Helper()
	prefix, err := probe()
	if err == nil {
		return slices.Clone(prefix)
	}
	if require {
		t.Fatalf("perfbudget: %s is set but no network namespace can be made: %v", RequireNetNSEnv, err)
		return nil
	}
	t.Logf("perfbudget: measuring with the network reachable; no network namespace: %v", err)
	return nil
}
