package evidence

// HIPAAFramework returns the HIPAA Security Rule compliance framework definition
// with 5 controls from 45 CFR 164.312 mapped to gdev's defense-in-depth layers.
func HIPAAFramework() Framework {
	return Framework{
		ID:          "hipaa",
		Name:        "HIPAA Security Rule",
		Version:     "2013",
		Description: "Health Insurance Portability and Accountability Act — Technical Safeguards (45 CFR 164.312)",
		Controls:    hipaaControls,
	}
}

func hipaaControls() []ControlDefinition {
	return []ControlDefinition{
		{
			ID:       "164.312(a)(1)",
			Name:     "Access Control",
			Desc:     "Implement technical policies and procedures for electronic information systems that maintain electronic protected health information to allow access only to those persons or software programs that have been granted access rights.",
			Category: "Technical Safeguards",
			Layers: []LayerMapping{
				{
					LayerName:   "pretooluse-hooks",
					Relevance:   "primary",
					Description: "PreToolUse hooks enforce access control policies on AI agent operations, ensuring only authorized tools and resources are accessed.",
				},
				{
					LayerName:   "nix-hardening",
					Relevance:   "supporting",
					Description: "Nix sandbox restricts process capabilities, limiting the attack surface that could lead to unauthorized ePHI access.",
				},
			},
		},
		{
			ID:       "164.312(b)",
			Name:     "Audit Controls",
			Desc:     "Implement hardware, software, and/or procedural mechanisms that record and examine activity in information systems that contain or use electronic protected health information.",
			Category: "Technical Safeguards",
			Layers:   []LayerMapping{},
			NotApplicableReason: "",
		},
		{
			ID:       "164.312(c)(1)",
			Name:     "Integrity",
			Desc:     "Implement policies and procedures to protect electronic protected health information from improper alteration or destruction.",
			Category: "Technical Safeguards",
			Layers: []LayerMapping{
				{
					LayerName:   "lock-file-enforcement",
					Relevance:   "primary",
					Description: "Lock file enforcement ensures dependency integrity by preventing unauthorized or untracked changes to the software supply chain.",
				},
			},
		},
		{
			ID:       "164.312(d)",
			Name:     "Person or Entity Authentication",
			Desc:     "Implement procedures to verify that a person or entity seeking access to electronic protected health information is the one claimed.",
			Category: "Technical Safeguards",
			Layers:   []LayerMapping{},
			NotApplicableReason: "Authentication is an infrastructure/platform concern outside the scope of the development environment. Recommend integrating with your organization's identity provider (SSO/MFA) at the platform level.",
		},
		{
			ID:       "164.312(e)(1)",
			Name:     "Transmission Security",
			Desc:     "Implement technical security measures to guard against unauthorized access to electronic protected health information that is being transmitted over an electronic communications network.",
			Category: "Technical Safeguards",
			Layers:   []LayerMapping{},
			NotApplicableReason: "Transmission security (TLS, encryption in transit) is an infrastructure/network concern outside the scope of the local development environment. Recommend enforcing TLS at the network and deployment layers.",
		},
	}
}
