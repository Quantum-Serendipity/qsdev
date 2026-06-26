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
		if err := c.addTool(name, owner); err != nil {
			t.Fatalf("addTool(%q,%q): %v", name, owner, err)
		}
	}
	// A resource may share a name string with a tool: the surfaces are distinct
	// registries, so registering "qsdev_a" as a resource must not collide.
	if err := c.addResource("qsdev_a", "framework"); err != nil {
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
	if err := c.addTool("qsdev_b", "intruder"); err == nil {
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
