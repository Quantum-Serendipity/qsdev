package detect

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var goVersionRe = regexp.MustCompile(`^go\s+(\d+\.\d+(?:\.\d+)?)`)

// detectGo checks for a Go module and extracts the Go version from go.mod.
func detectGo(root string) (detected bool, version string) {
	modPath := filepath.Join(root, "go.mod")
	if !fileExists(modPath) {
		return false, ""
	}

	f, err := os.Open(modPath)
	if err != nil {
		return true, ""
	}
	defer f.Close() //nolint:errcheck // best-effort read

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if m := goVersionRe.FindStringSubmatch(scanner.Text()); m != nil {
			return true, m[1]
		}
	}
	return true, ""
}

// detectNode checks for a Node.js project and determines the package manager
// and Node version.
func detectNode(root string) (detected bool, version string, packageManager string) {
	if !fileExists(root, "package.json") {
		return false, "", ""
	}

	// Determine package manager from lockfiles (order matters: most specific first).
	switch {
	case fileExists(root, "pnpm-lock.yaml"):
		packageManager = "pnpm"
	case fileExists(root, "yarn.lock"):
		packageManager = "yarn"
	case fileExists(root, "bun.lock") || fileExists(root, "bun.lockb"):
		packageManager = "bun"
	case fileExists(root, "package-lock.json"):
		packageManager = "npm"
	default:
		packageManager = "npm"
	}

	// Try .nvmrc first for version.
	if v := readFirstLine(filepath.Join(root, ".nvmrc")); v != "" {
		version = strings.TrimPrefix(v, "v")
		return true, version, packageManager
	}

	// Fall back to package.json engines.node.
	version = nodeVersionFromPackageJSON(filepath.Join(root, "package.json"))
	return true, version, packageManager
}

// nodeVersionFromPackageJSON extracts the engines.node field from package.json.
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

// detectPython checks for a Python project and determines the package manager
// and Python version.
func detectPython(root string) (detected bool, version string, pm string) {
	if !fileExists(root, "pyproject.toml") {
		return false, "", ""
	}

	// Determine package manager from lockfiles.
	switch {
	case fileExists(root, "uv.lock"):
		pm = "uv"
	case fileExists(root, "poetry.lock"):
		pm = "poetry"
	case fileExists(root, "requirements.txt"):
		pm = "pip"
	default:
		pm = "pip"
	}

	// Parse version from .python-version.
	version = readFirstLine(filepath.Join(root, ".python-version"))

	return true, version, pm
}

// detectRust checks for a Rust/Cargo project.
func detectRust(root string) bool {
	return fileExists(root, "Cargo.toml")
}

// detectJava checks for Maven and Gradle build files.
func detectJava(root string) (hasMaven, hasGradle bool) {
	hasMaven = fileExists(root, "pom.xml")
	hasGradle = fileExists(root, "build.gradle") || fileExists(root, "build.gradle.kts")
	return hasMaven, hasGradle
}

// detectDotNet checks for .NET project files (*.csproj or *.sln).
func detectDotNet(root string) bool {
	if matches, _ := filepath.Glob(filepath.Join(root, "*.csproj")); len(matches) > 0 {
		return true
	}
	if matches, _ := filepath.Glob(filepath.Join(root, "*.sln")); len(matches) > 0 {
		return true
	}
	return false
}

// detectDocker checks for Docker-related files.
func detectDocker(root string) bool {
	return fileExists(root, "Dockerfile") ||
		fileExists(root, "docker-compose.yml") ||
		fileExists(root, "docker-compose.yaml")
}

// detectTerraform checks for Terraform configuration files (*.tf).
func detectTerraform(root string) bool {
	matches, _ := filepath.Glob(filepath.Join(root, "*.tf"))
	return len(matches) > 0
}
