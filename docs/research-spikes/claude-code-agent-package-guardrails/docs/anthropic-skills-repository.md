---
source: https://github.com/anthropics/skills
retrieved: 2026-05-12
---

# Anthropic Skills Repository

133k stars, 15.7k forks, Python (84.4%), HTML (12.4%), Shell (1.9%), JavaScript (1.3%). Public, 34 commits on main branch.

## Directory Structure

```
anthropics/skills/
├── .claude-plugin/          # Claude plugin configuration
├── skills/                  # Main skills folder
│   ├── creative & design
│   ├── development & technical
│   ├── enterprise & communication
│   ├── document skills
│   ├── docx/               # Document creation (source-available)
│   ├── pdf/                # PDF handling (source-available)
│   ├── pptx/               # PowerPoint creation (source-available)
│   └── xlsx/               # Excel creation (source-available)
├── spec/                    # Agent Skills specification
├── template/                # Skill template
├── README.md
└── THIRD_PARTY_NOTICES.md
```

## Basic Skill Structure

```markdown
---
name: my-skill-name
description: A clear description of what this skill does and when to use it
---

# My Skill Name

[Instructions that Claude will follow when this skill is active]
```

## Skill Categories

1. Creative & Design Skills
2. Development & Technical Skills
3. Enterprise & Communication Skills
4. Document Skills (docx, pdf, pptx, xlsx)

## Access Methods (Claude Code)

```bash
/plugin marketplace add anthropics/skills
/plugin install document-skills@anthropic-agent-skills
/plugin install example-skills@anthropic-agent-skills
```

## Notes

- Most skills: Apache 2.0
- Document skills: source-available (not open source)
- No security-focused skills in the official repository
- Specification defined in ./spec (Agent Skills standard at agentskills.io)
