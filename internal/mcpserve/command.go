package mcpserve

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	qsdevconfig "github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/internal/logging"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/container"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/middleware"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/projectctx"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
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
// forces fail-closed authentication even with an empty allow-list.
// QSDEV_GATEWAY_ALLOW_NIX_RUN mounts qsdev_nix_run in gateway mode. QSDEV_BIND
// overrides the bind host (default loopback). The mTLS material falls back to
// the EnvTLS* keys defined in tlsconfig.go.
const (
	envDeployMode         = container.EnvDeployMode
	envGatewayAgents      = container.EnvGatewayAgents
	envGatewayRequireAuth = "QSDEV_GATEWAY_REQUIRE_AUTH"
	envGatewayNixRun      = "QSDEV_GATEWAY_ALLOW_NIX_RUN"
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
	// gatewayNixRun mounts qsdev_nix_run in gateway mode, where it is off by
	// default.
	gatewayNixRun bool
	// modules restricts the server to the named tool modules (tools.Select);
	// empty serves the full universal surface.
	modules []string
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
			"              stricter rate limits, for frameworks without native hooks;\n" +
			"              qsdev_nix_run is off unless --gateway-allow-nix-run\n" +
			"  standalone  requires an explicit --project-root and serves /health\n\n" +
			"--module restricts the server to the named tool modules (" +
			strings.Join(tools.ModuleNames(), ", ") + "), leaving out the project " +
			"context surface and the framework adapters. The tools still run behind " +
			"the full middleware chain and mcp.disabled_tools.",
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
	cmd.Flags().StringSliceVar(&opts.modules, "module", nil,
		"serve only the named tool module(s) (repeatable or comma-separated): "+
			strings.Join(tools.ModuleNames(), ", "))
	cmd.Flags().BoolVar(&opts.gatewayNixRun, "gateway-allow-nix-run", false,
		"gateway mode: mount qsdev_nix_run, which runs Nix packages on the gateway "+
			"host and is off by default there; falls back to "+envGatewayNixRun)

	return cmd
}

// LegacyModuleCommands returns hidden `qsdev mcp <module>` subcommands for the
// modules older releases served standalone (tools.LegacyServerModules). Each is
// an alias of `qsdev mcp serve --module <module>` with the serve defaults, so a
// .mcp.json written before those servers moved onto the universal server keeps
// starting them, now behind the full middleware chain, until `qsdev init
// --update` rewrites the entries.
func LegacyModuleCommands() []*cobra.Command {
	modules := tools.LegacyServerModules()
	cmds := make([]*cobra.Command, len(modules))
	for i, module := range modules {
		cmds[i] = legacyModuleCommand(module, runServe)
	}
	return cmds
}

// legacyModuleCommand builds the alias for module; run is runServe outside
// tests.
func legacyModuleCommand(module string, run func(context.Context, serveOptions) error) *cobra.Command {
	app := branding.Get().AppName
	return &cobra.Command{
		Use:    module,
		Short:  fmt.Sprintf("Deprecated alias of `%s mcp serve --module %s`", app, module),
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// stdout carries the MCP protocol, so the notice goes to stderr.
			fmt.Fprintf(cmd.ErrOrStderr(),
				"%[1]s mcp %[2]s is deprecated; running %[1]s mcp serve --module %[2]s. Run `%[1]s init --update` to rewrite .mcp.json.\n",
				app, module)
			return run(cmd.Context(), serveOptions{
				transport: string(TransportStdio),
				port:      defaultHTTPPort,
				modules:   []string{module},
			})
		},
	}
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
	// The server is launched by the agent for every session, so its log is
	// kept with the other automated sessions rather than evicting the logs of
	// user-run commands.
	session, _ := logging.Init(logging.Config{
		StderrToo:     true,
		ProjectRoot:   root,
		ProjectScoped: root != "",
		Automated:     true,
	})
	defer session.Close() // Close is nil-safe.

	// Derive the Guardrail permission policy from the project's .qsdev.yaml
	// (mcp.disabled_tools) BEFORE building the chain, so a disabled tool is actually
	// enforced on MCP calls — not merely reported denied by qsdev_policy_check. A
	// present-but-unparseable config fails startup rather than silently running
	// un-narrowed (fail closed).
	cfg, err := loadProjectConfig(root)
	if err != nil {
		return err
	}
	policy := middleware.PolicyFromConfig(cfg)

	// Select the middleware chain for the deployment mode. Native and standalone
	// run the standard six-layer chain; gateway wraps it with an outer
	// authentication layer and tighter rate limits (see container.GatewayChain).
	// Both branches receive the derived policy so enforcement matches reporting.
	// Build the tool modules' registrations first, so an unknown --module fails
	// startup before anything is served. The opt-in tools are included only
	// when toolOptions selects them.
	gatewayNixRun := opts.gatewayNixRun || envTruthy(os.Getenv(envGatewayNixRun))
	regs, err := tools.Select(opts.modules, root, policy, toolOptions(cfg, mode, gatewayNixRun))
	if err != nil {
		return err
	}
	srv := newServeServer(root, mode, policy, opts)
	srv.MountTools(regs)
	warnUnknownDisabledTools(policy.DenyToolSet(), MountableToolNames(spi.DefaultRegistry().All()))

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

// newServeServer constructs the server for the serve command. The full
// universal server mounts every applicable framework adapter (in New) and the
// generic project context surface; a --module server mounts neither, so it
// exposes exactly the selected tool modules. Both install the same
// mode-appropriate middleware chain.
func newServeServer(root string, mode container.DeployMode, policy *middleware.Policy, opts serveOptions) *Server {
	serverOpts := []Option{
		WithProjectRoot(root),
		WithChain(chainForMode(mode, policy)),
		WithMultiAdapter(opts.multiAdapter),
	}
	if len(opts.modules) > 0 {
		return New(append(serverOpts,
			WithName(branding.Get().AppName+"-"+strings.Join(opts.modules, "+")),
			WithAdapterRegistry(spi.NewAdapterRegistry()),
		)...)
	}

	srv := New(serverOpts...)
	// Mount the generic project context surface (tools/resources/prompts). A
	// failure here must not prevent the server from starting: log and continue so
	// adapter-contributed tooling and the protocol itself still work.
	if pc, err := projectctx.NewProjectContext(root); err != nil {
		slog.Warn("project context engine unavailable; generic tools not mounted", "error", err)
	} else {
		srv.MountProjectContext(pc)
	}
	return srv
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

// chainForMode returns the middleware chain for the deployment mode, feeding the
// derived Guardrail permission policy into BOTH branches. Native/standalone
// install it via middleware.WithPolicy; gateway installs it via
// GatewayOptions.Policy (which the gateway forwards to the same WithPolicy). A
// nil policy is treated as "keep the permissive default" by both sinks, so a
// project that disables nothing behaves exactly as before.
func chainForMode(mode container.DeployMode, policy *middleware.Policy) *spi.Chain {
	if mode == container.DeployGateway {
		return container.GatewayChain(container.GatewayOptions{
			AllowedAgents: splitAgents(os.Getenv(envGatewayAgents)),
			RequireAuth:   gatewayRequireAuth(),
			Policy:        policy,
		})
	}
	return middleware.DefaultChain(middleware.WithPolicy(policy))
}

// loadProjectConfig loads the project's .qsdev.yaml, from which the server
// derives its Guardrail permission policy (mcp.disabled_tools, via the single
// shared middleware.PolicyFromConfig derivation qsdev_policy_check also reports
// from) and its opt-in tools (toolOptions).
//
// It fails closed on a PRESENT-but-unparseable config: a nil config there would
// silently drop every mcp.disabled_tools deny the operator intended — the exact
// fail-open the strict decoder can trigger from a single unknown/typo'd key. An
// ABSENT config is benign (nothing to narrow, nothing opted in) and yields a nil
// config with no error. errors.Is unwraps the fmt-wrapped read error so a
// missing file is detected reliably (os.IsNotExist would not, and would
// misreport a missing config as a parse failure).
func loadProjectConfig(root string) (*types.QsdevConfig, error) {
	if root == "" {
		return nil, nil
	}
	path := filepath.Join(root, branding.Get().ConfigFile)
	cfg, err := qsdevconfig.ParseQsdevConfig(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("parsing project config %s for MCP guardrail policy "+
			"(refusing to serve un-narrowed): %w", path, err)
	}
	return cfg, nil
}

// toolOptions selects the opt-in tools to mount. qsdev_credential_vend follows
// the project's security.credential_vend (off unless enabled; a nil cfg leaves
// it off). qsdev_nix_run is on, except in gateway mode, where it would run Nix
// packages on the gateway host for every framework the gateway fronts, so it
// needs the operator's gatewayNixRun opt-in.
func toolOptions(cfg *types.QsdevConfig, mode container.DeployMode, gatewayNixRun bool) tools.Options {
	opts := tools.Options{NixRun: mode != container.DeployGateway || gatewayNixRun}
	if cfg != nil {
		opts.CredentialVend = cfg.Security.CredentialVend
	}
	if !opts.NixRun {
		slog.Info("qsdev_nix_run is not mounted in gateway mode; opt in with --gateway-allow-nix-run or " + envGatewayNixRun)
	}
	return opts
}

// warnUnknownDisabledTools logs every denied tool name the server cannot mount.
// Such a deny is inert, and usually means a misspelling that leaves the tool the
// operator meant to disable runnable. `qsdev check` fails on the same names
// (config.ValidateQsdevConfig); the server still starts, with the deny
// installed, so a config written for a newer qsdev does not take it down.
func warnUnknownDisabledTools(denied, mountable []string) {
	for _, name := range denied {
		if !slices.Contains(mountable, name) {
			slog.Warn("mcp.disabled_tools names a tool this server does not provide; the entry has no effect",
				"tool", name)
		}
	}
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
