package main

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/instance"
	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/canon"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/cmdscan"
	"github.com/Quantum-Serendipity/qsdev/internal/testutil/humantest"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// guardrailWeakeningCommands are the sensitive invocations an agent tries in
// the U18-WS1 / XS-WS2 journeys: each weakens or removes a guardrail.
// `update --configs-only` keeps the binary stage (and its network use) out
// even if the gate regressed.
func guardrailWeakeningCommands(t *testing.T) [][]string {
	t.Helper()
	return [][]string{
		{"teardown", "--force"},
		{"disable", safetyBlockTool(t).Name, "--force"},
		{"claude", "init", "--yes", "--force"},
		{"claude", "update", "--force"},
		{"repair", "--force"},
		{"defaults", "reset", "--yes"},
		{"defaults", "pin"},
		{"defaults", "pin", "--global"},
		{"update", "--configs-only", "--no-strict"},
	}
}

// TestSensitiveCommandsAnnotated pins the spec set the self-protection hook
// derives from the real command tree: every guardrail-weakening command is
// marked sensitive, so SP-014 and the binary's human gate both cover it.
// `init` is deliberately not: a regeneration always writes the guardrails
// back (the generator invariant), so an agent running it weakens nothing.
// It builds the real tree, so it is serial (TestRootBuildersAreSerial).
func TestSensitiveCommandsAnnotated(t *testing.T) {
	specs := cmdutil.SensitiveCommands(instance.NewRootCommand())
	has := func(path ...string) bool {
		return slices.ContainsFunc(specs, func(s cmdscan.CommandSpec) bool {
			if len(s.Path) != len(path) {
				return false
			}
			for i, names := range s.Path {
				if !slices.Contains(names, path[i]) {
					return false
				}
			}
			return true
		})
	}
	for _, path := range [][]string{
		{"teardown"}, {"disable"}, {"repair"}, {"claude", "init"}, {"claude", "update"},
		{"session", "allow"}, {"sandbox", "approve"}, {"defaults", "reset"}, {"defaults", "pin"},
		{"self-update"}, {"update"},
	} {
		if !has(path...) {
			t.Errorf("SensitiveCommands(root) has no spec for %q", path)
		}
	}
	if has("init") {
		t.Error("SensitiveCommands(root) marks init sensitive; regeneration restores guardrails")
	}
}

// TestHumanGate_AgentCannotWeakenGuardrails is the U18-WS1 step 3 E3 test: in
// an initialised project, each guardrail-weakening command run from an agent
// session without a terminal exits non-zero saying it requires a human, and
// both guard hooks stay registered.
func TestHumanGate_AgentCannotWeakenGuardrails(t *testing.T) {
	t.Parallel()
	for _, args := range guardrailWeakeningCommands(t) {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			t.Parallel()
			env := guardrailEnv(t)
			dir, _ := initialisedProject(t, env, false)
			out, code := runQsdev(t, agentEnv(env), dir, nil, args...)
			if code == 0 || !strings.Contains(out, "requires a human") {
				t.Fatalf("agent ran qsdev %s: exit %d, want non-zero with \"requires a human\"\n%s",
					strings.Join(args, " "), code, out)
			}
			assertGuardrails(t, dir, false)
		})
	}
}

// TestGuardrailEnvStripsAgentMarkers pins that the qsdev processes these
// tests start see no agent-session marker from a `go test` run inside an
// agent session, so the human-path helper is not refused by its own
// environment. Not parallel: it sets the markers with t.Setenv.
func TestGuardrailEnvStripsAgentMarkers(t *testing.T) {
	humantest.SetAgentSession(t, "1")
	env := guardrailEnv(t)
	for _, marker := range canon.AgentEnvMarkers() {
		if slices.ContainsFunc(env, func(kv string) bool { return strings.HasPrefix(kv, marker+"=") }) {
			t.Errorf("guardrailEnv keeps the agent-session marker %s", marker)
		}
	}
}

// TestHumanGate_SelfprotectBlocksSensitiveCommands pins SP-014 to the real
// command tree: the self-protection hook denies an agent's Bash call to each
// guardrail-weakening command, and still allows a read-only one.
func TestHumanGate_SelfprotectBlocksSensitiveCommands(t *testing.T) {
	t.Parallel()
	env := guardrailEnv(t)
	dir := newGuardrailProject(t, env)
	app := branding.Get().AppName
	verdict := func(t *testing.T, command string) (string, int) {
		t.Helper()
		payload, err := json.Marshal(map[string]any{
			"hook_event_name": "PreToolUse",
			"cwd":             dir,
			"tool_name":       "Bash",
			"tool_input":      map[string]string{"command": command},
		})
		if err != nil {
			t.Fatal(err)
		}
		return runQsdev(t, agentEnv(env), dir, bytes.NewReader(payload), "selfprotect")
	}
	// Each verdict is a separate hook process reading the same project.
	for _, args := range append(guardrailWeakeningCommands(t),
		[]string{"self-update", "--no-strict"}, []string{"session", "allow", "X", "--session", "s"},
		[]string{"sandbox", "approve"}) {
		command := app + " " + strings.Join(args, " ")
		t.Run("deny "+command, func(t *testing.T) {
			t.Parallel()
			if out, code := verdict(t, command); code != 2 || !strings.Contains(out, "SP-014") {
				t.Errorf("selfprotect on %q: exit %d, want 2 with SP-014\n%s", command, code, out)
			}
		})
	}
	for _, command := range []string{app + " status", app + " teardown --dry-run", app + " disable context7"} {
		t.Run("allow "+command, func(t *testing.T) {
			t.Parallel()
			if out, code := verdict(t, command); code != 0 {
				t.Errorf("selfprotect on %q: exit %d, want 0\n%s", command, code, out)
			}
		})
	}
}
