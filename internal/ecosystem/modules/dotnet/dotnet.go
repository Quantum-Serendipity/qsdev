// Package dotnet implements the C#/.NET ecosystem module for gdev-secure-devenv-bootstrap.
// It detects .NET projects via *.csproj, *.fsproj, *.sln, Directory.Build.props, and
// global.json, then generates devenv.nix fragments, security configs (nuget.config and
// Directory.Build.props), pre-commit hooks, deny rules, and CI commands for a hardened
// .NET development environment.
package dotnet

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/fileutil"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

// Compile-time interface compliance check.
var _ ecosystem.EcosystemModule = (*Module)(nil)

// Module is the stateless C#/.NET ecosystem module.
type Module struct{}

func init() {
	ecosystem.RegisterModule(&Module{})
}

// Name returns the canonical module identifier.
func (m *Module) Name() string { return "dotnet" }

// DisplayName returns the human-readable label.
func (m *Module) DisplayName() string { return "C#/.NET" }

// Tier returns the implementation priority tier (1 = core).
func (m *Module) Tier() int { return 1 }

// Detect scans projectRoot for .NET ecosystem indicators.
func (m *Module) Detect(projectRoot string) ecosystem.DetectionResult {
	result := ecosystem.DetectionResult{
		SuggestedConfig: ecosystem.ModuleConfig{
			PackageManager: "nuget",
			Extras:         make(map[string]string),
		},
	}

	// Check for *.csproj files.
	csprojMatches, _ := filepath.Glob(filepath.Join(projectRoot, "*.csproj"))
	if len(csprojMatches) > 0 {
		result.Detected = true
		result.Confidence = ecosystem.ConfidenceCertain
		result.Evidence = append(result.Evidence, "*.csproj")
	}

	// Check for *.fsproj files.
	fsprojMatches, _ := filepath.Glob(filepath.Join(projectRoot, "*.fsproj"))
	if len(fsprojMatches) > 0 {
		result.Detected = true
		result.Confidence = ecosystem.ConfidenceCertain
		result.Evidence = append(result.Evidence, "*.fsproj")
		result.SuggestedConfig.Extras["has_fsharp"] = "true"
	}

	// Check for *.sln files.
	slnMatches, _ := filepath.Glob(filepath.Join(projectRoot, "*.sln"))
	if len(slnMatches) > 0 {
		result.Detected = true
		result.Confidence = ecosystem.ConfidenceCertain
		result.Evidence = append(result.Evidence, "*.sln")
	}

	// Check for Directory.Build.props.
	if fileutil.FileExists(projectRoot, "Directory.Build.props") {
		result.Detected = true
		result.Confidence = ecosystem.ConfidenceCertain
		result.Evidence = append(result.Evidence, "Directory.Build.props")
	}

	// Check global.json for SDK version.
	if sdkVersion := parseGlobalJSON(filepath.Join(projectRoot, "global.json")); sdkVersion != "" {
		result.Detected = true
		result.Confidence = ecosystem.ConfidenceCertain
		result.Evidence = append(result.Evidence, "global.json")
		result.SuggestedConfig.Version = sdkVersion
	}

	return result
}

// DevenvNixFragment returns a Nix fragment that enables .NET in devenv.sh.
func (m *Module) DevenvNixFragment(config ecosystem.ModuleConfig) (string, error) {
	pkg := sdkVersionToNixPackage(config.Version)

	return fmt.Sprintf(`  languages.dotnet = {
    enable = true;
    package = pkgs.%s;
  };`, pkg), nil
}

// DevenvYamlInputs returns additional flake inputs for devenv.yaml (none for .NET).
func (m *Module) DevenvYamlInputs(_ ecosystem.ModuleConfig) []ecosystem.DevenvInput {
	return nil
}

// SecurityConfigs returns security-hardened configuration files for .NET.
func (m *Module) SecurityConfigs(config ecosystem.ModuleConfig) []types.GeneratedFile {
	return []types.GeneratedFile{
		{
			Path:           "nuget.config",
			Content:        buildNugetConfig(config.RegistryProxy),
			Mode:           0o644,
			Strategy:       types.Overwrite,
			SkipValidation: true,
		},
		{
			Path:           "Directory.Build.props",
			Content:        buildDirectoryBuildProps(),
			Mode:           0o644,
			Strategy:       types.Skip,
			SkipValidation: true,
		},
	}
}

// PreCommitHooks returns pre-commit hook definitions for .NET.
func (m *Module) PreCommitHooks(_ ecosystem.ModuleConfig) []ecosystem.HookConfig {
	return []ecosystem.HookConfig{
		{
			ID:          "dotnet-format",
			Name:        "dotnet-format",
			Description: "Check C#/F# code formatting with dotnet format",
			Entry:       "dotnet format --verify-no-changes",
			Language:    "system",
			Files:       `\.(cs|fs)$`,
			Stages:      []string{"pre-commit"},
			BuiltIn:     true,
		},
	}
}

// DenyRules returns Claude Code deny-rule patterns for .NET.
func (m *Module) DenyRules(_ ecosystem.ModuleConfig) []string {
	return []string{
		"Bash(dotnet add package *)",
		"Bash(nuget install *)",
	}
}

// CICommands returns CI pipeline commands for .NET.
func (m *Module) CICommands(_ ecosystem.ModuleConfig) []ecosystem.CICommand {
	return []ecosystem.CICommand{
		{
			Name:        "dotnet-restore-locked",
			Command:     "dotnet restore --locked-mode",
			Description: "Restore NuGet packages with locked dependencies",
			Phase:       ecosystem.CIPhaseInstall,
		},
		{
			Name:        "dotnet-vuln-scan",
			Command:     "dotnet list package --vulnerable --include-transitive",
			Description: "Scan NuGet packages for known vulnerabilities",
			Phase:       ecosystem.CIPhaseScan,
		},
	}
}

// PackageManagers returns metadata about NuGet.
func (m *Module) PackageManagers() []ecosystem.PackageManagerInfo {
	return []ecosystem.PackageManagerInfo{
		{
			Name:                 "nuget",
			LockFile:             "packages.lock.json",
			InstallCommand:       "dotnet restore",
			FrozenInstallCommand: "dotnet restore --locked-mode",
			AuditCommand:         "dotnet list package --vulnerable",
			AgeGatingSupport:     false,
		},
	}
}

// WizardFields returns wizard form fields for the .NET ecosystem.
func (m *Module) WizardFields() []ecosystem.WizardField {
	return []ecosystem.WizardField{
		{
			Key:         "dotnet_sdk_version",
			Label:       ".NET SDK version",
			Description: "Select the .NET SDK major version",
			Type:        ecosystem.FieldTypeSelect,
			// .NET 6 is EOL (end-of-life) and removed from wizard options.
			// Programmatic use of version "6" is still handled by sdkVersionToNixPackage.
			Options: []ecosystem.WizardOption{
				{Label: ".NET 9", Value: "9"},
				{Label: ".NET 8 (LTS)", Value: "8"},
				{Label: ".NET 7", Value: "7"},
			},
			Default: "8",
		},
	}
}

// VerificationCommands returns project verification commands for the .NET ecosystem.
func (m *Module) VerificationCommands(_ ecosystem.ModuleConfig) ecosystem.VerificationCommands {
	return ecosystem.VerificationCommands{
		Build:  []string{"dotnet build"},
		Test:   []string{"dotnet test"},
		Format: []string{"dotnet format --verify-no-changes"},
	}
}

// ManifestFiles returns manifest file metadata for the .NET ecosystem.
func (m *Module) ManifestFiles(_ ecosystem.ModuleConfig) []ecosystem.ManifestFileInfo {
	return []ecosystem.ManifestFileInfo{
		{
			Path:           "*.csproj",
			Ecosystem:      "nuget",
			VSSupported:    true,
			LockFile:       "packages.lock.json",
			LockFilePolicy: ecosystem.LockFilePolicyRecommended,
		},
	}
}

// globalJSONSchema represents the minimal structure of a global.json file.
type globalJSONSchema struct {
	SDK struct {
		Version string `json:"version"`
	} `json:"sdk"`
}

// parseGlobalJSON reads a global.json file and extracts the major SDK version
// (e.g., "8.0.301" → "8"). Returns "" if the file does not exist or cannot
// be parsed.
func parseGlobalJSON(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	var gj globalJSONSchema
	if err := json.Unmarshal(data, &gj); err != nil {
		return ""
	}

	version := gj.SDK.Version
	if version == "" {
		return ""
	}

	// Extract major version: everything before the first dot.
	for i, ch := range version {
		if ch == '.' {
			return version[:i]
		}
	}
	return version
}

// sdkVersionToNixPackage maps a major SDK version string to a Nix package name.
func sdkVersionToNixPackage(version string) string {
	switch version {
	case "9":
		return "dotnet-sdk_9"
	case "8":
		return "dotnet-sdk_8"
	case "7":
		return "dotnet-sdk_7"
	case "6":
		// .NET 6 is EOL but still mapped explicitly so programmatic callers
		// get the version they asked for rather than a silent upgrade.
		return "dotnet-sdk_6"
	default:
		return "dotnet-sdk_8"
	}
}

// xmlWriter wraps xml.Encoder to accumulate the first error across many
// EncodeToken calls, avoiding per-call error checking on bytes.Buffer writes.
type xmlWriter struct {
	enc *xml.Encoder
	err error
}

func (w *xmlWriter) token(t xml.Token) {
	if w.err == nil {
		w.err = w.enc.EncodeToken(t)
	}
}

func (w *xmlWriter) flush() error {
	if w.err != nil {
		return w.err
	}
	return w.enc.Flush()
}

// buildNugetConfig generates a security-hardened nuget.config XML file using
// token-by-token xml.Encoder emission to support XML comments.
func buildNugetConfig(registryProxy string) []byte {
	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0" encoding="utf-8"?>` + "\n")
	buf.WriteString("<!-- Generated by gdev-secure-devenv-bootstrap.\n")
	buf.WriteString("     Requires: NuGet >= 6.0 for signatureValidationMode=require.\n")
	buf.WriteString("     PackageReference format recommended (not packages.config). -->\n")

	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	w := &xmlWriter{enc: enc}

	w.token(xml.StartElement{Name: xml.Name{Local: "configuration"}})

	w.token(xml.Comment(" Package signature validation "))
	w.token(xml.StartElement{Name: xml.Name{Local: "config"}})
	w.token(xml.StartElement{
		Name: xml.Name{Local: "add"},
		Attr: []xml.Attr{
			{Name: xml.Name{Local: "key"}, Value: "signatureValidationMode"},
			{Name: xml.Name{Local: "value"}, Value: "require"},
		},
	})
	w.token(xml.EndElement{Name: xml.Name{Local: "add"}})
	w.token(xml.EndElement{Name: xml.Name{Local: "config"}})

	w.token(xml.Comment(" Trusted package signers "))
	w.token(xml.StartElement{Name: xml.Name{Local: "trustedSigners"}})
	w.token(xml.StartElement{
		Name: xml.Name{Local: "repository"},
		Attr: []xml.Attr{
			{Name: xml.Name{Local: "name"}, Value: "nuget.org"},
			{Name: xml.Name{Local: "serviceIndex"}, Value: "https://api.nuget.org/v3/index.json"},
		},
	})
	w.token(xml.StartElement{
		Name: xml.Name{Local: "certificate"},
		Attr: []xml.Attr{
			{Name: xml.Name{Local: "fingerprint"}, Value: "0E5F38F57DC1BCC806D8494F4F90FBCEDD988B46760709CBEEC6F4219AA6157D"},
			{Name: xml.Name{Local: "hashAlgorithm"}, Value: "SHA256"},
			{Name: xml.Name{Local: "allowUntrustedRoot"}, Value: "false"},
		},
	})
	w.token(xml.EndElement{Name: xml.Name{Local: "certificate"}})
	w.token(xml.StartElement{Name: xml.Name{Local: "owners"}})
	w.token(xml.CharData("*"))
	w.token(xml.EndElement{Name: xml.Name{Local: "owners"}})
	w.token(xml.EndElement{Name: xml.Name{Local: "repository"}})
	w.token(xml.EndElement{Name: xml.Name{Local: "trustedSigners"}})

	w.token(xml.Comment(" Package sources "))
	w.token(xml.StartElement{Name: xml.Name{Local: "packageSources"}})
	w.token(xml.StartElement{Name: xml.Name{Local: "clear"}})
	w.token(xml.EndElement{Name: xml.Name{Local: "clear"}})
	w.token(xml.StartElement{
		Name: xml.Name{Local: "add"},
		Attr: []xml.Attr{
			{Name: xml.Name{Local: "key"}, Value: "nuget.org"},
			{Name: xml.Name{Local: "value"}, Value: "https://api.nuget.org/v3/index.json"},
		},
	})
	w.token(xml.EndElement{Name: xml.Name{Local: "add"}})
	if registryProxy != "" {
		w.token(xml.StartElement{
			Name: xml.Name{Local: "add"},
			Attr: []xml.Attr{
				{Name: xml.Name{Local: "key"}, Value: "corporate-proxy"},
				{Name: xml.Name{Local: "value"}, Value: registryProxy},
			},
		})
		w.token(xml.EndElement{Name: xml.Name{Local: "add"}})
	}
	w.token(xml.EndElement{Name: xml.Name{Local: "packageSources"}})

	w.token(xml.Comment(" Audit settings "))
	w.token(xml.StartElement{Name: xml.Name{Local: "config"}})
	w.token(xml.StartElement{
		Name: xml.Name{Local: "add"},
		Attr: []xml.Attr{
			{Name: xml.Name{Local: "key"}, Value: "audit-level"},
			{Name: xml.Name{Local: "value"}, Value: "moderate"},
		},
	})
	w.token(xml.EndElement{Name: xml.Name{Local: "add"}})
	w.token(xml.StartElement{
		Name: xml.Name{Local: "add"},
		Attr: []xml.Attr{
			{Name: xml.Name{Local: "key"}, Value: "audit-mode"},
			{Name: xml.Name{Local: "value"}, Value: "all"},
		},
	})
	w.token(xml.EndElement{Name: xml.Name{Local: "add"}})
	w.token(xml.EndElement{Name: xml.Name{Local: "config"}})

	w.token(xml.EndElement{Name: xml.Name{Local: "configuration"}})

	_ = w.flush() //nolint:errcheck // writing to bytes.Buffer
	buf.WriteByte('\n')

	return buf.Bytes()
}

// buildDirectoryBuildProps generates a Directory.Build.props XML file with
// NuGet lockfile enforcement and central package management settings.
// Uses token-by-token xml.Encoder emission for the Condition attribute on
// RestoreLockedMode.
func buildDirectoryBuildProps() []byte {
	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0" encoding="utf-8"?>` + "\n")
	buf.WriteString("<!-- Generated by gdev-secure-devenv-bootstrap.\n")
	buf.WriteString("     Requires: .NET SDK >= 6.0 for central package management.\n")
	buf.WriteString("     Lock files require RestorePackagesWithLockFile=true. -->\n")

	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	w := &xmlWriter{enc: enc}

	w.token(xml.StartElement{Name: xml.Name{Local: "Project"}})
	w.token(xml.StartElement{Name: xml.Name{Local: "PropertyGroup"}})

	w.token(xml.Comment(" Enable NuGet package lock file "))
	w.token(xml.StartElement{Name: xml.Name{Local: "RestorePackagesWithLockFile"}})
	w.token(xml.CharData("true"))
	w.token(xml.EndElement{Name: xml.Name{Local: "RestorePackagesWithLockFile"}})

	w.token(xml.Comment(" Lock dependencies in CI "))
	w.token(xml.StartElement{
		Name: xml.Name{Local: "RestoreLockedMode"},
		Attr: []xml.Attr{
			{Name: xml.Name{Local: "Condition"}, Value: "'$(CI)' != ''"},
		},
	})
	w.token(xml.CharData("true"))
	w.token(xml.EndElement{Name: xml.Name{Local: "RestoreLockedMode"}})

	w.token(xml.Comment(" Enable central package management "))
	w.token(xml.StartElement{Name: xml.Name{Local: "ManagePackageVersionsCentrally"}})
	w.token(xml.CharData("true"))
	w.token(xml.EndElement{Name: xml.Name{Local: "ManagePackageVersionsCentrally"}})

	w.token(xml.EndElement{Name: xml.Name{Local: "PropertyGroup"}})
	w.token(xml.EndElement{Name: xml.Name{Local: "Project"}})

	_ = w.flush() //nolint:errcheck // writing to bytes.Buffer
	buf.WriteByte('\n')

	return buf.Bytes()
}

// SemgrepRuleSets returns Semgrep rule set identifiers relevant to C#/.NET projects.
func (m *Module) SemgrepRuleSets() []string {
	return []string{"p/csharp", "p/owasp-top-ten"}
}
