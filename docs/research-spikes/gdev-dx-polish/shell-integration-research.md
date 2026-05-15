# Shell Integration and Ergonomics Research

## Research Question

Beyond shell completion (already planned), what shell integrations would make gdev pleasant to use daily?

## Already Planned

- **Shell completions** (Phase 10) -- bash, zsh, fish, PowerShell via cobra's built-in completion generation
- **Starship integration** -- devenv already has a starship module that gdev's devenv addon can configure

## Starship Prompt Integration

### devenv's Existing Starship Module

devenv provides `starship.enable`, `starship.config.path`, and `starship.config.settings` options. When enabled, it:
1. Exports `STARSHIP_CONFIG` pointing to the config file
2. Initializes starship with shell-specific hooks
3. Unsets `STARSHIP_SHELL` to prevent direnv conflicts

### What gdev Should Generate

A starship.toml that shows project-relevant context:

```toml
# gdev-managed starship config
[custom.gdev]
command = "echo $QSDEV_PROJECT_NAME"
when = "test -n \"$QSDEV_PROJECT_NAME\""
format = "[$output]($style) "
style = "bold cyan"
description = "Active gdev project"

[custom.gdev_security]
command = "echo '🛡'"
when = "test -n \"$QSDEV_SECURITY_PROFILE\""
format = "[$output]($style) "
style = "green"
description = "Security hardening active"
```

The key env vars gdev would set in devenv.nix:
- `QSDEV_PROJECT_NAME` -- detected project name
- `QSDEV_SECURITY_PROFILE` -- active security profile (if any)
- `QSDEV_VERSION` -- qsdev config version
- `QSDEV_ECOSYSTEMS` -- comma-separated detected ecosystems

This lets any prompt tool (not just starship) display gdev context.

### Recommendation

**Include starship config generation as opt-in in the devenv addon.** Set the env vars unconditionally (they are useful beyond starship). Generate starship.toml only when `starship.enable = true` in devenv.nix.

## Aliases and Abbreviations

### The Case Against gdev-Managed Aliases

Aliases are deeply personal. Developers have strong opinions about their shell configuration. gdev should NOT:
- Add aliases to `.bashrc`/`.zshrc` (invasive, fragile)
- Ship a "recommended aliases" file that developers source
- Create shell functions that wrap common commands

### The Case For: devenv Shell Scripts

devenv's `scripts` option (now superseded by tasks, but still available) can expose project-specific commands. gdev already generates devenv.nix, so it can include:

```nix
# In generated devenv.nix
scripts.gcheck.exec = "qsdev devenv doctor";
scripts.gstatus.exec = "qsdev status";
```

But this is redundant -- `qsdev devenv doctor` and `qsdev status` are already short commands. Adding aliases for 4-character savings is not worth the cognitive overhead of "wait, is `gcheck` a real command or an alias?"

### Recommendation

**Do NOT include aliases.** The gdev command namespace is already clean (`qsdev devenv doctor`, `qsdev status`, `qsdev init`). Aliases add confusion without meaningful time savings. Developers who want aliases can create their own.

## Quick-Info Commands

### `qsdev status` (Already in gdev-health-reporting spike)

Shows what tools are active, what ecosystem is detected, config health. This is covered by the sibling spike.

### `qsdev info` (Lightweight Alternative)

For developers who want a quick glance without the full health report:

```
$ gdev info
Project: acme-frontend
Ecosystems: typescript (pnpm), docker
Security: consulting-default profile (6 tools active)
devenv: v2.1.0, shell healthy
gdev: v1.2.0 (config current)
```

This is `qsdev devenv doctor` without the checks -- just read cached state and display it. Subsecond response.

### Recommendation

**Include `qsdev info` as a lightweight status command.** It reads the qsdev config file and displays current state. No evaluation, no checks, instant response. Useful for "where am I? what's active?"

## Shell Hook for Environment Awareness

### Concept: gdev Shell Hook

Similar to devenv's hook, gdev could provide:

```bash
eval "$(gdev hook bash)"
```

This would:
1. Set `QSDEV_PROJECT_NAME` on directory entry
2. Display a one-line notification when entering a gdev-managed project
3. Optionally run `qsdev devenv doctor --quick` on first entry (cached, subsecond)

### Analysis

This duplicates devenv's hook functionality. devenv already activates on `cd`, sets environment variables, and can run enterShell tasks. Adding a second hook creates:
- Double activation delay
- Ordering issues (which hook runs first?)
- Confusion about which tool manages the environment

### Recommendation

**Do NOT add a separate gdev shell hook.** Use devenv's enterShell task to set gdev env vars and display project info. This is one integration point, not two competing hooks.

## Summary: Shell Integration Recommendations

| Feature | Recommendation | Rationale |
|---------|---------------|-----------|
| Shell completions | **Include** (already planned) | Standard for any CLI tool |
| Starship config generation | **Include (opt-in)** | Low cost, devenv has native support |
| gdev env vars in devenv.nix | **Include** | Enables any prompt tool, not just starship |
| Aliases/abbreviations | **Exclude** | Personal preference, cognitive overhead |
| `qsdev info` quick-status | **Include** | Subsecond response, instant context |
| gdev shell hook | **Exclude** | Duplicates devenv hook, creates conflicts |
| devenv enterShell notification | **Include** | One-line "gdev project: acme-frontend" on shell entry |

## Depth Checklist

- [x] Underlying mechanism explained -- starship module, env vars, devenv hooks, shell completion
- [x] Key tradeoffs -- automation vs personalization, multiple hooks vs single integration point
- [x] Compared to alternatives -- starship vs other prompts, gdev hook vs devenv enterShell task
- [x] Failure modes -- double hook activation, alias conflicts, stale env vars
- [x] Concrete examples -- starship.toml config, gdev info output, env var definitions
- [x] Standalone-readable -- yes
