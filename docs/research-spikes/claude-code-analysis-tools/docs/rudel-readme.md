<!-- Source: https://github.com/obsessiondb/rudel -->
<!-- Retrieved: 2026-03-26 -->

# Rudel: Claude Code Session Analytics

## Overview

Rudel is an analytics platform designed for Claude Code sessions. The project provides a dashboard with insights on coding sessions, including token usage, duration, activity patterns, and model usage metrics.

## Key Features

- Session transcript analytics and processing
- Token usage tracking and visualization
- Session duration and activity pattern analysis
- Model usage insights
- Team collaboration with organization management
- Batch session upload capabilities

## Installation & Getting Started

**Prerequisites:** Bun runtime must be installed.

**Setup Process:**
1. Create account at app.rudel.ai
2. Install CLI: `npm install -g rudel`
3. Authenticate: `rudel login`
4. Enable auto-upload: `rudel enable`
5. Optionally invite teammates via Settings > Organization

For existing sessions, use: `rudel upload` for batch processing.

## Architecture & Data Flow

1. CLI installation triggering registration of Claude Code hooks
2. Hook activation upon session completion
3. Automatic transcript upload to Rudel servers
4. ClickHouse storage with analytics processing

## Data Collection Details

Uploaded sessions contain:
- Session identifiers and timestamps
- User and organization IDs
- Project paths and package information
- Git context (repository, branch, SHA, remote)
- Complete session transcripts
- Sub-agent usage metrics

## Security & Privacy Considerations

**Important Notice:** "Rudel is designed to ingest full coding-agent session data for analytics" including potentially sensitive materials -- source code, prompts, tool output, and file contents.

The hosted service implements encryption preventing direct data access. Limited product analytics tracks core workflows without session replay or blanket click capture. Users should review privacy policies before enabling uploads.

## Development & Self-Hosting

Self-hosting instructions provided in docs/self-hosting.md.

**License:** MIT
