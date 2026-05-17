# Roadmap

Planned enhancements beyond the current release. Items are grouped by theme, not priority order.

## Cloud & Infrastructure

- **Cloud credential isolation** — per-project AWS, GCP, and Azure CLI credential scoping with doctor checks
- **Kubernetes ecosystem modules** — kubectl, Helm, and cloud-auth plugins with per-project KUBECONFIG enforcement
- **Expanded service detection** — Kafka, MinIO, Mailpit, Keycloak, NATS (11 total services, up from 6)

## Developer Experience

- **Non-language tool detection** — detect Git workflows, API tools, database tools, docs tooling; generate tasks and CLAUDE.md sections automatically
- **IDE and shell configuration** — project-level EditorConfig, VS Code extensions, user-level shell integration and personal tool preferences
- **Copier template integration** — templated project scaffolding with template registry and qsdev-init orchestration
- **Agentic quality patterns** — learning skills, project clarity templates, calm directives, and quality benchmarking

## MCP & Documentation

- **MCP server registry** — registry-driven system replacing the hardcoded MCP list, with security tiers and a 40-tool ceiling
- **Local documentation pipeline** — local-first docs via ZIM/DevDocs/man MCPs with prompt injection hardening

## Consulting & Team Management

- **Encrypted client profiles** — per-client configuration using sops+age with SecretSpec runtime resolution
- **Consulting enforcement hooks** — credential isolation, destructive-op prevention, and audit logging hooks for consulting environments
- **Observability and analytics** — OpenTelemetry sidecar, JSONL event collection, and cost visibility integration

## Security & Supply Chain

- **SBOM and attestation** — SBOM generation, cosign signing, OpenVEX vulnerability documents, SLSA Build Level 2
- **Native security library** — compiled Go policy engine with package risk assessment and MCP trust scoring
- **Agent self-protection** — absolute-deny rules preventing agents from dismantling security infrastructure, interactive bypass system, monitor mode, and Write/Edit hook protection
- **Consolidated security binary** — single compiled Go hook binary with tree-sitter Bash parsing and pre-compiled regex for performance
