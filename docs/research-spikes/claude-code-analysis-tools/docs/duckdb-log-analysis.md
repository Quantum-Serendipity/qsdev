# Analyzing Claude Code Interaction Logs with DuckDB

- **Source**: https://liambx.com/blog/claude-code-log-analysis-with-duckdb
- **Retrieved**: 2026-03-26
- **Note**: Content was AI-summarized by WebFetch.

## File Structure and Location
Claude Code interaction logs are stored in `~/.claude/projects/` within project-specific directories as JSONL files, where "each line is a JSON object representing a single event."

## Core Field Names
Based on the documented schema, key fields include:
- `parentUuid`, `uuid`, `sessionId` - identifiers
- `type` - event classification (user, assistant, summary)
- `timestamp` - ISO 8601 format
- `message` - content object
- `cwd` - current working directory
- `userType`, `isSidechain`, `version` - metadata

## Message Types
Three primary event types:
- **user**: User utterances and commands
- **assistant**: AI responses and tool invocations
- **summary**: Session summaries

## Assistant Message Structure
Assistant messages contain:
- `id` - message identifier
- `role` - "assistant"
- `model` - Claude model version
- `content` - array of content blocks
- `usage` - token metrics
- `stop_reason`, `stop_sequence` - termination indicators

## Token Usage Fields
The `usage` object tracks:
- `input_tokens`
- `cache_creation_input_tokens`
- `cache_read_input_tokens`
- `output_tokens`
- `service_tier`

## DuckDB Query Approach
Load JSONL directly: `CREATE TABLE conversation AS SELECT * FROM read_json_auto('filename.jsonl')`

Then query using standard SQL with JSON extraction functions for nested message analysis.
