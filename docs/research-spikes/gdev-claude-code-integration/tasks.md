# Tasks: gdev Claude Code Integration

## Phase 1: Scoping & Initial Research

### Pending

### Active

### Completed
- [x] **Scope confirmation** -- User provided explicit 6-area scope in initial request
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: Scope: skill format, command format, gdev operation mapping, CLI wrapper patterns, CLAUDE.md integration, safety considerations

## Phase 2: Research & Investigation

### Pending

### Active

### Completed
- [x] **Skill file format deep dive** -- Document SKILL.md format, frontmatter fields, string substitutions, dynamic context injection, supporting files, lifecycle
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: Full format documented from official docs. Key findings: skills replace commands, `!`command`` preprocessor, `allowed-tools` for pre-approval, `disable-model-invocation` for side-effect operations. See Section 1 of report.

- [x] **Command file format** -- Document legacy .claude/commands/ format and how it relates to skills
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: Commands are legacy, skills recommended. Same slash-command invocation. Skills win on name conflict. See Section 2 of report.

- [x] **gdev operations mapping** -- Map each gdev command to a skill with invocation model rationale
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: 10 operations mapped. 6 user-only (side effects), 4 Claude-invocable (read-only). Full SKILL.md examples for each. See Section 3 of report.

- [x] **CLI wrapper patterns** -- Research how other projects (Terraform, Docker, K8s) expose CLI tools to Claude Code
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: 5 patterns identified: knowledge+CLI, generator/validator pairs, dynamic state injection, multi-step workflow, script bundling. 1 anti-pattern (knowledge-only). See Section 4 of report.

- [x] **CLAUDE.md integration design** -- How to make Claude Code aware of gdev capabilities via CLAUDE.md
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: CLAUDE.md for quick reference + security policy; skills for detailed workflows. Section markers for safe updates. @-import for larger docs. See Section 5 of report.

- [x] **Safety considerations** -- Autonomous vs human-confirmed operations, permission models, sandboxing
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: 5-layer safety architecture. Read-only ops = autonomous, side-effect ops = user-confirmed. Dry-run pattern, enterprise managed settings, edge cases documented. See Section 6 of report.

## Phase 3: Synthesis & Review

### Pending

### Active

### Completed
- [x] **Depth checklist review** -- Run revision cycle on the research report
  - Outcome: success
  - Completed: 2026-05-12
  - Notes: All 6 depth checklist items pass. Report is comprehensive and standalone-readable.
