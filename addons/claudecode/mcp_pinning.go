package claudecode

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpregistry"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// ErrUnpinnedMCPServer marks an MCP server whose command fetches a package
// from a registry at launch without naming an exact version, so every Claude
// Code session would run whatever release the registry serves that day,
// outside the package guard, the Bash deny rules and lockfile pinning.
var ErrUnpinnedMCPServer = errors.New("MCP server launches an unpinned package")

// requirePinnedLaunch refuses an entry whose command is a package launcher
// (npx, uvx, pnpm dlx, docker run, ...) that does not pin an exact version or
// image digest.
func requirePinnedLaunch(name string, e MCPServerEntry) error {
	if e.Command == "" || !mcpregistry.LaunchesUnpinnedPackage(e.Command, e.Args) {
		return nil
	}
	return fmt.Errorf("%w: %q runs %q; pin an exact release (npm name@1.2.3, PyPI name==1.2.3, image@sha256:...)",
		ErrUnpinnedMCPServer, name, strings.Join(append([]string{e.Command}, e.Args...), " "))
}

// installedMCPServers returns the MCP server install records `qsdev mcp
// install` keeps in the project's claude state. With no project root, no
// state file or an unreadable one it returns nil, and every server keeps its
// pinned launcher.
func installedMCPServers(projectRoot string) map[string]types.McpServerState {
	if projectRoot == "" {
		return nil
	}
	s, err := state.LoadStateFromFile(filepath.Join(projectRoot, filepath.FromSlash(statePath())))
	if err != nil {
		return nil
	}
	return s.McpServers
}

// installedBin returns the executable def's install method provides when the
// project state records exactly the pinned release as installed by that
// method. A record of another version (the catalog pin moved since) or
// another method does not count: the binary on PATH is then not the release
// qsdev vouches for.
func installedBin(name string, def catalog.MCPServerDef, installed map[string]types.McpServerState) (string, bool) {
	if def.Bin == "" || def.Version == "" {
		return "", false
	}
	rec, ok := installed[name]
	if !ok || rec.InstallMethod != def.InstallMethod || rec.InstalledVersion != def.Version {
		return "", false
	}
	return def.Bin, true
}

// catalogServerEntry returns the .mcp.json entry for a catalog server: the
// binary `qsdev mcp install` installed when the project records it, otherwise
// the catalog's pinned launcher.
func catalogServerEntry(name string, def catalog.MCPServerDef, installed map[string]types.McpServerState) MCPServerEntry {
	if bin, ok := installedBin(name, def, installed); ok {
		return MCPServerEntry{Command: bin, Args: def.BinArgs, Env: def.Env}
	}
	return catalogDefToEntry(def)
}

// catalogServerVariants returns every entry generation may write for a
// catalog server: the pinned launcher and, when the install method provides
// one, the installed binary.
func catalogServerVariants(def catalog.MCPServerDef) []MCPServerEntry {
	variants := []MCPServerEntry{catalogDefToEntry(def)}
	if def.Bin != "" && def.Version != "" {
		variants = append(variants, MCPServerEntry{Command: def.Bin, Args: def.BinArgs, Env: def.Env})
	}
	return variants
}
