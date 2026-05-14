---
source: https://github.com/prefix-dev/pixi
retrieved: 2026-03-20
---

# Pixi: Comprehensive Project Overview

## Project Statistics
- **Stars**: 6.6k on GitHub
- **Repository**: prefix-dev/pixi (public)
- **Commits**: 2,941 on main branch
- **License**: BSD-3-Clause

## Core Description
Pixi is a cross-platform package management system built on conda foundations. The project describes itself as providing "an exceptional experience similar to popular package managers like cargo or npm, but for any language."

## Key Features
- **Multi-language support**: Python, C++, R, and other languages via conda packages
- **Cross-platform**: Linux, Windows, macOS (including Apple Silicon)
- **Automatic lock file**: Always maintains up-to-date lockfiles
- **Cargo-like CLI**: Clean, intuitive command-line interface
- **Flexible installation**: Per-project or system-wide tool deployment
- **Written in Rust**: Built atop the rattler library
- **Production-ready**: Backward-compatible file format ensures reliability

## Platforms Supported
- Linux (Arch, Alpine, and general distributions)
- macOS (with native Apple Silicon support)
- Windows (PowerShell support)
- Available via package managers: brew, winget, pacman, apk

## Installation Methods
- curl-based script (macOS/Linux)
- PowerShell (Windows)
- Distribution package managers (brew, winget, pacman, apk)
- Source compilation via cargo

## Shell Integration
Supports autocompletion for: Bash, Zsh, Fish, PowerShell, Nushell, and Elvish shells

## Core Capabilities
- **Workspace initialization** and management
- **Dependency addition/removal** with lock file updates
- **Task definition and execution**
- **Global package installation** (similar to pipx/condax)
- **Environment activation** via shell integration
- **GitHub Actions integration** with automatic caching
- **Configuration management** through CLI

## Notable Commands
- `pixi init`: Create workspace
- `pixi add`: Add dependencies
- `pixi run`: Execute tasks
- `pixi shell`: Activate environment
- `pixi global install`: System-wide tool installation
- `pixi exec`: Temporary environment execution

## Community & Development
- **Discord**: Active community server (discord.gg/kKV8ZxyzY4)
- **Contributing**: Welcomes new contributors, offers "good first issue" tickets
- **Developed by**: prefix.dev
- **Documentation**: Available at pixi.sh

## Planned Features
- Build and publish Conda packages
- Dependencies from source
- Enhanced global package determinism across machines
