# Cross-Platform End-to-End Validation Testing Infrastructure for gdev

Research date: 2026-05-12
Context: gdev is a single static Go binary (CGO_ENABLED=0) CLI tool that bootstraps developer environments. It detects OS/distro/package managers/shells and installs prerequisites (git, go, node, nix, direnv, pre-commit). Testing must cover Windows, macOS (Intel + Apple Silicon), and 12+ Linux distro families.

---

## Table of Contents

1. [VM/Container Automation Options](#1-vmcontainer-automation-options)
2. [GitHub Actions CI Matrix Strategy](#2-github-actions-ci-matrix-strategy)
3. [Container-Based Testing](#3-container-based-testing)
4. [Vagrant Box Availability](#4-vagrant-box-availability)
5. [Snapshot/Checkpoint Strategy](#5-snapshotcheckpoint-strategy)
6. [Network Isolation](#6-network-isolation)
7. [macOS Testing Challenges](#7-macos-testing-challenges)
8. [Windows Testing Challenges](#8-windows-testing-challenges)
9. [Test Execution Framework](#9-test-execution-framework)
10. [Parallel Execution](#10-parallel-execution)
11. [Recommended Testing Architecture](#11-recommended-testing-architecture)

---

## 1. VM/Container Automation Options

### 1.1 Comparison Matrix

| Platform | OS Coverage | Cold Start | Pkg Mgr Install | Shell RC | Cost | Maintenance |
|----------|------------|------------|-----------------|----------|------|-------------|
| **GitHub Actions runners** | Ubuntu, Windows, macOS | 10-30s | Yes (native OS) | Yes | Free (public) / metered | Zero |
| **Docker/Podman containers** | 15+ Linux distros | 1-5s | Yes (most) | Partial | Minimal | Low |
| **Incus/LXD system containers** | 20+ Linux distros | 2-10s | Yes (full) | Yes | Self-hosted | Medium |
| **Incus/LXD VMs** | 10+ distros + BSD | 15-45s | Yes (full) | Yes | Self-hosted | Medium |
| **Vagrant (VirtualBox)** | Broad but dated | 60-180s | Yes (full) | Yes | Free | High |
| **Vagrant (libvirt/KVM)** | Broad but dated | 30-90s | Yes (full) | Yes | Linux only | High |
| **QEMU/KVM direct** | Any bootable OS | 15-60s | Yes (full) | Yes | Self-hosted | High |
| **Tart (macOS)** | macOS + Linux on Apple Silicon | 10-30s | Yes | Yes | Apple HW only | Medium |
| **Anka** | macOS (Intel + Apple Silicon) | 10-30s | Yes | Yes | Licensed | Low |

### 1.2 GitHub Actions Runners (Recommended Primary)

**Coverage:**
- Linux: `ubuntu-latest` (24.04), `ubuntu-22.04`, `ubuntu-24.04-arm` -- Ubuntu only natively, but can run ANY Linux distro via `container:` directive or Docker
- macOS: `macos-latest` (15, ARM64), `macos-15-intel`, `macos-14` (ARM64), `macos-26` (ARM64), `macos-26-intel`
- Windows: `windows-latest` (2025), `windows-2022`, `windows-11-arm`

**Strengths:** Zero infrastructure to manage, free for public repos, matrix strategies for parallelism, pre-installed software (git, Go, Node, Docker, Homebrew on macOS, Chocolatey on Windows).

**Limitations:** Only Ubuntu for Linux (not Fedora, Arch, etc. natively). macOS has 5 concurrent job limit on free plan. No NixOS, Arch, Fedora, etc. as native runners. 6-hour job timeout. 256 max matrix jobs per workflow.

### 1.3 Docker/Podman Containers (Recommended for Linux Distros)

**Coverage:** Excellent. Official Docker Hub images exist for:
- `debian:12` (bookworm), `debian:11` (bullseye)
- `ubuntu:22.04`, `ubuntu:24.04`
- `fedora:40`, `fedora:41`, `fedora:42`
- `archlinux:latest`, `archlinux:base-devel`
- `alpine:3.20`, `alpine:3.21`
- `opensuse/tumbleweed`, `opensuse/leap:15.6`
- `rockylinux:9`, `almalinux:9`
- `gentoo/stage3`
- `voidlinux/voidlinux`
- `nixos/nix` (Nix package manager on a base image)

**Not directly available as official Docker images:** NixOS (full distro), Pop!_OS, Linux Mint, Garuda, Manjaro, EndeavourOS. However, community images exist for most (e.g., `linuxmintd/mint21.3-cinnamon`, `manjarolinux/base`).

**Strengths:** 1-5 second startup, excellent caching via layers, disposable, easy to integrate with GitHub Actions `container:` jobs.

**Limitations:** Not a full OS -- no systemd by default, no real init, no kernel. See Section 3 for detailed breakdown.

### 1.4 Incus/LXD System Containers (Recommended for Full-OS Linux Testing)

**Coverage:** The images.linuxcontainers.org server provides both container and VM images:

Container + VM images available: AlmaLinux (8/9/10), Alpine (3.21-3.23), Debian (bookworm/trixie), Fedora (42-44), Gentoo, NixOS (25.11/unstable), openSUSE (15.6/tumbleweed), Rocky (8/9/10), Ubuntu (jammy/noble).

Container-only: Arch Linux, Void Linux, Mint, Kali, Oracle Linux, CentOS, Devuan, Slackware.

**Strengths:** System containers behave like lightweight VMs -- they run a full init system (including systemd), have their own process tree, support full package management, and can modify shell RC files. Snapshots are instant. Much faster than VMs. Can run different distros sharing the host kernel.

**Limitations:** Linux-only (shares host kernel). Cannot test macOS or Windows. Cannot test different kernel versions. Requires a Linux host with Incus installed. Not directly usable in GitHub Actions standard runners (need self-hosted or nested virtualization).

### 1.5 Vagrant

**Coverage:** The `generic/*` boxes from Roboxes (Lavabit) provide broad distro coverage but with significant caveats:

Available: Alpine (3.5-3.17), Arch, AlmaLinux (8/9), CentOS (6-9), Debian (8-11), Fedora (25-37), RHEL (6-8), Rocky (8/9), Ubuntu (16.04-23.04), openSUSE (15/42).

**Critical gaps:** No ARM64 boxes at all (Roboxes explicitly lacks ARM hardware). Versions are significantly outdated (Fedora 37, Ubuntu 23.04, Alpine 3.17, Debian 11). No Void, Gentoo, NixOS, Manjaro, EndeavourOS, Pop!_OS, Mint, or Garuda.

For Apple Silicon: The `gyptazy/vagrant-arm64-boxes` project provides ARM64 boxes for VMware Fusion, but coverage is very limited (FreeBSD 14, a few Linux distros).

**Providers:** VirtualBox (x86 only until 7.1+, now also Apple Silicon), libvirt/KVM (Linux only), VMware Fusion (macOS), Parallels (macOS), Hyper-V (Windows).

**Strengths:** Full VM with real kernel, real systemd, real everything. Snapshot/restore supported. Declarative Vagrantfiles.

**Limitations:** Slow cold start (60-180s), heavy resource usage, outdated box versions, poor ARM64 support, fragmented provider ecosystem. Not practical in CI without self-hosted runners.

### 1.6 QEMU/KVM Direct

**Coverage:** Anything that has a bootable ISO or qcow2 image -- complete freedom.

**Strengths:** Full VM fidelity, qcow2 overlay snapshots for instant restore to clean state, supports any OS. Can run in GitHub Actions on ubuntu runners using nested virtualization (KVM is available).

**Limitations:** Requires building/maintaining base images, more complex orchestration, no pre-built ecosystem like Vagrant Cloud.

---

## 2. GitHub Actions CI Matrix Strategy

### 2.1 Available Runner Labels (as of May 2026)

**Linux (free for public repos):**
| Label | OS | Arch | CPUs | RAM |
|-------|-----|------|------|-----|
| `ubuntu-latest` / `ubuntu-24.04` | Ubuntu 24.04 | x64 | 4 (pub) / 2 (priv) | 16GB / 8GB |
| `ubuntu-22.04` | Ubuntu 22.04 | x64 | 4 / 2 | 16GB / 8GB |
| `ubuntu-24.04-arm` | Ubuntu 24.04 | ARM64 | 4 / 2 | 16GB / 8GB |

**macOS (free for public repos, 10x minute multiplier for private):**
| Label | OS | Arch | CPUs | RAM |
|-------|-----|------|------|-----|
| `macos-latest` / `macos-15` | macOS 15 Sequoia | ARM64 (M1) | 3 | 7GB |
| `macos-14` | macOS 14 Sonoma | ARM64 (M1) | 3 | 7GB |
| `macos-26` | macOS 26 | ARM64 | 3 | 7GB |
| `macos-15-intel` | macOS 15 | x64 | 4 | 14GB |
| `macos-26-intel` | macOS 26 | x64 | 4 | 14GB |

Note: `macos-15-intel` is the LAST Intel macOS image. x86_64 macOS support ends August 2027.

**Windows (free for public repos, 2x minute multiplier for private):**
| Label | OS | Arch | CPUs | RAM |
|-------|-----|------|------|-----|
| `windows-latest` / `windows-2025` | Windows Server 2025 | x64 | 4 / 2 | 16GB / 8GB |
| `windows-2022` | Windows Server 2022 | x64 | 4 / 2 | 16GB / 8GB |
| `windows-11-arm` | Windows 11 | ARM64 | 4 / 2 | 16GB / 8GB |

### 2.2 Recommended CI Matrix Design

The matrix uses three tiers based on signal value vs. cost:

**Tier 1 -- Native runners (every PR):** Direct `runs-on` with no containers. Fast feedback on the three core platforms.

```yaml
strategy:
  fail-fast: false
  matrix:
    include:
      # macOS Apple Silicon (primary)
      - runner: macos-15
        os: macos
        arch: arm64
        name: "macOS 15 ARM64"
      # macOS Intel
      - runner: macos-15-intel
        os: macos
        arch: amd64
        name: "macOS 15 Intel"
      # Windows
      - runner: windows-2025
        os: windows
        arch: amd64
        name: "Windows 2025"
      # Ubuntu (baseline Linux)
      - runner: ubuntu-24.04
        os: linux
        arch: amd64
        name: "Ubuntu 24.04"
      # Ubuntu ARM64
      - runner: ubuntu-24.04-arm
        os: linux
        arch: arm64
        name: "Ubuntu 24.04 ARM64"
```

**Tier 2 -- Container matrix (every PR):** Linux distro families tested via `container:` directive on ubuntu runners.

```yaml
jobs:
  linux-distros:
    runs-on: ubuntu-latest
    strategy:
      fail-fast: false
      matrix:
        include:
          # Debian family
          - container: "debian:12"
            name: "Debian 12"
            pkg_mgr: apt
          - container: "ubuntu:22.04"
            name: "Ubuntu 22.04"
            pkg_mgr: apt
          # RHEL family
          - container: "fedora:41"
            name: "Fedora 41"
            pkg_mgr: dnf
          - container: "rockylinux:9"
            name: "Rocky 9"
            pkg_mgr: dnf
          - container: "almalinux:9"
            name: "Alma 9"
            pkg_mgr: dnf
          # Arch family
          - container: "archlinux:latest"
            name: "Arch Linux"
            pkg_mgr: pacman
          # SUSE
          - container: "opensuse/tumbleweed"
            name: "openSUSE Tumbleweed"
            pkg_mgr: zypper
          - container: "opensuse/leap:15.6"
            name: "openSUSE Leap 15.6"
            pkg_mgr: zypper
          # Alpine
          - container: "alpine:3.20"
            name: "Alpine 3.20"
            pkg_mgr: apk
          # Void
          - container: "voidlinux/voidlinux"
            name: "Void Linux"
            pkg_mgr: xbps
          # Gentoo
          - container: "gentoo/stage3"
            name: "Gentoo"
            pkg_mgr: emerge
    container:
      image: ${{ matrix.container }}
    steps:
      - uses: actions/checkout@v4
      - name: Install gdev
        run: |
          # Container-appropriate install test
          ./scripts/install.sh
      - name: Run gdev devenv doctor
        run: gdev devenv doctor
      - name: Run gdev devenv setup (dry-run)
        run: gdev devenv setup --dry-run
```

**Tier 3 -- Extended matrix (nightly/release only):** WSL2, NixOS, derivative distros, edge cases.

```yaml
jobs:
  wsl2-test:
    runs-on: windows-2025
    steps:
      - uses: Vampire/setup-wsl@v4
        with:
          distribution: Ubuntu-24.04
          wsl-version: 2
      - uses: actions/checkout@v4
      - name: Test gdev in WSL2
        shell: wsl-bash {0}
        run: |
          ./scripts/install.sh
          gdev devenv doctor

  nixos-test:
    runs-on: ubuntu-latest
    container:
      image: nixos/nix
    steps:
      - uses: actions/checkout@v4
      - name: Test gdev on NixOS
        run: |
          # NixOS-specific testing
          ./scripts/install.sh
          gdev devenv doctor

  derivative-distros:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        include:
          - container: "linuxmintd/mint22-cinnamon"
            name: "Linux Mint 22"
          # Pop!_OS, Manjaro, EndeavourOS, Garuda
          # Community images -- verify maintenance status before relying on them
    container:
      image: ${{ matrix.container }}
    steps:
      - uses: actions/checkout@v4
      - run: ./scripts/install.sh && gdev devenv doctor
```

### 2.3 Cost Estimate (Private Repository)

Assuming a 15-target matrix run takes ~5 minutes per job:

| Tier | Jobs | Minutes/Job | Runner Cost | Total/Run |
|------|------|-------------|-------------|-----------|
| Tier 1 (native) | 5 | 5 | Mixed | ~$2.10 |
| - 2x macOS | 2 | 5 | $0.062/min | $0.62 |
| - 1x Windows | 1 | 5 | $0.010/min | $0.05 |
| - 2x Ubuntu | 2 | 5 | $0.006/min | $0.06 |
| Tier 2 (containers) | 11 | 5 | $0.006/min | $0.33 |
| Tier 3 (extended) | 5 | 10 | Mixed | ~$0.50 |
| **Total per PR** | **21** | | | **~$2.93** |

At 20 PRs/week: ~$60/week, ~$240/month. Very reasonable.

For public repositories, only macOS larger runners cost money. Standard runners are free and unlimited.

### 2.4 Self-Hosted Runner Needs

**Not needed for core testing.** GitHub-hosted runners cover:
- macOS Intel and ARM64 natively
- Windows natively
- All Linux distros via containers
- WSL2 via setup-wsl action

Self-hosted runners only needed for:
- Incus/LXD full-OS testing (requires Linux host with nested virt)
- Performance-sensitive benchmarking
- Testing on physical hardware (real shell RC file loading on login)

---

## 3. Container-Based Testing

### 3.1 What Works in Containers

| gdev Feature | Container Support | Notes |
|-------------|-------------------|-------|
| Binary installation (copy to /usr/local/bin) | YES | Works identically |
| OS/distro detection (/etc/os-release) | YES | Container has its own /etc/os-release |
| Package manager detection (apt, dnf, pacman, etc.) | YES | Package managers work normally |
| Package installation (apt install git, etc.) | YES | Full package management works |
| PATH manipulation | YES | Works normally |
| Environment variable detection | YES | Works normally |
| `gdev devenv doctor` (system diagnostics) | MOSTLY | Kernel info comes from host |
| `gdev devenv setup --dry-run` | YES | Detection works, actual install depends on feature |
| Shell detection ($SHELL) | PARTIAL | Default shell is `sh`, not user's shell |
| Shell RC file modification (.bashrc/.zshrc) | PARTIAL | Files exist but no interactive login context |
| direnv integration | PARTIAL | Can install, but hook activation needs shell RC |
| Nix installation | YES | Nix installs fine in containers |
| systemd service management | NO | No systemd PID 1 by default |
| Network interface detection | PARTIAL | Container networking differs from host |
| Windows Terminal detection | N/A | Linux containers only |

### 3.2 What Breaks in Containers

**systemd:** Docker containers run with a simple init (or no init). systemd requires either `--privileged` mode with cgroup mounts, or specific capabilities. For gdev testing, systemd is only needed if gdev manages systemd services (unlikely for a dev tool bootstrapper).

Workaround: Run with `--init` flag for basic signal handling. If systemd is truly needed, use:
```dockerfile
FROM fedora:41
RUN dnf install -y systemd && dnf clean all
CMD ["/usr/sbin/init"]
# Run with: docker run --privileged -v /sys/fs/cgroup:/sys/fs/cgroup:rw
```

**Shell RC files:** Containers start with `sh` as default shell. There's no login session, so `.bashrc`/`.zshrc` aren't sourced automatically. gdev can still write to these files, but verifying the changes "take effect" requires explicitly sourcing them.

Workaround: Test RC file modification by writing, then sourcing, then verifying:
```bash
gdev devenv setup  # writes to .bashrc
source ~/.bashrc
# Verify direnv hook is present, PATH changes took effect, etc.
```

**Interactive wizards:** `gdev init` has an interactive wizard. Containers don't have a TTY by default.

Workaround: Use `docker run -it` or test the non-interactive/flag-driven mode.

### 3.3 What MUST Run in Full VMs

| Test Scenario | Why Containers Fail |
|--------------|-------------------|
| Full login shell verification | Containers don't simulate real user login |
| Reboot-and-verify persistence | Containers don't reboot |
| Installer script (curl \| sh) from scratch | Need clean OS with real user account |
| WSL2-specific detection | Need Windows + WSL2 |
| macOS-specific paths (Homebrew, Xcode CLT) | Need real macOS |
| Windows-specific paths (winget, scoop, Terminal) | Need real Windows |
| systemd service integration | Need real systemd |
| Real network stack testing | Container networking is abstracted |

### 3.4 Docker Image Tags for Each Target

```yaml
# Debian family
debian:12          # Debian 12 Bookworm
debian:11          # Debian 11 Bullseye  
ubuntu:24.04       # Ubuntu 24.04 Noble
ubuntu:22.04       # Ubuntu 22.04 Jammy

# RHEL family
fedora:41          # Fedora 41
fedora:40          # Fedora 40
rockylinux:9       # Rocky Linux 9
almalinux:9        # AlmaLinux 9

# Arch family
archlinux:latest   # Arch Linux (rolling)
# Manjaro: manjarolinux/base (community, verify maintenance)
# EndeavourOS: no official Docker image

# SUSE family
opensuse/tumbleweed:latest  # openSUSE Tumbleweed (rolling)
opensuse/leap:15.6          # openSUSE Leap 15.6

# Alpine
alpine:3.20        # Alpine 3.20

# Void
voidlinux/voidlinux:latest  # Void Linux (community maintained)

# Gentoo
gentoo/stage3:latest        # Gentoo stage3 (official)

# NixOS
nixos/nix:latest            # Nix package manager (not full NixOS)
# For full NixOS testing, use Incus VM image or QEMU

# Derivatives (community images -- verify before relying)
# Pop!_OS: no official Docker image
# Linux Mint: linuxmintd/mint22-cinnamon (official Mint team)
# Garuda: no official Docker image
```

---

## 4. Vagrant Box Availability

### 4.1 Coverage Assessment by Target OS

| Target OS | generic/* Box | Current? | ARM64? | Provider | Verdict |
|-----------|--------------|----------|--------|----------|---------|
| Ubuntu 22.04 | `generic/ubuntu2204` | Yes | No | VBox, libvirt, VMware | Use Docker instead |
| Ubuntu 24.04 | Not available | -- | No | -- | Use Docker |
| Debian 12 | `generic/debian12` | Partial | No | VBox, libvirt | Use Docker |
| Fedora 40/41 | `generic/fedora39` max | No | No | VBox, libvirt | Use Docker |
| Rocky 9 | `generic/rocky9` | Yes | No | VBox, libvirt | Use Docker |
| Alma 9 | `generic/alma9` | Yes | No | VBox, libvirt | Use Docker |
| Arch Linux | `generic/arch` | Rolling | No | VBox, libvirt | Use Docker |
| Manjaro | None | -- | -- | -- | Community image or skip |
| EndeavourOS | None | -- | -- | -- | Community image or skip |
| openSUSE TW | `generic/opensuse15` | Old | No | VBox, libvirt | Use Docker |
| openSUSE Leap 15.6 | Partial | Old | No | VBox, libvirt | Use Docker |
| Alpine 3.20 | `generic/alpine317` max | Old | No | VBox, libvirt | Use Docker |
| Void Linux | None | -- | -- | -- | Use Docker |
| Gentoo | None | -- | -- | -- | Use Docker |
| NixOS | None | -- | -- | -- | Use Incus VM |
| Pop!_OS | None | -- | -- | -- | Skip (Ubuntu-based, test Ubuntu) |
| Linux Mint | None | -- | -- | -- | Use Docker or skip |
| Garuda | None | -- | -- | -- | Skip (Arch-based, test Arch) |
| macOS | None (legal) | -- | -- | -- | Use GitHub Actions or Tart |
| Windows | `gusztavvargadr/windows-11` | Partial | No | VBox, Hyper-V | Use GitHub Actions |

### 4.2 ARM64 Vagrant Box Status

**Effectively nonexistent for broad testing.** The Roboxes project (largest source of generic boxes) explicitly states they lack ARM hardware for builds. The `gyptazy/vagrant-arm64-boxes` project covers only a handful of distros for VMware Fusion on Apple Silicon.

**Recommendation:** Do NOT rely on Vagrant for ARM64 testing. Use GitHub Actions `ubuntu-24.04-arm` runner for ARM64 Linux, `macos-latest` for Apple Silicon macOS, and Incus/LXD ARM64 images for self-hosted.

### 4.3 Vagrant Verdict

Vagrant is **not recommended** as the primary testing platform for gdev. The box ecosystem is outdated, ARM64 support is nearly absent, cold start times are slow, and every target OS that matters has a better alternative (Docker containers, GitHub Actions runners, or Incus). Vagrant remains useful only for local developer testing of specific scenarios that need a full VM (e.g., testing the install script on a fresh Fedora desktop).

---

## 5. Snapshot/Checkpoint Strategy

### 5.1 Strategy by Platform

**Docker Layer Caching (containers -- fastest):**
- Build a "pre-gdev" base image per distro with Dockerfile
- Each test run starts from that cached layer
- Restore time: <1 second (layer already cached)
- GitHub Actions caches Docker layers natively
- Example: `docker build --cache-from` or use BuildKit cache

```dockerfile
# Dockerfile.test-fedora
FROM fedora:41
RUN dnf update -y && dnf install -y curl tar gzip bash
# This layer is cached. gdev install runs on top as a disposable layer.
```

**QEMU qcow2 Backing Files (VMs -- fast):**
- Create a "golden" base image per distro (fully installed, updated)
- For each test run, create a COW (copy-on-write) overlay:
  ```bash
  qemu-img create -f qcow2 -b base-fedora41.qcow2 -F qcow2 test-run.qcow2
  ```
- Overlay starts at 0 bytes, only grows as changes are written
- Restore: delete overlay, create new one (~instant)
- Test isolation: each test gets its own overlay

**Incus/LXD Snapshots (system containers -- fast):**
- `incus snapshot create <container> clean-state`
- `incus snapshot restore <container> clean-state`
- Restore time: 1-3 seconds
- ZFS or btrfs backing stores make snapshots instant and space-efficient

**Vagrant Snapshots (VMs -- slow):**
- `vagrant snapshot save clean`
- `vagrant snapshot restore clean`
- Restore time: 15-60 seconds depending on provider
- VirtualBox snapshots are slower than libvirt/KVM

**GitHub Actions (no snapshot -- disposable):**
- Each job starts with a fresh runner image
- No snapshot needed -- runners are inherently disposable
- Use caching (`actions/cache`) for downloaded artifacts (Go modules, etc.)

### 5.2 Recommended Approach

| Environment | Snapshot Strategy | Restore Time |
|-------------|------------------|--------------|
| GitHub Actions (CI) | Disposable runners + Docker layer cache | N/A (always fresh) |
| Docker containers (CI) | Layer caching, rebuild from cached base | <1s |
| Incus containers (self-hosted) | ZFS/btrfs snapshots | 1-3s |
| QEMU VMs (self-hosted) | qcow2 backing file overlays | <1s to create, 15s to boot |
| Vagrant VMs (local dev) | vagrant snapshot | 15-60s |

---

## 6. Network Isolation

### 6.1 The Problem

gdev's install scripts download binaries from GitHub Releases. Tests need to verify this works, but also need reproducibility and speed.

### 6.2 Strategies

**Strategy A: Real Downloads in CI (Recommended for E2E)**

Let install scripts download from real GitHub Releases during CI. This tests the actual user experience.

Pros: Tests real-world behavior, catches CDN/URL changes. 
Cons: Flaky if GitHub is down, slower, uses bandwidth.

Mitigation: Cache downloaded binaries with `actions/cache` keyed on version:
```yaml
- uses: actions/cache@v4
  with:
    path: /tmp/gdev-download-cache
    key: gdev-${{ env.GDEV_VERSION }}-${{ matrix.os }}
```

**Strategy B: Local HTTP Server (Recommended for Unit/Integration)**

Use Go's `httptest.NewServer()` to mock the GitHub Releases API and serve pre-built binaries from the test fixtures directory.

```go
func TestInstallScript(t *testing.T) {
    // Serve test binaries from local filesystem
    srv := httptest.NewServer(http.FileServer(http.Dir("testdata/releases")))
    defer srv.Close()

    // Override the download URL
    t.Setenv("GDEV_DOWNLOAD_URL", srv.URL)

    // Run install script pointing at local server
    cmd := exec.Command("bash", "scripts/install.sh")
    cmd.Env = append(os.Environ(), "GDEV_DOWNLOAD_URL="+srv.URL)
    output, err := cmd.CombinedOutput()
    // assertions...
}
```

**Strategy C: Pre-staged Binary (Fastest for Container Tests)**

Copy the gdev binary into the container at build time or via volume mount, bypassing the download entirely. Test the install script's download logic separately.

```yaml
steps:
  - uses: actions/checkout@v4
  - name: Build gdev
    run: go build -o gdev ./cmd/gdev
  - name: Copy and test
    run: |
      cp gdev /usr/local/bin/gdev
      chmod +x /usr/local/bin/gdev
      gdev devenv doctor
```

### 6.3 Recommended Hybrid Approach

| Test Type | Network Strategy |
|-----------|-----------------|
| Install script E2E (nightly) | Real GitHub downloads |
| Install script unit tests | Go httptest mock server |
| gdev devenv doctor / setup / init | Pre-staged binary (no download) |
| Container matrix tests | Pre-staged binary |
| Release validation | Real GitHub downloads (mandatory) |

---

## 7. macOS Testing Challenges

### 7.1 GitHub Actions macOS Runners

**Available labels (May 2026):**
- `macos-15` / `macos-latest`: ARM64 (M1), 3 CPUs, 7GB RAM -- **primary target**
- `macos-14`: ARM64 (M1), 3 CPUs, 7GB RAM -- older Sonoma
- `macos-26`: ARM64, 3 CPUs, 7GB RAM -- upcoming
- `macos-15-intel`: x64, 4 CPUs, 14GB RAM -- **last Intel macOS runner**
- `macos-26-intel`: x64, 4 CPUs, 14GB RAM

**Pre-installed software on macOS runners:** Homebrew, Xcode (multiple versions), Git, Go, Node.js, Python, Ruby, Swift, various build tools. This is both useful (fast tests) and problematic (doesn't test the "bare Mac" experience).

**Concurrency limit:** 5 concurrent macOS jobs on Free/Pro/Team plans. 50 on Enterprise. This is the binding constraint for macOS testing parallelism.

### 7.2 Testing Apple Silicon-Specific Paths

Use `macos-15` (ARM64/M1) runner. This natively tests:
- `/opt/homebrew` path detection (Apple Silicon Homebrew prefix)
- ARM64 binary selection during install
- Rosetta 2 detection (test by running x86 binary on ARM runner)
- Native ARM64 performance

For Intel-specific paths, use `macos-15-intel`:
- `/usr/local` Homebrew prefix
- x86_64 binary selection

### 7.3 Testing Without Homebrew / Xcode CLT

GitHub runners come with these pre-installed. To test bare-Mac scenarios:

```yaml
- name: Remove Homebrew for bare-Mac test
  run: |
    /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/uninstall.sh)" -- --force
    sudo rm -rf /opt/homebrew /usr/local/Homebrew
- name: Test gdev on bare Mac
  run: |
    ./scripts/install.sh
    gdev devenv doctor  # Should detect missing Homebrew
    gdev devenv setup   # Should offer to install Homebrew
```

### 7.4 macOS VM Alternatives

**Tart (recommended for self-hosted):**
- Apple Silicon only, uses Virtualization.Framework
- Near-native performance
- Actively maintained (v2.32.1, April 2026)
- Can push/pull VM images from OCI registries
- Integrates with Cirrus Runners (GitHub Actions alternative)
- Free and open source

**Anka (enterprise alternative):**
- Same Virtualization.Framework technology as Tart
- Supports both Intel and Apple Silicon
- Commercial license
- Better orchestration and management UI
- Used by large CI/CD operations

**Legal constraint:** macOS VMs can only legally run on Apple hardware. You cannot run macOS VMs in GitHub Actions linux runners, AWS EC2 (non-Mac instances), or any non-Apple hardware. This is an Apple EULA restriction. For self-hosted, you need Mac hardware (Mac Mini, Mac Studio, or Mac Pro).

### 7.5 Recommended macOS Testing Strategy

1. **Every PR:** `macos-15` (ARM64) + `macos-15-intel` (x86) on GitHub Actions
2. **Nightly:** Add `macos-14`, bare-Mac test (remove Homebrew), Xcode CLT removal test
3. **Self-hosted (optional):** Tart VMs on Mac Mini for clean-slate testing

---

## 8. Windows Testing Challenges

### 8.1 GitHub Actions Windows Runners

**Available labels:**
- `windows-2025` / `windows-latest`: x64, 4 CPUs, 16GB RAM
- `windows-2022`: x64 (older)
- `windows-11-arm`: ARM64

**Pre-installed software:** Chocolatey, Git, Go, Node, Python, Visual Studio, Docker (Windows containers), PowerShell 7, Windows SDK.

**Not pre-installed:** winget (available on Server 2025), Scoop.

### 8.2 Testing the PowerShell Install Script

```yaml
- name: Test PowerShell install script
  shell: pwsh
  run: |
    # Test the install script
    & ./scripts/install.ps1

    # Verify installation
    gdev --version
    gdev devenv doctor
```

Test scenarios to cover:
- Fresh install (no prior gdev)
- Upgrade over existing version
- Install with Scoop: `scoop bucket add gdev <url>; scoop install gdev`
- Install with winget (on Server 2025): `winget install gdev`
- Install with Chocolatey: `choco install gdev`

### 8.3 Testing WSL2 Inside GitHub Actions

WSL2 is testable on `windows-2025` runners using the `Vampire/setup-wsl` action:

```yaml
jobs:
  wsl2-test:
    runs-on: windows-2025
    steps:
      - uses: actions/checkout@v4
      - uses: Vampire/setup-wsl@v4
        with:
          distribution: Ubuntu-24.04
          wsl-version: 2
          additional-packages: curl git
      - name: Test gdev in WSL2 Ubuntu
        shell: wsl-bash {0}
        run: |
          # Verify WSL2 detection
          cat /proc/version  # Should contain "microsoft" or "WSL"
          ./scripts/install.sh
          gdev devenv doctor  # Should detect WSL2 environment

  wsl2-fedora:
    runs-on: windows-2025
    steps:
      - uses: Vampire/setup-wsl@v4
        with:
          distribution: Fedora
          wsl-version: 2
      - shell: wsl-bash {0}
        run: ./scripts/install.sh && gdev devenv doctor
```

**Supported WSL distros via setup-wsl:** Ubuntu (16.04-24.04), Debian (11-13), Alpine (3.17-3.23), openSUSE Leap 15.2, Kali Linux, Fedora.

**Limitation:** WSL1 is the default on older runners. `wsl --install` requires `windows-2025` runner. WSLv2 to WSLv1 switching may require reboot (not possible in CI).

### 8.4 Testing Windows Terminal Detection

Windows Terminal (`wt.exe`) is available on the Windows runners. gdev can detect it via:
- `$env:WT_SESSION` environment variable (set when running inside Windows Terminal)
- `Get-Process wt -ErrorAction SilentlyContinue`

### 8.5 Testing Package Manager Installation

```yaml
# Scoop test
- name: Install via Scoop
  shell: pwsh
  run: |
    iex "& {$(irm get.scoop.sh)} -RunAsAdmin"
    scoop bucket add gdev https://github.com/org/scoop-bucket
    scoop install gdev
    gdev --version

# winget test (Server 2025 has winget)
- name: Install via winget
  shell: pwsh
  run: |
    winget install --id OrgName.gdev --accept-source-agreements --accept-package-agreements
    gdev --version

# Chocolatey test (pre-installed on runners)
- name: Install via Chocolatey
  shell: pwsh
  run: |
    choco install gdev -y
    gdev --version
```

### 8.6 Recommended Windows Testing Strategy

1. **Every PR:** `windows-2025` -- PowerShell install script, `gdev devenv doctor`, `gdev devenv setup --dry-run`
2. **Nightly:** WSL2 with Ubuntu + Fedora, Scoop install, Chocolatey install, winget install
3. **Release:** All package manager installs, Windows ARM64 (`windows-11-arm`)

---

## 9. Test Execution Framework

### 9.1 Framework Comparison

| Framework | Language | Strengths | Weaknesses | Best For |
|-----------|----------|-----------|------------|----------|
| **Go test + os/exec** | Go | Native to project, type-safe, go test integration | Verbose for shell scenarios | Unit + integration tests |
| **testscript (Go)** | Go DSL | Coverage integration, env isolation, golden files | Limited to Go ecosystem, no shell freedom | CLI command testing |
| **Bats-core** | Bash | Simple, TAP-compliant, runs any command, setup/teardown | Bash-only, no Windows native | Install script testing |
| **Pester** | PowerShell | Native Windows, rich assertions | PowerShell-only | Windows install script testing |
| **Custom Go harness** | Go | Full control, cross-platform | Build cost | Orchestrating multi-platform |

### 9.2 How mise Tests Cross-Platform

mise (the dev tool manager, similar scope to gdev) uses:
- **Rust test framework** for unit/integration tests
- **GitHub Actions matrix** across macOS + Linux + Windows
- **Pester (.Tests.ps1)** for Windows-specific E2E tests
- **jdx/mise-action** GitHub Action for CI integration
- Generates `.github/workflows/test.yml` via `mise generate github-action`

### 9.3 How rustup Tests Cross-Platform

rustup (the Rust toolchain installer) uses:
- **Rust tests** for core logic
- **CI workflow templates** generated from YAML templates in `ci/actions-templates/`
- **Docker containers** matching `rust-lang/rust` CI images for Linux x86_64
- **Bare VM testing**: starts from minimal VMs, tests `rustup-init.sh` from the branch
- **Native runners** for platforms GitHub supports; cross-compilation only for others
- Tests the actual install script, not just the binary

### 9.4 Recommended Framework for gdev

**Three-layer testing strategy:**

**Layer 1: Go tests (testscript + os/exec)**
For testing gdev's Go code: OS detection, package manager detection, doctor logic, setup planning.

```go
// integration_test.go
func TestDoctorDetectsOS(t *testing.T) {
    cmd := exec.Command("./gdev", "doctor", "--json")
    output, err := cmd.CombinedOutput()
    require.NoError(t, err)

    var result DoctorOutput
    require.NoError(t, json.Unmarshal(output, &result))

    assert.Equal(t, runtime.GOOS, result.OS)
    assert.NotEmpty(t, result.Distro)
    assert.NotEmpty(t, result.PackageManager)
}
```

Using testscript for golden-file testing:
```
# testdata/script/doctor_linux.txt
exec gdev devenv doctor --json
stdout '"os":"linux"'
stdout '"package_manager":'
! stderr .
```

**Layer 2: Bats tests (install script testing)**
For testing the bash install script across Linux/macOS:

```bash
# test/install.bats
setup() {
    load 'test_helper/bats-support/load'
    load 'test_helper/bats-assert/load'
    export GDEV_INSTALL_DIR="$BATS_TEST_TMPDIR/bin"
}

@test "install script detects OS" {
    run bash scripts/install.sh --dry-run
    assert_success
    assert_output --partial "Detected OS:"
}

@test "install script downloads correct binary" {
    run bash scripts/install.sh --dry-run
    assert_success
    if [[ "$(uname -s)" == "Darwin" ]]; then
        assert_output --partial "darwin"
    else
        assert_output --partial "linux"
    fi
}

@test "install script creates binary" {
    run bash scripts/install.sh
    assert_success
    assert [ -x "$GDEV_INSTALL_DIR/gdev" ]
}

@test "installed binary runs" {
    bash scripts/install.sh
    run "$GDEV_INSTALL_DIR/gdev" --version
    assert_success
}
```

**Layer 3: Pester tests (PowerShell install script testing)**
For testing the PowerShell install script on Windows:

```powershell
# test/Install.Tests.ps1
Describe "gdev PowerShell installer" {
    BeforeAll {
        $env:GDEV_INSTALL_DIR = Join-Path $TestDrive "bin"
    }

    It "detects Windows OS" {
        $output = & ./scripts/install.ps1 -DryRun
        $output | Should -Match "Windows"
    }

    It "downloads correct binary" {
        $output = & ./scripts/install.ps1 -DryRun
        $output | Should -Match "windows.*amd64"
    }

    It "installs binary successfully" {
        & ./scripts/install.ps1
        Test-Path "$env:GDEV_INSTALL_DIR\gdev.exe" | Should -BeTrue
    }

    It "installed binary runs" {
        & "$env:GDEV_INSTALL_DIR\gdev.exe" --version
        $LASTEXITCODE | Should -Be 0
    }
}
```

---

## 10. Parallel Execution

### 10.1 GitHub Actions Concurrency Limits

| Plan | Max Concurrent Jobs | Max Concurrent macOS |
|------|-------------------|----------------------|
| Free | 20 | 5 |
| Pro | 40 | 5 |
| Team | 60 | 5 |
| Enterprise | 500 | 50 |

A matrix of 256 jobs is the max per workflow, but practically you'll hit concurrent job limits first.

### 10.2 Fan-Out Strategy

**Recommended: Two-workflow approach**

```
Workflow 1: "Quick Validation" (every PR, ~5 min)
├── Job: Build gdev binary (all platforms)
│   ├── linux/amd64
│   ├── linux/arm64
│   ├── darwin/amd64
│   ├── darwin/arm64
│   └── windows/amd64
├── Job: Go unit tests
├── Job: macOS ARM64 E2E
├── Job: macOS Intel E2E
├── Job: Windows E2E
└── Job: Ubuntu E2E

Workflow 2: "Full Matrix" (nightly + pre-release, ~15 min)
├── Job: Build (same as above)
├── Job: Linux container matrix (11 distros, parallel)
│   ├── debian:12
│   ├── ubuntu:22.04
│   ├── fedora:41
│   ├── rockylinux:9
│   ├── almalinux:9
│   ├── archlinux:latest
│   ├── opensuse/tumbleweed
│   ├── opensuse/leap:15.6
│   ├── alpine:3.20
│   ├── voidlinux/voidlinux
│   └── gentoo/stage3
├── Job: NixOS (container)
├── Job: WSL2 Ubuntu (on windows-2025)
├── Job: WSL2 Fedora (on windows-2025)
├── Job: macOS bare (remove Homebrew)
├── Job: Windows Scoop install
├── Job: Windows Chocolatey install
├── Job: Windows winget install
├── Job: Linux ARM64 E2E
└── Job: Derivative distros (Mint, etc.)
```

### 10.3 Optimization Techniques

**1. Build once, test many:**
Build the gdev binary in a single job, upload as artifact, download in each test job. Avoids recompiling in every matrix cell.

```yaml
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - run: |
          GOOS=linux GOARCH=amd64 go build -o gdev-linux-amd64 ./cmd/gdev
          GOOS=darwin GOARCH=arm64 go build -o gdev-darwin-arm64 ./cmd/gdev
          # etc.
      - uses: actions/upload-artifact@v4
        with:
          name: gdev-binaries
          path: gdev-*

  test-linux:
    needs: build
    runs-on: ubuntu-latest
    strategy:
      matrix:
        container: [debian:12, fedora:41, archlinux:latest, ...]
    container:
      image: ${{ matrix.container }}
    steps:
      - uses: actions/download-artifact@v4
      - run: |
          chmod +x gdev-binaries/gdev-linux-amd64
          cp gdev-binaries/gdev-linux-amd64 /usr/local/bin/gdev
          gdev devenv doctor
```

**2. Use `fail-fast: false`:**
Don't cancel all jobs if one distro fails. Each distro is an independent signal.

**3. Conditional matrix expansion:**
Run the full matrix only on nightly/release, subset on PRs.

```yaml
jobs:
  test:
    strategy:
      matrix:
        container: ${{ fromJSON(
          github.event_name == 'pull_request'
          && '["debian:12", "fedora:41", "archlinux:latest"]'
          || '["debian:12", "debian:11", "ubuntu:24.04", "ubuntu:22.04", "fedora:41", ...]'
        ) }}
```

**4. Cache aggressively:**
```yaml
- uses: actions/cache@v4
  with:
    path: |
      ~/go/pkg/mod
      ~/.cache/go-build
    key: go-${{ hashFiles('go.sum') }}
```

### 10.4 Time Estimate

| Scenario | Jobs | Wall Time (parallel) | Compute Time |
|----------|------|---------------------|--------------|
| Quick Validation (PR) | 6 | ~5 min | ~25 min |
| Full Matrix (nightly) | 21 | ~15 min | ~80 min |
| Release Validation | 25 | ~20 min | ~100 min |

The binding constraint is macOS concurrency (5 jobs max on free/pro). Linux container jobs run fast (<3 min each) and have higher concurrency limits (20+).

---

## 11. Recommended Testing Architecture

### 11.1 Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                     GitHub Actions CI                            │
│                                                                  │
│  ┌──────────────┐  ┌────────────────┐  ┌─────────────────────┐  │
│  │ Layer 1:     │  │ Layer 2:       │  │ Layer 3:            │  │
│  │ Go Tests     │  │ Install Script │  │ Full E2E            │  │
│  │              │  │ Tests          │  │                     │  │
│  │ - testscript │  │ - Bats (bash)  │  │ - Native runners    │  │
│  │ - unit tests │  │ - Pester (PS)  │  │   (macOS, Windows,  │  │
│  │ - httptest   │  │                │  │    Ubuntu)          │  │
│  │   mock server│  │                │  │ - Container matrix  │  │
│  │              │  │                │  │   (11 Linux distros)│  │
│  │              │  │                │  │ - WSL2 on Windows   │  │
│  └──────────────┘  └────────────────┘  └─────────────────────┘  │
│                                                                  │
│  Triggers:                                                       │
│  - PR: Layer 1 + Layer 2 + Layer 3 (core subset)                │
│  - Nightly: Full Layer 3 matrix                                  │
│  - Release: Everything + package manager installs                │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│              Optional: Self-Hosted (Incus/LXD)                   │
│                                                                  │
│  For tests requiring full OS fidelity:                           │
│  - Login shell verification                                      │
│  - systemd integration                                           │
│  - Real user account creation                                    │
│  - NixOS full-distro testing                                     │
│  - Reboot-and-verify                                             │
│                                                                  │
│  Platform: Linux server with Incus                               │
│  Images: linuxcontainers.org (NixOS, Fedora, Debian, etc.)      │
│  Snapshots: ZFS/btrfs for instant restore                        │
└─────────────────────────────────────────────────────────────────┘
```

### 11.2 Technology Choices

| Component | Choice | Rationale |
|-----------|--------|-----------|
| **CI Platform** | GitHub Actions | Zero infrastructure, matrix strategies, free for public repos |
| **Linux distro testing** | Docker containers via `container:` directive | 1-5s startup, broad distro coverage, integrates natively with GHA |
| **macOS testing** | `macos-15` + `macos-15-intel` runners | Native ARM64 + Intel, Homebrew pre-installed |
| **Windows testing** | `windows-2025` runner | Native, Chocolatey pre-installed, WSL2 available |
| **WSL2 testing** | `Vampire/setup-wsl@v4` on `windows-2025` | Supports Ubuntu, Debian, Fedora, Alpine in WSL2 |
| **Install script tests (bash)** | Bats-core | TAP-compliant, simple, mature, runs any command |
| **Install script tests (PS)** | Pester | Native PowerShell testing, rich assertions |
| **Go CLI tests** | testscript + Go test | Coverage integration, env isolation, golden files |
| **Network mocking** | Go httptest | Standard library, fast, reliable |
| **Full-OS testing (optional)** | Incus system containers/VMs | System containers with systemd, instant snapshots |
| **macOS VMs (optional)** | Tart | Open source, Apple Silicon native, OCI registry |
| **Snapshot/restore** | Docker layers (CI), qcow2 overlays (VMs), Incus snapshots | Each optimized for its platform |

### 11.3 Complete OS Coverage Matrix

| Target OS | Test Method | Trigger | Runner |
|-----------|-------------|---------|--------|
| macOS 15 ARM64 | Native runner | Every PR | `macos-15` |
| macOS 15 Intel | Native runner | Every PR | `macos-15-intel` |
| macOS bare (no Homebrew) | Native + uninstall | Nightly | `macos-15` |
| Windows 2025 | Native runner | Every PR | `windows-2025` |
| Windows ARM64 | Native runner | Release | `windows-11-arm` |
| WSL2 Ubuntu | setup-wsl action | Nightly | `windows-2025` |
| WSL2 Fedora | setup-wsl action | Nightly | `windows-2025` |
| Ubuntu 24.04 | Native runner | Every PR | `ubuntu-24.04` |
| Ubuntu 24.04 ARM64 | Native runner | Every PR | `ubuntu-24.04-arm` |
| Ubuntu 22.04 | Container | Every PR | `ubuntu-latest` + `ubuntu:22.04` |
| Debian 12 | Container | Every PR | `ubuntu-latest` + `debian:12` |
| Fedora 41 | Container | Every PR | `ubuntu-latest` + `fedora:41` |
| Rocky 9 | Container | Nightly | `ubuntu-latest` + `rockylinux:9` |
| Alma 9 | Container | Nightly | `ubuntu-latest` + `almalinux:9` |
| Arch Linux | Container | Every PR | `ubuntu-latest` + `archlinux:latest` |
| openSUSE TW | Container | Every PR | `ubuntu-latest` + `opensuse/tumbleweed` |
| openSUSE Leap 15.6 | Container | Nightly | `ubuntu-latest` + `opensuse/leap:15.6` |
| Alpine 3.20 | Container | Every PR | `ubuntu-latest` + `alpine:3.20` |
| Void Linux | Container | Nightly | `ubuntu-latest` + `voidlinux/voidlinux` |
| Gentoo | Container | Nightly | `ubuntu-latest` + `gentoo/stage3` |
| NixOS | Container (nix) | Nightly | `ubuntu-latest` + `nixos/nix` |
| Linux Mint | Container | Release | `ubuntu-latest` + `linuxmintd/mint22-cinnamon` |
| Pop!_OS | Skip (Ubuntu-based) | -- | Test via Ubuntu |
| Manjaro | Skip (Arch-based) | -- | Test via Arch |
| EndeavourOS | Skip (Arch-based) | -- | Test via Arch |
| Garuda | Skip (Arch-based) | -- | Test via Arch |

**Total: 24 test configurations covering all target OS families.**

Derivatives (Pop!_OS, Manjaro, EndeavourOS, Garuda) are safely skipped because gdev's OS detection uses `/etc/os-release` `ID_LIKE` field, which maps these to their parent distro's package manager. If gdev correctly handles Arch, it handles all Arch derivatives. The same logic applies to Ubuntu derivatives. Linux Mint is included because it's the most divergent Ubuntu derivative (different default repos/PPAs).

### 11.4 Implementation Priority

**Phase 1 (Week 1): Core CI pipeline**
- Go unit tests with testscript
- Bats tests for bash install script
- Pester tests for PowerShell install script
- GitHub Actions workflow with Tier 1 native runners (macOS, Windows, Ubuntu)

**Phase 2 (Week 2): Linux distro matrix**
- Docker container matrix for 8 core distros (Debian, Ubuntu, Fedora, Rocky, Arch, openSUSE TW, Alpine, Void)
- Build-once-test-many artifact pattern
- Conditional matrix (PR subset vs nightly full)

**Phase 3 (Week 3): Extended coverage**
- WSL2 testing on Windows runner
- NixOS testing
- Package manager install tests (Scoop, Chocolatey, winget, Homebrew tap, APT .deb, RPM)
- Bare-Mac testing (remove Homebrew)

**Phase 4 (Optional): Full-OS fidelity**
- Incus system container testing for login shell verification
- Tart macOS VMs for clean-slate testing
- Release validation workflow with all 24 targets

### 11.5 Key Design Decisions

1. **Containers over VMs for Linux:** Docker containers cover 90% of gdev's Linux testing needs at 1/100th the startup cost. Only resort to VMs for systemd/login-shell edge cases.

2. **GitHub Actions over self-hosted:** Zero maintenance, predictable cost, sufficient coverage. Self-hosted adds complexity that isn't justified unless you need Incus full-OS testing.

3. **Three test frameworks:** Go tests (core logic), Bats (bash installer), Pester (PowerShell installer). Each framework is native to the code it tests.

4. **Build once, test many:** Compile gdev binaries in a single job, distribute as artifacts. Avoids Go compilation overhead in every container.

5. **Tiered triggers:** PR (fast, core subset), nightly (full matrix), release (everything including package managers). This balances developer velocity against coverage.

6. **Skip derivative distros:** Test parent distros (Ubuntu, Arch, Fedora) and verify `/etc/os-release` `ID_LIKE` handling. Only include the most divergent derivative (Mint).

7. **Real downloads in release validation only:** Use pre-staged binaries for fast PR testing. Real GitHub Release downloads only in nightly/release workflows where flakiness is acceptable for the extra signal.

---

## Sources

- [GitHub Actions Limits](https://docs.github.com/en/actions/reference/limits)
- [GitHub-hosted Runners Reference](https://docs.github.com/en/actions/reference/runners/github-hosted-runners)
- [GitHub Actions Runner Pricing](https://docs.github.com/en/billing/reference/actions-runner-pricing)
- [GitHub Actions 2026 Pricing Changes](https://github.com/resources/insights/2026-pricing-changes-for-github-actions)
- [Running Jobs in a Container](https://docs.github.com/en/actions/using-jobs/running-jobs-in-a-container)
- [Tart macOS VM Manager](https://github.com/cirruslabs/tart)
- [Anka macOS Virtualization](https://veertu.com/)
- [Setup WSL GitHub Action](https://github.com/marketplace/actions/setup-wsl)
- [Cross-Platform GitHub Action](https://github.com/cross-platform-actions/action)
- [Bats-core Testing Framework](https://github.com/bats-core/bats-core)
- [testscript Go CLI Testing](https://bitfieldconsulting.com/posts/cli-testing)
- [Robox Vagrant Boxes](https://github.com/lavabit/robox)
- [Incus System Container Manager](https://github.com/lxc/incus)
- [Linux Containers Image Server](https://images.linuxcontainers.org/)
- [QEMU Snapshot Documentation](https://wiki.qemu.org/Documentation/CreateSnapshot)
- [Rustup CI Templates](https://github.com/rust-lang/rustup/blob/main/ci/actions-templates/README.md)
- [mise CI Documentation](https://mise.jdx.dev/continuous-integration.html)
- [macOS Runner Deprecation (macOS 13)](https://github.blog/changelog/2025-09-19-github-actions-macos-13-runner-image-is-closing-down/)
- [GitHub Actions macOS 15 Intel Runner](https://github.com/actions/runner-images/issues/13045)
- [WSL2 on windows-2025 Runner](https://dwozny.com/posts/windows-2025-docker-wsl2/)
- [Docker systemd in Containers](https://developers.redhat.com/blog/2016/09/13/running-systemd-in-a-non-privileged-container)
