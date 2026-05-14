<!-- Source: Multiple web searches -->
<!-- Retrieved: 2026-03-20 -->

# Coder Web Search Findings (Consolidated)

## Architecture (from search results and Coder docs)

**Core Components:**
- **Coderd**: The main service ("coder server"). Thin API connecting workspaces, provisioners, and users. Stores state in PostgreSQL. Only service that communicates with Postgres. Recommended to deploy with multiple replicas for HA. Each coderd replica hosts 3 Terraform provisioners by default.
- **Provisioner Daemon (Provisionerd)**: Execution context for infrastructure-modifying providers. Currently only Terraform (terraform). Each provisioner handles one concurrent workspace build.
- **Agent**: Core logic running inside workspaces. Supports DevContainers, remote SSH, startup/shutdown script execution.
- **PostgreSQL**: Version 13+ required. Sole datastore, accessed only by coderd.
- **WireGuard Tunnels**: Encrypted tunnels for workspace connectivity. Both agents and clients establish WireGuard tunnels using UDP on ephemeral (high) ports.
- **DERP Relay**: Designated Encrypted Relay for Packets. Used when direct connections aren't possible. Coder server runs a built-in DERP relay. Works for both public and air-gapped deployments.

**Deployment Platforms:**
- Kubernetes (recommended for enterprise, used by most Fortune 500 customers)
- Docker
- Standalone binary with systemd
- Air-gapped environments fully supported

## Pricing (from search results)

**Community Edition (Free, AGPL v3.0):**
- Unlimited workspaces, users, and templates
- Core IDE support (VS Code, JetBrains, SSH, web terminal)
- Basic SSO (OIDC, GitHub)
- Community support only

**Premium Edition (Paid, per-seat annual license):**
- Ticket-based support with SLAs
- Multi-organization tenancy
- Resource quotas
- Audit logging
- Unlimited Git/auth integrations
- High availability configurations
- Workspace proxies (global performance)
- Advanced cost control (enforced auto-stop, dormancy)
- Enhanced RBAC with template-level permissions
- OIDC/SCIM group sync
- UI customization/branding
- Custom roles (organization-scoped)
- SOC2 Type II certified

Exact per-seat pricing not publicly disclosed — requires contacting sales.

## Comparison to Codespaces

- Coder is open source, self-hosted, infrastructure-agnostic
- Codespaces is SaaS on Azure, GitHub-only repos
- Coder supports any VCS, any cloud, any IDE
- Coder cost = your own infra costs + optional Premium license (no per-hour compute charge from Coder)
- Codespaces ~$0.18-$2.88/hr compute + $0.07/GiB/mo storage
- A developer using 4-core Codespace 8hr/day = ~$60.48/mo (not including storage)
- Coder can be cheaper at scale since you control infra pricing and reserved instances

## Real-World Adoption

- Fortune 500 customers (automotive, finance, government, technology)
- Organizations like Palantir, Dropbox mentioned as users
- Key industries: finance, government, defense, healthcare
- SOC2 Type II certified
- Positioned for organizations handling highly sensitive IP
- Strong fit for air-gapped/compliance-heavy environments
- Total funding: ~$70M+ (Crunchbase)

## Air-Gapped Deployment

- All Coder features supported in air-gapped/offline environments
- Custom server image for Docker or Kubernetes
- CLI config for Terraform referring to external mirrors
- Coder modules: mirror with JFrog Artifactory or private git repo
- Air-gapped docs can be self-hosted as static files
- Built-in DERP relay works in air-gapped deployments

## Envbuilder

- Open-source tool by Coder for running devcontainers without Docker daemon
- Builds dev container images in-place from devcontainer.json
- Runs on Docker, Kubernetes, OpenShift
- Supports layer caching for faster builds
- Terraform provider available for template integration
- Alternative to native Docker-based dev container support
