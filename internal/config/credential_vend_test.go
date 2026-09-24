package config

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// credentialVendFixture is a fully populated security.credential_vend block.
func credentialVendFixture() types.CredentialVendConfig {
	return types.CredentialVendConfig{
		Enabled: true,
		AWS:     types.AWSCredentialVendConfig{RoleARNs: []string{"arn:aws:iam::123456789012:role/ci/deploy"}},
		GCP:     types.GCPCredentialVendConfig{ServiceAccounts: []string{"ci@proj.iam.gserviceaccount.com"}},
		Azure: types.AzureCredentialVendConfig{
			Scopes:     []string{"https://storage.azure.com/.default"},
			Identities: []string{"00000000-0000-0000-0000-000000000001"},
		},
	}
}

// TestParseCredentialVend checks the strict decoder accepts the
// security.credential_vend block and that an absent block leaves it disabled
// (F247: credential vending is opt-in).
func TestParseCredentialVend(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		data string
		want types.CredentialVendConfig
	}{
		{name: "absent is disabled", data: "version: 2\n"},
		{
			name: "full block",
			data: "version: 2\nsecurity:\n  credential_vend:\n    enabled: true\n" +
				"    aws:\n      role_arns: [\"arn:aws:iam::123456789012:role/ci/deploy\"]\n" +
				"    gcp:\n      service_accounts: [ci@proj.iam.gserviceaccount.com]\n" +
				"    azure:\n      scopes: [\"https://storage.azure.com/.default\"]\n" +
				"      identities: [00000000-0000-0000-0000-000000000001]\n",
			want: credentialVendFixture(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg, err := ParseQsdevConfigBytes([]byte(tt.data))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if !reflect.DeepEqual(cfg.Security.CredentialVend, tt.want) {
				t.Errorf("credential_vend = %+v, want %+v", cfg.Security.CredentialVend, tt.want)
			}
		})
	}
}

// TestMarshalOmitsUnsetCredentialVend checks a project that never opted in
// writes no credential_vend key.
func TestMarshalOmitsUnsetCredentialVend(t *testing.T) {
	t.Parallel()
	data, err := MarshalProjectConfig(types.QsdevConfig{Version: 2, Security: types.SecurityConfig{Level: "standard"}})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(data), "credential_vend") {
		t.Errorf("marshaled config mentions credential_vend:\n%s", data)
	}
}

// TestValidateCredentialVend checks each allow-list entry has the shape the
// tool matches exactly, so a wildcard or typo fails `qsdev check` instead of
// silently allowing nothing.
func TestValidateCredentialVend(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		mutate     func(*types.CredentialVendConfig)
		wantFields []string
	}{
		{name: "valid", mutate: func(*types.CredentialVendConfig) {}},
		{name: "govcloud role", mutate: func(c *types.CredentialVendConfig) {
			c.AWS.RoleARNs = []string{"arn:aws-us-gov:iam::123456789012:role/dev"}
		}},
		{name: "wildcard role", mutate: func(c *types.CredentialVendConfig) {
			c.AWS.RoleARNs = []string{"arn:aws:iam::123456789012:role/*"}
		}, wantFields: []string{"security.credential_vend.aws.role_arns"}},
		{name: "user not role", mutate: func(c *types.CredentialVendConfig) {
			c.AWS.RoleARNs = []string{"arn:aws:iam::123456789012:user/alice"}
		}, wantFields: []string{"security.credential_vend.aws.role_arns"}},
		{name: "service account with URL syntax", mutate: func(c *types.CredentialVendConfig) {
			c.GCP.ServiceAccounts = []string{"x@y.iam.gserviceaccount.com:signJwt#"}
		}, wantFields: []string{"security.credential_vend.gcp.service_accounts"}},
		{name: "scope with whitespace", mutate: func(c *types.CredentialVendConfig) {
			c.Azure.Scopes = []string{"https://storage.azure.com/.default "}
		}, wantFields: []string{"security.credential_vend.azure.scopes"}},
		{name: "identity not a GUID", mutate: func(c *types.CredentialVendConfig) {
			c.Azure.Identities = []string{"my-identity"}
		}, wantFields: []string{"security.credential_vend.azure.identities"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := types.QsdevConfig{Version: 2, Security: types.SecurityConfig{CredentialVend: credentialVendFixture()}}
			tt.mutate(&cfg.Security.CredentialVend)
			var fields []string
			for _, e := range ValidateQsdevConfig(&cfg, ValidateOptions{}) {
				fields = append(fields, e.Field)
			}
			if !slices.Equal(fields, tt.wantFields) {
				t.Errorf("error fields = %v, want %v", fields, tt.wantFields)
			}
		})
	}
}

// TestCredentialVendSurvivesResolveAndRecreate checks the resolver keeps the
// committed block as its own copy and `init --force` carries it over, while a
// local overlay cannot opt in or widen it.
func TestCredentialVendSurvivesResolveAndRecreate(t *testing.T) {
	t.Parallel()
	committed := &types.QsdevConfig{Version: 2, Security: types.SecurityConfig{CredentialVend: credentialVendFixture()}}

	resolved, err := ResolveConfig(&types.QsdevConfig{}, committed, nil)
	if err != nil {
		t.Fatalf("ResolveConfig: %v", err)
	}
	if !reflect.DeepEqual(resolved.Config.Security.CredentialVend, credentialVendFixture()) {
		t.Errorf("resolved credential_vend = %+v", resolved.Config.Security.CredentialVend)
	}
	resolved.Config.Security.CredentialVend.AWS.RoleARNs[0] = "changed"
	if committed.Security.CredentialVend.AWS.RoleARNs[0] == "changed" {
		t.Error("resolved config aliases the committed allow-list")
	}

	fresh := types.QsdevConfig{Version: 2}
	PreserveCommittedPolicy(&fresh, committed)
	if !reflect.DeepEqual(fresh.Security.CredentialVend, credentialVendFixture()) {
		t.Errorf("re-created credential_vend = %+v, want the committed block", fresh.Security.CredentialVend)
	}
}

// TestLocalCannotConfigureCredentialVend checks .qsdev.local.yaml cannot opt a
// project into credential vending or widen its allow-lists: the setting is
// dropped with a FloorViolation.
func TestLocalCannotConfigureCredentialVend(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		committed types.CredentialVendConfig
	}{
		{name: "opt in"},
		{name: "widen", committed: types.CredentialVendConfig{Enabled: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			project := &types.QsdevConfig{Version: 2, Security: types.SecurityConfig{CredentialVend: tt.committed}}
			local := &LocalConfig{Security: types.SecurityConfig{CredentialVend: credentialVendFixture()}}
			resolved, err := ResolveConfig(&types.QsdevConfig{}, project, local)
			if err != nil {
				t.Fatalf("ResolveConfig: %v", err)
			}
			if !reflect.DeepEqual(resolved.Config.Security.CredentialVend, tt.committed) {
				t.Errorf("credential_vend = %+v, want the committed %+v", resolved.Config.Security.CredentialVend, tt.committed)
			}
			if !slices.ContainsFunc(resolved.Violations, func(v FloorViolation) bool { return v.Field == "security.credential_vend" }) {
				t.Errorf("violations = %+v, want one for security.credential_vend", resolved.Violations)
			}
		})
	}
}
