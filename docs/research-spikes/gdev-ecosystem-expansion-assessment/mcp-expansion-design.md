# MCP Server Ecosystem Expansion — Implementation Unit Design

## Overview

This document specifies implementation units for expanding gdev's MCP server configuration beyond the current 5-server baseline (Context7, GitHub, Socket.dev, semble, PostgreSQL). It amends Phase 4 (Unit 3.5: .mcp.json generation) and Phase 12 (Unit 12.8: MCP Server Curation) of the gdev-secure-devenv-bootstrap implementation plan.

The design adds 9 new MCP servers organized into a three-tier auto-configuration policy, enforces a 40-tool ceiling, introduces a security model matching risk to automation level, and integrates with the Phase 12 lifecycle management system (`gdev enable`/`gdev disable`).

## Three-Tier Auto-Configuration Policy

| Tier | Behavior | Servers |
|------|----------|---------|
| **Auto-detect** | Configure automatically when project signals present. Low risk, read-only. | MySQL MCP, SQLite MCP |
| **Detect-and-offer** | Wizard prompt when signals detected. Medium risk, requires opt-in. | Terraform MCP, Sentry MCP |
| **Optional catalog** | Explicit `gdev enable mcp-<name>` only. Shown in `gdev list --category ai-agent`. | Atlassian, Linear, Slack, Datadog, Grafana, GitLab, AWS, Azure |

## Security Model

| Tier | Risk Level | Auth Pattern | Automation Level | Examples |
|------|-----------|--------------|------------------|----------|
| Low | DB read-only access | Connection string from devenv services | Auto-configure | MySQL, SQLite, PostgreSQL |
| Medium | Reads project data (tickets, errors) | API key / OAuth token via SecretSpec | Detect-and-offer (wizard prompt) | Terraform, Sentry |
| High | Can take actions (cloud ops, messaging) | OAuth + explicit credential setup | Explicit `gdev enable` only | Atlassian, Linear, Slack, Datadog, Grafana |

## 40-Tool Ceiling Enforcement

Each MCP server exposes 5-15 tools. With 5 current servers at ~8 tools average, the baseline is ~40 tools — already at ceiling. New servers must displace or coexist within this budget.

**Enforcement mechanism:**
- `.devinit/.gdev-init-answers.yaml` tracks enabled MCP servers per-project
- `gdev enable mcp-<name>` checks current tool count before enabling; warns if total would exceed 40
- `gdev mcp list` shows active servers with tool counts and total
- `gdev mcp check` validates the current configuration against the ceiling
- Database MCP servers are mutually exclusive with each other in the auto-detect tier (only the detected DB type is enabled, never all three simultaneously)

---

## Per-Server Specifications

### MySQL MCP

| Field | Value |
|-------|-------|
| **npm package** | `@benborla29/mcp-server-mysql` (community, mature, MIT) |
| **Install method** | `npx` via .mcp.json |
| **Security tier** | Low (read-only database access) |
| **Tool count** | ~8 tools (query, describe table, list tables, list databases, schema inspection) |
| **Detection heuristic** | `services.mysql.enable = true` in devenv.nix, OR `mysql` / `mariadb` service in docker-compose.yml |
| **Required credentials** | `MYSQL_HOST`, `MYSQL_PORT`, `MYSQL_USER`, `MYSQL_PASSWORD`, `MYSQL_DATABASE` |
| **SecretSpec integration** | Auto-declared when MySQL service detected; defaults to devenv service credentials |

**.mcp.json configuration:**
```json
"mysql": {
  "command": "npx",
  "args": ["-y", "@benborla29/mcp-server-mysql"],
  "env": {
    "MYSQL_HOST": "127.0.0.1",
    "MYSQL_PORT": "3306",
    "MYSQL_USER": "root",
    "MYSQL_PASSWORD": "",
    "MYSQL_DATABASE": ""
  }
}
```

### SQLite MCP

| Field | Value |
|-------|-------|
| **npm package** | `@modelcontextprotocol/server-sqlite` (archived reference server from MCP org) |
| **Install method** | `npx` via .mcp.json |
| **Security tier** | Low (file-based, local only) |
| **Tool count** | ~6 tools (read_query, write_query, create_table, list_tables, describe_table, append_insight) |
| **Detection heuristic** | `*.sqlite` or `*.db` files in project root or `data/` directory |
| **Required credentials** | None (file path only) |
| **SecretSpec integration** | None needed |

**.mcp.json configuration:**
```json
"sqlite": {
  "command": "npx",
  "args": ["-y", "@modelcontextprotocol/server-sqlite", "--db-path", "./data/local.db"]
}
```

**Note:** The `--db-path` argument is templated from the detected database file path.

### Terraform MCP (HashiCorp Official)

| Field | Value |
|-------|-------|
| **npm package** | `@hashicorp/terraform-mcp-server` (official HashiCorp, beta) |
| **Install method** | `npx` via .mcp.json |
| **Security tier** | Medium (registry docs + Terraform Cloud workspace metadata) |
| **Tool count** | ~10 tools (provider docs search, module info, sentinel policies, workspace list/create/update/delete, run management) |
| **Detection heuristic** | `*.tf` files in project root, `terraform/` directory, `.terraform.lock.hcl` |
| **Required credentials** | `TFC_TOKEN` (Terraform Cloud API token, optional — registry browsing works without it) |
| **SecretSpec integration** | `TFC_TOKEN` declared as optional secret; provider `keyring` for local, `env` for CI |

**.mcp.json configuration:**
```json
"terraform": {
  "command": "npx",
  "args": ["-y", "@hashicorp/terraform-mcp-server"],
  "env": {
    "TFC_TOKEN": ""
  }
}
```

**Note:** Functional without TFC_TOKEN for registry/docs browsing. Token required only for Terraform Cloud workspace operations.

### Sentry MCP

| Field | Value |
|-------|-------|
| **npm package** | `@sentry/mcp-server` (official Sentry, archived from reference repo) |
| **Install method** | `npx` via .mcp.json |
| **Security tier** | Medium (read-only issue/event data) |
| **Tool count** | ~8 tools (list issues, get issue details, get event, list project issues, search issues, resolve issue, assign issue, get stacktrace) |
| **Detection heuristic** | `SENTRY_DSN` in `.env` / `.env.example`, `@sentry/*` in package.json, `sentry_sdk` in requirements.txt/pyproject.toml, Sentry init in source files |
| **Required credentials** | `SENTRY_AUTH_TOKEN` (API auth token with project read scope) |
| **SecretSpec integration** | `SENTRY_AUTH_TOKEN` declared when Sentry detected; provider `keyring` for local, `env` for CI |

**.mcp.json configuration:**
```json
"sentry": {
  "command": "npx",
  "args": ["-y", "@sentry/mcp-server"],
  "env": {
    "SENTRY_AUTH_TOKEN": ""
  }
}
```

### Atlassian MCP (Jira/Confluence)

| Field | Value |
|-------|-------|
| **npm package** | `@anthropic-ai/atlassian-mcp-server` (official Atlassian, GA Feb 2026) |
| **Install method** | `npx` via .mcp.json |
| **Security tier** | High (reads/writes Jira issues, Confluence pages; respects existing Jira permissions) |
| **Tool count** | ~15 tools (search issues, get issue, create issue, update issue, transition issue, add comment, list projects, search Confluence, get page, create page, list spaces, get Compass components) |
| **Detection heuristic** | None (optional catalog only) |
| **Required credentials** | OAuth 2.1 flow via Atlassian; stores tokens locally. Requires `ATLASSIAN_SITE_URL` and OAuth setup. |
| **SecretSpec integration** | `ATLASSIAN_SITE_URL` declared on enable; OAuth handled by server's built-in flow |

**.mcp.json configuration:**
```json
"atlassian": {
  "command": "npx",
  "args": ["-y", "@anthropic-ai/atlassian-mcp-server"],
  "env": {
    "ATLASSIAN_SITE_URL": ""
  }
}
```

**Note:** Highest consulting value among optional servers. Jira integration reduces context-switching for consulting engineers who live in Jira daily.

### Linear MCP

| Field | Value |
|-------|-------|
| **npm package** | N/A — Linear MCP is a centrally hosted managed service, not an npm package |
| **Install method** | Remote MCP server URL in .mcp.json |
| **Security tier** | High (creates/updates issues, reads project data) |
| **Tool count** | ~10 tools (find issues, create issue, update issue, create project, add comment, list teams, search, create label) |
| **Detection heuristic** | None (optional catalog only) |
| **Required credentials** | OAuth 2.1 via `https://mcp.linear.app/mcp` (dynamic client registration), or `LINEAR_API_KEY` for direct auth |
| **SecretSpec integration** | `LINEAR_API_KEY` declared on enable as optional (OAuth preferred) |

**.mcp.json configuration:**
```json
"linear": {
  "type": "url",
  "url": "https://mcp.linear.app/mcp"
}
```

**Note:** Uses streamable HTTP transport (remote MCP), not stdio. No local npm package needed. OAuth 2.1 with dynamic client registration handles auth in-client.

### Slack MCP

| Field | Value |
|-------|-------|
| **npm package** | N/A — Slack MCP uses partner OAuth flow via Anthropic integration |
| **Install method** | Remote MCP server via partner setup. Configuration varies by MCP client. |
| **Security tier** | High (reads channel history, can post messages — communication exposure risk) |
| **Tool count** | ~12 tools (search messages, search files, search channels, read channel history, send message, list channels, list members, get profile, create canvas, read canvas, search members, get channel info) |
| **Detection heuristic** | None (optional catalog only) |
| **Required credentials** | OAuth via Slack partner flow (Claude is a supported partner). No raw API key — auth is delegated. |
| **SecretSpec integration** | None (OAuth handled by partner flow) |

**.mcp.json configuration:**
```json
"slack": {
  "command": "npx",
  "args": ["-y", "@anthropic-ai/mcp-server-slack"],
  "env": {
    "SLACK_TEAM_ID": "",
    "SLACK_BOT_TOKEN": ""
  }
}
```

**Note:** The partner OAuth flow is preferred for Claude Desktop/claude.ai. For Claude Code (CLI), the npm server with bot token is the practical path. `SLACK_BOT_TOKEN` requires a Slack app with appropriate scopes. Documentation must clearly state what the AI agent can read.

### Datadog MCP

| Field | Value |
|-------|-------|
| **npm package** | `@anthropic-ai/mcp-server-datadog` (official Datadog, GA March 2026) |
| **Install method** | `npx` via .mcp.json |
| **Security tier** | High (reads metrics, logs, traces, APM data; can create monitors and dashboards) |
| **Tool count** | ~12 tools (query metrics, search logs, list monitors, get dashboard, create monitor, list services, get trace, query events, list incidents, get SLO status, search hosts, create dashboard) |
| **Detection heuristic** | None (optional catalog only) |
| **Required credentials** | `DD_API_KEY` and `DD_APP_KEY` (Datadog API + Application keys) |
| **SecretSpec integration** | `DD_API_KEY` and `DD_APP_KEY` declared on enable; provider `keyring` for local |

**.mcp.json configuration:**
```json
"datadog": {
  "command": "npx",
  "args": ["-y", "@anthropic-ai/mcp-server-datadog"],
  "env": {
    "DD_API_KEY": "",
    "DD_APP_KEY": "",
    "DD_SITE": "datadoghq.com"
  }
}
```

### Grafana MCP

| Field | Value |
|-------|-------|
| **npm package** | `mcp-grafana` (official Grafana, token-optimized) |
| **Install method** | `npx` via .mcp.json |
| **Security tier** | High (reads dashboards, datasources, incidents; can query Prometheus/Loki) |
| **Tool count** | ~10 tools (list dashboards, get dashboard, query datasource, list datasources, search dashboards, get alert rules, list incidents, get incident, query Prometheus, query Loki) |
| **Detection heuristic** | None (optional catalog only) |
| **Required credentials** | `GRAFANA_URL` and `GRAFANA_API_KEY` (service account token with viewer role minimum) |
| **SecretSpec integration** | `GRAFANA_URL` and `GRAFANA_API_KEY` declared on enable; provider `keyring` for local |

**.mcp.json configuration:**
```json
"grafana": {
  "command": "npx",
  "args": ["-y", "mcp-grafana"],
  "env": {
    "GRAFANA_URL": "",
    "GRAFANA_API_KEY": ""
  }
}
```

**Note:** Grafana's MCP server is explicitly optimized for token efficiency — structures responses to minimize context window usage. This is particularly valuable given the 40-tool ceiling concern.

---

## Implementation Units

These units amend Phase 4 (Unit 3.5) and Phase 12 (Unit 12.8) of the existing plan. The numbering follows the convention: 3.5.X for Phase 4 amendments, 12.8.X for Phase 12 amendments.

---

### Unit 3.5.1: MCP Server Registry and Tool Budget Tracking

**Description:** Extend the .mcp.json generation infrastructure (Unit 3.5) with a typed MCP server registry that tracks per-server metadata — tool count, security tier, detection heuristic, required credentials — and enforces the 40-tool ceiling at generation time.

**Context:** Unit 3.5 defined basic .mcp.json generation via struct marshaling for 5 hardcoded servers. The expansion to 14+ possible servers requires a registry-driven approach where each server is a self-describing entry. The registry is the foundation for three-tier auto-configuration: it provides the metadata needed to decide which servers to enable, which to offer, and which to skip. The 40-tool ceiling is the binding constraint — without tracking, an engineer enabling 4-5 optional servers could silently degrade Claude Code's tool selection accuracy.

**Desired Outcome:** A `McpServerRegistry` that stores all 14 server definitions (5 existing + 9 new), exposes per-server tool counts, and provides `CanEnable(server) (bool, warning)` that checks the budget before adding a server.

**Steps:**
1. Define `McpServerDef` struct:
   ```go
   type McpServerDef struct {
       Name             string              // "mysql", "terraform", "sentry", etc.
       DisplayName      string              // "MySQL MCP (Read-Only Database)"
       Category         string              // "database", "iac", "observability", "ticketing", "communication"
       SecurityTier     SecurityTier        // Low, Medium, High
       ToolCount        int                 // Approximate tools exposed (used for budget)
       ConfigPolicy     ConfigPolicy        // AutoDetect, DetectAndOffer, OptionalCatalog
       DetectFunc       func(*DetectedProject) bool
       McpEntry         func(answers) McpServerEntry  // Returns the .mcp.json block
       RequiredSecrets  []SecretDecl        // Credentials needed
       Description      string              // One-line for `gdev mcp list`
       DocSection       string              // CLAUDE.md documentation paragraph
   }
   ```
2. Define `SecurityTier` enum: `Low` (auto-configure), `Medium` (wizard prompt), `High` (explicit enable only).
3. Define `ConfigPolicy` enum: `AutoDetect`, `DetectAndOffer`, `OptionalCatalog`.
4. Implement `McpServerRegistry` with `Register()`, `Get()`, `List()`, `ListEnabled()`, `TotalToolCount()`.
5. Implement `CanEnable(name string, currentEnabled []string) (bool, string)`:
   - Compute current total tool count from enabled servers.
   - If adding the new server would exceed 40, return false with a descriptive warning.
   - If adding would exceed 35 (soft ceiling), return true with a warning.
6. Register all 5 existing servers (Context7, GitHub, Socket.dev, semble, PostgreSQL) with accurate tool counts.
7. Register all 9 new servers with the per-server specifications from this document.
8. Write unit tests: verify tool count computation, verify ceiling enforcement, verify detection heuristics.

**Acceptance Criteria:**
- [ ] All 14 servers registered with accurate metadata
- [ ] `TotalToolCount()` returns correct sum for any combination of enabled servers
- [ ] `CanEnable()` blocks additions that would exceed 40-tool ceiling
- [ ] `CanEnable()` warns (but allows) additions that would exceed 35-tool soft ceiling
- [ ] Each server's `McpEntry()` returns valid .mcp.json configuration
- [ ] Registry is extensible (new servers added by implementing `McpServerDef`, no core changes)

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/api-db-mcp-research.md § 3.4` — 40-tool ceiling, 3-6 server sweet spot
- `research-spikes/gdev-ecosystem-expansion-assessment/docs/mcp-servers-claude-code-cursor-2026.md` — per-server tool counts, 4-6K tokens per server
- `implementation-plans/gdev-secure-devenv-bootstrap/phases/04-claude-code-addon-core-generation.md § Unit 3.5` — existing .mcp.json generation

**Status:** Not Started

---

### Unit 3.5.2: Auto-Detect Database MCP Servers (MySQL, SQLite)

**Description:** Implement auto-detection and configuration of MySQL MCP and SQLite MCP servers, paralleling the existing PostgreSQL MCP pattern. These are low-risk, read-only database servers that configure automatically when the corresponding database is detected in the project.

**Context:** PostgreSQL MCP is already auto-configured when `services.postgres.enable = true` is detected. MySQL and SQLite fill the remaining database gaps. MySQL is one of the 6 planned devenv services — its absence from the MCP set is the single biggest gap in the current configuration. SQLite is ubiquitous in local development and testing. Both are read-only by default, matching the Low security tier.

**Desired Outcome:** When `gdev init` detects MySQL/MariaDB as a devenv service or docker-compose service, the MySQL MCP server appears in .mcp.json with connection credentials from the service configuration. When `*.sqlite` or `*.db` files are detected, the SQLite MCP server appears with the detected file path.

**Steps:**
1. Implement MySQL detection heuristic:
   - Check `devenv.nix` for `services.mysql.enable = true` or `services.mariadb.enable = true`.
   - Check `docker-compose.yml` / `docker-compose.yaml` / `compose.yml` for `mysql` or `mariadb` service image.
   - Extract connection parameters from service configuration where possible (port, default database name).
2. Implement SQLite detection heuristic:
   - Glob for `*.sqlite`, `*.sqlite3`, `*.db` files in project root, `data/`, `db/`, `var/`.
   - Exclude files in `node_modules/`, `.devenv/`, `vendor/`, `.git/`.
   - If multiple SQLite files found, use the first alphabetically and note others in CLAUDE.md.
3. Register both servers in `McpServerRegistry` with `ConfigPolicy: AutoDetect`.
4. Generate .mcp.json entries using the concrete JSON blocks from the per-server specs above.
5. For MySQL: populate `env` block from detected service config. Default to `127.0.0.1:3306` / `root` / empty password for devenv services.
6. For SQLite: populate `--db-path` argument from detected file path.
7. Contribute SecretSpec declarations for MySQL credentials (parallel to existing PostgreSQL declarations).
8. Contribute CLAUDE.md sections documenting each database MCP's capabilities and read-only nature.
9. Ensure mutual tool budget awareness: if PostgreSQL + MySQL are both detected, warn if total database MCP tools exceed budget allocation.

**Acceptance Criteria:**
- [ ] MySQL MCP auto-configured when `services.mysql.enable = true` in devenv.nix
- [ ] MySQL MCP auto-configured when mysql/mariadb service in docker-compose
- [ ] SQLite MCP auto-configured when `.sqlite`/`.db` files detected
- [ ] Connection credentials populated from service configuration
- [ ] SecretSpec declarations generated for MySQL credentials
- [ ] CLAUDE.md sections document database MCP capabilities
- [ ] Tool budget checked — warning if 3 database MCPs would push past ceiling
- [ ] Detection excludes vendored/generated directories

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/api-db-mcp-research.md § 3.2 Database MCP Servers` — MySQL and SQLite assessment
- `research-spikes/gdev-ecosystem-expansion-assessment/docs/mcp-official-servers-github.md` — SQLite as archived reference server
- `research-spikes/gdev-ecosystem-expansion-assessment/docs/mcp-servers-claude-code-cursor-2026.md` — SQLite tool description
- `implementation-plans/gdev-secure-devenv-bootstrap/phases/04-claude-code-addon-core-generation.md § Unit 3.5` — existing PostgreSQL MCP pattern

**Status:** Not Started

---

### Unit 3.5.3: Detect-and-Offer MCP Servers (Terraform, Sentry)

**Description:** Implement detection-triggered wizard prompts for Terraform MCP and Sentry MCP servers. When project signals are detected, the gdev wizard offers these servers during setup; they are not auto-enabled.

**Context:** These are Medium security tier servers. Terraform MCP provides registry documentation and Terraform Cloud workspace access — valuable for IaC projects but involves cloud platform metadata. Sentry MCP provides error tracking data — closes the "alert-to-fix" loop but reads production error data. Both require explicit opt-in but should be proactively offered when signals are present, not buried in the optional catalog.

**Desired Outcome:** When `.tf` files are detected, the wizard includes a prompt: "Terraform files detected. Enable Terraform MCP server for registry docs and workspace access?" When Sentry SDK is detected, the wizard includes: "Sentry integration detected. Enable Sentry MCP server for error tracking?" Answering yes adds the server to .mcp.json with credential placeholders.

**Steps:**
1. Implement Terraform detection heuristic:
   - Glob for `*.tf` files in project root and immediate subdirectories.
   - Check for `.terraform.lock.hcl` (indicates initialized Terraform project).
   - Check for `terraform/` directory.
2. Implement Sentry detection heuristic:
   - Check `package.json` for `@sentry/node`, `@sentry/react`, `@sentry/nextjs`, `@sentry/browser`.
   - Check `requirements.txt` / `pyproject.toml` for `sentry-sdk`.
   - Check `Cargo.toml` for `sentry` crate.
   - Check `.env` / `.env.example` for `SENTRY_DSN`.
   - Grep source files for `Sentry.init(` or `sentry_sdk.init(` (limited to top-level source dirs).
3. Register both servers in `McpServerRegistry` with `ConfigPolicy: DetectAndOffer`.
4. Add wizard prompts to the detection-driven form group:
   - Terraform prompt: huh confirm field, default No, shown only when Terraform detected.
   - Sentry prompt: huh confirm field, default No, shown only when Sentry detected.
5. Generate .mcp.json entries when opt-in confirmed, using concrete JSON blocks from per-server specs.
6. Contribute SecretSpec declarations:
   - Terraform: `TFC_TOKEN` as optional secret (registry works without it).
   - Sentry: `SENTRY_AUTH_TOKEN` as required secret.
7. Contribute CLAUDE.md sections:
   - Terraform: document registry browsing vs workspace management capabilities, beta status caveat.
   - Sentry: document alert-to-fix workflow, available issue/event queries.
8. Both servers must check tool budget via `CanEnable()` before adding.

**Acceptance Criteria:**
- [ ] Terraform MCP offered when `.tf` files detected, not offered otherwise
- [ ] Sentry MCP offered when Sentry SDK/DSN detected, not offered otherwise
- [ ] Wizard prompts default to No (opt-in, not opt-out)
- [ ] Declining the prompt produces no .mcp.json entry
- [ ] Accepting produces correct .mcp.json entry with credential env vars
- [ ] SecretSpec declarations generated for required credentials
- [ ] CLAUDE.md sections document capabilities and security implications
- [ ] Tool budget checked before enabling
- [ ] `--yes` flag skips these prompts (does NOT auto-enable — detect-and-offer defaults to off in non-interactive mode)

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/api-db-mcp-research.md § 3.2 Cloud Provider MCP` — Terraform MCP assessment
- `research-spikes/gdev-ecosystem-expansion-assessment/api-db-mcp-research.md § 3.2 Observability MCP` — Sentry MCP assessment
- `research-spikes/gdev-ecosystem-expansion-assessment/docs/terraform-mcp-server-hashicorp.md` — Terraform MCP capabilities and limitations
- `research-spikes/gdev-ecosystem-expansion-assessment/docs/mcp-official-servers-github.md` — Sentry as archived reference server

**Status:** Not Started

---

### Unit 12.8.1: MCP Lifecycle Commands (`gdev mcp list`, `gdev enable/disable mcp-*`)

**Description:** Implement MCP-specific lifecycle commands that integrate with the Phase 12 tool lifecycle system (Unit 12.1), providing `gdev mcp list` for server status overview and wiring each MCP server as an individually toggleable tool via `gdev enable mcp-<name>` / `gdev disable mcp-<name>`.

**Context:** Unit 12.1 defines the generic tool lifecycle system (`gdev enable`/`gdev disable`). MCP servers need specialized handling because: (a) they share a single file (.mcp.json) rather than having dedicated config files, (b) tool budget enforcement requires MCP-specific logic, and (c) engineers need a focused view of MCP servers separate from the full tool list. The `gdev mcp list` command is the primary interface for understanding what's active and how much tool budget remains.

**Desired Outcome:** `gdev mcp list` shows all available MCP servers with enabled/disabled state, tool count, and remaining budget. `gdev enable mcp-terraform` adds the server to .mcp.json. `gdev disable mcp-terraform` removes it. The tool budget is always visible.

**Steps:**
1. Register each of the 9 new MCP servers as lifecycle-managed tools (per Unit 12.1 `Tool` struct):
   - Name: `mcp-mysql`, `mcp-sqlite`, `mcp-terraform`, `mcp-sentry`, `mcp-atlassian`, `mcp-linear`, `mcp-slack`, `mcp-datadog`, `mcp-grafana`
   - Category: `ai-agent`
   - OwnedFiles: shared ownership of `.mcp.json` (keyed by server name) + exclusive ownership of CLAUDE.md section
   - Default policy: matches ConfigPolicy (AutoDetect → `OnWhenDetected`, DetectAndOffer → `OnWhenDetected` with wizard gate, OptionalCatalog → `OptIn`)
2. Implement `gdev mcp list` command:
   ```
   MCP Servers                          Status      Tools  Security
   ─────────────────────────────────────────────────────────────────
   context7          Library docs        Enabled        5  Low
   github            Repo management     Enabled       12  Low
   socket-dev        Supply chain        Enabled        6  Low
   semble            Code search         Enabled        5  Low
   postgres          Database queries    Enabled        8  Low
   mysql             Database queries    Disabled       8  Low
   sqlite            Database queries    Disabled       6  Low
   terraform         IaC registry/TFC    Disabled      10  Medium
   sentry            Error tracking      Disabled       8  Medium
   atlassian         Jira/Confluence     Disabled      15  High
   linear            Issue tracking      Disabled      10  High
   slack             Team messaging      Disabled      12  High
   datadog           Observability       Disabled      12  High
   grafana           Dashboards/metrics  Disabled      10  High
   ─────────────────────────────────────────────────────────────────
   Active: 5 servers, ~36 tools (budget: 40)
   ```
3. Support `gdev mcp list --json` for machine-readable output.
4. Implement `.mcp.json` shared-file surgery in the lifecycle system:
   - Parse existing `.mcp.json`, add/remove server key, marshal back.
   - Preserve non-gdev entries (user-added servers not in the registry).
5. Wire `gdev enable mcp-<name>`:
   - Check `CanEnable()` — fail with budget warning if over ceiling.
   - For Medium/High security tiers: prompt for credentials (or accept via `--env KEY=VALUE` flags).
   - Add server entry to .mcp.json.
   - Add CLAUDE.md documentation section.
   - Add SecretSpec declarations if applicable.
   - Print summary: "Enabled Terraform MCP (10 tools). Active: 6 servers, ~46 tools. WARNING: exceeds 40-tool ceiling."
6. Wire `gdev disable mcp-<name>`:
   - Remove server entry from .mcp.json.
   - Remove CLAUDE.md documentation section.
   - Remove SecretSpec declarations.
   - Print summary: "Disabled Terraform MCP. Active: 5 servers, ~36 tools."
7. Implement `gdev mcp check`:
   - Validate all enabled servers have required credentials configured.
   - Report tool budget status.
   - Flag servers with missing credentials.

**Acceptance Criteria:**
- [ ] `gdev mcp list` shows all 14 servers with correct status, tool counts, and security tiers
- [ ] `gdev mcp list` shows total active tools and budget remaining
- [ ] `gdev mcp list --json` produces machine-readable output
- [ ] `gdev enable mcp-terraform` adds server to .mcp.json with correct config
- [ ] `gdev disable mcp-terraform` removes server from .mcp.json
- [ ] Enable fails with clear message when tool budget would be exceeded
- [ ] Enable prompts for credentials for Medium/High tier servers
- [ ] Non-gdev .mcp.json entries preserved during enable/disable
- [ ] CLAUDE.md sections added/removed with enable/disable
- [ ] `gdev mcp check` reports credential and budget status

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/api-db-mcp-research.md § 3.3-3.4` — expansion recommendations, tool ceiling
- `implementation-plans/gdev-secure-devenv-bootstrap/phases/12-extended-integrations-lifecycle.md § Unit 12.1` — tool lifecycle management system
- `implementation-plans/gdev-secure-devenv-bootstrap/phases/12-extended-integrations-lifecycle.md § Unit 12.8` — MCP server curation

**Status:** Not Started

---

### Unit 12.8.2: Optional Catalog — Ticketing MCP Servers (Atlassian, Linear)

**Description:** Implement Atlassian (Jira/Confluence) and Linear MCP servers as optional catalog entries, accessible only via explicit `gdev enable mcp-atlassian` or `gdev enable mcp-linear`. Include credential setup flow, security documentation, and CLAUDE.md integration.

**Context:** Atlassian MCP is the highest-value expansion candidate for a consulting organization. Engineers spend significant time context-switching between Jira and their IDE. Being able to query issues, read Confluence pages, and update ticket status from Claude Code eliminates this friction. Linear serves the same role for teams on Linear. Both are High security tier — they access project management data and can take actions (create/update issues). They must never be auto-enabled.

**Desired Outcome:** `gdev enable mcp-atlassian` walks the engineer through Atlassian site URL configuration, adds the server to .mcp.json, and documents available capabilities in CLAUDE.md. The same for Linear. Both are invisible during `gdev init` unless explicitly selected in the customize path.

**Steps:**
1. Register `mcp-atlassian` in MCP server registry:
   - ConfigPolicy: `OptionalCatalog`
   - SecurityTier: `High`
   - ToolCount: 15
   - RequiredSecrets: `ATLASSIAN_SITE_URL`
   - OAuth 2.1 handled by the server itself — gdev only needs the site URL.
2. Register `mcp-linear` in MCP server registry:
   - ConfigPolicy: `OptionalCatalog`
   - SecurityTier: `High`
   - ToolCount: 10
   - RequiredSecrets: `LINEAR_API_KEY` (optional — OAuth preferred)
   - Remote MCP transport (URL-based, not stdio).
3. Implement `gdev enable mcp-atlassian` flow:
   a. Prompt for Atlassian site URL (e.g., `https://mycompany.atlassian.net`).
   b. Validate URL format.
   c. Check tool budget via `CanEnable()`.
   d. Add .mcp.json entry with site URL in env.
   e. Add CLAUDE.md section: "Atlassian MCP provides access to Jira issues and Confluence pages. It respects your existing Jira permissions — you can only see what your Atlassian account can access. Available operations: search issues, read issue details, create issues, update status, read Confluence pages."
   f. Print security notice: "Atlassian MCP will have the same access as your Atlassian account. OAuth consent will be requested on first use."
4. Implement `gdev enable mcp-linear` flow:
   a. Check tool budget.
   b. Add .mcp.json entry with remote URL transport.
   c. Add CLAUDE.md section documenting Linear MCP capabilities.
   d. Print note: "Linear MCP uses OAuth 2.1. Consent will be requested on first use."
5. Add both to wizard customize path: "Project Management MCP Servers" section with multi-select for Atlassian, Linear.
6. Implement credential validation where possible (Atlassian URL format, Linear API key format).

**Acceptance Criteria:**
- [ ] `gdev enable mcp-atlassian` prompts for site URL and configures .mcp.json
- [ ] `gdev enable mcp-linear` configures remote MCP URL in .mcp.json
- [ ] Neither server appears during `gdev init` quick path
- [ ] Both available in wizard customize path under "Project Management" section
- [ ] CLAUDE.md sections clearly document what the AI agent can access
- [ ] Security notices printed during enable
- [ ] `gdev disable mcp-atlassian` cleanly removes all artifacts
- [ ] `gdev disable mcp-linear` cleanly removes all artifacts
- [ ] Tool budget enforced

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/api-db-mcp-research.md § 3.2 Ticketing & PM MCP` — Atlassian and Linear assessment
- `research-spikes/gdev-ecosystem-expansion-assessment/docs/linear-mcp-server-official.md` — Linear OAuth 2.1, remote transport
- `research-spikes/gdev-ecosystem-expansion-assessment/docs/mcp-bundles-best-servers-2026.md` — Linear maturity confirmation

**Status:** Not Started

---

### Unit 12.8.3: Optional Catalog — Communication MCP Server (Slack)

**Description:** Implement Slack MCP server as an optional catalog entry with explicit security documentation about communication exposure risks.

**Context:** Slack MCP enables searching messages, reading channel history, and posting messages. It is the most security-sensitive optional server because it provides AI agent access to team communications — conversations that may contain sensitive project information, client discussions, or personnel matters. The value proposition is clear (catching up on project context without manual Slack trawling), but the risk requires explicit acknowledgment.

**Desired Outcome:** `gdev enable mcp-slack` walks through setup with clear security warnings, configures the server, and documents what the AI agent can access. The security notice is unavoidable — not skippable with `--yes`.

**Steps:**
1. Register `mcp-slack` in MCP server registry:
   - ConfigPolicy: `OptionalCatalog`
   - SecurityTier: `High`
   - ToolCount: 12
   - RequiredSecrets: `SLACK_TEAM_ID`, `SLACK_BOT_TOKEN`
2. Implement `gdev enable mcp-slack` flow:
   a. Display security warning (not skippable):
      ```
      ⚠ Slack MCP gives Claude Code access to your team's Slack workspace.
      The AI agent will be able to:
        - Search and read message history in channels the bot can access
        - Read file attachments
        - Post messages on your behalf
        - Access member profiles

      This may expose sensitive team communications to the AI context window.
      Ensure your Slack app has minimal channel access scopes.
      ```
   b. Require explicit "I understand" confirmation.
   c. Prompt for `SLACK_TEAM_ID` and `SLACK_BOT_TOKEN`.
   d. Check tool budget.
   e. Add .mcp.json entry.
   f. Add CLAUDE.md section documenting Slack MCP scope and channel access.
3. Add SecretSpec declarations for Slack credentials.
4. Document Slack app creation process in CLAUDE.md section (link to Slack API docs, required scopes).

**Acceptance Criteria:**
- [ ] `gdev enable mcp-slack` shows unavoidable security warning
- [ ] Security warning cannot be skipped with `--yes` (requires explicit confirmation)
- [ ] Slack credentials stored via SecretSpec, never in .mcp.json plaintext
- [ ] CLAUDE.md section documents what the AI agent can access
- [ ] `gdev disable mcp-slack` cleanly removes all artifacts
- [ ] Tool budget enforced

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/api-db-mcp-research.md § 3.2 Communication MCP` — Slack assessment, security concerns
- `research-spikes/gdev-ecosystem-expansion-assessment/docs/slack-mcp-server-official.md` — Slack MCP capabilities, partner integrations

**Status:** Not Started

---

### Unit 12.8.4: Optional Catalog — Observability MCP Servers (Datadog, Grafana)

**Description:** Implement Datadog and Grafana MCP servers as optional catalog entries for teams using those observability platforms.

**Context:** Datadog and Grafana are the two dominant observability platforms in consulting engagements. Both went GA in early 2026 with mature MCP servers. They enable workflows like "find the error spike in Datadog, correlate with recent deployments, fix the code" without leaving Claude Code. Unlike Sentry (detect-and-offer), these are client infrastructure choices that gdev cannot detect — they require explicit enable.

**Desired Outcome:** `gdev enable mcp-datadog` and `gdev enable mcp-grafana` configure their respective servers with credential prompts. Both support the observability debugging workflow documented in CLAUDE.md.

**Steps:**
1. Register `mcp-datadog` in MCP server registry:
   - ConfigPolicy: `OptionalCatalog`
   - SecurityTier: `High`
   - ToolCount: 12
   - RequiredSecrets: `DD_API_KEY`, `DD_APP_KEY`, `DD_SITE`
2. Register `mcp-grafana` in MCP server registry:
   - ConfigPolicy: `OptionalCatalog`
   - SecurityTier: `High`
   - ToolCount: 10
   - RequiredSecrets: `GRAFANA_URL`, `GRAFANA_API_KEY`
3. Implement `gdev enable mcp-datadog` flow:
   a. Prompt for Datadog API key, app key, and site (default `datadoghq.com`).
   b. Check tool budget.
   c. Add .mcp.json entry.
   d. Add SecretSpec declarations.
   e. Add CLAUDE.md section documenting Datadog MCP use cases: onboarding, infrastructure optimization, incident root cause analysis.
4. Implement `gdev enable mcp-grafana` flow:
   a. Prompt for Grafana URL and API key (service account token with viewer role).
   b. Check tool budget.
   c. Add .mcp.json entry.
   d. Add SecretSpec declarations.
   e. Add CLAUDE.md section noting Grafana's token-optimized response design.
5. Add both to wizard customize path: "Observability MCP Servers" section.
6. Note: if Sentry (Unit 3.5.3) is also enabled, warn about observability tool budget — 3 observability servers would add ~30 tools.

**Acceptance Criteria:**
- [ ] `gdev enable mcp-datadog` prompts for credentials and configures .mcp.json
- [ ] `gdev enable mcp-grafana` prompts for credentials and configures .mcp.json
- [ ] SecretSpec declarations generated for all credentials
- [ ] CLAUDE.md sections document each server's use cases
- [ ] Warning when enabling multiple observability servers (Sentry + Datadog + Grafana)
- [ ] `gdev disable` cleanly removes all artifacts for each
- [ ] Tool budget enforced
- [ ] Both available in wizard customize path

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/api-db-mcp-research.md § 3.2 Observability MCP` — Datadog and Grafana assessment
- `research-spikes/gdev-ecosystem-expansion-assessment/docs/datadog-mcp-server-use-cases.md` — Datadog GA status, 4 primary use cases
- `research-spikes/gdev-ecosystem-expansion-assessment/docs/devops-mcp-servers-2026-medium.md` — Grafana token optimization design

**Status:** Not Started

---

### Unit 3.5.4: .mcp.json Generation with Registry-Driven Composition

**Description:** Rewrite Unit 3.5's .mcp.json generation to use the MCP server registry (Unit 3.5.1) instead of hardcoded server entries. The generator iterates over enabled servers in the registry, collects their `McpEntry()` outputs, and marshals the composed result.

**Context:** The original Unit 3.5 defined .mcp.json generation for a fixed set of 5 servers. With 14+ servers and three-tier auto-configuration, the generation logic must be registry-driven. This unit replaces the hardcoded approach with one that queries the registry for all servers whose policy and detection results indicate they should be enabled, then composes their entries into a single .mcp.json.

**Desired Outcome:** `.mcp.json` generation produces the correct server set based on: (1) always-on defaults, (2) auto-detected servers, (3) wizard-confirmed detect-and-offer servers, (4) explicitly enabled optional servers. The output is deterministic and respects tool budget.

**Steps:**
1. Refactor `McpJsonGenerator.Generate(answers)` to use `McpServerRegistry`:
   a. Collect always-on servers (Context7, GitHub).
   b. Collect auto-detected servers whose `DetectFunc` returns true (Socket.dev for JS/Python/Rust/Go, PostgreSQL/MySQL/SQLite for detected databases, semble for Python >=3.10).
   c. Collect detect-and-offer servers where wizard answer is "yes" (Terraform, Sentry).
   d. Collect optional catalog servers where `gdev enable` has been called (Atlassian, Linear, Slack, Datadog, Grafana).
   e. Validate total tool count — emit warning in generated CLAUDE.md if over soft ceiling.
2. For each enabled server, call `McpEntry(answers)` to get the typed entry, handling:
   - Servers with env vars: populate from wizard answers, SecretSpec, or leave placeholder.
   - Servers with remote URL transport (Linear): use `type: url` instead of `command`.
   - Servers with dynamic args (SQLite db path): populate from detection results.
3. Marshal composed result via `json.MarshalIndent()`.
4. Wrap in `GeneratedFile` with `ThreeWayMerge` strategy (preserves user-added servers).
5. Generate companion CLAUDE.md section listing active MCP servers with their purposes.
6. Write integration test: detect Go + PostgreSQL + Terraform project, confirm .mcp.json contains Context7 + GitHub + Socket.dev + PostgreSQL + Terraform (if wizard confirmed), and DOES NOT contain MySQL/SQLite/Sentry/Atlassian.

**Acceptance Criteria:**
- [ ] Generated .mcp.json contains exactly the servers that should be enabled per policy
- [ ] Always-on servers present regardless of project type
- [ ] Auto-detect servers present only when detection heuristic matches
- [ ] Detect-and-offer servers present only when wizard confirmed
- [ ] Optional catalog servers present only when explicitly enabled
- [ ] Remote URL transport (Linear) handled correctly
- [ ] Dynamic arguments (SQLite path) populated from detection
- [ ] ThreeWayMerge preserves user-added servers on re-generation
- [ ] Tool budget warning in CLAUDE.md if over soft ceiling
- [ ] Generated JSON passes `json.Unmarshal` validation

**Research Citations:**
- `implementation-plans/gdev-secure-devenv-bootstrap/phases/04-claude-code-addon-core-generation.md § Unit 3.5` — original .mcp.json generation design
- `research-spikes/gdev-ecosystem-expansion-assessment/api-db-mcp-research.md § 3.3` — expansion recommendations with tier assignments
- `research-spikes/gdev-ecosystem-expansion-assessment/api-db-mcp-research.md § 3.4` — per-project configuration, profile-based bundles

**Status:** Not Started

---

### Unit 12.8.5: MCP Security Documentation and Credential Hygiene

**Description:** Generate comprehensive security documentation for all MCP servers covering credential management, permission scopes, and risk acknowledgment. Ensure no credentials are written in plaintext to .mcp.json — all sensitive values route through SecretSpec or environment variables.

**Context:** MCP servers span from zero-auth (Context7, SQLite) to OAuth-protected (Atlassian, Linear) to API-key-authenticated (Datadog, Grafana, Sentry). The security model must be transparent — engineers should understand exactly what data each server exposes to the AI agent. Credential hygiene is critical: API keys in .mcp.json would be committed to git. SecretSpec integration ensures credentials are resolved at runtime from secure providers (keyring, env vars, 1Password).

**Desired Outcome:** Every MCP server's CLAUDE.md section includes a security notice. All credentials flow through SecretSpec or environment variable references — never plaintext in committed files. `gdev mcp check` validates credential hygiene.

**Steps:**
1. Define per-server security documentation templates:
   - Low tier: "This server has read-only access to [resource]. No authentication required / credentials auto-configured from devenv services."
   - Medium tier: "This server can read [resource types]. Requires [credential]. Your [API key / token] determines access scope."
   - High tier: "This server can read and modify [resource types]. It has the same permissions as your [platform] account. Review the access scope before enabling."
2. Generate CLAUDE.md `<!-- gdev:mcp-security -->` section aggregating all enabled servers' security notices.
3. Implement credential hygiene for .mcp.json:
   - Low tier (MySQL): env vars reference SecretSpec-managed values or devenv service defaults.
   - Medium tier (Terraform, Sentry): env vars reference SecretSpec entries; empty string placeholder in .mcp.json.
   - High tier: env vars reference SecretSpec entries; `gdev mcp check` warns if values are empty.
4. Add `.mcp.json` to `.gitignore` recommendation when High-tier servers are enabled (or use `.mcp.json.local` pattern).
5. Implement `gdev mcp check` credential validation:
   - For each enabled server with required credentials, check SecretSpec resolution.
   - Report: "mcp-datadog: DD_API_KEY ✓, DD_APP_KEY ✗ (not configured)"
   - Exit code 1 if any required credentials are missing.
6. Add pre-commit hook check: warn if .mcp.json contains non-empty credential values (potential plaintext leak).

**Acceptance Criteria:**
- [ ] Every MCP server has a security notice in CLAUDE.md
- [ ] Security notices are tier-appropriate (low/medium/high language)
- [ ] No plaintext credentials in .mcp.json committed to git
- [ ] SecretSpec declarations generated for all credential-requiring servers
- [ ] `gdev mcp check` validates credential configuration
- [ ] Pre-commit hook warns on plaintext credentials in .mcp.json
- [ ] `.gitignore` recommendation when High-tier servers enabled
- [ ] Credential validation supports `--ci` mode (strict, non-interactive)

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/api-db-mcp-research.md § 4.3` — security gradient model
- `implementation-plans/gdev-secure-devenv-bootstrap/phases/12-extended-integrations-lifecycle.md § Unit 12.6` — SecretSpec integration design
- `research-spikes/gdev-ecosystem-expansion-assessment/docs/slack-mcp-server-official.md` — partner OAuth security model
- `research-spikes/gdev-ecosystem-expansion-assessment/docs/linear-mcp-server-official.md` — OAuth 2.1 + API key auth options

**Status:** Not Started

---

## Dependency Graph

```
Unit 3.5.1 (MCP Server Registry)
  ├── Unit 3.5.2 (Auto-Detect: MySQL, SQLite) — depends on registry
  ├── Unit 3.5.3 (Detect-and-Offer: Terraform, Sentry) — depends on registry
  └── Unit 3.5.4 (.mcp.json Registry-Driven Composition) — depends on registry + 3.5.2 + 3.5.3
      └── Unit 12.8.5 (Security Docs & Credential Hygiene) — depends on all servers registered

Unit 12.1 (Tool Lifecycle System, existing) — prerequisite for all 12.8.X units
  ├── Unit 12.8.1 (MCP Lifecycle Commands) — depends on 3.5.1 + 12.1
  ├── Unit 12.8.2 (Ticketing: Atlassian, Linear) — depends on 12.8.1
  ├── Unit 12.8.3 (Communication: Slack) — depends on 12.8.1
  └── Unit 12.8.4 (Observability: Datadog, Grafana) — depends on 12.8.1
```

## Tool Budget Reference

| Server | Tools (approx) | Tier | Default |
|--------|---------------|------|---------|
| Context7 | 5 | Always-on | Enabled |
| GitHub | 12 | Always-on | Enabled |
| Socket.dev | 6 | Auto-detect | Enabled (JS/Python/Rust/Go) |
| semble | 5 | Auto-detect | Enabled (Python >=3.10) |
| PostgreSQL | 8 | Auto-detect | Enabled (PG service) |
| **MySQL** | **8** | **Auto-detect** | **Enabled (MySQL service)** |
| **SQLite** | **6** | **Auto-detect** | **Enabled (*.sqlite files)** |
| **Terraform** | **10** | **Detect-and-offer** | **Off (wizard prompt)** |
| **Sentry** | **8** | **Detect-and-offer** | **Off (wizard prompt)** |
| **Atlassian** | **15** | **Optional catalog** | **Off** |
| **Linear** | **10** | **Optional catalog** | **Off** |
| **Slack** | **12** | **Optional catalog** | **Off** |
| **Datadog** | **12** | **Optional catalog** | **Off** |
| **Grafana** | **10** | **Optional catalog** | **Off** |

**Typical budget scenarios:**
- Minimal (3 servers): Context7 + GitHub + Socket.dev = ~23 tools
- Standard (5 servers): + PostgreSQL + semble = ~36 tools
- Standard + 1 DB swap (5 servers): Context7 + GitHub + Socket.dev + MySQL + semble = ~36 tools
- Full auto (6 servers): Standard + Terraform = ~46 tools (OVER ceiling — wizard must warn)
- Consulting heavy (6 servers): Minimal + Atlassian + Sentry + Datadog = ~58 tools (OVER ceiling — requires disabling others)

The design ensures that the default auto-detect set (5 servers, ~36 tools) stays under ceiling, and any additions trigger budget awareness.
