package mcpserve

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/projectctx"
)

// TestMountProjectContext proves the public mounting path actually registers the
// generic project context surface on the underlying mcp-go server: after
// MountProjectContext the server lists the six tools, five resources, and five
// prompts.
func TestMountProjectContext(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/x\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".qsdev.yaml"), []byte("version: 1\n"), 0o644); err != nil {
		t.Fatalf("write .qsdev.yaml: %v", err)
	}

	pc, err := projectctx.NewProjectContext(dir)
	if err != nil {
		t.Fatalf("NewProjectContext: %v", err)
	}

	srv := New(WithProjectRoot(dir))
	srv.MountProjectContext(pc)

	tools := srv.MCPServer().ListTools()
	wantTools := []string{
		"qsdev_project_info", "qsdev_doctor", "qsdev_config_show",
		"qsdev_mcp_list", "qsdev_tool_list", "qsdev_detect",
	}
	for _, name := range wantTools {
		if _, ok := tools[name]; !ok {
			t.Errorf("tool %q not mounted; mounted=%v", name, keys(tools))
		}
	}

	resources := srv.MCPServer().ListResources()
	// The four concrete resources are registered as static resources.
	wantResources := []string{
		"qsdev://project/detection", "qsdev://project/config",
		"qsdev://project/state", "qsdev://project/mcp-servers",
	}
	for _, uri := range wantResources {
		if _, ok := resources[uri]; !ok {
			t.Errorf("resource %q not mounted; mounted=%v", uri, keys(resources))
		}
	}

	// The per-package URI carries a {package} variable, so it is registered as a
	// resource TEMPLATE, not a static resource: it must NOT appear in
	// ListResources (which returns only concrete resources) yet must be recorded
	// in the catalog. That a concrete read against it resolves over the protocol
	// is covered by TestChainEnforcementOverProtocol.
	const templateURI = "qsdev://project/{package}/context"
	if _, ok := resources[templateURI]; ok {
		t.Errorf("templated resource %q must be a resource template, not a static resource", templateURI)
	}
	_, recorded := srv.catalog.resources.Get(templateURI)
	if !recorded {
		t.Errorf("templated resource %q not recorded in the catalog", templateURI)
	}

	prompts := srv.MCPServer().ListPrompts()
	wantPrompts := []string{
		"onboard-project", "diagnose-health", "security-review",
		"add-dependency", "configure-ai-framework",
	}
	for _, name := range wantPrompts {
		if _, ok := prompts[name]; !ok {
			t.Errorf("prompt %q not mounted", name)
		}
	}
}

func keys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
