package answers

import (
	"github.com/Quantum-Serendipity/qsdev/internal/tier"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// EnforceInvariants applies the settings that no input path (flags, profile,
// answers file, wizard, join, update, claude subcommands) may leave off. Every
// path that hands answers to the generators or persists them calls it last,
// after policy, overlays and flags have been applied. The Claude Code hook
// registry independently always registers self-protection and registers
// package-guard unless the safety block was opted out of; this keeps the
// recorded answers truthful about both.
//
// Load does not call it: loaded answers are still changed by policy, overlays
// and flags before use, so only the last step can guarantee the invariant,
// and inferring the tier at load would pre-empt config.AdoptCommitted,
// replacing a team's committed tier with an inferred one. Load-time
// normalisation belongs to projectmodel.Resolve (XA-WS3).
func EnforceInvariants(a *types.WizardAnswers) {
	// The hook invariant has one definition: self-protection on, and the
	// safety block mirroring the recorded opt-out, whenever Claude Code is
	// configured.
	a.ApplyClaudeHookDefaults()
	// The tier is always recorded. Create paths resolve it in FillDefaults;
	// answers saved before that are legacy, so their tier is inferred once
	// here and then persisted (answers file and .qsdev.yaml) on save.
	if a.Tier == "" {
		a.Tier = tier.Infer(a.PermissionLevel, a.MCPServers).String()
	}
}
