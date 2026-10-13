package java

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// settingsXMLPath is where qsdev writes the hardened Maven settings.
const settingsXMLPath = ".mvn/settings.xml"

// maxPOMFiles bounds how many module POMs repository discovery reads, so a
// module cycle or a very large reactor cannot make init walk indefinitely.
const maxPOMFiles = 256

// pomRepository is a <repository> or <pluginRepository> a POM declares.
type pomRepository struct {
	ID  string `xml:"id"`
	URL string `xml:"url"`
}

// pomRepositories are the repository lists of a POM or one of its profiles.
type pomRepositories struct {
	Repositories       []pomRepository `xml:"repositories>repository"`
	PluginRepositories []pomRepository `xml:"pluginRepositories>pluginRepository"`
}

// pomProject is the part of a POM that repository discovery reads. The
// element names carry no namespace, so they match with or without the POM
// 4.0.0 xmlns.
type pomProject struct {
	pomRepositories
	Profiles []pomRepositories `xml:"profiles>profile"`
	Modules  []string          `xml:"modules>module"`
}

// all returns the project's and its profiles' repositories and plugin
// repositories, in document order.
func (p pomProject) all() []pomRepository {
	repos := slices.Concat(p.Repositories, p.PluginRepositories)
	for _, prof := range p.Profiles {
		repos = slices.Concat(repos, prof.Repositories, prof.PluginRepositories)
	}
	return repos
}

// declaredRepositories returns the repositories (plugin repositories and
// those of profiles included) that projectRoot's pom.xml and the module POMs
// it aggregates declare, deduplicated by id in discovery order. Modules that
// resolve outside projectRoot are not read. A project without pom.xml has
// none; an unreadable or malformed POM is an error.
func declaredRepositories(projectRoot string) ([]pomRepository, error) {
	var repos []pomRepository
	seenRepo := map[string]bool{}
	seenPOM := map[string]bool{}
	queue := []string{"pom.xml"}
	for len(queue) > 0 && len(seenPOM) < maxPOMFiles {
		rel := queue[0]
		queue = queue[1:]
		if seenPOM[rel] {
			continue
		}
		seenPOM[rel] = true

		pom, err := readPOM(projectRoot, rel)
		if errors.Is(err, fs.ErrNotExist) && rel != "pom.xml" {
			continue // a listed module without a POM of its own
		}
		if err != nil {
			return nil, err
		}
		for _, r := range pom.all() {
			id := strings.TrimSpace(r.ID)
			if id == "" || seenRepo[id] {
				continue
			}
			seenRepo[id] = true
			repos = append(repos, pomRepository{ID: id, URL: strings.TrimSpace(r.URL)})
		}
		for _, mod := range pom.Modules {
			if next, ok := modulePOM(rel, mod); ok {
				queue = append(queue, next)
			}
		}
	}
	return repos, nil
}

// modulePOM returns the project-relative path of the POM a <module> entry of
// the POM at pomRel names: the module directory's pom.xml, or the file itself
// when the entry names an .xml file. It reports false for an entry that
// resolves outside the project.
func modulePOM(pomRel, module string) (string, bool) {
	module = strings.TrimSpace(module)
	if module == "" {
		return "", false
	}
	p := filepath.Join(filepath.Dir(pomRel), filepath.FromSlash(module))
	if !strings.HasSuffix(strings.ToLower(p), ".xml") {
		p = filepath.Join(p, "pom.xml")
	}
	if !filepath.IsLocal(p) {
		return "", false
	}
	return p, true
}

// readPOM parses the POM at rel, a path relative to projectRoot.
func readPOM(projectRoot, rel string) (pomProject, error) {
	data, err := os.ReadFile(filepath.Join(projectRoot, rel))
	if err != nil {
		return pomProject{}, fmt.Errorf("reading %s: %w", filepath.ToSlash(rel), err)
	}
	var pom pomProject
	if err := xml.Unmarshal(data, &pom); err != nil {
		return pomProject{}, fmt.Errorf("parsing %s: %w", filepath.ToSlash(rel), err)
	}
	return pom, nil
}

// isMavenCentral reports whether a declared repository is Maven Central,
// which the mirror redirects to Maven Central anyway. The id alone does not
// decide it: a POM that redeclares "central" with a company Nexus URL
// overrides Central, and the mirror sends that repository back to Central.
func isMavenCentral(r pomRepository) bool {
	u := strings.TrimSuffix(strings.ToLower(r.URL), "/")
	if u == "" {
		return r.ID == "central"
	}
	for _, host := range []string{"repo.maven.apache.org/maven2", "repo1.maven.org/maven2"} {
		if u == "https://"+host || u == "http://"+host {
			return true
		}
	}
	return false
}

// SetupWarnings reports, for a Maven project, the repositories its POMs
// declare that the generated .mvn/settings.xml mirror redirects (every one
// that is neither Maven Central nor in java.repository_allowlist): artifacts
// only such a repository hosts then fail to resolve, with an error naming
// the mirror rather than the repository. It also reports an existing
// .mvn/settings.xml whose qsdev mirrors no longer match what qsdev generates,
// since qsdev never overwrites that file, and a Maven wrapper too old to
// load it (see mavenWrapperVersionWarning). For a Gradle project it reports
// missing verification metadata and, with a registry proxy, that the proxy
// replaces the Gradle Plugin Portal (see gradleWarnings).
func (m *Module) SetupWarnings(projectRoot string, config ecosystem.ModuleConfig) []string {
	bt := buildTool(config)
	var warnings []string
	if usesMaven(bt) {
		for _, w := range []string{redirectedRepositoryWarning(projectRoot, config), mavenWrapperVersionWarning(projectRoot)} {
			if w != "" {
				warnings = append(warnings, w)
			}
		}
		warnings = append(warnings, staleMirrorWarnings(projectRoot, config)...)
	}
	if usesGradle(bt) {
		warnings = append(warnings, gradleWarnings(projectRoot, config)...)
	}
	return warnings
}

// gradleWarnings reports a Gradle project without
// gradle/verification-metadata.xml, whose strict CI build then fails (Gradle
// itself would verify nothing), and, when a registry proxy is set, that the
// proxy init script removes the Gradle Plugin Portal along with every other
// Maven repository.
func gradleWarnings(projectRoot string, config ecosystem.ModuleConfig) []string {
	var warnings []string
	if _, err := os.Stat(filepath.Join(projectRoot, filepath.FromSlash(gradleVerificationMetadataPath))); errors.Is(err, fs.ErrNotExist) {
		warnings = append(warnings, fmt.Sprintf("%s is missing, so Gradle's strict dependency verification "+
			"checks nothing and the generated gradle-build CI step fails; run '%s', review the file and commit it",
			gradleVerificationMetadataPath, gradleVerificationBootstrap))
	}
	if config.RegistryProxy != "" {
		warnings = append(warnings, fmt.Sprintf("%s routes every Gradle Maven repository to the registry proxy %s, "+
			"the Gradle Plugin Portal (plugins.gradle.org) included, so the proxy must also serve Gradle plugin "+
			"markers (proxy https://plugins.gradle.org/m2/). The qsdev tasks and CI load it; a gradle run by hand "+
			"needs -I %s",
			gradleProxyScriptPath, config.RegistryProxy, gradleProxyScriptPath))
	}
	return warnings
}

// redirectedRepositoryWarning returns the warning listing the declared
// repositories the mirror redirects, or "" when there are none.
func redirectedRepositoryWarning(projectRoot string, config ecosystem.ModuleConfig) string {
	repos, err := declaredRepositories(projectRoot)
	if errors.Is(err, fs.ErrNotExist) {
		return ""
	}
	if err != nil {
		return fmt.Sprintf("could not read the repositories the project's POMs declare (%v); "+
			"the %s mirror redirects every repository not in java.repository_allowlist", err, settingsXMLPath)
	}
	var redirected []string
	for _, r := range repos {
		if isMavenCentral(r) || slices.Contains(config.RepositoryAllowlist, r.ID) {
			continue
		}
		redirected = append(redirected, fmt.Sprintf("%s (%s)", r.ID, r.URL))
	}
	if len(redirected) == 0 {
		return ""
	}
	target := "Maven Central"
	if config.RegistryProxy != "" {
		target = "the registry proxy " + config.RegistryProxy
	}
	return fmt.Sprintf("the project's POMs declare repositories that the %s mirror redirects to %s "+
		"(the devenv shell loads that file for every mvn run through MAVEN_ARGS): %s. "+
		"Artifacts only they host will fail to resolve; "+
		"to resolve them from their own URL, add their ids to java.repository_allowlist in %s",
		settingsXMLPath, target, strings.Join(redirected, ", "), branding.Get().ConfigFile)
}

// staleMirrorWarnings reports each qsdev mirror in an existing
// .mvn/settings.xml whose mirrorOf differs from the one qsdev now generates
// for the allowlist, and a qsdev file without the HTTP blocker (generated
// before .mvn/settings.xml replaced Maven's own global settings). A file that
// is absent or not parseable is left to generation (which creates it) and
// the settings check.
func staleMirrorWarnings(projectRoot string, config ecosystem.ModuleConfig) []string {
	data, err := os.ReadFile(filepath.Join(projectRoot, filepath.FromSlash(settingsXMLPath)))
	if err != nil {
		return nil
	}
	var existing Settings
	if err := xml.Unmarshal(data, &existing); err != nil {
		return nil
	}
	want := mirrorOfExcept(config.RepositoryAllowlist)
	var warnings []string
	qsdevFile, hasBlocker := false, false
	for _, mir := range existing.Mirrors.Mirror {
		if mir.ID == httpBlockerMirrorID {
			hasBlocker = true
			continue
		}
		if mir.ID != centralMirrorID && mir.ID != proxyMirrorID {
			continue
		}
		qsdevFile = true
		if strings.TrimSpace(mir.MirrorOf) == want {
			continue
		}
		warnings = append(warnings, fmt.Sprintf("%s mirror %q has mirrorOf %q but java.repository_allowlist needs %q; "+
			"%s never overwrites an existing %s, so set <mirrorOf>%s</mirrorOf> in it by hand",
			settingsXMLPath, mir.ID, mir.MirrorOf, want, branding.Get().AppName, settingsXMLPath, want))
	}
	if qsdevFile && !hasBlocker {
		b := httpBlockerMirror()
		warnings = append(warnings, fmt.Sprintf("%s has no %q mirror; it replaces Maven's global settings, "+
			"where that mirror is defined, so plain-http repositories are no longer blocked. %s never "+
			"overwrites an existing %s, so add <mirror><id>%s</id><url>%s</url><mirrorOf>%s</mirrorOf>"+
			"<blocked>true</blocked></mirror> after its %s or %s mirror by hand",
			settingsXMLPath, b.ID, branding.Get().AppName, settingsXMLPath, b.ID, b.URL, b.MirrorOf,
			centralMirrorID, proxyMirrorID))
	}
	return warnings
}
