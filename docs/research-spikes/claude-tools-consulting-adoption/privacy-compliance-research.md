# Privacy & Compliance Analysis: Claude Code Analysis Tools for Consulting Firms

## Executive Summary

Consulting firms handling client intellectual property face a layered privacy challenge when adopting Claude Code analysis tools. The first layer is Claude Code itself — every prompt, file read, and tool output is sent to Anthropic's API over the network and stored locally in `~/.claude/projects/` as JSONL transcripts. The second layer is the analysis tools that read those transcripts. Tools range from fully local (claude-history, Claude DevTools) to full SaaS upload (Rudel), with the privacy risk multiplying at each step: client code enters Claude Code, gets stored in local session files, and may then be read, indexed, uploaded, or shared by analysis tools.

For a consulting firm like Highspring Digital with multiple client engagements under separate NDAs, the critical questions are: (1) does using Claude Code itself violate client NDAs, (2) do the analysis tools create additional exposure, and (3) can sessions from different clients be isolated to prevent cross-contamination?

---

## 1. Data Classification: What Sensitive Data Appears in Claude Code Sessions

Based on the session data format analysis (`session-data-format-research.md`), Claude Code JSONL files contain the following categories of sensitive data:

### 1.1 Client Source Code (HIGH RISK)

Every `Read` tool result includes the **full file content** with path, line numbers, and metadata. Every `Edit` and `Write` tool call includes the content being written. A single Claude Code session working on a client project may contain dozens of complete source files embedded in tool results. Session files range from 2KB to 28MB+, with a typical substantive session at 1-3MB.

**Consulting-specific risk**: A session working on Client A's codebase contains Client A's proprietary source code. If that session is later analyzed by a tool that uploads transcripts (Rudel), or if the session file is inadvertently accessed during a Client B engagement, it constitutes an NDA breach.

### 1.2 Credentials and Secrets (CRITICAL RISK)

Sessions routinely contain:
- **API keys and tokens** in configuration files read by Claude Code
- **Database connection strings** in environment configs
- **Private keys and certificates** if read during debugging
- **OAuth secrets** in application configuration
- **Cloud provider credentials** (AWS, GCP, Azure) in deployment configs

These appear in tool results when Claude Code reads `.env` files, configuration files, or source code containing hardcoded credentials. The JSONL format stores them as plain text.

### 1.3 Architecture and Business Logic (MEDIUM-HIGH RISK)

Claude Code's thinking blocks and text responses contain:
- **System architecture descriptions** synthesized from reading codebases
- **Business logic explanations** generated during code analysis
- **Database schema details** from migration files
- **API endpoint structures** from route definitions
- **Infrastructure topology** from deployment configurations

This is arguably more dangerous than raw code because it's pre-synthesized into readable form — an attacker or unauthorized viewer gets a structured understanding of the system without needing to read code.

### 1.4 Personal Data / PII (MEDIUM RISK)

Sessions may contain:
- **Developer names and emails** from git commits, package.json, code comments
- **User data in test fixtures** or seed files
- **Customer PII in error messages** or log output from `Bash` tool runs
- **Email addresses and usernames** in configuration files

Under GDPR, code repositories contain personal data (developer names in git commits, emails in configs). When Claude Code reads these files, the PII enters session transcripts and is transmitted to Anthropic's API — this constitutes third-party data processing under European law.

### 1.5 Session Metadata (LOW-MEDIUM RISK)

Every JSONL entry includes:
- **Working directory path** (`cwd`) — reveals project name and directory structure
- **Git branch name** — reveals feature names, ticket numbers
- **Git remote URL** — stored in Rudel uploads, reveals repository location
- **Timestamps** — reveals work patterns, hours, engagement timeline
- **Model used** — reveals AI tool adoption patterns
- **Session slug** — human-readable name that may reference client or project

Even metadata alone can reveal which clients a firm works with, what projects are active, and development timelines.

---

## 2. Claude Code Itself: The Foundation Layer

Before evaluating analysis tools, the foundational privacy posture of Claude Code must be understood, because all analysis tools build on top of it.

### 2.1 What Data Leaves the Machine

**Every interaction with Claude Code transmits data to Anthropic's API**:
- All user prompts (including pasted code snippets)
- All tool call inputs (file paths, search queries, bash commands)
- All tool call results (full file contents, command outputs, search results)
- All model responses (analysis, generated code, thinking)

This is encrypted in transit (TLS) but stored on Anthropic's servers subject to retention policies.

### 2.2 Data Retention by Plan Type

| Plan | Training on Data? | Retention Period | ZDR Available? |
|------|-------------------|------------------|----------------|
| **Free** | Yes (opt-out available) | 5 years (if training on) / 30 days (if not) | No |
| **Pro** | Yes (opt-out available) | 5 years (if training on) / 30 days (if not) | No |
| **Max** | Yes (opt-out available) | 5 years (if training on) / 30 days (if not) | No |
| **Team** | No (unless opted in) | 30 days | No |
| **Enterprise** | No (unless opted in) | 30 days | Yes |
| **API** | No (unless opted in) | 30 days | Yes (with approval) |
| **Bedrock/Vertex** | No | Per cloud provider policy | Per provider |

**Critical for consulting**: Consumer plans (Free/Pro/Max) are unsuitable for client work. At minimum, Team or Enterprise plans are required to ensure no training on client data. Enterprise with ZDR is ideal for sensitive engagements.

### 2.3 Local Data Storage

Claude Code stores session transcripts locally at `~/.claude/projects/<encoded-path>/<session-uuid>.jsonl`. These files persist indefinitely on disk (local caching configurable up to 30 days, but the JSONL files themselves are not automatically purged in practice). Additional local storage includes:
- `~/.claude/file-history/` — versioned file snapshots
- `~/.claude/history.jsonl` — global command history across all projects
- `~/.claude/debug/` — debug logs per session
- `~/.claude/.credentials.json` — authentication credentials

### 2.4 Telemetry and Third-Party Services

| Service | Data Sent | Default State | Opt-Out |
|---------|-----------|---------------|---------|
| **Statsig** | Operational metrics (latency, usage patterns). No code or file paths | ON (Claude API) / OFF (Bedrock/Vertex) | `DISABLE_TELEMETRY=1` |
| **Sentry** | Error logs | ON (Claude API) / OFF (Bedrock/Vertex) | `DISABLE_ERROR_REPORTING=1` |
| **/feedback** | **Full conversation history including code** | ON | `DISABLE_FEEDBACK_COMMAND=1` |

**Critical warning**: The `/feedback` command sends the **entire conversation transcript** including all source code to Anthropic with a **5-year retention period**. Engineers must be trained to never use `/feedback` on client engagements, or the command should be disabled firm-wide.

### 2.5 Anthropic's Compliance Posture

Anthropic maintains:
- **SOC 2 Type II** attestation
- **ISO 27001:2022** certification
- **ISO/IEC 42001:2023** (AI management systems)
- **HIPAA** configurable options
- **GDPR** Standard Contractual Clauses (SCCs) for EU data transfers
- Data Processing Agreements (DPAs) available for Team/Enterprise/API

Resources available at [Anthropic Trust Center](https://trust.anthropic.com).

### 2.6 NDA Implications

Web search findings indicate that code under strict NDA or confidentiality agreements "typically prohibits sharing with third-party AI services, making Claude Code usage a potential contract violation." However, the legal landscape is evolving:
- AI clauses are expected to become standard in NDAs within 12-18 months
- The trend is moving from blanket AI prohibitions to nuanced frameworks allowing "secure, enterprise-grade AI tools or private closed environment AI systems"
- Enterprise deployments with no-training guarantees and ZDR are increasingly accepted under updated NDA frameworks

**Recommendation**: Review all active client NDAs for AI tool restrictions. Where NDAs are silent on AI, seek explicit client approval. Where NDAs prohibit third-party AI, either negotiate amendments or use Bedrock/Vertex deployments (which keep data within the client's cloud provider).

---

## 3. Per-Tool Privacy Posture

### 3.1 claude-history (Rust TUI)

| Dimension | Assessment |
|-----------|------------|
| **Data leaves machine** | None |
| **Data stored** | Binary cache of parsed JSONL (local only) |
| **Network connections** | None |
| **Open source** | Yes (MIT) |
| **Secret handling** | No redaction — displays raw session content |
| **Client isolation** | Sessions are already separated by project path in `~/.claude/projects/` |
| **Compliance risk** | Minimal — equivalent to reading local log files |

**Verdict**: Safe for individual use. No additional risk beyond Claude Code itself.

### 3.2 Claude DevTools (Electron desktop app)

| Dimension | Assessment |
|-----------|------------|
| **Data leaves machine** | None |
| **Data stored** | In-memory only during analysis |
| **Network connections** | None (read-only local file access) |
| **Open source** | Yes (MIT) |
| **Secret handling** | No redaction — displays raw session content including tool results |
| **Client isolation** | Can open specific session files; user controls which sessions to analyze |
| **Compliance risk** | Minimal — equivalent to reading local log files |

**Verdict**: Safe for individual use. The 7-category token attribution and compaction analysis add no privacy risk.

### 3.3 claude-replay (JavaScript, HTML generator)

| Dimension | Assessment |
|-----------|------------|
| **Data leaves machine** | Only if the generated HTML is shared externally |
| **Data stored** | Generated HTML files contain compressed session data; autosave to `~/.claude-replay/autosave/` |
| **Network connections** | Editor server runs on 127.0.0.1 only |
| **Open source** | Yes (MIT) |
| **Secret handling** | **12 regex pattern categories** for automatic redaction + custom `--redact` rules. Recursive object walking. Issue #1 (PII in compressed blobs) was identified and fixed — redaction now occurs before compression |
| **Client isolation** | Per-session export; user chooses which sessions to replay |
| **Compliance risk** | Low locally; **medium-high if HTML outputs are shared** without proper redaction |

**Secret redaction patterns** (12 categories):
- API keys, tokens, passwords, private keys, AWS credentials, database URLs, OAuth secrets, JWT tokens, and similar credential patterns

**Consulting-specific risk**: The primary use case (sharing replays) is exactly where consulting risk increases. An engineer creating a replay of a Client A session to share with colleagues could inadvertently expose Client A's source code, credentials, or architecture. The automatic redaction helps with credentials but does NOT redact business logic, architecture details, or source code — only secrets matching regex patterns.

**Verdict**: Safe for individual use with configuration. Requires security review and policy before any sharing of generated HTML files. Recommend: establish a firm policy that claude-replay HTML outputs of client sessions are never shared externally, and internal sharing requires redaction review.

### 3.4 Mantra (Closed-source desktop app)

| Dimension | Assessment |
|-----------|------------|
| **Data leaves machine** | Telemetry (anonymous usage stats with device ID) sent by default; optional paid Sync ($4/mo) and Publish ($8/mo) features upload session data |
| **Data stored** | Session data read from local JSONL; replay sandbox in `{app_data_dir}/replay/{session_id}/` |
| **Network connections** | Telemetry endpoint (default on); Sync/Publish servers (opt-in) |
| **Open source** | **No** — closed source, binary-only distribution |
| **Secret handling** | Rust-based local scanner for API keys, passwords, tokens, private keys; one-click content redaction before sharing |
| **Client isolation** | Per-project session browsing; user controls which sessions to view |
| **Compliance risk** | **Medium** — closed source means privacy claims cannot be independently verified; default-on telemetry with device IDs contradicts "privacy-first" marketing; v0.11.1 added device ID correlation |

**Consulting-specific concerns**:
1. **Closed source is a compliance blocker** for many enterprises. SOC 2 and ISO 27001 audits require documented understanding of data flows — a closed-source tool with undisclosed telemetry creates an audit gap.
2. **Device ID telemetry** — even if anonymous, correlating device IDs with usage patterns over time could theoretically identify which clients a developer works on and when.
3. **Sync and Publish features** — if enabled, session data leaves the device to Mantra's servers. No DPA, no compliance certifications, solo developer operation.
4. **Cannot verify** that the "sensitive data detection" actually catches all credential patterns without source code audit.

**Verdict**: Requires security review. Not recommended for client work without source code audit (which is impossible given closed-source nature). If used, disable telemetry, never enable Sync/Publish on client projects.

### 3.5 Rudel (SaaS analytics platform)

| Dimension | Assessment |
|-----------|------------|
| **Data leaves machine** | **Yes — complete session transcripts uploaded** to remote ClickHouse instance |
| **Data stored** | Full transcripts in ClickHouse (365-day TTL); auth data in Postgres; git metadata (remote URL, branch, SHA) |
| **Network connections** | HTTP POST to Rudel API (Fly.io) on every session completion; auth flows |
| **Open source** | Yes (MIT) |
| **Secret handling** | **No scrubbing or redaction mentioned** in documentation or source code. Transcripts uploaded verbatim |
| **Client isolation** | Organization-scoped dashboards, but all sessions within an org are aggregated. No per-client segmentation |
| **Compliance risk** | **Critical** — uploads complete source code, credentials, business logic, and architecture details to a third-party server |

**What gets uploaded** (from Rudel research): The README explicitly warns that "uploaded transcripts and related metadata may contain sensitive material, including source code, prompts, tool output, file contents, command output, URLs, and secrets that appeared during a session."

**Consulting-specific risks**:
1. **NDA violation** — uploading client source code to a third-party analytics platform almost certainly violates standard client NDAs
2. **No secret redaction** — API keys, credentials, and tokens in session transcripts are uploaded verbatim
3. **No client isolation** — all developers' sessions in an organization are aggregated. There is no mechanism to separate Client A sessions from Client B sessions, or to restrict which sessions are uploaded
4. **Third-party data risk** — ObsessionDB (the company behind Rudel) is an early-stage startup. Data custody, breach notification, and business continuity are undefined
5. **GDPR exposure** — developer PII (names, emails from git commits) in session transcripts is transmitted to servers without documented DPA
6. **SOC 2 audit failure** — using the hosted version would likely fail vendor management controls in a SOC 2 audit without extensive documentation of data flows and risk acceptance

**Self-hosting mitigation**: Self-hosting Rudel (ClickHouse + Postgres + Bun app server) keeps data within the firm's infrastructure but requires significant operational investment (three services, schema migrations with beta tooling). Even self-hosted, the lack of client isolation and secret redaction remain concerns.

**Verdict**: **Not recommended for client work** in hosted mode. Self-hosted requires extensive security review and custom development (client isolation, secret redaction) before it could be considered.

### 3.6 ccusage and Other Cost-Tracking Tools

| Dimension | Assessment |
|-----------|------------|
| **Data leaves machine** | None |
| **Data stored** | Aggregated token counts (no content) |
| **Network connections** | None |
| **Secret handling** | N/A — reads only usage/token fields, not content |
| **Client isolation** | Can filter by project path |
| **Compliance risk** | Minimal |

**Verdict**: Safe for individual use. Token counts and cost data contain no client IP.

### 3.7 OTel-based Tools (claude-code-otel, claudia)

| Dimension | Assessment |
|-----------|------------|
| **Data leaves machine** | Depends on OTel backend configuration — can be local (Prometheus/Grafana) or remote |
| **Data stored** | Trace spans, metrics; may include tool names and durations but typically not full content |
| **Network connections** | To configured OTel collector endpoint |
| **Open source** | Yes |
| **Secret handling** | OTel spans typically contain metadata, not full content — but custom instrumentation could expose content |
| **Client isolation** | Can be configured with labels/attributes per project |
| **Compliance risk** | Low if backend is self-hosted; medium if using SaaS OTel backends (Datadog, Honeycomb, etc.) |

**Verdict**: Safe with configuration. Use self-hosted OTel backends (Prometheus + Grafana on firm infrastructure) rather than SaaS backends for client work.

---

## 4. Client Isolation Analysis

### 4.1 How Claude Code Separates Projects

Claude Code stores sessions under `~/.claude/projects/<encoded-cwd>/`. The path encoding replaces `/` with `-`, so `/home/colin/Repos/client-a` becomes `-home-colin-Repos-client-a`. This provides **filesystem-level separation** between clients — sessions from different project directories are in different subdirectories.

However, this separation has limitations:
- **Global history**: `~/.claude/history.jsonl` contains every user input across ALL projects with project path and session ID. An analysis tool reading this file sees inputs from all clients.
- **No access control**: All session files are readable by the same user. Any tool running as the developer can read all clients' sessions.
- **Cross-project references**: If a developer reads a file from Client A's codebase while working in Client B's directory (via absolute path), Client A's code appears in Client B's session.

### 4.2 Per-Tool Client Isolation Capabilities

| Tool | Can Filter by Project? | Can Exclude Projects? | Automatic Isolation? |
|------|----------------------|----------------------|---------------------|
| **claude-history** | Yes (searches within project dirs) | No built-in exclusion | No |
| **Claude DevTools** | Yes (opens specific session files) | Manual (don't open other sessions) | No |
| **claude-replay** | Yes (per-session export) | Manual (choose which sessions) | No |
| **Mantra** | Yes (per-project browsing) | No built-in exclusion | No |
| **Rudel** | No — uploads all sessions from the org | No per-client filtering | No |
| **ccusage** | Yes (can filter by path) | Manual | No |

**No tool provides automatic client isolation**. Every tool requires the developer to manually select which sessions to analyze. There is no mechanism to tag sessions as "Client A" or "Client B" and enforce access boundaries.

### 4.3 Recommended Isolation Strategies

1. **Separate user accounts per client**: The strongest isolation — each client engagement uses a different OS user account. Session files are physically separated by user home directory. Operationally heavy.

2. **Separate machines or VMs per client**: Even stronger isolation. Practical with cloud development environments (Codespaces, devcontainers). Claude Code supports devcontainers for additional isolation.

3. **Directory discipline + tool policy**: Ensure each client's code lives in a clearly separated directory tree. Establish policy that analysis tools are only run on the current client's session directory. Weakest option — relies on developer discipline.

4. **Periodic session cleanup**: Delete session files from `~/.claude/projects/` after engagement completion. Reduces accumulation of cross-client data on a single machine.

---

## 5. Compliance Framework Mapping

### 5.1 SOC 2 Type II

SOC 2 Trust Service Criteria relevant to Claude Code analysis tools:

| Criterion | Requirement | Claude Code Baseline | Analysis Tool Impact |
|-----------|------------|---------------------|---------------------|
| **CC6.1** — Logical access | Restrict access to authorized users | API key auth, local file permissions | Rudel: org-level access only. Others: local OS permissions |
| **CC6.3** — Data classification | Classify data by sensitivity | No built-in classification | No tool classifies client data within sessions |
| **CC6.7** — Data transmission | Encrypt data in transit | TLS to Anthropic API | Rudel: TLS to API. Others: no transmission |
| **CC7.1** — Monitoring | Detect anomalies | OTel metrics available | OTel tools provide monitoring. Rudel provides analytics |
| **CC7.2** — Incident response | Respond to security events | Anthropic handles API-side | Rudel: no documented incident response. Others: N/A |
| **CC8.1** — Change management | Control changes to systems | Claude Code versioned via npm | All tools under active development; breaking changes possible |

**SOC 2 vendor management**: Using any external tool requires documenting it in the vendor register. For Rudel (hosted), this means completing a vendor risk assessment — which would likely flag: early-stage company, no SOC 2 of their own, full source code exposure, no DPA. For local-only tools, vendor management is simpler — the tool is software, not a service.

**Audit trail requirements**: SOC 2 requires audit trails for access to sensitive data. Claude Code's session files are not access-logged (standard filesystem). If an auditor asks "who accessed Client A's session data and when?", there's no answer without OS-level file access auditing.

### 5.2 ISO 27001

| Control | Requirement | Assessment |
|---------|------------|------------|
| **A.5.12** — Classification of information | Classify and label information | Claude Code sessions are unclassified by default. No tool supports classification labels |
| **A.5.13** — Labeling of information | Apply labels per classification | No session labeling capability in any tool |
| **A.5.34** — Privacy and PII protection | Protect personally identifiable information | Sessions contain developer PII (git names/emails); only claude-replay and Mantra offer redaction |
| **A.8.10** — Information deletion | Delete information when no longer needed | Local sessions persist indefinitely; no automated retention policy enforcement |
| **A.8.11** — Data masking | Mask data per policy | Only claude-replay (12 regex patterns) and Mantra (closed-source scanner) provide masking |
| **A.8.12** — Data leakage prevention | Prevent unauthorized data disclosure | No tool prevents leakage. Rudel actively facilitates data egress by design |
| **A.5.19-A.5.23** — Supplier management | Assess and monitor suppliers | Anthropic has SOC 2/ISO 27001. Rudel/ObsessionDB has neither. Mantra has neither |

### 5.3 GDPR (for EU client engagements)

| GDPR Principle | Requirement | Assessment |
|----------------|------------|------------|
| **Lawful basis (Art. 6)** | Legal basis for processing | Legitimate interest (developer tooling) or contractual necessity. Must be documented |
| **Data minimization (Art. 5.1c)** | Process only necessary data | Claude Code captures ALL file contents read — far beyond minimum necessary. Analysis tools inherit this over-collection |
| **Storage limitation (Art. 5.1e)** | Don't store longer than needed | Consumer plans: up to 5 years. Enterprise: 30 days or ZDR. Local sessions: indefinite |
| **Data processing agreements (Art. 28)** | DPA with processors | Anthropic provides DPAs for Team/Enterprise/API. Rudel: no DPA. Mantra: no DPA |
| **Data transfer (Art. 46)** | Safeguards for international transfer | Anthropic uses SCCs. Rudel: ClickHouse on Fly.io (US) — no transfer mechanism documented. Mantra: telemetry destination unknown |
| **Right to erasure (Art. 17)** | Delete personal data on request | Local sessions can be deleted. Anthropic: web sessions deletable. Rudel: no documented deletion mechanism for uploaded transcripts |
| **DPIA (Art. 35)** | Impact assessment for high-risk processing | AI-assisted code analysis likely qualifies as "new technology" requiring DPIA under Art. 35(1) |

**GDPR bottom line**: Using Claude Code on EU client projects requires at minimum a Team/Enterprise plan (for DPA), documentation of lawful basis, and a Data Protection Impact Assessment. Any tool that uploads session data (Rudel, Mantra Sync/Publish) without a DPA is a GDPR violation.

---

## 6. Secret Exposure Risk Analysis

### 6.1 How Secrets Enter Sessions

Claude Code sessions accumulate secrets through normal development workflows:

1. **`Read` tool on config files**: `.env`, `config.yml`, `application.properties`, `settings.json` — Claude reads these to understand the application, capturing credentials verbatim
2. **`Bash` tool output**: Running tests, builds, or deployment commands may print credentials in output, stack traces, or error messages
3. **`Read` tool on source code**: Hardcoded credentials in source files (common in early-stage projects)
4. **Git diff output**: Credentials in committed changes appear in diff output
5. **Error messages**: Database connection failures, API auth errors often include connection strings or tokens

### 6.2 Per-Tool Secret Handling

| Tool | Detects Secrets? | Redacts Before Display? | Redacts Before Export/Upload? | Custom Patterns? |
|------|-----------------|------------------------|------------------------------|-----------------|
| **claude-replay** | Yes (12 regex categories) | Yes | Yes (before compression) | Yes (`--redact` flag) |
| **Mantra** | Yes (Rust scanner) | Yes | Yes (before sharing) | Unknown (closed source) |
| **Claude DevTools** | No | No | N/A (no export) | No |
| **claude-history** | No | No | N/A (no export) | No |
| **Rudel** | **No** | **No** | **No — uploads verbatim** | **No** |
| **ccusage** | N/A | N/A | N/A | N/A |

**Highest risk**: Rudel uploads sessions containing secrets to a remote server with no redaction. This is the most dangerous scenario — credentials for client systems could be stored in a third-party ClickHouse database.

**Partial risk**: Claude-replay and Mantra detect common credential patterns, but regex-based detection has known limitations:
- Custom credential formats may not match patterns
- Credentials in non-standard formats (base64-encoded, split across lines) are missed
- Business-specific secrets (internal URLs, project codenames) are not detected
- Source code and business logic are never redacted — only credential-shaped strings

### 6.3 Mitigation Recommendations

1. **Pre-session hygiene**: Use `.claudeignore` files to prevent Claude Code from reading sensitive config files. Prefer environment variable references over file-based secrets.
2. **Post-session cleanup**: Periodically scan `~/.claude/projects/` for credential patterns and delete affected session files.
3. **Tool policy**: Only use tools with secret redaction (claude-replay) for any session export/sharing workflow. Never use Rudel on projects with credentials in session history.
4. **Credential rotation**: Treat any credential that appeared in a Claude Code session as potentially exposed. Include AI tool sessions in credential rotation schedules.

---

## 7. Recommendation Matrix

### For Highspring Digital (Multi-Client Consulting Firm)

| Tool | Recommendation | Conditions |
|------|---------------|------------|
| **claude-history** | **Safe for individual use** | No additional controls needed beyond Claude Code baseline |
| **Claude DevTools** | **Safe for individual use** | No additional controls needed |
| **ccusage / cost tools** | **Safe for individual use** | Useful for per-client cost attribution if projects are in separate directories |
| **claude-replay** | **Safe with configuration** | Enable secret redaction (`--redact`). Establish policy: generated HTML for client sessions never shared externally. Internal sharing requires redaction review. Disable editor autosave on client machines |
| **OTel tools (self-hosted)** | **Safe with configuration** | Use self-hosted backends only (Prometheus + Grafana). Do not send traces to SaaS OTel providers. Configure trace sampling to exclude content payloads |
| **Mantra** | **Requires security review** | Closed source blocks compliance audit. Disable telemetry if used. Never enable Sync/Publish. Cannot recommend for client work without source audit |
| **Rudel (hosted)** | **Not recommended for client work** | Uploads complete transcripts including source code and credentials to third-party server. No redaction, no client isolation, no DPA, no SOC 2. Incompatible with client NDAs |
| **Rudel (self-hosted)** | **Requires security review** | Data stays on firm infrastructure, but: no secret redaction, no client isolation, operational burden of 3 services. Would need custom development before client-safe |

### Decision Framework

```
Does the tool transmit session data off the machine?
├── NO → Low risk. Review: is it open source?
│   ├── YES → Safe for individual use (claude-history, DevTools, ccusage)
│   └── NO → Requires security review (Mantra core features)
└── YES → High risk. Review: where does data go?
    ├── ANTHROPIC (Claude Code itself) → Acceptable with Enterprise/Team plan + DPA
    ├── SELF-HOSTED infrastructure → Acceptable after security review
    └── THIRD-PARTY SaaS → Not recommended for client work (Rudel hosted, Mantra Sync)
```

### Pre-Adoption Checklist for Consulting Firms

1. **Plan selection**: Ensure Claude Code is on Team or Enterprise plan (no training on data, 30-day retention, DPA available). Enterprise with ZDR for highly sensitive engagements.
2. **NDA review**: Audit all active client NDAs for AI tool restrictions. Seek amendments or explicit approval where needed.
3. **Environment variables**: Set firm-wide via managed settings:
   - `DISABLE_FEEDBACK_COMMAND=1` (prevents accidental 5-year transcript upload)
   - `DISABLE_TELEMETRY=1` (optional, reduces data surface)
   - `DISABLE_ERROR_REPORTING=1` (optional)
4. **Directory discipline**: Enforce separate directory trees per client. Document in engineer onboarding.
5. **Session cleanup policy**: Define retention periods for local session files. Automate deletion after engagement completion.
6. **Tool allowlist**: Publish an approved list of analysis tools with usage guidelines. Block Rudel (hosted) and Mantra Sync/Publish.
7. **Secret scanning**: Implement pre-commit hooks and `.claudeignore` files to minimize secrets entering sessions.
8. **GDPR assessment**: If handling EU client data, complete a DPIA for Claude Code usage and ensure DPA is in place with Anthropic.
9. **SOC 2 documentation**: Add Claude Code and any analysis tools to the vendor register. Document data flows for auditors.
10. **Training**: Educate engineers on what data appears in sessions, why `/feedback` is dangerous on client projects, and how to use redaction in claude-replay.

---

## Depth Checklist

- [x] Underlying mechanism explained — data flows from Claude Code through analysis tools, what data is in sessions, how each tool processes it
- [x] Key tradeoffs identified — privacy vs. team analytics, local vs. SaaS, open vs. closed source, redaction coverage gaps
- [x] Compared alternatives — 7 tools/categories compared across privacy dimensions, compliance frameworks
- [x] Failure modes and edge cases — secret leakage paths, cross-client contamination via global history, NDA violation scenarios, GDPR violations
- [x] Concrete examples — specific credential exposure scenarios, NDA clause evolution, Rudel's own README warning about sensitive data
- [x] Standalone-readable — sufficient for compliance decisions without consulting individual tool reports

## Sources

- `../claude-code-analysis-tools/comparison-research.md` — Privacy spectrum, per-tool comparison
- `../claude-code-analysis-tools/rudel-research.md` — Full transcript upload details, privacy warnings
- `../claude-code-analysis-tools/mantra-research.md` — Telemetry, closed-source concerns, device ID tracking
- `../claude-code-analysis-tools/session-data-format-research.md` — JSONL format, what data is stored
- `../claude-code-analysis-tools/claude-replay-research.md` — Secret redaction patterns, sharing risks
- `docs/claude-code-security-docs.md` — Official Claude Code security documentation
- `docs/claude-code-data-usage-docs.md` — Official data retention and training policies
- `docs/web-search-ai-compliance-consulting.md` — Web search results on NDA clauses, GDPR, SOC 2, consulting firm AI governance
