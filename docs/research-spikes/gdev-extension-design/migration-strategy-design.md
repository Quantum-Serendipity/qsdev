# Re-Runnability and Migration Strategy Design

## Problem

`gdev init` generates config files. Users and teams customize those files. Later, team standards evolve and `gdev init` or `gdev update` needs to update the generated config without destroying customizations. This document defines the strategy per file type.

## Core Principle: Track What We Generated

Every generated file falls into one of two categories:

1. **Machine-owned** — Generated from config, not intended for hand-editing. Can be safely regenerated. (devenv.yaml, .envrc, .gitignore additions)
2. **Human-edited** — Generated as a starting point, then customized by the user. Cannot be blindly overwritten. (CLAUDE.md, devenv.nix, settings.json, skills)

The merge strategy differs per category.

## Tracking Mechanism

Each addon persists a `GeneratedState` in gdev's config:

```yaml
devenv:
  generated:
    last_run: "2026-05-12T14:30:00Z"
    files:
      devenv.yaml:
        hash: "sha256:abc123..."
      devenv.nix:
        hash: "sha256:def456..."
      .envrc:
        hash: "sha256:789abc..."

claudecode:
  generated:
    last_run: "2026-05-12T14:30:00Z"
    files:
      CLAUDE.md:
        hash: "sha256:..."
      .claude/settings.json:
        hash: "sha256:..."
```

The hash is SHA256 of the file content at generation time. On update, comparing current file hash against stored hash tells us if the user modified it.

## Merge Strategy Per File

### devenv.yaml — Machine-Owned, Regenerate

**Strategy:** Regenerate from config. Safe because devenv.yaml is structural (inputs, imports, nixpkgs settings) — not a place users add custom logic.

**Flow:**
1. Read current file, compute hash
2. Compare against stored hash
3. If hashes match (unmodified) → overwrite
4. If hashes differ (user modified) → show diff, prompt for overwrite/skip/merge

**Merge mode:** Deep YAML merge. Add new inputs, preserve existing. Remove inputs no longer needed (with confirmation).

### devenv.nix — Human-Edited, Careful

**Strategy:** devenv.nix contains Nix code that users heavily customize. Cannot auto-merge Nix syntax.

**Flow:**
1. If hash matches → safe to regenerate
2. If hash differs → generate new version to `.devenv.nix.new`, show diff, let user merge manually
3. Never auto-overwrite a modified devenv.nix

**Why no auto-merge:** Nix is a functional language. Merging Nix expressions requires understanding semantics, not just syntax. A user might reorganize `let` bindings, add conditionals, or define custom modules. There's no general Nix merge algorithm.

**Escape hatch:** `gdev devenv update --force` overwrites regardless, for when the user wants to start fresh.

### .envrc — Machine-Owned, Idempotent

**Strategy:** The .envrc is a single `use devenv` line (possibly with extras). Check if the line exists; if not, append it.

**Flow:** Always safe to regenerate. The content is trivial and standardized.

### CLAUDE.md — Human-Edited, Section-Based

**Strategy:** CLAUDE.md is free-form markdown that users customize heavily. Use section markers to identify generated vs user content.

**Generated CLAUDE.md structure:**
```markdown
# CLAUDE.md

<!-- BEGIN GENERATED SECTION — do not edit between markers -->
## Project Overview
...auto-generated content...

## Build Commands
...auto-generated content...
<!-- END GENERATED SECTION -->

## Custom Instructions
...user adds their own content here...
```

**Flow:**
1. On initial generation: Create with markers + an empty "Custom Instructions" section
2. On update: Replace content between markers, preserve everything outside
3. If markers are missing (user deleted them) → treat as fully user-owned, skip update (warn)
4. If user added content between markers → warn, show diff, prompt

**Why markers:** They're the simplest approach that works. Other options (separate files, frontmatter metadata) are more complex and fragile.

### .claude/settings.json — Machine-Owned with Additions

**Strategy:** Three-way merge. The JSON structure is well-defined and mergeable.

**Flow:**
1. Compute hash of current file
2. If hash matches stored → safe to regenerate
3. If hash differs → three-way merge:
   - **Base:** Last generated version (reconstructable from stored config)
   - **Theirs:** Current file on disk (user's modifications)
   - **Ours:** New generated version
   - Merge logic: union permission lists, prefer user's values for conflicts, add new keys

**Merge specifics:**
- `permissions.allow`: Union of generated + user-added patterns
- `permissions.deny`: Union
- `hooks`: Merge by hook name (user hooks preserved, generated hooks updated)
- `sandbox`: User overrides take precedence
- New top-level keys from generation: added
- User-added top-level keys: preserved

```go
func mergeSettings(base, theirs, ours SettingsJSON) SettingsJSON {
    result := theirs // start with user's version
    
    // Add new generated permissions (don't remove user's)
    if ours.Permissions != nil {
        if result.Permissions == nil {
            result.Permissions = ours.Permissions
        } else {
            result.Permissions.Allow = unionStrings(
                result.Permissions.Allow,
                ours.Permissions.Allow,
            )
        }
    }
    
    // Update generated hooks, preserve user hooks
    for name, hook := range ours.Hooks {
        if _, isGenerated := base.Hooks[name]; isGenerated {
            result.Hooks[name] = hook // update our hook
        }
        // else: new generated hook, add it
        if _, exists := result.Hooks[name]; !exists {
            result.Hooks[name] = hook
        }
    }
    
    return result
}
```

### .claude/skills/*.md — Library-Managed, Versioned

**Strategy:** Skills from the team library are source-of-truth. User-created skills are left alone.

**Flow:**
1. Track which skills were installed from the library (stored in config)
2. On update: overwrite library skills with latest versions, skip user-created skills
3. Detect user-created skills: any .md in skills/ not in the library manifest

**Versioning:** The team skill library has a version per skill (in manifest.yaml). Only update skills whose library version changed.

### .claude/rules/*.md — Same as Skills

Same strategy as skills. Library-managed rules get updated; user-created rules preserved.

### .mcp.json — Machine-Owned with Additions

**Strategy:** Same three-way merge as settings.json. MCP servers are keyed by name.

**Flow:**
1. Generated servers: update to latest config
2. User-added servers: preserve
3. Removed generated servers: remove (with confirmation in interactive mode)

### .gitignore — Append-Only

**Strategy:** Only add entries, never remove. Idempotent — check if each entry exists before adding.

## Update Command Workflow

```
gdev init --update        (or gdev devenv update / gdev claude update)
    │
    ├── Read stored GeneratedState from gdev config
    ├── Read current files from disk
    ├── Compute hashes
    │
    ├── For each file:
    │   ├── Hash matches stored? → Regenerate (safe)
    │   ├── File is machine-owned? → Regenerate (safe)
    │   ├── File is human-edited + modified?
    │   │   ├── Has merge strategy? → Three-way merge
    │   │   ├── Has section markers? → Replace within markers
    │   │   └── No merge possible? → Generate .new file, show diff
    │   └── File doesn't exist? → Generate fresh
    │
    ├── Show plan preview (what will change)
    ├── Prompt for confirmation
    ├── Write files atomically
    └── Update stored GeneratedState (new hashes)
```

## Team Standards Versioning

When team standards evolve (new permissions, updated skills, changed conventions), the update flow is:

1. Team updates the gdev binary with new addon defaults/templates
2. Developer runs `gdev init --update` (or it's part of `gdev bootstrap`)
3. The merge strategy handles the rest:
   - New permissions → added to union
   - Updated skills → overwritten from library
   - Changed CLAUDE.md template → updated within markers
   - New services/tools → not added automatically (requires re-running wizard)

**Forcing a full re-wizard:** `gdev init --reconfigure` clears stored answers and re-runs the wizard from scratch, using detection to pre-populate.

## Edge Cases

**File deleted by user:** If a generated file is missing from disk but tracked in GeneratedState, don't recreate it — the user intentionally deleted it. Only recreate if `--force` is used.

**Config key mismatch:** If stored config has fields the current addon version doesn't recognize (downgrade scenario), preserve unknown fields in YAML rather than dropping them.

**Concurrent modification:** Atomic writes prevent corruption. If the user edits a file while generation is running, the atomic rename overwrites their edit — acceptable because the plan preview showed what would change.

**Monorepo:** Each subdirectory may have its own devenv.nix and .claude/. The init wizard operates on the current directory. No cross-directory coordination needed.

## Summary Table

| File | Category | Merge Strategy | On Conflict |
|------|----------|---------------|-------------|
| devenv.yaml | Machine-owned | Regenerate / deep merge | Prompt |
| devenv.nix | Human-edited | Hash check → .new file + diff | Never auto-overwrite |
| .envrc | Machine-owned | Idempotent append | Safe |
| CLAUDE.md | Human-edited | Section markers | Replace between markers |
| settings.json | Machine + additions | Three-way merge | Union lists |
| skills/*.md | Library-managed | Overwrite library, skip user | Version check |
| rules/*.md | Library-managed | Overwrite library, skip user | Version check |
| .mcp.json | Machine + additions | Three-way merge | Preserve user servers |
| .gitignore | Append-only | Idempotent | Safe |
