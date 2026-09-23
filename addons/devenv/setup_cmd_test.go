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
	tests := []struct {
		name   string
		family string
		mgr    string
		want   string
	}{
		{"nix", "debian", "apt", "curl -sSf -L https://install.determinate.systems/nix | sh -s -- install"},
		{"claude", "debian", "apt", "npm install -g @anthropic-ai/claude-code"},
		{"devenv", "debian", "apt", strings.Join(devenvSpec.InstallCmd, " ")},
		{"git", "debian", "apt", "sudo apt-get install -y git"},
		{"git", "macos", "brew", "brew install git"},
	}

	for _, tt := range tests {
		got := installCommandForTool(tt.name, tt.family, tt.mgr)
		if got != tt.want {
			t.Errorf("installCommandForTool(%q, %q, %q) = %q, want %q", tt.name, tt.family, tt.mgr, got, tt.want)
		}
	}
}

func TestSetupCmd_PmInstallArgs(t *testing.T) {
	// Verify that pmInstallArgs produces correct arguments for each PM type.
	// We use a helper mock that just returns a name.
	tests := []struct {
		pmName string
		pkg    string
		want   []string
	}{
		{"apt", "git", []string{"install", "-y", "git"}},
		{"dnf", "git", []string{"install", "-y", "git"}},
		{"pacman", "git", []string{"-S", "--noconfirm", "git"}},
		{"apk", "git", []string{"add", "git"}},
		{"xbps", "git", []string{"-y", "git"}},
		{"emerge", "git", []string{"--ask=n", "git"}},
		{"brew", "git", []string{"install", "git"}},
	}

	for _, tt := range tests {
		pm := &pmNameOnly{name: tt.pmName}
		got := pmInstallArgs(pm, tt.pkg)
		if len(got) != len(tt.want) {
			t.Errorf("pmInstallArgs(%q, %q): len=%d, want len=%d", tt.pmName, tt.pkg, len(got), len(tt.want))
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("pmInstallArgs(%q, %q)[%d]=%q, want %q", tt.pmName, tt.pkg, i, got[i], tt.want[i])
			}
		}
	}
}

// pmNameOnly satisfies pkgmanager.PackageManager for Name() only.
// Other methods panic; only Name() is tested here.
type pmNameOnly struct {
	name string
}

func (p *pmNameOnly) Name() string                                 { return p.name }
func (p *pmNameOnly) Available() bool                              { panic("unused") }
func (p *pmNameOnly) NeedsElevation() bool                         { panic("unused") }
func (p *pmNameOnly) Install(_ context.Context, _ ...string) error { panic("unused") }
