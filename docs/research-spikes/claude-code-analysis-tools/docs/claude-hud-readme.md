<!-- Source: https://github.com/jarrodwatts/claude-hud -->
<!-- Retrieved: 2026-03-26 -->

# Claude HUD: A Claude Code Status Plugin

## Overview

Claude HUD is a plugin for Claude Code that displays real-time session information in your terminal's status line. Uses Claude Code's native statusline API -- no separate window or tmux needed.

## Core Features

Monitors:
- **Context window usage** with visual bar (green -> yellow -> red)
- **Rate limit consumption** for Claude subscribers
- **Active tools** (file reads, edits, searches)
- **Running agents** and their current tasks
- **Todo progress** tracking
- **Project path** (configurable 1-3 directory levels)
- **Git status** including branch and uncommitted changes

## Installation

Three-step setup inside Claude Code:
1. `/plugin marketplace add jarrodwatts/claude-hud`
2. `/plugin install claude-hud`
3. `/claude-hud:setup` to configure the statusline

## Display Layouts

Default two-line view:
- Line 1: Model name, project path, git branch
- Line 2: Context bar with usage limits

Optional expanded mode adds activity lines for tools, agents, and todos.

## Customization

Run `/claude-hud:configure` for interactive setup, or edit `~/.claude/plugins/claude-hud/config.json`. Available customizations:
- Layout mode (expanded vs compact)
- Directory levels in path
- Element visibility toggles
- Color schemes (256-color and hex support)
- Git display options

## Requirements

- Claude Code v1.0.80+
- Node.js 18+ or Bun

Updates every ~300ms by parsing transcript JSONL file for tools, agents, and todos.
