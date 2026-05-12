# Sandboxed Build Scripts in Rust: Project Goals

- **Source**: https://rust-lang.github.io/rust-project-goals/2024h2/sandboxed-build-script.html
- **Retrieved**: 2026-05-12

## Core Proposal

The Rust project aims to "explore different strategies for sandboxing build script executions in Cargo." This initiative, led by Weihang Lo, seeks to restrict what build scripts can access on systems without explicit permission.

## Primary Motivation

The security concern centers on trust. As the document explains, "build scripts in Cargo can do literally anything from network requests to executing arbitrary binaries." Currently, this relies entirely on developer trust within the community. When that trust breaks down, intensive code reviews become necessary across dependencies—a practically impossible task given release velocities.

The proposal identifies two key problems:
1. **Security vulnerability** - Build scripts operate as "an enormous `unsafe` block" without scrutiny mechanisms comparable to Rust's safety guarantees
2. **Non-determinism** - Uncontrolled file system and network access creates unpredictable builds that hamper reproducibility

## Technical Approach

### Sandbox Technologies Under Exploration

The initiative examines WebAssembly System Interface (WASI) and Cackle as potential sandboxing mechanisms. These would allow running build scripts in isolated environments while permitting necessary operations like linking system libraries.

### Design Principles

Key architectural axioms include:
- Restricting filesystem, network, and process access by default
- Maintaining cross-platform compatibility across tier-1 platforms
- Ensuring `-sys` crates (which probe system libraries) function properly
- Using declarative configuration for permission grants
- Allowing functionality when sandboxing is disabled

## Timeline and Milestones

Progress targets include:
- July 2024: Prior art summary
- August 2024: Basic prototype completion
- October 2024: Configuration interface design
- December 2024: Integration of interface with prototype

## Relationship to Procedural Macros

The project parallels a Google Summer of Code initiative exploring WebAssembly sandboxing for procedural macros, recognizing both features share "the same flaw — arbitrary code execution."

## Vision

The "shiny future" envisions sandboxed builds becoming default by the next Rust Edition, with unified configuration interfaces for both build scripts and proc-macros, plus displaying permission requirements on crates.io.
