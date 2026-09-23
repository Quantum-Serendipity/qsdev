// Package terraform implements the Terraform/OpenTofu ecosystem module for
// qsdev. It detects Terraform and OpenTofu projects by
// scanning for .tf files, .tf.json files, and lock/config directories, then
// generates devenv.nix fragments, security configs (.terraformrc), pre-commit
// hooks, deny rules, and CI commands for a hardened IaC development environment.
package terraform

import (
	"fmt"
	"maps"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/cloudcommon"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Compile-time interface compliance checks.
var _ ecosystem.EcosystemModule = (*Module)(nil)
var _ ecosystem.SecretDeclarer = (*Module)(nil)
var _ ecosystem.DenyRuleProvider = (*Module)(nil)
var _ ecosystem.WizardFieldProvider = (*Module)(nil)
var _ ecosystem.ManifestFileProvider = (*Module)(nil)
var _ ecosystem.SASTModule = (*Module)(nil)
var _ ecosystem.DevenvYamlInputProvider = (*Module)(nil)

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

	if !result.Detected {
		return result
	}
	if providers := cloudcommon.DetectTerraformProviders(projectRoot); len(providers) > 0 {
		names := slices.Sorted(maps.Keys(providers))
		result.SuggestedConfig.Extras[ExtraCloudProviders] = strings.Join(names, ",")
		result.Evidence = append(result.Evidence, "cloud providers: "+strings.Join(names, ", "))
	}

	return result
}

// ExtraCloudProviders is the ModuleConfig.Extras key holding the
// comma-separated Terraform cloud providers in use (aws, azurerm, google).
const ExtraCloudProviders = "cloud_providers"

// nixpkgsTerraformInput is the flake input devenv resolves
// languages.terraform.version against (config.lib.getInput for
// "nixpkgs-terraform"). It must be present in devenv.yaml whenever the
// fragment pins a Terraform version.
const nixpkgsTerraformInput = "github:stackbuilders/nixpkgs-terraform"

// terraformVersionRe matches the release versions nixpkgs-terraform
// publishes (1.8, 1.8.5).
var terraformVersionRe = regexp.MustCompile(`^[0-9]+\.[0-9]+(\.[0-9]+)?$`)

// DevenvNixFragment returns the Nix code fragment to include in devenv.nix
// for Terraform or OpenTofu language support.
//
// Only devenv's Terraform module has a version option (backed by the
// nixpkgs-terraform input, see DevenvYamlInputs); languages.opentofu has none,
// so an OpenTofu project always uses the nixpkgs opentofu package.
func (m *Module) DevenvNixFragment(config ecosystem.ModuleConfig) (string, error) {
	variant := config.Extra("variant", "terraform")
	version, err := pinnedVersion(config)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString("  languages.")
	b.WriteString(variant)
	b.WriteString(" = {\n")
	b.WriteString("    enable = true;\n")
	if version != "" {
		fmt.Fprintf(&b, "    version = %s;\n", ecosystem.NixString(version))
	}
	b.WriteString("  };\n")
	return b.String(), nil
}

// DevenvYamlInputs contributes the nixpkgs-terraform flake input when the
// fragment pins languages.terraform.version; devenv refuses to evaluate the
// version option without it. The input and the version line are an
// invariant pair.
func (m *Module) DevenvYamlInputs(config ecosystem.ModuleConfig) []ecosystem.DevenvInput {
	if version, err := pinnedVersion(config); err != nil || version == "" {
		return nil
	}
	return []ecosystem.DevenvInput{{URL: nixpkgsTerraformInput, Follows: "nixpkgs"}}
}

// pinnedVersion returns the version the fragment pins: "" for OpenTofu (no
// devenv version option) or when unset, and an error for a Terraform version
// nixpkgs-terraform cannot provide.
func pinnedVersion(config ecosystem.ModuleConfig) (string, error) {
	version := strings.TrimPrefix(strings.TrimSpace(config.Version), "v")
	if version == "" || config.Extra("variant", "terraform") == "opentofu" {
		return "", nil
	}
	if !terraformVersionRe.MatchString(version) {
		return "", fmt.Errorf("invalid Terraform version %q: want a release such as 1.8 or 1.8.5", config.Version)
	}
	return version, nil
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

	// Skip: an existing .terraformrc (credentials blocks, dev_overrides,
	// plugin cache settings) is user-owned and must never be replaced.
	return []types.GeneratedFile{
		{
			Path:     ".terraformrc",
			Content:  []byte(content.String()),
			Mode:     fileutil.ModeReadWrite,
			Strategy: types.Skip,
		},
	}
}

// PreCommitHooks returns pre-commit hook definitions for the Terraform/OpenTofu
// ecosystem, including format checking, validation, linting, and security scanning.
func (m *Module) PreCommitHooks(config ecosystem.ModuleConfig) []ecosystem.HookConfig {
	variant := config.Extra("variant", "terraform")
	binary := binaryName(variant)
	nixPkg := nixPackageName(variant)

	// These are custom hooks (BuiltIn:false), not git-hooks.nix built-ins: the
	// built-in `terraform-format` runs plain `terraform fmt`, which would discard
	// this module's binary selection (tofu for OpenTofu) and the `-check`/
	// `-recursive` flags. NixPackage puts the right binary on PATH so the custom
	// Entry resolves.
	return []ecosystem.HookConfig{
		{
			ID:            "terraform-format",
			Name:          "terraform-format",
			Description:   fmt.Sprintf("Check %s configuration formatting", variant),
			Entry:         binary + " fmt -check -recursive",
			Language:      "system",
			Types:         []string{"terraform"},
			Stages:        []string{"pre-commit"},
			PassFilenames: false,
			BuiltIn:       false,
			NixPackage:    nixPkg,
		},
		validateHook(variant),
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
			NixPackage:    "tflint",
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
			NixPackage:    "tfsec",
		},
	}
}

// validateHook returns the terraform-validate pre-commit hook.
//
// `validate` needs an initialized working directory (providers and modules
// installed), which a fresh clone does not have, and the agent is denied
// `<bin> init`. The hook therefore initializes first with
// `init -backend=false`: no backend or state is touched, but providers and
// modules are downloaded (through the provider mirror when one is
// configured). The two steps need a shell, so the entry runs through `sh -c`
// with NixPackage "bash" (entry rewriting turns `sh` into
// ${pkgs.bash}/bin/sh); the Terraform binary itself resolves from the devenv
// environment, where languages.<variant>.enable and the sibling hooks'
// NixPackage install it.
func validateHook(variant string) ecosystem.HookConfig {
	binary := binaryName(variant)
	return ecosystem.HookConfig{
		ID:            "terraform-validate",
		Name:          "terraform-validate",
		Description:   fmt.Sprintf("Initialize (without backend) and validate %s configuration", variant),
		Entry:         fmt.Sprintf("sh -c '%[1]s init -backend=false -input=false >/dev/null && %[1]s validate'", binary),
		Language:      "system",
		Types:         []string{"terraform"},
		Stages:        []string{"pre-commit"},
		PassFilenames: false,
		BuiltIn:       false,
		NixPackage:    "bash",
	}
}

// nixPackageName returns the nixpkgs package providing the CLI binary for the
// given Terraform variant: opentofu (tofu) or terraform.
func nixPackageName(variant string) string {
	if variant == "opentofu" {
		return "opentofu"
	}
	return "terraform"
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

// VerificationCommands returns project verification commands for the
// Terraform/OpenTofu ecosystem, using the correct binary for the variant.
func (m *Module) VerificationCommands(config ecosystem.ModuleConfig) ecosystem.VerificationCommands {
	bin := binaryName(config.Extra("variant", "terraform"))
	return ecosystem.VerificationCommands{
		Test:   []string{bin + " validate"},
		Lint:   []string{"tflint"},
		Format: []string{bin + " fmt -check"},
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

// SecretDeclarations returns the secrets a Terraform/OpenTofu project may use.
//
// Declarations follow the cloud providers the configuration actually uses
// (ExtraCloudProviders, recorded by Detect); a GCP- or Azure-only project
// declares no AWS keys. Static AWS keys are optional: the AWS module isolates
// credentials per project through AWS_PROFILE (SSO/profile auth), which is
// the preferred model, so requiring long-lived keys would push users away
// from it.
func (m *Module) SecretDeclarations(config ecosystem.ModuleConfig) []ecosystem.SecretDecl {
	if !slices.Contains(strings.Split(config.Extra(ExtraCloudProviders, ""), ","), "aws") {
		return nil
	}
	return []ecosystem.SecretDecl{
		{
			Name:        "AWS_ACCESS_KEY_ID",
			Description: "Optional static AWS access key for the Terraform AWS provider (prefer AWS_PROFILE / SSO)",
			Required:    false,
			Source:      "terraform",
		},
		{
			Name:        "AWS_SECRET_ACCESS_KEY",
			Description: "Optional static AWS secret key for the Terraform AWS provider (prefer AWS_PROFILE / SSO)",
			Required:    false,
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
