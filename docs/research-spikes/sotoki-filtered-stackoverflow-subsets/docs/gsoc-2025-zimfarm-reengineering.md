---
source: https://elfkuzco.github.io/gsoc-2025/
retrieved: 2026-05-14
type: blog-post
---

# Google Summer of Code 2025: ZIMFarm Reengineering Project

## Project Overview
Comprehensive modernization of ZIMFarm, "a semi-decentralized software solution to build ZIM files efficiently" through web scraping, packaging, and repository uploads.

## Architecture Changes

**Major Library Replacements:**
- Flask replaced with FastAPI for the REST API backend
- Marshmallow exchanged for Pydantic for data validation
- Paramiko and subprocess calls superseded by the Cryptography library
- JavaScript upgraded to TypeScript
- Vue 2 migrated to Vue 3

**Dependency Management:**
Hatch introduced as a dependency manager to pin all dependencies to a specific version.

## Problems Addressed

The reengineering tackled several existing fragilities:
- Query parameters with special characters crashed the server
- Missing required fields (like email addresses) caused user creation failures
- UI buttons remained active without pending changes
- ZIM metadata values lacked proper escaping
- Subprocess-based authentication verification was inelegant

## Proposed Solutions

**Security Enhancements:**
- Proper escaping of flag inputs when constructing offliner commands
- Support for ECDSA and Ed25519 SSH key algorithms

**Operational Improvements:**
- Context-based task filtering ensuring compatibility between tasks and workers
- ISO 639-3 standardization for language codes
- Modern type-checking tools (Pyright, Ruff) enforcement
- Enhanced UI responsiveness for mobile devices

The project encompassed over 100 pull requests, establishing API version 2.
