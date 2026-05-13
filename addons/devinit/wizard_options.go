package devinit

import (
	"fmt"
	"strings"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

// LanguageOption describes a single language entry for the wizard's
// multi-select list, annotated with detection status.
type LanguageOption struct {
	Label    string // e.g. "Go (detected: go.mod)"
	Value    string // e.g. "go"
	Detected bool
}

// allLanguages defines the canonical order and display names.
var allLanguages = []struct {
	Value   string
	Display string
}{
	{"go", "Go"},
	{"javascript", "JavaScript/TypeScript"},
	{"python", "Python"},
	{"rust", "Rust"},
	{"java", "Java/Kotlin"},
	{"dotnet", "C#/.NET"},
	{"docker", "Docker"},
	{"terraform", "Terraform/OpenTofu"},
	{"php", "PHP"},
	{"ruby", "Ruby"},
	{"scala", "Scala"},
	{"cpp", "C/C++"},
	{"helm", "Helm"},
	{"ansible", "Ansible"},
	{"shell", "Bash/Shell"},
	{"elixir", "Elixir"},
	{"dart", "Dart/Flutter"},
	{"swift", "Swift"},
	{"haskell", "Haskell"},
	{"clojure", "Clojure"},
	{"bazel", "Bazel"},
	{"nix", "Nix"},
	{"perl", "Perl"},
	{"r", "R"},
	{"lua", "Lua"},
	{"zig", "Zig"},
	{"powershell", "PowerShell"},
}

// BuildLanguageOptions returns all supported language options for the wizard.
// Detected languages appear first in the list, each annotated with the
// marker file that triggered detection.
func BuildLanguageOptions(detected types.DetectedProject) []LanguageOption {
	detectedSet := make(map[string]bool)
	var first, rest []LanguageOption

	for _, lang := range allLanguages {
		ann := DetectionAnnotation(lang.Value, detected)
		isDetected := ann != ""
		if isDetected {
			detectedSet[lang.Value] = true
		}

		opt := LanguageOption{
			Label:    lang.Display,
			Value:    lang.Value,
			Detected: isDetected,
		}
		if isDetected {
			opt.Label = fmt.Sprintf("%s %s", lang.Display, ann)
		}

		if isDetected {
			first = append(first, opt)
		} else {
			rest = append(rest, opt)
		}
	}

	_ = detectedSet // used only in loop above
	return append(first, rest...)
}

// DetectionAnnotation returns a human-readable annotation string for a
// detected language, or the empty string if the language was not detected.
func DetectionAnnotation(name string, detected types.DetectedProject) string {
	switch name {
	case "go":
		if detected.HasGoMod {
			return "(detected: go.mod)"
		}
	case "javascript":
		if detected.HasPackageJSON {
			base := "package.json"
			if lf := lockfileForPM(detected.PackageManager); lf != "" {
				return fmt.Sprintf("(detected: %s + %s)", base, lf)
			}
			return fmt.Sprintf("(detected: %s)", base)
		}
	case "python":
		if detected.HasPyProject {
			return "(detected: pyproject.toml)"
		}
	case "rust":
		if detected.HasCargoToml {
			return "(detected: Cargo.toml)"
		}
	case "java":
		switch {
		case detected.HasPomXML && detected.HasBuildGradle:
			return "(detected: pom.xml + build.gradle)"
		case detected.HasPomXML:
			return "(detected: pom.xml)"
		case detected.HasBuildGradle:
			return "(detected: build.gradle)"
		}
	case "dotnet":
		if detected.HasCsproj {
			return "(detected: *.csproj)"
		}
	case "docker":
		if detected.HasDockerfile {
			return "(detected: Dockerfile)"
		}
	case "terraform":
		if detected.HasTerraform {
			return "(detected: *.tf)"
		}
	case "php":
		if detected.Ecosystems["php"] {
			return "(detected: composer.json)"
		}
	case "ruby":
		if detected.Ecosystems["ruby"] {
			return "(detected: Gemfile)"
		}
	case "scala":
		if detected.Ecosystems["scala"] {
			return "(detected: build.sbt)"
		}
	case "cpp":
		if detected.Ecosystems["cpp"] {
			return "(detected: C/C++ project)"
		}
	case "helm":
		if detected.Ecosystems["helm"] {
			return "(detected: Chart.yaml)"
		}
	case "ansible":
		if detected.Ecosystems["ansible"] {
			return "(detected: ansible.cfg)"
		}
	case "shell":
		if detected.Ecosystems["shell"] {
			return "(detected: *.sh files)"
		}
	case "elixir":
		if detected.Ecosystems["elixir"] {
			return "(detected: mix.exs)"
		}
	case "dart":
		if detected.Ecosystems["dart"] {
			return "(detected: pubspec.yaml)"
		}
	case "swift":
		if detected.Ecosystems["swift"] {
			return "(detected: Package.swift)"
		}
	case "haskell":
		if detected.Ecosystems["haskell"] {
			return "(detected: *.cabal)"
		}
	case "clojure":
		if detected.Ecosystems["clojure"] {
			return "(detected: deps.edn)"
		}
	case "bazel":
		if detected.Ecosystems["bazel"] {
			return "(detected: MODULE.bazel)"
		}
	case "nix":
		if detected.Ecosystems["nix"] {
			return "(detected: flake.nix)"
		}
	case "perl":
		if detected.Ecosystems["perl"] {
			return "(detected: cpanfile)"
		}
	case "r":
		if detected.Ecosystems["r"] {
			return "(detected: renv.lock)"
		}
	case "lua":
		if detected.Ecosystems["lua"] {
			return "(detected: *.rockspec)"
		}
	case "zig":
		if detected.Ecosystems["zig"] {
			return "(detected: build.zig)"
		}
	case "powershell":
		if detected.Ecosystems["powershell"] {
			return "(detected: *.ps1)"
		}
	}
	return ""
}

// lockfileForPM returns the lockfile name associated with a package manager.
func lockfileForPM(pm string) string {
	switch pm {
	case "npm":
		return "package-lock.json"
	case "yarn":
		return "yarn.lock"
	case "pnpm":
		return "pnpm-lock.yaml"
	case "bun":
		return "bun.lockb"
	default:
		return ""
	}
}

// PreSelectedLanguages returns the list of language values that should be
// pre-selected in the wizard based on detection results.
func PreSelectedLanguages(detected types.DetectedProject) []string {
	var selected []string
	for _, lang := range allLanguages {
		if DetectionAnnotation(lang.Value, detected) != "" {
			selected = append(selected, lang.Value)
		}
	}
	return selected
}

// QuickPathSummary produces a concise summary string describing what the
// quick-path setup will include, e.g. "Go 1.24 + devenv.sh + direnv + Claude Code".
func QuickPathSummary(defaults types.WizardAnswers) string {
	var parts []string

	for _, lang := range defaults.Languages {
		label := languageDisplayName(lang.Name)
		if lang.Version != "" {
			label += " " + lang.Version
		}
		parts = append(parts, label)
	}

	// Always include devenv.sh since that's the core of the tool
	parts = append(parts, "devenv.sh")

	if defaults.Direnv {
		parts = append(parts, "direnv")
	}

	if defaults.ClaudeCode {
		parts = append(parts, "Claude Code")
	}

	return strings.Join(parts, " + ")
}

// languageDisplayName returns a short display name for a language value.
func languageDisplayName(value string) string {
	for _, lang := range allLanguages {
		if lang.Value == value {
			return lang.Display
		}
	}
	return value
}
