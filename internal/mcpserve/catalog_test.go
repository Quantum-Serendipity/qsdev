package mcpserve

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
)

// TestGenericToolReclaimsAdapterName is the server-level regression test for an
// adapter (mounted at construction, before the generic surfaces) claiming a
// generic tool name: the later generic registration must win, so the served
// handler is the generic one and the tool is recorded under the generic owner.
func TestGenericToolReclaimsAdapterName(t *testing.T) {
	t.Parallel()
	const shared = "qsdev_credential_vend"
	textHandler := func(text string) spi.ToolHandler {
		return func(context.Context, *spi.ToolCallContext, *spi.ToolRequest) (*spi.ToolResult, error) {
			return &spi.ToolResult{Text: text}, nil
		}
	}
	adapter := ccAdapter(true)
	adapter.tools = []spi.ToolRegistration{{Name: shared, Handler: textHandler("adapter")}}
	reg := spi.NewAdapterRegistry()
	if err := reg.Register(adapter); err != nil {
		t.Fatalf("registering adapter: %v", err)
	}
	srv := New(WithProjectRoot(t.TempDir()), WithAdapterRegistry(reg))
	srv.MountTools([]spi.ToolRegistration{{Name: shared, Handler: textHandler("generic")}})

	if owner, _ := srv.catalog.toolOwnerOf(shared); owner != genericOwner {
		t.Errorf("owner = %q, want %q", owner, genericOwner)
	}
	st := srv.MCPServer().GetTool(shared)
	if st == nil {
		t.Fatal("shared tool not mounted")
	}
	res, err := st.Handler(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("calling tool: %v", err)
	}
	if text := res.Content[0].(mcp.TextContent).Text; text != "generic" {
		t.Errorf("served handler returned %q, want the generic tool's output", text)
	}
}

// TestDuplicatePromptSkipped is the regression test for prompts bypassing
// collision detection: a second adapter prompt with the same name must be
// skipped (first wins) instead of silently replacing the first.
func TestDuplicatePromptSkipped(t *testing.T) {
	t.Parallel()
	srv := New(WithProjectRoot(t.TempDir()), WithAdapterRegistry(spi.NewAdapterRegistry()))
	srv.mountPromptOwned(spi.PromptRegistration{Name: "qsdev_review", Description: "first"}, "claudecode")
	srv.mountPromptOwned(spi.PromptRegistration{Name: "qsdev_review", Description: "second"}, "cursor")

	p, ok := srv.MCPServer().ListPrompts()["qsdev_review"]
	if !ok {
		t.Fatal("prompt not mounted")
	}
	if p.Prompt.Description != "first" {
		t.Errorf("description = %q, want the first registration to win", p.Prompt.Description)
	}
}

// TestCatalogCollisionPolicy covers the per-surface collision policy on all
// three surfaces: between adapters the first registration wins and the later
// one errors; a generic registration reclaims a key an adapter already holds
// (generic names are reserved, so an adapter mounted first cannot shadow a
// generic tool and inherit name-keyed policy); a generic duplicate of a generic
// key still errors.
func TestCatalogCollisionPolicy(t *testing.T) {
	t.Parallel()
	surfaces := []struct {
		name  string
		add   func(c *catalog, key, owner string) (string, error)
		owner func(c *catalog, key string) (string, bool)
	}{
		{"tool", (*catalog).addTool, (*catalog).toolOwnerOf},
		{"resource", (*catalog).addResource, func(c *catalog, k string) (string, bool) { return c.resources.Get(k) }},
		{"prompt", (*catalog).addPrompt, func(c *catalog, k string) (string, bool) { return c.prompts.Get(k) }},
	}
	for _, sf := range surfaces {
		t.Run(sf.name, func(t *testing.T) {
			t.Parallel()
			c := newCatalog()
			if _, err := sf.add(c, "k", "claudecode"); err != nil {
				t.Fatalf("first add: %v", err)
			}
			// Adapter vs adapter: first wins.
			if _, err := sf.add(c, "k", "cursor"); err == nil {
				t.Fatal("expected an adapter duplicate to error")
			}
			if got, _ := sf.owner(c, "k"); got != "claudecode" {
				t.Fatalf("owner = %q, want claudecode (first adapter registration must win)", got)
			}
			// Generic reclaims from an adapter.
			displaced, err := sf.add(c, "k", genericOwner)
			if err != nil || displaced != "claudecode" {
				t.Fatalf("generic add = (%q, %v), want (claudecode, nil)", displaced, err)
			}
			if got, _ := sf.owner(c, "k"); got != genericOwner {
				t.Fatalf("owner = %q, want %q after a generic reclaim", got, genericOwner)
			}
			// Generic vs generic, and adapter vs generic: the generic owner stays.
			for _, o := range []string{genericOwner, "claudecode"} {
				if _, err := sf.add(c, "k", o); err == nil {
					t.Errorf("add(%q) over a generic key: expected an error", o)
				}
			}
		})
	}
}

func TestCatalogToolOwnerOfMissing(t *testing.T) {
	t.Parallel()
	c := newCatalog()
	if _, ok := c.toolOwnerOf("absent"); ok {
		t.Error("toolOwnerOf reported a tool that was never recorded")
	}
}

// TestCatalogDistinctOwnersViaRegistry confirms the registry-backed catalog
// tracks several distinct tools/resources independently: each owner lookup
// returns the owner recorded at mount time, the tool and resource surfaces are
// separate namespaces, and a duplicate on either surface is rejected without
// disturbing the unrelated entries or the surviving first registration.
func TestCatalogDistinctOwnersViaRegistry(t *testing.T) {
	t.Parallel()
	c := newCatalog()

	tools := map[string]string{
		"qsdev_a": "claudecode",
		"qsdev_b": "devenv",
		"qsdev_c": genericOwner,
	}
	for name, owner := range tools {
		if _, err := c.addTool(name, owner); err != nil {
			t.Fatalf("addTool(%q,%q): %v", name, owner, err)
		}
	}
	// A resource may share a name string with a tool: the surfaces are distinct
	// registries, so registering "qsdev_a" as a resource must not collide.
	if _, err := c.addResource("qsdev_a", "framework"); err != nil {
		t.Fatalf("addResource on separate surface: %v", err)
	}

	for name, want := range tools {
		got, ok := c.toolOwnerOf(name)
		if !ok {
			t.Fatalf("toolOwnerOf(%q): not recorded", name)
		}
		if got != want {
			t.Errorf("toolOwnerOf(%q) = %q, want %q", name, got, want)
		}
	}

	// Duplicate on one surface must error and leave the first owner intact.
	if _, err := c.addTool("qsdev_b", "intruder"); err == nil {
		t.Fatal("expected duplicate tool to error, got nil")
	}
	if got, _ := c.toolOwnerOf("qsdev_b"); got != "devenv" {
		t.Errorf("first registration must win: owner = %q, want devenv", got)
	}
	// Unrelated entries are unaffected by the rejected duplicate.
	if got, _ := c.toolOwnerOf("qsdev_a"); got != "claudecode" {
		t.Errorf("unrelated tool perturbed: owner = %q, want claudecode", got)
	}
}
