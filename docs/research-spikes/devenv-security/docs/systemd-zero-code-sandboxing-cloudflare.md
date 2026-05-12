# Sandboxing in Linux with Zero Lines of Code (Cloudflare)
- **Source**: https://blog.cloudflare.com/sandboxing-in-linux-with-zero-lines-of-code/
- **Retrieved**: 2026-05-12

## Overview

The article discusses systemd as a "zero code seccomp" implementation, enabling operators to inject sandbox policies into processes without modifying source code.

## Key Systemd Directives

**SystemCallFilter=**
Defines which system calls a managed service can execute. The syntax uses a tilde (~) to deny specific calls. One can prohibit particular syscalls like uname without altering the application itself.

**SystemCallErrorNumber=**
Configures the kernel's response when a prohibited syscall is attempted. Rather than terminating the process with SIGSYS, it returns a specified error code, allowing the application to continue execution.

## Systemd-run for Ad-Hoc Sandboxing

The `systemd-run` command creates ephemeral service units with sandboxing properties:

```bash
systemd-run --user --pty --same-dir --wait --collect --service-type=exec \
  --property="SystemCallFilter=~uname" ./myos
```

This launches a process with the specified restrictions without permanent service configuration.

## Implicit Allowlist

Systemd automatically allows certain syscalls: "execve, exit, exit_group, getrlimit, rt_sigreturn, sigreturn" and time-related calls. This limitation exists because systemd itself must fork, call seccomp, and exec the target application.

## Limitations

A critical constraint: operators cannot explicitly block implicitly-allowed syscalls like execve through SystemCallFilter, restricting defenses against arbitrary code execution exploits.
