<!-- Source: https://github.com/Brads3290/cclogviewer -->
<!-- Retrieved: 2026-03-26 -->

# cclogviewer

Converts Claude Code JSONL log files into interactive HTML for easy review and analysis.

## Key Features

- Hierarchical conversation display with expandable sections
- Tool calls and results visualization
- Nested Task tool conversation support
- Token usage tracking and metrics
- Syntax-highlighted code blocks
- Timestamps and role indicators

## Installation

Requires Go 1.21+.

```bash
go install github.com/brads3290/cclogviewer/cmd/cclogviewer@latest
```

## Usage

```bash
cclogviewer -input session.jsonl              # auto-opens in browser
cclogviewer -input session.jsonl -output out.html  # save to file
```

**License:** MIT
