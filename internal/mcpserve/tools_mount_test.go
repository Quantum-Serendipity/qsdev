package mcpserve

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	qsdevcatalog "github.com/Quantum-Serendipity/qsdev/internal/catalog"
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

// vendEnabled is a committed security.credential_vend that enables the tool
// with one allow-listed role.
func vendEnabled() *types.QsdevConfig {
	return &types.QsdevConfig{Security: types.SecurityConfig{CredentialVend: types.CredentialVendConfig{
		Enabled: true,
		AWS:     types.AWSCredentialVendConfig{RoleARNs: []string{"arn:aws:iam::123456789012:role/dev"}},
	}}}
}

// envOf returns a getenv over m.
func envOf(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

// mountWithLog runs mountOptions with a captured logger and returns the
// options, the log text and the error.
func mountWithLog(t *testing.T, in mountInputs) (tools.Options, string, error) {
	t.Helper()
	var buf bytes.Buffer
	in.logger = slog.New(slog.NewTextHandler(&buf, nil))
	if in.getenv == nil {
		in.getenv = envOf(nil)
	}
	opts, err := mountOptions(in)
	return opts, buf.String(), err
}

// TestMountOptions is the serve command's opt-in table (F247, U21-WS2): each
// gated tool needs an operator opt-in (flag, env or the user defaults
// file's mcp_serve), credential vending also needs the committed allow-lists,
// and the gateway and standalone modes also need the --gateway-allow-* flag.
func TestMountOptions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		in         mountInputs
		wantVend   bool
		wantNixRun bool
	}{
		{name: "native without config", in: mountInputs{mode: container.DeployNative}},
		{name: "native with committed vend only", in: mountInputs{cfg: vendEnabled(), mode: container.DeployNative}},
		{
			name:     "native with committed vend and flag confirmation",
			in:       mountInputs{cfg: vendEnabled(), mode: container.DeployNative, flags: optInFlags{allowCredentialVend: true}},
			wantVend: true,
		},
		{
			name:       "native nix_run flag",
			in:         mountInputs{mode: container.DeployNative, flags: optInFlags{allowNixRun: true}},
			wantNixRun: true,
		},
		{name: "standalone without opt-in", in: mountInputs{mode: container.DeployStandalone}},
		{
			name: "standalone base opt-in without deployment flag",
			in:   mountInputs{mode: container.DeployStandalone, flags: optInFlags{allowNixRun: true}},
		},
		{
			name:       "standalone base and deployment flags",
			in:         mountInputs{mode: container.DeployStandalone, flags: optInFlags{allowNixRun: true, gatewayNixRun: true}},
			wantNixRun: true,
		},
		{
			name: "gateway with committed vend only",
			in:   mountInputs{cfg: vendEnabled(), mode: container.DeployGateway},
		},
		{
			name: "gateway deployment flag alone mounts nothing",
			in:   mountInputs{mode: container.DeployGateway, flags: optInFlags{gatewayNixRun: true}},
		},
		{
			name:       "gateway base and deployment flags",
			in:         mountInputs{mode: container.DeployGateway, flags: optInFlags{allowNixRun: true, gatewayNixRun: true}},
			wantNixRun: true,
		},
		{
			name: "native ignores a deployment flag alone",
			in:   mountInputs{mode: container.DeployNative, flags: optInFlags{gatewayNixRun: true}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, _, err := mountWithLog(t, tt.in)
			if err != nil {
				t.Fatalf("mountOptions: %v", err)
			}
			if got.CredentialVend.Enabled != tt.wantVend {
				t.Errorf("CredentialVend.Enabled = %t, want %t", got.CredentialVend.Enabled, tt.wantVend)
			}
			if tt.wantVend && len(got.CredentialVend.AWS.RoleARNs) != 1 {
				t.Errorf("CredentialVend allow-lists not carried: %+v", got.CredentialVend)
			}
			if !tt.wantVend && !got.CredentialVend.IsZero() {
				t.Errorf("unmounted CredentialVend still carries config: %+v", got.CredentialVend)
			}
			if got.NixRun != tt.wantNixRun {
				t.Errorf("NixRun = %t, want %t", got.NixRun, tt.wantNixRun)
			}
		})
	}
}

// writeProjectConfig writes body as dir's .qsdev.yaml.
func writeProjectConfig(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, ".qsdev.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestLoadProjectConfigReadsCredentialVend checks that the committed
// security.credential_vend reaches the serve command, that it alone does not
// mount the tool, and that an operator confirmation mounts it with the
// committed allow-lists.
func TestLoadProjectConfigReadsCredentialVend(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeProjectConfig(t, dir, "version: 2\nsecurity:\n  credential_vend:\n    enabled: true\n    gcp:\n      service_accounts:\n        - ci@proj.iam.gserviceaccount.com\n")
	cfg, err := loadProjectConfig(dir)
	if err != nil {
		t.Fatalf("loadProjectConfig: %v", err)
	}
	alone, _, err := mountWithLog(t, mountInputs{cfg: cfg, mode: container.DeployNative})
	if err != nil {
		t.Fatalf("mountOptions: %v", err)
	}
	if alone.CredentialVend.Enabled {
		t.Error("committed security.credential_vend alone mounted qsdev_credential_vend")
	}
	confirmed, _, err := mountWithLog(t, mountInputs{
		cfg: cfg, mode: container.DeployNative, user: qsdevcatalog.MCPServeOptIns{AllowCredentialVend: true},
	})
	if err != nil {
		t.Fatalf("mountOptions: %v", err)
	}
	if !confirmed.CredentialVend.Enabled || len(confirmed.CredentialVend.GCP.ServiceAccounts) != 1 {
		t.Errorf("CredentialVend = %+v, want enabled with one service account", confirmed.CredentialVend)
	}
}

// TestLoadProjectConfig_CommittedOptInKeysFailClosed (F7): the operator
// opt-ins have no key in the committed .qsdev.yaml. A file that tries one is
// rejected by the strict decoder, so the server refuses to start instead of
// serving.
func TestLoadProjectConfig_CommittedOptInKeysFailClosed(t *testing.T) {
	t.Parallel()
	for _, body := range []string{
		"version: 2\nmcp:\n  credential_vend:\n    enabled: true\n",
		"version: 2\nmcp:\n  nix_run:\n    enabled: true\n",
		"version: 2\nmcp:\n  allow_nix_run: true\n",
		"version: 2\nmcp_serve:\n  allow_nix_run: true\n",
	} {
		dir := t.TempDir()
		writeProjectConfig(t, dir, body)
		if cfg, err := loadProjectConfig(dir); err == nil {
			t.Errorf("loadProjectConfig(%q) = %+v, want an error", body, cfg)
		}
	}
}
