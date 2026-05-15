<!-- Source: https://raw.githubusercontent.com/eza-community/eza/main/README.md -->
<!-- Retrieved: 2026-05-15 -->
<!-- Note: Nearly complete raw content returned by WebFetch. -->

<div align="center">
   Sponsor banner: Warp, the AI terminal for developers

# eza

A modern replacement for ls.

Badges: Gitter, Built with Nix, Contributor Covenant, Unit tests, Crates.io version, License
</div>

![eza demo gif](docs/images/screenshots.png)

---

**eza** is a modern alternative for the venerable file-listing command-line program `ls` that ships with Unix and Linux operating systems, giving it more features and better defaults.
It uses colours to distinguish file types and metadata.
It knows about symlinks, extended attributes, and Git.
And it's **small**, **fast**, and just **one single binary**.

By deliberately making some decisions differently, eza attempts to be a more featureful, more user-friendly version of `ls`.

---

**eza** features not in exa (non-exhaustive):
- Fixes "The Grid Bug" introduced in exa 2021
- Hyperlink support
- Mount point details
- Selinux context output
- Git repo status output
- Human readable relative dates
- Several security fixes
- Support for `bright` terminal colours
- Many smaller bug fixes/changes
- Configuration `theme.yml` file for customization of colors and icons

---

## Try it!

### Nix
```
nix run github:eza-community/eza
```

# Installation

Available for Windows, macOS and Linux. Platform-specific instructions in INSTALL.md.

Repology packaging status badge included.

---

# Command-line options

Options organized with collapsible `<details>` sections:
- Display options (collapsible)
- Filtering options (collapsible)
- Long view options (collapsible)
- Custom Themes (collapsible)

# Hacking on eza

Code of conduct and contributing guide links.

Star History Chart included at bottom.
