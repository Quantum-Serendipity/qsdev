package cloudcommon

import (
	"slices"
	"strings"
	"testing"
)

func TestIsUnsetEnvValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"empty", "", true},
		{"whitespace", " \t", true},
		{"legacy AWS placeholder", "PLACEHOLDER -- set your AWS profile", true},
		{"lower-case placeholder", "placeholder", true},
		{"guidance template", "<gcloud configuration name>", true},
		{"changeme", "changeme", true},
		{"replace me", "REPLACE_ME", true},
		{"your prefix", "YOUR_PROFILE", true},
		{"real profile", "dev-project", false},
		{"subscription id", "00000000-1111-2222-3333-444444444444", false},
		{"nix expression", "config.env.OTHER", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := IsUnsetEnvValue(tt.value); got != tt.want {
				t.Errorf("IsUnsetEnvValue(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestReadRulePaths(t *testing.T) {
	t.Parallel()

	rules := []string{
		"Bash(aws configure set)",
		"Read(~/.aws/credentials)",
		"Read(~/.aws/sso/cache/**)",
		"Read()",
		"Read(unterminated",
	}
	got := ReadRulePaths(rules)
	want := []string{"~/.aws/credentials", "~/.aws/sso/cache/**", "~/.aws/sso/cache/*"}
	if !slices.Equal(got, want) {
		t.Errorf("ReadRulePaths() = %v, want %v", got, want)
	}
}

// generatedReadRules mirrors how the settings generator turns module read-deny
// paths into Read(...) permission rules ("/*" widened to "/**").
func generatedReadRules(provider CloudProvider) []string {
	var rules []string
	for _, p := range ReadDenyPaths(provider) {
		if base, ok := strings.CutSuffix(p, "/*"); ok {
			p = base + "/**"
		}
		rules = append(rules, "Read("+p+")")
	}
	return rules
}

func TestValidateFailSafe_StaticInputs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		provider  CloudProvider
		env       map[string]string
		deny      []string
		readDeny  []string
		wantLayer [3]bool
	}{
		{
			name:      "placeholder profile is not isolation",
			provider:  AWS,
			env:       map[string]string{"AWS_PROFILE": "PLACEHOLDER -- set your AWS profile"},
			deny:      BashDenyRules(AWS),
			readDeny:  ReadDenyPaths(AWS),
			wantLayer: [3]bool{false, true, true},
		},
		{
			name:      "guidance template is not isolation",
			provider:  GCP,
			env:       map[string]string{"CLOUDSDK_ACTIVE_CONFIG_NAME": "<gcloud configuration name>"},
			deny:      BashDenyRules(GCP),
			readDeny:  ReadDenyPaths(GCP),
			wantLayer: [3]bool{false, true, true},
		},
		{
			name:      "Read rules mask credential files without a sandbox",
			provider:  AWS,
			env:       map[string]string{"AWS_PROFILE": "dev"},
			deny:      append(BashDenyRules(AWS), generatedReadRules(AWS)...),
			wantLayer: [3]bool{true, true, true},
		},
		{
			name:      "Read rules mask GCP recursive paths",
			provider:  GCP,
			env:       map[string]string{"CLOUDSDK_ACTIVE_CONFIG_NAME": "dev"},
			deny:      append(BashDenyRules(GCP), generatedReadRules(GCP)...),
			wantLayer: [3]bool{true, true, true},
		},
		{
			name:      "nothing configured",
			provider:  Azure,
			wantLayer: [3]bool{false, false, false},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			report := ValidateFailSafe(tt.provider, tt.env, tt.deny, tt.readDeny)
			for i, s := range report.Statuses {
				if s.Active != tt.wantLayer[i] {
					t.Errorf("layer %s active = %v, want %v (%s)", s.Layer, s.Active, tt.wantLayer[i], s.Details)
				}
			}
			wantAll := tt.wantLayer == [3]bool{true, true, true}
			if report.AllLayersActive != wantAll {
				t.Errorf("AllLayersActive = %v, want %v", report.AllLayersActive, wantAll)
			}
		})
	}
}

func TestValidateFailSafe_PlaceholderDetails(t *testing.T) {
	t.Parallel()

	report := ValidateFailSafe(AWS, map[string]string{"AWS_PROFILE": "PLACEHOLDER"}, nil, nil)
	if got := report.Statuses[0].Details; !strings.Contains(got, "placeholder") {
		t.Errorf("layer 1 details = %q, want it to name the placeholder", got)
	}
}

func TestProviderForModule(t *testing.T) {
	t.Parallel()

	for _, p := range Providers() {
		got, ok := ProviderForModule(string(p))
		if !ok || got != p {
			t.Errorf("ProviderForModule(%q) = %q, %v; want %q, true", p, got, ok, p)
		}
		if EnvVarForProvider(p) == "" || len(ReadDenyPaths(p)) == 0 || len(BashDenyRules(p)) == 0 {
			t.Errorf("provider %q lacks isolation metadata", p)
		}
		if DisplayName(p) == string(p) {
			t.Errorf("DisplayName(%q) falls back to the raw name", p)
		}
	}
	if _, ok := ProviderForModule("golang"); ok {
		t.Error("ProviderForModule(golang) reported a cloud provider")
	}
}

func TestFailSafeLayerString(t *testing.T) {
	t.Parallel()

	for _, l := range []FailSafeLayer{LayerEnvironmentSeparation, LayerCredentialFileMasking, LayerAgentDenyRules} {
		if s := l.String(); s == "" || s == "unknown layer" {
			t.Errorf("FailSafeLayer(%d).String() = %q", l, s)
		}
	}
}
