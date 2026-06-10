package trust

type TrustProbeFunc func(info *McpServerInfo) ProbeResult

type TrustProbeRegistration struct {
	ID       string
	Category string
	Weight   float64
	Fn       TrustProbeFunc
}

var allTrustProbes = []TrustProbeRegistration{
	// Content Origin (0.45)
	{ID: "content-source-local", Category: "content-origin", Weight: 0.20, Fn: probeContentSourceLocal},
	{ID: "no-community-content", Category: "content-origin", Weight: 0.15, Fn: probeNoCommunityContent},
	{ID: "content-signing-verified", Category: "content-origin", Weight: 0.10, Fn: probeContentSigningVerified},

	// Installation & Update (0.30)
	{ID: "verified-installation-source", Category: "installation-update", Weight: 0.10, Fn: probeVerifiedInstallationSource},
	{ID: "pinned-version", Category: "installation-update", Weight: 0.05, Fn: probePinnedVersion},
	{ID: "update-mechanism-controlled", Category: "installation-update", Weight: 0.05, Fn: probeUpdateMechanismControlled},
	{ID: "offline-capable", Category: "installation-update", Weight: 0.10, Fn: probeOfflineCapable},

	// Vulnerability & Attestation (0.25)
	{ID: "no-known-vulnerabilities", Category: "vulnerability-attestation", Weight: 0.15, Fn: probeNoKnownVulnerabilities},
	{ID: "user-attestation", Category: "vulnerability-attestation", Weight: 0.10, Fn: probeUserAttestation},
}

var categoryWeights = map[string]float64{
	"content-origin":            0.45,
	"installation-update":       0.30,
	"vulnerability-attestation": 0.25,
}

func probeContentSourceLocal(info *McpServerInfo) ProbeResult {
	return ProbeResult{
		ProbeID:  "content-source-local",
		Category: "content-origin",
		Pass:     info.IsLocalBinary,
		Weight:   0.20,
	}
}

func probeNoCommunityContent(info *McpServerInfo) ProbeResult {
	return ProbeResult{
		ProbeID:  "no-community-content",
		Category: "content-origin",
		Pass:     !info.ServesCommunityCContent,
		Weight:   0.15,
	}
}

func probeContentSigningVerified(info *McpServerInfo) ProbeResult {
	return ProbeResult{
		ProbeID:  "content-signing-verified",
		Category: "content-origin",
		Pass:     info.HasContentSigning,
		Weight:   0.10,
	}
}

func probeVerifiedInstallationSource(info *McpServerInfo) ProbeResult {
	return ProbeResult{
		ProbeID:  "verified-installation-source",
		Category: "installation-update",
		Pass:     info.VerifiedInstallSource,
		Weight:   0.10,
	}
}

func probePinnedVersion(info *McpServerInfo) ProbeResult {
	return ProbeResult{
		ProbeID:  "pinned-version",
		Category: "installation-update",
		Pass:     info.PinnedVersion,
		Weight:   0.05,
	}
}

func probeUpdateMechanismControlled(info *McpServerInfo) ProbeResult {
	return ProbeResult{
		ProbeID:  "update-mechanism-controlled",
		Category: "installation-update",
		Pass:     info.ControlledUpdates,
		Weight:   0.05,
	}
}

func probeOfflineCapable(info *McpServerInfo) ProbeResult {
	return ProbeResult{
		ProbeID:  "offline-capable",
		Category: "installation-update",
		Pass:     info.OfflineCapable,
		Weight:   0.10,
	}
}

func probeNoKnownVulnerabilities(info *McpServerInfo) ProbeResult {
	return ProbeResult{
		ProbeID:  "no-known-vulnerabilities",
		Category: "vulnerability-attestation",
		Pass:     !info.HasKnownVulnerabilities,
		Weight:   0.15,
	}
}

func probeUserAttestation(info *McpServerInfo) ProbeResult {
	return ProbeResult{
		ProbeID:  "user-attestation",
		Category: "vulnerability-attestation",
		Pass:     info.HasUserAttestation,
		Weight:   0.10,
	}
}
