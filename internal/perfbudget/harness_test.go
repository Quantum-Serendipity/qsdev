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
// directories under root, the binary directory first on PATH, system git
// configuration off, and none of the caller's Claude Code, qsdev or git
// session variables. It sets the caller's environment, so it is serial.
func TestIsolatedEnv(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "caller-config"))
	t.Setenv("CLAUDE_PROJECT_DIR", "/caller/project")
	t.Setenv("QSDEV_TEST_FORBID_EXEC", "1")
	t.Setenv("GIT_DIR", "/caller/.git")
	t.Setenv("PERFBUDGET_KEPT", "kept")
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")

	env, err := IsolatedEnv(root, binDir)
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
	if p := vars["PATH"]; len(p) != 1 || !strings.HasPrefix(p[0], binDir+string(os.PathListSeparator)) {
		t.Errorf("PATH = %q, want one entry starting with %s", p, binDir)
	}
	if v := vars["GIT_CONFIG_NOSYSTEM"]; !slices.Equal(v, []string{"1"}) {
		t.Errorf("GIT_CONFIG_NOSYSTEM = %q, want 1", v)
	}
	if v := vars["PERFBUDGET_KEPT"]; !slices.Equal(v, []string{"kept"}) {
		t.Errorf("PERFBUDGET_KEPT = %q, want the caller's other variables kept", v)
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
