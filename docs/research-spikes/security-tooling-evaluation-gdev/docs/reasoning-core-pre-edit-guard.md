# reasoning-core pre_edit_guard.py

- **Source**: https://github.com/jakubkrzysztofsikora/reasoning-core/blob/main/src/hooks/pre_edit_guard.py
- **Retrieved**: 2026-05-15
- **Note**: Content returned via WebFetch AI summary — may not be verbatim

---

## Core Function

The hook validates code changes against a local sidecar service before allowing edits, implementing multiple security layers.

## Key Features

**Sidecar Integration**: Calls `POST http://127.0.0.1:8765/score` to evaluate changes for regressions. Returns exit code 0 (allowed) or 2 (blocked).

**Multi-Layer Gating System**:
1. Guard file protection — prevents modifications to hook scripts, settings, or sidecar code unless `RC_ALLOW_GUARD_EDIT=1`
2. Override mechanisms — kill switches and magic comments (`# rc:skip`) bypass scoring
3. Language fingerprinting — ensures edits stay within declared language families when `RC_LANG_LOCK=1`
4. Plan grounding validation — couples implementation to documented plans (iter-3)

**Change Extraction**:
- Edit: reconstructs post-edit state by applying old_string->new_string replacement
- Write: compares new content against disk state
- MultiEdit: applies sequential edits in-memory, scores final result vs. original

**Fail Modes**:
- Fail-open (default): allows edits if sidecar unreachable
- Fail-closed: blocks edits on sidecar unavailability when `S2_FAIL_CLOSED=1`
- Shadow mode: logs blocks without enforcing when `RC_SHADOW_MODE=1`

**Audit Logging**: Tracks all decisions (allowed/blocked/shadow_blocked) with signals including regression status, latency, cumulative drift, and risk vectors.

**Additional Gates**:
- Cumulative drift thresholds (warn at 4.0, deny at 6.0 by default)
- Mock detector for test code (RC_MOCK_DETECTOR=1)
- Mahalanobis calibration scoring (RC_CALIBRATION_ENABLED=1, P7 phase)

## Dependencies

Uses only Python stdlib (`urllib`, `json`, `pathlib`). Imports local hook modules (audit_log, _guard_paths, _dispatch, etc.) for extension and customization.
