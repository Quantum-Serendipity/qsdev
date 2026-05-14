# Learning Opportunities — Claude Code / Codex Skill Plugin
- **Source**: https://github.com/DrCatHicks/learning-opportunities
- **Retrieved**: 2026-05-14

## What This Project Is

Learning Opportunities is a skill/plugin system for Claude Code and Codex that integrates evidence-based learning exercises into AI-assisted software development workflows. It prompts developers to engage in deliberate skill-building activities after completing significant architectural work.

## Core Methodology

The project addresses five learning science risks created by rapid AI-assisted coding:

1. **Generation Effect**: Accepting generated code reduces active processing that builds understanding
2. **Fluency Illusion**: Clean code can feel more understood than truly comprehended
3. **Spacing Effect**: High velocity pushes constant work without reflection cadence
4. **Metacognition Gap**: Fast workflows lack room for self-assessment and expertise tracking
5. **Testing Deficit**: Agentic models providing complete answers reduce retrieval practice benefits

The skill counteracts these by introducing active generation, retrieval practice, deliberate pauses, and explicit metacognition.

## Exercise Types Implemented

- Prediction → Observation → Reflection (expect outcomes, observe results, analyze surprises)
- Generation → Comparison (sketch approaches before seeing implementations)
- Trace the Path (step-through execution with predictions)
- Debug This (identify failure points and causes)
- Teach It Back (explain components as if onboarding developers)
- Retrieval Check-in (recall previous session learning)

## Trigger Conditions

Claude offers 10-15 minute exercises after:
- Creating new files or modules
- Database schema changes
- Architectural decisions or refactors
- Implementing unfamiliar patterns
- Moments where developers asked "why" questions

**Suppression conditions**: No prompts if user declined earlier in session or completed 2 exercises already.

## Installation & Integration

**Codex Users**:
```
codex plugin marketplace add https://github.com/DrCatHicks/learning-opportunities.git
```

**Claude Code Users**:
```
/plugin marketplace add https://github.com/DrCatHicks/learning-opportunities.git
/plugin install learning-opportunities@learning-opportunities
```

Optional companion plugins:
- `learning-opportunities-auto`: Post-commit prompting hook (Linux/macOS/Windows)
- `orient`: Repository orientation lesson generator using codebase sampling strategies

## Research Foundation

Project grounded in peer-reviewed learning science literature including work on:
- Spaced repetition and spacing effects (Kang, 2016; Kornell, 2009)
- Testing effect and retrieval practice (Roediger & Karpicke, 2006)
- Expertise and deliberate practice (Ericsson et al., 2018)
- Digital cognitive load (Skulmowski & Xu, 2022)
- Metacognitive demands of generative AI (Tankelevitch et al., 2024)

Creator's empirical research with thousands of developers found that strong learning commitment predicts lower AI-anxiety and higher team effectiveness.

## Measurement Framework

**MEASURE-THIS.md** provides:
- Validated survey items on developer thriving and AI skill threat (peer-reviewed, open access under CC-BY-SA 4.0)
- Guidance on variance analysis and measurement interpretation
- "Team boast" template for communicating experiment results to leadership
- Claude-assisted statistical rigor guardrails

Sources: OSF preprints on AI Skill Threat (2gej5_v2) and Developer Thriving IEEE supplement.

## Customization Options

- Input existing technical expertise and learning goals
- Adjust exercise triggers for specific workflows
- Set exercise-per-session caps
- Add project-specific examples
- Include insights in project Claude.md
- Create domain-specific retrieval questions

## Key Design Principle

Claude intentionally pauses and waits for user input rather than providing complete answers, counteracting default agentic behavior to encourage active mental effort and deeper learning.

## Project Background

Developed by Dr. Cat Hicks (psychological scientist studying software teams) and Dr. Michael Mullarkey (ML engineer). Licensed under CC-BY 4.0. Connects to broader research on developer psychology, team effectiveness, and technology work transitions documented in upcoming book "The Psychology of Software Teams" (2026).
