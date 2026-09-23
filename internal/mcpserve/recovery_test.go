package mcpserve

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
)

// panickingMiddleware panics as the outermost chain layer, standing in for a
// faulty AuditSink or redaction edge case that sits outside ErrorHandling.
type panickingMiddleware struct{}

func (panickingMiddleware) Order() int { return 1 }

func (panickingMiddleware) Handle(context.Context, *spi.ToolCallContext, *spi.ToolRequest, spi.ToolHandler) (*spi.ToolResult, error) {
	panic("middleware exploded")
}

// TestResourceAndPromptPanicsAreRecovered is the regression test for missing
// panic recovery outside tool calls: a panic in the middleware chain during
// resources/read or prompts/get must become a JSON-RPC error rather than crash
// the server (over stdio these are served inline in the read loop).
func TestResourceAndPromptPanicsAreRecovered(t *testing.T) {
	t.Parallel()
	srv := New(WithProjectRoot(t.TempDir()), WithChain(spi.NewChain(panickingMiddleware{})))
	srv.mountResource(spi.ResourceRegistration{
		URI: "qsdev://test/panic", Name: "panic",
		Handler: func(context.Context, *spi.ToolCallContext, *spi.ResourceRequest) (*spi.ResourceResult, error) {
			return &spi.ResourceResult{}, nil
		},
	})
	srv.mountPrompt(spi.PromptRegistration{
		Name: "panic",
		Handler: func(context.Context, *spi.ToolCallContext, *spi.PromptRequest) (*spi.PromptResult, error) {
			return &spi.PromptResult{}, nil
		},
	})

	tests := []struct {
		name    string
		message string
	}{
		{"resources/read", `{"jsonrpc":"2.0","id":1,"method":"resources/read","params":{"uri":"qsdev://test/panic"}}`},
		{"prompts/get", `{"jsonrpc":"2.0","id":2,"method":"prompts/get","params":{"name":"panic"}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resp := srv.MCPServer().HandleMessage(context.Background(), json.RawMessage(tt.message))
			if _, ok := resp.(mcp.JSONRPCError); !ok {
				t.Errorf("%s response = %T (%+v), want a JSON-RPC error", tt.name, resp, resp)
			}
		})
	}
}
