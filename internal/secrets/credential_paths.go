package secrets

// CredentialPaths returns the per-user credential stores, home-relative and
// slash-separated (".ssh", ".config/gcloud"). It is the single definition of
// where credentials live under the home directory: the sandbox mount deny
// list (denylist.HomeDenyPaths) and the file-boundary read-path validator
// (validation.CheckBoundaryReadPath) both expand it. Each call returns a new
// slice, so callers may modify the result.
func CredentialPaths() []string {
	return []string{
		".ssh",
		".gnupg",
		".aws",
		".azure",
		".config/gcloud",
		".kube",
		".docker/config.json",
		".netrc",
		// Package-registry publish tokens and feed credentials.
		".cargo/credentials.toml",
		".cargo/credentials",
		".nuget/NuGet/NuGet.Config",
		".config/NuGet/NuGet.Config",
		// Container registry, Terraform and Helm credential stores.
		".config/containers/auth.json",
		".terraform.d/credentials.tfrc.json",
		".config/helm/registry",
		".config/helm/repositories.yaml",
	}
}

// SecretFilePatterns returns the base-name globs of secret-material files:
// dotenv files, private keys and certificates with their keys, keystores,
// encrypted secrets, and the config and text formats secrets are stored in.
// It is the single definition of what a secrets directory is guarded for, so
// the generated Claude Code deny rules read-deny these files inside a
// directory named secrets at any depth while source code in such a directory
// (internal/secrets/*.go) stays readable. Each call returns a new slice, so
// callers may modify the result.
func SecretFilePatterns() []string {
	return []string{
		".env",
		".env.*",
		"*.env",
		"*.key",
		"*.pem",
		"*.p12",
		"*.pfx",
		"*.jks",
		"*.keystore",
		"*.gpg",
		"*.age",
		"*.json",
		"*.yaml",
		"*.yml",
		"*.toml",
		"*.txt",
	}
}
