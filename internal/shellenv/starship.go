package shellenv

import (
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// GenerateStarshipToml produces a .starship.toml configuration file with
// custom qsdev prompt segments showing project name, security profile, and
// active tool count.
func GenerateStarshipToml(answers types.WizardAnswers) (*types.GeneratedFile, error) {
	var b strings.Builder

	b.WriteString("[custom.qsdev]\n")
	b.WriteString("command = \"echo $QSDEV_PROJECT_NAME\"\n")
	b.WriteString("when = 'test -n \"$QSDEV_PROJECT_NAME\"'\n")
	b.WriteString("format = \"[$output]($style) \"\n")
	b.WriteString("style = \"bold cyan\"\n")
	b.WriteString("description = \"Active qsdev project\"\n")
	b.WriteString("\n")

	b.WriteString("[custom.qsdev_security]\n")
	b.WriteString("command = '''\n")
	b.WriteString("  case \"$QSDEV_SECURITY_PROFILE\" in\n")
	b.WriteString("    enhanced) echo \"\xf0\x9f\x94\x92\" ;;\n")
	b.WriteString("    strict)   echo \"\xf0\x9f\x94\x90\" ;;\n")
	b.WriteString("    *)        echo \"\xf0\x9f\x9b\xa1\" ;;\n")
	b.WriteString("  esac\n")
	b.WriteString("'''\n")
	b.WriteString("when = 'test -n \"$QSDEV_SECURITY_PROFILE\"'\n")
	b.WriteString("format = \"[$output]($style) \"\n")
	b.WriteString("style = \"green\"\n")
	b.WriteString("description = \"qsdev security profile\"\n")
	b.WriteString("\n")

	b.WriteString("[custom.qsdev_tools]\n")
	b.WriteString("command = \"echo ${QSDEV_TOOL_COUNT:-0}\"\n")
	b.WriteString("when = 'test -n \"$QSDEV_PROJECT_NAME\"'\n")
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
