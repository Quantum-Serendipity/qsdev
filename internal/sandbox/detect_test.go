package sandbox

import (
	"context"
	"errors"
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
	landlockHelper  string // path returned by LandlockHelperPath ("" = unavailable)
	seccompFilter   string // path returned by SeccompFilterPath ("" = unavailable)
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

func (m *mockSandboxProber) LandlockHelperPath() string { return m.landlockHelper }

func (m *mockSandboxProber) SeccompFilterPath() string { return m.seccompFilter }

var _ SandboxProber = (*mockSandboxProber)(nil)

func TestProbeCapabilities_FullSupport(t *testing.T) {
	t.Parallel()
	mock := newMockProber()
	mock.lookPathResults["bwrap"] = "/usr/bin/bwrap"
	mock.lookPathResults["systemd-run"] = "/usr/bin/systemd-run"
	// Enforcement artifacts present: full support requires the tools that APPLY
	// Landlock and seccomp, not just a capable kernel.
	mock.landlockHelper = "/usr/bin/ll-restrict"
	mock.seccompFilter = "/nix/store/seccomp.bpf"
	mock.files["/proc/sys/kernel/unprivileged_userns_clone"] = []byte("1\n")
	mock.files["/proc/sys/kernel/seccomp/actions_avail"] = []byte("kill_process kill_thread trap errno trace log allow user_notif\n")
	mock.files["/proc/version"] = []byte("Linux version 6.8.0-40-generic (buildd@x86-64) #40-Ubuntu\n")
	mock.fileInfos["/sys/fs/cgroup/cgroup.controllers"] = true
	mock.files["/proc/self/status"] = []byte("Name:\ttest\nUid:\t1000\t1000\t1000\t1000\n")
	mock.files[delegatedControllersPath("1000")] = []byte("cpu memory pids\n")

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
	mock.seccompFilter = "/nix/store/seccomp.bpf"
	mock.files["/proc/self/status"] = []byte("Name:\ttest\nSeccomp:\t2\nSeccomp_filters:\t1\n")

	caps := ProbeCapabilities(context.Background(), mock)

	if !caps.HasSeccomp {
		t.Error("expected HasSeccomp = true (via /proc/self/status)")
	}
}

// TestProbeCapabilities_SeccompNotEnforceableWithoutFilter is a regression for
// NF-3: a seccomp-capable kernel does NOT make seccomp enforceable when the
// compiled BPF filter (that bwrap loads via --seccomp) is absent. Reporting
// true here would let DetermineTier advertise a syscall-filtering layer the
// tool cannot apply.
func TestProbeCapabilities_SeccompNotEnforceableWithoutFilter(t *testing.T) {
	t.Parallel()
	mock := newMockProber()
	// Kernel advertises seccomp with errno action, but no filter is provisioned.
	mock.files["/proc/sys/kernel/seccomp/actions_avail"] = []byte("kill errno trace\n")
	mock.files["/proc/self/status"] = []byte("Name:\ttest\nSeccomp:\t2\n")

	caps := ProbeCapabilities(context.Background(), mock)

	if caps.HasSeccomp {
		t.Error("expected HasSeccomp = false: seccomp-capable kernel but no BPF filter to enforce with")
	}
}

// delegatedControllersPath is where probeCgroupDelegation looks for the
// controllers delegated to the user's systemd service manager.
func delegatedControllersPath(uid string) string {
	return "/sys/fs/cgroup/user.slice/user-" + uid + ".slice/user@" + uid + ".service/cgroup.controllers"
}

func TestProbeCapabilities_CgroupDelegation(t *testing.T) {
	t.Parallel()

	const status1000 = "Name:\ttest\nUid:\t1000\t1000\t1000\t1000\n"
	tests := []struct {
		name   string
		status string
		env    map[string]string
		files  map[string]string
		want   bool
	}{
		{
			name:   "memory and pids delegated to user manager",
			status: status1000,
			files:  map[string]string{delegatedControllersPath("1000"): "cpu memory pids\n"},
			want:   true,
		},
		{
			// The root-owned user slice is not where --user scopes are created.
			name:   "controllers only on the user slice",
			status: status1000,
			files: map[string]string{
				"/sys/fs/cgroup/user.slice/user-1000.slice/cgroup.controllers": "cpu memory pids\n",
			},
			want: false,
		},
		{
			name:   "memory not delegated",
			status: status1000,
			files:  map[string]string{delegatedControllersPath("1000"): "cpu pids\n"},
			want:   false,
		},
		{
			// $UID is a shell variable a parent can set; only the kernel's
			// view of the process UID counts.
			name:   "spoofed UID env var ignored",
			status: status1000,
			env:    map[string]string{"UID": "0"},
			files:  map[string]string{delegatedControllersPath("0"): "memory pids\n"},
			want:   false,
		},
		{
			name:  "no proc status",
			files: map[string]string{delegatedControllersPath("1000"): "memory pids\n"},
			want:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mock := newMockProber()
			if tt.status != "" {
				mock.files["/proc/self/status"] = []byte(tt.status)
			}
			for k, v := range tt.env {
				mock.envVars[k] = v
			}
			for path, content := range tt.files {
				mock.files[path] = []byte(content)
			}

			caps := ProbeCapabilities(context.Background(), mock)

			if caps.HasCgroupDeleg != tt.want {
				t.Errorf("HasCgroupDeleg = %v, want %v", caps.HasCgroupDeleg, tt.want)
			}
		})
	}
}

// TestProbeCapabilities_LandlockABI pins that the reported ABI comes only from
// the helper's `--version` self-report. A modern kernel version must not be
// taken as proof: Landlock can be compiled in yet absent from the boot lsm=
// list, where the helper reports 0 and every ll-restrict run would fail.
func TestProbeCapabilities_LandlockABI(t *testing.T) {
	t.Parallel()

	const helper = "/usr/bin/ll-restrict"
	tests := []struct {
		name      string
		output    string
		outputErr error
		want      int
	}{
		{name: "helper reports ABI", output: "landlock-abi:3\n", want: 3},
		{name: "helper reports Landlock disabled", output: "landlock-abi:0\n", want: 0},
		{name: "helper without --version support", outputErr: errors.New("exit status 120"), want: 0},
		{name: "unparseable self-report", output: "ll-restrict 0.1.0\n", want: 0},
		{name: "negative ABI", output: "landlock-abi:-1\n", want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mock := newMockProber()
			mock.landlockHelper = helper
			// A Landlock-capable kernel version must not influence the result.
			mock.files["/proc/version"] = []byte("Linux version 6.8.0-40-generic\n")
			if tt.outputErr != nil {
				mock.outputErrors[helper] = tt.outputErr
			} else {
				mock.outputResults[helper] = []byte(tt.output)
			}

			caps := ProbeCapabilities(context.Background(), mock)

			if caps.LandlockABI != tt.want {
				t.Errorf("LandlockABI = %d, want %d", caps.LandlockABI, tt.want)
			}
		})
	}
}

// TestProbeCapabilities_LandlockNotEnforceableWithoutHelper is a regression for
// NF-3: a Landlock-capable kernel does NOT make Landlock enforceable when the
// ll-restrict helper is absent. Reporting a non-zero ABI here would let
// DetermineTier advertise filesystem isolation the tool cannot apply.
func TestProbeCapabilities_LandlockNotEnforceableWithoutHelper(t *testing.T) {
	t.Parallel()
	mock := newMockProber()
	// Modern kernel supports Landlock, but no ll-restrict helper is provisioned.
	mock.files["/proc/version"] = []byte("Linux version 6.8.0-40-generic\n")

	caps := ProbeCapabilities(context.Background(), mock)

	if caps.LandlockABI != 0 {
		t.Errorf("expected LandlockABI = 0 without ll-restrict helper, got %d", caps.LandlockABI)
	}
}

// TestProbeCapabilities_EffectiveTierHonestWithoutEnforcementTools is a
// regression for NF-3: on a host with bwrap + userns and a fully Landlock/
// seccomp-capable kernel, but WITHOUT the ll-restrict helper and seccomp filter,
// DetermineTier must not report TierFull. The effective tier degrades because
// the LSM layers cannot actually be applied.
func TestProbeCapabilities_EffectiveTierHonestWithoutEnforcementTools(t *testing.T) {
	t.Parallel()
	mock := newMockProber()
	mock.lookPathResults["bwrap"] = "/usr/bin/bwrap"
	mock.files["/proc/sys/kernel/unprivileged_userns_clone"] = []byte("1\n")
	mock.files["/proc/sys/kernel/seccomp/actions_avail"] = []byte("kill errno\n")
	mock.files["/proc/version"] = []byte("Linux version 6.8.0-40-generic\n")

	caps := ProbeCapabilities(context.Background(), mock)

	if got := DetermineTier(caps); got == TierFull {
		t.Errorf("DetermineTier = %v; must not report full when ll-restrict and seccomp filter are absent", got)
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
