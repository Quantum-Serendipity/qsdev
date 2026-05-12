<!-- Source: https://github.com/cachix/devenv/issues/1658 -->
<!-- Retrieved: 2026-05-12 -->

# Issue: Bypassing Cachix Binary Cache in Devenv

## The Core Question

User nalzok asked: "Do we have to rely on a cachix binary cache to use devenv?" They characterized "environment creation and binary caching" as separate concerns that shouldn't be inherently linked.

## Key Issue Details

The user raised two specific problems:

1. **Unclear relationship between devenv and cachix** - The user questioned whether binary caching is a necessary dependency for using devenv.

2. **Configuration setting ineffectiveness** - The user reported that setting `cachix.enable = false;` failed to suppress warnings, contradicting previous guidance they'd received.

## Current Status

The issue is closed with a "question" label. No resolution details about whether devenv can operate independently of binary caching infrastructure are visible in the discussion.

## Important Context

The issue references a prior discussion (Issue #1653) where similar concerns were raised about the dependency relationship between devenv's core functionality and the Cachix caching system.
