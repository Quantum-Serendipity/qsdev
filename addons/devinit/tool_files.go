package devinit

import (
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/internal/tier"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// toolFilesSource is the fragment source name of the tool-files producer.
const toolFilesSource = "tools"

// Compile-time interface check.
var _ types.Generator = toolFilesGenerator{}

// toolFilesGenerator produces the exclusive files of every enabled tool that
// does not configure the coding agent (gitleaks, semgrep, changelog,
// commitlint, secretspec, starship, ...). These files belong to the project
// rather than to either addon, so they are generated in every generation
// scope and whether or not Claude Code is configured; the claudecode
// generator emits only agent-tool files.
type toolFilesGenerator struct {
	registry *toolreg.Registry
}

// Generate runs the GenerateFunc of each non-agent tool that toolFilesWanted
// selects, tagging every file with its tool as owner.
func (g toolFilesGenerator) Generate(answers types.WizardAnswers) ([]types.GeneratedFile, error) {
	t := tier.Resolve(answers.Tier, answers.PermissionLevel, answers.MCPServers)
	var files []types.GeneratedFile
	for _, tool := range g.registry.All() {
		if tool.IsAgentTool() || tool.GenerateFunc == nil || !toolFilesWanted(tool, answers, t) {
			continue
		}
		toolFiles, err := tool.GenerateFunc(answers)
		if err != nil {
			return nil, fmt.Errorf("generating %s files: %w", tool.Name, err)
		}
		for i := range toolFiles {
			toolFiles[i].Owner = tool.Name
		}
		files = append(files, toolFiles...)
	}
	return files, nil
}

// toolFilesWanted reports whether a tool's files are generated, from its
// catalog default policy and its EnabledTools entry:
//   - opt-in (and always-off) tools are enabled only by an explicit choice,
//     so their files follow that choice at every tier, exactly as
//     `qsdev enable` writes them;
//   - always-on and on-when-detected tools are tier defaults that are enabled
//     implicitly, so their files start at the standard tier (supply-chain-only
//     adds no tool configuration). An always-on tool without an entry is on;
//     one force-disabled (EnabledTools[name]=false) stays off.
func toolFilesWanted(tool *toolreg.Tool, answers types.WizardAnswers, t tier.Tier) bool {
	enabled, set := answers.EnabledTools[tool.Name]
	switch tool.Default {
	case toolreg.AlwaysOn:
		return t >= tier.Standard && (!set || enabled)
	case toolreg.OnWhenDetected:
		return t >= tier.Standard && enabled
	default:
		return enabled
	}
}
