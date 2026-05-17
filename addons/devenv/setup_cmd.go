package devenv

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/doctor"
	"github.com/Quantum-Serendipity/qsdev/internal/pkgmanager"
	"github.com/Quantum-Serendipity/qsdev/internal/privilege"
	"github.com/Quantum-Serendipity/qsdev/internal/sysinfo"
)

// AutoSetupPrerequisites installs missing core prerequisites (nix, devenv, direnv)
// non-interactively. It is called by the init/join flow when --yes is set to deliver
// on the "one command and go" promise. Returns nil if all prerequisites are already
// present or were successfully installed.
func AutoSetupPrerequisites(ctx context.Context, w io.Writer) error {
	osInfo := sysinfo.DetectOS()
	checks := doctor.RunAllChecks(ctx, osInfo)

	coreTools := map[string]bool{"nix": true, "devenv": true, "direnv": true}
	var missing []string
	for _, ts := range checks {
		if coreTools[ts.Name] && (!ts.Installed || (ts.MinVersion != "" && !ts.VersionOK)) {
			if ts.AutoInstallable {
				missing = append(missing, ts.Name)
			}
		}
	}

	if len(missing) == 0 {
		return nil
	}

	// NixOS: prerequisites come from the system config, not imperative install.
	if osInfo.Distro == "nixos" {
		return nil
	}

	_, _ = fmt.Fprintf(w, "Installing prerequisites: %s\n", strings.Join(missing, ", "))
	if err := installToolsInOrder(ctx, w, missing, osInfo); err != nil {
		return err
	}

	_, _ = fmt.Fprintln(w, "Prerequisites installed.")
	return nil
}

// installLevel groups tools by dependency order.
type installLevel struct {
	level int
	tools []string
}

// toolLevels defines installation order: lower levels install first.
// Level 0: independent tools
// Level 1: nix (custom installer)
// Level 2: devenv, direnv (need nix)
// Level 3: node (can be level 0 from PM, but listed separately for clarity)
// Level 4: npm (bundled with node)
// Level 5: claude (needs npm)
var toolLevels = []installLevel{
	{0, []string{"git", "curl", "jq", "shellcheck", "shfmt", "hadolint", "python3", "pre-commit"}},
	{1, []string{"nix"}},
	{2, []string{"devenv", "direnv"}},
	{3, []string{"node"}},
	{4, []string{"npm"}},
	{5, []string{"claude"}},
}

func setupCmd() *cobra.Command {
	var yes, dryRun bool

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Install missing development prerequisites",
		Long: `Detect missing tools and install them automatically. Uses the system
package manager when possible and falls back to custom installers for
tools like Nix and Claude Code.

Use --dry-run to preview what would be installed without making changes.
Use --yes to skip the interactive confirmation prompt.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSetup(cmd, yes, dryRun)
		},
	}

	cmd.Flags().BoolVar(&yes, "yes", false, "Auto-install all auto-installable tools without prompting")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be installed without executing")

	return cmd
}

func runSetup(cmd *cobra.Command, yes, dryRun bool) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	w := cmd.OutOrStdout()

	osInfo := sysinfo.DetectOS()
	checks := doctor.RunAllChecks(ctx, osInfo)

	// Filter to missing tools that are auto-installable.
	var missing []missingTool
	for _, ts := range checks {
		if !ts.Installed || (ts.MinVersion != "" && !ts.VersionOK) {
			missing = append(missing, missingTool{
				name:            ts.Name,
				autoInstallable: ts.AutoInstallable,
			})
		}
	}

	if len(missing) == 0 {
		_, _ = fmt.Fprintln(w, "All tools are installed.")
		return nil
	}

	// NixOS special case: print declarative Nix expressions instead of installing.
	if osInfo.Distro == "nixos" {
		return printNixOSInstructions(w, missing)
	}

	// Filter to only auto-installable tools.
	var installable []string
	var notInstallable []string
	for _, m := range missing {
		if m.autoInstallable {
			installable = append(installable, m.name)
		} else {
			notInstallable = append(notInstallable, m.name)
		}
	}

	if len(installable) == 0 {
		_, _ = fmt.Fprintln(w, "No auto-installable tools to set up.")
		if len(notInstallable) > 0 {
			_, _ = fmt.Fprintf(w, "Manual installation required for: %s\n", strings.Join(notInstallable, ", "))
		}
		return nil
	}

	// Dry-run mode.
	if dryRun {
		return printDryRun(w, installable, osInfo)
	}

	// Auto-yes mode.
	var selected []string
	var confirmed bool
	if yes {
		selected = installable
		confirmed = true
	} else {
		// Interactive selection and confirmation.
		var err error
		selected, confirmed, err = promptSetupSelection(installable)
		if err != nil {
			return err
		}
	}

	if !confirmed || len(selected) == 0 {
		_, _ = fmt.Fprintln(w, "No tools selected for installation.")
		return nil
	}

	// Install tools in dependency order.
	if err := installToolsInOrder(ctx, w, selected, osInfo); err != nil {
		return err
	}

	// Re-run checks and print verification summary.
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Verifying installation...")
	postChecks := doctor.RunAllChecks(ctx, osInfo)
	printVerificationSummary(w, selected, postChecks)

	// Offer shell integration if direnv was installed.
	for _, name := range selected {
		if name == "direnv" {
			offerDirenvHook(w, osInfo)
			break
		}
	}

	return nil
}

// printNixOSInstructions prints declarative Nix package expressions for NixOS users.
func printNixOSInstructions(w io.Writer, missing []missingTool) error {
	_, _ = fmt.Fprintln(w, "NixOS detected. Add the following to your configuration.nix or home-manager config:")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "  environment.systemPackages = with pkgs; [")
	for _, m := range missing {
		nixPkg := toolToNixPkg(m.name)
		if nixPkg != "" {
			_, _ = fmt.Fprintf(w, "    %s\n", nixPkg)
		}
	}
	_, _ = fmt.Fprintln(w, "  ];")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Then run: sudo nixos-rebuild switch")
	return nil
}

// toolToNixPkg maps a tool name to its Nix package name.
func toolToNixPkg(name string) string {
	mapping := map[string]string{
		"git":        "git",
		"go":         "go",
		"node":       "nodejs",
		"npm":        "nodejs", // bundled with nodejs
		"nix":        "",       // already on NixOS
		"devenv":     "devenv",
		"direnv":     "direnv",
		"claude":     "claude-code",
		"pre-commit": "pre-commit",
		"shellcheck": "shellcheck",
		"shfmt":      "shfmt",
		"hadolint":   "hadolint",
		"jq":         "jq",
		"curl":       "curl",
		"python3":    "python3",
	}
	if pkg, ok := mapping[name]; ok {
		return pkg
	}
	return name
}

// printDryRun shows what would be installed without executing.
func printDryRun(w io.Writer, tools []string, osInfo *sysinfo.OSInfo) error {
	mgr := osInfo.PackageManager
	family := osInfo.Family

	_, _ = fmt.Fprintln(w, "Dry run: the following tools would be installed:")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "  %-14s %s\n", "TOOL", "INSTALL COMMAND")
	_, _ = fmt.Fprintln(w, "  "+strings.Repeat("-", 60))
	for _, name := range tools {
		cmd := installCommandForTool(name, family, mgr)
		_, _ = fmt.Fprintf(w, "  %-14s %s\n", name, cmd)
	}
	_, _ = fmt.Fprintln(w)

	pm := pkgmanager.DetectPackageManager(osInfo)
	if pm.NeedsElevation() && privilege.NeedsElevation() {
		_, _ = fmt.Fprintln(w, "Note: Some installations will require elevated privileges (sudo).")
	}

	return nil
}

// installCommandForTool returns a human-readable install command for a tool.
func installCommandForTool(name, family, mgr string) string {
	switch name {
	case "nix":
		return "curl -sSf -L https://install.determinate.systems/nix | sh -s -- install"
	case "claude":
		return "npm install -g @anthropic-ai/claude-code"
	case "devenv":
		return "nix profile install nixpkgs#devenv"
	default:
		cmd := pkgmanager.InstallCommand(name, family, mgr)
		if cmd == "" {
			return fmt.Sprintf("(install %s manually)", name)
		}
		return cmd
	}
}

// promptSetupSelection shows an interactive TUI for tool selection.
func promptSetupSelection(tools []string) (selected []string, confirmed bool, err error) {
	selected = make([]string, len(tools))
	copy(selected, tools)

	opts := make([]huh.Option[string], len(tools))
	for i, name := range tools {
		opts[i] = huh.NewOption(name, name)
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select tools to install").
				Description("All missing auto-installable tools are pre-selected.").
				Options(opts...).
				Value(&selected),
			huh.NewConfirm().
				Title("Proceed with installation?").
				Affirmative("Yes, install").
				Negative("No, cancel").
				Value(&confirmed),
		),
	).WithTheme(huh.ThemeDracula()).
		WithAccessible(isAccessible())

	if err := form.Run(); err != nil {
		if err == huh.ErrUserAborted {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("setup form: %w", err)
	}

	return selected, confirmed, nil
}

// installToolsInOrder installs tools in dependency order.
func installToolsInOrder(ctx context.Context, w io.Writer, selected []string, osInfo *sysinfo.OSInfo) error {
	selectedSet := make(map[string]bool, len(selected))
	for _, name := range selected {
		selectedSet[name] = true
	}

	pm := pkgmanager.DetectPackageManager(osInfo)
	family := osInfo.Family
	mgr := osInfo.PackageManager

	for _, level := range toolLevels {
		var levelTools []string
		for _, name := range level.tools {
			if selectedSet[name] {
				levelTools = append(levelTools, name)
			}
		}
		if len(levelTools) == 0 {
			continue
		}

		for _, name := range levelTools {
			_, _ = fmt.Fprintf(w, "Installing %s...\n", name)

			var err error
			switch name {
			case "nix":
				err = installNix(ctx, w)
			case "claude":
				err = installClaude(ctx, w)
			default:
				err = installWithPM(ctx, w, name, family, mgr, pm)
			}

			if err != nil {
				_, _ = fmt.Fprintf(w, "  Failed to install %s: %v\n", name, err)
				// Continue with other tools rather than aborting entirely.
			} else {
				_, _ = fmt.Fprintf(w, "  %s installed successfully.\n", name)
			}
		}
	}

	return nil
}

// installWithPM installs a tool using the detected package manager.
func installWithPM(ctx context.Context, w io.Writer, toolName, family, mgr string, pm pkgmanager.PackageManager) error {
	pkgName, ok := pkgmanager.ResolvePackageName(toolName, family, mgr)
	if !ok {
		// Fallback: try using the tool name directly.
		pkgName = toolName
	}

	if pm.NeedsElevation() && privilege.NeedsElevation() {
		_, _ = fmt.Fprintf(w, "  (requires elevated privileges)\n")
		return privilege.ElevatedExec(ctx, pmBinary(pm), pmInstallArgs(pm, pkgName)...)
	}

	return pm.Install(ctx, pkgName)
}

// pmBinary returns the binary name for a package manager.
func pmBinary(pm pkgmanager.PackageManager) string {
	switch pm.Name() {
	case "apt":
		return "apt-get"
	case "xbps":
		return "xbps-install"
	default:
		return pm.Name()
	}
}

// pmInstallArgs returns the install subcommand arguments for a package manager.
func pmInstallArgs(pm pkgmanager.PackageManager, pkg string) []string {
	switch pm.Name() {
	case "apt":
		return []string{"install", "-y", pkg}
	case "dnf":
		return []string{"install", "-y", pkg}
	case "pacman":
		return []string{"-S", "--noconfirm", pkg}
	case "zypper":
		return []string{"install", "-y", pkg}
	case "apk":
		return []string{"add", pkg}
	case "xbps":
		return []string{"-y", pkg}
	case "emerge":
		return []string{"--ask=n", pkg}
	default:
		return []string{"install", pkg}
	}
}

// installNix installs Nix using the Determinate Systems installer.
func installNix(ctx context.Context, w io.Writer) error {
	cmd := exec.CommandContext(ctx, "sh", "-c",
		"curl --proto '=https' --tlsv1.2 -sSf -L https://install.determinate.systems/nix | sh -s -- install --no-confirm")
	cmd.Stdout = w
	cmd.Stderr = w
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// installClaude installs Claude Code via npm.
func installClaude(ctx context.Context, w io.Writer) error {
	cmd := exec.CommandContext(ctx, "npm", "install", "-g", "@anthropic-ai/claude-code")
	cmd.Stdout = w
	cmd.Stderr = w
	return cmd.Run()
}

// printVerificationSummary re-checks installed tools and prints results.
func printVerificationSummary(w io.Writer, installed []string, checks []doctor.ToolStatus) {
	installedSet := make(map[string]bool, len(installed))
	for _, name := range installed {
		installedSet[name] = true
	}

	var successes, failures []string
	for _, ts := range checks {
		if !installedSet[ts.Name] {
			continue
		}
		if ts.Installed && ts.VersionOK {
			successes = append(successes, ts.Name)
		} else {
			failures = append(failures, ts.Name)
		}
	}

	if len(successes) > 0 {
		_, _ = fmt.Fprintf(w, "Successfully installed: %s\n", strings.Join(successes, ", "))
	}
	if len(failures) > 0 {
		_, _ = fmt.Fprintf(w, "Failed or not verified: %s\n", strings.Join(failures, ", "))
	}
}

// offerDirenvHook prints instructions for adding direnv hook to the shell RC file.
func offerDirenvHook(w io.Writer, osInfo *sysinfo.OSInfo) {
	if osInfo.ShellRCFile == "" {
		return
	}

	var hookLine string
	switch osInfo.Shell {
	case "bash":
		hookLine = `eval "$(direnv hook bash)"`
	case "zsh":
		hookLine = `eval "$(direnv hook zsh)"`
	case "fish":
		hookLine = `direnv hook fish | source`
	default:
		return
	}

	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "To enable direnv, add the following to your shell configuration:")
	_, _ = fmt.Fprintf(w, "  echo '%s' >> %s\n", hookLine, osInfo.ShellRCFile)
}

// missingTool is used internally by runSetup.
type missingTool struct {
	name            string
	autoInstallable bool
}
