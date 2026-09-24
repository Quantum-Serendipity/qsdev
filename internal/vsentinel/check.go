package vsentinel

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
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
	mf, err := parseGoModFile(path)
	if err != nil {
		return nil, err
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

// parseGoModFile reads and parses a go.mod file.
//
// ParseLax covers every require shape — single-line, block, commented, and
// quoted paths — uniformly, and tolerates directives this parser ignores (e.g.
// a newer toolchain line). It preserves the declared version text verbatim
// (keeping the "v" prefix), so declared-vs-locked comparison against go.sum
// stays apples-to-apples. Both direct and indirect requires are reported,
// matching the manifest's full require set. ParseLax does not populate Replace;
// see goModReplacements.
func parseGoModFile(path string) (*modfile.File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading go.mod: %w", err)
	}
	mf, err := modfile.ParseLax(path, data, nil)
	if err != nil {
		return nil, fmt.Errorf("parsing go.mod: %w", err)
	}
	return mf, nil
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

// cargoManifest models the dependency tables of a Cargo.toml: the regular,
// dev and build tables, their target-specific variants
// ([target.'cfg(unix)'.dependencies]), and a workspace root's
// [workspace.dependencies].
type cargoManifest struct {
	cargoDepTables
	Target    map[string]cargoDepTables `toml:"target"`
	Workspace struct {
		Dependencies map[string]any `toml:"dependencies"`
	} `toml:"workspace"`
}

type cargoDepTables struct {
	Dependencies      map[string]any `toml:"dependencies"`
	DevDependencies   map[string]any `toml:"dev-dependencies"`
	BuildDependencies map[string]any `toml:"build-dependencies"`
}

func (t cargoDepTables) all() []map[string]any {
	return []map[string]any{t.Dependencies, t.DevDependencies, t.BuildDependencies}
}

// cargoDep is one dependency declared in a Cargo.toml.
type cargoDep struct {
	Name    string // key in the manifest
	Package string // crate name recorded in Cargo.lock (differs when renamed via package = "...")
	Version string // version requirement; "" when the spec states none
	Source  string // "path", "git" or "workspace" for a non-registry or inherited spec
}

// parseCargoDeps decodes a Cargo.toml and returns every declared dependency, in
// a deterministic order. A spec is either a version string or a table whose
// "version" key holds the requirement; path, git and workspace-inherited specs
// without a version are returned with an empty Version and their Source set.
func parseCargoDeps(path string) ([]cargoDep, error) {
	var m cargoManifest
	if _, err := toml.DecodeFile(path, &m); err != nil {
		return nil, fmt.Errorf("parsing Cargo.toml: %w", err)
	}

	tables := append(m.all(), m.Workspace.Dependencies)
	targets := make([]string, 0, len(m.Target))
	for target := range m.Target {
		targets = append(targets, target)
	}
	sort.Strings(targets)
	for _, target := range targets {
		tables = append(tables, m.Target[target].all()...)
	}

	seen := make(map[cargoDep]bool)
	var deps []cargoDep
	for _, tbl := range tables {
		names := make([]string, 0, len(tbl))
		for name := range tbl {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			d := cargoDepFromSpec(name, tbl[name])
			if !seen[d] {
				seen[d] = true
				deps = append(deps, d)
			}
		}
	}
	return deps, nil
}

// cargoDepFromSpec interprets one dependency spec: `name = "1.0"` or
// `name = { version = "1.0", package = "real-name", path/git/workspace = ... }`.
func cargoDepFromSpec(name string, spec any) cargoDep {
	d := cargoDep{Name: name, Package: name}
	switch s := spec.(type) {
	case string:
		d.Version = s
	case map[string]any:
		if v, ok := s["version"].(string); ok {
			d.Version = v
		}
		if pkg, ok := s["package"].(string); ok && pkg != "" {
			d.Package = pkg
		}
		switch {
		case s["workspace"] == true:
			d.Source = "workspace"
		case s["path"] != nil:
			d.Source = "path"
		case s["git"] != nil:
			d.Source = "git"
		}
	}
	return d
}

func parseCargoToml(path string) ([]DepStatus, error) {
	cargoDeps, err := parseCargoDeps(path)
	if err != nil {
		return nil, err
	}
	deps := make([]DepStatus, 0, len(cargoDeps))
	for _, d := range cargoDeps {
		ver := d.Version
		if ver == "" && d.Source != "" {
			ver = "(" + d.Source + ")"
		}
		deps = append(deps, DepStatus{Name: d.Name, DeclaredVersion: ver})
	}
	return deps, nil
}

// pyproject models the dependency declarations of a pyproject.toml: PEP 621
// [project] dependencies and optional-dependencies, PEP 735
// [dependency-groups], and Poetry's [tool.poetry] dependency tables.
type pyproject struct {
	Project struct {
		Dependencies         []string            `toml:"dependencies"`
		OptionalDependencies map[string][]string `toml:"optional-dependencies"`
	} `toml:"project"`
	// Group entries are PEP 508 strings or {include-group = "..."} tables.
	DependencyGroups map[string][]any `toml:"dependency-groups"`
	Tool             struct {
		Poetry struct {
			Dependencies    map[string]any `toml:"dependencies"`
			DevDependencies map[string]any `toml:"dev-dependencies"`
			Group           map[string]struct {
				Dependencies map[string]any `toml:"dependencies"`
			} `toml:"group"`
		} `toml:"poetry"`
	} `toml:"tool"`
}

func parsePyprojectToml(path string) ([]DepStatus, error) {
	var pp pyproject
	if _, err := toml.DecodeFile(path, &pp); err != nil {
		return nil, fmt.Errorf("parsing pyproject.toml: %w", err)
	}

	var deps []DepStatus
	addPEP508 := func(spec string) {
		if name, ver := splitPythonDep(spec); name != "" {
			deps = append(deps, DepStatus{Name: name, DeclaredVersion: ver})
		}
	}
	for _, spec := range pp.Project.Dependencies {
		addPEP508(spec)
	}
	for _, extra := range sortedKeys(pp.Project.OptionalDependencies) {
		for _, spec := range pp.Project.OptionalDependencies[extra] {
			addPEP508(spec)
		}
	}
	for _, group := range sortedKeys(pp.DependencyGroups) {
		for _, entry := range pp.DependencyGroups[group] {
			if spec, ok := entry.(string); ok {
				addPEP508(spec)
			}
		}
	}

	poetry := pp.Tool.Poetry
	poetryTables := []map[string]any{poetry.Dependencies, poetry.DevDependencies}
	for _, group := range sortedKeys(poetry.Group) {
		poetryTables = append(poetryTables, poetry.Group[group].Dependencies)
	}
	for _, tbl := range poetryTables {
		for _, name := range sortedKeys(tbl) {
			if strings.EqualFold(name, "python") {
				continue // the interpreter constraint, not a package
			}
			deps = append(deps, DepStatus{Name: name, DeclaredVersion: poetryVersion(tbl[name])})
		}
	}

	return deps, nil
}

// poetryVersion extracts the version constraint from a Poetry dependency spec,
// which is either a constraint string or a table with a "version" key. Specs
// with no single constraint (path/git tables, multiple-constraint lists) yield "".
func poetryVersion(spec any) string {
	switch s := spec.(type) {
	case string:
		return s
	case map[string]any:
		v, _ := s["version"].(string)
		return v
	}
	return ""
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// pythonVersionOps are the PEP 440 comparison operators a PEP 508 version
// specifier may start with.
var pythonVersionOps = []string{"===", "==", "!=", "~=", ">=", "<=", ">", "<"}

// splitPythonDep splits a PEP 508 requirement into the distribution name and
// its version specifier. A trailing comment (" # ...") and environment marker
// ("; python_version < '3.12'") are dropped, as are extras ("pkg[extra]"). The
// specifier starts at the earliest operator, so a multi-clause specifier such
// as "pkg<2,>=1" stays whole. A direct reference ("pkg @ https://...") is
// returned with the reference as its version.
func splitPythonDep(s string) (string, string) {
	s = stripRequirementComment(s)
	if i := strings.IndexByte(s, ';'); i >= 0 {
		s = s[:i]
	}
	s = strings.TrimSpace(s)

	if i := strings.IndexByte(s, '@'); i >= 0 {
		return pythonDistName(s[:i]), strings.TrimSpace(s[i:])
	}

	idx := -1
	for _, op := range pythonVersionOps {
		if i := strings.Index(s, op); i >= 0 && (idx < 0 || i < idx) {
			idx = i
		}
	}
	if idx < 0 {
		return pythonDistName(s), ""
	}
	// A legacy parenthesized specifier ("pkg (>=1.0)") keeps its parentheses
	// around the operators; trim them.
	return pythonDistName(s[:idx]), strings.Trim(strings.TrimSpace(s[idx:]), "() ")
}

// stripRequirementComment removes a pip-style comment: a "#" at the start of
// the line or preceded by whitespace. A "#" inside a URL fragment is kept.
func stripRequirementComment(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] == '#' && (i == 0 || s[i-1] == ' ' || s[i-1] == '\t') {
			return s[:i]
		}
	}
	return s
}

// pythonDistName returns the distribution name of a requirement's leading
// part, dropping extras ("[...]"), a legacy "(" specifier opener and spaces.
func pythonDistName(s string) string {
	if i := strings.IndexAny(s, "[( \t"); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
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
