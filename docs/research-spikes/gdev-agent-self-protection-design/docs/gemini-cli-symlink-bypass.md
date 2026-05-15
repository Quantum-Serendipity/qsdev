<!-- Source: https://github.com/google-gemini/gemini-cli/issues/1121 -->
<!-- Retrieved: 2026-05-15 -->

# Symlink Bypass Vulnerability in Gemini CLI - Technical Analysis

## Attack Vector

The vulnerability exploits a **Time-of-Check to Time-of-Use (TOCTOU)** weakness in path validation. As described in the issue:

> "path validation is performed on the user-provided path, not the _real_, canonical path of the file being accessed"

**Concrete attack scenario:**
1. Workspace restricted to `/home/user/project`
2. Attacker creates: `ln -s /etc/ /home/user/project/etc-link`
3. Request to read `/home/user/project/etc-link/passwd`
4. Validation passes (path starts with allowed prefix)
5. Node.js `fs` functions follow the symlink to `/etc/passwd`, exposing sensitive data

## Vulnerable Code Pattern

The issue doesn't provide specific code snippets, but identifies the fundamental flaw: security checks occur on user-provided paths before symlink resolution. The validation logic checks the string path against a whitelist, while the actual file operations follow symlinks to unrestricted locations.

## Proposed Fix

The recommended solution involves two critical steps:

1. **Path Resolution:** "the provided path must be fully resolved to its absolute, canonical form using `fs.promises.realpath()`"

2. **Sequential Validation:** "All security and workspace boundary checks **must** be performed on this resolved path"

This ensures operations target the true canonical location before validation approval.

## Affected Components

The vulnerability impacts all filesystem tools:
- `read_file`
- `write_file`
- `replace`
- `list_directory`
- `glob`

The recommended approach centralizes path canonicalization in a shared utility function accessible to all affected tools.
