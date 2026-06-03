package sandbox

import (
	"context"
	"os"
	"testing"
)

type mockSandboxProber struct {
	lookPathResults map[string]string
	outputResults   map[string][]byte
	outputErrors    map[string]error
	files           map[string][]byte
	fileInfos       map[string]bool
	envVars         map[string]string
}

func newMockProber() *mockSandboxProber {
	return &mockSandboxProber{
		lookPathResults: make(map[string]string),
		outputResults:   make(map[string][]byte),
		outputErrors:    make(map[string]error),
		files:           make(map[string][]byte),
		fileInfos:       make(map[string]bool),
		envVars:         make(map[string]string),
	}
}

func (m *mockSandboxProber) LookPath(name string) (string, error) {
	if path, ok := m.lookPathResults[name]; ok {
		return path, nil
	}
	return "", &os.PathError{Op: "LookPath", Path: name, Err: os.ErrNotExist}
}

func (m *mockSandboxProber) Output(_ context.Context, name string, _ ...string) ([]byte, error) {
	if err, ok := m.outputErrors[name]; ok {
		return nil, err
	}
	if out, ok := m.outputResults[name]; ok {
		return out, nil
	}
	return nil, &os.PathError{Op: "exec", Path: name, Err: os.ErrNotExist}
}

func (m *mockSandboxProber) ReadFile(path string) ([]byte, error) {
	if data, ok := m.files[path]; ok {
		return data, nil
	}
	return nil, &os.PathError{Op: "ReadFile", Path: path, Err: os.ErrNotExist}
}

func (m *mockSandboxProber) Stat(path string) (os.FileInfo, error) {
	if m.fileInfos[path] {
		return nil, nil
	}
	return nil, &os.PathError{Op: "Stat", Path: path, Err: os.ErrNotExist}
}

func (m *mockSandboxProber) Getenv(key string) string {
	return m.envVars[key]
}

var _ SandboxProber = (*mockSandboxProber)(nil)

func TestProbeCapabilities_FullSupport(t *testing.T) {
	t.Parallel()
	mock := newMockProber()
	mock.lookPathResults["bwrap"] = "/usr/bin/bwrap"
	mock.lookPathResults["systemd-run"] = "/usr/bin/systemd-run"
	mock.files["/proc/sys/kernel/unprivileged_userns_clone"] = []byte("1\n")
	mock.files["/proc/sys/kernel/seccomp/actions_avail"] = []byte("kill_process kill_thread trap errno trace log allow user_notif\n")
	mock.files["/proc/version"] = []byte("Linux version 6.8.0-40-generic (buildd@x86-64) #40-Ubuntu\n")
	mock.fileInfos["/sys/fs/cgroup/cgroup.controllers"] = true
	mock.envVars["UID"] = "1000"
	mock.files["/sys/fs/cgroup/user.slice/user-1000.slice/cgroup.controllers"] = []byte("cpu memory pids\n")

	caps := ProbeCapabilities(context.Background(), mock)

	if !caps.HasBwrap {
		t.Error("expected HasBwrap = true")
	}
	if caps.BwrapPath != "/usr/bin/bwrap" {
		t.Errorf("BwrapPath = %q, want %q", caps.BwrapPath, "/usr/bin/bwrap")
	}
	if !caps.HasUserNS {
		t.Error("expected HasUserNS = true")
	}
	if !caps.HasSeccomp {
		t.Error("expected HasSeccomp = true")
	}
	if !caps.HasCgroupV2 {
		t.Error("expected HasCgroupV2 = true")
	}
	if !caps.HasCgroupDeleg {
		t.Error("expected HasCgroupDeleg = true")
	}
	if !caps.HasSystemdRun {
		t.Error("expected HasSystemdRun = true")
	}
	if caps.KernelVersion != "6.8.0-40-generic" {
		t.Errorf("KernelVersion = %q, want %q", caps.KernelVersion, "6.8.0-40-generic")
	}
}

func TestProbeCapabilities_NoBwrap(t *testing.T) {
	t.Parallel()
	mock := newMockProber()

	caps := ProbeCapabilities(context.Background(), mock)

	if caps.HasBwrap {
		t.Error("expected HasBwrap = false")
	}
	if caps.BwrapPath != "" {
		t.Errorf("BwrapPath = %q, want empty", caps.BwrapPath)
	}
}

func TestProbeCapabilities_UserNS_AppArmorRestricted(t *testing.T) {
	t.Parallel()
	mock := newMockProber()
	mock.files["/proc/sys/kernel/unprivileged_userns_clone"] = []byte("1\n")
	mock.files["/proc/sys/kernel/apparmor_restrict_unprivileged_userns"] = []byte("1\n")

	caps := ProbeCapabilities(context.Background(), mock)

	if caps.HasUserNS {
		t.Error("expected HasUserNS = false when AppArmor restricts userns")
	}
}

func TestProbeCapabilities_UserNS_Disabled(t *testing.T) {
	t.Parallel()
	mock := newMockProber()
	mock.files["/proc/sys/kernel/unprivileged_userns_clone"] = []byte("0\n")

	caps := ProbeCapabilities(context.Background(), mock)

	if caps.HasUserNS {
		t.Error("expected HasUserNS = false")
	}
}

func TestProbeCapabilities_UserNS_NixOS_Fallback(t *testing.T) {
	t.Parallel()
	mock := newMockProber()
	// NixOS doesn't have the sysctl, but has max_user_namespaces.
	mock.files["/proc/sys/user/max_user_namespaces"] = []byte("65536\n")

	caps := ProbeCapabilities(context.Background(), mock)

	if !caps.HasUserNS {
		t.Error("expected HasUserNS = true (NixOS fallback)")
	}
}

func TestProbeCapabilities_Seccomp_ViaStatus(t *testing.T) {
	t.Parallel()
	mock := newMockProber()
	mock.files["/proc/self/status"] = []byte("Name:\ttest\nSeccomp:\t2\nSeccomp_filters:\t1\n")

	caps := ProbeCapabilities(context.Background(), mock)

	if !caps.HasSeccomp {
		t.Error("expected HasSeccomp = true (via /proc/self/status)")
	}
}

func TestProbeCapabilities_CgroupDelegation_ViaUID(t *testing.T) {
	t.Parallel()
	mock := newMockProber()
	mock.files["/proc/self/status"] = []byte("Name:\ttest\nUid:\t1000\t1000\t1000\t1000\n")
	mock.files["/sys/fs/cgroup/user.slice/user-1000.slice/cgroup.controllers"] = []byte("cpu memory pids\n")

	caps := ProbeCapabilities(context.Background(), mock)

	if !caps.HasCgroupDeleg {
		t.Error("expected HasCgroupDeleg = true")
	}
}

func TestProbeCapabilities_LandlockViaKernelVersion(t *testing.T) {
	t.Parallel()
	mock := newMockProber()
	mock.files["/proc/version"] = []byte("Linux version 6.1.0-arch1 (builder@arch) #1 SMP\n")

	caps := ProbeCapabilities(context.Background(), mock)

	if caps.LandlockABI < 1 {
		t.Errorf("expected LandlockABI >= 1 for kernel 6.1, got %d", caps.LandlockABI)
	}
}

func TestProbeCapabilities_LandlockOldKernel(t *testing.T) {
	t.Parallel()
	mock := newMockProber()
	mock.files["/proc/version"] = []byte("Linux version 5.10.0-generic\n")

	caps := ProbeCapabilities(context.Background(), mock)

	if caps.LandlockABI != 0 {
		t.Errorf("expected LandlockABI = 0 for kernel 5.10, got %d", caps.LandlockABI)
	}
}

func TestParseKernelVersion(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{"Linux version 6.8.0-40-generic (buildd@x86) #40-Ubuntu", "6.8.0-40-generic"},
		{"Linux version 5.10.0", "5.10.0"},
		{"", ""},
		{"short", ""},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			if got := parseKernelVersion(tt.input); got != tt.want {
				t.Errorf("parseKernelVersion(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseKernelMajorMinor(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input     string
		wantMajor int
		wantMinor int
	}{
		{"6.8.0-40-generic", 6, 8},
		{"5.13.0", 5, 13},
		{"5.10", 5, 10},
		{"", 0, 0},
		{"abc", 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			major, minor := parseKernelMajorMinor(tt.input)
			if major != tt.wantMajor || minor != tt.wantMinor {
				t.Errorf("parseKernelMajorMinor(%q) = (%d, %d), want (%d, %d)",
					tt.input, major, minor, tt.wantMajor, tt.wantMinor)
			}
		})
	}
}
