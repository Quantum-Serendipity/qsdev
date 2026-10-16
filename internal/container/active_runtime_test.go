package container

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
)

// recordingProber records the commands a probe runs.
type recordingProber struct {
	*mockProber
	ran []string
}

func (r *recordingProber) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	r.ran = append(r.ran, strings.Join(append([]string{name}, args...), " "))
	return r.mockProber.Output(ctx, name, args...)
}

// TestActiveRuntime: ActiveRuntime picks the runtime Detect makes active on
// every host shape, and runs no version, compose or availability probe.
func TestActiveRuntime(t *testing.T) {
	t.Parallel()
	nspawn := func(m *mockProber) *mockProber {
		m.lookPathResults["systemd-nspawn"] = "/usr/bin/systemd-nspawn"
		return m
	}
	withDocker := func(m *mockProber, version string) *mockProber {
		m.lookPathResults["docker"] = "/usr/bin/docker"
		m.outputResults["docker --version"] = outputResult{output: []byte(version)}
		m.outputResults["docker compose version"] = outputResult{output: []byte("Docker Compose version v2.24.0\n")}
		return m
	}
	podmanBroken := func(m *mockProber) *mockProber {
		m.outputResults["podman info --format {{.Host.Security.Rootless}}"] = outputResult{err: errors.New("no subuid")}
		return m
	}
	const dockerVersion = "Docker version 24.0.7, build afdd53b\n"
	const aliasVersion = "podman version 4.9.3\n"

	tests := []struct {
		name    string
		prober  *mockProber
		wantRan []string
	}{
		{"none", &mockProber{lookPathResults: map[string]string{}}, nil},
		{"podman rootless", newPodmanRootlessProber(), []string{"podman info --format {{.Host.Security.Rootless}}"}},
		{"podman rootful", newPodmanRootfulProber(), []string{"podman info --format {{.Host.Security.Rootless}}"}},
		{"docker", withDocker(&mockProber{lookPathResults: map[string]string{}, outputResults: map[string]outputResult{}}, dockerVersion), []string{"docker --version"}},
		{"podman preferred over docker", withDocker(newPodmanRootlessProber(), dockerVersion), []string{"podman info --format {{.Host.Security.Rootless}}"}},
		{"broken podman falls back to docker", withDocker(podmanBroken(newPodmanRootlessProber()), dockerVersion),
			[]string{"podman info --format {{.Host.Security.Rootless}}", "docker --version"}},
		{"docker alias of a broken podman falls back to nspawn", nspawn(withDocker(podmanBroken(newPodmanRootlessProber()), aliasVersion)),
			[]string{"podman info --format {{.Host.Security.Rootless}}", "docker --version"}},
		{"nspawn", nspawn(&mockProber{lookPathResults: map[string]string{}}), nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			info, err := Detect(context.Background(), tt.prober)
			if err != nil {
				t.Fatalf("Detect: %v", err)
			}
			r := &recordingProber{mockProber: tt.prober}
			if got := ActiveRuntime(context.Background(), r); got != info.Active {
				t.Errorf("ActiveRuntime = %v, want Detect's %v", got, info.Active)
			}
			if !slices.Equal(r.ran, tt.wantRan) {
				t.Errorf("ActiveRuntime ran %q, want only %q", r.ran, tt.wantRan)
			}
		})
	}
}
