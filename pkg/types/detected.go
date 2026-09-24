package types

import (
	"slices"
	"strings"
)

// ecosystemAliases maps legacy Ecosystems keys that detection records
// alongside a canonical module name to that canonical name, so the alias is
// not offered as a separate language.
var ecosystemAliases = map[string]string{
	"node":   "javascript",
	"docker": "container",
}

// LanguageChoices derives the language selections implied by detection. It is
// the single mapping shared by the non-interactive defaults (FillDefaults) and
// the wizard's pre-populated defaults, so both paths configure identical
// ecosystems. Every choice is seeded from the module's suggested configuration
// (version, package manager, extras), overlaid with the dedicated detection
// fields. Well-known ecosystems come first in a fixed order; all other
// detected ecosystems follow sorted by name so the result is deterministic.
func (d DetectedProject) LanguageChoices() []LanguageChoice {
	var langs []LanguageChoice
	if d.HasGoMod {
		lc := d.suggestion("go")
		lc.Version = firstNonEmpty(d.GoVersion, lc.Version)
		langs = append(langs, lc)
	}
	if d.HasPackageJSON {
		lc := d.suggestion("javascript")
		lc.Version = firstNonEmpty(d.NodeVersion, lc.Version)
		lc.PackageManager = firstNonEmpty(d.PackageManager, lc.PackageManager)
		langs = append(langs, lc)
	}
	if d.HasPyProject {
		lc := d.suggestion("python")
		lc.Version = firstNonEmpty(d.PythonVersion, lc.Version)
		langs = append(langs, lc)
	}
	if d.HasCargoToml {
		langs = append(langs, d.suggestion("rust"))
	}
	if d.HasPomXML || d.HasBuildGradle {
		langs = append(langs, d.javaChoice())
	}
	if d.HasCsproj {
		langs = append(langs, d.suggestion("dotnet"))
	}
	if d.HasDockerfile {
		lc := d.suggestion("container")
		if d.ContainerRuntime != "" {
			lc.Extras = setExtra(lc.Extras, "container_runtime", d.ContainerRuntime)
		}
		if d.OSFamily != "" {
			lc.Extras = setExtra(lc.Extras, "os_family", d.OSFamily)
		}
		langs = append(langs, lc)
	}
	if d.HasTerraform {
		langs = append(langs, d.suggestion("terraform"))
	}

	// All other detected ecosystems, in sorted order for deterministic output.
	for _, name := range d.remainingEcosystems(langs) {
		langs = append(langs, d.suggestion(name))
	}

	// Cloud co-detection: when a cloud platform co-occurs with Kubernetes
	// (Helm charts), enable the cluster auth plugin extras.
	if d.Ecosystems["helm"] {
		for i := range langs {
			switch langs[i].Name {
			case "gcp", "azure":
				langs[i].Extras = appendUnique(langs[i].Extras, "k8s=true")
			}
		}
	}
	return langs
}

// remainingEcosystems returns the sorted names of detected ecosystems that are
// neither already present in langs, nor aliases of a canonical module, nor
// detected only with probable confidence.
func (d DetectedProject) remainingEcosystems(langs []LanguageChoice) []string {
	present := make(map[string]bool, len(langs))
	for _, l := range langs {
		present[l.Name] = true
	}
	var names []string
	for name, detected := range d.Ecosystems {
		// Ecosystems seen only through generic, probable markers (a bare
		// Makefile, *.ps1 scripts, a roles/ directory) are not auto-enabled:
		// that would install toolchains, hooks and build/test tasks for an
		// unrelated project. They remain available as explicit choices.
		if !detected || present[name] || d.ProbableEcosystems[name] {
			continue
		}
		if _, alias := ecosystemAliases[name]; alias {
			continue
		}
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// javaChoice returns the java selection, deriving build_tool from the
// dedicated marker fields when the module's suggestion does not carry one.
// build_tool gates Maven/Gradle enablement, settings.xml hardening, and the
// mvn/gradle deny rules, so it must never be lost.
func (d DetectedProject) javaChoice() LanguageChoice {
	lc := d.suggestion("java")
	if extraValue(lc.Extras, "build_tool") != "" {
		return lc
	}
	bt := ""
	switch {
	case d.HasPomXML && d.HasBuildGradle:
		bt = "both"
	case d.HasPomXML:
		bt = "maven"
	case d.HasBuildGradle:
		bt = "gradle"
	}
	if bt == "" {
		return lc
	}
	lc.Extras = setExtra(lc.Extras, "build_tool", bt)
	if lc.PackageManager == "" {
		// The java module (and its wizard field) read the build tool from
		// PackageManager first; keep the two in step as detection does.
		lc.PackageManager = bt
	}
	return lc
}

// suggestion returns a copy of the detector-suggested choice for name, or a
// bare choice when the module recorded no suggestion.
func (d DetectedProject) suggestion(name string) LanguageChoice {
	lc := d.Suggested[name]
	lc.Name = name
	lc.Extras = slices.Clone(lc.Extras)
	return lc
}

// extraValue returns the value of a "key=value" entry in extras, or "".
func extraValue(extras []string, key string) string {
	for _, e := range extras {
		if k, v, ok := strings.Cut(e, "="); ok && k == key {
			return v
		}
	}
	return ""
}

// setExtra sets key=value in extras, replacing any existing entry for key.
func setExtra(extras []string, key, value string) []string {
	entry := key + "=" + value
	for i, e := range extras {
		if k, _, ok := strings.Cut(e, "="); ok && k == key {
			extras[i] = entry
			return extras
		}
	}
	return append(extras, entry)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// RedactURLCredentials removes any userinfo (user name, password, or access
// token) from a scheme-qualified URL such as
// https://x-access-token:TOKEN@github.com/org/repo.git, so remote URLs can be
// persisted without leaking credentials. scp-style remotes (git@host:path)
// and plain paths carry no secret and are returned unchanged.
func RedactURLCredentials(raw string) string {
	scheme, rest, ok := strings.Cut(raw, "://")
	if !ok {
		return raw
	}
	authority, path, hasPath := strings.Cut(rest, "/")
	at := strings.LastIndex(authority, "@")
	if at < 0 {
		return raw
	}
	out := scheme + "://" + authority[at+1:]
	if hasPath {
		out += "/" + path
	}
	return out
}
