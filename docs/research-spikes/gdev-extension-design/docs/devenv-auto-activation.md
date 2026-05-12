# devenv.sh Auto-Activation (Native Shell Hook)
- **Source**: https://devenv.sh/auto-activation/
- **Retrieved**: 2026-05-12

## Overview

"devenv includes a built in shell hook that automatically activates your developer environment when you `cd` into a project directory. No external tools required."

## Setup Instructions

Add a single command to your shell configuration:

**Bash** (~/.bashrc):
```
eval "$(devenv hook bash)"
```

**Zsh** (~/.zshrc):
```
eval "$(devenv hook zsh)"
```

**Fish** (~/.config/fish/config.fish):
```
devenv hook fish | source
```

**Nushell** (config.nu):
```
devenv hook nu | save --force ~/.cache/devenv/hook.nu
source ~/.cache/devenv/hook.nu
```

## Trust Management

### Allowing Projects

```
cd ~/myproject
devenv allow
```

### Revoking Trust

```
cd ~/myproject
devenv revoke
```

## How the Hook Functions

1. Searches upward from current directory for a `devenv.yaml` file
2. Verifies the project exists in the trust database
3. Executes `devenv shell` in a subshell if trusted

**Important limitation**: "The hook only detects projects that have a `devenv.yaml` file. Projects with only `devenv.nix` (without `devenv.yaml`) are not detected."

## Automatic Deactivation

Environments terminate automatically when exiting the project directory.

## Re-entry Protection

The system prevents environment nesting within subdirectories.

## Comparison with direnv

| Feature | devenv hook | direnv |
|---------|-------------|--------|
| External dependencies | None | Requires direnv |
| Setup complexity | One shell config line | direnv installation + `.envrc` files |
| Trust scope | Per project directory | Per `.envrc` file |
| Implementation | Spawns a subshell | Modifies current shell in place |
| Exit behavior | Subshell exits automatically | direnv unloads variables |
