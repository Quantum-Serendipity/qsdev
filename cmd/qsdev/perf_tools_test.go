package main

import (
	"fmt"
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/check"
	"github.com/Quantum-Serendipity/qsdev/internal/doctor"
	"github.com/Quantum-Serendipity/qsdev/internal/procexec"
)

// probedTools returns the host tools TestCLIPerf stands an empty stub in
// for when a read-only command runs in project ("" before one exists), as
// qsdev's own registries list them: every tool doctor version-probes there
// (doctor.ProjectChecks, which adds the programs the project's hooks run)
// and every binary procexec's forbid-exec guard runs by name only for its
// version or to check syntax. A stub answers those as a tool of unknown
// version and valid input. A binary the guard queries (podman's mode,
// whether docker is Podman's alias, ps, getent, uname, ...) is left off:
// its empty answer would read as another host, such as rootful Podman, and
// init would record it. So is any tool outside both registries,
// such as the toolchains check's language requirements probe. A command
// finds those absent, and the host's never run. Sorted, without repeats.
func probedTools(project string) ([]string, error) {
	checks, err := doctor.ProjectChecks(project)
	if err != nil {
		return nil, fmt.Errorf("listing the tools doctor checks: %w", err)
	}
	names := procexec.LocalProbeNames(procexec.ProbeVersion, procexec.ProbeSyntax)
	for _, c := range checks {
		names = append(names, c.Binary)
		names = append(names, c.AltBinaries...)
	}
	queried := queriedTools()
	names = slices.DeleteFunc(names, func(name string) bool { return slices.Contains(queried, name) })
	slices.Sort(names)
	return slices.Compact(names), nil
}

// queriedTools returns the binaries the forbid-exec guard runs by name for
// an answer the caller parses (see procexec.ProbeQuery).
func queriedTools() []string {
	stubbable := procexec.LocalProbeNames(procexec.ProbeVersion, procexec.ProbeSyntax)
	return slices.DeleteFunc(procexec.LocalProbeNames(), func(name string) bool { return slices.Contains(stubbable, name) })
}

// TestProbedTools: the stubbed set holds every tool doctor probes and every
// binary the guard runs only for a version or syntax, never a queried one,
// each once.
func TestProbedTools(t *testing.T) {
	t.Parallel()
	got, err := probedTools("")
	if err != nil {
		t.Fatalf("probedTools: %v", err)
	}
	checkProbedTools(t, got, nil)
	for _, name := range procexec.LocalProbeNames(procexec.ProbeVersion, procexec.ProbeSyntax) {
		if !slices.Contains(got, name) {
			t.Errorf("probedTools lacks %q, which the guard runs for its version or syntax", name)
		}
	}
}

// TestProbedTools_GeneratedProject: in a generated project, the stubbed set
// also holds every program the project's hooks run, which doctor requires
// there. qsdev init generates the project in a process of its own, as the
// command tree keeps state between in-process runs (see cliHelperEnv).
func TestProbedTools_GeneratedProject(t *testing.T) {
	t.Parallel()
	env := guardrailEnv(t)
	dir := newGuardrailProject(t, env)
	mustQsdev(t, env, dir, "init", "--yes")
	programs, err := check.HookHostPrograms(dir)
	if err != nil {
		t.Fatalf("HookHostPrograms: %v", err)
	}
	if len(programs) == 0 {
		t.Fatal("the generated project's hooks run no host program; the fixture lost its hooks")
	}
	got, err := probedTools(dir)
	if err != nil {
		t.Fatalf("probedTools: %v", err)
	}
	checkProbedTools(t, got, programs)
}

// checkProbedTools fails t unless got is sorted without repeats, holds
// every binary of doctor's default registry and every name in extra, and no
// queried binary.
func checkProbedTools(t *testing.T, got, extra []string) {
	t.Helper()
	if !slices.IsSorted(got) || len(slices.Compact(slices.Clone(got))) != len(got) {
		t.Errorf("probedTools = %q, want sorted names without repeats", got)
	}
	queried := queriedTools()
	want := slices.Clone(extra)
	for _, c := range doctor.DefaultChecks() {
		want = append(want, c.Binary)
		want = append(want, c.AltBinaries...)
	}
	for _, name := range want {
		if !slices.Contains(got, name) && !slices.Contains(queried, name) {
			t.Errorf("probedTools lacks %q, which doctor probes", name)
		}
	}
	for _, name := range queried {
		if slices.Contains(got, name) {
			t.Errorf("probedTools stubs %q, whose queried answer a stub would fake", name)
		}
	}
}
