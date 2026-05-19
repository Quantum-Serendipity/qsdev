# qsdev Reference

## CLI Commands

### `qsdev init`
Initialize project with detection, wizard, and generation

### `qsdev init --update`
Update generated files to latest templates

### `qsdev init --mode join`
Join an existing qsdev-managed project

### `qsdev devenv doctor`
Check system prerequisites and project health

### `qsdev devenv setup`
Install missing prerequisites

### `qsdev enable <tool>`
Enable a tool (updates configs, adds to shared files)

### `qsdev disable <tool>`
Disable a tool (cleans up all owned files)

### `qsdev status`
Show enabled/disabled tools and configuration state

### `qsdev list`
Show all available tools with categories

### `qsdev check`
Validate configuration for CI enforcement

### `qsdev check --format json`
Machine-readable check output

### `qsdev check --audit-level medium`
Set minimum severity that fails CI

### `qsdev config migrate`
Migrate .qsdev.yaml to latest schema version

## Security Policy

- The package guard hook validates all package installs for vulnerabilities and age
- Use `qsdev enable <tool>` to add ecosystem tools (run `qsdev list` to see all)
- Use `qsdev devenv add-package <name>` to add system packages
- Use `qsdev devenv add-language <name>` to add language ecosystems
- Use `qsdev devenv add-service <name>` to add services (databases, caches)
- Never edit devenv.nix or .nix files directly — use qsdev commands
- Run `qsdev devenv doctor` after configuration changes

## Common Workflows

### Add a project dependency
Use the project's package manager within the devenv shell:
- `pnpm add <package>` (npm/pnpm projects)
- `cargo add <crate>` (Rust projects)
- Add to pyproject.toml/requirements.txt (Python projects)
- `go get <module>` (Go projects)
The package guard hook validates safety automatically. Commit the lockfile after.

### Add a system tool
`qsdev devenv add-package <name>` — then run `direnv allow` to activate.

### Enable a security/AI tool
`qsdev enable <tool>` — run `qsdev list` to see available tools.

## Troubleshooting

### qsdev commands not found
Install qsdev: see project README for installation instructions.

### devenv not activated
Run `direnv allow` in the project root, then `devenv shell`.

### Permission denied on tool operations
Check `.claude/settings.json` deny rules. Use `qsdev check --deny-rules` to validate.

