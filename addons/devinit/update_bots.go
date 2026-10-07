package devinit

import (
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"

	"github.com/Quantum-Serendipity/qsdev/internal/profile"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// projectUpdateBots returns the update tools whose config the project
// manages itself, from the bot configs detection found (configs,
// types.DetectedProject.UpdateBotConfigs), sorted. A config counts as the
// project's unless it is unmodified qsdev output: recorded by a local state
// file or the committed manifest (a fresh clone has only the latter) with the
// content on disk. A recorded config the project edited is the project's, so
// the first run after the edit already decides as every later one does.
//
// While qsdev manages a bot config (one is unmodified qsdev output), no
// config counts as the project's. qsdev's bot is then established and stays
// generated, so a profile switch is stable: the replaced bot's config, which
// the switch leaves on disk unrecorded, does not turn into a project bot on
// the next run and suppress the new profile's bot.
func projectUpdateBots(projectRoot string, configs []string) []string {
	if len(configs) == 0 {
		return nil
	}
	recorded := recordedOutputStates(projectRoot)
	var bots []string
	for _, rel := range configs {
		tool, ok := profile.UpdateToolForConfig(rel)
		if !ok {
			continue
		}
		content, err := os.ReadFile(filepath.Join(projectRoot, filepath.FromSlash(rel)))
		if err != nil {
			slog.Warn("reading dependency-update bot config", "path", rel, "error", err)
			continue
		}
		if state.IsRecordedOutput(recorded, rel, content) {
			return nil
		}
		bots = append(bots, string(tool))
	}
	slices.Sort(bots)
	return slices.Compact(bots)
}

// recordedOutputStates returns what qsdev recorded generating in the
// project: every local state file, each overlaid with the committed manifest
// (state.ExpectedState), or the manifest alone when there is no local state,
// as in a fresh clone.
func recordedOutputStates(projectRoot string) []types.GeneratedState {
	states, err := state.LoadProjectStates(projectRoot)
	if err != nil {
		slog.Warn("loading recorded state", "error", err)
	}
	manifest, err := state.LoadManifest(filepath.Join(projectRoot, state.ManifestFile()))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		slog.Warn("loading generated-file manifest", "error", err)
	}
	if len(states) == 0 {
		states = []types.GeneratedState{{}}
	}
	for i := range states {
		states[i] = state.ExpectedState(states[i], manifest)
	}
	return states
}
