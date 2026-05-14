<!-- Source: https://github.com/obsessiondb/rudel -->
<!-- Retrieved: 2026-03-26 -->

# Rudel - Claude Code Session Analytics

Analytics platform for Claude Code sessions. Dashboard displaying insights about coding sessions including token consumption, duration, activity trends, model selection, and more.

## Key Features

- **Session Analytics Dashboard**: Track token usage, session duration, activity patterns, model usage
- **CLI Tool**: Command-line interface for managing sessions and uploads
- **Automatic Uploads**: Registers Claude Code hook to auto-upload sessions on exit
- **Team Collaboration**: Invite teammates and share analytics within your organization
- **Batch Upload**: Upload past sessions in bulk

## Installation & Setup

Requires Bun runtime.

1. Create an account at the hosted service
2. Install CLI via npm and authenticate: `rudel login`
3. Enable automatic uploads: `rudel enable`
4. For existing sessions: `rudel upload` (interactive picker for batch upload)

## Data Collection

Each session upload includes: session identifiers and timestamps, user and organization IDs, project metadata and Git context, complete session transcripts, sub-agent usage information.

## Privacy & Security

Warns that "uploaded transcripts and related metadata may contain sensitive material, including source code, prompts, tool output, file contents, command output, URLs, and secrets."

## Technical Stack

Built with TypeScript (98.9%), Bun runtime, ClickHouse for transcript storage and analytics.
