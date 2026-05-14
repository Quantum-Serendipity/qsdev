<!-- Source: https://docs.mantra.gonewx.com/features/replay -->
<!-- Retrieved: 2026-03-26 -->

# Mantra Replay Mode — Documentation

## Overview

Replay Mode (推演模式) allows users to "step through AI's operations in a safe environment" rather than simply viewing history.

## Key Distinction

Playback is purely visual viewing of history without affecting local files, while Replay involves real file operations aimed at reconstructing code state or verifying AI solutions.

## Core Features

### 1. Step-by-Step Preview
Default mode displays AI's reasoning on the left and "pending changes" on the right, allowing users to execute, skip, or review previous steps.

### 2. Default Workspace
Mantra automatically creates a default replay directory: `{app_data_dir}/replay/{session_id}/` — eliminating manual directory selection.

### 3. Autoplay & Speed Control
Users can enable automatic execution with three speed options: 1x, 2x, or 5x playback.

### 4. Fault Tolerance
If a step fails, the system records the failure reason and provides retry or recovery from the most recent stable checkpoint.

## Technical Characteristics

- **Deterministic replay**: Operates without calling the LLM, using only historical operation instructions
- Executes file creation, code modifications, and command execution
- Generates diff previews before applying changes
- Runs in isolated temporary directories by default to prevent modifying original projects

## Usage Workflow

Users click the Replay button, confirm workspace path, then click "Start Replay" to enter the step-through interface.
