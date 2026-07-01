package devinit

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/container"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// gatewayComposePath is where the generated Gateway compose fragment is written,
// relative to the project root.
const gatewayComposePath = "docker-compose.gateway.yaml"

// maybeGenerateContainerConfig is the best-effort `qsdev update` integration for
// Unit 32.10. After the framework configs are regenerated it checks whether any
// framework present in the project lacks native hook enforcement and, if so,
// writes the Gateway docker-compose fragment and prints the .mcp.json entry to
// add. It is deliberately:
//
//   - opt-out via --skip-container,
//   - a silent no-op when no framework needs the gateway (so an all-native
//     project — e.g. Claude Code only — produces no output and the existing
//     update behavior is unchanged),
//   - non-fatal: any error is reported as a warning and never fails the update.
//
// It is the single live wiring point; the heavy lifting is the pure
// container.ContainerConfigGenerator, which is independently tested.
func maybeGenerateContainerConfig(cmd *cobra.Command, projectRoot string, answers types.WizardAnswers, opts UpdateOptions) {
	if opts.SkipContainer {
		return
	}

	profiles := detectGatewayProfiles(projectRoot, answers)
	art, err := container.ContainerConfigGenerator{}.Generate(container.GenerateOptions{
		ProjectRoot: projectRoot,
		Frameworks:  profiles,
	})
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Warning: container config generation skipped: %v\n", err)
		return
	}
	if !art.NeedsGateway {
		return // nothing to do; stay silent so all-native projects are unaffected
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "\nGateway container config (frameworks without native hooks: %s):\n",
		joinFrameworkIDs(art.GatewayFrameworks))

	if opts.DryRun {
		fmt.Fprintf(out, "  [dry-run] would write %s\n", gatewayComposePath)
		fmt.Fprintf(out, "  [dry-run] add the gateway to your framework .mcp.json:\n%s\n", art.MCPJSON)
		return
	}

	absPath := filepath.Join(projectRoot, gatewayComposePath)
	if err := fileutil.WriteFileAtomic(absPath, []byte(art.ComposeYAML), fileutil.ModeReadWrite); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Warning: writing %s: %v\n", gatewayComposePath, err)
		return
	}
	fmt.Fprintf(out, "  wrote %s\n", gatewayComposePath)
	fmt.Fprintf(out, "  add the gateway to your framework .mcp.json:\n%s\n", art.MCPJSON)
}

// detectGatewayProfiles derives the framework profiles present in the project.
// It probes the universal MCP server's adapter registry (populated by the
// blank-imported adapters in cmd/qsdev) using each adapter's project-marker
// Applies check, and augments it with the framework recorded in answers. Each
// detected framework is mapped to its enforcement tier via the authoritative
// container profile catalog — never a hardcoded skip-list. In unit tests, where
// the adapters are not imported, the registry is empty and only answers drive
// the result.
func detectGatewayProfiles(projectRoot string, answers types.WizardAnswers) []container.FrameworkProfile {
	seen := make(map[aiframework.FrameworkID]bool)
	var profiles []container.FrameworkProfile
	add := func(id aiframework.FrameworkID) {
		if seen[id] {
			return
		}
		seen[id] = true
		profiles = append(profiles, container.ProfileFor(id))
	}

	ctx := context.Background()
	for _, a := range spi.DefaultRegistry().All() {
		if a.Applies(ctx, projectRoot) {
			add(a.ID())
		}
	}
	if answers.ClaudeCode {
		add(aiframework.ClaudeCode)
	}
	return profiles
}

// joinFrameworkIDs renders a comma-separated list of framework ids.
func joinFrameworkIDs(ids []aiframework.FrameworkID) string {
	if len(ids) == 0 {
		return "none"
	}
	out := make([]byte, 0, len(ids)*8)
	for i, id := range ids {
		if i > 0 {
			out = append(out, ", "...)
		}
		out = append(out, id...)
	}
	return string(out)
}
