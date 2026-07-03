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

	// Unavailable maps a package manager name to actionable guidance for tools
	// that have no installable package under that manager (e.g. "pre-commit"
	// has no winget package; it is installed via pip). When a manager is present
	// here, ResolvePackageName reports the tool as having no package rather than
	// falling back to a bare generic name that would form a broken install.
	Unavailable map[string]string
}

// ToolEntry describes a tool in the registry.
type ToolEntry struct {
	// Name is the canonical tool identifier (e.g. "go", "node").
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
		Packages: PackageNames{
			Generic: "git",
			ByManager: map[string]string{
				"winget": "Git.Git",
			},
		},
	},
	"curl": {
		Name: "curl", Binary: "curl", VersionFlag: "--version",
		Packages: PackageNames{
			Generic: "curl",
			ByManager: map[string]string{
				"winget": "cURL.cURL",
			},
		},
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
	// Keyed "node" (not "nodejs") to match the tool name that flows through
	// doctor checks and the setup toolLevels; the installable package name is
	// still "nodejs" on most Linux managers.
	"node": {
		Name: "node", Binary: "node", VersionFlag: "--version",
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
	// npm ships bundled with Node.js; there is no standalone winget package.
	"npm": {
		Name: "npm", Binary: "npm", VersionFlag: "--version",
		Packages: PackageNames{
			Generic: "npm",
			Unavailable: map[string]string{
				"winget": "it is bundled with Node.js — install the 'node' tool (winget package OpenJS.NodeJS.LTS)",
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
				"winget": "koalaman.shellcheck",
			},
		},
	},
	"direnv": {
		Name: "direnv", Binary: "direnv", VersionFlag: "version",
		Packages: PackageNames{
			Generic: "direnv",
			ByManager: map[string]string{
				"emerge": "dev-util/direnv",
				"winget": "direnv.direnv",
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
			Unavailable: map[string]string{
				"winget": "install it via pip (pip install pre-commit) or with Nix",
			},
		},
	},
	"shfmt": {
		Name: "shfmt", Binary: "shfmt", VersionFlag: "--version",
		Packages: PackageNames{
			Generic: "shfmt",
			ByManager: map[string]string{
				"nix":    "shfmt",
				"winget": "mvdan.shfmt",
			},
		},
	},
	"hadolint": {
		Name: "hadolint", Binary: "hadolint", VersionFlag: "--version",
		Packages: PackageNames{
			Generic: "hadolint",
			ByManager: map[string]string{
				"nix":    "hadolint",
				"winget": "hadolint.hadolint",
			},
		},
	},
}

// ResolvePackageName returns the best package name for the given tool,
// considering OS family and package manager overrides.
// Lookup order: ByManager → ByFamily → Generic.
// Returns ("", false) if the tool is not in the registry, or if the tool has
// no installable package for the given manager (see PackageUnavailable).
func ResolvePackageName(toolName, family, manager string) (string, bool) {
	entry, ok := toolRegistry[toolName]
	if !ok {
		return "", false
	}
	// A tool may have no installable package for a given manager (e.g. pre-commit
	// or npm on winget). Report no package rather than falling back to a generic
	// name that would form a broken install command.
	if manager != "" && entry.Packages.Unavailable != nil {
		if _, unavailable := entry.Packages.Unavailable[manager]; unavailable {
			return "", false
		}
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

// PackageUnavailable reports whether toolName has no installable package for the
// given package manager. When unavailable is true, remedy contains actionable
// guidance for installing the tool by other means (e.g. via pip, or bundled
// with another tool). It returns ("", false) for tools that are not in the
// registry or that do have a package for the manager.
func PackageUnavailable(toolName, manager string) (remedy string, unavailable bool) {
	entry, ok := toolRegistry[toolName]
	if !ok || manager == "" || entry.Packages.Unavailable == nil {
		return "", false
	}
	remedy, unavailable = entry.Packages.Unavailable[manager]
	return remedy, unavailable
}

// InstallCommand returns a human-readable install command string for the
// given tool, e.g. "brew install git" or "sudo apt-get install -y golang".
func InstallCommand(toolName, family, manager string) string {
	// Tools with no package for this manager get actionable guidance instead of
	// a broken command line (e.g. "winget install --id pre-commit -e").
	if remedy, unavailable := PackageUnavailable(toolName, manager); unavailable {
		return fmt.Sprintf("no %s package for %s; %s", manager, toolName, remedy)
	}

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
