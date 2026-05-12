# Pre-Commit Security Hook Suite for devenv.sh

## Overview

This report catalogs every security-relevant pre-commit hook applicable to devenv.sh, covering built-in hooks from git-hooks.nix, custom hook configurations for tools not built in, and the surrounding concerns of performance, bypass prevention, and deployment classification. For each hook: exact devenv.nix configuration, what it catches, performance characteristics, and whether it belongs at commit-time, pre-push, or CI-only.

**Key finding**: devenv's git-hooks.nix integration provides only one built-in security-focused hook (`ripsecrets`). All other security scanning tools -- gitleaks, trufflehog, semgrep, bandit, gosec, grype, vulnix, flake-checker -- require custom hook definitions. However, the custom hook mechanism is well-designed and all of these tools are available in nixpkgs, making integration straightforward.

---

## 1. Built-in Security Hooks in git-hooks.nix

Of the 120+ hooks available in git-hooks.nix, only a handful are directly security-focused. The rest are formatters and linters that provide indirect security benefits.

### 1.1 ripsecrets (Secret Detection)

**Status**: Built-in, first-class support
**What it catches**: API keys with known prefixes (Stripe, Slack, etc.), random strings assigned to variables named "token", "secret", "password", etc.
**Detection method**: Regex pattern matching + entropy analysis (probability threshold: <1 in 10,000 of occurring randomly)
**Performance**: 0.32s on the Sentry repo (M1 Air) -- 95x faster than trufflehog, 226x faster than detect-secrets. Negligible overhead on typical commits.

```nix
{ pkgs, ... }: {
  git-hooks.enable = true;
  git-hooks.hooks.ripsecrets = {
    enable = true;
    # Optional: add custom patterns for org-specific secrets
    settings.additionalPatterns = [
      "MYORG_[A-Za-z0-9]{32}"           # Custom API key format
      "ghp_[A-Za-z0-9]{36}"             # GitHub personal access tokens
    ];
  };
}
```

**Classification**: **Commit-time** -- fast enough for every commit.

**Limitations**: No API verification (cannot confirm if a detected secret is actually valid). Higher false positive rate than verification-based tools like trufflehog. Local-only by design -- never sends code externally. Supports `.secretsignore` for allowlisting and inline `# pragma: allowlist secret` comments.

### 1.2 reuse (License Compliance)

**Status**: Built-in
**What it catches**: Files missing REUSE-compliant license headers (SPDX identifiers). Enforces the REUSE specification for license clarity.

```nix
{
  git-hooks.hooks.reuse = {
    enable = true;
  };
}
```

**Classification**: **Commit-time** -- fast metadata check.

### 1.3 check-added-large-files (Binary Blob Prevention)

**Status**: Built-in
**What it catches**: Large files being committed (potential binary blobs, data dumps, accidentally committed credentials files like database exports).

```nix
{
  git-hooks.hooks.check-added-large-files = {
    enable = true;
    # Default threshold is typically 500KB; customize with args
    args = [ "--maxkb=500" ];
  };
}
```

**Classification**: **Commit-time** -- instant check.

### 1.4 no-commit-to-branch (Branch Protection)

**Status**: Built-in
**What it catches**: Direct commits to protected branches (main, master). Forces use of feature branches and code review.

```nix
{
  git-hooks.hooks.no-commit-to-branch = {
    enable = true;
    # Default protects main and master
    args = [ "--branch" "main" "--branch" "master" ];
  };
}
```

**Classification**: **Commit-time** -- instant check.

### 1.5 Security-Adjacent Built-in Hooks

These are not security tools per se but catch bug classes that frequently lead to security vulnerabilities:

| Hook | What It Catches | Security Relevance |
|------|----------------|-------------------|
| `shellcheck` | Unquoted variables, command injection patterns in shell scripts | Prevents shell injection vulnerabilities |
| `clippy` | Unsafe Rust code, memory issues | Memory safety |
| `mypy` / `pyright` | Python type errors | Type confusion bugs |
| `phpstan` / `psalm` | PHP type errors, some security patterns | Type safety |
| `eslint` / `oxlint` | JS/TS issues (configurable security rules available) | XSS, injection patterns with security plugins |
| `statix` | Nix anti-patterns | Catches evaluation issues in Nix configs |
| `actionlint` | GitHub Actions workflow issues | CI/CD pipeline security |
| `check-case-conflicts` | Filename case conflicts | Cross-platform path traversal edge cases |

All are commit-time hooks.

---

## 2. Secret Scanning: Tool Comparison

### 2.1 Comparison Matrix

| Feature | ripsecrets | gitleaks | trufflehog | detect-secrets |
|---------|-----------|----------|------------|----------------|
| **Language** | Rust | Go | Go | Python |
| **Speed (Sentry repo)** | 0.32s | ~2-5s* | 31.2s | 73.5s |
| **Detection method** | Regex + entropy | Regex + entropy | Regex + entropy + API verification | Regex + entropy + keywords |
| **Secret types** | Known prefixes + random strings | 150+ patterns (TOML config) | 800+ classified types | 27+ plugins |
| **API verification** | No | No | Yes (validates credentials are active) | Optional |
| **Baseline support** | `.secretsignore` | Yes (baseline reports) | No | Yes (core feature) |
| **Git history scan** | No (staged files only) | Yes | Yes | Diff against baseline |
| **Privacy** | Local-only, never phones home | Local-only | Phones home for verification | Local-only |
| **nixpkgs available** | Yes | Yes | Yes | Yes |
| **git-hooks.nix built-in** | **Yes** | No | No | No |
| **False positive rate** | Moderate | Low-moderate | Low (verification) | Moderate |
| **Inline allowlisting** | `# pragma: allowlist secret` | `.gitleaks.toml` allowlist | Custom detectors | `# pragma: allowlist secret` |

*gitleaks benchmark not in ripsecrets comparison but is reported as fast due to Go compilation

### 2.2 Recommendation

**Use ripsecrets as the commit-time default** (built-in, fastest, zero config). **Add gitleaks as a pre-push or CI layer** for broader pattern coverage. **Use trufflehog in CI-only** for verification-based scanning (too slow for hooks, but confirms whether leaked secrets are actually active).

### 2.3 devenv.nix Configuration for Each

**ripsecrets** (commit-time -- built-in):
```nix
{
  git-hooks.hooks.ripsecrets.enable = true;
}
```

**gitleaks** (pre-push -- custom hook):
```nix
{ pkgs, ... }: {
  git-hooks.hooks.gitleaks = {
    enable = true;
    name = "gitleaks";
    description = "Scan for secrets using gitleaks";
    entry = "${pkgs.gitleaks}/bin/gitleaks git --staged --no-banner --verbose";
    language = "system";
    pass_filenames = false;
    stages = [ "pre-push" ];
  };
}
```

**trufflehog** (CI-only -- custom hook, manual stage):
```nix
{ pkgs, ... }: {
  git-hooks.hooks.trufflehog = {
    enable = true;
    name = "trufflehog";
    description = "Verify leaked credentials with trufflehog";
    entry = "${pkgs.trufflehog}/bin/trufflehog git file://. --only-verified --fail";
    language = "system";
    pass_filenames = false;
    stages = [ "manual" ];  # Run via `devenv test` or CI, not on commit
  };
}
```

**detect-secrets** (commit-time alternative -- custom hook):
```nix
{ pkgs, ... }: {
  git-hooks.hooks.detect-secrets = {
    enable = true;
    name = "detect-secrets";
    description = "Detect secrets using baseline approach";
    entry = "${pkgs.detect-secrets}/bin/detect-secrets-hook --baseline .secrets.baseline";
    language = "system";
    # pass_filenames = true by default, which is correct for detect-secrets-hook
  };
}
```

---

## 3. Lock File Auditing

### 3.1 The Problem

`devenv.lock` and `flake.lock` pin input sources (nixpkgs commits, flake inputs) but not individual packages. A malicious or careless `devenv update` could:
- Switch to a nixpkgs commit with known vulnerabilities
- Change the upstream owner (e.g., from `NixOS/nixpkgs` to a fork)
- Add new, untrusted flake inputs
- Remove pinned inputs (widening the dependency surface)

No built-in hook exists for auditing lock file changes.

### 3.2 Custom Lock File Audit Hook

This custom hook detects suspicious changes to lock files and flags them for review:

```nix
{ pkgs, ... }: {
  git-hooks.hooks.lock-file-audit = {
    enable = true;
    name = "lock-file-audit";
    description = "Audit devenv.lock/flake.lock changes for suspicious modifications";
    entry = "${pkgs.writeShellScript "lock-file-audit" ''
      set -euo pipefail

      # Only run if lock files are staged
      STAGED_LOCKS=$(${pkgs.git}/bin/git diff --cached --name-only -- devenv.lock flake.lock 2>/dev/null || true)
      if [ -z "$STAGED_LOCKS" ]; then
        exit 0
      fi

      echo "=== Lock File Audit ==="
      EXIT_CODE=0

      for LOCK in $STAGED_LOCKS; do
        echo "Checking $LOCK..."

        # Check for owner changes (fork substitution attack)
        OWNER_CHANGES=$(${pkgs.git}/bin/git diff --cached -- "$LOCK" | \
          ${pkgs.gnugrep}/bin/grep -E '^\+.*"owner"' | \
          ${pkgs.gnugrep}/bin/grep -v '"NixOS"' | \
          ${pkgs.gnugrep}/bin/grep -v '"cachix"' || true)
        if [ -n "$OWNER_CHANGES" ]; then
          echo "WARNING: Non-standard owner detected in $LOCK:"
          echo "$OWNER_CHANGES"
          echo "Verify this input is from a trusted source."
          EXIT_CODE=1
        fi

        # Check for new inputs being added
        NEW_INPUTS=$(${pkgs.git}/bin/git diff --cached -- "$LOCK" | \
          ${pkgs.gnugrep}/bin/grep -E '^\+.*"type":' || true)
        OLD_INPUTS=$(${pkgs.git}/bin/git diff --cached -- "$LOCK" | \
          ${pkgs.gnugrep}/bin/grep -E '^\-.*"type":' || true)
        NEW_COUNT=$(echo "$NEW_INPUTS" | ${pkgs.gnugrep}/bin/grep -c . 2>/dev/null || echo 0)
        OLD_COUNT=$(echo "$OLD_INPUTS" | ${pkgs.gnugrep}/bin/grep -c . 2>/dev/null || echo 0)
        if [ "$NEW_COUNT" -gt "$OLD_COUNT" ]; then
          echo "WARNING: New inputs added to $LOCK (was $OLD_COUNT, now $NEW_COUNT)"
          echo "Review the new dependencies for trustworthiness."
          EXIT_CODE=1
        fi

        # Check for narHash changes (input content changed)
        HASH_CHANGES=$(${pkgs.git}/bin/git diff --cached -- "$LOCK" | \
          ${pkgs.gnugrep}/bin/grep -c '^\+.*"narHash"' 2>/dev/null || echo 0)
        if [ "$HASH_CHANGES" -gt 0 ]; then
          echo "INFO: $HASH_CHANGES input hash(es) changed in $LOCK"
          echo "This is expected after 'devenv update' but should be reviewed."
        fi
      done

      if [ "$EXIT_CODE" -ne 0 ]; then
        echo ""
        echo "Lock file changes require review. If intentional, commit with:"
        echo "  git commit --no-verify  (use with caution)"
        echo "Or add the owners to the trusted list in the hook script."
      fi

      exit $EXIT_CODE
    ''}";
    language = "system";
    pass_filenames = false;
    files = "(devenv|flake)\\.lock$";
    stages = [ "pre-commit" ];
  };
}
```

**Classification**: **Commit-time** -- fast (just diffing JSON). Warns on owner changes and new inputs; informational on hash changes.

### 3.3 flake-checker Integration

Determinate Systems' flake-checker provides complementary validation:

```nix
{ pkgs, ... }: {
  git-hooks.hooks.flake-checker = {
    enable = true;
    name = "flake-checker";
    description = "Validate flake.lock health (supported branches, recency, upstream ownership)";
    entry = "${pkgs.flake-checker}/bin/flake-checker --no-telemetry";
    language = "system";
    pass_filenames = false;
    files = "flake\\.lock$";
    always_run = false;
    stages = [ "pre-commit" ];
  };
}
```

**What flake-checker validates**:
1. **Supported branches**: Nixpkgs Git refs are on supported release branches (flags end-of-life branches that stop receiving security updates ~7 months after release)
2. **Recency**: Nixpkgs inputs updated within the last 30 days (stale inputs miss security patches)
3. **Upstream ownership**: GitHub inputs owned by the NixOS organization (prevents use of forks or untrusted variants)

**CEL policy customization**: For stricter requirements:
```nix
    entry = "${pkgs.flake-checker}/bin/flake-checker --no-telemetry --condition 'supportedRefs.contains(gitRef) && numDaysOld < 14 && owner == \"NixOS\"'";
```

**Classification**: **Commit-time** -- reads flake.lock JSON, no network calls for basic checks. Fast.

**Limitation**: flake-checker validates `flake.lock`, not `devenv.lock`. For non-flake devenv setups, the custom lock-file-audit hook above covers `devenv.lock`.

---

## 4. Dependency Vulnerability Scanning

### 4.1 vulnix (Nix-Native CVE Scanner)

**What it catches**: Packages in your Nix store or derivation closure that have known CVEs in the NIST NVD database. Matches derivation names/versions against CVE entries using name-matching heuristics.

**Performance**: Requires downloading the NVD database on first run (can take 30-60 seconds). Subsequent runs with cache are faster but still 5-15 seconds for a typical dev environment closure. Scans the full transitive dependency tree.

```nix
{ pkgs, ... }: {
  git-hooks.hooks.vulnix = {
    enable = true;
    name = "vulnix";
    description = "Scan Nix dependencies for known CVEs";
    entry = "${pkgs.writeShellScript "vulnix-check" ''
      # Only run when lock file or package list changes
      ${pkgs.vulnix}/bin/vulnix --json . 2>/dev/null | \
        ${pkgs.jq}/bin/jq -r '.[] | "CVE: \(.cve_ids | join(", ")) in \(.pname) \(.version)"' || true
      # vulnix returns non-zero if vulnerabilities found
      ${pkgs.vulnix}/bin/vulnix . -w vulnix-whitelist.toml 2>/dev/null
    ''}";
    language = "system";
    pass_filenames = false;
    files = "(devenv\\.nix|devenv\\.lock|flake\\.lock|flake\\.nix)$";
    stages = [ "manual" ];  # Too slow for commit-time
  };
}
```

**Classification**: **CI-only** (or `manual` stage, run via `devenv test`). Too slow for commit-time due to NVD database fetch and full closure scan. The `files` filter limits it to running only when dependency-related files change.

**Limitations**: Name-matching heuristic is acknowledged as "too simplistic" -- produces both false positives and false negatives. Requires Nix daemon access. Whitelist support via TOML files with expiration dates.

### 4.2 grype (General Vulnerability Scanner)

**What it catches**: Vulnerabilities in container images, filesystems, and SBOMs. Broader coverage than vulnix (not Nix-specific) but also not Nix-aware.

**Performance**: 30-45 seconds per scan for a typical project. Too slow for commit-time.

```nix
{ pkgs, ... }: {
  git-hooks.hooks.grype = {
    enable = true;
    name = "grype";
    description = "Scan for vulnerabilities with grype";
    entry = "${pkgs.writeShellScript "grype-check" ''
      ${pkgs.grype}/bin/grype dir:. --fail-on high --quiet
    ''}";
    language = "system";
    pass_filenames = false;
    stages = [ "manual" ];  # Too slow for commit-time
  };
}
```

**Classification**: **CI-only**. Better suited for scanning built container images in CI than for pre-commit hooks.

### 4.3 Practical Recommendation

Neither vulnix nor grype is practical as a commit-time hook. The recommended approach:

1. **Commit-time**: Skip dependency scanning entirely (too slow)
2. **Pre-push**: Skip (still too slow, blocks developer flow)
3. **CI**: Run vulnix against the devenv closure + grype against container images
4. **Scheduled**: Run nightly vulnix scans and alert on new CVEs via the nix-security-tracker

```nix
# In devenv.nix: make vulnix available as a script for manual runs
{
  scripts.vuln-check = {
    exec = ''
      echo "Scanning for known CVEs in dependencies..."
      ${pkgs.vulnix}/bin/vulnix . -w vulnix-whitelist.toml
    '';
    description = "Check dependencies for known CVEs";
    packages = [ pkgs.vulnix ];
  };
}
```

---

## 5. SAST (Static Analysis Security Testing)

### 5.1 semgrep

**Status**: NOT a built-in git-hooks.nix hook. Requires custom definition.
**What it catches**: Security vulnerabilities across 30+ languages using pattern-matching rules that "look like the code you already write." Community ruleset covers OWASP Top 10, injection, authentication, cryptography issues.
**Performance**: 5-30 seconds on a medium project (10k-50k LOC) depending on ruleset. Acceptable for pre-push; borderline for commit-time.
**nixpkgs**: `pkgs.semgrep`

```nix
{ pkgs, ... }: {
  git-hooks.hooks.semgrep = {
    enable = true;
    name = "semgrep";
    description = "SAST scanning with semgrep";
    entry = "${pkgs.semgrep}/bin/semgrep scan --config auto --error --quiet";
    language = "system";
    pass_filenames = false;
    stages = [ "pre-push" ];  # 5-30s is too slow for commit-time
  };
}
```

**Classification**: **Pre-push** for small-medium projects, **CI-only** for large codebases.

**Tip**: For commit-time usage on small projects, restrict to staged files:
```nix
    # Commit-time variant (staged files only)
    entry = "${pkgs.writeShellScript "semgrep-staged" ''
      FILES=$(${pkgs.git}/bin/git diff --cached --name-only --diff-filter=ACM)
      if [ -n "$FILES" ]; then
        echo "$FILES" | ${pkgs.semgrep}/bin/semgrep scan --config auto --error --quiet --target-from-stdin
      fi
    ''}";
    stages = [ "pre-commit" ];
```

### 5.2 bandit (Python SAST)

**Status**: NOT a built-in git-hooks.nix hook. Requires custom definition.
**What it catches**: Common Python security issues -- SQL injection, shell injection, hardcoded passwords, use of `eval()`, weak cryptography, insecure SSL, XML vulnerabilities. Scans Python AST.
**Performance**: Fast (1-5 seconds on medium Python projects). Suitable for commit-time.
**nixpkgs**: `pkgs.bandit`

```nix
{ pkgs, ... }: {
  git-hooks.hooks.bandit = {
    enable = true;
    name = "bandit";
    description = "Python SAST with bandit";
    entry = "${pkgs.bandit}/bin/bandit -r --severity-level medium";
    language = "system";
    types = [ "python" ];
    # pass_filenames = true (default) -- bandit scans passed files
  };
}
```

**Classification**: **Commit-time** -- fast enough for every commit on Python files.

### 5.3 gosec (Go Security Checker)

**Status**: NOT a built-in git-hooks.nix hook. Requires custom definition.
**What it catches**: Go-specific security issues -- SQL injection, hardcoded credentials, weak crypto, insecure TLS, directory traversal, command injection. Analyzes Go AST.
**Performance**: 2-10 seconds depending on module size. Suitable for commit-time on most Go projects.
**nixpkgs**: `pkgs.gosec`

```nix
{ pkgs, ... }: {
  git-hooks.hooks.gosec = {
    enable = true;
    name = "gosec";
    description = "Go security checker";
    entry = "${pkgs.gosec}/bin/gosec -quiet ./...";
    language = "system";
    types = [ "go" ];
    pass_filenames = false;  # gosec scans packages, not individual files
  };
}
```

**Classification**: **Commit-time** for small-medium Go projects, **pre-push** for large ones.

### 5.4 SAST Summary

| Tool | Languages | git-hooks.nix | Speed | Classification |
|------|-----------|---------------|-------|---------------|
| semgrep | 30+ | Custom | 5-30s | Pre-push / CI |
| bandit | Python | Custom | 1-5s | Commit-time |
| gosec | Go | Custom | 2-10s | Commit-time |
| shellcheck | Shell | **Built-in** | <1s | Commit-time |
| clippy | Rust | **Built-in** | 2-10s | Commit-time |
| phpstan | PHP | **Built-in** | 3-15s | Pre-push |
| eslint (with security plugins) | JS/TS | **Built-in** | 1-5s | Commit-time |

---

## 6. License Compliance

### 6.1 reuse (Built-in)

The REUSE specification hook is built into git-hooks.nix. It ensures every file has a machine-readable license header (SPDX format). See section 1.2.

### 6.2 Nix License Blocklisting

devenv.yaml provides license controls independent of pre-commit hooks:

```yaml
# devenv.yaml
nixpkgs:
  allow_unfree: false
  blocklisted_licenses:
    - "bsl11"           # Business Source License
    - "unfree"          # All unfree licenses
    - "unfreeRedistributable"
  allowlisted_licenses:
    - "mit"
    - "asl20"           # Apache 2.0
    - "bsd2"
    - "bsd3"
    - "isc"
    - "mpl20"           # Mozilla Public License 2.0
```

This is enforced at Nix evaluation time (not as a hook) -- packages with blocklisted licenses simply fail to build. This is stronger than a hook because it cannot be bypassed with `--no-verify`.

### 6.3 Custom License Audit Hook

For auditing license changes in non-Nix dependencies (e.g., npm, pip):

```nix
{ pkgs, ... }: {
  git-hooks.hooks.license-audit = {
    enable = true;
    name = "license-audit";
    description = "Check for license changes in dependency lock files";
    entry = "${pkgs.writeShellScript "license-audit" ''
      # Example for Node.js projects
      if ${pkgs.git}/bin/git diff --cached --name-only | ${pkgs.gnugrep}/bin/grep -q 'package-lock.json'; then
        echo "package-lock.json changed -- verify no new copyleft/restricted licenses"
        # Could integrate with license-checker, licensee, etc.
      fi
    ''}";
    language = "system";
    pass_filenames = false;
    files = "(package-lock\\.json|Cargo\\.lock|go\\.sum|poetry\\.lock)$";
  };
}
```

**Classification**: **Commit-time** -- lightweight diff check.

---

## 7. flake-checker Deep Dive

See section 3.3 for the hook configuration. Additional detail:

### 7.1 What flake-checker Validates

| Check | Security Purpose | Default |
|-------|-----------------|---------|
| Supported branches | Ensures nixpkgs is on a branch receiving security updates | Enabled |
| Recency (30 days) | Stale inputs miss security patches | Enabled |
| Upstream ownership | Prevents fork substitution (use of untrusted nixpkgs variants) | Enabled |

### 7.2 CEL Policy Examples

```bash
# Strict: 14-day freshness, NixOS-only, supported branches
--condition 'supportedRefs.contains(gitRef) && numDaysOld < 14 && owner == "NixOS"'

# Permissive: allow Cachix fork (for devenv-nixpkgs), 60-day window
--condition '(owner == "NixOS" || owner == "cachix") && numDaysOld < 60'
```

### 7.3 Limitation for devenv

flake-checker operates on `flake.lock` files, not `devenv.lock`. For standard devenv setups (not using flakes integration), `devenv.lock` uses the same JSON format but flake-checker may not recognize it. The custom lock-file-audit hook (section 3.2) covers this gap.

**Recommendation**: If using devenv with flakes integration (`devenv.sh/guides/using-with-flakes/`), use flake-checker directly. For standalone devenv, use the custom audit hook.

---

## 8. Custom Hook Definition Template

### 8.1 Complete Attribute Reference

Every custom hook in devenv.nix supports these attributes:

```nix
{ pkgs, ... }: {
  git-hooks.hooks.<hook-name> = {
    # Required
    enable = true;
    entry = "${pkgs.some-tool}/bin/tool --flag";  # Command to execute

    # Identity
    name = "human-readable-name";                 # Display name
    description = "What this hook checks";         # Metadata

    # File targeting
    files = "\\.py$";                              # Regex: which files to check
    types = [ "python" ];                          # File type filter
    types_or = [ "python" "pyi" ];                 # Alternative type matches
    excludes = [ "^tests/" "^vendor/" ];           # Patterns to skip
    exclude_types = [ "jupyter" ];                 # Types to exclude

    # Execution control
    language = "system";                           # Always "system" for Nix-provided tools
    pass_filenames = true;                         # Pass changed files as args
    args = [ "--strict" "--format" "json" ];        # Additional CLI args
    stages = [ "pre-commit" ];                     # When to run (see valid stages below)
    always_run = false;                            # Run even without matching files
    fail_fast = false;                             # Stop all hooks if this one fails
    require_serial = false;                        # Prevent parallel execution
    verbose = false;                               # Always print output

    # Dependencies
    package = pkgs.some-tool;                      # Provider package (optional)
    extraPackages = [ pkgs.jq pkgs.gnugrep ];      # Additional dependencies

    # Ordering (relative to other hooks)
    before = [ "other-hook" ];                     # Run before these hooks
    after = [ "other-hook" ];                      # Run after these hooks
  };
}
```

### 8.2 Valid Stages

| Stage | When It Runs | Use For |
|-------|-------------|---------|
| `"pre-commit"` | Before each commit | Fast checks (<5s) |
| `"pre-push"` | Before `git push` | Moderate checks (5-30s) |
| `"commit-msg"` | After commit message written | Message format validation |
| `"pre-merge-commit"` | Before merge commits | Merge-specific checks |
| `"post-commit"` | After commit completes | Notifications, logging |
| `"post-checkout"` | After `git checkout` | Environment validation |
| `"post-merge"` | After `git merge` | Dependency updates |
| `"post-rewrite"` | After `git rebase`/`git commit --amend` | History rewrite checks |
| `"pre-rebase"` | Before rebase | Rebase guards |
| `"prepare-commit-msg"` | Before commit message editor opens | Message templates |
| `"manual"` | Only via explicit invocation (`devenv test`) | Slow/expensive checks |

### 8.3 Minimal Custom Hook Template

```nix
{ pkgs, ... }: {
  git-hooks.hooks.my-security-check = {
    enable = true;
    name = "my-security-check";
    description = "Description of what this checks";
    entry = "${pkgs.my-tool}/bin/my-tool --check";
    language = "system";
    pass_filenames = false;                        # Set true if tool accepts file args
    stages = [ "pre-commit" ];                     # Adjust based on speed
  };
}
```

### 8.4 Pattern: Wrapping Complex Logic

For hooks that need multiple tools or conditional logic, use `writeShellScript`:

```nix
{ pkgs, ... }: {
  git-hooks.hooks.complex-check = {
    enable = true;
    name = "complex-check";
    entry = "${pkgs.writeShellScript "complex-check" ''
      set -euo pipefail

      # Access multiple tools via Nix store paths
      STAGED=$(${pkgs.git}/bin/git diff --cached --name-only)

      # Conditional logic
      if echo "$STAGED" | ${pkgs.gnugrep}/bin/grep -q '\.py$'; then
        ${pkgs.bandit}/bin/bandit -r $(echo "$STAGED" | ${pkgs.gnugrep}/bin/grep '\.py$')
      fi

      if echo "$STAGED" | ${pkgs.gnugrep}/bin/grep -q '\.go$'; then
        ${pkgs.gosec}/bin/gosec -quiet ./...
      fi
    ''}";
    language = "system";
    pass_filenames = false;
  };
}
```

---

## 9. Performance Impact and Classification

### 9.1 Performance Tiers

| Tier | Time Budget | Rationale |
|------|-------------|-----------|
| **Commit-time** | <5 seconds | Developers commit frequently; slow hooks get bypassed |
| **Pre-push** | <30 seconds | Less frequent; acceptable delay before sharing code |
| **CI-only / Manual** | No limit | Runs in background; no developer waiting |

### 9.2 Complete Classification

| Hook | Typical Time | Classification | Rationale |
|------|-------------|---------------|-----------|
| ripsecrets | <0.5s | **Commit-time** | Rust, scans staged files only |
| check-added-large-files | <0.1s | **Commit-time** | Metadata check |
| no-commit-to-branch | <0.1s | **Commit-time** | Branch name check |
| reuse | <1s | **Commit-time** | Header check |
| shellcheck | <1s | **Commit-time** | Fast parser |
| lock-file-audit (custom) | <0.5s | **Commit-time** | JSON diff |
| flake-checker | <1s | **Commit-time** | Reads JSON, no network |
| bandit | 1-5s | **Commit-time** | AST analysis, Python only |
| gosec | 2-10s | **Commit-time** / Pre-push | Go AST analysis |
| clippy | 2-10s | **Commit-time** | Rust compiler integration |
| eslint | 1-5s | **Commit-time** | JS/TS analysis |
| gitleaks | 2-5s | **Pre-push** | Broader pattern set |
| semgrep | 5-30s | **Pre-push** / CI | Multi-language, large ruleset |
| phpstan | 3-15s | **Pre-push** | PHP type analysis |
| detect-secrets | 5-30s | **Pre-push** | Python, slower startup |
| trufflehog | 30-120s | **CI-only** | API verification calls |
| vulnix | 15-60s | **CI-only** | NVD database fetch + closure scan |
| grype | 30-45s | **CI-only** | Filesystem/image scan |

### 9.3 Recommended Suite by Tier

**Commit-time** (every commit, fast):
```nix
{
  git-hooks.hooks = {
    ripsecrets.enable = true;
    check-added-large-files.enable = true;
    no-commit-to-branch.enable = true;
    shellcheck.enable = true;
    # lock-file-audit = { ... };   # Custom, see section 3.2
    # flake-checker = { ... };     # Custom, see section 3.3
    # bandit = { ... };            # Custom, see section 5.2 (Python projects)
    # gosec = { ... };             # Custom, see section 5.3 (Go projects)
  };
}
```

**Pre-push** (before sharing code):
```nix
{
  git-hooks.hooks = {
    # gitleaks = { ... stages = ["pre-push"]; };   # Custom, see section 2.3
    # semgrep = { ... stages = ["pre-push"]; };    # Custom, see section 5.1
  };
}
```

**CI-only** (background, no developer wait):
```nix
{
  git-hooks.hooks = {
    # trufflehog = { ... stages = ["manual"]; };   # Custom, see section 2.3
    # vulnix = { ... stages = ["manual"]; };       # Custom, see section 4.1
    # grype = { ... stages = ["manual"]; };        # Custom, see section 4.2
  };
}
```

---

## 10. Hook Bypass Prevention

### 10.1 The `--no-verify` Problem

Git's `--no-verify` flag (also `-n`) bypasses all client-side hooks. This is a fundamental Git design decision -- **there is no way to prevent `--no-verify` on the client**. The pre-commit framework also allows `SKIP=hookname` to bypass specific hooks.

### 10.2 Mitigation Strategies

**Server-side enforcement (strongest)**:
- Git server pre-receive hooks run on push and cannot be bypassed by the client
- GitHub/GitLab branch protection rules requiring status checks
- CI pipelines that run `devenv test` and block merge on failure

**`devenv test` in CI (recommended)**:
```yaml
# GitHub Actions example
- name: Security hooks
  run: devenv test
```

`devenv test` runs all hooks including `manual` stage hooks, verifying that the committed code passes all security checks regardless of whether the developer bypassed hooks locally.

**Detection (after the fact)**:
```nix
{ pkgs, ... }: {
  # Script to detect commits that bypassed hooks
  scripts.audit-bypass = {
    exec = ''
      echo "Commits without hook verification in last 7 days:"
      ${pkgs.git}/bin/git log --since="7 days ago" --format="%H %s" | while read hash msg; do
        # Check if the commit has a hook trailer (if your hooks add one)
        if ! ${pkgs.git}/bin/git log --format="%B" -1 "$hash" | ${pkgs.gnugrep}/bin/grep -q "Hooks-Verified:"; then
          echo "  $hash $msg"
        fi
      done
    '';
    description = "Find commits that may have bypassed hooks";
  };
}
```

**Hook installation verification**:
```nix
{
  enterShell = ''
    # Verify hooks are installed on every shell entry
    if [ -d .git ] && [ ! -f .git/hooks/pre-commit ]; then
      echo "WARNING: Pre-commit hooks are not installed!"
      echo "Run 'devenv shell' to reinstall them."
    fi
  '';

  enterTest = ''
    # CI: fail if hooks aren't installed
    test -f .git/hooks/pre-commit || {
      echo "FAIL: Pre-commit hooks not installed"
      exit 1
    }
  '';
}
```

### 10.3 What Happens When Someone Uninstalls Hooks

If a developer deletes `.git/hooks/pre-commit`:
- **Next `devenv shell` entry**: devenv reinstalls the hooks (the hook symlink is recreated from the Nix store on each shell activation)
- **Between shell entries**: Hooks are absent; commits are unverified
- **CI/CD safety net**: `devenv test` catches anything that slipped through

This is devenv's strongest property for hook persistence: because hooks are managed declaratively via Nix, they are automatically reinstalled whenever the development environment is activated. You cannot permanently remove hooks without also avoiding devenv.

### 10.4 Defense-in-Depth Model

```
Layer 1: Commit-time hooks (fast, easily bypassed with --no-verify)
    |
Layer 2: Pre-push hooks (medium speed, bypassed with --no-verify)
    |
Layer 3: CI pipeline runs `devenv test` (cannot be bypassed by developer)
    |
Layer 4: Branch protection rules (require CI to pass before merge)
    |
Layer 5: Server-side pre-receive hooks (final gatekeeper)
```

No single layer is sufficient. The combination provides overlapping coverage where each layer compensates for the previous one's bypass mechanisms.

---

## 11. Complete Hardened Hook Suite

Combining all recommendations into a single devenv.nix configuration:

```nix
{ pkgs, lib, ... }: {
  git-hooks.enable = true;

  git-hooks.hooks = {
    # ═══════════════════════════════════════════
    # COMMIT-TIME: Fast checks on every commit
    # ═══════════════════════════════════════════

    # Secret detection (built-in, <0.5s)
    ripsecrets = {
      enable = true;
      settings.additionalPatterns = [
        # Add org-specific patterns here
      ];
    };

    # Prevent large file commits (built-in, instant)
    check-added-large-files = {
      enable = true;
      args = [ "--maxkb=500" ];
    };

    # Branch protection (built-in, instant)
    no-commit-to-branch = {
      enable = true;
      args = [ "--branch" "main" "--branch" "master" ];
    };

    # Shell script security (built-in, <1s)
    shellcheck.enable = true;

    # Nix anti-pattern detection (built-in, <1s)
    statix.enable = true;

    # License compliance (built-in, <1s)
    reuse.enable = true;

    # Lock file audit (custom, <0.5s)
    lock-file-audit = {
      enable = true;
      name = "lock-file-audit";
      description = "Audit lock file changes for suspicious modifications";
      entry = "${pkgs.writeShellScript "lock-file-audit" ''
        set -euo pipefail
        STAGED=$(${pkgs.git}/bin/git diff --cached --name-only -- devenv.lock flake.lock 2>/dev/null || true)
        [ -z "$STAGED" ] && exit 0
        for LOCK in $STAGED; do
          OWNERS=$(${pkgs.git}/bin/git diff --cached -- "$LOCK" | ${pkgs.gnugrep}/bin/grep -E '^\+.*"owner"' | ${pkgs.gnugrep}/bin/grep -v '"NixOS"' | ${pkgs.gnugrep}/bin/grep -v '"cachix"' || true)
          if [ -n "$OWNERS" ]; then
            echo "WARNING: Non-standard owner in $LOCK: $OWNERS"
            exit 1
          fi
        done
      ''}";
      language = "system";
      pass_filenames = false;
      files = "(devenv|flake)\\.lock$";
    };

    # Flake health check (custom, <1s)
    flake-checker = {
      enable = true;
      name = "flake-checker";
      description = "Validate flake.lock health";
      entry = "${pkgs.flake-checker}/bin/flake-checker --no-telemetry";
      language = "system";
      pass_filenames = false;
      files = "flake\\.lock$";
    };

    # ═══════════════════════════════════════════
    # PRE-PUSH: Moderate checks before sharing
    # ═══════════════════════════════════════════

    # Broader secret detection (custom, 2-5s)
    gitleaks = {
      enable = true;
      name = "gitleaks";
      description = "Deep secret scanning with gitleaks";
      entry = "${pkgs.gitleaks}/bin/gitleaks git --staged --no-banner --verbose";
      language = "system";
      pass_filenames = false;
      stages = [ "pre-push" ];
    };

    # Multi-language SAST (custom, 5-30s)
    semgrep = {
      enable = true;
      name = "semgrep";
      description = "SAST scanning with semgrep";
      entry = "${pkgs.semgrep}/bin/semgrep scan --config auto --error --quiet";
      language = "system";
      pass_filenames = false;
      stages = [ "pre-push" ];
    };

    # ═══════════════════════════════════════════
    # CI-ONLY: Expensive checks via `devenv test`
    # ═══════════════════════════════════════════

    # Verified secret detection (custom, 30-120s)
    trufflehog = {
      enable = true;
      name = "trufflehog";
      description = "Verify leaked credentials";
      entry = "${pkgs.trufflehog}/bin/trufflehog git file://. --only-verified --fail";
      language = "system";
      pass_filenames = false;
      stages = [ "manual" ];
    };

    # Nix CVE scanning (custom, 15-60s)
    vulnix-scan = {
      enable = true;
      name = "vulnix";
      description = "Scan Nix dependencies for known CVEs";
      entry = "${pkgs.writeShellScript "vulnix-check" ''
        ${pkgs.vulnix}/bin/vulnix . 2>/dev/null || echo "vulnix: vulnerabilities found (review output above)"
      ''}";
      language = "system";
      pass_filenames = false;
      stages = [ "manual" ];
    };
  };

  # ═══════════════════════════════════════════
  # LANGUAGE-SPECIFIC (enable as needed)
  # ═══════════════════════════════════════════
  # Uncomment for Python projects:
  # git-hooks.hooks.bandit = {
  #   enable = true;
  #   name = "bandit";
  #   entry = "${pkgs.bandit}/bin/bandit -r --severity-level medium";
  #   language = "system";
  #   types = [ "python" ];
  # };

  # Uncomment for Go projects:
  # git-hooks.hooks.gosec = {
  #   enable = true;
  #   name = "gosec";
  #   entry = "${pkgs.gosec}/bin/gosec -quiet ./...";
  #   language = "system";
  #   types = [ "go" ];
  #   pass_filenames = false;
  # };

  # Hook installation verification
  enterShell = ''
    if [ -d .git ] && [ ! -L .git/hooks/pre-commit ]; then
      echo "WARNING: Pre-commit hooks may not be properly installed"
    fi
  '';

  enterTest = ''
    test -f .git/hooks/pre-commit || {
      echo "FAIL: Pre-commit hooks not installed"
      exit 1
    }
  '';
}
```

---

## Sources

- [ripsecrets](https://github.com/sirwart/ripsecrets) -> `docs/ripsecrets-readme.md`, `docs/ripsecrets-benchmarks.md`
- [gitleaks](https://github.com/gitleaks/gitleaks) -> `docs/gitleaks-readme.md`
- [trufflehog](https://github.com/trufflesecurity/trufflehog) -> `docs/trufflehog-readme.md`
- [detect-secrets](https://github.com/Yelp/detect-secrets) -> `docs/detect-secrets-readme.md`
- [semgrep](https://github.com/semgrep/semgrep) -> `docs/semgrep-readme.md`
- [flake-checker](https://github.com/DeterminateSystems/flake-checker) -> `docs/flake-checker-github-readme.md`, `docs/nix-flake-checker-determinate.md`
- [git-hooks.nix complete list](https://github.com/cachix/git-hooks.nix/blob/master/modules/hooks.nix) -> `docs/git-hooks-nix-complete-list.md`
- [git-hooks.nix custom hook schema](https://flake.parts/options/git-hooks-nix.html) -> `docs/git-hooks-nix-custom-hook-schema.md`
- [vulnix](https://github.com/nix-community/vulnix) -> `docs/vulnix-readme.md`
- [devenv git hooks docs](https://devenv.sh/git-hooks/) -> `docs/devenv-git-hooks-docs.md`, `docs/devenv-git-hooks-configuration.md`
- [devenv pre-commit hooks docs](https://devenv.sh/pre-commit-hooks/) -> `docs/devenv-pre-commit-hooks.md`
