package doctor

import (
	"context"
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
	"github.com/Quantum-Serendipity/qsdev/internal/sandbox/backendselect"
)

// SandboxSection holds the hook sandbox check results for the doctor report.
type SandboxSection struct {
	Detected        bool                 `json:"detected"`
	Tier            string               `json:"tier"`
	TierDescription string               `json:"tier_description"`
	SecurityLevel   string               `json:"security_level"`
	Items           []ContainerCheckItem `json:"items"`
	Warnings        []string             `json:"warnings,omitempty"`
	Recommendations []string             `json:"recommendations,omitempty"`
}

// RunSandboxCheck probes sandbox capabilities and returns a section for the
// doctor report.
func RunSandboxCheck(ctx context.Context, prober sandbox.SandboxProber) *SandboxSection {
	caps := sandbox.ProbeCapabilities(ctx, prober)

	// Report the EFFECTIVE tier — the tier of the backend that will actually be
	// selected and run — rather than the probed DetermineTier(caps). A host may
	// support a stronger tier in principle, but if the backend binary is not
	// available at selection time the doctor must not overstate the isolation
	// the tool can deliver.
	_, tier := backendselect.ResolveBackend(*caps)

	section := &SandboxSection{
		Detected:        true,
		Tier:            tier.String(),
		TierDescription: sandbox.TierMessage(tier),
		SecurityLevel:   sandbox.TierSecurityLevel(tier),
	}

	section.Items = buildSandboxItems(caps)

	if msg := sandbox.TierMessage(tier); msg != "" {
		section.Warnings = append(section.Warnings, msg)
	}
	section.Warnings = append(section.Warnings, sandbox.FilteredNetworkNotice)

	switch tier {
	case sandbox.TierUnsandboxed:
		section.Recommendations = append(section.Recommendations,
			"Install bubblewrap for hook namespace isolation")
		section.Recommendations = append(section.Recommendations,
			"Enable unprivileged user namespaces in kernel config")
	case sandbox.TierSystemdRun:
		section.Recommendations = append(section.Recommendations,
			"Install bubblewrap for full namespace isolation (currently using systemd-run only)")
	case sandbox.TierBwrapWithoutLandlock:
		section.Recommendations = append(section.Recommendations, sandbox.LandlockRemediation)
	case sandbox.TierBwrapWithoutSeccomp:
		section.Recommendations = append(section.Recommendations, sandbox.SeccompRemediation)
	case sandbox.TierBwrapOnly:
		section.Recommendations = append(section.Recommendations,
			"Use a qsdev build that ships the ll-restrict helper and seccomp filter (the Nix build) for Landlock and seccomp layers")
	case sandbox.TierFull:
		// no recommendations needed
	}

	return section
}

func buildSandboxItems(caps *sandbox.SystemCapabilities) []ContainerCheckItem {
	var items []ContainerCheckItem

	items = append(items, boolItem("Bubblewrap", caps.HasBwrap,
		"installed", "not found — install bubblewrap"))
	items = append(items, boolItem("User Namespaces", caps.HasUserNS,
		"enabled", "disabled — enable unprivileged user namespaces"))
	items = append(items, landlockItem(caps.LandlockABI))
	items = append(items, boolItem("Seccomp", caps.HasSeccomp,
		"available", "not available"))
	items = append(items, boolItem("Cgroups v2", caps.HasCgroupV2,
		"unified hierarchy", "v1 or not detected"))
	items = append(items, boolItem("Cgroup Delegation", caps.HasCgroupDeleg,
		"user delegation active", "not delegated"))
	items = append(items, boolItem("systemd-run", caps.HasSystemdRun,
		"available", "not found"))

	return items
}

func boolItem(label string, ok bool, okSummary, failSummary string) ContainerCheckItem {
	if ok {
		return ContainerCheckItem{Label: label, Status: "ok", Summary: okSummary}
	}
	return ContainerCheckItem{Label: label, Status: "warn", Summary: failSummary}
}

// landlockScopingABI is the first Landlock ABI that scopes IPC (abstract UNIX
// sockets and signals) to the sandbox; ll-restrict enables it from there.
const landlockScopingABI = 6

func landlockItem(abi int) ContainerCheckItem {
	if abi >= landlockScopingABI {
		return ContainerCheckItem{
			Label:   "Landlock",
			Status:  "ok",
			Summary: fmt.Sprintf("ABI v%d", abi),
		}
	}
	if abi > 0 {
		// Abstract UNIX sockets are per network namespace: a hook in a
		// category that keeps the host network can reach host endpoints such
		// as X11 or D-Bus unless Landlock scopes them.
		return ContainerCheckItem{
			Label:  "Landlock",
			Status: "warn",
			Summary: fmt.Sprintf("ABI v%d: filesystem only; abstract UNIX sockets and signals are not scoped "+
				"(needs ABI v%d, Linux 6.12+), so network-allowed hooks can reach host sockets such as X11",
				abi, landlockScopingABI),
		}
	}
	return ContainerCheckItem{
		Label:   "Landlock",
		Status:  "warn",
		Summary: "not enforceable (needs the ll-restrict helper and Landlock enabled in the kernel's boot lsm= list)",
	}
}
