package mcpserve

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
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

// Deployment env keys honored as fallbacks for the corresponding flags. The
// compose template (build/docker) sets QSDEV_DEPLOY_MODE; the gateway allow-list
// is supplied via QSDEV_GATEWAY_AGENTS (comma-separated). QSDEV_GATEWAY_REQUIRE_AUTH
// forces fail-closed authentication even with an empty allow-list.
const (
	envDeployMode         = "QSDEV_DEPLOY_MODE"
	envGatewayAgents      = "QSDEV_GATEWAY_AGENTS"
	envGatewayRequireAuth = "QSDEV_GATEWAY_REQUIRE_AUTH"
)

// serveOptions bundles the resolved serve-command inputs so runServe keeps a
// short signature (go-conventions: prefer a struct over a long parameter list).
type serveOptions struct {
	transport    string
	deployMode   string
	projectRoot  string
	port         int
	multiAdapter bool
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

	if err := runTransport(ctx, srv, mode, t, opts.port); err != nil {
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
		agents := splitAgents(os.Getenv(envGatewayAgents))
		requireAuth := len(agents) > 0 || envTruthy(os.Getenv(envGatewayRequireAuth))
		return container.GatewayChain(container.GatewayOptions{
			AllowedAgents: agents,
			RequireAuth:   requireAuth,
		})
	}
	return middleware.DefaultChain()
}

// runTransport dispatches to the transport for the resolved mode. Standalone
// always serves over HTTP with a /health endpoint (orchestration needs a
// non-MCP liveness signal); the other modes honor the --transport flag.
func runTransport(ctx context.Context, srv *Server, mode container.DeployMode, t Transport, port int) error {
	addr := fmt.Sprintf(":%d", port)
	if mode == container.DeployStandalone {
		return srv.ServeHTTPWithHealth(ctx, addr)
	}
	if t == TransportHTTP {
		return srv.ServeHTTP(ctx, addr)
	}
	return srv.ServeStdio(ctx)
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
