package mcpregistry

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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
	def, ok := DefaultRegistry().ByName(serverName)
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
		}
	case InstallNpmGlobal:
		out, err := lc.CmdRunner.Run(ctx, "npm", "install", "-g", def.PackageName)
		if err != nil {
			result.Error = fmt.Sprintf("npm install -g failed: %v: %s", err, out)
		}
	case InstallNixPackage:
		result.Error = "nix packages are declarative; add to devenv.nix instead"
		return result, nil
	case InstallManual:
		result.Error = "manual installation required; see server documentation"
		return result, nil
	}

	// Fail closed: a failed package-manager command must never be recorded as a
	// successful install. Return before touching state so a broken install
	// cannot masquerade as installed.
	if result.Error != "" {
		return result, nil
	}

	// The command succeeded; resolve the real installed version and verify the
	// package is actually present before recording success.
	version, healthStatus := lc.verifyInstalled(ctx, def.InstallMethod, def.PackageName)
	result.Installed = true
	result.Version = version

	if err := lc.updateServerState(serverName, def.InstallMethod, version, healthStatus); err != nil {
		return result, fmt.Errorf("saving state for %q: %w", serverName, err)
	}

	return result, nil
}

// Update upgrades an installed MCP server to its latest version.
func (lc *McpLifecycle) Update(ctx context.Context, serverName string) (*UpdateResult, error) {
	def, ok := DefaultRegistry().ByName(serverName)
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
		}
	case InstallNpmGlobal:
		out, err := lc.CmdRunner.Run(ctx, "npm", "update", "-g", def.PackageName)
		if err != nil {
			result.Error = fmt.Sprintf("npm update -g failed: %v: %s", err, out)
		}
	case InstallNixPackage:
		result.Error = "nix packages are declarative; update devenv.nix instead"
		return result, nil
	case InstallManual:
		result.Error = "manual update required; see server documentation"
		return result, nil
	}

	// Fail closed: a failed upgrade must not overwrite state with a new
	// successful entry. Return before touching state.
	if result.Error != "" {
		return result, nil
	}

	// Resolve the real post-upgrade version and verify presence.
	version, healthStatus := lc.verifyInstalled(ctx, def.InstallMethod, def.PackageName)
	result.Updated = true
	result.NewVersion = version

	if err := lc.updateServerState(serverName, def.InstallMethod, version, healthStatus); err != nil {
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
	def, ok := DefaultRegistry().ByName(serverName)
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
// healthStatus is derived from a post-install verification probe and must
// reflect the real outcome — it is never assumed to be "installed".
func (lc *McpLifecycle) updateServerState(serverName string, method McpInstallMethod, version, healthStatus string) error {
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
		LastHealthStatus: healthStatus,
	}

	return lc.StateSaver(state)
}

// versionUnknown is recorded when the installed version cannot be determined.
const versionUnknown = "unknown"

// verifyInstalled resolves the actually-installed version of a package and
// probes that it is present in the package manager's inventory. For `uv tool`
// and `npm -g`, a package appearing in that inventory means its entry-point
// binary is on PATH. It returns the resolved version plus an honest health
// status: "installed" when the package is confirmed present, "unverified" when
// the install command reported success but presence could not be confirmed.
func (lc *McpLifecycle) verifyInstalled(ctx context.Context, method McpInstallMethod, pkg string) (version, healthStatus string) {
	version, found := lc.resolveInstalledVersion(ctx, method, pkg)
	if found {
		return version, "installed"
	}
	return version, "unverified"
}

// resolveInstalledVersion queries the relevant package manager for the
// installed version of pkg. The second return value reports whether the
// package was found in the manager's inventory.
func (lc *McpLifecycle) resolveInstalledVersion(ctx context.Context, method McpInstallMethod, pkg string) (string, bool) {
	switch method {
	case InstallUvTool:
		out, err := lc.CmdRunner.Run(ctx, "uv", "tool", "list")
		if err != nil {
			return versionUnknown, false
		}
		return parseUvToolVersion(out, pkg)
	case InstallNpmGlobal:
		// `npm ls` exits non-zero on peer/extraneous warnings while still
		// emitting valid JSON, so the exit code is intentionally ignored and
		// the output parsed directly.
		out, _ := lc.CmdRunner.Run(ctx, "npm", "ls", "-g", "--json", pkg)
		return parseNpmVersion(out, pkg)
	default:
		return versionUnknown, false
	}
}

// parseUvToolVersion extracts the installed version of pkg from `uv tool list`
// output. Package lines have the form "<name> v<version>" followed by indented
// entry-point lines prefixed with "-". Any extras suffix (e.g. "pkg[extra]") is
// stripped before matching, since uv lists the distribution name.
func parseUvToolVersion(out []byte, pkg string) (string, bool) {
	base := strings.SplitN(pkg, "[", 2)[0]
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "-") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] != base {
			continue
		}
		version := strings.TrimPrefix(fields[1], "v")
		if version == "" {
			return versionUnknown, false
		}
		return version, true
	}
	return versionUnknown, false
}

// parseNpmVersion extracts the installed version of pkg from
// `npm ls -g --json <pkg>` output.
func parseNpmVersion(out []byte, pkg string) (string, bool) {
	var parsed struct {
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(out, &parsed); err != nil {
		return versionUnknown, false
	}
	dep, ok := parsed.Dependencies[pkg]
	if !ok || dep.Version == "" {
		return versionUnknown, false
	}
	return dep.Version, true
}
