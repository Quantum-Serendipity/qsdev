package sandbox

import (
	"context"
	"slices"
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

	// Landlock ABI version, as reported by the ll-restrict helper.
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

// probeLandlock reports the ENFORCEABLE Landlock ABI. Landlock is only
// enforceable when the ll-restrict helper is present: the bwrap backend wraps
// hook commands with it, and without the helper the kernel may support Landlock
// but the sandbox applies no filesystem restriction. Reporting bare kernel
// capability here would let DetermineTier advertise "full" isolation the tool
// cannot deliver (NF-3), so the reported ABI is gated on the helper.
//
// The ABI is whatever `ll-restrict --version` reports ("landlock-abi:N", from
// the kernel's own landlock_create_ruleset version query). There is no
// kernel-version fallback: a >= 5.13 kernel can still have Landlock disabled
// (e.g. missing from the boot lsm= list), and then every ll-restrict run fails,
// so a helper that cannot confirm an ABI means Landlock is not enforceable.
func probeLandlock(ctx context.Context, prober SandboxProber) int {
	llPath := prober.LandlockHelperPath()
	if llPath == "" {
		return 0
	}

	out, err := prober.Output(ctx, llPath, "--version")
	if err != nil {
		return 0
	}
	abiStr, ok := strings.CutPrefix(strings.TrimSpace(string(out)), "landlock-abi:")
	if !ok {
		return 0
	}
	v, err := strconv.Atoi(abiStr)
	if err != nil || v < 0 {
		return 0
	}
	return v
}

// probeSeccomp reports whether seccomp syscall filtering is ENFORCEABLE. It is
// only enforceable when the compiled BPF filter is present: the bwrap backend
// loads it via --seccomp, and without it no syscall filtering is applied even
// on a seccomp-capable kernel. Reporting bare kernel capability here would let
// DetermineTier advertise a layer the tool cannot apply (NF-3), so the reported
// support is gated on the filter's presence.
func probeSeccomp(prober SandboxProber) bool {
	if prober.SeccompFilterPath() == "" {
		return false
	}
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

// probeCgroupDelegation checks whether the current user's systemd service
// manager has the memory and pids controllers delegated to it. Those are what
// the transient `systemd-run --user --scope` units used for hook resource
// limits (MemoryMax=, TasksMax=) need; they live under user@UID.service, not in
// the root-owned user-UID.slice above it.
func probeCgroupDelegation(prober SandboxProber) bool {
	uid := processUID(prober)
	if uid == "" {
		return false
	}

	path := "/sys/fs/cgroup/user.slice/user-" + uid + ".slice/user@" + uid + ".service/cgroup.controllers"
	data, err := prober.ReadFile(path)
	if err != nil {
		return false
	}
	controllers := strings.Fields(string(data))
	return slices.Contains(controllers, "memory") && slices.Contains(controllers, "pids")
}

// processUID returns the real UID of the current process from
// /proc/self/status. It deliberately does not consult $UID, which is a shell
// variable any parent can set.
func processUID(prober SandboxProber) string {
	data, err := prober.ReadFile("/proc/self/status")
	if err != nil {
		return ""
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		if rest, ok := strings.CutPrefix(line, "Uid:"); ok {
			fields := strings.Fields(rest)
			if len(fields) == 0 {
				return ""
			}
			if _, err := strconv.ParseUint(fields[0], 10, 32); err != nil {
				return ""
			}
			return fields[0]
		}
	}
	return ""
}

// parseKernelVersion extracts the kernel version string from /proc/version.
func parseKernelVersion(procVersion string) string {
	fields := strings.Fields(procVersion)
	if len(fields) >= 3 {
		return fields[2]
	}
	return ""
}
