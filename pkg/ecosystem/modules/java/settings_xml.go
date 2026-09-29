// Package java provides XML struct types and rendering helpers for generating
// a security-hardened Maven settings.xml.
package java

import (
	"bytes"
	"encoding/xml"
	"log/slog"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/validation"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// Settings is the root element of a Maven settings.xml file.
type Settings struct {
	XMLName        xml.Name       `xml:"settings"`
	Xmlns          string         `xml:"xmlns,attr"`
	Profiles       Profiles       `xml:"profiles"`
	ActiveProfiles ActiveProfiles `xml:"activeProfiles"`
	Mirrors        Mirrors        `xml:"mirrors"`
}

// Profiles wraps a list of Maven profiles.
type Profiles struct {
	Profile []Profile `xml:"profile"`
}

// Profile is a single Maven build profile.
type Profile struct {
	ID                 string             `xml:"id"`
	Repositories       Repositories       `xml:"repositories"`
	PluginRepositories PluginRepositories `xml:"pluginRepositories"`
}

// Repositories wraps a list of Maven repositories.
type Repositories struct {
	Repository []Repository `xml:"repository"`
}

// PluginRepositories wraps the repositories Maven resolves build plugins
// (code executed during the build) from.
type PluginRepositories struct {
	PluginRepository []Repository `xml:"pluginRepository"`
}

// Repository defines a single Maven artifact repository.
type Repository struct {
	ID        string `xml:"id"`
	URL       string `xml:"url"`
	Releases  Policy `xml:"releases"`
	Snapshots Policy `xml:"snapshots"`
}

// Policy defines the Maven update/checksum policy for a repository channel.
type Policy struct {
	Enabled        string `xml:"enabled"`
	ChecksumPolicy string `xml:"checksumPolicy,omitempty"`
}

// ActiveProfiles wraps the list of active profile IDs.
type ActiveProfiles struct {
	ActiveProfile []string `xml:"activeProfile"`
}

// Mirrors wraps a list of Maven mirror definitions.
type Mirrors struct {
	Mirror []Mirror `xml:"mirror"`
}

// Mirror defines a Maven repository mirror.
type Mirror struct {
	ID       string `xml:"id"`
	Name     string `xml:"name"`
	URL      string `xml:"url"`
	MirrorOf string `xml:"mirrorOf"`
}

// hardenedCentral returns the Maven Central repository definition with
// checksum failures fatal and snapshots disabled.
func hardenedCentral() Repository {
	return Repository{
		ID:  "central",
		URL: mavenCentralURL,
		Releases: Policy{
			Enabled:        "true",
			ChecksumPolicy: "fail",
		},
		Snapshots: Policy{
			Enabled: "false",
		},
	}
}

// buildSecuritySettings returns a Settings struct configured for supply-chain
// security: strict checksum enforcement for dependencies and build plugins,
// snapshot blocking, and mirror as the only mirror. Plugins resolve from
// pluginRepositories, not repositories, so overriding only the dependency
// repository would leave plugin downloads at Maven's default checksumPolicy
// of warn.
func buildSecuritySettings(mirror Mirror) Settings {
	return Settings{
		Xmlns: "http://maven.apache.org/SETTINGS/1.2.0",
		Profiles: Profiles{
			Profile: []Profile{
				{
					ID: "security-hardened",
					Repositories: Repositories{
						Repository: []Repository{hardenedCentral()},
					},
					PluginRepositories: PluginRepositories{
						PluginRepository: []Repository{hardenedCentral()},
					},
				},
			},
		},
		ActiveProfiles: ActiveProfiles{
			ActiveProfile: []string{"security-hardened"},
		},
		Mirrors: Mirrors{
			Mirror: []Mirror{mirror},
		},
	}
}

// Mirror ids qsdev generates. They are stable so that an existing
// .mvn/settings.xml can be recognised as qsdev's (see staleMirrorWarnings).
const (
	centralMirrorID = "central-only"
	proxyMirrorID   = "corporate-proxy"
)

// mavenCentralURL is the Maven Central repository every non-allowlisted
// repository is redirected to when no registry proxy is configured.
const mavenCentralURL = "https://repo.maven.apache.org/maven2"

// buildMirror returns the mirror that intercepts every repository except the
// allowlisted ones (see mirrorOfExcept): the registry proxy when one is
// configured, otherwise Maven Central.
func buildMirror(registryProxy string, allowlist []string) Mirror {
	if registryProxy != "" {
		return Mirror{
			ID:       proxyMirrorID,
			Name:     "Corporate registry proxy",
			URL:      registryProxy,
			MirrorOf: mirrorOfExcept(allowlist),
		}
	}
	return Mirror{
		ID:       centralMirrorID,
		Name:     "Maven Central for every repository not in java.repository_allowlist",
		URL:      mavenCentralURL,
		MirrorOf: mirrorOfExcept(allowlist),
	}
}

// mirrorOfExcept returns the mirrorOf value matching every repository except
// the allowlisted ids ("*,!id1,!id2"), which Maven then resolves from the URL
// the pom declares. Duplicates are dropped, as is any id that is not a plain
// token: in mirrorOf ',' separates entries and '!' and '*' are operators, so
// such an id would change which repositories the mirror matches. Config and
// answers validation reject those ids; this is the generator's own guard.
func mirrorOfExcept(allowlist []string) string {
	parts := []string{"*"}
	seen := make(map[string]bool, len(allowlist))
	for _, id := range allowlist {
		if seen[id] {
			continue
		}
		seen[id] = true
		if !validation.IsValidToken(id) {
			slog.Warn("java: ignoring invalid java.repository_allowlist id", "id", id)
			continue
		}
		parts = append(parts, "!"+id)
	}
	return strings.Join(parts, ",")
}

// xmlHeader returns the XML declaration and comment header prepended to rendered output.
func xmlHeader() string {
	return "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n" +
		"<!-- " + branding.GeneratedBy() + " — supply-chain security hardened.\n" +
		"     Requires: Maven >= 3.2.5 for checksumPolicy.\n" +
		"     checksumPolicy=fail for dependencies and plugins, snapshots disabled,\n" +
		"     every repository not in .qsdev.yaml java.repository_allowlist redirected\n" +
		"     to Maven Central (or the registry proxy) via a mirror.\n" +
		"     Maven reads this file only when passed with -s .mvn/settings.xml. -->\n"
}

// renderSettingsXML marshals a Settings struct to indented XML with
// the standard XML declaration and a descriptive comment header.
func renderSettingsXML(s Settings) ([]byte, error) {
	body, err := xml.MarshalIndent(s, "", "  ")
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	buf.WriteString(xmlHeader())
	buf.Write(body)
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}
