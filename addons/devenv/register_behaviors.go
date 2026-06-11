package devenv

import (
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func init() {
	r := toolreg.DefaultRegistry()

	r.AttachBehavior("changelog", toolreg.ToolBehavior{
		GenerateFunc: func(a types.WizardAnswers) ([]types.GeneratedFile, error) {
			f, err := GenerateCliffToml(a)
			if err != nil {
				return nil, err
			}
			return []types.GeneratedFile{*f}, nil
		},
	})

	r.AttachBehavior("commitlint", toolreg.ToolBehavior{
		GenerateFunc: func(_ types.WizardAnswers) ([]types.GeneratedFile, error) {
			f, err := GenerateCommitlintConfig()
			if err != nil {
				return nil, err
			}
			return []types.GeneratedFile{*f}, nil
		},
	})

	r.AttachBehavior("secretspec", toolreg.ToolBehavior{
		GenerateFunc: func(a types.WizardAnswers) ([]types.GeneratedFile, error) {
			f, err := GenerateSecretSpecToml(a, ecosystem.DefaultRegistry())
			if err != nil {
				return nil, err
			}
			if f == nil {
				return nil, nil
			}
			return []types.GeneratedFile{*f}, nil
		},
	})
}
