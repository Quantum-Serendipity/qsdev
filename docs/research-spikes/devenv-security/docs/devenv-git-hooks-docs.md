<!-- Source: https://devenv.sh/git-hooks/ -->
<!-- Retrieved: 2026-05-12 -->

# Devenv Git Hooks Documentation

## Overview

Devenv provides first-class integration with pre-commit through git-hooks.nix, enabling developers to enforce code quality standards automatically.

## Setup Approach

The recommended implementation follows a two-phase strategy:

### Phase 1: Commit-Time Enforcement

Configure hooks in `devenv.nix` to validate code as commits are created. When entering the devenv shell, the system displays: "pre-commit installed at .git/hooks/pre-commit". This ensures linters and formatters execute before code enters the repository.

### Phase 2: CI Verification

Run `devenv test` in continuous integration pipelines to verify formatting compliance across the codebase.

## Supported Hooks

Built-in hooks include:
- **shellcheck** - Shell script linting
- **mdsh** - Markdown code block execution
- **black** - Python formatting
- **ormolu** - Haskell formatting
- **clippy** - Rust linting

Each hook supports customization through package overrides and settings configuration.

## Configuration Details

Hooks are defined within the `git-hooks.hooks` configuration block. Key customization options include:

- **enable**: Activate specific hooks
- **package**: Override default tool versions
- **packageOverrides**: Customize multiple dependencies
- **settings**: Hook-specific configuration parameters

## .pre-commit-config.yaml Management

The `.pre-commit-config.yaml` file is automatically generated and symlinked from the Nix store. This file requires no manual maintenance or repository commits, and devenv automatically adds it to `.gitignore` during initialization.

## Custom Hooks

Developers can create custom hooks by specifying:
- **name**: Display identifier
- **entry**: Command to execute
- **files**: Pattern matching for target files
- **types**: File type filtering
- **excludes**: Files to skip
- **language**: Hook execution environment
- **pass_filenames**: Whether to pass changed files as arguments

Custom hooks integrate seamlessly with the standard pre-commit framework, enabling tailored workflows beyond built-in tools.
