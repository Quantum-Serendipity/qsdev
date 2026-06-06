package mcpserver_test

import (
	"context"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserver"
)

type testProvider struct {
	name  string
	tools []mcpserver.ToolDef
}

func (p *testProvider) Name() string               { return p.name }
func (p *testProvider) Description() string        { return "test provider" }
func (p *testProvider) Tools() []mcpserver.ToolDef { return p.tools }

func TestRegistry_RegisterAndGet(t *testing.T) {
	t.Parallel()

	reg := mcpserver.NewRegistry()
	prov := &testProvider{name: "test-server", tools: []mcpserver.ToolDef{
		{Name: "my_tool", Description: "does something", Handler: func(_ context.Context, _ map[string]any) (string, error) {
			return "ok", nil
		}},
	}}

	reg.Register(prov)

	got, ok := reg.Get("test-server")
	if !ok {
		t.Fatal("expected provider to be found")
	}
	if got.Name() != "test-server" {
		t.Errorf("Name() = %q, want %q", got.Name(), "test-server")
	}
	if len(got.Tools()) != 1 {
		t.Errorf("Tools() count = %d, want 1", len(got.Tools()))
	}
}

func TestRegistry_All_Sorted(t *testing.T) {
	t.Parallel()

	reg := mcpserver.NewRegistry()
	reg.Register(&testProvider{name: "zebra"})
	reg.Register(&testProvider{name: "alpha"})
	reg.Register(&testProvider{name: "middle"})

	all := reg.All()
	if len(all) != 3 {
		t.Fatalf("All() count = %d, want 3", len(all))
	}
	if all[0].Name() != "alpha" {
		t.Errorf("All()[0].Name() = %q, want %q", all[0].Name(), "alpha")
	}
	if all[1].Name() != "middle" {
		t.Errorf("All()[1].Name() = %q, want %q", all[1].Name(), "middle")
	}
	if all[2].Name() != "zebra" {
		t.Errorf("All()[2].Name() = %q, want %q", all[2].Name(), "zebra")
	}
}

func TestRegistry_GetMissing(t *testing.T) {
	t.Parallel()

	reg := mcpserver.NewRegistry()
	_, ok := reg.Get("nonexistent")
	if ok {
		t.Error("expected provider not to be found")
	}
}
