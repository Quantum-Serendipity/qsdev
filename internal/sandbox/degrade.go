package sandbox

// DetermineTier selects the strongest sandbox tier available given the
// detected system capabilities.
func DetermineTier(caps *SystemCapabilities) DegradationTier {
	if caps == nil {
		return TierUnsandboxed
	}

	hasBwrapUserns := caps.HasBwrap && caps.HasUserNS
	hasLandlock := caps.LandlockABI > 0
	hasSeccomp := caps.HasSeccomp

	switch {
	case hasBwrapUserns && hasLandlock && hasSeccomp:
		return TierFull
	case hasBwrapUserns && hasSeccomp:
		return TierBwrapWithoutLandlock
	case hasBwrapUserns && hasLandlock:
		return TierBwrapWithoutSeccomp
	case caps.HasSystemdRun:
		return TierSystemdRun
	default:
		return TierUnsandboxed
	}
}

// TierMessage returns a human-readable description and remediation for a
// degradation tier. Empty string for TierFull (no message needed).
func TierMessage(tier DegradationTier) string {
	switch tier {
	case TierFull:
		return ""
	case TierBwrapWithoutLandlock:
		return "Landlock filesystem restriction unavailable. Sandbox provides namespace " +
			"isolation but cannot restrict filesystem access within the namespace. " +
			"Upgrade to kernel >= 5.13 for full isolation."
	case TierBwrapWithoutSeccomp:
		return "Seccomp syscall filtering unavailable. Sandbox provides namespace and " +
			"filesystem isolation but cannot block dangerous syscalls. " +
			"Check /proc/sys/kernel/seccomp/actions_avail."
	case TierSystemdRun:
		return "Bubblewrap namespace isolation unavailable. Using systemd-run for " +
			"resource limits only (no filesystem or network isolation). " +
			"Install bubblewrap and enable unprivileged user namespaces for full isolation."
	case TierUnsandboxed:
		return "No sandbox isolation available. Hooks run with full user permissions. " +
			"Install bubblewrap, or enable systemd-run --user for basic resource limits. " +
			"Run 'qsdev doctor' for detailed remediation."
	default:
		return ""
	}
}

// TierSecurityLevel returns a qualitative label for the tier's isolation level.
func TierSecurityLevel(tier DegradationTier) string {
	switch tier {
	case TierFull:
		return "strong"
	case TierBwrapWithoutLandlock, TierBwrapWithoutSeccomp:
		return "moderate"
	case TierSystemdRun:
		return "minimal"
	case TierUnsandboxed:
		return "none"
	default:
		return "unknown"
	}
}
