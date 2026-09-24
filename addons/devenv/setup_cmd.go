package devenv

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/doctor"
	"github.com/Quantum-Serendipity/qsdev/internal/installer"
	"github.com/Quantum-Serendipity/qsdev/internal/pkgmanager"
	"github.com/Quantum-Serendipity/qsdev/internal/privilege"
	"github.com/Quantum-Serendipity/qsdev/internal/sysinfo"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// AutoSetupPrerequisites installs missing core prerequisites (nix, devenv, direnv)
// non-interactively. It is called by the init/join flow when --yes is set to deliver
// on the "one command and go" promise. Returns nil only if all prerequisites are
// already present or were successfully installed; otherwise the error names every
// prerequisite that is still missing.
func AutoSetupPrerequisites(ctx context.Context, w io.Writer) error {
	if os.Getenv(branding.Get().EnvPrefix+"SKIP_SETUP") == "1" {
		return nil
	}

	osInfo := sysinfo.DetectOS()
	checks := doctor.RunAllChecks(ctx, osInfo)

	coreTools := map[string]bool{"nix": true, "devenv": true, "direnv": true}
	var missing []doctor.ToolStatus
	for _, ts := range checks {
		if coreTools[ts.Name] && needsSetup(ts) {
			missing = append(missing, ts)
		}
	}

	if len(missing) == 0 {
		return nil
	}

	// NixOS: prerequisites come from the system config, not imperative install.
	if osInfo.Distro == "nixos" {
		return fmt.Errorf("missing prerequisites on NixOS: %s; add them to your system configuration",
			strings.Join(toolNames(missing), ", "))
	}

	installable, manual := partitionInstallable(missing)
	var errs []error
	if len(installable) > 0 {
		_, _ = fmt.Fprintf(w, "Installing prerequisites: %s\n", strings.Join(installable, ", "))
		errs = append(errs, installToolsInOrder(ctx, w, installable, osInfo))
	}
	if len(manual) > 0 {
		errs = append(errs, fmt.Errorf("manual installation required for: %s", strings.Join(manual, ", ")))
	}
	if err := errors.Join(errs...); err != nil {
		return err
	}

	_, _ = fmt.Fprintln(w, "Prerequisites installed.")
	return nil
}

// installDependencies records, for each tool that has one, the tool that must
// be installed first. Tools absent from this map have no install-time
// dependency. The install plan is derived from the selected doctor checks plus
// these edges, so every auto-installable check is always part of the plan.
var installDependencies = map[string]string{
	"nix":    "curl", // the Nix installer script downloads its binary with curl
	"devenv": "nix",  // installed with `nix profile install` from pinned nixpkgs
	"npm":    "node", // bundled with node
	"claude": "npm",  // installed with an age-gated, pinned `npm install -g`
}

// installDepth returns how many dependency edges precede name in the plan.
func installDepth(name string) int {
	depth := 0
	for dep, ok := installDependencies[name]; ok && depth <= len(installDependencies); dep, ok = installDependencies[dep] {
		depth++
	}
	return depth
}

// installPlan groups the selected tools into levels so that every tool is
// installed after its dependencies. Each selected tool appears exactly once;
// order within a level follows the selection order.
func installPlan(selected []string) [][]string {
	var levels [][]string
	seen := make(map[string]bool, len(selected))
	for _, name := range selected {
		if seen[name] {
			continue
		}
		seen[name] = true
		depth := installDepth(name)
		for len(levels) <= depth {
			levels = append(levels, nil)
		}
		levels[depth] = append(levels[depth], name)
	}
	return levels
}

// needsSetup reports whether a doctor result calls for installation.
func needsSetup(ts doctor.ToolStatus) bool {
	return !ts.Installed || (ts.MinVersion != "" && !ts.VersionOK)
}

// partitionInstallable splits missing tools into those setup can install and
// those that need manual installation. A tool that is not auto-installable only
// because its dependency is absent (e.g. devenv without Nix) becomes
// installable when that dependency is itself being installed in the same run.
func partitionInstallable(missing []doctor.ToolStatus) (installable, manual []string) {
	planned := make(map[string]bool, len(missing))
	for _, ts := range missing {
		if ts.AutoInstallable {
			planned[ts.Name] = true
		}
	}
	for changed := true; changed; {
		changed = false
		for _, ts := range missing {
			if !planned[ts.Name] && planned[installDependencies[ts.Name]] {
				planned[ts.Name] = true
				changed = true
			}
		}
	}
	for _, ts := range missing {
		if planned[ts.Name] {
			installable = append(installable, ts.Name)
		} else {
			manual = append(manual, ts.Name)
		}
	}
	return installable, manual
}

// toolNames returns the names of the given tool statuses.
func toolNames(statuses []doctor.ToolStatus) []string {
	names := make([]string, len(statuses))
	for i, ts := range statuses {
		names[i] = ts.Name
	}
	return names
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

	var missing []doctor.ToolStatus
	for _, ts := range checks {
		if needsSetup(ts) {
			missing = append(missing, ts)
		}
	}

	if len(missing) == 0 {
		_, _ = fmt.Fprintln(w, "All tools are installed.")
		return nil
	}

	// NixOS special case: print declarative Nix expressions instead of installing.
	if osInfo.Distro == "nixos" {
		return printNixOSInstructions(w, toolNames(missing))
	}

	installable, notInstallable := partitionInstallable(missing)

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

	// Install tools in dependency order. Failures are returned after the
	// verification summary so the command still exits non-zero.
	installErr := installToolsInOrder(ctx, w, selected, osInfo)

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

	if installErr != nil {
		return fmt.Errorf("setup incomplete: %w", installErr)
	}
	return nil
}

// printNixOSInstructions prints declarative Nix package expressions for NixOS users.
func printNixOSInstructions(w io.Writer, missing []string) error {
	_, _ = fmt.Fprintln(w, "NixOS detected. Add the following to your configuration.nix or home-manager config:")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "  environment.systemPackages = with pkgs; [")
	for _, name := range missing {
		nixPkg := toolToNixPkg(name)
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
	// Describe the manager setup will actually install with (Nix when present),
	// not osInfo.PackageManager.
	pm := pkgmanager.DetectPackageManager(osInfo)
	family := osInfo.Family

	_, _ = fmt.Fprintln(w, "Dry run: the following tools would be installed:")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "  %-14s %s\n", "TOOL", "INSTALL COMMAND")
	_, _ = fmt.Fprintln(w, "  "+strings.Repeat("-", 60))
	for _, name := range tools {
		cmd := installCommandForTool(name, family, pm)
		_, _ = fmt.Fprintf(w, "  %-14s %s\n", name, cmd)
	}
	_, _ = fmt.Fprintln(w)

	if pm.NeedsElevation() && privilege.NeedsElevation() {
		_, _ = fmt.Fprintln(w, "Note: Some installations will require elevated privileges (sudo).")
	}

	return nil
}

// installCommandForTool returns a human-readable install command for a tool.
func installCommandForTool(name, family string, pm pkgmanager.PackageManager) string {
	switch name {
	case "nix":
		return "curl -sSf -L https://install.determinate.systems/nix | sh -s -- install"
	case "claude", "devenv":
		cmd, err := bootstrapToolCmd(setupBootstrapTools[name])
		if err != nil {
			return fmt.Sprintf("(refused: %v)", err)
		}
		return strings.Join(cmd, " ")
	default:
		cmd := pkgmanager.InstallCommand(pm, family, name)
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

// installFunc installs a single tool by name.
type installFunc func(ctx context.Context, w io.Writer, name string) error

// installToolsInOrder installs tools in dependency order using the system
// package manager or a tool-specific installer. It returns an error joining
// every tool that failed or was skipped because a dependency failed.
func installToolsInOrder(ctx context.Context, w io.Writer, selected []string, osInfo *sysinfo.OSInfo) error {
	pm := pkgmanager.DetectPackageManager(osInfo)
	install := func(ctx context.Context, w io.Writer, name string) error {
		switch name {
		case "nix":
			return installNix(ctx, w)
		case "devenv":
			return installDevenv(ctx, w)
		case "claude":
			return installClaude(ctx, w)
		default:
			return installWithPM(ctx, w, name, osInfo.Family, pm)
		}
	}
	return runInstallPlan(ctx, w, selected, install)
}

// runInstallPlan installs the selected tools level by level. A failed tool
// does not abort the run, but tools depending on it are skipped. The returned
// error joins every failure so callers never report success while a selected
// tool is still missing.
func runInstallPlan(ctx context.Context, w io.Writer, selected []string, install installFunc) error {
	failed := make(map[string]bool)
	var errs []error
	for _, level := range installPlan(selected) {
		for _, name := range level {
			if dep := installDependencies[name]; failed[dep] {
				failed[name] = true
				_, _ = fmt.Fprintf(w, "Skipping %s: dependency %s failed to install.\n", name, dep)
				errs = append(errs, fmt.Errorf("%s: skipped because %s failed to install", name, dep))
				continue
			}

			_, _ = fmt.Fprintf(w, "Installing %s...\n", name)
			if err := install(ctx, w, name); err != nil {
				failed[name] = true
				_, _ = fmt.Fprintf(w, "  Failed to install %s: %v\n", name, err)
				errs = append(errs, fmt.Errorf("installing %s: %w", name, err))
				continue
			}
			_, _ = fmt.Fprintf(w, "  %s installed successfully.\n", name)
		}
	}
	return errors.Join(errs...)
}

// installWithPM installs a tool using the detected package manager. The
// package name is resolved for pm itself (not osInfo.PackageManager), and the
// elevated path runs pm's own install command line.
func installWithPM(ctx context.Context, w io.Writer, toolName, family string, pm pkgmanager.PackageManager) error {
	if !pm.Available() {
		return fmt.Errorf("package manager %s is not installed; install it or install %s manually", pm.Name(), toolName)
	}

	pkgName, ok := pkgmanager.PackageFor(pm, family, toolName)
	if !ok {
		// A tool may have no installable package for this manager (e.g. pre-commit
		// or npm on winget). Surface actionable guidance instead of attempting a
		// broken install with the bare tool name.
		if remedy, unavailable := pkgmanager.PackageUnavailable(toolName, pm.Name()); unavailable {
			return fmt.Errorf("no %s package for %s; %s", pm.Name(), toolName, remedy)
		}
		// Fallback: try using the tool name directly.
		pkgName = toolName
	}

	if pm.NeedsElevation() && privilege.NeedsElevation() {
		_, _ = fmt.Fprintf(w, "  (requires elevated privileges)\n")
		bin, args := pm.InstallArgs(pkgName)
		return privilege.ElevatedExec(ctx, bin, args...)
	}

	return pm.Install(ctx, pkgName)
}

// nixInstallerURL is the Determinate Systems Nix installer script.
var nixInstallerURL = "https://install.determinate.systems/nix"

// nixInstallerClient downloads the Nix installer. It refuses to follow a
// redirect away from HTTPS.
var nixInstallerClient = &http.Client{
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if req.URL.Scheme != "https" {
			return fmt.Errorf("refusing non-HTTPS redirect to %s", req.URL.Redacted())
		}
		if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}
		return nil
	},
}

// maxNixInstallerSize bounds the installer script download.
const maxNixInstallerSize = 16 << 20

// nixDefaultProfileBin is where a multi-user Nix install places the nix
// binary. A Nix installed earlier in the same run is not yet on PATH.
const nixDefaultProfileBin = "/nix/var/nix/profiles/default/bin/nix"

// installNix installs Nix using the Determinate Systems installer. The script
// is downloaded to a private temp file first and then executed directly, so a
// failed or empty download is reported as an error instead of being piped
// into a shell that exits 0 on empty input.
func installNix(ctx context.Context, w io.Writer) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	script, err := downloadNixInstaller(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(script) }()

	cmd := exec.CommandContext(ctx, "sh", script, "install", "--no-confirm")
	cmd.WaitDelay = 10 * time.Second
	cmd.Stdout = w
	cmd.Stderr = w
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running Nix installer: %w", err)
	}
	return nil
}

// downloadNixInstaller fetches the installer script over HTTPS into a new
// private temp file (mode 0600) and returns its path. The caller removes it.
func downloadNixInstaller(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, nixInstallerURL, nil)
	if err != nil {
		return "", fmt.Errorf("building Nix installer request: %w", err)
	}
	if req.URL.Scheme != "https" {
		return "", fmt.Errorf("refusing to download Nix installer over %s", req.URL.Scheme)
	}

	resp, err := nixInstallerClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("downloading Nix installer: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("downloading Nix installer: unexpected HTTP status %s", resp.Status)
	}

	f, err := os.CreateTemp("", "nix-installer-*.sh")
	if err != nil {
		return "", fmt.Errorf("creating Nix installer temp file: %w", err)
	}
	path := f.Name()
	n, copyErr := io.Copy(f, io.LimitReader(resp.Body, maxNixInstallerSize+1))
	closeErr := f.Close()

	var fail error
	switch {
	case copyErr != nil:
		fail = fmt.Errorf("downloading Nix installer: %w", copyErr)
	case closeErr != nil:
		fail = fmt.Errorf("writing Nix installer: %w", closeErr)
	case n == 0:
		fail = errors.New("downloading Nix installer: empty response")
	case n > maxNixInstallerSize:
		fail = fmt.Errorf("downloading Nix installer: script exceeds %d bytes", maxNixInstallerSize)
	}
	if fail != nil {
		_ = os.Remove(path)
		return "", fail
	}
	return path, nil
}

// installDevenv installs the devenv the catalog pins
// (bootstrap_tools.devenv) with Nix, from nixpkgs pinned to a commit and
// with the flake's nixConfig ignored. It resolves the nix binary from
// PATH or the default multi-user profile so that a Nix installed earlier in
// the same run can be used without restarting the shell.
func installDevenv(ctx context.Context, w io.Writer) error {
	nixBin, err := exec.LookPath("nix")
	if err != nil {
		if _, statErr := os.Stat(nixDefaultProfileBin); statErr != nil {
			return errors.New("nix not found on PATH or in the default profile; install Nix first")
		}
		nixBin = nixDefaultProfileBin
	}

	argv, err := bootstrapToolCmd(catalog.BootstrapToolDevenv)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, nixBin, argv[1:]...)
	cmd.Stdout = w
	cmd.Stderr = w
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("installing devenv with nix: %w", err)
	}
	return nil
}

// setupBootstrapTools maps the setup tools installed from a catalog
// bootstrap_tools pin to that entry.
var setupBootstrapTools = map[string]string{
	"claude": catalog.BootstrapToolClaudeCode,
	"devenv": catalog.BootstrapToolDevenv,
}

// bootstrapToolCmd returns the command that installs the release the
// catalog's bootstrap_tools.<name> entry pins, the same command the
// matching bootstrap step runs.
func bootstrapToolCmd(name string) ([]string, error) {
	cat, err := catalog.Default()
	if err != nil {
		return nil, fmt.Errorf("loading catalog: %w", err)
	}
	return installer.BootstrapToolInstallCmd(cat, name, time.Now())
}

// installClaude installs the catalog's pinned Claude Code release via npm.
func installClaude(ctx context.Context, w io.Writer) error {
	argv, err := bootstrapToolCmd(catalog.BootstrapToolClaudeCode)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Stdout = w
	cmd.Stderr = w
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("installing Claude Code with npm: %w", err)
	}
	return nil
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
