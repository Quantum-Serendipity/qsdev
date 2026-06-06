package vsentinel

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
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
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening go.mod: %w", err)
	}
	defer f.Close()

	var deps []DepStatus
	scanner := bufio.NewScanner(f)
	inRequire := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "require (" {
			inRequire = true
			continue
		}
		if inRequire && line == ")" {
			inRequire = false
			continue
		}

		if inRequire {
			line = strings.TrimSuffix(line, "// indirect")
			line = strings.TrimSpace(line)
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				deps = append(deps, DepStatus{
					Name:            parts[0],
					DeclaredVersion: parts[1],
				})
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning go.mod: %w", err)
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
