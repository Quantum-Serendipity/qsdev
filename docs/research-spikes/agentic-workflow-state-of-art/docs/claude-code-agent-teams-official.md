# Claude Code: Agent Teams (Swarms) — Official Documentation
- **Source**: https://code.claude.com/docs/en/agent-teams
- **Retrieved**: 2026-03-15
- **Type**: Official documentation

## Overview
Agent teams coordinate multiple Claude Code instances working together. One session acts as team lead, coordinating work, assigning tasks, and synthesizing results. Teammates work independently, each in its own context window, and can communicate directly with each other.

**Experimental feature**: Enable with CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS=1 in settings.json.

## Architecture
| Component | Role |
|---|---|
| Team lead | Main session that creates team, spawns teammates, coordinates work |
| Teammates | Separate Claude Code instances working on assigned tasks |
| Task list | Shared list of work items that teammates claim and complete |
| Mailbox | Messaging system for inter-agent communication |

## Agent Teams vs Subagents
| | Subagents | Agent teams |
|---|---|---|
| Context | Own context; results return to caller | Own context; fully independent |
| Communication | Report back to main agent only | Message each other directly |
| Coordination | Main agent manages all work | Shared task list with self-coordination |
| Best for | Focused tasks, result matters | Complex work requiring discussion |
| Token cost | Lower (summarized back) | Higher (each is separate instance) |

## Best Use Cases
- Research and review (parallel investigation)
- New modules/features (independent ownership)
- Debugging with competing hypotheses
- Cross-layer coordination (frontend/backend/tests)

## Display Modes
- In-process: All in main terminal, Shift+Down to cycle
- Split panes: Each teammate in own pane (requires tmux or iTerm2)

## Key Features
- Task dependency management (automatic unblocking)
- Plan approval (require teammates to plan before implementing)
- Direct teammate messaging (bypass lead)
- Task self-claiming with file locking
- Quality gates via TeammateIdle and TaskCompleted hooks

## Best Practices
- Start with 3-5 teammates
- 5-6 tasks per teammate
- Size tasks appropriately (not too small, not too large)
- Start with research/review before parallel implementation
- Avoid file conflicts (each teammate owns different files)
- Monitor and steer progress

## Limitations
- No session resumption with in-process teammates
- Task status can lag
- One team per session
- No nested teams
- Lead is fixed
- Permissions set at spawn
- Split panes require tmux or iTerm2
