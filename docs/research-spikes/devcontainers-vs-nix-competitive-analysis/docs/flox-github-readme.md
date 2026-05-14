---
source: https://github.com/flox/flox
retrieved: 2026-03-20
---

# Flox Project Information Extract

## Overview
Flox is a package manager and virtual environment tool built on Nix that enables developers to create portable development environments across their software lifecycle.

## Core Features
Based on the README, Flox allows users to:
- Create reproducible development environments
- Search and install packages from nixpkgs (80,000+ packages)
- Share environments with collaborators
- Build container images from environments
- Layer and replace dependencies selectively

## Package Repository
Flox integrates with nixpkgs, described as "the biggest open source repository" containing over 80,000 packages available for installation.

## Quick Start Workflow
The README demonstrates basic usage:
```
flox init → flox search [package] → flox install [package] → flox activate
```

## Community & Support
- **Discourse Forum**: discourse.flox.dev for Q&A
- **Slack Community**: go.flox.dev/slack
- **Twitter**: @floxdevelopment
- **Documentation**: flox.dev/docs

## Project Statistics
- **Stars**: 3.8k
- **Forks**: 114
- **Contributors**: 48 active contributors
- **Commits**: 4,805 on main branch
- **Latest Release**: v1.10.0 (March 11, 2026)

## Technical Stack
Primary languages: Rust (78.2%), Shell (12.6%), Nix (3.6%), with C++, Makefile, and C components.

## Licensing & Contribution
Licensed under GPLv2. The project explicitly welcomes contributions with a dedicated CONTRIBUTING.md guide. Security vulnerabilities should be reported to security@flox.dev.
