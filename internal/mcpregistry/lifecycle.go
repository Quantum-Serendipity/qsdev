package mcpregistry

import (
	"context"
	"fmt"
	"time"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// CommandRunner abstracts external command execution for testability.
type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

// McpLifecycle manages installation, update, and removal of MCP servers.
type McpLifecycle struct {
	CmdRunner   CommandRunner
	StateLoader func() (*types.GeneratedState, error)
	StateSaver  func(*types.GeneratedState) error
}

// InstallResult reports the outcome of installing an MCP server.
type InstallResult struct {
	ServerName string
	Method     McpInstallMethod
	Version    string
	Installed  bool
	Error      string
}

// UpdateResult reports the outcome of updating an MCP server.
type UpdateResult struct {
	ServerName  string
	PreviousVer string
	NewVersion  string
	Updated     bool
	Error       string
}

// RemoveResult reports the outcome of removing an MCP server.
type RemoveResult struct {
	ServerName string
	Removed    bool
	Error      string
}

// Install provisions an MCP server binary using the method specified in
// the registry definition and records the result in generated state.
func (lc *McpLifecycle) Install(ctx context.Context, serverName string) (*InstallResult, error) {
	def, ok := DefaultRegistry().Get(serverName)
	if !ok {
		return nil, fmt.Errorf("unknown server %q", serverName)
	}

	if def.InstallMethod == InstallManual && def.PackageName == "" {
		return nil, fmt.Errorf("unknown server %q", serverName)
	}

	result := &InstallResult{
		ServerName: serverName,
		Method:     def.InstallMethod,
	}

	switch def.InstallMethod {
	case InstallUvTool:
		out, err := lc.CmdRunner.Run(ctx, "uv", "tool", "install", def.PackageName)
		if err != nil {
			result.Error = fmt.Sprintf("uv tool install failed: %v: %s", err, out)
		} else {
			result.Installed = true
			result.Version = "latest"
		}
	case InstallNpmGlobal:
		out, err := lc.CmdRunner.Run(ctx, "npm", "install", "-g", def.PackageName)
		if err != nil {
			result.Error = fmt.Sprintf("npm install -g failed: %v: %s", err, out)
		} else {
			result.Installed = true
			result.Version = "latest"
		}
	case InstallNixPackage:
		result.Error = "nix packages are declarative; add to devenv.nix instead"
		return result, nil
	case InstallManual:
		result.Error = "manual installation required; see server documentation"
		return result, nil
	}

	if err := lc.updateServerState(serverName, def.InstallMethod, "latest"); err != nil {
		return result, fmt.Errorf("saving state for %q: %w", serverName, err)
	}

	return result, nil
}

// Update upgrades an installed MCP server to its latest version.
func (lc *McpLifecycle) Update(ctx context.Context, serverName string) (*UpdateResult, error) {
	def, ok := DefaultRegistry().Get(serverName)
	if !ok {
		return nil, fmt.Errorf("unknown server %q", serverName)
	}

	if def.InstallMethod == InstallManual && def.PackageName == "" {
		return nil, fmt.Errorf("unknown server %q", serverName)
	}

	result := &UpdateResult{
		ServerName: serverName,
	}

	// Read previous version from state.
	state, err := lc.StateLoader()
	if err != nil {
		return nil, fmt.Errorf("loading state: %w", err)
	}
	if state.McpServers != nil {
		if prev, exists := state.McpServers[serverName]; exists {
			result.PreviousVer = prev.InstalledVersion
		}
	}

	switch def.InstallMethod {
	case InstallUvTool:
		out, err := lc.CmdRunner.Run(ctx, "uv", "tool", "upgrade", def.PackageName)
		if err != nil {
			result.Error = fmt.Sprintf("uv tool upgrade failed: %v: %s", err, out)
		} else {
			result.Updated = true
			result.NewVersion = "latest"
		}
	case InstallNpmGlobal:
		out, err := lc.CmdRunner.Run(ctx, "npm", "update", "-g", def.PackageName)
		if err != nil {
			result.Error = fmt.Sprintf("npm update -g failed: %v: %s", err, out)
		} else {
			result.Updated = true
			result.NewVersion = "latest"
		}
	case InstallNixPackage:
		result.Error = "nix packages are declarative; update devenv.nix instead"
		return result, nil
	case InstallManual:
		result.Error = "manual update required; see server documentation"
		return result, nil
	}

	if err := lc.updateServerState(serverName, def.InstallMethod, "latest"); err != nil {
		return result, fmt.Errorf("saving state for %q: %w", serverName, err)
	}

	return result, nil
}

// UpdateAll upgrades all MCP servers recorded in generated state.
func (lc *McpLifecycle) UpdateAll(ctx context.Context) ([]*UpdateResult, error) {
	state, err := lc.StateLoader()
	if err != nil {
		return nil, fmt.Errorf("loading state: %w", err)
	}

	var results []*UpdateResult
	for name := range state.McpServers {
		r, err := lc.Update(ctx, name)
		if err != nil {
			results = append(results, &UpdateResult{
				ServerName: name,
				Error:      err.Error(),
			})
			continue
		}
		results = append(results, r)
	}
	return results, nil
}

// Remove uninstalls an MCP server and removes it from generated state.
func (lc *McpLifecycle) Remove(ctx context.Context, serverName string) (*RemoveResult, error) {
	def, ok := DefaultRegistry().Get(serverName)
	if !ok {
		return nil, fmt.Errorf("unknown server %q", serverName)
	}

	if def.InstallMethod == InstallManual && def.PackageName == "" {
		return nil, fmt.Errorf("unknown server %q", serverName)
	}

	result := &RemoveResult{
		ServerName: serverName,
	}

	switch def.InstallMethod {
	case InstallUvTool:
		out, err := lc.CmdRunner.Run(ctx, "uv", "tool", "uninstall", def.PackageName)
		if err != nil {
			result.Error = fmt.Sprintf("uv tool uninstall failed: %v: %s", err, out)
		} else {
			result.Removed = true
		}
	case InstallNpmGlobal:
		out, err := lc.CmdRunner.Run(ctx, "npm", "uninstall", "-g", def.PackageName)
		if err != nil {
			result.Error = fmt.Sprintf("npm uninstall -g failed: %v: %s", err, out)
		} else {
			result.Removed = true
		}
	case InstallNixPackage:
		result.Error = "nix packages are declarative; remove from devenv.nix instead"
		return result, nil
	case InstallManual:
		result.Error = "manual removal required; see server documentation"
		return result, nil
	}

	// Remove from state regardless of command success.
	state, err := lc.StateLoader()
	if err != nil {
		return result, fmt.Errorf("loading state: %w", err)
	}
	if state.McpServers != nil {
		delete(state.McpServers, serverName)
	}
	if err := lc.StateSaver(state); err != nil {
		return result, fmt.Errorf("saving state after removal of %q: %w", serverName, err)
	}

	return result, nil
}

// updateServerState records an MCP server's install state in generated state.
func (lc *McpLifecycle) updateServerState(serverName string, method McpInstallMethod, version string) error {
	state, err := lc.StateLoader()
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}

	if state.McpServers == nil {
		state.McpServers = make(map[string]types.McpServerState)
	}

	now := time.Now()
	state.McpServers[serverName] = types.McpServerState{
		InstalledVersion: version,
		InstallMethod:    method.String(),
		LastHealthCheck:  &now,
		LastHealthStatus: "installed",
	}

	return lc.StateSaver(state)
}
