package scala_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/scala"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// newModule returns a fresh Module for testing.
func newModule() *scala.Module {
	return &scala.Module{}
}

// --- Interface compliance ---

func TestInterfaceCompliance(t *testing.T) {
	var _ ecosystem.EcosystemModule = (*scala.Module)(nil)
}

// --- Basic metadata ---

func TestModuleIdentity(t *testing.T) {
	ecosystem.AssertModuleIdentity(t, newModule(), "scala", "Scala", 2)
}

// --- Detection tests ---

func TestDetect_BuildSbt(t *testing.T) {
	dir := t.TempDir()
	content := `scalaVersion := "3.3.1"

name := "myproject"
`
	if err := os.WriteFile(filepath.Join(dir, "build.sbt"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	r := m.Detect(dir)

	if !r.Detected {
		t.Fatal("expected Detected = true")
	}
	if r.Confidence < ecosystem.ConfidenceProbable {
		t.Errorf("Confidence = %v, want >= Probable", r.Confidence)
	}
	if len(r.Evidence) == 0 {
		t.Error("expected non-empty Evidence")
	}

	foundBuildSbt := false
	for _, e := range r.Evidence {
		if strings.Contains(e, "build.sbt") {
			foundBuildSbt = true
		}
	}
	if !foundBuildSbt {
		t.Error("Evidence should mention build.sbt")
	}

	if r.SuggestedConfig.Version != "3.3.1" {
		t.Errorf("SuggestedConfig.Version = %q, want %q", r.SuggestedConfig.Version, "3.3.1")
	}
	if bt := r.SuggestedConfig.Extras["build_tool"]; bt != "sbt" {
		t.Errorf("build_tool = %q, want %q", bt, "sbt")
	}
}

func TestDetect_BuildSc(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "build.sc"), []byte("import mill._\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	r := m.Detect(dir)

	if !r.Detected {
		t.Fatal("expected Detected = true")
	}
	if r.Confidence < ecosystem.ConfidenceProbable {
		t.Errorf("Confidence = %v, want >= Probable", r.Confidence)
	}
	if bt := r.SuggestedConfig.Extras["build_tool"]; bt != "mill" {
		t.Errorf("build_tool = %q, want %q", bt, "mill")
	}
}

func TestDetect_WithBuildProperties(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "build.sbt"), []byte("name := \"myproject\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "project"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "project", "build.properties"), []byte("sbt.version=1.9.7\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	r := m.Detect(dir)

	if !r.Detected {
		t.Fatal("expected Detected = true")
	}
	if sv := r.SuggestedConfig.Extras["sbt_version"]; sv != "1.9.7" {
		t.Errorf("sbt_version = %q, want %q", sv, "1.9.7")
	}
}

func TestDetect_NotPresent(t *testing.T) {
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

func TestDevenvNixFragment_NonEmpty(t *testing.T) {
	m := newModule()
	config := ecosystem.ModuleConfig{
		Extras: map[string]string{
			"build_tool":  "sbt",
			"jdk_version": "21",
		},
	}

	frag, err := m.DevenvNixFragment(config)
	if err != nil {
		t.Fatalf("DevenvNixFragment() error: %v", err)
	}
	if frag == "" {
		t.Error("DevenvNixFragment() returned empty string")
	}
	if !strings.Contains(frag, "languages.scala") {
		t.Errorf("fragment missing languages.scala:\n%s", frag)
	}
	if !strings.Contains(frag, "enable = true") {
		t.Errorf("fragment missing enable = true:\n%s", frag)
	}
	if !strings.Contains(frag, "languages.java") {
		t.Errorf("fragment missing languages.java:\n%s", frag)
	}
}

func TestDevenvNixFragment_Mill(t *testing.T) {
	m := newModule()
	config := ecosystem.ModuleConfig{
		Extras: map[string]string{
			"build_tool":  "mill",
			"jdk_version": "17",
		},
	}

	frag, err := m.DevenvNixFragment(config)
	if err != nil {
		t.Fatalf("DevenvNixFragment() error: %v", err)
	}
	// Mill comes from devenv's languages.scala.mill (built against the
	// project JDK), not a bare pkgs.mill.
	if strings.Contains(frag, "packages") || !strings.Contains(frag, "mill.enable = true;") {
		t.Errorf("Mill fragment should enable languages.scala.mill and add no packages:\n%s", frag)
	}
	if !strings.Contains(frag, "jdk17") {
		t.Errorf("fragment missing jdk17:\n%s", frag)
	}
	if !strings.Contains(frag, "languages.scala") {
		t.Errorf("fragment missing languages.scala:\n%s", frag)
	}
}

// --- SecurityConfigs tests ---

func TestSecurityConfigs(t *testing.T) {
	m := newModule()
	files := m.SecurityConfigs(ecosystem.ModuleConfig{})

	if len(files) != 1 {
		t.Fatalf("SecurityConfigs() returned %d files, want 1", len(files))
	}
	if files[0].Path != ".qsdev/sbt-security-plugins.sbt" {
		t.Errorf("Path = %q, want %q", files[0].Path, ".qsdev/sbt-security-plugins.sbt")
	}
}

// --- PreCommitHooks tests ---

func TestPreCommitHooks(t *testing.T) {
	m := newModule()
	hooks := m.PreCommitHooks(ecosystem.ModuleConfig{})

	if len(hooks) != 1 {
		t.Fatalf("PreCommitHooks() returned %d hooks, want 1", len(hooks))
	}
	if hooks[0].ID != "scalafmt" {
		t.Errorf("hook ID = %q, want %q", hooks[0].ID, "scalafmt")
	}
}

// --- DenyRules tests ---

func TestDenyRules(t *testing.T) {
	m := newModule()
	rules := m.DenyRules(ecosystem.ModuleConfig{})

	if len(rules) != 2 {
		t.Fatalf("DenyRules() returned %d rules, want 2", len(rules))
	}
}

// --- CICommands tests ---

func TestCICommands(t *testing.T) {
	m := newModule()
	cmds := m.CICommands(ecosystem.ModuleConfig{})

	if len(cmds) != 2 {
		t.Fatalf("CICommands() returned %d commands, want 2", len(cmds))
	}
}

// --- PackageManagers tests ---

func TestPackageManagers(t *testing.T) {
	m := newModule()
	pms := m.PackageManagers()

	if len(pms) != 1 {
		t.Fatalf("PackageManagers() returned %d entries, want 1", len(pms))
	}
	if pms[0].Name != "sbt" {
		t.Errorf("Name = %q, want %q", pms[0].Name, "sbt")
	}
	if pms[0].LockFile != "build.sbt.lock" {
		t.Errorf("LockFile = %q, want %q", pms[0].LockFile, "build.sbt.lock")
	}
}

// --- WizardFields tests ---

func TestWizardFields(t *testing.T) {
	m := newModule()
	fields := m.WizardFields()

	if len(fields) != 2 {
		t.Fatalf("WizardFields() returned %d fields, want 2", len(fields))
	}

	keys := make(map[string]bool)
	for _, f := range fields {
		keys[f.Key] = true
	}
	for _, key := range []string{"build_tool", "jdk_version"} {
		if !keys[key] {
			t.Errorf("missing wizard field %s", key)
		}
	}
}

// TestScalaSbtProxyWiring checks sbt resolves through the registry proxy: a
// committed repositories file (the qsdev directory is gitignored, so CI
// checkouts would lack it there) that the devenv shell points sbt at through
// SBT_OPTS. Without a proxy, and for Mill, neither is generated.
func TestScalaSbtProxyWiring(t *testing.T) {
	t.Parallel()
	const (
		proxy    = "https://proxy.corp.internal/repository/maven-central/"
		repoPath = "project/qsdev.repositories"
		sbtOpts  = `env.SBT_OPTS = "-Dsbt.repository.config=${config.devenv.root}/project/qsdev.repositories -Dsbt.override.build.repos=true";`
	)
	tests := []struct {
		name  string
		cfg   ecosystem.ModuleConfig
		wired bool
	}{
		{"sbt with proxy", ecosystem.ModuleConfig{RegistryProxy: proxy}, true},
		{"explicit sbt with proxy", ecosystem.ModuleConfig{RegistryProxy: proxy, Extras: map[string]string{"build_tool": "sbt"}}, true},
		{"sbt without proxy", ecosystem.ModuleConfig{}, false},
		{"mill with proxy", ecosystem.ModuleConfig{RegistryProxy: proxy, Extras: map[string]string{"build_tool": "mill"}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := newModule()

			var repos *types.GeneratedFile
			for _, f := range m.SecurityConfigs(tt.cfg) {
				if f.Path == repoPath {
					repos = &f
				}
			}
			if (repos != nil) != tt.wired {
				t.Fatalf("SecurityConfigs writes %s = %v, want %v", repoPath, repos != nil, tt.wired)
			}
			if repos != nil {
				if repos.Strategy != types.Overwrite {
					t.Errorf("%s Strategy = %v, want Overwrite", repoPath, repos.Strategy)
				}
				want := "[repositories]\n  local\n  proxy: " + proxy + "\n"
				if !strings.HasSuffix(string(repos.Content), want) {
					t.Errorf("%s content = %q, want it to end with %q", repoPath, repos.Content, want)
				}
			}

			frag, err := m.DevenvNixFragment(tt.cfg)
			if err != nil {
				t.Fatalf("DevenvNixFragment: %v", err)
			}
			if got := strings.Contains(frag, sbtOpts); got != tt.wired {
				t.Errorf("fragment contains %q = %v, want %v:\n%s", sbtOpts, got, tt.wired, frag)
			}
			if !tt.wired && strings.Contains(frag, "SBT_OPTS") {
				t.Errorf("fragment sets SBT_OPTS without sbt proxy wiring:\n%s", frag)
			}
		})
	}
}

// TestProxyKey checks Scala routes through the Maven proxy only for sbt.
func TestProxyKey(t *testing.T) {
	t.Parallel()
	m := newModule()
	for _, tt := range []struct {
		buildTool, want string
	}{{"", "maven"}, {"sbt", "maven"}, {"mill", ""}} {
		cfg := ecosystem.ModuleConfig{}
		if tt.buildTool != "" {
			cfg.Extras = map[string]string{"build_tool": tt.buildTool}
		}
		if got := m.ProxyKey(cfg); got != tt.want {
			t.Errorf("ProxyKey(build_tool=%q) = %q, want %q", tt.buildTool, got, tt.want)
		}
	}
}
