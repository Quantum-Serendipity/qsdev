package mcpserve

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/container"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// alwaysMountedTools are the security, devenv and status tools (Unit 32.9)
// mounted whatever tools.Options selects.
var alwaysMountedTools = []string{
	"qsdev_security_scan",
	"qsdev_policy_check",
	"qsdev_env_info",
	"qsdev_status",
	"qsdev_devenv_doctor",
}

// TestMountTools proves the security and devenv tools are registered on the
// underlying mcp-go server and therefore appear in tools/list, and that the
// opt-in tools appear only when tools.Options selects them (F247):
// qsdev_credential_vend needs an enabled security.credential_vend and
// qsdev_nix_run needs NixRun.
func TestMountTools(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		opts           tools.Options
		credentialVend bool
		nixRun         bool
	}{
		{name: "zero options mount neither opt-in tool", opts: tools.Options{}},
		{
			name:           "allow-lists without enabled mount no credential vending",
			opts:           tools.Options{CredentialVend: types.CredentialVendConfig{AWS: types.AWSCredentialVendConfig{AllowSessionToken: true}}},
			credentialVend: false,
		},
		{
			name:           "enabled credential vending and nix_run",
			opts:           tools.Options{CredentialVend: types.CredentialVendConfig{Enabled: true}, NixRun: true},
			credentialVend: true,
			nixRun:         true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			srv := New(WithProjectRoot(dir))
			srv.MountTools(tools.All(dir, nil, tt.opts))

			listed := srv.MCPServer().ListTools()
			for _, name := range alwaysMountedTools {
				if _, ok := listed[name]; !ok {
					t.Errorf("tool %q not mounted; mounted=%v", name, keys(listed))
				}
			}
			if _, ok := listed["qsdev_credential_vend"]; ok != tt.credentialVend {
				t.Errorf("qsdev_credential_vend mounted = %t, want %t", ok, tt.credentialVend)
			}
			if _, ok := listed["qsdev_nix_run"]; ok != tt.nixRun {
				t.Errorf("qsdev_nix_run mounted = %t, want %t", ok, tt.nixRun)
			}
		})
	}
}

// TestToolOptions checks the serve command's opt-in selection (F247):
// credential vending follows security.credential_vend only, and nix_run is off
// in gateway mode unless the operator opts in.
func TestToolOptions(t *testing.T) {
	t.Parallel()
	enabled := &types.QsdevConfig{Security: types.SecurityConfig{CredentialVend: types.CredentialVendConfig{
		Enabled: true,
		AWS:     types.AWSCredentialVendConfig{RoleARNs: []string{"arn:aws:iam::123456789012:role/dev"}},
	}}}
	tests := []struct {
		name          string
		cfg           *types.QsdevConfig
		mode          container.DeployMode
		gatewayNixRun bool
		wantVend      bool
		wantNixRun    bool
	}{
		{name: "native without config", mode: container.DeployNative, wantNixRun: true},
		{name: "native with opt-in", cfg: enabled, mode: container.DeployNative, wantVend: true, wantNixRun: true},
		{name: "standalone keeps nix_run", mode: container.DeployStandalone, wantNixRun: true},
		{name: "gateway drops nix_run by default", cfg: enabled, mode: container.DeployGateway, wantVend: true},
		{name: "gateway opt-in mounts nix_run", mode: container.DeployGateway, gatewayNixRun: true, wantNixRun: true},
		{name: "native ignores the gateway opt-in", mode: container.DeployNative, gatewayNixRun: true, wantNixRun: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := toolOptions(tt.cfg, tt.mode, tt.gatewayNixRun)
			if got.CredentialVend.Enabled != tt.wantVend {
				t.Errorf("CredentialVend.Enabled = %t, want %t", got.CredentialVend.Enabled, tt.wantVend)
			}
			if tt.wantVend && len(got.CredentialVend.AWS.RoleARNs) != 1 {
				t.Errorf("CredentialVend allow-lists not carried: %+v", got.CredentialVend)
			}
			if got.NixRun != tt.wantNixRun {
				t.Errorf("NixRun = %t, want %t", got.NixRun, tt.wantNixRun)
			}
		})
	}
}

// TestLoadProjectConfigReadsCredentialVend checks that the committed
// security.credential_vend reaches the serve command.
func TestLoadProjectConfigReadsCredentialVend(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	body := "version: 2\nsecurity:\n  credential_vend:\n    enabled: true\n    gcp:\n      service_accounts:\n        - ci@proj.iam.gserviceaccount.com\n"
	if err := os.WriteFile(filepath.Join(dir, ".qsdev.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadProjectConfig(dir)
	if err != nil {
		t.Fatalf("loadProjectConfig: %v", err)
	}
	opts := toolOptions(cfg, container.DeployNative, false)
	if !opts.CredentialVend.Enabled || len(opts.CredentialVend.GCP.ServiceAccounts) != 1 {
		t.Errorf("CredentialVend = %+v, want enabled with one service account", opts.CredentialVend)
	}
}
