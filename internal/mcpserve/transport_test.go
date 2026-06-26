package mcpserve

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestLoopbackGuard verifies the plain-HTTP DNS-rebinding guard: a loopback Host
// with an absent or loopback Origin passes; a rebound non-loopback Host or Origin
// is rejected with 403 before reaching the wrapped handler.
func TestLoopbackGuard(t *testing.T) {
	t.Parallel()
	pass := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	guarded := loopbackGuard(pass)

	cases := []struct {
		name   string
		host   string
		origin string
		want   int
	}{
		{"loopback host, no origin", "127.0.0.1:8765", "", http.StatusOK},
		{"localhost host", "localhost:8765", "", http.StatusOK},
		{"ipv6 loopback host", "[::1]:8765", "", http.StatusOK},
		{"loopback origin", "127.0.0.1:8765", "http://127.0.0.1:8765", http.StatusOK},
		{"opaque null origin", "127.0.0.1:8765", "null", http.StatusOK},
		{"rebinding host", "evil.example.com:8765", "", http.StatusForbidden},
		{"non-loopback origin", "127.0.0.1:8765", "http://evil.example.com", http.StatusForbidden},
		{"empty host", "", "", http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
			req.Host = tc.host
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			rec := httptest.NewRecorder()
			guarded.ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Errorf("Host=%q Origin=%q: status = %d, want %d", tc.host, tc.origin, rec.Code, tc.want)
			}
		})
	}
}

// TestHTTPHandlerPlainHTTPGuarded proves the loopback guard IS applied on the
// plain-HTTP path (tlsConfig == nil): a rebound non-loopback Host is rejected
// before the wrapped handler runs.
func TestHTTPHandlerPlainHTTPGuarded(t *testing.T) {
	t.Parallel()
	var reached bool
	mux := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	})
	h := httpHandler(mux, nil)

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Host = "evil.example.com:8765"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if reached {
		t.Error("rebinding request reached the handler; loopback guard not applied on plain HTTP")
	}
	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

// TestHTTPHandlerMTLSPathUnguarded proves the loopback guard is applied only on
// the plain-HTTP path: with a non-nil TLS config the verified client certificate
// is the gate, so a non-loopback Host (legitimate for an mTLS operator binding a
// DNS name) must pass through rather than be rejected as a rebinding attempt.
func TestHTTPHandlerMTLSPathUnguarded(t *testing.T) {
	t.Parallel()
	var reached bool
	mux := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	})
	h := httpHandler(mux, &tls.Config{MinVersion: tls.VersionTLS13})

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Host = "mcp.internal.example:8765"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if !reached || rec.Code != http.StatusOK {
		t.Errorf("mTLS path must not apply the loopback guard: reached=%v code=%d", reached, rec.Code)
	}
}
