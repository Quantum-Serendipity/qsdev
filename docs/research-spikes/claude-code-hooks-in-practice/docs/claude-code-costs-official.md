# Claude Code: Manage Costs Effectively — Official Documentation
- **Source**: https://code.claude.com/docs/en/costs
- **Retrieved**: 2026-03-27
- **Type**: Official documentation

## Overview

Claude Code consumes tokens for each interaction. Costs vary based on codebase size, query complexity, and conversation length. The average cost is $6 per developer per day, with daily costs remaining below $12 for 90% of users.

For team usage, Claude Code charges by API token consumption. On average, Claude Code costs ~$100-200/developer per month with Sonnet 4.6 though there is large variance depending on how many instances users are running and whether they're using it in automation.

## Track your costs

### Using the `/cost` command

The `/cost` command provides detailed token usage statistics for your current session:

```
Total cost:            $0.55
Total duration (API):  6m 19.7s
Total duration (wall): 6h 33m 10.2s
Total code changes:    0 lines added, 0 lines removed
```

## Managing costs for teams

When using Claude API, you can set workspace spend limits on the total Claude Code workspace spend. Admins can view cost and usage reporting in the Console.

When you first authenticate Claude Code with your Claude Console account, a workspace called "Claude Code" is automatically created for you.

On Bedrock, Vertex, and Foundry, Claude Code does not send metrics from your cloud. To get cost metrics, several large enterprises reported using LiteLLM, which is an open-source tool that helps companies track spend by key.

### Rate limit recommendations

| Team size     | TPM per user | RPM per user |
| ------------- | ------------ | ------------ |
| 1-5 users     | 200k-300k    | 5-7          |
| 5-20 users    | 100k-150k    | 2.5-3.5      |
| 20-50 users   | 50k-75k      | 1.25-1.75    |
| 50-100 users  | 25k-35k      | 0.62-0.87    |
| 100-500 users | 15k-20k      | 0.37-0.47    |
| 500+ users    | 10k-15k      | 0.25-0.35    |

## Reduce token usage

### Offload processing to hooks and skills

Custom hooks can preprocess data before Claude sees it. Instead of Claude reading a 10,000-line log file to find errors, a hook can grep for ERROR and return only matching lines, reducing context from tens of thousands of tokens to hundreds.

Example PreToolUse hook for filtering test output:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "~/.claude/hooks/filter-test-output.sh"
          }
        ]
      }
    ]
  }
}
```

Filter script checks if the command is a test runner and modifies it to show only failures:

```bash
#!/bin/bash
input=$(cat)
cmd=$(echo "$input" | jq -r '.tool_input.command')

# If running tests, filter to show only failures
if [[ "$cmd" =~ ^(npm test|pytest|go test) ]]; then
  filtered_cmd="$cmd 2>&1 | grep -A 5 -E '(FAIL|ERROR|error:)' | head -100"
  echo "{\"hookSpecificOutput\":{\"hookEventName\":\"PreToolUse\",\"permissionDecision\":\"allow\",\"updatedInput\":{\"command\":\"$filtered_cmd\"}}}"
else
  echo "{}"
fi
```

## Session JSONL Data

Sessions are stored as JSONL files in `~/.claude/projects/<encoded-cwd>/*.jsonl` where `<encoded-cwd>` is the absolute working directory with every non-alphanumeric character replaced by a hyphen.

Each line with `message.usage` contains token counts: `input_tokens`, `cache_creation_input_tokens`, `cache_read_input_tokens`, and `output_tokens`.

## Background token usage

Claude Code uses tokens for some background functionality even when idle:
- Conversation summarization: Background jobs that summarize previous conversations for the `claude --resume` feature
- Command processing: Some commands like `/cost` may generate requests to check status
- These background processes consume a small amount of tokens (typically under $0.04 per session)
