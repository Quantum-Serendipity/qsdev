// Package gcp implements the Google Cloud Platform ecosystem module for qsdev.
// It detects GCP projects by scanning for gcloud configuration, Terraform
// provider blocks, Cloud Build manifests, Firebase configs, and App Engine
// descriptors, then generates devenv.nix fragments with google-cloud-sdk,
// deny rules, read-deny paths, and doctor checks for a hardened GCP
// development environment.
package gcp

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
var _ ecosystem.PackageExprProvider = (*Module)(nil)
var _ ecosystem.DoctorCheckProvider = (*Module)(nil)

func init() {
	ecosystem.MustRegisterModule(&Module{})
}

// Module is the stateless GCP ecosystem module.
type Module struct{}

// Name returns the canonical module identifier.
func (m *Module) Name() string { return ecosystem.NameGCP }

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

	// app.yaml → Certain (App Engine). The name is generic (Kubernetes
	// manifests, app configs), so it only counts as an App Engine descriptor
	// when it declares a top-level runtime.
	if isAppEngineDescriptor(filepath.Join(projectRoot, "app.yaml")) {
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
		return ecosystem.DetectionAbsent()
	}

	return ecosystem.DetectionResult{
		Detected:   true,
		Confidence: confidence,
		Evidence:   evidence,
	}
}

// envHints are the per-project variables gcloud and the Google Cloud client
// libraries read to select a configuration and project.
var envHints = []cloudcommon.EnvVarHint{
	{Name: "CLOUDSDK_ACTIVE_CONFIG_NAME", Description: "gcloud configuration name"},
	{Name: "CLOUDSDK_CORE_PROJECT", Description: "GCP project ID"},
	{Name: "GOOGLE_CLOUD_PROJECT", Description: "GCP project ID"},
}

// gkeAuthPluginExpr is google-cloud-sdk with the gke-gcloud-auth-plugin
// component, which kubectl needs to authenticate against GKE clusters. The
// plain google-cloud-sdk package does not ship the plugin binary.
const gkeAuthPluginExpr = "(pkgs.google-cloud-sdk.withExtraComponents " +
	"[ pkgs.google-cloud-sdk.components.gke-gcloud-auth-plugin ])"

// DevenvNixFragment returns the Nix code fragment to include in devenv.nix
// for GCP. It documents the per-project environment variables without setting
// them (see cloudcommon.EnvGuidanceFragment) and, with
// cloud.isolate_cli_config, sets CLOUDSDK_CONFIG to a per-project directory.
func (m *Module) DevenvNixFragment(config ecosystem.ModuleConfig) (string, error) {
	fragment := cloudcommon.EnvGuidanceFragment("Google Cloud", envHints)
	if config.IsolateCLIConfig {
		fragment += "\n" + cloudcommon.IsolatedCLIConfigFragment(cloudcommon.GCP)
	}
	return fragment, nil
}

// DevenvPackages returns the Nix packages required for the GCP ecosystem.
// With the k8s extra, the SDK is provided by DevenvPackageExprs instead, so
// that two conflicting google-cloud-sdk builds never land in one profile.
func (m *Module) DevenvPackages(config ecosystem.ModuleConfig) []string {
	if needsGKEAuth(config) {
		return nil
	}
	return []string{"google-cloud-sdk"}
}

// DevenvPackageExprs returns google-cloud-sdk with the gke-gcloud-auth-plugin
// component when the k8s extra is set (GKE clusters), and nothing otherwise.
func (m *Module) DevenvPackageExprs(config ecosystem.ModuleConfig) []string {
	if needsGKEAuth(config) {
		return []string{gkeAuthPluginExpr}
	}
	return nil
}

// needsGKEAuth reports whether the project targets Kubernetes on GKE.
func needsGKEAuth(config ecosystem.ModuleConfig) bool {
	return config.Extra("k8s", "") == "true"
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

// isAppEngineDescriptor reports whether path is an App Engine app.yaml: a
// YAML file with a top-level "runtime:" key.
func isAppEngineDescriptor(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	for line := range strings.Lines(string(data)) {
		if strings.HasPrefix(line, "runtime:") {
			return true
		}
	}
	return false
}
