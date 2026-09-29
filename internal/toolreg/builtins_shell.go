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
			GenerateFunc: func(_ types.WizardAnswers) ([]types.GeneratedFile, error) {
				f, err := shellenv.GenerateStarshipToml()
				if err != nil {
					return nil, err
				}
				return []types.GeneratedFile{*f}, nil
			},
			SharedContent: map[SharedSection]SharedContentFunc{
				{Path: DevenvNixFile, SectionID: "starship"}: func(_ types.WizardAnswers) ([]byte, error) {
					return shellenv.StarshipNixContent(), nil
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

// otelServiceNameVar is filled from the project name rather than a constant.
const otelServiceNameVar = "OTEL_SERVICE_NAME"

// otelDefaults lists the OpenTelemetry variables the otel-config section
// sets, in emission order.
var otelDefaults = []struct{ name, value string }{
	{"OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4317"},
	{"OTEL_EXPORTER_OTLP_PROTOCOL", "grpc"},
	{otelServiceNameVar, ""},
	{"OTEL_TRACES_SAMPLER", "parentbased_traceidratio"},
	{"OTEL_TRACES_SAMPLER_ARG", "0.1"},
}

// otelConfigNixContent renders the otel-config devenv.nix section. Values are
// Nix-escaped because the project name comes from user input. Variables the
// user already set through answers.EnvVars (the `--env` flag) are skipped:
// the devenv.nix env block already emits them, and defining the same
// attribute again in this section makes Nix reject the file with
// "attribute 'env.X' already defined".
func otelConfigNixContent(answers types.WizardAnswers) ([]byte, error) {
	projectName := answers.ProjectName
	if projectName == "" {
		projectName = "unknown"
	}

	// Defaults only: a value the user set in answers.EnvVars (e.g. a custom
	// collector endpoint) is already rendered in devenv.nix's env block, and
	// defining the same attribute again is a Nix "already defined" error.
	lines := make([]string, 0, len(otelDefaults))
	for _, v := range otelDefaults {
		if _, userSet := answers.EnvVars[v.name]; userSet {
			continue
		}
		value := v.value
		if v.name == otelServiceNameVar {
			value = projectName
		}
		lines = append(lines, fmt.Sprintf("  env.%s = %s;", v.name, ecosystem.NixString(value)))
	}
	return []byte(strings.Join(lines, "\n")), nil
}
