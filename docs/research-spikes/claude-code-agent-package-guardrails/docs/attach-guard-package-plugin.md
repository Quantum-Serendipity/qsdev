<!-- Source: https://dev.to/hammadtariq/i-built-a-claude-code-plugin-that-blocks-compromised-packages-before-installation-1o3l -->
<!-- Retrieved: 2026-05-12 -->

# attach-guard: Claude Code Plugin for Blocking Compromised Packages

## Core Functionality

The plugin intercepts package installation commands through Claude Code's PreToolUse hooks. As stated in the article: "Hooks run automatically on every matching tool call. Claude cannot skip or override them."

## How It Works

1. **Interception**: Captures the install command before execution
2. **Risk Assessment**: Scores the package via Socket.dev's supply chain API
3. **Policy Evaluation**: Denies installation if it fails security thresholds
4. **Smart Suggestions**: Proposes the newest safe version rather than simply blocking

## Detection Capabilities

- Known malware and compromised packages
- Newly published packages (less than 48 hours old)
- Low supply chain scores (below 50 = denied; 50-70 = flagged)
- Support across npm, pip, Go, and Cargo ecosystems

## Real-World Example

"npm install axios -> latest (1.14.1) scores 40/100 -> blocked -> rewrites to axios@1.14.0 (71/100)"

## Installation

Users execute two commands and provide a Socket.dev API token (free tier available). MIT licensed, no data collection.
