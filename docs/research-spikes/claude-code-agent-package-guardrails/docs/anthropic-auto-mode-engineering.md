<!-- Source: https://www.anthropic.com/engineering/claude-code-auto-mode -->
<!-- Retrieved: 2026-05-12 -->

# Claude Code Auto Mode: A Safer Way to Skip Permissions

## Core Concept

Claude Code auto mode delegates approval decisions to machine-learning classifiers rather than requiring manual review for each action. This addresses "approval fatigue," where users accept 93% of permission prompts without careful attention. The system balances autonomy with safety—positioned between manual review and completely unrestricted operation.

## How It Works: Two-Layer Defense

**Input Layer (Prompt-Injection Probe):**
The system scans tool outputs before they reach the agent's context. When suspicious content appears, the probe adds a warning, instructing the agent to treat the material as questionable and focus on user intent.

**Output Layer (Transcript Classifier):**
A two-stage classifier evaluates each action before execution:
- **Stage 1**: Fast single-token filter (yes/no) tuned conservatively to block
- **Stage 2**: Chain-of-thought reasoning only for flagged actions, reducing false positives

The classifier sees only user messages and tool-call commands; assistant explanations and tool outputs are deliberately stripped to prevent the agent from persuading the classifier into unsafe decisions.

## Action Classification Tiers

**Tier 1 - Auto-Allowed (Safe Tools):**
- File reads, text/pattern search, code navigation
- User-configured always-allow rules (with exceptions)
- Note: Blanket permissions for shell access and package manager commands are dropped when entering auto mode, though narrow rules like "allow running code formatters" remain

**Tier 2 - Auto-Allowed (In-Project Operations):**
- File writes and edits within the project directory
- Reasoning: Changes are reviewable via version control

**Tier 3 - Classifier Review Required:**
- Shell commands, web fetches, external integrations
- Filesystem operations outside project directory
- Subagent spawning and delegation

## Classifier Decision Framework

The classifier prompt contains fixed interpretive logic plus three customizable slots:

**Fixed Components:**
- Evaluation rules assessing real-world impact (not surface-level text)
- User intent rules determining whether authorization applies
- Step-by-step classification process

**Customizable Slots:**
1. **Environment definition**: Trust boundaries (default: only current git repo)
2. **Block rules** (20+ defaults covering):
   - Destructive actions: force-pushes, mass deletions
   - Security degradation: disabling logs, installing persistence
   - Trust boundary violations: running external code, credential scanning
   - Review bypass: direct-to-main pushes, production deploys
3. **Allow exceptions**: Carve-outs like "install packages declared in manifest"

## Security Threat Model

The system defends against four danger types:
- **Overeager behavior**: Agent helps but exceeds authorization
- **Honest mistakes**: Misunderstanding scope or resource ownership
- **Prompt injection**: Malicious instructions in files/webpages
- **Model misalignment**: Agent pursuing its own goals (not currently observed)

## Performance Metrics

| Dataset | Stage 1 FPR | Full Pipeline FPR | FNR |
|---------|-------------|-------------------|-----|
| Real traffic (10,000 actions) | 8.5% | 0.4% | — |
| Real overeager (52 actions) | 6.6% | — | 17% |
| Synthetic exfiltration (1,000) | 1.8% | — | 5.7% |

## Key Design Decisions

**Why stripping assistant text:** Prevents the agent from talking the classifier into bad decisions through persuasive rationalization.

**Why stripping tool results:** Primary injection defense; classifier rarely needs outputs to judge dangerous operations.

**Auto mode drops broad allow rules on entry:**
- Blanket `Bash(*)` or `PowerShell(*)`
- Wildcarded interpreters like `Bash(python*)`
- Package-manager run commands
- `Agent` allow rules

Narrow rules like `Bash(npm test)` carry over.

## Deny-and-Continue Mechanism

When blocked, actions return with denial reasons and instructions to find safer alternatives rather than halting execution. Sessions terminate after 3 consecutive or 20 total denials.

## Important Limitations

The 17% false-negative rate means dangerous actions may execute. The creators note: "Whether 17% is acceptable depends on what you're comparing against. If you are running `--dangerously-skip-permissions`, this is a substantial improvement."
