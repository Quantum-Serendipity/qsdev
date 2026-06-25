package security

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// osvPackage is a single (ecosystem, name, version) dependency coordinate in the
// shape OSV.dev's query API expects.
type osvPackage struct {
	Name      string
	Version   string
	Ecosystem string
}

// lockFile pairs a lock filename with the OSV ecosystem and parser that handle
// it. Auto-detection walks this table in order; the first present file wins.
type lockFile struct {
	name      string
	ecosystem string
	parse     func(path string) ([]osvPackage, error)
}

// knownLockFiles enumerates the lock files security_scan can extract pinned
// dependency versions from, paired with their OSV ecosystem name and parser.
func knownLockFiles() []lockFile {
	return []lockFile{
		{"go.sum", "Go", parseGoSum},
		{"package-lock.json", "npm", parseNPMLock},
		{"Cargo.lock", "crates.io", parseTOMLPackages},
		{"poetry.lock", "PyPI", parseTOMLPackages},
		{"uv.lock", "PyPI", parseTOMLPackages},
		{"requirements.txt", "PyPI", parseRequirementsTxt},
	}
}

// detectLockFile returns the first known lock file present in projectRoot, or a
// false ok when none exists. The returned lockFile carries the parser bound to
// the discovered file.
func detectLockFile(projectRoot string) (lockFile, string, bool) {
	for _, lf := range knownLockFiles() {
		p := filepath.Join(projectRoot, lf.name)
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return lf, p, true
		}
	}
	return lockFile{}, "", false
}

// lockFileForPath resolves the parser for an explicitly supplied manifest path
// by matching its base name against the known lock files.
func lockFileForPath(path string) (lockFile, bool) {
	base := filepath.Base(path)
	for _, lf := range knownLockFiles() {
		if lf.name == base {
			return lf, true
		}
	}
	return lockFile{}, false
}

// parseGoSum extracts module@version pairs from a go.sum file. Each module
// appears on both a "<mod> <ver> h1:..." and a "<mod> <ver>/go.mod h1:..." line;
// the "/go.mod" suffix is stripped and duplicates are collapsed. The leading "v"
// of the module version is dropped because OSV's Go ecosystem indexes versions
// without it.
func parseGoSum(path string) ([]osvPackage, error) {
	f, err := os.Open(path) //nolint:gosec // path is a project-root lock file, not user input
	if err != nil {
		return nil, err
	}
	defer f.Close()

	seen := make(map[string]bool)
	var pkgs []osvPackage
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 2 {
			continue
		}
		mod := fields[0]
		ver := strings.TrimSuffix(fields[1], "/go.mod")
		ver = strings.TrimPrefix(ver, "v")
		key := mod + "@" + ver
		if seen[key] || mod == "" || ver == "" {
			continue
		}
		seen[key] = true
		pkgs = append(pkgs, osvPackage{Name: mod, Version: ver, Ecosystem: "Go"})
	}
	return pkgs, sc.Err()
}

// npmLock models the subset of package-lock.json (lockfileVersion 2/3) we read.
type npmLock struct {
	Packages map[string]struct {
		Version string `json:"version"`
	} `json:"packages"`
	Dependencies map[string]struct {
		Version string `json:"version"`
	} `json:"dependencies"`
}

// parseNPMLock extracts name@version pairs from a package-lock.json. It reads the
// lockfileVersion 2/3 "packages" map (keyed by node_modules path) and falls back
// to the lockfileVersion 1 "dependencies" map.
func parseNPMLock(path string) ([]osvPackage, error) {
	data, err := os.ReadFile(path) //nolint:gosec // project-root lock file
	if err != nil {
		return nil, err
	}
	var lock npmLock
	if err := json.Unmarshal(data, &lock); err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var pkgs []osvPackage
	add := func(name, ver string) {
		if name == "" || ver == "" {
			return
		}
		key := name + "@" + ver
		if seen[key] {
			return
		}
		seen[key] = true
		pkgs = append(pkgs, osvPackage{Name: name, Version: ver, Ecosystem: "npm"})
	}
	for k, v := range lock.Packages {
		if k == "" {
			continue // the root package has an empty key
		}
		idx := strings.LastIndex(k, "node_modules/")
		name := k
		if idx >= 0 {
			name = k[idx+len("node_modules/"):]
		}
		add(name, v.Version)
	}
	for name, v := range lock.Dependencies {
		add(name, v.Version)
	}
	return pkgs, nil
}

// parseTOMLPackages extracts [[package]] name/version entries from a minimal TOML
// lock file (Cargo.lock, poetry.lock, uv.lock). It is a line-oriented parser
// scoped to the [[package]] table arrays, sufficient for the flat name/version
// fields these lock files use without taking a TOML dependency. The ecosystem is
// inferred from the file's base name.
func parseTOMLPackages(path string) ([]osvPackage, error) {
	f, err := os.Open(path) //nolint:gosec // project-root lock file
	if err != nil {
		return nil, err
	}
	defer f.Close()

	eco := "crates.io"
	if filepath.Base(path) != "Cargo.lock" {
		eco = "PyPI"
	}

	var pkgs []osvPackage
	var inPkg bool
	var name, ver string
	flush := func() {
		if inPkg && name != "" && ver != "" {
			pkgs = append(pkgs, osvPackage{Name: name, Version: ver, Ecosystem: eco})
		}
		name, ver = "", ""
	}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		switch {
		case line == "[[package]]":
			flush()
			inPkg = true
		case strings.HasPrefix(line, "[") && line != "[[package]]":
			flush()
			inPkg = false
		case inPkg && strings.HasPrefix(line, "name = "):
			name = trimTOMLValue(line[len("name = "):])
		case inPkg && strings.HasPrefix(line, "version = "):
			ver = trimTOMLValue(line[len("version = "):])
		}
	}
	flush()
	return pkgs, sc.Err()
}

// parseRequirementsTxt extracts pinned "name==version" entries from a pip
// requirements.txt, ignoring comments, blank lines, and unpinned or
// VCS/editable specifiers it cannot resolve to an exact version.
func parseRequirementsTxt(path string) ([]osvPackage, error) {
	f, err := os.Open(path) //nolint:gosec // project-root lock file
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var pkgs []osvPackage
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") {
			continue
		}
		if i := strings.Index(line, "#"); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		name, ver, ok := strings.Cut(line, "==")
		if !ok {
			continue
		}
		name = strings.TrimSpace(name)
		ver = strings.TrimSpace(ver)
		// Drop any trailing extras/markers, e.g. "pkg[extra]==1.0 ; python<3".
		if i := strings.IndexAny(name, "[ ;"); i >= 0 {
			name = name[:i]
		}
		if i := strings.IndexAny(ver, " ;"); i >= 0 {
			ver = ver[:i]
		}
		if name != "" && ver != "" {
			pkgs = append(pkgs, osvPackage{Name: name, Version: ver, Ecosystem: "PyPI"})
		}
	}
	return pkgs, sc.Err()
}

// trimTOMLValue strips surrounding quotes and whitespace from a TOML scalar.
func trimTOMLValue(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, `"'`)
	return s
}
