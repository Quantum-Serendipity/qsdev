<!-- Source: https://news.ycombinator.com/item?id=47276604 -->
<!-- Retrieved: 2026-03-26 -->

# Show HN: Claude-replay – A video-like player for Claude Code sessions

**Submitter:** es617
**Repository:** https://github.com/es617/claude-replay
**Example Demo:** https://es617.github.io/assets/demos/peripheral-uart-demo.html

## Core Concept
The creator built a tool addressing a practical pain point: "I got tired of sharing AI demos with terminal screenshots or screen recordings." The solution converts Claude Code's locally-stored JSONL session logs into interactive HTML replays that enable stepping through sessions, inspecting tool calls, and reviewing full conversations in a self-contained file.

## Key Features Discussed
- Single HTML file output with no dependencies
- Works across email, web hosting, blog embedding, and mobile devices
- Allows timeline jumping and expansion of tool calls
- Supports structured step-by-step navigation via arrow keys

## Notable Discussion Points

**Use Cases Identified:**
- Team onboarding and knowledge sharing
- Demonstrating prompting techniques and best practices
- Hardware project workflows where tool usage matters
- Educational contexts for non-technical stakeholders

**Feature Requests:**
- Slack integration capabilities
- Session search/discovery functionality (many users manage hundreds of sessions)
- Cursor IDE support (author subsequently added this with heuristic thinking block estimation)
- Keyboard shortcuts for jumping between thinking blocks or tool calls

**Related Tools Mentioned:**
- claude-code-transcripts
- coding_agent_session_search
- agentlore (supports multiple coding agents with team log aggregation)

The discussion reflects genuine enthusiasm for sharing reproducible AI workflows beyond static screenshots or lengthy recordings.
