<!-- Source: https://github.com/carlosarraes/ccost -->
<!-- Retrieved: 2026-03-26 -->

# ccost - Claude API Cost Tracking Tool

Self-contained Rust-based CLI tool for analyzing Claude API usage and costs with intelligent message deduplication, multi-currency support, and real-time pricing via LiteLLM.

## Key Features

- Live pricing integration with LiteLLM
- Dual caching system (24-hour persistent for currency rates and model pricing)
- Enhanced deduplication using requestId priority with sessionId fallback
- Multi-currency support with live ECB exchange rates (EUR, GBP, JPY, CNY, BRL, 400+)
- Zero runtime dependencies -- single binary, no Node.js needed
- Project filtering with comma-separated support

## Installation

```bash
curl -sSf https://raw.githubusercontent.com/carlosarraes/ccost/main/install.sh | sh
```

## Usage

```bash
ccost              # Overall usage summary
ccost today        # Today's costs
ccost this-week    # Weekly analysis
ccost daily --days 7  # Daily breakdown
ccost projects proj1,proj2  # Specific projects
ccost today --currency EUR  # Convert to EUR
```

## Architecture

Billing-aligned deduplication: requestId primary -> sessionId fallback. SQLite persistence for offline operation and sub-second responses. Created as Node.js-free alternative to ccusage.
