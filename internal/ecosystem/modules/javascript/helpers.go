// Package javascript implements the JavaScript/TypeScript ecosystem module
// for gdev-secure-devenv-bootstrap. It detects Node.js projects, determines
// the package manager in use, and generates security-hardened configurations.
package javascript

import (
	"bufio"
	"encoding/json"
	"os"
	"strconv"
	"strings"
)

// fileExists reports whether a file at the given path exists and is not a directory.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// readFirstLine reads the first non-empty trimmed line from a file.
// Returns an empty string if the file cannot be read or is empty.
func readFirstLine(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close() //nolint:errcheck // best-effort read

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			return line
		}
	}
	return ""
}

// nodeVersionFromPackageJSON extracts the Node.js version constraint from
// the "engines.node" field in package.json at the given directory path.
// Returns an empty string if the field is absent or unparseable.
func nodeVersionFromPackageJSON(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	var pkg struct {
		Engines struct {
			Node string `json:"node"`
		} `json:"engines"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return ""
	}
	return pkg.Engines.Node
}

// extractMajorVersion strips common version prefixes (v, >=, ^, ~, =)
// and returns the first numeric segment as an integer.
// Returns 0 if parsing fails.
func extractMajorVersion(version string) int {
	if version == "" {
		return 0
	}

	// Strip common prefixes.
	v := version
	for _, prefix := range []string{">=", "<=", "^", "~", "=", "v", ">", "<"} {
		v = strings.TrimPrefix(v, prefix)
	}
	v = strings.TrimSpace(v)

	// Take the first numeric segment (before '.', '-', or ' ').
	for i, ch := range v {
		if ch == '.' || ch == '-' || ch == ' ' {
			v = v[:i]
			break
		}
	}

	n, err := strconv.Atoi(v)
	if err != nil {
		return 0
	}
	return n
}

// nodeNixPackage maps a major Node.js version to the corresponding
// nixpkgs attribute. Defaults to pkgs.nodejs_22 for unknown versions.
func nodeNixPackage(majorVersion int) string {
	switch majorVersion {
	case 18:
		return "pkgs.nodejs_18"
	case 20:
		return "pkgs.nodejs_20"
	case 22:
		return "pkgs.nodejs_22"
	case 24:
		return "pkgs.nodejs_24"
	default:
		return "pkgs.nodejs_22"
	}
}
