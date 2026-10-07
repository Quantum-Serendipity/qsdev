package canon

import (
	"slices"
	"testing"
)

// TestSegments_MatchProtection pins that Segments lists exactly the segments
// IsProtected matches, so the hook sandbox's guardrails cover them all.
func TestSegments_MatchProtection(t *testing.T) {
	t.Parallel()
	segs := Segments()
	if len(segs) != len(brandedTables().segments) {
		t.Fatalf("Segments() has %d entries, the tables %d", len(segs), len(brandedTables().segments))
	}
	for _, want := range []string{HookLogDir + "/", ".envrc"} {
		if !slices.Contains(segs, want) {
			t.Errorf("Segments() = %v, missing %q", segs, want)
		}
	}
}

// TestDevenvSourceFiles_AreEnvSources pins that the project's devenv files
// are environment sources and not the shell startup files of a home
// directory.
func TestDevenvSourceFiles_AreEnvSources(t *testing.T) {
	t.Parallel()
	env := EnvSourceFiles()
	for _, f := range DevenvSourceFiles() {
		if !slices.Contains(env, f) {
			t.Errorf("devenv source %q is not in EnvSourceFiles() %v", f, env)
		}
	}
	if slices.Contains(DevenvSourceFiles(), ".bashrc") {
		t.Error("DevenvSourceFiles lists a shell startup file")
	}
}

func TestGuardedConfigFiles_Sorted(t *testing.T) {
	t.Parallel()
	got := GuardedConfigFiles()
	if !slices.IsSorted(got) || len(got) == 0 {
		t.Errorf("GuardedConfigFiles() = %v, want a sorted non-empty list", got)
	}
}
