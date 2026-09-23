package container

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// Detect probes the system for available container runtimes and returns
// information about the preferred one. Podman rootless is preferred when
// available, falling back to Docker, then nspawn.
func Detect(ctx context.Context, prober Prober) (*RuntimeInfo, error) {
	info := &RuntimeInfo{
		Active: RuntimeNone,
	}

	// Check Podman first (preferred).
	if path, err := prober.LookPath("podman"); err == nil {
		detectPodman(ctx, prober, info, path)
	}

	// Check Docker.
	if path, err := prober.LookPath("docker"); err == nil {
		// Determine if docker is a Podman compatibility alias.
		if out, err := prober.Output(ctx, "docker", "--version"); err == nil {
			output := strings.ToLower(string(out))
			if strings.Contains(output, "podman") {
				info.HasDockerCompat = true
			} else {
				info.Available = append(info.Available, RuntimeDocker)
				if info.Active == RuntimeNone {
					info.Active = RuntimeDocker
					info.Path = path
					info.SocketPath = "/var/run/docker.sock"
					// Parse Docker version.
					parts := strings.Fields(strings.TrimSpace(string(out)))
					for i, p := range parts {
						if p == "version" && i+1 < len(parts) {
							info.Version = strings.TrimRight(parts[i+1], ",")
							break
						}
					}
				}
			}
		}
	}

	// Check systemd-nspawn.
	if path, err := prober.LookPath("systemd-nspawn"); err == nil {
		info.Available = append(info.Available, RuntimeNspawn)
		if info.Active == RuntimeNone {
			info.Active = RuntimeNspawn
			info.Path = path
		}
	}

	// Detect compose method.
	info.ComposeMethod = detectComposeMethod(ctx, prober, info)

	return info, nil
}

// detectPodman records the Podman installation at path in info. Its mode comes
// from `podman info`; when that fails (no subuid entries, broken storage, ...)
// the mode is unknown and Podman cannot run containers, so it is neither
// claimed rootless nor selected, and the failure is kept as a warning.
func detectPodman(ctx context.Context, prober Prober, info *RuntimeInfo, path string) {
	out, err := prober.Output(ctx, "podman", "info", "--format", "{{.Host.Security.Rootless}}")
	if err != nil {
		info.Warnings = append(info.Warnings, fmt.Sprintf(
			"podman found at %s but `podman info` failed (%v); it is unusable until fixed "+
				"(check /etc/subuid, /etc/subgid and the storage configuration)", path, err))
		return
	}

	info.Path = path
	if ver, verErr := prober.Output(ctx, "podman", "version", "--format", "{{.Client.Version}}"); verErr == nil {
		info.Version = strings.TrimSpace(string(ver))
	}

	info.Rootless = strings.TrimSpace(string(out)) == "true"
	if info.Rootless {
		info.Active = RuntimePodmanRootless
		if xdg := prober.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
			info.SocketPath = xdg + "/podman/podman.sock"
		}
	} else {
		info.Active = RuntimePodmanRootful
		info.SocketPath = "/run/podman/podman.sock"
	}
	info.Available = append(info.Available, info.Active)
}

// DetectDefault runs Detect with the default ExecProber.
func DetectDefault(ctx context.Context) (*RuntimeInfo, error) {
	return Detect(ctx, &ExecProber{})
}

func detectComposeMethod(ctx context.Context, prober Prober, info *RuntimeInfo) string {
	// Check podman-compose first.
	if _, err := prober.LookPath("podman-compose"); err == nil && info.Active.IsPodman() {
		return "podman-compose"
	}

	// Check docker-compose (Go binary or v2 plugin).
	if _, err := prober.LookPath("docker-compose"); err == nil {
		if info.Active.IsPodman() {
			return "docker-compose-via-socket"
		}
		return "docker-compose"
	}

	// Check docker compose (v2 subcommand).
	if _, err := prober.Output(ctx, "docker", "compose", "version"); err == nil {
		if info.Active.IsPodman() {
			return "docker-compose-via-socket"
		}
		return "docker-compose"
	}

	return "none"
}

// DetectCapabilities probes the system for container runtime capabilities
// that affect whether rootless mode is sufficient for the project rooted at
// projectRoot. Only NFS mounts that overlap projectRoot count; with an empty
// projectRoot no NFS mount is attributed to the project.
func DetectCapabilities(ctx context.Context, prober Prober, info *RuntimeInfo, projectRoot string) (*Capabilities, error) {
	if info == nil {
		return nil, fmt.Errorf("detecting capabilities: nil RuntimeInfo")
	}
	caps := &Capabilities{}

	caps.GPUPassthrough = hasGPUDevices(prober)

	// NFS detection: parse /proc/mounts for nfs/nfs4 entries that overlap
	// the project tree.
	if data, err := prober.ReadFile("/proc/mounts"); err == nil && projectRoot != "" {
		caps.NFSMounts = nfsMountOverlaps(string(data), projectRoot)
	}

	// subuid check: verify current user has user namespace mapping.
	username := prober.CurrentUser()
	if username != "" {
		if data, err := prober.ReadFile("/etc/subuid"); err == nil {
			for line := range strings.SplitSeq(string(data), "\n") {
				if strings.HasPrefix(line, username+":") {
					caps.UserNamespaceConfigured = true
					break
				}
			}
		}
	}

	// cgroups v2 detection.
	if _, err := prober.Stat("/sys/fs/cgroup/cgroup.controllers"); err == nil {
		caps.CgroupsV2 = true
	}

	// Privileged port threshold: true when rootless can bind traditionally
	// privileged ports (ip_unprivileged_port_start < 1024).
	if data, err := prober.ReadFile("/proc/sys/net/ipv4/ip_unprivileged_port_start"); err == nil {
		threshold := strings.TrimSpace(string(data))
		if n, parseErr := strconv.Atoi(threshold); parseErr == nil {
			caps.PrivilegedPorts = n < 1024
		}
	}

	// Rootless support.
	caps.RootlessSupported = info.Active.IsPodman() && info.Rootless

	return caps, nil
}

// hasGPUDevices reports whether GPU device nodes that rootless containers
// cannot pass through are present: NVIDIA (/dev/nvidia*) or AMD ROCm
// (/dev/kfd).
func hasGPUDevices(prober Prober) bool {
	if matches, err := prober.Glob("/dev/nvidia*"); err == nil && len(matches) > 0 {
		return true
	}
	_, err := prober.Stat("/dev/kfd")
	return err == nil
}

// nfsMountOverlaps reports whether any nfs/nfs4 entry of the /proc/mounts
// content overlaps projectRoot: a mount at or above it (the project lives on
// NFS) or below it (part of the project tree is NFS).
func nfsMountOverlaps(procMounts, projectRoot string) bool {
	root := filepath.Clean(projectRoot)
	for line := range strings.SplitSeq(procMounts, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 || (fields[2] != "nfs" && fields[2] != "nfs4") {
			continue
		}
		mountPoint := filepath.Clean(unescapeMountField(fields[1]))
		if pathContains(mountPoint, root) || pathContains(root, mountPoint) {
			return true
		}
	}
	return false
}

// pathContains reports whether path is dir or lies beneath it.
func pathContains(dir, path string) bool {
	rel, err := filepath.Rel(dir, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// unescapeMountField decodes the octal escapes the kernel uses in /proc/mounts
// fields (\040 for a space, \011 tab, \012 newline, \134 backslash).
func unescapeMountField(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+3 < len(s) {
			if n, err := strconv.ParseUint(s[i+1:i+4], 8, 8); err == nil {
				b.WriteByte(byte(n))
				i += 3
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
