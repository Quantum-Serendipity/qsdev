package risk

func ApplyCeilings(rawScore int, info *PackageInfo) (cappedScore int, ceilingName string) {
	if info.MalwareDetected {
		return 0, "malware"
	}

	if info.KEVListed {
		return min(rawScore, 5), "kev"
	}

	if info.CVECritical > 0 && info.FixAvailable {
		return min(rawScore, 15), "critical-cve-fix-available"
	}

	if info.CVECritical > 0 && !info.FixAvailable {
		return min(rawScore, 25), "critical-cve-no-fix"
	}

	if info.HasInstallScripts && !info.InstallScriptsBlocked {
		return min(rawScore, 40), "unblocked-install-scripts"
	}

	return rawScore, ""
}
