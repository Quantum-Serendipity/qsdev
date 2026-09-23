package contracttest

import (
	"context"
	"slices"
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

	t.Run("RenderKeepsPolicyRules", func(t *testing.T) {
		if fixtures.PolicyInput == nil {
			t.Skip("PolicyInput not provided")
		}
		if !renderer.Capabilities().RendersPermissions {
			t.Skip("renderer does not render permissions")
		}
		// Add rules no preset could contain, so a renderer that drops the
		// policy's own rules cannot pass by coincidence.
		input := *fixtures.PolicyInput
		perms := aiframework.PermissionPolicy{}
		if input.Permissions != nil {
			perms = *input.Permissions
		}
		perms.AllowRules = append(slices.Clone(perms.AllowRules), aiframework.PermissionRule{Pattern: "Bash(contract-allow-marker *)"})
		perms.DenyRules = append(slices.Clone(perms.DenyRules), aiframework.PermissionRule{Pattern: "Bash(contract-deny-marker *)"})
		perms.AskRules = append(slices.Clone(perms.AskRules), aiframework.PermissionRule{Pattern: "Bash(contract-ask-marker *)"})
		input.Permissions = &perms

		files, err := renderer.Render(context.Background(), &input)
		if err != nil {
			t.Fatalf("Render() error: %v", err)
		}
		for _, rules := range [][]aiframework.PermissionRule{perms.AllowRules, perms.DenyRules, perms.AskRules} {
			for _, pattern := range aiframework.RulePatterns(rules) {
				if !filesContain(files, pattern) {
					t.Errorf("Render() dropped policy rule %q", pattern)
				}
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
