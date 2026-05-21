package toolreg

import (
	"github.com/Quantum-Serendipity/qsdev/addons/devenv"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func init() {
	r := DefaultRegistry()

	r.AttachBehavior("changelog", ToolBehavior{
		EnableFunc: func(a *types.WizardAnswers) {
			if a.EnabledTools == nil {
				a.EnabledTools = make(map[string]bool)
			}
			a.EnabledTools["changelog"] = true
		},
		DisableFunc: func(a *types.WizardAnswers) {
			if a.EnabledTools == nil {
				a.EnabledTools = make(map[string]bool)
			}
			a.EnabledTools["changelog"] = false
		},
		GenerateFunc: func(a types.WizardAnswers) ([]types.GeneratedFile, error) {
			f, err := devenv.GenerateCliffToml(a)
			if err != nil {
				return nil, err
			}
			return []types.GeneratedFile{*f}, nil
		},
	})

	r.AttachBehavior("commitlint", ToolBehavior{
		EnableFunc: func(a *types.WizardAnswers) {
			if a.EnabledTools == nil {
				a.EnabledTools = make(map[string]bool)
			}
			a.EnabledTools["commitlint"] = true
		},
		DisableFunc: func(a *types.WizardAnswers) {
			if a.EnabledTools == nil {
				a.EnabledTools = make(map[string]bool)
			}
			a.EnabledTools["commitlint"] = false
		},
		GenerateFunc: func(a types.WizardAnswers) ([]types.GeneratedFile, error) {
			f, err := devenv.GenerateCommitlintConfig()
			if err != nil {
				return nil, err
			}
			return []types.GeneratedFile{*f}, nil
		},
	})

	r.AttachBehavior("secretspec", ToolBehavior{
		EnableFunc: func(a *types.WizardAnswers) {
			if a.EnabledTools == nil {
				a.EnabledTools = make(map[string]bool)
			}
			a.EnabledTools["secretspec"] = true
		},
		DisableFunc: func(a *types.WizardAnswers) {
			if a.EnabledTools == nil {
				a.EnabledTools = make(map[string]bool)
			}
			a.EnabledTools["secretspec"] = false
		},
		GenerateFunc: func(a types.WizardAnswers) ([]types.GeneratedFile, error) {
			f, err := devenv.GenerateSecretSpecToml(a, ecosystem.DefaultRegistry())
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
