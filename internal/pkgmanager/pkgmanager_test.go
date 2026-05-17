package pkgmanager

import (
	"context"
	"fmt"
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

	// OutputResults maps "name args..." to (output, error) results.
	OutputResults map[string]outputResult

	// Calls records all Run/Output invocations as "name arg1 arg2 ...".
	Calls []string
}

type lookPathResult struct {
	path string
	err  error
}

type outputResult struct {
	data []byte
	err  error
}

func NewMockRunner() *MockRunner {
	return &MockRunner{
		LookPathResults: make(map[string]lookPathResult),
		RunResults:      make(map[string]error),
		OutputResults:   make(map[string]outputResult),
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

func (m *MockRunner) Output(_ context.Context, name string, args ...string) ([]byte, error) {
	key := m.makeKey(name, args...)
	m.Calls = append(m.Calls, key)
	if r, ok := m.OutputResults[key]; ok {
		return r.data, r.err
	}
	return nil, nil
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
	// Constructors with nil runner should not panic.
	_ = NewApt(nil)
	_ = NewDnf(nil)
	_ = NewPacman(nil)
	_ = NewZypper(nil)
	_ = NewApk(nil)
	_ = NewXbps(nil)
	_ = NewEmerge(nil)
	_ = NewBrew(nil)
	_ = NewNix(nil, false)
	_ = NewWinget(nil)
	_ = NewScoop(nil)
	_ = NewChoco(nil)
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

func TestPacmanAURHelper(t *testing.T) {
	t.Run("paru preferred", func(t *testing.T) {
		mock := NewMockRunner()
		mock.LookPathResults["pacman"] = lookPathResult{path: "/usr/bin/pacman"}
		mock.LookPathResults["paru"] = lookPathResult{path: "/usr/bin/paru"}
		mock.LookPathResults["yay"] = lookPathResult{path: "/usr/bin/yay"}

		p := NewPacman(mock)
		if p.SearchCmd() != "paru -Ss" {
			t.Errorf("expected paru search, got %s", p.SearchCmd())
		}
	})

	t.Run("yay fallback", func(t *testing.T) {
		mock := NewMockRunner()
		mock.LookPathResults["pacman"] = lookPathResult{path: "/usr/bin/pacman"}
		mock.LookPathResults["yay"] = lookPathResult{path: "/usr/bin/yay"}

		p := NewPacman(mock)
		if p.SearchCmd() != "yay -Ss" {
			t.Errorf("expected yay search, got %s", p.SearchCmd())
		}
	})

	t.Run("no AUR helper", func(t *testing.T) {
		mock := NewMockRunner()
		mock.LookPathResults["pacman"] = lookPathResult{path: "/usr/bin/pacman"}

		p := NewPacman(mock)
		if p.SearchCmd() != "pacman -Ss" {
			t.Errorf("expected pacman search, got %s", p.SearchCmd())
		}
	})
}

func TestAptIsInstalled(t *testing.T) {
	mock := NewMockRunner()
	mock.LookPathResults["apt-get"] = lookPathResult{path: "/usr/bin/apt-get"}
	mock.OutputResults["dpkg -l git"] = outputResult{
		data: []byte("ii  git  1:2.39.2-1  amd64  fast, scalable, distributed revision control system\n"),
	}
	mock.OutputResults["dpkg -l missing"] = outputResult{
		data: []byte("dpkg-query: no packages found matching missing\n"),
		err:  fmt.Errorf("exit status 1"),
	}

	apt := NewApt(mock)
	if !apt.IsInstalled(context.Background(), "git") {
		t.Error("expected git to be installed")
	}
	if apt.IsInstalled(context.Background(), "missing") {
		t.Error("expected missing to not be installed")
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

func TestSearchCmds(t *testing.T) {
	mock := NewMockRunner()
	mock.LookPathResults["dnf"] = lookPathResult{path: "/usr/bin/dnf"}

	tests := []struct {
		pm     PackageManager
		expect string
	}{
		{NewApt(mock), "apt-cache search"},
		{NewDnf(mock), "dnf search"},
		{NewPacman(mock), "pacman -Ss"},
		{NewZypper(mock), "zypper search"},
		{NewApk(mock), "apk search"},
		{NewXbps(mock), "xbps-query -Rs"},
		{NewEmerge(mock), "emerge --search"},
		{NewBrew(mock), "brew search"},
		{NewNix(mock, false), "nix search nixpkgs"},
	}
	for _, tt := range tests {
		if got := tt.pm.SearchCmd(); got != tt.expect {
			t.Errorf("%s.SearchCmd()=%q, want %q", tt.pm.Name(), got, tt.expect)
		}
	}
}
