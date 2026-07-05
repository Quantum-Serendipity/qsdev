package ecosystem

// LockFilePair maps a manifest file to its expected lock file.
type LockFilePair struct {
	Manifest string
	Lockfile string
}

// ManifestsByEcosystem maps ecosystem names to their dependency *manifest*
// file(s): the human-edited files that declare dependencies. It is the
// counterpart to LockFilesByEcosystem (the generated, pinned files) and shares
// its keys, so every catalog ecosystem has both a manifest and a lockfile
// entry. Drift detection derives its coverage from these two maps, so adding an
// ecosystem here (and to LockFilesByEcosystem) extends coverage without having
// to touch the drift detector.
//
// A manifest name may be a glob (e.g. "*.csproj" for .NET). Some files appear
// in both maps for the same ecosystem (requirements.txt, pom.xml, vcpkg.json):
// they can be authored by hand yet also serve as the pinned artifact. Consumers
// must never treat a manifest as its own lockfile.
var ManifestsByEcosystem = map[string][]string{
	NameGo:         {"go.mod"},
	NameJavaScript: {"package.json"},
	NamePython:     {"pyproject.toml", "requirements.txt"},
	NameRust:       {"Cargo.toml"},
	NameJava:       {"pom.xml", "build.gradle", "build.gradle.kts"},
	NameDotnet:     {"*.csproj"},
	NameRuby:       {"Gemfile"},
	NamePHP:        {"composer.json"},
	NameNix:        {"flake.nix"},
	NameCpp:        {"conanfile.txt", "vcpkg.json"},
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
	NameNix:        {"flake.lock"},
	NameCpp:        {"conan.lock", "vcpkg.json"},
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
