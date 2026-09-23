package cloudcommon

import (
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/denyutil"
)

// TestDenyRules_CoverCredentialPrintingCommands guards the credential-printing
// subcommands that previously slipped past the AWS/GCP lists.
func TestDenyRules_CoverCredentialPrintingCommands(t *testing.T) {
	t.Parallel()
	tests := []struct {
		provider CloudProvider
		denied   []string
		allowed  []string
	}{
		{
			provider: AWS,
			denied: []string{
				"Bash(aws configure get aws_secret_access_key)",
				"Bash(aws configure get aws_access_key_id --profile prod)",
				"Bash(aws sso get-role-credentials --role-name r --account-id 1 --access-token t)",
				"Bash(aws ecr get-login-password --region us-east-1)",
				"Bash(aws codeartifact get-authorization-token --domain d)",
			},
			allowed: []string{"Bash(aws configure list)", "Bash(aws sso login)"},
		},
		{
			provider: GCP,
			denied:   []string{"Bash(gcloud config config-helper --format=json)"},
			allowed:  []string{"Bash(gcloud config list)"},
		},
	}
	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			t.Parallel()
			rules := BashDenyRules(tt.provider)
			matches := func(op string) bool {
				for _, r := range rules {
					if denyutil.MatchesDenyRule(r, op) {
						return true
					}
				}
				return false
			}
			for _, op := range tt.denied {
				if !matches(op) {
					t.Errorf("no %s deny rule blocks %q", tt.provider, op)
				}
			}
			for _, op := range tt.allowed {
				if matches(op) {
					t.Errorf("%s deny rule over-blocks benign %q", tt.provider, op)
				}
			}
		})
	}
}

func TestReadDenyPaths_CoverCachedCredentialStores(t *testing.T) {
	t.Parallel()
	want := map[CloudProvider][]string{
		AWS: {"~/.aws/cli/cache/*"},
		GCP: {"~/.config/gcloud/legacy_credentials/**", "~/.config/gcloud/configurations/*"},
	}
	for provider, paths := range want {
		got := ReadDenyPaths(provider)
		for _, p := range paths {
			if !slices.Contains(got, p) {
				t.Errorf("%s ReadDeny missing %q", provider, p)
			}
		}
	}
}
