package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/canon"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// openPty returns a fresh pseudo-terminal pair from /dev/ptmx, built with
// x/sys/unix rather than a host tool such as script(1).
func openPty(t *testing.T) (master, slave *os.File) {
	t.Helper()
	mfd, err := unix.Open("/dev/ptmx", unix.O_RDWR|unix.O_NOCTTY|unix.O_CLOEXEC, 0)
	if err != nil {
		t.Fatalf("opening /dev/ptmx: %v", err)
	}
	master = os.NewFile(uintptr(mfd), "/dev/ptmx")
	if err := unix.IoctlSetPointerInt(mfd, unix.TIOCSPTLCK, 0); err != nil {
		_ = master.Close()
		t.Fatalf("unlocking the pty: %v", err)
	}
	n, err := unix.IoctlGetUint32(mfd, unix.TIOCGPTN)
	if err != nil {
		_ = master.Close()
		t.Fatalf("reading the pty number: %v", err)
	}
	name := "/dev/pts/" + strconv.FormatUint(uint64(n), 10)
	sfd, err := unix.Open(name, unix.O_RDWR|unix.O_NOCTTY|unix.O_CLOEXEC, 0)
	if err != nil {
		_ = master.Close()
		t.Fatalf("opening %s: %v", name, err)
	}
	return master, os.NewFile(uintptr(sfd), name)
}

// challengeRE matches the human gate's confirmation prompt and its code.
var challengeRE = regexp.MustCompile(`type (\d{6}) and press Enter`)

// ptyTranscript is what a session on the pty's master side read; answer, if
// set, is typed once the gate shows its confirmation code.
type ptyTranscript struct {
	mu       sync.Mutex
	buf      bytes.Buffer
	answer   func(code string) string
	answered bool
}

// pump reads the master side until the pty hangs up, typing the answer to a
// challenge it sees.
func (p *ptyTranscript) pump(master *os.File) {
	chunk := make([]byte, 4096)
	for {
		n, err := master.Read(chunk)
		p.mu.Lock()
		p.buf.Write(chunk[:n])
		var reply string
		if p.answer != nil && !p.answered {
			if m := challengeRE.FindSubmatch(p.buf.Bytes()); m != nil {
				p.answered = true
				reply = p.answer(string(m[1]))
			}
		}
		p.mu.Unlock()
		if reply != "" {
			_, _ = master.WriteString(reply)
		}
		if err != nil {
			return
		}
	}
}

func (p *ptyTranscript) String() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.buf.String()
}

// runShOnPty runs `sh -c script` in dir as the leader of a new session whose
// controlling terminal, stdin, stdout and stderr are a fresh pty, and returns
// sh's pid, what the pty showed and sh's exit code. answer, if not nil,
// types its reply to the human gate's confirmation code.
func runShOnPty(t *testing.T, env []string, dir, script string, answer func(code string) string) (int, string, int) {
	t.Helper()
	master, slave := openPty(t)
	defer func() { _ = master.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", script)
	cmd.Dir = dir
	cmd.Env = env
	cmd.Stdin, cmd.Stdout, cmd.Stderr = slave, slave, slave
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true, Ctty: 0}
	cmd.Cancel = func() error { return unix.Kill(-cmd.Process.Pid, unix.SIGKILL) }
	if err := cmd.Start(); err != nil {
		_ = slave.Close()
		t.Fatalf("starting sh on the pty: %v", err)
	}
	// Only the session holds the slave now, so the master reads EIO once
	// every process on the pty has exited.
	_ = slave.Close()
	transcript := &ptyTranscript{answer: answer}
	done := make(chan struct{})
	go func() {
		defer close(done)
		transcript.pump(master)
	}()
	err := cmd.Wait()
	<-done
	var exitErr *exec.ExitError
	switch {
	case err == nil:
		return cmd.Process.Pid, transcript.String(), 0
	case errors.As(err, &exitErr):
		return cmd.Process.Pid, transcript.String(), exitErr.ExitCode()
	}
	t.Fatalf("sh -c %q on a pty: %v\n%s", script, err, transcript.String())
	return 0, "", -1
}

// shQuote quotes s as one POSIX shell word.
func shQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// withDefaultsFile writes the defaults file that `defaults reset` removes,
// below env's HOME, and returns env naming it as the org overlay, with its
// path and content. A test binary has no home overlay of its own
// (catalog.OrgConfigPath), so the variable is what points reset at it.
func withDefaultsFile(t *testing.T, env []string) (outEnv []string, path, content string) {
	t.Helper()
	b := branding.Get()
	path = filepath.Join(envValue(env, "HOME"), ".config", b.AppName, "defaults.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	content = "# the organisation's guardrail overlay\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return append(env, b.EnvPrefix+"ORG_CONFIG="+path), path, content
}

// TestHumanGate_EnvUnsetPtyRefused is the U18-WS6 E3 test of the binary's
// human gate (Linux). Inside an agent session (CLAUDECODE=1), stripping the
// markers with `env -u` or `unset` and running qsdev on a real pseudo-
// terminal is still refused: the agent's shell keeps the marker in the
// environment it started with, which the gate reads from /proc. The trailing
// `exit $?` keeps sh from exec'ing qsdev in its place, so sh stays its
// marker-bearing parent. The defaults file stays as it was.
func TestHumanGate_EnvUnsetPtyRefused(t *testing.T) {
	t.Parallel()
	bin := shQuote(os.Args[0])
	var unsetAll strings.Builder
	unsetAll.WriteString("env")
	for _, marker := range canon.AgentEnvMarkers() {
		unsetAll.WriteString(" -u " + marker)
	}
	for _, tt := range []struct{ name, script string }{
		{"env -u every marker", unsetAll.String() + " " + bin + " defaults reset --yes; exit $?"},
		{"unset", "unset CLAUDECODE; " + bin + " session allow X --session s; exit $?"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			env := guardrailEnv(t)
			dir := newGuardrailProject(t, env)
			env, defaultsFile, content := withDefaultsFile(t, env)
			env = append(agentEnv(env), "TERM=dumb")
			shPID, out, code := runShOnPty(t, env, dir, tt.script, nil)
			ancestor := fmt.Sprintf("ancestor process %d", shPID)
			if code == 0 || !strings.Contains(out, "requires a human") || !strings.Contains(out, ancestor) {
				t.Fatalf("sh -c %q on a pty: exit %d, want non-zero with \"requires a human\" naming %q\n%s",
					tt.script, code, ancestor, out)
			}
			if got, err := os.ReadFile(defaultsFile); err != nil || string(got) != content {
				t.Errorf("defaults file after the refused run: %q, %v; want it unchanged", got, err)
			}
		})
	}
}

// TestHumanGate_PtyChallenge drives the gate's confirmation challenge on a
// real pty, with no agent marker within the gate's ancestor walk: qsdev runs
// below a chain of more clean shells than the walk reads, so the test holds
// when `go test` itself runs inside an agent session. End of input (^D) in
// place of the code is refused and leaves the defaults file; the code typed
// back passes and removes it, which is the documented residual: something
// that drives the pty interactively passes the challenge.
func TestHumanGate_PtyChallenge(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name    string
		answer  func(code string) string
		refused bool
	}{
		{"end of input refuses", func(string) string { return "\x04" }, true},
		{"code typed back passes", func(code string) string { return code + "\n" }, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			env := guardrailEnv(t)
			dir := newGuardrailProject(t, env)
			env, defaultsFile, content := withDefaultsFile(t, env)
			nest := filepath.Join(t.TempDir(), "nest.sh")
			// Each level waits for the next ($? exit), so every shell stays
			// an ancestor; level 0 becomes qsdev.
			const nestScript = `if [ "$1" -gt 0 ]; then sh "$0" $(($1 - 1)); exit $?; fi
exec "$QSDEV_TEST_BIN" defaults reset --yes
`
			if err := os.WriteFile(nest, []byte(nestScript), 0o644); err != nil {
				t.Fatal(err)
			}
			env = append(humanlessEnv(env), "TERM=dumb", "QSDEV_TEST_BIN="+os.Args[0])
			script := "sh " + shQuote(nest) + " " + strconv.Itoa(maxGateAncestors) + "; exit $?"
			_, out, code := runShOnPty(t, env, dir, script, tt.answer)
			_, statErr := os.Stat(defaultsFile)
			if tt.refused {
				if code == 0 || !strings.Contains(out, "confirmation code was not typed") {
					t.Fatalf("^D at the challenge: exit %d, want a refusal for the missing code\n%s", code, out)
				}
				if got, err := os.ReadFile(defaultsFile); err != nil || string(got) != content {
					t.Errorf("defaults file after the refused run: %q, %v; want it unchanged", got, err)
				}
				return
			}
			if code != 0 || !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("code typed back: exit %d, defaults file %v; want the reset to run\n%s", code, statErr, out)
			}
		})
	}
}

// maxGateAncestors is the number of ancestors the human gate reads
// (cmdutil's maxAncestorDepth).
const maxGateAncestors = 32
