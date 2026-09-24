package devinit

import (
	"errors"
	"fmt"
	"io"
	"io/fs"

	qsdevconfig "github.com/Quantum-Serendipity/qsdev/internal/config"
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
	return nil
}

// warnPolicyViolations prints each setting the security floor raised.
func warnPolicyViolations(w io.Writer, policy *qsdevconfig.ProjectPolicy) {
	for _, msg := range policy.Warnings() {
		fmt.Fprintln(w, "Warning: "+msg)
	}
}
