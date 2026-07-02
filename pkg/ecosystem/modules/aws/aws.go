// Package aws implements the AWS CLI ecosystem module for qsdev. It detects
// AWS projects by scanning for CDK, SAM, Serverless Framework, CodeBuild,
// CodeDeploy, and Terraform AWS provider indicators, then generates devenv.nix
// fragments, deny rules, read-deny paths, wizard fields, and doctor checks for
// a hardened AWS development environment.
package aws

import (
	"os"
	"path/filepath"
	"strings"

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
var _ ecosystem.WizardFieldProvider = (*Module)(nil)
var _ ecosystem.DoctorCheckProvider = (*Module)(nil)

func init() {
	ecosystem.MustRegisterModule(&Module{})
}

// Module is the stateless AWS CLI ecosystem module.
type Module struct{}

// Name returns the canonical module identifier.
func (m *Module) Name() string { return "aws" }

// DisplayName returns the human-readable label.
func (m *Module) DisplayName() string { return "AWS CLI" }

// Tier returns the implementation priority tier (2 = standard).
func (m *Module) Tier() int { return 2 }

// Detect scans projectRoot for AWS ecosystem indicators. Definitive markers
// (cdk.json, serverless.yml, samconfig.toml, SAM templates, Terraform AWS
// provider) yield ConfidenceCertain. Weaker indicators (buildspec.yml,
// appspec.yml, .aws-sam/) yield ConfidenceProbable when no definitive marker
// has already been found.
func (m *Module) Detect(projectRoot string) ecosystem.DetectionResult {
	result := ecosystem.DetectionResult{
		SuggestedConfig: ecosystem.ModuleConfig{
			Extras: make(map[string]string),
		},
	}

	// Definitive indicators -> ConfidenceCertain.
	if fileutil.FileExists(projectRoot, "cdk.json") {
		result.Detected = true
		result.Confidence = ecosystem.ConfidenceCertain
		result.Evidence = append(result.Evidence, "cdk.json found (AWS CDK)")
		result.SuggestedConfig.Extras["cdk"] = "true"
	}

	if fileutil.FileExists(projectRoot, "serverless.yml") || fileutil.FileExists(projectRoot, "serverless.yaml") {
		result.Detected = true
		result.Confidence = ecosystem.ConfidenceCertain
		result.Evidence = append(result.Evidence, "serverless.yml found (Serverless Framework)")
	}

	if fileutil.FileExists(projectRoot, "samconfig.toml") {
		result.Detected = true
		result.Confidence = ecosystem.ConfidenceCertain
		result.Evidence = append(result.Evidence, "samconfig.toml found (AWS SAM)")
	}

	// Check for template.yaml/yml with AWS:: resources.
	m.detectSAMTemplate(projectRoot, &result)

	// Terraform provider detection.
	tfProviders := cloudcommon.DetectTerraformProviders(projectRoot)
	if tfProviders["aws"] {
		result.Detected = true
		result.Confidence = ecosystem.ConfidenceCertain
		result.Evidence = append(result.Evidence, "Terraform provider \"aws\" detected")
	}

	// Weaker indicators -> ConfidenceProbable (only if not already Certain).
	if fileutil.FileExists(projectRoot, "buildspec.yml") {
		result.Detected = true
		if result.Confidence < ecosystem.ConfidenceCertain {
			result.Confidence = ecosystem.ConfidenceProbable
		}
		result.Evidence = append(result.Evidence, "buildspec.yml found (AWS CodeBuild)")
	}

	if fileutil.FileExists(projectRoot, "appspec.yml") {
		result.Detected = true
		if result.Confidence < ecosystem.ConfidenceCertain {
			result.Confidence = ecosystem.ConfidenceProbable
		}
		result.Evidence = append(result.Evidence, "appspec.yml found (AWS CodeDeploy)")
	}

	if fileutil.DirExists(projectRoot, ".aws-sam") {
		result.Detected = true
		if result.Confidence < ecosystem.ConfidenceCertain {
			result.Confidence = ecosystem.ConfidenceProbable
		}
		result.Evidence = append(result.Evidence, ".aws-sam/ directory found")
	}

	return result
}

// detectSAMTemplate checks for template.yaml or template.yml containing AWS::
// resource declarations, indicating a SAM or CloudFormation template.
func (m *Module) detectSAMTemplate(projectRoot string, result *ecosystem.DetectionResult) {
	for _, name := range []string{"template.yaml", "template.yml"} {
		path := filepath.Join(projectRoot, name)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if strings.Contains(string(data), "AWS::") {
			result.Detected = true
			result.Confidence = ecosystem.ConfidenceCertain
			result.Evidence = append(result.Evidence, name+" with AWS:: resources (SAM/CloudFormation)")
			return
		}
	}
}

// DevenvNixFragment returns the Nix code fragment to include in devenv.nix
// for AWS environment variable placeholders.
func (m *Module) DevenvNixFragment(_ ecosystem.ModuleConfig) (string, error) {
	return `  env.AWS_PROFILE = "PLACEHOLDER -- set to your SSO/vault profile name";
  env.AWS_DEFAULT_REGION = "PLACEHOLDER -- set to your default region (e.g. us-east-1)";`, nil
}

// DevenvPackages returns the Nix packages required for the AWS ecosystem.
// When aws_vault is enabled, aws-vault is included alongside awscli2.
func (m *Module) DevenvPackages(config ecosystem.ModuleConfig) []string {
	pkgs := []string{"awscli2"}
	if config.Extra("aws_vault", "") == "true" {
		pkgs = append(pkgs, "aws-vault")
	}
	return pkgs
}

// SecurityConfigs returns nil; the AWS module does not generate security
// configuration files.
func (m *Module) SecurityConfigs(_ ecosystem.ModuleConfig) []types.GeneratedFile { return nil }

// PreCommitHooks returns nil; the AWS module does not contribute pre-commit hooks.
func (m *Module) PreCommitHooks(_ ecosystem.ModuleConfig) []ecosystem.HookConfig { return nil }

// CICommands returns nil; the AWS module does not contribute CI commands.
func (m *Module) CICommands(_ ecosystem.ModuleConfig) []ecosystem.CICommand { return nil }

// PackageManagers returns nil; AWS CLI is not a package manager.
func (m *Module) PackageManagers() []ecosystem.PackageManagerInfo { return nil }

// VerificationCommands returns an empty set; AWS CLI has no build/test/lint steps.
func (m *Module) VerificationCommands(_ ecosystem.ModuleConfig) ecosystem.VerificationCommands {
	return ecosystem.VerificationCommands{}
}

// DenyRules returns Claude Code Bash deny-rule patterns that prevent the agent
// from invoking dangerous AWS CLI commands (IAM mutations, STS assume-role,
// credential configuration).
func (m *Module) DenyRules(_ ecosystem.ModuleConfig) []string {
	return cloudcommon.BashDenyRules(cloudcommon.AWS)
}

// ReadDenyRules returns Claude Code read-deny path patterns that prevent the
// agent from reading sensitive AWS credential files.
func (m *Module) ReadDenyRules(_ ecosystem.ModuleConfig) []string {
	return cloudcommon.ReadDenyPaths(cloudcommon.AWS)
}

// WizardFields returns wizard form fields for AWS configuration.
func (m *Module) WizardFields() []ecosystem.WizardField {
	return []ecosystem.WizardField{
		{
			Key:         "aws_default_region",
			Label:       "AWS Default Region",
			Description: "Default AWS region for CLI operations",
			Type:        ecosystem.FieldTypeInput,
			Default:     "us-east-1",
		},
		{
			Key:         "aws_vault",
			Label:       "Enable aws-vault",
			Description: "Include aws-vault for secure credential management",
			Type:        ecosystem.FieldTypeConfirm,
			Default:     "false",
		},
	}
}

// DoctorChecks returns health checks for the AWS ecosystem, verifying
// authentication and environment variable configuration.
func (m *Module) DoctorChecks(_ ecosystem.ModuleConfig) []ecosystem.DoctorCheck {
	return []ecosystem.DoctorCheck{
		{
			Name:        "aws-auth",
			Description: "AWS caller identity",
			Command:     "aws sts get-caller-identity",
			Timeout:     5,
			Provider:    "aws",
		},
		{
			Name:        "aws-profile",
			Description: "AWS_PROFILE environment variable",
			EnvCheck:    "AWS_PROFILE",
			Timeout:     0,
			Provider:    "aws",
		},
	}
}
