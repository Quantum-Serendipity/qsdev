package vsentinel

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/mod/modfile"
)

var knownManifests = map[string]string{
	"go.mod":           "go",
	"package.json":     "javascript",
	"Cargo.toml":       "rust",
	"pyproject.toml":   "python",
	"requirements.txt": "python",
}

func CheckVersions(root string) (*VersionReport, error) {
	report := &VersionReport{
		LastCheckTime: time.Now(),
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("reading root directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		eco, ok := knownManifests[entry.Name()]
		if !ok {
			continue
		}

		path := filepath.Join(root, entry.Name())
		deps, err := parseManifest(path, entry.Name())
		if err != nil {
			return nil, fmt.Errorf("parsing manifest %s: %w", entry.Name(), err)
		}

		report.Manifests = append(report.Manifests, ManifestStatus{
			Path:         path,
			Ecosystem:    eco,
			Dependencies: deps,
		})
	}

	return report, nil
}

func parseManifest(path, filename string) ([]DepStatus, error) {
	switch filename {
	case "go.mod":
		return parseGoMod(path)
	case "package.json":
		return parsePackageJSON(path)
	case "Cargo.toml":
		return parseCargoToml(path)
	case "pyproject.toml":
		return parsePyprojectToml(path)
	case "requirements.txt":
		return parseRequirementsTxt(path)
	default:
		return nil, nil
	}
}

func parseGoMod(path string) ([]DepStatus, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading go.mod: %w", err)
	}

	// ParseLax covers every require shape — single-line, block, commented, and
	// quoted paths — uniformly, and tolerates directives this parser ignores
	// (e.g. a newer toolchain line). It preserves the declared version text
	// verbatim (keeping the "v" prefix), so declared-vs-locked comparison against
	// go.sum stays apples-to-apples. Both direct and indirect requires are
	// reported, matching the manifest's full require set.
	mf, err := modfile.ParseLax(path, data, nil)
	if err != nil {
		return nil, fmt.Errorf("parsing go.mod: %w", err)
	}

	deps := make([]DepStatus, 0, len(mf.Require))
	for _, r := range mf.Require {
		if r == nil {
			continue
		}
		deps = append(deps, DepStatus{Name: r.Mod.Path, DeclaredVersion: r.Mod.Version})
	}

	return deps, nil
}

func parsePackageJSON(path string) ([]DepStatus, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading package.json: %w", err)
	}

	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("parsing package.json: %w", err)
	}

	var deps []DepStatus
	for name, ver := range pkg.Dependencies {
		deps = append(deps, DepStatus{
			Name:            name,
			DeclaredVersion: ver,
		})
	}
	for name, ver := range pkg.DevDependencies {
		deps = append(deps, DepStatus{
			Name:            name,
			DeclaredVersion: ver,
		})
	}

	return deps, nil
}

func parseCargoToml(path string) ([]DepStatus, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening Cargo.toml: %w", err)
	}
	defer f.Close()

	var deps []DepStatus
	scanner := bufio.NewScanner(f)
	inDeps := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "[dependencies]" || line == "[dev-dependencies]" {
			inDeps = true
			continue
		}
		if strings.HasPrefix(line, "[") {
			inDeps = false
			continue
		}

		if inDeps && strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			name := strings.TrimSpace(parts[0])
			ver := strings.TrimSpace(parts[1])
			ver = strings.Trim(ver, "\"")
			if name != "" && ver != "" {
				deps = append(deps, DepStatus{
					Name:            name,
					DeclaredVersion: ver,
				})
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning Cargo.toml: %w", err)
	}

	return deps, nil
}

func parsePyprojectToml(path string) ([]DepStatus, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening pyproject.toml: %w", err)
	}
	defer f.Close()

	var deps []DepStatus
	scanner := bufio.NewScanner(f)
	inDeps := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "dependencies = [" {
			inDeps = true
			continue
		}
		if inDeps && line == "]" {
			inDeps = false
			continue
		}

		if inDeps {
			entry := strings.Trim(line, "\",")
			entry = strings.TrimSpace(entry)
			if entry == "" {
				continue
			}

			name, ver := splitPythonDep(entry)
			deps = append(deps, DepStatus{
				Name:            name,
				DeclaredVersion: ver,
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning pyproject.toml: %w", err)
	}

	return deps, nil
}

func splitPythonDep(s string) (string, string) {
	for _, op := range []string{">=", "<=", "==", "!=", "~=", ">", "<"} {
		if idx := strings.Index(s, op); idx >= 0 {
			return strings.TrimSpace(s[:idx]), strings.TrimSpace(s[idx:])
		}
	}
	return s, ""
}

func parseRequirementsTxt(path string) ([]DepStatus, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening requirements.txt: %w", err)
	}
	defer f.Close()

	var deps []DepStatus
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") {
			continue
		}

		name, ver := splitPythonDep(line)
		if name != "" {
			deps = append(deps, DepStatus{
				Name:            name,
				DeclaredVersion: ver,
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning requirements.txt: %w", err)
	}

	return deps, nil
}
