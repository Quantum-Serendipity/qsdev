package mcpserve

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/container"
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
