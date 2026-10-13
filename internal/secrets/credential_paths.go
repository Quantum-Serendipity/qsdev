package secrets

// CredentialStore is one per-user credential store.
type CredentialStore struct {
	// Rel is the store's path relative to the home directory, slash-separated
	// (".ssh", ".config/gcloud").
	Rel string
	// Dir reports that the store is a directory whose every file is
	// credential material, so a read rule must cover its contents too.
	Dir bool
}

// CredentialStores returns the per-user credential stores. It is the single
// definition of where credentials live under the home directory: the sandbox
// mount deny list (denylist.HomeDenyPaths), the file-boundary read-path
// validator (validation.CheckBoundaryReadPath) and the generated read denies
// all derive from it. Each call returns a new slice, so callers may modify
// the result.
func CredentialStores() []CredentialStore {
	return []CredentialStore{
		{Rel: ".ssh", Dir: true},
		{Rel: ".gnupg", Dir: true},
		{Rel: ".aws", Dir: true},
		{Rel: ".azure", Dir: true},
		{Rel: ".config/gcloud", Dir: true},
		{Rel: ".kube", Dir: true},
		{Rel: ".docker/config.json"},
		{Rel: ".netrc"},
		// Package-registry publish tokens and feed credentials.
		{Rel: ".cargo/credentials.toml"},
		{Rel: ".cargo/credentials"},
		{Rel: ".nuget/NuGet/NuGet.Config"},
		{Rel: ".config/NuGet/NuGet.Config"},
		{Rel: ".npmrc"},
		{Rel: ".yarnrc.yml"},
		{Rel: ".yarnrc"},
		{Rel: ".bunfig.toml"},
		{Rel: ".config/.bunfig.toml"},
		{Rel: ".pypirc"},
		{Rel: ".config/pip/pip.conf"},
		{Rel: ".pip/pip.conf"},
		{Rel: "Library/Application Support/pip/pip.conf"},
		{Rel: "AppData/Roaming/pip/pip.ini"},
		{Rel: ".gem/credentials"},
		{Rel: ".local/share/gem/credentials"},
		{Rel: ".bundle/config"},
		{Rel: ".config/composer/auth.json"},
		{Rel: ".composer/auth.json"},
		{Rel: "AppData/Roaming/Composer/auth.json"},
		{Rel: ".local/share/pypoetry/auth.toml"},
		{Rel: ".m2/settings.xml"},
		{Rel: ".gradle/gradle.properties"},
		// Container registry, Terraform and Helm credential stores.
		{Rel: ".config/containers/auth.json"},
		{Rel: ".terraform.d/credentials.tfrc.json"},
		{Rel: ".config/helm/registry", Dir: true},
		{Rel: ".config/helm/repositories.yaml"},
		// Source-host, database, Vault and Claude Code tokens.
		{Rel: ".config/gh", Dir: true},
		{Rel: ".git-credentials"},
		{Rel: ".pgpass"},
		{Rel: ".vault-token"},
		{Rel: ".claude/.credentials.json"},
	}
}

// CredentialPaths returns the Rel of each CredentialStores entry, in the same
// order. Each call returns a new slice, so callers may modify the result.
func CredentialPaths() []string {
	return projectStores(func(s CredentialStore) string { return s.Rel })
}

// CredentialHomePaths returns each credential store as a "~/"-prefixed path
// ("~/.ssh", "~/.npmrc"), in CredentialStores order: the form the sandbox
// read-deny list and the file-boundary hook take.
func CredentialHomePaths() []string {
	return projectStores(func(s CredentialStore) string { return "~/" + s.Rel })
}

// CredentialReadGlobs returns each credential store as a "~/"-prefixed read
// glob, in CredentialStores order: a directory store gains "/**", since a
// rule naming only the directory does not cover the files inside it.
func CredentialReadGlobs() []string {
	return projectStores(func(s CredentialStore) string {
		if s.Dir {
			return "~/" + s.Rel + "/**"
		}
		return "~/" + s.Rel
	})
}

// projectStores maps f over CredentialStores into a new slice.
func projectStores(f func(CredentialStore) string) []string {
	stores := CredentialStores()
	out := make([]string, len(stores))
	for i, s := range stores {
		out[i] = f(s)
	}
	return out
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
