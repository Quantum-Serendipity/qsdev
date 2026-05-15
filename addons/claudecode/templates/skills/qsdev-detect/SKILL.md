---
name: qsdev-detect
description: Detect project ecosystems, languages, package managers, and frameworks.
allowed-tools: Bash(qsdev *) Read Grep Glob
---

# qsdev detect

## Current Environment

!`qsdev detect --json 2>/dev/null || echo '{"ecosystems": []}'`

## Instructions

1. **Report detected ecosystems**: Present each detected language or platform ecosystem:
   - Language name and detected version
   - Package manager in use
   - Key framework or build tool markers found

2. **Detection confidence**: Note what markers were used for detection (e.g., go.mod, package.json, Cargo.toml, pyproject.toml).

3. **Recommended modules**: Based on the detected ecosystems, list the qsdev ecosystem modules that would be enabled and what security configurations they bring.

4. **Missing ecosystems**: If there are project files suggesting an ecosystem that was not detected, note them and explain how to manually configure them.
