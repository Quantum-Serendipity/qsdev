package cloudcommon

import "sort"

// CloudProvider identifies a cloud provider.
type CloudProvider string

const (
	AWS   CloudProvider = "aws"
	GCP   CloudProvider = "gcp"
	Azure CloudProvider = "azure"
)

// EnvVarForProvider returns the per-project isolation env var name for a provider.
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

// ReadDenyPaths returns the Sandbox.ReadDeny paths for a provider.
func ReadDenyPaths(provider CloudProvider) []string {
	switch provider {
	case AWS:
		return []string{
			"~/.aws/credentials",
			"~/.aws/config",
			"~/.aws/sso/cache/*",
		}
	case GCP:
		return []string{
			"~/.config/gcloud/application_default_credentials.json",
			"~/.config/gcloud/credentials.db",
			"~/.config/gcloud/access_tokens.db",
			"~/.config/gcloud/properties",
		}
	case Azure:
		return []string{
			"~/.azure/accessTokens.json",
			"~/.azure/msal_token_cache.json",
			"~/.azure/azureProfile.json",
			"~/.azure/service_principal_entries.json",
		}
	default:
		return nil
	}
}

// BashDenyRules returns the Permissions.Deny rules for a provider.
func BashDenyRules(provider CloudProvider) []string {
	switch provider {
	case AWS:
		return []string{
			"Bash(aws configure set *)",
			// No space before the glob: each of these subcommands prints temporary
			// credentials to stdout even in its bare, arg-less form, so the rule must
			// match the subcommand with or without trailing arguments (F-CAP-11.1-1).
			// A space before the glob (e.g. "assume-role *") would additionally miss
			// the hyphenated variants like assume-role-with-web-identity.
			"Bash(aws sts get-session-token*)",
			// "assume-role*" (no space) also covers assume-role-with-web-identity and
			// assume-role-with-saml, which likewise return usable credentials.
			"Bash(aws sts assume-role*)",
			"Bash(aws sts get-federation-token*)",
			// `aws configure export-credentials` (CLI v2) writes credentials to
			// stdout / process env in several formats.
			"Bash(aws configure export-credentials*)",
			"Bash(cat ~/.aws/credentials*)",
			"Bash(cat ~/.aws/config*)",
		}
	case GCP:
		return []string{
			"Bash(gcloud auth print-access-token*)",
			"Bash(gcloud auth print-identity-token*)",
			"Bash(gcloud auth application-default print-access-token*)",
			"Bash(cat ~/.config/gcloud/*)",
			"Bash(gcloud config set *)",
		}
	case Azure:
		return []string{
			"Bash(az account get-access-token*)",
			"Bash(az ad sp credential *)",
			"Bash(cat ~/.azure/*)",
			"Bash(az login --service-principal*)",
		}
	default:
		return nil
	}
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
