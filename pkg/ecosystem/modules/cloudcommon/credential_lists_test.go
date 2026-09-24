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

// TestBashDenyRules_OptionTolerantAndMinting checks the rules with Claude
// Code's own matching semantics: global options before the service, an env
// prefix, and the commands that mint persistent credentials or print stored
// secrets must all be denied (W130, W131).
func TestBashDenyRules_OptionTolerantAndMinting(t *testing.T) {
	t.Parallel()
	tests := []struct {
		provider CloudProvider
		denied   []string
		allowed  []string
	}{
		{
			provider: AWS,
			denied: []string{
				"aws sts get-session-token",
				"aws --profile prod sts get-session-token",
				"aws --no-cli-pager configure export-credentials --format env",
				"env aws sts get-session-token",
				"env AWS_PROFILE=prod aws --region eu-west-1 sts assume-role --role-arn x",
				"aws iam create-access-key --user-name ci",
				"aws --profile prod iam create-access-key",
				"aws iam create-login-profile --user-name u --password p",
				"aws secretsmanager get-secret-value --secret-id db",
				"aws ssm get-parameter --name /db/pass --with-decryption",
				"aws ssm get-parameters-by-path --with-decryption --path /db",
				"aws configure set aws_access_key_id x",
				"aws sts --profile prod get-session-token",
				"aws secretsmanager --region eu-west-1 get-secret-value --secret-id db",
				"aws ecr get-authorization-token",
				"aws eks get-token --cluster-name c",
				"aws iam create-service-specific-credential --user-name u --service-name codecommit.amazonaws.com",
			},
			allowed: []string{
				"aws sts get-caller-identity",
				"aws --profile prod s3 ls",
				"aws iam list-users",
				"aws ssm get-parameter --name /app/region",
				"aws configure list",
				"aws eks describe-cluster --name c",
				"aws ecr describe-repositories",
			},
		},
		{
			provider: GCP,
			denied: []string{
				"gcloud --quiet auth print-access-token",
				"env gcloud auth print-access-token",
				"gcloud iam service-accounts keys create key.json --iam-account sa@p.iam.gserviceaccount.com",
				"gcloud --project p secrets versions access latest --secret=db",
				"gcloud config set project p",
				"gcloud auth --quiet print-access-token",
				"gcloud secrets versions --project p access latest --secret=db",
			},
			allowed: []string{"gcloud auth list", "gcloud secrets list", "gcloud iam service-accounts keys list --iam-account sa"},
		},
		{
			provider: Azure,
			denied: []string{
				"az account get-access-token",
				"az --only-show-errors account get-access-token",
				"env az account get-access-token",
				"az ad sp create-for-rbac --name ci",
				"az ad sp credential reset --id x",
				"az ad app credential reset --id x",
				"az keyvault secret show --vault-name v --name db",
				"az storage account keys list --account-name s",
				"az aks get-credentials --resource-group g --name c --admin",
				"az aks get-credentials -g g -n c -a",
				"az account --only-show-errors get-access-token",
				"az login -u app-id --service-principal -p x --tenant t",
			},
			allowed: []string{
				"az account show",
				"az keyvault secret list --vault-name v",
				"az aks get-credentials --resource-group g --name c",
				"az aks get-credentials -g g -n prod-a",
				"az login",
			},
		},
	}
	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			t.Parallel()
			rules := BashDenyRules(tt.provider)
			matches := func(cmd string) bool {
				return slices.ContainsFunc(rules, func(r string) bool { return denyutil.MatchesBashRule(r, cmd) })
			}
			for _, cmd := range tt.denied {
				if !matches(cmd) {
					t.Errorf("no %s deny rule blocks %q", tt.provider, cmd)
				}
			}
			for _, cmd := range tt.allowed {
				if matches(cmd) {
					t.Errorf("%s deny rule over-blocks benign %q", tt.provider, cmd)
				}
			}
		})
	}
}
