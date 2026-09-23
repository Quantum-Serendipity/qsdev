package toolreg

import (
	"fmt"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/shellenv"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func shellBehaviors() map[string]ToolBehavior {
	return map[string]ToolBehavior{
		"starship-integration": {
			GenerateFunc: func(answers types.WizardAnswers) ([]types.GeneratedFile, error) {
				f, err := shellenv.GenerateStarshipToml(answers)
				if err != nil {
					return nil, err
				}
				return []types.GeneratedFile{*f}, nil
			},
			SharedContent: map[SharedSection]SharedContentFunc{
				{Path: DevenvNixFile, SectionID: "starship"}: func(_ types.WizardAnswers) ([]byte, error) {
					return []byte(`  env.STARSHIP_CONFIG = ".starship.toml";`), nil
				},
			},
		},
		"otel-config": {
			SharedContent: map[SharedSection]SharedContentFunc{
				{Path: DevenvNixFile, SectionID: "otel-config"}: otelConfigNixContent,
			},
		},
	}
}

func otelConfigNixContent(answers types.WizardAnswers) ([]byte, error) {
	projectName := answers.ProjectName
	if projectName == "" {
		projectName = "unknown"
	}

	// Defaults only: a value the user set in answers.EnvVars (e.g. a custom
	// collector endpoint) is already rendered in devenv.nix's env block.
	vars := []struct{ name, value string }{
		{"OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4317"},
		{"OTEL_EXPORTER_OTLP_PROTOCOL", "grpc"},
		{"OTEL_SERVICE_NAME", projectName},
		{"OTEL_TRACES_SAMPLER", "parentbased_traceidratio"},
		{"OTEL_TRACES_SAMPLER_ARG", "0.1"},
	}
	var lines []string
	for _, v := range vars {
		// devenv.nix already renders answers.EnvVars in its env block; defining
		// the same attribute again is a Nix "already defined" error.
		if _, userSet := answers.EnvVars[v.name]; userSet {
			continue
		}
		lines = append(lines, fmt.Sprintf("  env.%s = %s;", v.name, ecosystem.NixString(v.value)))
	}
	return []byte(strings.Join(lines, "\n")), nil
}
