package pkgmanager

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// Compile-time interface compliance checks for all 12 implementations.
var (
	_ PackageManager = (*Apt)(nil)
	_ PackageManager = (*Dnf)(nil)
	_ PackageManager = (*Pacman)(nil)
	_ PackageManager = (*Zypper)(nil)
	_ PackageManager = (*Apk)(nil)
	_ PackageManager = (*Xbps)(nil)
	_ PackageManager = (*Emerge)(nil)
	_ PackageManager = (*Brew)(nil)
	_ PackageManager = (*Nix)(nil)
	_ PackageManager = (*Winget)(nil)
	_ PackageManager = (*Scoop)(nil)
	_ PackageManager = (*Choco)(nil)
)

// Compile-time check that ExecRunner implements CommandRunner.
var _ CommandRunner = (*ExecRunner)(nil)

// MockRunner records command invocations for testing.
type MockRunner struct {
	// LookPathResults maps binary names to (path, error) results.
	LookPathResults map[string]lookPathResult

	// RunResults maps "name args..." to an error result.
	RunResults map[string]error

	// Calls records all Run invocations as "name arg1 arg2 ...".
	Calls []string
}

type lookPathResult struct {
	path string
	err  error
}

func NewMockRunner() *MockRunner {
	return &MockRunner{
		LookPathResults: make(map[string]lookPathResult),
		RunResults:      make(map[string]error),
	}
}

func (m *MockRunner) LookPath(name string) (string, error) {
	if r, ok := m.LookPathResults[name]; ok {
		return r.path, r.err
	}
	return "", fmt.Errorf("not found: %s", name)
}

func (m *MockRunner) Run(_ context.Context, name string, args ...string) error {
	key := m.makeKey(name, args...)
	m.Calls = append(m.Calls, key)
	if err, ok := m.RunResults[key]; ok {
		return err
	}
	return nil
}

func (m *MockRunner) makeKey(name string, args ...string) string {
	parts := append([]string{name}, args...)
	return strings.Join(parts, " ")
}

func TestMockRunnerRecordsCalls(t *testing.T) {
	mock := NewMockRunner()
	mock.LookPathResults["apt-get"] = lookPathResult{path: "/usr/bin/apt-get"}

	apt := NewApt(mock)
	if !apt.Available() {
		t.Fatal("expected apt to be available with mock")
	}

	ctx := context.Background()
	_ = apt.Install(ctx, "git", "curl")

	if len(mock.Calls) != 1 {
		t.Fatalf("expected 1 call, got %d: %v", len(mock.Calls), mock.Calls)
	}
	expected := "apt-get install -y git curl"
	if mock.Calls[0] != expected {
		t.Errorf("expected call %q, got %q", expected, mock.Calls[0])
	}
}

func TestMockRunnerLookPath(t *testing.T) {
	mock := NewMockRunner()
	mock.LookPathResults["brew"] = lookPathResult{path: "/opt/homebrew/bin/brew"}

	path, err := mock.LookPath("brew")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "/opt/homebrew/bin/brew" {
		t.Errorf("expected /opt/homebrew/bin/brew, got %s", path)
	}

	_, err = mock.LookPath("missing")
	if err == nil {
		t.Error("expected error for missing binary")
	}
}

func TestManagerNames(t *testing.T) {
	mock := NewMockRunner()
	tests := []struct {
		pm   PackageManager
		name string
	}{
		{NewApt(mock), "apt"},
		{NewDnf(mock), "dnf"},
		{NewPacman(mock), "pacman"},
		{NewZypper(mock), "zypper"},
		{NewApk(mock), "apk"},
		{NewXbps(mock), "xbps"},
		{NewEmerge(mock), "emerge"},
		{NewBrew(mock), "brew"},
		{NewNix(mock, false), "nix"},
		{NewWinget(mock), "winget"},
		{NewScoop(mock), "scoop"},
		{NewChoco(mock), "choco"},
	}
	for _, tt := range tests {
		if got := tt.pm.Name(); got != tt.name {
			t.Errorf("expected Name()=%q, got %q", tt.name, got)
		}
	}
}

func TestNilRunnerDefaults(t *testing.T) {
	t.Parallel()
	// A nil runner must be replaced by the production ExecRunner, streaming
	// to the process's output. (Winget/Scoop/Choco are runner-less stubs off
	// Windows, so they are not listed.)
	tests := []struct {
		name   string
		runner func() CommandRunner
	}{
		{"apt", func() CommandRunner { return NewApt(nil).runner }},
		{"dnf", func() CommandRunner { return NewDnf(nil).runner }},
		{"pacman", func() CommandRunner { return NewPacman(nil).runner }},
		{"zypper", func() CommandRunner { return NewZypper(nil).runner }},
		{"apk", func() CommandRunner { return NewApk(nil).runner }},
		{"xbps", func() CommandRunner { return NewXbps(nil).runner }},
		{"emerge", func() CommandRunner { return NewEmerge(nil).runner }},
		{"brew", func() CommandRunner { return NewBrew(nil).runner }},
		{"nix", func() CommandRunner { return NewNix(nil, false).runner }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			er, ok := tt.runner().(*ExecRunner)
			if !ok || er == nil {
				t.Fatalf("runner = %T, want non-nil *ExecRunner", tt.runner())
			}
			if er.Stdout != os.Stdout || er.Stderr != os.Stderr {
				t.Error("default runner does not stream to os.Stdout/os.Stderr")
			}
		})
	}
}

// TestExecRunnerHelperProcess is not a real test: ExecRunner tests re-run the
// test binary with pkgmanagerHelperEnv set to get a portable child process
// that writes to stdout and stderr and exits non-zero.
func TestExecRunnerHelperProcess(t *testing.T) {
	if os.Getenv(pkgmanagerHelperEnv) != "1" {
		return
	}
	fmt.Fprint(os.Stdout, "resolving dependencies")
	fmt.Fprint(os.Stderr, "E: Unable to locate package shellcheck")
	os.Exit(3)
}

const pkgmanagerHelperEnv = "QSDEV_PKGMANAGER_TEST_HELPER"

func TestExecRunnerRun_StreamsOutputAndReportsStderr(t *testing.T) {
	t.Setenv(pkgmanagerHelperEnv, "1")

	var stdout, stderr strings.Builder
	r := &ExecRunner{Stdout: &stdout, Stderr: &stderr}
	err := r.Run(context.Background(), os.Args[0], "-test.run=^TestExecRunnerHelperProcess$")
	if err == nil {
		t.Fatal("expected an error from a non-zero exit")
	}

	if !strings.Contains(stdout.String(), "resolving dependencies") {
		t.Errorf("stdout not streamed; got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Unable to locate package") {
		t.Errorf("stderr not streamed; got %q", stderr.String())
	}
	if !strings.Contains(err.Error(), "Unable to locate package shellcheck") {
		t.Errorf("error does not quote the command's stderr: %v", err)
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 3 {
		t.Errorf("error does not wrap the *exec.ExitError (code 3): %v", err)
	}
}

func TestExecRunnerRun_NilWritersStillReportStderr(t *testing.T) {
	t.Setenv(pkgmanagerHelperEnv, "1")

	err := (&ExecRunner{}).Run(context.Background(), os.Args[0], "-test.run=^TestExecRunnerHelperProcess$")
	if err == nil || !strings.Contains(err.Error(), "Unable to locate package shellcheck") {
		t.Fatalf("error = %v, want it to quote the command's stderr", err)
	}
}

func TestTailBufferKeepsLastBytes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		max    int
		writes []string
		want   string
	}{
		{"under limit", 10, []string{"abc", "def"}, "abcdef"},
		{"single write over limit", 4, []string{"abcdefgh"}, "efgh"},
		{"spans writes", 5, []string{"abc", "defg", "hi"}, "efghi"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tb := &tailBuffer{max: tt.max}
			for _, w := range tt.writes {
				if n, err := tb.Write([]byte(w)); err != nil || n != len(w) {
					t.Fatalf("Write(%q) = %d, %v", w, n, err)
				}
			}
			if got := tb.String(); got != tt.want {
				t.Errorf("tail = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNixNixOSReturnsError(t *testing.T) {
	mock := NewMockRunner()
	mock.LookPathResults["nix"] = lookPathResult{path: "/nix/store/bin/nix"}

	nix := NewNix(mock, true)
	err := nix.Install(context.Background(), "git", "curl")
	if err == nil {
		t.Fatal("expected error on NixOS install")
	}
	if !strings.Contains(err.Error(), "configuration.nix") {
		t.Errorf("expected error to mention configuration.nix, got: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "git") {
		t.Errorf("expected error to mention packages, got: %s", err.Error())
	}
}

func TestNixImperativeInstall(t *testing.T) {
	mock := NewMockRunner()
	mock.LookPathResults["nix"] = lookPathResult{path: "/nix/store/bin/nix"}

	nix := NewNix(mock, false)
	err := nix.Install(context.Background(), "git", "curl")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.Calls) != 2 {
		t.Fatalf("expected 2 calls, got %d: %v", len(mock.Calls), mock.Calls)
	}
	if mock.Calls[0] != "nix profile install nixpkgs#git" {
		t.Errorf("unexpected first call: %s", mock.Calls[0])
	}
	if mock.Calls[1] != "nix profile install nixpkgs#curl" {
		t.Errorf("unexpected second call: %s", mock.Calls[1])
	}
}

func TestDnfYumFallback(t *testing.T) {
	mock := NewMockRunner()
	// Only yum is available, not dnf.
	mock.LookPathResults["yum"] = lookPathResult{path: "/usr/bin/yum"}

	dnf := NewDnf(mock)
	if !dnf.Available() {
		t.Fatal("expected dnf to be available via yum fallback")
	}
	if dnf.cmd() != "yum" {
		t.Errorf("expected yum as command, got %s", dnf.cmd())
	}

	_ = dnf.Install(context.Background(), "git")
	if len(mock.Calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(mock.Calls))
	}
	if mock.Calls[0] != "yum install -y git" {
		t.Errorf("unexpected call: %s", mock.Calls[0])
	}
}

func TestElevation(t *testing.T) {
	mock := NewMockRunner()
	elevated := []PackageManager{
		NewApt(mock), NewDnf(mock), NewPacman(mock),
		NewZypper(mock), NewApk(mock), NewXbps(mock), NewEmerge(mock),
	}
	for _, pm := range elevated {
		if !pm.NeedsElevation() {
			t.Errorf("%s should need elevation", pm.Name())
		}
	}

	notElevated := []PackageManager{
		NewBrew(mock), NewNix(mock, false),
		NewWinget(mock), NewScoop(mock), NewChoco(mock),
	}
	for _, pm := range notElevated {
		if pm.NeedsElevation() {
			t.Errorf("%s should not need elevation", pm.Name())
		}
	}
}

// TestInstallRunsInstallArgs pins F474: Install must run exactly the command
// InstallArgs reports, since setup's elevated path and every displayed
// command are built from InstallArgs.
func TestInstallRunsInstallArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		pm   func(CommandRunner) PackageManager
		path []string // binaries present on PATH
	}{
		{"apt", func(r CommandRunner) PackageManager { return NewApt(r) }, []string{"apt-get"}},
		{"dnf", func(r CommandRunner) PackageManager { return NewDnf(r) }, []string{"dnf"}},
		{"yum fallback", func(r CommandRunner) PackageManager { return NewDnf(r) }, []string{"yum"}},
		{"pacman", func(r CommandRunner) PackageManager { return NewPacman(r) }, []string{"pacman"}},
		{"zypper", func(r CommandRunner) PackageManager { return NewZypper(r) }, []string{"zypper"}},
		{"apk", func(r CommandRunner) PackageManager { return NewApk(r) }, []string{"apk"}},
		{"xbps", func(r CommandRunner) PackageManager { return NewXbps(r) }, []string{"xbps-install"}},
		{"emerge", func(r CommandRunner) PackageManager { return NewEmerge(r) }, []string{"emerge"}},
		{"brew", func(r CommandRunner) PackageManager { return NewBrew(r) }, []string{"brew"}},
		{"nix", func(r CommandRunner) PackageManager { return NewNix(r, false) }, []string{"nix"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mock := NewMockRunner()
			for _, bin := range tt.path {
				mock.LookPathResults[bin] = lookPathResult{path: "/usr/bin/" + bin}
			}
			pm := tt.pm(mock)
			if err := pm.Install(context.Background(), "pkg"); err != nil {
				t.Fatalf("Install: %v", err)
			}
			bin, args := pm.InstallArgs("pkg")
			want := strings.Join(append([]string{bin}, args...), " ")
			if len(mock.Calls) != 1 || mock.Calls[0] != want {
				t.Errorf("Install ran %v, InstallArgs reports %q", mock.Calls, want)
			}
		})
	}
}
