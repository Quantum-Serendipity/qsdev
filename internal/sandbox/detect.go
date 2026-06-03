package sandbox

import (
	"context"
	"strconv"
	"strings"
)

// ProbeCapabilities detects available sandbox features on the current system.
func ProbeCapabilities(ctx context.Context, prober SandboxProber) *SystemCapabilities {
	caps := &SystemCapabilities{}

	// Bubblewrap binary.
	if path, err := prober.LookPath("bwrap"); err == nil {
		caps.HasBwrap = true
		caps.BwrapPath = path
	}

	// Unprivileged user namespaces.
	caps.HasUserNS = probeUserNamespaces(prober)

	// Landlock ABI version (kernel >= 5.13).
	caps.LandlockABI = probeLandlock(ctx, prober)

	// Seccomp support.
	caps.HasSeccomp = probeSeccomp(prober)

	// Cgroups v2 unified hierarchy.
	caps.HasCgroupV2 = probeCgroupV2(prober)

	// Cgroup delegation for the current user.
	caps.HasCgroupDeleg = probeCgroupDelegation(prober)

	// systemd-run.
	if path, err := prober.LookPath("systemd-run"); err == nil {
		caps.HasSystemdRun = true
		caps.SystemdRunPath = path
	}

	// Kernel version.
	if data, err := prober.ReadFile("/proc/version"); err == nil {
		caps.KernelVersion = parseKernelVersion(string(data))
	}

	return caps
}

// ProbeCapabilitiesDefault runs ProbeCapabilities with real system calls.
func ProbeCapabilitiesDefault(ctx context.Context) *SystemCapabilities {
	return ProbeCapabilities(ctx, &ExecSandboxProber{})
}

// probeUserNamespaces checks whether unprivileged user namespaces are available.
func probeUserNamespaces(prober SandboxProber) bool {
	// Check the sysctl knob (most distros).
	if data, err := prober.ReadFile("/proc/sys/kernel/unprivileged_userns_clone"); err == nil {
		val := strings.TrimSpace(string(data))
		if val == "1" {
			// Also check AppArmor restriction (Ubuntu 24.04+).
			if data, err := prober.ReadFile("/proc/sys/kernel/apparmor_restrict_unprivileged_userns"); err == nil {
				if strings.TrimSpace(string(data)) == "1" {
					return false
				}
			}
			return true
		}
		return false
	}

	// Sysctl doesn't exist (e.g., NixOS where it defaults to enabled).
	// Check /proc/sys/user/max_user_namespaces instead.
	if data, err := prober.ReadFile("/proc/sys/user/max_user_namespaces"); err == nil {
		if n, parseErr := strconv.Atoi(strings.TrimSpace(string(data))); parseErr == nil && n > 0 {
			return true
		}
	}

	return false
}

// probeLandlock checks Landlock ABI availability by examining kernel version
// and looking for the ll-restrict helper binary.
func probeLandlock(ctx context.Context, prober SandboxProber) int {
	// If ll-restrict binary is available, try probing its version output.
	if llPath := LLRestrictBin(); llPath != "" {
		if out, err := prober.Output(ctx, llPath, "--version"); err == nil {
			version := strings.TrimSpace(string(out))
			if abiStr, ok := strings.CutPrefix(version, "landlock-abi:"); ok {
				if v, parseErr := strconv.Atoi(abiStr); parseErr == nil {
					return v
				}
			}
		}
		// ll-restrict exists but can't determine ABI — assume at least v1.
		return 1
	}

	// Fall back to kernel version heuristic.
	if data, err := prober.ReadFile("/proc/version"); err == nil {
		kver := parseKernelVersion(string(data))
		major, minor := parseKernelMajorMinor(kver)
		if major > 5 || (major == 5 && minor >= 13) {
			return 1 // Landlock v1 available from 5.13
		}
	}

	return 0
}

// probeSeccomp checks whether seccomp is available.
func probeSeccomp(prober SandboxProber) bool {
	if data, err := prober.ReadFile("/proc/sys/kernel/seccomp/actions_avail"); err == nil {
		return strings.Contains(string(data), "errno")
	}
	// Fallback: check /proc/self/status for Seccomp field.
	if data, err := prober.ReadFile("/proc/self/status"); err == nil {
		for line := range strings.SplitSeq(string(data), "\n") {
			if strings.HasPrefix(line, "Seccomp:") {
				return true
			}
		}
	}
	return false
}

// probeCgroupV2 checks for the cgroup v2 unified hierarchy.
func probeCgroupV2(prober SandboxProber) bool {
	_, err := prober.Stat("/sys/fs/cgroup/cgroup.controllers")
	return err == nil
}

// probeCgroupDelegation checks whether the current user has cgroup delegation.
func probeCgroupDelegation(prober SandboxProber) bool {
	// On systems with systemd, check user slice delegation.
	uid := prober.Getenv("UID")
	if uid == "" {
		// Try reading from /proc/self/status.
		if data, err := prober.ReadFile("/proc/self/status"); err == nil {
			for line := range strings.SplitSeq(string(data), "\n") {
				if strings.HasPrefix(line, "Uid:") {
					fields := strings.Fields(line)
					if len(fields) >= 2 {
						uid = fields[1]
					}
					break
				}
			}
		}
	}
	if uid == "" {
		return false
	}

	path := "/sys/fs/cgroup/user.slice/user-" + uid + ".slice/cgroup.controllers"
	if data, err := prober.ReadFile(path); err == nil {
		return len(strings.TrimSpace(string(data))) > 0
	}
	return false
}

// parseKernelVersion extracts the kernel version string from /proc/version.
func parseKernelVersion(procVersion string) string {
	fields := strings.Fields(procVersion)
	if len(fields) >= 3 {
		return fields[2]
	}
	return ""
}

// parseKernelMajorMinor extracts major.minor from a kernel version string.
func parseKernelMajorMinor(kver string) (int, int) {
	parts := strings.SplitN(kver, ".", 3)
	if len(parts) < 2 {
		return 0, 0
	}
	major, _ := strconv.Atoi(parts[0])
	minor, _ := strconv.Atoi(parts[1])
	return major, minor
}
