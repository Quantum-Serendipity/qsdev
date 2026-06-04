package contracttest

import (
	"context"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
)

func TestConfigRenderer(t *testing.T, renderer aiframework.ConfigRenderer, fixtures ContractFixtures) {
	t.Helper()

	t.Run("FormatValid", func(t *testing.T) {
		format := renderer.Format()
		validFormats := map[string]bool{"json": true, "toml": true, "yaml": true, "mdc": true}
		if !validFormats[format] {
			t.Errorf("Format() = %q, want one of json/toml/yaml/mdc", format)
		}
	})

	t.Run("RenderProducesFiles", func(t *testing.T) {
		if fixtures.PolicyInput == nil {
			t.Skip("PolicyInput not provided")
		}
		files, err := renderer.Render(context.Background(), fixtures.PolicyInput)
		if err != nil {
			t.Fatalf("Render() error: %v", err)
		}
		if len(files) == 0 {
			t.Error("Render() produced no files")
		}
		for _, f := range files {
			if f.Path == "" {
				t.Error("generated file has empty path")
			}
			if len(f.Content) == 0 {
				t.Errorf("generated file %q has empty content", f.Path)
			}
		}
	})

	t.Run("SelfConsistency", func(t *testing.T) {
		if fixtures.PolicyInput == nil {
			t.Skip("PolicyInput not provided")
		}
		files, err := renderer.Render(context.Background(), fixtures.PolicyInput)
		if err != nil {
			t.Fatalf("Render() error: %v", err)
		}
		issues := renderer.Validate(context.Background(), files)
		for _, issue := range issues {
			if issue.Severity == aiframework.SeverityError {
				t.Errorf("self-validation error: %s: %s", issue.Path, issue.Message)
			}
		}
	})
}
