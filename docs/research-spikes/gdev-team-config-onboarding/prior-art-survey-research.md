# Prior Art Survey: Team-Level Developer Tooling

## Research Question

What patterns and lessons can gdev learn from existing tools that manage team-level developer configuration, project scaffolding, and environment standardization?

## Tool-by-Tool Analysis

### 1. Nx Workspace Generators

**What it does:** Nx is a monorepo build system with a plugin architecture. Custom generators create project scaffolding with org-specific conventions baked in. Custom presets wrap `create-nx-workspace` for branded setup.

**Configuration sharing model:** Nx plugins are published as npm packages. Generators encode org conventions (directory structure, tag schemas, CI templates) into code. Updates propagate via `nx migrate` which runs migration scripts when plugins are updated.

**Key lesson for gdev:** "When your tooling is easier to use than doing it wrong, your engineers are more likely to adopt conventions." Nx proves that wrapping generators with org logic is more effective than documentation. gdev's profile system is the equivalent.

**Limitation:** Deeply tied to the JS/npm ecosystem. Migration system is powerful but complex. Generators only work within Nx workspaces.

### 2. Yeoman Generators

**What it does:** Classic project scaffolding framework with a lifecycle-based generator system. Generators are npm packages with prompting -> configuring -> writing -> installing phases.

**Configuration sharing model:** Generators published to npm. Composability via `composeWith()` lets generators delegate to other generators. Answer persistence stores previous answers for re-use.

**Key lesson for gdev:** Yeoman's lifecycle phases (prompting -> writing -> installing) map directly to gdev's init flow. But Yeoman's decline shows that general-purpose generator frameworks lose to single-purpose tools with good defaults. gdev should not try to be a general generator framework.

**Why it declined:** Overhead of publishing generators to npm, composability never worked cleanly, replaced by simpler `create-*` tools.

### 3. Copier (Template-Based Generation)

**What it does:** Python tool that generates projects from Git-hosted templates. Supports `copier update` to propagate template changes to existing projects via three-way merge.

**Configuration sharing model:** Templates are git repositories. `.copier-answers.yml` tracks the template version and answers used to generate the project. Updates compute a diff between the original generation and the current project, apply the new template, then re-apply the developer's customizations.

**Key lesson for gdev:** Copier's three-way merge is the most sophisticated update mechanism in any template tool. gdev's migration strategy (from the extension-design spike) already has a similar approach: hash tracking to distinguish machine-owned from user-modified files, section markers for CLAUDE.md, three-way merge for settings.json. Copier validates that this approach works at scale.

**Limitation:** Copier is "permissive" -- it preserves user modifications but this means drift can creep back in. No enforcement mechanism.

### 4. Projen (Imperative Config Management)

**What it does:** AWS CDK companion tool that treats configuration files as build artifacts. Project config is defined in TypeScript code, and `npx projen` regenerates all config files. Generated files have a magic comment and are never manually edited.

**Configuration sharing model:** Custom Projen project types are published as npm packages. Organizations define a base project type that encodes all their standards. Individual projects extend it. Updates propagate by updating the Projen dependency version and re-running synthesis.

**Key lesson for gdev:** Projen's "generated files are build artifacts" philosophy is the extreme end of the spectrum. It eliminates drift entirely -- but at the cost of developer autonomy. gdev's approach is a middle ground: some files are machine-owned (like devenv.yaml, .envrc) and can be freely regenerated, while others are human-edited (devenv.nix, CLAUDE.md) and get careful merge strategies. This is the right balance for a consulting firm where projects have diverse needs.

**Limitation:** The "never edit generated files" rule is confusing and frustrating for developers who want to customize. Requires discipline and buy-in.

### 5. Dev Container Features

**What it does:** Standardized development environments via containers. Features are reusable install scripts + config that compose into a devcontainer.json.

**Configuration sharing model:** Three-tier hierarchy: Organization Base -> Team Override -> Project Specific. Custom features published to registries. Pre-built images for faster startup.

**Key lesson for gdev:** The three-tier hierarchy (org -> team -> project) validates gdev's proposed three-layer config model (compiled defaults -> .gdev.yaml -> .gdev.local.yaml). Dev containers prove that reducing onboarding to "open in container" is achievable. gdev's target of 3 commands is competitive with dev containers' 1-command setup, with the advantage of not requiring Docker.

**Limitation:** Requires Docker/container runtime. Cold start times can be long. Not all tools work well inside containers (GPU access, hardware debugging, etc.).

### 6. mise (formerly rtx)

**What it does:** Polyglot tool version manager with task runner. Manages Node, Python, Go, and 400+ other tools. Replaces asdf, nvm, pyenv, etc.

**Configuration sharing model:** `.mise.toml` (committed to git) defines project tool versions and tasks. `.mise.local.toml` (gitignored) for local overrides. Shell activation (`mise activate`) automatically picks up config when entering a project directory. Trust mechanism prevents untrusted configs from auto-activating.

**Key lesson for gdev:** mise's `.mise.toml` + `.mise.local.toml` split is exactly the pattern gdev should use for `.gdev.yaml` + `.gdev.local.yaml`. The trust mechanism is important for security: when an engineer clones a new project, gdev should prompt before auto-applying any configuration that could execute code (shell hooks, pre-commit scripts). The `trusted_config_paths` setting for pre-trusting org directories is a UX win.

**Onboarding flow:** `git clone` -> `cd project` -> `mise trust` (once) -> `mise install` -> tools are available. Four commands, but `mise trust` is the speed bump.

**Limitation:** mise manages tool versions but doesn't generate project configuration files. It complements gdev (gdev could use mise for tool version management) but doesn't replace it.

### 7. proto (moonrepo)

**What it does:** Toolchain manager with WASM plugin architecture. Manages tool versions with deep merge configuration.

**Configuration sharing model:** `.prototools` files at three levels (local, user, global) with four resolution modes. Deep merge with current directory taking highest precedence. Environment-specific configs via `PROTO_ENV`. Notable: proto itself can be pinned in the config (`proto = "0.38.0"`).

**Key lesson for gdev:** proto's resolution modes (local, global, upwards, upwards-global) show that different commands need different config scopes. gdev should consider whether `gdev check` (CI) should use only project-level config, while `gdev init` should merge all layers. proto's self-pinning pattern (`proto = "0.38.0"`) is equivalent to gdev's `gdev_version` constraint.

**Limitation:** WASM plugin architecture is powerful but complex. Smaller ecosystem than mise.

## Cross-Cutting Patterns

### Pattern: Committed Config + Gitignored Overrides

**Used by:** mise, proto, Angular, Rails, Django, most modern frameworks

This is the dominant pattern. A committed config file provides the team standard; a gitignored file allows individual customization.

| Tool | Committed | Gitignored |
|------|-----------|------------|
| mise | `.mise.toml` | `.mise.local.toml` |
| proto | `.prototools` | (user-level in home dir) |
| Rails | `config/database.yml` | `config/database.yml.local` |
| gdev (proposed) | `.gdev.yaml` | `.gdev.local.yaml` |

### Pattern: Hierarchical Resolution

**Used by:** EditorConfig, Biome, proto, CSS, devcontainer features

Configuration resolves by walking up the directory tree, merging files. Closer files override further ones.

**gdev adaptation:** gdev's three-layer hierarchy (compiled -> .gdev.yaml -> .gdev.local.yaml) is a simplified version. True hierarchical resolution (walking up directories) is unnecessary because gdev operates at the project root, not per-file.

### Pattern: Version Constraint in Config

**Used by:** Terraform (`required_version`), proto (self-pin), Node.js (`engines` in package.json), Cargo (MSRV)

The config file declares what version of the tool it's compatible with.

**gdev adaptation:** `gdev_version: ">= 0.15.0"` in `.gdev.yaml`. Checked before any operation.

### Pattern: Extends/Inherits from Named Presets

**Used by:** ESLint, Renovate, Biome, TypeScript (`tsconfig.json` extends)

A project config references a named base config and overrides specific fields.

**gdev adaptation:** `profile: go-web-service` in `.gdev.yaml` references a compiled profile. Unlike Renovate (which resolves presets from repos) or ESLint (which resolves from npm), gdev resolves from compiled-in profiles. This is simpler but means new profiles require a binary update.

### Pattern: Machine-Owned vs Human-Edited File Categories

**Used by:** Projen (all files machine-owned), Copier (three-way merge for human-edited), Angular CLI (schematics track ownership)

Different merge strategies for different file types.

**gdev adaptation:** Already designed in the migration strategy. Machine-owned files regenerated freely; human-edited files use hash tracking + format-specific merge strategies.

## Synthesis: What gdev Should Adopt

| Pattern | Source | gdev Adoption | Priority |
|---------|--------|---------------|----------|
| Committed config + gitignored overrides | mise, proto | `.gdev.yaml` + `.gdev.local.yaml` | Must have |
| Version constraint in config | Terraform, proto | `gdev_version` field | Must have |
| Config schema versioning | JSON best practices | `version` field + migration chain | Must have |
| Profile/preset selection | Renovate, ESLint | `profile` field referencing compiled profiles | Must have |
| Trust mechanism | mise | Prompt before first use on untrusted project | Should have |
| Three-way merge for updates | Copier | Already designed in migration strategy | Already planned |
| Machine-owned vs human-edited | Projen, Copier | Already designed in migration strategy | Already planned |
| CI validation command | OpenSSF Scorecard | `gdev check` with SARIF output | Must have |
| Hierarchical dir resolution | EditorConfig, Biome | Not needed (gdev operates at project root) | N/A |
| Package-based config sharing | ESLint, Nx | Not needed (profiles compiled into binary) | N/A |

## Depth Checklist

- [x] Underlying mechanism explained -- seven tools analyzed in depth
- [x] Key tradeoffs and limitations identified -- per-tool and cross-cutting
- [x] Compared to alternatives -- tools compared against each other and against gdev's needs
- [x] Failure modes and edge cases described -- Yeoman's decline, Projen's developer friction, Copier's drift
- [x] Concrete examples -- config file formats, CLI flows, comparison tables
- [x] Report is standalone-readable
