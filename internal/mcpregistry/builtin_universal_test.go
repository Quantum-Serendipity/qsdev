package mcpregistry

import "testing"

// TestRegistryIntegration proves that the universal MCP server (Phase 32) is a
// first-class built-in in the default registry — the single source `qsdev mcp
// list` reads from — with the infrastructure category, stdio transport, and the
// verified compliance grade the P26 builtin registration declares.
func TestRegistryIntegration(t *testing.T) {
	t.Cleanup(ResetDefaultRegistry)

	r := DefaultRegistry()

	def, ok := r.ByName(universalServerName)
	if !ok {
		t.Fatalf("server %q not registered in the default registry; names=%v",
			universalServerName, r.Names())
	}

	if def.DisplayName != "qsdev Universal MCP Server" {
		t.Errorf("DisplayName = %q, want %q", def.DisplayName, "qsdev Universal MCP Server")
	}
	if def.Category != CategoryInfrastructure {
		t.Errorf("Category = %q, want %q", def.Category, CategoryInfrastructure)
	}
	if def.Transport != TransportStdio {
		t.Errorf("Transport = %q, want %q", def.Transport, TransportStdio)
	}
	if def.Source != SourceBuiltin {
		t.Errorf("Source = %q, want %q", def.Source, SourceBuiltin)
	}
	if def.ComplianceGrade != ComplianceVerified {
		t.Errorf("ComplianceGrade = %q, want %q", def.ComplianceGrade, ComplianceVerified)
	}
	if def.Command != "qsdev" {
		t.Errorf("Command = %q, want %q", def.Command, "qsdev")
	}
	wantArgs := []string{"mcp", "serve"}
	if len(def.Args) != len(wantArgs) {
		t.Fatalf("Args = %v, want %v", def.Args, wantArgs)
	}
	for i := range wantArgs {
		if def.Args[i] != wantArgs[i] {
			t.Errorf("Args[%d] = %q, want %q", i, def.Args[i], wantArgs[i])
		}
	}

	// Confirm the server appears in the category listing `qsdev mcp list` would
	// render for the infrastructure group.
	found := false
	for _, d := range r.ByCategory(CategoryInfrastructure) {
		if d.Name == universalServerName {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("%q not returned by ByCategory(CategoryInfrastructure)", universalServerName)
	}
}
