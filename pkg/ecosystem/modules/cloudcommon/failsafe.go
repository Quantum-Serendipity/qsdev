package cloudcommon

import "strings"

// FailSafeLayer identifies one of the three credential isolation layers.
type FailSafeLayer int

const (
	LayerEnvironmentSeparation FailSafeLayer = iota
	LayerCredentialFileMasking
	LayerAgentDenyRules
)

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

// ValidateFailSafe checks all 3 isolation layers for a single provider.
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
	if v, ok := envVars[envVar]; ok && v != "" {
		layer1.Active = true
		layer1.Details = envVar + " is set"
	} else {
		layer1.Details = envVar + " is not set"
		allActive = false
	}
	report.Statuses = append(report.Statuses, layer1)

	// Layer 2: Credential file masking — ReadDeny paths present.
	layer2 := FailSafeStatus{Provider: provider, Layer: LayerCredentialFileMasking}
	requiredPaths := ReadDenyPaths(provider)
	missingPaths := findMissing(requiredPaths, readDenyPaths)
	if len(missingPaths) == 0 {
		layer2.Active = true
		layer2.Details = "all credential files masked"
	} else {
		layer2.Details = "missing ReadDeny for: " + strings.Join(missingPaths, ", ")
		allActive = false
	}
	report.Statuses = append(report.Statuses, layer2)

	// Layer 3: Agent deny rules — Bash deny patterns present.
	layer3 := FailSafeStatus{Provider: provider, Layer: LayerAgentDenyRules}
	requiredRules := BashDenyRules(provider)
	missingRules := findMissing(requiredRules, denyRules)
	if len(missingRules) == 0 {
		layer3.Active = true
		layer3.Details = "all credential commands blocked"
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
