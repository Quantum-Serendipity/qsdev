<!-- Source: https://news.ycombinator.com/item?id=47350416 -->
<!-- Retrieved: 2026-03-26 -->

# Show HN: Rudel – Claude Code Session Analytics

**Submitter:** keks0r
**URL:** https://github.com/obsessiondb/rudel
**Points:** 144 | **Comments:** 86

## Overview
The creators built analytics tooling for Claude Code sessions after realizing they lacked visibility into their AI coding workflows. Their dataset contains 1,573 real sessions, 15M+ tokens, and 270K+ interactions collected over three months from a six-person team.

## Key Findings
- Skills activated in only 4% of sessions (improved with Claude 4.6)
- 26% of sessions abandoned within 60 seconds
- Documentation tasks scored highest success; refactoring lowest
- Initial 2-minute error patterns predict abandonment reliably
- No established benchmarks for agentic session performance exist

## Main Discussion Threads

### Skills Usage Inconsistency
Users reported Claude inconsistently following skill instructions. Responses suggested simplifying CLAUDE.md, disabling skills, or using explicit skill invocation patterns. The team acknowledged version 4.6 significantly improved skill adoption.

### Abandonment Interpretation
Commenters questioned whether "abandoned" sessions represent failures or efficient quick completions. Some abandon sessions when Claude misdirects effort rather than continuing corrections.

### Privacy Concerns
Multiple users expressed hesitation uploading logs to third-party services. The team emphasized self-hosting options via Docker and local-only operation.

### Practical Applications
Developers shared strategies: explicit skill invocation via slash commands, prompt templates for stage identification, and structured pipelines with review gates that boosted first-pass acceptance from 73% to 90%.

## Related Tools Mentioned
- **AgentsView.io** – Similar analytics platform
- **K9 Audit** – Local-first causal auditing with hash chains
- **Linko** – MITM proxy for inspecting Claude Code API traffic (macOS only)
- **claude-code-log-analyzer** – Custom log analysis scripts
