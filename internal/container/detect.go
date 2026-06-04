package container

import (
	"context"
	"fmt"
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
		info.Path = path
		info.Available = append(info.Available, RuntimePodmanRootless) // may be corrected below

		// Determine version.
		if out, err := prober.Output(ctx, "podman", "version", "--format", "{{.Client.Version}}"); err == nil {
			info.Version = strings.TrimSpace(string(out))
		}

		// Determine rootless status.
		rootless := true
		if out, err := prober.Output(ctx, "podman", "info", "--format", "{{.Host.Security.Rootless}}"); err == nil {
			rootless = strings.TrimSpace(string(out)) == "true"
		}
		info.Rootless = rootless

		if rootless {
			info.Active = RuntimePodmanRootless
			// Replace the provisional entry.
			info.Available[len(info.Available)-1] = RuntimePodmanRootless
			xdg := prober.Getenv("XDG_RUNTIME_DIR")
			if xdg != "" {
				info.SocketPath = xdg + "/podman/podman.sock"
			}
		} else {
			info.Active = RuntimePodmanRootful
			info.Available[len(info.Available)-1] = RuntimePodmanRootful
			info.SocketPath = "/run/podman/podman.sock"
		}
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
// that affect whether rootless mode is sufficient.
func DetectCapabilities(ctx context.Context, prober Prober, info *RuntimeInfo) (*Capabilities, error) {
	if info == nil {
		return nil, fmt.Errorf("detecting capabilities: nil RuntimeInfo")
	}
	caps := &Capabilities{}

	// GPU detection: check for NVIDIA devices.
	if matches, err := prober.Glob("/dev/nvidia*"); err == nil && len(matches) > 0 {
		caps.GPUPassthrough = true
	}

	// NFS detection: parse /proc/mounts for nfs/nfs4 entries.
	if data, err := prober.ReadFile("/proc/mounts"); err == nil {
		for line := range strings.SplitSeq(string(data), "\n") {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				fsType := fields[2]
				if fsType == "nfs" || fsType == "nfs4" {
					caps.NFSMounts = true
					break
				}
			}
		}
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
