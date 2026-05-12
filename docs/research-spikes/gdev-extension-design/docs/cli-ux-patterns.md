<!-- Source: https://medium.com/@kaushalsinh73/top-8-cli-ux-patterns-users-will-brag-about-4427adb548b7 -->
<!-- Retrieved: 2026-05-12 -->
<!-- Note: Only partial content was extractable from this source. The full article covers 8 patterns but only the first was fully captured. -->

# CLI UX Patterns

Based on the Medium article by Neurobyte (Sep 2025), presenting eight patterns designed to make CLIs feel "fast, obvious, and trustworthy."

## Overview
"Great CLIs don't shout; they _anticipate_."

## Pattern 1: First-Run Wizard with Sensible Defaults

The initial user experience should include guided setup that generates configuration automatically. Rather than overwhelming users with questions, the approach uses a few important prompts paired with reasonable preset values and an easy exit option.

**Key principle:** New users succeed on their first attempt, while experienced users can bypass steps using command flags.

**Example flow:**
- Auto-detect repository and authentication details
- Ask for project name and cloud region with bracketed defaults
- Confirm to create the config file
- Write a config that users can tweak later

**Summary:** "Not a questionnaire, but just a few high-signal prompts with safe defaults and a clear escape hatch."

## Other Patterns (titles only, content not fully extractable)

2. **Helpful Help** — Help text that teaches, not just lists flags
3. **Dry-Run with Diff** — Show what would change before doing it
4. **Idempotent Retries** — Safe to re-run without side effects
5. **Structured Output** — Machine-readable output alongside human-readable
6. **Smart Errors** — Errors that suggest fixes
7. **Honest Progress** — Accurate progress indication
8. **Shell Completion** — Tab completion for commands and arguments
