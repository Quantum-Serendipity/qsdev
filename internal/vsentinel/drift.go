package vsentinel

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver/v3"
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
