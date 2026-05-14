package toolreg

import (
	"fmt"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/shellenv"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

func init() {
	r := DefaultRegistry()
	for _, t := range shellTools() {
		_ = r.Register(t)
	}
}

func shellTools() []Tool {
	return []Tool{
		starshipIntegrationTool(),
		otelConfigTool(),
	}
}

func starshipIntegrationTool() Tool {
	return Tool{
		Name:        "starship-integration",
		DisplayName: "Starship Prompt Integration",
		Category:    CategoryDevEx,
		Description: "Starship prompt configuration with gdev project name, security profile, and tool count segments",
		Default:     OptIn,
		OwnedFiles: []FileOwnership{
			{Path: ".starship.toml", Ownership: Exclusive},
			{Path: "devenv.nix", Ownership: Shared, SectionID: "starship"},
		},
		EnableFunc: func(a *types.WizardAnswers) {
			if a.EnabledTools == nil {
				a.EnabledTools = make(map[string]bool)
			}
			a.EnabledTools["starship-integration"] = true
		},
		DisableFunc: func(a *types.WizardAnswers) {
			if a.EnabledTools == nil {
				a.EnabledTools = make(map[string]bool)
			}
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
	}
}

func otelConfigTool() Tool {
	return Tool{
		Name:        "otel-config",
		DisplayName: "OpenTelemetry Configuration",
		Category:    CategoryInfrastructure,
		Description: "OpenTelemetry environment variable configuration for traces and metrics collection",
		Default:     OptIn,
		OwnedFiles: []FileOwnership{
			{Path: "devenv.nix", Ownership: Shared, SectionID: "otel-config"},
		},
		EnableFunc: func(a *types.WizardAnswers) {
			if a.EnabledTools == nil {
				a.EnabledTools = make(map[string]bool)
			}
			a.EnabledTools["otel-config"] = true
		},
		DisableFunc: func(a *types.WizardAnswers) {
			if a.EnabledTools == nil {
				a.EnabledTools = make(map[string]bool)
			}
			a.EnabledTools["otel-config"] = false
		},
		GenerateFunc: nil,
		SharedContent: map[string]SharedContentFunc{
			"otel-config": otelConfigNixContent,
		},
	}
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
