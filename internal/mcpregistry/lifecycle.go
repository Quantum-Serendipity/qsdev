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
	// Now returns the current time, from which the release-age cutoff is
	// computed. Nil means time.Now.
	Now func() time.Time
}

// Minimum release ages for MCP server packages. They mirror the policy qsdev
// generates for project dependencies (min-release-age=3 in .npmrc, and
// --exclude-newer=7d for uv), which a global install would otherwise bypass:
// npm does not read the project .npmrc in global mode, and the package-guard
// hook only sees the agent's own commands, not qsdev's subprocesses.
const (
	npmMinReleaseAge = 3 * 24 * time.Hour
	uvMinReleaseAge  = 7 * 24 * time.Hour
)

// pinnedSpec returns the package-manager spec for def's exact Version
// (name@1.2.3 for npm, name==1.2.3 for uv). A definition without an exact
// version is refused: installing it would take whatever release the registry
// serves, the fetch-on-run risk the pinned catalog exists to remove.
func pinnedSpec(def *McpServerDefinition) (string, error) {
	switch def.InstallMethod {
	case InstallNpmGlobal:
		if exactVersionPattern.MatchString(def.Version) {
			return def.PackageName + "@" + def.Version, nil
		}
	case InstallUvTool:
		if pythonExactVersionPattern.MatchString(def.Version) {
			return def.PackageName + "==" + def.Version, nil
		}
	default:
		return "", fmt.Errorf("install method %s has no package spec", def.InstallMethod)
	}
	return "", fmt.Errorf("%s has no exact pinned version (got %q); refusing to install an unpinned package", def.PackageName, def.Version)
}

// packageCommand builds the package-manager command that installs spec. The
// same command installs and updates: spec pins an exact version, so updating
// means installing the release the catalog now pins. Every command is
// hardened the way project installs are: only releases older than the minimum
// release age are eligible, and npm lifecycle scripts never run.
func (lc *McpLifecycle) packageCommand(method McpInstallMethod, spec string) (string, []string) {
	now := time.Now
	if lc.Now != nil {
		now = lc.Now
	}
	cutoff := func(age time.Duration) string {
		return now().Add(-age).UTC().Format(time.RFC3339)
	}

	switch method {
	case InstallUvTool:
		return "uv", []string{"tool", "install", "--exclude-newer", cutoff(uvMinReleaseAge), spec}
	case InstallNpmGlobal:
		return "npm", []string{"install", "-g", "--ignore-scripts", "--before=" + cutoff(npmMinReleaseAge), spec}
	default:
		return "", nil
	}
}

// installPinned installs def's pinned release and verifies the manager's
// inventory now holds exactly that release. It returns the installed version
// and a failure description, or "" on success.
func (lc *McpLifecycle) installPinned(ctx context.Context, def *McpServerDefinition) (string, string) {
	spec, err := pinnedSpec(def)
	if err != nil {
		return "", err.Error()
	}
	name, args := lc.packageCommand(def.InstallMethod, spec)
	if out, err := lc.CmdRunner.Run(ctx, name, args...); err != nil {
		return "", fmt.Sprintf("%s %s %s failed: %v: %s", name, args[0], args[1], err, out)
	}

	// The command succeeded; resolve the real installed version and verify it
	// is the pinned release. A package missing from the inventory is not
	// installed, whatever the exit code said, and a different release is not
	// the one qsdev vouches for.
	version, found := lc.resolveInstalledVersion(ctx, def.InstallMethod, def.PackageName)
	switch {
	case !found:
		return version, notInInventory(def.InstallMethod, def.PackageName)
	case version != def.Version:
		return version, fmt.Sprintf("%s installed %s %s, not the pinned %s", def.InstallMethod, def.PackageName, version, def.Version)
	}
	return version, ""
}

// notInInventory describes a package-manager command that exited 0 without
// the package appearing in the manager's inventory afterwards.
func notInInventory(method McpInstallMethod, pkg string) string {
	return fmt.Sprintf("%s reported success but %s is not in its inventory", method, pkg)
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
	case InstallUvTool, InstallNpmGlobal:
		result.Version, result.Error = lc.installPinned(ctx, def)
	case InstallNixPackage:
		result.Error = "nix packages are declarative; add to devenv.nix instead"
		return result, nil
	case InstallManual:
		result.Error = "manual installation required; see server documentation"
		return result, nil
	}

	// Fail closed: a failed or unverified install must never be recorded as a
	// successful one. Return before touching state so a broken install cannot
	// masquerade as installed.
	if result.Error != "" {
		return result, nil
	}
	version := result.Version
	result.Installed = true

	if err := lc.updateServerState(serverName, def.InstallMethod, version); err != nil {
		return result, fmt.Errorf("saving state for %q: %w", serverName, err)
	}

	return result, nil
}

// Update moves an installed MCP server to the release the catalog pins.
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
	prev, tracked := state.McpServers[serverName]
	if !tracked {
		// Only servers qsdev installed can be updated: `npm update -g` exits
		// 0 for a package that is not installed, which would otherwise record
		// a phantom install that UpdateAll then keeps "updating".
		result.Error = fmt.Sprintf("%s is not installed; run `qsdev mcp install %s` first", serverName, serverName)
		return result, nil
	}
	result.PreviousVer = prev.InstalledVersion

	switch def.InstallMethod {
	case InstallUvTool, InstallNpmGlobal:
		result.NewVersion, result.Error = lc.installPinned(ctx, def)
	case InstallNixPackage:
		result.Error = "nix packages are declarative; update devenv.nix instead"
		return result, nil
	case InstallManual:
		result.Error = "manual update required; see server documentation"
		return result, nil
	}

	// Fail closed: a failed or unverified update must not overwrite state with
	// a new successful entry. Return before touching state.
	if result.Error != "" {
		return result, nil
	}
	version := result.NewVersion
	result.Updated = true

	if err := lc.updateServerState(serverName, def.InstallMethod, version); err != nil {
		return result, fmt.Errorf("saving state for %q: %w", serverName, err)
	}

	return result, nil
}

// UpdateAll moves every MCP server recorded in generated state to its pinned
// release.
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

	// Fail closed: a failed uninstall leaves the package installed, so keep
	// its state entry and it stays covered by UpdateAll. Drop the entry only
	// when the manager's inventory could be read and no longer lists the
	// package; an unreadable inventory proves nothing.
	if result.Error != "" {
		_, found, err := lc.queryInventory(ctx, def.InstallMethod, def.PackageName)
		if found || err != nil {
			return result, nil
		}
	}

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
// Callers record only installs the post-install probe confirmed present.
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

// versionUnknown is recorded when the installed version cannot be determined.
const versionUnknown = "unknown"

// resolveInstalledVersion queries the relevant package manager for the
// installed version of pkg. The second return value reports whether the
// package was found in the manager's inventory; an inventory that cannot be
// read counts as not found.
func (lc *McpLifecycle) resolveInstalledVersion(ctx context.Context, method McpInstallMethod, pkg string) (string, bool) {
	version, found, _ := lc.queryInventory(ctx, method, pkg)
	return version, found
}

// queryInventory is resolveInstalledVersion with the difference between "not
// in the inventory" (found false, nil error) and "the inventory could not be
// read" (non-nil error) preserved, for callers that must not mistake the
// latter for proof of absence.
func (lc *McpLifecycle) queryInventory(ctx context.Context, method McpInstallMethod, pkg string) (string, bool, error) {
	switch method {
	case InstallUvTool:
		out, err := lc.CmdRunner.Run(ctx, "uv", "tool", "list")
		if err != nil {
			return versionUnknown, false, fmt.Errorf("listing uv tools: %w", err)
		}
		version, found := parseUvToolVersion(out, pkg)
		return version, found, nil
	case InstallNpmGlobal:
		// `npm ls` exits non-zero on peer/extraneous warnings, and when pkg
		// is not installed, while still emitting valid JSON, so the exit code
		// is intentionally ignored and the output parsed directly.
		out, runErr := lc.CmdRunner.Run(ctx, "npm", "ls", "-g", "--json", pkg)
		if !json.Valid(out) {
			return versionUnknown, false, fmt.Errorf("listing global npm packages: unreadable output (%v)", runErr)
		}
		version, found := parseNpmVersion(out, pkg)
		return version, found, nil
	default:
		return versionUnknown, false, fmt.Errorf("install method %s has no inventory", method)
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
