// Package golang implements the Go ecosystem module for qsdev.
// It detects Go projects by scanning for go.mod, generates devenv.nix fragments
// with security-hardened environment variables, and provides pre-commit hooks,
// CI commands, deny rules, and wizard fields for the Go toolchain.
package golang

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Compile-time interface compliance checks.
var _ ecosystem.EcosystemModule = (*Module)(nil)
var _ ecosystem.PackageProvider = (*Module)(nil)

func init() {
	ecosystem.MustRegisterModule(&Module{})
}

// goVersionRe matches the "go X.Y" or "go X.Y.Z" directive in go.mod.
var goVersionRe = regexp.MustCompile(`^go\s+(\d+\.\d+(?:\.\d+)?)`)

// Module implements ecosystem.EcosystemModule for the Go programming language.
type Module struct{}

// Name returns the canonical ecosystem identifier.
func (m *Module) Name() string { return "go" }

// DisplayName returns the human-readable label.
func (m *Module) DisplayName() string { return "Go" }

// Tier returns the implementation priority tier.
func (m *Module) Tier() int { return 1 }

// Detect scans projectRoot for a go.mod file and extracts the Go version directive.
func (m *Module) Detect(projectRoot string) ecosystem.DetectionResult {
	modPath := filepath.Join(projectRoot, "go.mod")
	if !fileutil.FileExists(modPath) {
		return ecosystem.DetectionResult{
			Detected:   false,
			Confidence: ecosystem.ConfidenceAbsent,
		}
	}

	version := parseGoVersion(projectRoot)

	evidence := []string{"go.mod found"}
	if version != "" {
		evidence = append(evidence, fmt.Sprintf("go version %s", version))
	}

	return ecosystem.DetectionResult{
		Detected:   true,
		Confidence: ecosystem.ConfidenceCertain,
		Evidence:   evidence,
		SuggestedConfig: ecosystem.ModuleConfig{
			Version: version,
		},
	}
}

// DevenvNixFragment returns the Nix code fragment to include in devenv.nix
// for Go language support with supply-chain security hardening.
func (m *Module) DevenvNixFragment(config ecosystem.ModuleConfig) (string, error) {
	var b strings.Builder
	b.WriteString("  languages.go = {\n")
	b.WriteString("    enable = true;\n")
	fmt.Fprintf(&b, "    package = %s;\n", goVersionToNixPackage(config.Version))
	b.WriteString("  };\n")
	b.WriteString("\n")
	b.WriteString("  # Enforce module-aware mode — prevents unvetted dependency additions\n")
	b.WriteString("  env.GOFLAGS = \"-mod=readonly\";\n")
	if config.RegistryProxy != "" {
		fmt.Fprintf(&b, "  env.GOPROXY = \"%s,direct\";\n", config.RegistryProxy)
	}
	b.WriteString("  # Ensure all modules are verified via the Go checksum database\n")
	b.WriteString("  env.GONOSUMCHECK = \"\";\n")
	b.WriteString("  # Ensure all modules use the Go notary for transparency\n")
	b.WriteString("  env.GONOSUMDB = \"\";\n")
	return b.String(), nil
}

// DevenvYamlInputs returns additional flake inputs for devenv.yaml.
// Go does not require any additional inputs.
func (m *Module) DevenvYamlInputs(_ ecosystem.ModuleConfig) []ecosystem.DevenvInput {
	return nil
}

// SecurityConfigs returns generated security configuration files.
// Go's security settings are handled via environment variables in DevenvNixFragment.
func (m *Module) SecurityConfigs(_ ecosystem.ModuleConfig) []types.GeneratedFile {
	return nil
}

// PreCommitHooks returns pre-commit hook definitions for the Go ecosystem.
func (m *Module) PreCommitHooks(_ ecosystem.ModuleConfig) []ecosystem.HookConfig {
	return []ecosystem.HookConfig{
		{
			ID:            "gofmt",
			Name:          "gofmt",
			Description:   "Format Go source code with gofmt",
			Entry:         "gofmt -l -w",
			Language:      "system",
			Types:         []string{"go"},
			Stages:        []string{"pre-commit"},
			PassFilenames: true,
			BuiltIn:       true,
		},
		{
			ID:            "govet",
			Name:          "govet",
			Description:   "Run go vet to detect suspicious constructs",
			Entry:         "go vet ./...",
			Language:      "system",
			Types:         []string{"go"},
			Stages:        []string{"pre-commit"},
			PassFilenames: false,
			BuiltIn:       true,
		},
		{
			ID:            "staticcheck",
			Name:          "staticcheck",
			Description:   "Run staticcheck for advanced static analysis",
			Entry:         "staticcheck ./...",
			Language:      "system",
			Types:         []string{"go"},
			Stages:        []string{"pre-commit"},
			PassFilenames: false,
			BuiltIn:       false,
			NixPackage:    "go-tools",
		},
		{
			ID:            "govulncheck",
			Name:          "govulncheck",
			Description:   "Check for known vulnerabilities in Go dependencies",
			Entry:         "govulncheck ./...",
			Language:      "system",
			Types:         []string{"go"},
			Stages:        []string{"pre-commit"},
			PassFilenames: false,
			BuiltIn:       false,
			NixPackage:    "govulncheck",
		},
	}
}

// DenyRules returns Claude Code deny-rule patterns for the Go ecosystem.
// These prevent direct dependency modification outside of controlled workflows.
func (m *Module) DenyRules(_ ecosystem.ModuleConfig) []string {
	// Package install commands (go get/install) are handled by base ask rules +
	// package-guard hook. Return empty — no Go-specific hard-deny patterns.
	return nil
}

// CICommands returns CI pipeline commands for the Go ecosystem.
func (m *Module) CICommands(_ ecosystem.ModuleConfig) []ecosystem.CICommand {
	return []ecosystem.CICommand{
		{
			Name:        "go-mod-download",
			Command:     "go mod download",
			Description: "Download Go module dependencies",
			Phase:       ecosystem.CIPhaseInstall,
		},
		{
			Name:        "go-mod-verify",
			Command:     "go mod verify",
			Description: "Verify Go module checksums against go.sum",
			Phase:       ecosystem.CIPhaseTest,
		},
		{
			Name:        "govulncheck",
			Command:     "govulncheck ./...",
			Description: "Scan Go dependencies for known vulnerabilities",
			Phase:       ecosystem.CIPhaseScan,
		},
	}
}

// PackageManagers returns metadata about Go's module system.
func (m *Module) PackageManagers() []ecosystem.PackageManagerInfo {
	return []ecosystem.PackageManagerInfo{
		{
			Name:                 "go modules",
			LockFile:             "go.sum",
			FrozenInstallCommand: "go mod download",
			AuditCommand:         "govulncheck ./...",
			AgeGatingSupport:     false,
		},
	}
}

// WizardFields returns additional wizard form fields for Go configuration.
func (m *Module) WizardFields() []ecosystem.WizardField {
	return []ecosystem.WizardField{
		{
			Key:         "go_version",
			Label:       "Go version",
			Description: "Specify the Go version to use (e.g. 1.22)",
			Type:        ecosystem.FieldTypeInput,
			Default:     "",
		},
	}
}

// VerificationCommands returns build/test/lint/format commands for Go projects.
func (m *Module) VerificationCommands(_ ecosystem.ModuleConfig) ecosystem.VerificationCommands {
	return ecosystem.VerificationCommands{
		Build:  []string{"go build ./..."},
		Test:   []string{"go test ./..."},
		Lint:   []string{"go vet ./...", "golangci-lint run"},
		Format: []string{"gofmt -l ."},
	}
}

// ManifestFiles returns manifest file metadata for Go projects.
func (m *Module) ManifestFiles(_ ecosystem.ModuleConfig) []ecosystem.ManifestFileInfo {
	return []ecosystem.ManifestFileInfo{
		{
			Path:           "go.mod",
			Ecosystem:      "go",
			VSSupported:    false,
			LockFile:       "go.sum",
			LockFilePolicy: ecosystem.LockFilePolicyRecommended,
		},
	}
}

// DevenvPackages returns standard Go development tool packages.
func (m *Module) DevenvPackages(_ ecosystem.ModuleConfig) []string {
	return []string{"gopls", "golangci-lint", "delve", "goreleaser"}
}

// goVersionToNixPackage maps a Go version string to the corresponding Nix package
// attribute. For example, "1.24.1" maps to "pkgs.go_1_24". If the version is empty
// or cannot be parsed into at least major.minor components, "pkgs.go" (latest) is returned.
func goVersionToNixPackage(version string) string {
	if version == "" {
		return "pkgs.go"
	}
	// Extract major.minor (e.g. "1.24.1" -> "1.24", "1.23" -> "1.23")
	parts := strings.SplitN(version, ".", 3)
	if len(parts) < 2 {
		return "pkgs.go"
	}
	return fmt.Sprintf("pkgs.go_%s_%s", parts[0], parts[1])
}

// parseGoVersion reads go.mod in projectRoot and extracts the Go version
// from the "go X.Y" or "go X.Y.Z" directive. Returns an empty string
// if the directive is not found or the file cannot be read.
func parseGoVersion(projectRoot string) string {
	modPath := filepath.Join(projectRoot, "go.mod")

	f, err := os.Open(modPath)
	if err != nil {
		return ""
	}
	defer f.Close() //nolint:errcheck // best-effort read

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if m := goVersionRe.FindStringSubmatch(scanner.Text()); m != nil {
			return m[1]
		}
	}
	return ""
}

// SemgrepRuleSets returns Semgrep rule set identifiers relevant to Go projects.
func (m *Module) SemgrepRuleSets() []string {
	return []string{"p/golang", "p/owasp-top-ten"}
}
