package doctor

import (
	"context"
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
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
	tier := sandbox.DetermineTier(caps)

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
		section.Recommendations = append(section.Recommendations,
			"Upgrade kernel to >= 5.13 for Landlock filesystem restriction")
	case sandbox.TierBwrapWithoutSeccomp:
		section.Recommendations = append(section.Recommendations,
			"Enable seccomp support for syscall filtering")
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

func landlockItem(abi int) ContainerCheckItem {
	if abi > 0 {
		return ContainerCheckItem{
			Label:   "Landlock",
			Status:  "ok",
			Summary: fmt.Sprintf("ABI v%d", abi),
		}
	}
	return ContainerCheckItem{
		Label:   "Landlock",
		Status:  "warn",
		Summary: "not available (kernel < 5.13)",
	}
}
