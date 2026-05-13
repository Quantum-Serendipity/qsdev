package sysinfo

// OSInfo holds a snapshot of the current system's identity, capabilities,
// and environment. Built once at startup by DetectOS.
type OSInfo struct {
	OS   string `json:"os"`   // runtime.GOOS: "linux", "darwin", "windows"
	Arch string `json:"arch"` // runtime.GOARCH: "amd64", "arm64"

	Family      string `json:"family"`       // "debian","rhel","arch","suse","alpine","void","gentoo","nixos","macos","windows","unknown"
	Distro      string `json:"distro"`       // ID from os-release: "ubuntu","fedora","arch", etc.
	DistroLike  string `json:"distro_like"`  // ID_LIKE from os-release
	Version     string `json:"version"`      // VERSION_ID
	VersionCode string `json:"version_code"` // VERSION_CODENAME
	PrettyName  string `json:"pretty_name"`  // PRETTY_NAME
	Kernel      string `json:"kernel"`       // uname -r or equivalent

	IsWSL       bool   `json:"is_wsl"`
	IsWSL2      bool   `json:"is_wsl2"`
	WSLDistro   string `json:"wsl_distro,omitempty"`
	IsContainer bool   `json:"is_container"`
	IsRosetta   bool   `json:"is_rosetta"`
	IsSELinux   bool   `json:"is_selinux"`

	Shell       string `json:"shell"`
	ShellPath   string `json:"shell_path"`
	ShellRCFile string `json:"shell_rc_file"`

	PackageManager string   `json:"package_manager"`
	AltPkgManagers []string `json:"alt_pkg_managers,omitempty"`

	HasNix         bool   `json:"has_nix"`
	HasHomebrew    bool   `json:"has_homebrew"`
	HomebrewPrefix string `json:"homebrew_prefix,omitempty"`

	XcodeCLT        bool `json:"xcode_clt"`
	WindowsTerminal bool `json:"windows_terminal"`
	GitBash         bool `json:"git_bash"`
}
