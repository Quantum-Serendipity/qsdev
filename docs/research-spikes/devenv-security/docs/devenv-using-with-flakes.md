<!-- Source: https://devenv.sh/guides/using-with-flakes/ -->
<!-- Retrieved: 2026-05-12 -->

# Using devenv with Nix Flakes - Complete Content

## Overview

Nix Flakes provide standardized project management by allowing you to "specify dependencies as inputs" and "pin those dependencies in a lock file." The devenv module system can integrate into Flakes as a `devShell` output.

## Key Recommendation

For most projects, the documentation recommends using devenv.nix with the dedicated devenv CLI rather than the Flake integration, citing advantages in simplicity, performance, and developer-focused design.

## When to Use Flake Integration

Consider Flake integration when:
- Maintaining existing flake-based ecosystems
- Developer environments need downstream flake consumption
- You're experienced with Nix
- You understand Flake technical limitations

## Feature Comparison

The devenv CLI offers advantages over Nix Flakes including external flake inputs, shared remote configs, built-in container support, garbage-collection protection, faster evaluation, evaluation caching, and cross-project references. However, Flakes provide pure evaluation capabilities.

## Basic Setup

Initialize with: `nix flake init --template github:cachix/devenv`

This creates `flake.nix` and `.envrc` files.

## Minimal flake.nix Configuration

The documentation provides a template using `devenv.lib.mkShell` with nixpkgs input and devenv as inputs. The configuration supports packages, enterShell scripts, and processes definitions within modules.

## Entering the Shell

Use `nix develop --no-pure-eval` to enter the shell. The `--no-pure-eval` flag is necessary because "Flakes use pure evaluation by default, which prevents devenv from figuring out the environment its running in."

## Process and Service Management

Once in the shell, `devenv up` launches processes and services. The documentation notes that "running tests with flakes doesn't support starting processes."

## direnv Integration

The template provides `.envrc` configuration for automatic shell activation. Setting `DEVENV_IN_DIRENV_SHELL=true` enables caching where `devenv up` skips re-evaluation.

## Multiple Development Shells

Define multiple shells in a central `flake.nix` under `devShells.${system}` with different names, then access them via `nix develop --no-pure-eval .#projectName`.

## External Flakes

Reference external flake repositories using paths or GitHub references without adding `flake.nix` to your project, though this approach lacks version certainty compared to local lock files.
