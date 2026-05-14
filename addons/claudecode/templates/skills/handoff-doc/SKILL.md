---
name: handoff-doc
description: Generate comprehensive client handoff documentation synthesizing ADRs, architecture, and operations.
disable-model-invocation: true
context: fork
agent: handoff-doc-generator
---

# Handoff Documentation

Delegate to the handoff-doc-generator agent to produce comprehensive client handoff documentation.

## Sources

Pull information from:

1. **ADRs**: Architecture Decision Records in `docs/adr/`.
2. **Runbooks**: Operational runbooks and playbooks.
3. **README**: Project README and setup instructions.
4. **CLAUDE.md**: Project documentation and conventions.
5. **Git log**: Recent commit history and contributors.
6. **Manifests**: Package manifests (package.json, go.mod, requirements.txt, etc.).
7. **CI configs**: GitHub Actions, GitLab CI, or other CI/CD configurations.

## Deliverables

Write to `docs/HANDOFF.md` with sections covering:

1. **Project Overview**: Purpose, stakeholders, current status.
2. **Architecture**: System design, component relationships, data flow.
3. **Key Decisions**: Summary of ADRs with rationale.
4. **Setup Guide**: Step-by-step development environment setup.
5. **Deployment**: How to deploy, environments, release process.
6. **Operations**: Monitoring, alerting, incident response.
7. **Known Issues**: Outstanding bugs, tech debt, planned improvements.
8. **Contacts**: Team members, roles, escalation paths.
