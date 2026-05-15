package doctor

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/mattn/go-isatty"

	"github.com/Quantum-Serendipity/qsdev/internal/pkgmanager"
	"github.com/Quantum-Serendipity/qsdev/internal/sysinfo"
)

// Report is the top-level output of qsdev doctor.
type Report struct {
	QsdevVersion        string       `json:"qsdev_version"`
	Timestamp          string       `json:"timestamp"`
	System             SystemInfo   `json:"system"`
	Shell              ShellInfo    `json:"shell"`
	PackageMgrs        []PkgMgrInfo `json:"package_managers"`
	RequiredTools      []ToolEntry  `json:"required_tools"`
	OptionalTools      []ToolEntry  `json:"optional_tools"`
	Recommendations    []string     `json:"recommendations,omitempty"`
	AllRequiredPresent bool         `json:"all_required_present"`
}

// SystemInfo captures OS-level details for the report.
type SystemInfo struct {
	OS          string `json:"os"`
	Distro      string `json:"distro,omitempty"`
	Version     string `json:"version,omitempty"`
	PrettyName  string `json:"pretty_name,omitempty"`
	Arch        string `json:"arch"`
	Kernel      string `json:"kernel,omitempty"`
	IsWSL       bool   `json:"is_wsl,omitempty"`
	IsWSL2      bool   `json:"is_wsl2,omitempty"`
	IsContainer bool   `json:"is_container,omitempty"`
}

// ShellInfo captures shell details for the report.
type ShellInfo struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	RCFile string `json:"rc_file,omitempty"`
}

// PkgMgrInfo identifies a package manager and whether it is the primary one.
type PkgMgrInfo struct {
	Name    string `json:"name"`
	Primary bool   `json:"primary"`
}

// ToolEntry is a summarised per-tool result in the report.
type ToolEntry struct {
	Name       string `json:"name"`
	Found      bool   `json:"found"`
	Version    string `json:"version,omitempty"`
	VersionOK  bool   `json:"version_ok"`
	Path       string `json:"path,omitempty"`
	FixCommand string `json:"fix_command,omitempty"`
}

// BuildReport constructs a Report from raw OS info and check results.
func BuildReport(osInfo *sysinfo.OSInfo, checks []ToolStatus, qsdevVersion string) *Report {
	r := &Report{
		QsdevVersion: qsdevVersion,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		System: SystemInfo{
			OS:          capitalOS(osInfo.OS),
			Distro:      osInfo.Distro,
			Version:     osInfo.Version,
			PrettyName:  osInfo.PrettyName,
			Arch:        osInfo.Arch,
			Kernel:      osInfo.Kernel,
			IsWSL:       osInfo.IsWSL,
			IsWSL2:      osInfo.IsWSL2,
			IsContainer: osInfo.IsContainer,
		},
		Shell: ShellInfo{
			Name:   osInfo.Shell,
			Path:   osInfo.ShellPath,
			RCFile: osInfo.ShellRCFile,
		},
		AllRequiredPresent: true,
	}

	// Package managers
	if osInfo.PackageManager != "" {
		r.PackageMgrs = append(r.PackageMgrs, PkgMgrInfo{
			Name:    osInfo.PackageManager,
			Primary: true,
		})
	}
	for _, alt := range osInfo.AltPkgManagers {
		r.PackageMgrs = append(r.PackageMgrs, PkgMgrInfo{
			Name:    alt,
			Primary: false,
		})
	}

	mgr := osInfo.PackageManager
	family := osInfo.Family

	for _, ts := range checks {
		entry := ToolEntry{
			Name:      ts.Name,
			Found:     ts.Installed,
			Version:   ts.Version,
			VersionOK: ts.VersionOK,
			Path:      ts.Path,
		}

		if !ts.Installed {
			entry.FixCommand = pkgmanager.InstallCommand(ts.Name, family, mgr)
		} else if ts.MinVersion != "" && !ts.VersionOK {
			entry.FixCommand = pkgmanager.InstallCommand(ts.Name, family, mgr)
		}

		if ts.Required {
			r.RequiredTools = append(r.RequiredTools, entry)
			if !ts.Installed || (ts.MinVersion != "" && !ts.VersionOK) {
				r.AllRequiredPresent = false
			}
		} else {
			r.OptionalTools = append(r.OptionalTools, entry)
		}
	}

	// Build recommendations
	for _, ts := range checks {
		if !ts.Installed {
			cmd := pkgmanager.InstallCommand(ts.Name, family, mgr)
			r.Recommendations = append(r.Recommendations, fmt.Sprintf("Install %s: %s", ts.Name, cmd))
		} else if ts.MinVersion != "" && !ts.VersionOK {
			cmd := pkgmanager.InstallCommand(ts.Name, family, mgr)
			r.Recommendations = append(r.Recommendations, fmt.Sprintf("Upgrade %s to >= %s: %s", ts.Name, ts.MinVersion, cmd))
		}
	}

	return r
}

// UseColor returns true if color output should be used for the given
// file descriptor. It respects NO_COLOR and TERM=dumb and checks isatty.
func UseColor(fd uintptr) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	return isatty.IsTerminal(fd)
}

// FormatReport writes a human-readable doctor report to w.
func FormatReport(w io.Writer, r *Report, useColor bool) {
	okSym, failSym, warnSym := "[OK]", "[FAIL]", "[WARN]"
	if useColor {
		okSym = "\033[32m✓\033[0m"
		failSym = "\033[31m✗\033[0m"
		warnSym = "\033[33m!\033[0m"
	}

	fmt.Fprintf(w, "qsdev doctor v%s\n", r.QsdevVersion)
	fmt.Fprintln(w, "============================")
	fmt.Fprintln(w)

	// System
	fmt.Fprintln(w, "System")
	osLabel := r.System.OS
	if r.System.PrettyName != "" {
		osLabel = fmt.Sprintf("%s (%s)", r.System.OS, r.System.PrettyName)
	}
	fmt.Fprintf(w, "  %-14s %s\n", "OS:", osLabel)
	fmt.Fprintf(w, "  %-14s %s\n", "Architecture:", r.System.Arch)
	if r.System.Kernel != "" {
		fmt.Fprintf(w, "  %-14s %s\n", "Kernel:", r.System.Kernel)
	}
	if r.System.IsWSL {
		wslVer := "1"
		if r.System.IsWSL2 {
			wslVer = "2"
		}
		fmt.Fprintf(w, "  %-14s WSL%s\n", "WSL:", wslVer)
	}
	if r.System.IsContainer {
		fmt.Fprintf(w, "  %-14s yes\n", "Container:")
	}
	fmt.Fprintln(w)

	// Shell
	fmt.Fprintln(w, "Shell")
	fmt.Fprintf(w, "  %-14s %s\n", "Shell:", r.Shell.Name)
	if r.Shell.RCFile != "" {
		fmt.Fprintf(w, "  %-14s %s\n", "RC File:", r.Shell.RCFile)
	}
	fmt.Fprintln(w)

	// Package Managers
	fmt.Fprintln(w, "Package Managers")
	for _, pm := range r.PackageMgrs {
		label := pm.Name
		if pm.Primary {
			label = pm.Name
			fmt.Fprintf(w, "  %-14s %s\n", "Primary:", label)
		} else {
			fmt.Fprintf(w, "  %-14s %s\n", "Alt:", label)
		}
	}
	fmt.Fprintln(w)

	// Required Tools
	if len(r.RequiredTools) > 0 {
		fmt.Fprintln(w, "Required Tools")
		fmt.Fprintf(w, "  %-14s %-8s %-11s %s\n", "NAME", "STATUS", "VERSION", "PATH")
		for _, t := range r.RequiredTools {
			sym := okSym
			if !t.Found {
				sym = failSym
			} else if !t.VersionOK {
				sym = warnSym
			}
			ver := t.Version
			if ver == "" {
				ver = "-"
			}
			p := t.Path
			if p == "" {
				p = "-"
			}
			fmt.Fprintf(w, "  %-14s %-8s %-11s %s\n", t.Name, sym, ver, p)
		}
		fmt.Fprintln(w)
	}

	// Optional Tools
	if len(r.OptionalTools) > 0 {
		fmt.Fprintln(w, "Optional Tools")
		fmt.Fprintf(w, "  %-14s %-8s %-11s %s\n", "NAME", "STATUS", "VERSION", "PATH")
		for _, t := range r.OptionalTools {
			sym := okSym
			if !t.Found {
				sym = warnSym
			} else if !t.VersionOK {
				sym = warnSym
			}
			ver := t.Version
			if ver == "" {
				ver = "-"
			}
			p := t.Path
			if p == "" {
				p = "-"
			}
			fmt.Fprintf(w, "  %-14s %-8s %-11s %s\n", t.Name, sym, ver, p)
		}
		fmt.Fprintln(w)
	}

	// Recommendations
	if len(r.Recommendations) > 0 {
		fmt.Fprintln(w, "Recommendations")
		for i, rec := range r.Recommendations {
			fmt.Fprintf(w, "  %d. %s\n", i+1, rec)
		}
		fmt.Fprintln(w)
	}
}

func capitalOS(os string) string {
	switch strings.ToLower(os) {
	case "linux":
		return "Linux"
	case "darwin":
		return "Darwin"
	case "windows":
		return "Windows"
	default:
		return os
	}
}
