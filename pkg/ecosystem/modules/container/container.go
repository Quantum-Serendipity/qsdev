// Package container implements the container ecosystem module for qsdev. It
// detects Dockerfiles, Containerfiles, and compose manifests, generates
// devenv.nix fragments with container tooling, and provides hadolint linting,
// image scanning, and signing CI commands alongside Claude Code deny rules.
package container

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Compile-time interface compliance checks.
var _ ecosystem.EcosystemModule = (*Module)(nil)
var _ ecosystem.SecretDeclarer = (*Module)(nil)
var _ ecosystem.PackageProvider = (*Module)(nil)
var _ ecosystem.DevenvYamlInputProvider = (*Module)(nil)
var _ ecosystem.WizardFieldProvider = (*Module)(nil)
var _ ecosystem.DenyRuleProvider = (*Module)(nil)
var _ ecosystem.SASTModule = (*Module)(nil)

func init() {
	ecosystem.MustRegisterModule(&Module{})
}

// Module implements ecosystem.EcosystemModule for containers.
type Module struct{}

// Name returns the canonical ecosystem identifier.
func (m *Module) Name() string { return "container" }

// DisplayName returns the human-readable label.
func (m *Module) DisplayName() string { return "Containers" }

// Tier returns the implementation priority tier.
func (m *Module) Tier() int { return 1 }

// Detect scans projectRoot for container-related files and returns a
// DetectionResult with accumulated evidence.
func (m *Module) Detect(projectRoot string) ecosystem.DetectionResult {
	var (
		evidence   []string
		confidence = ecosystem.ConfidenceAbsent
		detected   bool
		hasCompose bool
	)

	// Certain indicators.
	if fileutil.FileExists(projectRoot, "Dockerfile") {
		evidence = append(evidence, "Dockerfile found")
		confidence = ecosystem.ConfidenceCertain
		detected = true
	}
	if fileutil.FileExists(projectRoot, "Containerfile") {
		evidence = append(evidence, "Containerfile found")
		confidence = ecosystem.ConfidenceCertain
		detected = true
	}

	// Probable indicators.
	for _, name := range []string{"docker-compose.yml", "docker-compose.yaml", "compose.yaml"} {
		if fileutil.FileExists(projectRoot, name) {
			evidence = append(evidence, name+" found")
			hasCompose = true
			if confidence < ecosystem.ConfidenceProbable {
				confidence = ecosystem.ConfidenceProbable
			}
			detected = true
		}
	}
	if fileutil.FileExists(projectRoot, ".dockerignore") {
		evidence = append(evidence, ".dockerignore found")
		if confidence < ecosystem.ConfidenceProbable {
			confidence = ecosystem.ConfidenceProbable
		}
		detected = true
	}
	if fileutil.FileExists(projectRoot, ".containerignore") {
		evidence = append(evidence, ".containerignore found")
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

	config := ecosystem.ModuleConfig{
		Extras: make(map[string]string),
	}
	if hasCompose {
		config.Extras["has_compose"] = "true"
	}

	return ecosystem.DetectionResult{
		Detected:        true,
		Confidence:      confidence,
		Evidence:        evidence,
		SuggestedConfig: config,
	}
}

// DevenvPackages returns the Nix packages required for the configured
// container runtime. Docker gets docker/hadolint/dive; Podman gets
// podman/podman-compose/buildah/skopeo/hadolint/dive.
func (m *Module) DevenvPackages(config ecosystem.ModuleConfig) []string {
	rt := config.Extra("container_runtime", "")
	switch rt {
	case "podman-rootless", "podman-rootful":
		return []string{"podman", "podman-compose", "buildah", "skopeo", "hadolint", "dive"}
	default: // "docker" or empty — backward compatible
		return []string{"docker", "hadolint", "dive"}
	}
}

// DevenvNixFragment returns the Nix code fragment to include in devenv.nix
// for container tooling. Podman runtimes set env.DOCKER_HOST; Docker runtimes
// produce an empty fragment (packages are provided via DevenvPackages).
func (m *Module) DevenvNixFragment(config ecosystem.ModuleConfig) (string, error) {
	rt := config.Extra("container_runtime", "")
	switch rt {
	case "podman-rootless", "podman-rootful":
		return "  env.DOCKER_HOST = \"unix://${XDG_RUNTIME_DIR}/podman/podman.sock\";\n", nil
	default:
		return "", nil
	}
}

// DevenvYamlInputs returns additional flake inputs for devenv.yaml.
// Podman on NixOS adds the quadlet-nix input for systemd integration.
func (m *Module) DevenvYamlInputs(config ecosystem.ModuleConfig) []ecosystem.DevenvInput {
	rt := config.Extra("container_runtime", "")
	osFamily := config.Extra("os_family", "")

	if (rt == "podman-rootless" || rt == "podman-rootful") && osFamily == "nixos" {
		return []ecosystem.DevenvInput{
			{
				URL: "github:SEIAROTg/quadlet-nix",
			},
		}
	}
	return nil
}

// hadolintConfig is the structured representation of .hadolint.yaml.
type hadolintConfig struct {
	TrustedRegistries []string `yaml:"trustedRegistries"`
	FailureThreshold  string   `yaml:"failure-threshold"`
}

// defaultTrustedRegistries lists registries trusted by default.
var defaultTrustedRegistries = []string{
	"docker.io",
	"gcr.io",
	"ghcr.io",
}

// hadolintHeader is prepended to the generated .hadolint.yaml file.
const hadolintHeader = `# Hadolint configuration — generated by qsdev.
# Requires: hadolint (any version). Available via nixpkgs.
# trustedRegistries: only images from these registries pass the
#   DL3026 (use only trusted base images) rule.
# failure-threshold: lint warnings at or above this severity fail the check.
`

// SecurityConfigs returns the generated .hadolint.yaml configuration file.
func (m *Module) SecurityConfigs(config ecosystem.ModuleConfig) []types.GeneratedFile {
	registries := defaultTrustedRegistries

	if custom := config.Extra("trusted_registries", ""); custom != "" {
		parts := strings.Split(custom, ",")
		parsed := make([]string, 0, len(parts))
		for _, p := range parts {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				parsed = append(parsed, trimmed)
			}
		}
		if len(parsed) > 0 {
			registries = parsed
		}
	}

	cfg := hadolintConfig{
		TrustedRegistries: registries,
		FailureThreshold:  "warning",
	}

	var buf bytes.Buffer
	buf.WriteString(hadolintHeader)

	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&cfg); err != nil {
		// Should never happen with simple struct; degrade gracefully.
		fmt.Fprintf(&buf, "# error encoding hadolint config: %v\n", err)
	}
	_ = enc.Close() //nolint:errcheck

	return []types.GeneratedFile{
		{
			Path:     ".hadolint.yaml",
			Content:  buf.Bytes(),
			Mode:     0o644,
			Strategy: types.Overwrite,
		},
	}
}

// PreCommitHooks returns pre-commit hook definitions for the container ecosystem.
func (m *Module) PreCommitHooks(_ ecosystem.ModuleConfig) []ecosystem.HookConfig {
	return []ecosystem.HookConfig{
		{
			ID:            "hadolint",
			Name:          "hadolint",
			Description:   "Lint Dockerfiles with hadolint",
			Entry:         "hadolint",
			Language:      "system",
			Types:         []string{"dockerfile"},
			Stages:        []string{"pre-commit"},
			PassFilenames: true,
			Files:         `(Dockerfile|Containerfile)`,
			BuiltIn:       false,
		},
	}
}

// DenyRules returns Claude Code deny-rule patterns for the container ecosystem.
// Prevents uncontrolled image pulls and, for Podman, blocks privileged
// containers and Docker socket mounts.
func (m *Module) DenyRules(config ecosystem.ModuleConfig) []string {
	rt := config.Extra("container_runtime", "")
	switch rt {
	case "podman-rootless", "podman-rootful":
		return []string{
			"Bash(docker run -v /var/run/docker.sock*)",
			"Bash(docker pull *)",
			"Bash(podman run --privileged *)",
		}
	default:
		return []string{
			"Bash(docker pull *)",
		}
	}
}

// CICommands returns CI pipeline commands for the container ecosystem.
// Commands are runtime-aware: Podman runtimes use `podman`, Docker uses `docker`.
func (m *Module) CICommands(config ecosystem.ModuleConfig) []ecosystem.CICommand {
	rt := config.Extra("container_runtime", "")
	imgCmd := "docker"
	buildCmd := "docker"
	if rt == "podman-rootless" || rt == "podman-rootful" {
		imgCmd = "podman"
		buildCmd = "podman"
	}

	return []ecosystem.CICommand{
		{
			Name:        "hadolint",
			Command:     "hadolint Dockerfile",
			Description: "Lint Dockerfile for best-practice violations",
			Phase:       ecosystem.CIPhaseScan,
		},
		{
			Name:        "container-build",
			Command:     buildCmd + " build --no-cache .",
			Description: "Build container image without layer cache to verify reproducibility",
			Phase:       ecosystem.CIPhaseScan,
		},
		{
			Name:        "syft-sbom",
			Command:     fmt.Sprintf("syft scan $(%s images -q | head -1) -o spdx-json=sbom.spdx.json", imgCmd),
			Description: "Generate SPDX SBOM from container image with Syft",
			Phase:       ecosystem.CIPhaseScan,
		},
		{
			Name:        "grype-scan",
			Command:     "grype sbom:sbom.spdx.json --fail-on high",
			Description: "Scan SBOM for vulnerabilities with Grype",
			Phase:       ecosystem.CIPhaseScan,
		},
		{
			Name:        "cosign-verify",
			Command:     fmt.Sprintf("cosign verify --key cosign.pub $(%s images --format '{{.Repository}}:{{.Tag}}' | head -1)", imgCmd),
			Description: "Verify container image signature with cosign",
			Phase:       ecosystem.CIPhaseScan,
		},
	}
}

// PackageManagers returns nil — containers are not a package manager.
func (m *Module) PackageManagers() []ecosystem.PackageManagerInfo {
	return nil
}

// WizardFields returns additional wizard form fields for container configuration.
func (m *Module) WizardFields() []ecosystem.WizardField {
	return []ecosystem.WizardField{
		{
			Key:         "trusted_registries",
			Label:       "Trusted container registries",
			Description: "Comma-separated list of container registries to trust in hadolint",
			Type:        ecosystem.FieldTypeInput,
			Default:     "docker.io,gcr.io,ghcr.io",
		},
	}
}

// VerificationCommands returns project verification commands for the container
// ecosystem. Build commands use the detected runtime.
func (m *Module) VerificationCommands(config ecosystem.ModuleConfig) ecosystem.VerificationCommands {
	rt := config.Extra("container_runtime", "")
	buildCmd := "docker build ."
	if rt == "podman-rootless" || rt == "podman-rootful" {
		buildCmd = "podman build ."
	}
	return ecosystem.VerificationCommands{
		Build: []string{buildCmd},
		Lint:  []string{"hadolint Dockerfile"},
	}
}

// SecretDeclarations returns the secrets required by a Docker project.
func (m *Module) SecretDeclarations(_ ecosystem.ModuleConfig) []ecosystem.SecretDecl {
	return []ecosystem.SecretDecl{
		{
			Name:        "DOCKER_REGISTRY_TOKEN",
			Description: "Authentication token for private Docker registry",
			Required:    true,
			Source:      "container",
		},
	}
}

// SemgrepRuleSets returns Semgrep rule set identifiers relevant to Docker projects.
func (m *Module) SemgrepRuleSets() []string {
	return []string{"p/dockerfile"}
}
