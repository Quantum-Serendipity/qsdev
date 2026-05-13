package doctor

import (
	"os/exec"
	"strings"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/sysinfo"
)

// ToolCheck defines how to detect and classify a single tool.
type ToolCheck struct {
	Name         string
	Binary       string
	AltBinaries  []string
	VersionFlag  string
	Required     bool
	MinVersion   string
	ParseVersion func(raw string) string
	AutoInstall  func(osInfo *sysinfo.OSInfo) bool
	Notes        func(osInfo *sysinfo.OSInfo) string
}

// DefaultChecks returns the 15-tool registry used by gdev doctor.
func DefaultChecks() []ToolCheck {
	return []ToolCheck{
		{
			Name:        "git",
			Binary:      "git",
			VersionFlag: "--version",
			Required:    true,
			ParseVersion: func(raw string) string {
				// "git version 2.43.0" → "2.43.0"
				return extractLastField(raw, "git version ")
			},
			AutoInstall: alwaysInstallable,
		},
		{
			Name:        "go",
			Binary:      "go",
			VersionFlag: "version",
			Required:    true,
			ParseVersion: func(raw string) string {
				// "go version go1.22.3 linux/amd64" → "1.22.3"
				parts := strings.Fields(raw)
				for _, p := range parts {
					if strings.HasPrefix(p, "go") && len(p) > 2 {
						v := strings.TrimPrefix(p, "go")
						if len(v) > 0 && v[0] >= '0' && v[0] <= '9' {
							return v
						}
					}
				}
				return ""
			},
			AutoInstall: alwaysInstallable,
		},
		{
			Name:        "node",
			Binary:      "node",
			VersionFlag: "--version",
			Required:    true,
			ParseVersion: func(raw string) string {
				// "v20.11.0" → "20.11.0"
				return strings.TrimPrefix(strings.TrimSpace(raw), "v")
			},
			AutoInstall: alwaysInstallable,
		},
		{
			Name:        "npm",
			Binary:      "npm",
			VersionFlag: "--version",
			Required:    true,
			ParseVersion: func(raw string) string {
				// Already clean like "10.2.3"
				return strings.TrimSpace(raw)
			},
			AutoInstall: alwaysInstallable,
		},
		{
			Name:        "nix",
			Binary:      "nix",
			VersionFlag: "--version",
			Required:    false,
			ParseVersion: func(raw string) string {
				// "nix (Nix) 2.19.3" → "2.19.3"
				parts := strings.Fields(raw)
				if len(parts) >= 3 {
					return parts[len(parts)-1]
				}
				return ""
			},
			AutoInstall: func(osInfo *sysinfo.OSInfo) bool {
				return osInfo.OS != "windows" || osInfo.IsWSL || osInfo.IsWSL2
			},
		},
		{
			Name:        "devenv",
			Binary:      "devenv",
			VersionFlag: "version",
			Required:    false,
			ParseVersion: func(raw string) string {
				// devenv version output may be "devenv 1.4.1" or just "1.4.1"
				raw = strings.TrimSpace(raw)
				parts := strings.Fields(raw)
				if len(parts) == 0 {
					return ""
				}
				return parts[len(parts)-1]
			},
			AutoInstall: func(osInfo *sysinfo.OSInfo) bool {
				return osInfo.HasNix
			},
			Notes: func(osInfo *sysinfo.OSInfo) string {
				if !osInfo.HasNix {
					return "Requires Nix"
				}
				return ""
			},
		},
		{
			Name:        "direnv",
			Binary:      "direnv",
			VersionFlag: "--version",
			Required:    false,
			ParseVersion: func(raw string) string {
				return strings.TrimSpace(raw)
			},
			AutoInstall: alwaysInstallable,
		},
		{
			Name:        "claude",
			Binary:      "claude",
			VersionFlag: "--version",
			Required:    false,
			ParseVersion: func(raw string) string {
				return strings.TrimSpace(raw)
			},
			AutoInstall: func(_ *sysinfo.OSInfo) bool {
				_, err := exec.LookPath("npm")
				return err == nil
			},
			Notes: func(_ *sysinfo.OSInfo) string {
				return "Installed via npm"
			},
		},
		{
			Name:        "pre-commit",
			Binary:      "pre-commit",
			AltBinaries: []string{"prek"},
			VersionFlag: "--version",
			Required:    false,
			ParseVersion: func(raw string) string {
				// "pre-commit 3.7.0" → "3.7.0"
				return extractLastField(raw, "pre-commit ")
			},
			AutoInstall: alwaysInstallable,
		},
		{
			Name:        "shellcheck",
			Binary:      "shellcheck",
			VersionFlag: "--version",
			Required:    false,
			ParseVersion: func(raw string) string {
				// Multiline output: version is on the line starting with "version:"
				for _, line := range strings.Split(raw, "\n") {
					line = strings.TrimSpace(line)
					if strings.HasPrefix(line, "version:") {
						return strings.TrimSpace(strings.TrimPrefix(line, "version:"))
					}
				}
				return ""
			},
			AutoInstall: alwaysInstallable,
		},
		{
			Name:        "shfmt",
			Binary:      "shfmt",
			VersionFlag: "--version",
			Required:    false,
			ParseVersion: func(raw string) string {
				// "v3.8.0" → "3.8.0"
				return strings.TrimPrefix(strings.TrimSpace(raw), "v")
			},
			AutoInstall: alwaysInstallable,
		},
		{
			Name:        "hadolint",
			Binary:      "hadolint",
			VersionFlag: "--version",
			Required:    false,
			ParseVersion: func(raw string) string {
				// "Haskell Dockerfile Linter 2.12.0-no-git" → "2.12.0"
				parts := strings.Fields(raw)
				if len(parts) == 0 {
					return ""
				}
				ver := parts[len(parts)-1]
				// Strip everything from the first hyphen onward
				if idx := strings.Index(ver, "-"); idx > 0 {
					ver = ver[:idx]
				}
				return ver
			},
			AutoInstall: func(osInfo *sysinfo.OSInfo) bool {
				return osInfo.Family == "macos" || osInfo.HasNix
			},
		},
		{
			Name:        "jq",
			Binary:      "jq",
			VersionFlag: "--version",
			Required:    false,
			ParseVersion: func(raw string) string {
				// "jq-1.7.1" → "1.7.1"
				raw = strings.TrimSpace(raw)
				if strings.HasPrefix(raw, "jq-") {
					return strings.TrimPrefix(raw, "jq-")
				}
				return raw
			},
			AutoInstall: alwaysInstallable,
		},
		{
			Name:        "curl",
			Binary:      "curl",
			VersionFlag: "--version",
			Required:    false,
			ParseVersion: func(raw string) string {
				// "curl 8.5.0 (x86_64-pc-linux-gnu)" → "8.5.0"
				parts := strings.Fields(raw)
				if len(parts) >= 2 {
					return parts[1]
				}
				return ""
			},
			AutoInstall: alwaysInstallable,
		},
		{
			Name:        "python3",
			Binary:      "python3",
			VersionFlag: "--version",
			Required:    false,
			MinVersion:  "3.11",
			ParseVersion: func(raw string) string {
				// "Python 3.11.7" → "3.11.7"
				return extractLastField(raw, "Python ")
			},
			AutoInstall: alwaysInstallable,
		},
	}
}

// alwaysInstallable returns true for any OS.
func alwaysInstallable(_ *sysinfo.OSInfo) bool {
	return true
}

// extractLastField extracts the version after a known prefix,
// returning "" on empty/unexpected input.
func extractLastField(raw, prefix string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, prefix) {
		return strings.TrimSpace(strings.TrimPrefix(raw, prefix))
	}
	// Fallback: take the last field
	parts := strings.Fields(raw)
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}
