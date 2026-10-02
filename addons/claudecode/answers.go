package claudecode

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/Quantum-Serendipity/qsdev/internal/answers"
	qsdevconfig "github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// answersPath returns the full path to the answers file that claude
// subcommands read and write: the project's primary (devinit) answers file,
// the single source of truth for wizard answers across all addons.
func answersPath(projectRoot string) string {
	return answers.PrimaryPath(projectRoot)
}

// legacyAnswersPath returns the path of the per-addon answers copy that older
// releases kept in .claude/. It went stale whenever `qsdev enable/disable` or
// another addon updated the primary file, so it is never read; saveAnswers
// deletes it.
func legacyAnswersPath(projectRoot string) string {
	return filepath.Join(projectRoot, filepath.FromSlash(answers.LegacyClaudeCopyFile()))
}

// saveAnswers persists the wizard answers to the primary answers file and
// removes the obsolete per-addon copy.
func saveAnswers(projectRoot string, a types.WizardAnswers) error {
	if err := answers.SavePrimary(projectRoot, a); err != nil {
		return fmt.Errorf("saving primary answers: %w", err)
	}
	if err := os.Remove(legacyAnswersPath(projectRoot)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		// The legacy copy is never read, so failing to delete it is harmless.
		slog.Warn("removing legacy claude answers file", "error", err)
	}
	return nil
}

// SaveAnswers is an exported wrapper around saveAnswers, allowing other
// packages (e.g. devinit) to persist answers for the Claude Code addon.
func SaveAnswers(projectRoot string, a types.WizardAnswers) error {
	return saveAnswers(projectRoot, a)
}

// loadAnswers reads the saved wizard answers from the primary answers file. It
// returns an error telling the user to run init first if the file does not
// exist.
//
// The loaded answers are normalised (see normalizeAnswers), so what claude
// subcommands report and persist agrees with the settings they generate.
func loadAnswers(projectRoot string) (types.WizardAnswers, error) {
	a, err := loadSavedAnswers(projectRoot)
	if err != nil {
		return types.WizardAnswers{}, err
	}
	normalizeAnswers(projectRoot, &a)
	return a, nil
}

// loadSavedAnswers reads the primary answers file as saved, without
// normalising it.
func loadSavedAnswers(projectRoot string) (types.WizardAnswers, error) {
	return answers.LoadFromDir(projectRoot, answers.PrimaryDir(), answers.PrimaryFilename(), "claude init")
}

// normalizeAnswers applies the answers invariants the way devinit's update
// does: the answers first adopt the choices committed in .qsdev.yaml (see
// config.AdoptCommitted), so the invariants only infer a tier when the
// project records none and a claude regeneration never replaces the team's
// committed tier. Only claude subcommands normalise through here, and every
// one of them configures Claude Code, so the answers record Claude Code on
// whatever the file says: the invariants then force the self-protection hook
// the generated settings always register, and the answers persisted beside
// those settings agree with them.
func normalizeAnswers(projectRoot string, a *types.WizardAnswers) {
	qsdevconfig.AdoptCommitted(projectRoot, a)
	a.ClaudeCode = true
	answers.EnforceInvariants(a)
}

// overlayInitAnswers merges the answers `claude init` builds from its flags
// onto the project's saved answers, so re-running claude init in a
// qsdev-managed project updates only the Claude Code settings it owns and keeps
// the languages, services, tier and enabled tools recorded by `qsdev init` and
// `qsdev enable`. Without saved answers, the flag answers are returned as-is.
//
// The permission preset and the confirmation are always taken from the flags;
// skills and MCP servers only when the flags name some. The saved hooks are
// kept, so a `disable attach-guard --force` opt-out survives a re-init. The
// answers invariants are applied last, so the answers claude init persists
// record what it generates.
func overlayInitAnswers(projectRoot string, flags types.WizardAnswers) (types.WizardAnswers, error) {
	if _, err := os.Stat(answersPath(projectRoot)); errors.Is(err, fs.ErrNotExist) {
		normalizeAnswers(projectRoot, &flags)
		return flags, nil
	}
	// Normalised once, after the flags are applied, so an inferred tier
	// reflects the new permission preset and MCP servers.
	merged, err := loadSavedAnswers(projectRoot)
	if err != nil {
		return types.WizardAnswers{}, err
	}

	merged.ProjectRoot = flags.ProjectRoot
	if merged.ProjectName == "" {
		merged.ProjectName = flags.ProjectName
	}
	merged.ClaudeCode = flags.ClaudeCode
	merged.PermissionLevel = flags.PermissionLevel
	merged.Confirmed = flags.Confirmed
	if len(flags.Skills) > 0 {
		merged.Skills = flags.Skills
	}
	if len(flags.MCPServers) > 0 {
		merged.MCPServers = flags.MCPServers
	}
	normalizeAnswers(projectRoot, &merged)
	return merged, nil
}
