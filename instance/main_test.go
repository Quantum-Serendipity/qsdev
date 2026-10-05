package instance

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/logging"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/internal/projectctx"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

func TestStandardCommands(t *testing.T) {
	t.Parallel()

	logsCmd := &cobra.Command{Use: "logs"}
	cmds := standardCommands(logsCmd)
	if !slices.Contains(cmds, logsCmd) {
		t.Error("standard commands do not include the runtime's logs command")
	}
	var names []string
	for _, c := range cmds {
		names = append(names, c.Name())
	}
	for _, want := range []string{"self-update", "logs", "report"} {
		if !slices.Contains(names, want) {
			t.Errorf("standard commands %q missing %q", names, want)
		}
	}
}

// TestDefaultRuntime proves DefaultRuntime installs the wiring downstream
// tools previously had to copy from qsdev's package main, and installs it once.
func TestDefaultRuntime(t *testing.T) {
	b := branding.Get()
	t.Setenv(b.EnvNoUpdate, "1")
	t.Setenv(b.EnvLogVar, "")
	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"app", "--debug", "status"}
	catalog.ResetDefault()
	t.Cleanup(catalog.ResetDefault)

	rt := DefaultRuntime()
	if again := DefaultRuntime(); again != rt {
		t.Error("DefaultRuntime installed a second runtime")
	}

	if !slices.Equal(os.Args, []string{"app", "status"}) {
		t.Errorf("os.Args = %q, want --debug consumed", os.Args)
	}
	if got := os.Getenv(b.EnvLogVar); got != "debug" {
		t.Errorf("%s = %q, want debug", b.EnvLogVar, got)
	}
	if n := len(spi.DefaultRegistry().All()); n == 0 {
		t.Error("no framework adapters registered")
	}
	// The project defaults layer comes from the executing command's resolved
	// root (Runtime.initCommand), never from a walk before argv is parsed.
	if root := catalog.ProjectRoot(); root != "" {
		t.Errorf("DefaultRuntime set the project defaults root %q before any command was resolved", root)
	}
	if rt.logsCmd == nil || rt.logsCmd.Name() != "logs" {
		t.Errorf("runtime logs command = %v, want logs", rt.logsCmd)
	}

	rt.Finish()
	rt.Finish() // idempotent
}

// TestDownstreamExampleGetsDefaultRuntime builds examples/downstream, which
// imports only public packages, and proves it ships the standard commands,
// reports the version stamped into VersionPackage and writes a redacted
// session log under its own branding.
func TestDownstreamExampleGetsDefaultRuntime(t *testing.T) {
	const stamped = "v9.8.7"
	bin := buildDownstream(t, stamped)

	home := t.TempDir()
	logDir := t.TempDir()
	env := append(os.Environ(),
		"HOME="+home,
		// Windows resolves the home and config dirs from these instead of HOME.
		"USERPROFILE="+home,
		"APPDATA="+filepath.Join(home, "AppData", "Roaming"),
		"LOCALAPPDATA="+filepath.Join(home, "AppData", "Local"),
		"XDG_CONFIG_HOME="+filepath.Join(home, ".config"),
		"ACMEDEV_LOG_DIR="+logDir,
		"ACMEDEV_NO_UPDATE_CHECK=1",
	)
	run := func(args ...string) string {
		t.Helper()
		c := exec.Command(bin, args...)
		c.Dir = home
		c.Env = env
		out, _ := c.CombinedOutput()
		return string(out)
	}

	help := run("--help")
	for _, want := range []string{"logs", "report"} {
		if !strings.Contains(help, "\n  "+want+" ") {
			t.Errorf("acmedev --help does not list %q:\n%s", want, help)
		}
	}
	// self-update is hidden behind the "update" umbrella but still registered.
	if out := run("self-update", "--help"); !strings.Contains(out, "acmedev self-update") {
		t.Errorf("acmedev has no self-update command:\n%s", out)
	}

	// The release version stamped into VersionPackage is what the tool reports.
	if out := run("version"); !strings.Contains(out, stamped) {
		t.Errorf("acmedev version does not report the stamped %s:\n%s", stamped, out)
	}

	secret := "AKIA" + "ABCDEFGHIJKLMNOP" // split so secret scanners skip the fixture
	run("version", secret)
	logs, err := filepath.Glob(filepath.Join(logDir, "acmedev-*.jsonl"))
	if err != nil || len(logs) == 0 {
		t.Fatalf("acmedev wrote no session log to %s (err=%v)", logDir, err)
	}
	// Each invocation writes its own session log; check them together.
	var content []byte
	for _, l := range logs {
		b, err := os.ReadFile(l)
		if err != nil {
			t.Fatal(err)
		}
		content = append(content, b...)
	}
	if !strings.Contains(string(content), `"command starting"`) {
		t.Errorf("session log lacks the opening record:\n%s", content)
	}
	if strings.Contains(string(content), secret) || !strings.Contains(string(content), logging.RedactionMarker) {
		t.Errorf("session log is not redacted:\n%s", content)
	}
}

// buildDownstream builds examples/downstream stamped with version and
// returns the binary's path. It skips the test in -short mode or without a
// go toolchain.
func buildDownstream(t *testing.T, version string) string {
	t.Helper()
	if testing.Short() {
		t.Skip("builds a binary")
	}
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go toolchain not on PATH")
	}
	moduleRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	// Windows only executes (and exec.LookPath only resolves) files carrying
	// an executable extension, so the built binary must be named acmedev.exe.
	exeName := "acmedev"
	if runtime.GOOS == "windows" {
		exeName += ".exe"
	}
	bin := filepath.Join(t.TempDir(), exeName)
	build := exec.Command(goBin, "build", "-o", bin,
		"-ldflags", "-X "+VersionPackage+".version="+version, "./examples/downstream")
	build.Dir = moduleRoot
	build.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=vendor")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building examples/downstream: %v\n%s", err, out)
	}
	return bin
}

// TestDownstreamExample_NoUpdateNoticeWhenPiped pins the XS-WS1 A6 wiring
// in installDefaultRuntime: a real binary whose stderr is a pipe (as for a
// hook, an MCP server or CI) prints no update notice and leaves the cache
// untouched, even though the cache is fresh and names a newer release and the
// opt-out variable is unset.
func TestDownstreamExample_NoUpdateNoticeWhenPiped(t *testing.T) {
	bin := buildDownstream(t, "v0.1.0")
	home := t.TempDir()
	dir := filepath.Join(home, ".cache", "acmedev")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	data, err := json.Marshal(map[string]any{
		"checked_at": now, "attempted_at": now, "version": "9.9.9",
		"url": "https://example.invalid/releases/v9.9.9", "owner": "acme-corp", "repo": "acmedev",
	})
	if err != nil {
		t.Fatal(err)
	}
	cache := filepath.Join(dir, "update-check.json")
	if err := os.WriteFile(cache, data, 0o644); err != nil {
		t.Fatal(err)
	}

	var env []string
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "ACMEDEV_") {
			env = append(env, kv)
		}
	}
	c := exec.Command(bin, "version")
	c.Dir = home
	c.Env = append(env,
		"HOME="+home,
		"USERPROFILE="+home,
		"APPDATA="+filepath.Join(home, "AppData", "Roaming"),
		"LOCALAPPDATA="+filepath.Join(home, "AppData", "Local"),
		"XDG_CONFIG_HOME="+filepath.Join(home, ".config"),
		"XDG_CACHE_HOME="+filepath.Join(home, ".cache"),
		"XDG_STATE_HOME="+filepath.Join(home, ".local", "state"),
		"ACMEDEV_LOG_DIR="+t.TempDir(),
	)
	var stderr bytes.Buffer
	c.Stderr = &stderr // a pipe, not a terminal
	if err := c.Run(); err != nil {
		t.Fatalf("acmedev version: %v\n%s", err, stderr.String())
	}
	if strings.Contains(stderr.String(), "9.9.9") || strings.Contains(stderr.String(), "new version") {
		t.Errorf("piped stderr got an update notice:\n%s", stderr.String())
	}
	after, err := os.ReadFile(cache)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, after) {
		t.Errorf("cache changed:\nbefore %s\nafter  %s", data, after)
	}
}

// TestStartUpdateCheck_NonInteractive pins XS-WS1 A6: a non-interactive
// process (hook, MCP server, CI) starts no update check, so it neither prints
// a notice nor touches the cache (no request is attempted), whether the cache
// is fresh and names a newer version or has expired.
func TestStartUpdateCheck_NonInteractive(t *testing.T) {
	b := branding.Get()
	now := time.Now().UTC()
	old := now.Add(-30 * 24 * time.Hour)
	tests := []struct {
		name      string
		checkedAt time.Time
	}{
		{name: "fresh cache naming a newer version", checkedAt: now},
		{name: "expired cache", checkedAt: old},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)
			t.Setenv(b.EnvNoUpdate, "")
			cache := writeUpdateCache(t, home, tt.checkedAt)
			before, err := os.ReadFile(cache)
			if err != nil {
				t.Fatalf("reading cache: %v", err)
			}

			if ch := startUpdateCheck(false, "0.1.0"); ch != nil {
				t.Error("startUpdateCheck(non-interactive) returned a channel, want nil")
			}

			after, err := os.ReadFile(cache)
			if err != nil {
				t.Fatalf("re-reading cache: %v", err)
			}
			if !bytes.Equal(before, after) {
				t.Errorf("cache changed:\nbefore %s\nafter  %s", before, after)
			}
		})
	}

	// Control: the same fresh cache does yield a notice for an interactive
	// process (served from the cache, so no request is made).
	t.Run("interactive control", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		t.Setenv("USERPROFILE", home)
		t.Setenv(b.EnvNoUpdate, "")
		writeUpdateCache(t, home, now)

		ch := startUpdateCheck(true, "0.1.0")
		if ch == nil {
			t.Fatal("startUpdateCheck(interactive) returned nil")
		}
		select {
		case notice := <-ch:
			if !strings.Contains(notice, "9.9.9") {
				t.Errorf("notice = %q, want it to name 9.9.9", notice)
			}
		case <-time.After(10 * time.Second):
			t.Fatal("timed out waiting for the cached notice")
		}
	})
}

// writeUpdateCache points the per-user cache directory below home and writes
// an update-check cache there naming v9.9.9, checked (and attempted) at
// checkedAt, and returns its path.
func writeUpdateCache(t *testing.T, home string, checkedAt time.Time) string {
	t.Helper()
	b := branding.Get()
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	dirs, err := projectctx.UserDirs()
	if err != nil {
		t.Fatalf("locating the cache dir: %v", err)
	}
	dir := dirs.Cache
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("creating cache dir: %v", err)
	}
	data, err := json.Marshal(map[string]any{
		"checked_at":   checkedAt,
		"attempted_at": checkedAt,
		"version":      "9.9.9",
		"url":          "https://example.invalid/releases/v9.9.9",
		"owner":        b.GitHubOwner,
		"repo":         b.GitHubRepo,
	})
	if err != nil {
		t.Fatalf("encoding cache: %v", err)
	}
	path := filepath.Join(dir, "update-check.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("writing cache: %v", err)
	}
	return path
}
