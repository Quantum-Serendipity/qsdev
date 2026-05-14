---
source: https://raw.githubusercontent.com/devcontainers/spec/main/docs/specs/secrets-support.md
retrieved: 2026-03-20
type: specification
---

# Dev Containers Secrets Support Specification

## Overview

The specification introduces first-class secrets handling for dev containers, addressing the need to securely manage sensitive data like API keys and passwords.

## Key Features

**Core Capabilities:**
- Secure mechanism to pass secrets into dev containers
- Dynamic secret updates without container rebuilds
- Treatment similar to `remoteEnv` and `containerEnv` for accessibility
- Secure handling throughout the lifecycle

## Declaration and Format

Secrets are **not stored in `devcontainer.json`**. Instead, they're passed through external mechanisms. The specification suggests a JSON file approach as a simple implementation:

```json
{
	"API_KEY": "value",
	"NUGET_CONFIG": "configuration content",
	"PASSWORD": "credentials"
}
```

## Technical Requirements

Supporting tools (like the dev container CLI) must:

1. **Accept secrets via secure channels** - files, credential managers, keychains, or vault services
2. **Make secrets available** - similar to environment variables for consumption
3. **Handle securely** - prevent logging, exposure, or accidental disclosure

## Implementation Status

This feature has been implemented with references to:
- GitHub issue #219 (specification discussion)
- CLI PR #493 (implementation)

The specification intentionally avoids prescribing storage locations, allowing flexibility across different platforms and security infrastructures.
