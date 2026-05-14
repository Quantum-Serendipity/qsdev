<!-- Source: https://raw.githubusercontent.com/openzim/sotoki/main/CONTRIBUTING.md -->
<!-- Retrieved: 2026-05-14 -->

# Sotoki Contribution Guidelines

## Overview
The project welcomes community participation: "Anybody is welcome to improve Sotoki."

## Required Setup Steps

1. **Environment Installation**: Install hatch, then execute `hatch shell` to establish a development environment with all necessary dependencies.

2. **Code Quality Tools**: After setup, run `hatch run pre-commit install` to activate automated enforcement of style standards including ruff, black, pyright, trailing whitespace cleanup, and EOF corrections.

## Development Workflow

**Testing**: Contributors should verify their work by running `hatch run pytest tests/`

**Manual Code Checks**: Execute `hatch run pre-commit run --all-files` to manually validate code quality at any point.

## Documentation Requirements

Any modifications affecting users must include an entry in the `[Unreleased]` section of `CHANGELOG.md`.

## Process Gaps

The provided documentation does not explicitly detail:
- Pull request submission procedures
- Code review expectations
- Specific contribution restrictions or approval workflows
- Branch naming conventions
- Commit message standards

The guidelines emphasize automated quality enforcement and testing but lack explicit instructions for the PR submission process itself.
