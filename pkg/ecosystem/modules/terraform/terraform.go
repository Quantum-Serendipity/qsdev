// Package terraform implements the Terraform/OpenTofu ecosystem module for
// qsdev. It detects Terraform and OpenTofu projects by
// scanning for .tf/.tofu configuration files and lock files, then
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
	"github.com/Quantum-Serendipity/qsdev/pkg/denyutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/cloudcommon"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Compile-time interface compliance checks.
var _ ecosystem.EcosystemModule = (*Module)(nil)
var _ ecosystem.SecretDeclarer = (*Module)(nil)
var _ ecosystem.DenyRuleProvider = (*Module)(nil)
var _ ecosystem.ReadDenyRuleProvider = (*Module)(nil)
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

// Detect scans projectRoot and its subdirectories (up to
// ecosystem.ProjectScanDepth levels, the same walk cloudcommon uses for
// provider detection) for Terraform/OpenTofu configuration: .tf/.tf.json and
// .tofu/.tofu.json files are definitive, a .terraform.lock.hcl alone is
// probable. OpenTofu-only .tofu files select the opentofu variant; OpenTofu
// has no marker directory of its own. The directories holding configuration
// are recorded in Extras[ExtraConfigDirs] so hooks can target them.
func (m *Module) Detect(projectRoot string) ecosystem.DetectionResult {
	result := ecosystem.DetectionResult{
		SuggestedConfig: ecosystem.ModuleConfig{
			Extras: map[string]string{"variant": "terraform"},
		},
	}

	var tfFound, tfJSONFound, tofuFound, lockFound bool
	var dirs []string
	ecosystem.WalkProjectFiles(projectRoot, ecosystem.ProjectScanDepth, func(path string) bool {
		name := filepath.Base(path)
		switch {
		case name == ".terraform.lock.hcl":
			lockFound = true
			return true
		case strings.HasSuffix(name, ".tofu"), strings.HasSuffix(name, ".tofu.json"):
			tofuFound = true
		case strings.HasSuffix(name, ".tf"):
			tfFound = true
		case strings.HasSuffix(name, ".tf.json"):
			tfJSONFound = true
		default:
			return true
		}
		if rel, err := filepath.Rel(projectRoot, filepath.Dir(path)); err == nil {
			dirs = append(dirs, filepath.ToSlash(rel))
		}
		return true
	})

	for _, ev := range []struct {
		found bool
		text  string
	}{
		{tfFound, "*.tf files found"},
		{tfJSONFound, "*.tf.json files found"},
		{tofuFound, "*.tofu files found"},
	} {
		if ev.found {
			result.Detected = true
			result.Confidence = ecosystem.ConfidenceCertain
			result.Evidence = append(result.Evidence, ev.text)
		}
	}
	if tofuFound {
		result.SuggestedConfig.Extras["variant"] = "opentofu"
	}
	if lockFound {
		result.Evidence = append(result.Evidence, ".terraform.lock.hcl found")
		if !result.Detected {
			result.Detected = true
			result.Confidence = ecosystem.ConfidenceProbable
		}
	}

	if !result.Detected {
		return result
	}
	if configDirs := rootModuleDirs(ecosystem.ShellSafeDirs(dirs)); len(configDirs) > 0 && !slices.Equal(configDirs, []string{"."}) {
		result.SuggestedConfig.Extras[ExtraConfigDirs] = strings.Join(configDirs, ",")
		result.Evidence = append(result.Evidence, "configuration in: "+strings.Join(configDirs, ", "))
	}
	if providers := cloudcommon.DetectTerraformProviders(projectRoot); len(providers) > 0 {
		names := slices.Sorted(maps.Keys(providers))
		result.SuggestedConfig.Extras[ExtraCloudProviders] = strings.Join(names, ",")
		result.Evidence = append(result.Evidence, "cloud providers: "+strings.Join(names, ", "))
	}

	return result
}

// ExtraConfigDirs is the ModuleConfig.Extras key holding the comma-separated
// root-module directories (relative to the project root, "." for the root)
// that contain Terraform/OpenTofu configuration; child modules are left out
// (see rootModuleDirs). It is unset when the configuration lives only in the
// root.
const ExtraConfigDirs = "config_dirs"

// rootModuleDirs drops child-module directories (any path with a "modules"
// segment, the standard module structure) from dirs: they are validated
// through the root modules that call them, and running `init` in them would
// leave stray .terraform/ directories and lock files behind. A repository
// that holds only modules keeps all its directories.
func rootModuleDirs(dirs []string) []string {
	roots := slices.DeleteFunc(slices.Clone(dirs), func(d string) bool {
		return slices.Contains(strings.Split(d, "/"), "modules")
	})
	if len(roots) == 0 {
		return dirs
	}
	return roots
}

// configDirs returns the recorded configuration directories, or nil when the
// configuration lives only in the project root.
func configDirs(config ecosystem.ModuleConfig) []string {
	raw := config.Extra(ExtraConfigDirs, "")
	if raw == "" {
		return nil
	}
	return ecosystem.ShellSafeDirs(strings.Split(raw, ","))
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
	dirs := configDirs(config)
	tflintEntry := "tflint"
	if len(dirs) > 0 {
		// Configuration below the root: plain tflint inspects only the
		// working directory.
		tflintEntry = "tflint --recursive"
	}

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
			Files:         configFilesPattern,
			Stages:        []string{"pre-commit"},
			PassFilenames: false,
			BuiltIn:       false,
			NixPackage:    nixPkg,
		},
		validateHook(variant, dirs),
		{
			ID:            "tflint",
			Name:          "tflint",
			Description:   "Lint Terraform configurations with tflint",
			Entry:         tflintEntry,
			Language:      "system",
			Files:         configFilesPattern,
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
			Files:         configFilesPattern,
			Stages:        []string{"pre-commit"},
			PassFilenames: false,
			BuiltIn:       false,
			NixPackage:    "tfsec",
		},
	}
}

// configFilesPattern selects the files the Terraform hooks run on. The
// identify "terraform" type tag covers only .tf/.tfvars, so .tofu files and
// the JSON syntax would never trigger the hooks; an explicit pattern does.
const configFilesPattern = `\.(tf|tofu|tfvars)(\.json)?$`

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
//
// When the configuration lives below the root (dirs, see ExtraConfigDirs)
// each directory is initialized and validated through -chdir; validating the
// root would check an empty configuration.
func validateHook(variant string, dirs []string) ecosystem.HookConfig {
	binary := binaryName(variant)
	entry := fmt.Sprintf("sh -c '%[1]s init -backend=false -input=false >/dev/null && %[1]s validate'", binary)
	if len(dirs) > 0 {
		// $d is unquoted: the entry is embedded verbatim in a Nix string, so
		// it must not contain double quotes, and ShellSafeDirs guarantees the
		// names need no quoting.
		entry = fmt.Sprintf(`sh -c 'for d in %[2]s; do %[1]s -chdir=$d init -backend=false -input=false >/dev/null && %[1]s -chdir=$d validate || exit 1; done'`,
			binary, strings.Join(dirs, " "))
	}
	return ecosystem.HookConfig{
		ID:            "terraform-validate",
		Name:          "terraform-validate",
		Description:   fmt.Sprintf("Initialize (without backend) and validate %s configuration", variant),
		Entry:         entry,
		Language:      "system",
		Files:         configFilesPattern,
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

// deniedSubcommands are the Terraform/OpenTofu subcommands the agent must not
// run: ones that change real infrastructure or state (apply, destroy, import,
// state push/rm/mv, force-unlock), fetch providers or modules past the lock
// file (init, get, providers), and ones that print state secrets in plain
// text (state pull, output, show -json).
var deniedSubcommands = []string{
	"init",
	"apply",
	"destroy",
	"get",
	"import",
	"force-unlock",
	"providers",
	"state pull",
	"state push",
	"state rm",
	"state mv",
	"output",
	"show *-json*",
}

// iacBinaries are the CLIs the deny rules cover. Both are always covered,
// whatever the variant: either binary may be installed and both accept the
// same subcommands.
var iacBinaries = []string{"terraform", "tofu"}

// DenyRules returns Claude Code deny-rule patterns for Terraform/OpenTofu.
// Each subcommand in deniedSubcommands is denied for both binaries, plain and
// after global options such as -chdir=DIR (denyutil.SubcommandRules).
func (m *Module) DenyRules(_ ecosystem.ModuleConfig) []string {
	var rules []string
	for _, bin := range iacBinaries {
		rules = append(rules, denyutil.SubcommandRules(bin, deniedSubcommands...)...)
	}
	return rules
}

// ReadDenyRules returns the credential stores the agent's Read tool must not
// open: state files (plaintext resource secrets and backend credentials,
// including .terraform/terraform.tfstate), variable files, and the
// HCP Terraform / registry token `terraform login` writes.
func (m *Module) ReadDenyRules(_ ecosystem.ModuleConfig) []string {
	return []string{
		"**/*.tfstate*",
		"**/*.tfvars",
		"**/*.tfvars.json",
		"~/.terraform.d/credentials.tfrc.json",
	}
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
			Key:         "variant",
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
			Key:         types.SettingVersion,
			Label:       "Terraform version",
			Description: "The Terraform version to pin (OpenTofu uses the nixpkgs release); leave empty for the nixpkgs default",
			Type:        ecosystem.FieldTypeInput,
			Placeholder: "1.8.0",
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

// SemgrepRuleSets returns Semgrep rule set identifiers relevant to Terraform
// projects. p/terraform already includes the AWS rules; there is no
// p/terraform-aws registry ruleset.
func (m *Module) SemgrepRuleSets() []string {
	return []string{"p/terraform"}
}
