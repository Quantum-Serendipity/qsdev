package devinit

import (
	"context"
	"errors"
	"os"
	"runtime"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/container"
	"github.com/Quantum-Serendipity/qsdev/internal/detect"
	"github.com/Quantum-Serendipity/qsdev/internal/doctor"
	"github.com/Quantum-Serendipity/qsdev/internal/sysinfo"
)

// TestMain installs the fake host prober for every test, then removes the
// shared lifecycle project template once every test has run.
func TestMain(m *testing.M) {
	host = fakeHostProber()
	code := m.Run()
	removeLifecycleTemplate()
	os.Exit(code)
}

// fakeHostProber describes a fixed host: a Debian-family Linux with every
// devenv prerequisite installed and no container runtime. Without it every
// project a test initializes would run `docker --version`, `docker compose
// version`, `ps` and the prerequisite version probes against the real machine,
// and the generated output would depend on what that machine has installed.
func fakeHostProber() hostProber {
	return hostProber{
		detectOptions: []detect.Option{
			detect.WithContainerProber(noContainerRuntime{}),
			detect.WithOSDetector(func() *sysinfo.OSInfo {
				return &sysinfo.OSInfo{OS: runtime.GOOS, Arch: runtime.GOARCH, Family: "debian"}
			}),
		},
		prerequisites: allPrerequisitesInstalled,
	}
}

// useRealHostProber makes the test probe the real machine, for tests that
// exercise the probing itself. host is shared package state, so the test must
// not run in parallel: t.Setenv panics if it does.
func useRealHostProber(t *testing.T) {
	t.Helper()
	t.Setenv("QSDEV_TEST_REAL_HOST_PROBER", "1")
	prev := host
	host = hostProber{prerequisites: CheckPrerequisites}
	t.Cleanup(func() { host = prev })
}

func allPrerequisitesInstalled(context.Context) PrerequisiteResult {
	var r PrerequisiteResult
	for _, c := range doctor.RequiredChecks() {
		r.Tools = append(r.Tools, PrerequisiteStatus{
			Name:     c.Name,
			Found:    true,
			Path:     "/usr/bin/" + c.Binary,
			Version:  "1.0.0",
			Required: c.Required,
		})
	}
	return r
}

// noContainerRuntime is a container.Prober for a host with no container
// runtime: nothing is on PATH and no probe command succeeds.
type noContainerRuntime struct{}

var _ container.Prober = noContainerRuntime{}

var errNoRuntime = errors.New("no container runtime on the fake host")

func (noContainerRuntime) LookPath(string) (string, error) { return "", errNoRuntime }
func (noContainerRuntime) Output(context.Context, string, ...string) ([]byte, error) {
	return nil, errNoRuntime
}
func (noContainerRuntime) ReadFile(string) ([]byte, error)  { return nil, os.ErrNotExist }
func (noContainerRuntime) Stat(string) (os.FileInfo, error) { return nil, os.ErrNotExist }
func (noContainerRuntime) Glob(string) ([]string, error)    { return nil, nil }
func (noContainerRuntime) CurrentUser() string              { return "" }
func (noContainerRuntime) Getenv(string) string             { return "" }
