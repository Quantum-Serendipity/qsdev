package devinit

import (
	"context"
	"fmt"
	"io"
	"slices"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/container"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// gatewayCompose is the Gateway container config (Unit 32.10) for one
// `qsdev update`: the docker-compose fragment for frameworks that lack native
// hook enforcement, planned and recorded like every other generated file.
type gatewayCompose struct {
	// file is the compose fragment to generate; nil when no framework in the
	// project needs the gateway.
	file *types.GeneratedFile
	// art holds the generator's output for the user-facing hints; nil when
	// generation was skipped or failed.
	art *container.Artifacts
	// hold is true when generation was skipped (--skip-container) or failed:
	// a previously generated fragment is then neither regenerated nor treated
	// as an orphan, so it stays on disk and tracked as it is.
	hold bool
}

// planGatewayCompose generates the Gateway docker-compose fragment when a
// framework present in the project lacks native hook enforcement. The
// fragment is returned as a types.GeneratedFile so update plans, merges and
// records it like any other generated file: user edits are kept by a
// three-way merge, drift checks and teardown see it in state, and it is
// removed as an orphan once no framework needs it. Generation is opt-out via
// --skip-container and never fails the update: an error is reported on w as
// a warning.
func planGatewayCompose(w io.Writer, projectRoot string, answers types.WizardAnswers, opts UpdateOptions) gatewayCompose {
	if opts.SkipContainer {
		return gatewayCompose{hold: true}
	}

	art, err := container.ContainerConfigGenerator{}.Generate(container.GenerateOptions{
		// No ProjectRoot: the fragment is committed with the project, so it
		// mounts "." (compose resolves it relative to the file) rather than
		// this checkout's absolute path.
		Frameworks: detectGatewayProfiles(projectRoot, answers),
	})
	if err != nil {
		fmt.Fprintf(w, "Warning: container config generation skipped: %v\n", err)
		return gatewayCompose{hold: true}
	}
	if !art.NeedsGateway {
		return gatewayCompose{art: art}
	}
	return gatewayCompose{
		art: art,
		file: &types.GeneratedFile{
			Path:     art.ComposeFileName,
			Content:  []byte(art.ComposeYAML),
			Mode:     fileutil.ModeReadWrite,
			Strategy: types.ThreeWayMerge,
		},
	}
}

// keepHeld drops the orphan plan for a held compose fragment, so a skipped or
// failed generation never removes or untracks the file generated earlier.
func (g gatewayCompose) keepHeld(orphans []FileUpdatePlan) []FileUpdatePlan {
	if !g.hold {
		return orphans
	}
	return slices.DeleteFunc(orphans, func(fp FileUpdatePlan) bool {
		return fp.Path == container.ComposeFileName
	})
}

// printHints tells the user which frameworks need the gateway and the
// .mcp.json entry to add for it. It prints nothing when no framework needs
// the gateway, so all-native projects see no gateway output.
func (g gatewayCompose) printHints(w io.Writer) {
	if g.file == nil {
		return
	}
	fmt.Fprintf(w, "\nGateway container config: %s (frameworks without native hooks: %s)\n",
		g.file.Path, joinFrameworkIDs(g.art.GatewayFrameworks))
	fmt.Fprintf(w, "  add the gateway to your framework .mcp.json:\n%s\n", g.art.MCPJSON)
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
