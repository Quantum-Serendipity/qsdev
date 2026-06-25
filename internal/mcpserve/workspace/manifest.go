package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"golang.org/x/mod/modfile"
)

// ecosystemSpec ties one workspace ecosystem together: how to parse its root
// membership configuration, the per-member manifest filename used both for
// validation and watching, and how to read a member manifest's identity.
type ecosystemSpec struct {
	// id is the ecosystem identity (ecoNpm, ecoPnpm, ...).
	id string
	// configFile is the membership configuration filename at the workspace root.
	configFile string
	// manifest is the per-member manifest filename that a resolved directory must
	// contain to qualify as a package of this ecosystem.
	manifest string
	// parse extracts the include/exclude patterns from the root configuration.
	parse func(root string) (includes, excludes []string, err error)
	// readManifest extracts a member's declared name and dependency names from
	// its manifest file at absolute path.
	readManifest func(manifestPath string) (name string, deps []string, err error)
}

// allEcosystems returns the five workspace ecosystem specs in a stable order.
func allEcosystems() []ecosystemSpec {
	return []ecosystemSpec{
		{id: ecoNpm, configFile: fileNpmManifest, manifest: fileNpmManifest, parse: ParseNpmWorkspaces, readManifest: readNpmManifest},
		{id: ecoPnpm, configFile: filePnpmWorkspace, manifest: fileNpmManifest, parse: ParsePnpmWorkspaces, readManifest: readNpmManifest},
		{id: ecoCargo, configFile: fileCargoManifest, manifest: fileCargoManifest, parse: ParseCargoWorkspaces, readManifest: readCargoManifest},
		{id: ecoGo, configFile: fileGoWork, manifest: fileGoModManifest, parse: ParseGoWorkspaces, readManifest: readGoManifest},
		{id: ecoUv, configFile: filePyprojectToml, manifest: filePyprojectToml, parse: ParseUvWorkspaces, readManifest: readPyprojectManifest},
	}
}

// readNpmManifest reads a package.json and returns its "name" and the union of
// its "dependencies" and "devDependencies" keys, sorted.
func readNpmManifest(path string) (name string, deps []string, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var m struct {
		Name            string            `json:"name"`
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return "", nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	set := map[string]struct{}{}
	for k := range m.Dependencies {
		set[k] = struct{}{}
	}
	for k := range m.DevDependencies {
		set[k] = struct{}{}
	}
	return m.Name, sortedKeys(set), nil
}

// readCargoManifest reads a Cargo.toml and returns the [package] name and the
// keys of its [dependencies] table, sorted.
func readCargoManifest(path string) (name string, deps []string, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var m struct {
		Package struct {
			Name string `toml:"name"`
		} `toml:"package"`
		Dependencies map[string]any `toml:"dependencies"`
	}
	if err := toml.Unmarshal(data, &m); err != nil {
		return "", nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	set := map[string]struct{}{}
	for k := range m.Dependencies {
		set[k] = struct{}{}
	}
	return m.Package.Name, sortedKeys(set), nil
}

// readGoManifest reads a go.mod and returns the module path as the name and its
// require paths as dependencies.
func readGoManifest(path string) (name string, deps []string, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil, fmt.Errorf("reading %s: %w", path, err)
	}
	mf, err := modfile.Parse(path, data, nil)
	if err != nil {
		return "", nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if mf.Module != nil {
		name = mf.Module.Mod.Path
	}
	deps = make([]string, 0, len(mf.Require))
	for _, r := range mf.Require {
		if r == nil {
			continue
		}
		deps = append(deps, r.Mod.Path)
	}
	sort.Strings(deps)
	return name, deps, nil
}

// readPyprojectManifest reads a pyproject.toml and returns the [project] name
// and the package names parsed out of its [project] dependencies (PEP 508
// requirement strings, e.g. "requests>=2,<3" -> "requests"), sorted.
func readPyprojectManifest(path string) (name string, deps []string, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var m struct {
		Project struct {
			Name         string   `toml:"name"`
			Dependencies []string `toml:"dependencies"`
		} `toml:"project"`
	}
	if err := toml.Unmarshal(data, &m); err != nil {
		return "", nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	set := map[string]struct{}{}
	for _, req := range m.Project.Dependencies {
		if n := pep508Name(req); n != "" {
			set[n] = struct{}{}
		}
	}
	return m.Project.Name, sortedKeys(set), nil
}

// pep508Name extracts the distribution name from a PEP 508 requirement string by
// taking the leading run of name characters before any version specifier,
// extra, marker, or whitespace.
func pep508Name(req string) string {
	t := strings.TrimSpace(req)
	end := len(t)
	for i, r := range t {
		if !isPEP508NameRune(r) {
			end = i
			break
		}
	}
	return t[:end]
}

// isPEP508NameRune reports whether r is valid within a PEP 508 distribution
// name (letters, digits, and the separators "-", "_", ".").
func isPEP508NameRune(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		return true
	case r == '-' || r == '_' || r == '.':
		return true
	default:
		return false
	}
}

// sortedKeys returns the keys of set as a sorted slice.
func sortedKeys(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
