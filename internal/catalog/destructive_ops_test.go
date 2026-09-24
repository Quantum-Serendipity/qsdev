package catalog

import (
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/denyutil"
)

// destructiveCommands are common spellings of history-rewriting and
// recursive-delete commands that every preset carrying the destructive_ops
// deny set must block.
var destructiveCommands = []string{
	"git reset --hard",
	"git reset --hard HEAD~1",
	"git push --force",
	"git push --force origin main",
	"git push origin main --force",
	"git push origin --force main",
	"git push -f",
	"git push -f origin main",
	"git push origin main -f",
	"git push origin -f main",
	"git push --force-with-lease",
	"git push --force-with-lease origin main",
	"git push origin main --force-with-lease=main:abc123",
	"git push --force-if-includes origin main",
	"git push origin +main",
	"git push origin +HEAD:main",
	"rm -rf build",
	"rm -fr build",
	"rm -Rf build",
	"rm -fR build",
	"rm -r -f build",
	"rm -f -r build",
	"rm -R -f build",
	"rm -f -R build",
	"rm --recursive --force build",
	"rm --force --recursive build",
}

// Regression: destructive_ops only listed one spelling per command, so
// `git push -f`, `--force-with-lease`, `+refspec` pushes, `rm -fr` and the
// split-flag rm forms ran unprompted under presets that allow Bash(git *).
func TestDestructiveOpsDenyVariants(t *testing.T) {
	t.Parallel()

	cat, err := LoadEmbeddedOnly()
	if err != nil {
		t.Fatalf("LoadEmbeddedOnly() error: %v", err)
	}

	var presetsChecked int
	for _, name := range cat.PermissionPresets() {
		def, ok := cat.PermissionPreset(name)
		if !ok || !slices.Contains(def.DenySets, "destructive_ops") {
			continue
		}
		presetsChecked++
		var deny []string
		for _, set := range def.DenySets {
			deny = append(deny, cat.PermissionDenyRules(set)...)
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			for _, cmd := range destructiveCommands {
				op := "Bash(" + cmd + ")"
				if !slices.ContainsFunc(deny, func(rule string) bool {
					return denyutil.MatchesDenyRule(rule, op)
				}) {
					t.Errorf("preset %q does not deny %q", name, cmd)
				}
			}
		})
	}
	if presetsChecked == 0 {
		t.Fatal("no permission preset includes the destructive_ops deny set")
	}
}

// Claude Code's "cmd *" rule needs trailing arguments, so the bare forms of
// destructive commands must be denied by an exact rule as well.
func TestDestructiveOpsBareForms(t *testing.T) {
	t.Parallel()

	cat, err := LoadEmbeddedOnly()
	if err != nil {
		t.Fatalf("LoadEmbeddedOnly() error: %v", err)
	}
	rules := cat.PermissionDenyRules("destructive_ops")
	for _, bare := range []string{"git reset --hard", "git push --force", "git push -f"} {
		t.Run(bare, func(t *testing.T) {
			t.Parallel()
			if !slices.Contains(rules, "Bash("+bare+")") {
				t.Errorf("destructive_ops is missing the bare rule Bash(%s)", bare)
			}
		})
	}
}

func TestDestructiveOpsDoNotDenySafeCommands(t *testing.T) {
	t.Parallel()

	cat, err := LoadEmbeddedOnly()
	if err != nil {
		t.Fatalf("LoadEmbeddedOnly() error: %v", err)
	}
	rules := cat.PermissionDenyRules("destructive_ops")
	for _, cmd := range []string{
		"git push",
		"git push origin main",
		"git push -u origin feature",
		"git push --follow-tags",
		"git reset --soft HEAD~1",
		"rm build/out.txt",
		"rm -r build",
	} {
		t.Run(cmd, func(t *testing.T) {
			t.Parallel()
			op := "Bash(" + cmd + ")"
			for _, rule := range rules {
				if denyutil.MatchesDenyRule(rule, op) {
					t.Errorf("rule %q unexpectedly denies safe command %q", rule, cmd)
				}
			}
		})
	}
}
