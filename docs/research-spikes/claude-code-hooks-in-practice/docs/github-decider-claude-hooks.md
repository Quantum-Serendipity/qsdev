# Claude Hooks (Clean Code) - decider/claude-hooks
- **Source**: https://github.com/decider/claude-hooks
- **Retrieved**: 2026-03-27

## Overview
A lightweight Python-based hook system for Claude Code that provides automatic validation and quality checks. Integrates with Claude's workflow to enforce standards during development sessions.

## Hook Entry Points

1. **PreToolUse** - Executes before tools (Bash, Write, Edit) run
2. **PostToolUse** - Executes after tool completion
3. **Stop** - Executes when Claude finishes or stops a task

## Available Hooks

### 1. Code Quality Validator
- **Event:** PostToolUse
- **Purpose:** Enforces clean code standards on file edits
- **Enforced constraints:**
  - Maximum function length: 30 lines
  - Maximum file length: 200 lines
  - Maximum line length: 100 characters
  - Maximum nesting depth: 4 levels

### 2. Package Age Checker
- **Event:** PreToolUse
- **Purpose:** Prevents installation of outdated npm/yarn packages
- **Behavior:**
  - Blocks packages older than 180 days (configurable)
  - Displays latest available versions
  - Triggers on `npm install` and `yarn add` commands

### 3. Task Completion Notifier
- **Event:** Stop
- **Purpose:** Sends notifications upon task completion
- **Supported channels:**
  - Pushover (mobile push notifications)
  - macOS native notifications
  - Linux desktop notifications

## Configuration System

### Hierarchical Structure
- **Root:** `.claude/hooks.json` (project-wide defaults)
- **Directory-level:** `.claude-hooks.json` (directory overrides)
- **Inheritance:** Child directories inherit parent settings with override capability

### Environment Variables
- `MAX_AGE_DAYS` - Package age threshold (default: 180)
- `CLAUDE_HOOKS_TEST_MODE` - Enables test functionality
- `PUSHOVER_USER_KEY` - Pushover authentication
- `PUSHOVER_APP_TOKEN` - Pushover application credential

## Architecture

**Dispatcher Pattern:**
- `universal-*.py` files route events to specific handlers
- Individual hook scripts contain functionality logic
- JSON stdin/stdout for event communication

## Installation
Single-command setup: `python3 install-hooks.py`
Creates `.claude/` directory structure, copies hook scripts, generates configuration, adds `.claude/settings.local.json` to `.gitignore`.
