package ecosystem

import "slices"

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
	NamePython:     {"pyproject.toml", "requirements.txt", "Pipfile"},
	NameRust:       {"Cargo.toml"},
	NameJava:       {"pom.xml", "build.gradle", "build.gradle.kts"},
	NameDotnet:     {"*.csproj"},
	NameRuby:       {"Gemfile"},
	NamePHP:        {"composer.json"},
	NameNix:        {"flake.nix"},
	NameCpp:        {"conanfile.txt", "vcpkg.json"},
}

// LockFilesByEcosystem maps ecosystem names to their expected lock file(s).
// Entries that also appear in ManifestsByEcosystem for the same ecosystem
// (requirements.txt, pom.xml, vcpkg.json) are listed so pinned-version
// consumers such as the vulnerability scanner can read them; a check for
// "is there a real lockfile" must skip them (see ManifestsByEcosystem).
var LockFilesByEcosystem = map[string][]string{
	NameGo: {"go.sum"},
	// npm-shrinkwrap.json is npm's publishable lockfile (same format as
	// package-lock.json) and takes precedence over it when both exist, so it
	// is listed first. bun.lock is Bun's text lockfile (the default since Bun
	// 1.2); bun.lockb is the legacy binary format.
	NameJavaScript: {"npm-shrinkwrap.json", "package-lock.json", "yarn.lock", "pnpm-lock.yaml", "bun.lock", "bun.lockb"},
	NamePython:     {"requirements.txt", "poetry.lock", "uv.lock", "pdm.lock", "Pipfile.lock"},
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
//
// Pairs that share a manifest are ALTERNATIVES, one per package manager
// (package.json is locked by exactly one of npm-shrinkwrap.json,
// package-lock.json, pnpm-lock.yaml, yarn.lock, bun.lock or bun.lockb). A manifest is satisfied when ANY of its
// lockfiles exists; consumers must group pairs by manifest and must not report
// the unused alternatives as missing.
var ManifestLockfilePairs = []LockFilePair{
	{"package.json", "npm-shrinkwrap.json"},
	{"package.json", "package-lock.json"},
	{"package.json", "pnpm-lock.yaml"},
	{"package.json", "yarn.lock"},
	{"package.json", "bun.lock"},
	{"package.json", "bun.lockb"},
	{"pyproject.toml", "uv.lock"},
	{"pyproject.toml", "poetry.lock"},
	{"pyproject.toml", "pdm.lock"},
	{"Pipfile", "Pipfile.lock"},
	{"go.mod", "go.sum"},
	{"Cargo.toml", "Cargo.lock"},
	{"Gemfile", "Gemfile.lock"},
	{"composer.json", "composer.lock"},
	{"flake.nix", "flake.lock"},
}

// WorkspaceLockfiles are the lockfiles a package manager keeps once, at the
// workspace root, for every member manifest below it (npm/pnpm/yarn/bun
// workspaces, Cargo workspaces, uv workspaces). Any other lockfile belongs to
// the manifest in its own directory: a nested Go module, Gemfile or flake
// needs its own, and one in a parent directory does not lock it.
var WorkspaceLockfiles = []string{
	"package-lock.json", "npm-shrinkwrap.json", "pnpm-lock.yaml", "yarn.lock", "bun.lock", "bun.lockb",
	"Cargo.lock", "uv.lock",
}

// ManifestLockfiles groups a manifest with every lockfile that can satisfy it.
// Several package managers share a manifest (package.json is locked by npm,
// pnpm, yarn or bun), so the lockfiles are alternatives: one is enough.
type ManifestLockfiles struct {
	Manifest  string
	Lockfiles []string
	// Workspace is the subset of Lockfiles that may sit in an ancestor
	// directory (see WorkspaceLockfiles).
	Workspace []string
}

// GroupedManifestLockfiles folds ManifestLockfilePairs into one entry per
// manifest, preserving the catalog's order for both the manifests and their
// alternative lockfiles.
func GroupedManifestLockfiles() []ManifestLockfiles {
	var groups []ManifestLockfiles
	index := make(map[string]int)
	for _, pair := range ManifestLockfilePairs {
		i, ok := index[pair.Manifest]
		if !ok {
			i = len(groups)
			index[pair.Manifest] = i
			groups = append(groups, ManifestLockfiles{Manifest: pair.Manifest})
		}
		groups[i].Lockfiles = append(groups[i].Lockfiles, pair.Lockfile)
		if slices.Contains(WorkspaceLockfiles, pair.Lockfile) {
			groups[i].Workspace = append(groups[i].Workspace, pair.Lockfile)
		}
	}
	return groups
}
