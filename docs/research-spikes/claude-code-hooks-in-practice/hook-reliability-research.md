# Claude Code Hook Reliability, Performance, and Failure Modes

## Executive Summary

Claude Code hooks provide deterministic control over agent behavior, but the system has significant reliability gaps as of March 2026. The `command` handler type is the most reliable and widely tested, but even it suffers from platform-specific bugs (VS Code extension), version regressions, configuration gotchas, and edge cases around long sessions. The `prompt` and `agent` handler types are theoretically powerful but almost untested in community use, with known performance and cost implications. The hooks API is actively expanding (new events nearly every release) but has experienced multiple regressions and breaking changes, placing it firmly in the "rapidly maturing but not yet stable" category.

---

## 1. Reported Bugs and Issues

### 1.1 Systematic Bug Categories

Analysis of 20+ GitHub issues reveals five recurring failure categories:

#### Category A: Hooks Not Executing Despite Valid Configuration
The most frequently reported class of bug. Multiple root causes:

| Issue | Root Cause | Versions Affected |
|---|---|---|
| [#5093](https://github.com/anthropics/claude-code/issues/5093) | Wrong config file path (~/.claude.json instead of ~/.claude/settings.json) | v1.0.67 |
| [#6305](https://github.com/anthropics/claude-code/issues/6305) | PreToolUse/PostToolUse never fire; other event types work fine | v1.0.89 (OPEN) |
| [#8810](https://github.com/anthropics/claude-code/issues/8810) | Relative paths / ~ expansion fails from subdirectories | v2.0.5 |
| [#10814](https://github.com/anthropics/claude-code/issues/10814) | Complete regression: all hooks broken in v2.0.31 after fix in v2.0.30 | v2.0.27-29, v2.0.31 |
| [#11544](https://github.com/anthropics/claude-code/issues/11544) | /hooks shows "No hooks configured" despite valid JSON. Regression from prior versions | v2.0.37-v2.0.46 |
| [#2891](https://github.com/anthropics/claude-code/issues/2891) | Hooks not executing despite following documentation | Early versions |
| [#3828](https://github.com/anthropics/claude-code/issues/3828) | Hooks consistently ignored since v1.0.54 | v1.0.54+ |

**Pattern**: Hook loading/registration has been broken and re-broken across multiple version cycles. The v2.0.27-v2.0.31 regression timeline (broken -> fixed -> broken again within 5 days) is particularly concerning. Users cannot assume hooks work after upgrading.

#### Category B: VS Code Extension Hook Failures
The VS Code extension has a fundamentally different hook loading path than the CLI, causing a separate class of failures:

| Issue | Problem |
|---|---|
| [#18547](https://github.com/anthropics/claude-code/issues/18547) | Plugin hooks not loaded by VS Code (CLI works fine) — OPEN |
| [#16114](https://github.com/anthropics/claude-code/issues/16114) | Notification hooks not working in VS Code |
| [#8985](https://github.com/anthropics/claude-code/issues/8985) | Notification hook doesn't work in VS Code "native UI" mode |
| [#11156](https://github.com/anthropics/claude-code/issues/11156) | Notification hook does not fire in VS Code extension |
| [#28774](https://github.com/anthropics/claude-code/issues/28774) | Notification hooks (permission_prompt) don't fire in VS Code |
| [#21736](https://github.com/anthropics/claude-code/issues/21736) | Feature request: Hooks support in VS Code (implying incomplete) |

**Assessment**: The VS Code extension is a second-class citizen for hooks. CLI hooks are more reliable. Teams using VS Code should test hook behavior separately.

#### Category C: Silent Failures and Hangs
Hooks fail without any error indication:

| Issue | Problem |
|---|---|
| [#27467](https://github.com/anthropics/claude-code/issues/27467) | WorktreeCreate hooks hang indefinitely on unexpected stdout |
| [#16047](https://github.com/anthropics/claude-code/issues/16047) | Hooks silently stop after ~2.5 hours (48GB log file) |
| [#10225](https://github.com/anthropics/claude-code/issues/10225) | Plugin UserPromptSubmit hooks register and match but never execute |
| [#34457](https://github.com/anthropics/claude-code/issues/34457) | Hooks with shell commands cause 5+ minute hangs on Windows |
| [#21992](https://github.com/anthropics/claude-code/issues/21992) | Shell profile echo statements pollute hook stdout, breaking JSON parsing |

**Pattern**: Claude Code has weak error handling around hook execution. Unexpected stdout, large log files, and shell profile pollution all cause silent failures rather than meaningful error messages. The 48GB log file case (#16047) is instructive — hooks wrote unbounded logs until the filesystem effectively disabled them.

#### Category D: Event-Specific Execution Bugs
Specific hook events have unique reliability problems:

| Issue | Event | Problem |
|---|---|---|
| [#10373](https://github.com/anthropics/claude-code/issues/10373) | SessionStart | Output never processed for new conversations (works for /clear, /compact, resume) |
| [#8810](https://github.com/anthropics/claude-code/issues/8810) | UserPromptSubmit | Fails from subdirectories due to path resolution |
| [#9575](https://github.com/anthropics/claude-code/issues/9575) | Notification | ~25% fire rate reported |
| [#19225](https://github.com/anthropics/claude-code/issues/19225) | Stop | Stop hooks in Skills never fire |
| [#8320](https://github.com/anthropics/claude-code/issues/8320) | Notification | 60-second idle notifications not triggering |

**Key finding**: SessionStart hooks do not work for brand new conversations (#10373, 17+ upvotes). The `qz("startup")` function is only called during /compact, URL resume, and /clear — never for fresh sessions. This means context injection via SessionStart hooks is unreliable for the most common use case (starting work).

#### Category E: Security and Permission Interaction Bugs

| Issue | Problem |
|---|---|
| [#12176](https://github.com/anthropics/claude-code/issues/12176) | PermissionRequest race condition — dialog shows despite hook "allow" |
| [#37210](https://github.com/anthropics/claude-code/issues/37210) | permissionDecision "deny" ignored (user error: wrong exit code + missing wrapper) |
| [#33106](https://github.com/anthropics/claude-code/issues/33106) | deny not enforced for MCP server tool calls |
| [#36286](https://github.com/anthropics/claude-code/issues/36286) | PermissionDecision ignored in VS Code Extension |
| v2.1.78 security fix | PreToolUse "allow" was bypassing deny permission rules including enterprise managed settings |

**Critical**: The v2.1.78 security fix revealed that PreToolUse hooks returning "allow" could bypass deny permission rules, including enterprise managed settings. This was a security hole where hooks could escalate their own privileges.

### 1.2 Issue Volume and Trends

Based on the GitHub issue numbers and dates:
- Issues #2891-#6876: Early hooks (v1.0.x era, mid-2025) — fundamental execution failures
- Issues #8320-#12176: v2.0.x era (late 2025) — event-specific bugs, regressions
- Issues #16047-#21533: v2.1.x era (early 2026) — maturation bugs, edge cases, feature requests
- Issues #24327-#37210: Recent (2026) — model behavior issues, permission interaction bugs

The bug character is shifting from "hooks don't work at all" toward "hooks work but have edge cases." This suggests genuine maturation, though foundational issues (#6305 — PreToolUse never fires) remain open.

---

## 2. Handler Type Reliability Comparison

### 2.1 Command Handler (type: "command")

**Reliability: Moderate-High (when properly configured)**

The most tested and widely deployed handler type. Every community hook uses it.

**Success factors**:
- Deterministic — shell script either runs or doesn't
- Easy to test independently (`echo '{}' | ./hook.sh`)
- Quick feedback on failures via exit codes

**Known failure modes**:
| Failure Mode | Cause | Mitigation |
|---|---|---|
| Script not found | Wrong path, relative path, ~ expansion | Always use absolute paths |
| Permission denied | Missing chmod +x | Verify permissions |
| Silent JSON parse failure | Shell profile pollution (`.bashrc` echo) | Use `#!/usr/bin/env bash` with no profile, or redirect noise to stderr |
| Stdin not received | Inline commands vs script files behave differently | Use script files, not inline commands |
| PATH issues | Claude Code uses restricted PATH | Use absolute paths to executables |
| Timeout | Default 600s but complex operations can exceed | Set explicit shorter timeouts |
| Exit code confusion | Exit 1 = non-blocking (ignored), Exit 2 = blocking | Document exit code semantics clearly |
| Log file growth | Unbounded logging fills disk | Implement log rotation |

**Exit code behavior**:
- Exit 0: Success. JSON parsed from stdout. Execution proceeds.
- Exit 2: Blocking. Stderr fed to Claude as error. Tool call blocked (for PreToolUse).
- Exit 1, 3, 127, etc.: Non-blocking error. Stderr shown in verbose mode only. **This is the silent failure trap** — developers expect exit 1 to block but it doesn't.

**Performance**: Subprocess spawn overhead is typically <50ms. Script execution time dominates. Simple hooks (echo, jq parse) complete in <100ms. Complex hooks (npm test, tsc) can take 30-90 seconds.

### 2.2 HTTP Handler (type: "http")

**Reliability: Moderate (network-dependent)**

Used primarily by observability tools. Limited community adoption.

**Failure modes**:
| Failure Mode | Cause | Mitigation |
|---|---|---|
| Endpoint unreachable | Server down, network issue | Async mode + graceful timeout |
| Timeout | Default 30s; slow endpoints | Set explicit timeout |
| DNS resolution failure | Network configuration | Use localhost/IP for local servers |
| SSL/TLS errors | Certificate issues | Use HTTP for local, ensure valid certs for remote |

**Performance**: Network round-trip adds 1-50ms for local endpoints, potentially hundreds of ms for remote. Default 30s timeout. Deduplication by URL prevents duplicate requests.

**Key advantage**: Non-blocking via async mode. Ideal for telemetry/observability where response doesn't matter.

### 2.3 Prompt Handler (type: "prompt")

**Reliability: Low-Moderate (probabilistic by nature)**

Almost no community adoption. Theoretically useful for semantic evaluation.

**How it works**: Sends a prompt to a Claude model for single-turn evaluation. Model returns binary allow/deny decision. No tool access, no conversation history.

**Failure modes**:
| Failure Mode | Cause | Impact |
|---|---|---|
| False positive (wrongly allows) | Model misjudgment | Dangerous operation proceeds |
| False negative (wrongly blocks) | Model overly cautious | Legitimate operation blocked |
| Latency spike | API slowdown, rate limiting | Blocks tool execution for up to 30s |
| Token cost | Each evaluation consumes API tokens | Costs accumulate on PreToolUse (fires every tool call) |
| Model version drift | Default "fast model" changes between versions | Behavior changes without config changes |
| Race condition with PermissionRequest | Prompt hook takes >1-2s, dialog appears anyway | User sees dialog despite hook approval (#12176) |

**Performance**: Default 30s timeout. Typical latency 1-5 seconds depending on model and prompt complexity. Each invocation consumes API tokens. On PreToolUse (fires every tool call), this adds 1-5s latency per tool use.

**Fundamental limitation**: Single-turn evaluation without context. The evaluating model cannot see the conversation history, file contents, or broader intent. It judges a single tool call in isolation, making false positives and false negatives likely for nuanced policies.

**Community avoidance**: The community has overwhelmingly preferred deterministic `command` hooks with linters/tests over probabilistic `prompt` hooks. As noted in the community survey: "command type with linters/tests provides deterministic verification that developers trust more."

### 2.4 Agent Handler (type: "agent")

**Reliability: Unknown (near-zero community adoption)**

Spawns a full subagent with tool access for multi-step verification. The most powerful and most expensive handler type.

**Failure modes**:
| Failure Mode | Cause | Impact |
|---|---|---|
| High latency | Multi-turn tool-calling loop | 10-60+ seconds per evaluation |
| High token cost | Multiple model calls + tool calls | Significant cost on frequent events |
| Circular dependency | Agent hook on PreToolUse triggers its own tool calls | Officially prohibited for PreToolUse |
| Timeout | Default 60s, complex verifications can exceed | Hook fails, execution proceeds |
| Subagent error | Subagent crashes, hallucinates, or takes wrong action | Non-deterministic results |

**Performance**: The official documentation warns: "The sub-agent gets its own context and makes its own API calls, so on a large codebase where the hook fires frequently, you'll notice it in your bill." Default 60s timeout. Suitable only for infrequent events (Stop, TaskCompleted) — not for PreToolUse/PostToolUse which fire on every tool call.

**When it makes sense**: Quality gates on Stop or TaskCompleted events, where the hook fires once per turn (not per tool call) and thorough verification justifies the cost.

**Community status**: Almost no real-world adoption found. The JP Caparas Dev Genius article describes the concept. No published production configurations use agent hooks.

---

## 3. Performance Impact

### 3.1 Latency Model

Hooks execute at specific lifecycle points. The latency impact depends on event frequency and handler type:

| Event | Frequency | Impact of Slow Hook |
|---|---|---|
| PreToolUse | Every tool call (10-100+ per session) | **High** — directly delays each action |
| PostToolUse | Every tool call | **High** — delays before next action |
| Stop | Once per turn | **Low** — end of turn anyway |
| SessionStart | Once per session | **Low** — startup only |
| Notification | Occasional | **Low** — informational |
| PreCompact | Occasional (auto or manual) | **Low-Medium** — delays compaction |

### 3.2 Execution Architecture

**Within a matcher group**: Hooks run in **parallel**. Multiple hooks on the same event/matcher fire simultaneously. The slowest hook determines total latency.

**Across matcher groups**: Groups execute **sequentially**. If you have two matcher groups on PreToolUse, the second waits for the first to complete.

**Deduplication**: Identical command strings and HTTP URLs run once per event, preventing duplicate work.

**Async mode**: Only available for `command` hooks via `async: true`. Hook runs in background, does not block execution. Cannot affect decisions. Ideal for logging/telemetry.

### 3.3 Practical Latency Data

From community reports and documentation:

| Hook Type | Typical Latency | Notes |
|---|---|---|
| Simple command (echo, jq) | <100ms | Subprocess spawn overhead |
| File-based formatter (Prettier) | 200-500ms | File I/O + formatting |
| Linter (ESLint, Ruff) | 200-1000ms | Depends on project size |
| Test runner (npm test) | 5-90 seconds | Needs generous timeout |
| TypeScript compiler (tsc --noEmit) | 5-30 seconds | Project size dependent |
| HTTP to local server | 1-50ms | Network + processing |
| Prompt handler | 1-5 seconds | API call latency |
| Agent handler | 10-60+ seconds | Multi-turn tool loop |
| Async telemetry (SQLite write) | 50-200ms (non-blocking) | Runs in background |

### 3.4 Scaling Hooks

One developer reports running **95 hooks** across user-level and project-level settings "without noticeable latency because each hook completes in under 200ms." This suggests the practical limit is not hook count but per-hook latency.

**Key constraint**: Performance is determined by the slowest hook per event, not the total hook count. 95 fast hooks (<200ms each) running in parallel add ~200ms. 1 slow hook (npm test, 30s) dominates regardless of how many fast hooks accompany it.

**Context window pressure**: Hook output consumes context window space. Verbose hooks accelerate context compaction. The `async: true` mode helps for telemetry hooks, but any hook that returns `additionalContext` adds to context consumption.

### 3.5 Practical Limit on Hook Complexity

There is no hard limit on hook complexity, but UX degrades with:
- **>5 seconds per PreToolUse hook**: Noticeable pause on every tool call. Claude appears "stuck."
- **>30 seconds per PostToolUse hook**: Major workflow disruption. Users may interrupt.
- **>200ms per async observability hook**: Status messages create visual noise (addressed in v2.1.75 by suppressing async completion messages by default).

For quality gates on Stop events, longer hooks (30-120s for test suites) are acceptable since they run once per turn.

---

## 4. Edge Cases and Failure Modes

### 4.1 Hooks During Context Compaction

**PreCompact hook**: Fires before compaction. Receives `trigger` field (manual or automatic) and `custom_instructions` (user input for manual /compact). Cannot block compaction.

**PostCompact hook**: Fires after compaction completes (added in v2.1.76). Cannot block.

**Do hooks survive compaction?** Yes — hook configuration is separate from conversation context. Hooks are registered from settings.json at session start and remain active regardless of compaction. However:
- Hook **output** (additionalContext injected into context) may be summarized or lost during compaction, just like any other context content.
- Context preservation tools (who96/context-handoff, c0ntextKeeper) use PreCompact hooks specifically to save context before it is lost.
- Bug #19471 reports "Instructions completely ignored after context compaction" — CLAUDE.md rules get summarized away, but this affects CLAUDE.md, not hooks themselves. Hooks continue to execute; only their previously-injected context may be lost.

### 4.2 Hooks in Subagent Sessions

**Parent hooks DO apply to subagents.** PreToolUse, PostToolUse, and PostToolUseFailure events fire for subagent tool calls, with `agent_id` and `agent_type` fields in the JSON input. This allows hooks to:
- Apply the same safety gates to subagent actions
- Track which agent made each tool call
- Apply different policies for subagent vs main agent

**Subagent-specific events**:
- `SubagentStart`: Fires when subagent spawns. Cannot block, can inject context.
- `SubagentStop`: Fires when subagent finishes. CAN block (exit 2) to prevent subagent from stopping.

**Hooks defined in skill/agent frontmatter** are scoped to that component's lifecycle — they activate when the skill/agent is active and deactivate when it completes.

**Known issue**: #6522 reports subagents not executing configured hooks correctly (marked as duplicate of #6305).

### 4.3 Hooks with Long-Running Commands

**Default timeout: 600 seconds (10 minutes)** for command hooks. Configurable via `timeout` field.

**What happens on timeout**: The hook is killed and treated as a non-blocking error (equivalent to exit code 1). Execution proceeds. No explicit timeout error is shown to Claude unless the hook was expected to block.

**Practical issues**:
- npm test suites often need 60-90s. The 600s default is generous, but explicit timeouts are recommended.
- Windows users report 5+ minute hangs (#34457) from hooks with shell commands, independent of the configured timeout.
- The WorktreeCreate hang (#27467) is an infinite hang that never triggers timeout — the hook exits 0 but with unexpected stdout, causing Claude Code to parse indefinitely.

### 4.4 Race Conditions with Multiple Hooks

**Parallel execution within matcher groups creates race conditions** when hooks modify shared state:
- Two hooks both modifying the same file
- A formatter and a linter running simultaneously on the same file
- Multiple hooks writing to the same log file

**The PermissionRequest race condition (#12176)**: Permission dialog is rendered before hook results are awaited. If the hook takes >1-2 seconds, the user sees the dialog despite the hook approving.

**No sequential execution option**: Feature request #21533 proposed `"sequential": true` for hooks within a matcher group. Closed as "not planned." Gemini CLI supports this. The workaround is a single dispatcher script that runs sub-hooks sequentially.

### 4.5 Hooks That Modify State

Hooks can freely modify the filesystem, run commands, and make network requests. Claude Code imposes no sandboxing on hook execution. Known risks:

- **File modification during edit**: A PostToolUse formatter modifies a file that Claude is about to edit again — the edit may conflict.
- **Git state changes**: Auto-staging hooks (git add) can stage unintended files if they run concurrently with Claude's operations.
- **Environment variable leaks**: Pre-v2.1.78, hooks had access to the full process environment including API keys. The `CLAUDE_CODE_SUBPROCESS_ENV_SCRUB=1` setting (v2.1.78) strips credentials from subprocess environments.
- **Log file growth**: Unbounded logging caused a 48GB log file (#16047), silently disabling all hooks.

### 4.6 PreToolUse Block — Claude's Response Behavior

When a PreToolUse hook blocks a tool call (exit 2), Claude receives the denial reason as an error message. Claude's response is **non-deterministic**:

**Sometimes Claude**:
- Reads the error, fixes the issue, and retries (desired behavior)
- Tries an alternative approach

**Sometimes Claude**:
- Stops and waits for user input, treating it like a user denial (#24327)
- Acknowledges the block but moves on without addressing the underlying issue

**Root cause**: The model cannot distinguish between "automated quality gate with fixable feedback" and "user denied this action." The training signal for tool denials is conservative — stop and ask.

**Workaround**: Inject directives via UserPromptSubmit hooks telling Claude to treat hook blocks as quality gates, not user denials. Additionally, include the `additionalContext` field alongside denials to provide Claude with specific guidance on how to proceed.

### 4.7 Two Exit Code Systems (Critical Gotcha)

There are two distinct ways to block actions in PreToolUse hooks, and they have different semantics:

**Method 1: Exit code 2** (simple block)
- Stderr is fed to Claude as error message
- No JSON processing
- Claude sees a generic "blocked" signal

**Method 2: JSON permissionDecision with exit code 0** (structured denial)
- Must exit 0 (not 2!)
- Must include `hookSpecificOutput` wrapper
- Can specify `permissionDecision: "deny"`, `"allow"`, or `"ask"`
- Can include `permissionDecisionReason` and `additionalContext`
- Can modify tool input with `updatedInput`

**The trap**: Using exit code 2 with JSON permissionDecision. Exit code 2 causes Claude Code to ignore JSON output. The permissionDecision is silently discarded. Several bug reports (#37210) stem from this misunderstanding.

---

## 5. Maturity Assessment

### 5.1 API Stability

**The hooks API is not stable.** Evidence:

**Rapid event expansion**: New hook events added in nearly every minor version:
- v2.1.76: Elicitation, ElicitationResult, PostCompact
- v2.1.78: StopFailure
- v2.1.83: CwdChanged, FileChanged
- v2.1.84: TaskCreated
- v2.1.85: CronCreate, conditional `if` field

From 12 events at introduction to 21+ events in the current version. The event surface area nearly doubled.

**Version regressions**: Hooks have been completely broken and fixed multiple times:
- v2.0.27: All hooks broken
- v2.0.30: Fixed
- v2.0.31: Broken again (1 day later)
- v2.0.37+: Partial fix (only some event types work)
- Ongoing intermittent reports through v2.1.x

**Breaking changes**:
- v2.1.78: PreToolUse "allow" no longer bypasses deny permission rules (security fix, but changed behavior)
- v2.1.75: Async hook completion messages suppressed by default (UI behavior change)
- v2.1.85: New conditional `if` field (additive, backward-compatible)

**Release cadence**: 13 versions in 3 weeks of March 2026. This pace makes it difficult for hook authors to test against specific versions.

### 5.2 Security Track Record

Two CVEs directly involving hooks:
- **CVE-2025-59536** (CVSS 8.7): Malicious hooks in untrusted repos executed automatically. Fixed v1.0.111, October 2025.
- **CVE-2026-21852** (CVSS 5.3): API key exfiltration via configuration manipulation. Fixed v2.0.65, January 2026.
- **v2.1.78 security fix**: PreToolUse "allow" bypassed enterprise deny rules.

The security model has improved: trust dialogs now required, hook sources displayed in permission prompts (v2.1.75), credential scrubbing added (v2.1.78). But the presence of a CVSS 8.7 vulnerability in the hook system indicates the security surface was not fully considered at launch.

### 5.3 Platform Parity

| Feature | CLI | VS Code Extension |
|---|---|---|
| Command hooks (settings.json) | Reliable | Mostly works |
| Plugin hooks (hooks.json) | Works | Broken (#18547, OPEN) |
| Notification hooks | Works | Unreliable (multiple issues) |
| PermissionRequest hooks | Works (with race condition) | Unreliable |

The CLI is the primary development target. The VS Code extension lags in hook support.

### 5.4 Documentation Quality

**The exit code behavior is poorly documented**, leading to the most common user errors:
- Exit 1 vs exit 2 confusion (most developers expect exit 1 to block)
- JSON permissionDecision requiring exit 0 (counter-intuitive)
- `hookSpecificOutput` wrapper requirement not prominent
- SessionStart hook limitations (doesn't work for new sessions) undocumented

**The hooks reference page** is comprehensive for happy-path usage but sparse on failure modes, edge cases, and platform-specific caveats.

### 5.5 Overall Maturity Rating

| Aspect | Rating | Evidence |
|---|---|---|
| Core execution (command hooks, CLI) | **Beta** | Works for common cases, version regressions, edge case failures |
| VS Code extension hooks | **Alpha** | Fundamental loading bugs, plugin hooks broken |
| Prompt/Agent handlers | **Experimental** | Near-zero community adoption, performance concerns |
| API stability | **Unstable** | New events every release, behavioral changes, regressions |
| Security model | **Maturing** | CVEs fixed, trust model improved, but ongoing permission bugs |
| Documentation | **Adequate** | Covers happy path well, poor on failure modes |

**Bottom line**: Hooks are production-usable for simple `command`-type hooks on common events (PostToolUse formatting, PreToolUse safety gates, Stop notifications) in the CLI. Anything beyond that baseline — VS Code, plugins, prompt/agent handlers, long sessions, complex event interactions — requires careful testing and version pinning.

---

## 6. Recommendations for Consulting Adoption

1. **Pin Claude Code versions** in team configurations. Do not auto-update. Test hooks after each upgrade.
2. **Use command hooks exclusively** for production enforcement. Avoid prompt/agent handlers until maturity improves.
3. **Use absolute paths** in all hook commands. Never use `~` or relative paths.
4. **Prefer CLI over VS Code** for hook-dependent workflows.
5. **Implement log rotation** in any hook that writes logs.
6. **Test hooks independently** before relying on them: `echo '{"tool_name":"Bash","tool_input":{"command":"test"}}' | ./your-hook.sh`
7. **Use exit code 2** for simple blocking. Use JSON `permissionDecision` with exit 0 for structured denials. Never combine them.
8. **Add UserPromptSubmit directives** instructing Claude to treat hook blocks as quality gates, not user denials.
9. **Keep PreToolUse hooks fast** (<200ms). Use async mode for observability hooks.
10. **Monitor for the SessionStart new-session bug** (#10373) — use /clear workaround if needed.

---

## Sources

All raw source material saved to `docs/`:

### GitHub Issues (anthropics/claude-code)
- `docs/github-issue-16047-hooks-stop-after-2h.md` — Hooks stop executing after ~2.5 hours
- `docs/github-issue-27467-worktree-hooks-silent-hang.md` — WorktreeCreate hooks hang indefinitely
- `docs/github-issue-10225-plugin-hooks-not-executing.md` — Plugin UserPromptSubmit hooks never execute
- `docs/github-issue-5093-hooks-detected-not-executing.md` — Wrong config path common error
- `docs/github-issue-11544-hooks-not-loading.md` — /hooks shows no hooks (regression)
- `docs/github-issue-18547-plugin-hooks-vscode.md` — Plugin hooks broken in VS Code (OPEN)
- `docs/github-issue-10373-sessionstart-new-conversations.md` — SessionStart broken for new sessions
- `docs/github-issue-32954-silent-async-hooks.md` — Async hook status message noise
- `docs/github-issue-8810-subdirectory-hooks.md` — Path resolution from subdirectories
- `docs/github-issue-10814-hooks-regression-v2031.md` — v2.0.31 regression timeline
- `docs/github-issue-12176-permission-race-condition.md` — PermissionRequest race condition
- `docs/github-issue-21533-sequential-hooks-request.md` — Sequential execution request (rejected)
- `docs/github-issue-24327-pretooluse-block-stops-claude.md` — Exit code 2 stops Claude
- `docs/github-issue-37210-deny-ignored-edit-tool.md` — Two exit code systems gotcha
- `docs/github-issue-6305-pretooluse-not-executing.md` — Pre/PostToolUse never fire (OPEN)

### Official Documentation
- `docs/hooks-reference-official-detailed.md` — Detailed execution behavior from official hooks reference

### Changelog
- `docs/changelog-hooks-entries.md` — All hook-related changelog entries

### Security
- `docs/cve-2025-59536-hooks-security.md` — CVE-2025-59536 and CVE-2026-21852 details
