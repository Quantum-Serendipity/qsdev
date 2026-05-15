<!-- Source: https://raw.githubusercontent.com/ajeetdsouza/zoxide/main/README.md -->
<!-- Retrieved: 2026-05-15 -->
<!-- Note: Nearly complete raw content returned by WebFetch. -->

<div align="center">

Sponsor banners: Warp, Recall.ai

---

# zoxide

[![crates.io badge][crates.io-badge]][crates.io]
[![Downloads badge][downloads-badge]][releases]
[![Built with Nix badge][builtwithnix-badge]][builtwithnix]

zoxide is a **smarter cd command**, inspired by z and autojump.

It remembers which directories you use most frequently, so you can "jump" to
them in just a few keystrokes.
zoxide works on all major shells.

[Getting started](#getting-started) | [Installation](#installation) | [Configuration](#configuration) | [Integrations](#third-party-integrations)

</div>

## Getting started

![Tutorial][tutorial]

```sh
z foo              # cd into highest ranked directory matching foo
z foo bar          # cd into highest ranked directory matching foo and bar
z foo /            # cd into a subdirectory starting with foo

z ~/foo            # z also works like a regular cd command
z foo/             # cd into relative path
z ..               # cd one level up
z -                # cd into previous directory

zi foo             # cd with interactive selection (using fzf)

z foo<SPACE><TAB>  # show interactive completions (bash 4.4+/fish/zsh only)
```

## Installation

zoxide can be installed in 4 easy steps:

1. **Install binary** - Platform-specific instructions in collapsible sections:
   - Linux / WSL (install script + extensive package manager table)
   - macOS (Homebrew, cargo, MacPorts, etc.)
   - Windows (winget, Chocolatey, Scoop, cargo)
   - BSD (cargo, pkg)
   - Android (Termux)

2. **Setup zoxide on your shell** - Collapsible sections for each shell:
   Bash, Elvish, Fish, Nushell, PowerShell, Tcsh, Xonsh, Zsh, any POSIX shell

3. **Install fzf** (optional) - for completions / interactive selection

4. **Import your data** (optional) - from autojump, fasd, z, z.lua, zsh-z, atuin

## Configuration

### Flags
- `--cmd` to change prefix
- `--hook` to configure scoring behavior
- `--no-cmd` to prevent command definition

### Environment variables
- `_ZO_DATA_DIR`, `_ZO_ECHO`, `_ZO_EXCLUDE_DIRS`, `_ZO_FZF_OPTS`, `_ZO_MAXAGE`, `_ZO_RESOLVE_SYMLINKS`

## Third-party integrations

Extensive table of 20+ integrations (file managers, text editors, launchers, session managers, etc.)
