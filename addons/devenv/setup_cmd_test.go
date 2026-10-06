package devenv

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/doctor"
	"github.com/Quantum-Serendipity/qsdev/internal/pkgmanager"
	"github.com/Quantum-Serendipity/qsdev/internal/sysinfo"
)

func TestSetupCmd_Flags(t *testing.T) {
	cmd := setupCmd()

	if cmd.Use != "setup" {
		t.Errorf("Use = %q, want %q", cmd.Use, "setup")
	}

	yesFlag := cmd.Flags().Lookup("yes")
	if yesFlag == nil {
		t.Error("expected --yes flag to be registered")
	}
	dryRunFlag := cmd.Flags().Lookup("dry-run")
	if dryRunFlag == nil {
		t.Error("expected --dry-run flag to be registered")
	}
}

func TestSetupCmd_DryRun(t *testing.T) {
	cmd := setupCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--dry-run"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("setup --dry-run failed: %v", err)
	}

	output := buf.String()

	// Dry-run should produce output (either tool list or "all installed" message).
	if output == "" {
		t.Error("dry-run produced no output")
	}

	// On NixOS, the command prints declarative instructions instead of dry-run.
	// On other systems, it shows "Dry run" or "All tools are installed".
	validOutputs := []string{
		"Dry run",
		"All tools are installed",
		"NixOS detected",
		"No auto-installable tools",
	}
	found := false
	for _, valid := range validOutputs {
		if strings.Contains(output, valid) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("unexpected dry-run output: %s", output)
	}
}

func TestSetupCmd_NothingToInstall(t *testing.T) {
	// On a well-configured dev machine (which this test environment should be),
	// if all doctor checks pass, setup should say everything is installed.
	// This test is conditional: it only validates behavior, not environment state.
	cmd := setupCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--dry-run"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("setup --dry-run failed: %v", err)
	}

	output := buf.String()
	// The output should be meaningful regardless of environment state.
	if output == "" {
		t.Error("expected non-empty output from setup --dry-run")
	}
}

func TestInstallPlan_CoversEveryInstallableCheck(t *testing.T) {
	t.Parallel()
	var names []string
	for _, c := range doctor.DefaultChecks() {
		if c.AutoInstall != nil {
			names = append(names, c.Name)
		}
	}

	levelOf := make(map[string]int)
	for i, level := range installPlan(names) {
		for _, name := range level {
			if _, dup := levelOf[name]; dup {
				t.Errorf("tool %q planned more than once", name)
			}
			levelOf[name] = i
		}
	}

	for _, name := range names {
		lvl, ok := levelOf[name]
		if !ok {
			t.Errorf("auto-installable tool %q missing from install plan", name)
			continue
		}
		if dep, hasDep := installDependencies[name]; hasDep {
			if depLvl, planned := levelOf[dep]; planned && depLvl >= lvl {
				t.Errorf("%q (level %d) must install after its dependency %q (level %d)", name, lvl, dep, depLvl)
			}
		}
	}
}

func TestRunInstallPlan(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		selected      []string
		failing       map[string]bool
		wantInstalled []string
		wantErrParts  []string
	}{
		{
			name:          "tools outside the core set are attempted",
			selected:      []string{"go", "syft", "grype", "aws-cli", "gcloud", "az"},
			wantInstalled: []string{"go", "syft", "grype", "aws-cli", "gcloud", "az"},
		},
		{
			name:          "dependencies install first",
			selected:      []string{"claude", "devenv", "npm", "nix", "node", "curl"},
			wantInstalled: []string{"node", "curl", "npm", "nix", "claude", "devenv"},
		},
		{
			name:          "failure is returned and dependents are skipped",
			selected:      []string{"git", "nix", "devenv", "direnv"},
			failing:       map[string]bool{"nix": true},
			wantInstalled: []string{"git", "direnv", "nix"},
			wantErrParts:  []string{"installing nix", "devenv: skipped because nix failed"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var attempted []string
			install := func(_ context.Context, _ io.Writer, name string) error {
				attempted = append(attempted, name)
				if tt.failing[name] {
					return errors.New("boom")
				}
				return nil
			}

			var buf bytes.Buffer
			err := runInstallPlan(context.Background(), &buf, tt.selected, install)

			if !slices.Equal(attempted, tt.wantInstalled) {
				t.Errorf("attempted = %v, want %v", attempted, tt.wantInstalled)
			}
			if len(tt.wantErrParts) == 0 {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected an error when a tool fails to install")
			}
			for _, part := range tt.wantErrParts {
				if !strings.Contains(err.Error(), part) {
					t.Errorf("error %q missing %q", err, part)
				}
			}
		})
	}
}

func TestPartitionInstallable(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		missing         []doctor.ToolStatus
		wantInstallable []string
		wantManual      []string
	}{
		{
			name: "devenv installable when nix is installed in the same run",
			missing: []doctor.ToolStatus{
				{Name: "nix", AutoInstallable: true},
				{Name: "devenv", AutoInstallable: false},
				{Name: "direnv", AutoInstallable: true},
			},
			wantInstallable: []string{"nix", "devenv", "direnv"},
		},
		{
			name: "devenv manual when nix cannot be installed",
			missing: []doctor.ToolStatus{
				{Name: "nix", AutoInstallable: false},
				{Name: "devenv", AutoInstallable: false},
			},
			wantManual: []string{"nix", "devenv"},
		},
		{
			name: "transitive promotion",
			missing: []doctor.ToolStatus{
				{Name: "node", AutoInstallable: true},
				{Name: "npm", AutoInstallable: true},
				{Name: "claude", AutoInstallable: false},
			},
			wantInstallable: []string{"node", "npm", "claude"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			installable, manual := partitionInstallable(tt.missing)
			if !slices.Equal(installable, tt.wantInstallable) {
				t.Errorf("installable = %v, want %v", installable, tt.wantInstallable)
			}
			if !slices.Equal(manual, tt.wantManual) {
				t.Errorf("manual = %v, want %v", manual, tt.wantManual)
			}
		})
	}
}

// withNixInstallerServer points the Nix installer download at a TLS test
// server that responds with status and body.
func withNixInstallerServer(t *testing.T, status int, body string) {
	t.Helper()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)

	origURL, origClient := nixInstallerURL, nixInstallerClient
	nixInstallerURL = srv.URL + "/nix"
	nixInstallerClient = srv.Client()
	t.Cleanup(func() {
		nixInstallerURL, nixInstallerClient = origURL, origClient
	})
}

// Not parallel: these tests swap the package-level installer URL and client.
func TestInstallNix_ReportsInstallerFailures(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the Nix installer runs under sh")
	}
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr string
	}{
		{"http error", http.StatusNotFound, "not found", "unexpected HTTP status"},
		{"empty script", http.StatusOK, "", "empty response"},
		{"installer exits non-zero", http.StatusOK, "exit 3\n", "running Nix installer"},
		{"installer succeeds", http.StatusOK, "test \"$1\" = install || exit 9\n", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withNixInstallerServer(t, tt.status, tt.body)
			var buf bytes.Buffer
			err := installNix(context.Background(), &buf)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("installNix() = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("installNix() = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestDownloadNixInstaller_RefusesPlainHTTP(t *testing.T) {
	orig := nixInstallerURL
	nixInstallerURL = "http://install.example/nix"
	t.Cleanup(func() { nixInstallerURL = orig })

	if _, err := downloadNixInstaller(context.Background()); err == nil {
		t.Fatal("expected a plain-HTTP installer URL to be refused")
	}
}

func TestSetupCmd_ToolToNixPkg(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"git", "git"},
		{"go", "go"},
		{"node", "nodejs"},
		{"npm", "nodejs"},
		{"nix", ""},
		{"devenv", "devenv"},
		{"direnv", "direnv"},
		{"shellcheck", "shellcheck"},
		{"jq", "jq"},
		{"curl", "curl"},
		{"python3", "python3"},
	}

	for _, tt := range tests {
		got := toolToNixPkg(tt.name)
		if got != tt.want {
			t.Errorf("toolToNixPkg(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestSetupCmd_InstallCommandForTool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		family string
		pm     pkgmanager.PackageManager
		want   string
	}{
		{"nix", "debian", pkgmanager.NewApt(nil), "curl -sSf -L https://install.determinate.systems/nix | sh -s -- install"},
		{"git", "debian", pkgmanager.NewApt(nil), "sudo apt-get install -y git"},
		{"git", "macos", pkgmanager.NewBrew(nil), "brew install git"},
		// F466: on a debian host with Nix, setup installs through Nix, so the
		// dry run must show the Nix command and the nixpkgs attribute.
		{"go", "debian", pkgmanager.NewNix(nil, false), "nix profile install nixpkgs#go"},
	}

	for _, tt := range tests {
		t.Run(tt.name+"/"+tt.pm.Name(), func(t *testing.T) {
			t.Parallel()
			got := installCommandForTool(tt.name, tt.family, tt.pm)
			if got != tt.want {
				t.Errorf("installCommandForTool(%q, %q, %s) = %q, want %q", tt.name, tt.family, tt.pm.Name(), got, tt.want)
			}
		})
	}
}

// TestSetupCmd_InstallCommandForClaude covers F110: `devenv setup` installs
// the Claude Code release the catalog pins, age-gated, never the registry's
// latest release.
func TestSetupCmd_InstallCommandForClaude(t *testing.T) {
	t.Parallel()

	cat, err := catalog.Default()
	if err != nil {
		t.Fatalf("catalog.Default: %v", err)
	}
	def, ok := cat.BootstrapTool(catalog.BootstrapToolClaudeCode)
	if !ok {
		t.Fatal("catalog has no bootstrap_tools.claude-code pin")
	}

	got := strings.Fields(installCommandForTool("claude", "debian", pkgmanager.NewApt(nil)))
	if len(got) < 5 || !slices.Equal(got[:3], []string{"npm", "install", "-g"}) {
		t.Fatalf("install command = %q, want an npm install -g command", got)
	}
	if want := def.PackageName + "@" + def.Version; got[len(got)-1] != want {
		t.Errorf("installs %q, want the pinned %q", got[len(got)-1], want)
	}
	if !strings.HasPrefix(got[len(got)-2], "--before=") {
		t.Errorf("install command %q has no --before age gate", got)
	}
	if slices.Contains(got, "--ignore-scripts") == def.AllowInstallScripts {
		t.Errorf("--ignore-scripts present = %v with allow_install_scripts = %v", !def.AllowInstallScripts, def.AllowInstallScripts)
	}
}

// TestSetupCmd_InstallCommandForDevenv covers F289: `devenv setup` installs
// devenv from nixpkgs pinned to the commit the catalog names, never through
// the mutable nixpkgs registry entry, and does not accept the flake's
// nixConfig.
func TestSetupCmd_InstallCommandForDevenv(t *testing.T) {
	t.Parallel()

	cat, err := catalog.Default()
	if err != nil {
		t.Fatalf("catalog.Default: %v", err)
	}
	def, ok := cat.BootstrapTool(catalog.BootstrapToolDevenv)
	if !ok {
		t.Fatal("catalog has no bootstrap_tools.devenv pin")
	}

	got := strings.Fields(installCommandForTool("devenv", "debian", pkgmanager.NewApt(nil)))
	want := []string{"nix", "profile", "install", "--option", "accept-flake-config", "false", def.Flake + "#" + def.PackageName}
	if !slices.Equal(got, want) {
		t.Errorf("install command = %q, want %q", got, want)
	}
	if slices.Contains(got, "--accept-flake-config") {
		t.Errorf("install command %q accepts the flake's nixConfig", got)
	}
}

// recordingPM is a PackageManager that records Install calls.
type recordingPM struct {
	name      string
	available bool
	installed []string
}

func (p *recordingPM) Name() string         { return p.name }
func (p *recordingPM) Available() bool      { return p.available }
func (p *recordingPM) NeedsElevation() bool { return false }
func (p *recordingPM) InstallArgs(packages ...string) (string, []string) {
	return p.name, append([]string{"install"}, packages...)
}

func (p *recordingPM) Install(_ context.Context, packages ...string) error {
	p.installed = append(p.installed, packages...)
	return nil
}

// TestInstallWithPM resolves package names for the manager that actually runs
// the install (F466) and refuses to run a manager that is not installed (F477).
func TestInstallWithPM(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		tool          string
		family        string
		pm            *recordingPM
		wantInstalled []string
		wantErr       string
	}{
		{
			// dnf's ByManager name is "ShellCheck", which is not a nixpkgs attribute.
			name: "rhel host with nix uses nix names", tool: "shellcheck", family: "rhel",
			pm:            &recordingPM{name: "nix", available: true},
			wantInstalled: []string{"shellcheck"},
		},
		{
			// arch's ByFamily name "nodejs-lts-iron" is not a nixpkgs attribute.
			name: "arch host with nix uses nix names", tool: "node", family: "arch",
			pm:            &recordingPM{name: "nix", available: true},
			wantInstalled: []string{"nodejs"},
		},
		{
			name: "native manager keeps family names", tool: "shellcheck", family: "rhel",
			pm:            &recordingPM{name: "dnf", available: true},
			wantInstalled: []string{"ShellCheck"},
		},
		{
			name: "missing manager is reported", tool: "git", family: "macos",
			pm:      &recordingPM{name: "brew"},
			wantErr: "package manager brew is not installed",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			err := installWithPM(t.Context(), &out, tt.tool, tt.family, tt.pm)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want %q", err, tt.wantErr)
				}
				if len(tt.pm.installed) != 0 {
					t.Errorf("Install called with %v on an unavailable manager", tt.pm.installed)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if strings.Join(tt.pm.installed, ",") != strings.Join(tt.wantInstalled, ",") {
				t.Errorf("installed %v, want %v", tt.pm.installed, tt.wantInstalled)
			}
		})
	}
}

// TestVerifyInstalled covers U13-09: a selected tool counts as set up only
// when the re-run doctor check finds it at a version meeting its floor.
func TestVerifyInstalled(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		post      []doctor.ToolStatus
		wantParts []string // empty: verification passes
	}{
		{
			name: "ok",
			post: []doctor.ToolStatus{
				{Name: "devenv", Installed: true, Version: "2.1.2", MinVersion: "2.1", VersionOK: true},
				{Name: "direnv", Installed: true, VersionOK: true},
			},
		},
		{
			name: "not on PATH",
			post: []doctor.ToolStatus{
				{Name: "devenv", MinVersion: "2.1"},
				{Name: "direnv", Installed: true, VersionOK: true},
			},
			wantParts: []string{"devenv", "open a new shell or add ~/.nix-profile/bin to PATH"},
		},
		{
			name: "below minimum",
			post: []doctor.ToolStatus{
				{Name: "devenv", Installed: true, Version: "1.4.1", MinVersion: "2.1"},
				{Name: "direnv", Installed: true, VersionOK: true},
			},
			wantParts: []string{"devenv", "still below minimum 2.1 (found 1.4.1)"},
		},
		{
			name: "unknown version",
			post: []doctor.ToolStatus{
				{Name: "devenv", Installed: true, MinVersion: "2.1"},
				{Name: "direnv", Installed: true, VersionOK: true},
			},
			wantParts: []string{"devenv", "could not determine its version (need >= 2.1)"},
		},
		{
			name:      "missing from the re-check",
			post:      []doctor.ToolStatus{{Name: "direnv", Installed: true, VersionOK: true}},
			wantParts: []string{"devenv", "not verified"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := verifyInstalled([]string{"devenv", "direnv"}, tt.post)
			if len(tt.wantParts) == 0 {
				if err != nil {
					t.Fatalf("verifyInstalled: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("verifyInstalled passed, want an error")
			}
			for _, part := range tt.wantParts {
				if !strings.Contains(err.Error(), part) {
					t.Errorf("error %q missing %q", err, part)
				}
			}
			if strings.Contains(err.Error(), "direnv") {
				t.Errorf("error %q names direnv, which verified", err)
			}
		})
	}
}

// fakeSetupDeps returns setup dependencies whose first check reports
// devenv missing, whose install records the tools it was asked for and
// succeeds, and whose re-check reports post.
func fakeSetupDeps(installed *[]string, post []doctor.ToolStatus) setupDeps {
	calls := 0
	return setupDeps{
		osInfo: &sysinfo.OSInfo{OS: "linux", Distro: "ubuntu", Family: "debian", HasNix: true},
		install: func(_ context.Context, _ io.Writer, name string) error {
			*installed = append(*installed, name)
			return nil
		},
		checks: func(context.Context) []doctor.ToolStatus {
			calls++
			if calls == 1 {
				return []doctor.ToolStatus{
					{Name: "nix", Installed: true, Version: "2.28.0", MinVersion: "2.4", VersionOK: true},
					{Name: "devenv", MinVersion: "2.1", AutoInstallable: true},
					{Name: "direnv", Installed: true, VersionOK: true},
				}
			}
			return post
		},
	}
}

var (
	postDevenvMissing = []doctor.ToolStatus{
		{Name: "nix", Installed: true, Version: "2.28.0", MinVersion: "2.4", VersionOK: true},
		{Name: "devenv", MinVersion: "2.1"},
	}
	postDevenvOutdated = []doctor.ToolStatus{
		{Name: "nix", Installed: true, Version: "2.28.0", MinVersion: "2.4", VersionOK: true},
		{Name: "devenv", Installed: true, Version: "1.4.1", MinVersion: "2.1"},
	}
	postDevenvOK = []doctor.ToolStatus{
		{Name: "nix", Installed: true, Version: "2.28.0", MinVersion: "2.4", VersionOK: true},
		{Name: "devenv", Installed: true, Version: "2.1.2", MinVersion: "2.1", VersionOK: true},
	}
)

// TestRunSetup_VerificationFailureReturnsError covers U13-09: setup fails
// with ErrSetupIncomplete when an install command "succeeds" but the tool
// is still missing or below its floor.
func TestRunSetup_VerificationFailureReturnsError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		post     []doctor.ToolStatus
		wantPart string // "": setup succeeds
	}{
		{"not on PATH", postDevenvMissing, "open a new shell or add ~/.nix-profile/bin to PATH"},
		{"outdated after install", postDevenvOutdated, "still below minimum 2.1 (found 1.4.1)"},
		{"verified", postDevenvOK, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var installed []string
			var buf bytes.Buffer
			err := runSetup(context.Background(), &buf, true, false, fakeSetupDeps(&installed, tt.post))

			if !slices.Equal(installed, []string{"devenv"}) {
				t.Errorf("installed = %v, want [devenv]", installed)
			}
			if tt.wantPart == "" {
				if err != nil {
					t.Fatalf("runSetup: %v\n%s", err, buf.String())
				}
				return
			}
			if !errors.Is(err, ErrSetupIncomplete) {
				t.Fatalf("error = %v, want %v", err, ErrSetupIncomplete)
			}
			if !strings.Contains(buf.String(), tt.wantPart) {
				t.Errorf("output missing %q:\n%s", tt.wantPart, buf.String())
			}
		})
	}
}

// TestAutoSetupPrerequisites_VerifiesAfterInstall covers U13-09: the
// init/join auto-setup re-checks after installing and prints
// "Prerequisites installed." only when verification passes.
func TestAutoSetupPrerequisites_VerifiesAfterInstall(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		post     []doctor.ToolStatus
		wantPart string // "": auto-setup succeeds
	}{
		{"not on PATH", postDevenvMissing, "open a new shell or add ~/.nix-profile/bin to PATH"},
		{"outdated after install", postDevenvOutdated, "still below minimum 2.1 (found 1.4.1)"},
		{"verified", postDevenvOK, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var installed []string
			var buf bytes.Buffer
			err := autoSetup(context.Background(), &buf, fakeSetupDeps(&installed, tt.post))

			if !slices.Equal(installed, []string{"devenv"}) {
				t.Errorf("installed = %v, want [devenv]", installed)
			}
			printed := strings.Contains(buf.String(), "Prerequisites installed.")
			if tt.wantPart == "" {
				if err != nil {
					t.Fatalf("autoSetup: %v\n%s", err, buf.String())
				}
				if !printed {
					t.Errorf("output lacks 'Prerequisites installed.':\n%s", buf.String())
				}
				return
			}
			if !errors.Is(err, ErrSetupIncomplete) {
				t.Fatalf("error = %v, want %v", err, ErrSetupIncomplete)
			}
			if printed {
				t.Errorf("printed 'Prerequisites installed.' although verification failed:\n%s", buf.String())
			}
			if !strings.Contains(err.Error(), tt.wantPart) {
				t.Errorf("error %q missing %q", err, tt.wantPart)
			}
		})
	}
}

// TestAutoSetup_OutdatedNixNotReinstalled: the Nix installer installs, it
// does not upgrade, so an installed nix below its floor is left to a manual
// upgrade: setup never calls its installer and says how to upgrade. The
// doctor checks run for real against fake tools.
func TestAutoSetup_OutdatedNixNotReinstalled(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fake tools are POSIX shell scripts")
	}
	bin := t.TempDir()
	for name, out := range map[string]string{
		"nix":    "nix (Nix) 2.3.0",
		"devenv": "devenv 2.1.2 (x86_64-linux)",
		"direnv": "2.34.0",
		"git":    "git version 2.47.1",
	} {
		writeFileMode(t, filepath.Join(bin, name), "#!/bin/sh\necho '"+out+"'\n", 0o755)
	}
	t.Setenv("PATH", bin)
	osInfo := &sysinfo.OSInfo{OS: "linux", Distro: "ubuntu", Family: "debian", HasNix: true}
	var installed []string
	deps := setupDeps{
		osInfo: osInfo,
		checks: func(ctx context.Context) []doctor.ToolStatus {
			return doctor.RunChecks(ctx, osInfo, doctor.RequiredChecks())
		},
		install: func(_ context.Context, _ io.Writer, name string) error {
			installed = append(installed, name)
			return nil
		},
	}
	nix, _ := doctor.CheckNamed("nix")

	var buf bytes.Buffer
	err := autoSetup(context.Background(), &buf, deps)
	if len(installed) != 0 {
		t.Errorf("setup installed %q; an outdated nix must not be reinstalled", installed)
	}
	if err == nil || !strings.Contains(err.Error(), "nix ("+nix.UpgradeHint+")") {
		t.Errorf("autoSetup = %v; want an error naming the nix upgrade", err)
	}
	if strings.Contains(buf.String(), "Prerequisites installed.") {
		t.Errorf("autoSetup claimed success:\n%s", buf.String())
	}
}

// TestRunSetup_RequiredManualToolFails: setup must not exit 0 while a
// required tool it cannot install or upgrade (an outdated nix) still
// fails its check, whether or not other tools were installed in the run.
func TestRunSetup_RequiredManualToolFails(t *testing.T) {
	t.Parallel()
	const hint = "upgrade Nix in place"
	nixOld := doctor.ToolStatus{Name: "nix", Required: true, Installed: true, Version: "2.3.0", MinVersion: "2.4", UpgradeHint: hint}
	devenvOK := doctor.ToolStatus{Name: "devenv", Required: true, Installed: true, Version: "2.1.2", MinVersion: "2.1", VersionOK: true}
	shellcheckMissing := doctor.ToolStatus{Name: "shellcheck", AutoInstallable: true}
	shellcheckOK := doctor.ToolStatus{Name: "shellcheck", Installed: true, VersionOK: true}
	tests := []struct {
		name          string
		pre, post     []doctor.ToolStatus
		wantInstalled []string
		wantErr       bool
	}{
		{"nix only", []doctor.ToolStatus{nixOld, devenvOK}, nil, nil, true},
		{"nix and installable optional", []doctor.ToolStatus{nixOld, devenvOK, shellcheckMissing},
			[]doctor.ToolStatus{nixOld, devenvOK, shellcheckOK}, []string{"shellcheck"}, true},
		{"optional manual only", []doctor.ToolStatus{devenvOK, {Name: "hadolint"}}, nil, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var installed []string
			calls := 0
			deps := setupDeps{
				osInfo: &sysinfo.OSInfo{OS: "linux", Distro: "ubuntu", Family: "debian", HasNix: true},
				install: func(_ context.Context, _ io.Writer, name string) error {
					installed = append(installed, name)
					return nil
				},
				checks: func(context.Context) []doctor.ToolStatus {
					calls++
					if calls == 1 {
						return tt.pre
					}
					return tt.post
				},
			}
			var buf bytes.Buffer
			err := runSetup(context.Background(), &buf, true, false, deps)

			if !slices.Equal(installed, tt.wantInstalled) {
				t.Errorf("installed = %v, want %v", installed, tt.wantInstalled)
			}
			if !tt.wantErr {
				if err != nil {
					t.Fatalf("runSetup: %v\n%s", err, buf.String())
				}
				if !strings.Contains(buf.String(), "Manual installation required for: hadolint") {
					t.Errorf("output does not list the optional manual tool:\n%s", buf.String())
				}
				return
			}
			if !errors.Is(err, ErrSetupIncomplete) {
				t.Fatalf("error = %v, want %v\n%s", err, ErrSetupIncomplete, buf.String())
			}
			want := "nix (" + hint + ")"
			if !strings.Contains(err.Error(), want) {
				t.Errorf("error %q missing %q", err, want)
			}
			if !strings.Contains(buf.String(), "Manual installation required for: "+want) {
				t.Errorf("output missing the manual nix upgrade:\n%s", buf.String())
			}
		})
	}
}
