# Rudel: Claude Code Session Analytics
- **Source**: https://github.com/obsessiondb/rudel
- **Retrieved**: 2026-03-27

## Hook Mechanism
Rudel registers a Claude Code hook that runs when a session ends to automatically capture and upload session data. Uses Claude Code's native hook infrastructure rather than manual intervention.

## Hooked Events
The hook captures a complete session transcript when Claude Code exits, including:
- Session metadata (ID, timestamps for start and last interaction)
- User and organization identifiers
- Project context (path, package name, git repository details)
- Full conversation content (prompts and responses)
- Sub-agent usage metrics

## Configuration
Setup involves three commands:
1. `rudel login` — browser-based authentication
2. `rudel enable` — activates automatic session uploads
3. `rudel upload` — optional batch upload of historical sessions

The CLI integrates with the Claude Code environment to register hooks automatically.

## Data Upload Flow
Sessions are transmitted to Rudel's backend when Claude Code terminates. The platform stores transcripts in ClickHouse and processes them into actionable analytics.

## Analytics Provided
Dashboard offers insights on:
- Token consumption
- Session duration
- Activity timing patterns
- Model selection data
- Team collaboration metrics through organization features

## Key Limitations & Concerns
**Critical:** "Uploaded transcripts and related metadata may contain sensitive material, including source code, prompts, tool output, file contents, command output, URLs, and secrets." Users must consciously enable this only on appropriate projects.

The service implements intentionally limited product analytics (no autocapture or session replay) but users should review the privacy policy before deployment.
