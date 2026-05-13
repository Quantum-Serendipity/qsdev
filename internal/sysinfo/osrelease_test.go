package sysinfo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseOSRelease(t *testing.T) {
	type fixture struct {
		name    string
		content string
		want    map[string]string
	}

	fixtures := []fixture{
		{
			name: "Ubuntu 24.04",
			content: `PRETTY_NAME="Ubuntu 24.04 LTS"
NAME="Ubuntu"
VERSION_ID="24.04"
VERSION="24.04 LTS (Noble Numbat)"
VERSION_CODENAME=noble
ID=ubuntu
ID_LIKE=debian
HOME_URL="https://www.ubuntu.com/"
`,
			want: map[string]string{
				"PRETTY_NAME":      "Ubuntu 24.04 LTS",
				"NAME":             "Ubuntu",
				"VERSION_ID":       "24.04",
				"VERSION":          "24.04 LTS (Noble Numbat)",
				"VERSION_CODENAME": "noble",
				"ID":               "ubuntu",
				"ID_LIKE":          "debian",
				"HOME_URL":         "https://www.ubuntu.com/",
			},
		},
		{
			name: "Debian 12",
			content: `PRETTY_NAME="Debian GNU/Linux 12 (bookworm)"
NAME="Debian GNU/Linux"
VERSION_ID="12"
VERSION="12 (bookworm)"
VERSION_CODENAME=bookworm
ID=debian
HOME_URL="https://www.debian.org/"
`,
			want: map[string]string{
				"PRETTY_NAME":      "Debian GNU/Linux 12 (bookworm)",
				"NAME":             "Debian GNU/Linux",
				"VERSION_ID":       "12",
				"VERSION":          "12 (bookworm)",
				"VERSION_CODENAME": "bookworm",
				"ID":               "debian",
				"HOME_URL":         "https://www.debian.org/",
			},
		},
		{
			name: "Fedora 41",
			content: `NAME="Fedora Linux"
VERSION="41 (Workstation Edition)"
ID=fedora
VERSION_ID=41
PRETTY_NAME="Fedora Linux 41 (Workstation Edition)"
`,
			want: map[string]string{
				"NAME":        "Fedora Linux",
				"VERSION":     "41 (Workstation Edition)",
				"ID":          "fedora",
				"VERSION_ID":  "41",
				"PRETTY_NAME": "Fedora Linux 41 (Workstation Edition)",
			},
		},
		{
			name: "RHEL 9.4",
			content: `NAME="Red Hat Enterprise Linux"
VERSION="9.4 (Plow)"
ID="rhel"
ID_LIKE="fedora"
VERSION_ID="9.4"
PRETTY_NAME="Red Hat Enterprise Linux 9.4 (Plow)"
`,
			want: map[string]string{
				"NAME":        "Red Hat Enterprise Linux",
				"VERSION":     "9.4 (Plow)",
				"ID":          "rhel",
				"ID_LIKE":     "fedora",
				"VERSION_ID":  "9.4",
				"PRETTY_NAME": "Red Hat Enterprise Linux 9.4 (Plow)",
			},
		},
		{
			name: "Rocky 9.4",
			content: `NAME="Rocky Linux"
VERSION="9.4 (Blue Onyx)"
ID="rocky"
ID_LIKE="rhel centos fedora"
VERSION_ID="9.4"
PRETTY_NAME="Rocky Linux 9.4 (Blue Onyx)"
`,
			want: map[string]string{
				"NAME":        "Rocky Linux",
				"VERSION":     "9.4 (Blue Onyx)",
				"ID":          "rocky",
				"ID_LIKE":     "rhel centos fedora",
				"VERSION_ID":  "9.4",
				"PRETTY_NAME": "Rocky Linux 9.4 (Blue Onyx)",
			},
		},
		{
			name: "Arch",
			content: `NAME="Arch Linux"
PRETTY_NAME="Arch Linux"
ID=arch
BUILD_ID=rolling
`,
			want: map[string]string{
				"NAME":        "Arch Linux",
				"PRETTY_NAME": "Arch Linux",
				"ID":          "arch",
				"BUILD_ID":    "rolling",
			},
		},
		{
			name: "Manjaro",
			content: `NAME="Manjaro Linux"
ID=manjaro
ID_LIKE=arch
PRETTY_NAME="Manjaro Linux"
`,
			want: map[string]string{
				"NAME":        "Manjaro Linux",
				"ID":          "manjaro",
				"ID_LIKE":     "arch",
				"PRETTY_NAME": "Manjaro Linux",
			},
		},
		{
			name: "NixOS",
			content: `NAME=NixOS
ID=nixos
VERSION="24.11 (Vicuna)"
VERSION_CODENAME=vicuna
VERSION_ID="24.11"
PRETTY_NAME="NixOS 24.11 (Vicuna)"
`,
			want: map[string]string{
				"NAME":             "NixOS",
				"ID":               "nixos",
				"VERSION":          "24.11 (Vicuna)",
				"VERSION_CODENAME": "vicuna",
				"VERSION_ID":       "24.11",
				"PRETTY_NAME":      "NixOS 24.11 (Vicuna)",
			},
		},
		{
			name: "Alpine 3.20",
			content: `NAME="Alpine Linux"
ID=alpine
VERSION_ID=3.20.0
PRETTY_NAME="Alpine Linux v3.20"
`,
			want: map[string]string{
				"NAME":        "Alpine Linux",
				"ID":          "alpine",
				"VERSION_ID":  "3.20.0",
				"PRETTY_NAME": "Alpine Linux v3.20",
			},
		},
		{
			name: "Void",
			content: `NAME="Void"
ID=void
PRETTY_NAME="Void Linux"
`,
			want: map[string]string{
				"NAME":        "Void",
				"ID":          "void",
				"PRETTY_NAME": "Void Linux",
			},
		},
		{
			name: "Gentoo",
			content: `NAME=Gentoo
ID=gentoo
PRETTY_NAME="Gentoo/Linux"
`,
			want: map[string]string{
				"NAME":        "Gentoo",
				"ID":          "gentoo",
				"PRETTY_NAME": "Gentoo/Linux",
			},
		},
		{
			name: "openSUSE Tumbleweed",
			content: `NAME="openSUSE Tumbleweed"
ID="opensuse-tumbleweed"
ID_LIKE="opensuse suse"
PRETTY_NAME="openSUSE Tumbleweed"
`,
			want: map[string]string{
				"NAME":        "openSUSE Tumbleweed",
				"ID":          "opensuse-tumbleweed",
				"ID_LIKE":     "opensuse suse",
				"PRETTY_NAME": "openSUSE Tumbleweed",
			},
		},
		{
			name: "Pop!_OS",
			content: `NAME="Pop!_OS"
ID=pop
ID_LIKE="ubuntu debian"
VERSION_ID="22.04"
PRETTY_NAME="Pop!_OS 22.04 LTS"
`,
			want: map[string]string{
				"NAME":        "Pop!_OS",
				"ID":          "pop",
				"ID_LIKE":     "ubuntu debian",
				"VERSION_ID":  "22.04",
				"PRETTY_NAME": "Pop!_OS 22.04 LTS",
			},
		},
		{
			name: "Linux Mint",
			content: `NAME="Linux Mint"
ID=linuxmint
ID_LIKE="ubuntu debian"
VERSION_ID="21.3"
PRETTY_NAME="Linux Mint 21.3"
`,
			want: map[string]string{
				"NAME":        "Linux Mint",
				"ID":          "linuxmint",
				"ID_LIKE":     "ubuntu debian",
				"VERSION_ID":  "21.3",
				"PRETTY_NAME": "Linux Mint 21.3",
			},
		},
		{
			name: "EndeavourOS",
			content: `NAME="EndeavourOS"
ID=endeavouros
ID_LIKE=arch
PRETTY_NAME="EndeavourOS Linux"
`,
			want: map[string]string{
				"NAME":        "EndeavourOS",
				"ID":          "endeavouros",
				"ID_LIKE":     "arch",
				"PRETTY_NAME": "EndeavourOS Linux",
			},
		},
		{
			name: "comments and empty lines",
			content: `# This is a comment
NAME="Test OS"

# Another comment
ID=test

VERSION_ID="1.0"
`,
			want: map[string]string{
				"NAME":       "Test OS",
				"ID":         "test",
				"VERSION_ID": "1.0",
			},
		},
		{
			name: "single-quoted values",
			content: `NAME='Single Quoted OS'
ID='singlequote'
`,
			want: map[string]string{
				"NAME": "Single Quoted OS",
				"ID":   "singlequote",
			},
		},
		{
			name: "unquoted values",
			content: `NAME=UnquotedOS
ID=unquoted
VERSION_ID=1.0
`,
			want: map[string]string{
				"NAME":       "UnquotedOS",
				"ID":         "unquoted",
				"VERSION_ID": "1.0",
			},
		},
	}

	for _, tc := range fixtures {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "os-release")
			if err := os.WriteFile(path, []byte(tc.content), 0o644); err != nil {
				t.Fatalf("failed to write fixture: %v", err)
			}

			got := parseOSRelease(path)

			if len(got) != len(tc.want) {
				t.Errorf("key count: got %d, want %d\ngot:  %v\nwant: %v", len(got), len(tc.want), got, tc.want)
			}
			for k, wantV := range tc.want {
				if gotV, ok := got[k]; !ok {
					t.Errorf("missing key %q", k)
				} else if gotV != wantV {
					t.Errorf("key %q: got %q, want %q", k, gotV, wantV)
				}
			}
		})
	}

	t.Run("nonexistent file returns empty map", func(t *testing.T) {
		got := parseOSRelease("/nonexistent/path/os-release")
		if len(got) != 0 {
			t.Errorf("expected empty map, got %v", got)
		}
	})
}

func TestDetermineFamily(t *testing.T) {
	tests := []struct {
		name   string
		id     string
		idLike string
		want   string
	}{
		// Direct ID matches.
		{name: "ubuntu", id: "ubuntu", idLike: "", want: "debian"},
		{name: "debian", id: "debian", idLike: "", want: "debian"},
		{name: "fedora", id: "fedora", idLike: "", want: "rhel"},
		{name: "arch", id: "arch", idLike: "", want: "arch"},
		{name: "manjaro direct", id: "manjaro", idLike: "arch", want: "arch"},
		{name: "endeavouros direct", id: "endeavouros", idLike: "arch", want: "arch"},
		{name: "garuda direct", id: "garuda", idLike: "arch", want: "arch"},
		{name: "nixos", id: "nixos", idLike: "", want: "nixos"},
		{name: "alpine", id: "alpine", idLike: "", want: "alpine"},
		{name: "void", id: "void", idLike: "", want: "void"},
		{name: "gentoo", id: "gentoo", idLike: "", want: "gentoo"},
		{name: "opensuse-tumbleweed direct", id: "opensuse-tumbleweed", idLike: "opensuse suse", want: "suse"},
		{name: "opensuse-leap direct", id: "opensuse-leap", idLike: "opensuse suse", want: "suse"},

		// ID_LIKE chain fallbacks.
		{name: "pop via idlike ubuntu debian", id: "pop", idLike: "ubuntu debian", want: "debian"},
		{name: "linuxmint via idlike ubuntu debian", id: "linuxmint", idLike: "ubuntu debian", want: "debian"},
		{name: "rhel via idlike fedora", id: "rhel", idLike: "fedora", want: "rhel"},
		{name: "rocky via idlike rhel centos fedora", id: "rocky", idLike: "rhel centos fedora", want: "rhel"},
		{name: "almalinux via idlike rhel centos fedora", id: "almalinux", idLike: "rhel centos fedora", want: "rhel"},
		{name: "amzn via idlike fedora", id: "amzn", idLike: "fedora", want: "rhel"},
		{name: "sles via idlike suse", id: "sles", idLike: "suse", want: "suse"},

		// Unknown fallback.
		{name: "unknown distro", id: "somecustomos", idLike: "", want: "unknown"},

		// Edge cases.
		{name: "empty id and idlike", id: "", idLike: "", want: "unknown"},
		{name: "unknown id with debian-like", id: "kali", idLike: "debian", want: "debian"},
		{name: "unknown id with arch-like", id: "artix", idLike: "arch", want: "arch"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := determineFamily(tc.id, tc.idLike)
			if got != tc.want {
				t.Errorf("determineFamily(%q, %q) = %q, want %q", tc.id, tc.idLike, got, tc.want)
			}
		})
	}
}
