<!-- Source: https://devenv.sh/files-and-variables/ -->
<!-- Retrieved: 2026-05-12 -->

# devenv Files and Variables - Complete Content

## Files

**devenv.nix**
The foundational configuration file required for setting up a developer environment. It's created through `devenv init` and is comprehensively documented in the basics section.

**devenv.local.nix**
A supplementary configuration file mirroring `devenv.nix`'s structure but excluded from version control. Developers use this to customize their individual environment without affecting team configurations.

**devenv.local.yaml**
Introduced in version 1.10, this file parallels `devenv.yaml` functionality while remaining outside repository tracking, enabling personal environment modifications.

**devenv.yaml**
Handles configuration for inputs and imports, enabling dependency specification and composition management. Supports version constraints through `require_version` settings:
- `require_version: true` enforces CLI-module compatibility
- `require_version: ">=2.1"` applies explicit versioning constraints

**devenv.lock**
Captures pinned input versions, guaranteeing reproducible developer environments across team members.

**.envrc**
Integration logic for direnv automation, enabling automatic environment activation upon directory entry.

## Environment Variables

**$DEVENV_ROOT**
References the project directory containing `devenv.nix`.

**$DEVENV_DOTFILE**
Points to `$DEVENV_ROOT/.devenv`.

**$DEVENV_STATE**
References `$DEVENV_DOTFILE/state`.

**$DEVENV_RUNTIME**
Designates a temporary directory with paths unique to each project root, storing sockets and runtime files. Precedence: `$XDG_RUNTIME_DIR` → `$TMPDIR` → `/tmp`.

**$DEVENV_PROFILE**
References the Nix store path containing final packaged profiles, beneficial for exposing `/bin`, `/etc`, `/var` directories to external programs.

**$DEVENV_HOME**
Points to `~/.local/share/devenv` following XDG standards, preserving garbage collection roots and persistent user-specific data.
