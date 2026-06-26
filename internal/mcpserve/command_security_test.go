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
		name     string
		mode     container.DeployMode
		t        Transport
		bind     string
		material TLSMaterial
		wantErr  bool
	}{
		{"native stdio no tls ok", container.DeployNative, TransportStdio, "127.0.0.1", none, false},
		{"gateway stdio exempt", container.DeployGateway, TransportStdio, "127.0.0.1", none, false},
		{"loopback http no tls ok", container.DeployNative, TransportHTTP, "127.0.0.1", none, false},
		{"loopback http with tls ok", container.DeployNative, TransportHTTP, "127.0.0.1", full, false},
		{"gateway http no tls refused", container.DeployGateway, TransportHTTP, "127.0.0.1", none, true},
		{"gateway http with tls ok", container.DeployGateway, TransportHTTP, "127.0.0.1", full, false},
		{"non-loopback http no tls refused", container.DeployNative, TransportHTTP, "0.0.0.0", none, true},
		{"non-loopback http with tls ok", container.DeployNative, TransportHTTP, "0.0.0.0", full, false},
		{"standalone loopback no tls ok", container.DeployStandalone, TransportStdio, "127.0.0.1", none, false},
		{"standalone non-loopback no tls refused", container.DeployStandalone, TransportStdio, "10.0.0.1", none, true},
		{"partial material refused", container.DeployNative, TransportHTTP, "127.0.0.1", partial, true},
		{"non-loopback stdio native exempt", container.DeployNative, TransportStdio, "0.0.0.0", none, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			err := validateServeSecurity(c.mode, c.t, c.bind, c.material)
			if c.wantErr && err == nil {
				t.Fatalf("validateServeSecurity(%s,%s,%q) = nil, want error", c.mode, c.t, c.bind)
			}
			if !c.wantErr && err != nil {
				t.Fatalf("validateServeSecurity(%s,%s,%q) = %v, want nil", c.mode, c.t, c.bind, err)
			}
		})
	}
}
