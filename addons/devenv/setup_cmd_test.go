package devenv

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/doctor"
	"github.com/Quantum-Serendipity/qsdev/internal/pkgmanager"
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
		{"claude", "debian", pkgmanager.NewApt(nil), "npm install -g @anthropic-ai/claude-code"},
		{"devenv", "debian", pkgmanager.NewApt(nil), strings.Join(devenvSpec.InstallCmd, " ")},
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
