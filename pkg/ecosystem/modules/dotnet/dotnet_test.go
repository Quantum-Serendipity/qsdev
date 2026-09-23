package dotnet_test

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/dotnet"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// newModule returns a fresh Module for testing.
func newModule() *dotnet.Module {
	return &dotnet.Module{}
}

// --- Interface compliance ---

func TestInterfaceCompliance(t *testing.T) {
	var _ ecosystem.EcosystemModule = (*dotnet.Module)(nil)
}

// --- Basic metadata ---

func TestModuleIdentity(t *testing.T) {
	ecosystem.AssertModuleIdentity(t, newModule(), "dotnet", "C#/.NET", 1)
}

// --- Detection tests ---

func TestDetect_Csproj(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "MyApp.csproj"), []byte("<Project></Project>\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	r := m.Detect(dir)

	if !r.Detected {
		t.Fatal("expected Detected = true")
	}
	if r.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("Confidence = %v, want Certain", r.Confidence)
	}
	if !slices.Contains(r.Evidence, "*.csproj") {
		t.Errorf("Evidence = %v, want to contain %q", r.Evidence, "*.csproj")
	}
	if r.SuggestedConfig.PackageManager != "nuget" {
		t.Errorf("PackageManager = %q, want %q", r.SuggestedConfig.PackageManager, "nuget")
	}
}

func TestDetect_Fsproj(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "MyLib.fsproj"), []byte("<Project></Project>\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	r := m.Detect(dir)

	if !r.Detected {
		t.Fatal("expected Detected = true")
	}
	if r.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("Confidence = %v, want Certain", r.Confidence)
	}
	if !slices.Contains(r.Evidence, "*.fsproj") {
		t.Errorf("Evidence = %v, want to contain %q", r.Evidence, "*.fsproj")
	}
	if r.SuggestedConfig.Extras["has_fsharp"] != "true" {
		t.Errorf("Extras[has_fsharp] = %q, want %q", r.SuggestedConfig.Extras["has_fsharp"], "true")
	}
}

func TestDetect_Sln(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "MySolution.sln"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	r := m.Detect(dir)

	if !r.Detected {
		t.Fatal("expected Detected = true")
	}
	if r.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("Confidence = %v, want Certain", r.Confidence)
	}
	if !slices.Contains(r.Evidence, "*.sln") {
		t.Errorf("Evidence = %v, want to contain %q", r.Evidence, "*.sln")
	}
}

func TestDetect_DirectoryBuildProps(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Directory.Build.props"), []byte("<Project></Project>\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	r := m.Detect(dir)

	if !r.Detected {
		t.Fatal("expected Detected = true")
	}
	if r.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("Confidence = %v, want Certain", r.Confidence)
	}
	if !slices.Contains(r.Evidence, "Directory.Build.props") {
		t.Errorf("Evidence = %v, want to contain %q", r.Evidence, "Directory.Build.props")
	}
}

func TestDetect_GlobalJSON(t *testing.T) {
	dir := t.TempDir()
	globalJSON := `{"sdk":{"version":"8.0.301"}}`
	if err := os.WriteFile(filepath.Join(dir, "global.json"), []byte(globalJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	r := m.Detect(dir)

	if !r.Detected {
		t.Fatal("expected Detected = true")
	}
	if r.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("Confidence = %v, want Certain", r.Confidence)
	}
	if !slices.Contains(r.Evidence, "global.json") {
		t.Errorf("Evidence = %v, want to contain %q", r.Evidence, "global.json")
	}
	if r.SuggestedConfig.Version != "8" {
		t.Errorf("Version = %q, want %q", r.SuggestedConfig.Version, "8")
	}
}

func TestDetect_GlobalJSON_SDK9(t *testing.T) {
	dir := t.TempDir()
	globalJSON := `{"sdk":{"version":"9.0.100"}}`
	if err := os.WriteFile(filepath.Join(dir, "global.json"), []byte(globalJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	r := m.Detect(dir)

	if !r.Detected {
		t.Fatal("expected Detected = true")
	}
	if r.SuggestedConfig.Version != "9" {
		t.Errorf("Version = %q, want %q", r.SuggestedConfig.Version, "9")
	}
}

func TestDetect_GlobalJSON_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "global.json"), []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	r := m.Detect(dir)

	// Invalid JSON should not contribute to detection.
	if r.Detected {
		t.Error("expected Detected = false for invalid global.json without other indicators")
	}
}

func TestDetect_GlobalJSON_NoSDKVersion(t *testing.T) {
	dir := t.TempDir()
	globalJSON := `{"sdk":{}}`
	if err := os.WriteFile(filepath.Join(dir, "global.json"), []byte(globalJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	r := m.Detect(dir)

	// Empty SDK version should not count as detection.
	if r.Detected {
		t.Error("expected Detected = false for global.json without sdk.version")
	}
}

func TestDetect_MultipleIndicators(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "MyApp.csproj"), []byte("<Project/>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "MyLib.fsproj"), []byte("<Project/>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "MySolution.sln"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	globalJSON := `{"sdk":{"version":"9.0.100"}}`
	if err := os.WriteFile(filepath.Join(dir, "global.json"), []byte(globalJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	r := m.Detect(dir)

	if !r.Detected {
		t.Fatal("expected Detected = true")
	}
	if r.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("Confidence = %v, want Certain", r.Confidence)
	}

	// Should have evidence for all indicators.
	expectedEvidence := []string{"*.csproj", "*.fsproj", "*.sln", "global.json"}
	for _, ev := range expectedEvidence {
		if !slices.Contains(r.Evidence, ev) {
			t.Errorf("Evidence = %v, missing %q", r.Evidence, ev)
		}
	}

	if r.SuggestedConfig.Extras["has_fsharp"] != "true" {
		t.Errorf("Extras[has_fsharp] = %q, want %q", r.SuggestedConfig.Extras["has_fsharp"], "true")
	}
	if r.SuggestedConfig.Version != "9" {
		t.Errorf("Version = %q, want %q", r.SuggestedConfig.Version, "9")
	}
}

func TestDetect_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	m := newModule()
	r := m.Detect(dir)

	if r.Detected {
		t.Fatal("expected Detected = false for empty directory")
	}
	if r.Confidence != ecosystem.ConfidenceAbsent {
		t.Errorf("Confidence = %v, want Absent", r.Confidence)
	}
	if len(r.Evidence) != 0 {
		t.Errorf("Evidence = %v, want empty", r.Evidence)
	}
}

// --- DevenvNixFragment tests ---

func TestDevenvNixFragment_SDK8(t *testing.T) {
	m := newModule()
	config := ecosystem.ModuleConfig{Version: "8"}

	frag, err := m.DevenvNixFragment(config)
	if err != nil {
		t.Fatalf("DevenvNixFragment() error: %v", err)
	}

	if !strings.Contains(frag, "languages.dotnet") {
		t.Errorf("fragment missing languages.dotnet:\n%s", frag)
	}
	if !strings.Contains(frag, "enable = true") {
		t.Errorf("fragment missing enable = true:\n%s", frag)
	}
	if !strings.Contains(frag, "pkgs.dotnet-sdk_8") {
		t.Errorf("fragment missing pkgs.dotnet-sdk_8:\n%s", frag)
	}
}

func TestDevenvNixFragment_SDK9(t *testing.T) {
	m := newModule()
	config := ecosystem.ModuleConfig{Version: "9"}

	frag, err := m.DevenvNixFragment(config)
	if err != nil {
		t.Fatalf("DevenvNixFragment() error: %v", err)
	}

	if !strings.Contains(frag, "pkgs.dotnet-sdk_9") {
		t.Errorf("fragment missing pkgs.dotnet-sdk_9:\n%s", frag)
	}
}

func TestDevenvNixFragment_SDKVersionMapping(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		version  string
		wantPkg  string
		wantNote bool
	}{
		{name: "empty uses default LTS", version: "", wantPkg: "pkgs.dotnet-sdk_10"},
		{name: "net10", version: "10", wantPkg: "pkgs.dotnet-sdk_10"},
		{name: "net11", version: "11", wantPkg: "pkgs.dotnet-sdk_11"},
		{name: "net8", version: "8", wantPkg: "pkgs.dotnet-sdk_8"},
		{name: "insecure EOL net6 upgrades", version: "6", wantPkg: "pkgs.dotnet-sdk_8", wantNote: true},
		{name: "insecure EOL net7 upgrades", version: "7", wantPkg: "pkgs.dotnet-sdk_8", wantNote: true},
		{name: "older than any packaged SDK upgrades", version: "5", wantPkg: "pkgs.dotnet-sdk_8", wantNote: true},
		{name: "newer than any packaged SDK uses newest", version: "12", wantPkg: "pkgs.dotnet-sdk_11", wantNote: true},
		{name: "non-numeric uses default", version: "latest", wantPkg: "pkgs.dotnet-sdk_10", wantNote: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			frag, err := newModule().DevenvNixFragment(ecosystem.ModuleConfig{Version: tt.version})
			if err != nil {
				t.Fatalf("DevenvNixFragment() error: %v", err)
			}
			if !strings.Contains(frag, "package = "+tt.wantPkg+";") {
				t.Errorf("fragment should use %s:\n%s", tt.wantPkg, frag)
			}
			if hasNote := strings.Contains(frag, "  # "); hasNote != tt.wantNote {
				t.Errorf("substitution note present = %v, want %v:\n%s", hasNote, tt.wantNote, frag)
			}
		})
	}
}

func TestWizardFields_OfferedVersionsArePackaged(t *testing.T) {
	t.Parallel()
	f := newModule().WizardFields()[0]
	values := make([]string, 0, len(f.Options))
	for _, opt := range f.Options {
		values = append(values, opt.Value)
		frag, err := newModule().DevenvNixFragment(ecosystem.ModuleConfig{Version: opt.Value})
		if err != nil {
			t.Fatalf("DevenvNixFragment(%q) error: %v", opt.Value, err)
		}
		if !strings.Contains(frag, "pkgs.dotnet-sdk_"+opt.Value+";") {
			t.Errorf("wizard option %q is not a packaged SDK:\n%s", opt.Value, frag)
		}
	}
	for _, eol := range []string{"6", "7"} {
		if slices.Contains(values, eol) {
			t.Errorf("wizard offers EOL .NET %s", eol)
		}
	}
	if !slices.Contains(values, "10") {
		t.Errorf("wizard does not offer .NET 10 (LTS); options = %v", values)
	}
	if !slices.Contains(values, f.Default) {
		t.Errorf("wizard default %q is not one of the options %v", f.Default, values)
	}
}

// --- SecurityConfigs tests ---

func TestSecurityConfigs_Count(t *testing.T) {
	m := newModule()
	files := m.SecurityConfigs(ecosystem.ModuleConfig{})

	if len(files) != 2 {
		t.Fatalf("SecurityConfigs() returned %d files, want 2", len(files))
	}
}

func TestSecurityConfigs_NugetConfig(t *testing.T) {
	m := newModule()
	files := m.SecurityConfigs(ecosystem.ModuleConfig{})

	var nugetConfig *types.GeneratedFile
	for i := range files {
		if files[i].Path == "nuget.config" {
			nugetConfig = &files[i]
			break
		}
	}
	if nugetConfig == nil {
		t.Fatal("no nuget.config in SecurityConfigs output")
		return
	}

	content := string(nugetConfig.Content)

	// Check XML declaration.
	if !strings.HasPrefix(content, `<?xml version="1.0" encoding="utf-8"?>`) {
		t.Error("nuget.config should start with XML declaration")
	}

	// Verify it parses as valid XML.
	if err := xml.Unmarshal(nugetConfig.Content, new(any)); err != nil {
		t.Errorf("nuget.config is not valid XML: %v", err)
	}

	// Check version requirement comment.
	if !strings.Contains(content, "NuGet >= 6.0") {
		t.Error("nuget.config missing version requirement comment for NuGet >= 6.0")
	}

	// Check signatureValidationMode.
	if !strings.Contains(content, "signatureValidationMode") {
		t.Error("nuget.config missing signatureValidationMode")
	}
	if !strings.Contains(content, `value="require"`) {
		t.Error("nuget.config missing signatureValidationMode=require")
	}

	// Check certificate fingerprints.
	for _, fp := range []string{
		"0E5F38F57DC1BCC806D8494F4F90FBCEDD988B46760709CBEEC6F4219AA6157D",
		"5A2901D6ADA3D18260B9C6DFE2133C95D74B9EEF6AE0E5DC334C8454D1477DF4",
		"1F4B311D9ACC115C8DC8018B5A49E00FCE6DA8E2855F9F014CA6F34570BC482D",
	} {
		if !strings.Contains(content, fp) {
			t.Errorf("nuget.config missing nuget.org certificate fingerprint %s", fp)
		}
	}

	// Check clear + source.
	if !strings.Contains(content, "<clear>") {
		t.Error("nuget.config missing <clear> in packageSources")
	}
	if !strings.Contains(content, "https://api.nuget.org/v3/index.json") {
		t.Error("nuget.config missing nuget.org source URL")
	}

	// audit-level/audit-mode are not nuget.config <config> keys; NuGet audit
	// is configured through MSBuild properties in Directory.Build.props.
	for _, invalid := range []string{"audit-level", "audit-mode"} {
		if strings.Contains(content, invalid) {
			t.Errorf("nuget.config contains unsupported config key %q", invalid)
		}
	}

	// Check strategy.
	if nugetConfig.Strategy != types.Overwrite {
		t.Errorf("nuget.config Strategy = %v, want Overwrite", nugetConfig.Strategy)
	}

	// Check SkipValidation.
	if !nugetConfig.SkipValidation {
		t.Error("nuget.config SkipValidation should be true")
	}
}

func TestSecurityConfigs_NugetConfig_ValidXMLStructure(t *testing.T) {
	m := newModule()
	files := m.SecurityConfigs(ecosystem.ModuleConfig{})

	var nugetConfig *types.GeneratedFile
	for i := range files {
		if files[i].Path == "nuget.config" {
			nugetConfig = &files[i]
			break
		}
	}
	if nugetConfig == nil {
		t.Fatal("no nuget.config in SecurityConfigs output")
		return
	}

	// Parse the XML to verify structure.
	type xmlAdd struct {
		Key   string `xml:"key,attr"`
		Value string `xml:"value,attr"`
	}
	type xmlCertificate struct {
		Fingerprint        string `xml:"fingerprint,attr"`
		HashAlgorithm      string `xml:"hashAlgorithm,attr"`
		AllowUntrustedRoot string `xml:"allowUntrustedRoot,attr"`
	}
	type xmlOwners struct {
		Content string `xml:",chardata"`
	}
	type xmlRepository struct {
		Name         string           `xml:"name,attr"`
		ServiceIndex string           `xml:"serviceIndex,attr"`
		Certificates []xmlCertificate `xml:"certificate"`
		Owners       *xmlOwners       `xml:"owners"`
	}
	type xmlConfiguration struct {
		XMLName xml.Name `xml:"configuration"`
		Config  []struct {
			Add []xmlAdd `xml:"add"`
		} `xml:"config"`
		TrustedSigners struct {
			Repository xmlRepository `xml:"repository"`
		} `xml:"trustedSigners"`
		PackageSources struct {
			Add []xmlAdd `xml:"add"`
		} `xml:"packageSources"`
	}

	var cfg xmlConfiguration
	// Strip the XML declaration for Unmarshal.
	content := nugetConfig.Content
	if err := xml.Unmarshal(content, &cfg); err != nil {
		t.Fatalf("failed to unmarshal nuget.config: %v", err)
	}

	// Verify trusted signer repository.
	repo := cfg.TrustedSigners.Repository
	if repo.Name != "nuget.org" {
		t.Errorf("trustedSigners repository name = %q, want %q", repo.Name, "nuget.org")
	}
	// Every published nuget.org repository certificate must be trusted, not
	// just the original 2018 one, or packages signed after a rotation fail
	// with NU3034 under signatureValidationMode=require.
	wantFingerprints := []string{
		"0E5F38F57DC1BCC806D8494F4F90FBCEDD988B46760709CBEEC6F4219AA6157D",
		"5A2901D6ADA3D18260B9C6DFE2133C95D74B9EEF6AE0E5DC334C8454D1477DF4",
		"1F4B311D9ACC115C8DC8018B5A49E00FCE6DA8E2855F9F014CA6F34570BC482D",
	}
	gotFingerprints := make([]string, 0, len(repo.Certificates))
	for _, c := range repo.Certificates {
		gotFingerprints = append(gotFingerprints, c.Fingerprint)
		if c.HashAlgorithm != "SHA256" {
			t.Errorf("hashAlgorithm = %q, want %q", c.HashAlgorithm, "SHA256")
		}
		if c.AllowUntrustedRoot != "false" {
			t.Errorf("allowUntrustedRoot = %q, want %q", c.AllowUntrustedRoot, "false")
		}
	}
	if !slices.Equal(gotFingerprints, wantFingerprints) {
		t.Errorf("certificate fingerprints = %v, want %v", gotFingerprints, wantFingerprints)
	}
	// <owners> is an account list with no wildcard; omitting it trusts any
	// package with a valid nuget.org repository signature.
	if repo.Owners != nil {
		t.Errorf("trustedSigners repository has <owners>%s</owners>, want none", repo.Owners.Content)
	}

	// Verify package source.
	if len(cfg.PackageSources.Add) != 1 {
		t.Fatalf("packageSources has %d add entries, want 1", len(cfg.PackageSources.Add))
	}
	if cfg.PackageSources.Add[0].Key != "nuget.org" {
		t.Errorf("packageSources add key = %q, want %q", cfg.PackageSources.Add[0].Key, "nuget.org")
	}
}

func TestSecurityConfigs_NugetConfig_Comments(t *testing.T) {
	m := newModule()
	files := m.SecurityConfigs(ecosystem.ModuleConfig{})

	var nugetConfig *types.GeneratedFile
	for i := range files {
		if files[i].Path == "nuget.config" {
			nugetConfig = &files[i]
			break
		}
	}
	if nugetConfig == nil {
		t.Fatal("no nuget.config in SecurityConfigs output")
		return
	}

	content := string(nugetConfig.Content)

	// Verify comments are present.
	if !strings.Contains(content, "<!-- Package signature validation -->") {
		t.Error("nuget.config missing 'Package signature validation' comment")
	}
	if !strings.Contains(content, "<!-- Trusted package signers -->") {
		t.Error("nuget.config missing 'Trusted package signers' comment")
	}
	if !strings.Contains(content, "<!-- Package sources -->") {
		t.Error("nuget.config missing 'Package sources' comment")
	}
}

func TestSecurityConfigs_DirectoryBuildProps(t *testing.T) {
	m := newModule()
	files := m.SecurityConfigs(ecosystem.ModuleConfig{})

	var buildProps *types.GeneratedFile
	for i := range files {
		if files[i].Path == "Directory.Build.props" {
			buildProps = &files[i]
			break
		}
	}
	if buildProps == nil {
		t.Fatal("no Directory.Build.props in SecurityConfigs output")
		return
	}

	content := string(buildProps.Content)

	// Check XML declaration.
	if !strings.HasPrefix(content, `<?xml version="1.0" encoding="utf-8"?>`) {
		t.Error("Directory.Build.props should start with XML declaration")
	}

	// Verify it parses as valid XML.
	if err := xml.Unmarshal(buildProps.Content, new(any)); err != nil {
		t.Errorf("Directory.Build.props is not valid XML: %v", err)
	}

	// Check RestorePackagesWithLockFile.
	if !strings.Contains(content, "RestorePackagesWithLockFile") {
		t.Error("Directory.Build.props missing RestorePackagesWithLockFile")
	}

	// Check RestoreLockedMode with Condition.
	if !strings.Contains(content, "RestoreLockedMode") {
		t.Error("Directory.Build.props missing RestoreLockedMode")
	}
	// The xml.Encoder escapes single quotes as &#39; in attribute values, so
	// check for the escaped form that is produced by encoding/xml.
	hasConditionRaw := strings.Contains(content, `Condition="'$(CI)' != ''"`)
	hasConditionEscaped := strings.Contains(content, `Condition="&#39;$(CI)&#39; != &#39;&#39;"`)
	if !hasConditionRaw && !hasConditionEscaped {
		t.Errorf("Directory.Build.props missing Condition attribute on RestoreLockedMode.\nContent:\n%s", content)
	}

	// Central package management must not be forced on: without a
	// Directory.Packages.props every versioned PackageReference fails NU1008.
	if strings.Contains(content, "ManagePackageVersionsCentrally") {
		t.Error("Directory.Build.props must not enable ManagePackageVersionsCentrally")
	}

	// Check strategy is Skip.
	if buildProps.Strategy != types.Skip {
		t.Errorf("Directory.Build.props Strategy = %v, want Skip", buildProps.Strategy)
	}

	// Check SkipValidation.
	if !buildProps.SkipValidation {
		t.Error("Directory.Build.props SkipValidation should be true")
	}
}

func TestSecurityConfigs_DirectoryBuildProps_XMLStructure(t *testing.T) {
	m := newModule()
	files := m.SecurityConfigs(ecosystem.ModuleConfig{})

	var buildProps *types.GeneratedFile
	for i := range files {
		if files[i].Path == "Directory.Build.props" {
			buildProps = &files[i]
			break
		}
	}
	if buildProps == nil {
		t.Fatal("no Directory.Build.props in SecurityConfigs output")
		return
	}

	// Parse a simplified XML structure.
	type xmlPropertyGroup struct {
		RestorePackagesWithLockFile string `xml:"RestorePackagesWithLockFile"`
		RestoreLockedMode           string `xml:"RestoreLockedMode"`
		NuGetAudit                  string `xml:"NuGetAudit"`
		NuGetAuditLevel             string `xml:"NuGetAuditLevel"`
		NuGetAuditMode              string `xml:"NuGetAuditMode"`
	}
	type xmlProject struct {
		XMLName       xml.Name         `xml:"Project"`
		PropertyGroup xmlPropertyGroup `xml:"PropertyGroup"`
	}

	var proj xmlProject
	if err := xml.Unmarshal(buildProps.Content, &proj); err != nil {
		t.Fatalf("failed to unmarshal Directory.Build.props: %v", err)
	}

	if proj.PropertyGroup.RestorePackagesWithLockFile != "true" {
		t.Errorf("RestorePackagesWithLockFile = %q, want %q",
			proj.PropertyGroup.RestorePackagesWithLockFile, "true")
	}
	if proj.PropertyGroup.RestoreLockedMode != "true" {
		t.Errorf("RestoreLockedMode = %q, want %q",
			proj.PropertyGroup.RestoreLockedMode, "true")
	}
	for _, prop := range []struct{ name, got, want string }{
		{"NuGetAudit", proj.PropertyGroup.NuGetAudit, "true"},
		{"NuGetAuditLevel", proj.PropertyGroup.NuGetAuditLevel, "moderate"},
		{"NuGetAuditMode", proj.PropertyGroup.NuGetAuditMode, "all"},
	} {
		if prop.got != prop.want {
			t.Errorf("%s = %q, want %q", prop.name, prop.got, prop.want)
		}
	}
}

// --- Registry proxy tests ---

func TestSecurityConfigs_NugetConfig_RegistryProxy(t *testing.T) {
	m := newModule()
	proxy := "https://nuget.corp.example.com/v3/index.json"
	files := m.SecurityConfigs(ecosystem.ModuleConfig{RegistryProxy: proxy})

	var nugetConfig *types.GeneratedFile
	for i := range files {
		if files[i].Path == "nuget.config" {
			nugetConfig = &files[i]
			break
		}
	}
	if nugetConfig == nil {
		t.Fatal("no nuget.config in SecurityConfigs output")
		return
	}

	content := string(nugetConfig.Content)
	// Proxy source must be present.
	if !strings.Contains(content, "corporate-proxy") {
		t.Errorf("nuget.config missing corporate-proxy source when proxy is set\ncontent:\n%s", content)
	}
	if !strings.Contains(content, proxy) {
		t.Errorf("nuget.config missing proxy URL\ncontent:\n%s", content)
	}
	// Existing security settings must be preserved.
	if !strings.Contains(content, "signatureValidationMode") {
		t.Error("nuget.config missing signatureValidationMode when proxy is set")
	}
	if !strings.Contains(content, `value="require"`) {
		t.Error("nuget.config missing signatureValidationMode=require when proxy is set")
	}

	// The proxy must be the only package source: keeping nuget.org next to
	// it lets restores bypass the proxy and resolve same-named packages
	// directly from nuget.org.
	type xmlAdd struct {
		Key   string `xml:"key,attr"`
		Value string `xml:"value,attr"`
	}
	var cfg struct {
		PackageSources struct {
			Add []xmlAdd `xml:"add"`
		} `xml:"packageSources"`
	}
	if err := xml.Unmarshal(nugetConfig.Content, &cfg); err != nil {
		t.Fatalf("failed to unmarshal nuget.config: %v", err)
	}
	want := []xmlAdd{{Key: "corporate-proxy", Value: proxy}}
	if !slices.Equal(cfg.PackageSources.Add, want) {
		t.Errorf("packageSources = %+v, want only %+v", cfg.PackageSources.Add, want)
	}
}

func TestSecurityConfigs_NugetConfig_NoRegistryProxy(t *testing.T) {
	m := newModule()
	files := m.SecurityConfigs(ecosystem.ModuleConfig{})

	var nugetConfig *types.GeneratedFile
	for i := range files {
		if files[i].Path == "nuget.config" {
			nugetConfig = &files[i]
			break
		}
	}
	if nugetConfig == nil {
		t.Fatal("no nuget.config in SecurityConfigs output")
		return
	}

	content := string(nugetConfig.Content)
	if strings.Contains(content, "corporate-proxy") {
		t.Errorf("nuget.config should not contain corporate-proxy when proxy is empty\ncontent:\n%s", content)
	}
}

func TestSecurityConfigs_NugetConfig_RegistryProxyPreservesExisting(t *testing.T) {
	m := newModule()
	proxy := "https://nuget.corp.example.com/v3/index.json"
	files := m.SecurityConfigs(ecosystem.ModuleConfig{RegistryProxy: proxy})

	var nugetConfig *types.GeneratedFile
	for i := range files {
		if files[i].Path == "nuget.config" {
			nugetConfig = &files[i]
			break
		}
	}
	if nugetConfig == nil {
		t.Fatal("no nuget.config in SecurityConfigs output")
		return
	}

	content := string(nugetConfig.Content)
	// All existing elements must be present.
	for _, s := range []string{
		"signatureValidationMode",
		"trustedSigners",
		"0E5F38F57DC1BCC806D8494F4F90FBCEDD988B46760709CBEEC6F4219AA6157D",
		"5A2901D6ADA3D18260B9C6DFE2133C95D74B9EEF6AE0E5DC334C8454D1477DF4",
		"1F4B311D9ACC115C8DC8018B5A49E00FCE6DA8E2855F9F014CA6F34570BC482D",
		"<clear>",
	} {
		if !strings.Contains(content, s) {
			t.Errorf("nuget.config missing %q when proxy is set\ncontent:\n%s", s, content)
		}
	}

	// Verify it still parses as valid XML.
	if err := xml.Unmarshal(nugetConfig.Content, new(any)); err != nil {
		t.Errorf("nuget.config with proxy is not valid XML: %v", err)
	}
}

// --- PreCommitHooks tests ---

func TestPreCommitHooks(t *testing.T) {
	m := newModule()
	hooks := m.PreCommitHooks(ecosystem.ModuleConfig{})

	if len(hooks) != 1 {
		t.Fatalf("PreCommitHooks() returned %d hooks, want 1", len(hooks))
	}

	h := hooks[0]
	if h.ID != "dotnet-format" {
		t.Errorf("ID = %q, want %q", h.ID, "dotnet-format")
	}
	if h.Language != "system" {
		t.Errorf("Language = %q, want %q", h.Language, "system")
	}
	// dotnet-format is not a git-hooks.nix built-in, so it must be a custom
	// hook whose binary is resolved from a Nix package.
	if h.BuiltIn {
		t.Error("BuiltIn should be false (dotnet-format is not a git-hooks.nix built-in)")
	}
	if h.NixPackage != "dotnet-sdk" {
		t.Errorf("NixPackage = %q, want %q", h.NixPackage, "dotnet-sdk")
	}
	if h.Files != `\.(cs|fs)$` {
		t.Errorf("Files = %q, want %q", h.Files, `\.(cs|fs)$`)
	}
	if h.Entry != "dotnet format --verify-no-changes" {
		t.Errorf("Entry = %q, want %q", h.Entry, "dotnet format --verify-no-changes")
	}
}

// --- DenyRules tests ---

func TestDenyRules(t *testing.T) {
	m := newModule()
	rules := m.DenyRules(ecosystem.ModuleConfig{})

	if len(rules) != 2 {
		t.Fatalf("DenyRules() returned %d rules, want 2", len(rules))
	}

	expected := map[string]bool{
		"Bash(dotnet add package *)": true,
		"Bash(nuget install *)":      true,
	}
	for _, r := range rules {
		if !expected[r] {
			t.Errorf("unexpected deny rule: %q", r)
		}
	}
}

// --- CICommands tests ---

func TestCICommands(t *testing.T) {
	m := newModule()
	cmds := m.CICommands(ecosystem.ModuleConfig{})

	if len(cmds) != 2 {
		t.Fatalf("CICommands() returned %d commands, want 2", len(cmds))
	}

	foundInstall := false
	foundScan := false
	for _, c := range cmds {
		switch c.Phase {
		case ecosystem.CIPhaseInstall:
			foundInstall = true
			if c.Command != "dotnet restore --locked-mode" {
				t.Errorf("install command = %q, want %q", c.Command, "dotnet restore --locked-mode")
			}
		case ecosystem.CIPhaseScan:
			foundScan = true
			if c.Command != "dotnet list package --vulnerable --include-transitive" {
				t.Errorf("scan command = %q, want %q", c.Command, "dotnet list package --vulnerable --include-transitive")
			}
		default:
			t.Errorf("unexpected phase %v for command %q", c.Phase, c.Name)
		}
	}

	if !foundInstall {
		t.Error("missing CI command with Install phase")
	}
	if !foundScan {
		t.Error("missing CI command with Scan phase")
	}
}

// --- PackageManagers tests ---

func TestPackageManagers(t *testing.T) {
	m := newModule()
	pms := m.PackageManagers()

	if len(pms) != 1 {
		t.Fatalf("PackageManagers() returned %d entries, want 1", len(pms))
	}

	pm := pms[0]
	if pm.Name != "nuget" {
		t.Errorf("Name = %q, want %q", pm.Name, "nuget")
	}
	if pm.LockFile != "packages.lock.json" {
		t.Errorf("LockFile = %q, want %q", pm.LockFile, "packages.lock.json")
	}
	if pm.FrozenInstallCommand != "dotnet restore --locked-mode" {
		t.Errorf("FrozenInstallCommand = %q, want %q", pm.FrozenInstallCommand, "dotnet restore --locked-mode")
	}
	if pm.AuditCommand != "dotnet list package --vulnerable" {
		t.Errorf("AuditCommand = %q, want %q", pm.AuditCommand, "dotnet list package --vulnerable")
	}
	if pm.AgeGatingSupport {
		t.Error("AgeGatingSupport should be false for nuget")
	}
}

// --- WizardFields tests ---

func TestWizardFields(t *testing.T) {
	m := newModule()
	fields := m.WizardFields()

	if len(fields) != 1 {
		t.Fatalf("WizardFields() returned %d fields, want 1", len(fields))
	}

	f := fields[0]
	if f.Key != "dotnet_sdk_version" {
		t.Errorf("Key = %q, want %q", f.Key, "dotnet_sdk_version")
	}
	if f.Type != ecosystem.FieldTypeSelect {
		t.Errorf("Type = %v, want FieldTypeSelect", f.Type)
	}
	if f.Default != "10" {
		t.Errorf("Default = %q, want %q", f.Default, "10")
	}
	// EOL releases (.NET 6, 7) are not offered in the wizard.
	if len(f.Options) != 3 {
		t.Fatalf("Options count = %d, want 3", len(f.Options))
	}

	values := make(map[string]bool)
	for _, o := range f.Options {
		values[o.Value] = true
	}
	for _, v := range []string{"10", "9", "8"} {
		if !values[v] {
			t.Errorf("missing option value %q", v)
		}
	}
	if values["6"] {
		t.Error("option value \"6\" should not be present (EOL)")
	}
}
