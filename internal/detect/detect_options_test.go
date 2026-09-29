package detect

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/sysinfo"
)

// fakeProber is a container.Prober whose PATH holds only the given binaries.
// Output blocks until ctx is done when hang is set, modelling a wedged
// `podman info`; otherwise it returns outputs[name+" "+args].
type fakeProber struct {
	onPath  map[string]bool
	outputs map[string]string
	hang    bool
}

func (p *fakeProber) LookPath(name string) (string, error) {
	if p.onPath[name] {
		return "/usr/bin/" + name, nil
	}
	return "", errors.New("not found")
}

func (p *fakeProber) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	if p.hang {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	out, ok := p.outputs[name+" "+strings.Join(args, " ")]
	if !ok {
		return nil, errors.New("no output")
	}
	return []byte(out), nil
}

func (p *fakeProber) ReadFile(string) ([]byte, error)  { return nil, os.ErrNotExist }
func (p *fakeProber) Stat(string) (os.FileInfo, error) { return nil, os.ErrNotExist }
func (p *fakeProber) Glob(string) ([]string, error)    { return nil, nil }
func (p *fakeProber) CurrentUser() string              { return "tester" }
func (p *fakeProber) Getenv(string) string             { return "" }
func fakeOS(family string) func() *sysinfo.OSInfo {
	return func() *sysinfo.OSInfo { return &sysinfo.OSInfo{Family: family} }
}
func fakeUser(name string) func() (string, error) { return func() (string, error) { return name, nil } }
func failingUser() (string, error)                { return "", errors.New("no user") }
func noRuntimes() *fakeProber                     { return &fakeProber{} }
func hostFakes() []Option {
	return []Option{WithContainerProber(noRuntimes()), WithOSDetector(fakeOS("debian")), WithUserLookup(fakeUser("tester"))}
}

// TestDetect_HostFields covers the ContainerRuntime, OSFamily and Username
// branches, which were untestable while Detect called the host directly.
func TestDetect_HostFields(t *testing.T) {
	t.Parallel()

	docker := &fakeProber{
		onPath:  map[string]bool{"docker": true},
		outputs: map[string]string{"docker --version": "Docker version 27.0.1, build abc"},
	}

	tests := []struct {
		name        string
		opts        []Option
		wantRuntime string
		wantFamily  string
		wantUser    string
	}{
		{
			name:        "docker runtime",
			opts:        []Option{WithContainerProber(docker), WithOSDetector(fakeOS("arch")), WithUserLookup(fakeUser("alice"))},
			wantRuntime: "docker",
			wantFamily:  "arch",
			wantUser:    "alice",
		},
		{
			name:       "no runtime, user lookup fails",
			opts:       []Option{WithContainerProber(noRuntimes()), WithOSDetector(fakeOS("macos")), WithUserLookup(failingUser)},
			wantFamily: "macos",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dp := Detect(t.Context(), t.TempDir(), tt.opts...)
			if dp.ContainerRuntime != tt.wantRuntime {
				t.Errorf("ContainerRuntime = %q, want %q", dp.ContainerRuntime, tt.wantRuntime)
			}
			if dp.OSFamily != tt.wantFamily {
				t.Errorf("OSFamily = %q, want %q", dp.OSFamily, tt.wantFamily)
			}
			if dp.Username != tt.wantUser {
				t.Errorf("Username = %q, want %q", dp.Username, tt.wantUser)
			}
		})
	}
}

// TestDetect_HangingContainerProbeIsBounded pins F475/F524/F545: a wedged
// container runtime (e.g. `podman info` stuck on a storage lock) must not
// block detection past the probe timeout, and caller cancellation must stop
// it too.
func TestDetect_HangingContainerProbeIsBounded(t *testing.T) {
	t.Parallel()

	hung := &fakeProber{onPath: map[string]bool{"podman": true}, hang: true}

	tests := []struct {
		name string
		ctx  func(t *testing.T) context.Context
		opts []Option
	}{
		{
			name: "probe timeout",
			ctx:  func(t *testing.T) context.Context { return t.Context() },
			opts: []Option{WithContainerProbeTimeout(50 * time.Millisecond)},
		},
		{
			name: "caller cancellation",
			ctx: func(t *testing.T) context.Context {
				ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
				t.Cleanup(cancel)
				return ctx
			},
			// Only ctx can end the probe: the timeout alone would outlast the test.
			opts: []Option{WithContainerProbeTimeout(time.Hour)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			opts := append([]Option{WithContainerProber(hung), WithOSDetector(fakeOS("debian")), WithUserLookup(fakeUser("u"))}, tt.opts...)

			ctx, root := tt.ctx(t), t.TempDir()
			done := make(chan struct{})
			go func() {
				defer close(done)
				Detect(ctx, root, opts...)
			}()
			select {
			case <-done:
			case <-time.After(10 * time.Second):
				t.Fatal("Detect blocked on a hanging container probe")
			}
		})
	}
}
