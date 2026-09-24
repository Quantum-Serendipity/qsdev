package toolreg

import (
	"fmt"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/gitworkflow"
	"github.com/Quantum-Serendipity/qsdev/internal/validation"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func gitWorkflowBehaviors() map[string]ToolBehavior {
	return map[string]ToolBehavior{
		"pr-templates": {
			GenerateFunc: func(answers types.WizardAnswers) ([]types.GeneratedFile, error) {
				f, err := gitworkflow.GeneratePRTemplate(answers)
				if err != nil {
					return nil, err
				}
				return []types.GeneratedFile{*f}, nil
			},
		},
		"branch-naming": {
			SharedContent: map[SharedSection]SharedContentFunc{
				{Path: DevenvNixFile, SectionID: "branch-naming"}: branchNamingNixContent,
			},
		},
		"commit-ticket": {
			SharedContent: map[SharedSection]SharedContentFunc{
				{Path: DevenvNixFile, SectionID: "commit-ticket"}: commitTicketNixContent,
			},
		},
		"pr-labels": {
			GenerateFunc: func(answers types.WizardAnswers) ([]types.GeneratedFile, error) {
				return gitworkflow.GenerateLabelerConfig(answers)
			},
		},
	}
}

// The hook scripts below are Nix indented strings ('' ... '') inside a
// ${ ... } interpolation, which is Nix expression context: the script name
// is a plain Nix string literal and must not be backslash-escaped, and the
// script body reaches the shell verbatim, so it uses ordinary shell quoting.

// branchNamingNixContent renders the branch-naming pre-push hook for the
// committed git.branch_pattern (answers.BranchPattern), or the broad
// gitworkflow.DefaultBranchPattern when none is set. The pattern is validated
// again here because this is where it is spliced into devenv.nix: it lands
// in a shell single-quoted string inside the Nix indented string, so it must
// not contain a quote, and a "${" in it is escaped for Nix ("”${").
func branchNamingNixContent(answers types.WizardAnswers) ([]byte, error) {
	pattern := gitworkflow.EffectiveBranchPattern(answers.BranchPattern)
	if err := validation.CheckBranchPattern(pattern); err != nil {
		return nil, fmt.Errorf("rendering branch-naming hook for git.branch_pattern %q: %w", pattern, err)
	}
	b := branding.Get()
	r := strings.NewReplacer(
		"@PATTERN@", strings.ReplaceAll(pattern, "${", "''${"),
		"@CONFIG@", b.ConfigFile,
		"@APP@", b.AppName,
	)
	return []byte(r.Replace(branchNamingNixTemplate)), nil
}

// branchNamingNixTemplate is the hook section; branchNamingNixContent fills
// in the @...@ placeholders. A detached HEAD and the usual default branches
// are always accepted, so a custom pattern never blocks pushing them. The
// check is about the branch, not files, so always_run keeps pre-commit from
// skipping it when the pushed commits touch no matching file (e.g. only
// deletions).
const branchNamingNixTemplate = `  git-hooks.hooks.branch-naming = {
    enable = true;
    name = "Branch naming convention";
    description = "Validates the branch name against git.branch_pattern";
    entry = "${pkgs.writeShellScript "branch-naming" ''
      branch=$(git rev-parse --abbrev-ref HEAD)
      pattern='@PATTERN@'
      case "$branch" in
        HEAD|main|master|develop) exit 0 ;;
      esac
      if ! printf '%s\n' "$branch" | LC_ALL=C grep -qE -e "$pattern"; then
        echo "ERROR: Branch name '$branch' does not match the branch naming pattern."
        echo "Expected a name matching: $pattern"
        echo "Rename the branch (git branch -m <new-name>), or set git.branch_pattern"
        echo "in @CONFIG@ and run '@APP@ init --update'."
        exit 1
      fi
    ''}";
    language = "system";
    stages = [ "pre-push" ];
    pass_filenames = false;
    always_run = true;
  };`

func commitTicketNixContent(_ types.WizardAnswers) ([]byte, error) {
	nix := `  git-hooks.hooks.commit-ticket = {
    enable = true;
    name = "Commit ticket extraction";
    description = "Extracts ticket ID from branch name and prepends to commit message";
    entry = "${pkgs.writeShellScript "commit-ticket" ''
      COMMIT_MSG_FILE="$1"
      COMMIT_SOURCE="$2"
      # Only prepend for new commits (not amend, merge, etc.)
      if [ -n "$COMMIT_SOURCE" ]; then
        exit 0
      fi
      branch=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || true)
      ticket=$(echo "$branch" | grep -oE '[A-Z]+-[0-9]+' | head -1)
      if [ -n "$ticket" ]; then
        msg=$(cat "$COMMIT_MSG_FILE")
        # Don't add if already present.
        if ! echo "$msg" | grep -qF "$ticket"; then
          printf '%s %s' "$ticket" "$msg" > "$COMMIT_MSG_FILE"
        fi
      fi
    ''}";
    language = "system";
    stages = [ "prepare-commit-msg" ];
    pass_filenames = false;
  };`

	return []byte(nix), nil
}
