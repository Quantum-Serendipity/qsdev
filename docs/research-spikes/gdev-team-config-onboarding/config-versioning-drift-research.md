# Configuration Versioning and Drift

## Research Question

How should gdev handle version mismatches between the binary, project config, and generated files? How do template updates propagate without destroying customizations?

## Three Versioning Dimensions

gdev has three independent version axes that can drift:

1. **Binary version** -- the gdev binary itself (e.g., v0.15.0)
2. **Config schema version** -- the `.qsdev.yaml` format version (e.g., 1, 2, 3)
3. **Template version** -- the templates used to generate output files (tied to binary version, but manifested in file hashes)

Each axis can drift independently, creating a compatibility matrix.

## Prior Art: Version Compatibility Patterns

### Terraform's `required_version`

Terraform's approach is the gold standard for CLI tool version constraints:

```hcl
terraform {
  required_version = ">= 1.5.0, < 2.0.0"
}
```

Terraform consults this before any operation. If the binary version does not satisfy the constraint, Terraform halts with an actionable error message. Key properties:
- Constraint checked before any destructive action
- Supports semver operators: `=`, `!=`, `>`, `>=`, `<`, `<=`, `~>`
- Pre-release versions only match exact pins
- Root modules use tight constraints; libraries use loose minimum-only constraints

### Cargo's MSRV (Minimum Supported Rust Version)

Rust crates declare `rust-version = "1.70"` in `Cargo.toml`. The `cargo-msrv` tool verifies compatibility. Key insight: MSRV is a minimum floor, not a range -- the assumption is newer versions are backward-compatible.

### JSON Schema Versioning Best Practices

For configuration file format evolution:
- Root-level `schemaVersion` field (integer, not semver)
- Separate from application version
- Incremental migration functions: `v1 -> v2 -> v3` (never `v1 -> v3` directly)
- Explicit support window (e.g., "current and previous 2 versions")
- Validate against the declared version's schema, then migrate to latest internal representation

## Recommended Design for gdev

### `.qsdev.yaml` Version Fields

```yaml
# .qsdev.yaml
version: 1                    # Config schema version (integer)
gdev_version: ">= 0.15.0"    # Binary version constraint (semver)
```

**`version`** (integer): Format version of the `.qsdev.yaml` file itself. Bumped when the schema changes in a backward-incompatible way. gdev reads this, loads the appropriate schema parser, and migrates to the latest internal representation.

**`gdev_version`** (semver constraint string): Minimum (and optionally maximum) gdev binary version that can process this config. Checked before any operation.

### Version Check Flow

```
qsdev init (or any gdev command)
  |
  +-- Read .qsdev.yaml
  |
  +-- Check gdev_version constraint against binary version
  |     |
  |     +-- Satisfied -> continue
  |     +-- Not satisfied -> error with actionable message:
  |           "Your gdev version (0.14.2) does not satisfy the project's
  |            requirement (>= 0.15.0). Run: gdev self-update"
  |
  +-- Check config schema version
  |     |
  |     +-- Current version -> parse directly
  |     +-- Older version -> migrate v(n) -> v(n+1) -> ... -> v(current)
  |     +-- Newer version -> error: "This config requires gdev >= X.Y.Z"
  |
  +-- Proceed with operation
```

### Config Schema Migration

Each schema version has a migration function:

```go
type Migration struct {
    FromVersion int
    ToVersion   int
    Migrate     func(old map[string]any) (map[string]any, error)
}

var migrations = []Migration{
    {1, 2, migrateV1toV2},
    {2, 3, migrateV2toV3},
}

func migrateV1toV2(old map[string]any) (map[string]any, error) {
    // Example: "languages" was a string list in v1, became objects in v2
    if langs, ok := old["languages"].([]string); ok {
        langObjects := make([]map[string]any, len(langs))
        for i, l := range langs {
            langObjects[i] = map[string]any{"name": l, "version": "latest"}
        }
        old["languages"] = langObjects
    }
    old["version"] = 2
    return old, nil
}
```

**Key design decisions:**
- Migrations are always forward-only (old -> new), never backward
- Each migration is a pure function: `old config -> new config`
- Chain migrations: v1 -> v2 -> v3, never v1 -> v3 directly
- Support window: current version + 2 previous versions (configurable)
- Migrations run in-memory only -- the file on disk is not rewritten unless the user explicitly runs `qsdev config migrate`

### Template Drift Detection

The existing migration strategy design (from the gdev-extension-design spike) already covers per-file hash tracking. Here we extend it to cover the "team member using older gdev" scenario.

**Problem:** Engineer A runs gdev v0.16.0 which generates updated pre-commit hooks. Engineer B has gdev v0.15.0 and runs `qsdev init --update`, which would downgrade the hooks.

**Solution:** Generated file state includes the gdev version that produced it:

```yaml
# In gdev's internal state (not committed to git)
generated:
  last_run: "2026-05-12T14:30:00Z"
  gdev_version: "0.16.0"
  files:
    .pre-commit-config.yaml:
      hash: "sha256:abc123..."
      template_version: "0.16.0"
```

When gdev v0.15.0 encounters files generated by v0.16.0:

```
$ qsdev init --update
  ⚠ Some files were generated by a newer gdev version (0.16.0)
  Your version: 0.15.0
  
  Affected files:
  - .pre-commit-config.yaml (generated by 0.16.0)
  - .claude/settings.json (generated by 0.16.0)
  
  Options:
  1. Skip these files (recommended -- don't downgrade)
  2. Overwrite anyway (may remove newer features)
  3. Update gdev first: gdev self-update
```

**Default behavior:** Never downgrade generated files. An older binary refuses to overwrite files produced by a newer binary unless explicitly forced.

### Handling Team Version Skew

In practice, team members will run different gdev versions. The version skew strategy:

1. **`.qsdev.yaml` `gdev_version` sets the floor.** All team members must meet this minimum.

2. **Generated files record their producing version.** The internal state (per-developer, not committed) tracks which gdev version produced each file.

3. **Newer versions can update; older versions refuse to downgrade.** This creates a "ratchet" effect -- once anyone on the team generates with v0.16.0, the `.qsdev.yaml` should be updated to `gdev_version: ">= 0.16.0"`.

4. **`qsdev init --update` with `--bump-version` flag** updates the `gdev_version` constraint to match the current binary, signaling to the team that they need to update.

5. **CI enforcement** (covered in the standards enforcement research) can verify that all team members' gdev versions satisfy the project constraint.

## Configuration Format Migration Examples

### Non-Breaking Change (No Version Bump)

Adding an optional field with a sensible default:

```yaml
# v1 config (old)
languages:
  - go

# v1 config (new gdev version adds mcp_servers with default)
languages:
  - go
# mcp_servers not present -- gdev uses default ["context7", "github"]
```

No migration needed. The config loader provides defaults for missing optional fields.

### Breaking Change (Version Bump)

Restructuring the languages field from strings to objects:

```yaml
# version: 1
languages:
  - go
  - typescript

# version: 2
languages:
  - name: go
    version: "1.22"
  - name: typescript
    version: "22"
```

Migration function transforms the data. The file on disk retains `version: 1` until the user runs `qsdev config migrate`.

### Field Rename

```yaml
# version: 2
security:
  age_gating: true

# version: 3
security:
  package_age_gate:
    enabled: true
    threshold_hours: 72
```

Migration function maps the old boolean to the new structured format.

## Edge Cases

1. **`.qsdev.yaml` manually edited with syntax error:** gdev must report the YAML parse error with line number and suggest checking recent git changes. Never proceed with a partially-parsed config.

2. **Config version newer than binary understands:** Error with "This project's .qsdev.yaml uses config version 4, but your gdev only supports up to version 3. Run `qsdev self-update`."

3. **Migration chain has a gap:** If a binary supports versions 3-5 but encounters version 1, and the v1->v2 migration was dropped from the binary: Error with "Config version 1 is too old. Please update to at least version 3 manually or use gdev >= X.Y.Z which still supports migration from v1."

4. **Multiple engineers bump version simultaneously:** Git merge conflict on the `gdev_version` field in `.qsdev.yaml`. This is a feature, not a bug -- it forces the team to agree on the version floor.

5. **Rollback scenario:** Team discovers a gdev version has a bug and needs to roll back. The `gdev_version` constraint is lowered in `.qsdev.yaml`, but generated files from the buggy version remain. `qsdev init --update --force` regenerates everything from the rolled-back version.

## Depth Checklist

- [x] Underlying mechanism explained -- three version axes, migration chain, ratchet strategy
- [x] Key tradeoffs and limitations identified -- ratchet prevents downgrades, migration chain maintenance burden
- [x] Compared to alternatives -- Terraform required_version, Cargo MSRV, JSON schema versioning
- [x] Failure modes and edge cases described -- five scenarios
- [x] Concrete examples -- YAML configs, Go migration code, CLI output
- [x] Report is standalone-readable
