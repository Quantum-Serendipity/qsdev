<!-- Source: https://raw.githubusercontent.com/falcosecurity/prempti/main/rules/README.md -->
<!-- Retrieved: 2026-05-15 -->

# Falco Rules for Prempti: Overview

## Purpose & Structure

Falco rules govern security policies for coding agents in Prempti. The system organizes rules across three key locations: `default/coding_agents_rules.yaml` (shipped defaults), `user/` (custom rules preserved during upgrades), and `seen.yaml` (mandatory catch-all rule).

## Key Components

**Default Ruleset**: Addresses seven security domains including working-directory boundaries, sensitive paths, sandbox integrity, credential threats, dangerous commands, exfiltration prevention, supply chain risks, MCP/skill content, persistence vectors, and self-protection mechanisms. Provides reusable lists and macros for extension.

**User Rules Directory**: Developers place customizations here. The default file gets overwritten during upgrades.

**Seen Rule**: Mandatory catch-all that fires for every agent event and signals evaluation completion to the plugin broker. "Do not remove or modify this file."

## Verdict System

Three tags communicate decisions:
- `coding_agent_deny` blocks tool calls
- `coding_agent_ask` requires user confirmation
- `coding_agent_seen` signals evaluation completion

When multiple rules match, escalation applies: denial overrides requests for confirmation.

## Rule Authoring Requirements

All rules must specify `source: coding_agent` and use appropriate verdict tags. The output field should be "an LLM-friendly sentence explaining what happened and why," starting with "Falco" and avoiding jargon.

Available fields include correlation ID, agent identity information (name, OS, PID, session details), working directory data, tool specifications, and file paths in both raw and resolved formats.

## Best Practices

- Use `val()` for field comparisons
- Use `basename()` for filename matching
- Use `real_*` fields for policy enforcement, raw fields for audit
