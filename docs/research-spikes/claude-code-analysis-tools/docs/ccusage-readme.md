<!-- Source: https://github.com/ryoppippi/ccusage -->
<!-- Retrieved: 2026-03-26 -->

# ccusage - Claude Code Usage Analyzer

## Repository Overview

**ccusage** is a CLI tool that analyzes Claude Code token usage and costs from local JSONL files. The project includes a family of companion tools for different AI platforms.

## Key Features

**Reporting Capabilities:**
- Daily, monthly, and session-based usage aggregation
- 5-hour billing window tracking with active block monitoring
- Per-model cost breakdown showing which Claude versions were used
- Compact status line integration for IDE hooks (Beta)

**Data & Output:**
- JSON export functionality for structured data
- USD cost calculations per period
- Separate cache creation and cache read token tracking
- Offline mode using pre-cached pricing data

**Customization & Integration:**
- Date range filtering via `--since` and `--until` flags
- Timezone and locale support for localized reporting
- Multi-instance grouping by project with filtering
- Configuration files with IDE autocomplete validation
- Model Context Protocol (MCP) server integration

**User Experience:**
- Responsive table layouts that auto-adapt to terminal width
- Forced compact mode for screenshots and sharing
- Model names displayed as bulleted lists

## Installation

Run directly without installation:
```bash
npx ccusage@latest
```

Alternative runners: `bunx`, `pnpm dlx`, or `deno run` with specific flags.

## Usage Examples

```bash
npx ccusage daily              # Daily report
npx ccusage monthly            # Monthly aggregation
npx ccusage session            # Session-based view
npx ccusage --compact          # Force compact layout
npx ccusage daily --json       # JSON output
npx ccusage daily --breakdown  # Model cost breakdown
```

## Companion Tools

- **@ccusage/codex** - OpenAI Codex usage analyzer
- **@ccusage/opencode** - OpenCode usage tracking
- **@ccusage/pi** - Pi-agent session analyzer
- **@ccusage/amp** - Amp CLI session tracker
- **@ccusage/mcp** - MCP server integration

## Technical Details

- **Language:** TypeScript (99.2%)
- **Bundle Size:** Exceptionally small with "extreme attention to bundle size"
- **Development:** Nix flake-based environment with direnv support
- **License:** MIT
- **Repository Stats:** 12k stars, 438 forks, 108 releases, 62 contributors

Full documentation available at **ccusage.com**
