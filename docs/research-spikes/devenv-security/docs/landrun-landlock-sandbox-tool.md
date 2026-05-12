# Landrun: Kernel-Level Process Sandboxing via Landlock
- **Source**: https://github.com/Zouuup/landrun
- **Retrieved**: 2026-05-12

## How It Works

Landrun leverages Linux Landlock, a kernel-native security module that enables unprivileged processes to sandbox themselves. It provides "kernel-level security using Landlock" with "fine-grained filesystem and network access controls" without requiring root privileges, containers, or complex security configurations.

## What Gets Sandboxed

Landrun restricts:
- **File system access** (read, write, execute permissions on specific paths)
- **Directory operations** (creation, removal, traversal)
- **TCP network activity** (port binding and connection restrictions)
- **Process capabilities** through Landlock's access control rights

## Command-Line Flags

Core access control options:
- `--ro <path>`: Read-only access
- `--rox <path>`: Read-only with execution
- `--rw <path>`: Read-write access
- `--rwx <path>`: Read-write with execution
- `--bind-tcp <port>`: Allow TCP binding
- `--connect-tcp <port>`: Allow TCP connections
- `--env <var>`: Pass environment variables (none by default)

Additional options:
- `--best-effort`: Graceful degradation on older kernels
- `--log-level`: Set verbosity (error, info, debug)
- `--unrestricted-network`: Disable network restrictions
- `--unrestricted-filesystem`: Disable filesystem restrictions
- `--add-exec`: Auto-add executing binary to `--rox`
- `--ldd`: Auto-add required libraries to `--rox`

## Kernel Requirements

- **Minimum**: Linux 5.13 with Landlock enabled
- **Network restrictions**: Linux 6.7+ (Landlock ABI v4)
- **IOCTL operations**: Linux 6.10+ (Landlock ABI v5)

## Usage Examples

**Basic directory listing with restrictions:**
```
landrun --rox /usr/bin --ro /lib,/lib64 /usr/bin/ls /path/to/dir
```

**Web server with selective network access:**
```
landrun --rox /usr/bin --ro /lib,/lib64,/var/www --rwx /var/log \
  --bind-tcp 80,443 /usr/bin/nginx
```

**With environment variables:**
```
landrun --rox /usr --ro /etc --env HOME --env PATH -- env
```

**Systemd integration example** (nginx service):
```ini
ExecStart=/usr/bin/landrun --rox /usr/bin,/usr/lib \
  --ro /etc/nginx,/etc/ssl --rwx /var/log/nginx \
  --bind-tcp 80,443 /usr/bin/nginx -g 'daemon off;'
```

## Key Security Features

When no restrictions are specified, it "applies maximum restrictions available for the current kernel version."
