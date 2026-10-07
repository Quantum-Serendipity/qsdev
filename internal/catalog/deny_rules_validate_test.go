package catalog

import (
	"maps"
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/denyutil"
)

// TestEveryDenyRuleValidates checks every rule in every catalog permission
// set (deny, ask and allow) against denyutil.Validate, so a malformed rule or
// a Bash rule ending in the legacy ":*" marker, which Claude Code reads as a
// trailing " *" and which therefore blocks nothing it was written for, cannot
// ship.
func TestEveryDenyRuleValidates(t *testing.T) {
	t.Parallel()
	c, err := Default()
	if err != nil {
		t.Fatalf("loading catalog: %v", err)
	}
	for kind, sets := range map[string]map[string][]string{
		"deny":  c.permissionRules.DenyRules,
		"ask":   c.permissionRules.AskRules,
		"allow": c.permissionRules.AllowRules,
	} {
		if len(sets) == 0 {
			t.Errorf("catalog has no %s rule sets", kind)
		}
		for _, name := range slices.Sorted(maps.Keys(sets)) {
			for _, rule := range sets[name] {
				if err := denyutil.Validate(rule); err != nil {
					t.Errorf("%s set %q: %v", kind, name, err)
				}
			}
		}
	}
}

// TestColonRules_MatchWhatTheyGuard pins each catalog rule that guards text
// after a colon (deno npm:/jsr: specifiers, docker/podman host-root binds),
// which once ended in the legacy ":*" and matched nothing. Every rule must be
// in its set, block its command and leave the benign one alone.
func TestColonRules_MatchWhatTheyGuard(t *testing.T) {
	t.Parallel()
	c, err := Default()
	if err != nil {
		t.Fatalf("loading catalog: %v", err)
	}
	tests := []struct {
		set, rule, blocks, allows string
	}{
		{"remote_package_exec", "Bash(deno run *npm:**)", "deno run -A npm:evil-cli", "deno run main.ts"},
		{"remote_package_exec", "Bash(deno run *jsr:**)", "deno run jsr:@evil/cli", "deno run jsr.ts"},
		{"remote_package_exec", "Bash(deno serve *npm:**)", "deno serve npm:evil-server", "deno serve server.ts"},
		{"remote_package_exec", "Bash(deno serve *jsr:**)", "deno serve jsr:@evil/server", "deno serve server.ts"},
		{"remote_package_exec", "Bash(deno npm:**)", "deno npm:evil-cli", "deno npm"},
		{"remote_package_exec", "Bash(deno jsr:**)", "deno jsr:@evil/cli", "deno jsr"},
		{"remote_package_exec", "Bash(deno watch *npm:**)", "deno watch npm:evil-cli", "deno watch main.ts"},
		{"remote_package_exec", "Bash(deno watch *jsr:**)", "deno watch jsr:@evil/cli", "deno watch main.ts"},
		{"remote_package_exec", "Bash(deno -* npm:**)", "deno -A npm:evil-cli", "deno -A main.ts"},
		{"remote_package_exec", "Bash(deno -* jsr:**)", "deno -q jsr:@evil/cli", "deno -q main.ts"},
		{"container_escape", "Bash(docker * -v /:/*)", "docker run -v /:/host alpine", "docker run -v ./data:/data alpine"},
		{"container_escape", "Bash(docker * --volume /:/*)", "docker run --volume /:/h:ro alpine", "docker run --volume /srv:/srv alpine"},
		{"container_escape", "Bash(docker * --volume=/:/*)", "docker run --volume=/:/host alpine", "docker run --volume=/srv:/srv alpine"},
		{"container_escape", "Bash(docker * -v / *)", "docker run -v / alpine", "docker run -v /data alpine"},
		{"container_escape", "Bash(docker * --volume / *)", "docker run --volume / alpine", "docker run --volume /data alpine"},
		{"container_escape", "Bash(podman * -v /:/*)", "podman run -v /:/host alpine", "podman run -v ./data:/data alpine"},
		{"container_escape", "Bash(podman * --volume /:/*)", "podman run --volume /:/h:ro alpine", "podman run --volume /srv:/srv alpine"},
		{"container_escape", "Bash(podman * --volume=/:/*)", "podman run --volume=/:/host alpine", "podman run --volume=/srv:/srv alpine"},
		{"container_escape", "Bash(podman * -v / *)", "podman run -v / alpine", "podman run -v /data alpine"},
		{"container_escape", "Bash(podman * --volume / *)", "podman run --volume / alpine", "podman run --volume /data alpine"},
	}
	for _, tt := range tests {
		t.Run(tt.rule, func(t *testing.T) {
			t.Parallel()
			if !slices.Contains(c.PermissionDenyRules(tt.set), tt.rule) {
				t.Errorf("deny set %q does not contain %q", tt.set, tt.rule)
			}
			if !denyutil.MatchesBashRule(tt.rule, tt.blocks) {
				t.Errorf("%q does not block %q", tt.rule, tt.blocks)
			}
			if denyutil.MatchesBashRule(tt.rule, tt.allows) {
				t.Errorf("%q unexpectedly blocks %q", tt.rule, tt.allows)
			}
		})
	}
}
