package cloudcommon

import (
	"sort"

	"github.com/Quantum-Serendipity/qsdev/pkg/denyutil"
)

// CloudProvider identifies a cloud provider.
type CloudProvider string

const (
	AWS   CloudProvider = "aws"
	GCP   CloudProvider = "gcp"
	Azure CloudProvider = "azure"
)

// Providers returns every supported cloud provider. Each is also the name of
// the ecosystem module that configures it.
func Providers() []CloudProvider {
	return []CloudProvider{AWS, GCP, Azure}
}

// ProviderForModule returns the cloud provider an ecosystem module name
// configures; ok is false for a module that is not a cloud provider.
func ProviderForModule(name string) (CloudProvider, bool) {
	for _, p := range Providers() {
		if string(p) == name {
			return p, true
		}
	}
	return "", false
}

// DisplayName returns the provider's human-readable name.
func DisplayName(provider CloudProvider) string {
	switch provider {
	case AWS:
		return "AWS"
	case GCP:
		return "Google Cloud"
	case Azure:
		return "Azure"
	default:
		return string(provider)
	}
}

// EnvVarForProvider returns the per-project variable that selects a
// provider's default account: the AWS profile, the gcloud configuration or,
// for Terraform's azurerm provider, the Azure subscription. It selects a
// default only; the CLI still reads the logins and tokens every project
// shares under the home directory (the Azure CLI ignores ARM_* entirely). See
// CLIConfigDirEnvVar for the variables that give a CLI a per-project
// configuration directory.
func EnvVarForProvider(provider CloudProvider) string {
	switch provider {
	case AWS:
		return "AWS_PROFILE"
	case GCP:
		return "CLOUDSDK_ACTIVE_CONFIG_NAME"
	case Azure:
		return "ARM_SUBSCRIPTION_ID"
	default:
		return ""
	}
}

// ReadDenyPaths returns the Sandbox.ReadDeny paths for a provider: its CLI's
// credential files under the home directory and, for a CLI that
// cloud.isolate_cli_config can isolate, its per-project configuration
// directory (ProjectCLIConfigDir).
func ReadDenyPaths(provider CloudProvider) []string {
	switch provider {
	case AWS:
		return []string{
			"~/.aws/credentials",
			"~/.aws/config",
			"~/.aws/sso/cache/*",
			// assume-role / SSO credential-process results cached by the CLI:
			// JSON holding AccessKeyId, SecretAccessKey and SessionToken.
			"~/.aws/cli/cache/*",
		}
	case GCP:
		return append([]string{
			"~/.config/gcloud/application_default_credentials.json",
			"~/.config/gcloud/credentials.db",
			"~/.config/gcloud/access_tokens.db",
			"~/.config/gcloud/properties",
			// Per-account refresh tokens and client secrets (adc.json,
			// .boto) written by `gcloud auth login`.
			"~/.config/gcloud/legacy_credentials/**",
			"~/.config/gcloud/configurations/*",
		}, projectCLIConfigReadDeny(GCP)...)
	case Azure:
		return append([]string{
			"~/.azure/accessTokens.json",
			"~/.azure/msal_token_cache.json",
			"~/.azure/azureProfile.json",
			"~/.azure/service_principal_entries.json",
		}, projectCLIConfigReadDeny(Azure)...)
	default:
		return nil
	}
}

// BashDenyRules returns the Permissions.Deny rules for a provider.
//
// Each CLI operation is denied in its plain form, with global options before
// or between its words (`aws --profile prod sts ...`, `gcloud auth --quiet
// print-access-token`; all three CLIs accept global options anywhere) and
// behind an `env` prefix, which Claude Code does not strip before matching
// (see denyutil.InterspersedOptionRules). Bash rules match command text, so
// the list blocks the invocations Claude usually writes; it is not an
// exhaustive boundary around the CLIs.
func BashDenyRules(provider CloudProvider) []string {
	switch provider {
	case AWS:
		return append(denyutil.InterspersedOptionRules("aws", awsDeniedOps...),
			"Bash(cat ~/.aws/credentials*)",
			"Bash(cat ~/.aws/config*)",
		)
	case GCP:
		rules := append(denyutil.InterspersedOptionRules("gcloud", gcpDeniedOps...),
			"Bash(cat ~/.config/gcloud/*)",
		)
		return append(rules, projectCLIConfigBashDeny(GCP)...)
	case Azure:
		rules := append(denyutil.InterspersedOptionRules("az", azureDeniedOps...),
			"Bash(cat ~/.azure/*)",
		)
		return append(rules, projectCLIConfigBashDeny(Azure)...)
	default:
		return nil
	}
}

// awsDeniedOps are AWS CLI operations that change credential configuration,
// print credentials, or mint new long-lived ones.
//
// A trailing "*" with no space before it matches the operation bare and with
// arguments, and also its hyphenated variants: "sts assume-role*" covers
// assume-role-with-web-identity and assume-role-with-saml, which likewise
// return usable credentials (F-CAP-11.1-1).
var awsDeniedOps = []string{
	"configure set",
	"sts get-session-token*",
	"sts assume-role*",
	"sts get-federation-token*",
	// `aws configure export-credentials` (CLI v2) writes credentials to
	// stdout / process env in several formats.
	"configure export-credentials*",
	// `aws configure get aws_secret_access_key` prints the stored secret.
	"configure get*",
	// Each of these returns usable short-lived credentials or tokens.
	"sso get-role-credentials*",
	"ecr get-login-password*",
	"ecr get-authorization-token*",
	"eks get-token*",
	"codeartifact get-authorization-token*",
	// These mint persistent credentials that outlive the session.
	"iam create-access-key*",
	"iam create-login-profile*",
	"iam update-login-profile*",
	"iam create-service-specific-credential*",
	"iam reset-service-specific-credential*",
	// These print stored secrets.
	"secretsmanager get-secret-value*",
	"secretsmanager batch-get-secret-value*",
	"ssm get-parameter*--with-decryption*",
}

// gcpDeniedOps are gcloud operations that change the active configuration,
// print tokens, mint service-account keys, or print stored secrets.
var gcpDeniedOps = []string{
	"auth print-access-token*",
	"auth print-identity-token*",
	"auth application-default print-access-token*",
	// config-helper prints a live access token (credential.access_token).
	"config config-helper*",
	"config set",
	// Writes a new long-lived service-account private key.
	"iam service-accounts keys create*",
	"secrets versions access*",
}

// azureDeniedOps are az operations that print tokens, mint or reset service
// principal and app credentials, or print stored secrets and keys.
var azureDeniedOps = []string{
	"account get-access-token*",
	"ad sp credential",
	"ad app credential",
	// Creates a service principal and prints its client secret.
	"ad sp create-for-rbac*",
	"login*--service-principal*",
	"keyvault secret show*",
	"keyvault secret download*",
	"storage account keys list*",
	// Admin kubeconfig carries a cluster-admin client certificate.
	"aks get-credentials*--admin*",
	"aks get-credentials * -a",
}

// AllReadDenyPaths aggregates ReadDeny paths across providers, deduplicated and sorted.
func AllReadDenyPaths(providers []CloudProvider) []string {
	seen := make(map[string]bool)
	var result []string
	for _, p := range providers {
		for _, path := range ReadDenyPaths(p) {
			if !seen[path] {
				seen[path] = true
				result = append(result, path)
			}
		}
	}
	sort.Strings(result)
	return result
}

// AllBashDenyRules aggregates Bash deny rules across providers, deduplicated and sorted.
func AllBashDenyRules(providers []CloudProvider) []string {
	seen := make(map[string]bool)
	var result []string
	for _, p := range providers {
		for _, rule := range BashDenyRules(p) {
			if !seen[rule] {
				seen[rule] = true
				result = append(result, rule)
			}
		}
	}
	sort.Strings(result)
	return result
}
