<!-- Source: https://raw.githubusercontent.com/sharkdp/fd/master/README.md -->
<!-- Retrieved: 2026-05-15 -->
<!-- Note: Nearly complete raw content returned by WebFetch. -->

# fd

[![CICD](https://github.com/sharkdp/fd/actions/workflows/CICD.yml/badge.svg)](https://github.com/sharkdp/fd/actions/workflows/CICD.yml)
[![Version info](https://img.shields.io/crates/v/fd-find.svg)](https://crates.io/crates/fd-find)
[Chinese] [Korean]

`fd` is a program to find entries in your filesystem.
It is a simple, fast and user-friendly alternative to `find`.
While it does not aim to support all of `find`'s powerful functionality, it provides sensible
(opinionated) defaults for a majority of use cases.

[Installation](#installation) | [How to use](#how-to-use) | [Troubleshooting](#troubleshooting)

## Features

* Intuitive syntax: `fd PATTERN` instead of `find -iname '*PATTERN*'`.
* Regular expression (default) and glob-based patterns.
* Very fast due to parallelized directory traversal.
* Uses colors to highlight different file types (same as `ls`).
* Supports parallel command execution
* Smart case: case-insensitive by default, switches to case-sensitive if pattern contains uppercase
* Ignores hidden directories and files, by default.
* Ignores patterns from your `.gitignore`, by default.
* The command name is *50%* shorter than `find` :-)

## Demo

![Demo](doc/screencast.svg)

## How to use

Extensive usage section with examples:
- Simple search
- Regular expression search
- Specifying root directory
- List all files recursively
- Searching for file extensions
- Searching for file names
- Hidden and ignored files
- Matching full path
- Command execution (with placeholder syntax)
- Excluding files/directories
- Deleting files
- Full command-line options output

## Benchmark

Comparison against find on ~750,000 subdirectories and ~4 million files:
- find (regex): 19.922s
- find (glob): 11.226s  
- fd: 854.8ms

fd is approximately **23 times faster** than `find -iregex` and about **13 times faster** than `find -iname`.

## Troubleshooting

Common issues: fd not finding files (hidden/ignored), colorized output, regex patterns, aliases/functions.

## Integration with other programs

Examples with fzf, rofi, emacs, tree, xargs/parallel.

## Installation

Comprehensive section covering 20+ platforms with Repology badge:
Ubuntu, Debian, Fedora, Alpine, Arch, Gentoo, openSUSE, Void, ALT, Solus, RHEL/Alma/Rocky, macOS (Homebrew/MacPorts), Windows (Scoop/Chocolatey/Winget), GuixOS, Mise, NixOS, Flox, FreeBSD, npm, from source (cargo), from binaries.

MIT License and Apache License 2.0.
