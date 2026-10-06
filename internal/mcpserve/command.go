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

	qsdevcatalog "github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/claudesettings"
	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	qsdevconfig "github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/internal/logging"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpregistry"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/container"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/middleware"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/projectctx"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/canon"
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
// forces fail-closed authentication even with an empty allow-list. QSDEV_BIND
// overrides the bind host (default loopback). The mTLS material falls back to
// the EnvTLS* keys defined in tlsconfig.go.
//
// The opt-ins for the gated tools (see mountOptions) fall back to env keys as
// well: QSDEV_MCP_ALLOW_NIX_RUN and QSDEV_MCP_ALLOW_CREDENTIAL_VEND are the
// operator's base opt-ins, and QSDEV_GATEWAY_ALLOW_NIX_RUN and
// QSDEV_GATEWAY_ALLOW_CREDENTIAL_VEND the deployment opt-ins the gateway and
// standalone modes also need.
const (
	envDeployMode            = container.EnvDeployMode
	envGatewayAgents         = container.EnvGatewayAgents
	envGatewayRequireAuth    = "QSDEV_GATEWAY_REQUIRE_AUTH"
	envAllowNixRun           = "QSDEV_MCP_ALLOW_NIX_RUN"
	envAllowCredentialVend   = "QSDEV_MCP_ALLOW_CREDENTIAL_VEND"
	envGatewayNixRun         = "QSDEV_GATEWAY_ALLOW_NIX_RUN"
	envGatewayCredentialVend = "QSDEV_GATEWAY_ALLOW_CREDENTIAL_VEND"
	envBind                  = "QSDEV_BIND"
)

// Names of the opt-in sources, as the startup log and its hints spell them.
const (
	flagAllowNixRun           = "--allow-nix-run"
	flagAllowCredentialVend   = "--allow-credential-vend"
	flagGatewayNixRun         = "--gateway-allow-nix-run"
	flagGatewayCredentialVend = "--gateway-allow-credential-vend"
	userAllowNixRun           = "mcp_serve.allow_nix_run"
	userAllowCredentialVend   = "mcp_serve.allow_credential_vend"
	flagHTTPNoAuth            = "--http-no-auth"
	flagHTTPTokenFile         = "--http-token-file"
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
	// httpTokenFile is where plain HTTP publishes its bearer token; empty
	// means the default under the user state directory.
	httpTokenFile string
	// httpNoAuth serves plain HTTP without the bearer token.
	httpNoAuth bool
	// optIns are the opt-in flags for the gated tools (mountOptions).
	optIns optInFlags
	// modules restricts the server to the named tool modules (tools.Select);
	// empty serves the full universal surface.
	modules []string
	// trustedServers are the MCP server definitions configured into the binary,
	// trusted for mcp.list health probes alongside the catalog's.
	trustedServers map[string][]mcpregistry.LaunchSpec
}

// CommandOption configures the serve command.
type CommandOption func(*serveOptions)

// WithTrustedServers adds the MCP server definitions configured into the
// binary to the set the project context surface trusts for health probes, so
// mcp.list probes exactly what `qsdev mcp status --probe` would.
func WithTrustedServers(specs map[string][]mcpregistry.LaunchSpec) CommandOption {
	return func(o *serveOptions) { o.trustedServers = specs }
}

// Command returns the `serve` subcommand for the `qsdev mcp` command group. It
// launches the universal qsdev MCP server over the selected transport.
func Command(cmdOpts ...CommandOption) *cobra.Command {
	var opts serveOptions
	for _, o := range cmdOpts {
		o(&opts)
	}

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
			"  standalone  requires an explicit --project-root and serves /health\n\n" +
			"qsdev_nix_run and qsdev_credential_vend are off unless the operator opts\n" +
			"in: --allow-nix-run, " + envAllowNixRun + " or " + userAllowNixRun + "\n" +
			"in the user defaults file; credential vending also needs an enabled\n" +
			"security.credential_vend in .qsdev.yaml, confirmed by\n" +
			"--allow-credential-vend, " + envAllowCredentialVend + " or\n" +
			userAllowCredentialVend + ". Gateway and standalone also need\n" +
			"--gateway-allow-nix-run / --gateway-allow-credential-vend.\n\n" +
			"Plain HTTP (no mTLS) requires a per-launch bearer token, written to a\n" +
			"file only the user can read (" + flagHTTPTokenFile + ", default\n" +
			"<user state dir>/mcp/<bound port>.token); " + flagHTTPNoAuth + " serves without it\n" +
			"and then mounts neither tool above.\n\n" +
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
	cmd.Flags().StringVar(&opts.httpTokenFile, strings.TrimPrefix(flagHTTPTokenFile, "--"), "",
		"plain HTTP: file the per-launch bearer token is written to (mode 0600, removed "+
			"on shutdown); default <user state dir>/mcp/<bound port>.token")
	cmd.Flags().BoolVar(&opts.httpNoAuth, strings.TrimPrefix(flagHTTPNoAuth, "--"), false,
		"plain HTTP: serve without the bearer token, so any local process can call the "+
			"server; qsdev_nix_run and qsdev_credential_vend are then never mounted")
	addOptInFlags(cmd, &opts.optIns)

	return cmdutil.MarkProfile(cmd, cmdutil.ProfileMCPServer)
}

// addOptInFlags registers the gated tools' opt-in flags on cmd.
func addOptInFlags(cmd *cobra.Command, f *optInFlags) {
	cmd.Flags().BoolVar(&f.allowNixRun, strings.TrimPrefix(flagAllowNixRun, "--"), false,
		"mount qsdev_nix_run, which runs Nix packages as processes on this host; "+
			"falls back to "+envAllowNixRun+" and "+userAllowNixRun+" in the user defaults file")
	cmd.Flags().BoolVar(&f.allowCredentialVend, strings.TrimPrefix(flagAllowCredentialVend, "--"), false,
		"confirm the committed security.credential_vend and mount qsdev_credential_vend; "+
			"falls back to "+envAllowCredentialVend+" and "+userAllowCredentialVend+" in the user defaults file")
	cmd.Flags().BoolVar(&f.gatewayNixRun, strings.TrimPrefix(flagGatewayNixRun, "--"), false,
		"gateway and standalone modes: also allow qsdev_nix_run, which there runs "+
			"Nix packages for remote clients; falls back to "+envGatewayNixRun)
	cmd.Flags().BoolVar(&f.gatewayCredentialVend, strings.TrimPrefix(flagGatewayCredentialVend, "--"), false,
		"gateway and standalone modes: also allow qsdev_credential_vend; the gateway "+
			"needs a non-empty "+envGatewayAgents+"; falls back to "+envGatewayCredentialVend)
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
	return cmdutil.MarkProfile(&cobra.Command{
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
	}, cmdutil.ProfileMCPServer)
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
	token, err := resolveHTTPToken(mode, t, tlsConfig != nil, opts)
	if err != nil {
		return err
	}

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
	// startup before anything is served. The gated tools are included only
	// when mountOptions selects them, from the operator's opt-ins: the flags,
	// the env and the user-scope catalog, never the project's qsdev config.
	userScope, err := qsdevcatalog.LoadUserScope()
	if err != nil {
		return fmt.Errorf("loading the user defaults for the MCP tool opt-ins: %w", err)
	}
	unauthenticated := servesOverHTTP(mode, t) && tlsConfig == nil && token == nil
	if unauthenticated {
		slog.Warn("serving plain HTTP without authentication ("+flagHTTPNoAuth+"): any local process can call the server",
			"addr", addr)
	}
	toolOpts, err := mountOptions(mountInputs{
		cfg:                 cfg,
		mode:                mode,
		user:                userScope.MCPServeOptIns(),
		flags:               opts.optIns,
		getenv:              os.Getenv,
		unauthenticatedHTTP: unauthenticated,
	})
	if err != nil {
		return err
	}
	toolOpts, err = withNixRunDenyRules(toolOpts, func() ([]string, error) {
		return nixRunDenyRules(root, userScope, canon.ClaudeConfigDir)
	})
	if err != nil {
		return err
	}
	regs, err := tools.Select(opts.modules, root, policy, toolOpts)
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

	if err := runTransport(ctx, srv, mode, t, addr, tlsConfig, token); err != nil {
		// A cancelled context is the normal way the server stops on a signal;
		// do not surface it as a command error.
		if errors.Is(err, context.Canceled) {
			return nil
		}
		return err
	}
	return nil
}

// resolveHTTPToken returns the bearer token the server requires, or nil when
// it needs none: stdio is a local pipe, mTLS authenticates by client
// certificate, and --http-no-auth is the operator's explicit escape hatch.
// Plain HTTP otherwise gets a fresh token, published to --http-token-file or
// the default path under the user state directory once the listener is bound.
func resolveHTTPToken(mode container.DeployMode, t Transport, mtls bool, opts serveOptions) (*httpToken, error) {
	if !servesOverHTTP(mode, t) || mtls || opts.httpNoAuth {
		return nil, nil
	}
	// An empty --http-token-file leaves the path to publish, which names
	// the port the listener bound (see httpToken.publish).
	return newHTTPToken(opts.httpTokenFile)
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
	if pc, err := projectctx.NewProjectContext(root, projectctx.WithTrustedServers(opts.trustedServers)); err != nil {
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
// from) and the allow-lists of its credential-vending tool (mountOptions).
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

// optInFlags are the serve command's opt-in flags for the gated tools.
type optInFlags struct {
	allowNixRun           bool // --allow-nix-run
	allowCredentialVend   bool // --allow-credential-vend
	gatewayNixRun         bool // --gateway-allow-nix-run
	gatewayCredentialVend bool // --gateway-allow-credential-vend
}

// mountInputs are everything mountOptions decides from.
type mountInputs struct {
	// cfg is the committed .qsdev.yaml (nil when absent). It supplies the
	// credential-vending allow-lists, never an opt-in.
	cfg  *types.QsdevConfig
	mode container.DeployMode
	// user are the opt-ins of the user-scope catalog (the org overlay).
	user  qsdevcatalog.MCPServeOptIns
	flags optInFlags
	// getenv reads the env fallbacks; nil means os.Getenv.
	getenv func(string) string
	// logger receives one line per decision; nil means slog.Default().
	logger *slog.Logger
	// unauthenticatedHTTP is set when the server listens on plain HTTP with
	// --http-no-auth, where any local process can call it: neither gated tool
	// is mounted then, whatever the opt-ins.
	unauthenticatedHTTP bool
}

// optInSource is one way the operator can opt a gated tool in.
type optInSource struct {
	name string
	set  bool
}

// firstOptIn returns the name of the first source that is set, or "".
func firstOptIn(sources ...optInSource) string {
	for _, src := range sources {
		if src.set {
			return src.name
		}
	}
	return ""
}

// mountOptions decides which gated tools the server mounts, and logs each
// decision once. Both tools are off unless the operator opts in, from a flag,
// the env, or the mcp_serve section of the user-scope catalog, never from the
// project's qsdev configuration (.qsdev.yaml, .qsdev/defaults.yaml). A
// committed .mcp.json can still carry a flag or env opt-in for the qsdev
// entry, and a committed devenv.nix or .envrc can export the env key. That
// adds nothing to what either file can already do (run any command once the
// user approves the project's MCP servers or allows direnv), so those two
// approvals are the trust boundary for these opt-ins, as for the files:
//
//   - qsdev_nix_run starts processes on the host: --allow-nix-run,
//     QSDEV_MCP_ALLOW_NIX_RUN or mcp_serve.allow_nix_run.
//   - qsdev_credential_vend hands out cloud credentials: the committed
//     security.credential_vend must enable it (its allow-lists are the team's,
//     protected by self-protection), and the operator must confirm with
//     --allow-credential-vend, QSDEV_MCP_ALLOW_CREDENTIAL_VEND or
//     mcp_serve.allow_credential_vend. The committed config alone logs a
//     warning naming those.
//
// The gateway and standalone modes serve other clients, so each tool there
// also needs its deployment opt-in (--gateway-allow-nix-run /
// --gateway-allow-credential-vend or their env keys). Gateway credential
// vending without an agent allow-list (QSDEV_GATEWAY_AGENTS) is a startup
// error. Plain HTTP served with --http-no-auth mounts neither tool.
func mountOptions(in mountInputs) (tools.Options, error) {
	if in.getenv == nil {
		in.getenv = os.Getenv
	}
	if in.logger == nil {
		in.logger = slog.Default()
	}
	vend, err := mountCredentialVend(in)
	if err != nil {
		return tools.Options{}, err
	}
	return tools.Options{NixRun: mountNixRun(in), CredentialVend: vend}, nil
}

// unauthenticatedHTTPReason is why --http-no-auth leaves the gated tools out.
const unauthenticatedHTTPReason = "plain HTTP is served with " + flagHTTPNoAuth +
	", so any local process could call it; drop " + flagHTTPNoAuth + " to require the bearer token"

// deploymentOptIn returns the deployment opt-in source (the flag or its env
// key) the gateway and standalone modes need, or "" when neither is set.
func deploymentOptIn(in mountInputs, flag bool, flagName, env string) string {
	return firstOptIn(
		optInSource{flagName, flag},
		optInSource{env, envTruthy(in.getenv(env))},
	)
}

// mountNixRun decides whether qsdev_nix_run is mounted (see mountOptions).
func mountNixRun(in mountInputs) bool {
	base := firstOptIn(
		optInSource{flagAllowNixRun, in.flags.allowNixRun},
		optInSource{envAllowNixRun, envTruthy(in.getenv(envAllowNixRun))},
		optInSource{userAllowNixRun, in.user.AllowNixRun},
	)
	if base == "" {
		in.logger.Info("qsdev_nix_run not mounted: it runs Nix packages as processes on this host; opt in with "+
			flagAllowNixRun+", "+envAllowNixRun+" or "+userAllowNixRun+" in the user defaults file",
			"mode", in.mode)
		return false
	}
	var deploy string
	if in.mode != container.DeployNative {
		deploy = deploymentOptIn(in, in.flags.gatewayNixRun, flagGatewayNixRun, envGatewayNixRun)
		if deploy == "" {
			in.logger.Info("qsdev_nix_run not mounted: the "+string(in.mode)+" mode also needs "+
				flagGatewayNixRun+" or "+envGatewayNixRun, "mode", in.mode, "opt_in", base)
			return false
		}
	}
	if in.unauthenticatedHTTP {
		in.logger.Warn("qsdev_nix_run not mounted: "+unauthenticatedHTTPReason, "mode", in.mode, "opt_in", base)
		return false
	}
	in.logger.Info("qsdev_nix_run mounted", "mode", in.mode, "opt_in", base, "deployment_opt_in", deploy)
	return true
}

// mountCredentialVend returns the credential-vending config to mount: the
// committed security.credential_vend when every opt-in holds (see
// mountOptions), otherwise the zero config, which mounts nothing.
func mountCredentialVend(in mountInputs) (types.CredentialVendConfig, error) {
	var deploy string
	if in.mode != container.DeployNative {
		deploy = deploymentOptIn(in, in.flags.gatewayCredentialVend, flagGatewayCredentialVend, envGatewayCredentialVend)
	}
	if in.mode == container.DeployGateway && deploy != "" && len(splitAgents(in.getenv(envGatewayAgents))) == 0 {
		return types.CredentialVendConfig{}, fmt.Errorf("%s needs a non-empty %s: without an agent "+
			"allow-list the gateway would vend credentials to any caller", deploy, envGatewayAgents)
	}

	var committed types.CredentialVendConfig
	if in.cfg != nil {
		committed = in.cfg.Security.CredentialVend
	}
	confirm := firstOptIn(
		optInSource{flagAllowCredentialVend, in.flags.allowCredentialVend},
		optInSource{envAllowCredentialVend, envTruthy(in.getenv(envAllowCredentialVend))},
		optInSource{userAllowCredentialVend, in.user.AllowCredentialVend},
	)
	switch {
	case !committed.Enabled:
		in.logger.Info("qsdev_credential_vend not mounted: the committed config does not enable "+
			"security.credential_vend", "mode", in.mode, "opt_in", confirm)
		return types.CredentialVendConfig{}, nil
	case confirm == "":
		in.logger.Warn("qsdev_credential_vend not mounted: the committed config enables "+
			"security.credential_vend, but the operator has not confirmed it; confirm with "+
			flagAllowCredentialVend+", "+envAllowCredentialVend+" or "+userAllowCredentialVend+
			" in the user defaults file", "mode", in.mode)
		return types.CredentialVendConfig{}, nil
	case in.mode != container.DeployNative && deploy == "":
		in.logger.Warn("qsdev_credential_vend not mounted: the "+string(in.mode)+" mode also needs "+
			flagGatewayCredentialVend+" or "+envGatewayCredentialVend, "mode", in.mode, "opt_in", confirm)
		return types.CredentialVendConfig{}, nil
	case in.unauthenticatedHTTP:
		in.logger.Warn("qsdev_credential_vend not mounted: "+unauthenticatedHTTPReason, "mode", in.mode, "opt_in", confirm)
		return types.CredentialVendConfig{}, nil
	}
	in.logger.Info("qsdev_credential_vend mounted", "mode", in.mode, "opt_in", confirm, "deployment_opt_in", deploy)
	return committed, nil
}

// withNixRunDenyRules sets opts.NixRunDenyRules from load when qsdev_nix_run
// is mounted, and leaves opts as is otherwise, so a server without the tool
// reads nothing for it. A mounted tool whose rules cannot be loaded fails
// startup rather than running unchecked.
func withNixRunDenyRules(opts tools.Options, load func() ([]string, error)) (tools.Options, error) {
	if !opts.NixRun {
		return opts, nil
	}
	rules, err := load()
	if err != nil {
		return tools.Options{}, fmt.Errorf("building the Bash deny rules qsdev_nix_run is checked against: %w", err)
	}
	opts.NixRunDenyRules = rules
	return opts, nil
}

// nixRunDenyRules returns the deny rules qsdev_nix_run checks each call
// against: the user-scope catalog's deny rules (every set in
// permission_all_deny_sets) and those of the Claude Code settings an agent in
// this project runs under, the user file in the directory claudeDir returns
// and the project's committed and local files. The tool reaches the host the
// way Bash does, so it is held to the same deny rules; the ask rules are not
// used, since `Bash(nix run *)` asks for every call and the client's own
// prompt for the MCP tool is its ask.
func nixRunDenyRules(root string, userScope *qsdevcatalog.Catalog, claudeDir func() (string, error)) ([]string, error) {
	dir, err := claudeDir()
	if err != nil {
		return nil, fmt.Errorf("resolving the Claude Code user directory: %w", err)
	}
	settings, err := claudesettings.ReadWith(root, claudesettings.ReadOptions{UserDir: dir})
	if err != nil {
		return nil, fmt.Errorf("reading the Claude Code settings: %w", err)
	}
	return slices.Concat(userScope.AllPermissionDenyRules(), settings.Deny), nil
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
// HTTP / stdio); both are produced and validated by runServe. token is the
// bearer token plain HTTP requires (nil for none, see resolveHTTPToken).
func runTransport(ctx context.Context, srv *Server, mode container.DeployMode, t Transport, addr string, tlsConfig *tls.Config, token *httpToken) error {
	if mode == container.DeployStandalone {
		return srv.ServeHTTPWithHealth(ctx, addr, tlsConfig, token)
	}
	if t == TransportHTTP {
		return srv.ServeHTTP(ctx, addr, tlsConfig, token)
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
