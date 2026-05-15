<!-- Source: GitHub API (repos/falcosecurity/prempti) -->
<!-- Retrieved: 2026-05-15 -->

# Prempti GitHub Repository Metadata

- **Full name**: falcosecurity/prempti
- **Description**: Falco-powered policy and visibility layer for AI coding agents
- **Language**: Rust
- **License**: Apache-2.0
- **Created**: 2026-03-18
- **Last updated**: 2026-05-15
- **Last push**: 2026-05-15
- **Stars**: 42
- **Forks**: 10
- **Open issues**: 1
- **Default branch**: main
- **Archived**: No

## Contributors (4 total)
- leogr: 117 commits (primary author, Leonardo Grasso)
- c2ndev: 33 commits
- ekoops: 1 commit
- ldegio: 1 commit

## Release History
- v0.3.0 — 2026-05-13 (latest, in Cargo.toml)
- v0.2.1 — 2026-05-12
- v0.2.1-rc1 — 2026-05-07
- v0.2.0 — 2026-05-04
- v0.2.0-rc1 — 2026-04-30
- v0.1.0 — 2026-03-20 (initial release)

## Recent Commits (as of 2026-05-15)
- refactor(ctl): use rustix::termios::tcgetwinsize instead of ioctl
- refactor(ctl): use rustix::process for kill and process-alive checks
- refactor(interceptor): use rustix::process::getppid instead of unsafe FFI
- docs(plugin): record registered source ID 28 in SPEC
- fix(plugin): set registered source ID 28 for coding_agent
- chore: release v0.3.0
- refactor(plugin,interceptor): align agent_pid wire type to u64
- refactor(plugin): return DecodedPayload struct instead of 4-tuple
- refactor(plugin): extract NEWLINE_COUNT const for payload separator count
- test(e2e): assert agent.pid matches test process PID

## Repo Structure
Root: .cargo, .claude-plugin, .github, .gitignore, CLAUDE.md, Cargo.lock, Cargo.toml, LICENSE, Makefile, OWNERS, README.md, configs, demo.gif, docs, hooks, installers, plugins, rules, skills, tests, tools
