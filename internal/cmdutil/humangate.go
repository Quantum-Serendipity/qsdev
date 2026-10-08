package cmdutil

import (
	"bufio"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"strings"

	"github.com/charmbracelet/x/term"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/canon"
)

// maxAncestorDepth bounds the walk up the process tree: an agent's shell sits
// a few levels above the CLI, and a deeper tree is not worth reading.
const maxAncestorDepth = 32

// errAncestryUnknown reports a platform whose process ancestry the gate
// cannot read (anything but Linux and macOS, Windows included). The gate then
// relies on the terminal challenge alone.
var errAncestryUnknown = errors.New("process ancestry is not readable on this platform")

// challengeLineLimit bounds the reply the gate reads from the terminal.
const challengeLineLimit = 64

// humanGate decides whether a human runs a sensitive command. Its fields are
// the operating-system reads it makes, so a test can build its own gate
// without touching the process's real environment, ancestry or terminal;
// RequireHuman uses productionGate.
type humanGate struct {
	// getenv reads the process's own environment.
	getenv func(name string) string
	// isTerminal reports whether stdin is an interactive terminal.
	isTerminal func(in io.Reader) bool
	// self returns the process's pid.
	self func() int
	// parentOf returns pid's parent pid, or errAncestryUnknown.
	parentOf func(pid int) (int, error)
	// environOf returns the environment pid started with, as NAME=value.
	environOf func(pid int) ([]string, error)
	// programOf returns the executable pid runs ("" when unknown) and its
	// argv.
	programOf func(pid int) (exe string, argv []string, err error)
	// openTTY opens the process's controlling terminal for reading and
	// writing.
	openTTY func() (io.ReadWriteCloser, error)
	// newCode returns a fresh confirmation code.
	newCode func() (string, error)
}

// productionGate is the gate the shipped binary runs.
var productionGate = humanGate{
	getenv:     os.Getenv,
	isTerminal: stdinIsTerminal,
	self:       os.Getpid,
	parentOf:   parentPID,
	environOf:  processEnviron,
	programOf:  processProgram,
	openTTY:    openControllingTTY,
	newCode:    challengeCode,
}

// verify returns nil when a human runs the command label names ('qsdev
// defaults pin'), and otherwise the refusal. The checks, in order:
//
//  1. no agent-session marker (canon.AgentEnvMarkers) in the process's own
//     environment;
//  2. a Terminal test double answers for a human and skips checks 4 and 5,
//     so in-process tests stay deterministic inside an agent session; no
//     shipped type implements Terminal (TestNoProductionTerminalDouble);
//  3. stdin is an interactive terminal;
//  4. no ancestor process is an agent: none started with a marker in its
//     environment, which `env -u`/`unset` cannot rewrite, and none is the
//     Claude Code CLI itself (canon.IsClaudeProcess), which carries no
//     marker but sets them for the shells it starts, so a shell that
//     replaces itself (`exec env -u ... qsdev`) leaves it in the walk.
//     Stripping the markers and faking a terminal with a pseudo-terminal
//     therefore does not pass;
//  5. a code written to the controlling terminal is typed back on it, which
//     input fed blindly (`yes |`, </dev/null under script) cannot do. On
//     platforms whose ancestry is unknown, and for a process reparented away
//     from its agent, this is the only check left.
func (g humanGate) verify(in io.Reader, label string) error {
	if marker := firstMarker(g.getenv); marker != "" {
		return fmt.Errorf("%s requires a human: it is refused inside an AI agent session (%s is set); run it from your own terminal",
			label, marker)
	}
	if t, ok := in.(Terminal); ok {
		if t.IsTerminal() {
			return nil
		}
		return notAtTerminal(label)
	}
	if !g.isTerminal(in) {
		return notAtTerminal(label)
	}
	if pid, why := g.agentAncestor(); why != "" {
		return fmt.Errorf("%s requires a human: it is refused inside an AI agent session (%s ancestor process %d); run it from your own terminal",
			label, why, pid)
	}
	return g.challenge(label)
}

// notAtTerminal is the refusal for a run without an interactive terminal.
func notAtTerminal(label string) error {
	return fmt.Errorf("%s requires a human at an interactive terminal: it weakens or removes a guardrail", label)
}

// firstMarker returns the first agent-session marker getenv reports set, or
// "".
func firstMarker(getenv func(string) string) string {
	for _, name := range canon.AgentEnvMarkers() {
		if getenv(name) != "" {
			return name
		}
	}
	return ""
}

// agentAncestor walks up from the process's parent, at most
// maxAncestorDepth processes and never to pid 1, and returns the first
// ancestor that is an agent, with why, worded to precede "ancestor process
// <pid>": its starting environment holds an agent-session marker, or it is
// the Claude Code CLI. An ancestor whose environment or program cannot be
// read (another user's, such as sshd or sudo) is passed over for that
// check; an unreadable parent pid ends the walk, leaving the challenge to
// decide.
func (g humanGate) agentAncestor() (int, string) {
	pid := g.self()
	for range maxAncestorDepth {
		parent, err := g.parentOf(pid)
		if err != nil || parent <= 1 {
			return 0, ""
		}
		pid = parent
		if env, err := g.environOf(pid); err == nil {
			if marker := firstMarker(environLookup(env)); marker != "" {
				return pid, marker + " is set in the environment of"
			}
		}
		if exe, argv, err := g.programOf(pid); err == nil && canon.IsClaudeProcess(exe, argv) {
			return pid, "the Claude Code CLI runs as"
		}
	}
	return 0, ""
}

// environLookup returns a getenv over env's NAME=value entries.
func environLookup(env []string) func(string) string {
	return func(name string) string {
		for _, kv := range env {
			if k, v, ok := strings.Cut(kv, "="); ok && k == name {
				return v
			}
		}
		return ""
	}
}

// challenge writes a fresh code to the controlling terminal and refuses
// unless the same code is typed back on it.
func (g humanGate) challenge(label string) error {
	tty, err := g.openTTY()
	if err != nil {
		return fmt.Errorf("%s requires a human at an interactive terminal: the controlling terminal cannot be opened for confirmation: %w",
			label, err)
	}
	defer func() { _ = tty.Close() }()
	code, err := g.newCode()
	if err != nil {
		return fmt.Errorf("%s: generating a confirmation code: %w", label, err)
	}
	if _, err := fmt.Fprintf(tty, "%s weakens or removes a guardrail. To confirm you are running it yourself, type %s and press Enter: ",
		label, code); err != nil {
		return fmt.Errorf("%s requires a human at an interactive terminal: writing to the controlling terminal: %w", label, err)
	}
	line, err := bufio.NewReader(io.LimitReader(tty, challengeLineLimit)).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("%s requires a human at an interactive terminal: reading the confirmation code: %w", label, err)
	}
	if !strings.HasSuffix(line, "\n") || strings.TrimSpace(line) != code {
		return fmt.Errorf("%s requires a human: the confirmation code was not typed at the terminal", label)
	}
	return nil
}

// stdinIsTerminal reports whether in is a file open on an interactive
// terminal.
func stdinIsTerminal(in io.Reader) bool {
	f, ok := in.(*os.File)
	return ok && term.IsTerminal(f.Fd())
}

// challengeCode returns a random six-digit code from crypto/rand.
func challengeCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", fmt.Errorf("reading random bytes: %w", err)
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
