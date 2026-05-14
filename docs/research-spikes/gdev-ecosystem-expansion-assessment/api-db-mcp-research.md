# API Development Tools, Database Migration Frameworks & MCP Server Ecosystem Expansion

## Research Question

What API dev tools, DB migration tools, and MCP servers should gdev provide for consulting engineers? This assessment covers three gap categories from the coverage matrix: Category D (API Development & Testing), Category E (Database Migration & Schema Management), and Category J (MCP Server Ecosystem Expansion).

---

## 1. API Development & Testing Tools

### 1.1 HTTP Clients

| Tool | Description | Nixpkgs | Commonly Needed | gdev Action | Detection Heuristic |
|------|------------|---------|-----------------|-------------|---------------------|
| **curl** | Universal HTTP client, already ubiquitous | `curl` (always present) | Universal | No action needed | N/A -- always present |
| **httpie** | Human-friendly CLI HTTP client, colorized output | `httpie` (3.2.4) | High -- preferred by many devs over raw curl | **Detect & offer** | General API project (openapi.yaml, swagger.json, .http files) |
| **xh** | Rust rewrite of httpie, faster, single binary | `xh` (0.25.3) | Medium -- gaining traction as httpie alternative | Skip (httpie sufficient) | N/A |
| **bruno** | Open-source Postman alternative, Git-native collections | `bruno` (2.14.2) + `bruno-cli` | High for teams -- replaces Postman | **Detect & offer** | `.bru` files, `bruno.json`, `environments/` |
| **hurl** | HTTP file-based testing, Rust/libcurl, CI-friendly | `hurl` (7.1.0) | Medium -- growing in CI/CD pipelines | **Detect & offer** | `.hurl` files |

**Assessment**: httpie and bruno are the strongest candidates. httpie is the ergonomic curl replacement that consulting engineers actually want day one. Bruno is the clear winner for teams replacing Postman -- its Git-native storage (`.bru` files) aligns perfectly with gdev's version-control-first philosophy. xh is technically superior to httpie but lacks the ecosystem and name recognition; not worth the complexity of offering both.

### 1.2 GraphQL Tools

| Tool | Description | Nixpkgs | Commonly Needed | gdev Action | Detection Heuristic |
|------|------------|---------|-----------------|-------------|---------------------|
| **altair** | Feature-rich GraphQL client IDE | `altair` (8.3.0) | Medium -- useful for GraphQL projects | **Detect & offer** | `.graphql` files, `schema.graphql`, graphql deps in package.json |
| **graphqurl** | CLI for GraphQL queries | `graphqurl` (2.0.0) | Low -- niche | Skip | N/A |

**Assessment**: Altair is the only GraphQL tool worth detecting. graphqurl is too niche. GraphQL Playground is deprecated in favor of Apollo Sandbox/Altair. Detection via `.graphql` files or graphql dependencies in package.json is straightforward.

### 1.3 gRPC & Protobuf Tools

| Tool | Description | Nixpkgs | Commonly Needed | gdev Action | Detection Heuristic |
|------|------------|---------|-----------------|-------------|---------------------|
| **grpcurl** | curl for gRPC, reflection and proto file support | `grpcurl` (1.9.3) | High for gRPC projects | **Detect & install** | `.proto` files, `buf.yaml` |
| **buf** | Modern protobuf toolchain (lint, format, breaking change detection) | `buf` (1.59.0) | High for gRPC projects | **Detect & install** | `buf.yaml`, `buf.gen.yaml`, `buf.work.yaml` |
| **evans** | Interactive gRPC REPL client | `evans` (0.10.11) | Low -- niche interactive use | Skip | N/A |

**Assessment**: grpcurl and buf should be auto-detected together. Any project with `.proto` files benefits from both. buf is particularly valuable as it replaces a fragmented protobuf toolchain (protoc + various plugins) with a single coherent tool. Evans is interactive-only and too niche for default installation.

### 1.4 OpenAPI/Swagger Tools

| Tool | Description | Nixpkgs | Commonly Needed | gdev Action | Detection Heuristic |
|------|------------|---------|-----------------|-------------|---------------------|
| **openapi-generator-cli** | Generate client SDKs, server stubs, docs from OpenAPI specs | `openapi-generator-cli` (7.17.0) | High for API-first projects | **Detect & offer** | `openapi.yaml`, `openapi.json`, `swagger.yaml`, `swagger.json` |
| **redocly** | OpenAPI linting, bundling, previewing | `redocly` (2.11.1) | Medium -- growing alternative to spectral | **Detect & offer** | Same as above + `.redocly.yaml` |
| **swagger-codegen** | Older generation code generator | `swagger-codegen` (2.4.45) | Low -- superseded by openapi-generator | Skip | N/A |
| **spectral** | OpenAPI/AsyncAPI linting | NOT in nixpkgs | Medium | Skip (redocly covers this) | N/A |

**Assessment**: openapi-generator-cli and redocly are the two worth offering. swagger-codegen is the legacy predecessor to openapi-generator and should not be offered. Spectral is not in nixpkgs and redocly provides equivalent linting capability. Detection via `openapi.yaml`/`swagger.yaml` files is reliable.

### 1.5 API Testing Frameworks

| Tool | Description | Nixpkgs | Commonly Needed | gdev Action | Detection Heuristic |
|------|------------|---------|-----------------|-------------|---------------------|
| **newman** | Postman collection runner (CLI) | `newman` (6.2.1) | Medium -- declining as Bruno rises | **Detect & offer** | `.postman_collection.json`, `postman/` directory |
| **k6** | Load testing tool (Grafana) | `k6` (1.4.0) | Medium -- standard for load testing | **Detect & offer** | `k6/` directory, `*.k6.js` files |
| **pact** | Contract testing framework | NOT in nixpkgs | Low -- specialized | Skip | N/A |
| **dredd** | API testing against OpenAPI spec | NOT in nixpkgs | Low -- declining | Skip | N/A |

**Assessment**: Newman is worth detecting for legacy Postman collections, but Bruno CLI is the forward-looking replacement. k6 is the dominant load testing tool and worth offering when load test files are detected. Pact (contract testing) and Dredd are too specialized and not in nixpkgs.

### 1.6 API Tools Recommendation Summary

**Tier 1 -- Detect & install automatically** (when project signals are present):
- `grpcurl` + `buf` (triggered by `.proto` files or `buf.yaml`)

**Tier 2 -- Detect & offer** (prompt engineer to enable):
- `httpie` (general API projects)
- `bruno` + `bruno-cli` (`.bru` files detected)
- `hurl` (`.hurl` files detected)
- `openapi-generator-cli` + `redocly` (OpenAPI spec files detected)
- `altair` (GraphQL files detected)
- `k6` (load test files detected)
- `newman` (Postman collection files detected)

**Skip** (too niche, not in nixpkgs, or superseded):
- xh, graphqurl, evans, swagger-codegen, spectral, pact, dredd

---

## 2. Database Migration & Schema Management

### 2.1 The Core Question: gdev Concern vs Project Concern?

Migration tools are deeply per-project. A Go project uses goose or golang-migrate; a TypeScript project uses Prisma or Drizzle; a Java project uses Flyway. **gdev should NOT pick a migration tool for the project** -- that decision belongs to the team.

However, gdev can provide value in two ways:
1. **Detect the migration tool and ensure its CLI is available in devenv.nix** -- many migration CLIs have system-level dependencies (JVM for Flyway, specific libpq versions for sqlx-cli, etc.) that devenv.nix should handle.
2. **Document the detected migration tool in CLAUDE.md** -- so Claude Code agents understand the project's migration workflow (where migrations live, how to run them, naming conventions).

### 2.2 Migration Tools by Ecosystem

#### JVM Ecosystem

| Tool | Approach | Nixpkgs | gdev Action |
|------|----------|---------|-------------|
| **Flyway** | Versioned SQL files | `flyway` (11.14.1) | Detect (Maven/Gradle plugin or `flyway.conf`) and add to devenv.nix |
| **Liquibase** | Versioned changelogs (XML/YAML/JSON/SQL) | `liquibase` (5.0.1) | Detect (`liquibase.properties`, `changelog.xml`) and add to devenv.nix |

**Notes**: Flyway is simpler (just numbered SQL files). Liquibase supports more formats and has built-in rollback generation. Flyway Community Edition lost some features when Redgate pushed toward Enterprise in 2025, but Apache 2.0 core remains free. Both are well-established -- most JVM consulting projects use one or the other.

#### Go Ecosystem

| Tool | Approach | Nixpkgs | gdev Action |
|------|----------|---------|-------------|
| **goose** | Versioned SQL or Go migrations | `goose` (3.26.0) | Detect (`migrations/` with `.sql` + Go module) |
| **golang-migrate** | Versioned SQL migrations | `go-migrate` (4.19.0) | Detect (`migrate` CLI usage or import path) |
| **Atlas** | Declarative HCL (Terraform-like) | `atlas` (0.38.0) | Detect (`atlas.hcl`, `schema.hcl`) |

**Notes**: goose and golang-migrate are the dominant pair. Atlas is gaining ground with its declarative approach and is the most modern option. For greenfield Go projects, Atlas offers computed rollbacks and schema drift detection that the others lack.

#### JavaScript/TypeScript Ecosystem

| Tool | Approach | Nixpkgs | gdev Action |
|------|----------|---------|-------------|
| **Prisma** | Declarative schema-first | `prisma` (6.18.0) | Detect (`prisma/schema.prisma`) |
| **Drizzle** | TypeScript-first, SQL-like API | npm only (no nix pkg) | Detect (`drizzle.config.ts`) -- npm handles CLI |
| **Knex** | Versioned JS/TS migrations | npm only | Detect (`knexfile.ts/js`) -- npm handles CLI |
| **TypeORM** | Decorator-based migrations | npm only | Detect (`ormconfig.json`, `data-source.ts`) -- npm handles CLI |
| **Sequelize** | Model-based migrations | npm only | Detect (`sequelize-cli` in package.json) -- npm handles CLI |

**Notes**: Prisma is the only one that benefits from gdev installing a system package (prisma-engines require native binaries). The rest are pure npm packages where `npm install` handles everything. gdev's role for JS/TS migration tools is primarily **detection and CLAUDE.md documentation**, not installation.

#### Python Ecosystem

| Tool | Approach | Nixpkgs | gdev Action |
|------|----------|---------|-------------|
| **Alembic** | Versioned Python migrations (SQLAlchemy) | pip/poetry (no system pkg needed) | Detect (`alembic/`, `alembic.ini`) -- document in CLAUDE.md |
| **Django migrations** | Built into Django | Part of Django | Detect (Django project structure) -- document in CLAUDE.md |

**Notes**: Python migration tools are installed via pip/poetry within the project virtualenv. gdev should detect them and document migration commands in CLAUDE.md but does not need to add system packages.

#### .NET Ecosystem

| Tool | Approach | Nixpkgs | gdev Action |
|------|----------|---------|-------------|
| **EF Core migrations** | Built into Entity Framework | dotnet tool | Detect (`.csproj` with EF references) -- document in CLAUDE.md |

#### Rust Ecosystem

| Tool | Approach | Nixpkgs | gdev Action |
|------|----------|---------|-------------|
| **diesel-cli** | Versioned SQL migrations | `diesel-cli` (2.3.3) | Detect (`diesel.toml`) and add to devenv.nix |
| **sqlx-cli** | Versioned SQL migrations (compile-time checked) | `sqlx-cli` (0.8.6) | Detect (`sqlx-data.json`, `.sqlx/`) and add to devenv.nix |
| **sea-orm-cli** | Code-generation and migrations | `sea-orm-cli` (1.1.19) | Detect (sea-orm in Cargo.toml) and add to devenv.nix |

**Notes**: Rust migration CLIs have native dependencies (libpq, libmysqlclient, libsqlite3) that devenv.nix should provide. This is where gdev adds genuine value -- ensuring these system libraries are available.

#### Ruby Ecosystem

| Tool | Approach | Nixpkgs | gdev Action |
|------|----------|---------|-------------|
| **ActiveRecord** | Built into Rails | Part of Rails | Detect (Rails project) -- document in CLAUDE.md |

#### PHP Ecosystem

| Tool | Approach | Nixpkgs | gdev Action |
|------|----------|---------|-------------|
| **Laravel migrations** | Built into Laravel | Part of Laravel | Detect (Laravel project) -- document in CLAUDE.md |
| **Doctrine migrations** | Standalone migration package | Composer | Detect (`doctrine-migrations.yaml`) -- document in CLAUDE.md |

#### Schema-as-Code (Cross-Language)

| Tool | Approach | Nixpkgs | gdev Action |
|------|----------|---------|-------------|
| **Atlas** | Declarative HCL | `atlas` (0.38.0) | See Go section above |
| **dbmate** | Versioned SQL, language-agnostic | `dbmate` (2.28.0) | Detect (`db/migrations/` with dbmate format) |

**Notes**: dbmate is worth calling out separately as it is deliberately language-agnostic -- a good choice when the migration tool should not be tied to the application language. It is a single Go binary with no runtime dependencies.

### 2.3 Migration Tools Recommendation Summary

**gdev should detect and install** (system packages with native dependencies):
- `flyway`, `liquibase` (JVM projects)
- `diesel-cli`, `sqlx-cli`, `sea-orm-cli` (Rust projects)
- `prisma` (Node.js/TypeScript projects)
- `atlas`, `dbmate` (language-agnostic)
- `goose`, `go-migrate` (Go projects)

**gdev should detect and document in CLAUDE.md** (installed by project package manager):
- Drizzle, Knex, TypeORM, Sequelize (npm)
- Alembic, Django migrations (pip/poetry)
- ActiveRecord (bundler)
- Laravel migrations, Doctrine (composer)
- EF Core migrations (dotnet)

**Detection heuristics** (config files, directory patterns):
- `prisma/schema.prisma` -> Prisma
- `flyway.conf`, `db/migration/V*.sql` -> Flyway
- `liquibase.properties`, `changelog.xml` -> Liquibase
- `atlas.hcl` -> Atlas
- `diesel.toml` -> diesel-cli
- `.sqlx/` -> sqlx-cli
- `alembic.ini`, `alembic/` -> Alembic
- `drizzle.config.ts` -> Drizzle
- `knexfile.ts` -> Knex
- `db/migrations/` with `-- migrate:up` -> dbmate

### 2.4 Key Insight

The value gdev provides for migration tools is NOT choosing them -- it is **removing friction**. When gdev detects a migration tool, it should:
1. Ensure the CLI binary is in the devenv.nix environment (for tools with system dependencies)
2. Ensure required native libraries are available (libpq, libmysqlclient, etc.)
3. Add migration-specific context to CLAUDE.md (migration directory, run commands, naming conventions)
4. Optionally add convenience scripts to devenv.nix (`enterShell` aliases like `migrate-up`, `migrate-down`)

This is a natural extension of the Phase 2 language ecosystem detection that already exists in the plan.

---

## 3. MCP Server Ecosystem Expansion

### 3.1 Current State

gdev configures 5 MCP servers: Context7, GitHub, Socket.dev, semble, PostgreSQL.

The critical constraint (per multiple 2026 sources): **the practical ceiling is 40 active tools across all servers**. Each MCP server exposes 5-15 tools. More than ~6 servers degrades agent accuracy as the LLM struggles with tool selection. Each server's tool descriptions add 4-6K input tokens per request.

Prior gdev research established a sweet spot of 3-6 servers. The current 5 are well-chosen. Expansion must be selective.

### 3.2 MCP Server Assessment

#### Database MCP Servers

| Server | Exists? | Quality | Security | Consulting Value | gdev Action |
|--------|---------|---------|----------|------------------|-------------|
| **PostgreSQL** | Yes (official reference) | Mature, archived from official repo | Read-only by default | High | Already configured |
| **MySQL** | Yes (community + multi-DB servers) | Mature community implementations | Configurable read/write | High when MySQL detected | **Detect & configure** |
| **MongoDB** | Yes (community + multi-DB servers) | Stable community | Configurable access | Medium | Detect & offer |
| **Redis** | Yes (official reference, archived) | Mature | Key-value access, less risk than SQL | Medium | Detect & offer |
| **SQLite** | Yes (official reference, archived) | Mature | File-based, local only | Medium | Detect & configure |
| **Multi-DB (anydb-mcp, PineMCP)** | Yes (community) | Mixed quality | Broad access surface | Low -- prefer single-DB servers | Skip |

**Assessment**: The PostgreSQL MCP is already configured. **MySQL MCP should be auto-configured when MySQL/MariaDB is detected as a devenv service** -- this is the #1 gap since MySQL is one of the 6 planned devenv services. SQLite MCP is worth enabling when `.sqlite`/`.db` files are present. Redis and MongoDB are lower priority since their data models are less amenable to SQL-style exploration. Multi-DB servers should be avoided -- they expand the attack surface and tool count unnecessarily.

**Recommendation**: Add MySQL MCP as auto-configured (parallel to PostgreSQL). SQLite MCP as detect-and-configure. Redis and MongoDB as optional addons only.

#### Cloud Provider MCP Servers

| Server | Exists? | Quality | Security | Consulting Value | gdev Action |
|--------|---------|---------|----------|------------------|-------------|
| **AWS MCP** | Yes, official (AWS) | Preview mode (2026) | Broad AWS access -- significant risk | High for AWS projects | **Optional addon** |
| **Azure MCP** | Yes, official (Microsoft) | Production-ready | Azure RBAC integration | Medium | Optional addon |
| **GCP MCP** | Not found as standalone | Unknown | N/A | Medium | Skip until exists |
| **Terraform MCP** | Yes, official (HashiCorp) | Beta | Registry + TFC workspace access | High for IaC projects | **Detect & offer** |

**Assessment**: Cloud provider MCP servers are powerful but carry significant security risk -- they provide AI agents with access to cloud resources. The Terraform MCP is the safest entry point because it primarily provides documentation/registry access rather than direct resource manipulation. The AWS MCP merges knowledge and API capabilities, which is valuable but must be opt-in only.

**Recommendation**: Terraform MCP should be detect-and-offer when `*.tf` files are present. AWS/Azure MCPs should be optional addons that require explicit engineer opt-in, never auto-configured. GCP does not have a mature standalone MCP server.

#### Ticketing & Project Management MCP Servers

| Server | Exists? | Quality | Security | Consulting Value | gdev Action |
|--------|---------|---------|----------|------------------|-------------|
| **Jira/Confluence (Atlassian)** | Yes, official (Atlassian) | GA (Feb 2026) | OAuth 2.1, respects existing permissions | Very high for consulting | **Optional addon** |
| **Linear** | Yes, official (Linear) | Mature | OAuth 2.1, supports read-only API keys | High for Linear teams | **Optional addon** |
| **Asana** | Community only | Low maturity | Token-based | Low | Skip |

**Assessment**: The Atlassian MCP server is the highest-value expansion candidate for a consulting org. Consulting engineers live in Jira. Being able to query issues, update status, and read Confluence pages directly from Claude Code without context-switching is a major productivity win. The server went GA in February 2026 with OAuth 2.1 and respects existing Jira permissions.

Linear MCP is equally mature but relevant only for teams using Linear (more common in startups than enterprise consulting).

**Recommendation**: Atlassian (Jira/Confluence) MCP should be a first-class optional addon in gdev. Linear MCP as a secondary optional addon. Both require explicit configuration with credentials -- never auto-enabled.

#### Communication MCP Servers

| Server | Exists? | Quality | Security | Consulting Value | gdev Action |
|--------|---------|---------|----------|------------------|-------------|
| **Slack** | Yes, official (Slack) | Mature, 12 partner integrations | OAuth via partner flow | Medium-high | **Optional addon** |

**Assessment**: Slack MCP enables searching messages, reading channel history, and posting messages. Useful for catching up on project context. However, it carries communication security concerns -- an AI agent with Slack access could inadvertently expose sensitive conversations. The Slack MCP supports Claude as a first-class partner.

**Recommendation**: Optional addon. Useful for consulting engineers who need to catch up on project discussions, but the security implications of giving AI agents Slack access require explicit opt-in and clear documentation of what it can read.

#### Observability MCP Servers

| Server | Exists? | Quality | Security | Consulting Value | gdev Action |
|--------|---------|---------|----------|------------------|-------------|
| **Datadog** | Yes, official | GA (March 2026) | API authentication required | High for Datadog shops | **Optional addon** |
| **Grafana** | Yes, official | Mature, token-optimized | Dashboard/datasource access | High for Grafana shops | **Optional addon** |
| **Sentry** | Yes, official (archived from reference repo) | Mature | Issue/event read access | High -- closes alert-to-fix loop | **Detect & offer** |
| **Prometheus** | Yes, community | Functional | PromQL query access | Low standalone | Skip (use via Grafana) |

**Assessment**: Sentry is the strongest observability candidate for gdev. It is focused (error tracking only), low-risk (read-only issue/event data), and directly useful for debugging. The "alert-to-fix" loop -- seeing an error in Sentry and fixing it in code without leaving the IDE -- is a compelling workflow. Datadog and Grafana are valuable but client-dependent; they should be optional addons configured per-engagement.

**Recommendation**: Sentry MCP as detect-and-offer (when Sentry DSN is found in env vars or config). Datadog and Grafana as optional addons.

#### CI/CD MCP Servers

| Server | Exists? | Quality | Security | Consulting Value | gdev Action |
|--------|---------|---------|----------|------------------|-------------|
| **GitHub Actions** | Via GitHub MCP | Mature (part of GitHub server) | Same as GitHub MCP | Already covered | Already configured |
| **GitLab CI** | Yes, official (GitLab) | Stable | GitLab API access | Medium for GitLab projects | **Optional addon** |
| **Azure DevOps** | Yes, official (Microsoft) | Actively developed | Directory-based auth | Medium for ADO projects | Optional addon |

**Assessment**: GitHub Actions debugging is already covered by the GitHub MCP server that gdev configures. GitLab CI MCP is worth offering as an optional addon for projects on GitLab. Azure DevOps MCP is relevant for Microsoft-stack consulting engagements.

**Recommendation**: GitLab MCP as optional addon when `.gitlab-ci.yml` is detected. Azure DevOps as optional addon. No new server needed for GitHub Actions.

#### Search MCP Servers

| Server | Exists? | Quality | Security | Consulting Value | gdev Action |
|--------|---------|---------|----------|------------------|-------------|
| **Brave Search** | Yes (archived reference) | Mature | API key required, 2,000 free queries/month | Medium | **Consider as default** |
| **Exa** | Yes (exa-labs) | Good | API key required | Medium | Skip (Brave sufficient) |

**Assessment**: Brave Search MCP enables Claude Code to perform live web searches during coding sessions. This is useful for looking up error messages, API documentation, and current best practices. The free tier (2,000 queries/month) is sufficient for individual developer use. However, Claude Code already has built-in web search capability via the WebSearch tool, which may make an additional search MCP redundant.

**Recommendation**: Consider adding Brave Search MCP as an optional default. Only add it if Claude Code's built-in web search proves insufficient for coding workflows. Do not add both Brave and Exa -- pick one search server at most.

#### File/Knowledge MCP Servers

| Server | Exists? | Quality | Security | Consulting Value | gdev Action |
|--------|---------|---------|----------|------------------|-------------|
| **Memory** | Yes (official reference) | Mature | Local knowledge graph | Low -- requires active prompting | Skip |
| **Filesystem** | Yes (official reference) | Mature | Directory-scoped access | Low -- Claude Code has native filesystem | Skip |

**Assessment**: Both Memory and Filesystem MCP servers are reference implementations designed for Claude Desktop, not Claude Code. Claude Code already has native filesystem access and its own context management. Adding these servers would waste tool slots without benefit.

**Recommendation**: Skip both. Claude Code does not need them.

#### Terraform/IaC MCP Servers

Covered in the Cloud Provider section above. The Terraform MCP server (official, HashiCorp, beta) provides registry documentation and Terraform Cloud workspace management. It is the highest-value IaC MCP server.

### 3.3 MCP Expansion Recommendation Summary

**Current baseline (5 servers, keep as-is)**:
1. Context7 (library documentation)
2. GitHub (repo management, issues, PRs, Actions)
3. Socket.dev (supply chain security)
4. semble (semantic code search)
5. PostgreSQL (database queries)

**Add as auto-detected** (when project signals present):
6. MySQL MCP -- when MySQL/MariaDB is a devenv service
7. SQLite MCP -- when `.sqlite`/`.db` files present

**Add as detect-and-offer** (prompt engineer to enable):
8. Terraform MCP -- when `*.tf` files present
9. Sentry MCP -- when Sentry DSN found in config

**Add as optional addons** (explicit engineer configuration):
10. Atlassian (Jira/Confluence) MCP -- highest-value addon for consulting
11. Linear MCP -- for teams using Linear
12. Slack MCP -- for project context catch-up
13. Datadog MCP -- for Datadog-using clients
14. Grafana MCP -- for Grafana-using clients
15. GitLab MCP -- for GitLab-hosted projects
16. AWS MCP -- for AWS-heavy projects (preview, security-sensitive)
17. Azure MCP -- for Azure-heavy projects

**Skip** (insufficient value or redundant):
- MongoDB MCP (niche data model for MCP)
- Redis MCP (key-value model less useful for AI exploration)
- Multi-DB servers (security risk, tool bloat)
- Brave Search / Exa (Claude Code has built-in web search)
- Memory / Filesystem (Claude Code has native equivalents)
- Prometheus (use via Grafana instead)
- Asana MCP (community only, low maturity)

### 3.4 Respecting the Tool Ceiling

With 5 current servers exposing roughly 40-60 tools, gdev is already near the practical ceiling. The key architectural decisions:

1. **Never enable more than ~6 servers simultaneously by default.** The 5 current defaults plus one detected database server is the maximum auto-enabled set.
2. **Use per-project MCP configuration.** Servers like Terraform, Sentry, Atlassian should be configured in `.claude/settings.local.json` at the project level, not globally.
3. **Profile-based server bundles.** gdev could offer MCP "profiles" (e.g., `mcp-aws-stack`, `mcp-observability`, `mcp-enterprise-pm`) that enable/disable coherent server groups.
4. **Provide `gdev mcp add/remove/list` commands.** Engineers should be able to manage their MCP server set without editing JSON files directly.

---

## 4. Cross-Cutting Findings

### 4.1 Detection is the Core Value

Across all three areas, the pattern is consistent: gdev's value is not in choosing tools for engineers but in **detecting what the project already uses and removing friction**. This means:
- Scanning for config files (`.proto`, `openapi.yaml`, `prisma/schema.prisma`, `*.tf`, etc.)
- Adding detected tools to devenv.nix with correct dependencies
- Documenting detected tools in CLAUDE.md so AI agents understand the project's toolchain
- Configuring relevant MCP servers based on detected technology stack

### 4.2 The Three-Tier Model

| Tier | Meaning | Examples |
|------|---------|---------|
| **Always install** | Every consulting engineer needs this | curl, httpie (already standard) |
| **Detect & configure** | Project signals indicate this tool is needed | grpcurl+buf (`.proto`), MySQL MCP (MySQL service), Prisma (schema.prisma) |
| **Optional addon** | Engineer explicitly enables for their workflow | Atlassian MCP, Slack MCP, Datadog MCP |

### 4.3 Security Gradient

MCP servers have a clear security gradient that should inform auto-configuration policy:
- **Low risk**: Database MCPs in read-only mode, Context7, semble -- safe to auto-configure
- **Medium risk**: Ticketing MCPs (Jira, Linear), observability MCPs (Sentry, Datadog) -- opt-in with existing permissions respected
- **High risk**: Cloud provider MCPs (AWS, Azure), communication MCPs (Slack) -- explicit engineer opt-in only, never auto-configured

---

## Sources

- [MCP Official Servers Repository](https://github.com/modelcontextprotocol/servers) -> `docs/mcp-official-servers-github.md`
- [Best MCP Servers in 2026 (MCP Bundles)](https://www.mcpbundles.com/blog/best-mcp-servers) -> `docs/mcp-bundles-best-servers-2026.md`
- [18 Best DevOps MCP Servers for 2026 (Medium/k8slens)](https://medium.com/k8slens/18-best-devops-mcp-servers-for-2026-the-definitive-guide-bfde04654a35) -> `docs/devops-mcp-servers-2026-medium.md`
- [Terraform MCP Server (HashiCorp)](https://developer.hashicorp.com/terraform/mcp-server) -> `docs/terraform-mcp-server-hashicorp.md`
- [15 MCP Servers for Claude Code & Cursor (Codersera)](https://codersera.com/blog/best-mcp-servers-claude-code-cursor-2026/) -> `docs/mcp-servers-claude-code-cursor-2026.md`
- [DB Migration Tools Comparison (Codelit)](https://codelit.io/blog/database-migration-tools-comparison) -> `docs/db-migration-tools-comparison-codelit.md`
- [Postman Alternatives (Better Stack)](https://betterstack.com/community/comparisons/postman-alternative/) -> `docs/postman-alternatives-betterstack-2026.md`
- [Slack MCP Server (Official)](https://slack.com/help/articles/48855576908307-Guide-to-the-Slack-MCP-server) -> `docs/slack-mcp-server-official.md`
- [Linear MCP Server (Official)](https://linear.app/docs/mcp) -> `docs/linear-mcp-server-official.md`
- [Datadog MCP Server Use Cases (Official)](https://www.datadoghq.com/blog/datadog-mcp-server-use-cases/) -> `docs/datadog-mcp-server-use-cases.md`
- [Atlassian MCP Server GA Announcement](https://www.mindstudio.ai/blog/atlassian-mcp-server-ga-claude-reads-writes-jira-confluence-compass-oauth)
- [Atlassian MCP Server GitHub](https://github.com/atlassian/atlassian-mcp-server)
- Nixpkgs package availability verified via `nix eval` and `nix search` on NixOS (2026-05-14)
