package mcpserve

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/logging"
)

// defaultHTTPPort is the port used by the http transport when --port is unset.
const defaultHTTPPort = 8765

// Command returns the `serve` subcommand for the `qsdev mcp` command group. It
// launches the universal qsdev MCP server over the selected transport.
func Command() *cobra.Command {
	var (
		transport   string
		projectRoot string
		port        int
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the universal qsdev MCP server",
		Long: "Start the universal qsdev MCP server. It speaks the Model Context " +
			"Protocol (revision 2025-11-25) and exposes qsdev tooling to MCP " +
			"clients.\n\nThe default stdio transport reserves stdout for the JSON-RPC " +
			"protocol; all diagnostics are written to stderr.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runServe(cmd.Context(), transport, projectRoot, port)
		},
	}

	cmd.Flags().StringVar(&transport, "transport", string(TransportStdio),
		"transport to serve on: stdio or http")
	cmd.Flags().StringVar(&projectRoot, "project-root", "",
		"explicit project-root override (takes precedence over auto-detection)")
	cmd.Flags().IntVar(&port, "port", defaultHTTPPort,
		"listen port (http transport only)")

	return cmd
}

// runServe resolves the project root, initializes stderr logging, constructs the
// server, and runs it over the chosen transport until interrupted.
func runServe(ctx context.Context, transport, flagRoot string, port int) error {
	t := Transport(transport)
	if t != TransportStdio && t != TransportHTTP {
		return fmt.Errorf("unknown transport %q: want %q or %q", transport, TransportStdio, TransportHTTP)
	}

	root, err := ResolveProjectRoot(ResolveOptions{FlagRoot: flagRoot})
	if err != nil {
		return fmt.Errorf("resolving project root: %w", err)
	}

	// Force stderr logging: in stdio mode stdout carries the protocol, so every
	// diagnostic must land on stderr instead. A logging failure is non-fatal.
	session, _ := logging.Init(logging.Config{
		StderrToo:     true,
		ProjectRoot:   root,
		ProjectScoped: root != "",
	})
	defer session.Close() // Close is nil-safe.

	srv := New(WithProjectRoot(root))

	if ctx == nil {
		ctx = context.Background()
	}
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := runTransport(ctx, srv, t, port); err != nil {
		// A cancelled context is the normal way the server stops on a signal;
		// do not surface it as a command error.
		if errors.Is(err, context.Canceled) {
			return nil
		}
		return err
	}
	return nil
}

// runTransport dispatches to the selected transport.
func runTransport(ctx context.Context, srv *Server, t Transport, port int) error {
	switch t {
	case TransportHTTP:
		return srv.ServeHTTP(ctx, fmt.Sprintf(":%d", port))
	default:
		return srv.ServeStdio(ctx)
	}
}
