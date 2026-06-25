package mcpserve

import "testing"

func TestCatalogAddToolDuplicate(t *testing.T) {
	t.Parallel()
	c := newCatalog()
	if err := c.addTool("qsdev_x", "claudecode"); err != nil {
		t.Fatalf("first addTool: %v", err)
	}
	// A second tool with the same name (e.g. a colliding adapter) must be
	// flagged, and the first owner must be preserved.
	if err := c.addTool("qsdev_x", genericOwner); err == nil {
		t.Fatal("expected duplicate tool name to error, got nil")
	}
	owner, ok := c.toolOwnerOf("qsdev_x")
	if !ok {
		t.Fatal("tool not recorded after addTool")
	}
	if owner != "claudecode" {
		t.Errorf("owner = %q, want claudecode (first registration must win)", owner)
	}
}

func TestCatalogAddResourceDuplicate(t *testing.T) {
	t.Parallel()
	c := newCatalog()
	if err := c.addResource("qsdev://r", "claudecode"); err != nil {
		t.Fatalf("first addResource: %v", err)
	}
	if err := c.addResource("qsdev://r", genericOwner); err == nil {
		t.Fatal("expected duplicate resource URI to error, got nil")
	}
}

func TestCatalogToolOwnerOfMissing(t *testing.T) {
	t.Parallel()
	c := newCatalog()
	if _, ok := c.toolOwnerOf("absent"); ok {
		t.Error("toolOwnerOf reported a tool that was never recorded")
	}
}
