package check

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/cloudcommon"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// writeCloudSettings writes a .claude/settings.json whose permissions.deny
// holds deny.
func writeCloudSettings(t *testing.T, root string, deny []string) {
	t.Helper()
	doc := map[string]any{"permissions": map[string]any{"deny": deny}}
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".claude", "settings.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// generatedCloudDeny returns the deny rules the settings generator emits for
// provider: its Bash rules plus a Read rule per credential path.
func generatedCloudDeny(provider cloudcommon.CloudProvider) []string {
	rules := cloudcommon.BashDenyRules(provider)
	for _, p := range cloudcommon.ReadDenyPaths(provider) {
		if base, ok := strings.CutSuffix(p, "/*"); ok {
			p = base + "/**"
		}
		rules = append(rules, "Read("+p+")")
	}
	return rules
}

func TestCheckCloudIsolation(t *testing.T) {
	t.Parallel()

	awsConfig := &types.QsdevConfig{Languages: []types.LanguageConfig{{Name: "golang"}, {Name: "aws"}}}
	claudeEnabled := true
	awsClaudeConfig := &types.QsdevConfig{
		Languages:  awsConfig.Languages,
		ClaudeCode: types.ClaudeCodeConfig{Enabled: &claudeEnabled},
	}
	fullDeny := generatedCloudDeny(cloudcommon.AWS)

	type want struct {
		status   CheckStatus
		severity CheckSeverity
	}
	tests := []struct {
		name     string
		cfg      *types.QsdevConfig
		deny     []string // nil: no settings file
		settings string   // raw settings.json overriding deny
		env      map[string]string
		envErr   error
		want     map[string]want // by result name; nil means no results
	}{
		{
			name: "no config",
			deny: fullDeny,
		},
		{
			name: "no cloud provider",
			cfg:  &types.QsdevConfig{Languages: []types.LanguageConfig{{Name: "golang"}}},
			deny: fullDeny,
		},
		{
			name: "fully isolated",
			cfg:  awsConfig,
			deny: fullDeny,
			env:  map[string]string{"AWS_PROFILE": "dev-project"},
			want: map[string]want{
				"cloud_isolation_aws_environment_separation":  {StatusPass, SeverityInfo},
				"cloud_isolation_aws_credential_file_masking": {StatusPass, SeverityInfo},
				"cloud_isolation_aws_agent_deny_rules":        {StatusPass, SeverityInfo},
			},
		},
		{
			name: "placeholder profile warns",
			cfg:  awsConfig,
			deny: fullDeny,
			env:  map[string]string{"AWS_PROFILE": "PLACEHOLDER -- set your AWS profile"},
			want: map[string]want{
				"cloud_isolation_aws_environment_separation":  {StatusWarn, SeverityLow},
				"cloud_isolation_aws_credential_file_masking": {StatusPass, SeverityInfo},
				"cloud_isolation_aws_agent_deny_rules":        {StatusPass, SeverityInfo},
			},
		},
		{
			name:   "unreadable devenv module still warns",
			cfg:    awsConfig,
			deny:   fullDeny,
			envErr: errors.New("parsing devenv.local.nix: bad"),
			want: map[string]want{
				"cloud_isolation_aws_environment_separation":  {StatusWarn, SeverityLow},
				"cloud_isolation_aws_credential_file_masking": {StatusPass, SeverityInfo},
				"cloud_isolation_aws_agent_deny_rules":        {StatusPass, SeverityInfo},
			},
		},
		{
			name: "removed cloud rules fail high",
			cfg:  awsConfig,
			deny: []string{"Bash(rm -rf /)"},
			env:  map[string]string{"AWS_PROFILE": "dev-project"},
			want: map[string]want{
				"cloud_isolation_aws_environment_separation":  {StatusPass, SeverityInfo},
				"cloud_isolation_aws_credential_file_masking": {StatusFail, SeverityHigh},
				"cloud_isolation_aws_agent_deny_rules":        {StatusFail, SeverityHigh},
			},
		},
		{
			name: "missing settings file fails when Claude Code is enabled",
			cfg:  awsClaudeConfig,
			want: map[string]want{
				"cloud_isolation_aws_environment_separation":  {StatusWarn, SeverityLow},
				"cloud_isolation_aws_credential_file_masking": {StatusFail, SeverityHigh},
				"cloud_isolation_aws_agent_deny_rules":        {StatusFail, SeverityHigh},
			},
		},
		{
			name: "no Claude Code and no settings file judges the environment only",
			cfg:  awsConfig,
			env:  map[string]string{"AWS_PROFILE": "dev-project"},
			want: map[string]want{
				"cloud_isolation_aws_environment_separation": {StatusPass, SeverityInfo},
			},
		},
		{
			name:     "malformed settings fails",
			cfg:      awsConfig,
			settings: `{"permissions":`,
			want: map[string]want{
				"cloud_isolation_settings": {StatusFail, SeverityHigh},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			switch {
			case tt.settings != "":
				if err := os.MkdirAll(filepath.Join(root, ".claude"), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, ".claude", "settings.json"), []byte(tt.settings), 0o644); err != nil {
					t.Fatal(err)
				}
			case tt.deny != nil:
				writeCloudSettings(t, root, tt.deny)
			}
			ctx := CheckContext{ProjectRoot: root, QsdevConfig: tt.cfg, DeclaredEnv: tt.env, DeclaredEnvErr: tt.envErr}
			results := CheckCloudIsolation(ctx)
			if len(results) != len(tt.want) {
				t.Fatalf("got %d results, want %d: %+v", len(results), len(tt.want), results)
			}
			for _, r := range results {
				w, ok := tt.want[r.Name]
				if !ok {
					t.Errorf("unexpected result %q", r.Name)
					continue
				}
				if r.Status != w.status || r.Severity != w.severity {
					t.Errorf("%s = %s/%s, want %s/%s (%s)", r.Name, r.Status, r.Severity, w.status, w.severity, r.Message)
				}
				if r.Category != CategorySecurityHarden {
					t.Errorf("%s category = %s", r.Name, r.Category)
				}
				if r.Status != StatusPass && r.Remediation == "" {
					t.Errorf("%s has no remediation", r.Name)
				}
				if tt.envErr != nil && r.Status == StatusWarn && !strings.Contains(r.Message, "could not be read") {
					t.Errorf("%s message %q does not mention the unreadable module", r.Name, r.Message)
				}
			}
		})
	}
}

func TestRunAllChecks_IncludesCloudIsolation(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeCloudSettings(t, root, nil)
	ctx := CheckContext{
		ProjectRoot: root,
		QsdevConfig: &types.QsdevConfig{Languages: []types.LanguageConfig{{Name: "azure"}}},
	}
	report := RunAllChecks(ctx)
	found := false
	for _, r := range report.Checks {
		if r.Name == "cloud_isolation_azure_agent_deny_rules" {
			found = true
			if r.Status != StatusFail {
				t.Errorf("azure deny rules status = %s, want fail", r.Status)
			}
		}
	}
	if !found {
		t.Error("RunAllChecks did not run the cloud isolation check")
	}
	if !ShouldFail(report.Checks, AuditLevelMedium) {
		t.Error("missing cloud deny rules did not fail the run at the default audit level")
	}
}
