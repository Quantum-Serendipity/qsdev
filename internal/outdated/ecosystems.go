package outdated

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// ecosystemCommands lists each ecosystem's native outdated commands. Within an
// ecosystem, order is the fallback preference when neither a configured package
// manager nor a project marker file picks one (see selectCommand).
//
// OutdatedOnExit1 is set only where exit status 1 is the tool's documented
// "outdated packages found" signal, passing the flag that enables it where the
// tool needs one (cargo outdated --exit-code 1, composer outdated --strict).
// Any other non-zero exit is a failure of the check.
var ecosystemCommands = []EcosystemCommand{
	{Ecosystem: ecosystem.NameJavaScript, Binary: "npm", Args: []string{"outdated"}, OutdatedOnExit1: true, Markers: []string{"package-lock.json", "npm-shrinkwrap.json"}},
	{Ecosystem: ecosystem.NameJavaScript, Binary: "pnpm", Args: []string{"outdated"}, OutdatedOnExit1: true, Markers: []string{"pnpm-lock.yaml"}},
	{Ecosystem: ecosystem.NameJavaScript, Binary: "yarn", Args: []string{"outdated"}, OutdatedOnExit1: true, Markers: []string{"yarn.lock"}, Unsupported: yarnBerryUnsupported},
	{Ecosystem: ecosystem.NameJavaScript, Binary: "bun", Args: []string{"outdated"}, OutdatedOnExit1: false, Markers: []string{"bun.lock", "bun.lockb"}},
	{Ecosystem: ecosystem.NamePython, Binary: "pip", Args: []string{"list", "--outdated"}, OutdatedOnExit1: false},
	{Ecosystem: ecosystem.NamePython, Binary: "uv", Args: []string{"pip", "list", "--outdated"}, OutdatedOnExit1: false, Markers: []string{"uv.lock"}},
	{Ecosystem: ecosystem.NamePython, Binary: "poetry", Args: []string{"show", "--outdated"}, OutdatedOnExit1: false, Markers: []string{"poetry.lock"}},
	{Ecosystem: ecosystem.NameGo, Binary: "go", Args: []string{"list", "-m", "-u", "all"}, OutdatedOnExit1: false},
	{Ecosystem: ecosystem.NameRust, Binary: "cargo", Args: []string{"outdated", "--exit-code", "1"}, OutdatedOnExit1: true},
	{Ecosystem: ecosystem.NameDotnet, Binary: "dotnet", Args: []string{"list", "package", "--outdated"}, OutdatedOnExit1: false},
	{Ecosystem: ecosystem.NameRuby, Binary: "bundle", Args: []string{"outdated"}, OutdatedOnExit1: true},
	{Ecosystem: ecosystem.NamePHP, Binary: "composer", Args: []string{"outdated", "--direct", "--strict"}, OutdatedOnExit1: true},
	{Ecosystem: ecosystem.NameElixir, Binary: "mix", Args: []string{"hex.outdated"}, OutdatedOnExit1: false},
	{Ecosystem: ecosystem.NameJava, Binary: "mvn", Args: []string{"versions:display-dependency-updates"}, OutdatedOnExit1: false, Markers: []string{"pom.xml"}},
	{Ecosystem: ecosystem.NameJava, Binary: "gradle", Args: []string{"dependencyUpdates"}, OutdatedOnExit1: false, Markers: []string{"build.gradle", "build.gradle.kts", "settings.gradle", "settings.gradle.kts"}},
}

// CommandsForEcosystem returns matching commands for the given ecosystem.
// For ecosystems with multiple package managers (e.g., javascript has
// npm/pnpm/yarn/bun), it returns all of them in fallback order; RunOutdated
// picks one with selectCommand.
func CommandsForEcosystem(eco string) []EcosystemCommand {
	var result []EcosystemCommand
	for _, cmd := range ecosystemCommands {
		if cmd.Ecosystem == eco {
			result = append(result, cmd)
		}
	}
	return result
}

// SupportedEcosystems returns the list of ecosystem names that have outdated commands.
func SupportedEcosystems() []string {
	seen := make(map[string]bool)
	var result []string
	for _, cmd := range ecosystemCommands {
		if !seen[cmd.Ecosystem] {
			seen[cmd.Ecosystem] = true
			result = append(result, cmd.Ecosystem)
		}
	}
	return result
}

// yarnBerryUnsupported reports why `yarn outdated` cannot check a Yarn 2+
// ("Berry") project: Berry removed the outdated command, so it exits 1 with
// "command not found", which would otherwise read as "outdated packages found".
// Berry is recognized by its .yarnrc.yml or a packageManager field of yarn@2+.
func yarnBerryUnsupported(projectRoot string) string {
	const reason = "Yarn 2+ has no outdated command (use `yarn upgrade-interactive`)"
	if _, err := os.Stat(filepath.Join(projectRoot, ".yarnrc.yml")); err == nil {
		return reason
	}
	data, err := os.ReadFile(filepath.Join(projectRoot, "package.json")) //nolint:gosec // project-root manifest
	if err != nil {
		return ""
	}
	var pkg struct {
		PackageManager string `json:"packageManager"`
	}
	if json.Unmarshal(data, &pkg) != nil {
		return ""
	}
	ver, ok := strings.CutPrefix(pkg.PackageManager, "yarn@")
	if !ok {
		return ""
	}
	major, _, _ := strings.Cut(ver, ".")
	if n, err := strconv.Atoi(major); err == nil && n >= 2 {
		return reason
	}
	return ""
}
