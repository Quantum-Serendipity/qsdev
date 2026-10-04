package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// probeOverlay writes, at p, an org overlay that adds a tool named probe, so
// whether a run read it shows in `defaults show`.
func probeOverlay(t *testing.T, p, probe string) string {
	t.Helper()
	body := "tools:\n    " + probe + ":\n" +
		"        display_name: Probe\n        category: devex\n        description: x\n        default_policy: opt-in\n"
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// envValue returns the value env gives name, or "".
func envValue(env []string, name string) string {
	for _, kv := range env {
		if v, ok := strings.CutPrefix(kv, name+"="); ok {
			return v
		}
	}
	return ""
}

// TestOrgOverlayPin_AgentCannotChooseOverlay is the U18-WS1 round 4 E3 test
// of the structural org overlay control, in an initialised project with an
// isolated HOME and TMPDIR. Without a pin, an overlay QSDEV_ORG_CONFIG names
// outside the project and TMPDIR ($HOME/evil.yaml) is ignored, also by a run
// that looks like a human's (the spoof: no CLAUDECODE, a terminal), and init
// no longer records one; only 'defaults pin', which the agent may not run,
// approves an overlay; and the pin lies outside the checkout, so `git clean
// -fdX` does not remove it.
func TestOrgOverlayPin_AgentCannotChooseOverlay(t *testing.T) {
	t.Parallel()
	env := guardrailEnv(t)
	dir, _ := initialisedProject(t, env, false)
	home := envValue(env, "HOME")
	orgConfig := branding.Get().EnvPrefix + "ORG_CONFIG="
	evil := probeOverlay(t, filepath.Join(home, "evil.yaml"), "pin-probe-evil")
	good := probeOverlay(t, filepath.Join(t.TempDir(), "org", "defaults.yaml"), "pin-probe-good")

	reads := func(env []string, probe string) bool {
		t.Helper()
		out, code := runQsdev(t, env, dir, nil, "defaults", "show", "--section", "tools")
		if code != 0 {
			t.Fatalf("defaults show: exit %d\n%s", code, out)
		}
		return strings.Contains(out, probe)
	}
	withEvil := append(slices.Clone(env), orgConfig+evil)

	// (1) No pin: the overlay is ignored, for the agent and the spoof alike.
	if reads(append(agentEnv(env), orgConfig+evil), "pin-probe-evil") {
		t.Error("an agent run without a pin read the overlay QSDEV_ORG_CONFIG names below HOME")
	}
	if reads(withEvil, "pin-probe-evil") {
		t.Error("a run that looks like a human's read the overlay without a pin")
	}
	out, code := runQsdev(t, agentEnv(env), dir, nil, "check")
	if strings.Contains(out, "recorded for the project") {
		t.Errorf("check claims a recorded overlay where none is pinned (exit %d)\n%s", code, out)
	}

	// (3) init, even one that looks like a human's, approves nothing.
	mustQsdev(t, withEvil, dir, "init", "--update", "--yes")
	if reads(append(agentEnv(env), orgConfig+evil), "pin-probe-evil") {
		t.Error("init --update approved the overlay QSDEV_ORG_CONFIG names")
	}
	if out, code := runQsdev(t, append(agentEnv(env), orgConfig+evil), dir, nil, "defaults", "pin"); code == 0 {
		t.Errorf("an agent ran defaults pin\n%s", out)
	}

	// A human's pin is what approves an overlay.
	mustQsdev(t, append(slices.Clone(env), orgConfig+good), dir, "defaults", "pin")
	if !reads(append(agentEnv(env), orgConfig+good), "pin-probe-good") {
		t.Error("an agent run did not read the overlay a human pinned")
	}

	// (2) git clean -fdX removes the state directory, not the pin.
	clean := exec.Command("git", "clean", "-fdX")
	clean.Dir, clean.Env = dir, env
	if out, err := clean.CombinedOutput(); err != nil {
		t.Fatalf("git clean: %v\n%s", err, out)
	}
	agentEvil := append(agentEnv(env), orgConfig+evil)
	if reads(agentEvil, "pin-probe-evil") || !reads(agentEvil, "pin-probe-good") {
		t.Error("after git clean -fdX an agent run did not read the pinned overlay instead of its own")
	}
}
