package ecosystem

import (
	"reflect"
	"strings"
	"testing"
)

func TestAggregateCICommands(t *testing.T) {
	t.Parallel()

	install := func(c string) CICommand { return CICommand{Name: c, Command: c, Phase: CIPhaseInstall} }
	test := func(c string) CICommand { return CICommand{Name: c, Command: c, Phase: CIPhaseTest} }
	scan := func(c string) CICommand { return CICommand{Name: c, Command: c, Phase: CIPhaseScan} }

	tests := []struct {
		name    string
		modules []EcosystemModule
		want    []CIPhaseGroup
	}{
		{
			name:    "no modules",
			modules: nil,
			want:    nil,
		},
		{
			name:    "module without CI commands",
			modules: []EcosystemModule{&MockModule{NameVal: "a"}},
			want:    nil,
		},
		{
			name: "grouped in pipeline order across modules",
			modules: []EcosystemModule{
				&MockModule{NameVal: "rust", CICommandsVal: []CICommand{scan("cargo audit"), install("cargo build --locked")}},
				&MockModule{NameVal: "js", CICommandsVal: []CICommand{test("npm test"), install("npm ci --ignore-scripts")}},
			},
			want: []CIPhaseGroup{
				{Phase: CIPhaseInstall, Commands: []CICommand{install("cargo build --locked"), install("npm ci --ignore-scripts")}},
				{Phase: CIPhaseTest, Commands: []CICommand{test("npm test")}},
				{Phase: CIPhaseScan, Commands: []CICommand{scan("cargo audit")}},
			},
		},
		{
			name: "duplicate command line runs once",
			modules: []EcosystemModule{
				&MockModule{NameVal: "a", CICommandsVal: []CICommand{scan("grype dir:.")}},
				&MockModule{NameVal: "b", CICommandsVal: []CICommand{scan("grype dir:."), scan("hadolint Dockerfile")}},
			},
			want: []CIPhaseGroup{
				{Phase: CIPhaseScan, Commands: []CICommand{scan("grype dir:."), scan("hadolint Dockerfile")}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := AggregateCICommands(tt.modules, staticConfig)
			if err != nil {
				t.Fatalf("AggregateCICommands: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AggregateCICommands =\n%+v\nwant\n%+v", got, tt.want)
			}
		})
	}
}

func TestAggregateCICommands_UnknownPhase(t *testing.T) {
	t.Parallel()

	mod := &MockModule{NameVal: "odd", CICommandsVal: []CICommand{{Name: "x", Command: "x", Phase: CIPhase(99)}}}
	_, err := AggregateCICommands([]EcosystemModule{mod}, staticConfig)
	if err == nil || !strings.Contains(err.Error(), "unknown CI phase") {
		t.Fatalf("err = %v, want an unknown CI phase error", err)
	}
}

func TestAggregateCICommands_PassesModuleConfig(t *testing.T) {
	t.Parallel()

	mod := &configEchoModule{MockModule: MockModule{NameVal: "js"}}
	got, err := AggregateCICommands([]EcosystemModule{mod}, func(EcosystemModule) ModuleConfig {
		return ModuleConfig{PackageManager: "pnpm"}
	})
	if err != nil {
		t.Fatalf("AggregateCICommands: %v", err)
	}
	if len(got) != 1 || got[0].Commands[0].Command != "pnpm" {
		t.Errorf("AggregateCICommands = %+v, want the command built from the module's config", got)
	}
}

// configEchoModule returns a CI command naming the package manager it was
// configured with.
type configEchoModule struct {
	MockModule
}

func (m *configEchoModule) CICommands(config ModuleConfig) []CICommand {
	return []CICommand{{Name: "echo", Command: config.PackageManager, Phase: CIPhaseInstall}}
}
