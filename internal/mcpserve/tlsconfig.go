package mcpserve

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"
)

// Environment variable names for the mTLS material the HTTP transports consume.
// They form the stable public contract the container/compose generator wires
// into a gateway service (it redefines its own copies to avoid an import cycle,
// the same way it does for QSDEV_DEPLOY_MODE). Keep the names stable.
const (
	// EnvTLSCert names the server certificate (PEM) the server presents.
	EnvTLSCert = "QSDEV_TLS_CERT"
	// EnvTLSKey names the server private key (PEM) matching EnvTLSCert.
	EnvTLSKey = "QSDEV_TLS_KEY"
	// EnvTLSClientCA names the client-CA bundle (PEM) used to verify the
	// presented client certificate in mTLS (ClientCAs).
	EnvTLSClientCA = "QSDEV_TLS_CLIENT_CA"
)

// TLSMaterial is the resolved set of file paths for serving mutual TLS: the
// server certificate/key the server presents, and the client-CA bundle it uses
// to verify the caller's client certificate. All three are required together;
// none of them means "plain HTTP" and a partial set is a configuration error
// (see partiallyConfigured / ServerTLSConfig).
type TLSMaterial struct {
	// CertFile is the path to the server certificate (PEM).
	CertFile string
	// KeyFile is the path to the server private key (PEM).
	KeyFile string
	// ClientCAFile is the path to the client-CA bundle (PEM) used as ClientCAs.
	ClientCAFile string
}

// resolveTLSMaterial resolves the mTLS material from explicit flag values,
// falling back to the EnvTLS* environment variables for any blank flag. getenv
// is injected so callers (and tests) can supply a controlled environment; a nil
// getenv is treated as an empty environment.
func resolveTLSMaterial(certFlag, keyFlag, clientCAFlag string, getenv func(string) string) TLSMaterial {
	if getenv == nil {
		getenv = func(string) string { return "" }
	}
	pick := func(flag, env string) string {
		if v := strings.TrimSpace(flag); v != "" {
			return v
		}
		return strings.TrimSpace(getenv(env))
	}
	return TLSMaterial{
		CertFile:     pick(certFlag, EnvTLSCert),
		KeyFile:      pick(keyFlag, EnvTLSKey),
		ClientCAFile: pick(clientCAFlag, EnvTLSClientCA),
	}
}

// anySet reports whether at least one material path is present.
func (m TLSMaterial) anySet() bool {
	return m.CertFile != "" || m.KeyFile != "" || m.ClientCAFile != ""
}

// Complete reports whether all three material paths are present, i.e. mTLS can
// be served.
func (m TLSMaterial) Complete() bool {
	return m.CertFile != "" && m.KeyFile != "" && m.ClientCAFile != ""
}

// partiallyConfigured reports whether some but not all material is present — a
// misconfiguration the serve command must reject fail-closed rather than
// silently degrade to plain HTTP.
func (m TLSMaterial) partiallyConfigured() bool {
	return m.anySet() && !m.Complete()
}

// ServerTLSConfig builds a *tls.Config that serves mutual TLS: it presents the
// server certificate, verifies the caller's client certificate against the
// configured client-CA bundle, and refuses any connection that does not present
// a CA-signed client certificate (ClientAuth: RequireAndVerifyClientCert). The
// minimum protocol version is TLS 1.2 (security-rules.md). It returns an error
// when the material is incomplete or any file fails to load/parse.
func (m TLSMaterial) ServerTLSConfig() (*tls.Config, error) {
	if !m.Complete() {
		return nil, fmt.Errorf("incomplete mTLS material: set all of %s, %s, and %s", EnvTLSCert, EnvTLSKey, EnvTLSClientCA)
	}

	cert, err := tls.LoadX509KeyPair(m.CertFile, m.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("loading server certificate/key: %w", err)
	}

	caPEM, err := os.ReadFile(m.ClientCAFile)
	if err != nil {
		return nil, fmt.Errorf("reading client-CA bundle %q: %w", m.ClientCAFile, err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("client-CA bundle %q contains no usable PEM certificates", m.ClientCAFile)
	}

	return &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
		ClientCAs:    pool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	}, nil
}

// trustedAgentFromCert extracts the verified client identity from a completed
// mTLS handshake. It reads the leaf certificate of the first verified chain and
// returns its Subject CommonName, falling back to the first DNS SAN. The
// identity is cryptographically trustworthy precisely because it is taken from
// VerifiedChains (chains the TLS stack already validated against ClientCAs), not
// from the raw PeerCertificates the client merely presented. It reports false
// when the state carries no verified chain or yields no usable name.
func trustedAgentFromCert(cs *tls.ConnectionState) (string, bool) {
	if cs == nil || len(cs.VerifiedChains) == 0 || len(cs.VerifiedChains[0]) == 0 {
		return "", false
	}
	leaf := cs.VerifiedChains[0][0]
	if cn := strings.TrimSpace(leaf.Subject.CommonName); cn != "" {
		return cn, true
	}
	for _, dns := range leaf.DNSNames {
		if d := strings.TrimSpace(dns); d != "" {
			return d, true
		}
	}
	return "", false
}

// trustedAgentKey is the unexported context-key type for a verified transport
// identity. A dedicated zero-size struct type guarantees no collision with keys
// from other packages (per the context-value convention).
type trustedAgentKey struct{}

// withTrustedAgent returns a copy of ctx carrying a cryptographically-verified
// transport identity id (an mTLS client-certificate CN/SAN). The cert-identity
// middleware sets it on the request context so it reaches the tool handler, and
// callContext promotes it to the authoritative agent id (see bridge.go).
func withTrustedAgent(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, trustedAgentKey{}, id)
}

// trustedAgentFromContext returns the verified transport identity stored in ctx,
// if any. The boolean is false when none is present or it is empty, so callers
// fall back to the self-asserted (stdio/local) identity.
func trustedAgentFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(trustedAgentKey{}).(string)
	if !ok || id == "" {
		return "", false
	}
	return id, true
}
