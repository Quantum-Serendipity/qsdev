package evidence

// SOC2Framework returns the SOC2 Type II compliance framework definition
// with 6 controls mapped to qsdev's defense-in-depth layers. Control IDs are
// the 2017 Trust Services Criteria (revised 2022) common criteria.
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
			ID:                  "CC7.2",
			Name:                "Monitoring of System Components",
			Desc:                "The entity monitors system components and the operation of those components for anomalies that are indicative of malicious acts.",
			Category:            "Monitoring",
			Layers:              []LayerMapping{},
			NotApplicableReason: "",
		},
		// CC8.1 is the only Change Management criterion in the 2017 TSC
		// (revised 2022). Configuration baselines and change testing are
		// points of focus of CC8.1, not separate criteria, so they are mapped
		// here as supporting layers.
		{
			ID:       "CC8.1",
			Name:     "Change Management Process",
			Desc:     "The entity authorizes, designs, develops or acquires, configures, documents, tests, approves, and implements changes to infrastructure, data, software, and procedures to meet its objectives.",
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
				{
					LayerName:   "nix-hardening",
					Relevance:   "supporting",
					Description: "Nix-based configuration management provides a reproducible, declarative baseline configuration whose changes are tracked in version control (point of focus: creates baseline configuration).",
				},
				{
					LayerName:   "sast",
					Relevance:   "supporting",
					Description: "SAST provides automated code analysis as part of the change testing process, catching security issues before deployment (point of focus: tests system changes).",
				},
				{
					LayerName:   "vulnerability-scanning",
					Relevance:   "supporting",
					Description: "Vulnerability scanning validates that dependency changes do not introduce known security issues (point of focus: tests system changes).",
				},
			},
		},
	}
}
