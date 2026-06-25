package mcpserve

import (
	"context"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
)

// fakeClientSession is a minimal ClientSession that also satisfies
// server.SessionWithClientInfo so clientInfoFromContext can resolve a reported
// client name without a live transport.
type fakeClientSession struct {
	info mcp.Implementation
}

func (f *fakeClientSession) Initialize()                                         {}
func (f *fakeClientSession) Initialized() bool                                   { return true }
func (f *fakeClientSession) NotificationChannel() chan<- mcp.JSONRPCNotification { return nil }
func (f *fakeClientSession) SessionID() string                                   { return "test-session" }
func (f *fakeClientSession) GetClientInfo() mcp.Implementation                   { return f.info }
func (f *fakeClientSession) SetClientInfo(info mcp.Implementation)               { f.info = info }
func (f *fakeClientSession) GetClientCapabilities() mcp.ClientCapabilities {
	return mcp.ClientCapabilities{}
}
func (f *fakeClientSession) SetClientCapabilities(mcp.ClientCapabilities) {}

var _ server.SessionWithClientInfo = (*fakeClientSession)(nil)

// fakeFrameworkAdapter is a self-contained FrameworkAdapter + ClientMatcher used
// to drive the tool filter without importing a concrete adapter package (which
// would create an import cycle: a concrete adapter pulls in its addon, and the
// addon imports this package). It stands in for the Claude Code adapter: it
// owns one tool and matches any client whose name contains "claude".
type fakeFrameworkAdapter struct {
	id      aiframework.FrameworkID
	applies bool
	tools   []spi.ToolRegistration
	match   func(spi.ClientInfo) bool
}

func (a fakeFrameworkAdapter) ID() aiframework.FrameworkID          { return a.id }
func (a fakeFrameworkAdapter) Applies(context.Context, string) bool { return a.applies }
func (a fakeFrameworkAdapter) Tools() []spi.ToolRegistration        { return a.tools }
func (a fakeFrameworkAdapter) Resources() []spi.ResourceRegistration {
	return nil
}
func (a fakeFrameworkAdapter) Prompts() []spi.PromptRegistration   { return nil }
func (a fakeFrameworkAdapter) MatchesClient(c spi.ClientInfo) bool { return a.match(c) }

var (
	_ spi.FrameworkAdapter = fakeFrameworkAdapter{}
	_ spi.ClientMatcher    = fakeFrameworkAdapter{}
)

// fakeContributor mounts a single generic tool so the filter has a non-adapter
// tool to keep always-visible.
type fakeContributor struct {
	tool spi.ToolRegistration
}

func (c fakeContributor) Tools() []spi.ToolRegistration         { return []spi.ToolRegistration{c.tool} }
func (c fakeContributor) Resources() []spi.ResourceRegistration { return nil }
func (c fakeContributor) Prompts() []spi.PromptRegistration     { return nil }

const (
	genericToolName = "qsdev_test_generic"
	ccToolName      = "qsdev_cc_fake"
)

func okHandler(context.Context, *spi.ToolCallContext, *spi.ToolRequest) (*spi.ToolResult, error) {
	return &spi.ToolResult{Text: "ok"}, nil
}

// ccAdapter returns the Claude Code stand-in adapter with the given Applies
// result. It matches any client whose name contains "claude".
func ccAdapter(applies bool) fakeFrameworkAdapter {
	return fakeFrameworkAdapter{
		id:      aiframework.ClaudeCode,
		applies: applies,
		tools:   []spi.ToolRegistration{{Name: ccToolName, Description: "cc tool", Handler: okHandler}},
		match: func(c spi.ClientInfo) bool {
			return strings.Contains(strings.ToLower(c.Name), "claude")
		},
	}
}

// newFilterServer builds a Server whose registry contains only the CC stand-in
// adapter (Applies==true), plus one generic tool. multi toggles --multi-adapter.
func newFilterServer(t *testing.T, multi bool) *Server {
	t.Helper()
	reg := spi.NewAdapterRegistry()
	if err := reg.Register(ccAdapter(true)); err != nil {
		t.Fatalf("registering cc adapter: %v", err)
	}
	srv := New(
		WithProjectRoot(t.TempDir()),
		WithAdapterRegistry(reg),
		WithMultiAdapter(multi),
	)
	// Mount one generic tool through the normal seam so it is recorded under the
	// generic owner exactly as production does.
	srv.MountProjectContext(fakeContributor{tool: spi.ToolRegistration{
		Name:        genericToolName,
		Description: "generic test tool",
		Handler:     okHandler,
	}})
	return srv
}

// ctxWithClient returns a context carrying a session that reports the given
// client name, using the same context seam mcp-go populates in production.
func ctxWithClient(srv *Server, name string) context.Context {
	sess := &fakeClientSession{info: mcp.Implementation{Name: name}}
	return srv.mcp.WithContext(context.Background(), sess)
}

// toolNames extracts the names from a slice of mcp tools for assertions.
func toolNames(tools []mcp.Tool) map[string]bool {
	out := make(map[string]bool, len(tools))
	for _, t := range tools {
		out[t.Name] = true
	}
	return out
}

func allTools() []mcp.Tool {
	return []mcp.Tool{{Name: genericToolName}, {Name: ccToolName}}
}

func TestToolFilterClaudeClient(t *testing.T) {
	t.Parallel()
	srv := newFilterServer(t, false)

	got := toolNames(srv.toolFilter(ctxWithClient(srv, "claude-code"), allTools()))
	if !got[genericToolName] {
		t.Errorf("generic tool %q hidden from claude client", genericToolName)
	}
	if !got[ccToolName] {
		t.Errorf("CC tool %q hidden from claude client", ccToolName)
	}

	// Title-cased name resolves identically through the CC adapter's matcher.
	got2 := toolNames(srv.toolFilter(ctxWithClient(srv, "Claude Code"), allTools()))
	if !got2[ccToolName] {
		t.Errorf("CC tool %q hidden from 'Claude Code' client", ccToolName)
	}
}

func TestToolFilterUnknownClient(t *testing.T) {
	t.Parallel()
	srv := newFilterServer(t, false)

	got := toolNames(srv.toolFilter(ctxWithClient(srv, "someide"), allTools()))
	if !got[genericToolName] {
		t.Errorf("generic tool %q must stay visible for any client", genericToolName)
	}
	if got[ccToolName] {
		t.Errorf("CC tool %q leaked to unrelated client", ccToolName)
	}
}

func TestToolFilterNoSession(t *testing.T) {
	t.Parallel()
	srv := newFilterServer(t, false)

	// No session in context → zero ClientInfo → generic-only fallback.
	got := toolNames(srv.toolFilter(context.Background(), allTools()))
	if !got[genericToolName] {
		t.Errorf("generic tool %q hidden in fallback mode", genericToolName)
	}
	if got[ccToolName] {
		t.Errorf("CC tool %q visible without a matching client", ccToolName)
	}
}

func TestToolFilterMultiAdapter(t *testing.T) {
	t.Parallel()
	srv := newFilterServer(t, true)

	// multi-adapter mode short-circuits client matching: everything is visible,
	// even for a client that matches no framework and even with no session.
	got := toolNames(srv.toolFilter(ctxWithClient(srv, "someide"), allTools()))
	if !got[genericToolName] || !got[ccToolName] {
		t.Errorf("multi-adapter must expose all tools, got %v", got)
	}
	got2 := toolNames(srv.toolFilter(context.Background(), allTools()))
	if !got2[genericToolName] || !got2[ccToolName] {
		t.Errorf("multi-adapter must expose all tools without a session, got %v", got2)
	}
}

func TestMultiAdapterMountsRegardlessOfApplies(t *testing.T) {
	t.Parallel()
	// multi-adapter on: the adapter mounts even though Applies()==false.
	regOn := spi.NewAdapterRegistry()
	if err := regOn.Register(ccAdapter(false)); err != nil {
		t.Fatalf("register: %v", err)
	}
	srvOn := New(WithProjectRoot(t.TempDir()), WithAdapterRegistry(regOn), WithMultiAdapter(true))
	if owner, ok := srvOn.catalog.toolOwnerOf(ccToolName); !ok || owner != string(aiframework.ClaudeCode) {
		t.Errorf("multi-adapter: CC tool %q owner=(%q,%v), want %q mounted", ccToolName, owner, ok, aiframework.ClaudeCode)
	}

	// multi-adapter off + Applies()==false: the adapter is skipped at mount.
	regOff := spi.NewAdapterRegistry()
	if err := regOff.Register(ccAdapter(false)); err != nil {
		t.Fatalf("register: %v", err)
	}
	srvOff := New(WithProjectRoot(t.TempDir()), WithAdapterRegistry(regOff), WithMultiAdapter(false))
	if _, ok := srvOff.catalog.toolOwnerOf(ccToolName); ok {
		t.Errorf("CC tool %q mounted despite Applies()==false and multi-adapter off", ccToolName)
	}
}

func TestToolFilterUntrackedToolVisible(t *testing.T) {
	t.Parallel()
	srv := newFilterServer(t, false)
	// A tool the catalog never recorded must fail open (stay visible) rather
	// than be hidden.
	got := toolNames(srv.toolFilter(ctxWithClient(srv, "someide"), []mcp.Tool{{Name: "never_mounted"}}))
	if !got["never_mounted"] {
		t.Error("untracked tool was hidden; filter must fail open for unrecorded tools")
	}
}
