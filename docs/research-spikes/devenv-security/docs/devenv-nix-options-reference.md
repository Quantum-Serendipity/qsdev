# devenv.nix Security-Relevant Configuration Options
- **Source**: https://devenv.sh/reference/options/
- **Retrieved**: 2026-05-12

## Environment & Isolation

**dotenv.enable** - Controls loading of `.env` files into the shell environment. Default: disabled. Manages what secrets/variables are exposed.

**dotenv.filename** - Specifies which dotenv file to load. Default: `.env`. Determines credential exposure scope.

**env** - Sets environment variables directly in devenv configuration. Controls what binaries and paths are available to developers.

## Package & Binary Control

**packages** - Declares all available packages in the development environment. Limits what software can be executed.

**languages.\*** - Language-specific tooling configurations that determine available compilers, interpreters, and related tools.

**scripts.\*** - Custom executable scripts available in the shell. Controls what commands developers can invoke.

## Sandbox & Process Isolation

**processes.\<name\>.linux.capabilities** - Linux capability restrictions for individual processes. Controls privileged operations per process.

**process.manager.implementation** - Selects process manager (hivemind, honcho, mprocs, overmind, process-compose). Affects process isolation mechanisms.

**containers.\<name\>** - Container configurations defining what runs isolated. Includes layers, permissions, entry points controlling execution scope.

## Pre-commit & Git Hooks

**git-hooks.enable** - Activates pre-commit hook framework. Default: disabled. Controls automated code quality/security checks.

**git-hooks.hooks.\*** - Individual hook configurations (linting, formatting, secret scanning). Enforce security policies on commits.

**git-hooks.hooks.ripsecrets** - Detects secret patterns in code before commit. Critical for preventing credential leakage.

## Build & Cache Security

**cachix.enable** - Controls use of Cachix binary caching service. Affects build artifact provenance.

**cachix.pull/push** - Determines what caches are trusted for downloads or uploads. Influences supply chain security.

## Container Layer Security

**containers.\<name\>.layers.\*.perms** - File/directory permission configurations within container layers, controlling access control.

**containers.\<name\>.layers.\*.reproducible** - Enables reproducible builds. Ensures consistency and auditability.
