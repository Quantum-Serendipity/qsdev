package perfbudget

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/testutil"
)

// helperEnv makes the test binary act as a measured child instead of
// running tests. Its value is "<verb>:<arg>": "spin:<duration>" burns that
// much CPU time of its own, "exit:<code>" burns exitSpin of CPU, writes to
// stdout and stderr and exits with code, and "count:<file>" exits with the
// number in file and then increments it, so its exit code changes from run
// to run.
const helperEnv = "PERFBUDGET_TEST_HELPER"

// exitSpin is the CPU time an "exit" helper burns, so that a quick helper
// still registers CPU time where the OS accounts it in clock ticks
// (Windows' GetProcessTimes advances in 15.6 ms steps) and the tests
// asserting a positive measurement hold on every OS.
const exitSpin = 20 * time.Millisecond

func TestMain(m *testing.M) {
	if mode := os.Getenv(helperEnv); mode != "" {
		os.Exit(runHelper(mode))
	}
	os.Exit(m.Run())
}

func runHelper(mode string) int {
	verb, arg, _ := strings.Cut(mode, ":")
	switch verb {
	case "spin":
		d, err := time.ParseDuration(arg)
		if err == nil {
			err = spinCPU(d)
		}
		return helperResult(0, err)
	case "exit":
		code, err := strconv.Atoi(arg)
		if err == nil {
			err = spinCPU(exitSpin)
		}
		fmt.Print("helper stdout")
		fmt.Fprint(os.Stderr, "helper stderr")
		return helperResult(code, err)
	case "count":
		data, err := os.ReadFile(arg)
		if os.IsNotExist(err) {
			data, err = []byte("0"), nil
		}
		n, convErr := strconv.Atoi(string(data))
		if err == nil {
			err = convErr
		}
		if err == nil {
			err = os.WriteFile(arg, []byte(strconv.Itoa(n+1)), 0o644)
		}
		return helperResult(n, err)
	}
	return helperResult(0, fmt.Errorf("unknown verb %q", verb))
}

// helperResult is code, or 100 after reporting err.
func helperResult(code int, err error) int {
	if err != nil {
		fmt.Fprintln(os.Stderr, "helper:", err)
		return 100
	}
	return code
}

// helperSpec runs this test binary as a helper in mode.
func helperSpec(t *testing.T, mode string) Spec {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return Spec{Argv: []string{exe, "-test.run=^$"}, Env: append(os.Environ(), helperEnv+"="+mode)}
}

// runBudgets is a document with no warm-ups and no allowance, so a
// measured limit is exactly budget × factor.
func runBudgets(t *testing.T) Budgets {
	t.Helper()
	b, err := Parse([]byte(`allowance: 1
runs: 1
warmups: 0
` + validReference + `budgets:
  - id: hook
    surface: a hook
    cpu_p95_ms: 40
  - id: roomy
    surface: a budget no helper can miss
    cpu_p95_ms: 2000
  - id: tracked
    surface: a budget with a tracked skip
    cpu_p95_ms: 40
    skips:
      - {workstream: XD-WS3, subjects: [helper], reason: start-up, ceiling_ms: 2000}
  - id: capped
    surface: a budget with a tracked skip whose ceiling is near the budget
    cpu_p95_ms: 40
    skips:
      - {workstream: XD-WS3, subjects: [helper], reason: start-up, ceiling_ms: 60}
`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return b
}

// TestQuickHelperWithinBudget: a quick helper passes a budget far above
// process start-up, and Run keeps its exit code and output.
func TestQuickHelperWithinBudget(t *testing.T) {
	summaryFile(t)
	b := runBudgets(t)
	s := b.Run(t, helperSpec(t, "exit:3"), 3)
	if s.N != 3 || s.ExitCode != 3 || s.Stdout != "helper stdout" || s.Stderr != "helper stderr" {
		t.Errorf("Run = %+v, want 3 runs, rc 3 and the helper's output", s)
	}
	if s.CPU.P50 <= 0 || s.Wall.P50 <= 0 || s.CPU.P50 > s.CPU.P95 || s.CPU.P95 > s.CPU.Max {
		t.Errorf("Run statistics %+v are not ordered positive durations", s)
	}
	row := Row{Budget: "roomy", Subject: "helper", Payload: "bash_ls"}
	if r := b.Check(t, row, s, testutil.RaceScale); r.Verdict != VerdictPass {
		t.Errorf("Check = %+v, want a pass", r)
	}
}

func TestRunRejectsChangingExitCode(t *testing.T) {
	t.Parallel()
	b := runBudgets(t)
	b.Warmups = 1
	counter := filepath.Join(t.TempDir(), "count")
	_, err := b.measure(helperSpec(t, "count:"+counter), 2)
	if err == nil || !strings.Contains(err.Error(), "exit code 1 on run 2 after 0") {
		t.Fatalf("measure = %v, want an error naming the changed exit code", err)
	}
	if data, _ := os.ReadFile(counter); string(data) != "2" {
		t.Errorf("the helper ran %s times, want measuring to stop at the first change", data)
	}
}

func TestRunNeedsArgv(t *testing.T) {
	t.Parallel()
	b := runBudgets(t)
	tests := []struct {
		name string
		spec Spec
		n    int
		want string
	}{
		{name: "no argv", spec: Spec{}, n: 1, want: "no argv"},
		{name: "no timed runs", spec: helperSpec(t, "exit:0"), n: 0, want: "at least one"},
		{name: "cannot start", spec: Spec{Argv: []string{filepath.Join(t.TempDir(), "missing")}}, n: 1, want: "starting"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := b.measure(tt.spec, tt.n); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("measure = %v, want an error mentioning %q", err, tt.want)
			}
		})
	}
}

func TestCalibrationFactor(t *testing.T) {
	t.Parallel()
	ref := Reference{Machine: "ref", PythonCPU: 30 * time.Millisecond, NoopCPU: 10 * time.Millisecond}
	tests := []struct {
		name         string
		python, noop time.Duration
		want         float64
	}{
		{name: "same machine", python: 30 * time.Millisecond, noop: 10 * time.Millisecond, want: 1},
		{name: "twice as slow", python: 60 * time.Millisecond, noop: 20 * time.Millisecond, want: 2},
		{name: "faster", python: 15 * time.Millisecond, noop: 5 * time.Millisecond, want: 0.5},
		{name: "only the sum counts", python: 35 * time.Millisecond, noop: 5 * time.Millisecond, want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := calibrationFactor(ref, tt.python, tt.noop); got != tt.want {
				t.Errorf("calibrationFactor = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestCalibrateUsesNoopNotQsdev: calibration times python and the empty Go
// program, never qsdev, so a qsdev start-up regression cannot raise the
// factor that scales its own budget.
func TestCalibrateUsesNoopNotQsdev(t *testing.T) {
	t.Parallel()
	env := []string{"HOME=/isolated"}
	noop := filepath.Join(t.TempDir(), "perfbudget-noop")
	python, native := calibrationSpecs(noop, env)
	for _, s := range []Spec{python, native} {
		for _, arg := range s.Argv {
			if strings.Contains(strings.ToLower(filepath.Base(arg)), "qsdev") {
				t.Errorf("calibration runs %q, which names qsdev", s.Argv)
			}
		}
		if !slices.Equal(s.Env, env) {
			t.Errorf("calibration spec %q has env %q, want the caller's %q", s.Argv, s.Env, env)
		}
	}
	if !slices.Equal(python.Argv, []string{"python3", "-c", "pass"}) || !slices.Equal(native.Argv, []string{noop}) {
		t.Errorf("calibration specs = %q and %q, want python3 -c pass and %s", python.Argv, native.Argv, noop)
	}
	if !strings.HasSuffix(noopPackage, "/internal/perfbudget/testdata/noop") {
		t.Errorf("noopPackage = %q, want this package's testdata/noop", noopPackage)
	}
	if _, err := os.Stat(filepath.Join("testdata", "noop", "main.go")); err != nil {
		t.Errorf("the no-op program BuildNoop builds is missing: %v", err)
	}

	// The arithmetic end to end, with the helper standing in for both.
	b := runBudgets(t)
	quick := helperSpec(t, "exit:0")
	if f := b.calibrate(t, quick, quick, 2); f <= 0 {
		t.Errorf("calibrate = %v, want a positive factor", f)
	}
}
