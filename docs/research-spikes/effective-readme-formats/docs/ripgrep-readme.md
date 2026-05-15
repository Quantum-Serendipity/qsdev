<!-- Source: https://raw.githubusercontent.com/BurntSushi/ripgrep/master/README.md -->
<!-- Retrieved: 2026-05-15 -->
<!-- Note: Content was summarized by WebFetch tool; not raw source. Key structural elements preserved. -->

# ripgrep (rg)

ripgrep is a line-oriented search tool that recursively searches the current directory for a regex pattern. It automatically respects gitignore rules and skips hidden files and binaries by default.

## Key Features

- Recursive searching with automatic filtering enabled by default
- File type filtering (e.g., `rg -tpy foo` for Python files)
- Full Unicode support while maintaining speed
- Optional PCRE2 regex engine for advanced patterns
- Support for multiple file encodings and compressed files
- Configuration file support

## Performance / Benchmarks

Benchmarks demonstrate ripgrep's speed advantages. In testing against the Linux kernel source:
- ripgrep (Unicode): 0.082s
- git grep: 0.273s
- The Silver Searcher: 0.443s

## Installation

Available across major platforms:
- macOS: `brew install ripgrep`
- Linux distributions: Various package managers (apt, pacman, dnf, zypper, etc.)
- Windows: Chocolatey, Scoop, or Winget
- Rust developers: `cargo install ripgrep`

## Why Use It?

Developers should consider ripgrep if they value speed, sensible defaults with filtering, fewer bugs, and Unicode compatibility. However, those needing POSIX portability or ubiquitous tool availability might prefer traditional grep.
