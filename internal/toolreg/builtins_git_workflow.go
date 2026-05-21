package toolreg

import (
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/internal/gitworkflow"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func init() {
	r := DefaultRegistry()

	r.AttachBehavior("pr-templates", ToolBehavior{
		EnableFunc: func(a *types.WizardAnswers) {
			if a.EnabledTools == nil {
				a.EnabledTools = make(map[string]bool)
			}
			a.EnabledTools["pr-templates"] = true
		},
		DisableFunc: func(a *types.WizardAnswers) {
			if a.EnabledTools == nil {
				a.EnabledTools = make(map[string]bool)
			}
			a.EnabledTools["pr-templates"] = false
		},
		GenerateFunc: func(answers types.WizardAnswers) ([]types.GeneratedFile, error) {
			f, err := gitworkflow.GeneratePRTemplate(answers)
			if err != nil {
				return nil, err
			}
			return []types.GeneratedFile{*f}, nil
		},
	})

	r.AttachBehavior("branch-naming", ToolBehavior{
		EnableFunc: func(a *types.WizardAnswers) {
			if a.EnabledTools == nil {
				a.EnabledTools = make(map[string]bool)
			}
			a.EnabledTools["branch-naming"] = true
		},
		DisableFunc: func(a *types.WizardAnswers) {
			if a.EnabledTools == nil {
				a.EnabledTools = make(map[string]bool)
			}
			a.EnabledTools["branch-naming"] = false
		},
		SharedContent: map[string]SharedContentFunc{
			"branch-naming": branchNamingNixContent,
		},
	})

	r.AttachBehavior("commit-ticket", ToolBehavior{
		EnableFunc: func(a *types.WizardAnswers) {
			if a.EnabledTools == nil {
				a.EnabledTools = make(map[string]bool)
			}
			a.EnabledTools["commit-ticket"] = true
		},
		DisableFunc: func(a *types.WizardAnswers) {
			if a.EnabledTools == nil {
				a.EnabledTools = make(map[string]bool)
			}
			a.EnabledTools["commit-ticket"] = false
		},
		SharedContent: map[string]SharedContentFunc{
			"commit-ticket": commitTicketNixContent,
		},
	})

	r.AttachBehavior("pr-labels", ToolBehavior{
		EnableFunc: func(a *types.WizardAnswers) {
			if a.EnabledTools == nil {
				a.EnabledTools = make(map[string]bool)
			}
			a.EnabledTools["pr-labels"] = true
		},
		DisableFunc: func(a *types.WizardAnswers) {
			if a.EnabledTools == nil {
				a.EnabledTools = make(map[string]bool)
			}
			a.EnabledTools["pr-labels"] = false
		},
		GenerateFunc: func(answers types.WizardAnswers) ([]types.GeneratedFile, error) {
			return gitworkflow.GenerateLabelerConfig(answers)
		},
	})
}

func branchNamingNixContent(_ types.WizardAnswers) ([]byte, error) {
	pattern := `^(feat|fix|chore|docs|refactor|test|ci)/[a-z0-9._-]+$`

	nix := fmt.Sprintf(`  git-hooks.hooks.branch-naming = {
    enable = true;
    name = "Branch naming convention";
    description = "Validates branch name against allowed patterns";
    entry = "${pkgs.writeShellScript \"branch-naming\" ''
      branch=$(git rev-parse --abbrev-ref HEAD)
      pattern=\"%s\"
      if [ \"$branch\" = \"main\" ] || [ \"$branch\" = \"master\" ] || [ \"$branch\" = \"develop\" ]; then
        exit 0
      fi
      if ! echo \"$branch\" | grep -qE \"$pattern\"; then
        echo \"ERROR: Branch name '$branch' does not match convention.\"
        echo \"Expected: feat|fix|chore|docs|refactor|test|ci/<description>\"
        exit 1
      fi
    ''}";
    language = "system";
    stages = [ "pre-push" ];
    pass_filenames = false;
  };`, pattern)

	return []byte(nix), nil
}

func commitTicketNixContent(_ types.WizardAnswers) ([]byte, error) {
	nix := `  git-hooks.hooks.commit-ticket = {
    enable = true;
    name = "Commit ticket extraction";
    description = "Extracts ticket ID from branch name and prepends to commit message";
    entry = "${pkgs.writeShellScript \"commit-ticket\" ''
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
