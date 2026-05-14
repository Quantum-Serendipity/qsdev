<!-- Source: https://news.ycombinator.com/item?id=47004712 -->
<!-- Retrieved: 2026-03-26 -->

# Claude DevTools — Full HN Discussion (69 points, 44 comments)

## Submission
- **Title**: Show HN: I built a tool to un-dumb Claude Code's CLI output (Local Log Viewer)
- **Submitter**: matt1398
- **Score**: 69 points | 44 comments

## Key Themes in Discussion

### 1. "Why not just use alternatives?" (miroljub)
- OpenCode and Pi Code Agent suggested as full replacements
- azuanrb warns: using non-official tools with Claude Code subscription may violate ToS and risk account ban
- syabro: "nothing compares to the $200 Max CC subscription"
- mentalgear: asks for benchmarks comparing these alternatives

### 2. "Just use --verbose" (KingMob)
- matt1398 responds: --verbose floods terminal with noise, defeats purpose
- This tool is passive viewer for post-mortem debugging, not real-time wrapper
- Especially useful for retroactively debugging multiple parallel sessions after shutdown

### 3. OpenTelemetry should be standard (cjonas)
- matt1398: Claude Code supports basic OTel but limited to high-level metrics
- Log parsing remains the primary way to achieve real observability

### 4. Format stability concerns (6LLvveMx2koXfwn)
- "Won't this break every time the log format changes?"
- matt1398: Claude Code's official VS Code extension reads same .jsonl files, so format stability should match first-party extension stability. Adding new handlers is "trivial."
- kzahel: .jsonl format is undocumented API surface representing a risk. Built own project using Zod schema validations to track format changes. Notes CLI modifies files during runtime for cleanup/migration.

### 5. Token usage visibility (khoury)
- Frustrated that third-party tool needed just to check token usage
- matt1398: /usage and /status provide basic counts. This tool breaks down per-turn consumption (file reads vs tool output). Makes zero network calls so cannot access billing data.

### 6. Config file criticism (gregoriol)
- ">20 config files" for "a simple tool" — criticizes current development practices
- osener defends: "next level nitpicking," same as IDE configs
- matt1398: configs prevent agent derailment through immediate stderr feedback

### 7. "Why watch every keystroke?" (small_model)
- Suggests just reviewing diffs instead of babysitting
- matt1398: Not for watching every session — standard observability. Useful when agents get stuck or context fills unexpectedly. Helps find failure points across parallel sessions quickly.
- small_model accepts rationale, notes they use planning mode instead
- Grimblewald: depends on work type — academic/research needs frequent correction

### 8. Positive Reception
- igravious: "Anthropic should hire this person"
- eurekin: plans to test it, thanks developer
- Multiple users acknowledge observability gap this fills

### 9. Claude Code UX Frustration Thread
- bjt12345: confusing status messages like "Gittifying..." caused panic
- kzahel: Anthropic could add setting to disable cute phrases but hasn't
- JimDabell: Anthropic recently added spinnerVerbs config for custom messages
- miroljub: "Anthropic is mocking developers" by hiding actual info while allowing message customization

### 10. Creator's Design Philosophy (matt1398, multiple replies)
- "Passive viewer" — not a wrapper, preserves native terminal workflows
- Useful for post-mortem debugging of completed sessions
- Works with every session ever executed regardless of execution context
- Zero network calls, zero modification to Claude Code
- Adding handlers for format changes is "trivial"
- Standard observability for AI agents, not micromanagement
