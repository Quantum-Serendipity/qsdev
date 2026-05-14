---
name: onboard-me
description: Systematic codebase onboarding. Explores architecture, patterns, conventions, key files.
disable-model-invocation: true
context: fork
agent: codebase-explorer
---

# Codebase Onboarding

Delegate to the codebase-explorer agent to perform a systematic exploration of the codebase.

## Deliverables

Produce a comprehensive onboarding document covering:

1. **Tech Stack**: Languages, frameworks, libraries, and their versions.
2. **Architecture**: High-level system design, module boundaries, data flow.
3. **Build & Test**: How to build, test, lint, and run the project locally.
4. **Directory Guide**: Purpose of each top-level directory and key subdirectories.
5. **Patterns & Conventions**: Coding style, naming conventions, error handling patterns, logging approach.
6. **Data Model**: Core domain types, database schema, API contracts.
7. **External Integrations**: Third-party services, APIs, message queues, databases.
8. **Gotchas**: Non-obvious behaviors, known issues, common pitfalls.
9. **Key Contributors**: Most active areas of the codebase and recent change patterns.

## Output

Write the onboarding document to `docs/ONBOARDING.md`.

Format with clear headings, code examples where helpful, and links to relevant source files.
