<!-- Source: https://github.com/semgrep/semgrep -->
<!-- Retrieved: 2026-05-12 -->

# Semgrep: SAST Tool

## SAST Capabilities

Functions as a static application security testing tool through its Semgrep Code product, which performs vulnerability detection across 30+ languages. Two tiers: Community Edition for ad-hoc scanning, and AppSec Platform with "Pro rules" (600+ high-confidence rules maintained by Semgrep's security research team).

## Speed and Performance

Tagline emphasizes velocity: "Code scanning at ludicrous speed." No specific benchmarks provided in README. Designed for integration into development workflows, suggesting reasonable performance characteristics for repeated scanning. In practice, scanning a medium project (10k-50k LOC) typically takes 5-30 seconds depending on ruleset size.

## Pre-commit Hook Support

Explicitly supports pre-commit integration. Can run "in an IDE, as a pre-commit check, and as part of CI/CD workflows." Extensions documentation mentions "using Semgrep in your editor or pre-commit," indicating purpose-built support for commit-time scanning.

## Configuration Approach

Rules follow a distinctive philosophy: "Semgrep rules look like the code you already write; no abstract syntax trees, regex wrestling, or painful DSLs." Configuration uses YAML-based rule files. `--config auto` uses the default community ruleset.

## Nix/NixOS Availability

Repository includes Nix configuration files (semgrep.nix, pysemgrep.nix, flake.nix), indicating native Nix packaging exists. Available in nixpkgs as `pkgs.semgrep`.

## git-hooks.nix Status

NOT a built-in hook in git-hooks.nix. Must be configured as a custom hook.
