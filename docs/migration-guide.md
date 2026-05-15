# Migration Guide

This guide covers adding qsdev to projects with existing configuration files. It addresses four scenarios and provides step-by-step instructions for each.

## Pre-Migration Checklist

Before running `qsdev init`, verify:

1. Your project is a git repository with no uncommitted changes (`git status` is clean).
2. gdev, devenv.sh, and Nix with flakes are installed.
3. You know which languages, services, and security policies you want.

**Back up your existing configuration:**

```bash
git stash   # or commit any pending changes
```

## Scenario 1: No Existing Configuration

The simplest case. No `devenv.nix`, `devenv.yaml`, `.claude/`, or `.envrc` exists.

```bash
cd my-project
qsdev init --yes
```

The tool auto-detects languages from marker files (e.g., `go.mod`, `package.json`, `Cargo.toml`) and generates all configuration. Review with `--dry-run` first if you want to preview:

```bash
qsdev init --dry-run
```

## Scenario 2: Existing devenv Configuration

Your project already has `devenv.nix` and/or `devenv.yaml`.

### Detection

`qsdev init` detects existing devenv files and refuses to proceed:

```
Error: existing configuration found (devenv.nix, devenv.yaml); use --force to overwrite
```

### Option A: Fresh Start (Recommended for Non-Customized Setups)

If your existing `devenv.nix` is minimal or auto-generated:

```bash
qsdev init --force --yes
```

This overwrites all devenv files. Your old `devenv.nix` is replaced with the security-hardened version.

### Option B: Claude Code Only

If you want to keep your existing devenv setup and only add Claude Code configuration:

```bash
qsdev init --claude-only --yes
```

This generates `.claude/settings.json`, `CLAUDE.md`, hooks, skills, and rules without touching devenv files.

### Option C: Manual Merge

If your `devenv.nix` has significant customizations you want to preserve:

1. Generate the new configuration alongside your existing one:

   ```bash
   qsdev init --force --dry-run > /tmp/gdev-preview.txt
   ```

2. Review the preview to understand what would be generated.

3. Run the generation:

   ```bash
   qsdev init --force --yes
   ```

4. Use git to selectively merge:

   ```bash
   git diff devenv.nix   # review changes
   git checkout -p devenv.nix   # interactively restore sections you want to keep
   ```

5. After merging, run `qsdev init --update` to save state for the final version:

   ```bash
   qsdev init --update --force
   ```

### Preserving Custom Nix Expressions

The generated `devenv.nix` uses the `manual-merge` strategy. On subsequent updates via `qsdev init --update`, if you have modified `devenv.nix`, a `devenv.nix.new` sidecar file will be created instead of overwriting your changes. You will see a diff and can merge manually.

## Scenario 3: Existing Claude Code Configuration

Your project already has `.claude/settings.json`, `CLAUDE.md`, or `.mcp.json`.

### Detection

`qsdev init` detects existing Claude Code files:

```
Error: existing configuration found (.claude/settings.json, CLAUDE.md); use --force to overwrite
```

### Option A: Full Replacement

If your existing Claude Code configuration is ad-hoc:

```bash
qsdev init --force --yes
```

### Option B: devenv Only

If you want to keep your existing Claude Code setup and only add devenv:

```bash
qsdev init --devenv-only --yes
```

### Option C: Merge Existing Settings

The generated `.claude/settings.json` uses the `three-way-merge` strategy. To adopt gdev management while preserving your custom rules:

1. Run with `--force` to generate the initial configuration:

   ```bash
   qsdev init --force --yes
   ```

2. Add back your custom allow/deny rules by editing `.claude/settings.json`.

3. Run `qsdev init --update` to save state. Future updates will three-way merge, preserving your additions.

### Preserving Custom MCP Servers

`.mcp.json` also uses `three-way-merge`. Custom MCP server entries you add will be preserved during updates, as long as they do not conflict with built-in server names (`github`, `filesystem`, `postgres`, `fetch`, `socket`).

To add a custom server after initial setup:

```json
{
  "mcpServers": {
    "github": { ... },
    "my-custom-server": {
      "command": "npx",
      "args": ["@myorg/mcp-custom"],
      "env": {"CUSTOM_TOKEN": "${CUSTOM_TOKEN}"}
    }
  }
}
```

This entry will survive future `qsdev init --update` runs.

## Scenario 4: Both devenv and Claude Code Exist

Your project has configuration from both systems.

### Recommended Approach

1. **Preview** what gdev would generate:

   ```bash
   qsdev init --dry-run
   ```

2. **Choose your strategy** based on how much customization you have:
   - Minimal customization: `qsdev init --force --yes`
   - Heavy devenv customization: `qsdev init --claude-only --yes` first, then manually port devenv security settings
   - Heavy Claude Code customization: `qsdev init --devenv-only --yes` first, then merge Claude settings

3. **Validate** after migration:

   ```bash
   devenv shell   # verify the environment works
   devenv test    # verify security controls
   ```

## Post-Migration Steps

### 1. Activate direnv

If direnv was enabled (the default):

```bash
direnv allow
```

### 2. Verify Security Controls

```bash
devenv test
```

This runs the security validation suite:
- Pre-commit hooks are installed
- Credential variables are stripped from the environment
- No secrets detected in tracked files
- `DEVENV_SECURITY_HARDENED` sentinel is set

### 3. Commit Generated Files

```bash
git add devenv.yaml devenv.nix .envrc
git add .claude/ CLAUDE.md
git add .mcp.json           # if MCP servers were configured
git add .devinit/            # state files for update workflow
git add renovate.json        # or .github/dependabot.yml
git add .github/workflows/security-scan.yml
git add docs/security-overview.md
git commit -m "chore: add gdev security-hardened devenv configuration"
```

### 4. Team Communication

After committing, team members need to:

```bash
git pull
direnv allow        # if using direnv
# or
devenv shell        # manual activation
```

No additional `qsdev init` is needed for team members -- the committed files configure their environment.

## Common Issues

### `devenv.nix already exists; use --force to overwrite`

You have an existing devenv setup. See Scenario 2 above. Use `--force` to overwrite, `--claude-only` to skip devenv, or `--dry-run` to preview first.

### `.claude/settings.json already exists; use --force to overwrite`

You have an existing Claude Code setup. See Scenario 3 above. Use `--force` to overwrite or `--devenv-only` to skip Claude Code.

### `devenv-only` and `claude-only` cannot be used together

These flags are mutually exclusive. Use `qsdev init` without either flag to generate both, or run them separately:

```bash
qsdev init --devenv-only --yes
qsdev init --claude-only --yes
```

### `update` cannot be combined with `lang`, `service`, or `profile`

The `--update` flag regenerates from saved answers. To change languages or services, edit the saved answers file at `.devinit/.qsdev-init-answers.yaml` and then run `--update`, or run a fresh `qsdev init --force` with the new flags.

### Pre-commit hooks fail after migration

If pre-commit hooks report errors:

1. Ensure you are inside the devenv shell (`devenv shell` or direnv-activated).
2. Run `pre-commit install` to reinstall hooks.
3. Check that `ripsecrets` is available: `which ripsecrets`.

### Lock file warnings during first commit

The `lock-file-audit` hook warns when `devenv.lock` or `flake.lock` change. This is expected on first setup. The warning reminds reviewers to verify lock file changes -- acknowledge it and proceed.

### Credential variables still present in environment

If `devenv test` reports credential variables are set:

1. Verify `clean.enabled: true` in `devenv.yaml`.
2. Check that the variable is not in `clean.keep`.
3. Exit and re-enter the devenv shell (clean mode applies on shell entry, not retroactively).

### MCP server "unknown" error

Only five built-in MCP servers are recognized: `github`, `filesystem`, `postgres`, `fetch`, `socket`. Custom servers must be added via the Go API or by editing `.mcp.json` directly after generation.
