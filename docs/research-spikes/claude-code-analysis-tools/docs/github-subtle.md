<!-- Source: https://github.com/itsderek23/subtle -->
<!-- Retrieved: 2026-03-26 -->

# Subtle - Claude Code Session Analytics

Local, privacy-focused web application for exploring and analyzing Claude Code session logs. All data processing happens locally, no telemetry.

## Features

- Claude Code usage visualization over time (AI vs tool time)
- AI-assisted Git commit tracking
- Individual session execution traces
- Session filtering by message content
- Complete session transcripts

## Installation

```bash
pip install subtle-claude-code
# or
uv add subtle-claude-code
```

## Usage

```bash
subtle start                    # launches at http://127.0.0.1:8000
subtle start --port 3000        # custom port
```

## Requirements

Python 3.10+, Claude Code session logs in ~/.claude/projects/

## License
MIT
