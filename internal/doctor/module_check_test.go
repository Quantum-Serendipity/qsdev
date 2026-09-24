package doctor

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

func TestEvaluateModuleCheck(t *testing.T) {
	t.Parallel()

	envOf := func(vars map[string]string) func(string) (string, bool) {
		return func(k string) (string, bool) {
			v, ok := vars[k]
			return v, ok
		}
	}
	lookPathFound := func(name string) (string, error) { return "/nix/store/x/bin/" + name, nil }
	lookPathMissing := func(string) (string, error) { return "", errors.New("not found") }

	envCheck := ecosystem.DoctorCheck{Name: "azure-sub", Description: "ARM_SUBSCRIPTION_ID", EnvCheck: "ARM_SUBSCRIPTION_ID"}
	cmdCheck := ecosystem.DoctorCheck{Name: "azure-auth", Description: "Azure login status", Command: "az account show", Timeout: 5}

	tests := []struct {
		name       string
		check      ecosystem.DoctorCheck
		env        ModuleCheckEnv
		wantStatus string
		wantDetail string
	}{
		{
			name:       "env set in process",
			check:      envCheck,
			env:        ModuleCheckEnv{LookupEnv: envOf(map[string]string{"ARM_SUBSCRIPTION_ID": "0000-1111"})},
			wantStatus: ModuleCheckOK,
			wantDetail: "current environment",
		},
		{
			name:       "env declared by devenv",
			check:      envCheck,
			env:        ModuleCheckEnv{LookupEnv: envOf(nil), Declared: map[string]string{"ARM_SUBSCRIPTION_ID": "0000-1111"}},
			wantStatus: ModuleCheckOK,
			wantDetail: "devenv modules",
		},
		{
			name:       "process placeholder is unset",
			check:      envCheck,
			env:        ModuleCheckEnv{LookupEnv: envOf(map[string]string{"ARM_SUBSCRIPTION_ID": "PLACEHOLDER -- set your subscription"})},
			wantStatus: ModuleCheckWarn,
			wantDetail: "placeholder",
		},
		{
			name:  "declared placeholder is unset",
			check: envCheck,
			env: ModuleCheckEnv{
				LookupEnv: envOf(map[string]string{"ARM_SUBSCRIPTION_ID": "  "}),
				Declared:  map[string]string{"ARM_SUBSCRIPTION_ID": "PLACEHOLDER -- set your subscription"},
			},
			wantStatus: ModuleCheckWarn,
			wantDetail: "devenv.local.nix",
		},
		{
			name:  "declared placeholder overrides real process value",
			check: envCheck,
			env: ModuleCheckEnv{
				LookupEnv: envOf(map[string]string{"ARM_SUBSCRIPTION_ID": "0000-1111"}),
				Declared:  map[string]string{"ARM_SUBSCRIPTION_ID": "PLACEHOLDER -- set your subscription"},
			},
			wantStatus: ModuleCheckWarn,
			wantDetail: "declared by the project's devenv modules as a placeholder",
		},
		{
			name:  "declared value overrides process placeholder",
			check: envCheck,
			env: ModuleCheckEnv{
				LookupEnv: envOf(map[string]string{"ARM_SUBSCRIPTION_ID": "PLACEHOLDER"}),
				Declared:  map[string]string{"ARM_SUBSCRIPTION_ID": "0000-1111"},
			},
			wantStatus: ModuleCheckOK,
			wantDetail: "devenv modules",
		},
		{
			name:       "template value is unset",
			check:      envCheck,
			env:        ModuleCheckEnv{Declared: map[string]string{"ARM_SUBSCRIPTION_ID": "<Azure subscription ID>"}},
			wantStatus: ModuleCheckWarn,
		},
		{
			name:       "command on PATH is not run",
			check:      cmdCheck,
			env:        ModuleCheckEnv{LookPath: lookPathFound},
			wantStatus: ModuleCheckNotRun,
			wantDetail: "verify manually with: az account show",
		},
		{
			name:       "command missing from PATH",
			check:      cmdCheck,
			env:        ModuleCheckEnv{LookPath: lookPathMissing},
			wantStatus: ModuleCheckWarn,
			wantDetail: "az not found on PATH",
		},
		{
			name:       "no PATH resolver",
			check:      cmdCheck,
			wantStatus: ModuleCheckNotRun,
			wantDetail: "az account show",
		},
		{
			name:       "empty check",
			check:      ecosystem.DoctorCheck{Name: "noop"},
			wantStatus: ModuleCheckNotRun,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := EvaluateModuleCheck("azure", tt.check, tt.env)
			if got.Status != tt.wantStatus {
				t.Errorf("Status = %q, want %q (detail %q)", got.Status, tt.wantStatus, got.Detail)
			}
			if !strings.Contains(got.Detail, tt.wantDetail) {
				t.Errorf("Detail = %q, want it to contain %q", got.Detail, tt.wantDetail)
			}
			if got.Module != "azure" || got.Name != tt.check.Name || got.Command != tt.check.Command {
				t.Errorf("identity = %+v, want module azure, check %+v", got, tt.check)
			}
		})
	}
}

func TestNewModuleCheckSection(t *testing.T) {
	t.Parallel()

	if got := NewModuleCheckSection(nil, nil); got != nil {
		t.Errorf("NewModuleCheckSection(nil, nil) = %+v, want nil", got)
	}
	ms := NewModuleCheckSection(nil, []string{"bad config"})
	if ms == nil || !ms.Detected || len(ms.Warnings) != 1 {
		t.Errorf("NewModuleCheckSection(nil, warnings) = %+v, want detected section with warning", ms)
	}
}

func TestFormatReport_ModuleChecks(t *testing.T) {
	t.Parallel()

	r := &Report{}
	r.SetModuleCheckSection(NewModuleCheckSection([]ModuleCheckInfo{
		{Module: "gcp", Name: "gcp-config", Description: "CLOUDSDK_ACTIVE_CONFIG_NAME", Status: ModuleCheckWarn, Detail: "unset"},
		{Module: "gcp", Name: "gcp-auth", Description: "GCP authentication", Status: ModuleCheckNotRun, Detail: "verify manually"},
	}, []string{"parsing devenv.local.nix: bad"}))

	var buf bytes.Buffer
	FormatReport(&buf, r, false)
	out := buf.String()
	for _, want := range []string{"Ecosystem Checks", "[WARN] CLOUDSDK_ACTIVE_CONFIG_NAME", "- GCP authentication", "verify manually", "[WARN] parsing devenv.local.nix"} {
		if !strings.Contains(out, want) {
			t.Errorf("FormatReport output missing %q:\n%s", want, out)
		}
	}
}
