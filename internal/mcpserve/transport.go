package mcpserve

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/mark3labs/mcp-go/server"
)

// Transport identifies how the server communicates with its client.
type Transport string

const (
	// TransportStdio serves the MCP protocol over stdin/stdout (the default).
	TransportStdio Transport = "stdio"
	// TransportHTTP serves the MCP protocol over Streamable HTTP.
	TransportHTTP Transport = "http"
)

// mcpEndpointPath is the path the Streamable HTTP handler is mounted at. It
// mirrors mcp-go's default endpoint (server.WithEndpointPath default "/mcp"),
// which the generated .mcp.json entry points at (http://host:port/mcp).
const mcpEndpointPath = "/mcp"

// httpShutdownTimeout bounds graceful shutdown of the HTTP transport.
const httpShutdownTimeout = 5 * time.Second

// httpReadHeaderTimeout bounds how long an HTTP server waits for a request's
// headers, mitigating slowloris-style stalls on the health/MCP mux.
const httpReadHeaderTimeout = 10 * time.Second

// ServeStdio runs the server over the process's stdin/stdout until ctx is
// cancelled or stdin reaches EOF. In stdio mode stdout is reserved exclusively
// for the JSON-RPC protocol; all diagnostics go to stderr.
func (s *Server) ServeStdio(ctx context.Context) error {
	return s.listenStdio(ctx, os.Stdin, os.Stdout)
}

// listenStdio is the single stdio code path. Production passes
// os.Stdin/os.Stdout; tests pass an io.Pipe to drive a real handshake without a
// child process. Using NewStdioServer(...).Listen gives context cancellation
// over and above the package-level ServeStdio helper.
func (s *Server) listenStdio(ctx context.Context, stdin io.Reader, stdout io.Writer) error {
	stdio := server.NewStdioServer(s.mcp)
	// Route mcp-go's internal error logging to stderr so it never corrupts the
	// stdout protocol stream.
	stdio.SetErrorLogger(log.New(os.Stderr, "mcpserve: ", log.LstdFlags))
	return stdio.Listen(ctx, stdin, stdout)
}

// ServeHTTP runs the server over Streamable HTTP on addr (e.g.
// "127.0.0.1:8765") until ctx is cancelled, then shuts the HTTP server down
// gracefully. It builds its own *http.Server (rather than calling the vendored
// streamable.Start) so the transport controls TLS, timeouts, and the
// cert-identity middleware. When tlsConfig is non-nil the listener serves mutual
// TLS (the certificates live in tlsConfig); otherwise it serves plain HTTP,
// which the serve command only permits on a loopback bind in native/http mode.
func (s *Server) ServeHTTP(ctx context.Context, addr string, tlsConfig *tls.Config) error {
	streamable := server.NewStreamableHTTPServer(s.mcp)

	mux := http.NewServeMux()
	mux.Handle(mcpEndpointPath, streamable)

	return s.serveMux(ctx, addr, mux, tlsConfig)
}

// ServeHTTPWithHealth runs the server over Streamable HTTP on addr AND exposes a
// plain GET /health endpoint for container orchestration (liveness/readiness
// probes). The MCP protocol is served by the vendored streamable handler at the
// root; /health is mounted ahead of it on a shared mux. Used by the standalone
// deployment mode, where an orchestrator needs a non-MCP health signal. It shuts
// down gracefully when ctx is cancelled.
//
// Under mTLS (tlsConfig non-nil) every connection — including a /health probe —
// must present a CA-signed client certificate, because ClientAuth is
// RequireAndVerifyClientCert and the handshake is rejected before any handler
// (hence before path routing) runs. This is the deliberate fail-closed choice:
// the standalone server never exposes an unauthenticated surface. Configure the
// orchestrator's health probe to present the client certificate (e.g. an
// exec/curl probe with --cert/--key) rather than a bare HTTP GET.
func (s *Server) ServeHTTPWithHealth(ctx context.Context, addr string, tlsConfig *tls.Config) error {
	streamable := server.NewStreamableHTTPServer(s.mcp)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.Handle("/", streamable)

	return s.serveMux(ctx, addr, mux, tlsConfig)
}

// serveMux builds the transport's *http.Server around mux — applying the shared
// handler stack (cert-identity, plus a loopback Host/Origin guard on the
// plain-HTTP path), read-header timeout, and TLS config — and serves it until
// ctx is cancelled, shutting down gracefully. Both HTTP entrypoints route through
// it so the server's timeouts, middleware, and TLS wiring stay identical; they
// differ only in how they populate mux. When tlsConfig is nil the listener serves
// plain HTTP, which the serve command permits only on a loopback bind.
func (s *Server) serveMux(ctx context.Context, addr string, mux *http.ServeMux, tlsConfig *tls.Config) error {
	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           httpHandler(mux, tlsConfig),
		ReadHeaderTimeout: httpReadHeaderTimeout,
		TLSConfig:         tlsConfig,
	}
	return serveHTTPWithShutdown(ctx, httpSrv, tlsConfig != nil)
}

// serveHTTPWithShutdown starts httpSrv (TLS when useTLS) in a goroutine and
// blocks until ctx is cancelled — at which point it shuts the server down within
// httpShutdownTimeout — or the listener fails. A clean close
// (http.ErrServerClosed) is reported as success. When useTLS is true the
// certificates come from httpSrv.TLSConfig, so ListenAndServeTLS is called with
// empty cert/key paths.
func serveHTTPWithShutdown(ctx context.Context, httpSrv *http.Server, useTLS bool) error {
	errCh := make(chan error, 1)
	go func() {
		if useTLS {
			errCh <- httpSrv.ListenAndServeTLS("", "")
		} else {
			errCh <- httpSrv.ListenAndServe()
		}
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), httpShutdownTimeout)
		defer cancel()
		if err := httpSrv.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return ctx.Err()
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

// certIdentityMiddleware wraps an HTTP handler so that, when a request arrives
// over a verified mTLS connection, the verified client identity (the cert CN or
// first DNS SAN, taken from VerifiedChains) is injected into the request context
// before the inner handler runs. mcp-go propagates r.Context() through to the
// tool handler (it sets the per-request HTTPRequest.Context from r.Context()),
// so the injected identity reaches callContext, which promotes it to the
// authoritative cc.AgentID. A request without a verified chain (plain HTTP, or a
// TLS config that does not require client certs) passes through unchanged and
// falls back to the self-asserted identity.
func certIdentityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS != nil && len(r.TLS.VerifiedChains) > 0 {
			if id, ok := trustedAgentFromCert(r.TLS); ok {
				r = r.WithContext(withTrustedAgent(r.Context(), id))
			}
		}
		next.ServeHTTP(w, r)
	})
}

// httpHandler composes the transport's handler stack. certIdentityMiddleware
// always runs (it is a no-op on plain HTTP). On the plain-HTTP path
// (tlsConfig == nil) the stack is additionally wrapped in loopbackGuard: plain
// HTTP is only ever served on a loopback bind (validateServeSecurity enforces
// this), so requiring a loopback Host/Origin there costs nothing and blocks
// DNS-rebinding. Under mTLS the verified client certificate is the gate and the
// operator may legitimately bind a non-loopback DNS name, so the guard is omitted
// to avoid rejecting legitimate requests.
func httpHandler(mux http.Handler, tlsConfig *tls.Config) http.Handler {
	h := certIdentityMiddleware(mux)
	if tlsConfig == nil {
		h = loopbackGuard(h)
	}
	return h
}

// loopbackGuard rejects (403, before any tool handler runs) every request whose
// Host header is not a loopback address, or whose Origin header is present and
// not loopback. It defends the plain-HTTP transport against DNS-rebinding: a
// rebound browser request carries the attacker's domain in Host (and Origin),
// failing the loopback check. A request with no Origin — the usual non-browser
// MCP client — is allowed. CORS would not suffice: it only withholds the response
// from a cross-origin reader while a state-changing tool call still executes, so
// rejecting the request outright is the actual defense.
func loopbackGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !hostIsLoopback(r.Host) || !originIsLoopback(r.Header.Get("Origin")) {
			http.Error(w, "forbidden: non-loopback Host or Origin", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// hostIsLoopback reports whether an HTTP Host header (host or host:port) names a
// loopback address. An empty Host is rejected. It reuses isLoopbackHost so the
// transport guard and the serve-time bind validation share one loopback
// definition.
func hostIsLoopback(host string) bool {
	if host == "" {
		return false
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	return isLoopbackHost(host)
}

// originIsLoopback reports whether an Origin header is safe for the loopback
// transport: an absent or opaque ("null") origin carries no cross-site browsing
// context and is allowed; any other value must parse to a loopback host.
func originIsLoopback(origin string) bool {
	if origin == "" || origin == "null" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return isLoopbackHost(u.Hostname())
}

// handleHealth answers GET /health with 200 and a minimal JSON liveness body.
// Any non-GET method is rejected so probes cannot be confused with MCP traffic.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, `{"status":"method_not_allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodHead {
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":       "ok",
		"project_root": s.projectRoot,
	})
}
