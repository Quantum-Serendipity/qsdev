// Package terraform implements the Terraform/OpenTofu ecosystem module for
// qsdev. It detects Terraform and OpenTofu projects by
// scanning for .tf files, .tf.json files, and lock/config directories, then
// generates devenv.nix fragments, security configs (.terraformrc), pre-commit
// hooks, deny rules, and CI commands for a hardened IaC development environment.
package terraform

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Compile-time interface compliance checks.
var _ ecosystem.EcosystemModule = (*Module)(nil)
var _ ecosystem.SecretDeclarer = (*Module)(nil)

func init() {
	ecosystem.MustRegisterModule(&Module{})
}

// Module is the stateless Terraform/OpenTofu ecosystem module.
type Module struct{}

// Name returns the canonical module identifier.
func (m *Module) Name() string { return "terraform" }

// DisplayName returns the human-readable label.
func (m *Module) DisplayName() string { return "Terraform/OpenTofu" }

// Tier returns the implementation priority tier (1 = core).
func (m *Module) Tier() int { return 1 }

// Detect scans projectRoot for Terraform/OpenTofu ecosystem indicators.
// It checks for .tf files, .tf.json files, .terraform.lock.hcl, and the
// .opentofu/ directory to distinguish between Terraform and OpenTofu variants.
func (m *Module) Detect(projectRoot string) ecosystem.DetectionResult {
	result := ecosystem.DetectionResult{
		SuggestedConfig: ecosystem.ModuleConfig{
			Extras: make(map[string]string),
		},
	}

	// Determine variant: check for .opentofu/ directory first.
	if fileutil.DirExists(projectRoot, ".opentofu") {
		result.SuggestedConfig.Extras["variant"] = "opentofu"
	} else {
		result.SuggestedConfig.Extras["variant"] = "terraform"
	}

	// Check for .tf files (definitive Terraform/OpenTofu indicator).
	tfFiles, _ := filepath.Glob(filepath.Join(projectRoot, "*.tf"))
	if len(tfFiles) > 0 {
		result.Detected = true
		result.Confidence = ecosystem.ConfidenceCertain
		result.Evidence = append(result.Evidence, "*.tf files found")
	}

	// Check for .tf.json files (definitive indicator).
	tfJSONFiles, _ := filepath.Glob(filepath.Join(projectRoot, "*.tf.json"))
	if len(tfJSONFiles) > 0 {
		result.Detected = true
		result.Confidence = ecosystem.ConfidenceCertain
		result.Evidence = append(result.Evidence, "*.tf.json files found")
	}

	// Check for .terraform.lock.hcl (probable if no .tf files found).
	if fileutil.FileExists(projectRoot, ".terraform.lock.hcl") {
		result.Evidence = append(result.Evidence, ".terraform.lock.hcl found")
		if !result.Detected {
			result.Detected = true
			result.Confidence = ecosystem.ConfidenceProbable
		}
	}

	return result
}

// DevenvNixFragment returns the Nix code fragment to include in devenv.nix
// for Terraform or OpenTofu language support.
func (m *Module) DevenvNixFragment(config ecosystem.ModuleConfig) (string, error) {
	variant := config.Extra("variant", "terraform")

	var b strings.Builder
	b.WriteString("  languages.")
	b.WriteString(variant)
	b.WriteString(" = {\n")
	b.WriteString("    enable = true;\n")
	if config.Version != "" {
		fmt.Fprintf(&b, "    version = %q;\n", config.Version)
	}
	b.WriteString("  };\n")
	return b.String(), nil
}

// DevenvYamlInputs returns additional flake inputs for devenv.yaml.
// Terraform/OpenTofu does not require any additional inputs.
func (m *Module) DevenvYamlInputs(_ ecosystem.ModuleConfig) []ecosystem.DevenvInput {
	return nil
}

// SecurityConfigs returns a .terraformrc file with security-hardened settings.
// The configuration disables Terraform checkpoint telemetry and optionally
// configures a registry mirror for provider installations.
func (m *Module) SecurityConfigs(config ecosystem.ModuleConfig) []types.GeneratedFile {
	var content strings.Builder

	content.WriteString("# Security-hardened Terraform CLI configuration.\n")
	content.WriteString("# " + branding.GeneratedBy() + ".\n")
	content.WriteString("# Requires: Terraform >= 0.13 (provider_installation block) or OpenTofu >= 1.6.\n")
	content.WriteString("#\n")
	content.WriteString("# This file disables checkpoint telemetry to prevent\n")
	content.WriteString("# information leakage and optionally enforces a\n")
	content.WriteString("# registry mirror for provider supply chain security.\n\n")
	content.WriteString("disable_checkpoint = true\n")

	if mirror := config.Extra("registry_mirror", ""); mirror != "" {
		content.WriteString("\nprovider_installation {\n")
		content.WriteString("  network_mirror {\n")
		fmt.Fprintf(&content, "    url = %q\n", mirror)
		content.WriteString("  }\n")
		content.WriteString("  direct {\n")
		content.WriteString("    exclude = [\"registry.terraform.io/*/*\"]\n")
		content.WriteString("  }\n")
		content.WriteString("}\n")
	}

	return []types.GeneratedFile{
		{
			Path:     ".terraformrc",
			Content:  []byte(content.String()),
			Mode:     0o644,
			Strategy: types.Overwrite,
		},
	}
}

// PreCommitHooks returns pre-commit hook definitions for the Terraform/OpenTofu
// ecosystem, including format checking, validation, linting, and security scanning.
func (m *Module) PreCommitHooks(config ecosystem.ModuleConfig) []ecosystem.HookConfig {
	variant := config.Extra("variant", "terraform")
	binary := binaryName(variant)

	return []ecosystem.HookConfig{
		{
			ID:            "terraform_fmt",
			Name:          "terraform_fmt",
			Description:   fmt.Sprintf("Check %s configuration formatting", variant),
			Entry:         binary + " fmt -check -recursive",
			Language:      "system",
			Types:         []string{"terraform"},
			Stages:        []string{"pre-commit"},
			PassFilenames: false,
			BuiltIn:       true,
		},
		{
			ID:            "terraform_validate",
			Name:          "terraform_validate",
			Description:   fmt.Sprintf("Validate %s configuration syntax", variant),
			Entry:         binary + " validate",
			Language:      "system",
			Types:         []string{"terraform"},
			Stages:        []string{"pre-commit"},
			PassFilenames: false,
			BuiltIn:       true,
		},
		{
			ID:            "tflint",
			Name:          "tflint",
			Description:   "Lint Terraform configurations with tflint",
			Entry:         "tflint",
			Language:      "system",
			Types:         []string{"terraform"},
			Stages:        []string{"pre-commit"},
			PassFilenames: false,
			BuiltIn:       false,
		},
		{
			ID:            "tfsec",
			Name:          "tfsec",
			Description:   "Security scan Terraform configurations with tfsec",
			Entry:         "tfsec .",
			Language:      "system",
			Types:         []string{"terraform"},
			Stages:        []string{"pre-commit"},
			PassFilenames: false,
			BuiltIn:       false,
		},
	}
}

// DenyRules returns Claude Code deny-rule patterns for Terraform/OpenTofu.
// For Terraform, rules deny direct terraform init and apply without plan.
// For OpenTofu, rules cover both the tofu and terraform binaries.
func (m *Module) DenyRules(config ecosystem.ModuleConfig) []string {
	variant := config.Extra("variant", "terraform")

	rules := []string{
		"Bash(terraform init *)",
		"Bash(terraform apply *)",
		"Bash(terraform providers *)",
	}

	if variant == "opentofu" {
		rules = append(rules,
			"Bash(tofu init *)",
			"Bash(tofu apply *)",
			"Bash(tofu providers *)",
		)
	}

	return rules
}

// CICommands returns CI pipeline commands for Terraform/OpenTofu,
// covering initialization, validation, planning, linting, and security scanning.
func (m *Module) CICommands(config ecosystem.ModuleConfig) []ecosystem.CICommand {
	variant := config.Extra("variant", "terraform")
	binary := binaryName(variant)

	return []ecosystem.CICommand{
		{
			Name:        binary + "-init",
			Command:     binary + " init -backend=false",
			Description: fmt.Sprintf("Initialize %s providers without backend", variant),
			Phase:       ecosystem.CIPhaseInstall,
		},
		{
			Name:        binary + "-validate",
			Command:     binary + " validate",
			Description: fmt.Sprintf("Validate %s configuration syntax", variant),
			Phase:       ecosystem.CIPhaseTest,
		},
		{
			Name:        binary + "-plan",
			Command:     binary + " plan",
			Description: fmt.Sprintf("Generate %s execution plan", variant),
			Phase:       ecosystem.CIPhaseTest,
		},
		{
			Name:        "tflint",
			Command:     "tflint",
			Description: "Lint Terraform configurations with tflint",
			Phase:       ecosystem.CIPhaseScan,
		},
		{
			Name:        "tfsec",
			Command:     "tfsec .",
			Description: "Security scan Terraform configurations with tfsec",
			Phase:       ecosystem.CIPhaseScan,
		},
	}
}

// PackageManagers returns metadata about the Terraform registry provider system.
func (m *Module) PackageManagers() []ecosystem.PackageManagerInfo {
	return []ecosystem.PackageManagerInfo{
		{
			Name:                 "terraform-registry",
			LockFile:             ".terraform.lock.hcl",
			FrozenInstallCommand: "terraform init -lockfile=readonly",
			AgeGatingSupport:     false,
		},
	}
}

// WizardFields returns wizard form fields for Terraform/OpenTofu configuration.
func (m *Module) WizardFields() []ecosystem.WizardField {
	return []ecosystem.WizardField{
		{
			Key:         "terraform_variant",
			Label:       "IaC tool",
			Description: "Select the infrastructure-as-code tool to use",
			Type:        ecosystem.FieldTypeSelect,
			Options: []ecosystem.WizardOption{
				{Label: "Terraform", Value: "terraform"},
				{Label: "OpenTofu", Value: "opentofu"},
			},
			Default: "terraform",
		},
		{
			Key:         "terraform_version",
			Label:       "Version",
			Description: "Specify the Terraform/OpenTofu version (e.g. 1.8.0)",
			Type:        ecosystem.FieldTypeInput,
			Default:     "",
		},
	}
}

// VerificationCommands returns project verification commands for the Terraform/OpenTofu ecosystem.
func (m *Module) VerificationCommands(_ ecosystem.ModuleConfig) ecosystem.VerificationCommands {
	return ecosystem.VerificationCommands{
		Test:   []string{"terraform validate"},
		Lint:   []string{"tflint"},
		Format: []string{"terraform fmt -check"},
	}
}

// ManifestFiles returns manifest file metadata for the Terraform/OpenTofu ecosystem.
func (m *Module) ManifestFiles(_ ecosystem.ModuleConfig) []ecosystem.ManifestFileInfo {
	return []ecosystem.ManifestFileInfo{
		{
			Path:           "*.tf",
			Ecosystem:      "terraform",
			VSSupported:    false,
			LockFile:       ".terraform.lock.hcl",
			LockFilePolicy: ecosystem.LockFilePolicyRequired,
		},
	}
}

// SecretDeclarations returns the secrets required by a Terraform/OpenTofu project.
func (m *Module) SecretDeclarations(_ ecosystem.ModuleConfig) []ecosystem.SecretDecl {
	return []ecosystem.SecretDecl{
		{
			Name:        "AWS_ACCESS_KEY_ID",
			Description: "AWS access key for Terraform provider authentication",
			Required:    true,
			Source:      "terraform",
		},
		{
			Name:        "AWS_SECRET_ACCESS_KEY",
			Description: "AWS secret key for Terraform provider authentication",
			Required:    true,
			Source:      "terraform",
		},
	}
}

// --- helpers ---

// binaryName returns the CLI binary name for the given variant.
func binaryName(variant string) string {
	if variant == "opentofu" {
		return "tofu"
	}
	return "terraform"
}

// SemgrepRuleSets returns Semgrep rule set identifiers relevant to Terraform projects.
func (m *Module) SemgrepRuleSets() []string {
	return []string{"p/terraform", "p/terraform-aws"}
}
