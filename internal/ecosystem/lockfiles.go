package ecosystem

// LockFilePair maps a manifest file to its expected lock file.
type LockFilePair struct {
	Manifest string
	Lockfile string
}

// LockFilesByEcosystem maps ecosystem names to their expected lock file(s).
var LockFilesByEcosystem = map[string][]string{
	NameGo:         {"go.sum"},
	NameJavaScript: {"package-lock.json", "yarn.lock", "pnpm-lock.yaml", "bun.lockb"},
	NamePython:     {"requirements.txt", "poetry.lock", "uv.lock", "Pipfile.lock"},
	NameRust:       {"Cargo.lock"},
	NameJava:       {"gradle.lockfile", "pom.xml"},
	NameDotnet:     {"packages.lock.json"},
	NameRuby:       {"Gemfile.lock"},
	NamePHP:        {"composer.lock"},
}

// ManifestLockfilePairs maps manifest files to their corresponding lock files
// for drift detection.
var ManifestLockfilePairs = []LockFilePair{
	{"package.json", "package-lock.json"},
	{"package.json", "pnpm-lock.yaml"},
	{"package.json", "yarn.lock"},
	{"package.json", "bun.lockb"},
	{"pyproject.toml", "uv.lock"},
	{"pyproject.toml", "poetry.lock"},
	{"go.mod", "go.sum"},
	{"Cargo.toml", "Cargo.lock"},
}
