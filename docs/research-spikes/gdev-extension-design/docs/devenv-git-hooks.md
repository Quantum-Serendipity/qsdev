# devenv.sh Git Hooks and Pre-commit Integration
- **Source**: https://devenv.sh/git-hooks/
- **Retrieved**: 2026-05-12

## Overview

The devenv documentation explains that "devenv has first-class integration for pre-commit via git-hooks.nix." This integration enables developers to automatically run linters and formatters at commit time.

## Setup Approach

The recommended implementation uses a two-step process:

### Step 1: Enforce Formatting at Commit Time

Hooks are configured in `devenv.nix` with the `git-hooks.hooks` section. When developers enter the shell environment with `devenv shell`, the system automatically installs pre-commit hooks at `.git/hooks/pre-commit`.

**Example configuration:**

```nix
git-hooks.hooks = {
  shellcheck.enable = true;
  mdsh.enable = true;
  black.enable = true;
  ormolu.enable = true;
  clippy.enable = true;
}
```

Developers can override package versions and configure hook settings. For instance, "some hooks have more than one package, like clippy," allowing customization of individual components like cargo and clippy separately.

### Step 2: Verify in CI

Running `devenv test` validates formatting in continuous integration pipelines.

## .pre-commit-config.yaml File Management

The `.pre-commit-config.yaml` file is automatically generated as a symlink within the Nix store. According to the documentation, "it is not necessary to commit this file to your repository and it can safely be ignored." The system adds this filename to `.gitignore` by default during `devenv init`.

## Custom Hook Definition

Users can define custom hooks with these configurable properties:

- **name**: Display name in reports
- **entry**: Command to execute (required)
- **files**: Regex pattern matching specific file extensions
- **types**: File type filters (e.g., "text", "c")
- **excludes**: Patterns to exclude from processing
- **language**: Hook implementation language (default: "system")
- **pass_filenames**: Boolean controlling whether changed files are passed to the command

## Available Hook Types

The documentation references "the list of all available hooks" in the reference options section, including common tools like shellcheck, black (Python formatter), and ormolu (Haskell formatter).
