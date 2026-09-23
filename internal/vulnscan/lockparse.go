package vulnscan

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// Package is a single (ecosystem, name, version) dependency coordinate in the
// shape OSV.dev's query API expects.
type Package struct {
	Name      string
	Version   string
	Ecosystem string
}

// lockParser extracts the pinned (name, version) coordinates from the lock file
// at path, stamping each package with the OSV ecosystem identifier eco.
type lockParser func(path, eco string) ([]Package, error)

// LockFile pairs a lock filename with the OSV ecosystem and parser that handle
// it. Auto-detection walks the table knownLockFiles returns in order; the first
// present file wins. Obtain one via DetectLockFile or LockFileForPath; the zero
// value is not usable.
type LockFile struct {
	name       string
	ecosystem  string
	catalogEco string // pkg/ecosystem name, e.g. ecosystem.NamePython
	parse      func(path string) ([]Package, error)
}

// Name returns the lock file's base name (e.g. "go.sum").
func (lf LockFile) Name() string { return lf.name }

// Ecosystem returns the OSV.dev ecosystem namespace the lock file maps to
// (e.g. "Go", "npm", "PyPI", "crates.io").
func (lf LockFile) Ecosystem() string { return lf.ecosystem }

// Parse extracts the pinned dependency coordinates from the lock file at path,
// each stamped with the lock file's OSV ecosystem.
func (lf LockFile) Parse(path string) ([]Package, error) { return lf.parse(path) }

// osvEcosystems bridges qsdev's internal ecosystem catalog (pkg/ecosystem) to
// the namespaces OSV.dev indexes vulnerabilities under. An ecosystem absent here
// has no OSV coverage, so its lock files are skipped during detection.
var osvEcosystems = map[string]string{
	ecosystem.NameGo:         "Go",
	ecosystem.NameJavaScript: "npm",
	ecosystem.NamePython:     "PyPI",
	ecosystem.NameRust:       "crates.io",
}

// lockParsers binds each lock filename the scanner can read to its parser. A
// lock file is scanned only when it appears in both this registry and the
// canonical ecosystem.LockFilesByEcosystem set, so adding an ecosystem (or a new
// lock file to an existing one) in pkg/ecosystem extends coverage automatically
// once a parser is registered here.
var lockParsers = map[string]lockParser{
	"go.sum":            parseGoSum,
	"package-lock.json": parseNPMLock,
	// npm-shrinkwrap.json shares package-lock.json's format.
	"npm-shrinkwrap.json": parseNPMLock,
	"Cargo.lock":          parseTOMLPackages,
	"poetry.lock":         parseTOMLPackages,
	"uv.lock":             parseTOMLPackages,
	"Pipfile.lock":        parsePipfileLock,
	"requirements.txt":    parseRequirementsTxt,
}

// knownLockFiles returns the memoized lock-file table. The table is invariant
// (derived only from static package-level catalogs), so it is built once on
// first use and shared read-only.
var knownLockFiles = sync.OnceValue(buildKnownLockFiles)

// buildKnownLockFiles builds the table of lock files the scanner can extract
// pinned dependency versions from. The set is derived from the canonical
// ecosystem.LockFilesByEcosystem metadata intersected with the parser registry
// and the OSV ecosystem map, so coverage stays in sync with pkg/ecosystem as
// ecosystems are added there. The order is deterministic: ecosystems are visited
// alphabetically, and within an ecosystem dedicated lock files are preferred
// over loose manifests.
func buildKnownLockFiles() []LockFile {
	ecos := make([]string, 0, len(ecosystem.LockFilesByEcosystem))
	for eco := range ecosystem.LockFilesByEcosystem {
		ecos = append(ecos, eco)
	}
	sort.Strings(ecos)

	var out []LockFile
	for _, eco := range ecos {
		osvEco, ok := osvEcosystems[eco]
		if !ok {
			continue // no OSV namespace for this ecosystem; nothing to scan
		}
		for _, name := range ecosystem.OrderedLockFiles(eco) {
			parser, ok := lockParsers[name]
			if !ok {
				continue // no parser for this lock format yet
			}
			out = append(out, LockFile{
				name:       name,
				ecosystem:  osvEco,
				catalogEco: eco,
				parse: func(path string) ([]Package, error) {
					return parser(path, osvEco)
				},
			})
		}
	}
	return out
}

// DetectLockFile returns the first known lock file present in projectRoot,
// along with its full path, or a false ok when none exists.
func DetectLockFile(projectRoot string) (LockFile, string, bool) {
	for _, lf := range knownLockFiles() {
		p := filepath.Join(projectRoot, lf.name)
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return lf, p, true
		}
	}
	return LockFile{}, "", false
}

// LockFileForEcosystem returns the preferred scannable lock file present in
// projectRoot for one pkg/ecosystem ecosystem (e.g. ecosystem.NamePython),
// along with its full path, or a false ok when that ecosystem has none. It uses
// the same preference order as DetectLockFile, so a dedicated lock file such as
// poetry.lock wins over a loose requirements.txt; callers choosing which file to
// scan per ecosystem must use this rather than the raw catalog order.
func LockFileForEcosystem(projectRoot, eco string) (LockFile, string, bool) {
	for _, lf := range knownLockFiles() {
		if lf.catalogEco != eco {
			continue
		}
		p := filepath.Join(projectRoot, lf.name)
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return lf, p, true
		}
	}
	return LockFile{}, "", false
}

// DetectedLockFile is a known lock file found in a project, with its full path.
type DetectedLockFile struct {
	LockFile
	Path string
}

// DetectLockFiles returns the preferred lock file present in projectRoot for
// every ecosystem, in the same order DetectLockFile searches. A polyglot project
// (e.g. go.sum plus package-lock.json) yields one entry per ecosystem, so a scan
// can cover all of them instead of only the first one found. Within an ecosystem
// only the first present file is returned, matching DetectLockFile's preference
// for dedicated lock files over loose manifests.
func DetectLockFiles(projectRoot string) []DetectedLockFile {
	var out []DetectedLockFile
	seenEco := make(map[string]bool)
	for _, lf := range knownLockFiles() {
		if seenEco[lf.ecosystem] {
			continue
		}
		p := filepath.Join(projectRoot, lf.name)
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			seenEco[lf.ecosystem] = true
			out = append(out, DetectedLockFile{LockFile: lf, Path: p})
		}
	}
	return out
}

// LockFileForPath resolves the parser for an explicitly supplied manifest path
// by matching its base name against the known lock files.
func LockFileForPath(path string) (LockFile, bool) {
	base := filepath.Base(path)
	for _, lf := range knownLockFiles() {
		if lf.name == base {
			return lf, true
		}
	}
	return LockFile{}, false
}

// parseGoSum extracts the module@version pairs that are actually part of the
// build from a go.sum file. A module version that is downloaded and compiled has
// a module-zip hash line ("<mod> <ver> h1:..."); a line whose version carries a
// "/go.mod" suffix only records the go.mod hash module graph pruning read, and on
// its own marks a version that is never built. Only zip-hash versions are
// emitted, so advisories against stale graph-only versions are not reported as
// project vulnerabilities. The leading "v" of the module version is dropped
// because OSV's Go ecosystem indexes versions without it.
func parseGoSum(path, eco string) ([]Package, error) {
	f, err := os.Open(path) //nolint:gosec // path is a project-root lock file, not user input
	if err != nil {
		return nil, err
	}
	defer f.Close()

	seen := make(map[string]bool)
	var pkgs []Package
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 2 || strings.HasSuffix(fields[1], "/go.mod") {
			continue
		}
		mod := fields[0]
		ver := strings.TrimPrefix(fields[1], "v")
		key := mod + "@" + ver
		if seen[key] || mod == "" || ver == "" {
			continue
		}
		seen[key] = true
		pkgs = append(pkgs, Package{Name: mod, Version: ver, Ecosystem: eco})
	}
	return pkgs, sc.Err()
}

// npmLock models the subset of package-lock.json we read: the lockfileVersion
// 2/3 "packages" map and the lockfileVersion 1 "dependencies" tree.
type npmLock struct {
	Packages     map[string]npmPackageEntry `json:"packages"`
	Dependencies map[string]npmV1Dep        `json:"dependencies"`
}

// npmPackageEntry is one lockfileVersion 2/3 "packages" entry. Name is set when
// the installed package differs from its node_modules folder name (an npm alias
// such as "foo": "npm:lodash@4.17.4"); Link marks a symlink to a workspace or
// local folder, which is not a registry package.
type npmPackageEntry struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Link    bool   `json:"link"`
}

// npmV1Dep is one lockfileVersion 1 "dependencies" entry. Transitive packages
// that could not be hoisted are nested under their parent's Dependencies, so the
// tree must be walked recursively to see them. An aliased package records its
// real coordinates in Version as "npm:<name>@<version>".
type npmV1Dep struct {
	Version      string              `json:"version"`
	Dependencies map[string]npmV1Dep `json:"dependencies"`
}

// parseNPMLock extracts name@version pairs from a package-lock.json. It reads the
// lockfileVersion 2/3 "packages" map (keyed by node_modules path) and the
// lockfileVersion 1 "dependencies" tree, including its nested entries. Aliased
// packages are reported under their real registry name, since that is the name
// OSV indexes advisories under.
func parseNPMLock(path, eco string) ([]Package, error) {
	data, err := os.ReadFile(path) //nolint:gosec // project-root lock file
	if err != nil {
		return nil, err
	}
	var lock npmLock
	if err := json.Unmarshal(data, &lock); err != nil {
		return nil, fmt.Errorf("decoding npm lockfile %q: %w", path, err)
	}

	seen := make(map[string]bool)
	var pkgs []Package
	add := func(name, ver string) {
		if name == "" || ver == "" {
			return
		}
		key := name + "@" + ver
		if seen[key] {
			return
		}
		seen[key] = true
		pkgs = append(pkgs, Package{Name: name, Version: ver, Ecosystem: eco})
	}
	for k, v := range lock.Packages {
		if k == "" || v.Link {
			continue // the root package has an empty key; links are local folders
		}
		name := v.Name
		if name == "" {
			name = k
			if idx := strings.LastIndex(k, "node_modules/"); idx >= 0 {
				name = k[idx+len("node_modules/"):]
			}
		}
		add(name, v.Version)
	}
	walkNPMV1Deps(lock.Dependencies, add)
	return pkgs, nil
}

// walkNPMV1Deps visits every entry of a lockfileVersion 1 dependency tree,
// descending into nested dependencies, and resolves npm aliases
// ("npm:<name>@<version>") to their real coordinates.
func walkNPMV1Deps(deps map[string]npmV1Dep, add func(name, ver string)) {
	for name, d := range deps {
		ver := d.Version
		if spec, ok := strings.CutPrefix(ver, "npm:"); ok {
			// The real name may itself be scoped ("@scope/pkg@1.0.0"), so split
			// at the last "@".
			if at := strings.LastIndex(spec, "@"); at > 0 {
				name, ver = spec[:at], spec[at+1:]
			}
		}
		add(name, ver)
		walkNPMV1Deps(d.Dependencies, add)
	}
}

// tomlLock models the [[package]] table array that Cargo.lock, poetry.lock, and
// uv.lock share: a list of dependencies, each carrying a flat name and version.
type tomlLock struct {
	Package []struct {
		Name    string `toml:"name"`
		Version string `toml:"version"`
	} `toml:"package"`
}

// parseTOMLPackages extracts [[package]] name/version entries from a TOML lock
// file (Cargo.lock, poetry.lock, uv.lock). Duplicate (name, version)
// coordinates are collapsed; distinct versions of the same package are
// preserved.
func parseTOMLPackages(path, eco string) ([]Package, error) {
	var lock tomlLock
	if _, err := toml.DecodeFile(path, &lock); err != nil {
		return nil, fmt.Errorf("decoding TOML lock %q: %w", path, err)
	}

	seen := make(map[string]bool)
	var pkgs []Package
	for _, p := range lock.Package {
		if p.Name == "" || p.Version == "" {
			continue
		}
		key := p.Name + "@" + p.Version
		if seen[key] {
			continue
		}
		seen[key] = true
		pkgs = append(pkgs, Package{Name: p.Name, Version: p.Version, Ecosystem: eco})
	}
	return pkgs, nil
}

// pipfileLock models the subset of a Pipfile.lock we read: the "default" and
// "develop" dependency groups, each a map of package name to an entry whose
// "version" is a PEP 440 exact specifier such as "==2.31.0".
type pipfileLock struct {
	Default map[string]struct {
		Version string `json:"version"`
	} `json:"default"`
	Develop map[string]struct {
		Version string `json:"version"`
	} `json:"develop"`
}

// parsePipfileLock extracts name/version pairs from a pipenv Pipfile.lock. The
// recorded versions carry a leading "==" pin which is stripped; entries without
// an exact pinned version (e.g. VCS refs) are skipped.
func parsePipfileLock(path, eco string) ([]Package, error) {
	data, err := os.ReadFile(path) //nolint:gosec // project-root lock file
	if err != nil {
		return nil, err
	}
	var lock pipfileLock
	if err := json.Unmarshal(data, &lock); err != nil {
		return nil, fmt.Errorf("decoding Pipfile.lock %q: %w", path, err)
	}

	seen := make(map[string]bool)
	var pkgs []Package
	add := func(name, ver string) {
		ver = strings.TrimPrefix(strings.TrimSpace(ver), "==")
		if name == "" || ver == "" {
			return
		}
		key := name + "@" + ver
		if seen[key] {
			return
		}
		seen[key] = true
		pkgs = append(pkgs, Package{Name: name, Version: ver, Ecosystem: eco})
	}
	for name, v := range lock.Default {
		add(name, v.Version)
	}
	for name, v := range lock.Develop {
		add(name, v.Version)
	}
	return pkgs, nil
}

// parseRequirementsTxt extracts pinned "name==version" entries from a pip
// requirements.txt, ignoring comments, blank lines, and unpinned or
// VCS/editable specifiers it cannot resolve to an exact version.
func parseRequirementsTxt(path, eco string) ([]Package, error) {
	f, err := os.Open(path) //nolint:gosec // project-root lock file
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var pkgs []Package
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") {
			continue
		}
		if i := strings.Index(line, "#"); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		// "===" is PEP 440 arbitrary equality; check it before "==" so the
		// version is not left with a stray leading "=".
		name, ver, ok := strings.Cut(line, "===")
		if !ok {
			name, ver, ok = strings.Cut(line, "==")
		}
		if !ok {
			continue
		}
		name = strings.TrimSpace(name)
		ver = strings.TrimSpace(ver)
		// Drop any trailing extras/markers, e.g. "pkg[extra]==1.0 ; python<3".
		if i := strings.IndexAny(name, "[ ;"); i >= 0 {
			name = name[:i]
		}
		if i := strings.IndexAny(ver, " ;"); i >= 0 {
			ver = ver[:i]
		}
		if name != "" && ver != "" {
			pkgs = append(pkgs, Package{Name: name, Version: ver, Ecosystem: eco})
		}
	}
	return pkgs, sc.Err()
}
