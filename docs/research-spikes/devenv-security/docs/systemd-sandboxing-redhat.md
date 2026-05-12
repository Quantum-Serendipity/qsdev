# Mastering systemd: Securing and Sandboxing Applications
- **Source**: https://www.redhat.com/en/blog/mastering-systemd
- **Retrieved**: 2026-05-12

## Available Security Directives

### Common Directives

**PrivateTmp**: Creates a file system namespace under `/tmp/systemd-private-*-[unit name]-*/tmp` rather than a shared `/tmp` or `/var/tmp`, eliminating file prediction vulnerabilities.

**ProtectSystem**: Makes system directories read-only. The `strict` option also restricts `/dev`, `/proc`, and `/sys`.

**ProtectHome**: Makes `/home`, `/root`, and `/run/user` appear empty, with options for read-only or writeable ephemeral filesystems.

**NoNewPrivileges**: Prevents privilege escalation for the service and child processes.

**CapabilityBoundingSet**: Accepts a whitelist and blacklist of privileged capabilities for the unit, enabling fine-grained privilege control.

**ProtectDevices**: Creates a private `/dev` namespace with only pseudo-devices, disabling `CAP_MKNOD`.

### Advanced Directives

- **ProtectKernelTunables**: Disables `/proc` and `/sys` modification
- **ProtectKernelModules**: Prohibits module loading/unloading
- **ProtectControlGroups**: Disables write access to cgroup filesystem
- **SystemCallFilter**: Lets you whitelist and blacklist individual syscalls using groups like `@system-service`
- **MemoryDenyWriteExecute**: Disables simultaneous write-execute memory mapping
- **DynamicUser**: Dynamically creates transient users for applications
- **RestrictNamespaces**: Restricts specific Linux namespaces
- **PrivateMounts**: Runs services in private mount namespaces

## Practical Example Configuration

Hardening `httpd.service` with a drop-in:

```ini
[Service]
ProtectSystem=strict
ProtectHome=yes
PrivateDevices=yes
ProtectKernelTunables=yes
ProtectKernelModules=yes
ProtectControlGroups=yes
SystemCallFilter=@system-service
SystemCallErrorNumber=EPERM
NoNewPrivileges=yes
PrivateTmp=yes
```

## Analysis Tool

The `systemd-analyze security [unit]` command provides a quick snapshot of how the system is leveraging systemd's sandboxing, displaying exposure ratings and active security settings.
