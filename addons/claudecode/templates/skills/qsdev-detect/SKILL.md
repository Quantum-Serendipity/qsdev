---
name: qsdev-detect
description: Detect project ecosystems, languages, package managers, and frameworks.
allowed-tools: Bash(qsdev *) Read Grep Glob
---

# qsdev detect

## Current Environment

!`ls -a`

!`qsdev info --json 2>/dev/null || echo "ERROR: 'qsdev info --json' exited with status $?. Any output above may be partial; do not treat missing data as empty."`

## Instructions

1. **Report detected ecosystems**: Identify ecosystems from the manifest and lock files in the project listing above (use Glob to find manifests in subdirectories), and note which ones the `qsdev info` output shows as already configured. Present each language or platform ecosystem:
   - Language name and detected version
   - Package manager in use
   - Key framework or build tool markers found

2. **Detection confidence**: Note what markers were used for detection (e.g., go.mod, package.json, Cargo.toml, pyproject.toml).

3. **Recommended modules**: Based on the detected ecosystems, list the qsdev ecosystem modules that would be enabled and what security configurations they bring.

4. **Missing ecosystems**: If there are project files suggesting an ecosystem that was not detected, note them and suggest running `qsdev devenv add-language <name>` to add support.
