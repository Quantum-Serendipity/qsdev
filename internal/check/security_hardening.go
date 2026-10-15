package check

import (
	"bufio"
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/internal/profile"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// CheckSecurityHardening verifies lock files and security configuration
// settings for the configured ecosystems.
func CheckSecurityHardening(ctx CheckContext) []CheckResult {
	if ctx.QsdevConfig == nil {
		return []CheckResult{
			{
				Category: CategorySecurityHarden,
				Name:     "security_hardening",
				Status:   StatusSkip,
				Severity: SeverityInfo,
				Message:  configUnavailableMessage(ctx, "security hardening checks"),
			},
		}
	}

	if len(ctx.QsdevConfig.Languages) == 0 {
		return []CheckResult{
			{
				Category: CategorySecurityHarden,
				Name:     "security_hardening",
				Status:   StatusSkip,
				Severity: SeverityInfo,
				Message:  "No languages configured; skipping security hardening checks",
			},
		}
	}

	// The release-age window generation applied: the project's effective
	// compliance level's, so a strict project's configs must meet 14 days.
	minReleaseAge := catalog.EffectiveAgeGate(config.EffectiveSecurityLevel(ctx.QsdevConfig))
	// The infrastructure (registry proxy) generation wrote into the configs.
	infra := effectiveInfrastructure(ctx)

	var results []CheckResult

	// Check lock files for each language.
	for _, lang := range ctx.QsdevConfig.Languages {
		results = append(results, checkLanguageLockFiles(ctx.ProjectRoot, lang, minReleaseAge, infra)...)
	}

	// Check that the package-manager security configs qsdev generates for
	// JavaScript, Python and PHP still carry their hardening settings.
	for _, lang := range ctx.QsdevConfig.Languages {
		switch lang.Name {
		case ecosystem.NameJavaScript, ecosystem.NamePython, ecosystem.NamePHP:
			results = append(results, checkSecurityConfigSettings(ctx.ProjectRoot, lang, minReleaseAge, infra)...)
		}
	}

	// Check that the tools on PATH are new enough to honour those settings.
	for _, lang := range ctx.QsdevConfig.Languages {
		results = append(results, checkToolchainRequirements(ctx, lang)...)
	}

	if len(results) == 0 {
		// Nothing was verified, so report a skip rather than a pass: a green
		// result here would claim hardening that was never checked.
		results = append(results, CheckResult{
			Category: CategorySecurityHarden,
			Name:     "security_hardening",
			Status:   StatusSkip,
			Severity: SeverityInfo,
			Message:  "No ecosystem-specific hardening checks applicable",
		})
	}

	return results
}

// effectiveInfrastructure returns the infrastructure generation applied to
// the project: .qsdev.yaml's infrastructure block (.qsdev.local.yaml cannot
// carry it) resolved through its infra_profile exactly as the devenv
// generator resolves it (profile.EffectiveInfrastructure), so a profile's
// registry layout paths are expected where generation wrote them. A
// configuration generation would have rejected falls back to the raw block;
// the config comparison then reports whatever drifted.
func effectiveInfrastructure(ctx CheckContext) types.InfraConfig {
	cfg := ctx.QsdevConfig
	answers := config.ConfigToAnswers(cfg, types.DetectedProject{}, ctx.ProjectRoot)
	infra, _, err := profile.DefaultProfileRegistry().EffectiveInfrastructure(cfg.InfraProfile, cfg.Infrastructure, profile.ProjectInputsFromAnswers(answers))
	if err != nil {
		return cfg.Infrastructure
	}
	return infra
}

// checkLanguageLockFiles returns the lock file results for one configured
// language. An ecosystem whose manifest declares no dependencies is skipped
// (its package manager writes no lock file). Ecosystems in the shared
// LockFilesByEcosystem catalog use checkLockFile; every other ecosystem falls
// back to the lock files its module declares via ManifestFileProvider, so
// ecosystems such as Terraform (.terraform.lock.hcl) are not silently passed.
// minReleaseAge and infra are the compliance window and infrastructure
// generation applied (see detectedModuleConfig).
func checkLanguageLockFiles(projectRoot string, lang types.LanguageConfig, minReleaseAge time.Duration, infra types.InfraConfig) []CheckResult {
	mod, hasMod := ecosystem.DefaultRegistry().ByName(lang.Name)
	cfg := ecosystem.ModuleConfig{Version: lang.Version, PackageManager: lang.PackageManager}
	if hasMod {
		cfg = detectedModuleConfig(projectRoot, mod, lang, minReleaseAge, infra)
	}
	// The ecosystem's own project may sit in a subdirectory (frontend/).
	dir := cfg.Directory()
	ecoRoot := filepath.Join(projectRoot, filepath.FromSlash(dir))
	if hasMod {
		if dd, ok := mod.(ecosystem.DependencyDeclarer); ok {
			// An error leaves the answer unknown: enforce the lock file.
			if declares, err := dd.DeclaresDependencies(ecoRoot); err == nil && !declares {
				return []CheckResult{{
					Category: CategorySecurityHarden,
					Name:     "lockfile_" + lang.Name,
					Status:   StatusSkip,
					Severity: SeverityInfo,
					Message:  fmt.Sprintf("%s manifest declares no dependencies; no lock file expected", lang.Name),
				}}
			}
		}
	}

	if r, ok := checkLockFile(projectRoot, dir, lang.Name); ok {
		return []CheckResult{r}
	}
	if !hasMod {
		return nil
	}
	mfp, ok := mod.(ecosystem.ManifestFileProvider)
	if !ok {
		return nil
	}
	var results []CheckResult
	for _, mf := range mfp.ManifestFiles(cfg) {
		if r, ok := checkDeclaredLockFile(projectRoot, lang.Name, mf); ok {
			results = append(results, r)
		}
	}
	return results
}

// checkDeclaredLockFile checks one module-declared manifest/lock file pair.
// It applies only when the pair has a lock file with a required or
// recommended policy and the manifest (which may be a glob such as "*.tf") is
// present in the project root, where ecosystem detection looks. A missing
// required lock file fails; a missing recommended one warns.
func checkDeclaredLockFile(projectRoot, langName string, mf ecosystem.ManifestFileInfo) (CheckResult, bool) {
	if mf.LockFile == "" || mf.LockFilePolicy == ecosystem.LockFilePolicyNone {
		return CheckResult{}, false
	}
	matches, err := filepath.Glob(filepath.Join(projectRoot, mf.Path))
	if err != nil || len(matches) == 0 {
		return CheckResult{}, false
	}

	name := "lockfile_" + langName
	if _, err := os.Stat(filepath.Join(projectRoot, mf.LockFile)); err == nil {
		return CheckResult{
			Category: CategorySecurityHarden,
			Name:     name,
			Status:   StatusPass,
			Severity: SeverityInfo,
			Message:  fmt.Sprintf("Lock file %s found for %s", mf.LockFile, langName),
			FilePath: mf.LockFile,
		}, true
	}

	status, severity := StatusFail, SeverityMedium
	if mf.LockFilePolicy == ecosystem.LockFilePolicyRecommended {
		status, severity = StatusWarn, SeverityLow
	}
	return CheckResult{
		Category:    CategorySecurityHarden,
		Name:        name,
		Status:      status,
		Severity:    severity,
		Message:     fmt.Sprintf("No lock file found for %s (%s expected alongside %s)", langName, mf.LockFile, mf.Path),
		FilePath:    mf.LockFile,
		Remediation: fmt.Sprintf("Generate and commit %s to pin dependency versions", mf.LockFile),
	}, true
}

// checkLockFile verifies that a pinned lock file exists for the ecosystem. A
// file that is also the ecosystem's dependency manifest (pom.xml,
// requirements.txt, vcpkg.json) is never accepted as proof of pinning on its
// own: it yields a warning rather than a pass. ok is false when the ecosystem
// has no known lock files. dir is the ecosystem's project directory relative
// to projectRoot ("" for the root); reported paths include it.
func checkLockFile(projectRoot, dir, langName string) (CheckResult, bool) {
	lockFiles, ok := ecosystem.LockFilesByEcosystem[langName]
	if !ok {
		return CheckResult{}, false
	}
	manifests := ecosystem.ManifestsByEcosystem[langName]

	var manifestFound string
	var pureLockFiles []string
	for _, lf := range lockFiles {
		isManifest := slices.Contains(manifests, lf)
		if !isManifest {
			pureLockFiles = append(pureLockFiles, lf)
		}
		rel := path.Join(dir, lf)
		if _, err := os.Stat(filepath.Join(projectRoot, filepath.FromSlash(rel))); err != nil {
			continue
		}
		if !isManifest {
			return CheckResult{
				Category: CategorySecurityHarden,
				Name:     "lockfile_" + langName,
				Status:   StatusPass,
				Severity: SeverityInfo,
				Message:  fmt.Sprintf("Lock file %s found for %s", rel, langName),
				FilePath: rel,
			}, true
		}
		if manifestFound == "" {
			manifestFound = rel
		}
	}

	remediation := "Generate a lock file to pin dependency versions"
	if len(pureLockFiles) > 0 {
		remediation += " (" + strings.Join(pureLockFiles, ", ") + ")"
	}

	if manifestFound != "" {
		return CheckResult{
			Category:    CategorySecurityHarden,
			Name:        "lockfile_" + langName,
			Status:      StatusWarn,
			Severity:    SeverityMedium,
			Message:     fmt.Sprintf("Only %s found for %s; it is a dependency manifest and does not by itself pin versions", manifestFound, langName),
			FilePath:    manifestFound,
			Remediation: remediation + ", or make sure " + manifestFound + " pins exact versions",
		}, true
	}

	return CheckResult{
		Category:    CategorySecurityHarden,
		Name:        "lockfile_" + langName,
		Status:      StatusFail,
		Severity:    SeverityMedium,
		Message:     "No lock file found for " + langName,
		Remediation: remediation,
	}, true
}

// checkSecurityConfigSettings verifies that each security config file the
// ecosystem module generates for the configured package manager exists and
// still contains the hardening settings the generator writes. The expected
// settings are derived from the module itself so this check cannot drift
// from what qsdev generates, including the release-age gate the project's
// compliance window (minReleaseAge) requires and the registry proxy infra
// routes it through.
func checkSecurityConfigSettings(projectRoot string, lang types.LanguageConfig, minReleaseAge time.Duration, infra types.InfraConfig) []CheckResult {
	name := "security_config_" + lang.Name

	mod, ok := ecosystem.DefaultRegistry().ByName(lang.Name)
	if !ok {
		return []CheckResult{{
			Category: CategorySecurityHarden,
			Name:     name,
			Status:   StatusWarn,
			Severity: SeverityLow,
			Message:  fmt.Sprintf("No ecosystem module registered for %s; cannot verify its security configuration", lang.Name),
		}}
	}

	var results []CheckResult
	for _, gf := range mod.SecurityConfigs(detectedModuleConfig(projectRoot, mod, lang, minReleaseAge, infra)) {
		results = append(results, checkConfigFileSettings(projectRoot, name, lang.Name, gf))
	}
	return results
}

// detectedModuleConfig returns the ModuleConfig generation uses for lang: the
// configured version, package manager and extras completed from what the
// module detects in projectRoot, the same merge
// types.DetectedProject.WithSuggested applies. The checks therefore inspect the files generation actually wrote,
// such as a subproject's frontend/.npmrc or a Yarn Classic .yarnrc, against
// the release-age gates of the compliance window minReleaseAge, as
// ecosystem.ToGenerationConfig passes it, and with the registry proxy infra
// resolves for it (a Composer or npm proxy repository).
func detectedModuleConfig(projectRoot string, mod ecosystem.EcosystemModule, lang types.LanguageConfig, minReleaseAge time.Duration, infra types.InfraConfig) ecosystem.ModuleConfig {
	cfg := ecosystem.ModuleConfig{Version: lang.Version, PackageManager: lang.PackageManager, MinReleaseAge: minReleaseAge}
	if det := mod.Detect(projectRoot); det.Detected {
		if cfg.Version == "" {
			cfg.Version = det.SuggestedConfig.Version
		}
		if cfg.PackageManager == "" {
			cfg.PackageManager = det.SuggestedConfig.PackageManager
		}
		cfg.Extras = maps.Clone(det.SuggestedConfig.Extras)
	}
	// Committed extras take precedence over detected ones, as in
	// config.AdoptCommitted.
	if committed := ecosystem.ExtrasMap(lang.Extras); committed != nil {
		if cfg.Extras == nil {
			cfg.Extras = make(map[string]string, len(committed))
		}
		maps.Copy(cfg.Extras, committed)
	}
	// After the merge: the proxy key can depend on the package manager.
	cfg.RegistryProxy = ecosystem.ResolveRegistryProxy(lang.Name, cfg, infra)
	return cfg
}

// checkConfigFileSettings compares one generated security config file against
// its on-disk counterpart.
func checkConfigFileSettings(projectRoot, name, langName string, gf types.GeneratedFile) CheckResult {
	remediation := "Run 'qsdev init --update' or 'qsdev repair' to restore the security settings in " + gf.Path

	data, err := os.ReadFile(filepath.Join(projectRoot, filepath.FromSlash(gf.Path)))
	if err != nil {
		return CheckResult{
			Category:    CategorySecurityHarden,
			Name:        name,
			Status:      StatusFail,
			Severity:    SeverityMedium,
			Message:     fmt.Sprintf("%s security config %s could not be read: %v", langName, gf.Path, err),
			FilePath:    gf.Path,
			Remediation: remediation,
		}
	}

	expected, err := fileSettings(gf.Path, gf.Content)
	if err != nil {
		// The generator's own output is unparseable: nothing can be compared.
		return CheckResult{
			Category: CategorySecurityHarden,
			Name:     name,
			Status:   StatusWarn,
			Severity: SeverityLow,
			Message:  fmt.Sprintf("cannot derive the expected %s hardening settings for %s: %v", langName, gf.Path, err),
			FilePath: gf.Path,
		}
	}
	actual, err := fileSettings(gf.Path, data)
	if err != nil {
		return CheckResult{
			Category:    CategorySecurityHarden,
			Name:        name,
			Status:      StatusFail,
			Severity:    SeverityMedium,
			Message:     fmt.Sprintf("%s security config %s is not valid JSON: %v", langName, gf.Path, err),
			FilePath:    gf.Path,
			Remediation: remediation,
		}
	}

	values, shapes := missingSettings(expected, actual)
	if len(values)+len(shapes) > 0 {
		result := CheckResult{
			Category:    CategorySecurityHarden,
			Name:        name,
			Status:      StatusFail,
			Severity:    SeverityMedium,
			Message:     fmt.Sprintf("%s is missing or weakens hardening settings: %s", gf.Path, joinGaps(values, shapes)),
			FilePath:    gf.Path,
			Remediation: settingsRemediation(gf, strings.Join(values, ", ")),
		}
		if isIgnoreFile(gf.Path) {
			// Patterns only work together and in order: name both lists.
			result.Message = fmt.Sprintf("%s patterns differ: want %s; found %s",
				gf.Path, quotePatterns(ignorePatterns(gf.Content)), quotePatterns(ignorePatterns(data)))
			result.Remediation = settingsRemediation(gf, "")
		}
		return result
	}

	return CheckResult{
		Category: CategorySecurityHarden,
		Name:     name,
		Status:   StatusPass,
		Severity: SeverityInfo,
		Message:  fmt.Sprintf("%s contains the required hardening settings", gf.Path),
		FilePath: gf.Path,
	}
}

// settingsRemediation tells the user how to restore settings in the
// on-disk counterpart of gf. 'init --update' rewrites a skip-if-exists file
// only while it still holds what qsdev generated; an edited one is the
// user's, so the settings must be raised by hand.
// settings lists the value settings to restore; when empty (only a
// container's shape or an ignore file's patterns changed), the generated file
// as a whole is restored.
func settingsRemediation(gf types.GeneratedFile, settings string) string {
	if gf.Strategy == types.Skip {
		if settings == "" {
			return fmt.Sprintf("Restore the generated hardening settings in %s; 'qsdev init --update' rewrites it only while it is unchanged since qsdev generated it", gf.Path)
		}
		return fmt.Sprintf("Set %s (or stricter) in %s; 'qsdev init --update' rewrites it only while it is unchanged since qsdev generated it", settings, gf.Path)
	}
	if settings == "" {
		return fmt.Sprintf("Run 'qsdev init --update' or 'qsdev repair' to restore the generated %s", gf.Path)
	}
	return fmt.Sprintf("Run 'qsdev init --update' or 'qsdev repair' to restore %s in %s", settings, gf.Path)
}

// fileSettings extracts the settings of the config file at p: JSON files (a
// Composer config.json) through flattenJSONSettings, ignore files (the
// Composer home's nested .gitignore) through ignoreFileSettings, every other
// format through parseSettings.
func fileSettings(p string, content []byte) (map[string]string, error) {
	switch {
	case strings.EqualFold(path.Ext(p), ".json"):
		return flattenJSONSettings(content)
	case isIgnoreFile(p):
		return ignoreFileSettings(content), nil
	}
	return parseSettings(content), nil
}

// isIgnoreFile reports whether p is a Git ignore file.
func isIgnoreFile(p string) bool { return strings.EqualFold(path.Base(p), ".gitignore") }

// ignorePatterns returns the patterns of an ignore file in order: its
// non-blank, non-comment lines without trailing whitespace. Git drops
// trailing whitespace but keeps leading whitespace, which turns
// `  !config.json` into a literal pattern.
func ignorePatterns(content []byte) []string {
	var patterns []string
	for line := range strings.Lines(string(content)) {
		line = strings.TrimRight(line, " \t\r\n")
		if line != "" && !strings.HasPrefix(line, "#") {
			patterns = append(patterns, line)
		}
	}
	return patterns
}

// quotePatterns renders ignore patterns for a message, or "no patterns".
func quotePatterns(patterns []string) string {
	if len(patterns) == 0 {
		return "no patterns"
	}
	quoted := make([]string, len(patterns))
	for i, p := range patterns {
		quoted[i] = "`" + p + "`"
	}
	return strings.Join(quoted, ", ")
}

// ignoreFileSettings extracts the patterns of an ignore file (ignorePatterns)
// as settings, in order, since a later pattern overrides an earlier one: the
// n-th under "pattern.<n>", and their count (as jsonArrayMarker) under
// rootKey. A generated ignore file is written with the Overwrite strategy,
// so a missing pattern (an emptied file no longer ignores the auth.json
// link), an added one (a trailing `!auth.json` re-includes it) and a
// reordering all fail.
func ignoreFileSettings(content []byte) map[string]string {
	patterns := ignorePatterns(content)
	settings := make(map[string]string, len(patterns)+1)
	for i, p := range patterns {
		settings["pattern."+strconv.Itoa(i+1)] = p
	}
	settings[rootKey] = jsonArrayMarker(len(patterns))
	return settings
}

// rootKey is the key the marker of a whole document is recorded under: the
// top-level JSON container's (flattenJSONSettings), or an ignore file's
// pattern count (ignoreFileSettings).
const rootKey = "(root)"

// jsonObjectMarker and jsonArrayMarker are the values flattenJSONSettings
// records for an object or array: its entry count (comment keys excluded) in
// the container's brackets. A container replaced by a scalar
// (allow-plugins: {} set to true) or one that gained entries
// (allow-plugins: {"*": true}, a third repository, an audit.ignore list) no
// longer matches its generated marker. Generated JSON policy is written with
// the Overwrite strategy, so it has no legitimate added entries.
func jsonObjectMarker(n int) string { return "{" + strconv.Itoa(n) + "}" }

func jsonArrayMarker(n int) string { return "[" + strconv.Itoa(n) + "]" }

// flattenJSONSettings flattens a JSON document into dotted keys: scalar
// leaves map to their literal value (strings unquoted, numbers exactly as
// written), array elements use their index as a segment
// (repositories.1.packagist.org), and every object or array is also recorded
// with its entry count (jsonObjectMarker, jsonArrayMarker) under its own key,
// the top-level one under rootKey. Keys starting with "_" (comments) are
// skipped with everything under them. A malformed document, or one followed
// by trailing data, is an error.
func flattenJSONSettings(content []byte) (map[string]string, error) {
	dec := json.NewDecoder(bytes.NewReader(content))
	dec.UseNumber()
	var doc any
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, errors.New("parsing JSON: unexpected data after the top-level value")
	}
	settings := make(map[string]string)
	flattenJSONValue(settings, "", doc)
	return settings, nil
}

// flattenJSONValue records v under key (and its children below it) in
// settings. The top-level value (key "") records its marker under rootKey.
func flattenJSONValue(settings map[string]string, key string, v any) {
	child := func(seg string) string {
		if key == "" {
			return seg
		}
		return key + "." + seg
	}
	markerKey := cmp.Or(key, rootKey)
	switch v := v.(type) {
	case map[string]any:
		n := 0
		for k, val := range v {
			if !strings.HasPrefix(k, "_") {
				n++
				flattenJSONValue(settings, child(k), val)
			}
		}
		settings[markerKey] = jsonObjectMarker(n)
	case []any:
		settings[markerKey] = jsonArrayMarker(len(v))
		for i, val := range v {
			flattenJSONValue(settings, child(strconv.Itoa(i)), val)
		}
	case string:
		settings[key] = v
	case json.Number:
		settings[key] = v.String()
	case bool:
		settings[key] = strconv.FormatBool(v)
	case nil:
		settings[key] = "null"
	}
}

// parseSettings extracts top-level "key = value" (INI/TOML/.npmrc) and
// "key: value" (YAML) settings from a config file. Comments, section headers,
// and indented (nested) lines are ignored; values are unquoted and have
// trailing inline comments removed.
func parseSettings(content []byte) map[string]string {
	settings := make(map[string]string)
	sc := bufio.NewScanner(bytes.NewReader(content))
	for sc.Scan() {
		line := sc.Text()
		if line == "" || line[0] == ' ' || line[0] == '\t' {
			continue
		}
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "[") {
			continue
		}

		sep := strings.IndexAny(line, "=:")
		if sep <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:sep])
		value := strings.TrimSpace(line[sep+1:])
		if i := strings.Index(value, " #"); i >= 0 {
			value = strings.TrimSpace(value[:i])
		}
		settings[key] = strings.Trim(value, `"'`)
	}
	return settings
}

// missingSettings lists each expected setting that is absent from, or weaker
// in, the actual settings: a value setting as "key=value" (values), and a
// JSON object or array whose shape changed, or an ignore file whose pattern
// count changed, in words (shapes), since its marker (jsonObjectMarker) names
// nothing a user can set.
func missingSettings(expected, actual map[string]string) (values, shapes []string) {
	for _, key := range slices.Sorted(maps.Keys(expected)) {
		want := expected[key]
		got, ok := actual[key]
		if ok && settingSatisfied(want, got) {
			continue
		}
		if _, _, isMarker := parseMarker(want); isMarker {
			shapes = append(shapes, describeShape(key, want, got, ok, expected, actual))
			continue
		}
		values = append(values, key+"="+want)
	}
	return values, shapes
}

// joinGaps renders missingSettings' result for a message: the value settings
// comma-separated, then each shape description, separated by semicolons
// since a description may itself hold a list.
func joinGaps(values, shapes []string) string {
	parts := slices.Clone(shapes)
	if len(values) > 0 {
		parts = slices.Insert(parts, 0, strings.Join(values, ", "))
	}
	return strings.Join(parts, "; ")
}

// parseMarker reads a container marker (jsonObjectMarker, jsonArrayMarker):
// whether it is an object, its entry count, and whether s is a marker at all.
func parseMarker(s string) (object bool, n int, ok bool) {
	if len(s) < 3 {
		return false, 0, false
	}
	switch {
	case s[0] == '{' && s[len(s)-1] == '}':
		object = true
	case s[0] == '[' && s[len(s)-1] == ']':
	default:
		return false, 0, false
	}
	n, err := strconv.Atoi(s[1 : len(s)-1])
	return object, n, err == nil && n >= 0
}

// describeShape renders a container mismatch at key in words: the generated
// container want was replaced (got, present when ok) by a value or a
// container of another kind, or gained or lost entries. An object's
// unexpected entries are named.
func describeShape(key, want, got string, ok bool, expected, actual map[string]string) string {
	label := key
	if key == rootKey {
		label = "the top level"
	}
	object, n, _ := parseMarker(want)
	gotObject, m, gotMarker := parseMarker(got)
	switch {
	case !ok:
		return fmt.Sprintf("%s is missing (want %s)", label, containerWords(object, n))
	case !gotMarker || gotObject != object:
		if gotMarker {
			got = containerWords(gotObject, m)
		}
		return fmt.Sprintf("%s must be %s, not %s", label, containerWords(object, n), got)
	case object && m > n:
		if extra := unexpectedEntries(key, expected, actual); len(extra) > 0 {
			return fmt.Sprintf("%s has unexpected entries: %s", label, strings.Join(extra, ", "))
		}
	}
	return fmt.Sprintf("%s must have %s (found %d)", label, entryWords(n), m)
}

// containerWords names an object or array of n entries.
func containerWords(object bool, n int) string {
	switch {
	case object && n == 0:
		return "an empty object"
	case object:
		return "an object with " + entryWords(n)
	case n == 0:
		return "an empty list"
	default:
		return "a list of " + entryWords(n)
	}
}

// entryWords counts n entries.
func entryWords(n int) string {
	switch n {
	case 0:
		return "no entries"
	case 1:
		return "exactly 1 entry"
	default:
		return fmt.Sprintf("exactly %d entries", n)
	}
}

// unexpectedEntries returns the outermost settings below the object at key
// in actual that generation never wrote, relative to key: `*` for an added
// allow-plugins wildcard, `repositories` for an added top-level key.
func unexpectedEntries(key string, expected, actual map[string]string) []string {
	prefix := key + "."
	if key == rootKey {
		prefix = ""
	}
	var extra []string
	for _, k := range slices.Sorted(maps.Keys(actual)) {
		rel, under := strings.CutPrefix(k, prefix)
		if _, known := expected[k]; known || !under || k == rootKey {
			continue
		}
		if !slices.ContainsFunc(extra, func(e string) bool { return strings.HasPrefix(rel, e+".") }) {
			extra = append(extra, rel)
		}
	}
	return extra
}

// settingSatisfied reports whether got satisfies the generated value want:
// equal (booleans compared case-insensitively), or, for numeric thresholds
// with the same unit suffix (e.g. release-age gates "3", "4320", "7d"), at
// least as strict.
func settingSatisfied(want, got string) bool {
	if strings.EqualFold(want, got) {
		return true
	}
	wantN, wantUnit, okW := splitNumber(want)
	gotN, gotUnit, okG := splitNumber(got)
	return okW && okG && wantUnit == gotUnit && gotN >= wantN
}

// splitNumber splits a value such as "7d" or "4320" into its numeric part and
// unit suffix.
func splitNumber(s string) (float64, string, bool) {
	end := 0
	for end < len(s) && (s[end] >= '0' && s[end] <= '9' || s[end] == '.') {
		end++
	}
	if end == 0 {
		return 0, "", false
	}
	n, err := strconv.ParseFloat(s[:end], 64)
	if err != nil {
		return 0, "", false
	}
	return n, s[end:], true
}
