# Sandboxing Applications with Bubblewrap: Securing a Basic Shell
- **Source**: https://sloonz.github.io/posts/sandboxing-1/
- **Retrieved**: 2026-05-12

## Overview

Bubblewrap creates isolated filesystem namespaces for desktop applications. Unlike Flatpak (which combines sandboxing with package distribution), Bubblewrap focuses solely on "the sandboxing part" while allowing users to run their existing distribution applications in isolation.

## Core Approach: Empty Filesystem Namespace

The fundamental principle is that Bubblewrap creates an "empty" filesystem by default. The user must explicitly bind necessary system directories. Initial attempts fail because the sandbox lacks access to executables and their dependencies.

## Building a Functional Sandbox

**Step 1: Binding Essential Directories**

A basic shell requires read-only bindings for system libraries and executables:

```bash
bwrap --ro-bind /usr /usr --ro-bind /bin /bin \
      --ro-bind /lib /lib --ro-bind /lib64 /lib64 \
      --ro-bind /sbin /sbin --ro-bind /etc /etc \
      /usr/bin/zsh
```

**Step 2: Adding Virtual Filesystems**

Applications need `/dev`, `/proc`, and `/tmp`. Rather than exposing the host's `/dev` (which grants device access -- a security risk), use Bubblewrap's dedicated options:

```bash
bwrap --ro-bind /usr /usr --ro-bind /bin /bin \
      --ro-bind /lib /lib --ro-bind /lib64 /lib64 \
      --ro-bind /sbin /sbin --ro-bind /etc /etc \
      --proc /proc --dev /dev --tmpfs /tmp \
      /usr/bin/zsh
```

## Namespace Isolation

**Recommended Unsharing Options:**

- `--unshare-pid`: Isolates process visibility -- the compromised shell cannot see host processes
- `--clearenv`: Removes all inherited environment variables, preventing exposure of secrets like AWS credentials

```bash
bwrap --ro-bind /usr /usr --ro-bind /bin /bin \
      --ro-bind /lib /lib --ro-bind /lib64 /lib64 \
      --ro-bind /sbin /sbin --ro-bind /etc /etc \
      --proc /proc --dev /dev --tmpfs /tmp \
      --clearenv --unshare-pid /usr/bin/zsh
```

**Optional Unsharing:**

- `--unshare-uts`: Isolates hostname
- `--unshare-ipc`: Isolates System V IPC (rarely needed for desktop apps)
- `--unshare-user`: Creates separate user namespace (distribution-dependent behavior)

**Explicitly NOT Recommended:**

Network namespace unsharing prevents all network access unless intentionally configured, making it unsuitable for general-purpose shells.

## Filesystem Security Strategy

**Access Control:**

Use `--ro-bind` (read-only) for system directories to prevent malicious modifications. Compromised applications cannot alter `/etc` or system libraries, preventing persistence attacks.

**Home Directory Isolation:**

Instead of binding the actual home directory, create a temporary workspace:

```bash
mkdir ~/sandboxes/my-node-project
bwrap --ro-bind /usr /usr --ro-bind /bin /bin \
      --ro-bind /lib /lib --ro-bind /lib64 /lib64 \
      --ro-bind /sbin /sbin --ro-bind /etc /etc \
      --proc /proc --dev /dev --tmpfs /tmp \
      --clearenv --unshare-pid \
      --bind ~/sandboxes/my-node-project ~ \
      --chdir ~ /usr/bin/zsh
```

This allows npm packages or project files to be installed in the sandbox while protecting the real home directory.

**Configuration File Handling:**

Selectively bind configuration files as read-only:

```bash
--ro-bind ~/.zshrc ~/.zshrc --ro-bind ~/.config/nvim ~/.config/nvim
```

This prevents sandboxed processes from writing malicious payloads that execute in the non-sandboxed environment later.

## Isolation Results

After proper configuration, the sandboxed shell shows:

- Process list limited to sandbox contents (the shell, bwrap, and executed commands)
- Environment variables reduced to essential state
- No access to `/home` directory (absent explicit binding)
- Filesystem limited to mounted system directories

## Key Security Principle

"Any compromised program that is run in this shell session will be unable to access our personal data (absent any privilege escalation exploit given root access)" through proper filesystem isolation and namespace separation.
