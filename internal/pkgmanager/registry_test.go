package pkgmanager

import (
	"strings"
	"testing"
)

func TestResolvePackageName(t *testing.T) {
	tests := []struct {
		name    string
		tool    string
		family  string
		manager string
		want    string
		wantOK  bool
	}{
		{
			name: "go on debian family",
			tool: "go", family: "debian", manager: "",
			want: "golang", wantOK: true,
		},
		{
			name: "go on apt manager",
			tool: "go", family: "", manager: "apt",
			want: "golang", wantOK: true,
		},
		{
			name: "go on emerge",
			tool: "go", family: "gentoo", manager: "emerge",
			want: "dev-lang/go", wantOK: true,
		},
		{
			name: "go on winget",
			tool: "go", family: "windows", manager: "winget",
			want: "GoLang.Go", wantOK: true,
		},
		{
			name: "go generic fallback",
			tool: "go", family: "arch", manager: "pacman",
			want: "go", wantOK: true,
		},
		{
			name: "shellcheck on rhel family",
			tool: "shellcheck", family: "rhel", manager: "dnf",
			want: "ShellCheck", wantOK: true,
		},
		{
			name: "shellcheck on zypper",
			tool: "shellcheck", family: "suse", manager: "zypper",
			want: "ShellCheck", wantOK: true,
		},
		{
			name: "node on winget (node key, not nodejs)",
			tool: "node", family: "", manager: "winget",
			want: "OpenJS.NodeJS.LTS", wantOK: true,
		},
		{
			name: "node on scoop",
			tool: "node", family: "windows", manager: "scoop",
			want: "nodejs-lts", wantOK: true,
		},
		{
			name: "node on choco",
			tool: "node", family: "windows", manager: "choco",
			want: "nodejs-lts", wantOK: true,
		},
		{
			name: "node generic (nodejs package on apt)",
			tool: "node", family: "debian", manager: "apt",
			want: "nodejs", wantOK: true,
		},
		{
			name: "git on winget",
			tool: "git", family: "", manager: "winget",
			want: "Git.Git", wantOK: true,
		},
		{
			name: "curl on winget",
			tool: "curl", family: "", manager: "winget",
			want: "cURL.cURL", wantOK: true,
		},
		{
			name: "shellcheck on winget",
			tool: "shellcheck", family: "", manager: "winget",
			want: "koalaman.shellcheck", wantOK: true,
		},
		{
			name: "shfmt on winget",
			tool: "shfmt", family: "", manager: "winget",
			want: "mvdan.shfmt", wantOK: true,
		},
		{
			name: "hadolint on winget",
			tool: "hadolint", family: "", manager: "winget",
			want: "hadolint.hadolint", wantOK: true,
		},
		{
			name: "direnv on winget",
			tool: "direnv", family: "", manager: "winget",
			want: "direnv.direnv", wantOK: true,
		},
		{
			name: "pre-commit has no winget package (no bare generic)",
			tool: "pre-commit", family: "", manager: "winget",
			want: "", wantOK: false,
		},
		{
			name: "pre-commit still installable via nix",
			tool: "pre-commit", family: "", manager: "nix",
			want: "pre-commit", wantOK: true,
		},
		{
			name: "npm has no winget package (no bare generic)",
			tool: "npm", family: "", manager: "winget",
			want: "", wantOK: false,
		},
		{
			name: "git on any platform",
			tool: "git", family: "debian", manager: "apt",
			want: "git", wantOK: true,
		},
		{
			name: "git with empty family/manager",
			tool: "git", family: "", manager: "",
			want: "git", wantOK: true,
		},
		{
			name: "python3 on arch",
			tool: "python3", family: "arch", manager: "pacman",
			want: "python", wantOK: true,
		},
		{
			name: "python3 on winget",
			tool: "python3", family: "windows", manager: "winget",
			want: "Python.Python.3.11", wantOK: true,
		},
		{
			name: "direnv on emerge",
			tool: "direnv", family: "gentoo", manager: "emerge",
			want: "dev-util/direnv", wantOK: true,
		},
		{
			name: "jq on winget",
			tool: "jq", family: "windows", manager: "winget",
			want: "jqlang.jq", wantOK: true,
		},
		{
			name: "jq on emerge",
			tool: "jq", family: "gentoo", manager: "emerge",
			want: "app-misc/jq", wantOK: true,
		},
		{
			// The package must provide the "rustup" binary the entry checks;
			// dev-lang/rust ships only rustc/cargo.
			name: "rustup on emerge",
			tool: "rustup", family: "gentoo", manager: "emerge",
			want: "dev-util/rustup", wantOK: true,
		},
		{
			// Arch's plain nodejs package tracks a supported release, unlike
			// the pinned nodejs-lts-iron (Node 20, end of life).
			name: "node on arch",
			tool: "node", family: "arch", manager: "pacman",
			want: "nodejs", wantOK: true,
		},
		{
			name: "unknown tool",
			tool: "nonexistent-tool", family: "debian", manager: "apt",
			want: "", wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ResolvePackageName(tt.tool, tt.family, tt.manager)
			if ok != tt.wantOK {
				t.Errorf("ResolvePackageName(%q, %q, %q) ok=%v, want %v", tt.tool, tt.family, tt.manager, ok, tt.wantOK)
			}
			if got != tt.want {
				t.Errorf("ResolvePackageName(%q, %q, %q) = %q, want %q", tt.tool, tt.family, tt.manager, got, tt.want)
			}
		})
	}
}

func TestResolvePackageNameManagerOverridesFamily(t *testing.T) {
	// When both family and manager have entries, manager should win.
	// For "go": family "debian" -> "golang", manager "emerge" -> "dev-lang/go"
	got, ok := ResolvePackageName("go", "debian", "emerge")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != "dev-lang/go" {
		t.Errorf("expected manager override 'dev-lang/go', got %q", got)
	}
}

// managerWith returns the named manager backed by a mock runner on which only
// the manager's own binary is installed, so InstallArgs is deterministic.
func managerWith(t *testing.T, name string) PackageManager {
	t.Helper()
	mock := NewMockRunner()
	bin := map[string]string{"apt": "apt-get", "xbps": "xbps-install"}[name]
	if bin == "" {
		bin = name
	}
	mock.LookPathResults[bin] = lookPathResult{path: "/usr/bin/" + bin}
	pm := managerByName(name, mock)
	if pm.Name() != name {
		t.Fatalf("managerByName(%q) = %q", name, pm.Name())
	}
	return pm
}

func TestInstallCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		tool    string
		family  string
		manager string
		want    string
	}{
		{
			name: "brew install git",
			tool: "git", family: "macos", manager: "brew",
			want: "brew install git",
		},
		{
			name: "apt install golang",
			tool: "go", family: "debian", manager: "apt",
			want: "sudo apt-get install -y golang",
		},
		{
			name: "dnf install ShellCheck",
			tool: "shellcheck", family: "rhel", manager: "dnf",
			want: "sudo dnf install -y ShellCheck",
		},
		{
			name: "pacman install python",
			tool: "python3", family: "arch", manager: "pacman",
			want: "sudo pacman -S --noconfirm python",
		},
		{
			name: "nix profile install",
			tool: "git", family: "", manager: "nix",
			want: "nix profile install nixpkgs#git",
		},
		{
			name: "nix ignores debian family names",
			tool: "go", family: "debian", manager: "nix",
			want: "nix profile install nixpkgs#go",
		},
		{
			name: "nix ignores dnf manager names",
			tool: "shellcheck", family: "rhel", manager: "nix",
			want: "nix profile install nixpkgs#shellcheck",
		},
		{
			name: "nix make is gnumake",
			tool: "make", family: "debian", manager: "nix",
			want: "nix profile install nixpkgs#gnumake",
		},
		{
			name: "nix npm ships with nodejs",
			tool: "npm", family: "", manager: "nix",
			want: "nix profile install nixpkgs#nodejs",
		},
		{
			name: "winget install",
			tool: "node", family: "windows", manager: "winget",
			want: "winget install --id OpenJS.NodeJS.LTS -e --accept-source-agreements --accept-package-agreements",
		},
		{
			name: "winget install git",
			tool: "git", family: "windows", manager: "winget",
			want: "winget install --id Git.Git -e --accept-source-agreements --accept-package-agreements",
		},
		{
			name: "emerge with category",
			tool: "go", family: "gentoo", manager: "emerge",
			want: "sudo emerge --ask=n dev-lang/go",
		},
		{
			name: "unknown tool returns empty",
			tool: "nonexistent", family: "debian", manager: "apt",
			want: "",
		},
		{
			name: "scoop install",
			tool: "node", family: "windows", manager: "scoop",
			want: "scoop install nodejs-lts",
		},
		{
			name: "choco install",
			tool: "node", family: "windows", manager: "choco",
			want: "choco install -y nodejs-lts",
		},
		{
			name: "zypper install",
			tool: "shellcheck", family: "suse", manager: "zypper",
			want: "sudo zypper install -y ShellCheck",
		},
		{
			name: "apk add",
			tool: "git", family: "alpine", manager: "apk",
			want: "sudo apk add git",
		},
		{
			name: "xbps install",
			tool: "git", family: "void", manager: "xbps",
			want: "sudo xbps-install -y git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := InstallCommand(managerWith(t, tt.manager), tt.family, tt.tool)
			if got != tt.want {
				t.Errorf("InstallCommand(%s, %q, %q) = %q, want %q", tt.manager, tt.family, tt.tool, got, tt.want)
			}
		})
	}
}

// TestInstallCommandYumOnlyHost pins F474: on a host with yum but no dnf the
// displayed command must be the yum command Install runs, with sudo and -y and
// the dnf-family package names.
func TestInstallCommandYumOnlyHost(t *testing.T) {
	t.Parallel()
	mock := NewMockRunner()
	mock.LookPathResults["yum"] = lookPathResult{path: "/usr/bin/yum"}
	pm := managerByName("yum", mock)

	tests := []struct{ tool, want string }{
		{"shellcheck", "sudo yum install -y ShellCheck"},
		{"go", "sudo yum install -y golang"},
	}
	for _, tt := range tests {
		t.Run(tt.tool, func(t *testing.T) {
			t.Parallel()
			if got := InstallCommand(pm, "rhel", tt.tool); got != tt.want {
				t.Errorf("InstallCommand(yum-only, %q) = %q, want %q", tt.tool, got, tt.want)
			}
		})
	}
}

// TestResolvePackageNameManagerAliases verifies manager aliases resolve to
// the canonical manager's overrides (yum -> dnf, apt-get -> apt).
func TestResolvePackageNameManagerAliases(t *testing.T) {
	t.Parallel()
	tests := []struct{ tool, manager, want string }{
		{"shellcheck", "yum", "ShellCheck"},
		{"go", "apt-get", "golang"},
		{"go", "portage", "dev-lang/go"},
	}
	for _, tt := range tests {
		t.Run(tt.manager, func(t *testing.T) {
			t.Parallel()
			got, ok := ResolvePackageName(tt.tool, "", tt.manager)
			if !ok || got != tt.want {
				t.Errorf("ResolvePackageName(%q, \"\", %q) = %q, %v; want %q", tt.tool, tt.manager, got, ok, tt.want)
			}
		})
	}
}

func TestPackageUnavailable(t *testing.T) {
	tests := []struct {
		name            string
		tool            string
		manager         string
		wantUnavailable bool
	}{
		{"pre-commit has no winget package", "pre-commit", "winget", true},
		{"npm has no winget package", "npm", "winget", true},
		{"pre-commit is installable via nix", "pre-commit", "nix", false},
		{"git has a winget package", "git", "winget", false},
		{"node has a winget package", "node", "winget", false},
		{"unknown tool is not marked unavailable", "nonexistent", "winget", false},
		{"empty manager is never unavailable", "pre-commit", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			remedy, unavailable := PackageUnavailable(tt.tool, tt.manager)
			if unavailable != tt.wantUnavailable {
				t.Errorf("PackageUnavailable(%q, %q) unavailable=%v, want %v",
					tt.tool, tt.manager, unavailable, tt.wantUnavailable)
			}
			if unavailable && remedy == "" {
				t.Errorf("PackageUnavailable(%q, %q) returned empty remedy for an unavailable tool",
					tt.tool, tt.manager)
			}
			if !unavailable && remedy != "" {
				t.Errorf("PackageUnavailable(%q, %q) returned remedy %q for an available tool",
					tt.tool, tt.manager, remedy)
			}
		})
	}
}

func TestInstallCommandNoWingetPackage(t *testing.T) {
	// Tools without a standalone winget package must not produce a broken
	// "winget install --id <generic> -e"; they must surface actionable guidance.
	tests := []struct {
		name     string
		tool     string
		contains []string
	}{
		{"pre-commit points to pip", "pre-commit", []string{"no winget package for pre-commit", "pip"}},
		{"npm points to Node.js", "npm", []string{"no winget package for npm", "Node.js"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := InstallCommand(NewWinget(nil), "", tt.tool)
			if strings.HasPrefix(got, "winget install --id") {
				t.Fatalf("InstallCommand(%q, winget) = %q, must not emit a bare winget install", tt.tool, got)
			}
			for _, sub := range tt.contains {
				if !strings.Contains(got, sub) {
					t.Errorf("InstallCommand(%q, winget) = %q, want it to contain %q", tt.tool, got, sub)
				}
			}
		})
	}
}

func TestRegistryCompleteness(t *testing.T) {
	// Verify all core tools are present.
	expectedTools := []string{
		"git", "curl", "wget", "jq", "go", "node", "python3",
		"shellcheck", "direnv", "make", "docker", "terraform",
		"rustup", "unzip", "tree",
	}
	for _, name := range expectedTools {
		if _, ok := toolRegistry[name]; !ok {
			t.Errorf("expected tool %q in registry", name)
		}
	}
}

func TestInstallCommandSudoPresence(t *testing.T) {
	// Elevated managers should include "sudo" in the human-readable command.
	elevatedManagers := []string{"apt", "dnf", "pacman", "zypper", "apk", "xbps", "emerge"}
	for _, mgr := range elevatedManagers {
		cmd := InstallCommand(managerWith(t, mgr), "", "git")
		if !strings.HasPrefix(cmd, "sudo ") {
			t.Errorf("InstallCommand for %s should start with 'sudo', got: %s", mgr, cmd)
		}
	}

	// Non-elevated managers should not include "sudo".
	nonElevatedManagers := []string{"brew", "nix", "winget", "scoop", "choco"}
	for _, mgr := range nonElevatedManagers {
		cmd := InstallCommand(managerWith(t, mgr), "", "git")
		if strings.HasPrefix(cmd, "sudo ") {
			t.Errorf("InstallCommand for %s should not start with 'sudo', got: %s", mgr, cmd)
		}
	}
}
