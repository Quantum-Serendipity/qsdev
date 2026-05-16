---
name: qsdev-add-dep
description: Add a dependency or package to the project safely. Use when the user asks to install, add, or include a package, library, tool, language, or service.
allowed-tools: Bash(qsdev *) Bash(pnpm *) Bash(npm *) Bash(cargo *) Bash(go *) Bash(pip *) Read
---

# Add Dependency

## Instructions

When the user asks to add a dependency, determine the type and use the appropriate method:

### 1. Ecosystem tool (linter, scanner, MCP server, formatter, etc.)

Check if it's a registered qsdev tool first:

```bash
qsdev list
```

If found, enable it:

```bash
qsdev enable <tool-name>
```

### 2. System package (imagemagick, ffmpeg, jq, htop, etc.)

```bash
qsdev devenv add-package <name>
```

Tell the user to run `direnv allow` or re-enter `devenv shell` to activate.

### 3. Project dependency (runtime library, framework, etc.)

Use the project's package manager within the devenv shell. The package guard hook validates safety automatically:

- **npm/pnpm**: `pnpm add <package>`
- **Rust**: `cargo add <crate>`
- **Go**: `go get <module>`
- **Python**: Add to pyproject.toml or requirements.txt

Commit the lockfile after adding.

### 4. Language or service

- **Add a language**: `qsdev devenv add-language <name>`
- **Add a service**: `qsdev devenv add-service <name>`
- **Remove a language**: `qsdev devenv remove-language <name>`
- **Remove a service**: `qsdev devenv remove-service <name>`

### 5. Removing packages

- **System package**: `qsdev devenv remove-package <name>`
- **Project dependency**: Use the package manager (e.g., `pnpm remove <package>`)

## Important

- Never tell the user to edit devenv.nix or any .nix file directly.
- Never use `nix-env -i`, `nix profile install`, or `apt install`.
- Never use `npm install` or `pip install` outside the devenv shell.
- If a package install is blocked by the package guard, explain why and suggest a safe alternative.
