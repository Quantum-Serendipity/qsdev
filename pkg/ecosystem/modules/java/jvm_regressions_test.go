package java_test

import (
	"encoding/xml"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/java"
)

// TestDevenvNixFragment_JDKVersion checks that the requested Java version is
// provisioned, including the spellings .java-version files use, and that a
// JDK nixpkgs does not package is an error instead of a silent jdk21.
func TestDevenvNixFragment_JDKVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		version string
		want    string // "" means an error is expected
	}{
		{"25", "pkgs.jdk25"},
		{"8", "pkgs.jdk8"},
		{"1.8", "pkgs.jdk8"},
		{"17.0.2", "pkgs.jdk17"},
		{"temurin-21.0.1", "pkgs.jdk21"},
		{"22", ""},
		{"26", ""},
	}
	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			t.Parallel()
			frag, err := (&java.Module{}).DevenvNixFragment(ecosystem.ModuleConfig{
				Version: tt.version,
				Extras:  map[string]string{"build_tool": "maven"},
			})
			if tt.want == "" {
				if !errors.Is(err, ecosystem.ErrUnsupportedJDKVersion) {
					t.Fatalf("DevenvNixFragment(%q) = %q, %v; want ErrUnsupportedJDKVersion", tt.version, frag, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("DevenvNixFragment(%q): %v", tt.version, err)
			}
			assertContains(t, frag, "languages.java.jdk.package = "+tt.want+";")
		})
	}
}

// TestDetect_JavaVersionFile checks that a .java-version naming a JDK that
// cannot be provisioned is not suggested (it would fail generation), and the
// evidence says why.
func TestDetect_JavaVersionFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		content     string
		wantVersion string
		wantEvid    string
	}{
		{"17.0.2\n", "17.0.2", "Java version 17.0.2"},
		{"1.8\n", "1.8", "Java version 1.8"},
		{"22\n", "", "ignored"},
	}
	for _, tt := range tests {
		t.Run(strings.TrimSpace(tt.content), func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			writeFile(t, dir, "pom.xml", "<project/>")
			writeFile(t, dir, ".java-version", tt.content)
			res := (&java.Module{}).Detect(dir)
			if res.SuggestedConfig.Version != tt.wantVersion {
				t.Errorf("Version = %q, want %q", res.SuggestedConfig.Version, tt.wantVersion)
			}
			assertEvidenceContains(t, res.Evidence, tt.wantEvid)
		})
	}
}

// TestDevenvNixFragment_DottedLeaves guards the Java+Scala merge: a
// `languages.java = { ... }` block sets leaves another JVM fragment in the
// same devenv.nix attribute set may also touch, so every option is a dotted
// leaf assignment.
func TestDevenvNixFragment_DottedLeaves(t *testing.T) {
	t.Parallel()

	frag, err := (&java.Module{}).DevenvNixFragment(ecosystem.ModuleConfig{
		Version: "17",
		Extras:  map[string]string{"build_tool": "both"},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertNotContains(t, frag, "languages.java = {")
	for _, line := range []string{
		"languages.java.enable = true;",
		"languages.java.jdk.package = pkgs.jdk17;",
		"languages.java.maven.enable = true;",
		"languages.java.gradle.enable = true;",
	} {
		assertContains(t, frag, line)
	}
}

// TestPreCommitHooks_PMDChecksStagedFiles guards against PMD scanning the
// whole tree (`-d .`), which failed every Java commit on violations in
// untouched files, target/ output and generated sources.
func TestPreCommitHooks_PMDChecksStagedFiles(t *testing.T) {
	t.Parallel()

	for _, h := range (&java.Module{}).PreCommitHooks(ecosystem.ModuleConfig{}) {
		if h.ID != "pmd" {
			continue
		}
		if !h.PassFilenames {
			t.Error("pmd hook must receive the staged files (PassFilenames)")
		}
		if !strings.HasSuffix(h.Entry, " -d") {
			t.Errorf("pmd entry %q must end with -d so the staged files become its sources", h.Entry)
		}
		if !slices.Equal(h.Types, []string{"java"}) {
			t.Errorf("pmd types = %v, want [java]", h.Types)
		}
		return
	}
	t.Fatal("no pmd hook")
}

// TestGradleCommandsUseNixGradle guards against running the committed,
// unverified Gradle wrapper (./gradlew) instead of the pinned Nix gradle that
// languages.java.gradle.enable provides.
func TestGradleCommandsUseNixGradle(t *testing.T) {
	t.Parallel()

	m := &java.Module{}
	for _, bt := range []string{"gradle", "both"} {
		cfg := ecosystem.ModuleConfig{Extras: map[string]string{"build_tool": bt}}
		var cmds []string
		cmds = append(cmds, m.VerificationCommands(cfg).All()...)
		for _, c := range m.CICommands(cfg) {
			cmds = append(cmds, c.Command)
		}
		for _, pm := range m.PackageManagers() {
			cmds = append(cmds, pm.InstallCommand)
		}
		for _, c := range cmds {
			if strings.Contains(c, "gradlew") {
				t.Errorf("build_tool %s: command %q runs the Gradle wrapper", bt, c)
			}
		}
		if !slices.Contains(m.VerificationCommands(cfg).Build, "gradle build") {
			t.Errorf("build_tool %s: build commands %v lack `gradle build`", bt, m.VerificationCommands(cfg).Build)
		}
	}
}

// TestSecurityConfigs_SettingsXMLPluginRepositories checks that build plugins
// (code executed during the build) get checksumPolicy=fail like dependencies.
func TestSecurityConfigs_SettingsXMLPluginRepositories(t *testing.T) {
	t.Parallel()

	files := (&java.Module{}).SecurityConfigs(ecosystem.ModuleConfig{Extras: map[string]string{"build_tool": "maven"}})
	if len(files) == 0 || files[0].Path != ".mvn/settings.xml" {
		t.Fatalf("first generated file = %+v, want .mvn/settings.xml", files)
	}
	content := string(files[0].Content)
	assertNotContains(t, content, "HTTP blocking")

	var parsed struct {
		Profiles []struct {
			Plugin []struct {
				ID       string `xml:"id"`
				Checksum string `xml:"releases>checksumPolicy"`
				Snapshot string `xml:"snapshots>enabled"`
			} `xml:"pluginRepositories>pluginRepository"`
		} `xml:"profiles>profile"`
	}
	if err := xml.Unmarshal([]byte(content[strings.Index(content, "<settings"):]), &parsed); err != nil {
		t.Fatalf("parsing settings.xml: %v", err)
	}
	if len(parsed.Profiles) != 1 || len(parsed.Profiles[0].Plugin) != 1 {
		t.Fatalf("want one profile with one pluginRepository, got %+v", parsed.Profiles)
	}
	p := parsed.Profiles[0].Plugin[0]
	if p.ID != "central" || p.Checksum != "fail" || p.Snapshot != "false" {
		t.Errorf("pluginRepository = %+v, want central with checksumPolicy=fail and snapshots disabled", p)
	}
}

// TestManifestFiles_ReportsDetectedGradleFiles checks that Kotlin-DSL build
// scripts and the version catalog (where modern Gradle builds declare
// versions) are listed, and that a build.gradle the project lacks is not.
func TestManifestFiles_ReportsDetectedGradleFiles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		files []string
		want  []string
	}{
		{
			name:  "kotlin dsl with version catalog",
			files: []string{"build.gradle.kts", "settings.gradle.kts", "gradle/libs.versions.toml"},
			want:  []string{"build.gradle.kts", "gradle/libs.versions.toml", "settings.gradle.kts"},
		},
		{
			name:  "groovy dsl",
			files: []string{"build.gradle"},
			want:  []string{"build.gradle"},
		},
		{
			name:  "maven and gradle",
			files: []string{"pom.xml", "build.gradle.kts"},
			want:  []string{"pom.xml", "build.gradle.kts"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			for _, f := range tt.files {
				p := filepath.Join(dir, filepath.FromSlash(f))
				if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
					t.Fatal(err)
				}
				writeFile(t, filepath.Dir(p), filepath.Base(p), "")
			}
			m := &java.Module{}
			var got []string
			for _, mf := range m.ManifestFiles(m.Detect(dir).SuggestedConfig) {
				got = append(got, mf.Path)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("ManifestFiles = %v, want %v", got, tt.want)
			}
		})
	}
}
