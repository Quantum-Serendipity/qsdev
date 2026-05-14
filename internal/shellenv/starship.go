package shellenv

import (
	"strings"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

// GenerateStarshipToml produces a .starship.toml configuration file with
// custom gdev prompt segments showing project name, security profile, and
// active tool count.
func GenerateStarshipToml(answers types.WizardAnswers) (*types.GeneratedFile, error) {
	var b strings.Builder

	b.WriteString("[custom.gdev]\n")
	b.WriteString("command = \"echo $GDEV_PROJECT_NAME\"\n")
	b.WriteString("when = 'test -n \"$GDEV_PROJECT_NAME\"'\n")
	b.WriteString("format = \"[$output]($style) \"\n")
	b.WriteString("style = \"bold cyan\"\n")
	b.WriteString("description = \"Active gdev project\"\n")
	b.WriteString("\n")

	b.WriteString("[custom.gdev_security]\n")
	b.WriteString("command = '''\n")
	b.WriteString("  case \"$GDEV_SECURITY_PROFILE\" in\n")
	b.WriteString("    enhanced) echo \"\xf0\x9f\x94\x92\" ;;\n")
	b.WriteString("    strict)   echo \"\xf0\x9f\x94\x90\" ;;\n")
	b.WriteString("    *)        echo \"\xf0\x9f\x9b\xa1\" ;;\n")
	b.WriteString("  esac\n")
	b.WriteString("'''\n")
	b.WriteString("when = 'test -n \"$GDEV_SECURITY_PROFILE\"'\n")
	b.WriteString("format = \"[$output]($style) \"\n")
	b.WriteString("style = \"green\"\n")
	b.WriteString("description = \"gdev security profile\"\n")
	b.WriteString("\n")

	b.WriteString("[custom.gdev_tools]\n")
	b.WriteString("command = \"echo ${GDEV_TOOL_COUNT:-0}\"\n")
	b.WriteString("when = 'test -n \"$GDEV_PROJECT_NAME\"'\n")
	b.WriteString("format = \"[$output tools]($style) \"\n")
	b.WriteString("style = \"dimmed white\"\n")
	b.WriteString("description = \"Active tool count\"\n")

	return &types.GeneratedFile{
		Path:     ".starship.toml",
		Content:  []byte(b.String()),
		Mode:     0o644,
		Strategy: types.Overwrite,
	}, nil
}
