package profile

import (
	"strings"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestProjectInputsFromAnswers_MinReleaseAge pins that the project's release
// age window comes from its compliance level through the catalog, falling
// back to the lowest level (never below it) for an empty or unknown name.
func TestProjectInputsFromAnswers_MinReleaseAge(t *testing.T) {
	t.Parallel()
	tests := []struct {
		level string
		want  time.Duration
	}{
		{"baseline", 72 * time.Hour},
		{"enhanced", 168 * time.Hour},
		{"strict", 336 * time.Hour},
		{"", 72 * time.Hour},
		{"no-such-level", 72 * time.Hour},
	}
	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			t.Parallel()
			got := ProjectInputsFromAnswers(types.WizardAnswers{ComplianceLevel: tt.level}).MinReleaseAge
			if got != tt.want {
				t.Errorf("MinReleaseAge = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestConfigFiles_StrictUpdateDelayEveryProfile checks that at the strict
// level every built-in profile's update tool and security overview state a
// delay of at least 14 days.
func TestConfigFiles_StrictUpdateDelayEveryProfile(t *testing.T) {
	t.Parallel()
	in := ProjectInputsFromAnswers(types.WizardAnswers{
		ComplianceLevel: "strict",
		Detected:        types.DetectedProject{HasGoMod: true},
	})
	for _, p := range DefaultProfileRegistry().List() {
		t.Run(p.Name, func(t *testing.T) {
			t.Parallel()
			files := mustConfigFiles(t, p, in)
			if f, ok := findFile(files, "renovate.json"); ok {
				if got := renovateAges(t, f)[""]; got != "14 days" {
					t.Errorf("renovate default minimumReleaseAge = %q, want \"14 days\"", got)
				}
			}
			if f, ok := findFile(files, ".github/dependabot.yml"); ok {
				for _, u := range parseDependabot(t, f).Updates {
					if u.Cooldown == nil || u.Cooldown.DefaultDays < 14 {
						t.Errorf("%s: cooldown = %+v, want default-days >= 14", u.PackageEcosystem, u.Cooldown)
					}
				}
			}
			doc, ok := findFile(files, "docs/security-overview.md")
			if !ok {
				t.Fatal("no docs/security-overview.md")
			}
			if !strings.Contains(string(doc.Content), "waits 14 day(s)") {
				t.Errorf("security overview does not state the 14-day update delay:\n%s", doc.Content)
			}
		})
	}
}
