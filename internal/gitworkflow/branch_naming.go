package gitworkflow

// DefaultBranchPattern is the branch naming pattern the always-on
// branch-naming pre-push hook enforces when .qsdev.yaml sets no
// git.branch_pattern. It is deliberately broad, so no team's existing
// convention is rejected (feat/..., audit/..., users/jane/JIRA-12_fix,
// dependabot/npm_and_yarn/@types/node-20.1.0): a name must start with an
// ASCII letter or digit and continue with letters, digits and . _ / @ + -.
// What it rejects are names that are hazardous where branch names are
// interpolated unquoted, such as CI scripts (`${{ github.head_ref }}`) and
// shell prompts: shell metacharacters ($ ` ; | & ( ) < > quotes), a leading
// '-' that reads as an option, and non-ASCII characters (homoglyphs,
// bidirectional-override tricks).
const DefaultBranchPattern = `^[A-Za-z0-9][A-Za-z0-9._/@+-]*$`

// EffectiveBranchPattern returns the pattern the branch-naming hook enforces
// for a configured git.branch_pattern: the configured one, or
// DefaultBranchPattern when none is set.
func EffectiveBranchPattern(configured string) string {
	if configured == "" {
		return DefaultBranchPattern
	}
	return configured
}
