package mcpserve

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/server"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
)

// ---- ephemeral certificate authority (test-only, in-memory) ----------------

// testCA is an in-test certificate authority. It signs short-lived server and
// client leaf certificates from a freshly generated ECDSA key. Nothing here is
// persisted beyond t.TempDir(), and no real key material is committed.
type testCA struct {
	cert    *x509.Certificate
	key     *ecdsa.PrivateKey
	certPEM []byte
}

// newTestCA mints a self-signed CA.
func newTestCA(t *testing.T) *testCA {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generating CA key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          serial(t),
		Subject:               pkix.Name{CommonName: "qsdev-test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("creating CA cert: %v", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parsing CA cert: %v", err)
	}
	return &testCA{
		cert:    cert,
		key:     key,
		certPEM: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
	}
}

// pool returns an x509 pool trusting this CA.
func (ca *testCA) pool(t *testing.T) *x509.CertPool {
	t.Helper()
	p := x509.NewCertPool()
	if !p.AppendCertsFromPEM(ca.certPEM) {
		t.Fatal("appending CA cert to pool")
	}
	return p
}

// issue signs a leaf certificate with the given CommonName and DNS SANs. When
// forServer is true the cert carries server-auth EKU and loopback IP SANs (so a
// TLS client connecting to 127.0.0.1 verifies it); otherwise it carries
// client-auth EKU. The returned tls.Certificate bundles the leaf and its key.
func (ca *testCA) issue(t *testing.T, cn string, dnsNames []string, forServer bool) tls.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generating leaf key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: serial(t),
		Subject:      pkix.Name{CommonName: cn},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		DNSNames:     dnsNames,
	}
	if forServer {
		tmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
		tmpl.IPAddresses = []net.IP{net.ParseIP("127.0.0.1"), net.IPv6loopback}
	} else {
		tmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, ca.cert, &key.PublicKey, ca.key)
	if err != nil {
		t.Fatalf("creating leaf cert: %v", err)
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parsing leaf cert: %v", err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key, Leaf: leaf}
}

// writeMaterial writes a server cert/key plus this CA's bundle to t.TempDir()
// and returns the resolved TLSMaterial. The PEM is assembled in memory.
func (ca *testCA) writeMaterial(t *testing.T, serverCert tls.Certificate) TLSMaterial {
	t.Helper()
	dir := t.TempDir()
	certPath := filepath.Join(dir, "server.crt")
	keyPath := filepath.Join(dir, "server.key")
	caPath := filepath.Join(dir, "client-ca.crt")

	writePEM(t, certPath, "CERTIFICATE", serverCert.Certificate[0])
	keyDER, err := x509.MarshalECPrivateKey(serverCert.PrivateKey.(*ecdsa.PrivateKey))
	if err != nil {
		t.Fatalf("marshaling server key: %v", err)
	}
	writePEM(t, keyPath, "EC PRIVATE KEY", keyDER)
	writeBytes(t, caPath, ca.certPEM)

	return TLSMaterial{CertFile: certPath, KeyFile: keyPath, ClientCAFile: caPath}
}

func serial(t *testing.T) *big.Int {
	t.Helper()
	n, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		t.Fatalf("generating serial: %v", err)
	}
	return n
}

func writePEM(t *testing.T, path, blockType string, der []byte) {
	t.Helper()
	writeBytes(t, path, pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: der}))
}

func writeBytes(t *testing.T, path string, b []byte) {
	t.Helper()
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

// ---- tlsconfig unit tests --------------------------------------------------

func TestResolveTLSMaterial(t *testing.T) {
	t.Parallel()

	env := func(m map[string]string) func(string) string {
		return func(k string) string { return m[k] }
	}

	t.Run("flags win over env", func(t *testing.T) {
		t.Parallel()
		got := resolveTLSMaterial("c", "k", "ca", env(map[string]string{
			EnvTLSCert: "ec", EnvTLSKey: "ek", EnvTLSClientCA: "eca",
		}))
		if got != (TLSMaterial{CertFile: "c", KeyFile: "k", ClientCAFile: "ca"}) {
			t.Fatalf("flags did not win: %+v", got)
		}
	})

	t.Run("env fallback for blank flags", func(t *testing.T) {
		t.Parallel()
		got := resolveTLSMaterial("", "", "", env(map[string]string{
			EnvTLSCert: "ec", EnvTLSKey: "ek", EnvTLSClientCA: "eca",
		}))
		if !got.Complete() {
			t.Fatalf("expected complete material from env, got %+v", got)
		}
	})

	t.Run("complete and partial detection", func(t *testing.T) {
		t.Parallel()
		full := TLSMaterial{CertFile: "c", KeyFile: "k", ClientCAFile: "ca"}
		if !full.Complete() || full.partiallyConfigured() {
			t.Errorf("full material misclassified: %+v", full)
		}
		partial := TLSMaterial{CertFile: "c"}
		if partial.Complete() || !partial.partiallyConfigured() {
			t.Errorf("partial material misclassified: %+v", partial)
		}
		var empty TLSMaterial
		if empty.Complete() || empty.partiallyConfigured() || empty.anySet() {
			t.Errorf("empty material misclassified: %+v", empty)
		}
	})
}

func TestServerTLSConfig(t *testing.T) {
	t.Parallel()

	ca := newTestCA(t)
	mat := ca.writeMaterial(t, ca.issue(t, "test-server", nil, true))

	cfg, err := mat.ServerTLSConfig()
	if err != nil {
		t.Fatalf("ServerTLSConfig: %v", err)
	}
	if cfg.MinVersion != tls.VersionTLS12 {
		t.Errorf("MinVersion = %x, want TLS 1.2 (%x)", cfg.MinVersion, tls.VersionTLS12)
	}
	if cfg.ClientAuth != tls.RequireAndVerifyClientCert {
		t.Errorf("ClientAuth = %v, want RequireAndVerifyClientCert", cfg.ClientAuth)
	}
	if cfg.ClientCAs == nil {
		t.Error("ClientCAs is nil; mTLS would not verify client certs")
	}
	if len(cfg.Certificates) != 1 {
		t.Errorf("Certificates len = %d, want 1", len(cfg.Certificates))
	}

	t.Run("incomplete material errors", func(t *testing.T) {
		t.Parallel()
		if _, err := (TLSMaterial{CertFile: "x"}).ServerTLSConfig(); err == nil {
			t.Error("expected error for incomplete material")
		}
	})

	t.Run("bad CA bundle errors", func(t *testing.T) {
		t.Parallel()
		bad := mat
		badPath := filepath.Join(t.TempDir(), "bad-ca.pem")
		writeBytes(t, badPath, []byte("not a pem"))
		bad.ClientCAFile = badPath
		if _, err := bad.ServerTLSConfig(); err == nil {
			t.Error("expected error for a CA bundle with no usable certs")
		}
	})
}

func TestTrustedAgentFromCert(t *testing.T) {
	t.Parallel()

	chain := func(leaf *x509.Certificate) *tls.ConnectionState {
		return &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{leaf}}}
	}

	cases := []struct {
		name string
		cs   *tls.ConnectionState
		want string
		wOK  bool
	}{
		{"nil state", nil, "", false},
		{"no verified chains", &tls.ConnectionState{}, "", false},
		{
			"common name preferred",
			chain(&x509.Certificate{Subject: pkix.Name{CommonName: "agent-cn"}, DNSNames: []string{"dns-san"}}),
			"agent-cn", true,
		},
		{
			"dns san fallback",
			chain(&x509.Certificate{DNSNames: []string{"dns-san", "second"}}),
			"dns-san", true,
		},
		{
			"no usable name",
			chain(&x509.Certificate{}),
			"", false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, ok := trustedAgentFromCert(c.cs)
			if got != c.want || ok != c.wOK {
				t.Errorf("trustedAgentFromCert = (%q,%v), want (%q,%v)", got, ok, c.want, c.wOK)
			}
		})
	}
}

func TestTrustedAgentContextRoundTrip(t *testing.T) {
	t.Parallel()

	if _, ok := trustedAgentFromContext(context.Background()); ok {
		t.Error("empty context unexpectedly carried a trusted agent")
	}
	ctx := withTrustedAgent(context.Background(), "cert-cn")
	got, ok := trustedAgentFromContext(ctx)
	if !ok || got != "cert-cn" {
		t.Errorf("round trip = (%q,%v), want (cert-cn,true)", got, ok)
	}
	// An empty id is not a usable identity.
	if _, ok := trustedAgentFromContext(withTrustedAgent(context.Background(), "")); ok {
		t.Error("empty trusted id should not be reported as present")
	}
}

// TestCallContextTrustedOverride proves R2 at the unit level: a verified
// transport identity in ctx is authoritative and overrides a client-asserted
// _meta agentId.
func TestCallContextTrustedOverride(t *testing.T) {
	t.Parallel()

	srv := New(WithProjectRoot("/proj"))
	meta := map[string]any{MetaAgentIDKey: "attacker-asserted"}

	// Without a trusted transport identity, the self-asserted _meta wins.
	cc := srv.callContext(context.Background(), "tool", meta)
	if cc.AgentID != "attacker-asserted" {
		t.Fatalf("no-cert path AgentID = %q, want the self-asserted value", cc.AgentID)
	}

	// With a verified transport identity, the cert wins and _meta is ignored.
	ctx := withTrustedAgent(context.Background(), "verified-cn")
	cc = srv.callContext(ctx, "tool", meta)
	if cc.AgentID != "verified-cn" {
		t.Fatalf("cert path AgentID = %q, want verified-cn (cert overrides _meta)", cc.AgentID)
	}
}

// ---- real mTLS handshake (TLS-layer enforcement) ---------------------------

// TestMTLSHandshake serves the real ServerTLSConfig + certIdentityMiddleware over
// a real TLS socket and asserts (1) a CA-signed client cert connects and its
// CN reaches the request context, (2) a client with no cert is rejected at the
// TLS layer, and (3) a client whose cert is signed by an untrusted CA is
// rejected at the TLS layer.
func TestMTLSHandshake(t *testing.T) {
	t.Parallel()

	ca := newTestCA(t)
	serverCert := ca.issue(t, "test-server", nil, true)
	serverTLS, err := ca.writeMaterial(t, serverCert).ServerTLSConfig()
	if err != nil {
		t.Fatalf("ServerTLSConfig: %v", err)
	}

	// A probe handler that echoes the verified identity the middleware injected.
	probe := certIdentityMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, _ := trustedAgentFromContext(r.Context())
		_, _ = io.WriteString(w, id)
	}))
	ts := httptest.NewUnstartedServer(probe)
	ts.TLS = serverTLS
	ts.StartTLS()
	// t.Cleanup (not defer): the subtests below run in parallel and execute AFTER
	// this function returns, so a deferred Close would shut the server down before
	// they connect. Cleanup runs only once all parallel subtests complete.
	t.Cleanup(ts.Close)

	caPool := ca.pool(t)

	t.Run("valid client cert injects CN identity", func(t *testing.T) {
		t.Parallel()
		clientCert := ca.issue(t, "agent-alpha", nil, false)
		body, err := mtlsGet(ts.URL, &tls.Config{
			RootCAs:      caPool,
			Certificates: []tls.Certificate{clientCert},
			MinVersion:   tls.VersionTLS12,
		})
		if err != nil {
			t.Fatalf("valid mTLS request failed: %v", err)
		}
		if body != "agent-alpha" {
			t.Errorf("injected identity = %q, want agent-alpha", body)
		}
	})

	t.Run("no client cert is rejected at TLS", func(t *testing.T) {
		t.Parallel()
		if _, err := mtlsGet(ts.URL, &tls.Config{RootCAs: caPool, MinVersion: tls.VersionTLS12}); err == nil {
			t.Error("request with no client cert unexpectedly succeeded")
		}
	})

	t.Run("untrusted-CA client cert is rejected at TLS", func(t *testing.T) {
		t.Parallel()
		rogueCA := newTestCA(t)
		rogueCert := rogueCA.issue(t, "agent-alpha", nil, false) // same CN, untrusted issuer
		_, err := mtlsGet(ts.URL, &tls.Config{
			RootCAs:      caPool, // still trust the server's CA
			Certificates: []tls.Certificate{rogueCert},
			MinVersion:   tls.VersionTLS12,
		})
		if err == nil {
			t.Error("client cert from an untrusted CA unexpectedly succeeded")
		}
	})
}

// ---- end-to-end identity flow over mTLS into the MCP tool handler ----------

// TestMTLSIdentityReachesToolHandler is the full-stack R2 proof: a request over
// real mTLS, through the real Streamable HTTP handler and certIdentityMiddleware,
// resolves cc.AgentID to the client-cert CN inside a mounted probe tool — and a
// client-asserted _meta agentId does NOT override that verified identity.
func TestMTLSIdentityReachesToolHandler(t *testing.T) {
	t.Parallel()

	ca := newTestCA(t)
	serverTLS, err := ca.writeMaterial(t, ca.issue(t, "test-server", nil, true)).ServerTLSConfig()
	if err != nil {
		t.Fatalf("ServerTLSConfig: %v", err)
	}

	// A probe tool that returns the resolved authoritative agent id verbatim.
	srv := New(WithProjectRoot(t.TempDir()))
	srv.MountTools([]spi.ToolRegistration{{
		Name:        "qsdev_whoami",
		Description: "echoes the resolved agent id",
		Handler: func(_ context.Context, cc *spi.ToolCallContext, _ *spi.ToolRequest) (*spi.ToolResult, error) {
			return &spi.ToolResult{Text: cc.AgentID}, nil
		},
	}})

	// Build the same handler shape ServeHTTP serves (mux at /mcp wrapped by the
	// cert-identity middleware), over a real mTLS socket.
	streamable := server.NewStreamableHTTPServer(srv.MCPServer())
	mux := http.NewServeMux()
	mux.Handle(mcpEndpointPath, streamable)
	ts := httptest.NewUnstartedServer(certIdentityMiddleware(mux))
	ts.TLS = serverTLS
	ts.StartTLS()
	defer ts.Close()

	url := ts.URL + mcpEndpointPath
	client := mtlsClient(t, &tls.Config{
		RootCAs:      ca.pool(t),
		Certificates: []tls.Certificate{ca.issue(t, "agent-bravo", nil, false)},
		MinVersion:   tls.VersionTLS12,
	})

	session := mcpInitialize(t, client, url)
	// Call the probe with a HOSTILE _meta agentId; the verified cert CN must win.
	got := mcpToolCallText(t, client, url, session, "qsdev_whoami", map[string]any{
		"_meta": map[string]any{MetaAgentIDKey: "attacker-asserted"},
	})
	if got != "agent-bravo" {
		t.Fatalf("tool saw agent id %q, want the verified cert CN agent-bravo "+
			"(a client-asserted _meta agentId must not override it)", got)
	}
}

// ---- raw MCP-over-HTTP helpers --------------------------------------------

func mtlsClient(t *testing.T, cfg *tls.Config) *http.Client {
	t.Helper()
	return &http.Client{
		Timeout:   10 * time.Second,
		Transport: &http.Transport{TLSClientConfig: cfg},
	}
}

// mtlsGet performs a single GET with the given client TLS config and returns the
// response body, or an error (including TLS handshake failures).
func mtlsGet(url string, cfg *tls.Config) (string, error) {
	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: &http.Transport{TLSClientConfig: cfg},
	}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// mcpInitialize performs the MCP initialize handshake and returns the session id.
func mcpInitialize(t *testing.T, client *http.Client, url string) string {
	t.Helper()
	body := map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "initialize",
		"params": map[string]any{
			"protocolVersion": "2025-11-25",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "mtls-itest", "version": "0.0.1"},
		},
	}
	resp := mcpPost(t, client, url, "", body)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("initialize status = %d, want 200", resp.StatusCode)
	}
	sid := resp.Header.Get("Mcp-Session-Id")
	if sid == "" {
		t.Fatal("initialize did not return an Mcp-Session-Id header")
	}
	return sid
}

// mcpToolCallText invokes a tool and returns the concatenated text content.
func mcpToolCallText(t *testing.T, client *http.Client, url, session, name string, params map[string]any) string {
	t.Helper()
	call := map[string]any{"name": name, "arguments": map[string]any{}}
	for k, v := range params {
		call[k] = v
	}
	body := map[string]any{"jsonrpc": "2.0", "id": 2, "method": "tools/call", "params": call}
	resp := mcpPost(t, client, url, session, body)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("tools/call status = %d, body=%s", resp.StatusCode, raw)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading tools/call response: %v", err)
	}
	var out struct {
		Result struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
			IsError bool `json:"isError"`
		} `json:"result"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decoding tools/call response %q: %v", raw, err)
	}
	if out.Error != nil {
		t.Fatalf("tools/call returned a protocol error: %s", out.Error.Message)
	}
	var b bytes.Buffer
	for _, c := range out.Result.Content {
		if c.Type == "text" {
			b.WriteString(c.Text)
		}
	}
	return b.String()
}

func mcpPost(t *testing.T, client *http.Client, url, session string, body map[string]any) *http.Response {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshaling request: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("building request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if session != "" {
		req.Header.Set("Mcp-Session-Id", session)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	return resp
}
