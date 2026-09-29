package ecosystem

import (
	"path"
	"path/filepath"
	"slices"
	"strings"
)

// ExtraManifests is the Extras key under which a module's Detect records the
// dependency manifests it found, as a comma-separated list of
// project-relative, slash-separated paths (see RecordManifests).
const ExtraManifests = "manifests"

// RecordManifests returns the project-relative paths under projectRoot that
// match any of patterns (filepath.Glob syntax, slash-separated), sorted and
// comma-joined for Extras[ExtraManifests]. It returns "" when none exist.
func RecordManifests(projectRoot string, patterns ...string) string {
	var found []string
	for _, p := range patterns {
		matches, _ := filepath.Glob(filepath.Join(projectRoot, filepath.FromSlash(p)))
		for _, m := range matches {
			if rel, err := filepath.Rel(projectRoot, m); err == nil {
				found = append(found, filepath.ToSlash(rel))
			}
		}
	}
	slices.Sort(found)
	return strings.Join(slices.Compact(found), ",")
}

// DetectedManifests returns one entry per manifest Detect recorded under
// ExtraManifests that matches a candidate's Path (a path.Match pattern), with
// Path set to the recorded file. Manifest lists therefore name the files a
// project actually has (build.gradle.kts, not build.gradle). When no recorded
// file matches, as for a configuration chosen in the wizard before the build
// files exist, it returns fallback.
func DetectedManifests(config ModuleConfig, candidates, fallback []ManifestFileInfo) []ManifestFileInfo {
	var out []ManifestFileInfo
	for _, file := range strings.Split(config.Extra(ExtraManifests, ""), ",") {
		file = strings.TrimSpace(file)
		if file == "" {
			continue
		}
		for _, c := range candidates {
			if ok, _ := path.Match(c.Path, file); ok {
				c.Path = file
				out = append(out, c)
				break
			}
		}
	}
	if len(out) == 0 {
		return fallback
	}
	return out
}
