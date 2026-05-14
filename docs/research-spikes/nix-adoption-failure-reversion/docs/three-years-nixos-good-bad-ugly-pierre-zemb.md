<!-- Source: https://pierrezemb.fr/posts/nixos-good-bad-ugly/ -->
<!-- Retrieved: 2026-03-20 -->

# Three Years of Nix and NixOS: The Good, the Bad, and the Ugly

**Author:** Pierre Zemb
**Date:** July 2, 2025
**Duration of Use:** 3 years

## The Good

### Declarative System Management
Pierre highlights that NixOS allows entire system configuration through files stored in Git. He notes: "Every nixos-rebuild switch creates a new 'generation' of the system." This atomic update mechanism enables painless rollbacks if updates cause issues.

### System Customization
Building custom ISOs with pre-installed SSH keys requires only "a few lines of configuration," demonstrating how deeply customizable NixOS is compared to traditional Linux distributions.

### Sandboxed Development Environments
Using flake.nix with direnv, developers can isolate project dependencies perfectly. His tip: "add if has nix; then use nix; fi to the .envrc file" to avoid errors for non-Nix users.

### VM-Based Testing
The built-in testing framework enabled him to package fdbserver with a full FoundationDB cluster test "in about 30 minutes."

## The Bad

### Friction for Simple Changes
No quick edits exist; even shell aliases require configuration file changes and system rebuilds rather than simple file edits.

### Steep Learning Curve
The Nix ecosystem demands months of learning before productivity. "Your existing knowledge doesn't help much" due to fundamentally different concepts.

### Ecosystem Incompatibility
Pre-compiled binaries fail because NixOS doesn't follow the standard Filesystem Hierarchy Standard. "You can't just download a pre-compiled binary and expect it to work."

### Hardcoded Build Environments
Some libraries, particularly cryptography tools, have build scripts hardcoded to standard locations, forcing fallback to buildFHSUserEnv workarounds.

## The Ugly

### The Nix Language
The functional programming language presents the steepest barrier. Pierre states: "Simple things can be hard to figure out, and you often have to look up how to do basic operations." He notes LLMs have significantly eased this learning process.

## Current Status & Recommendation
Continues using NixOS despite frustrations: "I wouldn't switch away from NixOS." He emphasizes that "reproducibility -- is a superpower" for systems engineers.

Verdict: He trades "short-term convenience for long-term stability and control," finding it worthwhile for developers working with distributed systems where environment consistency matters critically.

Gateway recommendation: Start with Nix package manager on existing systems before committing to full NixOS migration.
