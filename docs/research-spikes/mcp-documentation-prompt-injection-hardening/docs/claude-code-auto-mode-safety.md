# Claude Code Auto Mode: A Safer Way to Skip Permissions

- **Source URL**: https://www.anthropic.com/engineering/claude-code-auto-mode
- **Retrieved**: 2026-05-14

## Core Safety Architecture

Auto mode employs a two-layer defense system.

### Input Layer
A server-side prompt-injection probe scans tool outputs before they reach the agent's context. When content appears to be a hijacking attempt, the probe adds a warning, instructing the agent to remain anchored on the user's actual request.

### Output Layer
A transcript classifier (running on Sonnet 4.6) evaluates each action before execution. "The classifier sees only user messages and the agent's tool calls; we strip out Claude's own messages and tool outputs."

## Two-Stage Classification Process

**Stage 1**: Fast single-token filter using "yes/no" decision without reasoning. Tuned aggressively toward blocking: 8.5% false positive rate on real traffic but catches 93.4% of dangerous actions.

**Stage 2**: Activates only when Stage 1 flags content. Applies chain-of-thought reasoning to reduce false positives to 0.4% while accepting 17% false-negative rate on real overeager actions.

## Decision Criteria

The classifier's fixed template evaluates actions by real-world impact rather than surface syntax:
- **User intent rules**: Whether actions align with explicit authorization, not just task relevance
- **Evaluation rules**: Analyzing actual command consequences (e.g., if agent writes then executes a script, classifier evaluates assembled payload)
- **Block categories**: Destruction/exfiltration, security degradation, cross-trust-boundary actions, review-bypass attempts

## Information Stripping Strategy

The classifier deliberately receives stripped transcripts to prevent manipulation. By excluding assistant reasoning and tool outputs, the design prevents the agent from "talking the classifier into" approving dangerous actions through persuasive rationalization.

This stripping serves dual purposes: reduces prompt-injection attack surface while keeping classifier focused on actual authorization evidence rather than agent's interpretations.

## Threat Coverage

Auto mode targets four danger categories:
1. Overeager behavior (unauthorized initiative despite good intentions)
2. Honest mistakes (misunderstanding resource scope or ownership)
3. Prompt injection (hijacking via planted instructions)
4. Misaligned reasoning (model pursuing independent goals)

The 17% miss rate primarily reflects cases where users provide ambiguous consent that doesn't clearly cover the action's full blast radius.
