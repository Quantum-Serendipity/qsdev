package contracttest

import (
	"context"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
)

func TestToolAdapter(t *testing.T, adapter aiframework.ToolAdapter, fixtures ContractFixtures) {
	t.Helper()

	t.Run("EnforcementTierValid", func(t *testing.T) {
		tier := adapter.EnforcementTier()
		if tier.String() == "unknown" {
			t.Error("EnforcementTier() returned unknown tier")
		}
	})

	t.Run("TranslatePermissionsNonEmpty", func(t *testing.T) {
		policy := &aiframework.PermissionPolicy{
			DenyRules: []aiframework.PermissionRule{
				{Pattern: "Bash(rm -rf *)", Reason: "destructive"},
			},
		}
		artifacts, err := adapter.TranslatePermissions(context.Background(), policy)
		if err != nil {
			t.Fatalf("TranslatePermissions() error: %v", err)
		}
		if artifacts == nil {
			t.Fatal("TranslatePermissions() returned nil")
		}
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
		}
	})

	t.Run("CredentialInjectionNoLeak", func(t *testing.T) {
		scope := &aiframework.CredentialScope{
			SandboxFilters: aiframework.DefaultSandboxFilters(),
		}
		artifacts, err := adapter.InjectCredentials(context.Background(), scope)
		if err != nil {
			t.Fatalf("InjectCredentials() error: %v", err)
		}
		if artifacts == nil {
			t.Fatal("InjectCredentials() returned nil")
		}
		for _, f := range artifacts.GeneratedFiles {
			content := string(f.Content)
			for _, pattern := range scope.SandboxFilters {
				if strings.Contains(content, pattern) {
					t.Errorf("generated file %q contains credential pattern %q", f.Path, pattern)
				}
			}
		}
	})
}
