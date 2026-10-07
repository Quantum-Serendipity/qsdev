package perfbudget

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/procexec"
)

// Spec is a command to measure. Env nil inherits the test's environment;
// Dir "" runs in the test's working directory.
type Spec struct {
	Argv  []string
	Stdin []byte
	Env   []string
	Dir   string
}

// Run runs spec b.Warmups times untimed and then n times timed, and returns
// the timed runs' statistics. A run's CPU time is its process's user plus
// system time, which on Linux comes from wait4's rusage and so includes
// every descendant the process waited for: `sh -c hook.py` counts python.
// Wall time is recorded for display only. A non-zero exit is a result, not
// an error (hooks deny by exiting 2), but a command that cannot be started,
// or whose exit code changes between runs, fails t.
func (b Budgets) Run(t testing.TB, spec Spec, n int) Stats {
	t.Helper()
	s, err := b.measure(spec, n)
	if err != nil {
		t.Fatalf("perfbudget: %v", err)
	}
	return s
}

func (b Budgets) measure(spec Spec, n int) (Stats, error) {
	if len(spec.Argv) == 0 {
		return Stats{}, errors.New("measuring a spec with no argv")
	}
	if n < 1 {
		return Stats{}, fmt.Errorf("measuring %q: %d timed runs, need at least one", spec.Argv, n)
	}
	cpu := make([]time.Duration, 0, n)
	wall := make([]time.Duration, 0, n)
	var last sample
	for i := range b.Warmups + n {
		r, err := runOnce(spec)
		if err != nil {
			return Stats{}, fmt.Errorf("measuring %q: %w", spec.Argv, err)
		}
		if i > 0 && r.exitCode != last.exitCode {
			return Stats{}, fmt.Errorf("measuring %q: exit code %d on run %d after %d on the runs before; stderr: %s",
				spec.Argv, r.exitCode, i+1, last.exitCode, strings.TrimSpace(r.stderr))
		}
		last = r
		if i < b.Warmups {
			continue
		}
		cpu = append(cpu, r.cpu)
		wall = append(wall, r.wall)
	}
	return Stats{
		N: n, CPU: newDist(cpu), Wall: newDist(wall),
		ExitCode: last.exitCode, Stdout: last.stdout, Stderr: last.stderr,
	}, nil
}

type sample struct {
	cpu, wall time.Duration
	exitCode  int
	stdout    string
	stderr    string
}

// runOnce runs spec to completion.
func runOnce(spec Spec) (sample, error) {
	cmd := procexec.Command(spec.Argv[0], spec.Argv[1:]...)
	cmd.Stdin = bytes.NewReader(spec.Stdin)
	cmd.Env = spec.Env
	cmd.Dir = spec.Dir
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	start := time.Now()
	err := cmd.Run()
	wall := time.Since(start)
	ps := cmd.ProcessState
	if ps == nil {
		return sample{}, fmt.Errorf("starting: %w", err)
	}
	return sample{
		cpu:      ps.UserTime() + ps.SystemTime(),
		wall:     wall,
		exitCode: ps.ExitCode(),
		stdout:   stdout.String(),
		stderr:   stderr.String(),
	}, nil
}

// RecalibrationRuns is how many timed runs of each calibration command a
// suite takes next to every measurement. Calibrating beside each one,
// rather than once per suite, keeps the factor true to the load that
// measurement saw on a machine whose load drifts, and five runs are cheap.
const RecalibrationRuns = 5

// Calibrate returns how much more CPU this machine needs than the reference
// machine: the CPU p50 of `python3 -c pass` plus that of the empty Go
// program noop (see BuildNoop), over the same sum in budgets.yaml. CPU
// budgets are multiplied by it, so the gate measures the code rather than
// the runner. Neither program is qsdev, so a qsdev start-up regression
// cannot raise the factor and hide itself. Each runs n timed times with env
// (nil inherits the test's), so a hermetic suite calibrates in the
// environment it measures in. On a machine whose load drifts, recalibrate
// next to each measurement with a small n.
func (b Budgets) Calibrate(t testing.TB, noop string, env []string, n int) float64 {
	t.Helper()
	python, native := calibrationSpecs(noop, env)
	return b.calibrate(t, python, native, n)
}

// calibrationSpecs returns the two calibration commands.
func calibrationSpecs(noop string, env []string) (python, native Spec) {
	return Spec{Argv: []string{"python3", "-c", "pass"}, Env: env}, Spec{Argv: []string{noop}, Env: env}
}

func (b Budgets) calibrate(t testing.TB, python, native Spec, n int) float64 {
	t.Helper()
	py := b.Run(t, python, n).CPU.P50
	nat := b.Run(t, native, n).CPU.P50
	f := calibrationFactor(b.Reference, py, nat)
	t.Logf("perfbudget: calibration %.2f (CPU p50: python3 %s ms, noop %s ms; reference %s: %s ms, %s ms)",
		f, ms(py), ms(nat), b.Reference.Machine, ms(b.Reference.PythonCPU), ms(b.Reference.NoopCPU))
	return f
}

// calibrationFactor is the measured calibration sum over the reference one.
func calibrationFactor(ref Reference, python, noop time.Duration) float64 {
	return float64(python+noop) / float64(ref.PythonCPU+ref.NoopCPU)
}
