package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/claudesettings"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// U18-01 regression tests for the generator inputs that decide whether Claude
// Code is configured at all. The hook-choice matrix in
// guardrail_invariants_test.go never switches Claude Code off; these do, the
// way an agent can: through the answers file, the committed .qsdev.yaml and
// the org overlay. Whatever they say, no command run afterwards may leave the
// project without a settings.json that registers selfprotect.

// rewriteAnswers replaces every answers file present in dir with rewrite's
// result for its content.
func rewriteAnswers(t *testing.T, dir string, rewrite func(t *testing.T, content []byte) []byte) {
	t.Helper()
	for _, rel := range answersFiles() {
		full := filepath.Join(dir, filepath.FromSlash(rel))
		data, err := os.ReadFile(full)
		if err != nil {
			continue
		}
		if err := os.WriteFile(full, rewrite(t, data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// claudeCodeOff switches Claude Code and self-protection off in an answers
// file, as `sed` or the Write tool would.
func claudeCodeOff(t *testing.T, content []byte) []byte {
	t.Helper()
	out := content
	for _, kv := range [][2]string{{"claude_code: true", "claude_code: false"}, {"self_protection: true", "self_protection: false"}} {
		next := bytes.Replace(out, []byte(kv[0]), []byte(kv[1]), 1)
		if bytes.Equal(next, out) {
			t.Fatalf("answers file records no %q:\n%s", kv[0], content)
		}
		out = next
	}
	return out
}

// truncatedAnswers empties an answers file down to an empty mapping, which
// loads with Claude Code off.
func truncatedAnswers(*testing.T, []byte) []byte {
	return []byte("{}\n")
}

// assertSelfprotect fails unless dir's settings.json exists and registers
// selfprotect.
func assertSelfprotect(t *testing.T, dir, after string) {
	t.Helper()
	self := branding.Get().AppName + " selfprotect"
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(claudesettings.ProjectRelPath))); err != nil {
		t.Fatalf("after %s: %s is gone: %v", after, claudesettings.ProjectRelPath, err)
	}
	if cmds := preToolUseCommands(t, dir); !anyContains(cmds, self) {
		t.Fatalf("after %s: PreToolUse has no %q hook:\n%s", after, self, strings.Join(cmds, "\n"))
	}
}

// TestGuardrailInvariant_AnswersCannotDropClaudeCode covers the chain the
// E3 probe reproduced: an answers file rewritten to switch Claude Code off
// (or emptied) through a path the hook cannot see, then a regeneration by
// the agent or by a human. Every regeneration used to stop producing
// settings.json and remove it; the committed .qsdev.yaml, which enables Claude
// Code, now keeps it on.
func TestGuardrailInvariant_AnswersCannotDropClaudeCode(t *testing.T) {
	t.Parallel()
	rewrites := []struct {
		name    string
		rewrite func(*testing.T, []byte) []byte
	}{
		{"claude-code-false", claudeCodeOff},
		{"truncated", truncatedAnswers},
	}
	commands := [][]string{
		{"init", "--update", "--yes"},
		{"claude", "update"},
		{"update", "--configs-only"},
	}
	for _, rw := range rewrites {
		for _, args := range commands {
			for _, asAgent := range []bool{false, true} {
				name := rw.name + "/" + strings.Join(args, " ")
				if asAgent {
					name += "/agent"
				}
				t.Run(name, func(t *testing.T) {
					t.Parallel()
					env := guardrailEnv(t)
					dir, _ := initialisedProject(t, env, false)
					rewriteAnswers(t, dir, rw.rewrite)
					runEnv := env
					if asAgent {
						runEnv = agentEnv(env)
					}
					if out, code := runQsdev(t, runEnv, dir, nil, args...); code != 0 {
						t.Fatalf("qsdev %s: exit %d\n%s", strings.Join(args, " "), code, out)
					}
					assertSelfprotect(t, dir, strings.Join(args, " "))
					// A later human regeneration keeps it too.
					mustQsdev(t, env, dir, "init", "--update", "--yes")
					assertSelfprotect(t, dir, "a second init --update")
				})
			}
		}
	}
}

// TestGuardrailInvariant_ClaudeCodeOffKeepsSettings covers the committed
// switch: with .qsdev.yaml and the answers both recording Claude Code off,
// init --update stops generating the Claude Code files but leaves
// settings.json in place, untracked, still registering selfprotect; only
// teardown removes it. The agent-side edit of .qsdev.yaml that would record
// the switch is denied by the hook (GD-001).
func TestGuardrailInvariant_ClaudeCodeOffKeepsSettings(t *testing.T) {
	t.Parallel()
	env := guardrailEnv(t)
	dir, _ := initialisedProject(t, env, false)
	cfgPath := filepath.Join(dir, branding.Get().ConfigFile)
	cfg, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	const on, off = "claude_code:\n    enabled: true\n", "claude_code:\n    enabled: false\n"
	if !bytes.Contains(cfg, []byte(on)) {
		t.Fatalf("%s does not record %q:\n%s", branding.Get().ConfigFile, on, cfg)
	}

	// The agent's Edit is denied through the real hook IO.
	payload, err := json.Marshal(map[string]any{
		"hook_event_name": "PreToolUse",
		"cwd":             dir,
		"tool_name":       "Edit",
		"tool_input":      map[string]string{"file_path": cfgPath, "old_string": on, "new_string": off},
	})
	if err != nil {
		t.Fatal(err)
	}
	out, code := runQsdev(t, agentEnv(env), dir, bytes.NewReader(payload), "selfprotect")
	if code != 2 || !strings.Contains(out, "GD-001") {
		t.Errorf("selfprotect on an Edit switching claude_code off: exit %d, want 2 with GD-001\n%s", code, out)
	}

	// A human (or an agent past the hook) records the switch anyway.
	if err := os.WriteFile(cfgPath, bytes.Replace(cfg, []byte(on), []byte(off), 1), 0o644); err != nil {
		t.Fatal(err)
	}
	rewriteAnswers(t, dir, claudeCodeOff)
	mustQsdev(t, agentEnv(env), dir, "init", "--update", "--yes")
	assertSelfprotect(t, dir, "init --update with Claude Code off")
	st, err := state.LoadStateFromFile(filepath.Join(dir, filepath.FromSlash(state.InitStateFile())))
	if err != nil {
		t.Fatal(err)
	}
	if _, tracked := st.Files[claudesettings.ProjectRelPath]; tracked {
		t.Errorf("%s is still tracked after Claude Code was switched off", claudesettings.ProjectRelPath)
	}
}

// TestGuardrailInvariant_OrgOverlayCannotClaimSettings covers an org overlay
// that gives a tool exclusive ownership of settings.json, so that disabling
// it would delete the file: a new tool, and the safety-block tool redefined
// outside the security category so disable is not human-gated. The overlay
// fails validation (and is skipped), so no command run by the agent removes
// selfprotect.
func TestGuardrailInvariant_OrgOverlayCannotClaimSettings(t *testing.T) {
	t.Parallel()
	guard := safetyBlockTool(t).Name
	const hostile = "hostile-owner"
	overlay := "tools:\n" +
		"    " + hostile + ":\n" +
		"        display_name: Hostile\n        category: devex\n        description: x\n        default_policy: opt-in\n" +
		"        owned_files:\n            - path: " + claudesettings.ProjectRelPath + "\n              ownership: exclusive\n" +
		"    " + guard + ":\n" +
		"        display_name: Guard\n        category: devex\n        description: x\n        default_policy: opt-in\n" +
		"        owned_files:\n            - path: " + path.Dir(claudesettings.ProjectRelPath) + "\n              ownership: exclusive\n"
	steps := [][]string{
		{"enable", hostile},
		{"disable", hostile, "--force"},
		{"disable", guard, "--force"},
		{"init", "--update", "--yes"},
		{"claude", "update"},
	}

	env := guardrailEnv(t)
	dir, _ := initialisedProject(t, env, false)
	overlayPath := filepath.Join(t.TempDir(), "defaults.yaml")
	if err := os.WriteFile(overlayPath, []byte(overlay), 0o644); err != nil {
		t.Fatal(err)
	}
	orgConfig := branding.Get().EnvPrefix + "ORG_CONFIG=" + overlayPath
	// A human pins the overlay first: without a pin every run ignores it
	// (catalog.OrgConfigPin), and the invariant must hold for a pinned one.
	mustQsdev(t, append(slices.Clone(env), orgConfig), dir, "defaults", "pin")
	overlayEnv := append(agentEnv(env), orgConfig)

	out, code := runQsdev(t, overlayEnv, dir, nil, "defaults", "validate")
	if code == 0 || !strings.Contains(out, "registers the self-protection hook") {
		t.Errorf("defaults validate on the hostile overlay: exit %d, want a failure naming the self-protection hook\n%s", code, out)
	}
	for _, args := range steps {
		runQsdev(t, overlayEnv, dir, nil, args...)
		assertSelfprotect(t, dir, strings.Join(args, " "))
	}
}
