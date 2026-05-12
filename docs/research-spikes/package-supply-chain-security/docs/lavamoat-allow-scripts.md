# @lavamoat/allow-scripts: Configuration Guide

- **Source**: https://lavamoat.github.io/guides/allow-scripts/
- **Retrieved**: 2026-05-12

## Overview

`@lavamoat/allow-scripts` is a CLI tool that restricts dependency lifecycle hooks to an approved allowlist. The documentation describes it as enabling teams to "execute _only_ the dependency lifecycle hooks specified in an _allowlist_."

## Installation Options

**Global Installation (Recommended for Setup)**
Install globally to initialize projects without triggering other dependency installations:
```
npm i -g @lavamoat/allow-scripts
```

**Project-Local Installation**
For team contributors, install as a dev dependency using your package manager (npm, Yarn, pnpm, or Yarn Berry v3+).

## Setup Process

### Initialization
Running the `setup` command performs several actions:

1. Adds `ignore-scripts=true` to `.npmrc` (or `enableScripts: false` to `.yarnrc.yml`)
2. Creates these configuration files if they don't exist
3. Adds `@lavamoat/preinstall-always-fail` as a failsafe dev dependency that throws errors if scripts somehow execute anyway

### Configuration Methods

**Automatic Configuration**
The `auto` command generates an allowlist by analyzing current dependencies and writing it to package.json's `lavamoat` property.

**Manual Configuration**
Configuration uses a `Record<PackageName, boolean>` structure where:
- `true` permits script execution
- `false` blocks script execution
- Missing entries generate warnings

### Configuration Structure

The allowlist appears in `package.json` like this:

```json
{
  "lavamoat": {
    "allowScripts": {
      "keccak#3.0.4": true,
      "rezeplayer>core-js#3.49.0": false,
      "some-package-denied-all-versions": false
    }
  }
}
```

**Version Pinning Strategy**
Versions are mandatory for allowed packages (protecting against maintainer compromise). For denied packages, removing the version suffix prevents updates while maintaining the blocklist across version changes.

## Running Allowed Scripts

Execute `allow-scripts run` (or invoke without a command) to run lifecycle scripts for configured packages. The tool fails if unconfigured dependencies attempt execution, prompting users to run `allow-scripts auto`.

## Security Considerations

The documentation emphasizes the "Principle of Least Privilege," recommending developers limit allowlists rather than permit everything historically executed. The guide suggests using `can-i-ignore-scripts` to evaluate which scripts are genuinely necessary.

## Advanced: Bin Script Protection

An experimental `--experimental-bins` flag addresses shell injection attacks where malicious `bin` scripts match executables in the user's PATH. This feature:

- Disables automatic bin script linking during setup
- Generates an allowlist of permitted bin scripts
- Replaces disallowed scripts with failing executables

## Workflow Integration

A recommended practice involves creating a `setup` script in package.json combining installation and post-processing:

```json
{
  "scripts": {
    "setup": "npm install && npm exec allow-scripts && tsc -b"
  }
}
```

## Yarn Berry Integration

For Yarn v3+, a plugin can be imported to automatically execute `allow-scripts` after installation, streamlining the developer experience.
