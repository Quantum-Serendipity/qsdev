# pnpm Build Script Security

- **Source**: https://deepwiki.com/pnpm/pnpm/3.5-build-script-security
- **Retrieved**: 2026-05-12

## Core Security Architecture

pnpm implements a multi-layered defense system against supply chain attacks through build script controls. The framework operates across several integrated components that validate and manage package execution during installation.

## Primary Security Mechanisms

**Build Allowlists (`allowBuilds`)**
The system uses explicit permission controls determining which packages may execute lifecycle scripts. During installation, pnpm creates policy functions that evaluate whether specific packages are permitted to run build operations. Blocked packages are tracked in an `ignoredBuilds` set, maintaining a clean separation between approved and unapproved code execution.

**Temporal Protection (`minimumReleaseAge`)**
This feature implements a time-based quarantine mechanism. Packages published within a specified window (measured in minutes) are filtered out during resolution unless explicitly exempted. When strict mode is enabled, additional publisher verification adds another validation layer.

**Trust Policies (`trustPolicy: 'no-downgrade'`)**
This setting prevents resolution to older versions already in lockfiles, defending against downgrade attacks where malicious actors force dependency regression to known-vulnerable releases.

## Security Integration Points

**Build Approval Workflow**
When unapproved packages with build scripts are encountered, pnpm either blocks execution or prompts users for approval. The `pnpm dlx` command includes specialized authorization prompts for temporary environments, allowing selective approval without persisting permissions.

**Configuration Persistence**
Security settings are stored in workspace state, enabling consistency checks across installation runs. The system detects when `allowBuilds` configurations change, flagging dependencies as potentially out-of-date to trigger re-evaluation.

**Installation Integrity**
Failed builds prevent package persistence in `package.json` and lockfiles, ensuring incomplete or malicious installations don't remain in project structures.

This architecture provides defense-in-depth without apparent sandboxing mechanisms — relying instead on explicit approval workflows and temporal delays.
