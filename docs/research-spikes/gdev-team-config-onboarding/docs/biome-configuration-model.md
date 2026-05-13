<!-- Source: https://biomejs.dev/guides/configure-biome/ -->
<!-- Retrieved: 2026-05-12 -->

# Biome Configuration Model: Complete Overview

## Configuration File Format

Biome uses `biome.json` or `biome.jsonc` files (with JSONC supporting comments). These files are typically placed at a project's root, alongside `package.json`.

The configuration organizes around three core tools: formatter, linter, and assist -- all enabled by default. Each tool's settings can be disabled individually using the `<tool>.enabled` field.

## Configuration Structure

Settings follow a hierarchical pattern:

- **General options**: Applied across all languages within a tool
- **Language-specific options**: Placed under `<language>.<tool>` fields, allowing overrides

For example, a `formatter.lineWidth` setting applies globally, while `javascript.formatter.lineWidth` overrides it specifically for JavaScript files.

As the documentation notes: "Biome refers to all variants of the JavaScript language as `javascript`. This includes TypeScript, JSX and TSX."

## File Resolution Strategy

Biome searches for configuration files in this order:

1. `biome.json`
2. `biome.jsonc`
3. `.biome.json`
4. `.biome.jsonc`

Discovery occurs recursively, starting from:
- The current working directory
- Parent folders (recursively)
- The home directory (OS-dependent locations)

If no configuration is found, Biome applies default settings.

## Nested Configuration Files (Monorepo Support)

Biome supports multiple `biome.json` files in nested directories. Each folder resolves to the nearest configuration file in its hierarchy. This enables different teams within a monorepo to maintain distinct linting and formatting standards without conflicts.

## File Inclusion and Exclusion

Three mechanisms control which files process:

**CLI specification**: Direct file/folder listing in commands
```
biome format file1.js src/
```

**Configuration-based control**: Using `files.includes` with glob patterns
```json
{
  "files": {
    "includes": ["src/**/*.js", "test/**/*.js", "!**/*.min.js"]
  }
}
```

**VCS integration**: Respecting version control ignore files

## Protected Files

Biome automatically ignores specific lock files from analysis:
- `composer.lock`
- `npm-shrinkwrap.json`
- `package-lock.json`
- `yarn.lock`

## Well-Known File Handling

Biome treats certain configuration files specially, parsing them as JSON with customized parser options. Files like `tsconfig.json`, `jsconfig.json`, and `.babelrc.json` allow comments and trailing commas, while stricter files like `.eslintrc.json` permit only comments.

## Tool-Specific Refinement

Beyond global `files.includes`, individual tools support their own `<tool>.includes` fields. The documentation clarifies: "Any file or folder that doesn't match `files.includes` is excluded from use by any of Biome's tools."

This creates a filtering hierarchy where global inclusion rules take precedence over tool-specific ones.
