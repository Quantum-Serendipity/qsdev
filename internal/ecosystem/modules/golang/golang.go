// Package golang implements the Go ecosystem module for gdev-secure-devenv-bootstrap.
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

	"fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem"
	"fastcat.org/go/gdev-secure-devenv-bootstrap/pkg/types"
)

// Compile-time interface compliance check.
var _ ecosystem.EcosystemModule = (*Module)(nil)

func init() {
	if err := ecosystem.DefaultRegistry().Register(&Module{}); err != nil {
		panic(fmt.Sprintf("golang: failed to register ecosystem module: %v", err))
	}
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
	if !fileExists(modPath) {
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
func (m *Module) DevenvNixFragment(_ ecosystem.ModuleConfig) (string, error) {
	var b strings.Builder
	b.WriteString("  languages.go = {\n")
	b.WriteString("    enable = true;\n")
	b.WriteString("    package = pkgs.go;\n")
	b.WriteString("  };\n")
	b.WriteString("\n")
	b.WriteString("  # Enforce module-aware mode — prevents unvetted dependency additions\n")
	b.WriteString("  env.GOFLAGS = \"-mod=readonly\";\n")
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
		},
	}
}

// DenyRules returns Claude Code deny-rule patterns for the Go ecosystem.
// These prevent direct dependency modification outside of controlled workflows.
func (m *Module) DenyRules(_ ecosystem.ModuleConfig) []string {
	return []string{
		"Bash(go get *)",
		"Bash(go install *)",
	}
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

// fileExists reports whether a file at the given path exists and is not a directory.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
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
