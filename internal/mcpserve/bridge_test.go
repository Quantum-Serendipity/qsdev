package mcpserve

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/middleware"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
)

func TestBuildMCPTool(t *testing.T) {
	t.Parallel()

	t.Run("with input schema", func(t *testing.T) {
		t.Parallel()
		reg := spi.ToolRegistration{
			Name:        "search",
			Description: "search the repo",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"q": map[string]any{"type": "string"}},
				"required":   []string{"q"},
			},
		}
		tool := buildMCPTool(reg)
		if tool.Name != "search" || tool.Description != "search the repo" {
			t.Fatalf("unexpected tool meta: %+v", tool)
		}
		if len(tool.RawInputSchema) == 0 {
			t.Fatalf("expected RawInputSchema to be populated")
		}
		var decoded map[string]any
		if err := json.Unmarshal(tool.RawInputSchema, &decoded); err != nil {
			t.Fatalf("raw schema not valid JSON: %v", err)
		}
		if decoded["type"] != "object" {
			t.Errorf("raw schema type = %v, want object", decoded["type"])
		}
	})

	t.Run("without input schema", func(t *testing.T) {
		t.Parallel()
		tool := buildMCPTool(spi.ToolRegistration{Name: "ping", Description: "ping"})
		if len(tool.RawInputSchema) != 0 {
			t.Errorf("expected no RawInputSchema, got %s", tool.RawInputSchema)
		}
		if tool.InputSchema.Type != "object" {
			t.Errorf("default input schema type = %q, want object", tool.InputSchema.Type)
		}
	})
}

func TestSpiResultToMCP(t *testing.T) {
	t.Parallel()

	t.Run("nil result becomes error", func(t *testing.T) {
		t.Parallel()
		res := spiResultToMCP(nil)
		if !res.IsError {
			t.Errorf("nil result should be an error result")
		}
	})

	t.Run("text result", func(t *testing.T) {
		t.Parallel()
		res := spiResultToMCP(&spi.ToolResult{Text: "hello"})
		if res.IsError {
			t.Errorf("text result should not be an error")
		}
		if got := firstText(t, res); got != "hello" {
			t.Errorf("text = %q, want hello", got)
		}
	})

	t.Run("error result", func(t *testing.T) {
		t.Parallel()
		res := spiResultToMCP(&spi.ToolResult{Text: "boom", IsError: true})
		if !res.IsError {
			t.Errorf("expected IsError true")
		}
		if got := firstText(t, res); got != "boom" {
			t.Errorf("text = %q, want boom", got)
		}
	})

	t.Run("structured result", func(t *testing.T) {
		t.Parallel()
		res := spiResultToMCP(&spi.ToolResult{Structured: map[string]any{"a": 1}, Text: "fallback"})
		if res.StructuredContent == nil {
			t.Errorf("expected StructuredContent to be set")
		}
		if res.IsError {
			t.Errorf("non-error structured result should not be an error")
		}
	})

	t.Run("structured error preserves IsError", func(t *testing.T) {
		t.Parallel()
		res := spiResultToMCP(&spi.ToolResult{
			Structured: map[string]any{"status": "not_configured"},
			Text:       `{"status":"not_configured"}`,
			IsError:    true,
		})
		if res.StructuredContent == nil {
			t.Errorf("expected StructuredContent to be set")
		}
		if !res.IsError {
			t.Errorf("structured error must surface IsError true to the client")
		}
	})
}

func TestServerCallContext(t *testing.T) {
	t.Parallel()

	srv := New(WithProjectRoot("/proj"))

	t.Run("meta override with no session", func(t *testing.T) {
		t.Parallel()
		cc := srv.callContext(context.Background(), "toolX", map[string]any{MetaAgentIDKey: "override"})
		if cc.AgentID != "override" {
			t.Errorf("AgentID = %q, want override", cc.AgentID)
		}
		if cc.ProjectRoot != "/proj" {
			t.Errorf("ProjectRoot = %q, want /proj", cc.ProjectRoot)
		}
		if cc.ToolName != "toolX" {
			t.Errorf("ToolName = %q, want toolX", cc.ToolName)
		}
	})

	t.Run("no session, no meta yields unknown", func(t *testing.T) {
		t.Parallel()
		cc := srv.callContext(context.Background(), "toolY", nil)
		if cc.AgentID != unknownAgentID {
			t.Errorf("AgentID = %q, want %q", cc.AgentID, unknownAgentID)
		}
		if (cc.Client != spi.ClientInfo{}) {
			t.Errorf("expected zero ClientInfo without a session, got %+v", cc.Client)
		}
	})
}

// TestResourceReadCategoryEnforced is the BUG A regression: a resource
// registered WITH a category is now routed through the Guardrail under that
// category, so a category-scoped deny blocks the read. Before the fix a resource
// carried no category, fell through to Default=Allow, and was admitted despite
// the deny — the "uncategorized" sub-test pins that pre-fix behavior as the
// contrast.
func TestResourceReadCategoryEnforced(t *testing.T) {
	t.Parallel()

	const deniedCat = middleware.CategoryEnvironment
	policy := &middleware.Policy{
		ByToolType: map[string]middleware.Verdict{deniedCat: middleware.VerdictDeny},
		Default:    middleware.VerdictAllow,
	}
	srv := New(WithProjectRoot("/proj"),
		WithChain(middleware.DefaultChain(middleware.WithPolicy(policy))))

	handler := func(_ context.Context, _ *spi.ToolCallContext, _ *spi.ResourceRequest) (*spi.ResourceResult, error) {
		return &spi.ResourceResult{Contents: []spi.ResourceContent{{URI: "qsdev://env", Text: "ok"}}}, nil
	}
	readOf := func(reg spi.ResourceRegistration) ([]mcp.ResourceContents, error) {
		read := srv.resourceReadHandler(reg)
		var req mcp.ReadResourceRequest
		req.Params.URI = reg.URI
		return read(context.Background(), req)
	}

	t.Run("category-denied resource is blocked", func(t *testing.T) {
		t.Parallel()
		_, err := readOf(spi.ResourceRegistration{
			URI: "qsdev://env", Name: "env", Category: deniedCat, Handler: handler,
		})
		if err == nil {
			t.Fatal("resource with a denied category was allowed; the category was not plumbed to the Guardrail")
		}
	})

	t.Run("uncategorized resource still allowed under the same deny", func(t *testing.T) {
		t.Parallel()
		out, err := readOf(spi.ResourceRegistration{
			URI: "qsdev://env", Name: "env", Handler: handler, // empty Category
		})
		if err != nil {
			t.Fatalf("uncategorized resource was blocked under a category-scoped deny: %v", err)
		}
		if len(out) == 0 {
			t.Fatal("expected resource contents from the allowed read")
		}
	})
}

func firstText(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	if len(res.Content) == 0 {
		t.Fatalf("result has no content")
	}
	tc, ok := res.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("first content is not text: %T", res.Content[0])
	}
	return tc.Text
}
