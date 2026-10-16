package perfbudget

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// TestIsolatedEnv: a measured command sees its own HOME, temp and XDG
// directories under root, exactly the PATH it is given, system git
// configuration off, and none of the caller's Claude Code, qsdev or git
// session variables. It sets the caller's environment, so it is serial.
func TestIsolatedEnv(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "caller-config"))
	t.Setenv("CLAUDE_PROJECT_DIR", "/caller/project")
	t.Setenv("QSDEV_TEST_FORBID_EXEC", "1")
	t.Setenv("GIT_DIR", "/caller/.git")
	t.Setenv("PERFBUDGET_KEPT", "kept")
	root := t.TempDir()
	path := strings.Join([]string{filepath.Join(root, "bin"), filepath.Join(root, "stubs")}, string(os.PathListSeparator))

	env, err := IsolatedEnv(root, path)
	if err != nil {
		t.Fatalf("IsolatedEnv: %v", err)
	}
	vars := map[string][]string{}
	for _, kv := range env {
		k, v, _ := strings.Cut(kv, "=")
		vars[strings.ToUpper(k)] = append(vars[strings.ToUpper(k)], v)
	}
	for _, k := range []string{"HOME", "USERPROFILE", "TMPDIR", "TMP", "TEMP", "XDG_CONFIG_HOME", "XDG_CACHE_HOME", "XDG_DATA_HOME", "XDG_STATE_HOME"} {
		vs := vars[k]
		if len(vs) != 1 {
			t.Errorf("%s set %d times (%q), want once", k, len(vs), vs)
			continue
		}
		if rel, err := filepath.Rel(root, vs[0]); err != nil || strings.HasPrefix(rel, "..") {
			t.Errorf("%s = %q, want a directory under %s", k, vs[0], root)
		}
		if fi, err := os.Stat(vs[0]); err != nil || !fi.IsDir() {
			t.Errorf("%s = %q does not exist as a directory: %v", k, vs[0], err)
		}
	}
	for _, k := range []string{"CLAUDE_PROJECT_DIR", "QSDEV_TEST_FORBID_EXEC", "GIT_DIR"} {
		if vs, ok := vars[k]; ok {
			t.Errorf("%s = %q leaked from the caller", k, vs)
		}
	}
	if p := vars["PATH"]; !slices.Equal(p, []string{path}) {
		t.Errorf("PATH = %q, want exactly %q", p, path)
	}
	if v := vars["GIT_CONFIG_NOSYSTEM"]; !slices.Equal(v, []string{"1"}) {
		t.Errorf("GIT_CONFIG_NOSYSTEM = %q, want 1", v)
	}
	if v := vars["PERFBUDGET_KEPT"]; !slices.Equal(v, []string{"kept"}) {
		t.Errorf("PERFBUDGET_KEPT = %q, want the caller's other variables kept", v)
	}
}

// TestHostPath: a measured host program finds the binary under test first,
// then the caller's PATH.
func TestHostPath(t *testing.T) {
	t.Setenv("PATH", strings.Join([]string{"/caller/a", "/caller/b"}, string(os.PathListSeparator)))
	want := strings.Join([]string{"/bin-dir", "/caller/a", "/caller/b"}, string(os.PathListSeparator))
	if got := HostPath("/bin-dir"); got != want {
		t.Errorf("HostPath = %q, want %q", got, want)
	}
}

// requireSymlinks skips t where this host cannot create symbolic links
// (Windows without the privilege), which HermeticPath links tools with.
func requireSymlinks(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	if err := os.Symlink(filepath.Join(dir, "target"), filepath.Join(dir, "link")); err != nil {
		t.Skipf("cannot create symbolic links here: %v", err)
	}
}

// TestHermeticPath: the PATH holds the binary directory, the host tools the
// fixture needs, linked to the host's, and a stub for every other probed
// tool, and nothing else: none of the caller's PATH directories and no
// other name.
func TestHermeticPath(t *testing.T) {
	t.Parallel()
	requireSymlinks(t)
	hostDir := t.TempDir()
	lookPath := func(name string) (string, error) {
		if name == "missing" {
			return "", errors.New("not found")
		}
		return filepath.Join(hostDir, name), nil
	}
	callerPath := filepath.SplitList(os.Getenv("PATH"))
	binDir := filepath.Join(hostDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(executable(binDir, "qsdev"), nil, 0o755); err != nil { //nolint:gosec // stands in for the binary under test
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		host      []string
		probed    []string
		wantHost  []string
		wantStubs []string
		wantErr   string
	}{
		{
			name:      "stubs every probed tool but the host ones, once",
			host:      []string{"git"},
			probed:    []string{"nix", "git", "docker", "nix", "ps"},
			wantHost:  []string{"git"},
			wantStubs: []string{"docker", "nix", "ps"},
		},
		{
			name:      "the binary under test probing itself is not stubbed",
			probed:    []string{"qsdev", "nix"},
			wantStubs: []string{"nix"},
		},
		{name: "nothing probed", host: []string{"git"}, wantHost: []string{"git"}},
		{name: "host tool named twice", host: []string{"git", "git"}, probed: []string{"git"}, wantHost: []string{"git"}},
		{name: "host tool missing", host: []string{"missing"}, wantErr: `host tool "missing"`},
		{name: "probed name with a separator", probed: []string{"../escape"}, wantErr: "not a bare program name"},
		{name: "probed name with a backslash", probed: []string{`a\b`}, wantErr: "not a bare program name"},
		{name: "drive-relative name", probed: []string{"C:x"}, wantErr: "not a bare program name"},
		{name: "name holding a PATH separator", host: []string{"a;b"}, wantErr: "not a bare program name"},
		{name: "dot", probed: []string{"."}, wantErr: "not a bare program name"},
		{name: "dot dot", probed: []string{".."}, wantErr: "not a bare program name"},
		{name: "empty probed name", probed: []string{""}, wantErr: "not a bare program name"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := filepath.Join(t.TempDir(), "path")
			tools := HermeticTools{Bin: binDir, Host: tt.host, Probed: tt.probed, Stub: filepath.Join(hostDir, "noop")}
			path, err := hermeticPath(dir, tools, lookPath)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("hermeticPath = %q, %v; want an error containing %q", path, err, tt.wantErr)
				}
				if _, statErr := os.Stat(dir); statErr == nil && strings.Contains(tt.wantErr, "bare program name") {
					t.Errorf("hermeticPath created %s before rejecting a name", dir)
				}
				return
			}
			if err != nil {
				t.Fatalf("hermeticPath: %v", err)
			}
			host, stubs := filepath.Join(dir, hostToolsDir), filepath.Join(dir, stubToolsDir)
			entries := filepath.SplitList(path)
			if want := []string{tools.Bin, host, stubs}; !slices.Equal(entries, want) {
				t.Errorf("PATH = %q, want %q", entries, want)
			}
			for _, e := range entries {
				if slices.Contains(callerPath, e) {
					t.Errorf("PATH holds the caller's PATH directory %s", e)
				}
			}
			checkLinks(t, host, tt.wantHost, func(name string) string { return filepath.Join(hostDir, name) })
			checkLinks(t, stubs, tt.wantStubs, func(string) string { return tools.Stub })
			if again, err := hermeticPath(dir, tools, lookPath); err == nil {
				t.Errorf("a second hermeticPath into %s = %q, want an error: it is single use", dir, again)
			}
		})
	}
}

// checkLinks fails t unless dir holds exactly the tools names, each a link
// to target(name).
func checkLinks(t *testing.T, dir string, names []string, target func(string) string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(entries))
	for _, e := range entries {
		got = append(got, strings.TrimSuffix(e.Name(), ".exe"))
	}
	if !slices.Equal(got, names) {
		t.Errorf("%s holds %q, want exactly %q", filepath.Base(dir), got, names)
	}
	for _, name := range names {
		link := executable(dir, name)
		if dest, err := os.Readlink(link); err != nil || dest != target(name) {
			t.Errorf("%s links to %q (%v), want %q", link, dest, err, target(name))
		}
	}
}

// fatalRecorder is a recorder that also records Fatalf instead of ending
// the test.
type fatalRecorder struct {
	recorder
	fatals []string
}

func (r *fatalRecorder) Fatalf(format string, args ...any) {
	r.fatals = append(r.fatals, fmt.Sprintf(format, args...))
}

// TestNetNSRequire: commands run in a fresh network namespace when one can
// be made; otherwise they run unwrapped with a log line, unless the
// namespace is required, which fails.
func TestNetNSRequire(t *testing.T) {
	t.Parallel()
	prefix := []string{"/run/unshare", "--map-current-user", "--net"}
	works := func() ([]string, error) { return prefix, nil }
	broken := func() ([]string, error) { return nil, errors.New("unprivileged user namespaces are restricted") }

	tests := []struct {
		name    string
		require bool
		probe   func() ([]string, error)
		want    []string
		fatal   bool
		logged  bool
	}{
		{name: "available", require: true, probe: works, want: prefix},
		{name: "available, optional", probe: works, want: prefix},
		{name: "unavailable, optional", probe: broken, logged: true},
		{name: "unavailable, required", require: true, probe: broken, fatal: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := &fatalRecorder{recorder: recorder{TB: t}}
			got := netNS(r, tt.require, tt.probe)
			if !slices.Equal(got, tt.want) {
				t.Errorf("netNS = %q, want %q", got, tt.want)
			}
			if fatal := len(r.fatals) > 0; fatal != tt.fatal {
				t.Errorf("fatals = %q, want fatal %v", r.fatals, tt.fatal)
			}
			if msg := strings.Join(r.fatals, "\n"); tt.fatal && (!strings.Contains(msg, RequireNetNSEnv) || !strings.Contains(msg, "restricted")) {
				t.Errorf("fatal %q does not name %s and the cause", msg, RequireNetNSEnv)
			}
			if logged := len(r.logs) > 0; logged != tt.logged {
				t.Errorf("logs = %q, want logged %v", r.logs, tt.logged)
			}
		})
	}
}

// TestBuildPerfBin copies the binary QSDEV_PERF_BIN names into dir under
// the name hooks invoke, so hooks run it whatever it is called, and rejects
// one that does not exist.
func TestBuildPerfBin(t *testing.T) {
	src := filepath.Join(t.TempDir(), "qsdev-scratch-build")
	if err := os.WriteFile(src, []byte("stub binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(BinEnv, src)
	dir := filepath.Join(t.TempDir(), "bin")
	got, err := Build(dir)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if name := strings.TrimSuffix(filepath.Base(got), ".exe"); filepath.Dir(got) != dir || name != "qsdev" {
		t.Errorf("Build = %q, want qsdev in %s", got, dir)
	}
	if data, err := os.ReadFile(got); err != nil || string(data) != "stub binary" {
		t.Errorf("Build's binary holds %q (%v), want a copy of %s", data, err, src)
	}
	t.Setenv(BinEnv, src+"-missing")
	if _, err := Build(t.TempDir()); err == nil || !strings.Contains(err.Error(), BinEnv) {
		t.Errorf("Build with a missing %s = %v, want an error naming it", BinEnv, err)
	}
}

// TestWriteGoProject writes the module the corpus's file payloads name.
func TestWriteGoProject(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "project")
	if err := WriteGoProject(dir); err != nil {
		t.Fatalf("WriteGoProject: %v", err)
	}
	for rel, want := range goProjectFiles {
		if got, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel))); err != nil || string(got) != want {
			t.Errorf("%s = %q (%v), want %q", rel, got, err, want)
		}
	}
	for _, p := range Payloads() {
		if f, ok := p.Input(dir)["file_path"].(string); ok && p.Tool != "Write" {
			if _, err := os.Stat(f); err != nil {
				t.Errorf("payload %s names %s, which the project lacks: %v", p.Name, f, err)
			}
		}
	}
}

// TestRunIn: a fixture step that succeeds leaves t alone; one that fails,
// or names no command, fails t with the command and its output.
func TestRunIn(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		mode      string // helper mode, "" for no command
		wantFatal []string
	}{
		{name: "succeeds", mode: "exit:0"},
		{name: "fails", mode: "exit:3", wantFatal: []string{"exit status 3", "helper stderr"}},
		{name: "no command", wantFatal: []string{"needs a command"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := &fatalRecorder{recorder: recorder{TB: t}}
			var argv []string
			var env []string
			if tt.mode != "" {
				spec := helperSpec(t, tt.mode)
				argv, env = spec.Argv, spec.Env
			}
			RunIn(r, t.TempDir(), env, argv...)
			msg := strings.Join(r.fatals, "\n")
			if (len(r.fatals) > 0) != (tt.wantFatal != nil) {
				t.Fatalf("fatals = %q, want fatal %v", r.fatals, tt.wantFatal != nil)
			}
			for _, w := range tt.wantFatal {
				if !strings.Contains(msg, w) {
					t.Errorf("fatal %q does not contain %q", msg, w)
				}
			}
		})
	}
}
