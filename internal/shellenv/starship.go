package shellenv

import (
	"fmt"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// GenerateStarshipToml produces a .starship.toml configuration file with
// custom qsdev prompt segments showing project name, security profile, and
// active tool count.
func GenerateStarshipToml(answers types.WizardAnswers) (*types.GeneratedFile, error) {
	b := branding.Get()
	prefix := b.EnvPrefix
	app := b.AppName
	var s strings.Builder

	s.WriteString(fmt.Sprintf("[custom.%s]\n", app))
	s.WriteString(fmt.Sprintf("command = \"echo $%sPROJECT_NAME\"\n", prefix))
	s.WriteString(fmt.Sprintf("when = 'test -n \"$%sPROJECT_NAME\"'\n", prefix))
	s.WriteString("format = \"[$output]($style) \"\n")
	s.WriteString("style = \"bold cyan\"\n")
	s.WriteString(fmt.Sprintf("description = \"Active %s project\"\n", app))
	s.WriteString("\n")

	s.WriteString(fmt.Sprintf("[custom.%s_security]\n", app))
	s.WriteString("command = '''\n")
	s.WriteString(fmt.Sprintf("  case \"$%sSECURITY_PROFILE\" in\n", prefix))
	s.WriteString("    enhanced) echo \"\xf0\x9f\x94\x92\" ;;\n")
	s.WriteString("    strict)   echo \"\xf0\x9f\x94\x90\" ;;\n")
	s.WriteString("    *)        echo \"\xf0\x9f\x9b\xa1\" ;;\n")
	s.WriteString("  esac\n")
	s.WriteString("'''\n")
	s.WriteString(fmt.Sprintf("when = 'test -n \"$%sSECURITY_PROFILE\"'\n", prefix))
	s.WriteString("format = \"[$output]($style) \"\n")
	s.WriteString("style = \"green\"\n")
	s.WriteString(fmt.Sprintf("description = \"%s security profile\"\n", app))
	s.WriteString("\n")

	s.WriteString(fmt.Sprintf("[custom.%s_tools]\n", app))
	s.WriteString(fmt.Sprintf("command = \"echo ${%sTOOL_COUNT:-0}\"\n", prefix))
	s.WriteString(fmt.Sprintf("when = 'test -n \"$%sPROJECT_NAME\"'\n", prefix))
	s.WriteString("format = \"[$output tools]($style) \"\n")
	s.WriteString("style = \"dimmed white\"\n")
	s.WriteString("description = \"Active tool count\"\n")

	return &types.GeneratedFile{
		Path:     ".starship.toml",
		Content:  []byte(s.String()),
		Mode:     0o644,
		Strategy: types.Overwrite,
	}, nil
}
