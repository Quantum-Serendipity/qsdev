# Git Hooks in devenv
- **Source**: https://devenv.sh/git-hooks/
- **Retrieved**: 2026-05-12

## Overview

devenv integrates with pre-commit through git-hooks.nix, enabling automated linting and formatting checks at commit time and in CI/CD pipelines.

## Configuration

Git hooks are configured in `devenv.nix` under the `git-hooks.hooks` section. The framework uses a declarative approach where you enable hooks and customize their behavior.

### Basic Setup Example

```nix
git-hooks.hooks = {
  shellcheck.enable = true;
  mdsh.enable = true;
  black.enable = true;
};
```

## Available Hooks

The documentation references "the list of all available hooks" in the reference section but doesn't enumerate them on this page. Notable examples mentioned include:

- **shellcheck** - Shell script linting
- **mdsh** - Execute examples from Markdown
- **black** - Python code formatting
- **ormolu** - Haskell formatting
- **clippy** - Rust linting

## Configuration Options

### Package Overrides
```nix
ormolu.enable = true;
ormolu.package = pkgs.haskellPackages.ormolu;
```

### Hook Settings
```nix
clippy.settings.allFeatures = true;
```

## Custom Hooks

| Option | Purpose | Default |
|--------|---------|---------|
| `enable` | Activate the hook | Required |
| `name` | Display name in reports | - |
| `entry` | Command to execute | Mandatory |
| `files` | Regex pattern for files | "" (all) |
| `types` | File types to match | [ "file" ] |
| `excludes` | Patterns to exclude | [ ] |
| `language` | Hook installation method | "system" |
| `pass_filenames` | Pass changed files as args | true |

### Custom Hook Example

```nix
git-hooks.hooks.unit-tests = {
  enable = true;
  name = "Unit tests";
  entry = "make check";
  files = "\\.(c|h)$";
  types = [ "text" "c" ];
  language = "system";
};
```

## Execution Model

**Commit-Time**: Hooks run automatically when committing, installed via `devenv shell`.

**CI Verification**: Run `devenv test` to verify formatting in continuous integration.

## Generated Configuration

The `.pre-commit-config.yaml` file is auto-generated and symlinked -- not committed to version control. It's automatically added to `.gitignore` by `devenv init`.
