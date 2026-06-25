package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"golang.org/x/mod/modfile"
	"gopkg.in/yaml.v3"
)

// Ecosystem identities assigned to detected packages. They double as the prefix
// in {ecosystem}:{name} qualified names.
const (
	ecoNpm   = "npm"
	ecoPnpm  = "pnpm"
	ecoCargo = "cargo"
	ecoGo    = "go"
	ecoUv    = "uv"
)

// Workspace membership configuration filenames (parsed at the workspace root).
const (
	fileNpmManifest    = "package.json"
	filePnpmWorkspace  = "pnpm-workspace.yaml"
	fileCargoManifest  = "Cargo.toml"
	fileGoWork         = "go.work"
	filePyprojectToml  = "pyproject.toml"
	fileGoModManifest  = "go.mod"
	filePnpmWorkspace2 = "pnpm-workspace.yml"
)

// ParseNpmWorkspaces reads the root package.json and returns its workspace
// include and exclude patterns. The "workspaces" field has two forms: an array
// of patterns, or (Yarn) an object whose "packages" array holds them. Entries
// prefixed with "!" are negations and are returned as excludes.
func ParseNpmWorkspaces(root string) (includes, excludes []string, err error) {
	data, err := os.ReadFile(filepath.Join(root, fileNpmManifest))
	if err != nil {
		return nil, nil, fmt.Errorf("reading %s: %w", fileNpmManifest, err)
	}

	var manifest struct {
		Workspaces json.RawMessage `json:"workspaces"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, nil, fmt.Errorf("parsing %s: %w", fileNpmManifest, err)
	}
	if len(manifest.Workspaces) == 0 {
		return nil, nil, nil
	}

	var patterns []string
	if err := json.Unmarshal(manifest.Workspaces, &patterns); err != nil {
		// Not an array; try the Yarn object form {"packages": [...]}.
		var obj struct {
			Packages []string `json:"packages"`
		}
		if err2 := json.Unmarshal(manifest.Workspaces, &obj); err2 != nil {
			return nil, nil, fmt.Errorf("parsing %s workspaces field: %w", fileNpmManifest, err)
		}
		patterns = obj.Packages
	}

	inc, exc := partitionNegations(patterns)
	return inc, exc, nil
}

// ParsePnpmWorkspaces reads pnpm-workspace.yaml (or .yml) and returns its
// "packages" patterns. pnpm encodes exclusions as "!"-prefixed entries, which
// are returned as excludes.
func ParsePnpmWorkspaces(root string) (includes, excludes []string, err error) {
	path := filepath.Join(root, filePnpmWorkspace)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Accept the .yml spelling as a fallback.
			path = filepath.Join(root, filePnpmWorkspace2)
			data, err = os.ReadFile(path)
		}
		if err != nil {
			return nil, nil, fmt.Errorf("reading %s: %w", filePnpmWorkspace, err)
		}
	}

	var doc struct {
		Packages []string `yaml:"packages"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, nil, fmt.Errorf("parsing %s: %w", filePnpmWorkspace, err)
	}

	inc, exc := partitionNegations(doc.Packages)
	return inc, exc, nil
}

// ParseCargoWorkspaces reads the root Cargo.toml [workspace] table and returns
// its members as includes and its exclude entries as excludes. Cargo's exclude
// uses prefix-match semantics (a named directory and its subtree); the
// GlobResolver applies that semantics, so the raw entries are returned as-is.
func ParseCargoWorkspaces(root string) (includes, excludes []string, err error) {
	data, err := os.ReadFile(filepath.Join(root, fileCargoManifest))
	if err != nil {
		return nil, nil, fmt.Errorf("reading %s: %w", fileCargoManifest, err)
	}

	var manifest struct {
		Workspace struct {
			Members []string `toml:"members"`
			Exclude []string `toml:"exclude"`
		} `toml:"workspace"`
	}
	if err := toml.Unmarshal(data, &manifest); err != nil {
		return nil, nil, fmt.Errorf("parsing %s: %w", fileCargoManifest, err)
	}

	return manifest.Workspace.Members, manifest.Workspace.Exclude, nil
}

// ParseGoWorkspaces reads go.work and returns each "use" directory as an
// include. go.work has no exclude concept, so excludes is always empty. The
// root entry ("use .") is preserved as ".".
func ParseGoWorkspaces(root string) (includes, excludes []string, err error) {
	path := filepath.Join(root, fileGoWork)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("reading %s: %w", fileGoWork, err)
	}

	wf, err := modfile.ParseWork(path, data, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing %s: %w", fileGoWork, err)
	}

	includes = make([]string, 0, len(wf.Use))
	for _, u := range wf.Use {
		if u == nil || strings.TrimSpace(u.Path) == "" {
			continue
		}
		includes = append(includes, u.Path)
	}
	return includes, nil, nil
}

// ParseUvWorkspaces reads pyproject.toml [tool.uv.workspace] and returns its
// members as includes and its exclude entries as excludes.
func ParseUvWorkspaces(root string) (includes, excludes []string, err error) {
	data, err := os.ReadFile(filepath.Join(root, filePyprojectToml))
	if err != nil {
		return nil, nil, fmt.Errorf("reading %s: %w", filePyprojectToml, err)
	}

	var manifest struct {
		Tool struct {
			Uv struct {
				Workspace struct {
					Members []string `toml:"members"`
					Exclude []string `toml:"exclude"`
				} `toml:"workspace"`
			} `toml:"uv"`
		} `toml:"tool"`
	}
	if err := toml.Unmarshal(data, &manifest); err != nil {
		return nil, nil, fmt.Errorf("parsing %s: %w", filePyprojectToml, err)
	}

	ws := manifest.Tool.Uv.Workspace
	return ws.Members, ws.Exclude, nil
}

// partitionNegations splits patterns into positive includes and the excludes
// implied by a leading "!" (npm/pnpm negation). The "!" prefix is stripped.
func partitionNegations(patterns []string) (includes, excludes []string) {
	for _, p := range patterns {
		t := strings.TrimSpace(p)
		if t == "" {
			continue
		}
		if strings.HasPrefix(t, "!") {
			excludes = append(excludes, strings.TrimSpace(t[1:]))
			continue
		}
		includes = append(includes, t)
	}
	return includes, excludes
}
