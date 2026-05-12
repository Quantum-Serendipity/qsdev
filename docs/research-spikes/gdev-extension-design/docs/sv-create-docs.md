<!-- Source: https://svelte.dev/docs/cli/sv-create -->
<!-- Retrieved: 2026-05-12 -->

# Svelte CLI `sv create` Command Documentation

## Purpose
The `sv create` command initializes a new SvelteKit project with customizable configurations for templates, type checking, and additional tooling.

## Basic Usage
```
npx sv create [options] [path]
```

## Template Options
Three project templates are available:
- **minimal**: Basic scaffolding for new applications
- **demo**: Sample application featuring a word guessing game that functions without JavaScript
- **library**: Setup for Svelte library development using `svelte-package`

## Type Checking Configuration
The `--types` flag determines typechecking approach:
- `ts`: Defaults to TypeScript files with `lang="ts"` in Svelte components
- `jsdoc`: Uses JSDoc syntax for type annotations

The `--no-types` flag disables typechecking entirely (discouraged).

## Package Manager Selection
Use `--install <package-manager>` to specify dependency installation with npm, pnpm, yarn, bun, or deno. The `--no-install` flag skips dependency installation.

## Add-ons Integration
The `--add` flag enables adding tools during project creation, such as eslint and prettier. The `--no-add-ons` flag suppresses the interactive add-ons prompt.

## Special Features
- **`--from-playground`**: Converts a playground URL into a complete SvelteKit project
- **`--no-dir-check`**: Bypasses validation of target directory emptiness
