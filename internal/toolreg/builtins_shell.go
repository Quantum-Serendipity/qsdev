package toolreg

import (
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/internal/shellenv"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func init() {
	r := DefaultRegistry()

	r.AttachBehavior("starship-integration", ToolBehavior{
		EnableFunc: func(a *types.WizardAnswers) {
			ensureEnabledTools(a)
			a.EnabledTools["starship-integration"] = true
		},
		DisableFunc: func(a *types.WizardAnswers) {
			ensureEnabledTools(a)
			a.EnabledTools["starship-integration"] = false
		},
		GenerateFunc: func(answers types.WizardAnswers) ([]types.GeneratedFile, error) {
			f, err := shellenv.GenerateStarshipToml(answers)
			if err != nil {
				return nil, err
			}
			return []types.GeneratedFile{*f}, nil
		},
		SharedContent: map[string]SharedContentFunc{
			"starship": func(_ types.WizardAnswers) ([]byte, error) {
				return []byte(`  env.STARSHIP_CONFIG = ".starship.toml";`), nil
			},
		},
	})

	r.AttachBehavior("otel-config", ToolBehavior{
		EnableFunc: func(a *types.WizardAnswers) {
			ensureEnabledTools(a)
			a.EnabledTools["otel-config"] = true
		},
		DisableFunc: func(a *types.WizardAnswers) {
			ensureEnabledTools(a)
			a.EnabledTools["otel-config"] = false
		},
		SharedContent: map[string]SharedContentFunc{
			"otel-config": otelConfigNixContent,
		},
	})
}

func otelConfigNixContent(answers types.WizardAnswers) ([]byte, error) {
	projectName := answers.ProjectName
	if projectName == "" {
		projectName = "unknown"
	}

	endpoint := "http://localhost:4317"
	if answers.EnvVars != nil {
		if ep, ok := answers.EnvVars["OTEL_EXPORTER_OTLP_ENDPOINT"]; ok && ep != "" {
			endpoint = ep
		}
	}

	nix := fmt.Sprintf(`  env.OTEL_EXPORTER_OTLP_ENDPOINT = "%s";
  env.OTEL_EXPORTER_OTLP_PROTOCOL = "grpc";
  env.OTEL_SERVICE_NAME = "%s";
  env.OTEL_TRACES_SAMPLER = "parentbased_traceidratio";
  env.OTEL_TRACES_SAMPLER_ARG = "0.1";`, endpoint, projectName)

	return []byte(nix), nil
}
