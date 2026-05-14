package evidence

// SOC2Framework returns the SOC2 Type II compliance framework definition
// with 8 controls mapped to gdev's defense-in-depth layers.
func SOC2Framework() Framework {
	return Framework{
		ID:          "soc2",
		Name:        "SOC 2 Type II",
		Version:     "2017",
		Description: "AICPA Trust Services Criteria for Security, Availability, Processing Integrity, Confidentiality, and Privacy",
		Controls:    soc2Controls,
	}
}

func soc2Controls() []ControlDefinition {
	return []ControlDefinition{
		{
			ID:       "CC6.1",
			Name:     "Logical and Physical Access Controls",
			Desc:     "The entity implements logical access security software, infrastructure, and architectures over protected information assets to protect them from security events.",
			Category: "Access Control",
			Layers: []LayerMapping{
				{
					LayerName:   "pretooluse-hooks",
					Relevance:   "primary",
					Description: "PreToolUse hooks enforce permission boundaries on AI agent operations, preventing unauthorized access to protected resources.",
				},
				{
					LayerName:   "nix-hardening",
					Relevance:   "supporting",
					Description: "Nix sandbox and restricted eval settings limit the attack surface of the build environment.",
				},
			},
		},
		{
			ID:       "CC6.6",
			Name:     "System Boundary Protection",
			Desc:     "The entity implements logical access security measures to protect against threats from sources outside its system boundaries.",
			Category: "Access Control",
			Layers: []LayerMapping{
				{
					LayerName:   "install-script-blocking",
					Relevance:   "primary",
					Description: "Blocks execution of unverified install scripts from external package sources, protecting against supply chain injection.",
				},
				{
					LayerName:   "age-gating",
					Relevance:   "supporting",
					Description: "Age-gating quarantines newly published packages, reducing exposure to supply chain attacks via typosquatting or account compromise.",
				},
			},
		},
		{
			ID:       "CC6.8",
			Name:     "Prevention and Detection of Malicious Code",
			Desc:     "The entity implements controls to prevent or detect and act upon the introduction of unauthorized or malicious software.",
			Category: "Security Operations",
			Layers: []LayerMapping{
				{
					LayerName:   "secrets-scanning",
					Relevance:   "primary",
					Description: "Gitleaks and ripsecrets detect secrets and credentials in source code before they are committed.",
				},
				{
					LayerName:   "sast",
					Relevance:   "primary",
					Description: "Semgrep performs static analysis to identify security vulnerabilities, code quality issues, and malicious patterns.",
				},
				{
					LayerName:   "vulnerability-scanning",
					Relevance:   "primary",
					Description: "Dependency vulnerability scanning (Grype, Socket) identifies known vulnerabilities in third-party packages.",
				},
				{
					LayerName:   "age-gating",
					Relevance:   "supporting",
					Description: "Package age-gating prevents installation of recently published packages that may contain malicious code.",
				},
			},
		},
		{
			ID:       "CC7.1",
			Name:     "Detection of Changes and Anomalies",
			Desc:     "To meet its objectives, the entity uses detection and monitoring procedures to identify changes to configurations and new vulnerabilities.",
			Category: "Monitoring",
			Layers: []LayerMapping{
				{
					LayerName:   "sast",
					Relevance:   "primary",
					Description: "SAST tools detect anomalous code patterns, security anti-patterns, and configuration changes that may indicate compromise.",
				},
				{
					LayerName:   "secrets-scanning",
					Relevance:   "primary",
					Description: "Pre-commit secrets scanning detects unauthorized credential exposure as code changes are made.",
				},
			},
		},
		{
			ID:       "CC7.2",
			Name:     "Monitoring of System Components",
			Desc:     "The entity monitors system components and the operation of those components for anomalies that are indicative of malicious acts.",
			Category: "Monitoring",
			Layers:   []LayerMapping{},
			NotApplicableReason: "",
		},
		{
			ID:       "CC8.1",
			Name:     "Change Management Process",
			Desc:     "The entity authorizes, designs, develops or acquires, configures, documents, tests, approves, and implements changes to infrastructure and software.",
			Category: "Change Management",
			Layers: []LayerMapping{
				{
					LayerName:   "lock-file-enforcement",
					Relevance:   "primary",
					Description: "Lock file enforcement ensures all dependency changes are explicit, reproducible, and trackable through version control.",
				},
				{
					LayerName:   "pretooluse-hooks",
					Relevance:   "supporting",
					Description: "PreToolUse hooks enforce change authorization policies for AI agent operations, preventing unauthorized modifications.",
				},
			},
		},
		{
			ID:       "CC8.2",
			Name:     "Configuration Management",
			Desc:     "The entity establishes a process for the identification, documentation, and approval of system configuration changes.",
			Category: "Change Management",
			Layers: []LayerMapping{
				{
					LayerName:   "nix-hardening",
					Relevance:   "primary",
					Description: "Nix-based configuration management provides reproducible, declarative system configuration with full audit trail.",
				},
			},
		},
		{
			ID:       "CC8.3",
			Name:     "Testing of Changes",
			Desc:     "The entity tests changes to meet objectives before moving changes to production.",
			Category: "Change Management",
			Layers: []LayerMapping{
				{
					LayerName:   "sast",
					Relevance:   "primary",
					Description: "SAST provides automated code analysis as part of the change testing process, catching security issues before deployment.",
				},
				{
					LayerName:   "vulnerability-scanning",
					Relevance:   "supporting",
					Description: "Vulnerability scanning validates that dependency changes do not introduce known security issues.",
				},
			},
		},
	}
}
