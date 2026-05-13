package cigeneration

import "sort"

// StepContributor provides CI steps for one or more jobs in a workflow.
type StepContributor interface {
	CISteps(cfg GenerateConfig) []CIStep
}

// StepRegistry collects all step contributors and merges their output.
type StepRegistry struct {
	contributors map[string]StepContributor
}

// NewStepRegistry returns an empty StepRegistry.
func NewStepRegistry() *StepRegistry {
	return &StepRegistry{
		contributors: make(map[string]StepContributor),
	}
}

// Register adds a named contributor.
func (r *StepRegistry) Register(name string, c StepContributor) {
	r.contributors[name] = c
}

// CollectSteps gathers steps from all contributors, grouped by job ID.
// Within each job the steps are sorted by Order ascending.
func (r *StepRegistry) CollectSteps(cfg GenerateConfig) map[CIJobID][]CIStep {
	result := make(map[CIJobID][]CIStep)
	for _, c := range r.contributors {
		for _, step := range c.CISteps(cfg) {
			result[step.JobID] = append(result[step.JobID], step)
		}
	}
	for jobID := range result {
		sort.SliceStable(result[jobID], func(i, j int) bool {
			return result[jobID][i].Order < result[jobID][j].Order
		})
	}
	return result
}

// DefaultStepRegistry returns a registry pre-loaded with all built-in contributors.
func DefaultStepRegistry() *StepRegistry {
	r := NewStepRegistry()
	r.Register("harden-runner", &HardenRunnerContributor{})
	r.Register("checkout", &CheckoutContributor{})
	r.Register("semgrep", &SemgrepContributor{})
	r.Register("gitleaks", &GitleaksContributor{})
	r.Register("container-security", &ContainerSecurityContributor{})
	r.Register("vulnerability-scan", &VulnerabilityScanContributor{})
	r.Register("license-compliance", &LicenseComplianceContributor{})
	r.Register("security-review", &SecurityReviewContributor{})
	return r
}

// ---------- Built-in contributors ----------

// HardenRunnerContributor adds the step-security/harden-runner step
// as the first step (Order -100) in every job that has tool steps.
type HardenRunnerContributor struct{}

func (h *HardenRunnerContributor) CISteps(cfg GenerateConfig) []CIStep {
	jobs := activeJobIDs(cfg)
	var steps []CIStep
	for _, jid := range jobs {
		steps = append(steps, CIStep{
			ToolName:    "harden-runner",
			JobID:       jid,
			Name:        "Harden Runner",
			Uses:        ActionHardenRunner.String(),
			UsesComment: ActionHardenRunner.Comment(),
			With:        map[string]string{"egress-policy": "audit"},
			Order:       -100,
		})
	}
	return steps
}

// CheckoutContributor adds the actions/checkout step (Order -50)
// in every job that has tool steps.
type CheckoutContributor struct{}

func (c *CheckoutContributor) CISteps(cfg GenerateConfig) []CIStep {
	jobs := activeJobIDs(cfg)
	var steps []CIStep
	for _, jid := range jobs {
		steps = append(steps, CIStep{
			ToolName:    "checkout",
			JobID:       jid,
			Name:        "Checkout",
			Uses:        ActionCheckout.String(),
			UsesComment: ActionCheckout.Comment(),
			Order:       -50,
		})
	}
	return steps
}

// SemgrepContributor contributes a Semgrep SAST scan to the lint-sast job.
type SemgrepContributor struct{}

func (s *SemgrepContributor) CISteps(cfg GenerateConfig) []CIStep {
	return []CIStep{
		{
			ToolName:    "semgrep",
			JobID:       JobLintSAST,
			Name:        "Semgrep SAST Scan",
			Uses:        ActionSemgrep.String(),
			UsesComment: ActionSemgrep.Comment(),
			Env:         map[string]string{"SEMGREP_APP_TOKEN": "${{ secrets.SEMGREP_APP_TOKEN }}"},
			Order:       10,
		},
	}
}

// GitleaksContributor contributes secret scanning to the secret-scan job.
type GitleaksContributor struct{}

func (g *GitleaksContributor) CISteps(cfg GenerateConfig) []CIStep {
	return []CIStep{
		{
			ToolName: "gitleaks",
			JobID:    JobSecretScan,
			Name:     "Gitleaks Secret Scan",
			Run:      "gitleaks detect --source . --report-format sarif --report-path gitleaks.sarif --no-banner",
			Order:    10,
		},
		{
			ToolName:    "gitleaks-upload",
			JobID:       JobSecretScan,
			Name:        "Upload Gitleaks SARIF",
			Uses:        ActionUploadSarif.String(),
			UsesComment: ActionUploadSarif.Comment(),
			With: map[string]string{
				"sarif_file": "gitleaks.sarif",
				"category":   "gitleaks",
			},
			Condition: "always()",
			Order:     20,
		},
	}
}

// ContainerSecurityContributor adds SBOM generation, vulnerability scanning,
// and cosign verification steps to the container-security job.
// Only active when HasDocker is true.
type ContainerSecurityContributor struct{}

func (c *ContainerSecurityContributor) CISteps(cfg GenerateConfig) []CIStep {
	if !cfg.HasDocker {
		return nil
	}
	return []CIStep{
		{
			ToolName:    "syft",
			JobID:       JobContainerSecurity,
			Name:        "Generate SBOM with Syft",
			Uses:        ActionSyft.String(),
			UsesComment: ActionSyft.Comment(),
			With: map[string]string{
				"artifact-name": "sbom",
				"output-file":   "sbom.spdx.json",
			},
			Order: 10,
		},
		{
			ToolName:    "grype",
			JobID:       JobContainerSecurity,
			Name:        "Scan SBOM with Grype",
			Uses:        ActionGrype.String(),
			UsesComment: ActionGrype.Comment(),
			With: map[string]string{
				"sbom":          "sbom.spdx.json",
				"fail-build":    "true",
				"output-format": "sarif",
			},
			Order: 20,
		},
		{
			ToolName:    "cosign",
			JobID:       JobContainerSecurity,
			Name:        "Install Cosign",
			Uses:        ActionCosignInstaller.String(),
			UsesComment: ActionCosignInstaller.Comment(),
			Order:       30,
		},
	}
}

// VulnerabilityScanContributor adds OSV Scanner to the vulnerability-scan job.
type VulnerabilityScanContributor struct{}

func (v *VulnerabilityScanContributor) CISteps(cfg GenerateConfig) []CIStep {
	return []CIStep{
		{
			ToolName:    "osv-scanner",
			JobID:       JobVulnerabilityScan,
			Name:        "OSV Vulnerability Scanner",
			Uses:        ActionOSVScanner.String(),
			UsesComment: ActionOSVScanner.Comment(),
			Order:       10,
		},
	}
}

// LicenseComplianceContributor adds license scanning to the license-compliance job.
type LicenseComplianceContributor struct{}

func (l *LicenseComplianceContributor) CISteps(cfg GenerateConfig) []CIStep {
	return []CIStep{
		{
			ToolName: "scancode",
			JobID:    JobLicenseCompliance,
			Name:     "License Compliance Scan",
			Run:      "pip install scancode-toolkit && scancode --license --json-pp license-report.json .",
			Order:    10,
		},
		{
			ToolName:    "scancode-upload",
			JobID:       JobLicenseCompliance,
			Name:        "Upload License Report",
			Uses:        ActionUploadArtifact.String(),
			UsesComment: ActionUploadArtifact.Comment(),
			With: map[string]string{
				"name": "license-report",
				"path": "license-report.json",
			},
			Condition: "always()",
			Order:     20,
		},
	}
}

// SecurityReviewContributor adds Claude Code review to the security-review job.
// Only active when HasClaudeCode is true.
type SecurityReviewContributor struct{}

func (s *SecurityReviewContributor) CISteps(cfg GenerateConfig) []CIStep {
	if !cfg.HasClaudeCode {
		return nil
	}
	return []CIStep{
		{
			ToolName:    "claude-code-review",
			JobID:       JobSecurityReview,
			Name:        "Claude Code Security Review",
			Uses:        ActionClaudeCodeReview.String(),
			UsesComment: ActionClaudeCodeReview.Comment(),
			With: map[string]string{
				"review_type": "security",
			},
			Order: 10,
		},
	}
}

// activeJobIDs returns the set of job IDs that will have at least one tool step
// for the given config. This is used by HardenRunner and Checkout contributors
// to avoid adding infrastructure steps to empty jobs.
func activeJobIDs(cfg GenerateConfig) []CIJobID {
	var jobs []CIJobID
	// lint-sast: always active (semgrep)
	jobs = append(jobs, JobLintSAST)
	// secret-scan: always active (gitleaks)
	jobs = append(jobs, JobSecretScan)
	// vulnerability-scan: always active (osv-scanner)
	jobs = append(jobs, JobVulnerabilityScan)
	// container-security: only with Docker
	if cfg.HasDocker {
		jobs = append(jobs, JobContainerSecurity)
	}
	// security-review: only with Claude Code
	if cfg.HasClaudeCode {
		jobs = append(jobs, JobSecurityReview)
	}
	// license-compliance: always active (scancode)
	jobs = append(jobs, JobLicenseCompliance)
	return jobs
}
