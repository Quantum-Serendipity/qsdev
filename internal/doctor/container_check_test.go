package doctor

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/sysinfo"
)

// doctorMockProber implements container.Prober for doctor-level tests.
// This is a local copy because test types cannot be imported cross-package.
type doctorMockProber struct {
	lookPathResults map[string]string
	outputResults   map[string]mockOutputResult
	files           map[string][]byte
	fileInfos       map[string]bool
	globResults     map[string][]string
	user            string
	env             map[string]string
}

type mockOutputResult struct {
	output []byte
	err    error
}

func (m *doctorMockProber) LookPath(name string) (string, error) {
	if path, ok := m.lookPathResults[name]; ok {
		return path, nil
	}
	return "", fmt.Errorf("not found: %s", name)
}

func (m *doctorMockProber) Output(_ context.Context, name string, args ...string) ([]byte, error) {
	key := name
	if len(args) > 0 {
		key = name + " " + strings.Join(args, " ")
	}
	if r, ok := m.outputResults[key]; ok {
		return r.output, r.err
	}
	return nil, fmt.Errorf("no mock output for: %s", key)
}

func (m *doctorMockProber) ReadFile(path string) ([]byte, error) {
	if data, ok := m.files[path]; ok {
		return data, nil
	}
	return nil, os.ErrNotExist
}

func (m *doctorMockProber) Stat(path string) (os.FileInfo, error) {
	if m.fileInfos != nil {
		if exists, ok := m.fileInfos[path]; ok && exists {
			return &doctorMockFileInfo{}, nil
		}
	}
	return nil, os.ErrNotExist
}

func (m *doctorMockProber) Glob(pattern string) ([]string, error) {
	if matches, ok := m.globResults[pattern]; ok {
		return matches, nil
	}
	return nil, nil
}

func (m *doctorMockProber) CurrentUser() string { return m.user }

func (m *doctorMockProber) Getenv(key string) string {
	if m.env != nil {
		return m.env[key]
	}
	return ""
}

type doctorMockFileInfo struct{}

func (f *doctorMockFileInfo) Name() string       { return "mock" }
func (f *doctorMockFileInfo) Size() int64        { return 0 }
func (f *doctorMockFileInfo) Mode() os.FileMode  { return 0o644 }
func (f *doctorMockFileInfo) ModTime() time.Time { return time.Time{} }
func (f *doctorMockFileInfo) IsDir() bool        { return false }
func (f *doctorMockFileInfo) Sys() any           { return nil }

// newPodmanRootlessCleanProber returns a mock with Podman rootless, cgroupsv2,
// subuid configured, no GPU, no NFS.
func newPodmanRootlessCleanProber() *doctorMockProber {
	return &doctorMockProber{
		lookPathResults: map[string]string{
			"podman":         "/usr/bin/podman",
			"podman-compose": "/usr/bin/podman-compose",
		},
		outputResults: map[string]mockOutputResult{
			"podman version --format {{.Client.Version}}":      {output: []byte("5.2.0\n")},
			"podman info --format {{.Host.Security.Rootless}}": {output: []byte("true\n")},
		},
		files: map[string][]byte{
			"/etc/subuid":  []byte("testuser:100000:65536\n"),
			"/proc/mounts": []byte("tmpfs /tmp tmpfs rw 0 0\n"),
		},
		fileInfos: map[string]bool{
			"/sys/fs/cgroup/cgroup.controllers": true,
		},
		globResults: map[string][]string{},
		user:        "testuser",
		env: map[string]string{
			"XDG_RUNTIME_DIR": "/run/user/1000",
		},
	}
}

func defaultOSInfo() *sysinfo.OSInfo {
	return &sysinfo.OSInfo{
		OS:     "linux",
		Arch:   "amd64",
		Family: "linux",
	}
}

func nixosOSInfo() *sysinfo.OSInfo {
	return &sysinfo.OSInfo{
		OS:     "linux",
		Arch:   "amd64",
		Family: "nixos",
		Distro: "NixOS",
	}
}

func findItem(cs *ContainerSection, label string) *ContainerCheckItem {
	for i := range cs.Items {
		if cs.Items[i].Label == label {
			return &cs.Items[i]
		}
	}
	return nil
}

func TestRunContainerCheck_NoRuntime(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prober := &doctorMockProber{
		lookPathResults: map[string]string{},
		outputResults:   map[string]mockOutputResult{},
	}

	cs := RunContainerCheck(ctx, prober, defaultOSInfo())
	if cs != nil {
		t.Errorf("expected nil ContainerSection when no runtime detected, got %+v", cs)
	}
}

func TestRunContainerCheck_PodmanRootlessClean(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prober := newPodmanRootlessCleanProber()

	cs := RunContainerCheck(ctx, prober, defaultOSInfo())
	if cs == nil {
		t.Fatal("expected non-nil ContainerSection")
	}
	if !cs.Detected {
		t.Error("Detected = false, want true")
	}
	if cs.RuntimeName != "podman-rootless" {
		t.Errorf("RuntimeName = %q, want %q", cs.RuntimeName, "podman-rootless")
	}
	if !cs.Rootless {
		t.Error("Rootless = false, want true")
	}
	if cs.Runtime != "Podman 5.2.0 (rootless)" {
		t.Errorf("Runtime = %q, want %q", cs.Runtime, "Podman 5.2.0 (rootless)")
	}

	// All items should be "ok".
	for _, item := range cs.Items {
		if item.Status != "ok" {
			t.Errorf("item %q status = %q, want %q", item.Label, item.Status, "ok")
		}
	}
	if len(cs.Warnings) != 0 {
		t.Errorf("expected no warnings, got %v", cs.Warnings)
	}
}

func TestRunContainerCheck_PodmanRootlessGPU(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prober := newPodmanRootlessCleanProber()
	prober.globResults["/dev/nvidia*"] = []string{"/dev/nvidia0", "/dev/nvidiactl"}

	cs := RunContainerCheck(ctx, prober, defaultOSInfo())
	if cs == nil {
		t.Fatal("expected non-nil ContainerSection")
	}

	gpu := findItem(cs, "GPU")
	if gpu == nil {
		t.Fatal("missing GPU item")
	}
	if gpu.Status != "warn" {
		t.Errorf("GPU status = %q, want %q", gpu.Status, "warn")
	}
	if !strings.Contains(gpu.Summary, "rootless cannot pass through") {
		t.Errorf("GPU summary = %q, expected rootless warning", gpu.Summary)
	}
	if len(cs.Warnings) == 0 {
		t.Error("expected fallback warnings for GPU")
	}
}

func TestRunContainerCheck_PodmanRootlessNFS(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prober := newPodmanRootlessCleanProber()
	prober.files["/proc/mounts"] = []byte("server:/export /mnt/nfs nfs4 rw,relatime 0 0\n")

	cs := RunContainerCheck(ctx, prober, defaultOSInfo())
	if cs == nil {
		t.Fatal("expected non-nil ContainerSection")
	}

	nfs := findItem(cs, "NFS")
	if nfs == nil {
		t.Fatal("missing NFS item")
	}
	if nfs.Status != "warn" {
		t.Errorf("NFS status = %q, want %q", nfs.Status, "warn")
	}
	if !strings.Contains(nfs.Summary, "rootless cannot bind-mount") {
		t.Errorf("NFS summary = %q, expected rootless warning", nfs.Summary)
	}
}

func TestRunContainerCheck_PodmanRootlessBothGPUNFS(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prober := newPodmanRootlessCleanProber()
	prober.globResults["/dev/nvidia*"] = []string{"/dev/nvidia0"}
	prober.files["/proc/mounts"] = []byte("server:/vol /data nfs rw 0 0\n")

	cs := RunContainerCheck(ctx, prober, defaultOSInfo())
	if cs == nil {
		t.Fatal("expected non-nil ContainerSection")
	}

	gpu := findItem(cs, "GPU")
	if gpu == nil || gpu.Status != "warn" {
		t.Errorf("expected GPU warn, got %+v", gpu)
	}
	nfs := findItem(cs, "NFS")
	if nfs == nil || nfs.Status != "warn" {
		t.Errorf("expected NFS warn, got %+v", nfs)
	}
	if len(cs.Warnings) < 2 {
		t.Errorf("expected at least 2 warnings (GPU+NFS), got %d: %v", len(cs.Warnings), cs.Warnings)
	}
}

func TestRunContainerCheck_PodmanRootlessNoSubuid(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prober := newPodmanRootlessCleanProber()
	prober.files["/etc/subuid"] = []byte("otheruser:100000:65536\n")

	cs := RunContainerCheck(ctx, prober, defaultOSInfo())
	if cs == nil {
		t.Fatal("expected non-nil ContainerSection")
	}

	userns := findItem(cs, "User NS")
	if userns == nil {
		t.Fatal("missing User NS item")
	}
	if userns.Status != "error" {
		t.Errorf("User NS status = %q, want %q", userns.Status, "error")
	}
	if userns.Summary != "not configured" {
		t.Errorf("User NS summary = %q, want %q", userns.Summary, "not configured")
	}
	if !strings.Contains(userns.Detail, "sudo usermod") {
		t.Errorf("User NS detail = %q, expected usermod remediation", userns.Detail)
	}
}

func TestRunContainerCheck_PodmanRootlessNoSubuidNixOS(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prober := newPodmanRootlessCleanProber()
	prober.files["/etc/subuid"] = []byte("otheruser:100000:65536\n")

	cs := RunContainerCheck(ctx, prober, nixosOSInfo())
	if cs == nil {
		t.Fatal("expected non-nil ContainerSection")
	}

	userns := findItem(cs, "User NS")
	if userns == nil {
		t.Fatal("missing User NS item")
	}
	if userns.Status != "error" {
		t.Errorf("User NS status = %q, want %q", userns.Status, "error")
	}
	if !strings.Contains(userns.Detail, "subUidRanges") {
		t.Errorf("User NS detail = %q, expected NixOS-specific remediation", userns.Detail)
	}
	if !strings.Contains(userns.Detail, "docs/nixos-podman-rootless.md") {
		t.Errorf("User NS detail = %q, expected doc link", userns.Detail)
	}
}

func TestRunContainerCheck_DockerRuntime(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prober := &doctorMockProber{
		lookPathResults: map[string]string{
			"docker": "/usr/bin/docker",
		},
		outputResults: map[string]mockOutputResult{
			"docker --version": {output: []byte("Docker version 24.0.7, build afdd53b\n")},
		},
		files: map[string][]byte{
			"/proc/mounts": []byte("tmpfs /tmp tmpfs rw 0 0\n"),
		},
		fileInfos: map[string]bool{
			"/sys/fs/cgroup/cgroup.controllers": true,
		},
		globResults: map[string][]string{},
		user:        "testuser",
		env:         map[string]string{},
	}

	cs := RunContainerCheck(ctx, prober, defaultOSInfo())
	if cs == nil {
		t.Fatal("expected non-nil ContainerSection")
	}

	if cs.RuntimeName != "docker" {
		t.Errorf("RuntimeName = %q, want %q", cs.RuntimeName, "docker")
	}
	if cs.Rootless {
		t.Error("Rootless = true, want false for Docker")
	}

	// Docker should get migration recommendation.
	found := false
	for _, rec := range cs.Recommendations {
		if strings.Contains(rec, "Podman rootless") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected migration recommendation, got %v", cs.Recommendations)
	}
}

func TestRunContainerCheck_PodmanRootful(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prober := &doctorMockProber{
		lookPathResults: map[string]string{
			"podman": "/usr/bin/podman",
		},
		outputResults: map[string]mockOutputResult{
			"podman version --format {{.Client.Version}}":      {output: []byte("5.2.0\n")},
			"podman info --format {{.Host.Security.Rootless}}": {output: []byte("false\n")},
		},
		files: map[string][]byte{
			"/proc/mounts": []byte("server:/vol /data nfs rw 0 0\n"),
		},
		fileInfos: map[string]bool{
			"/sys/fs/cgroup/cgroup.controllers": true,
		},
		globResults: map[string][]string{
			"/dev/nvidia*": {"/dev/nvidia0"},
		},
		user: "testuser",
		env:  map[string]string{},
	}

	cs := RunContainerCheck(ctx, prober, defaultOSInfo())
	if cs == nil {
		t.Fatal("expected non-nil ContainerSection")
	}
	if cs.Rootless {
		t.Error("Rootless = true, want false")
	}
	if cs.RuntimeName != "podman-rootful" {
		t.Errorf("RuntimeName = %q, want %q", cs.RuntimeName, "podman-rootful")
	}
	if !strings.Contains(cs.Runtime, "rootful") {
		t.Errorf("Runtime = %q, expected 'rootful' in label", cs.Runtime)
	}

	// Rootful should not have rootless-specific warnings for GPU/NFS.
	gpu := findItem(cs, "GPU")
	if gpu == nil {
		t.Fatal("missing GPU item")
	}
	if gpu.Status != "ok" {
		t.Errorf("GPU status = %q, want %q for rootful", gpu.Status, "ok")
	}
	nfs := findItem(cs, "NFS")
	if nfs == nil {
		t.Fatal("missing NFS item")
	}
	if nfs.Status != "ok" {
		t.Errorf("NFS status = %q, want %q for rootful", nfs.Status, "ok")
	}
	if len(cs.Warnings) != 0 {
		t.Errorf("expected no warnings for rootful, got %v", cs.Warnings)
	}
}

func TestRunContainerCheck_CgroupsV1(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prober := newPodmanRootlessCleanProber()
	// Remove cgroupsv2 indicator.
	prober.fileInfos = map[string]bool{}

	cs := RunContainerCheck(ctx, prober, defaultOSInfo())
	if cs == nil {
		t.Fatal("expected non-nil ContainerSection")
	}

	cgroups := findItem(cs, "Cgroups")
	if cgroups == nil {
		t.Fatal("missing Cgroups item")
	}
	if cgroups.Status != "warn" {
		t.Errorf("Cgroups status = %q, want %q", cgroups.Status, "warn")
	}
	if !strings.Contains(cgroups.Summary, "v1") {
		t.Errorf("Cgroups summary = %q, expected v1 mention", cgroups.Summary)
	}
}
