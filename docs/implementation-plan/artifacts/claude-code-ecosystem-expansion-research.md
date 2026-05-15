# Claude Code Ecosystem Expansion Research

Comprehensive survey of Claude Code plugins, skills, hooks, MCP servers, and marketplace tools beyond the security-focused tools already integrated into the gdev plan. Organized by category, ranked by relevance and maturity within each.

Research date: 2026-05-12.

**Deduplication note** -- the following tools are already in the gdev plan and are NOT covered here:
- attach-guard (PreToolUse hook plugin)
- Security Phoenix (skills + hooks suite)
- Trail of Bits skills (40+ security skills)
- Trail of Bits claude-code-config (production defaults)
- Claude Code Security Review GitHub Action
- agent-postmortem-skill (verification protocol)
- Version-Sentinel (dependency version guardrails)
- semble (semantic code search MCP server)
- Socket.dev MCP server
- SonarQube plugin
- Aikido plugin

---

## 1. Code Quality and Review

### 1.1 Anthropic Code Review (Official Plugin)

- **URL**: https://github.com/anthropics/claude-code/tree/main/plugins/code-review
- **What**: Official multi-agent code review plugin. Dispatches 5 independent reviewer agents in parallel (CLAUDE.md compliance, bug detection, git history analysis, prior PR comment review, code comment verification). Each finding scored 0-100 confidence; only 80+ are posted as inline PR comments.
- **Mechanism**: Plugin (installed via `/plugin install code-review@claude-plugins-official`)
- **Maturity**: Official Anthropic plugin. Research preview for Team/Enterprise subscriptions.
- **License**: Proprietary (Anthropic)
- **Cost**: $15-25 per review, scales with PR size. 20-minute average completion.
- **Consulting relevance**: HIGH. Enterprise-grade code review for teams producing AI-assisted code. The confidence scoring reduces false positive noise that plagues other review tools.
- **gdev integration**: Configure via `claudecode` addon when Team/Enterprise subscription detected. Add as optional CI step in generated GitHub Actions.

### 1.2 CodeRabbit (Official Marketplace Partner)

- **URL**: https://github.com/coderabbitai/claude-plugin
- **What**: AI-powered code review with 40+ bundled deterministic linters (ESLint, Pylint, Golint, RuboCop, etc.). Multi-file logic analysis. PR comment management from Claude Code.
- **Mechanism**: Plugin (official marketplace)
- **Maturity**: 43 stars on plugin repo. Commercial product with deep Claude Code integration. Active maintenance.
- **License**: MIT (plugin); Commercial (service)
- **Consulting relevance**: HIGH. The 40+ bundled linters cover every Tier 1-2 language in gdev's ecosystem. Better at cross-file logic bugs than pure linting.
- **gdev integration**: Addon enables via `.claude/settings.json` plugin config when CodeRabbit API key present. Pair with code-review plugin for defense-in-depth.

### 1.3 Qodo Skills (Official Marketplace Partner)

- **URL**: Official marketplace (`qodo-skills`)
- **What**: Shift-left code review with organization-level coding rules. Quality, testing, security, and compliance review. Built on open-source PR-Agent.
- **Mechanism**: Plugin (official marketplace)
- **Maturity**: Commercial product. Open-source core (PR-Agent).
- **License**: Commercial (plugin); Apache-2.0 (PR-Agent core)
- **Consulting relevance**: MEDIUM-HIGH. Org-level coding rules are valuable for consulting engagements enforcing client standards. Better at subtle cross-file logic bugs than CodeRabbit per comparative reviews.
- **gdev integration**: Configure via settings.json when Qodo API key present. Complementary to CodeRabbit (different detection strengths).

### 1.4 minimal-claude (Auto-Configure Linting + Fixing)

- **URL**: https://github.com/KenKaiii/minimal-claude
- **What**: Auto-detects project linting/typechecking tools, configures them, and provides parallel agent-based fixing. Includes `/setup-commits` for quality-gated commits with AI-generated messages.
- **Mechanism**: Plugin (commands + agents)
- **Maturity**: 18 stars. Created Oct 2025, still updated May 2026. No license declared.
- **License**: None declared
- **Consulting relevance**: MEDIUM. Auto-detection of linting tools aligns with gdev's "detect, don't assume" principle. The `/commit` command with quality checks is useful.
- **gdev integration**: Reference architecture for the `claudecode` addon's linting integration. The auto-detection pattern is directly applicable. Too immature to bundle directly.

### 1.5 Greptile (Official Marketplace Partner)

- **URL**: https://www.greptile.com/docs/integrations/claude-code
- **What**: AI-powered codebase search and code review. Indexes entire repositories into semantic code graphs. Multi-hop investigation tracing dependencies across files. Natural language codebase queries.
- **Mechanism**: Plugin (official marketplace)
- **Maturity**: Commercial product. Official Anthropic marketplace partner.
- **License**: Commercial
- **Consulting relevance**: MEDIUM-HIGH. Valuable for onboarding onto new client codebases. Semantic code graph is complementary to semble's AST-based search.
- **gdev integration**: Configure in settings.json when Greptile API key present. Position as optional enhancement for large codebase navigation.

### 1.6 Serena (Official Marketplace Partner)

- **URL**: https://github.com/oraios/serena
- **What**: LSP-powered MCP server providing symbol-level code navigation, semantic retrieval, and refactoring across 30+ languages. IDE-like capabilities without an IDE.
- **Mechanism**: MCP server + Plugin (official marketplace)
- **Maturity**: 24,130 stars. Created Mar 2025, actively maintained. MIT license.
- **License**: MIT
- **Consulting relevance**: HIGH. Zero-cost, open-source, works with any language. LSP-based understanding is more reliable than embedding-based search for refactoring tasks.
- **gdev integration**: Strong candidate for default MCP server in `.mcp.json`. Zero API key requirement. Complements semble (semantic search) with LSP-based navigation. Configure via `claudecode` addon.

### 1.7 claude-context (Code Search MCP by Zilliz)

- **URL**: https://github.com/zilliztech/claude-context
- **What**: Code search MCP server for Claude Code. Makes entire codebase the context for coding agents using vector search.
- **Mechanism**: MCP server
- **Maturity**: 10,991 stars. Created Jun 2025, actively maintained. MIT license.
- **License**: MIT
- **Consulting relevance**: MEDIUM. Alternative to semble and Serena for codebase understanding. Vector-based approach.
- **gdev integration**: Could be offered as alternative to semble in addon configuration. Less specialized than Serena for code navigation.

---

## 2. Testing

### 2.1 ATDD (Acceptance Test Driven Development)

- **URL**: https://github.com/swingerman/atdd
- **What**: Enforces Uncle Bob's ATDD methodology with Claude Code. Three validation layers: acceptance tests (Given/When/Then specs), unit tests, and mutation testing. Builds project-specific AST-walking mutation tools or integrates Stryker/mutmut/PIT. Language-agnostic.
- **Mechanism**: Plugin (skill + commands)
- **Maturity**: 93 stars. Created Feb 2026. MIT license. Shell-based.
- **License**: MIT
- **Consulting relevance**: HIGH. Mutation testing validation layer is the gold standard for test quality. Language-agnostic approach fits multi-language consulting environments. The 73% equivalent mutant detection rate (with Trailmark) demonstrates real value.
- **gdev integration**: Include as optional testing skill. Pair with Trailmark for graph-based mutation triage. Configure via `claudecode` addon's skills directory.

### 2.2 Trail of Bits Trailmark

- **URL**: https://github.com/trailofbits/skills (trailmark plugin)
- **What**: Turns source code into a queryable call graph (functions, classes, call relationships, semantic metadata). Powers 8 Claude Code skills for blast radius analysis, taint propagation, and mutation test triage. Supports 17 languages. On one codebase, classified 73% of surviving mutants as equivalent.
- **Mechanism**: Plugin (skills + Python library on PyPI)
- **Maturity**: Part of Trail of Bits skills repo (5,144 stars). Apache-2.0 license. April 2026 release.
- **License**: Apache-2.0
- **Consulting relevance**: HIGH. Professional-grade code analysis from a leading security firm. The mutation triage capability alone justifies inclusion. 17-language support covers all gdev tiers.
- **gdev integration**: Bundle Trailmark skills in the `claudecode` addon's skills library alongside the already-integrated ToB security skills. Requires Python 3.10+ (already a devenv.nix dependency for semble).

### 2.3 clear-solutions/unit-tests-skills

- **URL**: https://github.com/clear-solutions/unit-tests-skills
- **What**: AI agent skills for generating high-quality unit tests. Two skills: Generate Tests (full workflow with code analysis, test case output, code generation) and Generate Test Cases (Given-When-Then structured output only). Java-focused (JUnit 5, Mockito, AssertJ).
- **Mechanism**: Skill (SKILL.md files)
- **Maturity**: 12 stars. Created Jan 2026. No license. Java-only.
- **License**: None declared
- **Consulting relevance**: LOW-MEDIUM. Java-specific. The test generation workflow pattern (analyze -> output cases -> generate code) is a good reference architecture, but the language limitation restricts direct use.
- **gdev integration**: Reference architecture only. The workflow pattern could inform a language-agnostic test generation skill.

### 2.4 Qodo Testing Skills

- **URL**: Official marketplace (`qodo-skills`)
- **What**: Test generation, coverage analysis, and quality assessment integrated with Qodo's code review. Generates tests covering happy paths, edge cases, error conditions.
- **Mechanism**: Plugin (official marketplace)
- **Maturity**: Commercial product with active development.
- **License**: Commercial
- **Consulting relevance**: MEDIUM. Overlaps with ATDD for test generation. Commercial dependency.
- **gdev integration**: Configure alongside Qodo code review when API key present.

---

## 3. Documentation

### 3.1 levnikolaevich/claude-code-skills (Documentation Pipeline)

- **URL**: https://github.com/levnikolaevich/claude-code-skills
- **What**: Full delivery lifecycle plugin suite including documentation-pipeline plugin. Generates API docs, README files, changelogs. Includes documentation fact-checker that extracts verifiable claims from markdown and cross-checks against codebase. Also includes hex-line (hash-verified editing MCP), hex-graph (code knowledge graph MCP), and hex-ssh (remote SSH MCP).
- **Mechanism**: Plugin (multiple installable sub-plugins) + MCP servers
- **Maturity**: 464 stars. Created Oct 2025, actively maintained. MIT license. JavaScript.
- **License**: MIT
- **Consulting relevance**: MEDIUM-HIGH. Documentation fact-checker is uniquely valuable -- prevents doc drift. The hash-verified editing MCP (hex-line) is interesting for audit trails. Full delivery pipeline may be too opinionated for gdev's modular approach.
- **gdev integration**: Cherry-pick documentation-pipeline and codebase-audit-suite plugins. Configure as optional addons. The fact-checker pattern could be adapted into a standalone skill.

### 3.2 Context7 (Live Documentation MCP)

- **URL**: https://github.com/upstash/context7
- **What**: MCP server that fetches up-to-date, version-specific documentation and code examples for 50+ frameworks directly into prompts. Solves the stale training data problem. Includes auto-trigger for documentation lookups and a docs-researcher agent.
- **Mechanism**: MCP server + Plugin (official marketplace)
- **Maturity**: 55,134 stars. Created Mar 2025, actively maintained. MIT license. By Upstash.
- **License**: MIT
- **Consulting relevance**: HIGH. Every developer hits stale documentation issues. Version-specific docs for React 19, Next.js 15, etc. are essential for consulting teams working across client stacks. Zero cost.
- **gdev integration**: Strong candidate for default MCP server in `.mcp.json`. Zero configuration required (`npx -y @upstash/context7-mcp@latest`). No API key needed. Configure via `claudecode` addon.

### 3.3 shinpr/claude-code-workflows

- **URL**: https://github.com/shinpr/claude-code-workflows
- **What**: Production-ready development workflows. Includes claude-code-discover (PRDs from feature ideas), metronome (detects AI shortcut-taking behavior), linear-prism (requirements -> Linear tasks). Full-stack recipe routing.
- **Mechanism**: Plugin (commands + agents)
- **Maturity**: 347 stars. Created Oct 2025, actively maintained. MIT license.
- **License**: MIT
- **Consulting relevance**: MEDIUM. The metronome behavior detection (anti-shortcutting) is unique and valuable for ensuring AI code quality. PRD generation from feature ideas could help consulting discovery phases.
- **gdev integration**: Reference architecture for workflow patterns. The metronome concept could be adapted as a hook in the `claudecode` addon.

---

## 4. DevOps and Infrastructure

### 4.1 HashiCorp Agent Skills (Official)

- **URL**: https://github.com/hashicorp/agent-skills
- **What**: Official HashiCorp skills for Terraform and Packer. Includes terraform-code-generation, terraform-module-generation, terraform-provider-development, terraform-style-guide, terraform-test, packer-builders, and packer-hcp.
- **Mechanism**: Skill (marketplace)
- **Maturity**: 613 stars. Created Nov 2025, actively maintained. MPL-2.0 license. Official vendor support.
- **License**: MPL-2.0
- **Consulting relevance**: HIGH. Official vendor skills for the most common IaC tool. Terraform style guide enforcement and testing skills are directly applicable to consulting engagements.
- **gdev integration**: Auto-enable when Terraform detected in project. Configure via `claudecode` addon. Add marketplace reference to generated settings.json.

### 4.2 antonbabenko/terraform-skill

- **URL**: https://github.com/antonbabenko/terraform-skill
- **What**: Community Terraform & OpenTofu best-practices skill. Testing strategies, module patterns, CI/CD workflows, and production infrastructure code. By Anton Babenko (terraform-aws-modules maintainer).
- **Mechanism**: Skill (SKILL.md)
- **Maturity**: 1,826 stars. Created Jan 2026. Apache-2.0 license (listed as NOASSERTION in API). Well-maintained by a prominent Terraform community member.
- **License**: Apache-2.0
- **Consulting relevance**: HIGH. Highly starred, from a trusted community figure. Covers OpenTofu (important for clients avoiding BSL license). Complements HashiCorp's official skills with community best practices.
- **gdev integration**: Include as default skill when Terraform/OpenTofu detected. Can coexist with HashiCorp official skills (different focus areas).

### 4.3 LukasNiessen/terrashark

- **URL**: https://github.com/LukasNiessen/terrashark
- **What**: Terraform skill focused on eliminating LLM hallucinations with Terraform. Grounded in official HashiCorp best practices. Modular and secure code patterns.
- **Mechanism**: Skill (SKILL.md)
- **Maturity**: 330 stars. Created Feb 2026. MIT license.
- **License**: MIT
- **Consulting relevance**: MEDIUM. Anti-hallucination focus is valuable but overlaps with HashiCorp official skills and antonbabenko's skill.
- **gdev integration**: Optional alternative/complement to the above Terraform skills. Lower priority than HashiCorp official + antonbabenko.

### 4.4 ahmedasmar/devops-claude-skills

- **URL**: https://github.com/ahmedasmar/devops-claude-skills
- **What**: DevOps skills marketplace covering Terraform/Terragrunt (with state inspection and module validators), Kubernetes troubleshooting (pod diagnostics, incident response playbooks), CI/CD pipeline design (GitHub Actions, GitLab CI), monitoring/observability (Prometheus alerts, OpenTelemetry config, incident runbooks), AWS cost optimization, and GitOps workflows.
- **Mechanism**: Skill (marketplace)
- **Maturity**: 150 stars. Created Oct 2025, maintained. No license declared. Python.
- **License**: None declared
- **Consulting relevance**: MEDIUM-HIGH. Broad coverage of DevOps workflows. The Kubernetes troubleshooting playbooks and CI/CD pipeline skills are directly applicable. Production-ready monitoring templates save significant setup time.
- **gdev integration**: Cherry-pick individual skills (k8s-troubleshooting, ci-cd, monitoring) rather than bundling the whole marketplace. Configure via `claudecode` addon's skills directory.

### 4.5 Docker-Claude-Skill-Package

- **URL**: https://github.com/OpenAEC-Foundation/Docker-Claude-Skill-Package
- **What**: 22 deterministic Claude skills for Docker and Docker Compose. Dockerfile best practices, multi-stage builds, Compose orchestration, container security, Hadolint integration.
- **Mechanism**: Skill (SKILL.md files)
- **Maturity**: 7 stars. Created Mar 2026. MIT license. Very new.
- **License**: MIT
- **Consulting relevance**: LOW-MEDIUM. Docker skills are useful but 7 stars suggests minimal community validation. The Hadolint pre-commit integration pattern is valuable.
- **gdev integration**: Reference architecture for Docker skills. The Hadolint pre-commit pattern should inform gdev's pre-commit hook generation for Docker projects.

### 4.6 Terraform MCP Server (Official Marketplace)

- **URL**: Official marketplace (`terraform` external plugin)
- **What**: Official Terraform plugin in the Anthropic marketplace. Direct integration with Terraform and HashiCorp tools.
- **Mechanism**: Plugin (official marketplace)
- **Maturity**: Official marketplace partner.
- **License**: Commercial (HashiCorp)
- **Consulting relevance**: HIGH. Official integration path.
- **gdev integration**: Configure alongside HashiCorp Agent Skills when Terraform detected.

---

## 5. Performance and Observability

### 5.1 claude-code-otel (OpenTelemetry Stack)

- **URL**: https://github.com/ColeMurray/claude-code-otel
- **What**: Complete observability solution for monitoring Claude Code usage, performance, and costs. Architecture: Claude Code -> OTel Collector -> Prometheus (metrics) + Loki (events/logs) -> Grafana (visualization). Tracks active sessions, cost, token usage, lines of code changed. Deploys in under 90 seconds.
- **Mechanism**: Infrastructure (Docker Compose stack)
- **Maturity**: 396 stars. Created Jun 2025. MIT license. Self-contained stack with ~50MB container images.
- **License**: MIT
- **Consulting relevance**: HIGH. Essential for consulting firms tracking Claude Code costs and usage across teams. The Grafana dashboards provide immediate visibility into spend per developer/project.
- **gdev integration**: Include docker-compose.yml template in `claudecode` addon. Configure Claude Code's OTel export env vars in generated devenv.nix. The `CLAUDE_CODE_ENABLE_TELEMETRY=1` and `CLAUDE_CODE_ENHANCED_TELEMETRY_BETA=1` settings belong in the profile system.

### 5.2 disler/claude-code-hooks-multi-agent-observability

- **URL**: https://github.com/disler/claude-code-hooks-multi-agent-observability
- **What**: Real-time monitoring for Claude Code agents through hook event tracking. Provides visibility into what subagents are doing (normally invisible in the terminal).
- **Mechanism**: Hooks (lifecycle hooks)
- **Maturity**: 1,411 stars. Created Jul 2025. No license declared. Python.
- **License**: None declared
- **Consulting relevance**: MEDIUM-HIGH. Multi-agent observability is critical for debugging complex Claude Code workflows. The insight into subagent behavior is unique.
- **gdev integration**: Reference architecture for observability hooks. The hook patterns could inform gdev's generated hook configurations.

### 5.3 simple10/agents-observe

- **URL**: https://github.com/simple10/agents-observe
- **What**: Real-time observability plugin for Claude Code sessions and multi-agents. Marketplace-installable. Provides visibility into autonomous agent operations.
- **Mechanism**: Plugin (marketplace)
- **Maturity**: 554 stars. Created Mar 2026, actively maintained. MIT license. TypeScript.
- **License**: MIT
- **Consulting relevance**: MEDIUM. More recent and actively maintained than disler's hooks. Marketplace installation is simpler.
- **gdev integration**: Configure as optional plugin when multi-agent workflows detected.

### 5.4 TechNickAI/claude_telemetry

- **URL**: https://github.com/TechNickAI/claude_telemetry
- **What**: Drop-in OpenTelemetry wrapper for Claude Code CLI. Swaps `claude` for `claudia` command. Logs tool calls, token usage, costs, execution traces to Logfire, Sentry, Honeycomb, or Datadog.
- **Mechanism**: CLI wrapper (drop-in replacement)
- **Maturity**: 23 stars. Created Oct 2025. MIT license. Python. Low activity.
- **License**: MIT
- **Consulting relevance**: LOW. Claude Code's native OTel support (env vars) has largely superseded this approach. The wrapper pattern introduces an extra layer of indirection.
- **gdev integration**: Not recommended. Use native OTel configuration instead (covered by claude-code-otel).

### 5.5 nexus-labs-automation/agent-observability

- **URL**: https://github.com/nexus-labs-automation/agent-observability
- **What**: Plugin providing observability best practices for AI agents. 14 skills, 2 agents, 2 commands (/instrument, /audit), 7 anti-pattern hooks. 9 framework guides (LangChain, LangGraph, Claude Agent SDK, CrewAI, etc.). 10 vendor integrations (Langfuse, LangSmith, Arize, OTel, Sentry, Datadog).
- **Mechanism**: Plugin (skills + hooks + agents)
- **Maturity**: 6 stars. Created Dec 2025. MIT license. Very low adoption.
- **License**: MIT
- **Consulting relevance**: LOW. Comprehensive in scope but minimal community adoption. The anti-pattern hooks and vendor integration guides are useful reference material.
- **gdev integration**: Reference only. Too immature for direct integration.

### 5.6 levnikolaevich/claude-code-skills (Observability Auditor)

- **URL**: https://github.com/levnikolaevich/claude-code-skills (codebase-audit-suite plugin)
- **What**: Observability auditor skill that checks codebases for logging, monitoring, and instrumentation gaps. Part of the broader codebase-audit-suite.
- **Mechanism**: Skill (within plugin)
- **Maturity**: 464 stars (parent repo). MIT license.
- **License**: MIT
- **Consulting relevance**: MEDIUM. Useful for auditing client codebases' observability posture.
- **gdev integration**: Bundle the observability-auditor skill if the parent codebase-audit-suite is included.

---

## 6. Database and Migration

### 6.1 Prisma MCP Server (Official)

- **URL**: https://www.prisma.io/mcp
- **What**: Official MCP server built into Prisma CLI. Full migration lifecycle: check status, create migrations from schema diffs, execute, reset. No separate installation needed for Prisma projects.
- **Mechanism**: MCP server (built into `prisma` CLI)
- **Maturity**: Prisma has 43k+ GitHub stars. Official vendor MCP. Production-ready.
- **License**: Apache-2.0 (Prisma)
- **Consulting relevance**: HIGH for TypeScript projects. Most popular TS ORM. Migration management from Claude Code is high-value.
- **gdev integration**: Auto-configure in `.mcp.json` when `prisma` detected in project dependencies. Only for Prisma projects (ORM-locked).

### 6.2 PostgreSQL MCP Server (Anthropic Reference)

- **URL**: https://github.com/modelcontextprotocol/servers (postgres server)
- **What**: Official reference MCP server for PostgreSQL. Read-only by default. Direct SQL query access.
- **Mechanism**: MCP server
- **Maturity**: Part of official MCP servers repo (85,528 stars). TypeScript.
- **License**: Unspecified (in servers monorepo)
- **Consulting relevance**: HIGH. PostgreSQL is the most common database across consulting projects.
- **gdev integration**: Include in `.mcp.json` template with read-only configuration. Prompt for connection string during `qsdev init` wizard.

### 6.3 Neon MCP Server

- **URL**: Neon documentation
- **What**: MCP server for Neon serverless Postgres. Database branching support (create isolated database branches for development/testing).
- **Mechanism**: MCP server
- **Maturity**: Official vendor MCP. Neon is a well-funded startup with active development.
- **License**: Vendor-provided
- **Consulting relevance**: MEDIUM. Relevant for teams using Neon. Database branching is a killer feature for AI-assisted development (safe experimentation).
- **gdev integration**: Optional MCP server when Neon detected or configured via profile.

### 6.4 Supabase MCP Server

- **URL**: Official Anthropic marketplace
- **What**: MCP for database operations, project management, SQL execution, and backend interaction on Supabase platform.
- **Mechanism**: MCP server
- **Maturity**: Official marketplace. Supabase is a well-established platform.
- **License**: Vendor-provided
- **Consulting relevance**: MEDIUM. Common in startup/SaaS consulting engagements.
- **gdev integration**: Optional MCP server when Supabase detected or configured via profile.

### 6.5 alirezarezvani/claude-skills (Database Designer)

- **URL**: https://github.com/alirezarezvani/claude-skills/blob/main/engineering/database-designer/SKILL.md
- **What**: Expert-level database design, optimization, and migration skill. Part of 245+ skill collection.
- **Mechanism**: Skill (SKILL.md)
- **Maturity**: 14,555 stars (parent repo). MIT license. Python.
- **License**: MIT
- **Consulting relevance**: MEDIUM. Generic database design guidance. Less valuable than ORM-specific MCP servers.
- **gdev integration**: Optional skill for database-heavy projects.

---

## 7. Compliance and Licensing

### 7.1 melodic-software SBOM Management Skill

- **URL**: LobeHub Skills Marketplace (`melodic-software/claude-code-plugins sbom-management`)
- **What**: Comprehensive SBOM management: generation, normalization, storage, vulnerability monitoring. Exports SPDX and CycloneDX formats. CI/CD integration. Real-time CVE/NVD correlation. Policy enforcement with license block/allow lists. Cryptographic signing for attestation. Meets Executive Order 14028 and EU CRA requirements.
- **Mechanism**: Skill (installable via `npx skillfish add`)
- **Maturity**: Community skill on LobeHub marketplace. Maturity unclear -- no GitHub star data available.
- **License**: Unknown
- **Consulting relevance**: HIGH. SBOM generation is increasingly required by enterprise clients (EO 14028, EU CRA). License compliance checking is essential for consulting firms delivering to regulated industries.
- **gdev integration**: Include as optional compliance skill. Configure license block/allow lists via gdev profile system. The SPDX/CycloneDX export integrates with artifact-keeper's scanning pipeline.

### 7.2 AgentSecOps/SecOpsAgentKit (SBOM + Compliance Skills)

- **URL**: https://github.com/AgentSecOps/SecOpsAgentKit
- **What**: 25+ security operations skills including sbom-syft (SBOM generation using Syft), policy-opa (OPA policy-as-code enforcement), secrets-gitleaks (secret detection), sast-horusec (18+ language SAST), dast-zap (DAST with ZAP). Also includes incident response and threat modeling skills.
- **Mechanism**: Skills (SKILL.md files organized by category)
- **Maturity**: 134 stars. Created Nov 2025, maintained to Apr 2026. Python. License listed as NOASSERTION.
- **License**: Not clearly declared
- **Consulting relevance**: HIGH. The sbom-syft and policy-opa skills fill critical gaps. OPA policy enforcement is particularly valuable for consulting firms needing client-specific compliance rules. The breadth of security operations coverage extends beyond the plan's current security tools.
- **gdev integration**: Cherry-pick sbom-syft and policy-opa skills. These complement the security tools already in the plan without overlap. Configure OPA policies via gdev profile system.

### 7.3 Snyk Agent Scan

- **URL**: https://github.com/snyk/agent-scan
- **What**: Security scanner for AI agents, MCP servers, and agent skills. Discovers and scans agent components for prompt injections and vulnerabilities. Scans agents, MCP servers, and skills installed on the developer's machine.
- **Mechanism**: CLI tool (standalone scanner)
- **Maturity**: 2,392 stars. Created Apr 2025, actively maintained. Apache-2.0. Python.
- **License**: Apache-2.0
- **Consulting relevance**: HIGH. Meta-security: scans the AI tooling itself for vulnerabilities. As gdev installs multiple plugins and MCP servers, scanning them for injection risks is essential defense-in-depth.
- **gdev integration**: Run as a post-setup validation step in `qsdev devenv doctor`. Verify all installed MCP servers and plugins are clean. Include in CI pipeline.

### 7.4 Snyk MCP Server

- **URL**: https://docs.snyk.io/integrations/snyk-studio-agentic-integrations/quickstart-guides-for-snyk-studio/claude-code-guide
- **What**: 11 security scanning tools via MCP: SAST, SCA (open-source vulnerability + license compliance), container image scanning, IaC misconfiguration detection (Terraform, K8s, CloudFormation).
- **Mechanism**: MCP server
- **Maturity**: Official vendor MCP. Snyk is an established security platform. Requires Snyk token.
- **License**: Commercial (Snyk)
- **Consulting relevance**: HIGH. Comprehensive security scanning from a single MCP server. License compliance analysis via SCA is directly relevant.
- **gdev integration**: Configure in `.mcp.json` when Snyk token present in profile. Position as enterprise-tier alternative/complement to Socket.dev MCP.

---

## 8. AI Agent Workflow

### 8.1 claude-mem (Persistent Session Memory)

- **URL**: https://github.com/thedotmack/claude-mem
- **What**: Persistent context across sessions. Captures everything Claude does, compresses with AI, injects relevant context into future sessions. Uses 5 lifecycle hooks, SQLite + ChromaDB storage, TypeScript hooks. Supports 28 languages. 65k+ stars.
- **Mechanism**: Plugin (hooks + worker service + skills)
- **Maturity**: 75,212 stars. Created Aug 2025, actively maintained. Apache-2.0 license. TypeScript.
- **License**: Apache-2.0
- **Consulting relevance**: HIGH. Session persistence is the #1 developer pain point with Claude Code. The AI-compressed context injection is more sophisticated than CLAUDE.md memory. Massive community validation.
- **gdev integration**: Include as recommended plugin. Configure via `claudecode` addon. Note: runs a worker service on port 37777 -- needs firewall consideration in security-hardened environments. The SQLite + ChromaDB storage adds system dependencies.

### 8.2 everything-claude-code

- **URL**: https://github.com/affaan-m/everything-claude-code
- **What**: Agent harness performance optimization system. Skills, instincts, memory, security, and research-first development. PM2 multi-agent orchestration with 6 commands. 80 focused plugins optimized for minimal token usage. 153 specialized skills with progressive disclosure.
- **Mechanism**: Plugin suite (skills + hooks + agents)
- **Maturity**: 180,441 stars. Created Jan 2026, actively maintained. MIT license. JavaScript.
- **License**: MIT
- **Consulting relevance**: MEDIUM. Massive star count but very broad scope. The "instincts" concept (behavioral patterns) and token-optimized plugin design are valuable ideas. May conflict with gdev's curated approach.
- **gdev integration**: Reference architecture for token efficiency patterns. Cherry-pick specific optimizations rather than bundling the full suite. Too opinionated for gdev's modular approach.

### 8.3 wshobson/agents

- **URL**: https://github.com/wshobson/agents
- **What**: Intelligent automation and multi-agent orchestration for Claude Code. Agent orchestration plugin with specialized agents for different tasks.
- **Mechanism**: Plugin (agents + orchestration)
- **Maturity**: 35,271 stars. Created Jul 2025, actively maintained. MIT license. Python.
- **License**: MIT
- **Consulting relevance**: MEDIUM. Multi-agent orchestration is increasingly important as Claude Code workflows become more complex. Built-in agent teams (native feature) may supersede this.
- **gdev integration**: Reference architecture. Claude Code's native agent teams feature may make this redundant.

### 8.4 corca-ai/claude-plugins (Structured Workflow)

- **URL**: https://github.com/corca-ai/claude-plugins
- **What**: Structured development workflow plugin (cwf). Chains: gather -> clarify -> plan -> review(plan) -> impl -> review(code) -> refactor -> retro -> ship. Explicit gates prevent premature execution. Unresolved ambiguity triggers user decisions.
- **Mechanism**: Plugin (workflow commands)
- **Maturity**: 77 stars. Created Jan 2026. MIT license. Shell.
- **License**: MIT
- **Consulting relevance**: MEDIUM. The gated workflow pattern (plan review before implementation) aligns with consulting best practices. The explicit ambiguity resolution is valuable.
- **gdev integration**: Reference architecture for workflow gates. The plan-then-implement pattern could inform gdev's CLAUDE.md workflow instructions.

### 8.5 severity1/claude-code-prompt-improver

- **URL**: https://github.com/severity1/claude-code-prompt-improver
- **What**: Hook-based prompt improver. Automatically transforms vague user prompts into structured, precision prompts before Claude processes them. "Type vibes, ship precision."
- **Mechanism**: Hook (UserPromptSubmit)
- **Maturity**: 1,454 stars. Created Oct 2025, maintained. MIT license. Python.
- **License**: MIT
- **Consulting relevance**: MEDIUM-HIGH. Improves output quality without requiring developers to learn prompt engineering. The hook-based approach is zero-friction.
- **gdev integration**: Include as optional hook. Configure via `claudecode` addon settings. Note: adds latency to every prompt (needs benchmarking). May conflict with structured workflow plugins.

---

## 9. MCP Servers (Beyond Already-Integrated)

### 9.1 Ecosystem Overview

The MCP ecosystem has grown to 23,000+ servers across multiple registries. Key registries:
- **Official MCP Registry**: https://registry.modelcontextprotocol.io/
- **Glama**: 23,378 servers
- **Smithery**: 7,000+ servers (closest to Docker Hub for MCP)
- **PulseMCP**: 9,080+ servers

The official `modelcontextprotocol/servers` repo (85,528 stars) provides 6 reference servers: filesystem, PostgreSQL, Brave Search, GitHub, Puppeteer, Google Maps.

### 9.2 Project Management MCP Servers

| Server | Type | Stars/Status | Consulting Relevance |
|--------|------|-------------|---------------------|
| **Atlassian (Official)** | Remote MCP | 669 stars, Apache-2.0 | HIGH -- Jira + Confluence unified access, OAuth |
| **sooperset/mcp-atlassian** | Community MCP | 5,162 stars, MIT | HIGH -- More stars than official, longer track record |
| **Linear (Official)** | Remote MCP | Official vendor | MEDIUM -- Popular with startups, clean API |
| **Notion (Official)** | Remote MCP | Official vendor | MEDIUM -- Common for product documentation |
| **Slack (Official)** | Remote MCP | Official vendor | MEDIUM -- Channel search and summarization |
| **Asana (Official)** | Plugin + MCP | Official marketplace | MEDIUM -- Project management integration |
| **GitHub (Official)** | MCP | Official reference server | HIGH -- essential for all development |

**gdev integration**: The `claudecode` addon should detect project management tools and offer MCP configuration. Atlassian MCP is highest priority for enterprise consulting. GitHub MCP should be default-enabled.

### 9.3 Monitoring and Observability MCP Servers

| Server | What | Stars/Status |
|--------|------|-------------|
| **Datadog MCP (Official)** | Query metrics, logs, traces, APM data | Official vendor, documented |
| **Sentry MCP** | Error tracking, performance monitoring | Available in registries |
| **Grafana MCP** | Dashboard queries, alert management | Available in registries |
| **PagerDuty MCP** | Incident management | Available in registries |

**gdev integration**: Configure based on profile. Enterprise profiles should include Datadog or equivalent. The claude-code-otel stack (section 5.1) handles Claude Code's own observability; these MCP servers provide access to application-level monitoring.

### 9.4 Browser and Testing MCP Servers

| Server | What | Stars/Status |
|--------|------|-------------|
| **Playwright MCP (Microsoft)** | Browser automation, testing | 5,600 views, 414 installs on FastMCP |
| **Puppeteer (Anthropic Reference)** | Browser control | Official reference server |
| **Browserbase MCP** | Cloud browser sessions | Available in registries |

**gdev integration**: Configure Playwright MCP when e2e testing framework detected (Playwright, Cypress). Optional browser automation for web projects.

### 9.5 Search and Knowledge MCP Servers

| Server | What | Stars/Status |
|--------|------|-------------|
| **Brave Search (Anthropic Reference)** | Web search | Official reference server |
| **Exa MCP** | AI-optimized search | Popular in registries |
| **Perplexity MCP** | AI search with citations | Available in registries |

**gdev integration**: Configure one search MCP in `.mcp.json` for research/documentation tasks. Brave Search is the lowest-friction (official reference, minimal config).

---

## 10. Prompt Engineering Skills

### 10.1 ckelsoe/prompt-architect

- **URL**: https://github.com/ckelsoe/claude-skill-prompt-architect
- **What**: Transforms vague prompts into structured expert-level prompts using 7 research-backed frameworks (CO-STAR, RISEN, RISE, TIDD-EC, RTF, CoT, CoD). Analyzes prompts across 5 quality dimensions, recommends best framework for each use case.
- **Mechanism**: Skill (SKILL.md)
- **Maturity**: 156 stars. Created Nov 2025. MIT license. Python.
- **License**: MIT
- **Consulting relevance**: LOW. Prompt optimization is valuable but this is a developer tool, not something that integrates into automated workflows. Claude Code's native capabilities handle most prompt engineering needs.
- **gdev integration**: Optional skill for teams doing prompt engineering work. Not part of default configuration.

### 10.2 Piebald-AI/claude-code-system-prompts

- **URL**: https://github.com/Piebald-AI/claude-code-system-prompts
- **What**: Extracted and documented Claude Code system prompts, including all 24 built-in tool descriptions, sub-agent prompts (Plan/Explore/Task), and utility prompts. Updated for each Claude Code version.
- **Mechanism**: Reference documentation (not a tool)
- **Maturity**: Reference material, regularly updated.
- **License**: Not specified
- **Consulting relevance**: MEDIUM. Understanding Claude Code's system prompt helps write better CLAUDE.md files and skills. Useful for gdev's CLAUDE.md generation.
- **gdev integration**: Reference material for optimizing gdev's generated CLAUDE.md content. Not a bundled tool.

---

## Cross-Category Recommendations for gdev

### Tier 1: Default-Enable (High Value, Zero/Low Cost, Broad Applicability)

| Tool | Category | Why Default |
|------|----------|------------|
| **Context7 MCP** | Documentation | Zero config, zero cost, solves universal stale-docs problem |
| **Serena MCP** | Code Quality | Zero cost, 30+ languages, LSP-based accuracy |
| **GitHub MCP** | MCP Infrastructure | Essential for all development workflows |
| **PostgreSQL MCP** | Database | Most common database, read-only safe default |
| **claude-code-otel** | Observability | Team cost tracking is essential for consulting firms |

### Tier 2: Auto-Enable When Detected (Conditional on Project Type)

| Tool | Trigger | Why |
|------|---------|-----|
| **HashiCorp Agent Skills** | Terraform files detected | Official vendor, high quality |
| **antonbabenko/terraform-skill** | Terraform/OpenTofu detected | Community best practices, OpenTofu support |
| **Prisma MCP** | `prisma` in dependencies | Official ORM integration |
| **Playwright MCP** | Playwright in dependencies | E2E testing automation |
| **Atlassian MCP** | Jira/Confluence configured in profile | Enterprise PM integration |
| **ATDD plugin** | Testing framework detected | Mutation testing validation |
| **Trailmark skills** | Python 3.10+ available | Code graph analysis + mutation triage |

### Tier 3: Opt-In via Profile (Requires API Keys or Org Decision)

| Tool | Requirement | Why |
|------|------------|-----|
| **CodeRabbit plugin** | CodeRabbit API key | Commercial code review |
| **Qodo Skills** | Qodo API key | Org-level coding rules |
| **Greptile plugin** | Greptile API key | Large codebase navigation |
| **Snyk MCP** | Snyk token | Enterprise security scanning |
| **Datadog MCP** | Datadog API key | Application monitoring |
| **claude-mem** | Explicit opt-in | Session persistence (runs worker service) |
| **SBOM Management** | Compliance profile | Regulatory requirements |

### Tier 4: Reference Architecture Only (Inform gdev Design, Don't Bundle)

| Tool | Takeaway for gdev |
|------|-------------------|
| **minimal-claude** | Auto-detection pattern for linting tools |
| **everything-claude-code** | Token-efficient plugin design patterns |
| **corca-ai/claude-plugins** | Gated workflow (plan -> review -> implement) |
| **shinpr/claude-code-workflows** | Anti-shortcut detection (metronome) |
| **disler/multi-agent-observability** | Hook-based observability patterns |
| **Piebald-AI system prompts** | Optimizing generated CLAUDE.md content |
| **severity1/prompt-improver** | UserPromptSubmit hook for prompt quality |

---

## Ecosystem Maturity Assessment

### Stars Distribution (Sampled Tools)

| Range | Count | Examples |
|-------|-------|---------|
| 50,000+ | 3 | everything-claude-code (180k), MCP servers (85k), claude-mem (75k) |
| 10,000-50,000 | 4 | Context7 (55k), wshobson/agents (35k), Serena (24k), alirezarezvani/claude-skills (14.5k) |
| 1,000-10,000 | 6 | ToB skills (5.1k), Atlassian MCP (5.2k), Snyk agent-scan (2.4k), terraform-skill (1.8k), prompt-improver (1.5k), disler observability (1.4k) |
| 100-1,000 | 8 | HashiCorp (613), agents-observe (554), claude-code-skills (464), claude-code-otel (396), shinpr workflows (347), terrashark (330), devops-skills (150), SecOpsAgentKit (134) |
| <100 | 5 | ATDD (93), corca-ai (77), CodeRabbit plugin (43), prompt-optimizer (39), minimal-claude (18) |

### Key Observations

1. **The ecosystem is extremely active** -- most tools were created in 2025-2026 and are still being maintained.
2. **Star inflation is real** -- some repos with 100k+ stars are broad collections rather than production-hardened tools. Evaluate functionality over popularity.
3. **Official marketplace is curated** -- the `claude-plugins-official` repo has ~36 internal + ~15 external partner plugins. This is the quality tier.
4. **MCP servers are the growth area** -- 23,000+ servers available, but quality varies wildly. Stick to official vendor MCPs and the reference servers.
5. **Skills are the lowest-friction integration** -- SKILL.md files are just markdown instructions. No runtime, no dependencies, no security surface. Best for most gdev addons.
6. **Hooks and plugins have security implications** -- they execute code on every tool call. Managed settings should restrict to vetted hooks only in enterprise deployments.

---

## Sources

- https://github.com/anthropics/claude-plugins-official
- https://github.com/anthropics/claude-code/tree/main/plugins
- https://github.com/modelcontextprotocol/servers
- https://github.com/hashicorp/agent-skills
- https://github.com/antonbabenko/terraform-skill
- https://github.com/thedotmack/claude-mem
- https://github.com/upstash/context7
- https://github.com/oraios/serena
- https://github.com/zilliztech/claude-context
- https://github.com/trailofbits/skills
- https://github.com/AgentSecOps/SecOpsAgentKit
- https://github.com/ColeMurray/claude-code-otel
- https://github.com/disler/claude-code-hooks-multi-agent-observability
- https://github.com/simple10/agents-observe
- https://github.com/swingerman/atdd
- https://github.com/ahmedasmar/devops-claude-skills
- https://github.com/levnikolaevich/claude-code-skills
- https://github.com/KenKaiii/minimal-claude
- https://github.com/coderabbitai/claude-plugin
- https://github.com/affaan-m/everything-claude-code
- https://github.com/wshobson/agents
- https://github.com/corca-ai/claude-plugins
- https://github.com/shinpr/claude-code-workflows
- https://github.com/ckelsoe/claude-skill-prompt-architect
- https://github.com/severity1/claude-code-prompt-improver
- https://github.com/snyk/agent-scan
- https://github.com/LukasNiessen/terrashark
- https://github.com/sooperset/mcp-atlassian
- https://github.com/atlassian/atlassian-mcp-server
- https://github.com/alirezarezvani/claude-skills
- https://github.com/Piebald-AI/claude-code-system-prompts
- https://github.com/TechNickAI/claude_telemetry
- https://github.com/nexus-labs-automation/agent-observability
- https://github.com/OpenAEC-Foundation/Docker-Claude-Skill-Package
- https://github.com/clear-solutions/unit-tests-skills
- https://www.prisma.io/mcp
- https://code.claude.com/docs/en/code-review
- https://code.claude.com/docs/en/monitoring-usage
- https://registry.modelcontextprotocol.io/
- https://blog.trailofbits.com/2026/04/23/trailmark-turns-code-into-graphs/
