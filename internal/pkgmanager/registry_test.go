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
			name: "nodejs on winget",
			tool: "nodejs", family: "windows", manager: "winget",
			want: "OpenJS.NodeJS.LTS", wantOK: true,
		},
		{
			name: "nodejs on scoop",
			tool: "nodejs", family: "windows", manager: "scoop",
			want: "nodejs-lts", wantOK: true,
		},
		{
			name: "nodejs on choco",
			tool: "nodejs", family: "windows", manager: "choco",
			want: "nodejs-lts", wantOK: true,
		},
		{
			name: "nodejs generic",
			tool: "nodejs", family: "debian", manager: "apt",
			want: "nodejs", wantOK: true,
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

func TestInstallCommand(t *testing.T) {
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
			name: "winget install",
			tool: "nodejs", family: "windows", manager: "winget",
			want: "winget install --id OpenJS.NodeJS.LTS -e",
		},
		{
			name: "emerge with category",
			tool: "go", family: "gentoo", manager: "emerge",
			want: "sudo emerge dev-lang/go",
		},
		{
			name: "unknown tool returns empty",
			tool: "nonexistent", family: "debian", manager: "apt",
			want: "",
		},
		{
			name: "scoop install",
			tool: "nodejs", family: "windows", manager: "scoop",
			want: "scoop install nodejs-lts",
		},
		{
			name: "choco install",
			tool: "nodejs", family: "windows", manager: "choco",
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
			got := InstallCommand(tt.tool, tt.family, tt.manager)
			if got != tt.want {
				t.Errorf("InstallCommand(%q, %q, %q) = %q, want %q", tt.tool, tt.family, tt.manager, got, tt.want)
			}
		})
	}
}

func TestLookupTool(t *testing.T) {
	entry, ok := LookupTool("git")
	if !ok {
		t.Fatal("expected git to be in registry")
	}
	if entry.Binary != "git" {
		t.Errorf("expected binary 'git', got %q", entry.Binary)
	}
	if entry.VersionFlag != "--version" {
		t.Errorf("expected version flag '--version', got %q", entry.VersionFlag)
	}

	_, ok = LookupTool("nonexistent")
	if ok {
		t.Error("expected nonexistent tool to not be in registry")
	}
}

func TestRegistryCompleteness(t *testing.T) {
	// Verify all 15 tools are present.
	expectedTools := []string{
		"git", "curl", "wget", "jq", "go", "nodejs", "python3",
		"shellcheck", "direnv", "make", "docker", "terraform",
		"rustup", "unzip", "tree",
	}
	for _, name := range expectedTools {
		if _, ok := LookupTool(name); !ok {
			t.Errorf("expected tool %q in registry", name)
		}
	}
}

func TestInstallCommandSudoPresence(t *testing.T) {
	// Elevated managers should include "sudo" in the human-readable command.
	elevatedManagers := []string{"apt", "dnf", "pacman", "zypper", "apk", "xbps", "emerge"}
	for _, mgr := range elevatedManagers {
		cmd := InstallCommand("git", "", mgr)
		if !strings.HasPrefix(cmd, "sudo ") {
			t.Errorf("InstallCommand for %s should start with 'sudo', got: %s", mgr, cmd)
		}
	}

	// Non-elevated managers should not include "sudo".
	nonElevatedManagers := []string{"brew", "nix", "winget", "scoop", "choco"}
	for _, mgr := range nonElevatedManagers {
		cmd := InstallCommand("git", "", mgr)
		if strings.HasPrefix(cmd, "sudo ") {
			t.Errorf("InstallCommand for %s should not start with 'sudo', got: %s", mgr, cmd)
		}
	}
}
