package container

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

type mockProber struct {
	lookPathResults map[string]string
	outputResults   map[string]outputResult
	files           map[string][]byte
	fileInfos       map[string]bool // path -> exists
	globResults     map[string][]string
	user            string
	env             map[string]string
}

type outputResult struct {
	output []byte
	err    error
}

func (m *mockProber) LookPath(name string) (string, error) {
	if path, ok := m.lookPathResults[name]; ok {
		return path, nil
	}
	return "", fmt.Errorf("not found: %s", name)
}

func (m *mockProber) Output(_ context.Context, name string, args ...string) ([]byte, error) {
	key := name
	if len(args) > 0 {
		key = name + " " + strings.Join(args, " ")
	}
	if r, ok := m.outputResults[key]; ok {
		return r.output, r.err
	}
	return nil, fmt.Errorf("no mock output for: %s", key)
}

func (m *mockProber) ReadFile(path string) ([]byte, error) {
	if data, ok := m.files[path]; ok {
		return data, nil
	}
	return nil, os.ErrNotExist
}

func (m *mockProber) Stat(path string) (os.FileInfo, error) {
	if m.fileInfos != nil {
		if exists, ok := m.fileInfos[path]; ok && exists {
			return &mockFileInfo{}, nil
		}
	}
	return nil, os.ErrNotExist
}

func (m *mockProber) Glob(pattern string) ([]string, error) {
	if matches, ok := m.globResults[pattern]; ok {
		return matches, nil
	}
	return nil, nil
}

func (m *mockProber) CurrentUser() string { return m.user }

func (m *mockProber) Getenv(key string) string {
	if m.env != nil {
		return m.env[key]
	}
	return ""
}

type mockFileInfo struct{}

func (f *mockFileInfo) Name() string       { return "mock" }
func (f *mockFileInfo) Size() int64        { return 0 }
func (f *mockFileInfo) Mode() os.FileMode  { return 0o644 }
func (f *mockFileInfo) ModTime() time.Time { return time.Time{} }
func (f *mockFileInfo) IsDir() bool        { return false }
func (f *mockFileInfo) Sys() any           { return nil }

func newPodmanRootlessProber() *mockProber {
	return &mockProber{
		lookPathResults: map[string]string{
			"podman": "/usr/bin/podman",
		},
		outputResults: map[string]outputResult{
			"podman version --format {{.Client.Version}}":      {output: []byte("4.9.3\n")},
			"podman info --format {{.Host.Security.Rootless}}": {output: []byte("true\n")},
		},
		env: map[string]string{
			"XDG_RUNTIME_DIR": "/run/user/1000",
		},
	}
}

func newPodmanRootfulProber() *mockProber {
	return &mockProber{
		lookPathResults: map[string]string{
			"podman": "/usr/bin/podman",
		},
		outputResults: map[string]outputResult{
			"podman version --format {{.Client.Version}}":      {output: []byte("4.9.3\n")},
			"podman info --format {{.Host.Security.Rootless}}": {output: []byte("false\n")},
		},
	}
}

func newDockerProber() *mockProber {
	return &mockProber{
		lookPathResults: map[string]string{
			"docker": "/usr/bin/docker",
		},
		outputResults: map[string]outputResult{
			"docker --version": {output: []byte("Docker version 24.0.7, build afdd53b\n")},
		},
	}
}

func TestDetect_PodmanRootlessOnly(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prober := newPodmanRootlessProber()

	info, err := Detect(ctx, prober)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if info.Active != RuntimePodmanRootless {
		t.Errorf("Active = %v, want %v", info.Active, RuntimePodmanRootless)
	}
	if !info.Rootless {
		t.Error("Rootless = false, want true")
	}
	if info.Version != "4.9.3" {
		t.Errorf("Version = %q, want %q", info.Version, "4.9.3")
	}
	if info.Path != "/usr/bin/podman" {
		t.Errorf("Path = %q, want %q", info.Path, "/usr/bin/podman")
	}
	if info.SocketPath != "/run/user/1000/podman/podman.sock" {
		t.Errorf("SocketPath = %q, want %q", info.SocketPath, "/run/user/1000/podman/podman.sock")
	}
	if len(info.Available) != 1 || info.Available[0] != RuntimePodmanRootless {
		t.Errorf("Available = %v, want [podman-rootless]", info.Available)
	}
}

func TestDetect_PodmanRootfulOnly(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prober := newPodmanRootfulProber()

	info, err := Detect(ctx, prober)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if info.Active != RuntimePodmanRootful {
		t.Errorf("Active = %v, want %v", info.Active, RuntimePodmanRootful)
	}
	if info.Rootless {
		t.Error("Rootless = true, want false")
	}
	if info.SocketPath != "/run/podman/podman.sock" {
		t.Errorf("SocketPath = %q, want %q", info.SocketPath, "/run/podman/podman.sock")
	}
	if len(info.Available) != 1 || info.Available[0] != RuntimePodmanRootful {
		t.Errorf("Available = %v, want [podman-rootful]", info.Available)
	}
}

func TestDetect_DockerOnly(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prober := newDockerProber()

	info, err := Detect(ctx, prober)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if info.Active != RuntimeDocker {
		t.Errorf("Active = %v, want %v", info.Active, RuntimeDocker)
	}
	if info.Path != "/usr/bin/docker" {
		t.Errorf("Path = %q, want %q", info.Path, "/usr/bin/docker")
	}
	if info.SocketPath != "/var/run/docker.sock" {
		t.Errorf("SocketPath = %q, want %q", info.SocketPath, "/var/run/docker.sock")
	}
	if info.Version != "24.0.7" {
		t.Errorf("Version = %q, want %q", info.Version, "24.0.7")
	}
	if len(info.Available) != 1 || info.Available[0] != RuntimeDocker {
		t.Errorf("Available = %v, want [docker]", info.Available)
	}
}

func TestDetect_BothPodmanAndDocker(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prober := &mockProber{
		lookPathResults: map[string]string{
			"podman": "/usr/bin/podman",
			"docker": "/usr/bin/docker",
		},
		outputResults: map[string]outputResult{
			"podman version --format {{.Client.Version}}":      {output: []byte("4.9.3\n")},
			"podman info --format {{.Host.Security.Rootless}}": {output: []byte("true\n")},
			"docker --version": {output: []byte("Docker version 24.0.7, build afdd53b\n")},
		},
		env: map[string]string{
			"XDG_RUNTIME_DIR": "/run/user/1000",
		},
	}

	info, err := Detect(ctx, prober)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	// Podman should be preferred.
	if info.Active != RuntimePodmanRootless {
		t.Errorf("Active = %v, want %v (Podman preferred)", info.Active, RuntimePodmanRootless)
	}
	if len(info.Available) != 2 {
		t.Errorf("Available has %d entries, want 2: %v", len(info.Available), info.Available)
	}
	// Docker should still be listed as available.
	hasDocker := false
	for _, r := range info.Available {
		if r == RuntimeDocker {
			hasDocker = true
		}
	}
	if !hasDocker {
		t.Errorf("Available = %v, expected docker in list", info.Available)
	}
}

func TestDetect_DockerIsPodmanCompat(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prober := &mockProber{
		lookPathResults: map[string]string{
			"podman": "/usr/bin/podman",
			"docker": "/usr/bin/docker",
		},
		outputResults: map[string]outputResult{
			"podman version --format {{.Client.Version}}":      {output: []byte("4.9.3\n")},
			"podman info --format {{.Host.Security.Rootless}}": {output: []byte("true\n")},
			"docker --version": {output: []byte("podman version 4.9.3\n")},
		},
		env: map[string]string{
			"XDG_RUNTIME_DIR": "/run/user/1000",
		},
	}

	info, err := Detect(ctx, prober)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if !info.HasDockerCompat {
		t.Error("HasDockerCompat = false, want true")
	}
	// Docker should NOT appear as a separate runtime.
	for _, r := range info.Available {
		if r == RuntimeDocker {
			t.Error("Docker should not appear in Available when it is a Podman alias")
		}
	}
}

func TestDetect_NspawnOnly(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prober := &mockProber{
		lookPathResults: map[string]string{
			"systemd-nspawn": "/usr/bin/systemd-nspawn",
		},
		outputResults: map[string]outputResult{},
	}

	info, err := Detect(ctx, prober)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if info.Active != RuntimeNspawn {
		t.Errorf("Active = %v, want %v", info.Active, RuntimeNspawn)
	}
	if info.Path != "/usr/bin/systemd-nspawn" {
		t.Errorf("Path = %q, want %q", info.Path, "/usr/bin/systemd-nspawn")
	}
	if len(info.Available) != 1 || info.Available[0] != RuntimeNspawn {
		t.Errorf("Available = %v, want [nspawn]", info.Available)
	}
}

func TestDetect_NoneAvailable(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prober := &mockProber{
		lookPathResults: map[string]string{},
		outputResults:   map[string]outputResult{},
	}

	info, err := Detect(ctx, prober)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if info.Active != RuntimeNone {
		t.Errorf("Active = %v, want %v", info.Active, RuntimeNone)
	}
	if len(info.Available) != 0 {
		t.Errorf("Available = %v, want empty", info.Available)
	}
	if info.ComposeMethod != "none" {
		t.Errorf("ComposeMethod = %q, want %q", info.ComposeMethod, "none")
	}
}

func TestDetect_ComposeMethod_PodmanCompose(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prober := newPodmanRootlessProber()
	prober.lookPathResults["podman-compose"] = "/usr/bin/podman-compose"

	info, err := Detect(ctx, prober)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if info.ComposeMethod != "podman-compose" {
		t.Errorf("ComposeMethod = %q, want %q", info.ComposeMethod, "podman-compose")
	}
}

func TestDetect_ComposeMethod_DockerCompose(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prober := newDockerProber()
	prober.lookPathResults["docker-compose"] = "/usr/bin/docker-compose"

	info, err := Detect(ctx, prober)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if info.ComposeMethod != "docker-compose" {
		t.Errorf("ComposeMethod = %q, want %q", info.ComposeMethod, "docker-compose")
	}
}

func TestDetect_ComposeMethod_ViaSocket(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prober := newPodmanRootlessProber()
	prober.lookPathResults["docker-compose"] = "/usr/bin/docker-compose"

	info, err := Detect(ctx, prober)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if info.ComposeMethod != "docker-compose-via-socket" {
		t.Errorf("ComposeMethod = %q, want %q", info.ComposeMethod, "docker-compose-via-socket")
	}
}

func TestDetect_ComposeMethod_None(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prober := newDockerProber()
	// No compose binary available, and docker compose subcommand also fails.
	prober.outputResults["docker compose version"] = outputResult{err: fmt.Errorf("not available")}

	info, err := Detect(ctx, prober)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if info.ComposeMethod != "none" {
		t.Errorf("ComposeMethod = %q, want %q", info.ComposeMethod, "none")
	}
}

func TestDetect_ComposeMethod_DockerComposeV2Subcommand(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prober := newDockerProber()
	// docker compose v2 subcommand works.
	prober.outputResults["docker compose version"] = outputResult{output: []byte("Docker Compose version v2.24.5\n")}

	info, err := Detect(ctx, prober)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if info.ComposeMethod != "docker-compose" {
		t.Errorf("ComposeMethod = %q, want %q", info.ComposeMethod, "docker-compose")
	}
}

func TestDetect_PodmanNoXDGRuntime(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prober := &mockProber{
		lookPathResults: map[string]string{
			"podman": "/usr/bin/podman",
		},
		outputResults: map[string]outputResult{
			"podman version --format {{.Client.Version}}":      {output: []byte("4.9.3\n")},
			"podman info --format {{.Host.Security.Rootless}}": {output: []byte("true\n")},
		},
		// No XDG_RUNTIME_DIR set.
		env: map[string]string{},
	}

	info, err := Detect(ctx, prober)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if info.Active != RuntimePodmanRootless {
		t.Errorf("Active = %v, want %v", info.Active, RuntimePodmanRootless)
	}
	if info.SocketPath != "" {
		t.Errorf("SocketPath = %q, want empty when XDG_RUNTIME_DIR is unset", info.SocketPath)
	}
}
