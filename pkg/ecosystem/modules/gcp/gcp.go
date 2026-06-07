// Package gcp implements the Google Cloud Platform ecosystem module for qsdev.
// It detects GCP projects by scanning for gcloud configuration, Terraform
// provider blocks, Cloud Build manifests, Firebase configs, and App Engine
// descriptors, then generates devenv.nix fragments with google-cloud-sdk,
// deny rules, read-deny paths, and doctor checks for a hardened GCP
// development environment.
package gcp

import (
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
var _ ecosystem.DoctorCheckProvider = (*Module)(nil)

func init() {
	ecosystem.MustRegisterModule(&Module{})
}

// Module is the stateless GCP ecosystem module.
type Module struct{}

// Name returns the canonical module identifier.
func (m *Module) Name() string { return "gcp" }

// DisplayName returns the human-readable label.
func (m *Module) DisplayName() string { return "Google Cloud CLI" }

// Tier returns the implementation priority tier (2 = standard).
func (m *Module) Tier() int { return 2 }

// Detect scans projectRoot for GCP ecosystem indicators.
func (m *Module) Detect(projectRoot string) ecosystem.DetectionResult {
	var (
		evidence   []string
		confidence = ecosystem.ConfidenceAbsent
		detected   bool
	)

	// Terraform provider "google" → Certain.
	providers := cloudcommon.DetectTerraformProviders(projectRoot)
	if providers["google"] || providers["google-beta"] {
		evidence = append(evidence, "Terraform provider \"google\" found")
		confidence = ecosystem.ConfidenceCertain
		detected = true
	}

	// cloudbuild.yaml → Certain (Cloud Build).
	if fileutil.FileExists(projectRoot, "cloudbuild.yaml") {
		evidence = append(evidence, "cloudbuild.yaml found")
		confidence = ecosystem.ConfidenceCertain
		detected = true
	}

	// firebase.json → Certain (Firebase).
	if fileutil.FileExists(projectRoot, "firebase.json") {
		evidence = append(evidence, "firebase.json found")
		confidence = ecosystem.ConfidenceCertain
		detected = true
	}

	// app.yaml → Certain (App Engine).
	if fileutil.FileExists(projectRoot, "app.yaml") {
		evidence = append(evidence, "app.yaml found")
		confidence = ecosystem.ConfidenceCertain
		detected = true
	}

	// .gcloudignore → Probable.
	if fileutil.FileExists(projectRoot, ".gcloudignore") {
		evidence = append(evidence, ".gcloudignore found")
		if confidence < ecosystem.ConfidenceProbable {
			confidence = ecosystem.ConfidenceProbable
		}
		detected = true
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
// for GCP environment variables.
func (m *Module) DevenvNixFragment(config ecosystem.ModuleConfig) (string, error) {
	var b strings.Builder
	b.WriteString("  env.CLOUDSDK_ACTIVE_CONFIG_NAME = \"PLACEHOLDER -- set to your gcloud config name\";\n")
	b.WriteString("  env.CLOUDSDK_CORE_PROJECT = \"PLACEHOLDER -- set to your GCP project ID\";\n")
	b.WriteString("  env.GOOGLE_CLOUD_PROJECT = \"PLACEHOLDER -- same as CLOUDSDK_CORE_PROJECT\";\n")

	if config.Extra("k8s", "") == "true" {
		b.WriteString("  # GKE auth: google-cloud-sdk is installed with gke-gcloud-auth-plugin component\n")
	}

	return b.String(), nil
}

// DevenvPackages returns the Nix packages required for the GCP ecosystem.
func (m *Module) DevenvPackages(_ ecosystem.ModuleConfig) []string {
	return []string{"google-cloud-sdk"}
}

// DenyRules returns Claude Code bash deny-rule patterns for GCP.
func (m *Module) DenyRules(_ ecosystem.ModuleConfig) []string {
	return cloudcommon.BashDenyRules(cloudcommon.GCP)
}

// ReadDenyRules returns Claude Code read-deny path patterns for GCP.
func (m *Module) ReadDenyRules(_ ecosystem.ModuleConfig) []string {
	return cloudcommon.ReadDenyPaths(cloudcommon.GCP)
}

// DoctorChecks returns health checks for the GCP ecosystem.
func (m *Module) DoctorChecks(_ ecosystem.ModuleConfig) []ecosystem.DoctorCheck {
	return []ecosystem.DoctorCheck{
		{
			Name:        "gcp-auth",
			Description: "GCP authentication",
			Command:     "gcloud auth print-access-token",
			Timeout:     5,
			Provider:    "gcp",
		},
		{
			Name:        "gcp-config",
			Description: "CLOUDSDK_ACTIVE_CONFIG_NAME",
			EnvCheck:    "CLOUDSDK_ACTIVE_CONFIG_NAME",
			Provider:    "gcp",
		},
	}
}

// SecurityConfigs returns nil; GCP does not generate security config files
// at the module level.
func (m *Module) SecurityConfigs(_ ecosystem.ModuleConfig) []types.GeneratedFile {
	return nil
}

// PreCommitHooks returns nil; GCP does not contribute pre-commit hooks.
func (m *Module) PreCommitHooks(_ ecosystem.ModuleConfig) []ecosystem.HookConfig {
	return nil
}

// CICommands returns nil; GCP does not contribute CI commands at the module level.
func (m *Module) CICommands(_ ecosystem.ModuleConfig) []ecosystem.CICommand {
	return nil
}

// PackageManagers returns nil; GCP is not a package manager.
func (m *Module) PackageManagers() []ecosystem.PackageManagerInfo {
	return nil
}

// VerificationCommands returns an empty set; GCP does not define
// standard verification commands.
func (m *Module) VerificationCommands(_ ecosystem.ModuleConfig) ecosystem.VerificationCommands {
	return ecosystem.VerificationCommands{}
}
