package java_test

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/java"
)

// Compile-time interface compliance check.
var _ ecosystem.EcosystemModule = (*java.Module)(nil)

func TestName(t *testing.T) {
	m := &java.Module{}
	if got := m.Name(); got != "java" {
		t.Errorf("Name() = %q, want %q", got, "java")
	}
}

func TestDisplayName(t *testing.T) {
	m := &java.Module{}
	if got := m.DisplayName(); got != "Java/Kotlin (JVM)" {
		t.Errorf("DisplayName() = %q, want %q", got, "Java/Kotlin (JVM)")
	}
}

func TestTier(t *testing.T) {
	m := &java.Module{}
	if got := m.Tier(); got != 1 {
		t.Errorf("Tier() = %d, want %d", got, 1)
	}
}

// ---------------------------------------------------------------------------
// Detection tests
// ---------------------------------------------------------------------------

func TestDetect_MavenOnly(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pom.xml", "<project/>")

	m := &java.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true when pom.xml is present")
	}
	if result.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("Confidence = %v, want ConfidenceCertain", result.Confidence)
	}
	if result.SuggestedConfig.Extras["build_tool"] != "maven" {
		t.Errorf("build_tool = %q, want %q", result.SuggestedConfig.Extras["build_tool"], "maven")
	}
	assertEvidenceContains(t, result.Evidence, "pom.xml")
}

func TestDetect_GradleOnly(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "build.gradle", "apply plugin: 'java'")

	m := &java.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true when build.gradle is present")
	}
	if result.SuggestedConfig.Extras["build_tool"] != "gradle" {
		t.Errorf("build_tool = %q, want %q", result.SuggestedConfig.Extras["build_tool"], "gradle")
	}
	assertEvidenceContains(t, result.Evidence, "Gradle")
}

func TestDetect_GradleKts(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "build.gradle.kts", `plugins { java }`)

	m := &java.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true when build.gradle.kts is present")
	}
	if result.SuggestedConfig.Extras["build_tool"] != "gradle" {
		t.Errorf("build_tool = %q, want %q", result.SuggestedConfig.Extras["build_tool"], "gradle")
	}
}

func TestDetect_SettingsGradle(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "settings.gradle", "rootProject.name = 'test'")

	m := &java.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true when settings.gradle is present")
	}
	if result.SuggestedConfig.Extras["build_tool"] != "gradle" {
		t.Errorf("build_tool = %q, want %q", result.SuggestedConfig.Extras["build_tool"], "gradle")
	}
}

func TestDetect_SettingsGradleKts(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "settings.gradle.kts", `rootProject.name = "test"`)

	m := &java.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true when settings.gradle.kts is present")
	}
	if result.SuggestedConfig.Extras["build_tool"] != "gradle" {
		t.Errorf("build_tool = %q, want %q", result.SuggestedConfig.Extras["build_tool"], "gradle")
	}
}

func TestDetect_Both(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pom.xml", "<project/>")
	writeFile(t, dir, "build.gradle", "apply plugin: 'java'")

	m := &java.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true when both are present")
	}
	if result.SuggestedConfig.Extras["build_tool"] != "both" {
		t.Errorf("build_tool = %q, want %q", result.SuggestedConfig.Extras["build_tool"], "both")
	}
	assertEvidenceContains(t, result.Evidence, "pom.xml")
	assertEvidenceContains(t, result.Evidence, "Gradle")
}

func TestDetect_JavaVersion(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pom.xml", "<project/>")
	writeFile(t, dir, ".java-version", "17\n")

	m := &java.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true")
	}
	if result.SuggestedConfig.Version != "17" {
		t.Errorf("Version = %q, want %q", result.SuggestedConfig.Version, "17")
	}
	assertEvidenceContains(t, result.Evidence, "17")
}

func TestDetect_KotlinFromKtFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "build.gradle", "apply plugin: 'java'")
	writeFile(t, dir, "Main.kt", "fun main() {}")

	m := &java.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true")
	}
	if result.SuggestedConfig.Extras["kotlin"] != "true" {
		t.Errorf("kotlin = %q, want %q", result.SuggestedConfig.Extras["kotlin"], "true")
	}
	assertEvidenceContains(t, result.Evidence, "Kotlin")
}

func TestDetect_KotlinFromBuildGradle(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "build.gradle", `
plugins {
    id 'org.jetbrains.kotlin.jvm' version '1.9.0'
}
`)

	m := &java.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true")
	}
	if result.SuggestedConfig.Extras["kotlin"] != "true" {
		t.Errorf("kotlin = %q, want %q", result.SuggestedConfig.Extras["kotlin"], "true")
	}
}

func TestDetect_KotlinFromBuildGradleKts(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "build.gradle.kts", `
plugins {
    kotlin("jvm") version "1.9.0"
}
`)

	m := &java.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true")
	}
	if result.SuggestedConfig.Extras["kotlin"] != "true" {
		t.Errorf("kotlin = %q, want %q", result.SuggestedConfig.Extras["kotlin"], "true")
	}
}

func TestDetect_NoKotlin(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "build.gradle", "apply plugin: 'java'")

	m := &java.Module{}
	result := m.Detect(dir)

	if result.SuggestedConfig.Extras["kotlin"] != "false" {
		t.Errorf("kotlin = %q, want %q", result.SuggestedConfig.Extras["kotlin"], "false")
	}
}

func TestDetect_NoFiles(t *testing.T) {
	dir := t.TempDir()

	m := &java.Module{}
	result := m.Detect(dir)

	if result.Detected {
		t.Error("expected Detected=false when no JVM files present")
	}
	if result.Confidence != ecosystem.ConfidenceAbsent {
		t.Errorf("Confidence = %v, want ConfidenceAbsent", result.Confidence)
	}
}

// ---------------------------------------------------------------------------
// DevenvNixFragment tests
// ---------------------------------------------------------------------------

func TestDevenvNixFragment_MavenJDK21(t *testing.T) {
	m := &java.Module{}
	cfg := ecosystem.ModuleConfig{
		Version: "21",
		Extras:  map[string]string{"build_tool": "maven", "kotlin": "false"},
	}
	fragment, err := m.DevenvNixFragment(cfg)
	if err != nil {
		t.Fatalf("DevenvNixFragment() error: %v", err)
	}

	assertContains(t, fragment, "languages.java")
	assertContains(t, fragment, "enable = true")
	assertContains(t, fragment, "pkgs.jdk21")
	assertContains(t, fragment, "maven.enable = true")
	assertNotContains(t, fragment, "gradle.enable")
	assertNotContains(t, fragment, "languages.kotlin")
}

func TestDevenvNixFragment_GradleJDK17(t *testing.T) {
	m := &java.Module{}
	cfg := ecosystem.ModuleConfig{
		Version: "17",
		Extras:  map[string]string{"build_tool": "gradle", "kotlin": "false"},
	}
	fragment, err := m.DevenvNixFragment(cfg)
	if err != nil {
		t.Fatalf("DevenvNixFragment() error: %v", err)
	}

	assertContains(t, fragment, "pkgs.jdk17")
	assertContains(t, fragment, "gradle.enable = true")
	assertNotContains(t, fragment, "maven.enable")
	assertNotContains(t, fragment, "languages.kotlin")
}

func TestDevenvNixFragment_Both(t *testing.T) {
	m := &java.Module{}
	cfg := ecosystem.ModuleConfig{
		Version: "11",
		Extras:  map[string]string{"build_tool": "both", "kotlin": "false"},
	}
	fragment, err := m.DevenvNixFragment(cfg)
	if err != nil {
		t.Fatalf("DevenvNixFragment() error: %v", err)
	}

	assertContains(t, fragment, "pkgs.jdk11")
	assertContains(t, fragment, "maven.enable = true")
	assertContains(t, fragment, "gradle.enable = true")
}

func TestDevenvNixFragment_WithKotlin(t *testing.T) {
	m := &java.Module{}
	cfg := ecosystem.ModuleConfig{
		Version: "21",
		Extras:  map[string]string{"build_tool": "gradle", "kotlin": "true"},
	}
	fragment, err := m.DevenvNixFragment(cfg)
	if err != nil {
		t.Fatalf("DevenvNixFragment() error: %v", err)
	}

	assertContains(t, fragment, "languages.kotlin.enable = true")
}

func TestDevenvNixFragment_DefaultVersion(t *testing.T) {
	m := &java.Module{}
	cfg := ecosystem.ModuleConfig{
		Version: "",
		Extras:  map[string]string{"build_tool": "maven", "kotlin": "false"},
	}
	fragment, err := m.DevenvNixFragment(cfg)
	if err != nil {
		t.Fatalf("DevenvNixFragment() error: %v", err)
	}

	// Default should map to jdk21.
	assertContains(t, fragment, "pkgs.jdk21")
}

// ---------------------------------------------------------------------------
// DevenvYamlInputs test
// ---------------------------------------------------------------------------

func TestDevenvYamlInputs(t *testing.T) {
	m := &java.Module{}
	inputs := m.DevenvYamlInputs(ecosystem.ModuleConfig{})
	if inputs != nil {
		t.Errorf("DevenvYamlInputs() = %v, want nil", inputs)
	}
}

// ---------------------------------------------------------------------------
// SecurityConfigs tests
// ---------------------------------------------------------------------------

func TestSecurityConfigs_MavenOnly(t *testing.T) {
	m := &java.Module{}
	cfg := ecosystem.ModuleConfig{
		Extras: map[string]string{"build_tool": "maven", "kotlin": "false"},
	}
	files := m.SecurityConfigs(cfg)

	if len(files) != 1 {
		t.Fatalf("SecurityConfigs() returned %d files, want 1", len(files))
	}

	f := files[0]
	if f.Path != ".mvn/settings.xml" {
		t.Errorf("Path = %q, want %q", f.Path, ".mvn/settings.xml")
	}
	content := string(f.Content)
	assertContains(t, content, "checksumPolicy")
	assertContains(t, content, "fail")
	assertContains(t, content, "<enabled>false</enabled>")
	assertContains(t, content, "mirrorOf")
	assertContains(t, content, "<?xml version")
	assertContains(t, content, "Maven >= 3.2.5")
}

func TestSecurityConfigs_GradleOnly(t *testing.T) {
	m := &java.Module{}
	cfg := ecosystem.ModuleConfig{
		Extras: map[string]string{"build_tool": "gradle", "kotlin": "false"},
	}
	files := m.SecurityConfigs(cfg)

	if len(files) != 1 {
		t.Fatalf("SecurityConfigs() returned %d files, want 1", len(files))
	}

	f := files[0]
	if f.Path != "gradle.properties" {
		t.Errorf("Path = %q, want %q", f.Path, "gradle.properties")
	}
	content := string(f.Content)
	assertContains(t, content, "dependencyLocking.lockMode=STRICT")
	assertContains(t, content, "systemProp.org.gradle.dependency.verification=strict")
}

func TestSecurityConfigs_Both(t *testing.T) {
	m := &java.Module{}
	cfg := ecosystem.ModuleConfig{
		Extras: map[string]string{"build_tool": "both", "kotlin": "false"},
	}
	files := m.SecurityConfigs(cfg)

	if len(files) != 2 {
		t.Fatalf("SecurityConfigs() returned %d files, want 2", len(files))
	}

	paths := make(map[string]bool)
	for _, f := range files {
		paths[f.Path] = true
	}
	if !paths[".mvn/settings.xml"] {
		t.Error("expected .mvn/settings.xml in generated files")
	}
	if !paths["gradle.properties"] {
		t.Error("expected gradle.properties in generated files")
	}
}

func TestSecurityConfigs_XMLRoundTrip(t *testing.T) {
	m := &java.Module{}
	cfg := ecosystem.ModuleConfig{
		Extras: map[string]string{"build_tool": "maven", "kotlin": "false"},
	}
	files := m.SecurityConfigs(cfg)
	if len(files) == 0 {
		t.Fatal("expected at least one generated file")
	}

	content := files[0].Content
	// Find the start of the XML document (after the comment header).
	xmlStart := strings.Index(string(content), "<settings")
	if xmlStart < 0 {
		t.Fatal("could not find <settings element in generated XML")
	}

	// Parse the XML portion back into a struct.
	type settingsRoundTrip struct {
		XMLName xml.Name `xml:"settings"`
		Xmlns   string   `xml:"xmlns,attr"`
	}
	var s settingsRoundTrip
	if err := xml.Unmarshal(content[xmlStart:], &s); err != nil {
		t.Fatalf("failed to unmarshal generated settings.xml: %v", err)
	}
	if s.Xmlns == "" {
		t.Error("expected xmlns attribute in settings element")
	}
}

func TestSecurityConfigs_SettingsXMLChecksumPolicy(t *testing.T) {
	m := &java.Module{}
	cfg := ecosystem.ModuleConfig{
		Extras: map[string]string{"build_tool": "maven", "kotlin": "false"},
	}
	files := m.SecurityConfigs(cfg)
	content := string(files[0].Content)

	// Verify checksum policy is "fail".
	if !strings.Contains(content, "<checksumPolicy>fail</checksumPolicy>") {
		t.Error("expected checksumPolicy=fail in settings.xml")
	}
}

func TestSecurityConfigs_SettingsXMLSnapshotBlocking(t *testing.T) {
	m := &java.Module{}
	cfg := ecosystem.ModuleConfig{
		Extras: map[string]string{"build_tool": "maven", "kotlin": "false"},
	}
	files := m.SecurityConfigs(cfg)
	content := string(files[0].Content)

	// Verify snapshots are disabled.
	// The XML should contain a <snapshots> block with <enabled>false</enabled>.
	if !strings.Contains(content, "<enabled>false</enabled>") {
		t.Error("expected snapshots disabled (enabled=false) in settings.xml")
	}
}

func TestSecurityConfigs_SettingsXMLMirrorConfig(t *testing.T) {
	m := &java.Module{}
	cfg := ecosystem.ModuleConfig{
		Extras: map[string]string{"build_tool": "maven", "kotlin": "false"},
	}
	files := m.SecurityConfigs(cfg)
	content := string(files[0].Content)

	// Verify mirror blocks non-central repos.
	if !strings.Contains(content, "<mirrorOf>*</mirrorOf>") {
		t.Error("expected mirrorOf=* in settings.xml to block non-central repos")
	}
	if !strings.Contains(content, "repo.maven.apache.org") {
		t.Error("expected Maven Central URL in mirror configuration")
	}
}

func TestSecurityConfigs_GradlePropertiesContent(t *testing.T) {
	m := &java.Module{}
	cfg := ecosystem.ModuleConfig{
		Extras: map[string]string{"build_tool": "gradle", "kotlin": "false"},
	}
	files := m.SecurityConfigs(cfg)
	content := string(files[0].Content)

	assertContains(t, content, "dependencyLocking.lockMode=STRICT")
	assertContains(t, content, "systemProp.org.gradle.dependency.verification=strict")
	// Should have comment header.
	assertContains(t, content, "gdev-secure-devenv-bootstrap")
	assertContains(t, content, "Gradle >= 6.1")
}

// ---------------------------------------------------------------------------
// PreCommitHooks tests
// ---------------------------------------------------------------------------

func TestPreCommitHooks_JavaOnly(t *testing.T) {
	m := &java.Module{}
	cfg := ecosystem.ModuleConfig{
		Extras: map[string]string{"build_tool": "maven", "kotlin": "false"},
	}
	hooks := m.PreCommitHooks(cfg)

	if len(hooks) != 2 {
		t.Fatalf("PreCommitHooks() returned %d hooks, want 2", len(hooks))
	}

	expectedIDs := []string{"google-java-format", "spotbugs"}
	for i, hook := range hooks {
		if hook.ID != expectedIDs[i] {
			t.Errorf("hooks[%d].ID = %q, want %q", i, hook.ID, expectedIDs[i])
		}
		if hook.Language != "system" {
			t.Errorf("hooks[%d].Language = %q, want %q", i, hook.Language, "system")
		}
	}
}

func TestPreCommitHooks_WithKotlin(t *testing.T) {
	m := &java.Module{}
	cfg := ecosystem.ModuleConfig{
		Extras: map[string]string{"build_tool": "gradle", "kotlin": "true"},
	}
	hooks := m.PreCommitHooks(cfg)

	if len(hooks) != 3 {
		t.Fatalf("PreCommitHooks() returned %d hooks, want 3", len(hooks))
	}

	expectedIDs := []string{"google-java-format", "spotbugs", "ktlint"}
	for i, hook := range hooks {
		if hook.ID != expectedIDs[i] {
			t.Errorf("hooks[%d].ID = %q, want %q", i, hook.ID, expectedIDs[i])
		}
	}

	// Verify ktlint targets Kotlin files.
	ktlint := hooks[2]
	if len(ktlint.Types) != 1 || ktlint.Types[0] != "kotlin" {
		t.Errorf("ktlint.Types = %v, want [\"kotlin\"]", ktlint.Types)
	}
	if ktlint.Files == "" {
		t.Error("ktlint.Files should have a pattern for Kotlin files")
	}
}

// ---------------------------------------------------------------------------
// DenyRules tests
// ---------------------------------------------------------------------------

func TestDenyRules_Maven(t *testing.T) {
	m := &java.Module{}
	cfg := ecosystem.ModuleConfig{
		Extras: map[string]string{"build_tool": "maven", "kotlin": "false"},
	}
	rules := m.DenyRules(cfg)

	if len(rules) != 2 {
		t.Fatalf("DenyRules() returned %d rules, want 2", len(rules))
	}
	expected := []string{
		"Bash(mvn install *)",
		"Bash(mvn dependency:resolve *)",
	}
	for i, rule := range rules {
		if rule != expected[i] {
			t.Errorf("rules[%d] = %q, want %q", i, rule, expected[i])
		}
	}
}

func TestDenyRules_Gradle(t *testing.T) {
	m := &java.Module{}
	cfg := ecosystem.ModuleConfig{
		Extras: map[string]string{"build_tool": "gradle", "kotlin": "false"},
	}
	rules := m.DenyRules(cfg)

	if len(rules) != 2 {
		t.Fatalf("DenyRules() returned %d rules, want 2", len(rules))
	}
	expected := []string{
		"Bash(gradle dependencies *)",
		"Bash(./gradlew dependencies *)",
	}
	for i, rule := range rules {
		if rule != expected[i] {
			t.Errorf("rules[%d] = %q, want %q", i, rule, expected[i])
		}
	}
}

func TestDenyRules_Both(t *testing.T) {
	m := &java.Module{}
	cfg := ecosystem.ModuleConfig{
		Extras: map[string]string{"build_tool": "both", "kotlin": "false"},
	}
	rules := m.DenyRules(cfg)

	if len(rules) != 4 {
		t.Fatalf("DenyRules() returned %d rules, want 4", len(rules))
	}
	expected := []string{
		"Bash(mvn install *)",
		"Bash(mvn dependency:resolve *)",
		"Bash(gradle dependencies *)",
		"Bash(./gradlew dependencies *)",
	}
	for i, rule := range rules {
		if rule != expected[i] {
			t.Errorf("rules[%d] = %q, want %q", i, rule, expected[i])
		}
	}
}

// ---------------------------------------------------------------------------
// CICommands tests
// ---------------------------------------------------------------------------

func TestCICommands_Maven(t *testing.T) {
	m := &java.Module{}
	cfg := ecosystem.ModuleConfig{
		Extras: map[string]string{"build_tool": "maven", "kotlin": "false"},
	}
	cmds := m.CICommands(cfg)

	if len(cmds) != 1 {
		t.Fatalf("CICommands() returned %d commands, want 1", len(cmds))
	}
	if cmds[0].Command != "mvn verify --strict-checksums" {
		t.Errorf("Command = %q, want %q", cmds[0].Command, "mvn verify --strict-checksums")
	}
	if cmds[0].Phase != ecosystem.CIPhaseTest {
		t.Errorf("Phase = %v, want CIPhaseTest", cmds[0].Phase)
	}
}

func TestCICommands_Gradle(t *testing.T) {
	m := &java.Module{}
	cfg := ecosystem.ModuleConfig{
		Extras: map[string]string{"build_tool": "gradle", "kotlin": "false"},
	}
	cmds := m.CICommands(cfg)

	if len(cmds) != 2 {
		t.Fatalf("CICommands() returned %d commands, want 2", len(cmds))
	}
	if cmds[0].Command != "./gradlew build" {
		t.Errorf("cmds[0].Command = %q, want %q", cmds[0].Command, "./gradlew build")
	}
	if cmds[1].Command != "./gradlew --write-verification-metadata sha256,pgp" {
		t.Errorf("cmds[1].Command = %q, want %q", cmds[1].Command, "./gradlew --write-verification-metadata sha256,pgp")
	}
	if cmds[0].Phase != ecosystem.CIPhaseTest {
		t.Errorf("cmds[0].Phase = %v, want CIPhaseTest", cmds[0].Phase)
	}
	if cmds[1].Phase != ecosystem.CIPhaseScan {
		t.Errorf("cmds[1].Phase = %v, want CIPhaseScan", cmds[1].Phase)
	}
}

func TestCICommands_Both(t *testing.T) {
	m := &java.Module{}
	cfg := ecosystem.ModuleConfig{
		Extras: map[string]string{"build_tool": "both", "kotlin": "false"},
	}
	cmds := m.CICommands(cfg)

	if len(cmds) != 3 {
		t.Fatalf("CICommands() returned %d commands, want 3", len(cmds))
	}
	// First should be Maven.
	if cmds[0].Command != "mvn verify --strict-checksums" {
		t.Errorf("cmds[0].Command = %q, want Maven verify", cmds[0].Command)
	}
	// Then Gradle build + verification metadata.
	if cmds[1].Command != "./gradlew build" {
		t.Errorf("cmds[1].Command = %q, want Gradle build", cmds[1].Command)
	}
	if cmds[2].Command != "./gradlew --write-verification-metadata sha256,pgp" {
		t.Errorf("cmds[2].Command = %q, want Gradle verification metadata", cmds[2].Command)
	}
}

// ---------------------------------------------------------------------------
// PackageManagers tests
// ---------------------------------------------------------------------------

func TestPackageManagers(t *testing.T) {
	m := &java.Module{}
	pms := m.PackageManagers()

	if len(pms) != 2 {
		t.Fatalf("PackageManagers() returned %d entries, want 2", len(pms))
	}

	maven := pms[0]
	if maven.Name != "maven" {
		t.Errorf("pms[0].Name = %q, want %q", maven.Name, "maven")
	}
	if maven.LockFile != "pom.xml" {
		t.Errorf("pms[0].LockFile = %q, want %q", maven.LockFile, "pom.xml")
	}
	if maven.AuditCommand == "" {
		t.Error("pms[0].AuditCommand should not be empty")
	}

	gradle := pms[1]
	if gradle.Name != "gradle" {
		t.Errorf("pms[1].Name = %q, want %q", gradle.Name, "gradle")
	}
	if gradle.LockFile != "gradle.lockfile" {
		t.Errorf("pms[1].LockFile = %q, want %q", gradle.LockFile, "gradle.lockfile")
	}
	if gradle.AuditCommand == "" {
		t.Error("pms[1].AuditCommand should not be empty")
	}
}

// ---------------------------------------------------------------------------
// WizardFields tests
// ---------------------------------------------------------------------------

func TestWizardFields(t *testing.T) {
	m := &java.Module{}
	fields := m.WizardFields()

	if len(fields) != 3 {
		t.Fatalf("WizardFields() returned %d fields, want 3", len(fields))
	}

	// Build tool select.
	bt := fields[0]
	if bt.Key != "java_build_tool" {
		t.Errorf("fields[0].Key = %q, want %q", bt.Key, "java_build_tool")
	}
	if bt.Type != ecosystem.FieldTypeSelect {
		t.Errorf("fields[0].Type = %v, want FieldTypeSelect", bt.Type)
	}
	if len(bt.Options) < 2 {
		t.Errorf("fields[0].Options has %d entries, want at least 2", len(bt.Options))
	}

	// JDK version select.
	jdk := fields[1]
	if jdk.Key != "java_jdk_version" {
		t.Errorf("fields[1].Key = %q, want %q", jdk.Key, "java_jdk_version")
	}
	if jdk.Type != ecosystem.FieldTypeSelect {
		t.Errorf("fields[1].Type = %v, want FieldTypeSelect", jdk.Type)
	}
	// Should have 21, 17, 11 options.
	if len(jdk.Options) != 3 {
		t.Errorf("fields[1].Options has %d entries, want 3", len(jdk.Options))
	}
	foundVersions := make(map[string]bool)
	for _, opt := range jdk.Options {
		foundVersions[opt.Value] = true
	}
	for _, v := range []string{"21", "17", "11"} {
		if !foundVersions[v] {
			t.Errorf("JDK version options missing %q", v)
		}
	}

	// Kotlin confirm.
	kt := fields[2]
	if kt.Key != "java_kotlin" {
		t.Errorf("fields[2].Key = %q, want %q", kt.Key, "java_kotlin")
	}
	if kt.Type != ecosystem.FieldTypeConfirm {
		t.Errorf("fields[2].Type = %v, want FieldTypeConfirm", kt.Type)
	}
}

// ---------------------------------------------------------------------------
// Registration test
// ---------------------------------------------------------------------------

func TestRegistration(t *testing.T) {
	reg := ecosystem.DefaultRegistry()
	mod, ok := reg.ByName("java")
	if !ok {
		t.Fatal("expected module 'java' to be registered in DefaultRegistry")
	}
	if mod.Name() != "java" {
		t.Errorf("registered module Name() = %q, want %q", mod.Name(), "java")
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("expected string to contain %q, got:\n%s", substr, s)
	}
}

func assertNotContains(t *testing.T, s, substr string) {
	t.Helper()
	if strings.Contains(s, substr) {
		t.Errorf("expected string NOT to contain %q, got:\n%s", substr, s)
	}
}

func assertEvidenceContains(t *testing.T, evidence []string, substr string) {
	t.Helper()
	for _, e := range evidence {
		if strings.Contains(e, substr) {
			return
		}
	}
	t.Errorf("evidence %v should contain an entry mentioning %q", evidence, substr)
}
