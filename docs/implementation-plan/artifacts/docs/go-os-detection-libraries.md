<!-- Sources: https://github.com/wille/osutil https://github.com/makifdb/packer https://github.com/jpikl/pm https://github.com/Hayao0819/go-distro https://pkg.go.dev/github.com/quay/claircore/osrelease -->
<!-- Retrieved: 2026-05-12 -->

# Go Libraries for OS Detection and Package Manager Abstraction

## OS/Distro Detection

### wille/osutil
- Detects OS name, version, and display string
- Supports: macOS, Windows, Linux, FreeBSD, OpenBSD, NetBSD, DragonFly BSD, Solaris
- API: `osutil.Name`, `osutil.GetVersion()`, `osutil.GetDisplay()`
- MIT license, 100% Go

### Hayao0819/go-distro
- Detects Linux distribution, version, codename
- Uses os-release ID and package managers (pacman, dpkg, dnf/rpm, zypper)
- Go library

### quay/claircore/osrelease
- Parses /etc/os-release and /usr/lib/os-release
- Returns key-value pairs
- Part of the Clair container security scanner

### dekobon/distro-detect
- Detects Linux distro by analyzing file contents
- No external program calls

### zcalusic/sysinfo
- Comprehensive Linux system info (OS, kernel, hardware)
- No external dependencies

## Package Manager Abstraction

### makifdb/packer
- Simple system package management for Go
- API: Check(), Install(), Remove(), Update(), DetectManager(), Command()
- Supports: apt (Ubuntu/Debian), dnf (Fedora/RHEL), pacman (Arch/Manjaro), zypper (OpenSUSE), apk (Alpine), brew (macOS), choco (Windows)
- MIT license, `go get github.com/makifdb/packer`

### jpikl/pm (shell script, not Go)
- Wraps: pacman, paru, yay, apt, dnf, zypper, apk, brew, scoop
- Auto-detects by checking binary availability in order
- Override with PM env var
- Unified commands: install, remove, upgrade, fetch, info, list, search

## Standard Library

### runtime.GOOS / runtime.GOARCH
- Built-in OS/arch detection at runtime
- GOOS: darwin, linux, windows, freebsd, etc.
- GOARCH: amd64, arm64, 386, arm, etc.

### /etc/os-release parsing (manual)
- Standard file on all modern Linux distros
- Key fields: ID, ID_LIKE, VERSION_ID, PRETTY_NAME
- Fallback: /usr/lib/os-release
- Simple key=value format, trivial to parse
