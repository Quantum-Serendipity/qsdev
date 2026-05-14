# Learning Opportunities — SKILL.md (Full Content)
- **Source**: https://github.com/DrCatHicks/learning-opportunities/blob/main/learning-opportunities/skills/learning-opportunities/SKILL.md
- **Retrieved**: 2026-05-14
- **Method**: gh api (raw content via base64 decode)

---
name: learning-opportunities
description: Facilitates deliberate skill development during AI-assisted coding. Offers interactive learning exercises after architectural work (new files, schema changes, refactors). Use when completing features, making design decisions, or when user asks to understand code better. Supports the user's stated goal of understanding design choices as learning opportunities.
argument-hint: "[orient]"
license: CC-BY-4.0
---

[Full content saved separately — see learning-opportunities-research.md for analysis. The SKILL.md is ~400 lines covering: Purpose, When to offer exercises, When not to offer, Scope, Core principle (pause for input), Exercise types (6 types), Techniques to weave in, Hands-on code exploration, Facilitation guidelines, and Orientation mode.]

Key structural observations:
- Uses Claude Code YAML frontmatter format (name, description, argument-hint, license)
- No `disable-model-invocation` — Claude CAN invoke this autonomously
- No `allowed-tools` restriction — uses default Claude Code tool set
- The `argument-hint: "[orient]"` enables `/learning-opportunities orient` to trigger orientation mode
- References PRINCIPLES.md via relative link to resources/ directory
- Orientation mode looks for orientation.md at `.claude/skills/learning-opportunities/resources/orientation.md`
