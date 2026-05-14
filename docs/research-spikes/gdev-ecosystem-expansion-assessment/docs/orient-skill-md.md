# Orient — SKILL.md (Full Content)
- **Source**: https://github.com/DrCatHicks/learning-opportunities/blob/main/orient/skills/orient/SKILL.md
- **Retrieved**: 2026-05-14
- **Method**: gh api (raw content via base64 decode)

---
name: orient
description: Generates a repo-specific orientation.md resource for the learning-opportunities skill. Invoke directly when the user asks for repo orientation; do not trigger automatically.
argument-hint: "[showboat]"
disable-model-invocation: true
allowed-tools: Read, Glob, Grep, Bash, Write
---

Key structural observations:
- Uses `disable-model-invocation: true` — must be explicitly invoked via `/orient`
- Restricts tools to: Read, Glob, Grep, Bash, Write (no web access)
- Two modes: default (structured comprehension-based exploration) and showboat (uses `uvx showboat` CLI)
- Writes output to `.claude/skills/learning-opportunities/resources/orientation.md`
- 5-step process: find write path, detect languages, explore repo (6 sub-steps), synthesize, confirm
- Exploration methodology grounded in program comprehension research (Spinellis 2003, Hermans 2021, Storey 2006)
- Generates exactly 2 orientation exercises that direct learners to read specific artifacts then synthesize
- orient-bibliography.md provides full academic citations for the methodology
