package doctor

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/mattn/go-isatty"

	"github.com/Quantum-Serendipity/qsdev/internal/mcphealth"
	"github.com/Quantum-Serendipity/qsdev/internal/pkgmanager"
	"github.com/Quantum-Serendipity/qsdev/internal/sysinfo"
	"github.com/Quantum-Serendipity/qsdev/internal/termutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// Report is the top-level output of qsdev doctor.
type Report struct {
	QsdevVersion string       `json:"qsdev_version"`
	Timestamp    string       `json:"timestamp"`
	System       SystemInfo   `json:"system"`
	Shell        ShellInfo    `json:"shell"`
	PackageMgrs  []PkgMgrInfo `json:"package_managers"`
	// ProjectRoot is the project the project-scoped checks ran against, or
	// "" outside a project, where they are skipped. It is always emitted so
	// JSON consumers can tell "skipped" from "clean".
	ProjectRoot       string              `json:"project_root"`
	ContainerRuntime  *ContainerSection   `json:"container_runtime,omitempty"`
	SandboxRuntime    *SandboxSection     `json:"sandbox_runtime,omitempty"`
	MCPServers        *MCPSection         `json:"mcp_servers,omitempty"`
	CloudProviders    *CloudSection       `json:"cloud_providers,omitempty"`
	ModuleChecks      *ModuleCheckSection `json:"module_checks,omitempty"`
	ProjectToolchains []string            `json:"project_toolchains,omitempty"` // see ecosystem.ToolchainChecker
	OrgOverlayDrift   string              `json:"org_overlay_drift,omitempty"`  // see catalog.OrgConfigDrift
	// HookProgramsWarning is set when the programs the project's Claude
	// Code hooks need could not be determined (see HookProgramsWarning).
	HookProgramsWarning string      `json:"hook_programs_warning,omitempty"`
	RequiredTools       []ToolEntry `json:"required_tools"`
	OptionalTools       []ToolEntry `json:"optional_tools"`
	Recommendations     []string    `json:"recommendations,omitempty"`
	AllRequiredPresent  bool        `json:"all_required_present"`
}

// SetContainerSection attaches a container runtime check result to the report.
func (r *Report) SetContainerSection(cs *ContainerSection) {
	r.ContainerRuntime = cs
}

// SetSandboxSection attaches a hook sandbox check result to the report.
func (r *Report) SetSandboxSection(ss *SandboxSection) {
	r.SandboxRuntime = ss
}

// MCPSection holds the static validation of the MCP servers the project's
// .mcp.json configures (see NewMCPSection). No server is started, so it
// reports configuration problems, not whether a server answers.
type MCPSection struct {
	Detected bool            `json:"detected"`
	Servers  []MCPServerInfo `json:"servers"`
	Warnings []string        `json:"warnings,omitempty"`
}

// MCPServerInfo summarises one configured MCP server. Status is "ok",
// "degraded" (only warnings, such as an unset environment variable) or
// "misconfigured" (an error, such as a command that is not on PATH).
type MCPServerInfo struct {
	Name      string     `json:"name"`
	Transport string     `json:"transport"`
	Status    string     `json:"status"`
	Issues    []MCPIssue `json:"issues,omitempty"`
}

// MCPIssue is one configuration problem found in an MCP server entry.
type MCPIssue struct {
	Severity    string `json:"severity"`
	Message     string `json:"message"`
	Remediation string `json:"remediation,omitempty"`
}

// SetMCPSection attaches MCP server health check results to the report.
func (r *Report) SetMCPSection(ms *MCPSection) {
	r.MCPServers = ms
}

// CloudSection holds the cloud credential isolation results for the doctor
// report: one entry per cloud provider the project configures.
type CloudSection struct {
	Detected  bool                `json:"detected"`
	Providers []CloudProviderInfo `json:"providers"`
	Warnings  []string            `json:"warnings,omitempty"`
}

// CloudProviderInfo summarises one cloud provider's credential isolation.
// Status is "isolated", "degraded" (only the advisory environment layer is
// missing) or "misconfigured" (a generated layer is missing).
type CloudProviderInfo struct {
	Name        string           `json:"name"`
	DisplayName string           `json:"display_name"`
	Status      string           `json:"status"`
	Detail      string           `json:"detail,omitempty"`
	Layers      []CloudLayerInfo `json:"layers,omitempty"`
}

// CloudLayerInfo is the state of one credential isolation layer.
type CloudLayerInfo struct {
	Name     string `json:"name"`
	Active   bool   `json:"active"`
	Enforced bool   `json:"enforced"`
	Detail   string `json:"detail,omitempty"`
}

// SetCloudSection attaches cloud credential isolation results to the report.
func (r *Report) SetCloudSection(cs *CloudSection) {
	r.CloudProviders = cs
}

// SetProjectToolchains attaches the project's toolchain mismatch warnings
// to the report.
func (r *Report) SetProjectToolchains(warnings []string) {
	r.ProjectToolchains = warnings
}

// SetOrgOverlayDrift attaches why the org overlay this run resolves is not
// the one the CLI reads for the project ("" when it is).
func (r *Report) SetOrgOverlayDrift(drift string) {
	r.OrgOverlayDrift = drift
}

// SetHookProgramsError records that ProjectChecks failed with err, so the
// hook programs were not checked; nil clears it.
func (r *Report) SetHookProgramsError(err error) {
	r.HookProgramsWarning = ""
	if err != nil {
		r.HookProgramsWarning = HookProgramsWarning(err)
	}
}

// SystemInfo captures OS-level details for the report.
type SystemInfo struct {
	OS          string `json:"os"`
	Distro      string `json:"distro,omitempty"`
	Version     string `json:"version,omitempty"`
	PrettyName  string `json:"pretty_name,omitempty"`
	Arch        string `json:"arch"`
	Kernel      string `json:"kernel,omitempty"`
	IsWSL       bool   `json:"is_wsl,omitempty"`
	IsWSL2      bool   `json:"is_wsl2,omitempty"`
	IsContainer bool   `json:"is_container,omitempty"`
}

// ShellInfo captures shell details for the report.
type ShellInfo struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	RCFile string `json:"rc_file,omitempty"`
}

// PkgMgrInfo identifies a package manager and whether it is the primary one.
type PkgMgrInfo struct {
	Name    string `json:"name"`
	Primary bool   `json:"primary"`
}

// ToolEntry is a summarised per-tool result in the report.
type ToolEntry struct {
	Name       string `json:"name"`
	Found      bool   `json:"found"`
	Version    string `json:"version,omitempty"`
	MinVersion string `json:"min_version,omitempty"`
	VersionOK  bool   `json:"version_ok"`
	Path       string `json:"path,omitempty"`
	// InProject reports that the binary lies inside the project and was
	// not run (see ToolStatus.InProject).
	InProject  bool   `json:"in_project,omitempty"`
	FixCommand string `json:"fix_command,omitempty"`
	// RequiredBy says what needs a required tool that is not an
	// environment prerequisite (see ToolStatus.RequiredBy).
	RequiredBy string `json:"required_by,omitempty"`
}

// problem returns the one-line remediation for a tool that is missing, below
// its floor or of a version that could not be determined, and false for a
// tool that is fine. Both the recommendations and RequiredProblems use it.
func (t ToolEntry) problem() (string, bool) {
	// The name, path and fix can come from repository content (a program a
	// hook runs), so each is made safe for the terminal.
	name, fix := termutil.Safe(t.Name), termutil.Safe(t.FixCommand)
	switch {
	case !t.Found:
		return withFix(t.withReason("Install "+name), fix), true
	case t.VersionOK || t.MinVersion == "":
		return "", false
	case t.Version == "" && t.InProject:
		return t.withReason(fmt.Sprintf("%s at %s is inside the project, so doctor does not run it; verify it is >= %s",
			name, termutil.Safe(t.Path), t.MinVersion)), true
	case t.Version == "":
		return t.withReason(fmt.Sprintf("Could not determine %s version (need >= %s)", name, t.MinVersion)), true
	default:
		return withFix(t.withReason(fmt.Sprintf("Upgrade %s to >= %s", name, t.MinVersion)), fix), true
	}
}

// withReason appends ", <RequiredBy>" to line when the tool has that reason.
func (t ToolEntry) withReason(line string) string {
	if t.RequiredBy == "" {
		return line
	}
	return line + ", " + t.RequiredBy
}

// withFix appends ": fix" to line when there is a fix command.
func withFix(line, fix string) string {
	if fix == "" {
		return line
	}
	return line + ": " + fix
}

// RequiredProblems returns one line per required tool that keeps this host
// from running the environment: "Install X: <fix>", "Upgrade X to >= Y:
// <fix>" or "Could not determine X version (need >= Y)". It is empty when
// every required tool is present and meets its floor.
func (r *Report) RequiredProblems() []string {
	var problems []string
	for _, t := range r.RequiredTools {
		if line, ok := t.problem(); ok {
			problems = append(problems, line)
		}
	}
	return problems
}

// BuildReport constructs a Report from raw OS info and check results.
func BuildReport(osInfo *sysinfo.OSInfo, checks []ToolStatus, qsdevVersion string) *Report {
	r := &Report{
		QsdevVersion: qsdevVersion,
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		System: SystemInfo{
			OS:          capitalOS(osInfo.OS),
			Distro:      osInfo.Distro,
			Version:     osInfo.Version,
			PrettyName:  osInfo.PrettyName,
			Arch:        osInfo.Arch,
			Kernel:      osInfo.Kernel,
			IsWSL:       osInfo.IsWSL,
			IsWSL2:      osInfo.IsWSL2,
			IsContainer: osInfo.IsContainer,
		},
		Shell: ShellInfo{
			Name:   osInfo.Shell,
			Path:   osInfo.ShellPath,
			RCFile: osInfo.ShellRCFile,
		},
		AllRequiredPresent: true,
	}

	// Package managers
	if osInfo.PackageManager != "" {
		r.PackageMgrs = append(r.PackageMgrs, PkgMgrInfo{
			Name:    osInfo.PackageManager,
			Primary: true,
		})
	}
	for _, alt := range osInfo.AltPkgManagers {
		r.PackageMgrs = append(r.PackageMgrs, PkgMgrInfo{
			Name:    alt,
			Primary: false,
		})
	}

	// Recommend the manager setup actually installs with (Nix when present),
	// not osInfo.PackageManager, so the advice matches `devenv setup`.
	pm := pkgmanager.DetectPackageManager(osInfo)
	family := osInfo.Family

	for _, ts := range checks {
		entry := ToolEntry{
			Name:       ts.Name,
			Found:      ts.Installed,
			Version:    ts.Version,
			MinVersion: ts.MinVersion,
			VersionOK:  ts.VersionOK,
			Path:       ts.Path,
			InProject:  ts.InProject,
			RequiredBy: ts.RequiredBy,
		}
		if ts.NeedsSetup() {
			// fixCommand never returns "": a tool with no known package
			// gets explicit guidance rather than an empty "Install X: ".
			entry.FixCommand = fixCommand(pm, family, ts)
		}

		if line, ok := entry.problem(); ok {
			r.Recommendations = append(r.Recommendations, line)
			if ts.Required {
				r.AllRequiredPresent = false
			}
		}
		if ts.Required {
			r.RequiredTools = append(r.RequiredTools, entry)
		} else {
			r.OptionalTools = append(r.OptionalTools, entry)
		}
	}

	return r
}

// fixCommand returns the remediation for a missing or outdated tool. On a Nix
// host it never suggests an imperative profile install, which the generated
// security rules and deny list forbid: required prerequisites point at
// `devenv setup`, and other tools at `devenv add-package <attr>`, which pins
// them in the project's devenv.nix. A tool with no known package for the
// manager gets an explicit note instead of an empty command. A tool required
// for another reason (ToolStatus.RequiredBy) is not a prerequisite setup
// installs, so it gets the advice of an optional tool. An installed tool
// with an UpgradeHint (one setup cannot upgrade) gets that hint.
func fixCommand(pm pkgmanager.PackageManager, family string, ts ToolStatus) string {
	if ts.Installed && ts.UpgradeHint != "" {
		return ts.UpgradeHint
	}
	app := branding.Get().AppName
	if _, isNix := pm.(*pkgmanager.Nix); isNix {
		if ts.Required && ts.RequiredBy == "" {
			return app + " devenv setup"
		}
		if pkg, ok := pkgmanager.PackageFor(pm, family, ts.Name); ok {
			return fmt.Sprintf("%s devenv add-package %s", app, pkg)
		}
	}
	if cmd := pkgmanager.InstallCommand(pm, family, ts.Name); cmd != "" {
		return cmd
	}
	return fmt.Sprintf("no %s package is known for %s; install it from its official distribution", pm.Name(), ts.Name)
}

// UseColor returns true if color output should be used for the given
// file descriptor. It respects NO_COLOR and TERM=dumb and checks isatty.
func UseColor(fd uintptr) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	return isatty.IsTerminal(fd)
}

// FormatReport writes a human-readable doctor report to w.
func FormatReport(w io.Writer, r *Report, useColor bool) {
	okSym, failSym, warnSym := "[OK]", "[FAIL]", "[WARN]"
	if useColor {
		okSym = "\033[32m✓\033[0m"
		failSym = "\033[31m✗\033[0m"
		warnSym = "\033[33m!\033[0m"
	}

	fmt.Fprintf(w, "%s doctor v%s\n", branding.Get().AppName, r.QsdevVersion)
	fmt.Fprintln(w, "============================")
	fmt.Fprintln(w)

	// System
	fmt.Fprintln(w, "System")
	osLabel := r.System.OS
	if r.System.PrettyName != "" {
		osLabel = fmt.Sprintf("%s (%s)", r.System.OS, r.System.PrettyName)
	}
	fmt.Fprintf(w, "  %-14s %s\n", "OS:", osLabel)
	fmt.Fprintf(w, "  %-14s %s\n", "Architecture:", r.System.Arch)
	if r.System.Kernel != "" {
		fmt.Fprintf(w, "  %-14s %s\n", "Kernel:", r.System.Kernel)
	}
	if r.System.IsWSL {
		wslVer := "1"
		if r.System.IsWSL2 {
			wslVer = "2"
		}
		fmt.Fprintf(w, "  %-14s WSL%s\n", "WSL:", wslVer)
	}
	if r.System.IsContainer {
		fmt.Fprintf(w, "  %-14s yes\n", "Container:")
	}
	fmt.Fprintln(w)

	// Shell
	fmt.Fprintln(w, "Shell")
	fmt.Fprintf(w, "  %-14s %s\n", "Shell:", r.Shell.Name)
	if r.Shell.RCFile != "" {
		fmt.Fprintf(w, "  %-14s %s\n", "RC File:", r.Shell.RCFile)
	}
	fmt.Fprintln(w)

	// Package Managers
	fmt.Fprintln(w, "Package Managers")
	for _, pm := range r.PackageMgrs {
		label := pm.Name
		if pm.Primary {
			label = pm.Name
			fmt.Fprintf(w, "  %-14s %s\n", "Primary:", label)
		} else {
			fmt.Fprintf(w, "  %-14s %s\n", "Alt:", label)
		}
	}
	fmt.Fprintln(w)
	if r.ProjectRoot == "" {
		fmt.Fprintf(w, "Not inside a %s project — project checks skipped.\n\n", branding.Get().AppName)
	}

	// Container Runtime
	if r.ContainerRuntime != nil && r.ContainerRuntime.Detected {
		formatContainerSection(w, r.ContainerRuntime, okSym, warnSym, failSym)
	}

	// Hook Sandbox
	if r.SandboxRuntime != nil && r.SandboxRuntime.Detected {
		formatSandboxSection(w, r.SandboxRuntime, okSym, warnSym, failSym)
	}

	// MCP Servers
	if r.MCPServers != nil && r.MCPServers.Detected {
		formatMCPSection(w, r.MCPServers, okSym, warnSym, failSym)
	}

	// Cloud Providers
	if r.CloudProviders != nil && r.CloudProviders.Detected {
		formatCloudSection(w, r.CloudProviders, okSym, warnSym, failSym)
	}

	// Ecosystem module checks
	if r.ModuleChecks != nil && r.ModuleChecks.Detected {
		formatModuleCheckSection(w, r.ModuleChecks, okSym, warnSym)
	}

	// Project Toolchains
	if len(r.ProjectToolchains) > 0 {
		fmt.Fprintln(w, "Project Toolchains")
		for _, warn := range r.ProjectToolchains {
			fmt.Fprintf(w, "  %s %s\n", warnSym, warn)
		}
		fmt.Fprintln(w)
	}

	// Org overlay
	if r.OrgOverlayDrift != "" {
		fmt.Fprintln(w, "Org Overlay")
		fmt.Fprintf(w, "  %s %s; it is ignored (approve it with '%s defaults pin' at your own terminal)\n", warnSym, r.OrgOverlayDrift, branding.Get().AppName)
		fmt.Fprintln(w)
	}

	// Claude Code hooks
	if r.HookProgramsWarning != "" {
		fmt.Fprintln(w, "Claude Code Hooks")
		fmt.Fprintf(w, "  %s %s\n", warnSym, r.HookProgramsWarning)
		fmt.Fprintln(w)
	}

	// Required Tools
	if len(r.RequiredTools) > 0 {
		fmt.Fprintln(w, "Required Tools")
		fmt.Fprintf(w, "  %-14s %-8s %-11s %s\n", "NAME", "STATUS", "VERSION", "PATH")
		for _, t := range r.RequiredTools {
			// A required tool below its floor fails --check, so it is a
			// failure here too, not a warning.
			sym := okSym
			if _, bad := t.problem(); bad {
				sym = failSym
			}
			ver := t.Version
			if ver == "" {
				ver = "-"
			}
			p := t.Path
			if p == "" {
				p = "-"
			}
			p = termutil.Safe(p)
			if t.RequiredBy != "" {
				p += "  " + t.RequiredBy
			}
			fmt.Fprintf(w, "  %-14s %-8s %-11s %s\n", termutil.Safe(t.Name), sym, termutil.Safe(ver), p)
		}
		fmt.Fprintln(w)
	}

	// Optional Tools
	if len(r.OptionalTools) > 0 {
		fmt.Fprintln(w, "Optional Tools")
		fmt.Fprintf(w, "  %-14s %-8s %-11s %s\n", "NAME", "STATUS", "VERSION", "PATH")
		for _, t := range r.OptionalTools {
			sym := okSym
			if !t.Found {
				sym = warnSym
			} else if !t.VersionOK {
				sym = warnSym
			}
			ver := t.Version
			if ver == "" {
				ver = "-"
			}
			p := t.Path
			if p == "" {
				p = "-"
			}
			fmt.Fprintf(w, "  %-14s %-8s %-11s %s\n", termutil.Safe(t.Name), sym, termutil.Safe(ver), termutil.Safe(p))
		}
		fmt.Fprintln(w)
	}

	// Recommendations
	if len(r.Recommendations) > 0 {
		fmt.Fprintln(w, "Recommendations")
		for i, rec := range r.Recommendations {
			fmt.Fprintf(w, "  %d. %s\n", i+1, rec)
		}
		fmt.Fprintln(w)
	}
}

func formatContainerSection(w io.Writer, cs *ContainerSection, okSym, warnSym, failSym string) {
	fmt.Fprintln(w, "Container Runtime")
	fmt.Fprintf(w, "  %-14s %s\n", "Runtime:", cs.Runtime)
	if cs.SocketPath != "" {
		fmt.Fprintf(w, "  %-14s %s\n", "Socket:", cs.SocketPath)
	}
	if cs.ComposeMethod != "" && cs.ComposeMethod != "none" {
		fmt.Fprintf(w, "  %-14s %s\n", "Compose:", cs.ComposeMethod)
	}
	for _, item := range cs.Items {
		sym := okSym
		switch item.Status {
		case "warn":
			sym = warnSym
		case "error":
			sym = failSym
		}
		fmt.Fprintf(w, "  %-14s %s %s\n", item.Label+":", sym, item.Summary)
	}
	if len(cs.Warnings) > 0 {
		fmt.Fprintln(w)
		for _, warn := range cs.Warnings {
			fmt.Fprintf(w, "  %s %s\n", warnSym, warn)
		}
	}
	fmt.Fprintln(w)
}

func formatSandboxSection(w io.Writer, ss *SandboxSection, okSym, warnSym, _ string) {
	fmt.Fprintln(w, "Hook Sandbox")
	fmt.Fprintf(w, "  %-14s %s (%s)\n", "Tier:", ss.Tier, ss.SecurityLevel)
	for _, item := range ss.Items {
		sym := okSym
		if item.Status == "warn" {
			sym = warnSym
		}
		fmt.Fprintf(w, "  %-14s %s %s\n", item.Label+":", sym, item.Summary)
	}
	if len(ss.Warnings) > 0 {
		fmt.Fprintln(w)
		for _, warn := range ss.Warnings {
			fmt.Fprintf(w, "  %s %s\n", warnSym, warn)
		}
	}
	fmt.Fprintln(w)
}

func formatMCPSection(w io.Writer, ms *MCPSection, okSym, warnSym, failSym string) {
	fmt.Fprintln(w, "MCP Servers (configuration only; no server was started)")
	for _, srv := range ms.Servers {
		sym := okSym
		switch srv.Status {
		case MCPStatusDegraded:
			sym = warnSym
		case MCPStatusMisconfigured:
			sym = failSym
		}
		fmt.Fprintf(w, "  %-20s %s %s (%s)\n", termutil.Safe(srv.Name), sym, srv.Status, srv.Transport)
		for _, is := range srv.Issues {
			isym := warnSym
			if is.Severity == mcphealth.SeverityError {
				isym = failSym
			}
			fmt.Fprintf(w, "    %s %s\n", isym, is.Message)
			if is.Remediation != "" {
				fmt.Fprintf(w, "      fix: %s\n", is.Remediation)
			}
		}
	}
	for _, warn := range ms.Warnings {
		fmt.Fprintf(w, "  %s %s\n", warnSym, warn)
	}
	fmt.Fprintln(w)
}

func formatCloudSection(w io.Writer, cs *CloudSection, okSym, warnSym, failSym string) {
	fmt.Fprintln(w, "Cloud Credential Isolation")
	for _, p := range cs.Providers {
		sym := okSym
		switch p.Status {
		case "degraded":
			sym = warnSym
		case "missing", "misconfigured":
			sym = failSym
		}
		fmt.Fprintf(w, "  %-20s %s %s\n", p.DisplayName, sym, p.Status)
		if p.Detail != "" {
			fmt.Fprintf(w, "    %s\n", p.Detail)
		}
		for _, l := range p.Layers {
			lsym := okSym
			if !l.Active {
				lsym = warnSym
				if l.Enforced {
					lsym = failSym
				}
			}
			fmt.Fprintf(w, "    %s %s: %s\n", lsym, l.Name, l.Detail)
		}
	}
	for _, warn := range cs.Warnings {
		fmt.Fprintf(w, "  %s %s\n", warnSym, warn)
	}
	fmt.Fprintln(w)
}

func capitalOS(os string) string {
	switch strings.ToLower(os) {
	case "linux":
		return "Linux"
	case "darwin":
		return "Darwin"
	case "windows":
		return "Windows"
	default:
		return os
	}
}
