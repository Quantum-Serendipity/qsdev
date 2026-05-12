# Validation Findings

Cross-spike validation performed 2026-05-12 by four parallel sub-agents. Each verified research findings against current source code, tool availability, and API behavior.

## gdev Architecture — All Confirmed

- **Addon[T] generic struct**: Unchanged at `addons/addon.go:15-19`. Two-phase lifecycle (customization → lockdown) intact.
- **10 extension points**: All confirmed present and unchanged.
- **Bootstrap addon**: Wizard step infrastructure confirmed. Now uses **charmbracelet/huh** for TUI (validates our design recommendation). New env detection helpers: `SkipInContainer()`, `SkipIfNoGUI()`.
- **_template addon**: Still exists as authoring guide.
- **Cobra**: Still used (v1.10.2).
- **No breaking changes** to addon API.

### New Since Research
- `gocache-sftp` sub-addon (SFTP build cache storage)
- `ServiceWithSource` interface extending `Service`
- `AddPreStartHookType[T]()` convenience generic
- `lib/sys` (daemon mgmt, D-Bus, process isolation) and `lib/httpx` utility packages
- Go toolchain bumped to 1.26.2 in go.work

## Supply Chain Security — Confirmed with Minor Updates

- **"92% within 24h" claim**: Confirmed. Source: PyPI 2025 Year in Review. Precise: "92% of PyPI malware *removed* within 24h." PyPI-specific metric, not universal.
- **6-step implementation roadmap**: Confirmed sound (age-gating → install scripts → lock files → scanning → Harden-Runner → private registry).
- **Per-ecosystem findings**: npm most threatened, Go most secure — both confirmed.
- **All tools maintained** except **Phylum** (acquired by Veracode Jan 2025 → use Socket.dev instead).

### Config File Updates Needed
- pnpm: Use `pnpm-workspace.yaml` format (minutes) not `.npmrc` format (milliseconds) to avoid unit confusion
- npm: `min-release-age` has no internal package exemption (open issue)
- Version-specific configs should include minimum tool version comments

## Claude Code Guardrails — Confirmed with Issues to Fix

- **PreToolUse hook mechanism**: Confirmed accurate (fire-before-permissions, exit codes, `updatedInput`, managed settings).
- **OSV.dev API**: Endpoints and response format confirmed valid.
- **Deny rule glob patterns**: All 48 rules covering 15+ package managers confirmed correct.
- **settings.json schema**: Confirmed correct.

### Issues to Fix in Implementation
1. **Bash CVSS parsing broken**: `unified-architecture.md` bash script's jq expression `split("/")[0] | tonumber` fails silently on OSV's CVSS vector strings. **Use Python reference script instead.**
2. **`if` field `||` syntax unverified**: `reference-hook-settings.json` uses compound `||` in `if` field. May not work. **Use individual hook entries or omit `if` field.**
3. **npm age check `modified` field imprecise**: Uses metadata modification time, not version creation time. **Fix: look up `time[dist-tags.latest]` instead of `time.modified`.**
4. **OSV versionless queries return ALL vulnerabilities**: High false-positive rate for well-maintained packages. **Fix: query for latest version first, then check that specific version.**
5. **Fail-closed default**: Blocks all installs during API outages. **Make configurable per deployment profile.**

## devenv Security — Confirmed with Critical Updates

- **All nix.conf hardening settings**: Confirmed valid (sandbox, require-sigs, trusted-users, etc.).
- **Trust model**: All 8 trust dependencies accurately described.
- **Hook configuration syntax**: Confirmed valid.
- **Tool availability**: ripsecrets, gitleaks, trufflehog, detect-secrets, semgrep, grype, vulnix, flake-checker — all still maintained.

### Critical Updates for Implementation
1. **devenv 2.0: git-hooks input must be explicitly declared** in devenv.yaml. Without `inputs.git-hooks.url: github:cachix/git-hooks.nix`, all hook configs are silently ignored. Highest-priority fix.
2. **prek replaces pre-commit** as default hook runner in devenv 1.11+. Same functionality, different binary name.
3. **NixOS 25.11 approaching EOL** (2026-06-30). Pin to `nixos-26.05` once released (late May 2026). For now, use `nixos-25.11` with a TODO.
4. **Native activation (`devenv hook`)**: Replaces direnv for 2.0+ users. Eliminates `.envrc`. Per-directory trust via `devenv allow`/`devenv revoke`. Trust gap (devenv.nix changes not re-checked) persists in both mechanisms.
5. **SecretSpec 0.7-0.8**: New providers (AWS SM, Vault/OpenBao, Pass), declarative generation, `as_path` attribute. Research guidance architecturally correct but provider list outdated.
6. **Trivy supply chain compromise (March 2026)**: Malicious versions published, GH Actions tags force-pushed to malware, Docker images poisoned. Real-world validation of trust model warnings. If trivy is referenced, add prominent warning.
7. **Betterleaks**: Drop-in gitleaks replacement by same team (Aikido-backed). Not urgent but worth tracking.
8. **New devenv.yaml options**: `strict_ports` (errors on port conflicts), new `nixpkgs.*` options (`allow_non_source`, `allowlisted_licenses`, `cuda_support`, `rocm_support`).
9. **Native Rust process manager**: Replaces process-compose as default in 2.0. New syntax for dependencies, restart policies, `linux.capabilities`.

## Impact on Implementation Plan

### Must-have changes from validation:
- Generate explicit `git-hooks` input in devenv.yaml (not optional)
- Use Python PreToolUse hook script, not bash version
- Fix npm age check to use version creation time
- Fix OSV queries to include version when available
- Replace Phylum references with Socket.dev
- Use individual hook entries instead of `||` compound `if` syntax
- Add nixpkgs pin version as a configurable with sensible default

### Should-have improvements:
- Support both direnv (.envrc) and native activation (devenv hook)
- Updated SecretSpec provider list
- Trivy compromise warning in generated security documentation
- pnpm-workspace.yaml as canonical config path (not .npmrc)
- Version check comments in generated package manager configs
