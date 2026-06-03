package doctor

import (
	"context"
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/internal/container"
	"github.com/Quantum-Serendipity/qsdev/internal/sysinfo"
)

// ContainerSection holds the container runtime check results for the doctor report.
type ContainerSection struct {
	Detected        bool                 `json:"detected"`
	Runtime         string               `json:"runtime"`
	RuntimeName     string               `json:"runtime_name"`
	Version         string               `json:"version"`
	Rootless        bool                 `json:"rootless"`
	SocketPath      string               `json:"socket_path,omitempty"`
	ComposeMethod   string               `json:"compose_method,omitempty"`
	Items           []ContainerCheckItem `json:"items"`
	Warnings        []string             `json:"warnings,omitempty"`
	Recommendations []string             `json:"recommendations,omitempty"`
}

// ContainerCheckItem is a single check within the container runtime section.
type ContainerCheckItem struct {
	Label   string `json:"label"`
	Status  string `json:"status"` // "ok", "warn", "error"
	Summary string `json:"summary"`
	Detail  string `json:"detail,omitempty"`
}

// RunContainerCheck probes the system for container runtimes, detects
// capabilities, and returns a ContainerSection for the doctor report.
// Returns nil when no container runtime is detected.
func RunContainerCheck(ctx context.Context, prober container.Prober, osInfo *sysinfo.OSInfo) *ContainerSection {
	info, err := container.Detect(ctx, prober)
	if err != nil || info.Active == container.RuntimeNone {
		return nil
	}

	caps, err := container.DetectCapabilities(ctx, prober, info)
	if err != nil {
		return nil
	}

	rootless := info.Active == container.RuntimePodmanRootless

	cs := &ContainerSection{
		Detected:      true,
		RuntimeName:   string(info.Active),
		Version:       info.Version,
		Rootless:      rootless,
		SocketPath:    info.SocketPath,
		ComposeMethod: info.ComposeMethod,
	}

	// Build formatted runtime label.
	cs.Runtime = buildRuntimeLabel(info)

	// Build check items.
	cs.Items = append(cs.Items, buildCgroupsItem(caps))
	cs.Items = append(cs.Items, buildUserNSItem(caps, rootless, osInfo))
	cs.Items = append(cs.Items, buildGPUItem(caps, rootless))
	cs.Items = append(cs.Items, buildNFSItem(caps, rootless))

	// Warnings from NeedsRootfulFallback when Podman rootless.
	if rootless {
		cs.Warnings = caps.NeedsRootfulFallback()
	}

	// Recommendations.
	if info.Active == container.RuntimeDocker {
		cs.Recommendations = append(cs.Recommendations,
			"Consider migrating to Podman rootless for improved security. Run: qsdev container migrate")
	}
	if rootless && len(caps.NeedsRootfulFallback()) > 0 {
		cs.Recommendations = append(cs.Recommendations,
			"Rootful fallback needed for GPU/NFS workloads. See docs/nixos-podman-rootless.md")
	}

	return cs
}

func buildRuntimeLabel(info *container.RuntimeInfo) string {
	name := "Podman"
	if info.Active == container.RuntimeDocker {
		name = "Docker"
	}

	if info.Active.IsPodman() {
		mode := "rootful"
		if info.Active == container.RuntimePodmanRootless {
			mode = "rootless"
		}
		return fmt.Sprintf("%s %s (%s)", name, info.Version, mode)
	}

	return fmt.Sprintf("%s %s", name, info.Version)
}

func buildCgroupsItem(caps *container.Capabilities) ContainerCheckItem {
	if caps.CgroupsV2 {
		return ContainerCheckItem{
			Label:   "Cgroups",
			Status:  "ok",
			Summary: "v2",
		}
	}
	return ContainerCheckItem{
		Label:   "Cgroups",
		Status:  "warn",
		Summary: "v1 (resource limits unavailable for rootless)",
	}
}

func buildUserNSItem(caps *container.Capabilities, rootless bool, osInfo *sysinfo.OSInfo) ContainerCheckItem {
	if caps.UserNamespaceConfigured {
		return ContainerCheckItem{
			Label:   "User NS",
			Status:  "ok",
			Summary: "configured (subuid/subgid present)",
		}
	}

	if !rootless {
		// Not rootless Podman, so subuid is not critical.
		return ContainerCheckItem{
			Label:   "User NS",
			Status:  "ok",
			Summary: "not required for rootful mode",
		}
	}

	item := ContainerCheckItem{
		Label:   "User NS",
		Status:  "error",
		Summary: "not configured",
	}

	if osInfo != nil && osInfo.Family == "nixos" {
		item.Detail = "Add to NixOS configuration:\n" +
			"  users.users.<username>.subUidRanges = [{ startUid = 100000; count = 65536; }];\n" +
			"  users.users.<username>.subGidRanges = [{ startGid = 100000; count = 65536; }];\n" +
			"See docs/nixos-podman-rootless.md"
	} else {
		item.Detail = "Run: sudo usermod --add-subuids 100000-165535 --add-subgids 100000-165535 USERNAME"
	}

	return item
}

func buildGPUItem(caps *container.Capabilities, rootless bool) ContainerCheckItem {
	if caps.GPUPassthrough && rootless {
		return ContainerCheckItem{
			Label:   "GPU",
			Status:  "warn",
			Summary: "NVIDIA GPU detected -- rootless cannot pass through GPU devices",
		}
	}
	if caps.GPUPassthrough {
		return ContainerCheckItem{
			Label:   "GPU",
			Status:  "ok",
			Summary: "NVIDIA GPU detected",
		}
	}
	return ContainerCheckItem{
		Label:   "GPU",
		Status:  "ok",
		Summary: "no GPU devices detected",
	}
}

func buildNFSItem(caps *container.Capabilities, rootless bool) ContainerCheckItem {
	if caps.NFSMounts && rootless {
		return ContainerCheckItem{
			Label:   "NFS",
			Status:  "warn",
			Summary: "NFS mounts detected -- rootless cannot bind-mount NFS paths",
		}
	}
	if caps.NFSMounts {
		return ContainerCheckItem{
			Label:   "NFS",
			Status:  "ok",
			Summary: "NFS mounts detected",
		}
	}
	return ContainerCheckItem{
		Label:   "NFS",
		Status:  "ok",
		Summary: "no NFS mounts in project tree",
	}
}
