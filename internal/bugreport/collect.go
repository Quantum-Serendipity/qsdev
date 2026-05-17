package bugreport

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/info"
	"github.com/Quantum-Serendipity/qsdev/internal/sysinfo"
	"github.com/Quantum-Serendipity/qsdev/internal/version"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// Environment holds auto-collected diagnostic information.
type Environment struct {
	QsdevVersion string
	Commit       string
	GoVersion    string
	OS           string
	Arch         string
	Family       string
	Shell        string
	HasNix       bool
	DevenvVer    string
	GhVer        string

	Ecosystems      []string
	ActiveToolCount int
	SecurityProfile string
	ConfigVersion   int
}

// CollectEnvironment gathers system and project information.
func CollectEnvironment(projectRoot string) Environment {
	bi := version.Info()
	osInfo := sysinfo.DetectOS()

	env := Environment{
		QsdevVersion: bi.Version,
		Commit:       bi.Commit,
		GoVersion:    bi.GoVersion,
		OS:           osInfo.OS,
		Arch:         osInfo.Arch,
		Family:       osInfo.Family,
		Shell:        osInfo.Shell,
		HasNix:       osInfo.HasNix,
		DevenvVer:    toolVersion("devenv", "version"),
		GhVer:        toolVersion("gh", "--version"),
	}

	if projectRoot != "" {
		if pi, err := info.CollectInfo(projectRoot); err == nil {
			env.Ecosystems = pi.Ecosystems
			env.ActiveToolCount = pi.ActiveToolCount
			env.SecurityProfile = pi.SecurityProfile
			env.ConfigVersion = pi.ConfigVersion
		}
	}

	return env
}

// FormatTable renders the environment as a markdown table.
func (e Environment) FormatTable() string {
	var b strings.Builder
	b.WriteString("| Field | Value |\n")
	b.WriteString("|-------|-------|\n")
	fmt.Fprintf(&b, "| %s version | %s (%s) |\n", branding.Get().AppName, e.QsdevVersion, e.Commit)
	fmt.Fprintf(&b, "| Go version | %s |\n", e.GoVersion)
	fmt.Fprintf(&b, "| OS | %s/%s (%s) |\n", e.OS, e.Arch, e.Family)
	fmt.Fprintf(&b, "| Shell | %s |\n", e.Shell)
	fmt.Fprintf(&b, "| Nix | %s |\n", boolStr(e.HasNix))
	if e.DevenvVer != "" {
		fmt.Fprintf(&b, "| devenv | %s |\n", e.DevenvVer)
	}
	if len(e.Ecosystems) > 0 {
		fmt.Fprintf(&b, "| Ecosystems | %s |\n", strings.Join(e.Ecosystems, ", "))
		fmt.Fprintf(&b, "| Active tools | %d |\n", e.ActiveToolCount)
		fmt.Fprintf(&b, "| Security profile | %s |\n", e.SecurityProfile)
	}
	return b.String()
}

func toolVersion(name string, arg string) string {
	out, err := exec.Command(name, arg).CombinedOutput()
	if err != nil {
		return ""
	}
	line := strings.Split(strings.TrimSpace(string(out)), "\n")[0]
	return strings.TrimSpace(line)
}

func boolStr(b bool) string {
	if b {
		return "installed"
	}
	return "not found"
}
