# Learning Opportunities Auto — Hook Implementation (Full Content)
- **Source**: https://github.com/DrCatHicks/learning-opportunities/tree/main/learning-opportunities-auto/
- **Retrieved**: 2026-05-14
- **Method**: gh api (raw content via base64 decode)

## Plugin Architecture

Three files compose this plugin:

### .claude-plugin/plugin.json
```json
{
  "name": "learning-opportunities-auto",
  "version": "1.0.1",
  "description": "Automatically nudges Claude to offer learning exercises after git commits. Requires the learning-opportunities plugin.",
  "license": "CC-BY-4.0"
}
```

### hooks/hooks.json (Claude Code format)
```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "bash ${CLAUDE_PLUGIN_ROOT}/hooks/post-tool-use.sh"
          }
        ]
      }
    ]
  }
}
```

### hooks.codex.json (Codex format)
```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Bash|exec_command",
        "hooks": [
          {
            "type": "command",
            "command": "bash ./hooks/post-tool-use.sh"
          }
        ]
      }
    ]
  }
}
```

### hooks/post-tool-use.sh
- Fires after every Bash tool use
- Checks if the command was a `git commit` via grep on the JSON payload
- Extracts session_id for rate limiting
- Tracks offers per session via temp file (STATE_FILE keyed on session ID)
- Stops after 2 offers per session
- Outputs structured JSON with `hookSpecificOutput` containing the nudge message
- The nudge tells Claude to consider offering a learning exercise, not to start one directly
- No external dependencies beyond bash and standard Unix tools
