# Consulting-Specific Workflow Differentiation

## Executive Summary

Consulting software engineering has structural differences from product development that fundamentally affect how agentic workflows should be designed. The five key differentiators are: (1) unfamiliar codebases as the norm, not the exception; (2) client compliance and security requirements that vary per engagement; (3) time pressure with billable hour economics; (4) mandatory handoff documentation; and (5) multi-client context switching. These drive specific requirements for gdev's generated workflows that general-purpose developer tools miss.

## 1. Unfamiliar Codebases as Default Operating Mode

### The Problem
Product engineers work in familiar codebases for months or years. Consulting engineers regularly encounter new codebases -- sometimes weekly. The first task on any engagement is understanding what exists and how it works, under time pressure.

### Impact on Workflow Design

**Onboarding is not optional, it's the highest-frequency workflow**. Every consulting engagement starts with it. gdev's `/onboard` skill and `codebase-explorer` agent are not nice-to-haves; they're the most-used tools in the catalog.

**Architecture discovery must be systematic, not exploratory**. A product engineer can afford to learn organically. A consultant needs structured output in 30 minutes. The `/onboard` skill produces a document, not an ad-hoc conversation.

**Pattern recognition across codebases is valuable**. The `codebase-explorer` agent with `memory: project` accumulates knowledge across sessions, but a consulting firm also needs cross-project pattern libraries. Agents should encode architectural patterns the firm has encountered (e.g., "this looks like a typical Django REST framework project with DRF serializers and Celery task queues").

### gdev Design Requirements
- `/onboard` as a first-class, heavily-optimized skill (not an afterthought)
- Agent memory scoped to `project` for per-client knowledge accumulation
- CLAUDE.md templates that include "how to understand this project" instructions
- Rules for common framework patterns (`.claude/rules/django-patterns.md`, `.claude/rules/nextjs-patterns.md`) that help Claude understand unfamiliar code faster

## 2. Variable Client Compliance Requirements

### The Problem
Different clients have different security, compliance, and process requirements. One client might require OWASP ASVS Level 2 compliance, another might need SOC 2 controls, a third might have no formal requirements but expects HIPAA awareness. These requirements change per engagement, not per engineer.

### Impact on Workflow Design

**Security review intensity must be configurable per project**. gdev's wizard should ask about compliance requirements and adjust the generated configuration:

| Client Compliance | Generated Config |
|---|---|
| No specific requirements | Basic security rules, `/review-quick` |
| SOC 2 | Enhanced deny rules, `/security-review`, audit logging hooks |
| HIPAA | Data handling rules, PII detection hooks, `/compliance-check` skill |
| PCI-DSS | Payment-specific rules, `/security-assessment` (full OWASP) |
| Government/FedRAMP | Maximum deny rules, managed settings, no cloud API calls |

**Deny rules and hooks must be profile-driven**. gdev's profile system (designed in the gdev-extension-design spike) should include compliance profiles:

```yaml
claudecode:
  compliance_profile: soc2
  # This sets:
  # - Enhanced deny rules for data exfiltration
  # - SessionStart hook: audit dependency versions
  # - PostToolUse hooks: log all file modifications
  # - /compliance-check skill enabled
  # - security-reviewer agent with SOC 2 checklist
```

**Client-specific CLAUDE.md sections**. Consulting CLAUDE.md needs client-context sections:

```markdown
## Client Requirements
- Compliance: SOC 2 Type II
- Data classification: All customer data is PII
- Deployment: AWS us-east-1 only (data residency requirement)
- Review requirements: All changes require peer review before merge
```

### gdev Design Requirements
- Compliance profile selection in wizard
- Profile-driven deny rules, hooks, and skill configurations
- Client-context sections in CLAUDE.md template
- `/compliance-check` skill that validates against selected profile

## 3. Time Pressure and Billable Hour Economics

### The Problem
Consulting engineers bill by the hour. Every minute spent fighting tools or re-doing work is money. The value proposition of agentic workflows is 2-5x productivity -- but only if the tools don't create friction.

### Impact on Workflow Design

**Zero-friction setup is critical**. gdev's "quick path" wizard (accept defaults in <5 seconds) is essential. A consultant starting a new engagement Monday morning cannot spend 30 minutes configuring Claude Code.

**Graduated skill tiers map to billing**. Light review (seconds, pennies) for daily use; deep review (minutes, dollars) for milestone deliverables. Every skill should have an estimated time and cost:

```yaml
---
name: review-quick
description: Quick change review. ~30 seconds, ~$0.02.
---
```

```yaml
---
name: security-assessment
description: Full OWASP Top 10 + ASVS assessment. ~15 minutes, ~$8-10.
disable-model-invocation: true  # Don't trigger accidentally
---
```

**Handoff documentation is a deliverable, not overhead**. `/write-adr`, `/write-runbook`, `/handoff-doc` aren't internal tools -- they produce client-facing artifacts. The output format matters.

**Context switching between clients must be fast**. If a consultant works with 2-3 clients per week, switching between project-level Claude Code configs must be instant. gdev's per-project configuration model handles this naturally (`.claude/` per repo), but the consultant also needs personal skills that work across all projects.

### gdev Design Requirements
- Quick-path wizard for new engagements (< 5 seconds)
- Cost and time estimates in skill descriptions
- Client-deliverable output formatting in documentation skills
- Personal (user-level) skills for cross-project workflows
- `/estimate-effort` skill for scoping tasks

## 4. Mandatory Handoff Documentation

### The Problem
Consulting engagements end. The client needs to maintain what was built. Every engagement should produce documentation that enables the client's team to understand, operate, and modify the delivered system. This is often the last thing done and the first thing cut when time runs out.

### Impact on Workflow Design

**Documentation generation should be continuous, not end-of-engagement**. Rather than writing a handoff doc on the last day, gdev should encourage documentation at every milestone:

- After each major feature: `/write-adr` for the decision
- After each service setup: `/write-runbook` for operations
- After each complex module: inline documentation via `/add-docs`
- At engagement end: `/handoff-doc` synthesizes everything

### Skill: `/handoff-doc`

```yaml
---
name: handoff-doc
description: Generate a comprehensive handoff document for a client engagement. Synthesizes ADRs, runbooks, architecture decisions, and maintenance guidance into a single deliverable. Use at the end of a consulting engagement.
disable-model-invocation: true
context: fork
agent: codebase-explorer
---

Generate a comprehensive client handoff document for this project.

## Document Structure

### 1. Executive Summary
- What was built and why
- Key technical decisions and their rationale
- Current state of the system

### 2. Architecture Overview
- System diagram (text-based)
- Component responsibilities
- Data flow
- External integrations

### 3. Technology Stack
- Languages, frameworks, versions
- Why each was chosen
- Known limitations or planned upgrades

### 4. Development Guide
- How to set up a development environment
- How to build, test, and deploy
- Code organization and conventions
- Where to find things

### 5. Operations Guide
- How to deploy (step by step)
- How to monitor health
- How to handle common incidents
- Backup and recovery procedures

### 6. Decisions Log
- Reference existing ADRs in docs/adr/
- Summarize key decisions not captured in ADRs

### 7. Known Issues and Technical Debt
- Current bugs and workarounds
- Technical debt items with priority
- Planned improvements that weren't completed

### 8. Contact and Support
- Who built this and how to reach them
- Recommended next steps

Write the output to `docs/HANDOFF.md`.

Pull information from:
- Existing ADRs in docs/adr/
- Existing runbooks in docs/runbooks/
- README.md and CLAUDE.md
- Git log for recent changes and contributors
- Package manifests for dependency information

$ARGUMENTS
```

### gdev Design Requirements
- `/handoff-doc` as a first-class engagement completion skill
- Continuous documentation encouragement via SessionEnd hook reminders
- ADR and runbook skills that feed into handoff synthesis
- Client-facing output formatting (no internal jargon)

## 5. Multi-Client Context Switching

### The Problem
A consultant may work with 3 different clients in a week, each with different tech stacks, conventions, compliance requirements, and codebases. Context switching between these is cognitively expensive.

### Impact on Workflow Design

**Project-level config isolation is essential**. Each client repo has its own `.claude/` directory with tailored rules, skills, agents, and settings. When the consultant `cd`s to a different repo, Claude Code automatically loads the right configuration. gdev handles this naturally.

**Personal skills bridge projects**. Workflows that are consultant-standard (not client-specific) should be user-level skills (`~/.claude/skills/`):
- `/review-quick` -- same quality bar regardless of client
- `/estimate-effort` -- consultant's estimation methodology
- `/incident-debug` -- same debugging methodology
- `/write-adr` -- same ADR format

Project-level skills are client-specific:
- `/deploy` -- client's deployment procedure
- `/run-e2e` -- client's end-to-end test suite
- `/check-compliance` -- client's compliance requirements

**Named sessions for client separation**. Claude Code's session naming (`/rename`) and resume (`--resume`) support fast context switching:
```bash
# Morning: Client A
claude --resume "client-a-auth-refactor"

# Afternoon: Client B
claude --resume "client-b-api-migration"
```

### gdev Design Requirements
- Two-tier skill installation: user-level (consultant standard) + project-level (client specific)
- Session naming conventions in CLAUDE.md (`/rename` instructions)
- Per-project compliance profiles that don't leak between clients
- gdev wizard that detects existing client project configs and offers merge

## 6. Summary: What Makes Consulting Workflows Different

| Dimension | Product Development | Consulting Engineering |
|---|---|---|
| Codebase familiarity | High (months/years) | Low (days/weeks) |
| Onboarding frequency | Rare (new hire) | Frequent (every engagement) |
| Compliance requirements | Fixed for company | Variable per client |
| Security review intensity | Company standard | Client-mandated, variable |
| Documentation motivation | Internal knowledge | Client deliverable |
| Time model | Salary (time flexible) | Billable hours (time = money) |
| Context switching | Rare (one product) | Frequent (multi-client) |
| Config portability | One config, forever | Per-project config, discarded at end |
| Output audience | Internal team | Client team (unfamiliar with system) |
| Handoff requirement | None (continuous team) | Mandatory (engagement ends) |

### Top 5 gdev Differentiators for Consulting

1. **Onboarding-first workflow library**: `/onboard` and `codebase-explorer` are the flagship tools, not afterthoughts
2. **Compliance profiles**: Per-engagement security/compliance config driven by wizard selection
3. **Handoff documentation pipeline**: Continuous doc generation culminating in `/handoff-doc`
4. **Two-tier skill architecture**: User-level consultant standards + project-level client specifics
5. **Time-aware skill design**: Cost/time estimates, graduated tiers, zero-friction quick-path

## Depth Checklist

- [x] Underlying mechanism explained (5 structural differences, their impact on workflow design)
- [x] Key tradeoffs identified (user-level vs project-level skills, quick-path vs customized, continuous vs end-of-engagement docs)
- [x] Compared to alternatives (consulting vs product development workflows)
- [x] Failure modes described (client compliance mismatch, context leakage between clients, handoff doc too late)
- [x] Concrete examples found (compliance profiles, handoff-doc skill, session naming patterns)
- [x] Standalone-readable
