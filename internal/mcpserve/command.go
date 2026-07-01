package mcpserve

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/logging"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/container"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/middleware"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/projectctx"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools"
)

// defaultHTTPPort is the port used by the http transport when --port is unset.
const defaultHTTPPort = 8765

// defaultBindHost is the address the HTTP transports bind by default. It is
// loopback, so an out-of-the-box server is never reachable beyond the local
// host; a non-loopback bind must be opted into (and then requires mTLS).
const defaultBindHost = "127.0.0.1"

// Deployment env keys honored as fallbacks for the corresponding flags. The
// compose template (build/docker) sets QSDEV_DEPLOY_MODE; the gateway allow-list
// is supplied via QSDEV_GATEWAY_AGENTS (comma-separated). QSDEV_GATEWAY_REQUIRE_AUTH
// forces fail-closed authentication even with an empty allow-list. QSDEV_BIND
// overrides the bind host (default loopback). The mTLS material falls back to
// the EnvTLS* keys defined in tlsconfig.go.
const (
	envDeployMode         = "QSDEV_DEPLOY_MODE"
	envGatewayAgents      = "QSDEV_GATEWAY_AGENTS"
	envGatewayRequireAuth = "QSDEV_GATEWAY_REQUIRE_AUTH"
	envBind               = "QSDEV_BIND"
)

// serveOptions bundles the resolved serve-command inputs so runServe keeps a
// short signature (go-conventions: prefer a struct over a long parameter list).
type serveOptions struct {
	transport    string
	deployMode   string
	projectRoot  string
	port         int
	multiAdapter bool
	bind         string
	tlsCert      string
	tlsKey       string
	tlsClientCA  string
}

// Command returns the `serve` subcommand for the `qsdev mcp` command group. It
// launches the universal qsdev MCP server over the selected transport.
func Command() *cobra.Command {
	var opts serveOptions

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the universal qsdev MCP server",
		Long: "Start the universal qsdev MCP server. It speaks the Model Context " +
			"Protocol (revision 2025-11-25) and exposes qsdev tooling to MCP " +
			"clients.\n\nThe default stdio transport reserves stdout for the JSON-RPC " +
			"protocol; all diagnostics are written to stderr.\n\n" +
			"Deployment modes (--deploy-mode, or QSDEV_DEPLOY_MODE):\n" +
			"  native      local stdio process with the standard chain (default)\n" +
			"  gateway     enforcing MCP proxy: adds an authentication layer and\n" +
			"              stricter rate limits, for frameworks without native hooks\n" +
			"  standalone  requires an explicit --project-root and serves /health",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runServe(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVar(&opts.transport, "transport", string(TransportStdio),
		"transport to serve on: stdio or http")
	cmd.Flags().StringVar(&opts.deployMode, "deploy-mode", "",
		"deployment mode: native (default), gateway, or standalone; "+
			"falls back to QSDEV_DEPLOY_MODE")
	cmd.Flags().StringVar(&opts.projectRoot, "project-root", "",
		"explicit project-root override (takes precedence over auto-detection); "+
			"required in standalone mode")
	cmd.Flags().IntVar(&opts.port, "port", defaultHTTPPort,
		"listen port (http transport, gateway, and standalone modes)")
	cmd.Flags().BoolVar(&opts.multiAdapter, "multi-adapter", false,
		"mount and expose every registered framework adapter regardless of project "+
			"detection or client identity (testing/diagnostics)")
	cmd.Flags().StringVar(&opts.bind, "bind", "",
		"bind host for HTTP serving (default 127.0.0.1, loopback); a non-loopback "+
			"bind requires mTLS material and is opt-in; falls back to QSDEV_BIND")
	cmd.Flags().StringVar(&opts.tlsCert, "tls-cert", "",
		"path to the server certificate (PEM) for mTLS; falls back to "+EnvTLSCert)
	cmd.Flags().StringVar(&opts.tlsKey, "tls-key", "",
		"path to the server private key (PEM) for mTLS; falls back to "+EnvTLSKey)
	cmd.Flags().StringVar(&opts.tlsClientCA, "tls-client-ca", "",
		"path to the client-CA bundle (PEM) verifying client certs; falls back to "+EnvTLSClientCA)

	return cmd
}

// runServe resolves the deployment mode and project root, initializes stderr
// logging, constructs the server with the mode-appropriate middleware chain, and
// runs it over the chosen transport until interrupted.
func runServe(ctx context.Context, opts serveOptions) error {
	t := Transport(opts.transport)
	if t != TransportStdio && t != TransportHTTP {
		return fmt.Errorf("unknown transport %q: want %q or %q", opts.transport, TransportStdio, TransportHTTP)
	}

	mode, err := container.ParseDeployMode(opts.deployMode, os.Getenv(envDeployMode))
	if err != nil {
		return err
	}

	root, err := resolveRootForMode(mode, opts.projectRoot)
	if err != nil {
		return err
	}

	// Resolve the bind host (loopback by default) and the mTLS material, then
	// enforce the fail-closed network-exposure rules BEFORE any listener opens.
	bindHost := resolveBindHost(opts.bind, os.Getenv(envBind))
	material := resolveTLSMaterial(opts.tlsCert, opts.tlsKey, opts.tlsClientCA, os.Getenv)
	if verr := validateServeSecurity(mode, t, bindHost, material, gatewayRequireAuth()); verr != nil {
		return verr
	}
	var tlsConfig *tls.Config
	if material.Complete() {
		tlsConfig, err = material.ServerTLSConfig()
		if err != nil {
			return fmt.Errorf("building mTLS config: %w", err)
		}
	}
	addr := net.JoinHostPort(bindHost, strconv.Itoa(opts.port))

	// Force stderr logging: in stdio mode stdout carries the protocol, so every
	// diagnostic must land on stderr instead. A logging failure is non-fatal.
	session, _ := logging.Init(logging.Config{
		StderrToo:     true,
		ProjectRoot:   root,
		ProjectScoped: root != "",
	})
	defer session.Close() // Close is nil-safe.

	// Select the middleware chain for the deployment mode. Native and standalone
	// run the standard six-layer chain; gateway wraps it with an outer
	// authentication layer and tighter rate limits (see container.GatewayChain).
	srv := New(
		WithProjectRoot(root),
		WithChain(chainForMode(mode)),
		WithMultiAdapter(opts.multiAdapter),
	)

	// Mount the generic project context surface (tools/resources/prompts). A
	// failure here must not prevent the server from starting: log and continue so
	// adapter-contributed tooling and the protocol itself still work.
	if pc, perr := projectctx.NewProjectContext(root); perr != nil {
		slog.Warn("project context engine unavailable; generic tools not mounted", "error", perr)
	} else {
		srv.MountProjectContext(pc)
	}

	// Mount the security and devenv tool surface (Unit 32.9). These are
	// framework-agnostic and always visible, like the project context tools.
	srv.MountTools(tools.All(root))

	if ctx == nil {
		ctx = context.Background()
	}
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Loudly flag any non-loopback exposure. Reaching here means it is already
	// gated by validateServeSecurity (mTLS is present), but the operator should
	// still see that the server is reachable beyond localhost.
	if servesOverHTTP(mode, t) && !isLoopbackHost(bindHost) {
		slog.Warn("binding a non-loopback address: the MCP server is reachable beyond localhost",
			"addr", addr, "mtls", tlsConfig != nil)
	}

	if err := runTransport(ctx, srv, mode, t, addr, tlsConfig); err != nil {
		// A cancelled context is the normal way the server stops on a signal;
		// do not surface it as a command error.
		if errors.Is(err, context.Canceled) {
			return nil
		}
		return err
	}
	return nil
}

// resolveRootForMode resolves the project root with mode-specific rules.
// Standalone refuses CWD auto-detection (a container's CWD is not meaningful):
// it requires an explicit --project-root or QSDEV_PROJECT_ROOT. Native and
// gateway use the standard resolver.
func resolveRootForMode(mode container.DeployMode, flagRoot string) (string, error) {
	if mode == container.DeployStandalone {
		root, err := container.StandaloneProjectRoot(flagRoot, os.Getenv)
		if err != nil {
			return "", err
		}
		// Normalize through the standard resolver so the explicit root is made
		// absolute and walked to the nearest marker, identically to other modes.
		return ResolveProjectRoot(ResolveOptions{FlagRoot: root})
	}
	root, err := ResolveProjectRoot(ResolveOptions{FlagRoot: flagRoot})
	if err != nil {
		return "", fmt.Errorf("resolving project root: %w", err)
	}
	return root, nil
}

// chainForMode returns the middleware chain for the deployment mode.
func chainForMode(mode container.DeployMode) *spi.Chain {
	if mode == container.DeployGateway {
		return container.GatewayChain(container.GatewayOptions{
			AllowedAgents: splitAgents(os.Getenv(envGatewayAgents)),
			RequireAuth:   gatewayRequireAuth(),
		})
	}
	return middleware.DefaultChain()
}

// gatewayRequireAuth reports whether gateway allow-list authorization is being
// enforced: a non-empty agent allow-list, or an explicit QSDEV_GATEWAY_REQUIRE_AUTH.
// It is the single source of truth shared by the gateway chain (chainForMode) and
// the fail-closed startup validation (validateServeSecurity), so the two cannot
// disagree about whether authorization is on.
func gatewayRequireAuth() bool {
	return len(splitAgents(os.Getenv(envGatewayAgents))) > 0 || envTruthy(os.Getenv(envGatewayRequireAuth))
}

// runTransport dispatches to the transport for the resolved mode. Standalone
// always serves over HTTP with a /health endpoint (orchestration needs a
// non-MCP liveness signal); the other modes honor the --transport flag. addr is
// the resolved host:port and tlsConfig is the mTLS configuration (nil for plain
// HTTP / stdio); both are produced and validated by runServe.
func runTransport(ctx context.Context, srv *Server, mode container.DeployMode, t Transport, addr string, tlsConfig *tls.Config) error {
	if mode == container.DeployStandalone {
		return srv.ServeHTTPWithHealth(ctx, addr, tlsConfig)
	}
	if t == TransportHTTP {
		return srv.ServeHTTP(ctx, addr, tlsConfig)
	}
	return srv.ServeStdio(ctx)
}

// servesOverHTTP reports whether the resolved mode/transport opens a network
// listener (standalone always serves HTTP; other modes do so only when the http
// transport is selected). Stdio opens no socket — it is a local pipe (native, or
// a docker-exec gateway admin channel) governed by the OS process boundary — so
// the bind/TLS network rules below do not apply to it.
func servesOverHTTP(mode container.DeployMode, t Transport) bool {
	return mode == container.DeployStandalone || t == TransportHTTP
}

// resolveBindHost resolves the HTTP bind host: an explicit --bind flag wins,
// then QSDEV_BIND, then the loopback default. A loopback default keeps an
// out-of-the-box server unreachable beyond the local host.
func resolveBindHost(flag, env string) string {
	if h := strings.TrimSpace(flag); h != "" {
		return h
	}
	if h := strings.TrimSpace(env); h != "" {
		return h
	}
	return defaultBindHost
}

// isLoopbackHost reports whether host is a loopback bind. "localhost" and any
// loopback IP literal (127.0.0.0/8, ::1) are loopback; an empty host (all
// interfaces) or a non-loopback IP/hostname is NOT, and is treated fail-closed
// (it requires mTLS).
func isLoopbackHost(host string) bool {
	host = strings.TrimSpace(host)
	if host == "localhost" {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	return false
}

// validateServeSecurity enforces the fail-closed network-exposure rules before a
// listener opens. It returns a clear error describing the refusal:
//
//   - Incomplete mTLS material (some but not all of cert/key/client-CA) is a
//     misconfiguration and is rejected outright on any HTTP path.
//   - Gateway mode served over HTTP REQUIRES mTLS: the gateway fronts untrusted
//     frameworks, and authentication is now the client certificate, so it must
//     never expose an unauthenticated surface (even on loopback).
//   - Any non-loopback HTTP bind REQUIRES mTLS, so the server is never reachable
//     beyond localhost without client-certificate authentication.
//   - A loopback HTTP bind in native/http (and standalone) without TLS is allowed
//     — local trusted dev.
//
// Stdio opens no socket and is exempt from the network rules — but gateway
// allow-list authorization cannot be enforced there (see the requireAuth check
// below), so that combination is refused first.
func validateServeSecurity(mode container.DeployMode, t Transport, bindHost string, material TLSMaterial, requireAuth bool) error {
	// Gateway allow-list authorization can only be ENFORCED on a transport that
	// authenticates the caller (mTLS over HTTP). Over stdio the identity is
	// self-asserted — resolveAgentID reads the client-supplied _meta/clientInfo —
	// so the allow-list would gate on a forgeable value while appearing to enforce.
	// Refuse the combination rather than offer false assurance: the stdio admin
	// channel is governed by the OS process boundary.
	if mode == container.DeployGateway && requireAuth && t == TransportStdio {
		return fmt.Errorf("gateway allow-list authorization (%s / %s) cannot be enforced over "+
			"stdio, where the client identity is self-asserted and forgeable; use --transport http "+
			"with mTLS material (%s, %s, %s), or unset the allow-list to rely on the OS process boundary",
			envGatewayAgents, envGatewayRequireAuth, EnvTLSCert, EnvTLSKey, EnvTLSClientCA)
	}
	if !servesOverHTTP(mode, t) {
		return nil
	}
	if material.partiallyConfigured() {
		return fmt.Errorf("incomplete mTLS material: set all of %s, %s, and %s (or none of them)",
			EnvTLSCert, EnvTLSKey, EnvTLSClientCA)
	}
	tlsReady := material.Complete()
	if mode == container.DeployGateway && !tlsReady {
		return fmt.Errorf("gateway mode served over HTTP requires mTLS material "+
			"(set %s, %s, and %s); refusing to start an unauthenticated gateway",
			EnvTLSCert, EnvTLSKey, EnvTLSClientCA)
	}
	if !isLoopbackHost(bindHost) && !tlsReady {
		return fmt.Errorf("binding non-loopback address %q over HTTP requires mTLS material "+
			"(set %s, %s, and %s); refusing to expose an unauthenticated server",
			bindHost, EnvTLSCert, EnvTLSKey, EnvTLSClientCA)
	}
	return nil
}

// splitAgents parses a comma-separated agent allow-list, dropping blanks.
func splitAgents(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// envTruthy reports whether an env value is an affirmative boolean-ish string.
func envTruthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
