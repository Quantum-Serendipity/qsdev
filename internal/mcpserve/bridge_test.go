package mcpserve

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

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
