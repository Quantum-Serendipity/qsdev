<!-- Source: https://docs.mantra.gonewx.com/features/time-travel -->
<!-- Retrieved: 2026-03-26 -->

# Mantra Time Travel Feature — Documentation

## Core Functionality

Time Travel enables users to "replay" an entire AI programming session like video playback. Click any conversation message to instantly see the code state at that moment.

## How Git Integration Works

### Snapshot Reconstruction Process

The system reconstructs code states through Git history analysis:

1. **Timestamp Matching**: Mantra reads timestamps from conversation records
2. **Commit Lookup**: The system identifies the closest Git commit preceding each message timestamp
3. **Code Extraction**: Code snapshots are extracted from Git history at that commit

### Technical Requirements

Projects must meet these conditions:
- Contain a `.git` directory (initialized Git repository)
- Have existing commit history with at least one prior commit
- Commit frequency directly affects precision — more frequent commits = more accurate snapshots

## Display Architecture

### Dual-Panel Layout

**Left Panel (Conversation)**:
- Complete dialogue chronology with timestamps
- Distinguishes message types (user queries, AI responses, code execution)
- Highlights current viewing position

**Right Panel (Code Snapshots)**:
- File tree with Git submodule nesting support
- Current code view at selected timepoint
- Diff mode (toggle via toolbar "Diff" button or `D` keyboard shortcut)
- Supports syntax highlighting and multi-file navigation

### TimberLine Timeline Control

A bottom-interface timeline includes:
- **Colored tick marks**: Blue circles (user messages), green squares (Git commits), transparent points (AI responses)
- **Navigation**: Drag slider, hover for timestamps, click to jump
- **Keyboard controls**: Arrow keys for 1% increments, Home/End for session boundaries

## Important Limitation

Code snapshots display the code state when the conversation occurred, not current local file contents. Displayed code reflects committed Git history, not unsaved local modifications.

## Activation Requirement

Time Travel displays only in Replay Mode; switching to Compact Mode hides the timeline for focused text editing.
