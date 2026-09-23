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
	"strconv"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Compile-time interface compliance checks.
var _ ecosystem.EcosystemModule = (*Module)(nil)
var _ ecosystem.PackageProvider = (*Module)(nil)
var _ ecosystem.WizardFieldProvider = (*Module)(nil)
var _ ecosystem.ManifestFileProvider = (*Module)(nil)
var _ ecosystem.SASTModule = (*Module)(nil)

func init() {
	ecosystem.MustRegisterModule(&Module{})
}

// goVersionRe matches the "go X.Y" or "go X.Y.Z" directive in go.mod.
var goVersionRe = regexp.MustCompile(`^go\s+(\d+\.\d+(?:\.\d+)?)`)

// goReleaseRe parses a Go 1.x release version ("1.24" or "1.24.1") and
// captures the minor version.
var goReleaseRe = regexp.MustCompile(`^1\.(\d+)(?:\.\d+)?$`)

// supportedGoMinors lists the Go 1.x minor versions that have a stable
// go_1_<minor> attribute in nixpkgs, in ascending order. goVersionToNixPackage
// never emits an attribute outside this list.
var supportedGoMinors = []int{25, 26}

// Module implements ecosystem.EcosystemModule for the Go programming language.
type Module struct{}

// Name returns the canonical ecosystem identifier.
func (m *Module) Name() string { return ecosystem.NameGo }

// DisplayName returns the human-readable label.
func (m *Module) DisplayName() string { return "Go" }

// Tier returns the implementation priority tier.
func (m *Module) Tier() int { return 1 }

// Detect scans projectRoot for a go.mod file and extracts the Go version directive.
func (m *Module) Detect(projectRoot string) ecosystem.DetectionResult {
	modPath := filepath.Join(projectRoot, "go.mod")
	if !fileutil.FileExists(modPath) {
		return ecosystem.DetectionAbsent()
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
	envVars := []ecosystem.NixEnvVar{
		{
			Key:     "GOFLAGS",
			Value:   `"-mod=readonly"`,
			Comment: "Fail instead of silently updating go.mod/go.sum — prevents unvetted dependency additions",
		},
	}
	if config.RegistryProxy != "" {
		envVars = append(envVars, ecosystem.NixEnvVar{
			Key:   "GOPROXY",
			Value: fmt.Sprintf(`"%s,direct"`, ecosystem.NixEscapeString(config.RegistryProxy)),
		})
	}
	// GOSUMDB is the variable that controls checksum verification; pinning it
	// overrides an inherited GOSUMDB=off. Go fetches the checksum database
	// through GOPROXY when the proxy supports it. Modules matched by
	// GOPRIVATE/GONOSUMDB remain exempt, as they must be for private code.
	envVars = append(envVars, ecosystem.NixEnvVar{
		Key:     "GOSUMDB",
		Value:   `"sum.golang.org"`,
		Comment: "Verify all public modules against the Go checksum database",
	})

	pkg, note := goVersionToNixPackage(config.Version)
	fragment := ecosystem.BuildLanguageFragment(ecosystem.NixLangConfig{
		EnablePath: "languages.go",
		Properties: []ecosystem.NixProperty{
			{Key: "package", Value: pkg},
		},
		EnvVars: envVars,
	})
	if note != "" {
		fragment = "  # " + note + "\n" + fragment
	}
	return fragment, nil
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
			Description: "Specify the Go version to use (e.g. 1.25)",
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

// goVersionToNixPackage maps a Go version string to a nixpkgs attribute.
// The go.mod directive is a minimum, so the requested minor version maps to
// the oldest supported go_1_<minor> attribute at or above it (for example,
// "1.22" maps to "pkgs.go_1_25" and "1.26.1" to "pkgs.go_1_26"). An empty
// version maps to "pkgs.go" (latest). A version that is not a Go 1.x release,
// or is newer than every supported attribute, also maps to "pkgs.go". note is
// non-empty whenever the result differs from the requested version, so the
// substitution can be surfaced instead of emitting a nonexistent attribute.
func goVersionToNixPackage(version string) (pkg, note string) {
	if version == "" {
		return "pkgs.go", ""
	}
	m := goReleaseRe.FindStringSubmatch(version)
	if m == nil {
		return "pkgs.go", fmt.Sprintf("unrecognized Go version %q; using pkgs.go (latest)", version)
	}
	minor, err := strconv.Atoi(m[1])
	if err != nil {
		return "pkgs.go", fmt.Sprintf("unrecognized Go version %q; using pkgs.go (latest)", version)
	}
	for _, v := range supportedGoMinors {
		if v < minor {
			continue
		}
		attr := fmt.Sprintf("pkgs.go_1_%d", v)
		if v == minor {
			return attr, ""
		}
		return attr, fmt.Sprintf("go %s is not packaged in nixpkgs; using %s (Go is backward compatible)", version, attr)
	}
	return "pkgs.go", fmt.Sprintf("go %s is newer than any pinned toolchain; using pkgs.go (latest)", version)
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
