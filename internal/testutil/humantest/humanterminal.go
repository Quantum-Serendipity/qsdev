// Package humantest holds the test stand-ins for a human at a terminal that
// the CLI's human gate (cmdutil.RequireHuman) accepts. It is separate from
// testutil because it reads canon, whose own tests import testutil.
package humantest

import (
	"io"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/canon"
)

// HumanTerminal is stdin a test presents to the CLI's human gate
// (cmdutil.RequireHuman) as an interactive terminal, or not. It implements
// cmdutil.Terminal, so with TTY set it answers for a human: the gate takes it
// at its word and neither walks the test's process ancestry nor opens a
// controlling terminal for the confirmation challenge. That keeps the
// in-process tests deterministic when `go test` itself runs inside an agent
// session. The agent-environment check still applies (see SetAgentSession).
// No shipped code may implement cmdutil.Terminal
// (cmdutil.TestNoProductionTerminalDouble).
type HumanTerminal struct {
	io.Reader
	TTY bool
}

// IsTerminal reports whether the test presents the input as a terminal.
func (h HumanTerminal) IsTerminal() bool { return h.TTY }

// SetAgentSession sets every agent-session marker (canon.AgentEnvMarkers) in
// the test's environment to value: "" runs as a human outside any agent
// session, even when `go test` runs inside one, and a non-empty value runs
// inside one. It uses t.Setenv, so the test must not be parallel.
func SetAgentSession(t testing.TB, value string) {
	t.Helper()
	for _, name := range canon.AgentEnvMarkers() {
		t.Setenv(name, value)
	}
}
