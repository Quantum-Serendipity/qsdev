package doctor

import (
	"encoding/json"
	"os/exec"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/sysinfo"
	"github.com/Quantum-Serendipity/qsdev/internal/toolcheck"
)

// ToolCheck defines how to detect and classify a single tool.
type ToolCheck struct {
	Name        string
	Binary      string
	AltBinaries []string
	VersionFlag string
	Required    bool
	MinVersion  string
	// InstallHint tells the user how to install a missing tool.
	InstallHint string
	// ParseVersion extracts the version from the tool's full version output
	// (which may span several lines).
	ParseVersion func(raw string) string
	AutoInstall  func(osInfo *sysinfo.OSInfo) bool
	Notes        func(osInfo *sysinfo.OSInfo) string
}

// DefaultChecks returns the 20-tool registry used by qsdev doctor. It is the
// single prerequisite registry: the tools marked Required are those the
// devenv environment cannot work without (nix, devenv, direnv, git), and are
// also what init checks before generating (see RequiredChecks). Language
// toolchains are optional because devenv provides them per project.
func DefaultChecks() []ToolCheck {
	return []ToolCheck{
		{
			Name:        "nix",
			Binary:      "nix",
			VersionFlag: "--version",
			Required:    true,
			InstallHint: "Install Nix: https://nixos.org/download.html",
			ParseVersion: func(raw string) string {
				// "nix (Nix) 2.19.3" → "2.19.3"
				parts := strings.Fields(toolcheck.FirstLine(raw))
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
			Required:    true,
			InstallHint: "Install devenv: https://devenv.sh/getting-started/",
			ParseVersion: func(raw string) string {
				// "devenv 2.1.2 (x86_64-linux)" or just "1.4.1" → the version field
				parts := strings.Fields(toolcheck.FirstLine(raw))
				if len(parts) >= 2 && parts[0] == "devenv" {
					return parts[1]
				}
				for _, p := range parts {
					if p[0] >= '0' && p[0] <= '9' {
						return p
					}
				}
				return ""
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
			Required:    true,
			InstallHint: "Install direnv: https://direnv.net/docs/installation.html",
			ParseVersion: func(raw string) string {
				return toolcheck.FirstLine(raw)
			},
			AutoInstall: alwaysInstallable,
		},
		{
			Name:        "git",
			Binary:      "git",
			VersionFlag: "--version",
			Required:    true,
			InstallHint: "Install git via your system package manager.",
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
			Required:    false,
			ParseVersion: func(raw string) string {
				// "go version go1.22.3 linux/amd64" → "1.22.3"
				for p := range strings.FieldsSeq(toolcheck.FirstLine(raw)) {
					if v, ok := strings.CutPrefix(p, "go"); ok && len(v) > 0 && v[0] >= '0' && v[0] <= '9' {
						return v
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
			Required:    false,
			ParseVersion: func(raw string) string {
				// "v20.11.0" → "20.11.0"
				return strings.TrimPrefix(toolcheck.FirstLine(raw), "v")
			},
			AutoInstall: alwaysInstallable,
		},
		{
			Name:        "npm",
			Binary:      "npm",
			VersionFlag: "--version",
			Required:    false,
			ParseVersion: func(raw string) string {
				// Already clean like "10.2.3"
				return toolcheck.FirstLine(raw)
			},
			AutoInstall: alwaysInstallable,
		},
		{
			Name:        "claude",
			Binary:      "claude",
			VersionFlag: "--version",
			Required:    false,
			ParseVersion: func(raw string) string {
				return toolcheck.FirstLine(raw)
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
				for line := range strings.SplitSeq(raw, "\n") {
					if val, ok := strings.CutPrefix(strings.TrimSpace(line), "version:"); ok {
						return strings.TrimSpace(val)
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
				return strings.TrimPrefix(toolcheck.FirstLine(raw), "v")
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
				parts := strings.Fields(toolcheck.FirstLine(raw))
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
				raw = toolcheck.FirstLine(raw)
				if val, ok := strings.CutPrefix(raw, "jq-"); ok {
					return val
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
				parts := strings.Fields(toolcheck.FirstLine(raw))
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
		{
			Name:        "syft",
			Binary:      "syft",
			VersionFlag: "version",
			ParseVersion: func(raw string) string {
				// "syft 1.4.1", or multi-line "Application: syft\nVersion: 1.4.1"
				if v := labeledField(raw, "Version:"); v != "" {
					return v
				}
				return extractLastField(raw, "syft ")
			},
			AutoInstall: alwaysInstallable,
		},
		{
			Name:        "grype",
			Binary:      "grype",
			VersionFlag: "version",
			ParseVersion: func(raw string) string {
				if v := labeledField(raw, "Version:"); v != "" {
					return v
				}
				return extractLastField(raw, "grype ")
			},
			AutoInstall: alwaysInstallable,
		},
		{
			Name:        "aws-cli",
			Binary:      "aws",
			VersionFlag: "--version",
			Required:    false,
			ParseVersion: func(raw string) string {
				// "aws-cli/2.15.0 Python/3.11.6 ..." → "2.15.0"
				for _, part := range strings.Fields(toolcheck.FirstLine(raw)) {
					if v, ok := strings.CutPrefix(part, "aws-cli/"); ok {
						return v
					}
				}
				return ""
			},
			AutoInstall: alwaysInstallable,
		},
		{
			Name:        "gcloud",
			Binary:      "gcloud",
			VersionFlag: "version",
			Required:    false,
			ParseVersion: func(raw string) string {
				// "Google Cloud SDK 462.0.1\n..." → "462.0.1"
				for line := range strings.SplitSeq(raw, "\n") {
					if val, ok := strings.CutPrefix(strings.TrimSpace(line), "Google Cloud SDK "); ok {
						return strings.TrimSpace(val)
					}
				}
				return ""
			},
			AutoInstall: alwaysInstallable,
		},
		{
			Name:        "az",
			Binary:      "az",
			VersionFlag: "version",
			Required:    false,
			ParseVersion: func(raw string) string {
				// `az version` prints JSON: {"azure-cli": "2.58.0", ...}.
				var out map[string]any
				if err := json.Unmarshal([]byte(raw), &out); err == nil {
					if v, ok := out["azure-cli"].(string); ok {
						return v
					}
					return ""
				}
				// Older CLIs print "azure-cli   2.58.0 *" lines.
				return labeledField(raw, "azure-cli")
			},
			AutoInstall: alwaysInstallable,
		},
	}
}

// RequiredChecks returns the Required subset of DefaultChecks, in order.
func RequiredChecks() []ToolCheck {
	var required []ToolCheck
	for _, tc := range DefaultChecks() {
		if tc.Required {
			required = append(required, tc)
		}
	}
	return required
}

// alwaysInstallable returns true for any OS.
func alwaysInstallable(_ *sysinfo.OSInfo) bool {
	return true
}

// extractLastField extracts the version after a known prefix,
// returning "" on empty/unexpected input.
func extractLastField(raw, prefix string) string {
	raw = toolcheck.FirstLine(raw)
	if raw == "" {
		return ""
	}
	if val, ok := strings.CutPrefix(raw, prefix); ok {
		return strings.TrimSpace(val)
	}
	// Fallback: take the last field
	parts := strings.Fields(raw)
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

// labeledField returns the first whitespace-separated field following label
// on the first line that starts with label, or "".
func labeledField(raw, label string) string {
	for line := range strings.SplitSeq(raw, "\n") {
		if val, ok := strings.CutPrefix(strings.TrimSpace(line), label); ok {
			if fields := strings.Fields(val); len(fields) > 0 {
				return fields[0]
			}
		}
	}
	return ""
}
