# reasoning-core rc_cli.py (Operator CLI)

- **Source**: https://github.com/jakubkrzysztofsikora/reasoning-core/blob/main/src/rc_cli.py
- **Retrieved**: 2026-05-15
- **Note**: Content returned via WebFetch AI summary — may not be verbatim

---

## Purpose
CLI tool for operators to inspect system status, audit decisions, manage bypasses, and control file-skipping behavior.

## Subcommands
- `rc status` — displays environment variables, kill-switch state, calibration metrics, and audit log statistics
- `rc explain <decision-id>` — retrieves full audit records for specific decisions
- `rc bypass-next` — arms a one-time bypass for the next hook invocation
- `rc skip-file <path>` — adds files to a per-session exclusion list
- `rc unskip-file <path>` — removes files from the exclusion list

## Environment Configuration
The system reads 23+ environment knobs controlling device settings, timeouts, thresholds, model backends, and operational modes (shadow mode, mock detection, etc.).

## Calibration Monitoring
Functions examine sentinel files tracking Mahalanobis model thresholds, CDGS gate metrics, and pending recalibration signals with age-tracking.

## Audit System
Maintains timestamped JSONL logs organized by date under a configurable root directory, with transparent gzip decompression support.

## Kill-Switch Integration
Reads shared state from `src/hooks/_kill_switches.py` for bypass flagging and file-skipping configuration.

Implementation uses argparse for command dispatch and includes proper path handling, JSON parsing with error recovery, and both text and compressed log file support.
