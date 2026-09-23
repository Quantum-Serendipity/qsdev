package check

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestCheckConfigIntegrity_ValidConfig(t *testing.T) {
	ctx := CheckContext{
		QsdevConfig: &types.QsdevConfig{
			Version: 1,
			Languages: []types.LanguageConfig{
				{Name: "go", Version: "1.24"},
			},
			Services: []types.ServiceConfig{
				{Name: "postgres"},
			},
			Profile: "go-web",
		},
		ProfileNames: []string{"go-web", "ts-fullstack"},
		ToolNames:    []string{"safety-block", "pre-commit"},
	}

	results := CheckConfigIntegrity(ctx)

	// Should have a passing config_exists check.
	var configExists *CheckResult
	for i := range results {
		if results[i].Name == "config_exists" {
			configExists = &results[i]
			break
		}
	}

	if configExists == nil {
		t.Fatal("expected config_exists result")
		return
	}
	if configExists.Status != StatusPass {
		t.Errorf("config_exists.Status = %s, want %s", configExists.Status, StatusPass)
	}

	// No validation failures expected.
	for _, r := range results {
		if r.Status == StatusFail {
			t.Errorf("unexpected failure: %s: %s", r.Name, r.Message)
		}
	}
}

func TestCheckConfigIntegrity_InvalidLanguage(t *testing.T) {
	ctx := CheckContext{
		QsdevConfig: &types.QsdevConfig{
			Version: 1,
			Languages: []types.LanguageConfig{
				{Name: "cobol"},
			},
		},
	}

	results := CheckConfigIntegrity(ctx)

	hasFail := false
	for _, r := range results {
		if r.Status == StatusFail && r.Name == "config_validation" {
			hasFail = true
			break
		}
	}

	if !hasFail {
		t.Error("expected a config_validation failure for unknown language")
	}
}

func TestCheckConfigIntegrity_NoConfig(t *testing.T) {
	ctx := CheckContext{}

	results := CheckConfigIntegrity(ctx)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != StatusFail {
		t.Errorf("Status = %s, want %s", results[0].Status, StatusFail)
	}
	if results[0].Severity != SeverityCritical {
		t.Errorf("Severity = %s, want %s", results[0].Severity, SeverityCritical)
	}
}

func TestCheckConfigIntegrity_InvalidProfile(t *testing.T) {
	ctx := CheckContext{
		QsdevConfig: &types.QsdevConfig{
			Version: 1,
			Profile: "unknown-profile",
		},
		ProfileNames: []string{"go-web", "ts-fullstack"},
	}

	results := CheckConfigIntegrity(ctx)

	hasFail := false
	for _, r := range results {
		if r.Status == StatusFail {
			hasFail = true
			break
		}
	}

	if !hasFail {
		t.Error("expected a validation failure for unknown profile")
	}
}

func TestCheckConfigIntegrity_InvalidService(t *testing.T) {
	ctx := CheckContext{
		QsdevConfig: &types.QsdevConfig{
			Version: 1,
			Services: []types.ServiceConfig{
				{Name: "oracle"},
			},
		},
	}

	results := CheckConfigIntegrity(ctx)

	hasFail := false
	for _, r := range results {
		if r.Status == StatusFail {
			hasFail = true
			break
		}
	}

	if !hasFail {
		t.Error("expected a validation failure for unknown service")
	}
}

// TestCheckConfigIntegrity_ParseErrorIsNotNotFound verifies that a config that
// exists but fails to parse is reported with its real error rather than as
// "not found", and that dependent checks explain why they are skipped.
func TestCheckConfigIntegrity_ParseErrorIsNotNotFound(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		content     string // "" means the file does not exist
		wantName    string
		wantMessage string
		wantSkip    string
	}{
		{name: "missing", wantName: "config_exists", wantMessage: "not found", wantSkip: "found; skipping"},
		{name: "newer schema", content: "version: 2\n", wantName: "config_parse", wantMessage: "newer than this binary", wantSkip: "could not be parsed"},
		{name: "unknown key", content: "version: 1\nsecurity:\n  script_blockng: true\n", wantName: "config_parse", wantMessage: "script_blockng", wantSkip: "could not be parsed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), ".qsdev.yaml")
			if tt.content != "" {
				if err := os.WriteFile(path, []byte(tt.content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			cfg, err := config.ParseQsdevConfig(path)
			ctx := CheckContext{QsdevConfig: cfg, ConfigErr: err}

			results := CheckConfigIntegrity(ctx)
			if len(results) != 1 {
				t.Fatalf("expected 1 result, got %d", len(results))
			}
			r := results[0]
			if r.Name != tt.wantName || r.Status != StatusFail || r.Severity != SeverityCritical {
				t.Errorf("got %s %s/%s, want %s fail/critical", r.Name, r.Status, r.Severity, tt.wantName)
			}
			if !strings.Contains(r.Message, tt.wantMessage) {
				t.Errorf("Message = %q, want it to contain %q", r.Message, tt.wantMessage)
			}

			for _, dep := range [][]CheckResult{CheckRequiredTools(ctx), CheckSecurityHardening(ctx), CheckBinaryCompatibility(ctx)} {
				if len(dep) != 1 || dep[0].Status != StatusSkip || !strings.Contains(dep[0].Message, tt.wantSkip) {
					t.Errorf("dependent check = %+v, want skip mentioning %q", dep, tt.wantSkip)
				}
			}
		})
	}
}
