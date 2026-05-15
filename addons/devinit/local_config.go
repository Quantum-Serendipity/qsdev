package devinit

import (
	"fmt"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// GenerateLocalConfigTemplate returns a commented YAML template for
// .qsdev.local.yaml. The template includes sections relevant to the
// answers (e.g., language version overrides, Claude Code settings).
// All keys are commented out so the file has no active configuration
// by default.
func GenerateLocalConfigTemplate(answers types.WizardAnswers, detected types.DetectedProject) []byte {
	var b strings.Builder

	b.WriteString("# .qsdev.local.yaml — Local developer overrides (gitignored)\n")
	b.WriteString("# Uncomment and modify lines below to customize your local environment.\n")
	b.WriteString("# These settings override .qsdev.yaml but cannot lower security settings.\n")
	b.WriteString("#\n")

	// Extra packages section.
	b.WriteString("# extra_packages:\n")
	b.WriteString("#   - neovim\n")
	b.WriteString("#   - lazygit\n")
	b.WriteString("#   - ripgrep\n")

	// Language version overrides.
	if len(answers.Languages) > 0 {
		b.WriteString("#\n")
		b.WriteString("# languages:\n")
		for _, lang := range answers.Languages {
			exampleVersion := exampleVersionForLanguage(lang.Name, lang.Version)
			if exampleVersion != "" {
				fmt.Fprintf(&b, "#   - name: %s\n", lang.Name)
				fmt.Fprintf(&b, "#     version: \"%s\"\n", exampleVersion)
			}
		}
	}

	// Claude Code section.
	if answers.ClaudeCode {
		b.WriteString("#\n")
		b.WriteString("# claude_code:\n")
		b.WriteString("#   permission_level: permissive\n")
	}

	// Tools section.
	b.WriteString("#\n")
	b.WriteString("# tools:\n")
	b.WriteString("#   enabled:\n")
	b.WriteString("#     - changelog\n")

	return []byte(b.String())
}

// exampleVersionForLanguage returns an example version string for the
// given language, based on the current version or a sensible default.
func exampleVersionForLanguage(name, currentVersion string) string {
	if currentVersion != "" {
		return currentVersion
	}
	switch name {
	case "go":
		return "1.24"
	case "javascript":
		return "22"
	case "python":
		return "3.12"
	case "rust":
		return "stable"
	case "java":
		return "21"
	default:
		return ""
	}
}
