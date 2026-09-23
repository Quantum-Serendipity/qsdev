package catalog

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

// mcpPackageTarget is the registry package an npx- or uvx-launched MCP
// server runs.
type mcpPackageTarget struct {
	registry string // "npm" or "pypi"
	name     string
	version  string // "" when unpinned
}

// mcpTarget extracts the package an MCP server definition launches. ok is
// false for servers that are not started through npx or uvx.
func mcpTarget(def MCPServerDef) (mcpPackageTarget, bool) {
	switch def.Command {
	case "npx":
		for _, a := range def.Args {
			if strings.HasPrefix(a, "-") {
				continue
			}
			name, version := a, ""
			if i := strings.LastIndex(a, "@"); i > 0 {
				name, version = a[:i], a[i+1:]
			}
			return mcpPackageTarget{registry: "npm", name: name, version: version}, true
		}
	case "uvx":
		spec := ""
		for i, a := range def.Args {
			if a == "--from" && i+1 < len(def.Args) {
				spec = def.Args[i+1]
				break
			}
			if !strings.HasPrefix(a, "-") && spec == "" {
				spec = a
			}
		}
		if spec == "" {
			return mcpPackageTarget{}, false
		}
		name, version, _ := strings.Cut(spec, "==")
		if i := strings.Index(name, "["); i > 0 {
			name = name[:i] // drop extras: semble[mcp] -> semble
		}
		return mcpPackageTarget{registry: "pypi", name: name, version: version}, true
	}
	return mcpPackageTarget{}, false
}

// Regression: filesystem and fetch launched @anthropic-ai/mcp-* packages that
// do not exist on npm, so they could never start. They must run the real
// reference servers with exact version pins.
func TestMCPServers_ReferenceServerPackages(t *testing.T) {
	t.Parallel()

	cat, err := LoadEmbeddedOnly()
	if err != nil {
		t.Fatalf("LoadEmbeddedOnly() error: %v", err)
	}
	servers := cat.MCPServers()

	tests := []struct {
		server   string
		registry string
		name     string
	}{
		{"filesystem", "npm", "@modelcontextprotocol/server-filesystem"},
		{"fetch", "pypi", "mcp-server-fetch"},
	}
	for _, tt := range tests {
		t.Run(tt.server, func(t *testing.T) {
			t.Parallel()
			def, ok := servers[tt.server]
			if !ok {
				t.Fatalf("catalog has no %q MCP server", tt.server)
			}
			got, ok := mcpTarget(def)
			if !ok {
				t.Fatalf("%q is not launched through npx/uvx: %+v", tt.server, def)
			}
			if got.registry != tt.registry || got.name != tt.name {
				t.Errorf("%q launches %s package %q, want %s package %q",
					tt.server, got.registry, got.name, tt.registry, tt.name)
			}
			if got.version == "" {
				t.Errorf("%q launches %q without an exact version pin", tt.server, got.name)
			}
		})
	}
}

func TestMCPServers_NpxUvxHavePackageTarget(t *testing.T) {
	t.Parallel()

	cat, err := LoadEmbeddedOnly()
	if err != nil {
		t.Fatalf("LoadEmbeddedOnly() error: %v", err)
	}
	for name, def := range cat.MCPServers() {
		if def.Command != "npx" && def.Command != "uvx" {
			continue
		}
		if tgt, ok := mcpTarget(def); !ok || tgt.name == "" {
			t.Errorf("MCP server %q runs %s without a package argument: %v", name, def.Command, def.Args)
		}
	}
}

// TestMCPServerPackagesResolve verifies that every npx/uvx package the
// catalog launches exists on its registry (at the pinned version, when
// pinned). Network-dependent, so it is opt-in: set VERIFY_MCP_PACKAGES=1.
func TestMCPServerPackagesResolve(t *testing.T) {
	if os.Getenv("VERIFY_MCP_PACKAGES") == "" {
		t.Skip("set VERIFY_MCP_PACKAGES=1 to verify MCP server packages against npm and PyPI")
	}

	cat, err := LoadEmbeddedOnly()
	if err != nil {
		t.Fatalf("LoadEmbeddedOnly() error: %v", err)
	}
	client := &http.Client{Timeout: 30 * time.Second}

	for name, def := range cat.MCPServers() {
		tgt, ok := mcpTarget(def)
		if !ok {
			continue
		}
		t.Run(name, func(t *testing.T) {
			u := registryURL(tgt)
			resp, err := client.Get(u)
			if err != nil {
				t.Fatalf("GET %s: %v", u, err)
			}
			_ = resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Errorf("MCP server %q launches %s package %q (version %q), which does not resolve: GET %s -> HTTP %d",
					name, tgt.registry, tgt.name, tgt.version, u, resp.StatusCode)
			}
		})
	}
}

func registryURL(tgt mcpPackageTarget) string {
	switch tgt.registry {
	case "npm":
		version := tgt.version
		if version == "" {
			version = "latest"
		}
		return fmt.Sprintf("https://registry.npmjs.org/%s/%s",
			strings.Replace(tgt.name, "/", "%2f", 1), url.PathEscape(version))
	default:
		if tgt.version == "" {
			return fmt.Sprintf("https://pypi.org/pypi/%s/json", url.PathEscape(tgt.name))
		}
		return fmt.Sprintf("https://pypi.org/pypi/%s/%s/json", url.PathEscape(tgt.name), url.PathEscape(tgt.version))
	}
}
