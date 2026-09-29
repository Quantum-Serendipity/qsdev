package evidence

// ASVSFramework returns the OWASP Application Security Verification Standard
// framework definition with 5 controls mapped to qsdev's defense-in-depth layers.
// Control IDs, names and descriptions follow the ASVS 4.0.3 requirement text.
func ASVSFramework() Framework {
	return Framework{
		ID:          "asvs",
		Name:        "OWASP ASVS",
		Version:     "4.0.3",
		Description: "OWASP Application Security Verification Standard — Supply Chain and Configuration verification requirements",
		Controls:    asvsControls,
	}
}

func asvsControls() []ControlDefinition {
	return []ControlDefinition{
		{
			ID:       "10.2.1",
			Name:     "No Unauthorized Phone Home",
			Desc:     "Verify that the application source code and third party libraries do not contain unauthorized phone home or data collection capabilities. Where such functionality exists, obtain the user's permission for it to operate before collecting any data.",
			Category: "Malicious Code",
			Layers: []LayerMapping{
				{
					LayerName:   "age-gating",
					Relevance:   "primary",
					Description: "Age-gating quarantines newly published packages, preventing installation of recently compromised or typosquatted packages.",
				},
				{
					LayerName:   "install-script-blocking",
					Relevance:   "primary",
					Description: "Blocks execution of unverified install scripts that could exfiltrate data or establish unauthorized communication channels.",
				},
			},
		},
		{
			ID:       "10.3.2",
			Name:     "Integrity Protections",
			Desc:     "Verify that the application employs integrity protections, such as code signing or subresource integrity. The application must not load or execute code from untrusted sources, such as loading includes, modules, plugins, code, or libraries from untrusted sources or the Internet.",
			Category: "Malicious Code",
			Layers: []LayerMapping{
				{
					LayerName:   "vulnerability-scanning",
					Relevance:   "primary",
					Description: "Dependency vulnerability scanning identifies known CVEs and security advisories in third-party packages.",
				},
				{
					LayerName:   "sast",
					Relevance:   "primary",
					Description: "Static analysis detects code patterns associated with malicious behavior, backdoors, and integrity violations.",
				},
			},
		},
		{
			ID:       "14.2.1",
			Name:     "Up-to-Date Dependencies",
			Desc:     "Verify that all components are up to date, preferably using a dependency checker during build or compile time.",
			Category: "Configuration",
			Layers: []LayerMapping{
				{
					LayerName:   "vulnerability-scanning",
					Relevance:   "primary",
					Description: "Vulnerability scanning identifies outdated dependencies with known security issues, driving timely updates.",
				},
				{
					LayerName:   "lock-file-enforcement",
					Relevance:   "supporting",
					Description: "Lock file enforcement ensures dependency versions are explicit and tracked, enabling reproducible updates.",
				},
			},
		},
		{
			ID:       "14.2.2",
			Name:     "Unnecessary Features Disabled",
			Desc:     "Verify that all unnecessary features, documentation, samples, and configurations are removed.",
			Category: "Configuration",
			Layers: []LayerMapping{
				{
					LayerName:   "nix-hardening",
					Relevance:   "primary",
					Description: "Nix hardening restricts eval, disables sandbox escape paths, and enforces minimal configuration surfaces.",
				},
			},
		},
		{
			ID:       "1.2.1",
			Name:     "Low-Privilege Service Accounts",
			Desc:     "Verify the use of unique or special low-privilege operating system accounts for all application components, services, and servers.",
			Category: "Architecture",
			Layers: []LayerMapping{
				{
					LayerName:   "nix-hardening",
					Relevance:   "primary",
					Description: "Nix sandboxing and restricted-eval enforce privilege separation and minimal capability sets for build processes.",
				},
			},
		},
	}
}
