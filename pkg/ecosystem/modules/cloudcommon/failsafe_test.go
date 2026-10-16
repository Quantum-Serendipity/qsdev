package cloudcommon

import (
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/denyutil"
)

// providerHomeStore is the home directory holding each provider's credential
// files, the credential canon store whose shell-read tripwire the settings
// generator emits.
var providerHomeStore = map[CloudProvider]string{
	AWS:   ".aws",
	GCP:   ".config/gcloud",
	Azure: ".azure",
}

// homeTripwire returns the shell-read tripwire over provider's home store.
func homeTripwire(provider CloudProvider) []string {
	return denyutil.HomeShellReadRules(providerHomeStore[provider], true)
}

// agentDenyRules returns a complete Layer 3 rule set for provider: its
// credential-command rules and the tripwire over its home store.
func agentDenyRules(provider CloudProvider) []string {
	return append(BashDenyRules(provider), homeTripwire(provider)...)
}

// without returns rules with every occurrence of rule removed.
func without(rules []string, rule string) []string {
	return slices.DeleteFunc(slices.Clone(rules), func(r string) bool { return r == rule })
}

func TestValidateFailSafe_AllActive(t *testing.T) {
	t.Parallel()

	envVars := map[string]string{
		"AWS_PROFILE": "dev-project",
	}
	denyRules := agentDenyRules(AWS)
	readDeny := ReadDenyPaths(AWS)

	report := ValidateFailSafe(AWS, envVars, denyRules, readDeny)

	if !report.AllLayersActive {
		t.Error("expected AllLayersActive to be true")
	}
	if len(report.Statuses) != 3 {
		t.Errorf("expected 3 statuses, got %d", len(report.Statuses))
	}
	for _, s := range report.Statuses {
		if !s.Active {
			t.Errorf("expected layer %d to be active, got details: %s", s.Layer, s.Details)
		}
	}
}

func TestValidateFailSafe_Layer1Missing(t *testing.T) {
	t.Parallel()

	envVars := map[string]string{} // No AWS_PROFILE set.
	denyRules := agentDenyRules(AWS)
	readDeny := ReadDenyPaths(AWS)

	report := ValidateFailSafe(AWS, envVars, denyRules, readDeny)

	if report.AllLayersActive {
		t.Error("expected AllLayersActive to be false when env var missing")
	}

	layer1 := report.Statuses[0]
	if layer1.Layer != LayerEnvironmentSeparation {
		t.Errorf("expected first status to be LayerEnvironmentSeparation, got %d", layer1.Layer)
	}
	if layer1.Active {
		t.Error("expected Layer1 to be inactive")
	}

	// Layers 2 and 3 should still be active.
	if !report.Statuses[1].Active {
		t.Error("expected Layer2 to be active")
	}
	if !report.Statuses[2].Active {
		t.Error("expected Layer3 to be active")
	}
}

func TestValidateFailSafe_Layer2Missing(t *testing.T) {
	t.Parallel()

	envVars := map[string]string{
		"AWS_PROFILE": "dev-project",
	}
	denyRules := agentDenyRules(AWS)
	// Provide only the first ReadDeny path, omitting the rest.
	readDeny := ReadDenyPaths(AWS)[:1]

	report := ValidateFailSafe(AWS, envVars, denyRules, readDeny)

	if report.AllLayersActive {
		t.Error("expected AllLayersActive to be false when ReadDeny paths missing")
	}

	layer2 := report.Statuses[1]
	if layer2.Layer != LayerCredentialFileMasking {
		t.Errorf("expected second status to be LayerCredentialFileMasking, got %d", layer2.Layer)
	}
	if layer2.Active {
		t.Error("expected Layer2 to be inactive")
	}

	// Layers 1 and 3 should still be active.
	if !report.Statuses[0].Active {
		t.Error("expected Layer1 to be active")
	}
	if !report.Statuses[2].Active {
		t.Error("expected Layer3 to be active")
	}
}

func TestValidateFailSafe_Layer3Missing(t *testing.T) {
	t.Parallel()

	envVars := map[string]string{
		"AWS_PROFILE": "dev-project",
	}
	// Provide only the first credential-command rule, omitting the rest.
	denyRules := append(BashDenyRules(AWS)[:1:1], homeTripwire(AWS)...)
	readDeny := ReadDenyPaths(AWS)

	report := ValidateFailSafe(AWS, envVars, denyRules, readDeny)

	if report.AllLayersActive {
		t.Error("expected AllLayersActive to be false when deny rules missing")
	}

	layer3 := report.Statuses[2]
	if layer3.Layer != LayerAgentDenyRules {
		t.Errorf("expected third status to be LayerAgentDenyRules, got %d", layer3.Layer)
	}
	if layer3.Active {
		t.Error("expected Layer3 to be inactive")
	}

	// Layers 1 and 2 should still be active.
	if !report.Statuses[0].Active {
		t.Error("expected Layer1 to be active")
	}
	if !report.Statuses[1].Active {
		t.Error("expected Layer2 to be active")
	}
}

func TestValidateAllProviders_MultiProviderPartialDegradation(t *testing.T) {
	t.Parallel()

	envVars := map[string]string{
		"AWS_PROFILE": "dev-project",
		// GCP env var intentionally missing.
	}

	// Combine all deny rules and read-deny paths for both providers.
	allDenyRules := append(agentDenyRules(AWS), agentDenyRules(GCP)...)
	allReadDeny := append(ReadDenyPaths(AWS), ReadDenyPaths(GCP)...)

	reports := ValidateAllProviders([]CloudProvider{AWS, GCP}, envVars, allDenyRules, allReadDeny)

	if len(reports) != 2 {
		t.Fatalf("expected 2 reports, got %d", len(reports))
	}

	// AWS should be fully active.
	awsReport := reports[0]
	if awsReport.Provider != AWS {
		t.Errorf("expected first report to be AWS, got %s", awsReport.Provider)
	}
	if !awsReport.AllLayersActive {
		t.Error("expected AWS AllLayersActive to be true")
	}

	// GCP should be degraded (missing env var).
	gcpReport := reports[1]
	if gcpReport.Provider != GCP {
		t.Errorf("expected second report to be GCP, got %s", gcpReport.Provider)
	}
	if gcpReport.AllLayersActive {
		t.Error("expected GCP AllLayersActive to be false (missing env var)")
	}

	// GCP Layer 1 should be inactive, Layers 2 and 3 should be active.
	if gcpReport.Statuses[0].Active {
		t.Error("expected GCP Layer1 to be inactive")
	}
	if !gcpReport.Statuses[1].Active {
		t.Error("expected GCP Layer2 to be active")
	}
	if !gcpReport.Statuses[2].Active {
		t.Error("expected GCP Layer3 to be active")
	}
}

// layer3 returns the agent deny rules status of a full-environment report
// over denyRules, with the provider's read-deny paths masked.
func layer3(t *testing.T, provider CloudProvider, denyRules []string) (FailSafeStatus, bool) {
	t.Helper()
	env := map[string]string{EnvVarForProvider(provider): "dev"}
	report := ValidateFailSafe(provider, env, denyRules, ReadDenyPaths(provider))
	s := report.Statuses[LayerAgentDenyRules]
	if s.Layer != LayerAgentDenyRules {
		t.Fatalf("status %d is layer %s, want %s", LayerAgentDenyRules, s.Layer, LayerAgentDenyRules)
	}
	return s, report.AllLayersActive
}

// TestValidateFailSafe_ShellReadTripwireRequired is the regression test for
// Layer 3 no longer noticing a deleted shell-read deny once the modules'
// Bash(cat ~/.aws/credentials*)-style rules moved into the canon tripwire:
// removing any one tripwire rule over the provider's home store deactivates
// the layer and names the uncovered home paths.
func TestValidateFailSafe_ShellReadTripwireRequired(t *testing.T) {
	t.Parallel()
	for provider := range providerHomeStore {
		t.Run(string(provider), func(t *testing.T) {
			t.Parallel()
			full := agentDenyRules(provider)
			if s, all := layer3(t, provider, full); !s.Active || !all {
				t.Fatalf("complete rules: layer 3 active = %v, all = %v (%s)", s.Active, all, s.Details)
			}
			for _, rule := range homeTripwire(provider) {
				s, all := layer3(t, provider, without(full, rule))
				if s.Active || all {
					t.Errorf("without %s: layer 3 active = %v, all = %v", rule, s.Active, all)
					continue
				}
				if !strings.Contains(s.Details, "no shell-read deny covers ~/"+providerHomeStore[provider]+"/") {
					t.Errorf("without %s: details %q do not name an uncovered home path", rule, s.Details)
				}
			}
		})
	}
}

// TestValidateFailSafe_ShellReadCoverageForms pins which tripwires cover the
// home paths: per-file and containing-directory rule sets do, and a sibling
// directory that merely shares the name prefix does not.
func TestValidateFailSafe_ShellReadCoverageForms(t *testing.T) {
	t.Parallel()

	perPath := BashDenyRules(AWS)
	for _, p := range ReadDenyPaths(AWS) {
		rel := strings.TrimPrefix(p, "~/")
		if dir, ok := strings.CutSuffix(rel, "/*"); ok {
			perPath = append(perPath, denyutil.HomeShellReadRules(dir, true)...)
			continue
		}
		perPath = append(perPath, denyutil.HomeShellReadRules(rel, false)...)
	}
	tests := []struct {
		name   string
		deny   []string
		active bool
	}{
		{"home store directory", agentDenyRules(AWS), true},
		{"each path or its directory", perPath, true},
		{"sibling directory", append(BashDenyRules(AWS), denyutil.HomeShellReadRules(".awsx", true)...), false},
		{"directory as a file", append(BashDenyRules(AWS), denyutil.HomeShellReadRules(".aws", false)...), false},
		{"no tripwire", BashDenyRules(AWS), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s, _ := layer3(t, AWS, tt.deny)
			if s.Active != tt.active {
				t.Errorf("layer 3 active = %v, want %v (%s)", s.Active, tt.active, s.Details)
			}
			if !tt.active && !strings.Contains(s.Details, "no shell-read deny covers ~/.aws/credentials") {
				t.Errorf("details %q do not name ~/.aws/credentials", s.Details)
			}
		})
	}
}

// TestValidateFailSafe_CLIOperationRuleStillRequired keeps, as on base, the
// literal requirement for every credential-command rule: the tripwire does
// not stand in for a deleted CLI operation rule.
func TestValidateFailSafe_CLIOperationRuleStillRequired(t *testing.T) {
	t.Parallel()
	for provider := range providerHomeStore {
		t.Run(string(provider), func(t *testing.T) {
			t.Parallel()
			full := agentDenyRules(provider)
			for _, rule := range BashDenyRules(provider) {
				s, all := layer3(t, provider, without(full, rule))
				if s.Active || all {
					t.Errorf("without %s: layer 3 active = %v, all = %v", rule, s.Active, all)
				} else if !strings.Contains(s.Details, "missing Deny for: ") || !strings.Contains(s.Details, rule) {
					t.Errorf("without %s: details %q do not name it", rule, s.Details)
				}
			}
		})
	}
	const getSessionToken = "Bash(aws *sts *get-session-token*)"
	if !slices.Contains(BashDenyRules(AWS), getSessionToken) {
		t.Fatalf("BashDenyRules(AWS) lacks %s", getSessionToken)
	}
	if s, _ := layer3(t, AWS, without(agentDenyRules(AWS), getSessionToken)); s.Active {
		t.Errorf("without %s: layer 3 still active", getSessionToken)
	}
}
