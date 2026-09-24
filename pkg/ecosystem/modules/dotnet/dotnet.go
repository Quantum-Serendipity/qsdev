// Package dotnet implements the C#/.NET ecosystem module for qsdev.
// It detects .NET projects via project files (*.csproj, *.fsproj, *.vbproj), solutions
// (*.sln, *.slnx), Directory.Build.props, and global.json, then generates devenv.nix fragments, security configs (nuget.config and
// Directory.Build.props), pre-commit hooks, deny rules, and CI commands for a hardened
// .NET development environment.
package dotnet

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Compile-time interface compliance checks.
var _ ecosystem.EcosystemModule = (*Module)(nil)
var _ ecosystem.SASTModule = (*Module)(nil)
var _ ecosystem.ReadDenyRuleProvider = (*Module)(nil)

// Module is the stateless C#/.NET ecosystem module.
type Module struct{}

func init() {
	ecosystem.MustRegisterModule(&Module{})
}

// Name returns the canonical module identifier.
func (m *Module) Name() string { return ecosystem.NameDotnet }

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

	// Project and solution files. Solutions sit at the root, but projects
	// conventionally live below it (src/App/App.csproj), so they are
	// searched a few levels deep.
	for _, ext := range findDotnetFiles(projectRoot) {
		result.Detected = true
		result.Confidence = ecosystem.ConfidenceCertain
		result.Evidence = append(result.Evidence, "*"+ext)
		if ext == ".fsproj" {
			result.SuggestedConfig.Extras["has_fsharp"] = "true"
		}
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

// dotnetProjectExts are the MSBuild project file extensions (C#, F#, VB).
var dotnetProjectExts = []string{".csproj", ".fsproj", ".vbproj"}

// dotnetSolutionExts are the solution file extensions; .slnx is the default
// `dotnet new sln` format since .NET 10.
var dotnetSolutionExts = []string{".sln", ".slnx"}

// maxProjectScanDepth bounds how many directory levels below projectRoot are
// searched for project files (root is depth 0), covering src/App/App.csproj
// and src/Services/Api/Api.csproj without walking a whole monorepo.
const maxProjectScanDepth = 3

// skippedScanDirs never hold the project's own project files: build output
// and third-party trees. Hidden directories (.git, .vs, ...) are skipped too.
var skippedScanDirs = map[string]bool{
	"bin":          true,
	"obj":          true,
	"node_modules": true,
	"packages":     true,
}

// findDotnetFiles returns, in a stable order, the project and solution file
// extensions present in projectRoot: solutions at the root only, projects
// down to maxProjectScanDepth levels.
func findDotnetFiles(projectRoot string) []string {
	projectRoot = filepath.Clean(projectRoot)
	seen := make(map[string]bool)
	_ = filepath.WalkDir(projectRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil //nolint:nilerr // best-effort scan
		}
		if d.IsDir() {
			if path == projectRoot {
				return nil
			}
			name := d.Name()
			if skippedScanDirs[name] || strings.HasPrefix(name, ".") || scanDepth(projectRoot, path) > maxProjectScanDepth {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		atRoot := filepath.Dir(path) == projectRoot
		if slices.Contains(dotnetProjectExts, ext) || (atRoot && slices.Contains(dotnetSolutionExts, ext)) {
			seen[ext] = true
		}
		return nil
	})
	var found []string
	for _, ext := range append(slices.Clone(dotnetProjectExts), dotnetSolutionExts...) {
		if seen[ext] {
			found = append(found, ext)
		}
	}
	return found
}

// scanDepth returns how many directory levels dir is below root.
func scanDepth(root, dir string) int {
	rel, err := filepath.Rel(root, dir)
	if err != nil {
		return maxProjectScanDepth + 1
	}
	return strings.Count(filepath.ToSlash(rel), "/") + 1
}

// DevenvNixFragment returns a Nix fragment that enables .NET in devenv.sh.
// When the requested SDK major version has no matching nixpkgs attribute, the
// substitution is recorded as a Nix comment above the language block.
func (m *Module) DevenvNixFragment(config ecosystem.ModuleConfig) (string, error) {
	pkg, note := sdkVersionToNixPackage(config.Version)

	fragment := ecosystem.BuildLanguageFragment(ecosystem.NixLangConfig{
		EnablePath: "languages.dotnet",
		Properties: []ecosystem.NixProperty{
			{Key: "package", Value: "pkgs." + pkg},
		},
	})
	if note != "" {
		fragment = "  # " + note + "\n" + fragment
	}
	return fragment, nil
}

// SecurityConfigs returns security-hardened configuration files for .NET.
func (m *Module) SecurityConfigs(config ecosystem.ModuleConfig) []types.GeneratedFile {
	return []types.GeneratedFile{
		{
			Path:           "nuget.config",
			Content:        buildNugetConfig(config.RegistryProxy),
			Mode:           fileutil.ModeReadWrite,
			Strategy:       types.Skip,
			SkipValidation: true,
		},
		{
			Path:           "Directory.Build.props",
			Content:        buildDirectoryBuildProps(),
			Mode:           fileutil.ModeReadWrite,
			Strategy:       types.Skip,
			SkipValidation: true,
		},
	}
}

// PreCommitHooks returns pre-commit hook definitions for .NET.
//
// The hook runs `dotnet` from the same SDK attribute as languages.dotnet, so
// it can build the project's target frameworks and satisfy its global.json,
// and no second, colliding dotnet binary is added to the profile.
func (m *Module) PreCommitHooks(config ecosystem.ModuleConfig) []ecosystem.HookConfig {
	sdk, _ := sdkVersionToNixPackage(config.Version)
	return []ecosystem.HookConfig{
		{
			ID:          "dotnet-format",
			Name:        "dotnet-format",
			Description: "Check C#/F# code formatting with dotnet format",
			Entry:       "dotnet format --verify-no-changes",
			Language:    "system",
			Files:       `\.(cs|fs)$`,
			Stages:      []string{"pre-commit"},
			BuiltIn:     false,
			NixPackage:  sdk,
		},
	}
}

// DenyRules returns Claude Code deny-rule patterns for .NET.
//
// Adding a package reference (`dotnet add package`, `dotnet add <PROJECT>
// package` and the .NET 10 noun-first `dotnet package add`) is not denied: the
// catalog's dotnet ask set prompts for it and package-guard checks the NuGet
// version it would pick for advisories and publication age first. What stays
// denied is every command that downloads and runs a NuGet package or
// re-resolves references without naming what it installs: tool installs and
// one-shot tool execution (`dotnet tool exec`, `dnx` and the `dotnet dnx` it
// forwards to), `dotnet package update`, template packages (including the
// pre-.NET 7 `dotnet new -i` form) and the standalone nuget CLI.
func (m *Module) DenyRules(_ ecosystem.ModuleConfig) []string {
	return []string{
		"Bash(dotnet package update *)",
		"Bash(dotnet tool install *)",
		"Bash(dotnet tool update *)",
		"Bash(dotnet tool exec *)",
		"Bash(dotnet tool run *)",
		"Bash(dnx *)",
		"Bash(dotnet dnx *)",
		"Bash(dotnet new install *)",
		"Bash(dotnet new -i *)",
		"Bash(dotnet new --install *)",
		"Bash(nuget *)",
		"Bash(nuget.exe *)",
		"Bash(mono nuget.exe *)",
	}
}

// ReadDenyRules returns the user-level NuGet configuration files, which hold
// packageSourceCredentials and push API keys for the user's feeds.
func (m *Module) ReadDenyRules(_ ecosystem.ModuleConfig) []string {
	return []string{
		"~/.nuget/NuGet/NuGet.Config",
		"~/.config/NuGet/NuGet.Config",
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
			Name:           "nuget",
			LockFile:       "packages.lock.json",
			InstallCommand: "dotnet restore",
		},
	}
}

// WizardFields returns wizard form fields for the .NET ecosystem.
func (m *Module) WizardFields() []ecosystem.WizardField {
	return []ecosystem.WizardField{
		{
			Key:         types.SettingVersion,
			Label:       ".NET SDK version",
			Description: "Select the .NET SDK major version",
			Type:        ecosystem.FieldTypeSelect,
			// EOL releases (.NET 6, 7) are not offered in the wizard.
			// Programmatic use of those versions is mapped to a supported
			// SDK by sdkVersionToNixPackage.
			Options: []ecosystem.WizardOption{
				{Label: ".NET 10 (LTS)", Value: "10"},
				{Label: ".NET 9", Value: "9"},
				{Label: ".NET 8 (LTS)", Value: "8"},
			},
			Default: strconv.Itoa(defaultSDKMajor),
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

// supportedSDKMajors lists the .NET SDK major versions that have a
// dotnet-sdk_<N> attribute in nixpkgs that evaluates without extra
// configuration, in ascending order. The EOL dotnet-sdk_6 and dotnet-sdk_7
// attributes still exist but are marked insecure, so nixpkgs refuses to
// evaluate them and devenv shell would fail; requests for those versions map
// to the oldest supported SDK, which can still target the older frameworks.
// Requests for any other major version are mapped onto this list by
// sdkVersionToNixPackage.
var supportedSDKMajors = []int{8, 9, 10, 11}

// defaultSDKMajor is the SDK used when no version is configured: the newest
// LTS release. Newer SDKs build projects targeting older frameworks.
const defaultSDKMajor = 10

// sdkVersionToNixPackage maps a major SDK version string to a nixpkgs
// attribute name. A version in supportedSDKMajors maps to dotnet-sdk_<N>.
// Any other numeric version maps to the nearest supported SDK at or above it
// (or the newest one), and an empty or non-numeric version maps to the
// default SDK. note is non-empty whenever the result is not the SDK that was
// asked for, so callers can surface the substitution instead of silently
// downgrading.
func sdkVersionToNixPackage(version string) (pkg, note string) {
	if version == "" {
		return sdkAttr(defaultSDKMajor), ""
	}
	major, err := strconv.Atoi(version)
	if err != nil || major <= 0 {
		return sdkAttr(defaultSDKMajor), fmt.Sprintf(
			"unrecognized .NET SDK version %q; using %s", version, sdkAttr(defaultSDKMajor))
	}
	for _, v := range supportedSDKMajors {
		if v == major {
			return sdkAttr(v), ""
		}
		if v > major {
			return sdkAttr(v), fmt.Sprintf(
				".NET SDK %d is not available from nixpkgs; using the next newer %s", major, sdkAttr(v))
		}
	}
	newest := supportedSDKMajors[len(supportedSDKMajors)-1]
	return sdkAttr(newest), fmt.Sprintf(
		".NET SDK %d is newer than any packaged SDK; using %s", major, sdkAttr(newest))
}

// sdkAttr returns the nixpkgs attribute name for a .NET SDK major version.
func sdkAttr(major int) string {
	return "dotnet-sdk_" + strconv.Itoa(major)
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

// nugetOrgServiceIndex is the nuget.org V3 service index URL.
const nugetOrgServiceIndex = "https://api.nuget.org/v3/index.json"

// nugetOrgRepositoryFingerprints are the SHA-256 fingerprints of every
// repository-signing certificate nuget.org has published (see the
// trustedSigners section of the nuget.config reference). nuget.org rotates
// this certificate and packages keep the signature of the certificate that
// was current when they were published, so all of them must stay trusted or
// signatureValidationMode=require rejects those packages (NU3034).
var nugetOrgRepositoryFingerprints = []string{
	"0E5F38F57DC1BCC806D8494F4F90FBCEDD988B46760709CBEEC6F4219AA6157D",
	"5A2901D6ADA3D18260B9C6DFE2133C95D74B9EEF6AE0E5DC334C8454D1477DF4",
	"1F4B311D9ACC115C8DC8018B5A49E00FCE6DA8E2855F9F014CA6F34570BC482D",
}

// buildNugetConfig generates a security-hardened nuget.config XML file using
// token-by-token xml.Encoder emission to support XML comments.
//
// When registryProxy is set it becomes the only package source, so every
// restore goes through the proxy and a same-named package on nuget.org cannot
// win resolution. Packages the proxy serves from nuget.org keep their
// nuget.org repository signature, so the nuget.org trusted signer still
// applies.
func buildNugetConfig(registryProxy string) []byte {
	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0" encoding="utf-8"?>` + "\n")
	buf.WriteString("<!-- " + branding.GeneratedBy() + ".\n")
	buf.WriteString("     Requires: NuGet >= 6.0 for signatureValidationMode=require.\n")
	buf.WriteString("     PackageReference format recommended (not packages.config).\n")
	buf.WriteString("     NuGet vulnerability auditing is configured in Directory.Build.props. -->\n")

	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	w := &xmlWriter{enc: enc}

	w.token(xml.StartElement{Name: xml.Name{Local: "configuration"}})

	w.token(xml.Comment(" Package signature validation "))
	w.token(xml.StartElement{Name: xml.Name{Local: "config"}})
	w.addEntry("signatureValidationMode", "require")
	w.token(xml.EndElement{Name: xml.Name{Local: "config"}})

	// No <owners> element: a repository signer without owners trusts every
	// package carrying a valid nuget.org repository signature. <owners> is a
	// semicolon-separated list of nuget.org accounts and has no wildcard.
	w.token(xml.Comment(" Trusted package signers "))
	w.token(xml.StartElement{Name: xml.Name{Local: "trustedSigners"}})
	w.token(xml.StartElement{
		Name: xml.Name{Local: "repository"},
		Attr: []xml.Attr{
			{Name: xml.Name{Local: "name"}, Value: "nuget.org"},
			{Name: xml.Name{Local: "serviceIndex"}, Value: nugetOrgServiceIndex},
		},
	})
	for _, fp := range nugetOrgRepositoryFingerprints {
		w.token(xml.StartElement{
			Name: xml.Name{Local: "certificate"},
			Attr: []xml.Attr{
				{Name: xml.Name{Local: "fingerprint"}, Value: fp},
				{Name: xml.Name{Local: "hashAlgorithm"}, Value: "SHA256"},
				{Name: xml.Name{Local: "allowUntrustedRoot"}, Value: "false"},
			},
		})
		w.token(xml.EndElement{Name: xml.Name{Local: "certificate"}})
	}
	w.token(xml.EndElement{Name: xml.Name{Local: "repository"}})
	w.token(xml.EndElement{Name: xml.Name{Local: "trustedSigners"}})

	w.token(xml.Comment(" Package sources "))
	w.token(xml.StartElement{Name: xml.Name{Local: "packageSources"}})
	w.token(xml.StartElement{Name: xml.Name{Local: "clear"}})
	w.token(xml.EndElement{Name: xml.Name{Local: "clear"}})
	if registryProxy != "" {
		w.addEntry("corporate-proxy", registryProxy)
	} else {
		w.addEntry("nuget.org", nugetOrgServiceIndex)
	}
	w.token(xml.EndElement{Name: xml.Name{Local: "packageSources"}})

	w.token(xml.EndElement{Name: xml.Name{Local: "configuration"}})

	_ = w.flush() //nolint:errcheck // writing to bytes.Buffer
	buf.WriteByte('\n')

	return buf.Bytes()
}

// addEntry emits an <add key="..." value="..."/> element.
func (w *xmlWriter) addEntry(key, value string) {
	w.token(xml.StartElement{
		Name: xml.Name{Local: "add"},
		Attr: []xml.Attr{
			{Name: xml.Name{Local: "key"}, Value: key},
			{Name: xml.Name{Local: "value"}, Value: value},
		},
	})
	w.token(xml.EndElement{Name: xml.Name{Local: "add"}})
}

// buildDirectoryBuildProps generates a Directory.Build.props XML file with
// NuGet lockfile enforcement and NuGet vulnerability auditing settings.
// Uses token-by-token xml.Encoder emission for the Condition attribute on
// RestoreLockedMode.
//
// Central package management (ManagePackageVersionsCentrally) is deliberately
// not enabled: it requires a Directory.Packages.props and version-less
// PackageReference items, so forcing it onto an existing project fails every
// restore with NU1008. Projects that use it already declare it themselves.
func buildDirectoryBuildProps() []byte {
	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0" encoding="utf-8"?>` + "\n")
	buf.WriteString("<!-- " + branding.GeneratedBy() + ".\n")
	buf.WriteString("     Requires: .NET SDK >= 8.0.100 (NuGet 6.8) for the NuGetAudit properties.\n")
	buf.WriteString("     Lock files require RestorePackagesWithLockFile=true. -->\n")

	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	w := &xmlWriter{enc: enc}

	w.token(xml.StartElement{Name: xml.Name{Local: "Project"}})
	w.token(xml.StartElement{Name: xml.Name{Local: "PropertyGroup"}})

	w.token(xml.Comment(" Enable NuGet package lock file "))
	w.property("RestorePackagesWithLockFile", "true")

	w.token(xml.Comment(" Lock dependencies in CI "))
	w.token(xml.StartElement{
		Name: xml.Name{Local: "RestoreLockedMode"},
		Attr: []xml.Attr{
			{Name: xml.Name{Local: "Condition"}, Value: "'$(CI)' != ''"},
		},
	})
	w.token(xml.CharData("true"))
	w.token(xml.EndElement{Name: xml.Name{Local: "RestoreLockedMode"}})

	w.token(xml.Comment(" Audit direct and transitive packages for moderate+ vulnerabilities "))
	w.property("NuGetAudit", "true")
	w.property("NuGetAuditLevel", "moderate")
	w.property("NuGetAuditMode", "all")

	w.token(xml.EndElement{Name: xml.Name{Local: "PropertyGroup"}})
	w.token(xml.EndElement{Name: xml.Name{Local: "Project"}})

	_ = w.flush() //nolint:errcheck // writing to bytes.Buffer
	buf.WriteByte('\n')

	return buf.Bytes()
}

// property emits a <name>value</name> MSBuild property element.
func (w *xmlWriter) property(name, value string) {
	w.token(xml.StartElement{Name: xml.Name{Local: name}})
	w.token(xml.CharData(value))
	w.token(xml.EndElement{Name: xml.Name{Local: name}})
}

// SemgrepRuleSets returns Semgrep rule set identifiers relevant to C#/.NET projects.
func (m *Module) SemgrepRuleSets() []string {
	return []string{"p/csharp", "p/owasp-top-ten"}
}
