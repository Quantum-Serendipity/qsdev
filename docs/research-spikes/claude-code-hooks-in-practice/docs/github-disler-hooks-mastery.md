# Claude Code Hooks Mastery - disler/claude-code-hooks-mastery
- **Source**: https://github.com/disler/claude-code-hooks-mastery
- **Retrieved**: 2026-03-27

## Overview
This repository provides comprehensive mastery of Claude Code hooks, enabling deterministic control over Claude Code behavior through 13 distinct hook lifecycle events. The implementation uses UV single-file scripts for clean separation of hook logic.

## The 13 Hook Events

### 1. UserPromptSubmit Hook
- **Fires:** Immediately when user submits a prompt (before Claude processes)
- **Payload:** prompt text, session_id, timestamp
- **Capability:** Can block prompts and inject context
- **Use Cases:** Prompt validation, logging, context injection, security filtering

### 2. PreToolUse Hook
- **Fires:** Before any tool execution
- **Payload:** tool_name, tool_input parameters
- **Capability:** Can block tool execution (exit code 2)
- **Use Cases:** Security blocking (e.g., `rm -rf`, .env access), parameter validation

### 3. PostToolUse Hook
- **Fires:** After successful tool completion
- **Payload:** tool_name, tool_input, tool_response with results
- **Capability:** Cannot block (tool already executed)
- **Use Cases:** Logging, transcript conversion, result validation, cleanup

### 4. PostToolUseFailure Hook
- **Fires:** When tool execution fails
- **Payload:** tool_name, tool_input, tool_use_id, error object
- **Capability:** Cannot block
- **Use Cases:** Structured error logging with full context

### 5. Notification Hook
- **Fires:** When Claude Code sends notifications
- **Payload:** message content
- **Capability:** Cannot block
- **Use Cases:** TTS alerts, logging, custom notifications

### 6. Stop Hook
- **Fires:** When Claude Code finishes responding
- **Payload:** stop_hook_active boolean flag
- **Capability:** Can block stopping (forces continuation)
- **Use Cases:** AI-generated completion messages with TTS, ensuring task completion

### 7. SubagentStart Hook
- **Fires:** When subagents (Task tools) spawn
- **Payload:** agent_id, agent_type, session info
- **Capability:** Cannot block
- **Use Cases:** Spawn logging, optional TTS announcement

### 8. SubagentStop Hook
- **Fires:** When subagents finish responding
- **Payload:** stop_hook_active boolean flag
- **Capability:** Can block subagent stopping
- **Use Cases:** TTS playback ("Subagent Complete"), ensuring completion

### 9. PreCompact Hook
- **Fires:** Before compaction operations
- **Payload:** trigger ("manual" or "auto"), custom_instructions, session info
- **Capability:** Cannot block
- **Use Cases:** Transcript backup, context preservation, pre-compaction logging

### 10. SessionStart Hook
- **Fires:** When sessions start or resume
- **Payload:** source ("startup", "resume", or "clear"), session info
- **Capability:** Cannot block
- **Use Cases:** Development context loading (git status, recent issues), environment setup

### 11. SessionEnd Hook
- **Fires:** When session ends (exit, sigint, or error)
- **Payload:** session_id, transcript_path, cwd, permission_mode, reason
- **Capability:** Cannot block
- **Use Cases:** Session logging, optional cleanup (removes temp files, stale logs)

### 12. PermissionRequest Hook
- **Fires:** When user shown permission dialog
- **Payload:** tool_name, tool_input, tool_use_id, session info
- **Capability:** Cannot block
- **Use Cases:** Permission auditing, auto-allow for read-only ops (Read, Glob, Grep, safe Bash)

### 13. Setup Hook
- **Fires:** When Claude enters repository (init) or periodically (maintenance)
- **Payload:** trigger ("init" or "maintenance"), session info
- **Capability:** Cannot block
- **Use Cases:** Environment persistence via `CLAUDE_ENV_FILE`, context injection via `additionalContext`

## Key Implementation Details

### Exit Code Behavior
- **0:** Success (stdout shown to user in transcript mode)
- **2:** Blocking error (stderr fed back to Claude automatically for PreToolUse/UserPromptSubmit; shows error for other hooks)
- **Other:** Non-blocking error (stderr shown to user, execution continues)

### Project Structure
```
.claude/
├── settings.json                 # Hook configuration with permissions
├── hooks/                        # Python scripts using UV
│   ├── user_prompt_submit.py    # Prompt validation, logging
│   ├── pre_tool_use.py          # Security blocking
│   ├── post_tool_use.py         # Logging, transcript conversion
│   ├── post_tool_use_failure.py # Error logging
│   ├── notification.py          # Logging with optional TTS
│   ├── stop.py                  # AI-generated messages with TTS
│   ├── subagent_stop.py         # Simple TTS announcements
│   ├── subagent_start.py        # Spawn logging with optional TTS
│   ├── pre_compact.py           # Transcript backup
│   ├── session_start.py         # Development context loading
│   ├── session_end.py           # Session cleanup
│   ├── permission_request.py    # Permission auditing
│   ├── setup.py                 # Repository initialization
│   ├── validators/
│   │   ├── ruff_validator.py   # Python linting (PostToolUse)
│   │   └── ty_validator.py     # Type checking (PostToolUse)
│   └── utils/
│       ├── tti/tts_queue.py    # Queue-based TTS management
│       └── llm/task_summarizer.py # LLM-powered summaries
├── status_lines/                # Real-time terminal displays (v1-v9)
├── output-styles/               # Response formatting configs
├── commands/                    # Custom slash commands
├── agents/                      # Sub-agent configurations
└── logs/                        # JSON logs of all hook executions
```

## Key Features
- **Complete coverage:** All 13 hooks implemented and logging (11/13 validated via automation)
- **Intelligent TTS system:** AI-generated audio with voice priority (ElevenLabs > OpenAI > pyttsx3)
- **Security enhancements:** Blocks dangerous commands at multiple levels
- **Automatic logging:** All hook events logged as JSON to `logs/` directory
- **Team-based validation:** Builder/Validator agent pattern with code quality hooks
- **Chat transcript extraction:** PostToolUse hook converts JSONL transcripts to readable JSON

## Architecture: UV Single-File Scripts
Each hook is self-contained with embedded dependency declarations, providing:
- **Isolation:** Hook logic separate from project dependencies
- **Portability:** Each script declares its own dependencies
- **No venv management:** UV handles dependencies automatically
- **Fast execution:** Lightning-fast dependency resolution
- **Self-contained:** Understand and modify hooks independently
