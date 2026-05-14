# Rudel: Claude Code Session Analytics Platform — Deep Dive

## Overview

Rudel is a team-oriented analytics platform for AI coding agent sessions, built by the team behind ObsessionDB (a managed ClickHouse provider). It ingests full session transcripts from Claude Code and OpenAI Codex via CLI hooks, stores them in ClickHouse, and provides a 15-view React dashboard covering developer productivity, project health, ROI, error patterns, and session archetypes. It is the most architecturally ambitious tool in the Claude Code analysis ecosystem — effectively a full SaaS product with multi-tenant organizations, not just a developer utility.

**Repository**: [github.com/obsessiondb/rudel](https://github.com/obsessiondb/rudel)
**Hosted service**: app.rudel.ai
**License**: MIT | **Stars**: 223 | **Forks**: 12 | **Contributors**: 7
**Created**: 2026-02-16 | **Latest**: v0.1.9 (2026-03-11)

## Architecture

### Data Flow

```
Claude Code Session Ends
        │
        ▼
Claude Code Hook (registered by `rudel enable`)
        │
        ▼
`rudel hook-upload` (reads session from stdin)
        │  Attaches: git remote, branch, SHA, package name, org ID
        │  Classifies: session content tag (research, bug_fix, refactoring, etc.)
        ▼
HTTP POST → Rudel API Server (Bun on Fly.io)
        │
        ├──► Postgres (Neon) — auth tables (users, sessions, accounts, tokens)
        │    via Drizzle ORM + better-auth
        │
        └──► ClickHouse (ObsessionDB) — session transcripts + analytics
             via chkit toolkit
             │
             ├── claude_sessions table (ReplacingMergeTree, monthly partitions, 365d TTL, S3 storage)
             ├── codex_sessions table (same structure, minus subagents)
             └── session_analytics table ◄── populated by materialized views
                 (40+ columns, 5 SET indexes)
```

The key architectural insight: ClickHouse materialized views do the heavy lifting. When a session is inserted into `claude_sessions`, a materialized view automatically parses the JSONL content, extracts timestamps, calculates inference gaps, aggregates token usage, identifies skills/subagents/commands, and classifies the session into an archetype. This means analytics are computed at write time, not query time — enabling fast dashboard reads at scale.

### Monorepo Structure

Turborepo with Bun, three apps and five packages:

- **apps/api** — Bun HTTP server handling auth, RPC endpoints, and ClickHouse ingestion
- **apps/cli** — npm-distributed CLI (`rudel`) with 9 commands: login/logout/whoami, enable/disable, upload, hook-upload, set-org, config-show
- **apps/web** — React SPA (Vite-built, served as static files from the API)
- **packages/agent-adapters** — Pluggable adapter interface for different AI agents (currently Claude Code and Codex)
- **packages/api-routes** — Shared RPC contracts using `@orpc/contract` for type-safe API validation
- **packages/ch-schema** — ClickHouse schema definitions with `chkit` migration toolkit
- **packages/sql-schema** — Postgres schema via Drizzle ORM

### Hook Mechanism

`rudel enable` registers a Claude Code hook that fires on session completion. The hook pipes session data via stdin to `rudel hook-upload`, which:

1. Reads session metadata (ID, transcript path, working directory) from stdin
2. Loads auth credentials from local config
3. Builds upload request via the agent adapter (attaches git info, org ID)
4. Classifies session content and assigns a tag
5. POSTs to the API endpoint with retry logic

The adapter pattern is designed for extensibility — OpenAI Codex support was added in v0.1.6 by implementing a new adapter, not modifying core upload logic.

### Query Layer

The API uses `@orpc/contract` for type-safe RPC with six analytics endpoint categories:
- Overview metrics (KPIs, usage trends, success rates)
- Developer-level analysis
- Project tracking
- Session details
- ROI calculations
- Error monitoring and learning insights

ClickHouse's columnar storage and pre-computed materialized views make these aggregation queries fast even across thousands of sessions.

### Frontend

React SPA with 15 dashboard pages:
- **OverviewPage** — Org-wide KPIs, usage trends, success rates
- **DevelopersListPage / DeveloperDetailPage** — Per-developer metrics and drill-downs
- **ProjectsListPage / ProjectDetailPage** — Project-level analytics
- **SessionsListPage / SessionDetailPage** — Session browsing and individual analysis
- **ErrorsPage** — Error pattern monitoring
- **LearningsPage** — Extracted learning insights
- **ROIPage** — Return on investment calculations
- **OrganizationPage / InvitationsPage / CreateOrgPage** — Multi-tenant org management
- **ProfilePage / AdminPage** — User and admin functions

## Key Features and Metrics

### Session Archetypes

ClickHouse materialized views classify each session into one of six archetypes based on duration, token ratios, error frequency, and interaction patterns:

| Archetype | Meaning |
|-----------|---------|
| **quick_win** | Short, successful sessions |
| **deep_work** | Extended, focused sessions |
| **struggle** | High error rates, many retries |
| **exploration** | Broad, investigative sessions |
| **abandoned** | Sessions dropped early |
| **standard** | Default classification |

### Computed Analytics (40+ columns)

- **Token metrics**: input_tokens, output_tokens, cache_creation_tokens, cache_read_tokens
- **Interaction timing**: quick_responses, normal_responses, long_pauses (derived from inference gap analysis)
- **Performance scoring**: success_score (composite metric based on commit presence, token ratios, error frequency)
- **Content extraction**: model_used, skills detected, subagent counts, commands used
- **Quality ranking**: Based on commit presence, output/input token ratio, error density

### Team-Level Analytics

This is Rudel's differentiator. While ccusage, Subtle, and other tools provide individual developer metrics, Rudel provides:

- **Organization-scoped dashboards**: All team members' sessions aggregated
- **Developer comparison**: See which team members are most effective with AI coding
- **Project-level rollups**: Token spend and success rates per project across all developers
- **ROI calculations**: Attempt to quantify the value of AI-assisted development
- **Error pattern detection**: Cross-developer error trends (what types of tasks fail most?)
- **Learning insights**: Extract patterns about what makes sessions succeed or fail

### Multi-Agent Support

The adapter system (`packages/agent-adapters`) provides a clean interface for adding new AI coding agents. Each adapter handles:
- Session discovery (knowing where files live on disk)
- Hook management (installing/removing the trigger)
- Upload request building (parsing agent-specific formats)
- Server-side ingestion (extracting timestamps, writing to ClickHouse)

Currently supports Claude Code and OpenAI Codex, with the architecture ready for more.

## Real-World Usage: The 1,573-Session Dataset

From the HN discussion (144 points, 86 comments), the Rudel team shared findings from analyzing 1,573 sessions, 15M+ tokens, and 270K+ interactions over three months from their six-person team:

- **Skills activated in only 4% of sessions** — a surprisingly low adoption rate for CLAUDE.md skills. Improved with Claude 4.6 model.
- **26% of sessions abandoned within 60 seconds** — interpretation debated: failure or efficient quick-check?
- **Documentation tasks scored highest success; refactoring scored lowest** — validates intuition about AI coding strengths.
- **Initial 2-minute error patterns predict abandonment reliably** — early errors are a strong signal that a session will be abandoned.
- **No established benchmarks exist for agentic session performance** — the team noted this gap, positioning Rudel as a benchmarking platform.
- **Structured pipelines with review gates boosted first-pass acceptance from 73% to 90%** — shared by a commenter, not Rudel's own data.

## Tradeoffs and Limitations

### Deployment Burden

ClickHouse is not a lightweight dependency. Self-hosting requires:
- ClickHouse instance (the team's own ObsessionDB offers free tier, but running your own ClickHouse is operationally heavy)
- Postgres instance (for auth — lighter, but still a second database)
- Bun/Node.js app server
- Schema migrations via `chkit` (beta toolkit with known bug: generates materialized views before dependent tables)

The self-hosting guide targets Fly.io + Neon + ObsessionDB, all with free tiers, but this is still three external services to manage. Docker Compose exists for local development but isn't documented as a production path.

Compared to tools like ccusage (zero external dependencies, reads local files) or Subtle (single binary), this is a significant jump in operational complexity.

### Privacy Implications

This is the most serious concern. Rudel uploads **complete session transcripts** to a remote server. The README explicitly warns:

> "Uploaded transcripts and related metadata may contain sensitive material, including source code, prompts, tool output, file contents, command output, URLs, and secrets that appeared during a session."

This means:
- **Source code** from every file Claude reads or writes goes to Rudel's servers
- **Secrets** (API keys, credentials) that appear in sessions are transmitted
- **Proprietary business logic** is exposed
- **Git context** (remote URLs, branches, SHAs) reveals project structure

The hosted service claims encryption and limited product analytics (no session replay or blanket click capture in their own analytics). But the data is still in a third-party ClickHouse instance.

Self-hosting mitigates this but adds operational burden. For enterprise teams with security requirements, the hosted version is likely unacceptable without self-hosting.

HN commenters raised this concern prominently. The team's response was to emphasize self-hosting via Docker and local-only operation.

### SaaS vs. Self-Hosted

| Aspect | Hosted (app.rudel.ai) | Self-Hosted |
|--------|----------------------|-------------|
| Setup effort | Minutes (npm install, login, enable) | Hours (3 services, migrations, DNS) |
| Operational burden | Zero | Ongoing (ClickHouse, Postgres, app server) |
| Data privacy | Third-party controlled | Full control |
| Cost | Free tier available | Infrastructure costs (free tiers exist) |
| Updates | Automatic | Manual (pull, migrate, deploy) |
| Enterprise suitability | Likely blocked by security review | Viable with proper deployment |

### Maturity Assessment

**Positive signals:**
- 223 stars, 144 HN points — strong initial reception
- 7 contributors, rapid iteration (5 releases in 9 days: v0.1.5 through v0.1.9)
- MIT license, clean monorepo structure
- Real-world validation on 1,573+ sessions
- Professional engineering: type-safe RPC, pluggable adapters, conventional commits, CI enforcement

**Caution signals:**
- Very early (v0.1.x, created 2026-02-16 — ~5 weeks old at time of research)
- Schema migration tool (`chkit`) is beta with known ordering bugs
- Prompt injection vulnerability found and fixed in v0.1.9 — suggests security review is still maturing
- Auth flow changed significantly between versions (loopback token to device code)
- Only 3 ClickHouse migrations total — schema is still evolving
- The team behind it (ObsessionDB) is also an early-stage company — Rudel may partly serve as a showcase for their ClickHouse hosting

### Scale Considerations

ClickHouse is designed for massive analytical workloads, so scale is unlikely to be a bottleneck for session analytics. The 365-day TTL and monthly partitioning are sensible defaults. S3 storage policy (added in v0.1.8) moves data to object storage for cost efficiency.

However:
- Materialized views that parse full JSONL content at write time could become expensive with very large sessions
- The session_analytics table with 40+ columns and 5 SET indexes adds write amplification
- For a small team (the stated use case), this is fine. For a 100-person org uploading thousands of daily sessions, ingestion latency and ClickHouse sizing would need attention.

## Comparison: Team Analytics vs. Individual Tools

| Dimension | Rudel | ccusage | Subtle |
|-----------|-------|---------|--------|
| **Scope** | Team/org analytics | Individual usage tracking | Individual session search |
| **Deployment** | SaaS or self-hosted (3 services) | Local CLI, zero deps | Local CLI, single binary |
| **Data location** | Remote (ClickHouse) or self-hosted | Local only | Local only |
| **Session content** | Uploaded in full | Read locally | Indexed locally |
| **Key insight** | Cross-developer patterns, ROI | Cost tracking, daily/monthly spend | Semantic search over sessions |
| **Privacy risk** | High (full transcripts uploaded) | None (local only) | None (local only) |
| **Multi-agent** | Claude Code + Codex | Claude Code only | Claude Code only |
| **Team features** | Orgs, invitations, dev comparison | None | None |
| **Maturity** | v0.1.9, 5 weeks old | v1.x+, 12k stars | Early, ~100 stars |

Rudel occupies a unique position: it is the only tool that provides organizational-level analytics across multiple developers and projects. Every other tool in the ecosystem is designed for individual use. This makes it valuable for engineering managers and team leads who want to understand AI coding patterns at scale — but the privacy tradeoff is significant.

## Failure Modes

1. **Sensitive data leakage**: The most critical failure mode. Sessions containing API keys, credentials, or proprietary code are uploaded verbatim. No scrubbing or redaction is mentioned.

2. **Hook reliability**: If the Claude Code hook fails silently (process crash, network timeout), sessions are lost without notification. The retry mechanism (added v0.1.6) helps but requires manual `rudel upload --retry`.

3. **Schema evolution**: With only 3 migrations and a beta migration toolkit that generates operations in wrong order, schema changes require careful manual intervention. Breaking migrations could corrupt analytics data.

4. **Prompt injection**: v0.1.9 fixed a prompt injection vulnerability in the session classifier. This suggests the classifier uses LLM-based content analysis — meaning adversarial session content could manipulate classification. The fix was applied, but the attack surface exists.

5. **Vendor lock-in to ObsessionDB**: The self-hosting guide defaults to ObsessionDB for ClickHouse. While any ClickHouse instance works, the `chkit` toolkit and `@chkit/*` packages are ObsessionDB products (all at beta versions). If ObsessionDB pivots or shuts down, the migration tooling would need replacement.

6. **ClickHouse operational complexity**: For teams that self-host, ClickHouse requires tuning (memory, merge settings, disk space for S3 offload). This is not a "deploy and forget" database.

## Key Findings

1. **Rudel is architecturally the most sophisticated tool in the ecosystem** — a full SaaS analytics platform with multi-tenant orgs, pluggable agent adapters, materialized view analytics, and a 15-view dashboard. Nothing else comes close in scope.

2. **The team analytics angle is unique and valuable** — no other tool provides cross-developer comparison, project-level rollups, or ROI calculations. For engineering leaders, this is the only option.

3. **The privacy tradeoff is the central tension** — full session transcripts (including source code, secrets, and file contents) uploaded to a remote server. Self-hosting mitigates but adds significant operational cost.

4. **Maturity is early but velocity is high** — 5 releases in 9 days, MIT licensed, 7 contributors, professional engineering practices. But it is 5 weeks old with beta-stage tooling.

5. **The 1,573-session dataset produced genuinely novel insights** — session archetype classification, abandonment prediction from early errors, and skill adoption rates are metrics that did not exist before this tool.

6. **ObsessionDB has a strategic interest** — Rudel showcases their managed ClickHouse product. This could mean sustained investment, or it could mean the tool is deprioritized if their core business changes direction.

## Sources

- `docs/rudel-readme.md` — GitHub README summary
- `docs/github-rudel.md` — GitHub page overview
- `docs/hn-rudel.md` — HN discussion (144 points, 86 comments)
- `docs/rudel-self-hosting.md` — Full self-hosting documentation
- `docs/rudel-architecture-deep-dive.md` — Monorepo structure, schema, CLI, adapter interface, release history
