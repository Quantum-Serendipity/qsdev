<!-- Source: https://raw.githubusercontent.com/lateos-ai/npm-scan/main/backend/policy.js -->
<!-- Retrieved: 2026-05-15 -->

# @lateos/npm-scan — backend/policy.js (Full Source)

Policy-as-code engine supporting YAML/JSON configuration files.

Key exports: loadPolicy, applyPolicy, isAllowed, getPackageReputationTier, matchesContext

## Features
- YAML/JSON policy loading with validation
- Package allowlists
- Severity overrides per ATK ID
- Suppress rules with context matching (file path glob, dist/build, test/fixture, lifecycle hook, safe domain, reputation tier)
- fail_on threshold enforcement
- Known reputable packages list (react, express, lodash, etc.)
- Lifecycle hook and multi-layer findings are unsuppressible (safety guard)

## Suppress rule matching
- Matches on atk_id (required), package (optional, default '*')
- Context-aware: is_dist_build, is_test_fixture, is_lifecycle_hook, is_known_safe_domain, file_path glob, url_domain pattern
- Reputation tier matching (trusted/unknown based on hardcoded KNOWN_REPUTABLE_PACKAGES set)

## Severity system
- Order: none < low < medium < high < critical
- fail_on checks if any finding >= threshold severity
- severity_overrides can change ATK finding severity

## Safety guards
- Cannot suppress lifecycle hook findings (is_lifecycle_hook = true)
- Cannot suppress multi-layer obfuscation findings (is_multi_layer = true)
