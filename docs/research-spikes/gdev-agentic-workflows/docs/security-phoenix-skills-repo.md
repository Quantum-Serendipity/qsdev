<!-- Source: https://github.com/Security-Phoenix-demo/security-skills-claude-code -->
<!-- Retrieved: 2026-05-12 -->

# Security Phoenix Skills for Claude Code

MIT license. Open-source security automation toolkit.

## Core Skills
- CTI Domain Research: 595+ curated security sources, authority-ranked, tier system
- Secure PRD Generator: Security-focused product requirements with STRIDE threat modeling
- OpenGrep Rule Generator: SAST rules for 30+ languages with FP reduction
- OpenGrep Rule Generator Research: CVE/CWE research then auto-generate detection rules
- NotebookLM Connector: Citation-backed research via Google NotebookLM
- Global Research Pipeline: Systematic web/video research with NotebookLM ingestion
- Project Documentation: Auto-generate project docs with architecture maps
- Security Assessment Suite: /security-0day, /security-review, /security-assessment, /threatmodel

## Security Assessment Suite
- /security-0day [base-ref]: End-of-cycle diff scanning (~$0.05-$0.20)
- /security-review [scope]: Pre-merge endpoint/auth/render checks
- /security-assessment [scope]: Full OWASP Top 10 + ASVS Level 1 (~$8-$10)
- /threatmodel [scope]: STRIDE + DREAD threat modeling

## Active Hooks (full install)
- SessionStart: Fingerprints project, runs dependency audit
- PreToolUse on Bash: Gates package manager invocations, blocks malicious packages
- PostToolUse on Edit: Pattern scans for SQL injection, XSS, hardcoded secrets
- SessionEnd: Reminds to run /security-0day for unscanned changes

## Phoenix Pipeline (12-Role Feature Descriptor)
Context Curator, Scope Cutter, Constraint Distiller, Requirements Engineer, Ambiguity Hunter, Security Engineer, Contract Architect, Verification Matrix, Batch Planner, Final Gate, Pipeline Navigator, Orchestrator

## Domain Tier System
- Tier 1 (Authoritative): CISA, NVD, MSRC, NCSC, Red Hat
- Tier 2 (Vendor Research): Unit42, Talos, Securelist, DFIR Report, Mandiant
- Tier 3 (News/Community): BleepingComputer, Krebs, Hacker News
- Tier 4 (OSINT/PoC): any.run, VulnCheck, AttackerKB, GreyNoise

## Installation
- Claude Marketplace search
- git clone + bash install.sh
- Direct copy to ~/.claude/skills/
