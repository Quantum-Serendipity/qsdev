package vsentinel

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

		if _, err := os.Stat(manifestPath); err != nil {
			continue
		}
		if _, err := os.Stat(lockfilePath); err != nil {
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

// jsSemverSatisfies does basic check: exact match, or caret/tilde prefix match.
// For exact versions, compares directly. For ^/~ prefixed constraints, checks
// that the locked version starts with the expected major (^) or major.minor (~).
func jsSemverSatisfies(constraint, locked string) bool {
	if constraint == locked {
		return true
	}

	clean := strings.TrimLeft(constraint, "^~>=<")
	if clean == locked {
		return true
	}

	prefix := ""
	if len(constraint) > 0 {
		prefix = string(constraint[0])
	}

	switch prefix {
	case "^":
		// Caret: compatible with major version
		cParts := strings.SplitN(clean, ".", 2)
		lParts := strings.SplitN(locked, ".", 2)
		if len(cParts) < 1 || len(lParts) < 1 {
			return false
		}
		return cParts[0] == lParts[0]
	case "~":
		// Tilde: compatible with major.minor
		cParts := strings.SplitN(clean, ".", 3)
		lParts := strings.SplitN(locked, ".", 3)
		if len(cParts) < 2 || len(lParts) < 2 {
			return false
		}
		return cParts[0] == lParts[0] && cParts[1] == lParts[1]
	default:
		return false
	}
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
		clean := strings.TrimLeft(dep.DeclaredVersion, "^~>=<")
		if !strings.HasPrefix(lockedVer, clean) {
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
