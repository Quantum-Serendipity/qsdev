# Nix Per-User Configuration Limitations
- **Source**: https://nix.dev/manual/nix/2.28/command-ref/conf-file
- **Retrieved**: 2026-05-12

## Configuration File Loading Order

The manual specifies that nix reads settings "in that order":

1. **System-wide** (`/etc/nix/nix.conf`)
2. **User-specific** (via `NIX_USER_CONF_FILES` or `XDG_CONFIG_HOME/nix/nix.conf`)
3. **Environment variable** (`NIX_CONFIG`)

A critical constraint: "Values loaded in this file are not forwarded to the Nix daemon. The client assumes that the daemon has already loaded them."

## Settings Restricted to Trusted Users

Several settings are explicitly restricted:

- **`diff-hook`**: "When using the Nix daemon, `diff-hook` must be set in the `nix.conf` configuration file, and cannot be passed at the command line."
- **`run-diff-hook`**: Same daemon-only restriction applies.
- **`post-build-hook`**: "This option is only settable in the global `nix.conf`, or on the command line by trusted users."

## Substituter Access for Unprivileged Users

The document states that "At least one of the following conditions must be met for Nix to use a substituter":

- The substituter is in the trusted-substituters list
- The calling user is in the trusted-users list

Unprivileged users face a constraint: "Unprivileged users (those set in only `allowed-users` but not `trusted-users`) can pass as `substituters` only those URLs listed in `trusted-substituters`."

## Extra-* Prefix Behavior

The manual explains the mechanism: "for settings that take a list of items, you can prefix the name of the setting by `extra-` to _append_ to the previous value."

This applies to list-based settings like `extra-substituters`, `extra-trusted-public-keys`, and `extra-trusted-substituters`. However, the **substituter access restrictions above still apply** -- unprivileged users cannot bypass trusted-substituters limitations using the extra- prefix.

## Accept-Flake-Config Setting

This experimental feature setting controls whether "Nix will accept Nix configuration settings from a flake without prompting." The default is `false`. This requires the `flakes` experimental feature to be enabled first.

## Key Takeaway

Unprivileged users have **severely restricted** per-user configuration capabilities when using the Nix daemon. System administrators control critical settings through system-wide configuration.
