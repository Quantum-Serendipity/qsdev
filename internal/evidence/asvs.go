package evidence

// ASVSFramework returns the OWASP Application Security Verification Standard
// framework definition with 6 controls mapped to gdev's defense-in-depth layers.
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
			ID:       "10.3.1",
			Name:     "Trusted Package Sources",
			Desc:     "Verify that the application source code and third-party libraries do not contain unauthorized phone home or data collection capabilities.",
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
			Name:     "Software Composition Analysis",
			Desc:     "Verify that the application employs integrity protections, such as code signing or subresource integrity. The application must not load or execute code from untrusted sources.",
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
			ID:       "10.3.3",
			Name:     "Unused Dependencies",
			Desc:     "Verify that the application does not include unused packages, frameworks, or libraries that are not necessary for its operation.",
			Category: "Malicious Code",
			Layers:   []LayerMapping{},
			NotApplicableReason: "Unused dependency detection requires language-specific dead-code analysis beyond the scope of gdev's supply chain security layers. Recommend using language-specific tools (e.g., depcheck for Node.js, go mod tidy for Go).",
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
			ID:       "1.14.1",
			Name:     "Configuration Verification",
			Desc:     "Verify the use of a unique or special low-privilege operating system account for all application components, services, and servers.",
			Category: "Configuration",
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
