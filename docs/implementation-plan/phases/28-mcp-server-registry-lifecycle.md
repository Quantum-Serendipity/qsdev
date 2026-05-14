# Phase 28: MCP Server Registry & Lifecycle Management

## Goal

Replace the MVP's hardcoded 5-server `.mcp.json` generation with a registry-driven system. Each MCP server has structured metadata: tool count, security tier, config policy, detection function, and credential requirements. The registry enables `gdev mcp list/enable/disable` commands, enforces a hard 40-tool ceiling, and generates per-tier security notices in CLAUDE.md. Detection-based auto-enable, detect-and-offer confirmation, and an explicit optional catalog provide three graduated paths for adding servers.

## Dependencies

Phase 4 complete (Claude Code addon — `.mcp.json` generation, section markers, settings.json deny rules). Phase 12 complete (tool lifecycle management — shared-file surgery, file ownership registry, section markers for surgical add/remove).

## Phase Outputs

- `McpServerRegistry` Go struct with per-server metadata (tool count, security tier, config policy, detection function, credential requirements)
- All MVP Phase 4 servers migrated to registry format
- MySQL and SQLite MCP servers as new auto-detection entries
- Terraform and Sentry MCP servers as detect-and-offer entries
- 5 optional catalog servers (Atlassian, Linear, Slack, Datadog, Grafana)
- `gdev mcp list/enable/disable/check` commands
- MCP security documentation generator (per-tier notices in CLAUDE.md)
- Registry-driven `.mcp.json` generation with devenv 2.0 native module support

---

### Unit 28.1: McpServerRegistry Go Struct

**Description:** Define the `McpServerRegistry` struct and per-server `McpServerEntry` metadata type that drives all MCP operations. Migrate all MVP Phase 4 servers into registry format. Implement the 40-tool ceiling check.

**Context:** The MVP's Phase 4 generates `.mcp.json` by iterating a hardcoded list of servers with minimal metadata. The registry replaces this with a first-class data structure that encodes everything gdev needs to reason about MCP servers: how many tools each server exposes (for ceiling enforcement), how dangerous each server is (security tier), when to auto-enable vs ask vs require explicit opt-in (config policy), what env vars are needed (credential requirements), and how to detect relevance (detection function).

The 40-tool ceiling is a Claude Code context budget constraint. Each MCP tool consumes context window space in every agent turn. Exceeding the ceiling degrades performance noticeably. The ceiling is a hard constraint enforced at enable time, not a soft warning.

**Code-Grounded Note:** The Phase 4 `.mcp.json` generation currently lives in `addons/claudecode/generate_mcp.go` (or equivalent). This unit defines the registry type in `internal/mcp/registry.go` and a `MigrateHardcodedServers()` function that constructs the registry from the existing hardcoded server list. The existing code's server list is the canonical reference for the MVP servers; do not invent server names or configurations that differ from what Phase 4 generates.

**Desired Outcome:** A complete `McpServerRegistry` type with all MVP servers migrated in. The registry is the single source of truth for every MCP operation: generation, listing, enabling, disabling, documentation, and compliance checking. The 40-tool ceiling is enforced at enable time with a clear error.

**Steps:**
1. Define the core registry types in `internal/mcp/registry.go`:
   ```go
   // SecurityTier classifies the risk level of an MCP server.
   type SecurityTier string
   const (
       // TierLow: read-only, local data sources, no network calls, no credentials.
       TierLow SecurityTier = "low"
       // TierMedium: read-only, makes network calls to remote services, may hold API keys.
       TierMedium SecurityTier = "medium"
       // TierHigh: has write access to external services, or is credential-holding with broad scope.
       TierHigh SecurityTier = "high"
   )

   // ConfigPolicy controls how and when a server is enabled.
   type ConfigPolicy string
   const (
       // AutoDetect: enabled automatically when detection function returns true. No wizard question.
       AutoDetect ConfigPolicy = "auto_detect"
       // DetectAndOffer: detected but wizard confirms before enabling. Credentials shown upfront.
       DetectAndOffer ConfigPolicy = "detect_and_offer"
       // OptionalCatalog: only via explicit 'gdev enable mcp-<name>' or wizard customize path.
       OptionalCatalog ConfigPolicy = "optional_catalog"
   )

   // McpServerEntry describes a single MCP server in the registry.
   type McpServerEntry struct {
       // Unique slug: used in 'gdev mcp enable <name>' and '.mcp.json' keys.
       Name string

       // Human-readable description shown in 'gdev mcp list'.
       Description string

       // Approximate number of tools this server exposes to the Claude context window.
       ToolCount int

       // Security classification for documentation and compliance enforcement.
       SecurityTier SecurityTier

       // Config policy: how this server gets enabled.
       ConfigPolicy ConfigPolicy

       // DetectFunc returns true if this server is relevant to the current project.
       // Called during wizard and 'gdev detect' to pre-populate wizard choices.
       // nil means "never auto-detect" (OptionalCatalog servers may omit this).
       DetectFunc func(state ProjectState) bool

       // CredentialsNeeded lists the environment variable names this server requires.
       // Empty means no credentials needed.
       CredentialsNeeded []string

       // ServerConfig produces the JSON object for this server's entry in .mcp.json.
       // Receives the project state and resolved env var values.
       ServerConfig func(state ProjectState) McpServerConfig
   }

   // McpServerConfig is the JSON-serializable config for a single server in .mcp.json.
   type McpServerConfig struct {
       // For stdio-transport servers.
       Command string   `json:"command,omitempty"`
       Args    []string `json:"args,omitempty"`
       Env     map[string]string `json:"env,omitempty"`

       // For HTTP-transport servers.
       URL     string `json:"url,omitempty"`
       Headers map[string]string `json:"headers,omitempty"`
   }

   // McpServerRegistry holds the full set of known MCP servers.
   type McpServerRegistry struct {
       Servers []*McpServerEntry
   }

   // NewRegistry returns the default global registry with all built-in servers.
   func NewRegistry() *McpServerRegistry

   // Get returns the entry for the given name, or nil if not found.
   func (r *McpServerRegistry) Get(name string) *McpServerEntry

   // Enabled returns all servers currently enabled for the project.
   func (r *McpServerRegistry) Enabled(projectRoot string) ([]*McpServerEntry, error)

   // TotalToolCount sums ToolCount across all enabled servers.
   func (r *McpServerRegistry) TotalToolCount(enabled []*McpServerEntry) int

   // CanEnable returns an error if enabling the given server would exceed the 40-tool ceiling.
   func (r *McpServerRegistry) CanEnable(name string, currentEnabled []*McpServerEntry) error
   ```
2. Implement the 40-tool ceiling check:
   ```go
   const MaxMcpToolBudget = 40

   func (r *McpServerRegistry) CanEnable(name string, currentEnabled []*McpServerEntry) error {
       entry := r.Get(name)
       if entry == nil {
           return fmt.Errorf("unknown MCP server: %s", name)
       }
       if entry.ToolCount <= 0 {
           // Unknown tool count: treat as 1 for ceiling purposes, do not block.
           return nil
       }
       currentTotal := r.TotalToolCount(currentEnabled)
       projected := currentTotal + entry.ToolCount
       if projected > MaxMcpToolBudget {
           return &ToolBudgetExceededError{
               Server:          name,
               ServerToolCount: entry.ToolCount,
               CurrentTotal:    currentTotal,
               WouldBe:         projected,
               Budget:          MaxMcpToolBudget,
           }
       }
       return nil
   }

   type ToolBudgetExceededError struct {
       Server          string
       ServerToolCount int
       CurrentTotal    int
       WouldBe         int
       Budget          int
   }

   func (e *ToolBudgetExceededError) Error() string {
       return fmt.Sprintf(
           "cannot enable %s: would bring total MCP tools to %d (budget: %d)\n"+
           "  Current: %d tools across enabled servers\n"+
           "  %s adds: ~%d tools\n\n"+
           "  Disable another server first: gdev mcp list",
           e.Server, e.WouldBe, e.Budget,
           e.CurrentTotal, e.Server, e.ServerToolCount,
       )
   }
   ```
3. Migrate MVP Phase 4 servers into registry entries. The MVP Phase 4 `.mcp.json` generates these servers (read from existing `generate_mcp.go` to get exact names and configurations):
   - `context7` — documentation lookup, TierLow, AutoDetect (always), ~10 tools
   - `github` — GitHub API integration, TierMedium, DetectAndOffer (detected by `.github/` or GitHub remote), ~15 tools, `GITHUB_TOKEN` optional
   - Ecosystem-specific servers added by Phase 4's ecosystem addon (e.g., database servers, language-specific servers) — enumerate from existing code
4. Implement `LoadEnabledServers(projectRoot string) ([]*McpServerEntry, error)`:
   - Read `.mcp.json` from `projectRoot`.
   - Parse the `mcpServers` map keys.
   - Look up each key in the registry.
   - Return the matching entries (skip unrecognized keys with a debug log).
5. Implement `SaveEnabledServers(projectRoot string, enabled []*McpServerEntry) error`:
   - Generate `.mcp.json` content from the enabled entries.
   - Write using Phase 12 shared-file surgery to preserve any manually-added servers outside gdev's section markers.
6. Write unit tests:
   - Registry `Get` returns correct entry by name.
   - `CanEnable` blocks when adding a server would exceed 40.
   - `CanEnable` allows when total stays at or below 40.
   - `TotalToolCount` sums correctly across multiple entries.
   - `ToolBudgetExceededError.Error()` produces actionable message.
   - MVP servers all present in `NewRegistry()`.

**Acceptance Criteria:**
- [ ] `McpServerEntry` struct has all required fields: Name, Description, ToolCount, SecurityTier, ConfigPolicy, DetectFunc, CredentialsNeeded, ServerConfig
- [ ] `SecurityTier` enum: Low, Medium, High
- [ ] `ConfigPolicy` enum: AutoDetect, DetectAndOffer, OptionalCatalog
- [ ] `CanEnable` enforces hard 40-tool ceiling
- [ ] `ToolBudgetExceededError` shows current total, server tool count, and budget remaining
- [ ] All MVP Phase 4 servers migrated to registry entries with correct metadata
- [ ] `NewRegistry()` returns all built-in servers
- [ ] `LoadEnabledServers` reads enabled servers from existing `.mcp.json`
- [ ] Registry entries for MVP servers have correct `ToolCount`, `SecurityTier`, and `ConfigPolicy`

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/mcp-registry-research.md` — McpServerEntry struct design, 40-tool ceiling rationale, security tier classification
- `phases/04-claude-code-addon.md` — MVP server list, existing `.mcp.json` generation code reference

**Status:** Not Started

---

### Unit 28.2: MCP Auto-Detection Servers

**Description:** Add MySQL and SQLite MCP servers to the registry as `AutoDetect` entries that activate when relevant services or files are detected in the project.

**Context:** Auto-detection is the lowest-friction path: the server is enabled without the developer taking any action. This is appropriate for servers where the project's presence alone is sufficient signal of relevance (a project with SQLite files almost certainly wants an SQLite MCP server), and where the security tier is Low or Medium with no mandatory credentials. The detection functions must be specific enough to avoid false positives that waste the tool budget.

MySQL detection is coupled to devenv service detection (not just MySQL code imports) because the devenv MySQL service is the signal that the developer is actively working against a local MySQL database — the most common use case. SQLite detection is broader because SQLite is embedded and commonly used without a service.

**Code-Grounded Note:** The `ProjectState` type passed to `DetectFunc` should include the `devenv.nix` parsed service list (from the Phase 1 detection engine). MySQL detection checks both the devenv service AND a MySQL import/connection string in code, requiring both to reduce false positives. SQLite detection checks for `.sqlite`/`.db` files OR SQLite import patterns in the primary language's code.

**Desired Outcome:** MySQL and SQLite MCP servers are auto-enabled when the project shows clear signals of using those databases, without any wizard question or developer action.

**Steps:**
1. Implement MySQL MCP detection function:
   ```go
   func detectMySQL(state ProjectState) bool {
       // Require BOTH: devenv MySQL service AND MySQL usage evidence in code
       hasDevenvMySQL := slices.Contains(state.DevenvServices, "mysql")
       hasCodeUsage := detectMySQLCodeUsage(state.ProjectRoot, state.Ecosystems)
       return hasDevenvMySQL && hasCodeUsage
   }

   func detectMySQLCodeUsage(root string, ecosystems []string) bool {
       // Go: look for "github.com/go-sql-driver/mysql" or "gorm.io/driver/mysql" imports
       // Python: look for "import mysql" or "PyMySQL" or "mysqlclient" in requirements
       // JavaScript/TypeScript: look for "mysql2" or "mysql" in package.json dependencies
       // Generic: look for connection strings like "mysql://" in .env files
       // Returns true if any of these patterns match.
   }
   ```
2. Implement SQLite MCP detection function:
   ```go
   func detectSQLite(state ProjectState) bool {
       // Signal 1: .sqlite or .db files in project root or data/ subdirectory
       hasSQLiteFiles := globMatch(state.ProjectRoot, "*.{sqlite,sqlite3,db}")

       // Signal 2: SQLite imports in code
       hasCodeUsage := detectSQLiteCodeUsage(state.ProjectRoot, state.Ecosystems)

       return hasSQLiteFiles || hasCodeUsage
   }
   ```
3. Register MySQL MCP server entry:
   ```go
   &McpServerEntry{
       Name:         "mysql",
       Description:  "MySQL database operations — query, schema inspection, data exploration",
       ToolCount:    8,
       SecurityTier: TierMedium, // reads/writes database; credentials via env
       ConfigPolicy: AutoDetect,
       DetectFunc:   detectMySQL,
       CredentialsNeeded: []string{}, // reads from devenv-provided env vars
       ServerConfig: func(state ProjectState) McpServerConfig {
           return McpServerConfig{
               Command: "npx",
               Args:    []string{"-y", "@benborla29/mcp-server-mysql"},
               Env: map[string]string{
                   "MYSQL_HOST":     "${MYSQL_HOST:-127.0.0.1}",
                   "MYSQL_PORT":     "${MYSQL_PORT:-3306}",
                   "MYSQL_USER":     "${MYSQL_USER:-root}",
                   "MYSQL_PASSWORD": "${MYSQL_PASSWORD}",
                   "MYSQL_DATABASE": "${MYSQL_DATABASE}",
               },
           }
       },
   }
   ```
4. Register SQLite MCP server entry:
   ```go
   &McpServerEntry{
       Name:         "sqlite",
       Description:  "SQLite database operations — query, schema inspection, data exploration",
       ToolCount:    6,
       SecurityTier: TierLow, // local file access only, no network
       ConfigPolicy: AutoDetect,
       DetectFunc:   detectSQLite,
       CredentialsNeeded: []string{},
       ServerConfig: func(state ProjectState) McpServerConfig {
           return McpServerConfig{
               Command: "npx",
               Args:    []string{"-y", "@modelcontextprotocol/server-sqlite"},
           }
       },
   }
   ```
5. Wire auto-detection into the `gdev init` wizard:
   - After ecosystem detection, run `DetectFunc` for all `AutoDetect` servers.
   - Servers whose `DetectFunc` returns `true` are added to the enabled list without prompting.
   - Log at `--verbose` level: "Auto-enabled MCP server: mysql (MySQL service detected in devenv + mysql2 in package.json)".
6. Wire auto-detection into `gdev init --update`:
   - Re-run detection for all `AutoDetect` servers.
   - If a new server is now detected (e.g., developer added SQLite files): auto-enable and notify.
   - If a previously auto-enabled server is no longer detected: do NOT auto-disable (developer may have intentionally kept it).
7. Write unit tests:
   - MySQL: both devenv service AND code usage required (neither alone triggers).
   - SQLite: `.sqlite` file alone triggers; code usage alone triggers.
   - Disabled project state: `detectMySQL` returns false when no MySQL service.
   - Auto-enable in wizard: enabled list contains mysql when both signals present.
   - Update: new SQLite file detected, server auto-enabled on re-run.

**Acceptance Criteria:**
- [ ] MySQL MCP (`@benborla29/mcp-server-mysql`, ~8 tools) registered as `AutoDetect`
- [ ] MySQL detection requires BOTH devenv MySQL service AND MySQL code usage evidence
- [ ] SQLite MCP (`@modelcontextprotocol/server-sqlite`, ~6 tools) registered as `AutoDetect`
- [ ] SQLite detection triggers on `.sqlite`/`.db` files OR SQLite imports in code
- [ ] MySQL `SecurityTier` is `Medium` (network-accessible database, credentials via env)
- [ ] SQLite `SecurityTier` is `Low` (local files only)
- [ ] Auto-detect servers enabled silently during `gdev init` (no wizard question)
- [ ] `--verbose` logs which detection signal triggered each auto-enable
- [ ] `gdev init --update` detects newly added SQLite files and auto-enables the server
- [ ] Auto-disabled servers are not removed on re-detection loss (preserved if manually kept)

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/mcp-registry-research.md` — auto-detection server list, MySQL and SQLite detection signal design, security tier assignment

**Status:** Not Started

---

### Unit 28.3: Detect-and-Offer MCP Servers

**Description:** Add Terraform and Sentry MCP servers as `DetectAndOffer` entries that are detected automatically but require explicit wizard confirmation before enabling, with credential requirements shown before the confirmation.

**Context:** `DetectAndOffer` sits between auto-detection and manual opt-in. The server is relevant (detected by project signals) but requires either credentials (Sentry needs `SENTRY_AUTH_TOKEN`) or represents a potentially high-impact integration (Terraform has write access to infrastructure state). Showing credential requirements before asking for confirmation gives developers the information they need to make an informed decision.

The `--yes` flag does NOT auto-enable `DetectAndOffer` servers. This is intentional: these servers may require credentials that `--yes` mode cannot supply, and blindly enabling them in an automated context could cause unexpected network calls or permission issues.

**Code-Grounded Note:** The wizard in Phase 6 uses `huh` forms. `DetectAndOffer` servers are presented as a `huh.Confirm` form per server, shown after auto-detect servers are silently enabled. Each confirmation includes the server's `CredentialsNeeded` list so the developer knows what env vars to set before enabling.

**Desired Outcome:** Terraform and Sentry MCP servers appear as confirmation prompts in the wizard when detected, with credential requirements clearly shown. `--yes` mode skips them rather than auto-enabling.

**Steps:**
1. Implement Terraform MCP detection:
   ```go
   func detectTerraform(state ProjectState) bool {
       // Presence of *.tf files anywhere in the project
       return globMatch(state.ProjectRoot, "**/*.tf")
   }
   ```
2. Register Terraform MCP server:
   ```go
   &McpServerEntry{
       Name:         "terraform",
       Description:  "Terraform infrastructure operations — plan, apply, state inspection, module documentation",
       ToolCount:    10,
       SecurityTier: TierHigh, // can apply infrastructure changes
       ConfigPolicy: DetectAndOffer,
       DetectFunc:   detectTerraform,
       CredentialsNeeded: []string{"TFC_TOKEN"}, // optional: Terraform Cloud token
       ServerConfig: func(state ProjectState) McpServerConfig {
           cfg := McpServerConfig{
               Command: "npx",
               Args:    []string{"-y", "@hashicorp/terraform-mcp-server"},
           }
           if os.Getenv("TFC_TOKEN") != "" {
               cfg.Env = map[string]string{"TFC_TOKEN": "${TFC_TOKEN}"}
           }
           return cfg
       },
   }
   ```
3. Implement Sentry MCP detection:
   ```go
   func detectSentry(state ProjectState) bool {
       // Sentry SDK imports in primary language
       // Python: import sentry_sdk
       // JavaScript/TypeScript: @sentry/node, @sentry/browser, @sentry/react, etc. in package.json
       // Go: github.com/getsentry/sentry-go in go.mod
       return detectSentryImports(state.ProjectRoot, state.Ecosystems)
   }
   ```
4. Register Sentry MCP server:
   ```go
   &McpServerEntry{
       Name:         "sentry",
       Description:  "Sentry error monitoring — query issues, traces, and release health",
       ToolCount:    8,
       SecurityTier: TierMedium, // read-only API calls to Sentry
       ConfigPolicy: DetectAndOffer,
       DetectFunc:   detectSentry,
       CredentialsNeeded: []string{"SENTRY_AUTH_TOKEN"},
       ServerConfig: func(state ProjectState) McpServerConfig {
           return McpServerConfig{
               Command: "npx",
               Args:    []string{"-y", "@sentry/mcp-server"},
               Env: map[string]string{
                   "SENTRY_AUTH_TOKEN": "${SENTRY_AUTH_TOKEN}",
               },
           }
       },
   }
   ```
5. Implement `DetectAndOffer` wizard prompts in `internal/wizard/mcp.go`:
   ```go
   func offerDetectedServers(detected []*McpServerEntry, enabled []*McpServerEntry) ([]*McpServerEntry, error) {
       for _, server := range detected {
           if server.ConfigPolicy != DetectAndOffer {
               continue
           }

           credInfo := ""
           if len(server.CredentialsNeeded) > 0 {
               credInfo = fmt.Sprintf("\n  Requires: %s", strings.Join(server.CredentialsNeeded, ", "))
           }

           var confirm bool
           huh.NewConfirm().
               Title(fmt.Sprintf("Enable %s MCP server?", server.Name)).
               Description(fmt.Sprintf("%s%s", server.Description, credInfo)).
               Value(&confirm).Run()

           if confirm {
               if err := registry.CanEnable(server.Name, enabled); err != nil {
                   fmt.Printf("Cannot enable %s: %s\n", server.Name, err)
                   continue
               }
               enabled = append(enabled, server)
           }
       }
       return enabled, nil
   }
   ```
6. Enforce `--yes` skip behavior:
   ```go
   // In non-interactive mode, skip all DetectAndOffer servers.
   if nonInteractive {
       for _, server := range detected {
           if server.ConfigPolicy == DetectAndOffer {
               fmt.Printf("Skipping %s (detect-and-offer requires confirmation; use 'gdev mcp enable %s')\n",
                   server.Name, server.Name)
           }
       }
   }
   ```
7. Write unit tests:
   - Terraform detection: `*.tf` file present triggers; no `.tf` files does not.
   - Sentry detection: `@sentry/node` in package.json triggers; bare project does not.
   - Wizard prompt shows credential requirements in description.
   - `--yes` mode: `DetectAndOffer` servers skipped with informational message.
   - `DetectAndOffer` server enabled after confirmation.
   - Budget ceiling checked before enabling confirmed server.

**Acceptance Criteria:**
- [ ] Terraform MCP (`@hashicorp/terraform-mcp-server`, ~10 tools) registered as `DetectAndOffer`, `TierHigh`
- [ ] Terraform detected by presence of `*.tf` files in project
- [ ] Sentry MCP (`@sentry/mcp-server`, ~8 tools) registered as `DetectAndOffer`, `TierMedium`
- [ ] Sentry detected by Sentry SDK imports in Python, JavaScript/TypeScript, or Go
- [ ] Wizard shows credential requirements (`TFC_TOKEN`, `SENTRY_AUTH_TOKEN`) before confirmation
- [ ] `--yes` flag does NOT auto-enable `DetectAndOffer` servers
- [ ] `--yes` mode prints informational skip message for detected `DetectAndOffer` servers
- [ ] 40-tool ceiling checked before enabling a confirmed server
- [ ] `TFC_TOKEN` included in Terraform server config only when env var is set

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/mcp-registry-research.md` — detect-and-offer server list, credential requirement display design, --yes skip rationale
- `phases/06-wizard-orchestration.md` — huh form library, non-interactive mode handling

**Status:** Not Started

---

### Unit 28.4: Optional Catalog MCP Servers

**Description:** Register 5 optional catalog MCP servers (Atlassian, Linear, Slack, Datadog, Grafana) that require explicit enablement via `gdev enable mcp-<name>` or the wizard's customize path. Slack requires an unavoidable security acknowledgment that cannot be bypassed with `--yes`.

**Context:** Optional catalog servers are for integrations that are clearly team-specific, require OAuth or API credentials that gdev cannot detect, or carry significant security implications (Slack with write access to team communications). No project signal can reliably detect these — a project may use Jira without any `.jira` file or import. The catalog path requires the developer to make an intentional choice: `gdev enable mcp-atlassian`.

Atlassian (Jira + Confluence) represents the highest consulting value in this tier: project management and documentation access are the two most common requests from consulting developers. Linear and Datadog/Grafana are narrow-audience but high-value for teams that use them.

The Slack MCP's security warning cannot be bypassed: Slack has write access to team communications, which is uniquely sensitive. Even `--yes` must pause and require explicit text acknowledgment. This is the one place in gdev where automation is intentionally interrupted.

**Code-Grounded Note:** OAuth 2.1 servers (Atlassian, Linear) require a different `ServerConfig` shape than API-key servers: they use the `url` transport field (HTTP transport) rather than `command`/`args` (stdio transport). The `.mcp.json` format supports both; the registry's `McpServerConfig` struct already has both `Command`/`Args` (stdio) and `URL`/`Headers` (HTTP) fields from Unit 28.1.

**Desired Outcome:** Developers can enable any catalog server with `gdev enable mcp-<name>`. The Slack server requires a security acknowledgment that cannot be skipped. OAuth servers use HTTP transport in `.mcp.json`. `gdev mcp list` shows all catalog servers as `disabled`.

**Steps:**
1. Register Atlassian MCP server:
   ```go
   &McpServerEntry{
       Name:         "atlassian",
       Description:  "Jira and Confluence integration — issues, projects, pages, spaces (OAuth 2.1)",
       ToolCount:    15,
       SecurityTier: TierMedium, // read+write Jira/Confluence, no infrastructure access
       ConfigPolicy: OptionalCatalog,
       DetectFunc:   nil,
       CredentialsNeeded: []string{"ATLASSIAN_SITE_URL"},
       ServerConfig: func(state ProjectState) McpServerConfig {
           return McpServerConfig{
               // OAuth 2.1: uses HTTP transport with MCP remote server
               URL: fmt.Sprintf("https://%s/rest/mcp/v1/mcp",
                   os.Getenv("ATLASSIAN_SITE_URL")),
           }
       },
   }
   ```
2. Register Linear MCP server:
   ```go
   &McpServerEntry{
       Name:         "linear",
       Description:  "Linear issue tracking — issues, projects, cycles, roadmap (OAuth 2.1)",
       ToolCount:    12,
       SecurityTier: TierMedium,
       ConfigPolicy: OptionalCatalog,
       DetectFunc:   nil,
       CredentialsNeeded: []string{}, // OAuth: no static token needed
       ServerConfig: func(state ProjectState) McpServerConfig {
           return McpServerConfig{
               URL: "https://mcp.linear.app/mcp",
           }
       },
   }
   ```
3. Register Slack MCP server with unavoidable security warning:
   ```go
   &McpServerEntry{
       Name:         "slack",
       Description:  "Slack workspace integration — messages, channels, users (write access)",
       ToolCount:    10,
       SecurityTier: TierHigh, // write access to team communications
       ConfigPolicy: OptionalCatalog,
       DetectFunc:   nil,
       CredentialsNeeded: []string{"SLACK_BOT_TOKEN", "SLACK_TEAM_ID"},
       // EnableHook is called before enabling; must return nil to proceed.
       // This is where the mandatory security acknowledgment is enforced.
       EnableHook: func() error {
           return requireSlackAcknowledgment()
       },
       ServerConfig: func(state ProjectState) McpServerConfig {
           return McpServerConfig{
               Command: "npx",
               Args:    []string{"-y", "@modelcontextprotocol/server-slack"},
               Env: map[string]string{
                   "SLACK_BOT_TOKEN": "${SLACK_BOT_TOKEN}",
                   "SLACK_TEAM_ID":   "${SLACK_TEAM_ID}",
               },
           }
       },
   }
   ```
4. Implement `requireSlackAcknowledgment()`:
   ```go
   func requireSlackAcknowledgment() error {
       fmt.Println(`
   ┌─────────────────────────────────────────────────────────────┐
   │  SECURITY WARNING: Slack MCP Server                        │
   │                                                            │
   │  This server grants Claude Code write access to your Slack │
   │  workspace, including the ability to:                      │
   │    • Read all accessible channels and direct messages      │
   │    • Post messages as your bot user                        │
   │    • Access file metadata                                  │
   │                                                            │
   │  Review all operations before confirming in Claude Code.   │
   │  This warning cannot be disabled.                          │
   └─────────────────────────────────────────────────────────────┘`)

       fmt.Print("Type 'I understand the risks' to continue: ")
       var input string
       fmt.Scanln(&input)
       if input != "I understand the risks" {
           return fmt.Errorf("acknowledgment not confirmed")
       }
       return nil
   }
   ```
   - This function is called regardless of `--yes` mode.
   - `--yes` mode reaches this point and fails with: "Slack MCP server requires explicit security acknowledgment. Run `gdev mcp enable slack` interactively."
5. Register Datadog MCP server:
   ```go
   &McpServerEntry{
       Name:         "datadog",
       Description:  "Datadog observability — metrics, logs, traces, monitors, dashboards",
       ToolCount:    12,
       SecurityTier: TierMedium,
       ConfigPolicy: OptionalCatalog,
       DetectFunc:   nil,
       CredentialsNeeded: []string{"DD_API_KEY", "DD_APP_KEY"},
       ServerConfig: func(state ProjectState) McpServerConfig {
           return McpServerConfig{
               Command: "npx",
               Args:    []string{"-y", "@datadog/mcp-server-datadog"},
               Env: map[string]string{
                   "DD_API_KEY": "${DD_API_KEY}",
                   "DD_APP_KEY": "${DD_APP_KEY}",
               },
           }
       },
   }
   ```
6. Register Grafana MCP server:
   ```go
   &McpServerEntry{
       Name:         "grafana",
       Description:  "Grafana dashboards and Loki logs — query metrics, logs, alerts",
       ToolCount:    8,
       SecurityTier: TierMedium,
       ConfigPolicy: OptionalCatalog,
       DetectFunc:   nil,
       CredentialsNeeded: []string{"GRAFANA_URL", "GRAFANA_API_KEY"},
       ServerConfig: func(state ProjectState) McpServerConfig {
           return McpServerConfig{
               Command: "npx",
               Args:    []string{"-y", "@grafana/mcp-server"},
               Env: map[string]string{
                   "GRAFANA_URL":     "${GRAFANA_URL}",
                   "GRAFANA_API_KEY": "${GRAFANA_API_KEY}",
               },
           }
       },
   }
   ```
7. Extend `McpServerEntry` with an optional `EnableHook` field:
   ```go
   // EnableHook is called before enabling the server.
   // If it returns an error, the enable operation is aborted.
   // Used for mandatory security acknowledgments (e.g., Slack).
   EnableHook func() error `json:"-"`
   ```
8. Wire `EnableHook` into `gdev mcp enable`:
   - Before calling `CanEnable`, call `EnableHook()` if non-nil.
   - If `EnableHook` returns an error, abort with the error message.
9. Write unit tests:
   - Atlassian server config uses `URL` field (HTTP transport), not `Command`.
   - Linear server config uses `https://mcp.linear.app/mcp`.
   - Slack `requireSlackAcknowledgment` returns error on wrong input.
   - Slack `requireSlackAcknowledgment` proceeds on correct input.
   - `--yes` mode attempting Slack enable produces actionable error.
   - Datadog and Grafana configs include correct env var references.
   - All catalog servers have `nil` `DetectFunc`.

**Acceptance Criteria:**
- [ ] Atlassian MCP registered: ~15 tools, `TierMedium`, HTTP transport with `ATLASSIAN_SITE_URL`
- [ ] Linear MCP registered: ~12 tools, `TierMedium`, `https://mcp.linear.app/mcp` URL
- [ ] Slack MCP registered: ~10 tools, `TierHigh`, requires `SLACK_BOT_TOKEN` + `SLACK_TEAM_ID`
- [ ] Slack security acknowledgment cannot be bypassed: `--yes` mode fails with instructions
- [ ] Datadog MCP registered: ~12 tools, `TierMedium`, `DD_API_KEY` + `DD_APP_KEY`
- [ ] Grafana MCP registered: ~8 tools, `TierMedium`, `GRAFANA_URL` + `GRAFANA_API_KEY`
- [ ] `McpServerEntry.EnableHook` field added for mandatory pre-enable checks
- [ ] All 5 catalog servers have `ConfigPolicy: OptionalCatalog` and `DetectFunc: nil`
- [ ] OAuth servers (Atlassian, Linear) use HTTP transport `URL` field in generated `.mcp.json`

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/mcp-registry-research.md` — catalog server list, Slack security warning design, OAuth 2.1 HTTP transport for Atlassian/Linear, Datadog/Grafana token requirements

**Status:** Not Started

---

### Unit 28.5: `gdev mcp` Commands

**Description:** Implement the `gdev mcp` command group with subcommands: `list`, `enable`, `disable`, and `check --compliance`.

**Context:** The `gdev mcp` command group is the primary interface for developers managing their MCP server configuration. `gdev mcp list` provides an at-a-glance view of the tool budget and available servers. `gdev mcp enable`/`disable` are the manual counterparts to wizard-driven enablement. `gdev mcp check --compliance` runs the `@yawlabs/mcp-compliance` suite and grades each server.

The `--strict` mode for `gdev mcp check --compliance` is designed for CI: it fails on any server graded below B. This allows teams to maintain a minimum quality bar on their MCP integrations as server implementations evolve.

**Code-Grounded Note:** The Phase 12 tool lifecycle system (`gdev enable`/`gdev disable`) provides the pattern for `gdev mcp enable`/`disable`. The MCP commands should delegate to the same underlying `EnableTool`/`DisableTool` functions (or equivalent) used by the general lifecycle, with MCP-specific pre/post hooks (budget check, credential check, security notice).

**Desired Outcome:** `gdev mcp list` shows the full registry state at a glance. `gdev mcp enable`/`disable` modify `.mcp.json` via tool lifecycle shared-file surgery. `gdev mcp check --compliance` grades each enabled server with a pass/fail for CI.

**Steps:**
1. Implement `gdev mcp list`:
   - Print a table of all servers in the registry: name, status (enabled/disabled), tool count, security tier, description.
   - Enabled servers shown first.
   - Print summary line: "Active: N servers, ~M tools (budget: 40 | remaining: R)".
   - `--json` flag: output structured JSON matching `McpListResult` schema.
   - Example output:
     ```
     MCP Servers                         Status    Tools  Tier    Description
     ─────────────────────────────────────────────────────────────────────────
     context7                            enabled      10  low     Documentation lookup
     github                              enabled      15  medium  GitHub API integration
     sqlite                              enabled       6  low     SQLite database operations
     ─────────────────────────────────────────────────────────────────────────
     terraform                           disabled     10  high    Terraform infrastructure
     atlassian                           disabled     15  medium  Jira and Confluence
     linear                              disabled     12  medium  Linear issue tracking
     slack                               disabled     10  high    Slack workspace
     datadog                             disabled     12  medium  Datadog observability
     grafana                             disabled      8  medium  Grafana dashboards
     sentry                              disabled      8  medium  Sentry error monitoring
     mysql                               disabled      8  medium  MySQL database
     ─────────────────────────────────────────────────────────────────────────
     Active: 3 servers, ~31 tools (budget: 40 | remaining: 9)
     ```
2. Implement `gdev mcp enable <name>`:
   - Look up `name` in registry; fail with "Unknown MCP server. Run `gdev mcp list`." if not found.
   - Load currently enabled servers.
   - Run `EnableHook` if present (for Slack acknowledgment).
   - Call `CanEnable` to check 40-tool ceiling.
   - Prompt for required credentials if `CredentialsNeeded` is non-empty and env vars are not set.
   - Append server to enabled list.
   - Regenerate `.mcp.json` via registry-driven generation.
   - Print: "Enabled mcp-<name> (~N tools). Total: M/40 tools."
3. Implement `gdev mcp disable <name>`:
   - Look up `name` in registry.
   - Remove from enabled list.
   - Regenerate `.mcp.json` via registry-driven generation.
   - Print: "Disabled mcp-<name>. Total: M/40 tools."
4. Implement `gdev mcp check --compliance`:
   ```go
   func runMcpComplianceCheck(enabled []*McpServerEntry, strict bool) error {
       for _, server := range enabled {
           result, err := runComplianceSuite(server)
           if err != nil {
               fmt.Printf("  %s: ERROR — %s\n", server.Name, err)
               continue
           }
           grade := gradeResult(result)
           fmt.Printf("  %s: %s (%d/%d checks passed)\n",
               server.Name, grade, result.Passed, result.Total)
       }

       if strict {
           for _, server := range enabled {
               grade := gradeResult(getResult(server))
               if gradeToInt(grade) < gradeToInt("B") {
                   return fmt.Errorf("compliance check failed: %s graded %s (below B)", server.Name, grade)
               }
           }
       }
       return nil
   }
   ```
   - Grades: A (≥90%), B (≥80%), C (≥70%), D (≥60%), F (<60%).
   - `--strict`: exit 1 if any enabled server grades below B.
   - Uses `@yawlabs/mcp-compliance` npm package if available; degrades gracefully if not installed.
5. Implement credential prompt for `gdev mcp enable` when credentials are missing:
   ```
   Enabling atlassian requires ATLASSIAN_SITE_URL.
   Enter ATLASSIAN_SITE_URL (or press Enter to skip and set manually):
   > mycompany.atlassian.net

   Note: Set ATLASSIAN_SITE_URL in your .env or shell before using this server.
   The value you entered has been added to .gdev.local.yaml for local reference.
   The value is NOT committed to .mcp.json or .gdev.yaml (security).
   ```
6. Wire the command group into the cobra CLI:
   ```go
   mcpCmd := &cobra.Command{Use: "mcp", Short: "Manage MCP server configuration"}
   mcpCmd.AddCommand(mcpListCmd, mcpEnableCmd, mcpDisableCmd, mcpCheckCmd)
   rootCmd.AddCommand(mcpCmd)
   ```
7. Write unit tests:
   - `gdev mcp list` output includes all registry servers.
   - `gdev mcp list --json` produces valid JSON.
   - `gdev mcp enable unknown-server` fails with clear error.
   - `gdev mcp enable <server>` over budget fails with `ToolBudgetExceededError`.
   - `gdev mcp disable <server>` removes from enabled list.
   - `--compliance --strict` exits 1 on sub-B grade.
   - Credential prompt shown when env var missing and not `--yes`.

**Acceptance Criteria:**
- [ ] `gdev mcp list` shows all registry servers with status, tool count, security tier, description
- [ ] Summary line shows active server count, total tools, budget, and remaining
- [ ] `gdev mcp list --json` produces structured JSON output
- [ ] `gdev mcp enable <name>` enforces 40-tool ceiling with actionable error
- [ ] `gdev mcp enable <name>` calls `EnableHook` before enabling (for Slack acknowledgment)
- [ ] `gdev mcp enable <name>` prompts for missing credentials
- [ ] `gdev mcp disable <name>` removes server from `.mcp.json`
- [ ] `gdev mcp check --compliance` grades each enabled server A-F
- [ ] `gdev mcp check --compliance --strict` exits 1 if any server grades below B
- [ ] `gdev mcp enable unknown` fails with "Unknown MCP server. Run `gdev mcp list`."

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/mcp-registry-research.md` — `gdev mcp` command design, compliance check grading, --strict CI mode
- `phases/12-tool-lifecycle-management.md` — shared-file surgery pattern for add/remove operations

**Status:** Not Started

---

### Unit 28.6: MCP Security Documentation Generation

**Description:** Generate per-tier security notices in the CLAUDE.md MCP section, produce `.claude/rules/mcp-security.md` with per-server trust boundaries, and validate that no plaintext credentials appear in `.mcp.json`.

**Context:** Claude Code reads `.claude/rules/` as agent context. The `mcp-security.md` rule file tells the agent which servers are high-risk and what operations to be cautious about. This is not just documentation: the agent reads these rules before taking actions, so clear per-server trust boundaries reduce the chance of unintended writes to production systems. The notices scale with tier: Low tier servers get no warning (read-only local data is fine), Medium tier gets a network-call notice, High tier gets an explicit write-access warning.

Credential validation catches a common mistake: developers embedding API keys directly in `.mcp.json` instead of using `${ENV_VAR}` references. This is a security finding that `gdev check` should catch and `gdev mcp enable` should prevent at creation time.

**Code-Grounded Note:** Phase 4's CLAUDE.md generation uses section markers (`# BEGIN gdev:mcp-servers` / `# END gdev:mcp-servers`). The MCP security section is appended to CLAUDE.md within these markers. The `.claude/rules/mcp-security.md` file is a new gdev-owned file deployed alongside the existing rules from Phase 4.

**Desired Outcome:** The generated `mcp-security.md` rule file contains actionable per-server trust boundaries. CLAUDE.md MCP section includes appropriate tier-based notices. No plaintext credential values appear in `.mcp.json`.

**Steps:**
1. Implement `GenerateMcpSecurityNotice(server *McpServerEntry) string`:
   ```go
   func GenerateMcpSecurityNotice(server *McpServerEntry) string {
       switch server.SecurityTier {
       case TierLow:
           return "" // No notice needed: read-only local access
       case TierMedium:
           return fmt.Sprintf(
               "**%s** makes network requests to %s. "+
               "Review responses before using data in code changes.",
               server.Name, serverServiceName(server),
           )
       case TierHigh:
           return fmt.Sprintf(
               "**%s** has WRITE ACCESS to %s. "+
               "Review every proposed operation before confirming. "+
               "Prefer read-only operations when exploring.",
               server.Name, serverServiceName(server),
           )
       }
       return ""
   }
   ```
2. Implement `GenerateMcpSecurityRulesFile(enabled []*McpServerEntry) string`:
   - Header: `# MCP Server Trust Boundaries — generated by gdev`
   - One section per enabled server that has a non-empty security notice.
   - Each section lists the server name, tier, and trust boundary description.
   - Footer: "Generated by `gdev`. Re-run `gdev init --update` to refresh after changing enabled servers."
3. Write `.claude/rules/mcp-security.md` during `gdev init` and `gdev mcp enable/disable`:
   - Write the file unconditionally (machine-owned, no human-edited sections).
   - Register with Phase 12 file ownership registry.
4. Update CLAUDE.md MCP section via section markers:
   - Within `# BEGIN gdev:mcp-servers` / `# END gdev:mcp-servers`, add tier notices for Medium and High servers.
   - Low-tier servers: listed without notices.
   - Medium-tier: "(network)" label in the server listing.
   - High-tier: "(write access)" label with a one-line warning.
5. Implement credential plaintext validator:
   ```go
   // ValidateMcpJsonCredentials checks that no server config contains literal
   // credential values. Values matching common secret patterns are flagged.
   func ValidateMcpJsonCredentials(mcpJson map[string]McpServerConfig) []CredentialLeak {
       var leaks []CredentialLeak
       for serverName, cfg := range mcpJson {
           for envKey, envValue := range cfg.Env {
               if looksLikeSecret(envKey, envValue) {
                   leaks = append(leaks, CredentialLeak{
                       Server:   serverName,
                       EnvKey:   envKey,
                       Hint:     fmt.Sprintf("Use ${%s} instead of a literal value", envKey),
                   })
               }
           }
       }
       return leaks
   }

   func looksLikeSecret(key, value string) bool {
       // Flags if:
       // - Key contains TOKEN, KEY, SECRET, PASSWORD, CREDENTIAL, AUTH (case-insensitive)
       // - Value does NOT start with "${" (not an env var reference)
       // - Value length > 8 (avoid false positives on short values like "true")
       secretKeyPattern := regexp.MustCompile(`(?i)(token|key|secret|password|credential|auth)`)
       return secretKeyPattern.MatchString(key) &&
           !strings.HasPrefix(value, "${") &&
           len(value) > 8
   }
   ```
6. Call `ValidateMcpJsonCredentials` in:
   - `gdev mcp enable <name>`: validate the server config before writing.
   - `gdev check` (Unit 28.5's `gdev mcp check --compliance`): report leaks as `SeverityCritical` findings.
   - `gdev init --update`: validate after regeneration.
7. Write unit tests:
   - Low tier server: `GenerateMcpSecurityNotice` returns empty string.
   - Medium tier server: notice mentions "network requests".
   - High tier server: notice mentions "WRITE ACCESS".
   - `looksLikeSecret`: flags `SLACK_BOT_TOKEN = "xoxb-actual-token"`, not `SLACK_BOT_TOKEN = "${SLACK_BOT_TOKEN}"`.
   - `looksLikeSecret`: does not flag `MYSQL_PORT = "3306"`.
   - `GenerateMcpSecurityRulesFile`: only includes servers with non-empty notices.
   - Credential validation blocks `gdev mcp enable` when literal value provided.

**Acceptance Criteria:**
- [ ] Low-tier servers produce no security notice
- [ ] Medium-tier servers produce "makes network requests to [service]" notice
- [ ] High-tier servers produce "has WRITE ACCESS to [service]" notice
- [ ] `.claude/rules/mcp-security.md` generated with per-server trust boundaries for non-Low servers
- [ ] CLAUDE.md MCP section updated with tier labels for Medium and High servers
- [ ] `ValidateMcpJsonCredentials` detects literal secret values in `.mcp.json` env fields
- [ ] `gdev mcp enable` blocks if server config contains a likely-literal credential value
- [ ] `gdev check` reports credential leaks as `SeverityCritical`
- [ ] Env var references (`${TOKEN}`) correctly excluded from credential leak detection

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/mcp-registry-research.md` — per-tier notice design, trust boundary language, credential leak detection approach
- `phases/04-claude-code-addon.md` — CLAUDE.md section marker pattern, `.claude/rules/` file deployment

**Status:** Not Started

---

### Unit 28.7: Registry-Driven .mcp.json Generation

**Description:** Rewrite Phase 4's `.mcp.json` generation to iterate the registry, support devenv 2.0's native `claude.code.mcpServers` module as the primary path, and fall back to direct `.mcp.json` generation for devenv < 2.0. Section markers enable surgical add/remove by the tool lifecycle system.

**Context:** Phase 4 generates `.mcp.json` from a hardcoded template. Unit 28.7 replaces the hardcoded iteration with a registry pass: for each enabled server, call `ServerConfig(state)` to get the JSON-serializable config, marshal it, and write the file. This is a breaking change to the generation code but a non-breaking change to the output format — the same `.mcp.json` structure is produced.

devenv 2.0 introduced a native `claude.code.mcpServers` NixOS module option (confirmed in `gdev-ecosystem-expansion-assessment` spike). When devenv >= 2.0 is detected, gdev should prefer writing to `devenv.nix` via the native module (which generates `.mcp.json` as a devenv output) rather than writing `.mcp.json` directly. This avoids conflicts when devenv also manages `.mcp.json`. For devenv < 2.0, the existing direct-write path remains.

**Code-Grounded Note:** The Phase 12 shared-file surgery pattern uses section markers to allow `gdev mcp enable/disable` to add/remove individual servers without rewriting the entire file. The markers in `.mcp.json` are structured as comments is not valid JSON — use a different approach: maintain a separate `~/.qsdev/mcp-state/<project-hash>/enabled.json` file as the canonical enabled-server list, and regenerate `.mcp.json` from scratch on every change. This is simpler than surgical JSON editing and avoids invalid JSON from comments.

**Desired Outcome:** `.mcp.json` is generated from the registry. Adding or removing servers via `gdev mcp enable/disable` regenerates the file cleanly. devenv 2.0 projects use the native module path. The generated file never contains plaintext credentials.

**Steps:**
1. Implement the canonical enabled-server state file:
   ```go
   // EnabledServersState tracks which servers are enabled for a project.
   // Stored at: .gdev/mcp-enabled.json (inside the gdev state directory)
   type EnabledServersState struct {
       Servers []string `json:"servers"` // server names in registry
   }

   func LoadEnabledServersState(projectRoot string) (*EnabledServersState, error)
   func SaveEnabledServersState(projectRoot string, state *EnabledServersState) error
   ```
2. Implement registry-driven `.mcp.json` generation:
   ```go
   func GenerateMcpJson(enabled []*McpServerEntry, projectState ProjectState) ([]byte, error) {
       type mcpRoot struct {
           McpServers map[string]McpServerConfig `json:"mcpServers"`
       }

       servers := make(map[string]McpServerConfig)
       for _, entry := range enabled {
           servers[entry.Name] = entry.ServerConfig(projectState)
       }

       root := mcpRoot{McpServers: servers}
       return json.MarshalIndent(root, "", "  ")
   }
   ```
3. Implement devenv version detection:
   ```go
   func detectDevenvVersion(projectRoot string) (string, error) {
       out, err := exec.Command("devenv", "--version").Output()
       if err != nil {
           return "", err
       }
       // Parse version from output, e.g., "devenv 2.1.0"
       return parseVersionFromOutput(string(out)), nil
   }

   func supportsNativeMcpModule(devenvVersion string) bool {
       v, err := semver.Parse(devenvVersion)
       if err != nil {
           return false
       }
       return v.GTE(semver.MustParse("2.0.0"))
   }
   ```
4. Implement the devenv 2.0 native module path:
   ```go
   func writeMcpViaDevenvModule(enabled []*McpServerEntry, projectState ProjectState) error {
       // Generate Nix expression for claude.code.mcpServers
       nixExpr := generateMcpNixConfig(enabled, projectState)

       // Inject into devenv.nix within the gdev-mcp section markers
       return injectIntoDevenvNix(nixExpr, "gdev:mcp-servers")
   }

   func generateMcpNixConfig(enabled []*McpServerEntry, state ProjectState) string {
       // Produce:
       // claude.code.mcpServers = {
       //   context7 = { command = "npx"; args = ["-y" "@upstash/context7-mcp"]; };
       //   github = { ... };
       // };
   }
   ```
5. Implement the fallback direct-write path (devenv < 2.0):
   ```go
   func writeMcpJsonDirect(enabled []*McpServerEntry, projectState ProjectState, projectRoot string) error {
       content, err := GenerateMcpJson(enabled, projectState)
       if err != nil {
           return err
       }

       // Validate: no plaintext credentials
       var parsed map[string]map[string]McpServerConfig
       json.Unmarshal(content, &parsed)
       leaks := ValidateMcpJsonCredentials(parsed["mcpServers"])
       if len(leaks) > 0 {
           return fmt.Errorf("refusing to write .mcp.json: credential values detected\n%s",
               formatLeaks(leaks))
       }

       return os.WriteFile(filepath.Join(projectRoot, ".mcp.json"), content, 0644)
   }
   ```
6. Implement the dispatch function called by `gdev mcp enable/disable` and `gdev init`:
   ```go
   func WriteMcpConfig(enabled []*McpServerEntry, projectState ProjectState, projectRoot string) error {
       devenvVer, _ := detectDevenvVersion(projectRoot)
       if supportsNativeMcpModule(devenvVer) {
           return writeMcpViaDevenvModule(enabled, projectState)
       }
       return writeMcpJsonDirect(enabled, projectState, projectRoot)
   }
   ```
7. Migrate Phase 4's `generateMcp()` function to call `WriteMcpConfig` instead of the hardcoded template.
8. Write unit tests:
   - Registry-driven generation: enabled servers produce correct `.mcp.json` structure.
   - Empty enabled list: produces `{"mcpServers": {}}`.
   - Server with HTTP transport: `url` field in output, no `command`/`args`.
   - devenv version < 2.0: direct `.mcp.json` write path taken.
   - devenv version >= 2.0: devenv module path taken.
   - Plaintext credential detection blocks write.
   - Order of servers in output is deterministic (alphabetical by name).

**Acceptance Criteria:**
- [ ] `.mcp.json` generation iterates registry instead of hardcoded server list
- [ ] Each enabled server's `ServerConfig(state)` function called to produce JSON config
- [ ] HTTP transport servers (Atlassian, Linear) use `url` field in output, not `command`/`args`
- [ ] devenv >= 2.0 detected; native `claude.code.mcpServers` module used as primary path
- [ ] devenv < 2.0: direct `.mcp.json` write path used as fallback
- [ ] `WriteMcpConfig` called by `gdev mcp enable`, `gdev mcp disable`, and `gdev init`
- [ ] Plaintext credential values block `.mcp.json` write with clear error
- [ ] Empty enabled server list produces valid `{"mcpServers": {}}` output
- [ ] Output is deterministic: same enabled servers always produce identical `.mcp.json`
- [ ] Phase 4's `generateMcp()` migrated to use registry-driven path

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/mcp-registry-research.md` — registry-driven generation design, devenv 2.0 native module path
- `phases/04-claude-code-addon.md` — existing `.mcp.json` generation code reference, section marker pattern

**Status:** Not Started

---

## Code-Grounded Implementation Notes

### New Files

| File | Purpose |
|------|---------|
| `internal/mcp/registry.go` | `McpServerRegistry`, `McpServerEntry`, `McpServerConfig` types |
| `internal/mcp/servers.go` | All server registrations (MVP migrated + new servers) |
| `internal/mcp/detect.go` | Detection functions: MySQL, SQLite, Terraform, Sentry |
| `internal/mcp/generate.go` | Registry-driven `.mcp.json` generation and devenv 2.0 module path |
| `internal/mcp/security.go` | `GenerateMcpSecurityNotice`, `ValidateMcpJsonCredentials`, rules file generation |
| `internal/mcp/compliance.go` | `gdev mcp check --compliance` implementation |
| `cmd/mcp.go` | `gdev mcp` cobra command group |

### Existing Code to Migrate

| Location | Change |
|----------|--------|
| `addons/claudecode/generate_mcp.go` (Phase 4) | Replace hardcoded server iteration with `WriteMcpConfig(registry.Enabled(...), state, root)` |
| Phase 4 `.mcp.json` server list | Enumerate exact server names and configs; use these to populate `internal/mcp/servers.go` registry entries |

### Registry Server Count Summary

| Config Policy | Servers | Total Tools |
|---------------|---------|-------------|
| AutoDetect | context7, github, sqlite, mysql (+ ecosystem-specific from Phase 4) | ~39 |
| DetectAndOffer | terraform, sentry | ~18 |
| OptionalCatalog | atlassian, linear, slack, datadog, grafana | ~57 |

Note: The 40-tool ceiling applies to the set of *enabled* servers at any given time, not to the total registry. A project can have context7 + github + sqlite enabled (31 tools) without touching the catalog servers.

### Security Tier Summary

| Tier | Servers | What Justifies It |
|------|---------|-------------------|
| Low | context7, sqlite, ecosystem-doc servers | Local files or static data, no network, no credentials |
| Medium | github, mysql, sentry, atlassian, linear, datadog, grafana | Network calls or credential-holding, read-only or low-blast-radius writes |
| High | terraform, slack | Infrastructure apply or team communication write access |

---

## Phase Completion Criteria

- [ ] All seven units pass acceptance criteria
- [ ] All MVP Phase 4 servers migrated to registry format with correct ToolCount, SecurityTier, and ConfigPolicy
- [ ] `gdev mcp list` shows full registry state including tool budget summary
- [ ] `gdev mcp enable context7` succeeds; `gdev mcp enable atlassian` (over budget) fails with clear error
- [ ] MySQL auto-detection: detected in a project with devenv MySQL service + mysql2 dependency
- [ ] SQLite auto-detection: detected in a project with `.sqlite` files
- [ ] Terraform detect-and-offer: wizard prompt shown when `*.tf` files present
- [ ] Sentry detect-and-offer: wizard prompt shown when Sentry SDK imported
- [ ] Slack security acknowledgment cannot be bypassed; `--yes` flag fails with instructions
- [ ] `gdev mcp check --compliance --strict` exits 1 on sub-B graded server
- [ ] `.mcp.json` contains no plaintext credential values (validated on write)
- [ ] `.claude/rules/mcp-security.md` generated with correct notices for Medium/High tier servers
- [ ] devenv 2.0 native `claude.code.mcpServers` module path used when available
- [ ] devenv < 2.0 direct `.mcp.json` write path used as fallback
