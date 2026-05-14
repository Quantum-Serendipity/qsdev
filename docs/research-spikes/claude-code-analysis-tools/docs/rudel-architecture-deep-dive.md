<!-- Source: Multiple GitHub URLs from https://github.com/obsessiondb/rudel -->
<!-- Retrieved: 2026-03-26 -->

# Rudel Architecture Deep Dive

## Repository Metadata (as of 2026-03-26)

- **Stars**: 223
- **Forks**: 12
- **Contributors**: 7
- **Open Issues**: 3
- **Language**: TypeScript 98.9%
- **License**: MIT
- **Created**: 2026-02-16
- **Latest**: v0.1.9 (2026-03-11)
- **Package Manager**: Bun 1.3.5

## Monorepo Structure (Turborepo)

```
rudel/
├── apps/
│   ├── api/        — @rudel/api: HTTP server (Bun), auth, RPC, ClickHouse ingestion
│   ├── cli/        — rudel: npm CLI for session upload, hooks, login
│   └── web/        — @rudel/web: React SPA dashboard
├── packages/
│   ├── agent-adapters/  — Pluggable adapters (Claude Code, Codex)
│   ├── api-routes/      — Shared RPC contracts (@orpc/contract)
│   ├── ch-schema/       — ClickHouse schema + chkit migrations
│   ├── sql-schema/      — Postgres schema (Drizzle ORM)
│   └── typescript-config/
├── docker-compose.yml   — Local dev: Postgres 16 + ClickHouse
├── Dockerfile           — Single-stage Bun build for deployment
├── fly.toml             — Fly.io deployment config
└── turbo.json           — Turborepo pipeline
```

## CLI Commands

From `apps/cli/src/commands/`:
- `login.ts` / `logout.ts` / `whoami.ts` — Authentication
- `enable.ts` — Install Claude Code hooks + select org + optional retroactive upload
- `disable.ts` — Remove hooks
- `upload.ts` — Batch upload with interactive project picker
- `hook-upload.ts` — Hook-triggered upload (reads stdin, logs to file)
- `set-org.ts` — Change organization for current project
- `config-show.ts` — Display current configuration
- `hooks/` — Hook definitions subdirectory
- `dev/` — Development utilities

## Agent Adapter System

The `agent-adapters` package provides a pluggable adapter interface:

```typescript
interface AgentAdapter {
  name: string;
  source: Source;  // type-safe enum
  rawTableName: string;

  // Session Discovery
  getSessionsBaseDir(): string;
  findProjectSessions(projectPath: string): Promise<SessionFile[]>;
  scanAllSessions(): Promise<ScannedProject[]>;

  // Hook Management
  getHookConfigPath(): string;
  installHook(): void;
  removeHook(): void;
  isHookInstalled(): boolean;

  // Upload Request Building
  buildUploadRequest(session: SessionFile, context: UploadContext): Promise<IngestSessionInput>;

  // Server-side Ingestion
  extractTimestamps(content: string): { sessionDate: string; lastInteractionDate: string } | null;
  ingest(ingestor: Ingestor, input: IngestSessionInput, context: IngestContext): Promise<void>;
}
```

Current adapters: `claude-code`, `codex` (OpenAI Codex, added in v0.1.6)

## ClickHouse Schema

Three tables in `rudel` database:

### claude_sessions
- ReplacingMergeTree, partitioned by month, 365-day TTL
- Fields: session_id, organization_id, project_path, git metadata, subagents (Map type)
- S3 storage policy (migrated in v0.1.8)

### codex_sessions
- Same structure minus subagents field
- S3 storage policy

### session_analytics (materialized view target)
- 40+ columns populated by materialized views
- Token metrics: input_tokens, output_tokens, cache_creation_tokens, cache_read_tokens
- Interaction metrics: quick_responses, normal_responses, long_pauses
- Performance: success_score, session_archetype
- SET indexes on: git_remote, model_used, project_path, source, user_id
- Session archetypes: quick_win, deep_work, struggle, exploration, abandoned, standard

## Dashboard Pages (15 views)

From `apps/web/src/pages/dashboard/`:
1. **OverviewPage** — KPIs, usage trends, success rates
2. **DevelopersListPage** — Team member listing
3. **DeveloperDetailPage** — Individual developer metrics
4. **ProjectsListPage** — Project-level analytics
5. **ProjectDetailPage** — Per-project drill-down
6. **SessionsListPage** — Session browsing
7. **SessionDetailPage** — Individual session analysis
8. **ErrorsPage** — Error monitoring
9. **LearningsPage** — Learning insights extraction
10. **ROIPage** — Return on investment calculations
11. **OrganizationPage** — Org settings
12. **InvitationsPage** — Team invitations
13. **CreateOrgPage** — Organization creation
14. **ProfilePage** — User profile
15. **AdminPage** — Administrative functions

## API Routes Architecture

Uses `@orpc/contract` for type-safe RPC. Analytics subsystem has 6 major categories:
- Overview metrics (KPIs, usage trends, success rates)
- Developer-level analysis
- Project tracking
- Session details
- ROI calculations
- Error monitoring + learning insights

## Deployment Architecture

- **Hosted**: app.rudel.ai (Fly.io + ObsessionDB ClickHouse + Neon Postgres)
- **Self-hosted**: Docker Compose for local, or any Bun/Node + Postgres + ClickHouse
- **Product analytics**: PostHog (opt-in via build args), Chatwoot for support chat
- **Auth**: better-auth library with GitHub/Google OAuth or email/password
- **CLI analytics**: PostHog events for enable/disable/upload tracking

## Release History

| Version | Date | Key Changes |
|---------|------|-------------|
| v0.1.5 | 2026-03-02 | Multi-tenant orgs, interactive upload picker, retroactive uploads |
| v0.1.6 | 2026-03-02 | OpenAI Codex support, structured logging, retry/progress for uploads |
| v0.1.7 | 2026-03-03 | Fix npm install (agent-adapters to devDependencies) |
| v0.1.8 | 2026-03-03 | Developer name resolution via git remote, removed repository column |
| v0.1.9 | 2026-03-11 | Prompt injection fix in session classifier, device code auth flow |

## Key Technical Details

- `chkit` toolkit for ClickHouse schema management (beta: v0.1.0-beta.16)
- Known issue: chkit generates materialized views before dependent tables — manual reordering required
- Conventional commits enforced, Release Please for versioning
- CLI auth transitioned from loopback token to device code flow (v0.1.9)
- Session classifier had a prompt injection vulnerability (fixed v0.1.9)
