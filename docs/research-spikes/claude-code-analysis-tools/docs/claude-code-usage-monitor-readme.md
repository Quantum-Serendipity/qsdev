<!-- Source: https://github.com/Maciek-roboblog/Claude-Code-Usage-Monitor -->
<!-- Retrieved: 2026-03-26 -->

# Claude Code Usage Monitor

Real-time terminal application for tracking Claude AI token consumption with advanced analytics and machine learning-based predictions.

## Key Features

- Real-time monitoring with configurable refresh rates (0.1-20 Hz)
- Machine learning-powered P90 percentile predictions for intelligent limit detection
- Multi-view options: realtime, daily, and monthly usage aggregation
- Cost analytics with model-specific pricing and cache token calculations
- WCAG-compliant Rich UI with automatic terminal background detection
- Smart auto-detection for plan switching with custom limit discovery

## Plans Supported

- **Pro**: ~19,000 tokens
- **Max5**: ~88,000 tokens
- **Max20**: ~220,000 tokens
- **Custom**: P90-based auto-detection (default)

## Installation

**Recommended:**
```bash
uv tool install claude-monitor
```

**Alternatives:** pip, pipx, conda/mamba, from source.

## Command Aliases

`claude-monitor` (primary), `cmonitor`, `ccmonitor`, `ccm` (shortcuts)

## Usage

```bash
claude-monitor --plan pro --theme dark
claude-monitor --plan custom --refresh-rate 5
claude-monitor --timezone America/New_York
```

## Advanced Features

- Configuration persistence to `~/.claude-monitor/last_used.json`
- ML analysis of usage patterns from last 192 hours (8 days)
- Modular design with Pydantic-based type-safe configuration
- 100+ comprehensive test cases
- Optional Sentry integration for production monitoring
