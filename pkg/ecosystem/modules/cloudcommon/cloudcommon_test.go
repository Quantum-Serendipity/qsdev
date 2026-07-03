package cloudcommon

import (
	"sort"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/denyutil"
)

func TestReadDenyPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		provider CloudProvider
		want     int
	}{
		{name: "AWS", provider: AWS, want: 3},
		{name: "GCP", provider: GCP, want: 4},
		{name: "Azure", provider: Azure, want: 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			paths := ReadDenyPaths(tt.provider)
			if len(paths) != tt.want {
				t.Errorf("ReadDenyPaths(%s) returned %d paths, want %d", tt.provider, len(paths), tt.want)
			}
		})
	}
}

func TestBashDenyRules(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		provider CloudProvider
		want     int
	}{
		{name: "AWS", provider: AWS, want: 7},
		{name: "GCP", provider: GCP, want: 5},
		{name: "Azure", provider: Azure, want: 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rules := BashDenyRules(tt.provider)
			if len(rules) != tt.want {
				t.Errorf("BashDenyRules(%s) returned %d rules, want %d", tt.provider, len(rules), tt.want)
			}
		})
	}
}

// TestAWSDenyRules_CoverCredentialExfilCommands is the M6 regression: every AWS
// STS/CLI command that emits usable credentials must be blocked by some deny rule
// (matched with the real denyutil matcher), and benign commands must not be. The
// space-before-glob in the old "assume-role *" rule missed the hyphenated
// assume-role-with-web-identity/-with-saml variants, and get-federation-token /
// configure export-credentials had no rule at all.
func TestAWSDenyRules_CoverCredentialExfilCommands(t *testing.T) {
	t.Parallel()
	rules := BashDenyRules(AWS)

	anyRuleMatches := func(op string) bool {
		for _, r := range rules {
			if denyutil.MatchesDenyRule(r, op) {
				return true
			}
		}
		return false
	}

	denied := []string{
		"Bash(aws sts get-session-token)",
		"Bash(aws sts get-session-token --duration-seconds 3600)",
		"Bash(aws sts assume-role --role-arn arn:aws:iam::1:role/x --role-session-name s)",
		"Bash(aws sts assume-role-with-web-identity --role-arn arn:aws:iam::1:role/x --web-identity-token t)",
		"Bash(aws sts assume-role-with-saml --role-arn arn:aws:iam::1:role/x --principal-arn arn:aws:iam::1:saml/y)",
		"Bash(aws sts get-federation-token --name temp)",
		"Bash(aws configure export-credentials)",
		"Bash(aws configure export-credentials --format env)",
	}
	for _, op := range denied {
		if !anyRuleMatches(op) {
			t.Errorf("no AWS deny rule blocks %q — a credential-exfil command reaches the agent", op)
		}
	}

	allowed := []string{
		"Bash(aws sts decode-authorization-message --encoded-message m)",
		"Bash(aws s3 ls)",
		"Bash(aws configure list)",
	}
	for _, op := range allowed {
		if anyRuleMatches(op) {
			t.Errorf("AWS deny rule over-blocks benign command %q", op)
		}
	}
}

func TestAllBashDenyRules_MultiProvider(t *testing.T) {
	t.Parallel()

	rules := AllBashDenyRules([]CloudProvider{AWS, GCP, Azure})

	// Verify no duplicates.
	seen := make(map[string]bool)
	for _, r := range rules {
		if seen[r] {
			t.Errorf("duplicate rule: %s", r)
		}
		seen[r] = true
	}

	// Verify sorted.
	if !sort.StringsAreSorted(rules) {
		t.Error("AllBashDenyRules result is not sorted")
	}

	// Total should be 7 + 5 + 4 = 16 (no overlaps between providers).
	if len(rules) != 16 {
		t.Errorf("AllBashDenyRules returned %d rules, want 16", len(rules))
	}
}

func TestAllReadDenyPaths_MultiProvider(t *testing.T) {
	t.Parallel()

	paths := AllReadDenyPaths([]CloudProvider{AWS, GCP, Azure})

	// Verify no duplicates.
	seen := make(map[string]bool)
	for _, p := range paths {
		if seen[p] {
			t.Errorf("duplicate path: %s", p)
		}
		seen[p] = true
	}

	// Verify sorted.
	if !sort.StringsAreSorted(paths) {
		t.Error("AllReadDenyPaths result is not sorted")
	}

	// Total should be 3 + 4 + 4 = 11 (no overlaps between providers).
	if len(paths) != 11 {
		t.Errorf("AllReadDenyPaths returned %d paths, want 11", len(paths))
	}
}

func TestEnvVarForProvider(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		provider CloudProvider
		want     string
	}{
		{name: "AWS", provider: AWS, want: "AWS_PROFILE"},
		{name: "GCP", provider: GCP, want: "CLOUDSDK_ACTIVE_CONFIG_NAME"},
		{name: "Azure", provider: Azure, want: "ARM_SUBSCRIPTION_ID"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := EnvVarForProvider(tt.provider)
			if got != tt.want {
				t.Errorf("EnvVarForProvider(%s) = %q, want %q", tt.provider, got, tt.want)
			}
		})
	}
}

func TestEnvVarForProvider_Unknown(t *testing.T) {
	t.Parallel()

	got := EnvVarForProvider(CloudProvider("unknown"))
	if got != "" {
		t.Errorf("EnvVarForProvider(unknown) = %q, want empty string", got)
	}
}
