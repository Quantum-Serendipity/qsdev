package risk

import "time"

type ProbeFunc func(info *PackageInfo) ProbeResult

type ProbeRegistration struct {
	ID         string
	Category   string
	Weight     float64
	Ecosystems []Ecosystem
	Fn         ProbeFunc
}

var allProbes = []ProbeRegistration{
	// publication (0.15)
	{ID: "package-age", Category: "publication", Weight: 0.15, Fn: probePackageAge},
	{ID: "version-age", Category: "publication", Weight: 0.15, Fn: probeVersionAge},
	{ID: "release-frequency", Category: "publication", Weight: 0.15, Fn: stubProbe("release-frequency", "publication", 0.15)},
	{ID: "version-churn", Category: "publication", Weight: 0.15, Fn: stubProbe("version-churn", "publication", 0.15)},
	{ID: "pre-release-version", Category: "publication", Weight: 0.15, Fn: probePreRelease},

	// maintainer (0.12)
	{ID: "maintainer-count", Category: "maintainer", Weight: 0.12, Fn: stubProbe("maintainer-count", "maintainer", 0.12)},
	{ID: "publisher-switching", Category: "maintainer", Weight: 0.12, Fn: stubProbe("publisher-switching", "maintainer", 0.12)},
	{ID: "org-vs-individual", Category: "maintainer", Weight: 0.12, Fn: stubProbe("org-vs-individual", "maintainer", 0.12)},
	{ID: "2fa-status", Category: "maintainer", Weight: 0.12, Fn: stubProbe("2fa-status", "maintainer", 0.12)},
	{ID: "trust-level", Category: "maintainer", Weight: 0.12, Fn: stubProbe("trust-level", "maintainer", 0.12)},

	// behavioral (0.20)
	{ID: "install-scripts-present", Category: "behavioral", Weight: 0.20, Fn: probeInstallScriptsPresent},
	{ID: "install-scripts-blocked", Category: "behavioral", Weight: 0.20, Fn: probeInstallScriptsBlocked},
	{ID: "network-at-install", Category: "behavioral", Weight: 0.20, Fn: stubProbe("network-at-install", "behavioral", 0.20)},
	{ID: "filesystem-writes-at-install", Category: "behavioral", Weight: 0.20, Fn: stubProbe("filesystem-writes-at-install", "behavioral", 0.20)},
	{ID: "obfuscation-detected", Category: "behavioral", Weight: 0.20, Fn: stubProbe("obfuscation-detected", "behavioral", 0.20)},
	{ID: "binary-included", Category: "behavioral", Weight: 0.20, Fn: probeBinaryIncluded},

	// vulnerability (0.35)
	{ID: "cve-critical", Category: "vulnerability", Weight: 0.35, Fn: probeCVECritical},
	{ID: "cve-high", Category: "vulnerability", Weight: 0.35, Fn: probeCVEHigh},
	{ID: "cve-medium", Category: "vulnerability", Weight: 0.35, Fn: stubProbe("cve-medium", "vulnerability", 0.35)},
	{ID: "cve-low", Category: "vulnerability", Weight: 0.35, Fn: stubProbe("cve-low", "vulnerability", 0.35)},
	{ID: "epss-max", Category: "vulnerability", Weight: 0.35, Fn: stubProbe("epss-max", "vulnerability", 0.35)},
	{ID: "kev-listed", Category: "vulnerability", Weight: 0.35, Fn: probeKEVListed},
	{ID: "reachable", Category: "vulnerability", Weight: 0.35, Ecosystems: []Ecosystem{EcosystemGo}, Fn: stubProbe("reachable", "vulnerability", 0.35)},
	{ID: "fix-available", Category: "vulnerability", Weight: 0.35, Fn: stubProbe("fix-available", "vulnerability", 0.35)},

	// popularity (0.08)
	{ID: "download-count", Category: "popularity", Weight: 0.08, Fn: stubProbe("download-count", "popularity", 0.08)},
	{ID: "dependent-count", Category: "popularity", Weight: 0.08, Fn: stubProbe("dependent-count", "popularity", 0.08)},
	{ID: "github-stars", Category: "popularity", Weight: 0.08, Fn: stubProbe("github-stars", "popularity", 0.08)},

	// provenance (0.10)
	{ID: "slsa-level", Category: "provenance", Weight: 0.10, Fn: stubProbe("slsa-level", "provenance", 0.10)},
	{ID: "sigstore-signature", Category: "provenance", Weight: 0.10, Fn: stubProbe("sigstore-signature", "provenance", 0.10)},
	{ID: "npm-provenance", Category: "provenance", Weight: 0.10, Ecosystems: []Ecosystem{EcosystemNpm}, Fn: stubProbe("npm-provenance", "provenance", 0.10)},
	{ID: "checksum-verified", Category: "provenance", Weight: 0.10, Fn: probeChecksumVerified},
	{ID: "trusted-publishing", Category: "provenance", Weight: 0.10, Fn: stubProbe("trusted-publishing", "provenance", 0.10)},
}

func stubProbe(id, category string, weight float64) ProbeFunc {
	return func(_ *PackageInfo) ProbeResult {
		return ProbeResult{
			ProbeID:  id,
			Category: category,
			Status:   ProbeDataUnavailable,
			Weight:   weight,
		}
	}
}

func statusFromScore(score float64) ProbeStatus {
	if score >= 50 {
		return ProbePass
	}
	return ProbeFail
}

func probePackageAge(info *PackageInfo) ProbeResult {
	if info.FirstPublishedAt == nil {
		return ProbeResult{
			ProbeID:  "package-age",
			Category: "publication",
			Status:   ProbeDataUnavailable,
			Weight:   0.15,
		}
	}

	age := time.Since(*info.FirstPublishedAt)
	var score float64
	switch {
	case age < 7*24*time.Hour:
		score = 20
	case age < 30*24*time.Hour:
		score = 50
	case age < 365*24*time.Hour:
		score = 80
	default:
		score = 100
	}

	return ProbeResult{
		ProbeID:  "package-age",
		Category: "publication",
		Status:   statusFromScore(score),
		RawValue: age.Hours() / 24,
		Weight:   0.15,
		Score:    score,
	}
}

func probeVersionAge(info *PackageInfo) ProbeResult {
	if info.PublishedAt == nil {
		return ProbeResult{
			ProbeID:  "version-age",
			Category: "publication",
			Status:   ProbeDataUnavailable,
			Weight:   0.15,
		}
	}

	age := time.Since(*info.PublishedAt)
	var score float64
	switch {
	case age < 3*24*time.Hour:
		score = 30
	case age < 30*24*time.Hour:
		score = 60
	case age < 365*24*time.Hour:
		score = 90
	default:
		score = 100
	}

	return ProbeResult{
		ProbeID:  "version-age",
		Category: "publication",
		Status:   statusFromScore(score),
		RawValue: age.Hours() / 24,
		Weight:   0.15,
		Score:    score,
	}
}

func probePreRelease(info *PackageInfo) ProbeResult {
	score := 100.0
	if info.IsPreRelease {
		score = 90
	}
	return ProbeResult{
		ProbeID:  "pre-release-version",
		Category: "publication",
		Status:   statusFromScore(score),
		RawValue: boolToFloat(info.IsPreRelease),
		Weight:   0.15,
		Score:    score,
	}
}

func probeInstallScriptsPresent(info *PackageInfo) ProbeResult {
	var score float64
	switch {
	case !info.HasInstallScripts:
		score = 100
	case info.InstallScriptsBlocked:
		score = 50
	default:
		score = 0
	}
	return ProbeResult{
		ProbeID:  "install-scripts-present",
		Category: "behavioral",
		Status:   statusFromScore(score),
		RawValue: boolToFloat(info.HasInstallScripts),
		Weight:   0.20,
		Score:    score,
	}
}

func probeInstallScriptsBlocked(info *PackageInfo) ProbeResult {
	score := 100.0
	if info.HasInstallScripts && !info.InstallScriptsBlocked {
		score = 0
	}
	return ProbeResult{
		ProbeID:  "install-scripts-blocked",
		Category: "behavioral",
		Status:   statusFromScore(score),
		RawValue: boolToFloat(info.InstallScriptsBlocked),
		Weight:   0.20,
		Score:    score,
	}
}

func probeBinaryIncluded(info *PackageInfo) ProbeResult {
	score := 100.0
	if info.HasBinaries {
		score = 95
	}
	return ProbeResult{
		ProbeID:  "binary-included",
		Category: "behavioral",
		Status:   statusFromScore(score),
		RawValue: boolToFloat(info.HasBinaries),
		Weight:   0.20,
		Score:    score,
	}
}

func probeCVECritical(info *PackageInfo) ProbeResult {
	score := 100.0
	if info.CVECritical > 0 {
		score = 0
	}
	return ProbeResult{
		ProbeID:  "cve-critical",
		Category: "vulnerability",
		Status:   statusFromScore(score),
		RawValue: float64(info.CVECritical),
		Weight:   0.35,
		Score:    score,
	}
}

func probeCVEHigh(info *PackageInfo) ProbeResult {
	penalty := min(info.CVEHigh*30, 100)
	score := float64(100 - penalty)
	return ProbeResult{
		ProbeID:  "cve-high",
		Category: "vulnerability",
		Status:   statusFromScore(score),
		RawValue: float64(info.CVEHigh),
		Weight:   0.35,
		Score:    score,
	}
}

func probeKEVListed(info *PackageInfo) ProbeResult {
	score := 100.0
	if info.KEVListed {
		score = 0
	}
	return ProbeResult{
		ProbeID:  "kev-listed",
		Category: "vulnerability",
		Status:   statusFromScore(score),
		RawValue: boolToFloat(info.KEVListed),
		Weight:   0.35,
		Score:    score,
	}
}

func probeChecksumVerified(info *PackageInfo) ProbeResult {
	score := 0.0
	if info.HasChecksumVerification {
		score = 100
	}
	return ProbeResult{
		ProbeID:  "checksum-verified",
		Category: "provenance",
		Status:   statusFromScore(score),
		RawValue: boolToFloat(info.HasChecksumVerification),
		Weight:   0.10,
		Score:    score,
	}
}

func boolToFloat(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
