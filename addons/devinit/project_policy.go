package devinit

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"

	qsdevconfig "github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// applyCommittedPolicy tightens answers to the security floor and client
// policy of the project's committed .qsdev.yaml, resolved together with the
// developer's .qsdev.local.yaml through config.ResolveConfig, and reports
// each setting the floor raised. Create (over an existing .qsdev.yaml) and
// update call it before inferring tools, so hook-implied tools follow the
// raised hooks. A project without .qsdev.yaml has no policy to apply; an
// unreadable one is an error, since its policy cannot be honoured.
func applyCommittedPolicy(w io.Writer, projectRoot string, a *types.WizardAnswers) error {
	policy, err := qsdevconfig.LoadProjectPolicy(projectRoot)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	warnPolicyViolations(w, policy)
	policy.Apply(a)
	// Apply installs the committed java block as is; reject an invalid one
	// here, since update does not revalidate the answers it loads.
	if errs := validateJavaConfig(a.Java); len(errs) > 0 {
		return fmt.Errorf("%s: %s", branding.Get().ConfigFile, strings.Join(errs, "; "))
	}
	return nil
}

// warnPolicyViolations prints each setting the security floor raised.
func warnPolicyViolations(w io.Writer, policy *qsdevconfig.ProjectPolicy) {
	for _, msg := range policy.Warnings() {
		fmt.Fprintln(w, "Warning: "+msg)
	}
}

// localGenerationAnswers returns the answers generation uses: a copy of
// answers with the developer's .qsdev.local.yaml added through
// ProjectPolicy.ApplyLocal (additions and tightenings only), validated like
// any other answers, with the tools the additions imply inferred. answers
// itself is left as it is, because it is what init, join and update persist
// to .qsdev.yaml and the saved answers: a local override must never reach the
// team. A project without a committed .qsdev.yaml has nothing yet for local
// overrides to add to, so its answers are returned unchanged. The overrides
// the floor dropped were already reported when the committed policy was
// applied.
func localGenerationAnswers(projectRoot string, answers types.WizardAnswers) (types.WizardAnswers, error) {
	policy, err := qsdevconfig.LoadProjectPolicy(projectRoot)
	if errors.Is(err, fs.ErrNotExist) {
		return answers, nil
	}
	if err != nil {
		return types.WizardAnswers{}, err
	}
	if policy.Effective.Local == nil {
		return answers, nil
	}
	gen := cloneAnswers(answers)
	policy.ApplyLocal(&gen)
	if err := ValidateAnswers(gen); err != nil {
		return types.WizardAnswers{}, fmt.Errorf("applying %s: %w", branding.Get().LocalConfig, err)
	}
	toolreg.MergeInferredTools(&gen, toolreg.DefaultRegistry())
	enforceAnswerInvariants(&gen)
	return gen, nil
}
