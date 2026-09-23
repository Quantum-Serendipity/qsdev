package vsentinel

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/Masterminds/semver/v3"
	"golang.org/x/mod/modfile"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// lockfilePair is a single manifest/lockfile relationship to check for one
// ecosystem. It is derived from the ecosystem catalog (see driftPairs), never
// hardcoded, so every catalog ecosystem is covered automatically. The embedded
// specificChecker is the zero value for generic (unparsed) ecosystems.
type lockfilePair struct {
	manifest string // manifest file to look for under root; may be a glob (e.g. "*.csproj")
	eco      string
	specificChecker
}

// specificChecker binds an ecosystem that has a dedicated manifest/lockfile
// parser to its precise drift checker. Ecosystems absent from specificCheckers
// get the zero value: no diffable primary lockfile, and nil funcs that select
// the generic, presence-based checks (fail closed on a missing lockfile). The
// map lives here — not in the catalog — because it wires ecosystems to parser
// functions defined in this package; it is not a hardcoded coverage list.
// Coverage itself is derived from the catalog.
type specificChecker struct {
	// primaryLock is the one lockfile format this checker can version-diff.
	// Other catalog-valid lockfiles (e.g. pnpm/yarn/bun for JS) still count as
	// "pinned" but are not version-diffed.
	primaryLock string
	// checker version-diffs manifest vs lockfile.
	checker func(manifestPath, lockfilePath string) ([]DriftEntry, error)
	// declaredCount returns how many dependencies the manifest declares, used to
	// distinguish a dependency-free manifest (legitimately unlocked) from an
	// unpinned one. Generic ecosystems cannot parse deps, so a present manifest
	// with no lockfile always fails closed.
	declaredCount func(manifestPath string) (int, error)
}

var specificCheckers = map[string]specificChecker{
	ecosystem.NameGo:         {"go.sum", checkGoDrift, goDeclaredCount},
	ecosystem.NameJavaScript: {"package-lock.json", checkJSDrift, jsDeclaredCount},
	ecosystem.NameRust:       {"Cargo.lock", checkCargoDrift, cargoDeclaredCount},
}

func goDeclaredCount(p string) (int, error)    { d, err := parseGoMod(p); return len(d), err }
func jsDeclaredCount(p string) (int, error)    { d, err := parsePackageJSON(p); return len(d), err }
func cargoDeclaredCount(p string) (int, error) { d, err := parseCargoDeps(p); return len(d), err }

// notInLockfile is the LockedVersion reported for a declared dependency that
// the lockfile does not pin at all — typically one added to the manifest
// without regenerating the lockfile, so it would resolve fresh at install time.
const notInLockfile = "(not in lockfile)"

// driftPairs derives the manifest/lockfile pairs to check from the ecosystem
// catalog (ManifestsByEcosystem + LockFilesByEcosystem) so every catalog
// ecosystem is covered. Parsed ecosystems (go/js/rust) keep their precise
// dep-count checker; all others get a generic presence-based checker that fails
// closed when a manifest is present without any lockfile. Ecosystems are
// visited in a stable order so reports are deterministic.
func driftPairs() []lockfilePair {
	ecos := make([]string, 0, len(ecosystem.ManifestsByEcosystem))
	for eco := range ecosystem.ManifestsByEcosystem {
		ecos = append(ecos, eco)
	}
	sort.Strings(ecos)

	var pairs []lockfilePair
	for _, eco := range ecos {
		for _, manifest := range ecosystem.ManifestsByEcosystem[eco] {
			pairs = append(pairs, lockfilePair{
				manifest:        manifest,
				eco:             eco,
				specificChecker: specificCheckers[eco],
			})
		}
	}
	return pairs
}

func DetectDrift(root string) (*DriftReport, error) {
	report := &DriftReport{}

	for _, pair := range driftPairs() {
		manifestPath, ok := manifestPresent(root, pair.manifest)
		if !ok {
			// No manifest for this pair: nothing to check.
			continue
		}

		// Any catalog-valid lockfile for this ecosystem pins the dependencies.
		// Only when NONE is present is the manifest potentially unpinned — this
		// is what stops a correctly-locked pnpm/yarn/bun project (whose lockfile
		// is not package-lock.json) from being reported as "missing lockfile".
		presentLock := firstPresentLockfile(root, pair)

		if presentLock == "" {
			// No lockfile at all. For a PARSED ecosystem, a manifest that declares
			// zero dependencies legitimately has none (e.g. a stdlib-only go.mod has
			// no go.sum), so report drift only when it actually declares deps.
			// GENERIC ecosystems (declaredCount == nil) cannot parse deps and so
			// cannot distinguish a zero-dep manifest — they deliberately fail closed:
			// a present manifest with no lockfile is treated as unpinned.
			if pair.declaredCount != nil {
				count, err := pair.declaredCount(manifestPath)
				if err != nil {
					return nil, fmt.Errorf("parsing %s: %w", pair.manifest, err)
				}
				if count == 0 {
					report.Manifests = append(report.Manifests, DriftManifestStatus{
						Path: manifestPath, Ecosystem: pair.eco, DriftCount: 0,
					})
					continue
				}
			}
			// Manifest present but nothing pins it — the most dangerous state.
			// Fail closed by reporting it as drift.
			report.Manifests = append(report.Manifests, DriftManifestStatus{
				Path:       manifestPath,
				Ecosystem:  pair.eco,
				DriftCount: 1,
				Drifted: []DriftEntry{{
					Name:            pair.manifest,
					DeclaredVersion: "requires a lockfile",
					LockedVersion:   "(missing lockfile — dependencies unpinned)",
				}},
			})
			continue
		}

		// A lockfile is present. Generic ecosystems have no parser, so the best we
		// can verify is that a valid lockfile exists; likewise a non-primary but
		// valid lockfile (pnpm/yarn/bun) pins the deps but cannot be version-diffed
		// by this ecosystem's checker. Either way, report present-and-pinned
		// (0 drift) rather than misreading it.
		if pair.checker == nil || presentLock != filepath.Join(root, pair.primaryLock) {
			report.Manifests = append(report.Manifests, DriftManifestStatus{
				Path: manifestPath, Ecosystem: pair.eco, DriftCount: 0,
			})
			continue
		}

		drifted, err := pair.checker(manifestPath, presentLock)
		if err != nil {
			return nil, fmt.Errorf("checking drift for %s: %w", pair.manifest, err)
		}

		report.Manifests = append(report.Manifests, DriftManifestStatus{
			Path:       manifestPath,
			Ecosystem:  pair.eco,
			DriftCount: len(drifted),
			Drifted:    drifted,
		})
	}

	return report, nil
}

// manifestPresent reports whether the manifest for a pair exists under root and
// returns its concrete path. The manifest name may be a glob (e.g. "*.csproj"
// for .NET); the first match wins.
func manifestPresent(root, manifest string) (string, bool) {
	if strings.ContainsAny(manifest, "*?[") {
		matches, err := filepath.Glob(filepath.Join(root, manifest))
		if err != nil || len(matches) == 0 {
			return "", false
		}
		return matches[0], true
	}
	p := filepath.Join(root, manifest)
	if _, err := os.Stat(p); err != nil {
		return "", false
	}
	return p, true
}

// firstPresentLockfile returns the path of the first catalog-valid lockfile for
// the pair's ecosystem that exists under root, preferring the primary (diffable)
// lockfile so it wins when several are present; "" when none exists.
//
// The manifest is never treated as its own lockfile: some files are listed as
// both a manifest and a lockfile for an ecosystem (e.g. requirements.txt for
// Python, vcpkg.json for C++). A lone requirements.txt is an unpinned manifest,
// not a pinned one, so it must not satisfy its own lockfile requirement.
func firstPresentLockfile(root string, pair lockfilePair) string {
	var candidates []string
	if pair.primaryLock != "" {
		candidates = append(candidates, pair.primaryLock)
	}
	for _, lf := range ecosystem.LockFilesByEcosystem[pair.eco] {
		if lf != pair.primaryLock && lf != pair.manifest {
			candidates = append(candidates, lf)
		}
	}
	for _, lf := range candidates {
		p := filepath.Join(root, lf)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// checkGoDrift reports go.mod requirements whose exact module version go.sum
// does not checksum. Go has no single "locked version" per module: go.sum lists
// every version the module graph touched (older ones often as /go.mod-only
// lines), while go.mod itself pins the selected version. So the check is
// membership — is the required mod@version in go.sum — not a comparison with
// one go.sum version. A require redirected by a replace directive is checked
// against its replacement; one replaced by a local directory has no go.sum
// entry and is skipped.
func checkGoDrift(manifestPath, lockfilePath string) ([]DriftEntry, error) {
	mf, err := parseGoModFile(manifestPath)
	if err != nil {
		return nil, err
	}

	sum, err := parseGoSum(lockfilePath)
	if err != nil {
		return nil, err
	}

	replaces := goModReplacements(mf)
	var drifted []DriftEntry
	for _, r := range mf.Require {
		if r == nil {
			continue
		}
		mod, ver := r.Mod.Path, r.Mod.Version
		if rep, ok := replaces.lookup(mod, ver); ok {
			if rep.Version == "" {
				continue // local directory replacement: not checksummed in go.sum
			}
			mod, ver = rep.Path, rep.Version
		}
		if sum.has(mod, ver) {
			continue
		}
		locked := strings.Join(sum.versions[mod], ", ")
		if locked == "" {
			locked = notInLockfile
		}
		drifted = append(drifted, DriftEntry{
			Name:            r.Mod.Path,
			DeclaredVersion: r.Mod.Version,
			LockedVersion:   locked,
		})
	}

	return drifted, nil
}

// goSum is the set of module versions a go.sum checksums.
type goSum struct {
	entries  map[string]bool     // "mod@version", from zip-hash or /go.mod lines
	versions map[string][]string // module -> versions in file order, deduplicated
}

func (g goSum) has(mod, ver string) bool { return g.entries[mod+"@"+ver] }

func parseGoSum(path string) (goSum, error) {
	f, err := os.Open(path)
	if err != nil {
		return goSum{}, fmt.Errorf("opening go.sum: %w", err)
	}
	defer f.Close()

	sum := goSum{entries: make(map[string]bool), versions: make(map[string][]string)}
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) < 2 {
			continue
		}

		// go.sum has entries like "module v1.2.3/go.mod h1:..." and "module v1.2.3 h1:...".
		mod := parts[0]
		ver := strings.TrimSuffix(parts[1], "/go.mod")
		key := mod + "@" + ver
		if !sum.entries[key] {
			sum.entries[key] = true
			sum.versions[mod] = append(sum.versions[mod], ver)
		}
	}

	if err := scanner.Err(); err != nil {
		return goSum{}, fmt.Errorf("scanning go.sum: %w", err)
	}

	return sum, nil
}

// goReplacements maps a replaced module ("path" or "path@version") to its
// replacement. A replacement with an empty Version is a local directory.
type goReplacements map[string]goModuleVersion

type goModuleVersion struct{ Path, Version string }

// lookup returns the replacement for mod@ver: a version-specific replace wins
// over one that covers every version of the module.
func (r goReplacements) lookup(mod, ver string) (goModuleVersion, bool) {
	if rep, ok := r[mod+"@"+ver]; ok {
		return rep, true
	}
	rep, ok := r[mod]
	return rep, ok
}

// goModReplacements reads the replace directives of a go.mod. modfile.ParseLax
// (used so newer directives do not break parsing) skips replace statements, so
// they are read from the parsed syntax tree instead: each is
// "old [old-version] => new [new-version]".
func goModReplacements(mf *modfile.File) goReplacements {
	reps := make(goReplacements)
	add := func(tokens []string) {
		arrow := slices.Index(tokens, "=>")
		if arrow < 1 || arrow > 2 || len(tokens)-arrow-1 < 1 || len(tokens)-arrow-1 > 2 {
			return // malformed; the go command would reject it too
		}
		key := unquoteModToken(tokens[0])
		if arrow == 2 {
			key += "@" + unquoteModToken(tokens[1])
		}
		rep := goModuleVersion{Path: unquoteModToken(tokens[arrow+1])}
		if len(tokens) == arrow+3 {
			rep.Version = unquoteModToken(tokens[arrow+2])
		}
		reps[key] = rep
	}
	if mf == nil || mf.Syntax == nil {
		return reps
	}
	for _, stmt := range mf.Syntax.Stmt {
		switch x := stmt.(type) {
		case *modfile.Line:
			if len(x.Token) > 0 && x.Token[0] == "replace" {
				add(x.Token[1:])
			}
		case *modfile.LineBlock:
			if len(x.Token) > 0 && x.Token[0] == "replace" {
				for _, l := range x.Line {
					add(l.Token)
				}
			}
		}
	}
	return reps
}

func unquoteModToken(tok string) string {
	if u, err := strconv.Unquote(tok); err == nil {
		return u
	}
	return tok
}

func checkJSDrift(manifestPath, lockfilePath string) ([]DriftEntry, error) {
	declared, err := parsePackageJSON(manifestPath)
	if err != nil {
		return nil, err
	}

	locked, err := parsePackageLock(lockfilePath)
	if err != nil {
		return nil, err
	}

	var drifted []DriftEntry
	for _, dep := range declared {
		lockedVer, ok := locked[dep.Name]
		if !ok {
			// A "workspace:" spec is a pnpm/yarn workspace link, which
			// package-lock.json never records; anything else npm would have
			// locked, so its absence means the lockfile is stale.
			if !strings.HasPrefix(dep.DeclaredVersion, "workspace:") {
				drifted = append(drifted, DriftEntry{
					Name:            dep.Name,
					DeclaredVersion: dep.DeclaredVersion,
					LockedVersion:   notInLockfile,
				})
			}
			continue
		}
		if !jsSemverSatisfies(dep.DeclaredVersion, lockedVer) {
			drifted = append(drifted, DriftEntry{
				Name:            dep.Name,
				DeclaredVersion: dep.DeclaredVersion,
				LockedVersion:   lockedVer,
			})
		}
	}

	return drifted, nil
}

// parsePackageLock reads the locked versions from a package-lock.json.
//
// NOTE: this is deliberately NOT merged with internal/vulnscan/lockparse.go.
// That parser produces []Package for OSV queries and strips the leading "v" from
// go.sum versions; drift detection must instead keep versions in the SAME textual
// form the manifest declares (go.mod uses "v1.2.3") so declared-vs-locked
// comparison is apples-to-apples. Merging the two would break that comparison.
func parsePackageLock(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading package-lock.json: %w", err)
	}

	var lockfile struct {
		Packages map[string]struct {
			Version string `json:"version"`
		} `json:"packages"`
		// lockfileVersion 1 has no "packages" map — top-level deps live under
		// "dependencies". Read it too so a v1 lock is not parsed as empty (which
		// would let real drift go undetected).
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(data, &lockfile); err != nil {
		return nil, fmt.Errorf("parsing package-lock.json: %w", err)
	}

	versions := make(map[string]string)
	for key, pkg := range lockfile.Packages {
		// Top-level dependencies are under "node_modules/<name>"
		if !strings.HasPrefix(key, "node_modules/") {
			continue
		}
		name := strings.TrimPrefix(key, "node_modules/")
		// Skip nested node_modules
		if strings.Contains(name, "node_modules/") {
			continue
		}
		versions[name] = pkg.Version
	}

	// lockfileVersion 1 fallback: only when the modern packages map yielded
	// nothing, so a v2/v3 lock is never double-counted.
	if len(versions) == 0 {
		for name, dep := range lockfile.Dependencies {
			versions[name] = dep.Version
		}
	}

	return versions, nil
}

// jsSemverSatisfies reports whether an npm lockfile version satisfies a
// package.json constraint. npm range syntax passes through unchanged: caret
// (^4.18.0) allows >=4.18.0 <5.0.0 — with the correct 0.x cap, so ^0.2.3
// allows >=0.2.3 <0.3.0 — tilde (~4.18.0) allows >=4.18.0 <4.19.0, and a bare
// full version (4.17.21) is an exact pin.
func jsSemverSatisfies(constraint, locked string) bool {
	return constraintSatisfies(constraint, locked)
}

// cargoSemverSatisfies reports whether a Cargo.lock version satisfies a
// Cargo.toml requirement. Cargo's default requirement is caret semantics, so
// a bare version ("1.0") is rewritten to "^1.0" before evaluation; explicit
// operators (^, ~, =, >=, wildcards, ...) pass through unchanged.
func cargoSemverSatisfies(constraint, locked string) bool {
	constraint = strings.TrimSpace(constraint)
	if isBareVersion(constraint) {
		constraint = "^" + constraint
	}
	return constraintSatisfies(constraint, locked)
}

// constraintSatisfies reports whether locked satisfies the declared range.
// This decides drift: a locked version outside the range is drift, and any
// parse failure — of the constraint or of the locked version — is also
// treated as drift (fail closed) rather than silently passing.
func constraintSatisfies(constraint, locked string) bool {
	c, err := semver.NewConstraint(strings.TrimSpace(constraint))
	if err != nil {
		return false
	}
	v, err := semver.NewVersion(strings.TrimSpace(locked))
	if err != nil {
		return false
	}
	return c.Check(v)
}

// isBareVersion reports whether the requirement string carries no explicit
// range operator or wildcard, i.e. it starts with a (possibly v-prefixed)
// numeric version like "1", "1.0", or "1.0.203".
func isBareVersion(constraint string) bool {
	c := strings.TrimPrefix(constraint, "v")
	return c != "" && c[0] >= '0' && c[0] <= '9'
}

// checkCargoDrift reports Cargo.toml version requirements that no Cargo.lock
// entry for the crate satisfies. A crate can be locked at several versions (two
// majors pulled in by different dependents), so the requirement is met when any
// of them satisfies it. Path, git and workspace-inherited specs with no version
// requirement of their own are not compared.
func checkCargoDrift(manifestPath, lockfilePath string) ([]DriftEntry, error) {
	declared, err := parseCargoDeps(manifestPath)
	if err != nil {
		return nil, err
	}

	locked, err := parseCargoLock(lockfilePath)
	if err != nil {
		return nil, err
	}

	var drifted []DriftEntry
	for _, dep := range declared {
		if dep.Version == "" {
			continue
		}
		versions := locked[dep.Package]
		if len(versions) == 0 {
			drifted = append(drifted, DriftEntry{
				Name:            dep.Name,
				DeclaredVersion: dep.Version,
				LockedVersion:   notInLockfile,
			})
			continue
		}
		// Cargo version requirements default to caret semantics, so a bare "1.0"
		// means >=1.0.0 <2.0.0.
		satisfied := slices.ContainsFunc(versions, func(v string) bool {
			return cargoSemverSatisfies(dep.Version, v)
		})
		if !satisfied {
			drifted = append(drifted, DriftEntry{
				Name:            dep.Name,
				DeclaredVersion: dep.Version,
				LockedVersion:   strings.Join(versions, ", "),
			})
		}
	}

	return drifted, nil
}

// parseCargoLock returns every locked version of each crate in a Cargo.lock.
func parseCargoLock(path string) (map[string][]string, error) {
	var lock struct {
		Package []struct {
			Name    string `toml:"name"`
			Version string `toml:"version"`
		} `toml:"package"`
	}
	if _, err := toml.DecodeFile(path, &lock); err != nil {
		return nil, fmt.Errorf("parsing Cargo.lock: %w", err)
	}

	versions := make(map[string][]string)
	for _, p := range lock.Package {
		if p.Name == "" || p.Version == "" || slices.Contains(versions[p.Name], p.Version) {
			continue
		}
		versions[p.Name] = append(versions[p.Name], p.Version)
	}
	return versions, nil
}
