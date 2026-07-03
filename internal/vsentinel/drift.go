package vsentinel

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// lockfilePair is a single manifest/lockfile relationship to check for one
// ecosystem. It is derived from the ecosystem catalog (see driftPairs), never
// hardcoded, so every catalog ecosystem is covered automatically.
type lockfilePair struct {
	manifest string // manifest file to look for under root; may be a glob (e.g. "*.csproj")
	eco      string
	// lockfile is the PRIMARY lockfile this ecosystem's checker can diff against
	// the manifest. Other catalog-valid lockfiles (e.g. pnpm/yarn/bun for JS)
	// still count as "pinned" but are not version-diffed. It is empty for a
	// generic (unparsed) ecosystem, which has no diffable lockfile format.
	lockfile string
	// checker version-diffs manifest vs lockfile. It is nil for generic
	// ecosystems, which fall back to presence-based, fail-closed detection.
	checker func(manifestPath, lockfilePath string) ([]DriftEntry, error)
	// declaredCount returns how many dependencies the manifest declares, used to
	// distinguish a dependency-free manifest (legitimately unlocked) from an
	// unpinned one. It is nil for generic ecosystems: they cannot parse deps, so
	// a present manifest with no lockfile always fails closed.
	declaredCount func(manifestPath string) (int, error)
}

// specificChecker binds an ecosystem that has a dedicated manifest/lockfile
// parser to its precise drift checker. Ecosystems absent from this map fall
// back to the generic, presence-based checker (fail closed on a missing
// lockfile). This map lives here — not in the catalog — because it wires
// ecosystems to parser functions defined in this package; it is not a
// hardcoded coverage list. Coverage itself is derived from the catalog.
type specificChecker struct {
	// primaryLock is the one lockfile format this checker can version-diff.
	primaryLock   string
	checker       func(manifestPath, lockfilePath string) ([]DriftEntry, error)
	declaredCount func(manifestPath string) (int, error)
}

var specificCheckers = map[string]specificChecker{
	ecosystem.NameGo:         {"go.sum", checkGoDrift, goDeclaredCount},
	ecosystem.NameJavaScript: {"package-lock.json", checkJSDrift, jsDeclaredCount},
	ecosystem.NameRust:       {"Cargo.lock", checkCargoDrift, cargoDeclaredCount},
}

func goDeclaredCount(p string) (int, error)    { d, err := parseGoMod(p); return len(d), err }
func jsDeclaredCount(p string) (int, error)    { d, err := parsePackageJSON(p); return len(d), err }
func cargoDeclaredCount(p string) (int, error) { d, err := parseCargoToml(p); return len(d), err }

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
		sc, parsed := specificCheckers[eco]
		for _, manifest := range ecosystem.ManifestsByEcosystem[eco] {
			p := lockfilePair{manifest: manifest, eco: eco}
			if parsed {
				p.lockfile = sc.primaryLock
				p.checker = sc.checker
				p.declaredCount = sc.declaredCount
			}
			pairs = append(pairs, p)
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
		// can verify is that a valid lockfile exists — report it pinned (0 drift).
		if pair.checker == nil {
			report.Manifests = append(report.Manifests, DriftManifestStatus{
				Path: manifestPath, Ecosystem: pair.eco, DriftCount: 0,
			})
			continue
		}

		// A non-primary but valid lockfile (pnpm/yarn/bun) still pins the deps,
		// but this ecosystem's checker can only version-diff the primary format.
		// Report it present-and-pinned (0 drift) rather than misreading it.
		if presentLock != filepath.Join(root, pair.lockfile) {
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
	if pair.lockfile != "" {
		candidates = append(candidates, pair.lockfile)
	}
	for _, lf := range ecosystem.LockFilesByEcosystem[pair.eco] {
		if lf != pair.lockfile && lf != pair.manifest {
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

func checkGoDrift(manifestPath, lockfilePath string) ([]DriftEntry, error) {
	declared, err := parseGoMod(manifestPath)
	if err != nil {
		return nil, err
	}

	sumVersions, err := parseGoSum(lockfilePath)
	if err != nil {
		return nil, err
	}

	var drifted []DriftEntry
	for _, dep := range declared {
		locked, ok := sumVersions[dep.Name]
		if !ok {
			continue
		}
		if locked != dep.DeclaredVersion {
			drifted = append(drifted, DriftEntry{
				Name:            dep.Name,
				DeclaredVersion: dep.DeclaredVersion,
				LockedVersion:   locked,
			})
		}
	}

	return drifted, nil
}

func parseGoSum(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening go.sum: %w", err)
	}
	defer f.Close()

	versions := make(map[string]string)
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		mod := parts[0]
		ver := parts[1]

		// go.sum has entries like "module v1.2.3/go.mod h1:..." and "module v1.2.3 h1:..."
		// Strip /go.mod suffix from version
		ver = strings.TrimSuffix(ver, "/go.mod")

		// Keep first version seen per module (avoid overwriting with /go.mod variant)
		if _, exists := versions[mod]; !exists {
			versions[mod] = ver
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning go.sum: %w", err)
	}

	return versions, nil
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

func checkCargoDrift(manifestPath, lockfilePath string) ([]DriftEntry, error) {
	declared, err := parseCargoToml(manifestPath)
	if err != nil {
		return nil, err
	}

	locked, err := parseCargoLock(lockfilePath)
	if err != nil {
		return nil, err
	}

	var drifted []DriftEntry
	for _, dep := range declared {
		lockedVer, ok := locked[dep.Name]
		if !ok {
			continue
		}
		// Cargo version requirements default to caret semantics, so a bare "1.0"
		// means >=1.0.0 <2.0.0.
		if !cargoSemverSatisfies(dep.DeclaredVersion, lockedVer) {
			drifted = append(drifted, DriftEntry{
				Name:            dep.Name,
				DeclaredVersion: dep.DeclaredVersion,
				LockedVersion:   lockedVer,
			})
		}
	}

	return drifted, nil
}

func parseCargoLock(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening Cargo.lock: %w", err)
	}
	defer f.Close()

	versions := make(map[string]string)
	scanner := bufio.NewScanner(f)
	var currentName string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if v, ok := strings.CutPrefix(line, "name = "); ok {
			currentName = strings.Trim(v, "\"")
		}
		if v, ok := strings.CutPrefix(line, "version = "); ok && currentName != "" {
			versions[currentName] = strings.Trim(v, "\"")
			currentName = ""
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning Cargo.lock: %w", err)
	}

	return versions, nil
}
