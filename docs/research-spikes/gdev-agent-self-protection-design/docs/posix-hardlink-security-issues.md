<!-- Source: https://michael.orlitzky.com/articles/posix_hardlink_heartache.xhtml -->
<!-- Retrieved: 2026-05-15 -->

# POSIX Hardlink Security Issues

## How Hardlinks Bypass Security

Hardlinks represent a fundamental architectural vulnerability in POSIX systems. Unlike symlinks that can be avoided using `O_NOFOLLOW`, hardlinks cannot be avoided by programs using filenames because "every name for a file is a hardlink." This creates an inescapable security gap in path-based security checks.

The core problem: "a special little chunk can become dangerous without changing its contents, inode, vnode, file table entry, or file descriptors." A regular user can transform a "safe" hardlink into a "dangerous" one after security checks occur but before operations execute.

## Available Protections

Linux offers two sysctl parameters:
- **fs.protected_hardlinks**: Prevents hardlink creation to files you cannot write to
- **fs.protected_symlinks**: Restricts symlinks in sticky world-writable directories

However, these are Linux-specific. Cross-platform applications cannot rely on them.

## The TOCTOU Vulnerability

Most filesystem operations involve two steps:
1. Checking whether action is safe
2. Executing the action

Between these steps, an attacker can create additional hardlinks, changing the `st_nlink` count from 1 to >1, making a previously "safe" file appear dangerous -- even when using file descriptors.

## Concrete Attack Example

1. A program obtains a file descriptor for "hardlink"
2. An attacker deletes the hardlink, reducing `st_nlink` from 2 to 1
3. The program checks `st_nlink` and finds it safe (value=1)
4. The program calls `fchmod()` on the descriptor, modifying the original file
5. The original file's permissions are changed despite the safety check

## Why File Descriptors Don't Solve This

File descriptors maintain references to the underlying inode, bypassing path traversal attacks. However, they cannot prevent the `st_nlink` count from changing externally.

## Recommended Mitigation

- Linux: Enable fs.protected_hardlinks (default on modern kernels)
- Application-level: Drop privileges before operating in user-writable directories
- For gdev: realpath resolves to the same canonical path regardless of which hardlink name is used, so hardlinks are NOT a bypass for path-based matching (unlike symlinks). The canonical path is the same inode content -- the concern is only if hardlinks point FROM protected locations TO unprotected locations.
