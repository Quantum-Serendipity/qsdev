---
source: Multiple web searches (flox.dev/docs, flox.dev/blog)
retrieved: 2026-03-20
---

# Flox manifest.toml Format and Services

## manifest.toml Overview

Flox uses a single TOML file (manifest.toml) that declares the software you want in your environment along with environment variables and activation scripts. The manifest is a declarative specification for the environment. TOML was chosen for its readability — even basic LLMs can read and understand Flox environment manifests.

## Main Sections

### [install] — Packages
Defines the version of packages needed for the environment.

### [vars] — Environment Variables
Used to set environment variables. Note: these variables may not reference one another.

### [hook] — Activation Scripts
Contains the on-activate field where you can write inline shell scripts that run when entering the environment.

### [profile] — Shell-specific Configuration
For shell-specific configuration scripts.

### [services] — Background Processes
Services are defined in the [services] section. Services have a simple schema:
- A command to run to start the service
- Any vars you want set specifically for the service
- Whether the service spawns a background process

## Example: Service with Environment Variables

```toml
[services.database]
command = "postgres start"

[services.database.vars]
PGUSER = "myuser"
PGPASSWORD = "super-secret"
PGDATABASE = "mydb"
PGPORT = "9001"
```

## Example: Hook with On-Activate Script

```toml
[hook]
on-activate = """
echo ""
echo "Start the server with 'npm start'"
echo ""
"""
```

## Environment Sharing Mechanisms

1. **Version control**: Commit the .flox directory (containing manifest.toml) to your project's repo. Teammates git clone and `flox activate`.
2. **FloxHub**: Push environments with `flox push`, pull with `flox pull`, or activate remotely with `flox activate --remote`.
3. **Centrally managed environments**: Multiple projects or systems consume a shared environment that is versioned with generations.

## Use Cases for Centrally Managed Environments
- Base environments for projects of similar tech stacks
- Reproducing issues on specific systems
- Quickly sharing tools across team members

## Composition
Environments can be composed and layered — activated on top of each other, combining toolsets while keeping them discrete.
