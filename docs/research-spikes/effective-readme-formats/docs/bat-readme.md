<!-- Source: https://raw.githubusercontent.com/sharkdp/bat/master/README.md -->
<!-- Retrieved: 2026-05-15 -->
<!-- Note: Nearly complete raw content returned by WebFetch. -->

<p align="center">
  <img src="doc/logo-header.svg" alt="bat - a cat clone with wings"><br>
  <a href="https://github.com/sharkdp/bat/actions?query=workflow%3ACICD"><img src="https://github.com/sharkdp/bat/workflows/CICD/badge.svg" alt="Build Status"></a>
  <img src="https://img.shields.io/crates/l/bat.svg" alt="license">
  <a href="https://crates.io/crates/bat"><img src="https://img.shields.io/crates/v/bat.svg?colorB=319e8c" alt="Version info"></a><br>
  A <i>cat(1)</i> clone with syntax highlighting and Git integration.
</p>

<p align="center">
  <a href="#syntax-highlighting">Key Features</a> |
  <a href="#how-to-use">How To Use</a> |
  <a href="#installation">Installation</a> |
  <a href="#customization">Customization</a> |
  <a href="#project-goals-and-alternatives">Project goals, alternatives</a><br>
  [English] [Chinese] [Japanese] [Korean] [Russian]
</p>

### Syntax highlighting

`bat` supports syntax highlighting for a large number of programming and markup languages:

![Syntax highlighting example](https://imgur.com/rGsdnDe.png)

### Git integration

`bat` communicates with `git` to show modifications with respect to the index (see left sidebar):

![Git integration example](https://i.imgur.com/2lSW4RE.png)

### Show non-printable characters

You can use the `-A`/`--show-all` option to show and highlight non-printable characters:

![Non-printable character example](https://i.imgur.com/WndGp9H.png)

### Automatic paging

By default, `bat` pipes its own output to a pager (e.g. `less`) if the output is too large for one screen.

## How to use

Display a single file on the terminal:
```bash
bat README.md
```

Display multiple files at once:
```bash
bat src/*.rs
```

Read from stdin, determine the syntax automatically:
```bash
curl -s https://sh.rustup.rs | bat
```

### Integration with other tools

Extensive integration examples with fzf, find/fd, ripgrep, tail -f, git, git diff, xclip, man, prettier/shfmt/rustfmt, and --help message highlighting.

## Installation

Comprehensive installation section covering:
- Ubuntu/Debian (apt, .deb packages)
- Alpine Linux, Arch Linux, Fedora, Gentoo, FreeBSD, OpenBSD
- Nix, openSUSE, Snap
- macOS (Homebrew, MacPorts)
- Windows (WinGet, Chocolatey, Scoop, prebuilt binaries)
- From source (cargo install)

Includes Repology packaging status badge.

## Customization

Extensive customization section covering:
- Highlighting themes (with fzf-based theme previewer)
- 8-bit themes (ansi, base16, base16-256)
- Output style options
- Adding new syntaxes and themes
- File type associations
- Pager configuration
- Dark mode support

## Project goals and alternatives

- Provide beautiful, advanced syntax highlighting
- Integrate with Git to show file modifications
- Be a drop-in replacement for (POSIX) cat
- Offer a user-friendly command-line interface

References doc/alternatives.md for comparison.

MIT License or Apache License 2.0.
