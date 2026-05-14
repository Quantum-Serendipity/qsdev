# Mise IDE Integration Documentation
- **Source**: https://mise.jdx.dev/ide-integration.html
- **Retrieved**: 2026-05-14

## Core Integration Patterns

Mise employs three primary integration methods:

1. **Shims in PATH**: Adding mise's shim directory to the shell's PATH enables IDEs to discover and execute tools. "IDEs work better with shims than they do environment variable modifications."

2. **Direct SDK Selection**: Some JetBrains IDEs (Java, Python, etc.) have native mise support, allowing developers to select tool versions directly in IDE settings.

3. **Plugin-Based Integration**: Custom plugins handle environment variable loading and tool discovery without requiring shell profile modifications.

## Editor-Specific Approaches

**VS Code** offers the most sophisticated integration. The `mise-vscode` plugin automatically configures extensions, manages mise tasks, and loads environment variables from `mise.toml` files. For macOS, developers can configure the automation profile with `["--login"]` arguments to read shell profiles.

**JetBrains IDEs** support direct SDK selection in language settings. When direct support isn't available, developers can symlink the mise directory to mirror asdf's layout, making tools discoverable through existing plugin infrastructure.

**Neovim and Vim** require manual PATH manipulation — prepending the shims directory through init configuration files (lua for Neovim, vimscript for Vim).

**Emacs** can use either traditional PATH configuration or the `mise.el` package, which sets environment variables on a per-buffer basis.

## Key Characteristic

Mise doesn't auto-generate IDE configurations. Instead, it relies on editors reading shell profiles or using dedicated plugins for deeper integration.
