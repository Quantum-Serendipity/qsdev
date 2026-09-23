package java_test

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/java"
)

// TestBuildTool_PackageManagerHonoured verifies that the build tool set via
// --java-build-tool (stored as PackageManager) drives every generator, and
// that all methods agree on the tool when nothing is configured.
func TestBuildTool_PackageManagerHonoured(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		cfg        ecosystem.ModuleConfig
		wantMaven  bool
		wantGradle bool
	}{
		{"pm gradle", ecosystem.ModuleConfig{PackageManager: "gradle"}, false, true},
		{"pm maven", ecosystem.ModuleConfig{PackageManager: "maven"}, true, false},
		{"pm both", ecosystem.ModuleConfig{PackageManager: "both"}, true, true},
		{"pm wins over extra", ecosystem.ModuleConfig{PackageManager: "gradle", Extras: map[string]string{"build_tool": "maven"}}, false, true},
		{"extra only", ecosystem.ModuleConfig{Extras: map[string]string{"build_tool": "gradle"}}, false, true},
		{"unset defaults to maven", ecosystem.ModuleConfig{}, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := &java.Module{}

			frag, err := m.DevenvNixFragment(tt.cfg)
			if err != nil {
				t.Fatalf("DevenvNixFragment() error: %v", err)
			}
			if got := strings.Contains(frag, "maven.enable = true;"); got != tt.wantMaven {
				t.Errorf("fragment maven.enable = %v, want %v\n%s", got, tt.wantMaven, frag)
			}
			if got := strings.Contains(frag, "gradle.enable = true;"); got != tt.wantGradle {
				t.Errorf("fragment gradle.enable = %v, want %v\n%s", got, tt.wantGradle, frag)
			}

			paths := map[string]bool{}
			for _, f := range m.SecurityConfigs(tt.cfg) {
				paths[f.Path] = true
			}
			if paths[".mvn/settings.xml"] != tt.wantMaven {
				t.Errorf("settings.xml generated = %v, want %v", paths[".mvn/settings.xml"], tt.wantMaven)
			}
			if paths["gradle.properties"] != tt.wantGradle {
				t.Errorf("gradle.properties generated = %v, want %v", paths["gradle.properties"], tt.wantGradle)
			}

			rules := strings.Join(m.DenyRules(tt.cfg), "\n")
			if got := strings.Contains(rules, "mvn "); got != tt.wantMaven {
				t.Errorf("maven deny rules present = %v, want %v", got, tt.wantMaven)
			}
			if got := strings.Contains(rules, "gradle"); got != tt.wantGradle {
				t.Errorf("gradle deny rules present = %v, want %v", got, tt.wantGradle)
			}

			build := strings.Join(m.VerificationCommands(tt.cfg).Build, "\n")
			if got := strings.Contains(build, "mvn "); got != tt.wantMaven {
				t.Errorf("maven verification present = %v, want %v (%q)", got, tt.wantMaven, build)
			}
			if got := strings.Contains(build, "gradlew"); got != tt.wantGradle {
				t.Errorf("gradle verification present = %v, want %v (%q)", got, tt.wantGradle, build)
			}

			manifests := map[string]bool{}
			for _, mf := range m.ManifestFiles(tt.cfg) {
				manifests[mf.Path] = true
			}
			if manifests["pom.xml"] != tt.wantMaven || manifests["build.gradle"] != tt.wantGradle {
				t.Errorf("ManifestFiles() = %v, want pom.xml=%v build.gradle=%v", manifests, tt.wantMaven, tt.wantGradle)
			}
		})
	}
}

func TestDevenvNixFragment_InvalidBuildTool(t *testing.T) {
	t.Parallel()
	m := &java.Module{}
	_, err := m.DevenvNixFragment(ecosystem.ModuleConfig{PackageManager: "ant"})
	if err == nil {
		t.Fatal("DevenvNixFragment() with unsupported build tool: want error, got nil")
	}
	if !strings.Contains(err.Error(), "ant") {
		t.Errorf("error %q should name the rejected value", err)
	}
}
