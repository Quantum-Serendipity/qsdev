package cloudcommon

import "strings"

// FailSafeLayer identifies one of the three credential isolation layers.
type FailSafeLayer int

const (
	LayerEnvironmentSeparation FailSafeLayer = iota
	LayerCredentialFileMasking
	LayerAgentDenyRules
)

// String returns the layer's human-readable name.
func (l FailSafeLayer) String() string {
	switch l {
	case LayerEnvironmentSeparation:
		return "environment separation"
	case LayerCredentialFileMasking:
		return "credential file masking"
	case LayerAgentDenyRules:
		return "agent deny rules"
	default:
		return "unknown layer"
	}
}

// FailSafeStatus reports the state of a single isolation layer for a provider.
type FailSafeStatus struct {
	Provider CloudProvider
	Layer    FailSafeLayer
	Active   bool
	Details  string
}

// FailSafeReport aggregates layer statuses for a single provider.
type FailSafeReport struct {
	Provider        CloudProvider
	Statuses        []FailSafeStatus
	AllLayersActive bool
}

// ValidateFailSafe checks all 3 isolation layers for a single provider from
// static inputs: the project's declared environment variables, its Claude
// Code permission deny rules and its sandbox read-deny paths. It runs no cloud
// CLI.
//
// Layer 1 treats an empty or placeholder value (see IsUnsetEnvValue) as unset,
// so a template value left in place does not count as isolation. Layer 2
// counts a credential path as masked when it is in readDenyPaths or a
// Read(...) rule in denyRules covers it (see ReadRulePaths): the generator
// always emits the Read rules and adds the sandbox read-deny list only when
// the Bash sandbox is enabled.
func ValidateFailSafe(
	provider CloudProvider,
	envVars map[string]string,
	denyRules []string,
	readDenyPaths []string,
) FailSafeReport {
	report := FailSafeReport{Provider: provider}
	allActive := true

	// Layer 1: Environment separation — per-project env var is set.
	envVar := EnvVarForProvider(provider)
	layer1 := FailSafeStatus{Provider: provider, Layer: LayerEnvironmentSeparation}
	switch v, ok := envVars[envVar]; {
	case !ok || strings.TrimSpace(v) == "":
		layer1.Details = envVar + " is not set"
		allActive = false
	case IsUnsetEnvValue(v):
		layer1.Details = envVar + " holds a placeholder value, not a real setting"
		allActive = false
	default:
		layer1.Active = true
		layer1.Details = envVar + " is set"
	}
	report.Statuses = append(report.Statuses, layer1)

	// Layer 2: Credential file masking — ReadDeny paths present.
	layer2 := FailSafeStatus{Provider: provider, Layer: LayerCredentialFileMasking}
	requiredPaths := ReadDenyPaths(provider)
	masked := append(append([]string(nil), readDenyPaths...), ReadRulePaths(denyRules)...)
	missingPaths := findMissing(requiredPaths, masked)
	if len(missingPaths) == 0 {
		layer2.Active = true
		layer2.Details = "all credential files masked"
	} else {
		layer2.Details = "missing ReadDeny for: " + strings.Join(missingPaths, ", ")
		allActive = false
	}
	report.Statuses = append(report.Statuses, layer2)

	// Layer 3: Agent deny rules — Bash deny patterns present. Bash rules match
	// command text, so a complete rule set blocks the invocations Claude
	// usually writes, not every way of running the CLI; the report says the
	// rules are present, not that credential access is impossible.
	layer3 := FailSafeStatus{Provider: provider, Layer: LayerAgentDenyRules}
	requiredRules := BashDenyRules(provider)
	missingRules := findMissing(requiredRules, denyRules)
	if len(missingRules) == 0 {
		layer3.Active = true
		layer3.Details = "all credential-command deny rules present"
	} else {
		layer3.Details = "missing Deny for: " + strings.Join(missingRules, ", ")
		allActive = false
	}
	report.Statuses = append(report.Statuses, layer3)

	report.AllLayersActive = allActive
	return report
}

// ValidateAllProviders runs ValidateFailSafe for each detected provider.
func ValidateAllProviders(
	providers []CloudProvider,
	envVars map[string]string,
	denyRules []string,
	readDenyPaths []string,
) []FailSafeReport {
	reports := make([]FailSafeReport, 0, len(providers))
	for _, p := range providers {
		reports = append(reports, ValidateFailSafe(p, envVars, denyRules, readDenyPaths))
	}
	return reports
}

// placeholderMarkers are substrings (compared upper-cased) that mark an
// environment value as a template left unfilled rather than a real setting.
var placeholderMarkers = []string{"PLACEHOLDER", "CHANGEME", "CHANGE_ME", "REPLACE_ME", "YOUR_"}

// IsUnsetEnvValue reports whether an isolation variable's value leaves the
// variable effectively unset: empty or whitespace, a "<description>" template
// as EnvGuidanceFragment renders it, or a value carrying a placeholder marker
// such as "PLACEHOLDER".
func IsUnsetEnvValue(v string) bool {
	v = strings.TrimSpace(v)
	if v == "" {
		return true
	}
	if strings.HasPrefix(v, "<") && strings.HasSuffix(v, ">") {
		return true
	}
	upper := strings.ToUpper(v)
	for _, m := range placeholderMarkers {
		if strings.Contains(upper, m) {
			return true
		}
	}
	return false
}

// ReadRulePaths returns the paths that Read(...) permission deny rules cover.
// A "dir/**" rule also yields "dir/*", the form ReadDenyPaths uses for the
// same directory (the settings generator widens "/*" to "/**").
func ReadRulePaths(denyRules []string) []string {
	var paths []string
	for _, rule := range denyRules {
		inner, ok := strings.CutPrefix(rule, "Read(")
		if !ok {
			continue
		}
		p, ok := strings.CutSuffix(inner, ")")
		if !ok || p == "" {
			continue
		}
		paths = append(paths, p)
		if base, ok := strings.CutSuffix(p, "/**"); ok {
			paths = append(paths, base+"/*")
		}
	}
	return paths
}

func findMissing(required, present []string) []string {
	set := make(map[string]bool, len(present))
	for _, s := range present {
		set[s] = true
	}
	var missing []string
	for _, r := range required {
		if !set[r] {
			missing = append(missing, r)
		}
	}
	return missing
}
