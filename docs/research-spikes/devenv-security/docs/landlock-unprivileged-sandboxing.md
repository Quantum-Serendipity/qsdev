# Landlock: Unprivileged Sandboxing
- **Source**: https://landlock.io/
- **Retrieved**: 2026-05-12

## What It Is

Landlock is a Linux Security Module (LSM) designed to enable unprivileged sandboxing. The core purpose is: "restricting ambient rights (e.g. global filesystem access) for a set of processes."

## Key Capabilities

Landlock operates as a "stackable LSM," meaning it can be layered alongside existing system-wide access controls. "Any process, including unprivileged ones, can securely restrict themselves."

## Security Goals

The mechanism aims to "help mitigate the security impact of bugs or unexpected/malicious behaviors in user space applications" by enabling safe security sandboxes.

## ABI Version History

- **ABI v1 (Linux 5.13, 2021)**: Basic filesystem access control -- EXECUTE, WRITE_FILE, READ_FILE, READ_DIR, REMOVE_DIR, REMOVE_FILE, MAKE_* rights
- **ABI v2 (Linux 5.19)**: Added LANDLOCK_ACCESS_FS_REFER (file reparenting across directories)
- **ABI v3 (Linux 6.2)**: Added LANDLOCK_ACCESS_FS_TRUNCATE
- **ABI v4 (Linux 6.7)**: Network restrictions (TCP bind/connect)
- **ABI v5 (Linux 6.10)**: IOCTL operations

## Key Properties

- Restrictions can only increase, never decrease (monotonic)
- No root required
- Stackable with other LSMs (AppArmor, SELinux)
- Process restricts itself (self-sandboxing model)
