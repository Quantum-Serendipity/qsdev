package mcpserve

import (
	"path/filepath"
	"strings"
	"testing"

	qsdevcatalog "github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/container"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestResolveBindHost(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, flag, env, want string
	}{
		{"flag wins", "0.0.0.0", "10.0.0.1", "0.0.0.0"},
		{"env fallback", "", "10.0.0.1", "10.0.0.1"},
		{"default loopback", "", "", defaultBindHost},
		{"trims blanks", "  ", "  ", defaultBindHost},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := resolveBindHost(c.flag, c.env); got != c.want {
				t.Errorf("resolveBindHost(%q,%q) = %q, want %q", c.flag, c.env, got, c.want)
			}
		})
	}
}

func TestIsLoopbackHost(t *testing.T) {
	t.Parallel()
	cases := []struct {
		host string
		want bool
	}{
		{"127.0.0.1", true},
		{"127.0.0.5", true},
		{"::1", true},
		{"localhost", true},
		{"0.0.0.0", false},
		{"192.168.1.10", false},
		{"::", false},
		{"", false}, // all-interfaces / unknown => fail-closed
		{"example.com", false},
	}
	for _, c := range cases {
		t.Run(c.host, func(t *testing.T) {
			t.Parallel()
			if got := isLoopbackHost(c.host); got != c.want {
				t.Errorf("isLoopbackHost(%q) = %v, want %v", c.host, got, c.want)
			}
		})
	}
}

// TestValidateServeSecurity covers the fail-closed network-exposure rules: the
// loopback default permits plain HTTP, while a gateway / non-loopback bind / any
// network exposure without complete mTLS material is refused before serving.
func TestValidateServeSecurity(t *testing.T) {
	t.Parallel()

	full := TLSMaterial{CertFile: "c", KeyFile: "k", ClientCAFile: "ca"}
	partial := TLSMaterial{CertFile: "c"}
	var none TLSMaterial

	cases := []struct {
		name        string
		mode        container.DeployMode
		t           Transport
		bind        string
		material    TLSMaterial
		requireAuth bool
		wantErr     bool
	}{
		{"native stdio no tls ok", container.DeployNative, TransportStdio, "127.0.0.1", none, false, false},
		{"gateway stdio no allowlist ok", container.DeployGateway, TransportStdio, "127.0.0.1", none, false, false},
		// Gateway allow-list authorization cannot be enforced over the self-asserted
		// stdio identity, so the combination is refused at startup.
		{"gateway stdio with allowlist refused", container.DeployGateway, TransportStdio, "127.0.0.1", none, true, true},
		{"loopback http no tls ok", container.DeployNative, TransportHTTP, "127.0.0.1", none, false, false},
		{"loopback http with tls ok", container.DeployNative, TransportHTTP, "127.0.0.1", full, false, false},
		{"gateway http no tls refused", container.DeployGateway, TransportHTTP, "127.0.0.1", none, false, true},
		{"gateway http with tls ok", container.DeployGateway, TransportHTTP, "127.0.0.1", full, false, false},
		// Allow-list over an authenticating (mTLS) transport is exactly the supported case.
		{"gateway http with tls and allowlist ok", container.DeployGateway, TransportHTTP, "127.0.0.1", full, true, false},
		{"non-loopback http no tls refused", container.DeployNative, TransportHTTP, "0.0.0.0", none, false, true},
		{"non-loopback http with tls ok", container.DeployNative, TransportHTTP, "0.0.0.0", full, false, false},
		{"standalone loopback no tls ok", container.DeployStandalone, TransportStdio, "127.0.0.1", none, false, false},
		{"standalone non-loopback no tls refused", container.DeployStandalone, TransportStdio, "10.0.0.1", none, false, true},
		{"partial material refused", container.DeployNative, TransportHTTP, "127.0.0.1", partial, false, true},
		{"non-loopback stdio native exempt", container.DeployNative, TransportStdio, "0.0.0.0", none, false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			err := validateServeSecurity(c.mode, c.t, c.bind, c.material, c.requireAuth)
			if c.wantErr && err == nil {
				t.Fatalf("validateServeSecurity(%s,%s,%q) = nil, want error", c.mode, c.t, c.bind)
			}
			if !c.wantErr && err != nil {
				t.Fatalf("validateServeSecurity(%s,%s,%q) = %v, want nil", c.mode, c.t, c.bind, err)
			}
		})
	}
}

// TestMountOptions_NixRunOffByDefault (U21-02): qsdev_nix_run starts a process
// on the host, so no mode mounts it without an operator opt-in, and each of
// the three opt-in sources mounts it in native mode.
func TestMountOptions_NixRunOffByDefault(t *testing.T) {
	t.Parallel()
	empty := &types.QsdevConfig{}
	for _, mode := range []container.DeployMode{container.DeployNative, container.DeployStandalone, container.DeployGateway} {
		for _, cfg := range []*types.QsdevConfig{nil, empty} {
			got, log, err := mountWithLog(t, mountInputs{cfg: cfg, mode: mode})
			if err != nil {
				t.Fatalf("%s: mountOptions: %v", mode, err)
			}
			if got.NixRun {
				t.Errorf("%s (cfg %v): NixRun = true without an opt-in", mode, cfg)
			}
			if !strings.Contains(log, "qsdev_nix_run not mounted") {
				t.Errorf("%s: no not-mounted decision logged:\n%s", mode, log)
			}
		}
	}

	sources := []struct {
		name string
		in   mountInputs
	}{
		{"flag", mountInputs{flags: optInFlags{allowNixRun: true}}},
		{"env", mountInputs{getenv: envOf(map[string]string{envAllowNixRun: "1"})}},
		{"user overlay", mountInputs{user: qsdevcatalog.MCPServeOptIns{AllowNixRun: true}}},
	}
	for _, src := range sources {
		in := src.in
		in.mode = container.DeployNative
		got, log, err := mountWithLog(t, in)
		if err != nil {
			t.Fatalf("%s: mountOptions: %v", src.name, err)
		}
		if !got.NixRun {
			t.Errorf("%s: NixRun = false, want true", src.name)
		}
		if !strings.Contains(log, "qsdev_nix_run mounted") {
			t.Errorf("%s: no mounted decision logged:\n%s", src.name, log)
		}
	}
}

// TestMountOptions_StandaloneNeedsDeploymentFlag: standalone and gateway
// serve other hosts' clients, so a base opt-in is not enough there; the
// deployment flag or its env var is needed too, and alone it mounts nothing.
func TestMountOptions_StandaloneNeedsDeploymentFlag(t *testing.T) {
	t.Parallel()
	for _, mode := range []container.DeployMode{container.DeployStandalone, container.DeployGateway} {
		base, log, _ := mountWithLog(t, mountInputs{mode: mode, flags: optInFlags{allowNixRun: true}})
		if base.NixRun {
			t.Errorf("%s: base opt-in alone mounted qsdev_nix_run", mode)
		}
		if !strings.Contains(log, "--gateway-allow-nix-run") {
			t.Errorf("%s: log does not name the deployment flag:\n%s", mode, log)
		}
		deploy, _, _ := mountWithLog(t, mountInputs{mode: mode, flags: optInFlags{gatewayNixRun: true}})
		if deploy.NixRun {
			t.Errorf("%s: deployment flag alone mounted qsdev_nix_run", mode)
		}
		both, _, _ := mountWithLog(t, mountInputs{
			mode:   mode,
			user:   qsdevcatalog.MCPServeOptIns{AllowNixRun: true},
			getenv: envOf(map[string]string{envGatewayNixRun: "true"}),
		})
		if !both.NixRun {
			t.Errorf("%s: base opt-in plus deployment env did not mount qsdev_nix_run", mode)
		}
	}

	vend, _, err := mountWithLog(t, mountInputs{
		cfg: vendEnabled(), mode: container.DeployStandalone,
		flags: optInFlags{allowCredentialVend: true},
	})
	if err != nil {
		t.Fatalf("mountOptions: %v", err)
	}
	if vend.CredentialVend.Enabled {
		t.Error("standalone mounted qsdev_credential_vend without --gateway-allow-credential-vend")
	}
	vend, _, err = mountWithLog(t, mountInputs{
		cfg: vendEnabled(), mode: container.DeployStandalone,
		flags: optInFlags{allowCredentialVend: true, gatewayCredentialVend: true},
	})
	if err != nil {
		t.Fatalf("mountOptions: %v", err)
	}
	if !vend.CredentialVend.Enabled {
		t.Error("standalone with confirmation and deployment flag did not mount qsdev_credential_vend")
	}
}

// TestMountOptions_CredentialVendNeedsOperatorConfirm (U21-V02): the committed
// config supplies the allow-lists but cannot turn the tool on by itself; an
// operator confirmation from a flag, the env or the user defaults file does.
func TestMountOptions_CredentialVendNeedsOperatorConfirm(t *testing.T) {
	t.Parallel()
	got, log, err := mountWithLog(t, mountInputs{cfg: vendEnabled(), mode: container.DeployNative})
	if err != nil {
		t.Fatalf("mountOptions: %v", err)
	}
	if got.CredentialVend.Enabled {
		t.Error("committed config alone mounted qsdev_credential_vend")
	}
	if !strings.Contains(log, "level=WARN") {
		t.Errorf("no warning logged for committed config without confirmation:\n%s", log)
	}
	for _, want := range []string{"--allow-credential-vend", envAllowCredentialVend, "mcp_serve.allow_credential_vend"} {
		if !strings.Contains(log, want) {
			t.Errorf("warning does not name %s:\n%s", want, log)
		}
	}

	confirms := []struct {
		name string
		in   mountInputs
	}{
		{"flag", mountInputs{flags: optInFlags{allowCredentialVend: true}}},
		{"env", mountInputs{getenv: envOf(map[string]string{envAllowCredentialVend: "yes"})}},
		{"user overlay", mountInputs{user: qsdevcatalog.MCPServeOptIns{AllowCredentialVend: true}}},
	}
	for _, c := range confirms {
		in := c.in
		in.mode = container.DeployNative
		in.cfg = vendEnabled()
		got, log, err := mountWithLog(t, in)
		if err != nil {
			t.Fatalf("%s: mountOptions: %v", c.name, err)
		}
		if !got.CredentialVend.Enabled || len(got.CredentialVend.AWS.RoleARNs) != 1 {
			t.Errorf("%s: CredentialVend = %+v, want enabled with the committed allow-lists", c.name, got.CredentialVend)
		}
		if !strings.Contains(log, "qsdev_credential_vend mounted") {
			t.Errorf("%s: no mounted decision logged:\n%s", c.name, log)
		}

		// The same confirmation without committed allow-lists mounts nothing.
		in.cfg = nil
		got, _, err = mountWithLog(t, in)
		if err != nil {
			t.Fatalf("%s without config: mountOptions: %v", c.name, err)
		}
		if !got.CredentialVend.IsZero() {
			t.Errorf("%s without config: CredentialVend = %+v, want zero", c.name, got.CredentialVend)
		}
	}
}

// TestMountOptions_GatewayCredentialVendNeedsFlagAndAgents (U21-07): the
// gateway vends credentials only with the deployment flag, and the flag
// without an agent allow-list is a startup error rather than a gateway that
// vends to any caller.
func TestMountOptions_GatewayCredentialVendNeedsFlagAndAgents(t *testing.T) {
	t.Parallel()
	confirmed := optInFlags{allowCredentialVend: true}
	got, log, err := mountWithLog(t, mountInputs{cfg: vendEnabled(), mode: container.DeployGateway, flags: confirmed})
	if err != nil {
		t.Fatalf("mountOptions: %v", err)
	}
	if got.CredentialVend.Enabled {
		t.Error("gateway mounted qsdev_credential_vend without --gateway-allow-credential-vend")
	}
	if !strings.Contains(log, "--gateway-allow-credential-vend") {
		t.Errorf("log does not name the deployment flag:\n%s", log)
	}

	withFlag := optInFlags{allowCredentialVend: true, gatewayCredentialVend: true}
	for _, agents := range []string{"", " , "} {
		_, _, err = mountWithLog(t, mountInputs{
			cfg: vendEnabled(), mode: container.DeployGateway, flags: withFlag,
			getenv: envOf(map[string]string{envGatewayAgents: agents}),
		})
		if err == nil || !strings.Contains(err.Error(), envGatewayAgents) {
			t.Errorf("agents %q: error = %v, want a startup error naming %s", agents, err, envGatewayAgents)
		}
	}
	_, _, err = mountWithLog(t, mountInputs{
		mode: container.DeployGateway, getenv: envOf(map[string]string{envGatewayCredentialVend: "1"}),
	})
	if err == nil {
		t.Error("deployment env without agents: no startup error")
	}

	got, _, err = mountWithLog(t, mountInputs{
		cfg: vendEnabled(), mode: container.DeployGateway, flags: withFlag,
		getenv: envOf(map[string]string{envGatewayAgents: "claude-code"}),
	})
	if err != nil {
		t.Fatalf("mountOptions with agents: %v", err)
	}
	if !got.CredentialVend.Enabled {
		t.Error("gateway with confirmation, flag and agents did not mount qsdev_credential_vend")
	}
}

// TestMountOptions_HTTPNoAuthRefusesGatedTools (U21-04): --http-no-auth serves
// plain HTTP that any local process can call, so neither gated tool is mounted
// there whatever the opt-ins say, and the refusal is logged as a warning.
func TestMountOptions_HTTPNoAuthRefusesGatedTools(t *testing.T) {
	t.Parallel()
	all := optInFlags{allowNixRun: true, allowCredentialVend: true, gatewayNixRun: true, gatewayCredentialVend: true}
	for _, mode := range []container.DeployMode{container.DeployNative, container.DeployStandalone} {
		got, log, err := mountWithLog(t, mountInputs{cfg: vendEnabled(), mode: mode, flags: all, unauthenticatedHTTP: true})
		if err != nil {
			t.Fatalf("%s: mountOptions: %v", mode, err)
		}
		if got.NixRun || !got.CredentialVend.IsZero() {
			t.Errorf("%s: unauthenticated HTTP mounted a gated tool: %+v", mode, got)
		}
		for _, want := range []string{"level=WARN", flagHTTPNoAuth, "qsdev_nix_run not mounted", "qsdev_credential_vend not mounted"} {
			if !strings.Contains(log, want) {
				t.Errorf("%s: log does not contain %q:\n%s", mode, want, log)
			}
		}

		// The same opt-ins with authentication mount both.
		got, _, err = mountWithLog(t, mountInputs{cfg: vendEnabled(), mode: mode, flags: all})
		if err != nil {
			t.Fatalf("%s authenticated: mountOptions: %v", mode, err)
		}
		if !got.NixRun || !got.CredentialVend.Enabled {
			t.Errorf("%s authenticated: gated tools not mounted: %+v", mode, got)
		}
	}
}

// TestResolveHTTPToken: a bearer token is issued exactly when the server
// listens on plain HTTP without --http-no-auth; stdio and mTLS need none, and
// --http-token-file chooses where it is written.
func TestResolveHTTPToken(t *testing.T) {
	t.Parallel()
	custom := filepath.Join(t.TempDir(), "custom.token")
	cases := []struct {
		name     string
		mode     container.DeployMode
		t        Transport
		mtls     bool
		opts     serveOptions
		wantTok  bool
		wantPath string
	}{
		{"stdio", container.DeployNative, TransportStdio, false, serveOptions{}, false, ""},
		{"native http", container.DeployNative, TransportHTTP, false, serveOptions{httpTokenFile: custom}, true, custom},
		{"standalone", container.DeployStandalone, TransportStdio, false, serveOptions{httpTokenFile: custom}, true, custom},
		{"mtls", container.DeployNative, TransportHTTP, true, serveOptions{httpTokenFile: custom}, false, ""},
		{"no auth", container.DeployNative, TransportHTTP, false, serveOptions{httpNoAuth: true}, false, ""},
		// The default path names the bound port, so publish resolves it.
		{"default path", container.DeployNative, TransportHTTP, false, serveOptions{}, true, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			c.opts.port = 8765
			tok, err := resolveHTTPToken(c.mode, c.t, c.mtls, c.opts)
			if err != nil {
				t.Fatalf("resolveHTTPToken: %v", err)
			}
			if (tok != nil) != c.wantTok {
				t.Fatalf("token = %+v, want issued: %v", tok, c.wantTok)
			}
			if tok == nil {
				return
			}
			if tok.path != c.wantPath {
				t.Errorf("token path = %q, want %q", tok.path, c.wantPath)
			}
			if len(tok.value) != 2*httpTokenBytes {
				t.Errorf("token value has %d hex chars, want %d", len(tok.value), 2*httpTokenBytes)
			}
		})
	}
}
