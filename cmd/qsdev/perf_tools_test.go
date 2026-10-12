package main

import (
	"fmt"
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/doctor"
	"github.com/Quantum-Serendipity/qsdev/internal/procexec"
)

// probedTools returns the host tools a read-only command may run in
// project ("" before one exists), as qsdev's own registries list them: the
// binaries procexec's forbid-exec guard lets it run by name, and every tool
// doctor version-probes there (doctor.ProjectChecks, which adds the
// programs the project's hooks run). TestCLIPerf stands a stub in for each,
// so its rows measure qsdev's probing and not the host's tools; a tool
// outside both registries is not on its PATH, so a probe of it finds the
// tool absent rather than running the host's. Sorted, without repeats.
func probedTools(project string) ([]string, error) {
	checks, err := doctor.ProjectChecks(project)
	if err != nil {
		return nil, fmt.Errorf("listing the tools doctor checks: %w", err)
	}
	names := procexec.LocalProbeNames()
	for _, c := range checks {
		names = append(names, c.Binary)
		names = append(names, c.AltBinaries...)
	}
	slices.Sort(names)
	return slices.Compact(names), nil
}

// TestProbedTools: the stubbed set covers every binary the forbid-exec guard
// allows by name and every tool doctor probes, each once.
func TestProbedTools(t *testing.T) {
	t.Parallel()
	got, err := probedTools("")
	if err != nil {
		t.Fatalf("probedTools: %v", err)
	}
	if !slices.IsSorted(got) || len(slices.Compact(slices.Clone(got))) != len(got) {
		t.Errorf("probedTools = %q, want sorted names without repeats", got)
	}
	want := procexec.LocalProbeNames()
	for _, c := range doctor.DefaultChecks() {
		want = append(want, c.Binary)
		want = append(want, c.AltBinaries...)
	}
	for _, name := range want {
		if !slices.Contains(got, name) {
			t.Errorf("probedTools lacks %q, which a read-only command may probe", name)
		}
	}
}
