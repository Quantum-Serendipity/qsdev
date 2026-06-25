package projectctx

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
)

const validQsdevYAML = "version: 1\nsecurity:\n  level: enhanced\n"
const goMod = "module example.com/proj\n\ngo 1.21\n"

// writeFile writes content to dir/rel, creating parent directories.
func writeFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", rel, err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

// newGoProject creates a temp dir with a go.mod and .qsdev.yaml and returns its
// path plus a ProjectContext for it.
func newGoProject(t *testing.T) (string, *ProjectContext) {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", goMod)
	writeFile(t, dir, ".qsdev.yaml", validQsdevYAML)

	pc, err := NewProjectContext(dir)
	if err != nil {
		t.Fatalf("NewProjectContext: %v", err)
	}
	return dir, pc
}

// callTool finds a tool by name and invokes its handler with the given args.
func callTool(t *testing.T, pc *ProjectContext, name string, args map[string]any) *spi.ToolResult {
	t.Helper()
	for _, reg := range pc.Tools() {
		if reg.Name == name {
			res, err := reg.Handler(context.Background(), &spi.ToolCallContext{}, &spi.ToolRequest{Name: name, Arguments: args})
			if err != nil {
				t.Fatalf("tool %s handler error: %v", name, err)
			}
			return res
		}
	}
	t.Fatalf("tool %s not registered", name)
	return nil
}

func TestNewProjectContext(t *testing.T) {
	t.Parallel()

	t.Run("succeeds in a project dir", func(t *testing.T) {
		t.Parallel()
		dir, pc := newGoProject(t)
		if pc.ProjectRoot() != dir {
			t.Errorf("ProjectRoot = %q, want %q", pc.ProjectRoot(), dir)
		}
		if !pc.Detection().HasGoMod {
			t.Errorf("expected HasGoMod true for a dir with go.mod")
		}
	})

	t.Run("rejects empty root", func(t *testing.T) {
		t.Parallel()
		if _, err := NewProjectContext(""); err == nil {
			t.Errorf("expected error for empty project root")
		}
	})

	t.Run("tolerates an uninitialized project", func(t *testing.T) {
		t.Parallel()
		// No .qsdev.yaml, no state file: construction must still succeed.
		if _, err := NewProjectContext(t.TempDir()); err != nil {
			t.Errorf("NewProjectContext on empty dir: %v", err)
		}
	})
}

func TestSurfaceCounts(t *testing.T) {
	t.Parallel()
	_, pc := newGoProject(t)
	if got := len(pc.Tools()); got != 6 {
		t.Errorf("Tools() count = %d, want 6", got)
	}
	if got := len(pc.Resources()); got != 5 {
		t.Errorf("Resources() count = %d, want 5", got)
	}
	if got := len(pc.Prompts()); got != 5 {
		t.Errorf("Prompts() count = %d, want 5", got)
	}
}

func TestProjectInfo(t *testing.T) {
	t.Parallel()
	_, pc := newGoProject(t)
	res := callTool(t, pc, toolProjectInfo, nil)
	if res.IsError {
		t.Fatalf("project_info should not be an error: %q", res.Text)
	}
	if strings.TrimSpace(res.Text) == "" {
		t.Errorf("project_info text is empty")
	}
	structured, ok := res.Structured.(map[string]any)
	if !ok {
		t.Fatalf("project_info structured is %T, want map", res.Structured)
	}
	langs, _ := structured["languages"].([]string)
	if !containsPrefix(langs, "go") {
		t.Errorf("languages %v should include go", langs)
	}
}

func TestConfigShow(t *testing.T) {
	t.Parallel()

	t.Run("returns valid YAML", func(t *testing.T) {
		t.Parallel()
		_, pc := newGoProject(t)
		res := callTool(t, pc, toolConfigShow, nil)
		if res.IsError {
			t.Fatalf("config_show unexpected error: %q", res.Text)
		}
		var decoded map[string]any
		if err := yaml.Unmarshal([]byte(res.Text), &decoded); err != nil {
			t.Fatalf("config_show output is not valid YAML: %v\n%s", err, res.Text)
		}
		if len(decoded) == 0 {
			t.Errorf("config_show YAML decoded to an empty map")
		}
	})

	t.Run("not_configured when uninitialized", func(t *testing.T) {
		t.Parallel()
		pc, err := NewProjectContext(t.TempDir())
		if err != nil {
			t.Fatalf("NewProjectContext: %v", err)
		}
		res := callTool(t, pc, toolConfigShow, nil)
		assertNotConfigured(t, res)
	})

	t.Run("include_local merges local overrides", func(t *testing.T) {
		t.Parallel()
		dir, pc := newGoProject(t)
		writeFile(t, dir, ".qsdev.local.yaml", "extra_packages:\n  - neovim\n")
		res := callTool(t, pc, toolConfigShow, map[string]any{"include_local": true})
		if res.IsError {
			t.Fatalf("config_show include_local error: %q", res.Text)
		}
		if err := yaml.Unmarshal([]byte(res.Text), &map[string]any{}); err != nil {
			t.Fatalf("config_show include_local invalid YAML: %v", err)
		}
	})
}

func TestDetectReflectsNewFile(t *testing.T) {
	t.Parallel()
	dir, pc := newGoProject(t)

	// The startup snapshot has no node ecosystem.
	before := callTool(t, pc, toolProjectInfo, nil)
	beforeLangs, _ := before.Structured.(map[string]any)["languages"].([]string)
	if containsPrefix(beforeLangs, "node") {
		t.Fatalf("did not expect node before adding package.json")
	}

	// Add a package.json, then qsdev_detect must observe it.
	writeFile(t, dir, "package.json", `{"name":"x","version":"1.0.0"}`)
	res := callTool(t, pc, toolDetect, map[string]any{"force": true})
	if res.IsError {
		t.Fatalf("detect error: %q", res.Text)
	}
	structured := res.Structured.(map[string]any)
	langs, _ := structured["languages"].([]string)
	if !containsPrefix(langs, "node") {
		t.Errorf("detect languages %v should include node after adding package.json", langs)
	}
	if structured["forced"] != true {
		t.Errorf("expected forced=true echoed back")
	}
}

func TestDoctor(t *testing.T) {
	t.Parallel()
	_, pc := newGoProject(t)
	res := callTool(t, pc, toolDoctor, map[string]any{"verbose": true})
	if res.IsError {
		t.Fatalf("doctor should not be a tool error: %q", res.Text)
	}
	structured, ok := res.Structured.(map[string]any)
	if !ok {
		t.Fatalf("doctor structured is %T, want map", res.Structured)
	}
	if _, ok := structured["pass"].(bool); !ok {
		t.Errorf("doctor structured missing bool pass field: %v", structured)
	}
	if _, ok := structured["checks"]; !ok {
		t.Errorf("doctor structured missing checks field")
	}
}

func TestToolListReflectsState(t *testing.T) {
	t.Parallel()
	dir, pc := newGoProject(t)

	all := pc.toolReg.All()
	if len(all) == 0 {
		t.Skip("tool registry empty; nothing to assert")
	}
	target := all[0].Name

	// Write a state file marking the first tool enabled, then rebuild.
	stateYAML := "files: {}\nenabled_tools:\n  " + target + ": true\n"
	writeFile(t, dir, ".devinit/.qsdev-init-state.yaml", stateYAML)

	pc2, err := NewProjectContext(dir)
	if err != nil {
		t.Fatalf("rebuild NewProjectContext: %v", err)
	}
	res := callTool(t, pc2, toolToolList, nil)
	structured := res.Structured.(map[string]any)
	tools, _ := structured["tools"].([]map[string]any)
	found := false
	for _, tm := range tools {
		if tm["name"] == target {
			found = true
			if tm["enabled"] != true {
				t.Errorf("tool %q should be enabled per state", target)
			}
		}
	}
	if !found {
		t.Errorf("target tool %q not present in tool_list", target)
	}
}

func TestMCPList(t *testing.T) {
	t.Parallel()
	_, pc := newGoProject(t)
	res := callTool(t, pc, toolMCPList, nil)
	if res.IsError {
		t.Fatalf("mcp_list error: %q", res.Text)
	}
	structured := res.Structured.(map[string]any)
	if _, ok := structured["servers"].([]map[string]any); !ok {
		t.Errorf("mcp_list structured missing servers list: %v", structured)
	}
	if structured["health_probed"] != false {
		t.Errorf("expected health_probed=false without the flag")
	}
}

// assertNotConfigured checks the canonical graceful-degradation contract: a
// tool-level error whose Text is a JSON object with status=not_configured.
func assertNotConfigured(t *testing.T, res *spi.ToolResult) {
	t.Helper()
	if !res.IsError {
		t.Fatalf("expected IsError true for not_configured, got %q", res.Text)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(res.Text), &payload); err != nil {
		t.Fatalf("not_configured text is not JSON: %v\n%s", err, res.Text)
	}
	if payload["status"] != "not_configured" {
		t.Errorf("status = %v, want not_configured", payload["status"])
	}
}

func containsPrefix(s []string, prefix string) bool {
	for _, v := range s {
		if v == prefix || strings.HasPrefix(v, prefix+" ") {
			return true
		}
	}
	return false
}
