<!-- Source: https://mise.jdx.dev/ -->
<!-- Retrieved: 2026-05-12 -->

# mise Configuration Model Overview

## Configuration File
The primary configuration file is **mise.toml**, which serves as a centralized hub for project setup. The documentation shows it contains tool declarations, environment variables, and task definitions in one location.

## Configuration Sharing & Hierarchy
mise operates with a directory-based activation model: when developers navigate into a project directory, "mise picks up mise.toml and updates the shell." This suggests a project-level configuration that automatically activates when relevant. The documentation mentions loading env vars from "mise.toml, .env files, shell commands, and more," implying a hierarchy exists.

Key hierarchy elements:
- `.mise.toml` - project-level config (checked into git)
- `.mise.local.toml` - local overrides (gitignored)
- `.env` - shared environment variables
- `.env.local` - local secrets (gitignored)

## Version Pinning
Tool versions are pinned declaratively. The example shows: `mise use node@24 python@3.13 terraform@1` creating a mise.toml file, followed by `mise install` to activate those versions. This approach ensures "reproducibly" managing tools across teams.

## Task Definitions
Tasks are defined "next to the tools and env vars they need." The demo shows multi-step task execution like `mise run deploy` running a 4-step pipeline (build, test, migrate, ship).

## Onboarding Workflow
When cloning a project, developers run `mise install` after mise detects the existing mise.toml, which "installs" the pre-configured tools automatically, streamlining team onboarding.
