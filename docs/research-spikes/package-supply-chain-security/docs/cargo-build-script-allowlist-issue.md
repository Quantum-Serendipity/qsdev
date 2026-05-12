# Cargo Build Script Allowlist Mode - Issue #13681

- **Source**: https://github.com/rust-lang/cargo/issues/13681
- **Retrieved**: 2026-05-12

## Overview

This GitHub issue proposes a security-focused feature for Rust's Cargo package manager to control build script execution through an allowlist mechanism.

## The Problem

Build scripts expand the attack surface for supply chain security threats by enabling arbitrary code execution. While most crates don't require build scripts, those that do present a potential vulnerability vector.

## Proposed Solution

The feature would introduce a configurable allowlist mode that:

- **Blocks execution** of build scripts when the mode is enabled
- **Fails compilation** if any dependency contains a build script that isn't approved
- **Requires explicit allowlisting** of crates with audited build scripts before execution proceeds
- **Configuration** through `Cargo.toml` and related configuration files

## Key Characteristics

**Status:** Open proposal requiring further design work (labeled "S-needs-design")

**Scope:** Complementary to, but distinct from, ongoing sandboxing initiatives

**Additional Benefits:** May encourage developers to minimize unnecessary build script usage

## Related Discussions

The proposal references ongoing parallel efforts:
- Sandbox/jail build scripts initiative (#5720)
- Community discussion on sandboxing build.rs and proc macros
- Pre-RFC on sandboxed WebAssembly-based proc macro compilation

## Current State

The issue remains unassigned with no active development or pull requests associated with it as of the documentation date.
