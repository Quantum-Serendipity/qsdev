//go:build linux

package sysinfo

import (
	"os"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/fileutil"
)

// detectPlatform populates Linux-specific fields in info by reading
// os-release, /proc metadata, and checking for containers and WSL.
func detectPlatform(info *OSInfo) {
	rel := parseOSRelease("/etc/os-release")
	if len(rel) == 0 {
		rel = parseOSRelease("/usr/lib/os-release")
	}

	id := rel["ID"]
	idLike := rel["ID_LIKE"]

	info.Distro = id
	info.DistroLike = idLike
	info.Version = rel["VERSION_ID"]
	info.VersionCode = rel["VERSION_CODENAME"]
	info.PrettyName = rel["PRETTY_NAME"]
	info.Family = determineFamily(id, idLike)

	// Kernel version from procfs (faster than exec uname).
	info.Kernel = fileutil.ReadFirstLine("/proc/sys/kernel/osrelease")

	detectWSL(info)
	detectContainer(info)
	detectSELinux(info)
}

// detectWSL checks whether the system is running under Windows Subsystem for Linux.
func detectWSL(info *OSInfo) {
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return
	}
	lower := strings.ToLower(string(data))
	switch {
	case strings.Contains(lower, "microsoft-standard"):
		info.IsWSL = true
		info.IsWSL2 = true
	case strings.Contains(lower, "microsoft"):
		info.IsWSL = true
	}
	if info.IsWSL {
		info.WSLDistro = os.Getenv("WSL_DISTRO_NAME")
	}
}

// detectContainer checks whether the process is running inside a container.
func detectContainer(info *OSInfo) {
	if fileutil.FileExists("/.dockerenv") {
		info.IsContainer = true
		return
	}
	if fileutil.FileExists("/run/.containerenv") {
		info.IsContainer = true
		return
	}
	if os.Getenv("container") != "" {
		info.IsContainer = true
		return
	}
	data, err := os.ReadFile("/proc/1/cgroup")
	if err != nil {
		return
	}
	lower := strings.ToLower(string(data))
	if strings.Contains(lower, "docker") || strings.Contains(lower, "containerd") {
		info.IsContainer = true
	}
}

// detectSELinux checks whether SELinux is enforcing. It uses the sysfs
// interface rather than executing commands for speed.
func detectSELinux(info *OSInfo) {
	data, err := os.ReadFile("/sys/fs/selinux/enforce")
	if err != nil {
		return
	}
	if strings.TrimSpace(string(data)) == "1" {
		info.IsSELinux = true
	}
}
