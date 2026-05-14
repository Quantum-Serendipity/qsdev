# SWE-Agent: Agent-Computer Interface Design

- **Source**: https://swe-agent.com/latest/background/, https://arxiv.org/pdf/2405.15793, https://github.com/SWE-agent/SWE-agent
- **Retrieved**: 2026-03-15

## Overview

SWE-agent uses a language model to interact with a computer to solve software engineering tasks. The key innovation is the Agent-Computer Interface (ACI) — custom-built commands and feedback formats designed to complement LM limitations and abilities.

## Key Insight

Just as good UI design matters for human-computer interaction, good ACI design matters for agent-computer interaction. A baseline agent without a well-tuned ACI does much worse than SWE-agent. The ACI shapes actions, documentation, and environment feedback to complement LM capabilities.

## Custom Commands

The ACI provides specialized commands instead of raw shell:
- **find_file**: Search for files by name
- **search_file**: Search within a file
- **search_dir**: Search across directories, outputting summary results
- **open**: Open a file at a specific line
- **edit**: Edit a range of lines with replacement text
- **scroll_up/scroll_down**: Navigate within open file
- **create**: Create a new file

## File Viewer

Instead of raw `cat`, SWE-agent supplies a specialized file viewer:
- Displays 100 lines at a time (agents get overwhelmed with more — interestingly, humans work similarly)
- Commands for scrolling up/down and searching within file
- Line numbers shown for reference
- Current position tracking

## Linter Integration

The linter is critical for edit quality:
- Runs when edit command is issued
- Invalid edits (syntactically incorrect code) are **rejected entirely**
- Select errors shown to agent with before/after snippets
- Agent must retry the edit until it passes linting
- Prevents cascading errors from bad edits being applied

## Typical Workflow

1. **Reproduction/Localization**: Begin with either writing reproduction code (create) or finding relevant files (find_file/search_dir)
2. **Edit-Execute Loops**: From turn 5 onwards, most frequent actions are edit and python
3. **Verification**: Run tests or reproduction scripts to verify fix

## Common Failure Modes

- Agent misunderstanding which directory it's in
- Duplicating path components in file paths
- Getting stuck in loops during retry attempts with windowed edit
- Overwhelming context when too many lines shown at once

## Performance

Published at NeurIPS 2024. The ACI design contributed significantly to performance — the interface design alone (independent of the LLM) was a major factor in results.
