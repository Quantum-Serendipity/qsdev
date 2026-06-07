// Package azure implements the Azure CLI ecosystem module for qsdev. It
// detects Azure projects by scanning for Terraform azurerm providers,
// azure-pipelines.yml, Bicep files, Azure Developer CLI config, and the
// .azure directory. It generates devenv.nix fragments with ARM environment
// variables, provides deny rules and read-deny paths for credential
// protection, and contributes doctor checks for Azure authentication.
package azure

import (
	"path/filepath"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/cloudcommon"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Compile-time interface compliance checks.
var _ ecosystem.EcosystemModule = (*Module)(nil)
var _ ecosystem.DenyRuleProvider = (*Module)(nil)
var _ ecosystem.ReadDenyRuleProvider = (*Module)(nil)
var _ ecosystem.PackageProvider = (*Module)(nil)
var _ ecosystem.DoctorCheckProvider = (*Module)(nil)

func init() {
	ecosystem.MustRegisterModule(&Module{})
}

// Module is the stateless Azure CLI ecosystem module.
type Module struct{}

// Name returns the canonical module identifier.
func (m *Module) Name() string { return "azure" }

// DisplayName returns the human-readable label.
func (m *Module) DisplayName() string { return "Azure CLI" }

// Tier returns the implementation priority tier (2 = standard).
func (m *Module) Tier() int { return 2 }

// Detect scans projectRoot for Azure ecosystem indicators. Terraform azurerm
// providers, azure-pipelines.yml, Bicep files, and azure.yaml yield Certain
// confidence. The .azure/ directory alone yields Probable.
func (m *Module) Detect(projectRoot string) ecosystem.DetectionResult {
	var (
		evidence   []string
		confidence = ecosystem.ConfidenceAbsent
		detected   bool
	)

	// Check for Terraform azurerm provider (Certain).
	providers := cloudcommon.DetectTerraformProviders(projectRoot)
	if providers["azurerm"] {
		evidence = append(evidence, `Terraform provider "azurerm" found`)
		confidence = ecosystem.ConfidenceCertain
		detected = true
	}

	// Check for azure-pipelines.yml (Certain).
	if fileutil.FileExists(projectRoot, "azure-pipelines.yml") {
		evidence = append(evidence, "azure-pipelines.yml found")
		confidence = ecosystem.ConfidenceCertain
		detected = true
	}

	// Check for Bicep files (Certain).
	bicepFiles, _ := filepath.Glob(filepath.Join(projectRoot, "*.bicep"))
	if len(bicepFiles) > 0 {
		evidence = append(evidence, "*.bicep files found")
		confidence = ecosystem.ConfidenceCertain
		detected = true
	}

	// Check for azure.yaml — Azure Developer CLI (Certain).
	if fileutil.FileExists(projectRoot, "azure.yaml") {
		evidence = append(evidence, "azure.yaml found")
		confidence = ecosystem.ConfidenceCertain
		detected = true
	}

	// Check for .azure/ directory (Probable).
	if fileutil.DirExists(projectRoot, ".azure") {
		evidence = append(evidence, ".azure/ directory found")
		if !detected {
			confidence = ecosystem.ConfidenceProbable
			detected = true
		}
	}

	if !detected {
		return ecosystem.DetectionResult{
			Detected:   false,
			Confidence: ecosystem.ConfidenceAbsent,
		}
	}

	return ecosystem.DetectionResult{
		Detected:   true,
		Confidence: confidence,
		Evidence:   evidence,
	}
}

// DevenvNixFragment returns the Nix code fragment to include in devenv.nix
// for Azure CLI support. It sets placeholder environment variables for
// ARM_SUBSCRIPTION_ID and ARM_TENANT_ID.
func (m *Module) DevenvNixFragment(_ ecosystem.ModuleConfig) (string, error) {
	return `  env.ARM_SUBSCRIPTION_ID = "PLACEHOLDER -- set to your Azure subscription ID";
  env.ARM_TENANT_ID = "PLACEHOLDER -- set to your Azure tenant ID";
`, nil
}

// DevenvPackages returns the Nix packages required for the Azure ecosystem.
// When the k8s extra is set, kubelogin is appended for AKS authentication.
func (m *Module) DevenvPackages(config ecosystem.ModuleConfig) []string {
	pkgs := []string{"azure-cli"}
	if config.Extra("k8s", "") == "true" {
		pkgs = append(pkgs, "kubelogin")
	}
	return pkgs
}

// DenyRules returns Claude Code deny-rule patterns for the Azure ecosystem.
func (m *Module) DenyRules(_ ecosystem.ModuleConfig) []string {
	return cloudcommon.BashDenyRules(cloudcommon.Azure)
}

// ReadDenyRules returns file-path patterns that should be denied read access
// for Azure credential files.
func (m *Module) ReadDenyRules(_ ecosystem.ModuleConfig) []string {
	return cloudcommon.ReadDenyPaths(cloudcommon.Azure)
}

// DoctorChecks returns health checks for verifying Azure CLI authentication
// and environment variable configuration.
func (m *Module) DoctorChecks(_ ecosystem.ModuleConfig) []ecosystem.DoctorCheck {
	return []ecosystem.DoctorCheck{
		{
			Name:        "azure-auth",
			Description: "Azure login status",
			Command:     "az account show",
			Timeout:     5,
			Provider:    "azure",
		},
		{
			Name:        "azure-sub",
			Description: "ARM_SUBSCRIPTION_ID",
			EnvCheck:    "ARM_SUBSCRIPTION_ID",
			Provider:    "azure",
		},
	}
}

// SecurityConfigs returns nil. Azure CLI configuration is user-specific and
// not managed at the project level.
func (m *Module) SecurityConfigs(_ ecosystem.ModuleConfig) []types.GeneratedFile {
	return nil
}

// PreCommitHooks returns nil. Azure has no project-level pre-commit hooks.
func (m *Module) PreCommitHooks(_ ecosystem.ModuleConfig) []ecosystem.HookConfig {
	return nil
}

// CICommands returns nil. Azure CI commands are pipeline-specific and not
// generated at the module level.
func (m *Module) CICommands(_ ecosystem.ModuleConfig) []ecosystem.CICommand {
	return nil
}

// PackageManagers returns nil. Azure CLI is not a package manager.
func (m *Module) PackageManagers() []ecosystem.PackageManagerInfo {
	return nil
}

// VerificationCommands returns an empty set. Azure has no standard
// verification commands at the module level.
func (m *Module) VerificationCommands(_ ecosystem.ModuleConfig) ecosystem.VerificationCommands {
	return ecosystem.VerificationCommands{}
}
