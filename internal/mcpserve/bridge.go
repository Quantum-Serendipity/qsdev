package mcpserve

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
)

// mountTool mounts a generic (non-adapter) tool: it is recorded under the
// generic owner so the per-request tool filter always keeps it visible. The
// handler resolves a ToolCallContext, drives the registration through the
// middleware chain, and converts the result back.
func (s *Server) mountTool(reg spi.ToolRegistration) {
	s.mountToolOwned(reg, genericOwner)
}

// mountToolOwned records reg's name under owner in the catalog and, when that
// succeeds, adds it to the underlying server. A duplicate tool name across the
// composite surface is skipped with a logged warning rather than aborting
// construction, so a single colliding adapter cannot prevent the server from
// starting — the first registration wins.
func (s *Server) mountToolOwned(reg spi.ToolRegistration, owner string) {
	if err := s.catalog.addTool(reg.Name, owner); err != nil {
		slog.Warn("skipping tool with duplicate name", "tool", reg.Name, "owner", owner, "error", err)
		return
	}
	s.mcp.AddTool(buildMCPTool(reg), s.toolHandler(reg))
}

// buildMCPTool builds an mcp.Tool from a registration. When an InputSchema is
// supplied it is marshaled and used verbatim as the raw JSON Schema; otherwise
// a permissive empty-object schema is produced.
func buildMCPTool(reg spi.ToolRegistration) mcp.Tool {
	if len(reg.InputSchema) > 0 {
		if raw, err := json.Marshal(reg.InputSchema); err == nil {
			return mcp.NewToolWithRawSchema(reg.Name, reg.Description, raw)
		}
		// A schema that fails to marshal is a programming error in the
		// registration; fall through to a permissive schema rather than
		// panicking so a single bad tool cannot take down the server.
	}
	return mcp.NewTool(reg.Name, mcp.WithDescription(reg.Description))
}

// toolHandler returns the mcp-go handler that bridges a CallToolRequest through
// the middleware chain into the registration's handler.
func (s *Server) toolHandler(reg spi.ToolRegistration) server.ToolHandlerFunc {
	handler := reg.Handler
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		meta := metaFromMCP(req.Params.Meta)
		cc := s.callContext(ctx, reg.Name, meta)
		// Populate the tool's taxonomy metadata at construction time (before the
		// chain runs) so per-category middleware (rate-limiting, guardrail) can
		// read it. This is the only place the registration's Category/Tier are in
		// scope; resource/prompt paths legitimately have neither.
		cc.Category = reg.Category
		cc.Tier = reg.Tier

		args := req.GetArguments()
		if args == nil {
			args = map[string]any{}
		}
		sreq := &spi.ToolRequest{Name: reg.Name, Arguments: args, Meta: meta}

		res, err := s.chain.Execute(ctx, cc, sreq, handler)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("tool execution failed", err), nil
		}
		return spiResultToMCP(res), nil
	}
}

// spiResultToMCP converts a neutral tool result into an mcp-go CallToolResult.
func spiResultToMCP(res *spi.ToolResult) *mcp.CallToolResult {
	if res == nil {
		return mcp.NewToolResultError("tool returned no result")
	}
	if res.Structured != nil {
		// Preserve IsError on the structured path: a tool may return a
		// structured error payload (e.g. a not_configured object or a failed
		// scan result), and the flag must still reach the client. The mcp-go
		// constructor leaves IsError false, so set it explicitly.
		out := mcp.NewToolResultStructured(res.Structured, res.Text)
		out.IsError = res.IsError
		return out
	}
	if res.IsError {
		return mcp.NewToolResultError(res.Text)
	}
	return mcp.NewToolResultText(res.Text)
}

// mountResource mounts a generic (non-adapter) resource, recorded under the
// generic owner. mcp-go exposes no resource filter, so resource visibility is
// not scoped per client (all mounted resources are listable by every client);
// the owner is recorded only for collision detection and future use.
func (s *Server) mountResource(reg spi.ResourceRegistration) {
	s.mountResourceOwned(reg, genericOwner)
}

// mountResourceOwned records reg's URI under owner in the catalog and, when that
// succeeds, converts the neutral resource registration into mcp-go types and
// adds it to the underlying server. A duplicate URI is skipped with a logged
// warning (first registration wins). Resource reads do not pass through the
// tool middleware chain.
func (s *Server) mountResourceOwned(reg spi.ResourceRegistration, owner string) {
	if err := s.catalog.addResource(reg.URI, owner); err != nil {
		slog.Warn("skipping resource with duplicate URI", "uri", reg.URI, "owner", owner, "error", err)
		return
	}
	resource := mcp.Resource{
		URI:         reg.URI,
		Name:        reg.Name,
		Description: reg.Description,
		MIMEType:    reg.MIMEType,
	}
	handler := reg.Handler
	s.mcp.AddResource(resource, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		cc := s.callContext(ctx, reg.URI, nil)
		sreq := &spi.ResourceRequest{URI: req.Params.URI, Arguments: req.Params.Arguments}
		res, err := handler(ctx, cc, sreq)
		if err != nil {
			return nil, fmt.Errorf("reading resource %s: %w", reg.URI, err)
		}
		return resourceContentsToMCP(reg.URI, res), nil
	})
}

// resourceContentsToMCP converts neutral resource contents into mcp-go contents,
// defaulting each block's URI to the registration URI when unset.
func resourceContentsToMCP(defaultURI string, res *spi.ResourceResult) []mcp.ResourceContents {
	if res == nil {
		return nil
	}
	out := make([]mcp.ResourceContents, 0, len(res.Contents))
	for _, c := range res.Contents {
		uri := c.URI
		if uri == "" {
			uri = defaultURI
		}
		out = append(out, mcp.TextResourceContents{
			URI:      uri,
			MIMEType: c.MIMEType,
			Text:     c.Text,
		})
	}
	return out
}

// mountPrompt converts a neutral prompt registration into mcp-go types and adds
// it to the underlying server.
func (s *Server) mountPrompt(reg spi.PromptRegistration) {
	prompt := mcp.Prompt{
		Name:        reg.Name,
		Description: reg.Description,
		Arguments:   promptArgsToMCP(reg.Arguments),
	}
	handler := reg.Handler
	s.mcp.AddPrompt(prompt, func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		cc := s.callContext(ctx, reg.Name, nil)
		sreq := &spi.PromptRequest{Name: req.Params.Name, Arguments: req.Params.Arguments}
		res, err := handler(ctx, cc, sreq)
		if err != nil {
			return nil, fmt.Errorf("rendering prompt %s: %w", reg.Name, err)
		}
		return promptResultToMCP(res), nil
	})
}

func promptArgsToMCP(args []spi.PromptArgument) []mcp.PromptArgument {
	if len(args) == 0 {
		return nil
	}
	out := make([]mcp.PromptArgument, 0, len(args))
	for _, a := range args {
		out = append(out, mcp.PromptArgument{
			Name:        a.Name,
			Description: a.Description,
			Required:    a.Required,
		})
	}
	return out
}

func promptResultToMCP(res *spi.PromptResult) *mcp.GetPromptResult {
	if res == nil {
		return &mcp.GetPromptResult{}
	}
	msgs := make([]mcp.PromptMessage, 0, len(res.Messages))
	for _, m := range res.Messages {
		msgs = append(msgs, mcp.PromptMessage{
			Role:    roleToMCP(m.Role),
			Content: mcp.NewTextContent(m.Text),
		})
	}
	return &mcp.GetPromptResult{Description: res.Description, Messages: msgs}
}

func roleToMCP(r spi.PromptRole) mcp.Role {
	if r == spi.PromptRoleAssistant {
		return mcp.RoleAssistant
	}
	return mcp.RoleUser
}

// callContext builds the request-scoped ToolCallContext: it resolves the
// handshake client info from the session and the effective agent id (honoring
// the optional _meta override).
func (s *Server) callContext(ctx context.Context, name string, meta map[string]any) *spi.ToolCallContext {
	client := clientInfoFromContext(ctx)
	return &spi.ToolCallContext{
		AgentID:     resolveAgentID(client, meta),
		Client:      client,
		ProjectRoot: s.projectRoot,
		ToolName:    name,
	}
}

// clientInfoFromContext extracts the initialize-handshake client info stored on
// the active session, or the zero ClientInfo when none is available.
func clientInfoFromContext(ctx context.Context) spi.ClientInfo {
	session := server.ClientSessionFromContext(ctx)
	if withInfo, ok := session.(server.SessionWithClientInfo); ok {
		return clientInfoToSPI(withInfo.GetClientInfo())
	}
	return spi.ClientInfo{}
}
