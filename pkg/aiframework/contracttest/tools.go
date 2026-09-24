package contracttest

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestToolAdapter(t *testing.T, adapter aiframework.ToolAdapter, fixtures ContractFixtures) {
	t.Helper()

	t.Run("EnforcementTierValid", func(t *testing.T) {
		tier := adapter.EnforcementTier()
		if _, err := tier.MarshalText(); err != nil {
			t.Errorf("EnforcementTier() returned invalid tier: %v", err)
		}
	})

	t.Run("TranslatePermissionsEmitsDenyRules", func(t *testing.T) {
		const deny = "Bash(rm -rf *)"
		policy := &aiframework.PermissionPolicy{
			DenyRules: []aiframework.PermissionRule{{Pattern: deny, Reason: "destructive"}},
		}
		artifacts, err := adapter.TranslatePermissions(context.Background(), policy)
		if err != nil {
			t.Fatalf("TranslatePermissions() error: %v", err)
		}
		if artifacts == nil {
			t.Fatal("TranslatePermissions() returned nil")
			return
		}
		if len(artifacts.GeneratedFiles) == 0 {
			t.Fatal("TranslatePermissions() produced no files")
		}
		if _, err := artifacts.ActiveTier.MarshalText(); err != nil {
			t.Errorf("TranslatePermissions() left ActiveTier invalid: %v", err)
		}
		if !filesContain(artifacts.GeneratedFiles, deny) {
			t.Errorf("TranslatePermissions() dropped deny rule %q", deny)
		}
	})

	t.Run("TranslateIgnorePatternsRendered", func(t *testing.T) {
		renderer, ok := adapter.(aiframework.ConfigRenderer)
		if !ok || !renderer.Capabilities().RendersIgnore {
			t.Skip("adapter does not claim to render ignore patterns")
		}
		const pattern = "./contract-ignored-marker/**"
		files, err := adapter.TranslateIgnorePatterns(context.Background(), []aiframework.IgnorePattern{
			{Pattern: pattern, Category: aiframework.CategoryBinary},
		})
		if err != nil {
			t.Fatalf("TranslateIgnorePatterns() error: %v", err)
		}
		if !filesContain(files, pattern) {
			t.Errorf("TranslateIgnorePatterns() did not render %q despite RendersIgnore", pattern)
		}
	})

	t.Run("NilPolicyTolerated", func(t *testing.T) {
		if _, err := adapter.TranslatePermissions(context.Background(), nil); err != nil {
			t.Errorf("TranslatePermissions(nil) error: %v", err)
		}
		// Must not panic.
		_ = adapter.ReportGaps(context.Background(), nil)
	})

	t.Run("ReportGapsForDenyRules", func(t *testing.T) {
		policy := &aiframework.PermissionPolicy{
			DenyRules: []aiframework.PermissionRule{
				{Pattern: "Bash(rm -rf *)", Reason: "destructive"},
			},
		}
		gaps := adapter.ReportGaps(context.Background(), policy)
		if len(gaps) == 0 {
			t.Error("ReportGaps() returned no gaps for non-empty policy")
		}
		for _, g := range gaps {
			if g.Description == "" {
				t.Error("gap has empty Description")
			}
			if g.Mitigation == "" {
				t.Error("gap has empty Mitigation")
			}
			if _, err := g.RequiredTier.MarshalText(); err != nil {
				t.Errorf("gap %q has invalid RequiredTier: %v", g.Description, err)
			}
			if _, err := g.ActualTier.MarshalText(); err != nil {
				t.Errorf("gap %q has invalid ActualTier: %v", g.Description, err)
			}
		}
	})

	t.Run("CredentialInjectionNoLeak", func(t *testing.T) {
		// PATH is always set, so its value is a canary: an adapter that
		// resolves required variables into artifacts, instead of passing them
		// through by reference, embeds this value somewhere.
		const canaryVar = "PATH"
		canary := os.Getenv(canaryVar)
		scope := &aiframework.CredentialScope{
			APIKeys:        []aiframework.APIKeyRequirement{{Provider: "contract", EnvVar: canaryVar, Required: true}},
			SandboxFilters: aiframework.DefaultSandboxFilters(),
		}
		artifacts, err := adapter.InjectCredentials(context.Background(), scope)
		if err != nil {
			t.Fatalf("InjectCredentials() error: %v", err)
		}
		if artifacts == nil {
			t.Fatal("InjectCredentials() returned nil")
			return
		}
		for _, f := range artifacts.GeneratedFiles {
			content := string(f.Content)
			for _, pattern := range scope.SandboxFilters {
				if strings.Contains(content, pattern) {
					t.Errorf("generated file %q contains credential pattern %q", f.Path, pattern)
				}
			}
		}
		if len(canary) < 16 {
			return // too short to be a reliable canary
		}
		if filesContain(artifacts.GeneratedFiles, canary) {
			t.Errorf("generated files embed the value of %s instead of a reference", canaryVar)
		}
		for k, v := range artifacts.EnvVars {
			if strings.Contains(v, canary) {
				t.Errorf("EnvVars[%q] embeds the value of %s instead of a reference", k, canaryVar)
			}
		}
	})
}

// filesContain reports whether any generated file's content contains s.
func filesContain(files []types.GeneratedFile, s string) bool {
	for _, f := range files {
		if strings.Contains(string(f.Content), s) {
			return true
		}
	}
	return false
}
