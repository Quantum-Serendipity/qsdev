# Migration Guide

This guide covers adding qsdev to projects with existing configuration files. It addresses four scenarios and provides step-by-step instructions for each.

## Pre-Migration Checklist

Before running `qsdev init`, verify:

1. Your project is a git repository with no uncommitted changes (`git status` is clean).
2. qsdev is installed (see [README](../README.md) for installation methods).
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
   qsdev init --force --dry-run > /tmp/qsdev-preview.txt
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

5. After merging, run `qsdev update` to save state for the final version:

   ```bash
   qsdev update --force
   ```

### Preserving Custom Nix Expressions

The generated `devenv.nix` uses the `manual-merge` strategy. On subsequent runs of `qsdev update`, if you have modified `devenv.nix`, a `devenv.nix.new` sidecar file will be created instead of overwriting your changes. You will see a diff and can merge manually.

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

The generated `.claude/settings.json` uses the `three-way-merge` strategy. To adopt qsdev management while preserving your custom rules:

1. Run with `--force` to generate the initial configuration:

   ```bash
   qsdev init --force --yes
   ```

2. Add back your custom allow/deny rules by editing `.claude/settings.json`.

3. Run `qsdev update` to save state. Future updates will three-way merge, preserving your additions.

### Preserving Custom MCP Servers

`.mcp.json` also uses `three-way-merge`. Custom MCP server entries you add will be preserved during updates, as long as they do not conflict with built-in server names (`context7`, `github`, `semble`, `socket`). Documentation servers (`local-docs-devdocs`, `local-docs-zim`, `man-pages`, `mcp-nixos`) may also be present if enabled via `qsdev docs enable` — avoid conflicting with these names as well.

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

This entry will survive future `qsdev update` runs.

## Scenario 4: Both devenv and Claude Code Exist

Your project has configuration from both systems.

### Recommended Approach

1. **Preview** what qsdev would generate:

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
   qsdev status   # verify security posture
   ```

## Post-Migration Steps

### 1. Activate direnv

If direnv was enabled (the default):

```bash
direnv allow
```

### 2. Verify Security Controls

```bash
qsdev status
devenv test
```

`qsdev status` shows your security posture with a score and grade. `devenv test` runs the security validation suite:
- Pre-commit hooks are installed
- Credential variables are stripped from the environment
- No secrets detected in tracked files
- `DEVENV_SECURITY_HARDENED` sentinel is set

### 3. Commit Generated Files

```bash
git add devenv.nix devenv.yaml .envrc .pre-commit-config.yaml
git add .claude/settings.json .claude/hooks/package-guard.py
git add .claude/skills/ .claude/rules/
git add .mcp.json CLAUDE.md
git add .npmrc              # or pip.conf, etc. (per-ecosystem configs)
git add .gitignore
git add .devinit/           # state files for update workflow
git commit -m "chore: add qsdev security-hardened dev environment"
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

## Updating from Earlier Versions

If your project was initialized before qsdev 0.7.0, running `qsdev update` introduces several new components:

**Policy engine files** — `qsdev update` adds `.qsdev/policy/` with YAML security policy definitions. The policy engine evaluates tool calls against these rules. Existing allowlisted packages remain allowed.

**Self-protection hook** — The `qsdev selfprotect` binary now runs as the first PreToolUse hook, evaluating 18 rules against tool calls. Existing custom hooks are not affected but run after self-protection.

**Cloud deny rules** — If AWS, GCP, or Azure CLI tools are detected in your project, new deny rules are added to `.claude/settings.json` restricting credential file access and authentication commands. The three-way merge preserves your custom rules.

**New MCP servers** — `agent-postmortem` and `version-sentinel` are embedded MCP servers available via `qsdev mcp <name>`. They do not appear in `.mcp.json` but are available through the CLI.

**Documentation servers** — Local documentation is now available via `qsdev docs enable`. Documentation MCP servers are opt-in and do not activate automatically.

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

### Changing languages or services after initial setup

To change languages or services, edit the saved answers file at `.devinit/.qsdev-init-answers.yaml` and then run `qsdev update`, or run a fresh `qsdev init --force` with the new flags.

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

qsdev configures 4 default MCP servers in `.mcp.json`: `context7`, `github`, `socket`, `semble`. Additional servers (`agent-postmortem`, `version-sentinel`, `local-docs-devdocs`, `local-docs-zim`, `man-pages`, `mcp-nixos`) activate based on tool enablement and project detection. Custom servers can be added by editing `.mcp.json` directly after generation — the three-way merge preserves custom entries on update.

### Corrupted or drifted configuration files

If generated files have been accidentally modified or corrupted:

```bash
qsdev repair
```

This restores managed files to their expected state while preserving your customizations in merge-strategy files.

Alternatively, `qsdev check --auto-fix` can restore deleted generated files and add missing deny rules in a single pass.

### Self-protection rule blocked a tool call

Exit code 2 from a PreToolUse hook indicates a self-protection denial. Review the output to see which rule triggered. Self-protection rules (SP-001 through SP-014, MCP-001/002/005, INT-001) use `bypass_tier: enforce_always` and cannot be bypassed. If a legitimate operation is blocked, check whether the tool call is writing to or reading from a protected path.

### Cloud deny rules appeared after update

When `qsdev update` detects AWS, GCP, or Azure project files, it adds deny rules blocking credential file access and authentication commands. Cloud CLIs remain available for read-only operations. To use cloud authentication within the agent, configure per-project credential isolation through the cloud ecosystem module's environment variables.

### Policy warnings on existing dependencies

The package risk scoring system may flag existing dependencies with risk grades on first evaluation. Run `qsdev policy check` to review findings. Low-grade packages are not blocked by default — only policy rules with `block` actions take effect. Use `qsdev policy list` to see which rules are active and their severity levels.

### Docker module renamed to container

The `docker` ecosystem module was renamed to `container` to support both Docker and Podman. Existing `.qsdev.yaml` files with `name: docker` are handled transparently — no manual migration is needed.
