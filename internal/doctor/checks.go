package doctor

import (
	"encoding/json"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/sysinfo"
	"github.com/Quantum-Serendipity/qsdev/internal/toolcheck"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// ToolCheck defines how to detect and classify a single tool.
type ToolCheck struct {
	Name        string
	Binary      string
	AltBinaries []string
	// VersionFlag is the argument that makes the binary print its version.
	// Empty makes the check lookup-only: the binary is looked up on PATH
	// and never run.
	VersionFlag string
	Required    bool
	// RequiredBy says what needs a tool that is not an environment
	// prerequisite (see RequireBinaries); "" for a prerequisite.
	RequiredBy string
	MinVersion string
	// Constraint, when set, is a version constraint in qsdev_version syntax
	// (">= 1.2", "^1.2") the version must satisfy, in place of MinVersion.
	Constraint string
	// PathHint, when set, replaces the install and upgrade advice for a tool
	// no package manager provides: the CLI's own binary, which the hooks find
	// on PATH (see ProjectChecks).
	PathHint string
	// InstallHint tells the user how to install a missing tool.
	InstallHint string
	// ParseVersion extracts the version from the tool's full version output
	// (which may span several lines).
	ParseVersion func(raw string) string
	// ProjectVersion reads the version of a binary found inside the project,
	// which doctor never runs, from metadata beside it; nil or "" leaves the
	// version unknown.
	ProjectVersion func(binPath string) string
	// UpgradeHint, when set, is how to upgrade an installed tool that is
	// below its floor: setup cannot upgrade it in place, so such a tool is
	// not auto-installable.
	UpgradeHint string
	AutoInstall func(osInfo *sysinfo.OSInfo) bool
	Notes       func(osInfo *sysinfo.OSInfo) string
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
			MinVersion:  types.MinNix,
			InstallHint: "Install Nix: https://nixos.org/download.html",
			// The Nix installer installs; it does not upgrade an existing Nix.
			UpgradeHint: "upgrade Nix in place: https://nix.dev/manual/nix/stable/installation/upgrading",
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
			MinVersion:  types.MinDevenv,
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
			// The floor the Python security hooks enforce (D20).
			MinVersion: types.MinHookPython,
			ParseVersion: func(raw string) string {
				// "Python 3.11.7" → "3.11.7"
				return extractLastField(raw, "Python ")
			},
			// An in-project virtualenv interpreter (.venv, devenv's
			// .devenv/state/venv) is not run; its pyvenv.cfg says its version.
			ProjectVersion: pyvenvVersion,
			AutoInstall:    alwaysInstallable,
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

// CheckNamed returns the DefaultChecks entry called name, so other code
// probing the same tool shares its version flag, parser and floor.
func CheckNamed(name string) (ToolCheck, bool) {
	checks := DefaultChecks()
	i := slices.IndexFunc(checks, func(c ToolCheck) bool { return c.Name == name })
	if i < 0 {
		return ToolCheck{}, false
	}
	return checks[i], true
}

// RequireBinaries returns checks with every program in binaries required,
// with requiredBy as the reason. The check whose Binary names the program
// becomes required for that binary alone (no AltBinaries), keeping its floor
// and version probe; a program a check names only as an alternative, or as
// another version of a versioned binary (python or python3.11 for
// python3), gets a required copy of that check for the program itself, so it
// keeps the floor and finding another name would not let the hook run; and a
// program no check names gets a required lookup-only check (see
// ToolCheck.VersionFlag), so it is never run. Names compare as this OS's
// PATH lookup does (see programKey). A check that is already required keeps
// no reason, as it is a prerequisite anyway. checks itself is not modified.
func RequireBinaries(checks []ToolCheck, binaries []string, requiredBy string) []ToolCheck {
	return requireBinaries(checks, binaries, requiredBy, runtime.GOOS, os.Getenv("PATHEXT"))
}

// requireBinaries is RequireBinaries for the given OS and PATHEXT.
func requireBinaries(checks []ToolCheck, binaries []string, requiredBy, goos, pathExt string) []ToolCheck {
	key := func(name string) string { return programKey(name, goos, pathExt) }
	out := slices.Clone(checks)
	for _, b := range binaries {
		k := key(b)
		if i := slices.IndexFunc(out, func(c ToolCheck) bool { return key(c.Binary) == k }); i >= 0 {
			out[i].AltBinaries = nil
			if !out[i].Required {
				out[i].Required, out[i].RequiredBy = true, requiredBy
			}
			continue
		}
		tc := ToolCheck{Name: b}
		if i := slices.IndexFunc(out, func(c ToolCheck) bool {
			return slices.ContainsFunc(c.AltBinaries, func(a string) bool { return key(a) == k }) ||
				otherVersion(k, key(c.Binary))
		}); i >= 0 {
			tc = out[i]
			tc.Name = b
		}
		tc.Binary, tc.AltBinaries = b, nil
		tc.Required, tc.RequiredBy = true, requiredBy
		out = append(out, tc)
	}
	return out
}

// defaultPathExt is the Windows PATHEXT default, used when it is unset.
const defaultPathExt = ".COM;.EXE;.BAT;.CMD"

// programKey is the name a PATH lookup of program resolves by on goos. On
// Windows the lookup ignores case and appends a PATHEXT extension, so
// "Python3.EXE" and "python3" name the same program there.
func programKey(program, goos, pathExt string) string {
	if goos != "windows" {
		return program
	}
	if pathExt == "" {
		pathExt = defaultPathExt
	}
	k := strings.ToLower(program)
	for _, ext := range strings.Split(strings.ToLower(pathExt), ";") {
		if base, ok := strings.CutSuffix(k, ext); ok && ext != "" && base != "" {
			return base
		}
	}
	return k
}

var (
	// versionedBinaryRe splits a binary named with a trailing version
	// ("python3") into its stem ("python").
	versionedBinaryRe = regexp.MustCompile(`^(.*[^0-9.])[0-9][0-9.]*$`)
	// versionTailRe matches a version appended to a stem ("3", "3.11").
	versionTailRe = regexp.MustCompile(`^[0-9][0-9.]*$`)
)

// otherVersion reports whether program names another version of binary, a
// name carrying a trailing version: its stem alone or with any version
// ("python" or "python3.11" for "python3").
func otherVersion(program, binary string) bool {
	m := versionedBinaryRe.FindStringSubmatch(binary)
	if m == nil || program == binary {
		return false
	}
	rest, ok := strings.CutPrefix(program, m[1])
	return ok && (rest == "" || versionTailRe.MatchString(rest))
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
