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
