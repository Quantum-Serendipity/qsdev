package mcpserve

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
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

// httpShutdownTimeout bounds graceful shutdown of the HTTP transport.
const httpShutdownTimeout = 5 * time.Second

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

// ServeHTTP runs the server over Streamable HTTP on addr (e.g. ":8765") until
// ctx is cancelled, then shuts the HTTP server down gracefully.
func (s *Server) ServeHTTP(ctx context.Context, addr string) error {
	httpSrv := server.NewStreamableHTTPServer(s.mcp)

	errCh := make(chan error, 1)
	go func() { errCh <- httpSrv.Start(addr) }()

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
