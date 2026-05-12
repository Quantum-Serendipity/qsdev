# set-minimum-package-release-age — Open Source Age-Gating Tool

- **Source URL**: https://github.com/dehrenschwender/set-minimum-package-release-age
- **Retrieved**: 2026-05-12

## Purpose

This project provides bash scripts that configure package managers to enforce a minimum age requirement — defaulting to 7 days — before installing package releases. The mechanism reduces supply-chain attack risks by avoiding newly published versions that haven't had time to be scrutinized.

## Core Functionality

The tool operates through platform-specific scripts (`set_package_min_age_linux.sh` and `set_package_min_age_macos.sh`) that share a common library. For each supported package manager, the scripts:

1. Verify whether the current setting matches the target
2. Preserve existing configurations via backup
3. Write or update age-gate settings using manager-native formats
4. Apply exception rules where supported
5. Execute version checks for tools with documented minimum requirements
6. Validate configurations and compare against backups
7. Display a comprehensive results table

## Supported Package Managers

**Python:**
- `pip` (uses `uploaded-prior-to` timestamp in `~/.config/pip/pip.conf`)
- `uv` (native age gate with per-package exceptions; includes daily cron refresh)

**JavaScript:**
- `npm` (native age gate in `~/.npmrc`)
- `pnpm` (native gate with exclusion patterns in `~/.config/pnpm/rc` or macOS equivalent)
- `bun` (native gate with package excludes in `~/.bunfig.toml`)
- `yarn classic` (cache TTL workaround via `~/.yarnrc`)
- `yarn berry` (native gate with preapproved patterns in `~/.yarnrc.yml`)

## Usage Examples

```bash
bash set_package_min_age_linux.sh           # Default 7-day minimum
bash set_package_min_age_linux.sh 14        # Custom 14-day minimum
bash set_package_min_age_linux.sh 1d        # Duration format
bash set_package_min_age_linux.sh --remove  # Disable age-gating
```

Exception flags support tool-specific formats:
```bash
bash set_package_min_age_linux.sh 7 \
  --exception "uv:setuptools=false" \
  --exception "pnpm:@myorg/*" \
  --exception "yarn-berry:@internal/*"
```

Scoped removal targets specific tools:
```bash
bash set_package_min_age_linux.sh --remove-tool uv --remove-tool npm
```

## Key Design Characteristics

- **Idempotent:** Safe for repeated execution; skips unchanged settings
- **Backward-compatible:** Backs up configurations before modifications
- **Validated:** Includes syntax checks and a pure-Bash test suite
- **Version-aware:** Preflight checks confirm tool compatibility with native features

The project is licensed under MIT.
