<!-- Source: https://github.com/direnv/direnv/issues/445 -->
<!-- Retrieved: 2026-05-12 -->

# .envrc File Permissions Security Issue (#445)

## Core Security Concern

The issue raised by pmav99 questions whether direnv should permit `.envrc` files to be writable by users other than the owner. This creates a potential security vulnerability.

## The Vulnerability: TOCTOU Attack

The primary risk is a **Time-of-Check-Time-of-Use (TOCTOU)** attack. The scenario demonstrates this attack pattern:

1. A developer inspects the `.envrc` file contents and verifies it appears safe
2. Before executing `direnv allow`, a malicious actor modifies the file
3. The developer unknowingly executes the compromised environment configuration

**Example from the issue:** A user with world-writable directory permissions (chmod 777) could have their `.envrc` altered by an unprivileged account between inspection and execution.

## Security Recommendation

The proposal suggests implementing a **permissions hierarchy**:
- **Default behavior:** Only permit `.envrc` files writable exclusively by the file owner
- **Optional relaxation:** Allow more permissive settings (group-writable, etc.) only through explicit user configuration

This approach balances security by default with flexibility for collaborative environments where team members legitimately need shared access to environment configurations.

## Current Behavior Problem

"direnv is happy to allow `.envrc` to be writeable by other users," creating an implicit security risk without requiring users to consciously accept the trade-off.
