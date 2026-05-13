# Configuration Drift Detection for gdev

## Problem

After `gdev init` generates config files, multiple forms of drift can occur:
1. Someone manually edits a machine-owned file (accidental or intentional)
2. gdev releases a new version with updated defaults, but the project hasn't run `gdev update`
3. New tools become available that didn't exist when `gdev init` was run
4. Pre-commit hooks are uninstalled or corrupted
5. Lock files become stale relative to their source manifests

Detecting and reporting these drifts is essential for maintaining security posture over time.

## Existing Foundation

gdev's migration strategy design (from the extension design spike) already establishes the core drift detection mechanism:

**SHA256 hash tracking:** Each addon persists a `GeneratedState` in gdev's config tracking the hash of every generated file at creation time. Comparing current file hash against stored hash detects modifications.

**File categories:**
- **Machine-owned** (devenv.yaml, .envrc): modification = unexpected drift
- **Human-edited** (devenv.nix, CLAUDE.md, settings.json): modification = expected, but generated sections should be intact
- **Exclusive** (per-tool configs like .semgrep.yml, .gitleaks.toml): modification = user customization, track but don't alarm

## Drift Categories

### Category 1: Unauthorized File Modification

**Detection:** Compare current SHA256 of machine-owned files against stored hash.

```
State file: .gdev/state.yaml
  files:
    devenv.yaml:
      hash: "sha256:abc123..."
      generated_at: "2026-05-12T14:30:00Z"
      gdev_version: "1.2.0"
      category: "machine-owned"
```

**Reporting:**
- `current`: hash matches stored hash
- `modified`: hash differs from stored hash
- `missing`: file expected but absent
- `unknown`: file exists but no stored hash (generated before hash tracking)

**Severity:**
- Machine-owned modified without `gdev update` -> warning (someone broke config)
- Human-edited with intact section markers -> info (expected)
- Human-edited with missing section markers -> warning (generated sections may be lost)
- Missing file -> error (defense may be inactive)

### Category 2: Version Drift

**Detection:** Compare `gdev_version` in state file against running gdev binary version.

```
Project initialized with gdev 1.1.0
Running gdev 1.2.0
Changes in 1.2.0:
  - New defense layer: license-compliance
  - Updated Semgrep default rules
  - New pre-commit hook: ripsecrets
```

**Implementation:**
- gdev binary embeds a changelog of config-affecting changes per version
- On `gdev status`, compare project's `gdev_version` against binary version
- Report new tools available, updated defaults, deprecated settings
- Suggest `gdev update` with preview of changes

### Category 3: Tool Availability Drift

**Detection:** Compare currently enabled tools against the full tool registry for the detected ecosystems.

```
Detected ecosystems: npm, python, go
Available tools not enabled:
  - license-compliance (available since gdev 1.2.0)
  - container-security (Dockerfile detected since last scan)
  - secretspec (devenv 2.0 feature, newly available)
```

**Implementation:**
- Detection engine re-runs on `gdev status` to check for new project files
- Tool registry knows which tools are applicable per ecosystem
- Delta between "applicable tools" and "enabled tools" is the availability drift

### Category 4: Section Marker Integrity

**Detection:** For human-edited files with section markers (CLAUDE.md, devenv.nix), verify markers are intact.

```
CLAUDE.md section markers:
  [✓] <!-- BEGIN GENERATED SECTION --> present
  [✓] <!-- END GENERATED SECTION --> present
  [✓] <!-- gdev:semgrep --> ... <!-- /gdev:semgrep --> intact
  [✗] <!-- gdev:gitleaks --> marker found but <!-- /gdev:gitleaks --> missing
```

**Implementation:**
- Parse files for expected marker pairs
- Report missing/broken markers
- Suggest `gdev update --repair-markers` to restore

### Category 5: Lock File Drift

**Detection:** Compare lock file modification time against source manifest modification time.

```
package.json: modified 2026-05-12T10:00:00Z
package-lock.json: modified 2026-05-10T08:00:00Z
Status: STALE — lock file older than manifest
```

**Implementation:**
- For each detected ecosystem, check manifest-to-lock-file freshness
- `valid`: lock file newer than or equal to manifest
- `stale`: manifest modified after lock file
- `missing`: manifest exists but no lock file
- `corrupt`: lock file fails integrity check (e.g., `npm ci` would fail)

### Category 6: Pre-Commit Hook Drift

**Detection:** Verify pre-commit hooks are installed and functional.

```
Pre-commit hooks status:
  [✓] .git/hooks/pre-commit exists
  [✓] Hook runner: prek (devenv 1.11+)
  [✓] Hooks configured: 5 (semgrep, gitleaks, ripsecrets, commitlint, shellcheck)
  [✗] Hook not in .git/hooks: commit-msg (commitlint)
```

**Implementation:**
- Check `.git/hooks/` for expected hook files
- Verify hook files reference the expected runner (prek/pre-commit)
- Compare configured hooks against installed hooks
- Detect `--no-verify` usage patterns in git log (optional, privacy-sensitive)

## Drift Detection Engine Design

### When Drift Detection Runs

1. **`gdev status`** (always): Full drift scan, results included in posture report
2. **`gdev update --dry-run`** (on demand): Show what `gdev update` would change
3. **Shell hook** (optional): Lightweight check on `cd` into project directory
4. **CI** (automated): `gdev status --json` captures drift state as artifact

### Detection Performance

| Check | Speed | Network Required |
|-------|-------|-----------------|
| File hash comparison | < 10ms per file | No |
| Version comparison | < 1ms | No |
| Tool availability | < 50ms | No |
| Section marker parsing | < 10ms per file | No |
| Lock file freshness | < 5ms per file | No |
| Pre-commit hook status | < 20ms | No |

All drift detection is local-only and completes in under 100ms for a typical project. This is a key advantage over vulnerability scanning which requires network access.

### State Storage

```yaml
# .gdev/state.yaml
version: "1.2.0"
initialized_at: "2026-05-12T14:30:00Z"
last_update: "2026-05-12T14:30:00Z"
profile: "consulting-default"

tools_enabled:
  - semgrep
  - gitleaks
  - ripsecrets
  - attach-guard
  - agent-postmortem
  - version-sentinel
  - semble
  - context7
  - commitlint
  - changelog

files:
  devenv.yaml:
    hash: "sha256:abc123..."
    generated_at: "2026-05-12T14:30:00Z"
    category: "machine-owned"
  devenv.nix:
    hash: "sha256:def456..."
    generated_at: "2026-05-12T14:30:00Z"
    category: "human-edited"
    markers:
      - "# --- semgrep ---"
      - "# --- gitleaks ---"
      - "# --- ripsecrets ---"
  CLAUDE.md:
    hash: "sha256:789abc..."
    generated_at: "2026-05-12T14:30:00Z"
    category: "human-edited"
    markers:
      - "<!-- BEGIN GENERATED SECTION -->"
      - "<!-- gdev:semgrep -->"
      - "<!-- gdev:gitleaks -->"

ecosystems_detected:
  - name: "npm"
    manifest: "package.json"
    lockfile: "package-lock.json"
  - name: "python"
    manifest: "pyproject.toml"
    lockfile: "requirements.txt"
  - name: "go"
    manifest: "go.mod"
    lockfile: "go.sum"
```

## Remediation Strategies

### Auto-Fixable Drift

| Drift Type | Auto-Fix |
|------------|----------|
| Machine-owned file modified | `gdev update` regenerates |
| Section markers missing | `gdev update --repair-markers` restores |
| Pre-commit hooks uninstalled | `gdev hooks install` reinstalls |
| Lock file stale | `npm install` / `go mod tidy` / etc. |
| New tools available | `gdev enable <tool>` |

### Manual-Fix Required

| Drift Type | Action |
|------------|--------|
| Human-edited file conflicts | `gdev update` shows diff, user merges |
| Version drift with breaking changes | `gdev update` with migration guide |
| Missing lock file for new ecosystem | User must run package manager install |
| Corrupt lock file | User must regenerate from manifest |

### `gdev update` Command

```
gdev update                    # Interactive update with diff preview
gdev update --dry-run          # Show what would change
gdev update --yes              # Auto-apply all changes
gdev update --repair-markers   # Restore section markers only
gdev update --force            # Overwrite everything including human-edited files
```

## Comparison to Infrastructure Drift Detection

**Terraform drift:** Compares state file against actual cloud resources. Analogous to gdev comparing state.yaml against actual files. Terraform can auto-remediate via `terraform apply`; gdev can auto-remediate via `gdev update`.

**Kubernetes drift:** Controllers continuously reconcile desired vs actual state. gdev's model is on-demand (run `gdev status`) rather than continuous, which is appropriate for file-based config.

**Key difference:** Infrastructure drift detection operates on remote resources. gdev operates entirely locally on files in the working directory, making it fast and offline-capable.

## Tradeoffs

**State file as source of truth:** If `.gdev/state.yaml` is lost or corrupted, drift detection breaks. The state file should be committed to git so it's version-controlled alongside the configs. If missing, `gdev status` should offer to rebuild it via `gdev init --rebuild-state`.

**False positives on human-edited files:** Flagging modifications to devenv.nix or CLAUDE.md as "drift" is wrong -- they're supposed to be edited. The category system (machine-owned vs human-edited) prevents this, but users need to understand the distinction.

**Marker fragility:** Section markers in CLAUDE.md and devenv.nix are strings in files that users might accidentally delete while editing. Making markers more distinctive (HTML comments with gdev prefix) reduces this risk but doesn't eliminate it.

**Git-awareness:** Should drift detection consider git status? A file might be "modified" in working directory but uncommitted. Or it might be modified and committed. For most purposes, current file state (not git state) is what matters for security posture.

## Depth Checklist

- [x] Underlying mechanism explained: SHA256 hashing, version comparison, marker parsing, state file design
- [x] Key tradeoffs and limitations identified: State file dependency, false positives, marker fragility
- [x] Compared to at least one alternative: Terraform drift, Kubernetes reconciliation
- [x] Failure modes and edge cases: Lost state file, deleted markers, corrupt lock files, offline mode
- [x] Concrete examples or reference implementations: State file schema, detection performance table, remediation matrix
- [x] Report is standalone-readable: Complete drift detection engine specification
