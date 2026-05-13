# MCP Servers & Developer Experience Tools Ecosystem Research

Research date: 2026-05-12. Conducted for the gdev secure development environment bootstrap implementation plan. Focus: tools that enhance an AI-powered, security-hardened, multi-language consulting development environment serving hundreds of engineers across diverse client engagements.

**Existing plan integrations** (not re-evaluated here): semble (semantic code search MCP), Socket.dev MCP (supply chain scoring), attach-guard (package guardrails), Trail of Bits skills/config, Security Phoenix, Claude Code Security Review GitHub Action.

---

## Table of Contents

1. [Database MCP Servers](#1-database-mcp-servers)
2. [API Documentation MCP Servers](#2-api-documentation-mcp-servers)
3. [Cloud Provider MCP Servers](#3-cloud-provider-mcp-servers)
4. [Monitoring/Observability MCP Servers](#4-monitoringobservability-mcp-servers)
5. [Git/GitHub/GitLab MCP Servers](#5-gitgithubgitlab-mcp-servers)
6. [Container/Orchestration MCP Servers](#6-containerorchestration-mcp-servers)
7. [CI/CD MCP Servers](#7-cicd-mcp-servers)
8. [Knowledge/Documentation MCP Servers](#8-knowledgedocumentation-mcp-servers)
9. [Communication MCP Servers](#9-communication-mcp-servers)
10. [Security-Specific MCP Servers](#10-security-specific-mcp-servers)
11. [Infrastructure as Code MCP Servers](#11-infrastructure-as-code-mcp-servers)
12. [Developer Experience Tools](#12-developer-experience-tools)
13. [Relevance Rankings & Integration Recommendations](#13-relevance-rankings--integration-recommendations)

---

## 1. Database MCP Servers

### 1.1 MCP Toolbox for Databases (Google)

| Field | Value |
|-------|-------|
| **URL** | https://github.com/googleapis/mcp-toolbox |
| **What** | Universal database MCP gateway supporting 15+ database engines through a single interface. Provides prebuilt tools (list_tables, execute_sql) plus custom tool definitions with connection pooling and integrated auth. |
| **Type** | MCP server |
| **Stars** | 15,200 |
| **Language** | Go |
| **License** | Apache 2.0 |
| **Last updated** | May 2026 (v1.2.0) |
| **Install** | Binary, Homebrew, Docker, npm, `go install` |
| **Platforms** | macOS, Linux, Windows, Docker |
| **Databases** | PostgreSQL, MySQL, SQL Server, Oracle, MongoDB, Redis, Elasticsearch, CockroachDB, ClickHouse, Couchbase, Neo4j, Snowflake, Trino, AlloyDB, BigQuery, Cloud SQL, Spanner, Firestore |
| **Consulting relevance** | **Critical.** One server covers every database a consulting team will encounter across clients. YAML-based tool definitions make it easy to pre-configure per-client database access patterns. The Go implementation matches gdev's stack. |

### 1.2 Postgres MCP Pro

| Field | Value |
|-------|-------|
| **URL** | https://github.com/crystaldba/postgres-mcp |
| **What** | PostgreSQL-specific MCP server with read/write access controls, performance analysis, index tuning via industrial-strength algorithms, query plan explanation, and database health monitoring. |
| **Type** | MCP server |
| **Stars** | 2,700 |
| **Language** | Python |
| **License** | MIT |
| **Last updated** | May 2025 (v0.3.0) |
| **Platforms** | macOS, Linux, Windows (Python) |
| **Consulting relevance** | **High for PostgreSQL-heavy clients.** The restricted read-only mode is essential for production safety. Performance analysis and index tuning are valuable for client engagements focused on optimization. |

### 1.3 MongoDB MCP Server (Official)

| Field | Value |
|-------|-------|
| **URL** | https://github.com/mongodb-js/mongodb-mcp-server |
| **What** | Official MongoDB MCP server for MongoDB databases and Atlas clusters. Supports CRUD operations, aggregation pipelines, vector search indexes, and Atlas cluster management. |
| **Type** | MCP server |
| **Stars** | ~2,000 (estimated from search context) |
| **Language** | TypeScript |
| **License** | Apache 2.0 |
| **Last updated** | Winter 2026 edition |
| **Platforms** | macOS, Linux, Windows (Node.js) |
| **Consulting relevance** | **High.** MongoDB is common in consulting engagements, especially Node.js stacks. Official support means long-term maintenance. |

### 1.4 Redis MCP Server (Official)

| Field | Value |
|-------|-------|
| **URL** | https://github.com/redis/mcp-redis |
| **What** | Official Redis MCP server with natural language queries. Covers strings, hashes, lists, sets, sorted sets, streams, JSON, pub/sub, vector search, and consumer groups. |
| **Type** | MCP server |
| **Stars** | 509 |
| **Language** | Python |
| **License** | MIT |
| **Last updated** | March 2026 (v0.5.0) |
| **Platforms** | macOS, Linux, Windows (Python) |
| **Consulting relevance** | **Medium.** Redis is ubiquitous but the MCP server is more useful for debugging/inspection than daily development. Useful for session debugging and cache analysis during client incidents. |

### 1.5 Elasticsearch MCP Server (Official, Deprecated)

| Field | Value |
|-------|-------|
| **URL** | https://github.com/elastic/mcp-server-elasticsearch |
| **What** | Elasticsearch MCP with index listing, mapping retrieval, search (DSL + ES|QL), and shard info. **Deprecated** in favor of Elastic Agent Builder MCP endpoint (Elasticsearch 9.2.0+). |
| **Type** | MCP server (deprecated) |
| **Stars** | 658 |
| **Language** | Rust |
| **License** | Apache 2.0 |
| **Last updated** | October 2025 (v0.4.6, security fixes only) |
| **Consulting relevance** | **Low for new deployments** (deprecated). Teams on Elasticsearch 9.2+ should use the Elastic Agent Builder MCP endpoint instead. |

### Recommendation

**MCP Toolbox for Databases is the clear winner** for a consulting firm. It covers all databases through one server, uses YAML configuration (easy to template in gdev), and is backed by Google with active development. For PostgreSQL-heavy clients, add Postgres MCP Pro for its performance analysis capabilities. MongoDB and Redis official servers are worth offering as optional additions in the gdev wizard.

---

## 2. API Documentation MCP Servers

### 2.1 OpenAPI MCP Server (AWS Labs)

| Field | Value |
|-------|-------|
| **URL** | https://awslabs.github.io/mcp/servers/openapi-mcp-server |
| **What** | Dynamically generates MCP tools from OpenAPI 3.x specs. Supports multi-spec composition, tag-based filtering, auth (Basic, Bearer, API Key, Cognito), and 70-75% token reduction through dynamic prompt generation. |
| **Type** | MCP server |
| **Language** | Python |
| **License** | Apache 2.0 (AWS Labs) |
| **Install** | `pip install awslabs.openapi-mcp-server` |
| **Platforms** | macOS, Linux, Windows (Python) |
| **Consulting relevance** | **Critical.** Consulting teams constantly onboard onto new client APIs. Point this at a client's OpenAPI spec and get instant MCP tools for every endpoint. Multi-spec composition means combining multiple client microservice specs into one MCP server. The token reduction is significant for large API surfaces. |

### 2.2 API Docs MCP

| Field | Value |
|-------|-------|
| **URL** | https://mcpservers.org/servers/elifuzz/api-docs-mcp |
| **What** | Loads OpenAPI/Swagger (YAML/JSON) and GraphQL schemas from local files or remote URLs. Provides schema exploration and endpoint documentation. |
| **Type** | MCP server |
| **Consulting relevance** | **Medium.** Useful for GraphQL-heavy clients where the AWS OpenAPI server doesn't apply. Less mature than the AWS option for REST APIs. |

### 2.3 Swagger MCP (Vizioz)

| Field | Value |
|-------|-------|
| **URL** | https://github.com/Vizioz/Swagger-MCP |
| **What** | Connects to a Swagger specification and helps AI build required models to generate an MCP server for that service. Meta-tool for generating MCP servers from API specs. |
| **Type** | MCP server generator |
| **Consulting relevance** | **Low.** The AWS OpenAPI MCP Server already handles dynamic tool generation without a separate generation step. |

### 2.4 Context7

| Field | Value |
|-------|-------|
| **URL** | https://github.com/upstash/context7 |
| **What** | Provides up-to-date, version-specific library documentation and code examples for 33,000+ libraries. Resolves library IDs and fetches current docs directly into the AI context. Prevents hallucination of outdated or non-existent APIs. |
| **Type** | MCP server (remote hosted) |
| **Stars** | 53,300 |
| **License** | Apache 2.0 |
| **Last updated** | Active (890K weekly npm downloads) |
| **Install** | Remote: `https://mcp.context7.com/mcp` with API key |
| **Platforms** | Any (remote server) |
| **Consulting relevance** | **Critical.** Consulting teams work across dozens of libraries and frameworks. Context7 eliminates the "AI suggests deprecated API" problem that wastes developer time. The 33,000+ library coverage means it works regardless of client tech stack. This is one of the highest-impact MCP servers for daily developer productivity. |

### Recommendation

**AWS OpenAPI MCP Server + Context7** is the recommended combination. OpenAPI server handles client-specific API documentation; Context7 handles third-party library documentation. Together they cover the two main documentation needs in consulting.

---

## 3. Cloud Provider MCP Servers

### 3.1 AWS MCP Servers (Official, 55+ servers)

| Field | Value |
|-------|-------|
| **URL** | https://github.com/awslabs/mcp / https://awslabs.github.io/mcp/ |
| **What** | Suite of 55+ official MCP servers spanning the full AWS product catalog. Key servers include: AWS API (CLI operations), CloudWatch (metrics/logs/alarms), EKS (Kubernetes), ECS (containers), S3/DynamoDB/Aurora (data), IAM (access management), Lambda (serverless), Billing/Pricing (cost), CloudTrail (audit). |
| **Type** | MCP server collection |
| **License** | Apache 2.0 |
| **Last updated** | May 2026 (GA announced May 6, 2026) |
| **Language** | Various (Go, Python, TypeScript) |
| **Install** | Per-server via npm, pip, or binary |
| **Platforms** | macOS, Linux, Windows |
| **Consulting relevance** | **Critical for AWS clients.** The most mature cloud provider MCP ecosystem. gdev should offer AWS profile selection that auto-configures the relevant subset of servers (e.g., "AWS Lambda developer" gets Lambda + CloudWatch + S3 + IAM servers). |

**Key servers for consulting:**
- **AWS Knowledge MCP Server** (GA) -- up-to-date AWS docs, code samples, best practices
- **AWS CloudWatch MCP Server** -- metrics, alarms, log analysis
- **Amazon EKS MCP Server** -- Kubernetes cluster management
- **AWS IAM MCP Server** -- user, role, policy management
- **AWS Billing MCP Server** -- cost insights (critical for cost-conscious clients)
- **AWS CloudTrail MCP Server** -- API activity audit
- **AWS Well-Architected Security Assessment** -- security posture analysis

### 3.2 Google Cloud MCP Servers (Official, 40+ products)

| Field | Value |
|-------|-------|
| **URL** | https://docs.cloud.google.com/mcp/supported-products |
| **What** | Official MCP servers covering 40+ GCP products including Compute Engine, Cloud Run, GKE, BigQuery, Spanner, Cloud SQL, Firestore, Cloud Logging, Cloud Monitoring, Cloud Trace, Pub/Sub, Cloud Storage, AlloyDB, and more. Also covers Google Workspace (Drive, Gmail, Calendar, Chat). |
| **Type** | MCP server collection |
| **Status** | Remote MCP servers in preview (as of March 2026) |
| **Platforms** | Any (remote servers) |
| **Consulting relevance** | **Critical for GCP clients.** Coverage is comprehensive though less mature than AWS. The Workspace MCP servers (Drive, Gmail, Calendar) have cross-cloud utility for any team using Google Workspace. |

### 3.3 Azure DevOps MCP Server (Official)

| Field | Value |
|-------|-------|
| **URL** | https://github.com/microsoft/azure-devops-mcp |
| **What** | Official Microsoft MCP server providing access to Azure DevOps work items, pull requests, builds, test plans, and documentation. Supports both local and remote (hosted) modes with OAuth and PAT authentication. |
| **Type** | MCP server |
| **Status** | Remote server in public preview; local server GA |
| **Last updated** | April 2026 |
| **Platforms** | macOS, Linux, Windows |
| **Consulting relevance** | **Critical for Azure/enterprise clients.** Many consulting firm clients use Azure DevOps. The remote hosted version eliminates local setup. WIQL query support enables complex work item searches. |

### Recommendation

gdev should offer **cloud provider profiles** that auto-configure the relevant MCP server subset. A "cloud provider" wizard step with AWS/GCP/Azure/Multi-cloud/None options is the right UX. Most consulting engagements use one primary cloud.

---

## 4. Monitoring/Observability MCP Servers

### 4.1 Datadog MCP Server (Official)

| Field | Value |
|-------|-------|
| **URL** | https://docs.datadoghq.com/bits_ai/mcp_server/ |
| **What** | Official Datadog MCP server bridging AI agents to metrics, logs, traces, APM, monitors, dashboards, and security signals. Rate-limited (50 req/10s, 5K daily, 50K monthly). Supports Claude Code, Cursor, VS Code, Gemini CLI. |
| **Type** | MCP server (remote hosted by Datadog) |
| **Auth** | Datadog API/App keys |
| **Platforms** | Any (remote) |
| **Consulting relevance** | **High for Datadog clients.** Datadog is the most common observability platform in consulting engagements. AI-assisted incident investigation ("show me error rates for the checkout service") is a high-value workflow. Rate limits are generous for development use. |

### 4.2 Grafana MCP Server (Official)

| Field | Value |
|-------|-------|
| **URL** | https://github.com/grafana/mcp-grafana |
| **What** | Official Grafana MCP server with dashboard management, multi-datasource queries (Prometheus, Loki, ClickHouse, CloudWatch, Elasticsearch, Snowflake, InfluxDB, Graphite), alerting, incident management, on-call schedules, and dashboard PNG rendering. |
| **Type** | MCP server |
| **Stars** | 3,000 |
| **Language** | Go |
| **License** | Apache 2.0 |
| **Last updated** | May 2026 (v0.14.0) |
| **Install** | uvx, Docker, binary, Helm |
| **Platforms** | macOS, Linux, Windows, Kubernetes |
| **Consulting relevance** | **High.** Grafana is the open-source observability standard. The multi-datasource support means one MCP server covers Prometheus, Loki, CloudWatch, and more. The Go implementation and Apache 2.0 license align well with gdev. Dashboard PNG rendering is useful for report generation. |

### 4.3 Prometheus MCP Servers (Community)

| Field | Value |
|-------|-------|
| **URL** | https://github.com/pab1it0/prometheus-mcp-server (most mature); also giantswarm/mcp-prometheus (18 tools), yanmxa/prometheus-mcp-server |
| **What** | Translates natural language to PromQL and executes against Prometheus. Multiple implementations available. giantswarm version exposes 18 read-only tools wrapping the full Prometheus HTTP API. |
| **Type** | MCP server (multiple implementations) |
| **Platforms** | macOS, Linux (Python or Go depending on implementation) |
| **Consulting relevance** | **Medium.** Most teams using Prometheus also use Grafana, and the Grafana MCP server already supports Prometheus datasources. Standalone Prometheus MCP is useful for teams without Grafana or for direct metric exploration. |

### Recommendation

**Grafana MCP Server** is the primary recommendation -- it covers Prometheus, Loki, and other datasources through one server. **Datadog MCP** as an alternative for Datadog-shop clients. Both should be wizard options.

---

## 5. Git/GitHub/GitLab MCP Servers

### 5.1 GitHub MCP Server (Official)

| Field | Value |
|-------|-------|
| **URL** | https://github.com/github/github-mcp-server |
| **What** | Official GitHub MCP server with 100+ tools: repository management, PR review, issue tracking, code search, GitHub Actions monitoring, Dependabot alerts, security scanning, discussions, projects, notifications. Supports read-only mode, dynamic discovery, lockdown mode, and GitHub Enterprise. |
| **Type** | MCP server |
| **Stars** | 29,800 |
| **Language** | Go |
| **License** | MIT |
| **Last updated** | May 2026 (v1.0.4) |
| **Install** | Docker, remote (OAuth), binary |
| **Platforms** | macOS, Linux, Windows |
| **Consulting relevance** | **Critical.** The most-starred MCP server for good reason. Consulting teams live in GitHub. 100+ tools means comprehensive coverage. Read-only mode is important for security-conscious client repos. GitHub Enterprise support matters for enterprise clients. **Note:** Claude Code already has built-in GitHub integration via `gh` CLI, so this MCP server adds value primarily for advanced workflows (cross-repo search, Dependabot alert triage, Actions monitoring) beyond what the built-in provides. |

### 5.2 GitLab MCP Server

| Field | Value |
|-------|-------|
| **URL** | https://github.com/zereight/gitlab-mcp |
| **What** | Comprehensive GitLab MCP with 154+ tools: projects, merge requests, issues, pipelines, CI/CD validation, wikis, releases, milestones, labels, deployments, webhooks, work items, GraphQL execution. Supports PAT, OAuth2, and remote auth. |
| **Type** | MCP server |
| **Stars** | 1,500 |
| **Language** | TypeScript |
| **License** | MIT |
| **Last updated** | May 2026 (v2.1.10) |
| **Install** | npm, Docker |
| **Platforms** | macOS, Linux, Windows |
| **Consulting relevance** | **High for GitLab clients.** Many enterprise clients use GitLab. 154 tools with full pipeline management is comprehensive. gdev wizard should detect `.gitlab-ci.yml` and offer this server. |

### 5.3 Azure DevOps MCP Server

(See Section 3.3 above -- covers PR review and work items for Azure DevOps clients.)

### Recommendation

**GitHub MCP Server** as default (most consulting work uses GitHub). **GitLab MCP** and **Azure DevOps MCP** as alternatives detected from project files. All three should be wizard options with auto-detection.

---

## 6. Container/Orchestration MCP Servers

### 6.1 Kubernetes MCP Server

| Field | Value |
|-------|-------|
| **URL** | https://github.com/containers/kubernetes-mcp-server |
| **What** | Go-native Kubernetes/OpenShift MCP server. Covers pods (list, get, delete, logs, exec, run), generic CRUD on any resource, namespaces, events, Helm chart management, Tekton pipelines, Kiali service mesh, and KubeVirt VMs. No external kubectl/helm dependencies required. |
| **Type** | MCP server |
| **Stars** | 1,600 |
| **Language** | Go |
| **License** | Apache 2.0 |
| **Last updated** | May 2026 (v0.0.62) |
| **Install** | npm, pip, Docker, binary, Helm |
| **Platforms** | macOS, Linux, Windows |
| **Consulting relevance** | **High for K8s-using teams.** Self-contained (no kubectl dependency), multi-platform, and covers OpenShift (important for enterprise clients). Helm chart management is particularly valuable -- AI-assisted helm deployments reduce onboarding friction for new team members. |

### 6.2 Amazon EKS MCP Server (AWS Official)

| Field | Value |
|-------|-------|
| **URL** | Part of AWS MCP collection |
| **What** | Fully managed EKS MCP server for deploying applications, troubleshooting, and upgrading clusters using natural language. |
| **Type** | MCP server (managed) |
| **Status** | Preview |
| **Consulting relevance** | **Medium.** Useful for AWS EKS-specific workflows but the generic Kubernetes MCP server covers broader use cases. |

### 6.3 Docker MCP Gateway

| Field | Value |
|-------|-------|
| **What** | Container-native approach: isolates each MCP server in its own container, exposes through a single gateway endpoint. |
| **Type** | MCP infrastructure |
| **Consulting relevance** | **Medium.** Interesting for enterprise security (MCP server isolation) but adds complexity. Worth watching for security-conscious deployments. |

### Recommendation

**Kubernetes MCP Server** for teams doing K8s development. Auto-detect via presence of `kubeconfig`, `Helm` charts, or K8s manifests.

---

## 7. CI/CD MCP Servers

### 7.1 Jenkins MCP Server

| Field | Value |
|-------|-------|
| **URL** | https://github.com/kud/mcp-jenkins |
| **What** | Jenkins MCP with 37 tools covering ~95% of Jenkins API: job operations (11 tools), build operations (9), testing/artifacts (3), queue management (2), system/nodes (8), views (2), instance admin (2). Multi-instance support. |
| **Type** | MCP server |
| **Stars** | 7 |
| **Language** | TypeScript |
| **License** | MIT |
| **Last updated** | 2026 (62 commits) |
| **Install** | Node.js 20+ |
| **Platforms** | macOS, Linux, Windows |
| **Consulting relevance** | **Medium.** Jenkins is declining (28% market share in 2026 vs GitHub Actions' 33%) but still common in enterprise/legacy clients. Very low star count suggests early maturity -- monitor but don't recommend as default. |

### 7.2 GitLab CI (via GitLab MCP)

The GitLab MCP server (Section 5.2) includes full pipeline management, CI/CD validation, job management, and artifact handling as part of its 154+ tool set.

### 7.3 GitHub Actions (via GitHub MCP)

The GitHub MCP server (Section 5.1) includes Actions workflow monitoring, run management, and job status as part of its 100+ tool set.

### Recommendation

CI/CD is already covered by the source control MCP servers (GitHub MCP handles Actions, GitLab MCP handles GitLab CI). Jenkins MCP is too immature for default recommendation but should be documented as available for legacy clients.

---

## 8. Knowledge/Documentation MCP Servers

### 8.1 Notion MCP Server (Official)

| Field | Value |
|-------|-------|
| **URL** | https://github.com/makenotion/notion-mcp-server |
| **What** | Official Notion MCP with 22 tools: query/create/update data sources, page operations, comment management, search. Version 2.0+ uses "data sources" as primary abstraction. |
| **Type** | MCP server |
| **Stars** | 4,300 |
| **Language** | TypeScript |
| **License** | MIT |
| **Last updated** | January 2026 (v2.1.0) |
| **Install** | `npx @notionhq/notion-mcp-server`, Docker |
| **Platforms** | macOS, Linux, Windows |
| **Consulting relevance** | **High.** Many consulting firms use Notion for project documentation, runbooks, and knowledge bases. Giving AI agents access to institutional knowledge reduces "where's the doc for X?" friction. Official support ensures stability. |

### 8.2 Atlassian Confluence MCP Server (Official Remote)

| Field | Value |
|-------|-------|
| **URL** | https://www.atlassian.com/blog/announcements/remote-mcp-server (Official); https://github.com/aashari/mcp-server-atlassian-confluence (Community) |
| **What** | **Official (Remote):** Atlassian's hosted remote MCP server for Jira + Confluence access. **Community:** Node.js server with 5 generic HTTP tools, TOON output format (30-60% token savings), JMESPath filtering. |
| **Type** | MCP server (remote official + community local) |
| **Stars** | 53 (community); official is hosted |
| **Last updated** | Official: active; Community: December 2025 |
| **Platforms** | Any (remote); macOS/Linux/Windows (community) |
| **Consulting relevance** | **High for enterprise clients.** Confluence is the dominant enterprise wiki. Many consulting firm clients have years of documentation in Confluence. The official remote server (Jira + Confluence combined) is the preferred option -- eliminates local setup. |

### 8.3 Sentry MCP Server (Official)

| Field | Value |
|-------|-------|
| **URL** | https://github.com/getsentry/sentry-mcp |
| **What** | Official Sentry MCP for issues, errors, projects, and Seer AI analysis. Remote hosted at `mcp.sentry.dev/mcp` with OAuth -- nothing to install. 16 tool calls plus prompts. |
| **Type** | MCP server (remote hosted) |
| **Platforms** | Any (remote) |
| **Consulting relevance** | **High.** Sentry is the dominant error tracking platform. AI-assisted error investigation ("what's causing the spike in this error?") accelerates debugging. Zero-install remote hosting makes it trivial to configure in gdev. |

### Recommendation

**Notion MCP** (for Notion-using firms) and **Atlassian Remote MCP** (for Confluence/Jira clients) are both high-value. **Sentry MCP** should be default for any project using Sentry. gdev wizard should detect Sentry DSN and auto-configure.

---

## 9. Communication MCP Servers

### 9.1 Slack MCP Server (Official)

| Field | Value |
|-------|-------|
| **URL** | https://docs.slack.dev/ai/slack-mcp-server/ |
| **What** | Official Slack MCP with message search, channel reading, thread reading, user profile lookup, message sending, canvas management, and workflow automation. Real-Time Search API GA as of 2026. MCP tool calls grown 25x since October 2025 launch. |
| **Type** | MCP server (official, remote) |
| **Status** | GA |
| **Platforms** | Any (remote) |
| **Consulting relevance** | **Medium-High.** Slack is ubiquitous in consulting firms. Use cases: AI searching project channels for prior decisions, creating tickets from Slack threads, notifying teams. However, message sending capabilities require careful scoping -- default should be read-only for security. |

### 9.2 Microsoft Teams MCP

| Field | Value |
|-------|-------|
| **What** | Enterprise messaging integration for Teams workspaces with channels, DMs, and bot capabilities. |
| **Type** | MCP server |
| **Status** | Available (enterprise) |
| **Consulting relevance** | **Medium for Teams-using clients.** Less common in developer-focused workflows than Slack. Include as wizard option for enterprise clients. |

### Recommendation

**Slack MCP** as opt-in with read-only default. Communication MCP servers have the highest risk of misuse (AI sending messages on behalf of developer) so they should default to read-only and require explicit opt-in.

---

## 10. Security-Specific MCP Servers

### 10.1 Snyk MCP Server (via Snyk CLI)

| Field | Value |
|-------|-------|
| **URL** | https://github.com/sammcj/mcp-snyk (community); Official: integrated into Snyk CLI |
| **What** | Security scanning MCP with `snyk_sca_scan` (open source vulnerabilities), `snyk_code_scan` (proprietary code), `snyk_iac_scan` (infrastructure as code), `snyk_container_scan` (containers), `snyk_sbom_scan` (SBOM analysis), `snyk_aibom` (AI Bill of Materials). |
| **Type** | MCP server (via CLI) |
| **Auth** | Snyk token |
| **Platforms** | macOS, Linux, Windows |
| **Consulting relevance** | **High for Snyk-licensed teams.** Comprehensive scanning across all categories. The AIBOM tool is unique -- generates AI-specific bill of materials. Community MCP server available for teams without Snyk enterprise licenses. |

### 10.2 CVE Search MCP Server

| Field | Value |
|-------|-------|
| **URL** | https://github.com/roadwy/cve-search_mcp |
| **What** | MCP server for querying CVE-Search API: browse vendors/products, retrieve CVE details by ID, access recent CVEs with CAPEC, CWE, and CPE expansions. |
| **Type** | MCP server |
| **Platforms** | macOS, Linux, Windows |
| **Consulting relevance** | **Medium.** Useful for security-focused engagements and incident response. Quick CVE lookup during code review is a practical workflow. |

### 10.3 NVD MCP Server

| Field | Value |
|-------|-------|
| **URL** | https://www.pulsemcp.com/servers/marcoeg-nvd |
| **What** | NIST National Vulnerability Database access via `get_cve` and `search_cve` tools. |
| **Type** | MCP server |
| **Consulting relevance** | **Medium.** Complementary to CVE Search. NVD provides CVSS scoring and severity data. |

### 10.4 Snyk Agent Scan

| Field | Value |
|-------|-------|
| **URL** | https://github.com/snyk/agent-scan |
| **What** | Security scanner specifically for AI agents, MCP servers, and agent skills. Discovers and scans agent components on your machine for prompt injections, sensitive data handling, and malware payloads. |
| **Type** | CLI tool |
| **Consulting relevance** | **High.** Meta-security: scans the MCP servers themselves for vulnerabilities. Valuable for a security-hardened development environment that uses many MCP servers. gdev could run this as a post-configuration validation step. |

### Recommendation

**Socket.dev MCP** (already planned) + **Snyk MCP** (for licensed teams) + **Snyk Agent Scan** (for MCP server validation). CVE/NVD MCP servers are nice-to-have for security-focused engagements.

---

## 11. Infrastructure as Code MCP Servers

### 11.1 Terraform MCP Server (Official HashiCorp)

| Field | Value |
|-------|-------|
| **URL** | https://github.com/hashicorp/terraform-mcp-server |
| **What** | Official Terraform MCP providing registry API integration: provider schemas, module interfaces, workspace details, variables, policies, and org-level settings. 35+ tools. |
| **Type** | MCP server |
| **Stars** | ~1,300 |
| **License** | MPL 2.0 |
| **Platforms** | macOS, Linux, Windows |
| **Consulting relevance** | **High.** Terraform is already a Tier 1 ecosystem in gdev. This MCP server accelerates IaC development by providing registry metadata, schema lookups, and module interface exploration. Auto-detect via `.tf` files. |

### 11.2 Pulumi MCP Server (Official)

| Field | Value |
|-------|-------|
| **URL** | https://www.pulumi.com/docs/ai/mcp-server/ |
| **What** | Official Pulumi MCP with stack/resource queries, cross-org resource search, Pulumi Registry access, policy violation reports, and Neo delegation for infrastructure tasks. Includes prompts for common workflows (deploy-to-aws, convert-terraform-to-typescript). |
| **Type** | MCP server (local + remote) |
| **Platforms** | macOS, Linux, Windows |
| **Consulting relevance** | **Medium.** Pulumi is less common than Terraform in consulting but growing, especially in TypeScript-heavy teams. The convert-terraform-to-typescript prompt is useful for migration engagements. |

### Recommendation

**Terraform MCP** for Terraform users (auto-detect `.tf` files). **Pulumi MCP** as opt-in for Pulumi teams. Both align with gdev's Tier 1 ecosystem coverage.

---

## 12. Developer Experience Tools

### 12.1 Environment Validation & Drift Detection

#### 12.1.1 Sentry devenv

| Field | Value |
|-------|-------|
| **URL** | https://github.com/getsentry/devenv |
| **What** | Unified dev environment management with `bootstrap`, `fetch`, `sync`, and `doctor` commands. Isolates dev environment in `[reporoot]/.devenv`. Repo-specific health checks and fixes via `devenv/checks`. |
| **Type** | CLI tool / library |
| **Language** | Python |
| **Version** | 1.28.0 |
| **Consulting relevance** | **Reference architecture, not a direct dependency.** gdev's `gdev doctor` command should implement similar patterns: repo-specific checks, guided fixes, diagnostic output. The check/fix pattern is proven at Sentry's scale. |

#### 12.1.2 dev-env-health-check

| Field | Value |
|-------|-------|
| **URL** | https://github.com/reggienitro/dev-env-health-check |
| **What** | AI-powered development environment validation tool specifically for Claude Code setups, MCP servers, and API keys. |
| **Type** | CLI tool |
| **Consulting relevance** | **Low as dependency, high as reference.** Very targeted at Claude Code environments, which aligns with gdev's goals. Review for inspiration on Claude Code-specific validation checks. |

### 12.2 Secret Management

#### 12.2.1 Infisical

| Field | Value |
|-------|-------|
| **URL** | https://github.com/infisical/infisical |
| **What** | Open-source secrets management platform. CLI injects secrets into local development and CI/CD. Detects 140+ secret types in files. Pre-commit hook prevents secret commits. Client SDKs in Node, Python, Go, Ruby, Java, .NET. Self-hosted (PostgreSQL + Redis) or cloud. |
| **Type** | Platform + CLI |
| **Stars** | 26,800 |
| **Language** | TypeScript |
| **License** | MIT (core) |
| **Platforms** | macOS, Linux, Windows, Docker, Kubernetes |
| **Consulting relevance** | **High.** The open-source, self-hostable nature is critical for a consulting firm managing secrets across multiple client projects. CLI secret injection into local dev is the key workflow. The secret scanning + pre-commit hooks complement gdev's existing pre-commit hook suite. MIT license means no licensing complexity across client engagements. |

#### 12.2.2 dotenvx

| Field | Value |
|-------|-------|
| **URL** | https://www.dotenv.org/ |
| **What** | Encrypts `.env` files. Drop-in replacement for dotenv. Cross-platform, multi-environment, secrets-as-code via `.env.vault` files that are safe to commit. Language-agnostic encryption. |
| **Type** | CLI tool |
| **License** | BSD-2-Clause |
| **Platforms** | macOS, Linux, Windows |
| **Consulting relevance** | **Medium.** Simpler than Infisical for small projects. The "commit encrypted .env.vault" pattern is practical for consulting where you can't always set up a full secrets platform per client. Good lightweight option alongside Infisical. |

#### 12.2.3 devenv SecretSpec (devenv.sh 2.0)

| Field | Value |
|-------|-------|
| **URL** | https://devenv.sh/ |
| **What** | devenv 2.0 ships SecretSpec 0.7.2 for declarative, provider-agnostic secrets management. Declare secrets in `secretspec.toml`; pluggable backends resolve them. |
| **Type** | Built into devenv.sh |
| **Consulting relevance** | **High.** gdev already generates devenv configurations. SecretSpec integration means secrets management comes for free with the devenv addon. This is the recommended path for devenv-based projects. |

### 12.3 Template/Scaffolding Tools

#### 12.3.1 Copier

| Field | Value |
|-------|-------|
| **URL** | https://github.com/copier-org/copier |
| **What** | Library and CLI for rendering project templates with lifecycle management. Templates can be updated when the upstream template evolves. Supports Git URLs and local paths. Jinja2 templating with conditional logic. |
| **Type** | CLI tool |
| **Stars** | 3,400 |
| **Language** | Python |
| **License** | MIT |
| **Platforms** | macOS, Linux, Windows (Python 3.10+, Git 2.27+) |
| **Consulting relevance** | **Medium.** gdev itself IS the scaffolding tool for dev environments. Copier would be useful for project template scaffolding (new service templates, new microservice boilerplate) which is a complementary but separate concern. Could be recommended in gdev's generated CLAUDE.md for teams wanting project templates. |

#### 12.3.2 Dev Containers CLI

| Field | Value |
|-------|-------|
| **URL** | https://github.com/devcontainers/cli |
| **What** | Reference implementation CLI for the Dev Containers specification. Creates and configures dev containers from `devcontainer.json`. Modular ecosystem of Features (reusable config units) and Templates. |
| **Type** | CLI tool |
| **Language** | TypeScript |
| **License** | MIT |
| **Platforms** | macOS, Linux (x64, arm64); requires Node.js 20+ |
| **Consulting relevance** | **Medium-Low.** gdev uses devenv.sh (Nix-based), not dev containers. However, some client engagements may mandate dev containers. gdev could detect `devcontainer.json` and skip devenv generation, instead configuring Claude Code within the existing dev container setup. |

### 12.4 Local Development Orchestration

#### 12.4.1 Docker Compose

| Field | Value |
|-------|-------|
| **What** | Standard multi-container orchestration. `depends_on` with `service_healthy` ensures dependency startup order. Profiles enable optional service subsets. |
| **Type** | CLI tool (bundled with Docker) |
| **License** | Apache 2.0 |
| **Platforms** | macOS, Linux, Windows |
| **Consulting relevance** | **Critical.** The universal local service orchestration tool. Every consulting project with dependent services uses Docker Compose. gdev should detect `docker-compose.yml` / `compose.yml` and configure accordingly. Already supported as Tier 1 ecosystem (Docker). |

#### 12.4.2 Tilt (Docker-owned)

| Field | Value |
|-------|-------|
| **URL** | https://tilt.dev/ |
| **What** | Local Kubernetes development with web dashboard, live updates, and multi-service management. Acquired by Docker. Uses Tiltfile (Starlark-based) configuration. |
| **Type** | CLI tool + web UI |
| **License** | Apache 2.0 |
| **Platforms** | macOS, Linux, Windows |
| **Consulting relevance** | **Medium.** Only relevant for Kubernetes-native development workflows. Steeper learning curve than Docker Compose. Include as optional for K8s teams. |

#### 12.4.3 DevSpace

| Field | Value |
|-------|-------|
| **URL** | https://www.devspace.sh/ |
| **What** | Kubernetes development CLI with file sync, port forwarding, and dev containers. Standard YAML configuration (lower learning curve than Tilt). Supports cross-repo dependency management. |
| **Type** | CLI tool |
| **License** | Apache 2.0 |
| **Platforms** | macOS, Linux, Windows |
| **Consulting relevance** | **Medium.** Better learning curve than Tilt for onboarding new team members. Worth recommending for K8s-heavy consulting engagements. |

### 12.5 Developer Portal / Platform

#### 12.5.1 Backstage (Spotify/CNCF)

| Field | Value |
|-------|-------|
| **URL** | https://backstage.io/ |
| **What** | Open-source developer portal framework. Software catalog, unified infrastructure tooling, TechDocs. Self-hosted, plugin-based. |
| **Type** | Platform (self-hosted) |
| **License** | Apache 2.0 |
| **Consulting relevance** | **Low for gdev integration, high for firm-level platform.** Backstage is a framework, not a product -- 56% of adopters cite upgrades as biggest pain point. A consulting firm's internal platform team could benefit, but it's orthogonal to gdev's per-project scope. gdev could generate Backstage catalog-info.yaml for teams using Backstage. |

#### 12.5.2 Port

| Field | Value |
|-------|-------|
| **URL** | https://www.port.io/ |
| **What** | No-code internal developer portal with customizable blueprints, dynamic scorecards, self-service actions, and workflow automation. SaaS with free tier. |
| **Type** | SaaS platform |
| **Consulting relevance** | **Low for gdev integration.** Same as Backstage -- firm-level concern, not per-project. Lower operational burden than Backstage (SaaS). |

---

## 13. Relevance Rankings & Integration Recommendations

### Tier 1 -- Recommend as Default (auto-configure in gdev wizard)

These tools provide high value across most consulting engagements and should be offered as defaults in the gdev wizard with auto-detection.

| Tool | Category | Why Default |
|------|----------|-------------|
| **Context7** | Library docs | 33K+ libraries, eliminates API hallucination, zero-install remote server |
| **MCP Toolbox for Databases** | Database | One server covers 15+ databases, YAML config, Google-backed |
| **GitHub MCP Server** | Source control | 100+ tools, most consulting work uses GitHub, security features |
| **AWS OpenAPI MCP Server** | API docs | Dynamic tool generation from any OpenAPI spec, multi-spec composition |
| **Sentry MCP Server** | Error tracking | Zero-install remote, high-value debugging workflow, auto-detect via DSN |
| **Terraform MCP Server** | IaC | Auto-detect via `.tf` files, registry metadata for IaC development |
| **devenv SecretSpec** | Secrets | Built into devenv.sh 2.0, zero additional tooling |

### Tier 2 -- Recommend Based on Detection (offer in wizard when project signals detected)

| Tool | Category | Detection Signal |
|------|----------|-----------------|
| **Grafana MCP Server** | Observability | Grafana URL in env/config |
| **Datadog MCP Server** | Observability | `DD_API_KEY` or Datadog config |
| **Kubernetes MCP Server** | Orchestration | `kubeconfig`, Helm charts, K8s manifests |
| **GitLab MCP Server** | Source control | `.gitlab-ci.yml`, GitLab remote URL |
| **Azure DevOps MCP** | Source control/PM | Azure DevOps project config |
| **Notion MCP Server** | Knowledge base | Notion integration token in env |
| **Atlassian Remote MCP** | Knowledge base | Atlassian domain in config |
| **Slack MCP Server** | Communication | Slack workspace config (read-only default) |
| **Snyk MCP Server** | Security | `SNYK_TOKEN` in env |
| **Infisical** | Secrets | `.infisical.json` or `INFISICAL_TOKEN` |
| **Pulumi MCP Server** | IaC | `Pulumi.yaml` in project root |
| **Postgres MCP Pro** | Database | Heavy PostgreSQL usage detected |
| **MongoDB MCP Server** | Database | `mongodb://` connection strings |

### Tier 3 -- Document as Available (reference in generated CLAUDE.md)

| Tool | Category | When Useful |
|------|----------|-------------|
| **AWS MCP Servers** (55+) | Cloud | Cloud profile selection in wizard |
| **GCP MCP Servers** (40+) | Cloud | Cloud profile selection in wizard |
| **Redis MCP Server** | Database | Cache debugging workflows |
| **Jenkins MCP Server** | CI/CD | Legacy enterprise clients |
| **Prometheus MCP Server** | Observability | Direct Prometheus (no Grafana) |
| **CVE Search / NVD MCP** | Security | Security audit engagements |
| **Snyk Agent Scan** | Security | Post-config MCP server validation |
| **Copier** | Scaffolding | Project template management |
| **Tilt / DevSpace** | Local dev | Kubernetes-native development |
| **Docker Compose** | Local dev | Already covered in Tier 1 ecosystems |
| **dotenvx** | Secrets | Lightweight per-project encryption |

### Tier 4 -- Watch List (too new, deprecated, or niche)

| Tool | Status | Notes |
|------|--------|-------|
| **Elasticsearch MCP** | Deprecated | Use Elastic Agent Builder MCP (9.2.0+) |
| **Teams MCP** | Limited adoption | Low developer workflow value |
| **Backstage / Port** | Firm-level concern | Orthogonal to gdev's per-project scope |
| **Dev Containers CLI** | Alternative paradigm | gdev uses devenv.sh, not dev containers |
| **Docker MCP Gateway** | Emerging | Interesting for MCP server isolation security |

### Cloud Provider Profile Recommendations

gdev should offer cloud provider selection as a wizard step, auto-configuring the appropriate MCP server subset:

| Profile | MCP Servers |
|---------|-------------|
| **AWS** | AWS Knowledge, CloudWatch, EKS, IAM, S3, Lambda, Billing, CloudTrail |
| **GCP** | GCP Compute, Cloud Run, GKE, BigQuery, Cloud SQL, Logging, Monitoring |
| **Azure** | Azure DevOps, Azure MCP (built into VS 2026) |
| **Multi-cloud** | Core servers from each + MCP Toolbox for Databases |
| **None/Self-hosted** | Skip cloud servers, focus on generic tools |

### Integration Architecture

```
.mcp.json (generated by gdev claudecode addon)
├── Always-on servers
│   ├── context7          (library docs, remote)
│   ├── openapi-mcp       (client API docs, local)
│   └── socket-dev        (supply chain scoring, existing)
│
├── Detected servers (auto-configured from project signals)
│   ├── mcp-toolbox       (database, if DB config detected)
│   ├── github-mcp        (if GitHub remote)
│   ├── gitlab-mcp        (if GitLab remote)
│   ├── sentry-mcp        (if Sentry DSN detected)
│   ├── terraform-mcp     (if .tf files present)
│   ├── kubernetes-mcp    (if kubeconfig/manifests present)
│   ├── grafana-mcp       (if Grafana URL configured)
│   └── datadog-mcp       (if DD_API_KEY present)
│
├── Profile servers (selected via cloud provider wizard step)
│   ├── aws-*-mcp         (AWS profile)
│   ├── gcp-*-mcp         (GCP profile)
│   └── azure-devops-mcp  (Azure profile)
│
└── Optional servers (explicit opt-in via wizard)
    ├── notion-mcp        (knowledge base)
    ├── confluence-mcp    (enterprise wiki)
    ├── slack-mcp         (communication, read-only)
    ├── snyk-mcp          (if licensed)
    └── infisical         (secrets management)
```

### Token Budget Considerations

Each MCP server adds to the context window. With potentially 10+ servers active, token management matters:

- **Context7** has built-in token optimization
- **AWS OpenAPI MCP** achieves 70-75% token reduction via dynamic prompts
- **Atlassian community MCP** uses TOON format for 30-60% savings
- **GitHub MCP** supports dynamic discovery (loads tools on-demand)
- **Grafana MCP** supports tool filtering by datasource

gdev should configure MCP servers with **minimal tool exposure** by default, enabling additional tools on demand. Most servers support toolset filtering via environment variables.

### Security Considerations for MCP Server Deployment

1. **Read-only defaults**: GitHub MCP, Slack MCP, and database MCP servers should default to read-only mode.
2. **Auth token management**: MCP servers requiring API tokens (Datadog, Snyk, GitHub PAT) should use devenv SecretSpec or Infisical, not `.env` files.
3. **MCP server validation**: Run Snyk Agent Scan after configuration to check for prompt injection vulnerabilities in configured servers.
4. **Network isolation**: Consider Docker MCP Gateway for security-sensitive deployments (MCP server sandboxing).
5. **CVE awareness**: Multiple MCP server CVEs disclosed in 2025-2026 (see Section 10). Pin versions and monitor for updates.

---

## Sources

### Web Searches (2026-05-12)
- MCP database servers: PostgreSQL, MySQL, MongoDB, Redis, Elasticsearch
- MCP API documentation: OpenAPI, Swagger, GraphQL
- MCP cloud providers: AWS (55+ servers), GCP (40+ products), Azure DevOps
- MCP monitoring: Datadog, Grafana, Prometheus
- MCP source control: GitHub (29.8K stars), GitLab (1.5K stars), Azure DevOps
- MCP containers: Kubernetes (1.6K stars), EKS, Docker Gateway
- MCP CI/CD: Jenkins, GitHub Actions, GitLab CI
- MCP knowledge: Notion (4.3K stars), Confluence, Sentry
- MCP communication: Slack (official GA), Teams
- MCP security: Snyk, CVE Search, NVD, Agent Scan
- MCP IaC: Terraform (~1.3K stars), Pulumi
- Developer tools: Infisical (26.8K stars), dotenvx, Copier (3.4K stars), devenv, DevSpace, Tilt
- Context7: 53.3K stars, 33K+ libraries, 890K weekly npm downloads

### Repository Pages Fetched
- googleapis/mcp-toolbox (15.2K stars, Apache 2.0, Go)
- github/github-mcp-server (29.8K stars, MIT, Go)
- grafana/mcp-grafana (3K stars, Apache 2.0, Go)
- containers/kubernetes-mcp-server (1.6K stars, Apache 2.0, Go)
- zereight/gitlab-mcp (1.5K stars, MIT, TypeScript)
- makenotion/notion-mcp-server (4.3K stars, MIT, TypeScript)
- aashari/mcp-server-atlassian-confluence (53 stars, TypeScript)
- redis/mcp-redis (509 stars, MIT, Python)
- elastic/mcp-server-elasticsearch (658 stars, Apache 2.0, Rust, deprecated)
- crystaldba/postgres-mcp (2.7K stars, MIT, Python)
- kud/mcp-jenkins (7 stars, MIT, TypeScript)
- infisical/infisical (26.8K stars, MIT, TypeScript)
- copier-org/copier (3.4K stars, MIT, Python)
- awslabs/mcp (55+ servers, Apache 2.0)
- hashicorp/terraform-mcp-server (~1.3K stars, MPL 2.0)
- upstash/context7 (53.3K stars, Apache 2.0)

### Documentation Pages Fetched
- AWS MCP server collection: awslabs.github.io/mcp/
- GCP MCP supported products: docs.cloud.google.com/mcp/supported-products
- AWS OpenAPI MCP Server: awslabs.github.io/mcp/servers/openapi-mcp-server
- Datadog MCP Server: docs.datadoghq.com/bits_ai/mcp_server/
