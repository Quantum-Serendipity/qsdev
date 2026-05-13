<!-- Sources: https://rustup.rs/ https://mise.jdx.dev/installing-mise.html https://volta.sh/ https://determinate.systems/nix-installer/ https://github.com/hashicorp/hc-install -->
<!-- Retrieved: 2026-05-12 -->

# Self-Bootstrapping Installer Patterns

## Rustup Pattern
- `curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh`
- Downloads pre-built rustup-init binary for detected platform
- rustup-init is a Rust binary (not a shell script) that installs the toolchain
- Windows: separate rustup-init.exe download (no curl|bash)
- Manages PATH, creates shims, handles updates

## Mise Pattern
- `curl https://mise.run | sh`
- Downloads pre-built binary for detected OS/arch
- Supports: macos-x64, macos-arm64, linux-x64, linux-x64-musl, linux-arm64, linux-armv6/7
- Also distributed via: brew, apt, dnf, pacman, apk, scoop, winget, npm, cargo
- Shell-specific variants: `curl https://mise.run/bash | sh` (auto-configures shell)

## Volta Pattern
- `curl https://get.volta.sh | bash`
- Built in Rust, ships as static binary
- Creates shim directory in PATH
- Shims intercept commands and delegate to correct version
- No shell integration needed (shims handle everything)

## Nix/Determinate Systems Pattern
- `curl --proto '=https' --tlsv1.2 -sSf -L https://install.determinate.systems/nix | sh -s -- install`
- Written in Rust (not Bash) - avoids Bash compatibility issues
- Single static binary with no external deps
- Supports macOS, Linux, WSL2, containers, CI
- Clean uninstall: `/nix/nix-installer uninstall`
- macOS: graphical .pkg installer available

## HashiCorp hc-install Pattern
- Go library for downloading/locating HashiCorp binaries
- Verifies signatures and checksums
- Supports: filesystem search, release downloads, checkpoint, source builds
- Does NOT: determine install path, manage PATH, upgrade binaries
- CLI: `hc-install install -version 1.3.7 terraform`

## Common Patterns Across All

1. curl|sh downloads a small bootstrapper (not the full tool)
2. Bootstrapper detects OS/arch, downloads correct binary
3. Binary is pre-compiled (no build tools needed on target)
4. SHA256/GPG verification of downloads
5. PATH modification (shell rc files or shim directories)
6. Windows gets separate treatment (exe download or PowerShell)
