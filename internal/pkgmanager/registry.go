package pkgmanager

import "fmt"

// PackageNames holds platform-specific package names for a tool.
type PackageNames struct {
	// Generic is the default package name used when no family/manager-specific
	// override exists.
	Generic string

	// ByFamily maps OSInfo.Family to the package name for that family.
	ByFamily map[string]string

	// ByManager maps package manager name to the package name for that manager.
	ByManager map[string]string
}

// ToolEntry describes a tool in the registry.
type ToolEntry struct {
	// Name is the canonical tool identifier (e.g. "go", "nodejs").
	Name string

	// Binary is the executable name to look up on PATH.
	Binary string

	// VersionFlag is the flag passed to Binary to get the version string.
	VersionFlag string

	// Packages holds the platform-specific package names.
	Packages PackageNames
}

// toolRegistry holds the built-in tool-to-package-name mappings.
var toolRegistry = map[string]ToolEntry{
	"git": {
		Name: "git", Binary: "git", VersionFlag: "--version",
		Packages: PackageNames{Generic: "git"},
	},
	"curl": {
		Name: "curl", Binary: "curl", VersionFlag: "--version",
		Packages: PackageNames{Generic: "curl"},
	},
	"wget": {
		Name: "wget", Binary: "wget", VersionFlag: "--version",
		Packages: PackageNames{Generic: "wget"},
	},
	"jq": {
		Name: "jq", Binary: "jq", VersionFlag: "--version",
		Packages: PackageNames{
			Generic: "jq",
			ByManager: map[string]string{
				"winget": "jqlang.jq",
				"emerge": "app-misc/jq",
			},
		},
	},
	"go": {
		Name: "go", Binary: "go", VersionFlag: "version",
		Packages: PackageNames{
			Generic: "go",
			ByFamily: map[string]string{
				"debian": "golang",
				"rhel":   "golang",
			},
			ByManager: map[string]string{
				"apt":    "golang",
				"dnf":    "golang",
				"winget": "GoLang.Go",
				"emerge": "dev-lang/go",
			},
		},
	},
	"nodejs": {
		Name: "nodejs", Binary: "node", VersionFlag: "--version",
		Packages: PackageNames{
			Generic: "nodejs",
			ByFamily: map[string]string{
				"arch": "nodejs-lts-iron",
			},
			ByManager: map[string]string{
				"winget": "OpenJS.NodeJS.LTS",
				"scoop":  "nodejs-lts",
				"choco":  "nodejs-lts",
				"emerge": "net-libs/nodejs",
			},
		},
	},
	"python3": {
		Name: "python3", Binary: "python3", VersionFlag: "--version",
		Packages: PackageNames{
			Generic: "python3",
			ByFamily: map[string]string{
				"arch": "python",
			},
			ByManager: map[string]string{
				"winget": "Python.Python.3.11",
				"pacman": "python",
				"emerge": "dev-lang/python",
			},
		},
	},
	"shellcheck": {
		Name: "shellcheck", Binary: "shellcheck", VersionFlag: "--version",
		Packages: PackageNames{
			Generic: "shellcheck",
			ByManager: map[string]string{
				"dnf":    "ShellCheck",
				"zypper": "ShellCheck",
				"emerge": "dev-util/shellcheck",
			},
		},
	},
	"direnv": {
		Name: "direnv", Binary: "direnv", VersionFlag: "version",
		Packages: PackageNames{
			Generic: "direnv",
			ByManager: map[string]string{
				"emerge": "dev-util/direnv",
			},
		},
	},
	"make": {
		Name: "make", Binary: "make", VersionFlag: "--version",
		Packages: PackageNames{
			Generic: "make",
			ByFamily: map[string]string{
				"debian": "build-essential",
			},
			ByManager: map[string]string{
				"apt":    "build-essential",
				"emerge": "sys-devel/make",
			},
		},
	},
	"docker": {
		Name: "docker", Binary: "docker", VersionFlag: "--version",
		Packages: PackageNames{
			Generic: "docker",
			ByFamily: map[string]string{
				"debian": "docker.io",
			},
			ByManager: map[string]string{
				"apt":    "docker.io",
				"winget": "Docker.DockerDesktop",
				"emerge": "app-containers/docker",
			},
		},
	},
	"terraform": {
		Name: "terraform", Binary: "terraform", VersionFlag: "--version",
		Packages: PackageNames{
			Generic: "terraform",
			ByManager: map[string]string{
				"winget": "Hashicorp.Terraform",
			},
		},
	},
	"rustup": {
		Name: "rustup", Binary: "rustup", VersionFlag: "--version",
		Packages: PackageNames{
			Generic: "rustup",
			ByManager: map[string]string{
				"emerge": "dev-lang/rust",
			},
		},
	},
	"unzip": {
		Name: "unzip", Binary: "unzip", VersionFlag: "-v",
		Packages: PackageNames{
			Generic: "unzip",
			ByManager: map[string]string{
				"emerge": "app-arch/unzip",
			},
		},
	},
	"tree": {
		Name: "tree", Binary: "tree", VersionFlag: "--version",
		Packages: PackageNames{Generic: "tree"},
	},
	"devenv": {
		Name: "devenv", Binary: "devenv", VersionFlag: "version",
		Packages: PackageNames{
			Generic: "devenv",
			ByManager: map[string]string{
				"nix": "devenv",
			},
		},
	},
	"pre-commit": {
		Name: "pre-commit", Binary: "pre-commit", VersionFlag: "--version",
		Packages: PackageNames{
			Generic: "pre-commit",
			ByManager: map[string]string{
				"nix": "pre-commit",
			},
		},
	},
	"shfmt": {
		Name: "shfmt", Binary: "shfmt", VersionFlag: "--version",
		Packages: PackageNames{
			Generic: "shfmt",
			ByManager: map[string]string{
				"nix": "shfmt",
			},
		},
	},
	"hadolint": {
		Name: "hadolint", Binary: "hadolint", VersionFlag: "--version",
		Packages: PackageNames{
			Generic: "hadolint",
			ByManager: map[string]string{
				"nix": "hadolint",
			},
		},
	},
}

// ResolvePackageName returns the best package name for the given tool,
// considering OS family and package manager overrides.
// Lookup order: ByManager → ByFamily → Generic.
// Returns ("", false) if the tool is not in the registry.
func ResolvePackageName(toolName, family, manager string) (string, bool) {
	entry, ok := toolRegistry[toolName]
	if !ok {
		return "", false
	}
	// Prefer manager-specific name.
	if manager != "" && entry.Packages.ByManager != nil {
		if name, ok := entry.Packages.ByManager[manager]; ok {
			return name, true
		}
	}
	// Then family-specific name.
	if family != "" && entry.Packages.ByFamily != nil {
		if name, ok := entry.Packages.ByFamily[family]; ok {
			return name, true
		}
	}
	// Fall back to generic.
	if entry.Packages.Generic != "" {
		return entry.Packages.Generic, true
	}
	return "", false
}

// LookupTool returns the ToolEntry for the given tool name, if it exists.
func LookupTool(name string) (ToolEntry, bool) {
	e, ok := toolRegistry[name]
	return e, ok
}

// InstallCommand returns a human-readable install command string for the
// given tool, e.g. "brew install git" or "sudo apt-get install -y golang".
func InstallCommand(toolName, family, manager string) string {
	pkgName, ok := ResolvePackageName(toolName, family, manager)
	if !ok {
		return ""
	}

	switch manager {
	case "apt":
		return fmt.Sprintf("sudo apt-get install -y %s", pkgName)
	case "dnf":
		return fmt.Sprintf("sudo dnf install -y %s", pkgName)
	case "pacman":
		return fmt.Sprintf("sudo pacman -S --noconfirm %s", pkgName)
	case "zypper":
		return fmt.Sprintf("sudo zypper install -y %s", pkgName)
	case "apk":
		return fmt.Sprintf("sudo apk add %s", pkgName)
	case "xbps":
		return fmt.Sprintf("sudo xbps-install -y %s", pkgName)
	case "emerge":
		return fmt.Sprintf("sudo emerge %s", pkgName)
	case "brew":
		return fmt.Sprintf("brew install %s", pkgName)
	case "nix":
		return fmt.Sprintf("nix profile install nixpkgs#%s", pkgName)
	case "winget":
		return fmt.Sprintf("winget install --id %s -e", pkgName)
	case "scoop":
		return fmt.Sprintf("scoop install %s", pkgName)
	case "choco":
		return fmt.Sprintf("choco install -y %s", pkgName)
	default:
		// Generic fallback.
		return fmt.Sprintf("%s install %s", manager, pkgName)
	}
}
