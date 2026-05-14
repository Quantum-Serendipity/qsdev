# Phase 26: Non-Language Tool Detection Modules

## Goal

Add four non-language tool detection modules — Git Platform CLI, Documentation Tools, API Tools, and Database Migration Tools — that detect project tooling from config files, directory conventions, and dependency patterns rather than language runtimes. These modules run as optional post-processing passes alongside the existing language ecosystem detection, enriching devenv.nix packages and CLAUDE.md context without competing with Tier 1/2 language module detection.

## Dependencies

Phase 1 complete (shared types, `EcosystemModule` interface, detection engine, template engine). Phase 3 complete (devenv addon core generation, devenv.nix emit pipeline). Phase 4 complete (Claude Code addon, CLAUDE.md section marker system, skill library). Phase 6 complete (wizard orchestration, customize path, non-interactive flags).

## Phase Outputs

- 4 non-language detection modules implementing the existing `EcosystemModule` interface
- Module parallel execution: non-language modules run after language modules, can run in parallel with each other
- devenv task definitions per detected migration tool (`devenv task db:migrate`, `devenv task db:rollback`)
- devenv task definitions per detected documentation tool (`devenv task docs:build`, `devenv task docs:serve`)
- CLAUDE.md sections for all detected non-language tools (platform commands, migration commands, doc commands)
- Pre-commit hook additions for gRPC (buf lint) and OpenAPI (validation) tools
- Wizard customize path entries: non-language tools are supplementary and never appear in quick path

---

### Unit 26.1: Git Platform CLI Module

**Description:** Implement the Git Platform CLI detection module. Detects `gh`, `glab`, and `git-lfs` from project structure and git remote configuration, adds them to devenv.nix packages, and generates CLAUDE.md sections with platform-specific commands. Also catalogs optional TUI and productivity tools available via `gdev enable`.

**Context:** GitHub CLI (`gh`) is practically essential for GitHub-hosted projects — it enables PR creation, CI debugging, release management, and API queries from the terminal. GitLab CLI (`glab`) mirrors `gh` for GitLab projects, which are common in enterprise consulting engagements. `git-lfs` is non-negotiable: LFS repos simply do not work without it, and the detection signal (`.gitattributes` containing `filter=lfs`) is unambiguous. The research notes that `gh` and the GitHub MCP server are complementary — gh is the human developer's tool for PR workflow while the MCP server is the AI agent's tool. Both can coexist.

Bitbucket CLI is explicitly skipped: no stable nixpkgs package exists and the authentication model is in flux (app passwords deprecated September 2025, gone June 2026). The `delta` diff pager and `lazygit` TUI are catalog-only (not auto-detected): they are personal preference tools surfaced via `gdev enable delta` and `gdev enable lazygit`.

**Code-Grounded Note:** `gh` and `glab` require authentication (`gh auth login`, `glab auth login`) which is interactive and cannot be automated. The module adds these binaries to devenv.nix packages and generates auth instructions in CLAUDE.md — it does not attempt to configure auth. The `git-lfs` package requires a one-time `git lfs install` call after installation; this should be added to devenv.nix's `enterShell` hook.

**Desired Outcome:** Projects with `.github/` or github.com remotes get `gh` in devenv.nix packages and GitHub-specific commands in CLAUDE.md. Projects with `.gitlab-ci.yml` get `glab`. Projects with LFS get `git-lfs` plus the `git lfs install` setup call.

**Steps:**

1. Create `internal/modules/git-platform/git_platform.go` implementing the `EcosystemModule` interface:
   ```go
   type GitPlatformModule struct{}

   func (m *GitPlatformModule) Name() string    { return "git-platform" }
   func (m *GitPlatformModule) IsLanguage() bool { return false }
   func (m *GitPlatformModule) RunOrder() int   { return RunOrderPost }  // after language modules
   ```

2. Implement `Detect(projectRoot string) DetectionResult`:
   - **GitHub CLI (`gh`) signals**:
     - `.github/` directory exists in project root (any content: Actions, CODEOWNERS, etc.)
     - `git remote get-url origin` output contains `github.com` (run via exec, fall back to parsing `.git/config`)
     - `GITHUB_TOKEN` or `GH_TOKEN` env var referenced in CI config or `.env.example`
   - **GitLab CLI (`glab`) signals**:
     - `.gitlab-ci.yml` exists in project root
     - `git remote get-url origin` output contains `gitlab.com` or a pattern matching common self-hosted GitLab (`gitlab.` prefix)
     - `GITLAB_TOKEN` or `CI_JOB_TOKEN` referenced in CI config or env files
   - **git-lfs signals**:
     - `.gitattributes` file containing the string `filter=lfs` (unambiguous LFS indicator)
   - Return `DetectionResult` with per-tool detection flags: `{gh: bool, glab: bool, gitLFS: bool}`

3. Implement `Generate(detected DetectionResult) GenerationResult`:
   - Build packages list from detected tools:
     ```go
     var packages []string
     if detected.Tools["gh"]     { packages = append(packages, "gh") }
     if detected.Tools["glab"]   { packages = append(packages, "glab") }
     if detected.Tools["gitLFS"] { packages = append(packages, "git-lfs") }
     ```
   - devenv.nix packages block fragment:
     ```nix
     # Git platform tools
     packages = [
       pkgs.gh       # GitHub CLI — run "gh auth login" after first devenv shell
       pkgs.git-lfs  # Git Large File Storage
     ];
     ```
   - devenv.nix enterShell hook for git-lfs (when detected):
     ```nix
     enterShell = ''
       # Initialize git-lfs for this repository (idempotent)
       git lfs install --local 2>/dev/null || true
     '';
     ```
   - CLAUDE.md section content (GitHub example):
     ```markdown
     ## Git Platform

     **GitHub CLI** (`gh`):
     - Create PR: `gh pr create --fill`
     - View CI status: `gh run list`
     - Review PR: `gh pr checkout <number>`
     - Debug failed run: `gh run view <id> --log-failed`
     - Auth: `gh auth login` (first time only)
     ```
   - GitLab variant when `glab` detected:
     ```markdown
     **GitLab CLI** (`glab`):
     - Create MR: `glab mr create --fill`
     - View pipeline: `glab ci list`
     - Review MR: `glab mr checkout <number>`
     - Auth: `glab auth login` (first time only)
     ```
   - git-lfs section when detected:
     ```markdown
     **git-lfs**: Large files tracked via LFS. Run `git lfs pull` after clone to fetch LFS content.
     ```

4. Implement `SecurityConfig() SecurityHardening`:
   - No pre-commit hook additions
   - No deny rules
   - CLAUDE.md note on token handling: "Never commit GITHUB_TOKEN or GITLAB_TOKEN — use CI secrets or SecretSpec"

5. Add catalog entries for opt-in tools (no auto-detection, available via `gdev enable`):
   ```go
   var GitPlatformCatalog = []CatalogTool{
       {
           Name:        "lazygit",
           NixPackage:  "lazygit",
           Description: "Terminal UI for git — staging, rebasing, branch management",
           EnableCmd:   "gdev enable lazygit",
       },
       {
           Name:        "delta",
           NixPackage:  "delta",
           Description: "Syntax-highlighted diff pager for git log/diff/show",
           EnableCmd:   "gdev enable delta",
           PostInstall: "Adds pager config to .gitconfig fragment in enterShell",
       },
   }
   ```
   - These appear in `gdev list --category git` output but are never auto-added to devenv.nix

6. Write unit tests:
   - `.github/` present → `gh` detected
   - `.github/` absent, `.git/config` has `github.com` remote → `gh` detected
   - `.gitlab-ci.yml` present → `glab` detected
   - `.gitattributes` with `filter=lfs` → `git-lfs` detected
   - `.gitattributes` without `filter=lfs` → `git-lfs` not detected
   - All three signals present → all three tools in generated packages
   - Generated devenv.nix includes `git lfs install --local` in enterShell when git-lfs detected
   - CLAUDE.md section includes correct platform commands based on detected tools
   - No `.github/` and no github.com remote → `gh` not detected

**Acceptance Criteria:**
- [ ] Module implements `EcosystemModule` interface with `IsLanguage() = false` and `RunOrder() = RunOrderPost`
- [ ] GitHub CLI detection: `.github/` directory OR github.com git remote
- [ ] GitLab CLI detection: `.gitlab-ci.yml` file OR gitlab.com git remote
- [ ] git-lfs detection: `.gitattributes` containing `filter=lfs` (specific, no false positives)
- [ ] Detected tools added to devenv.nix packages list
- [ ] `git lfs install --local` added to enterShell hook when git-lfs detected (idempotent with `|| true`)
- [ ] CLAUDE.md section generated with platform-specific PR/MR/pipeline commands
- [ ] `lazygit` and `delta` registered as catalog tools (available via `gdev enable`, not auto-detected)
- [ ] Bitbucket CLI explicitly skipped — no detection, no catalog entry, no packages
- [ ] Unit tests cover detection logic, generated packages, enterShell hook, and CLAUDE.md content

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/git-docs-ide-research.md` — gh/glab/git-lfs assessment, detection heuristics, delta/lazygit catalog rationale, Bitbucket skip rationale

**Status:** Not Started

---

### Unit 26.2: Documentation Tools Module

**Description:** Implement the Documentation Tools detection module. Detects mkdocs, mdbook, d2, PlantUML, and adr-tools from their configuration files and source content. Adds detected tools to devenv.nix packages and generates `devenv task docs:build` and `devenv task docs:serve` task definitions per tool.

**Context:** Documentation generators are project-level dependencies with clear, unambiguous detection heuristics: each has a distinctive config file (`mkdocs.yml`, `book.toml`, `*.d2`, `*.puml`). This is the same detection philosophy as the language ecosystem modules — if the config file exists, the tool is in use. The consulting relevance is high: mkdocs with Material theme is the dominant enterprise documentation pattern, mdbook is the Rust ecosystem standard, and D2 is gaining rapid adoption as a diagrams-as-code tool. The devenv task generation converts what would otherwise be ad-hoc shell commands into well-known, discoverable `devenv task` invocations that Claude Code agents can also use.

**Code-Grounded Note:** mkdocs-material is a Python package that ships as `python3Packages.mkdocs-material` in nixpkgs. When `mkdocs.yml` contains `theme: material` or `theme:\n  name: material`, the Material theme package should be added alongside base mkdocs. mkdocs plugins (mermaid2, macros, etc.) referenced in `mkdocs.yml` under `plugins:` should also be included if they are in nixpkgs — otherwise a note is added to CLAUDE.md. PlantUML requires the JVM (`pkgs.jre`); this is a significant dependency and should be flagged in the generated devenv.nix comment.

**Desired Outcome:** Projects with documentation tooling get correct devenv.nix packages, `devenv task docs:build` and `devenv task docs:serve` definitions per tool, and CLAUDE.md documentation of build/serve commands. PlantUML installation is clearly annotated as pulling in the JVM.

**Steps:**

1. Create `internal/modules/docs-tools/docs_tools.go` implementing `EcosystemModule`:
   ```go
   type DocsToolsModule struct{}

   func (m *DocsToolsModule) Name() string    { return "docs-tools" }
   func (m *DocsToolsModule) IsLanguage() bool { return false }
   func (m *DocsToolsModule) RunOrder() int   { return RunOrderPost }
   ```

2. Implement `Detect(projectRoot string) DetectionResult`:
   - **mkdocs**: `mkdocs.yml` or `mkdocs.yaml` in project root or `docs/`
     - Sub-detection: parse YAML for `theme: material` or `theme:\n  name: material` → `materialTheme: true`
     - Sub-detection: parse `plugins:` list → extract plugin names that have nixpkgs equivalents
   - **mdbook**: `book.toml` in project root
   - **d2**: any `*.d2` file anywhere in the project (recursive scan, depth-limited to 5 levels)
   - **PlantUML**: any `*.puml` or `*.plantuml` file; or `plantuml` referenced in CI workflows
   - **adr-tools**: `docs/adr/` directory containing `*.md` files, OR `doc/adr/` directory
   - Return per-tool detection flags with confidence scores

3. Implement `Generate(detected DetectionResult) GenerationResult`:
   - mkdocs packages:
     ```nix
     pkgs.mkdocs
     # mkdocs-material theme (adds ~200MB Python dependencies)
     pkgs.python3Packages.mkdocs-material
     ```
   - mdbook packages:
     ```nix
     pkgs.mdbook
     ```
   - d2 packages:
     ```nix
     pkgs.d2
     ```
   - PlantUML packages (with annotation):
     ```nix
     # PlantUML requires JRE (~200MB) — needed for *.puml files in this project
     pkgs.plantuml
     pkgs.jre  # PlantUML dependency
     ```
   - adr-tools packages:
     ```nix
     pkgs.adr-tools
     ```

4. Generate devenv task definitions per detected tool. In devenv.nix:
   ```nix
   devenv.tasks = {
     # mkdocs tasks (when mkdocs detected)
     "docs:build" = {
       exec = "mkdocs build --strict";
       description = "Build documentation site";
     };
     "docs:serve" = {
       exec = "mkdocs serve --dev-addr localhost:8000";
       description = "Serve documentation with live reload at http://localhost:8000";
     };

     # mdbook tasks (when mdbook detected, replaces mkdocs tasks if both detected)
     "docs:build" = {
       exec = "mdbook build";
       description = "Build mdbook documentation";
     };
     "docs:serve" = {
       exec = "mdbook serve --port 8000";
       description = "Serve mdbook with live reload at http://localhost:8000";
     };
   };
   ```
   - If multiple doc tools are detected, prefix the task names: `docs:mkdocs:build`, `docs:mdbook:build`
   - Always include a generic `docs:build` alias pointing to the primary detected tool

5. Generate CLAUDE.md documentation section:
   ```markdown
   ## Documentation

   **mkdocs** (Material theme):
   - Build: `devenv task docs:build` or `mkdocs build --strict`
   - Serve: `devenv task docs:serve` — live reload at http://localhost:8000
   - Config: `mkdocs.yml`

   **D2 diagrams** (`*.d2` files):
   - Render: `d2 <file>.d2 <file>.svg`
   - Watch: `d2 --watch <file>.d2 <file>.svg`
   ```

6. Generate pre-commit hook suggestions for D2 (optional, not auto-added):
   - If D2 detected, add note to CLAUDE.md: "D2 diagrams can be validated in pre-commit with `d2 fmt --check`"
   - Do not auto-add this hook — diagram tooling pre-commit configuration is project-specific

7. Handle the "both mkdocs and mdbook" edge case:
   - Both can coexist in the same project (e.g., Rust project with mdbook for API docs and mkdocs for architecture docs)
   - Generate tasks with tool-specific prefixes when both detected
   - CLAUDE.md lists both with their respective task commands

8. Write unit tests:
   - `mkdocs.yml` in root → mkdocs detected
   - `mkdocs.yml` with `theme: material` → material theme package added
   - `book.toml` → mdbook detected
   - `diagram.d2` in `docs/` → d2 detected
   - `architecture.puml` in `docs/` → plantuml detected (with jre dependency)
   - `docs/adr/0001-initial.md` directory pattern → adr-tools detected
   - mkdocs + mdbook both present → both detected, task names prefixed
   - Generated devenv tasks include `docs:build` and `docs:serve`
   - PlantUML packages include `plantuml` and `jre`
   - No documentation files → nothing detected

**Acceptance Criteria:**
- [ ] Module implements `EcosystemModule` interface with `IsLanguage() = false` and `RunOrder() = RunOrderPost`
- [ ] mkdocs detection: `mkdocs.yml`; Material theme sub-detection: parses YAML for theme name
- [ ] mdbook detection: `book.toml`
- [ ] d2 detection: any `*.d2` file (recursive scan, depth-limited)
- [ ] PlantUML detection: `*.puml` or `*.plantuml` files
- [ ] adr-tools detection: `docs/adr/` or `doc/adr/` directory with markdown files
- [ ] mkdocs-material package added when Material theme detected
- [ ] PlantUML package includes `pkgs.jre` with comment explaining JVM dependency
- [ ] `devenv task docs:build` and `devenv task docs:serve` generated for each detected doc tool
- [ ] Task names prefixed when multiple doc tools detected simultaneously
- [ ] CLAUDE.md section documents build/serve commands with devenv task invocations
- [ ] Unit tests cover all detection signals, Material theme sub-detection, multi-tool edge case, generated task definitions

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/git-docs-ide-research.md` — mkdocs/mdbook/d2/PlantUML/adr-tools assessment, nixpkgs availability, detection heuristics, JVM dependency note

**Status:** Not Started

---

### Unit 26.3: API Tools Module

**Description:** Implement the API Tools detection module. Detects gRPC/protobuf tooling, OpenAPI/Swagger specifications, and Bruno API collections. Adds detected tools to devenv.nix packages, generates pre-commit hooks for schema validation, and produces CLAUDE.md sections with API workflow commands.

**Context:** API tooling has clear, file-based detection signals with no ambiguity: `.proto` files mean gRPC, `openapi.yaml` means OpenAPI, `.bru` files mean Bruno. These are not inferred from dependencies — they are the primary artifact of API-first development. grpcurl and buf are always installed together for gRPC projects: buf provides the modern protobuf toolchain (lint, format, breaking change detection) replacing fragmented `protoc` + plugin setups, while grpcurl provides the CLI REPL for testing live services. The openapi-generator-cli and redocly pairing covers both code generation (openapi-generator) and linting/preview (redocly) for OpenAPI projects.

Bruno is the open-source Postman replacement and its Git-native storage (`.bru` files and `bruno.json`) aligns with gdev's version-control-first philosophy. The `bruno-cli` package enables non-interactive collection running in CI — the same `.bru` files that developers use interactively can run automated in pipelines.

**Code-Grounded Note:** `grpcurl` and `buf` are both in nixpkgs. `openapi-generator-cli` and `redocly` are both in nixpkgs. `bruno-cli` is in nixpkgs as `bruno-cli`. buf integration with pre-commit is straightforward: `buf lint` and `buf breaking` commands are designed for CI and pre-commit use. The `buf.yaml` file, when present, is the primary buf configuration and should be respected by generated pre-commit hooks.

**Desired Outcome:** Projects with `.proto` files get grpcurl + buf in devenv.nix and a buf lint pre-commit hook. Projects with OpenAPI specs get openapi-generator-cli + redocly and an openapi-validate pre-commit hook. Projects with Bruno collections get bruno-cli and collection-runner documentation in CLAUDE.md.

**Steps:**

1. Create `internal/modules/api-tools/api_tools.go` implementing `EcosystemModule`:
   ```go
   type APIToolsModule struct{}

   func (m *APIToolsModule) Name() string    { return "api-tools" }
   func (m *APIToolsModule) IsLanguage() bool { return false }
   func (m *APIToolsModule) RunOrder() int   { return RunOrderPost }
   ```

2. Implement `Detect(projectRoot string) DetectionResult`:
   - **gRPC/protobuf signals**:
     - Any `*.proto` file anywhere in the project (recursive scan, depth-limited to 6 levels)
     - `buf.yaml`, `buf.gen.yaml`, or `buf.work.yaml` in project root or `proto/` directory
   - **OpenAPI signals**:
     - Files named `openapi.yaml`, `openapi.json`, `swagger.yaml`, `swagger.json` in root or `api/` or `docs/`
     - `.redocly.yaml` in project root (Redocly config → OpenAPI in use)
     - `openapi-generator.json` or `.openapi-generator/` directory
   - **Bruno signals**:
     - `*.bru` files (recursive scan)
     - `bruno.json` in project root or a subdirectory
   - **httpie signal** (opt-in catalog only, not auto-detected):
     - No auto-detection for httpie — it is a general-purpose HTTP client with no distinctive project files
   - Return per-tool detection flags: `{grpc: bool, openapi: bool, bruno: bool}`

3. Implement `Generate(detected DetectionResult) GenerationResult`:
   - gRPC packages:
     ```nix
     pkgs.grpcurl  # gRPC CLI client — test live services and reflection
     pkgs.buf      # Modern protobuf toolchain: lint, format, breaking change detection
     ```
   - OpenAPI packages:
     ```nix
     # openapi-generator-cli requires JRE
     pkgs.openapi-generator-cli  # Generate client SDKs and server stubs from OpenAPI specs
     pkgs.redocly                # Lint, bundle, and preview OpenAPI specs
     ```
   - Bruno packages:
     ```nix
     pkgs.bruno-cli  # Run Bruno API collections in CI
     ```
   - Note: openapi-generator-cli requires JRE. Add `pkgs.jre` if not already in packages (check against existing language module outputs — Java/Kotlin language module will have already added it)

4. Generate pre-commit hook additions:
   - For gRPC (buf lint, only when `buf.yaml` present):
     ```yaml
     - repo: https://github.com/bufbuild/buf
       rev: v1.59.0  # pin to current buf version in nixpkgs
       hooks:
         - id: buf-lint
           args: ['--config', 'buf.yaml']
         - id: buf-format
           args: ['--diff', '--exit-code']
     ```
   - For OpenAPI (redocly lint):
     ```yaml
     - repo: local
       hooks:
         - id: redocly-lint
           name: Lint OpenAPI spec
           entry: redocly lint
           language: system
           files: '^(openapi|swagger)\.(yaml|json)$'
           pass_filenames: true
     ```
   - Pre-commit hooks are added to `.pre-commit-config.yaml` via the Phase 3 section marker system

5. Generate CLAUDE.md API tooling section:
   - gRPC section:
     ```markdown
     ## API Tools

     **gRPC** (`.proto` files):
     - Test service: `grpcurl -plaintext localhost:<port> <package>.<Service>/<Method>`
     - List services: `grpcurl -plaintext localhost:<port> list`
     - Lint protos: `buf lint`
     - Format protos: `buf format --write`
     - Check breaking changes: `buf breaking --against '.git#branch=main'`
     ```
   - OpenAPI section:
     ```markdown
     **OpenAPI spec** (`openapi.yaml`):
     - Lint spec: `redocly lint openapi.yaml`
     - Preview docs: `redocly preview-docs openapi.yaml` (http://localhost:8080)
     - Generate TypeScript client: `openapi-generator-cli generate -i openapi.yaml -g typescript-fetch -o ./src/api`
     ```
   - Bruno section:
     ```markdown
     **Bruno API collections** (`*.bru`):
     - Run collection: `bru run <collection-dir>/`
     - Run in CI: `bru run <collection-dir>/ --env <environment>`
     ```

6. Register httpie as catalog-only (no auto-detection):
   ```go
   var APIToolsCatalog = []CatalogTool{
       {
           Name:        "httpie",
           NixPackage:  "httpie",
           Description: "Human-friendly CLI HTTP client with colorized output",
           EnableCmd:   "gdev enable httpie",
       },
   }
   ```

7. Write unit tests:
   - `service.proto` in `proto/` → gRPC detected
   - `buf.yaml` in root → gRPC detected (even without .proto files — buf config implies protobuf usage)
   - `openapi.yaml` in root → OpenAPI detected
   - `.redocly.yaml` in root → OpenAPI detected
   - `requests.bru` in `api-tests/` → Bruno detected
   - `bruno.json` in root → Bruno detected
   - gRPC detection → generated packages include grpcurl + buf
   - OpenAPI detection → generated packages include openapi-generator-cli + redocly
   - Bruno detection → generated packages include bruno-cli
   - gRPC + `buf.yaml` → pre-commit buf-lint hook generated
   - OpenAPI detected → pre-commit redocly-lint hook generated
   - Neither gRPC nor OpenAPI nor Bruno → nothing detected

**Acceptance Criteria:**
- [ ] Module implements `EcosystemModule` interface with `IsLanguage() = false` and `RunOrder() = RunOrderPost`
- [ ] gRPC detection: any `*.proto` file OR `buf.yaml`/`buf.gen.yaml`/`buf.work.yaml`
- [ ] OpenAPI detection: `openapi.yaml`/`openapi.json`/`swagger.yaml`/`swagger.json`/`.redocly.yaml`
- [ ] Bruno detection: `*.bru` files OR `bruno.json`
- [ ] gRPC packages: `grpcurl` + `buf`
- [ ] OpenAPI packages: `openapi-generator-cli` + `redocly` (with jre if not already present)
- [ ] Bruno packages: `bruno-cli`
- [ ] buf lint pre-commit hook generated when gRPC + `buf.yaml` both detected
- [ ] redocly lint pre-commit hook generated when OpenAPI detected
- [ ] CLAUDE.md sections for each detected API category with specific command examples
- [ ] `httpie` registered as catalog tool (no auto-detection signal)
- [ ] Unit tests cover all detection signals, generated packages, pre-commit hooks, false-positive avoidance

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/api-db-mcp-research.md` — grpcurl/buf/openapi-generator/redocly/bruno assessment, detection heuristics, nixpkgs versions, auto-install vs offer tiers

**Status:** Not Started

---

### Unit 26.4: Database Migration Tool Module

**Description:** Implement the Database Migration Tool detection module with two integration levels: full devenv.nix packages for tools with system-level dependencies (Flyway, Prisma, diesel-cli, Atlas), and CLAUDE.md documentation-only for tools managed by their language package manager (Alembic, Drizzle, Knex). Generates `devenv task db:migrate` and `devenv task db:rollback` per detected tool.

**Context:** The key architectural insight from the migration tools research is that gdev should NOT choose a migration tool — it detects whichever tool the project already uses and removes friction. Friction takes two forms: missing system packages (Flyway needs JRE, diesel-cli needs libpq, Prisma needs native binaries) and missing workflow documentation (what is the migrate command for this tool?). gdev addresses both without imposing tool choices. Tools like Alembic and Drizzle are pure package manager installs — there is nothing useful gdev can add to devenv.nix beyond what `pip install` or `npm install` already handles. For these, CLAUDE.md documentation is the right integration level.

Detection must not conflict with language ecosystem detection. Prisma is a Node.js tool — the Node.js ecosystem module (Phase 2) may already add `prisma` to the environment. diesel-cli is a Rust tool — the Rust module may install it. The migration module should check what language modules have already detected before adding redundant packages.

**Code-Grounded Note:** Flyway community edition (Apache 2.0 core, some features moved to paid tiers in 2025) is in nixpkgs as `flyway`. diesel-cli is `diesel-cli` in nixpkgs with feature flags for database backends (postgres, mysql, sqlite). Atlas is `atlas` in nixpkgs. Prisma's native engines are pre-compiled binaries; nixpkgs' `prisma` package includes them. devenv.nix `devenv.tasks` is the target for task generation — introduced in devenv 2.0, confirmed as the canonical task runner for gdev projects (the DX polish spike explicitly rejected adding just/Taskfile/mise as separate task runners).

**Desired Outcome:** Projects with Flyway get `flyway` in devenv.nix and `devenv task db:migrate` targeting the detected migration directory. Projects with Alembic get a CLAUDE.md section documenting `alembic upgrade head` and `alembic downgrade -1`. All migrations tool paths appear in CLAUDE.md so Claude Code agents can run migrations correctly.

**Steps:**

1. Create `internal/modules/db-migration/db_migration.go` implementing `EcosystemModule`:
   ```go
   type DBMigrationModule struct{}

   func (m *DBMigrationModule) Name() string    { return "db-migration" }
   func (m *DBMigrationModule) IsLanguage() bool { return false }
   func (m *DBMigrationModule) RunOrder() int   { return RunOrderPostLanguage }  // after language modules
   ```

2. Define integration levels:
   ```go
   type MigrationIntegration int
   const (
       IntegrationFullPackage MigrationIntegration = iota  // devenv.nix packages + tasks + CLAUDE.md
       IntegrationDocOnly                                    // CLAUDE.md only
   )

   type MigrationTool struct {
       Name        string
       Integration MigrationIntegration
       NixPackage  string  // empty if IntegrationDocOnly
   }
   ```

3. Implement `Detect(projectRoot string) DetectionResult`:
   - **Full Package integration detections**:
     - Flyway: `flyway.conf` OR `db/migration/V*.sql` pattern OR Maven/Gradle `flyway-core` dependency
     - Prisma: `prisma/schema.prisma` file
     - diesel-cli: `diesel.toml` in project root
     - Atlas: `atlas.hcl` OR `schema.hcl` in project root
     - goose: `migrations/*.sql` with goose-format header (`-- +goose Up`) OR `goose` in go.mod
     - golang-migrate: `.sqlx/` directory OR `github.com/golang-migrate/migrate` in go.mod
     - dbmate: `db/migrations/` directory with `-- migrate:up` comment pattern
   - **Doc-only detections**:
     - Alembic: `alembic.ini` OR `alembic/` directory with `env.py`
     - Drizzle: `drizzle.config.ts` OR `drizzle.config.js` in project root
     - Knex: `knexfile.ts` OR `knexfile.js` in project root
     - Liquibase: `liquibase.properties` OR `changelog.xml` / `changelog.yaml`
     - EF Core: `.csproj` files containing `Microsoft.EntityFrameworkCore.Tools`
   - Return detection map: `{flyway: {detected: true, integration: Full}, alembic: {detected: true, integration: DocOnly}, ...}`
   - Conflict check: if Prisma already added by Node.js language module, mark as `alreadyInPackages: true`

4. Implement `Generate(detected DetectionResult) GenerationResult`:
   - Build packages from full-integration detected tools (skip `alreadyInPackages` ones):
     ```nix
     # Database migration tools
     pkgs.flyway        # Java-based migrations (Flyway Community)
     pkgs.diesel-cli    # Rust migration CLI (libpq, libmysqlclient included via devenv)
     pkgs.atlas         # Declarative schema management
     ```
   - Flyway requires JRE. Add `pkgs.jre` if not already present from a language module.
   - diesel-cli database backend note:
     ```nix
     # diesel-cli compiled with postgres+mysql+sqlite backends
     # Requires libpq (provided by services.postgres if enabled)
     pkgs.diesel-cli
     ```

5. Generate devenv task definitions. Use discovered migration directories:
   ```nix
   devenv.tasks = {
     # Flyway tasks
     "db:migrate" = {
       exec = "flyway migrate";
       description = "Run pending Flyway migrations";
     };
     "db:rollback" = {
       exec = "flyway undo";
       description = "Undo last Flyway migration (requires Teams edition for undo)";
     };
     "db:status" = {
       exec = "flyway info";
       description = "Show Flyway migration status";
     };
   };
   ```
   - Alembic tasks (when detected):
     ```nix
     "db:migrate" = {
       exec = "alembic upgrade head";
       description = "Apply all pending Alembic migrations";
     };
     "db:rollback" = {
       exec = "alembic downgrade -1";
       description = "Revert last Alembic migration";
     };
     ```
   - If multiple migration tools detected: use primary tool for bare `db:migrate`, prefix others: `db:flyway:migrate`, `db:alembic:migrate`

6. Generate CLAUDE.md migration section with all detected tools:
   ```markdown
   ## Database Migrations

   **Flyway** (SQL files in `db/migration/`):
   - Run: `devenv task db:migrate` or `flyway migrate`
   - Status: `devenv task db:status` or `flyway info`
   - New migration: create `db/migration/V<version>__<description>.sql`

   **Alembic** (Python, managed by pip):
   - Run: `devenv task db:migrate` or `alembic upgrade head`
   - Revert: `devenv task db:rollback` or `alembic downgrade -1`
   - New migration: `alembic revision --autogenerate -m "<description>"`
   - Config: `alembic.ini`, migrations in `alembic/versions/`
   ```

7. Handle conflicting migration tools (two tools in same project):
   - Both Prisma and Alembic in a polyglot project → both documented in CLAUDE.md
   - Primary task `db:migrate` goes to the tool matching the primary detected language
   - Both tools get prefixed tasks

8. Write unit tests:
   - `flyway.conf` → Flyway detected as full-integration
   - `prisma/schema.prisma` → Prisma detected as full-integration
   - `diesel.toml` → diesel-cli detected as full-integration
   - `atlas.hcl` → Atlas detected as full-integration
   - `alembic.ini` → Alembic detected as doc-only
   - `drizzle.config.ts` → Drizzle detected as doc-only
   - `knexfile.ts` → Knex detected as doc-only
   - `Microsoft.EntityFrameworkCore.Tools` in `.csproj` → EF Core detected as doc-only
   - Flyway detected → packages include `flyway` and `jre`
   - Prisma already in packages (from Node module) → not re-added
   - Generated tasks include `db:migrate` and `db:rollback`
   - Alembic doc-only → no packages added, CLAUDE.md section generated

**Acceptance Criteria:**
- [ ] Module implements `EcosystemModule` interface with `IsLanguage() = false` and `RunOrder() = RunOrderPostLanguage`
- [ ] Full-package integration: Flyway, Prisma, diesel-cli, Atlas, goose, golang-migrate, dbmate — devenv.nix packages + tasks + CLAUDE.md
- [ ] Doc-only integration: Alembic, Drizzle, Knex, Liquibase, EF Core — CLAUDE.md only, no packages
- [ ] Flyway and Atlas package installations include JRE dependency (with explanatory comment)
- [ ] diesel-cli package includes note about libpq dependency and devenv service integration
- [ ] Prisma package not added if already detected by Node.js language module
- [ ] `devenv task db:migrate` and `devenv task db:rollback` generated per detected tool
- [ ] Multiple migration tools: bare `db:migrate` goes to primary language's tool, others prefixed
- [ ] CLAUDE.md section documents all detected tools with migration commands, file paths, and naming conventions
- [ ] Unit tests cover full-integration detection, doc-only detection, conflict avoidance with language modules, generated tasks, multi-tool handling

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/api-db-mcp-research.md` — migration tool landscape, two integration levels, detection heuristics, nixpkgs package names, language ecosystem conflict analysis

**Status:** Not Started

---

### Unit 26.5: Module Integration & Detection Priority

**Description:** Integrate the four non-language modules into the main detection pipeline, ensure they run after language modules to avoid conflicts, wire them into the wizard customize path (never quick path), and write an integration test covering a project with mixed signals.

**Context:** Non-language modules are supplementary: they enrich the generated environment with tools that are project-specific but not language-specific. They must not compete with language module detection (Prisma is a Node.js concern first; diesel-cli is a Rust concern first). Running non-language modules after language modules gives them access to what was already detected, enabling the conflict avoidance logic in Unit 26.4. The `EcosystemModule` interface's `RunOrder()` method controls sequencing in the detection engine.

All four non-language modules share the same wizard placement policy: they are supplementary and never appear in the wizard quick path. Quick path is reserved for language runtimes and Tier 1 services. Non-language tool suggestions appear at the end of the customize path under a "Project Tools" section, pre-checked when strongly detected.

**Code-Grounded Note:** The Phase 1 detection engine in `internal/detect/detect.go` processes modules in run-order sequence. Two `RunOrder` values are needed: `RunOrderStandard` (language modules, Tier 1 services) and `RunOrderPost` (non-language tool modules). The wizard customize form in `addons/devinit/wizard.go` must gain a "Project Tools" section rendered after the language and service sections.

**Desired Outcome:** A project with `.proto` files + `openapi.yaml` + `mkdocs.yml` + `.github/` correctly detects all three non-language modules without conflicts, outputs correct devenv.nix packages, and presents the detected tools in the wizard customize path under "Project Tools".

**Steps:**

1. Add `RunOrder` constants to `internal/modules/types.go`:
   ```go
   const (
       RunOrderStandard    = 10  // language modules, Tier 1 services
       RunOrderPost        = 20  // non-language tool modules (after language modules complete)
       RunOrderPostLanguage = 20  // alias for clarity in migration module
   )
   ```

2. Update the Phase 1 detection engine to honor run order:
   ```go
   func DetectAll(projectRoot string) *DetectionResults {
       results := &DetectionResults{}

       // Pass 1: Standard run-order (language + Tier 1 services)
       for _, mod := range sortByRunOrder(AllModules, RunOrderStandard) {
           result := mod.Detect(projectRoot)
           results.Add(mod, result)
       }

       // Pass 2: Post run-order (non-language tools, can see Pass 1 results)
       for _, mod := range sortByRunOrder(AllModules, RunOrderPost) {
           result := mod.DetectWithContext(projectRoot, results)
           results.Add(mod, result)
       }

       return results
   }
   ```
   - `DetectWithContext` variant receives prior detection results for conflict avoidance

3. Register all four non-language modules in `internal/modules/registry.go`:
   ```go
   var NonLanguageModules = []EcosystemModule{
       &gitplatform.GitPlatformModule{},
       &docstools.DocsToolsModule{},
       &apitools.APIToolsModule{},
       &dbmigration.DBMigrationModule{},
   }

   var AllModules = append(LanguageModules, NonLanguageModules...)
   ```

4. Update wizard customize path to include "Project Tools" section:
   - Add new section after Languages, Services, and Security in the customize form
   - Section header: "Project Tools (detected from config files)"
   - Pre-check detected tools; uncheck undetected tools
   - Form renders four sub-groups matching module WizardGroup():
     ```
     ── Git Platform ────────────────────────────────────────
     [x] gh (GitHub CLI)       (detected: .github/ directory)
     [ ] glab (GitLab CLI)
     [x] git-lfs               (detected: .gitattributes filter=lfs)

     ── Documentation Tools ─────────────────────────────────
     [x] mkdocs + mkdocs-material  (detected: mkdocs.yml)
     [ ] mdbook
     [ ] d2

     ── API Tools ───────────────────────────────────────────
     [x] grpcurl + buf         (detected: *.proto files)
     [x] openapi-generator + redocly  (detected: openapi.yaml)
     [ ] bruno-cli

     ── Database Migrations ─────────────────────────────────
     [ ] flyway
     [ ] diesel-cli
     [ ] prisma (managed by npm)
     ```

5. Ensure non-language modules respect the `--answers-file` bypass:
   - `--answers-file` can specify non-language tools explicitly:
     ```yaml
     project_tools:
       git_platform:
         - gh
         - git-lfs
       docs_tools:
         - mkdocs
       api_tools:
         - grpc   # installs grpcurl + buf
       db_migrations:
         - flyway
     ```
   - When `project_tools` is absent from answers file, non-language detection runs normally

6. Write integration test: mixed-signal project:
   - Create test fixture with: `*.proto` file, `openapi.yaml`, `mkdocs.yml`, `.github/` directory
   - Run `DetectAll()` on the fixture
   - Assert: GitPlatformModule detects `gh`; DocsToolsModule detects mkdocs; APIToolsModule detects both gRPC and OpenAPI; DBMigrationModule detects nothing
   - Assert: no conflicts between modules (no duplicate package entries)
   - Assert: generated devenv.nix packages include `gh`, `mkdocs`, `grpcurl`, `buf`, `openapi-generator-cli`, `redocly`
   - Assert: generated devenv task definitions include `docs:build`, `docs:serve`
   - Assert: pre-commit hooks include buf-lint and redocly-lint
   - Assert: wizard customize path includes "Project Tools" section with `gh` and mkdocs pre-checked, others unchecked

7. Verify non-interference with existing modules:
   - A pure Go project with no non-language signals: non-language modules add nothing
   - A Node.js project with Prisma: Prisma detected by DB migration module as full-integration, but conflict check sees Node.js module already handled it → package not re-added
   - A Rust project with diesel-cli: diesel-cli detected as full-integration by DB migration module, not added by Rust language module → packages added correctly

8. Write unit tests for run-order sequencing:
   - Language modules always run before non-language modules
   - Non-language modules receive prior detection context
   - Detection engine correctly separates pass 1 and pass 2 execution

**Acceptance Criteria:**
- [ ] `RunOrderStandard` and `RunOrderPost` constants defined; detection engine executes in two passes
- [ ] All four non-language modules registered in `AllModules` with `RunOrder = RunOrderPost`
- [ ] Non-language modules receive prior language detection results via `DetectWithContext`
- [ ] Wizard customize path gains "Project Tools" section after Languages/Services/Security
- [ ] Detected non-language tools pre-checked in wizard; undetected tools unchecked
- [ ] Wizard never places non-language tool suggestions in quick path
- [ ] `--answers-file` `project_tools` key accepted for non-interactive non-language tool configuration
- [ ] Integration test: project with `.proto` + `openapi.yaml` + `mkdocs.yml` + `.github/` → all 3 non-language modules detected correctly without conflicts
- [ ] No duplicate packages: Prisma conflict check prevents double-addition when Node.js module runs first
- [ ] Pure language-only project: non-language modules add nothing (no false positives)
- [ ] Run-order unit tests confirm language modules always precede non-language modules

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/coverage-matrix-research.md` — detection architecture, run-order rationale, wizard placement policy
- `phases/01-foundation-shared-infrastructure.md` — `EcosystemModule` interface, detection engine, run-order mechanism
- `phases/06-wizard-orchestration.md` — customize path structure, pre-check logic, answers-file format

**Status:** Not Started

---

## Code-Grounded Implementation Notes

### Existing Interfaces to Implement

| Interface | Location | New Methods Required |
|-----------|----------|---------------------|
| `EcosystemModule` | `internal/modules/types.go` | `IsLanguage() bool`, `RunOrder() int` — add to interface if not present |
| `DetectWithContext()` | `internal/modules/types.go` | New optional method variant; detection engine calls it for post-order modules |

### New Module Packages

| Module | Package | RunOrder | WizardGroup |
|--------|---------|----------|-------------|
| `GitPlatformModule` | `internal/modules/git-platform` | `RunOrderPost` | "Git Platform" |
| `DocsToolsModule` | `internal/modules/docs-tools` | `RunOrderPost` | "Documentation Tools" |
| `APIToolsModule` | `internal/modules/api-tools` | `RunOrderPost` | "API Tools" |
| `DBMigrationModule` | `internal/modules/db-migration` | `RunOrderPostLanguage` | "Database Migrations" |

### Package Dependencies

All packages listed below are confirmed in nixpkgs as of May 2026:
- `gh` (GitHub CLI)
- `glab` (GitLab CLI)
- `git-lfs`
- `lazygit` (catalog only)
- `delta` (catalog only)
- `mkdocs`
- `python3Packages.mkdocs-material`
- `mdbook`
- `d2`
- `plantuml` (requires `jre`)
- `adr-tools`
- `grpcurl`
- `buf`
- `openapi-generator-cli` (requires `jre`)
- `redocly`
- `bruno-cli`
- `flyway` (requires `jre`)
- `diesel-cli`
- `atlas`
- `prisma` (conflict-checked against Node.js language module)

### Wizard Form Integration

Non-language tool suggestions are placed in a new "Project Tools" section at the end of the customize form. They are pre-checked when detected (strong signal), unchecked otherwise. They never appear in the quick path. The quick path continues to present only language selection and Tier 1 service detection.

---

## Phase Completion Criteria

- [ ] All five units pass acceptance criteria
- [ ] All four non-language modules implement `EcosystemModule` interface with correct `IsLanguage() = false` and `RunOrder = RunOrderPost`
- [ ] Detection pipeline two-pass execution confirmed: language modules complete before non-language modules start
- [ ] Integration test: mixed-signal project (proto + openapi + mkdocs + .github/) detects all modules correctly with no package conflicts
- [ ] Wizard customize path "Project Tools" section renders with detected tools pre-checked
- [ ] `devenv task docs:build`, `devenv task docs:serve` generated for each detected documentation tool
- [ ] `devenv task db:migrate`, `devenv task db:rollback` generated for each detected migration tool
- [ ] Pre-commit hooks added for buf (gRPC) and redocly (OpenAPI) when respective tools detected
- [ ] Conflict avoidance: Prisma not double-added when already detected by Node.js language module
- [ ] CLAUDE.md sections for all detected non-language tools include specific command examples
- [ ] `--answers-file` `project_tools` key accepted for all four module categories
- [ ] False-positive avoidance: boto3 alone does not trigger MinIO (Phase 25 related); `gh` not added to projects without `.github/` or github.com remote
