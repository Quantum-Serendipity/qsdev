<!-- Source: https://github.com/gitleaks/gitleaks -->
<!-- Retrieved: 2026-05-12 -->

# Gitleaks: Secret Detection Tool Overview

## Core Functionality

Gitleaks is a command-line tool designed to detect secrets like passwords, API keys, and tokens in git repositories, files, and streamed input. It performs detection primarily through regex pattern matching combined with entropy analysis.

## Detection Mechanism

Two main detection strategies:

1. **Regex Matching**: Uses Go-style regular expressions to identify potential secrets based on characteristic patterns
2. **Entropy Analysis**: Calculates Shannon entropy values to distinguish actual secrets from false positives, with configurable minimum thresholds

The creators explain their approach in a blog post titled "Regex is (almost) all you need," suggesting regex forms the backbone of their detection strategy.

## Configuration System

TOML-based configuration files with clear precedence order:
- Command-line `--config` flag (highest priority)
- Environment variables (`GITLEAKS_CONFIG` or `GITLEAKS_CONFIG_TOML`)
- Repository-level `.gitleaks.toml` file
- Built-in default configuration

The configuration framework supports:
- Custom rule definitions with regex patterns and entropy thresholds
- Rule inheritance and extension from default or custom base configs
- Multiple allowlist strategies (global, rule-specific, and targeted)
- Composite rules requiring multiple pattern matches within defined proximity

## Pre-Commit Integration

Gitleaks can function as a git pre-commit hook through:
- Direct script installation in `.git/hooks/`
- Pre-commit framework integration via `.pre-commit-config.yaml`
- Ability to skip checks using `SKIP=gitleaks` environment variable

## Scanning Modes

Three operational modes:
- **git**: Analyzes repository history using `git log -p`
- **dir**: Scans directories and individual files
- **stdin**: Processes piped input

## Advanced Features

**Decoding Support**: Automatically finds and decodes encoded text (percent, hex, base64) with recursive decoding up to configurable depths

**Archive Scanning**: Extracts and scans contents of compressed archives (zip, tar, gzip, etc.) with nested archive support

**Reporting Formats**: Generates output in JSON, CSV, JUnit, SARIF, or custom template formats

## Baseline Management

Users can create baselines from previous scans to ignore known findings, streamlining incremental security checks on large repositories.

## Nix/devenv Integration

No mentions of Nix or devenv integration in the documentation. Available in nixpkgs as `pkgs.gitleaks`.
