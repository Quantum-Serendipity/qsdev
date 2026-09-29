package instance

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/logging"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
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
	if catalog.ProjectRoot() == "" {
		t.Error("project defaults root not set")
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
	const stamped = "v9.8.7"
	build := exec.Command(goBin, "build", "-o", bin,
		"-ldflags", "-X "+VersionPackage+".version="+stamped, "./examples/downstream")
	build.Dir = moduleRoot
	build.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=vendor")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building examples/downstream: %v\n%s", err, out)
	}

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
