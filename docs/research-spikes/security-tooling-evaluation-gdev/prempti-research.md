# Prempti (falcosecurity/prempti) — Deep Dive Evaluation for gdev

## Executive Summary

Prempti is a **Falco-powered policy and visibility layer for AI coding agents**, built in Rust by the Falco Security project (a CNCF graduated project). It intercepts every Claude Code tool call before execution, evaluates it against customizable YAML rules via Falco's rule engine, and returns allow/deny/ask verdicts. It operates entirely in user space with no root, kernel modules, or containers. The project is experimental (v0.3.0, created March 2026), written by a small team (2 active contributors), and explicitly positions itself as a cooperative policy layer that complements — but does not replace — sandboxes and system hardening.

**Recommendation for gdev**: Prempti should be integrated as an **optional configuration** (user enables it via `gdev enable prempti`), not as a default. Its rule library and architectural patterns are also a valuable **concept/implementation source** for gdev's own PreToolUse hook rules. It should not be a default because it adds significant infrastructure weight (Falco binary + plugin + supervisor daemon), overlaps substantially with gdev's existing 6-layer defense architecture, and is too immature (42 stars, 2 months old) for an always-on default in a consulting tool.

---

## 1. What Problem Does Prempti Solve?

AI coding agents (Claude Code, Codex, Gemini CLI) operate within user sessions with access to credentials, SSH keys, cloud configs, and system files. Developers lack structured visibility into what these agents actually do at runtime. Documented cases include agents reading files outside project scope, exfiltrating environment variables, and making unauthorized network calls.

Prempti addresses this by providing:
1. **Pre-execution interception** — every tool call is evaluated before it runs
2. **Policy enforcement** — rules can block (deny), require confirmation (ask), or allow actions
3. **Audit trail** — every tool call is logged with full context for forensic analysis
4. **LLM-friendly feedback** — blocked actions return explanations the agent can adapt to

## 2. Architecture and Mechanisms

### 2.1 Pipeline Flow

```
Claude Code PreToolUse hook
    |
    v
Interceptor (Rust binary, hooks/claude-code/)
    | Unix domain socket
    v
Falco Plugin Broker (embedded in plugin)
    | source plugin API (next_batch)
    v
Falco Rule Engine (nodriver mode, no kernel)
    | http_output (localhost)
    v
Plugin HTTP Server -> Verdict Resolution
    | Unix domain socket response
    v
Interceptor -> hook response to Claude Code
```

### 2.2 Components

| Component | Language | Role |
|-----------|----------|------|
| **Interceptor** (`hooks/claude-code/`) | Rust | Thin passthrough: reads hook JSON from stdin, sends to broker via Unix socket, maps verdict to stdout. Optimized for size (`opt-level = "z"`). |
| **Plugin** (`plugins/coding-agents-plugin/`) | Rust (falco_plugin SDK) | Falco source+extract plugin with embedded broker. Parses events, feeds Falco, receives alerts, resolves verdicts. Optimized for speed (`opt-level = 2`). |
| **Supervisor** (`tools/premptictl/src/daemon/`) | Rust | Spawns Falco, captures/rotates logs, owns hook lifecycle, exposes control socket. |
| **Rules** (`rules/`) | YAML (Falco rule language) | 58 default rules + 79 macros covering 7 security domains. |
| **premptictl** | Rust | CLI for status, mode switching, log viewing, service management. |

### 2.3 Key Design Decisions

**Broker embedded in plugin**: Reduces moving parts — Falco is the only long-lived process. The plugin spawns threads for the Unix socket server (interceptor connections) and HTTP server (Falco alert feedback).

**Tag-based verdict resolution**: Verdicts encoded in rule `tags:` field, not output strings. `coding_agent_deny` blocks, `coding_agent_ask` prompts, absence of deny/ask = allow. Escalation: deny > ask > allow when multiple rules match.

**Catch-all "seen" rule**: A mandatory rule fires for every event, signaling evaluation completion. If no deny/ask alert arrived for a correlation ID, the broker resolves as allow. Requires `rule_matching: all` in Falco config (non-default).

**Fail-closed**: If Falco/plugin is unreachable, all tool calls are denied. During service restart, all Claude Code tool calls are blocked by design.

**LLM-friendly output**: Rule output fields start with "Falco" and contain natural-language explanations. Example: `"Falco blocked writing to /etc/passwd because it is a sensitive path"`.

### 2.4 Event Schema

One data source (`coding_agent`) with 16 fields covering:
- Correlation: `correlation.id` (monotonic u64)
- Agent identity: `agent.name`, `agent.os`, `agent.pid`, `agent.session_id`, `agent.permission_mode`
- Context: `agent.cwd`, `agent.real_cwd` (canonicalized), `agent.transcript_path`
- Tool: `tool.name`, `tool.input` (full JSON), `tool.input_command`, `tool.file_path`, `tool.real_file_path`

Raw/real path pairs enable both audit (raw) and policy matching (canonicalized, symlinks resolved).

### 2.5 Platform Support

| Platform | Falco Source | Service Management |
|----------|-------------|-------------------|
| Linux (x86_64, aarch64) | Pre-built binaries | systemd user unit |
| macOS (ARM64, x86_64) | Built from source (patched for http_output) | launchd user agent |
| Windows (x64, ARM64) | Built from source (patched for http_output + vcpkg SChannel) | Registry Run key + PowerShell launcher |

Installs under `~/.prempti/` with isolated Falco config (no `/etc/falco/` contamination).

## 3. Default Ruleset Analysis

The default ruleset contains **58 rules** and **79 macros** organized into 7 security domains:

| Domain | Deny Rules | Ask Rules | Coverage |
|--------|-----------|-----------|----------|
| **Working directory boundary** | 0 | 1 | File writes outside cwd require confirmation |
| **Sensitive paths** | 2 | 0 | Block read/write to ~/.ssh/, ~/.aws/, .env, /etc/ |
| **Sandbox disable** | 5 | 2 | Block/confirm attempts to disable agent sandbox (Claude, Codex, Gemini) |
| **Threats** | 12 | 3 | Credentials, destructive commands, pipe-to-shell, encoded payloads, exfiltration, IMDS, reverse shells, supply chain installs |
| **MCP/skill content** | 5 | 5 | MCP poisoning, skill injection, untrusted host installs, command encoding |
| **Persistence vectors** | 0 | 12 | Hook injection, git hooks, registry redirects, API base URL overrides, API key leaks |
| **Self-protection** | 5 | 1 | Block premptictl invocation, service stop, writes to ~/.prempti/, settings.json, policy-limits.json |

### Notable Overlap with gdev's Existing Defenses

gdev's 6-layer defense-in-depth already covers many of the same concerns:
- **Layer 1 (age-gating)** — Prempti has no equivalent; gdev is stronger here
- **Layer 2 (install script blocking)** — Prempti does not address this; gdev handles via package manager configs
- **Layer 3 (lock file enforcement)** — Prempti does not address this; gdev handles via package manager configs
- **Layer 4 (vulnerability scanning)** — Prempti does not address this; gdev integrates OSV Scanner
- **Layer 5 (PreToolUse hooks)** — **Direct overlap**. Prempti's core function is PreToolUse interception. gdev already deploys custom hook scripts + attach-guard
- **Layer 6 (hardened Nix settings)** — Prempti does not address Nix; gdev handles via nix.conf generation

Prempti adds capabilities gdev does **not** currently have:
- **Audit trail** — every tool call logged with full context (correlation ID, session, PID, paths)
- **Ask verdict** — interactive confirmation for gray-area operations (gdev hooks only deny or allow)
- **Monitor mode** — observe-only mode for rule tuning before enforcement
- **Cross-agent protection** — blocks one agent from reading another's credentials (Gemini/Codex/Cursor)
- **Self-protection** — blocks agent from disabling the security tool itself
- **MCP/skill content inspection** — detects poisoned MCP configs and malicious skill files
- **Working directory boundary** — asks before file operations outside project dir

## 4. Maturity and Maintenance Assessment

| Metric | Value | Assessment |
|--------|-------|------------|
| Created | 2026-03-18 | 2 months old |
| Stars | 42 | Very early adoption |
| Forks | 10 | Minimal community |
| Contributors | 4 (2 active: leogr 117 commits, c2ndev 33) | Small team |
| Releases | 5 (v0.1.0 through v0.3.0) | Rapid iteration |
| License | Apache-2.0 | Permissive, compatible |
| Language | Rust | Good for security tooling but niche build dependency |
| Commit cadence | Daily (as of 2026-05-15) | Actively maintained |
| Backing organization | Falco Security / Sysdig (CNCF graduated) | Strong institutional backing |
| Self-description | "Experimental" | Authors acknowledge immaturity |

**Assessment**: The project benefits from serious institutional backing (CNCF/Sysdig) and engineering quality (deep CLAUDE.md, comprehensive E2E tests, cross-platform CI). However, it is explicitly experimental, has minimal community adoption, and depends on Falco 0.43 which requires source compilation on macOS and Windows. The rapid release cadence (5 releases in 2 months) suggests the API and rule format are still stabilizing.

## 5. Integration Fit with gdev

### 5.1 As a Configuration Option (Recommended)

**Fit**: Good. Prempti aligns with gdev's "every tool is individually toggleable" principle (Design Principle 13) and "curate don't reinvent" (Principle 4).

**Implementation**:
- `gdev enable prempti` would:
  1. Check if Prempti is installed (detect `~/.prempti/bin/premptictl`)
  2. If not installed, guide user to installer or offer to download
  3. Generate custom Falco rules in `~/.prempti/rules/user/gdev-rules.yaml` that complement gdev's existing defenses
  4. Register Prempti as a known tool in gdev's lifecycle management
  5. Document interaction with existing gdev PreToolUse hooks in CLAUDE.md

**Value-add over bare install**: gdev would provide curated rules tuned to the project's detected ecosystems, avoid rule duplication with existing gdev hooks, and handle the lifecycle (enable/disable/status).

**Concerns**:
- **Dual hook overhead**: Both gdev's PreToolUse hooks AND Prempti's interceptor would fire for every tool call. Need to verify performance impact and avoid conflicting deny/allow decisions.
- **Installation complexity**: Prempti requires Falco binary + plugin + supervisor. On NixOS specifically, Falco packaging may need a Nix expression (no official Nix package exists).
- **Fail-closed risk**: Prempti denies ALL tool calls if service is down. If a user enables it and the service crashes, Claude Code becomes completely non-functional until `premptictl start` or `gdev repair`.

### 5.2 As a Default (Not Recommended)

**Why not**:
1. **Infrastructure weight**: Adds a persistent daemon (Falco + supervisor), Unix sockets, log rotation — substantial for a tool that targets "single binary, zero prerequisites"
2. **Overlap**: 70%+ of Prempti's default rules duplicate protections gdev already provides through simpler mechanisms (deny rules in settings.json, PreToolUse hook scripts)
3. **Maturity**: Too new (42 stars, 2 months, self-described "experimental") for an always-on default in consulting environments
4. **Platform friction**: Falco requires source compilation on macOS/Windows. NixOS has no official Falco package
5. **Fail-closed failure mode**: Service outage = total Claude Code paralysis, unacceptable as a default
6. **Dependency on external binary**: Violates gdev's "single binary, zero prerequisites" principle

### 5.3 As Concept/Implementation Source (Highly Recommended)

Prempti's design contains several patterns gdev should borrow:

1. **Rule categories and coverage map**: Prempti's 7-domain taxonomy (boundary, sensitive paths, sandbox, threats, MCP/skill, persistence, self-protection) is an excellent checklist for gdev's own deny rules and hook scripts. Several categories gdev does not currently cover:
   - **Self-protection rules** (blocking agent from disabling security tooling)
   - **Cross-agent credential isolation** (blocking Gemini from reading Claude's auth)
   - **MCP config poisoning** (detecting malicious `command` or `url` in `.mcp.json`)
   - **Skill/command file injection** (detecting pipe-to-shell or IOC domains in skill files)
   - **Agent settings file protection** (blocking writes to `~/.claude/settings.json`)

2. **Ask verdict pattern**: gdev's current hooks only deny or allow. The "ask" pattern (require interactive confirmation for gray-area operations) is valuable for operations like writing outside cwd, modifying shell startup files, or editing git hooks. Claude Code's hook response format already supports `"permissionDecision": "ask"`.

3. **LLM-friendly output convention**: Rule output starting with "Falco" and containing natural-language explanation is a good pattern. gdev's deny rules should follow a similar convention for agent-readable block reasons.

4. **Canonical path resolution**: Prempti's raw/real path pairs (symlink-resolved vs. as-reported) prevent path traversal bypasses. gdev's hook scripts should canonicalize paths before matching.

5. **Monitor mode for rule development**: The ability to observe rule matches without enforcement is useful for tuning. gdev could implement this as `gdev hooks --monitor` that logs what would be blocked without actually blocking.

6. **Audit trail**: The comprehensive event logging (correlation IDs, session tracking, full tool input) is a capability gdev does not currently provide. Worth considering for `gdev evidence` compliance reporting.

## 6. Tradeoffs, Limitations, and Failure Modes

### 6.1 Fundamental Limitations

- **Not a sandbox**: Prempti sees declared tool calls, not runtime side effects. An agent that runs `gcc main.c -o main && ./main` — Prempti sees the commands but not what the compiled binary does internally.
- **Cooperative layer**: The agent receives feedback and adapts, but a determined agent could theoretically circumvent hook mechanisms (though Claude Code's hook system makes this difficult).
- **No content inspection of compiled/binary artifacts**: Only inspects the command strings and file paths, not the content of generated executables.
- **String matching only**: Rules use string containment/regex, not semantic analysis. An obfuscated command could bypass pattern matching (though encoding rules catch base64/hex).

### 6.2 Operational Risks

- **Fail-closed can be fail-deadly**: If the Falco service crashes, ALL Claude Code tool calls are denied until manual recovery. For a developer in the middle of work, this is disruptive.
- **Single-threaded output worker**: Falco's `http_output` worker thread is shared across all output channels. A slow HTTP handler blocks everything. The plugin mitigates this by keeping the handler fast, but it's a systemic risk.
- **Service restart window**: During `premptictl mode` or `premptictl restart`, all tool calls are blocked (by design, for fail-closed). This creates brief outages during config changes.
- **Log rotation is supervisor-owned**: If the supervisor dies, logs are not rotated and could grow unbounded.

### 6.3 NixOS-Specific Concerns

- Falco has no official Nix package in nixpkgs
- The Linux installer (`install.sh`) assumes a traditional FHS layout, may not work cleanly on NixOS
- The systemd user unit would need NixOS-specific configuration (home-manager module)
- Building Falco from source on NixOS requires CMake, which works but adds build complexity

### 6.4 Performance Considerations

- Every tool call spawns the interceptor binary (Rust, optimized for size, fast startup)
- Round-trip: stdin read -> Unix socket -> Falco rule evaluation -> HTTP alert -> socket response -> stdout
- Default timeout: 5000ms per tool call
- In practice, latency should be sub-100ms for local Unix socket communication, but adds up across hundreds of tool calls in a session

## 7. Comparison to Alternatives

### 7.1 vs. gdev's Native PreToolUse Hooks

| Aspect | gdev Native Hooks | Prempti |
|--------|-------------------|---------|
| **Architecture** | Shell/Python scripts in settings.json | Falco rule engine + Rust interceptor + daemon |
| **Rule format** | JSON hook config + script logic | YAML Falco rules |
| **Verdict types** | deny, allow | deny, ask, allow |
| **Infrastructure** | Zero (scripts execute directly) | Falco binary + plugin + supervisor daemon |
| **Performance** | Script startup per call | Binary startup + IPC per call |
| **Audit trail** | None built-in | Comprehensive (correlation IDs, session tracking) |
| **Monitor mode** | Not available | Built-in |
| **Rule count** | ~48 deny rules (gdev reference) | 58 rules + 79 macros |
| **Maintenance** | gdev team maintains | Falco community maintains |
| **NixOS fit** | Excellent (scripts, no external deps) | Poor (no Nix package, FHS assumptions) |

### 7.2 vs. rulebricks/claude-code-guardrails

| Aspect | Prempti | Rulebricks Guardrails |
|--------|---------|----------------------|
| **Architecture** | Local Falco engine | Cloud API (Rulebricks SaaS) |
| **Dependencies** | Falco binary (local) | External API (network) |
| **Offline** | Yes | No |
| **Rule management** | YAML files (local) | Web UI (SaaS) |
| **Team sync** | Git-managed rule files | Instant via API |
| **Audit** | Local logs | Cloud audit trail |
| **Pricing** | Free (Apache-2.0) | SaaS pricing |
| **Stars** | 42 | 67 |

### 7.3 vs. attach-guard (already in gdev plan)

attach-guard is already planned for gdev Phase 4 integration as a reference PreToolUse hook plugin. It focuses specifically on package install guardrails (OSV.dev + age checking), while Prempti covers a much broader surface (file operations, MCP, persistence, etc.). They are complementary, not competing.

## 8. Recommendations

### Immediate Actions

1. **Add Prempti as an optional tool in Phase 12** (Tool Lifecycle & Integration): `gdev enable prempti` / `gdev disable prempti`. Low priority — below existing Phase 12 tools.

2. **Borrow rule patterns for Phase 4** (Claude Code Addon): Incorporate Prempti's rule categories that gdev does not currently cover:
   - Self-protection (block agent from disabling gdev hooks/settings)
   - MCP config poisoning detection
   - Skill/command file injection detection
   - Agent settings file protection
   - Working directory boundary enforcement (ask verdict)

3. **Implement "ask" verdict in gdev hooks**: Claude Code already supports `permissionDecision: "ask"` in hook responses. gdev should use this for gray-area operations rather than only deny/allow.

4. **Adopt canonical path resolution**: gdev's hook scripts should resolve symlinks and normalize paths before matching, following Prempti's raw/real pattern.

### Watch Items

- **Maturity**: Re-evaluate when Prempti reaches v1.0 or accumulates 200+ stars
- **Nix packaging**: Watch for a community Nix expression or official Falco Nix support
- **Codex integration**: When Prempti ships Codex support, evaluate whether gdev should support multi-agent security
- **Falco 0.44+**: Track whether macOS/Windows http_output patches are merged upstream (would simplify builds)

### Not Recommended

- Making Prempti a default: too heavy, too immature, too much overlap
- Depending on Prempti for any security guarantee: it's a cooperative layer, not a sandbox
- Forking Prempti's Rust code: different language (gdev is Go), different architecture

---

## Sources

| File | Content |
|------|---------|
| `docs/prempti-readme-raw.md` | Repository README summary |
| `docs/prempti-claude-md.md` | Full architecture document (CLAUDE.md from repo) |
| `docs/prempti-falco-blog-introducing.md` | Falco blog post announcing Prempti |
| `docs/prempti-sysdig-blog.md` | Sysdig blog post on Prempti |
| `docs/prempti-cargo-toml.md` | Cargo workspace configuration |
| `docs/prempti-default-rules-inventory.md` | Complete inventory of 58 rules and 79 macros |
| `docs/prempti-interceptor-architecture.md` | Interceptor source analysis |
| `docs/prempti-rules-readme.md` | Rule authoring conventions |
| `docs/prempti-github-metadata.md` | Repository metadata, releases, contributors |
| `docs/rulebricks-claude-code-guardrails.md` | Alternative: Rulebricks cloud-based guardrails |

## Depth Checklist

- [x] Underlying mechanism explained — full pipeline from hook to verdict, including broker, Falco engine, correlation, and fail-safety
- [x] Key tradeoffs and limitations identified — not a sandbox, fail-closed risks, NixOS friction, performance overhead
- [x] Compared to alternatives — vs. gdev native hooks, vs. Rulebricks, vs. attach-guard
- [x] Failure modes and edge cases described — service crash = total block, log rotation dependency, string-matching bypasses, restart windows
- [x] Concrete examples found — 58 default rules analyzed, interceptor architecture traced, blog demos reviewed
- [x] Report is standalone-readable — sufficient for integration decisions without consulting original sources
