package vsentinel

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type lockfilePair struct {
	manifest string
	lockfile string
	eco      string
	checker  func(manifestPath, lockfilePath string) ([]DriftEntry, error)
}

var lockfilePairs = []lockfilePair{
	{"go.mod", "go.sum", "go", checkGoDrift},
	{"package.json", "package-lock.json", "javascript", checkJSDrift},
	{"Cargo.toml", "Cargo.lock", "rust", checkCargoDrift},
}

func DetectDrift(root string) (*DriftReport, error) {
	report := &DriftReport{}

	for _, pair := range lockfilePairs {
		manifestPath := filepath.Join(root, pair.manifest)
		lockfilePath := filepath.Join(root, pair.lockfile)

		// No manifest for this ecosystem: nothing to check.
		if _, err := os.Stat(manifestPath); err != nil {
			continue
		}

		// Manifest present but lockfile absent is the most dangerous state
		// (fully unpinned dependencies). Fail closed by reporting it as drift
		// rather than silently skipping the ecosystem.
		if _, err := os.Stat(lockfilePath); err != nil {
			report.Manifests = append(report.Manifests, DriftManifestStatus{
				Path:       manifestPath,
				Ecosystem:  pair.eco,
				DriftCount: 1,
				Drifted: []DriftEntry{{
					Name:            pair.manifest,
					DeclaredVersion: "requires " + pair.lockfile,
					LockedVersion:   "(missing lockfile — dependencies unpinned)",
				}},
			})
			continue
		}

		drifted, err := pair.checker(manifestPath, lockfilePath)
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

func parsePackageLock(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading package-lock.json: %w", err)
	}

	var lockfile struct {
		Packages map[string]struct {
			Version string `json:"version"`
		} `json:"packages"`
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

	return versions, nil
}

// jsSemverSatisfies reports whether an npm lockfile version satisfies a
// package.json constraint. A bare version (no ^/~) is treated as an exact pin.
func jsSemverSatisfies(constraint, locked string) bool {
	return semverSatisfies(constraint, locked, 0)
}

// semverSatisfies reports whether locked satisfies the declared constraint.
//
// Unlike a naive prefix/major comparison, it enforces BOTH bounds:
//   - the lower bound: locked must be >= the declared floor, so a within-major
//     downgrade below the floor (e.g. ^4.18.0 vs 4.0.0) is flagged as drift; and
//   - the upper bound implied by the range operator: caret (^) allows anything
//     within the same major, tilde (~) within the same major.minor, and an exact
//     pin requires all three components to match.
//
// bareOp is the operator assumed when the constraint carries no explicit ^/~
// prefix: 0 means exact (npm's default), '^' means caret (Cargo's default).
func semverSatisfies(constraint, locked string, bareOp byte) bool {
	constraint = strings.TrimSpace(constraint)
	locked = strings.TrimSpace(locked)
	if constraint == locked {
		return true
	}

	op := bareOp
	if len(constraint) > 0 {
		switch constraint[0] {
		case '^', '~':
			op = constraint[0]
		}
	}

	floor := strings.TrimLeft(constraint, "^~>=<vV ")
	fMaj, fMin, fPat := parseSemver(floor)
	lMaj, lMin, lPat := parseSemver(locked)

	// Lower bound: locked must not be below the declared floor.
	if compareSemver(lMaj, lMin, lPat, fMaj, fMin, fPat) < 0 {
		return false
	}

	// Upper bound.
	switch op {
	case '^':
		return lMaj == fMaj
	case '~':
		return lMaj == fMaj && lMin == fMin
	default:
		return lMaj == fMaj && lMin == fMin && lPat == fPat
	}
}

// parseSemver parses a dotted version into up to three numeric components.
// Missing components default to 0; a build/pre-release suffix ("-"/"+") and any
// non-numeric segment are ignored (treated as 0).
func parseSemver(v string) (major, minor, patch int) {
	v = strings.TrimSpace(v)
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	parts := strings.SplitN(v, ".", 3)
	get := func(i int) int {
		if i >= len(parts) {
			return 0
		}
		n, err := strconv.Atoi(strings.TrimSpace(parts[i]))
		if err != nil {
			return 0
		}
		return n
	}
	return get(0), get(1), get(2)
}

// compareSemver returns -1, 0, or 1 comparing (aMaj.aMin.aPat) to (bMaj.bMin.bPat).
func compareSemver(aMaj, aMin, aPat, bMaj, bMin, bPat int) int {
	for _, d := range [][2]int{{aMaj, bMaj}, {aMin, bMin}, {aPat, bPat}} {
		if d[0] < d[1] {
			return -1
		}
		if d[0] > d[1] {
			return 1
		}
	}
	return 0
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
		// means >=1.0.0 <2.0.0. Using segment-aware comparison avoids the old
		// string-prefix false negatives (e.g. declared "1" matching "10.0.0").
		if !semverSatisfies(dep.DeclaredVersion, lockedVer, '^') {
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
