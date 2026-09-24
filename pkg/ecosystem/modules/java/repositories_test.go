package java_test

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/java"
)

// generatedMirrors returns the mirrors of the settings.xml SecurityConfigs
// generates for cfg.
func generatedMirrors(t *testing.T, cfg ecosystem.ModuleConfig) []java.Mirror {
	t.Helper()
	for _, f := range (&java.Module{}).SecurityConfigs(cfg) {
		if f.Path != ".mvn/settings.xml" {
			continue
		}
		var s java.Settings
		if err := xml.Unmarshal(f.Content, &s); err != nil {
			t.Fatalf("parsing settings.xml: %v", err)
		}
		return s.Mirrors.Mirror
	}
	t.Fatal("no .mvn/settings.xml generated")
	return nil
}

// TestSecurityConfigs_RepositoryAllowlistMirrorOf checks that the settings.xml
// mirror excludes exactly the allowlisted repository ids, for Maven Central
// and for a registry proxy, so Maven resolves them from their declared URL.
func TestSecurityConfigs_RepositoryAllowlistMirrorOf(t *testing.T) {
	t.Parallel()
	const proxy = "https://maven.corp.example/repository/all/"
	tests := []struct {
		name      string
		proxy     string
		allowlist []string
		wantID    string
		wantURL   string
		wantOf    string
	}{
		{name: "no allowlist", wantID: "central-only", wantURL: "https://repo.maven.apache.org/maven2", wantOf: "*"},
		{name: "one id", allowlist: []string{"confluent"}, wantID: "central-only", wantURL: "https://repo.maven.apache.org/maven2", wantOf: "*,!confluent"},
		{
			name:      "duplicates dropped, order kept",
			allowlist: []string{"company-nexus", "confluent", "company-nexus"},
			wantID:    "central-only", wantURL: "https://repo.maven.apache.org/maven2",
			wantOf: "*,!company-nexus,!confluent",
		},
		{
			// ',' and '!' are mirrorOf syntax: such an id would widen or
			// invert the exclusion, so the generator refuses it.
			name:      "ids that are not plain tokens are dropped",
			allowlist: []string{"a,*", "!central", "ok.repo_1-x", "has space"},
			wantID:    "central-only", wantURL: "https://repo.maven.apache.org/maven2",
			wantOf: "*,!ok.repo_1-x",
		},
		{name: "registry proxy", proxy: proxy, allowlist: []string{"confluent"}, wantID: "corporate-proxy", wantURL: proxy, wantOf: "*,!confluent"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mirrors := generatedMirrors(t, ecosystem.ModuleConfig{
				Extras:              map[string]string{"build_tool": "maven"},
				RegistryProxy:       tt.proxy,
				RepositoryAllowlist: tt.allowlist,
			})
			if len(mirrors) != 1 {
				t.Fatalf("mirrors = %+v, want exactly one", mirrors)
			}
			m := mirrors[0]
			if m.ID != tt.wantID || m.URL != tt.wantURL || m.MirrorOf != tt.wantOf {
				t.Errorf("mirror = %+v, want id %q url %q mirrorOf %q", m, tt.wantID, tt.wantURL, tt.wantOf)
			}
		})
	}
}

const pomWithRepositories = `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
  <modelVersion>4.0.0</modelVersion>
  <repositories>
    <repository><id>central</id><url>https://repo.maven.apache.org/maven2</url></repository>
    <repository><id>confluent</id><url>https://packages.confluent.io/maven/</url></repository>
    <repository><id>maven-central-alias</id><url>https://repo1.maven.org/maven2/</url></repository>
  </repositories>
  <pluginRepositories>
    <pluginRepository><id>jitpack</id><url>https://jitpack.io</url></pluginRepository>
  </pluginRepositories>
  <profiles>
    <profile>
      <id>milestones</id>
      <repositories>
        <repository><id>spring-milestones</id><url>https://repo.spring.io/milestone</url></repository>
      </repositories>
    </profile>
  </profiles>
  <modules>
    <module>service</module>
    <module>../outside</module>
    <module>missing</module>
  </modules>
</project>
`

const modulePOM = `<project xmlns="http://maven.apache.org/POM/4.0.0">
  <repositories>
    <repository><id>company-nexus</id><url>https://nexus.corp.example/repository/releases</url></repository>
    <repository><id>confluent</id><url>https://packages.confluent.io/maven/</url></repository>
  </repositories>
</project>
`

// outsidePOM lives beside the project; a <module> entry pointing there must
// not be read.
const outsidePOM = `<project><repositories>
  <repository><id>outside-repo</id><url>https://outside.example</url></repository>
</repositories></project>
`

// writeProjectFile writes content to rel under dir, creating parents.
func writeProjectFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestSetupWarnings_RedirectedRepositories checks the init warning that lists
// the pom-declared repositories the mirror redirects: every non-Central
// repository (plugin, profile and module repositories included) that is not
// allowlisted, and nothing for Gradle-only projects.
func TestSetupWarnings_RedirectedRepositories(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		pom         string
		buildTool   string
		proxy       string
		allowlist   []string
		wantIDs     []string
		wantAbsent  []string
		wantMessage string
	}{
		{
			name:       "every non-central repository is listed",
			pom:        pomWithRepositories,
			wantIDs:    []string{"confluent (https://packages.confluent.io/maven/)", "jitpack (https://jitpack.io)", "spring-milestones", "company-nexus"},
			wantAbsent: []string{"central (", "maven-central-alias", "outside-repo"},
		},
		{
			name:       "allowlisted ids are not listed",
			pom:        pomWithRepositories,
			allowlist:  []string{"confluent", "company-nexus"},
			wantIDs:    []string{"jitpack", "spring-milestones"},
			wantAbsent: []string{"confluent (", "company-nexus"},
		},
		{
			name:        "registry proxy is named as the target",
			pom:         pomWithRepositories,
			proxy:       "https://maven.corp.example/all/",
			wantIDs:     []string{"jitpack"},
			wantMessage: "redirects to the registry proxy https://maven.corp.example/all/",
		},
		{
			name:      "fully allowlisted project has no warning",
			pom:       pomWithRepositories,
			allowlist: []string{"confluent", "jitpack", "spring-milestones", "company-nexus"},
		},
		{name: "central-only project has no warning", pom: `<project><repositories><repository><id>central</id><url>https://repo.maven.apache.org/maven2</url></repository></repositories></project>`},
		{
			// Redeclaring "central" with another URL replaces Central; the
			// mirror sends it back to Central, so it must be reported.
			name:    "central id with a non-central url is listed",
			pom:     `<project><repositories><repository><id>central</id><url>https://nexus.corp.example/maven/</url></repository></repositories></project>`,
			wantIDs: []string{"central (https://nexus.corp.example/maven/)"},
		},
		{name: "central id without a url has no warning", pom: `<project><repositories><repository><id>central</id></repository></repositories></project>`},
		{name: "no pom.xml has no warning"},
		{name: "gradle project is not checked", pom: pomWithRepositories, buildTool: "gradle"},
		{
			name:        "malformed pom is reported",
			pom:         "<project><repositories>",
			wantMessage: "could not read the repositories",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := filepath.Join(t.TempDir(), "project")
			writeProjectFile(t, dir, "service/pom.xml", modulePOM)
			writeProjectFile(t, filepath.Dir(dir), "outside/pom.xml", outsidePOM)
			if tt.pom != "" {
				writeProjectFile(t, dir, "pom.xml", tt.pom)
			}
			bt := tt.buildTool
			if bt == "" {
				bt = "maven"
			}
			got := (&java.Module{}).SetupWarnings(dir, ecosystem.ModuleConfig{
				Extras:              map[string]string{"build_tool": bt},
				RegistryProxy:       tt.proxy,
				RepositoryAllowlist: tt.allowlist,
			})
			if len(tt.wantIDs) == 0 && tt.wantMessage == "" {
				if len(got) != 0 {
					t.Fatalf("SetupWarnings() = %q, want none", got)
				}
				return
			}
			if len(got) != 1 {
				t.Fatalf("SetupWarnings() = %q, want one warning", got)
			}
			for _, s := range append(tt.wantIDs, tt.wantMessage) {
				if !strings.Contains(got[0], s) {
					t.Errorf("warning %q lacks %q", got[0], s)
				}
			}
			for _, s := range tt.wantAbsent {
				if strings.Contains(got[0], s) {
					t.Errorf("warning %q should not mention %q", got[0], s)
				}
			}
			if len(tt.wantIDs) > 0 && !strings.Contains(got[0], "java.repository_allowlist") {
				t.Errorf("warning %q does not say how to allow the repositories", got[0])
			}
		})
	}
}

// TestSetupWarnings_StaleSettingsXML checks that an existing settings.xml,
// which qsdev never overwrites, is reported when its mirror does not match
// the allowlist, and not when it does.
func TestSetupWarnings_StaleSettingsXML(t *testing.T) {
	t.Parallel()
	generated := func(allowlist []string) string {
		for _, f := range (&java.Module{}).SecurityConfigs(ecosystem.ModuleConfig{
			Extras: map[string]string{"build_tool": "maven"}, RepositoryAllowlist: allowlist,
		}) {
			if f.Path == ".mvn/settings.xml" {
				return string(f.Content)
			}
		}
		t.Fatal("no .mvn/settings.xml generated")
		return ""
	}
	tests := []struct {
		name     string
		existing string
		want     string
	}{
		{name: "matching mirror", existing: generated([]string{"confluent"})},
		{name: "mirror from before the allowlist", existing: generated(nil), want: `mirrorOf "*" but java.repository_allowlist needs "*,!confluent"`},
		{name: "user mirror ids are not qsdev's", existing: `<settings><mirrors><mirror><id>mine</id><mirrorOf>*</mirrorOf></mirror></mirrors></settings>`},
		{name: "unparseable file is left to the settings check", existing: "<settings>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			writeProjectFile(t, dir, "pom.xml", `<project><repositories><repository><id>confluent</id><url>https://packages.confluent.io/maven/</url></repository></repositories></project>`)
			writeProjectFile(t, dir, ".mvn/settings.xml", tt.existing)
			got := (&java.Module{}).SetupWarnings(dir, ecosystem.ModuleConfig{
				Extras:              map[string]string{"build_tool": "maven"},
				RepositoryAllowlist: []string{"confluent"},
			})
			if tt.want == "" {
				if len(got) != 0 {
					t.Fatalf("SetupWarnings() = %q, want none", got)
				}
				return
			}
			if len(got) != 1 || !strings.Contains(got[0], tt.want) {
				t.Fatalf("SetupWarnings() = %q, want one warning containing %q", got, tt.want)
			}
		})
	}
}
