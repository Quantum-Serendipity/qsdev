# Coder: Deep Dive for Multi-Client Consulting

## Executive Summary

Coder is a self-hosted, open-source (AGPL v3.0) platform for provisioning cloud development environments using Terraform templates. Unlike GitHub Codespaces (SaaS on Azure), Coder runs entirely on your own infrastructure — any cloud, on-premises, or air-gapped — giving organizations full control over data residency, networking, and cost. The Terraform-based template system is both Coder's greatest strength and highest barrier: it enables precise, auditable infrastructure-as-code for workspaces but requires Terraform/Kubernetes expertise to operate. For multi-client consulting, Coder's Organizations feature (Premium) provides genuine multi-tenant isolation with separate provisioners, templates, credentials, and admin roles per client. The self-hosted model directly addresses the #1 consulting pain point with Codespaces: client code never leaves infrastructure you control. The trade-off is operational complexity — you are responsible for running the platform, managing templates, and maintaining infrastructure.

---

## Architecture

### Core Components

```
Developer (VS Code / JetBrains / SSH / Web Terminal)
  └─ WireGuard encrypted tunnel (or DERP relay if direct UDP blocked)
       └─ Coderd (control plane, Go binary)
            ├─ PostgreSQL 13+ (sole datastore)
            ├─ Provisionerd (Terraform execution engine)
            │    └─ terraform apply / terraform destroy
            └─ Dashboard (TypeScript/React web UI)
                 └─ Workspace (VM / K8s pod / Docker container)
                      └─ Coder Agent (SSH, terminal, IDE connections,
                           startup scripts, dev container sub-agents)
```

**Coderd** is the main service — a thin API layer connecting workspaces, provisioners, and users. It serves the web dashboard and API, stores all state in PostgreSQL, and is the only component that communicates with the database. For HA, deploy multiple coderd replicas behind a load balancer. Each replica hosts 3 built-in Terraform provisioners by default.

**Provisionerd** executes Terraform during workspace and template builds. Each provisioner handles one concurrent workspace build. Provisioners can run embedded in coderd or as external daemons — critical for multi-tenant isolation where each organization needs provisioners with access to different cloud credentials.

**Coder Agent** runs inside each workspace. It provides SSH access, terminal sessions, IDE connectivity (VS Code, JetBrains), startup/shutdown script execution, and dev container sub-agent management. Agents connect back to coderd via WireGuard tunnels.

**WireGuard Tunnels** provide encrypted connectivity between developers and workspaces. Both the client and agent establish WireGuard tunnels using UDP on ephemeral ports. When direct UDP isn't possible (NAT, firewalls), traffic routes through DERP (Designated Encrypted Relay for Packets) relays. Coderd runs a built-in DERP relay; additional relays can be deployed for geographic distribution.

### Technology Stack

- **Go** (76% of codebase) — control plane, provisioner, agent
- **TypeScript** (22%) — dashboard web UI
- **HCL** (0.4%) — Terraform template definitions
- **PLpgSQL** (0.3%) — database migrations

### Infrastructure Requirements

| Component | Requirement |
|-----------|-------------|
| Database | PostgreSQL 13+ |
| Deployment | Kubernetes (recommended), Docker, or standalone binary |
| Networking | HTTPS for dashboard, UDP for WireGuard, DERP relay fallback |
| Compute | Depends on workspace templates (VMs, K8s pods, Docker containers) |

---

## Terraform Template System

### How Templates Work

Templates are the defining architectural concept of Coder. Every workspace is provisioned by a Terraform template that declares the exact infrastructure: compute type, storage, networking, installed tools, IDE configuration, and lifecycle behavior. When a developer creates a workspace, Coder runs `terraform apply` against the template; when they delete it, Coder runs `terraform destroy`.

This means anything Terraform can provision, Coder can use as a workspace:
- **EC2 instances** (or any cloud VM)
- **Kubernetes pods** (most common for enterprise)
- **Docker containers** (simplest for development/testing)
- **Bare metal** via SSH provisioning
- **Google Cloud Workstations**, **Azure VMs**, etc.

### Template Anatomy

A template consists of:

1. **Terraform HCL files** — declare infrastructure resources, the `coder_agent`, and any `coder_app` entries (web IDEs, dashboards, etc.)
2. **Parameters** — configurable options presented to users at workspace creation (disk size, instance type, region, GPU, etc.). Dynamic Parameters (v2.24+) enable Terraform-defined validation and conditional logic.
3. **Provisioner tags** — route builds to specific provisioner instances (critical for org isolation)
4. **Workspace tags** — metadata for categorization and policy enforcement

### Template Lifecycle

1. **Author** — Template admin writes Terraform HCL
2. **Push** — Upload via CLI (`coder templates push`) or GitOps pipeline
3. **Version** — Templates are versioned; users can be on different versions
4. **Use** — Developers create workspaces from templates, filling in parameters
5. **Update** — New versions can enforce auto-stop to pick up updates
6. **Deprecate** — Mark templates as deprecated (no new workspaces, existing ones continue)

### Template Best Practices

- **Image management**: Pre-bake container images with languages and tools; reference in templates
- **GitOps**: Store templates in Git, push via CI/CD pipeline (not manual CLI)
- **Resource protection**: Hardened templates prevent accidental data destruction
- **Starter templates**: Coder provides pre-configured templates for AWS, GCP, Azure, Kubernetes, Docker

### Customization Depth

Because templates are pure Terraform, the customization ceiling is essentially unlimited:
- Install any toolchain via startup scripts or container images
- Configure networking rules per workspace
- Attach GPUs, mount NFS volumes, configure DNS
- Inject credentials from Vault, AWS Secrets Manager, etc.
- Define multiple `coder_app` resources for web-based tools (Jupyter, pgAdmin, etc.)
- Use Terraform modules from the Coder Registry for common patterns

---

## Workspace Lifecycle

### States

| State | Description | Resource Cost |
|-------|-------------|---------------|
| **Running** | Ready, accepting connections | Full compute + storage |
| **Stopped** | Ephemeral resources destroyed, persistent storage idle | Storage only |
| **Failed** | Provisioning error, no resources running | None |
| **Unhealthy** | Resources exist but agent can't connect | Full compute |
| **Deleted** | All resources destroyed | None |
| **Dormant** | Marked for auto-deletion (Premium) | Storage only |

### Resource Persistence

Workspaces distinguish between:
- **Ephemeral resources** — destroyed on stop, recreated on start (compute instances, pods)
- **Persistent resources** — survive stop/start cycles (disks, volumes), destroyed only on delete

### Scheduling and Cost Control

**Autostart**: Launch workspaces at scheduled times/days (e.g., 9 AM weekdays). Template admin must enable.

**Autostop**: Stop after configurable hours of inactivity. Activity detection covers VS Code, JetBrains, SSH, web terminal, and AI agent sessions. Activity "bumps" the shutdown timer by a configurable duration (default 1 hour).

**Autostop Requirement** (Premium): Force workspace stops at intervals (days/weeks) regardless of activity — ensures template updates are applied.

**Dormancy** (Premium): Automatically delete workspaces that remain stopped for a configurable period. Prevents abandoned workspace accumulation.

**Resource Quotas** (Premium): Limit compute/storage per user, group, or organization.

---

## IDE Support

Coder is genuinely IDE-agnostic — a major differentiator from Codespaces (VS Code-centric) and Gitpod (browser-first).

| IDE | Connection Method | Notes |
|-----|-------------------|-------|
| **VS Code Desktop** | Coder extension + SSH | Full local VS Code, remote workspace |
| **VS Code Browser** | code-server (web) | Defined as `coder_app` in template |
| **JetBrains IDEs** | Gateway plugin or SSH | IntelliJ, PyCharm, GoLand, etc. via Gateway |
| **JetBrains Toolbox** | Coder plugin | Streamlined JetBrains connection |
| **Cursor** | Coder extension (VS Code compat) | AI-enhanced IDE |
| **Windsurf** | Coder extension (VS Code compat) | Codeium's AI editor |
| **SSH** | `coder ssh <workspace>` | Any terminal or SSH client |
| **Web terminal** | Built-in dashboard | Always available |

JetBrains support includes specific documentation for air-gapped deployments (pre-downloading IDE backends).

---

## Dev Container Support

### Native Integration (v2.24+)

Coder has first-class devcontainer.json support, but it works differently from Codespaces or DevPod. Rather than *being* a Dev Container runtime, Coder treats dev containers as **sub-agents within a workspace**.

**How it works:**

1. A Coder workspace (VM or K8s pod) runs Docker
2. The Coder agent inside the workspace discovers `devcontainer.json` files in cloned repos
3. It uses `@devcontainers/cli` to build and run the dev container
4. Each dev container becomes a sub-agent with its own terminal, SSH access, and apps in the dashboard

**Two configuration approaches:**

1. **Explicit**: Use `coder_devcontainer` Terraform resource to declare which dev containers to start (recommended)
2. **Auto-discovery**: Agent scans repos for `devcontainer.json` and surfaces them in the dashboard

**Multi-container**: Multiple `coder_devcontainer` resources can be defined, each pointing to different repos — each runs as a separate sub-agent.

### Envbuilder (Alternative)

For workspaces without Docker (e.g., Kubernetes pods without DinD), Coder provides **Envbuilder** — an open-source tool that builds dev container images *without* a Docker daemon. It reads `devcontainer.json`, builds in-place, and supports layer caching via a container registry. Envbuilder has its own Terraform provider for template integration.

### Limitations

- **Linux only** — no Windows or macOS dev container support
- **Manual rebuild** — changes to `devcontainer.json` require dashboard-initiated rebuild
- **Port forwarding** — `forwardPorts` doesn't support `host:port` syntax for Docker Compose sidecars
- Requires Coder v2.24.0+, Docker in workspace, and `@devcontainers/cli`

### Key Difference from Codespaces/DevPod

In Codespaces and DevPod, the dev container *is* the workspace. In Coder, the workspace is a VM or pod that *contains* dev containers. This is a meaningful architectural difference: it means Coder workspaces can run multiple dev containers simultaneously, but it also means there's an extra layer of infrastructure (the host workspace) to manage.

---

## Enterprise Features

### RBAC Roles

| Role | Scope | Capabilities |
|------|-------|-------------|
| **Owner** | Site-wide | Full platform control, manages organizations and global settings |
| **User Admin** | Site-wide | Manage user accounts, suspension/activation, password resets |
| **Template Admin** | Site-wide | Manage all templates across the deployment |
| **Auditor** | Site-wide | Read access to audit logs and administrative data |
| **Member** | Site-wide | Create workspaces from available templates (default role) |

**Template-level permissions** (Premium): Assign per-template Use or Admin roles to specific users or groups, without granting site-wide Template Admin.

**Custom roles** (Premium): Organization-scoped custom roles (e.g., Provisioner Admin, Template Editor, Template Pusher).

### Audit Logging (Premium)

Comprehensive audit trail of platform actions. Required for compliance in regulated industries.

### Multi-Organization Tenancy (Premium)

See "Multi-Tenant Isolation" section below.

### High Availability

Multiple coderd replicas behind a load balancer, shared PostgreSQL database. Requires Coder v2.16+ with Premium license.

### Workspace Proxies (Premium)

Deploy proxy servers in multiple regions to reduce latency for geographically distributed teams. Proxies handle workspace connections while the control plane remains centralized.

### SSO / Identity Provider Integration

- **OpenID Connect**: Okta, KeyCloak, PingFederate, Azure AD
- **GitHub**: Including GitHub Enterprise
- **SCIM** (Premium): Automated user provisioning and deprovisioning
- **Group & Role Sync**: Automatic organization/role/group assignment from IdP claims (e.g., `memberOf`)

### SOC2 Type II Certified

Coder holds SOC2 Type II certification for security and availability controls.

---

## Pricing

### Community Edition (Free)

- **License**: AGPL v3.0 open source
- **Unlimited**: Workspaces, users, templates
- **Included**: Core IDE support, SSH, web terminal, basic SSO (OIDC, GitHub), Dev Container integration, auto-start/stop
- **Not included**: Multi-org tenancy, template permissions, audit logs, HA, workspace proxies, dormancy, quotas, SCIM sync, custom roles, enforced auto-stop, branding

### Premium Edition (Paid)

- **License**: Annual, per-seat
- **Pricing**: Not publicly disclosed — requires contacting sales
- **Added features**: Multi-organization tenancy, resource quotas, audit logging, HA, workspace proxies, advanced cost control (enforced auto-stop, dormancy), enhanced RBAC with template-level permissions, OIDC/SCIM group sync, custom roles, UI customization/branding, ticket-based support with SLAs

### True Cost

Coder itself is free (Community) or per-seat licensed (Premium). But the **total cost of ownership** includes:
1. **Infrastructure**: Cloud compute, storage, networking for workspaces (you pay your cloud provider directly)
2. **Platform operations**: Engineering time to deploy, maintain, and update Coder and templates
3. **Terraform expertise**: Template creation and maintenance requires HCL proficiency
4. **Premium license** (optional): For enterprise features

Unlike Codespaces ($0.18-$2.88/hr per workspace), there's no per-workspace-hour charge from Coder. You control infrastructure costs through reserved instances, spot pricing, auto-stop, and resource quotas. At scale (50+ developers), this is typically significantly cheaper than Codespaces.

---

## Multi-Tenant Isolation (Consulting-Critical)

### The Organizations Feature (Premium)

Coder's multi-organization support is the key feature for consulting use cases. Each organization gets:

- **Separate templates** — client-specific workspace configurations
- **Separate provisioners** — each org's provisioner runs in isolated infrastructure with its own cloud credentials. "Templates in one organization cannot use the same provisioner as templates in another organization."
- **Separate groups and users** — users can belong to multiple organizations, but see only their org's templates
- **Separate admins** — each org can have its own template admins and custom roles
- **Isolated credential injection** — IaaS API keys and secrets are scoped to the org's provisioners, not shared globally

### How This Maps to Consulting

For a consulting firm with multiple clients:

```
Coder Instance (self-hosted)
├─ Organization: "Client-A"
│   ├─ Provisioner → Client-A's AWS account
│   ├─ Templates → Client-A specific (K8s on EKS, their tools)
│   ├─ Workspaces → Developers assigned to Client-A
│   └─ Secrets → Client-A credentials only
├─ Organization: "Client-B"
│   ├─ Provisioner → Client-B's GCP project
│   ├─ Templates → Client-B specific (Docker on GCE, their stack)
│   ├─ Workspaces → Developers assigned to Client-B
│   └─ Secrets → Client-B credentials only
└─ Organization: "Internal"
    ├─ Provisioner → Company's own infra
    ├─ Templates → Internal tools, training, shared resources
    └─ Workspaces → All developers
```

A developer assigned to both Client-A and Client-B sees both orgs' templates but cannot cross-contaminate credentials or provisioning infrastructure.

### Provisioner Isolation Detail

This is architecturally strong. Provisioners operate in separate infrastructure with isolated authentication keys. The control plane submits provisioner jobs (simple build requests) — it never has direct access to the cloud resources that provisioners create. This means even if the Coder control plane were compromised, it couldn't directly access client infrastructure.

### Limitations

- **Organizations require Premium license** — not available in the free Community Edition
- **Templates cannot be moved between organizations** — must deprecate and recreate
- **Each org needs at least one dedicated provisioner** — operational overhead per client
- **Manual user assignment** is discouraged; IdP sync (Okta, etc.) is recommended but adds another integration
- **Increased maintenance overhead** — Coder's own docs warn to "only deploy organizations when necessary"

### Alternative: Separate Coder Instances

Before the Organizations feature existed, the recommended pattern was deploying a separate Coder instance per client. This is still viable and provides even stronger isolation (separate databases, control planes, and blast radius), but at higher operational cost and licensing complexity.

---

## Credential and Secret Management

### Approaches (Weakest to Strongest)

1. **Manual secrets** — User writes credentials to persistent files after workspace creation. Simple but unmanaged.

2. **Dynamic secrets via Terraform** — Template provisions API keys, service accounts, or tokens at workspace creation time using Terraform providers (AWS, GCP, Vault). Injected as environment variables on the `coder_agent`. This is the recommended pattern.

3. **Cloud provider native** — Provision per-workspace service accounts with cloud IAM; workspace accesses secrets through the provider's native secret manager (AWS Secrets Manager, GCP Secret Manager, etc.).

4. **HashiCorp Vault integration** — Template provisions Vault tokens; workspace retrieves secrets from Vault at runtime. Strongest isolation and rotation capabilities.

### Security Properties

- **SSH keys**: Auto-generated per user, never stored on disk in workspaces, fetched in-memory only when SSH is invoked
- **Template parameters**: **Never use for secrets** — they display in cleartext and are visible to anyone with workspace view permissions
- **`coder_metadata`**: Can surface secrets in dashboard UI with sensitive flagging
- **Per-org credential isolation**: With Organizations, each org's provisioner has its own cloud credentials, preventing cross-client credential access

### Consulting Implications

The Terraform-based dynamic secrets model is well-suited for consulting: each client organization's template can provision client-specific credentials from client-specific secret stores. A developer switching from Client-A to Client-B workspaces naturally gets different credentials without manual intervention.

---

## Comparison to GitHub Codespaces

| Dimension | Coder | GitHub Codespaces |
|-----------|-------|-------------------|
| **Deployment** | Self-hosted (any infra) | SaaS (Azure only) |
| **Data residency** | Full control | Limited (preview) |
| **VCS support** | Any Git provider | GitHub only |
| **IDE support** | VS Code, JetBrains, SSH, web, Cursor, Windsurf | VS Code, JetBrains, SSH, web |
| **Offline capable** | Yes (if infra is local/on-prem) | No |
| **Air-gapped** | Full support | No |
| **Cost model** | Infra costs + optional license | Per-hour compute + storage |
| **Template system** | Terraform (full IaC) | devcontainer.json |
| **Multi-tenant** | Organizations (Premium) | Per-GitHub-org |
| **Client code location** | Your infrastructure | GitHub/Azure |
| **Setup complexity** | High (Terraform, K8s, PostgreSQL) | None |
| **Dev Container support** | Yes (as sub-agents, v2.24+) | Yes (native runtime) |
| **Prebuilds** | Envbuilder + layer caching | Native prebuild system |
| **GPU support** | Whatever your infra provides | Deprecated (Aug 2025) |

### Key Positioning

Coder explicitly positions itself as "the self-hosted GitHub Codespaces alternative." The primary value proposition is: same concept (cloud dev environments), but you control the infrastructure. This directly addresses the biggest consulting pain point with Codespaces — client code on third-party infrastructure.

---

## Real-World Adoption

### Who Uses Coder

- **Fortune 500 companies** across automotive, finance, government, and technology
- **Named users**: Palantir, Dropbox (mentioned in Coder marketing)
- **Industries**: Finance, government, defense, healthcare — sectors with strict data sovereignty
- **Scale**: 12.6k GitHub stars, 1.2k forks, 287 releases (v2.30.4 as of research date)
- **Funding**: ~$70M+ total funding (Crunchbase)
- **Certification**: SOC2 Type II

### Typical Adopter Profile

Organizations that:
- Have platform engineering teams with Terraform/Kubernetes expertise
- Need to keep source code on-premises or in controlled cloud accounts
- Operate in regulated industries (finance, defense, healthcare)
- Want to standardize dev environments across large engineering orgs
- Need air-gapped or compliance-heavy deployments

---

## Limitations and Pain Points

### Operational Complexity

**This is the primary concern for small/mid-size consulting firms.** Coder requires:
- PostgreSQL database (HA for production)
- Kubernetes cluster or Docker host (recommended: K8s)
- Terraform expertise for template creation and maintenance
- Ongoing platform engineering: updates, monitoring, troubleshooting
- Template maintenance as client needs evolve

A consulting firm with 5-20 developers likely doesn't have a dedicated platform engineering team. Coder's operational burden may exceed the benefit unless the firm already runs Kubernetes infrastructure.

### Template Maintenance

Templates are powerful but require ongoing care:
- Base images need regular security updates
- Terraform provider updates can break templates
- Each client may need custom templates — multiplies maintenance
- Template versioning must be managed (users on old versions, forced updates)

### Learning Curve

- Terraform proficiency required for template authors
- Developers need to learn Coder-specific workflow (different from local dev)
- Debugging failed workspaces requires understanding Terraform state

### Dev Container Limitations

- Linux only (no Windows/macOS dev containers)
- Extra layer vs. Codespaces: workspace VM → Docker → dev container (more indirection)
- Manual rebuild required for devcontainer.json changes
- Requires Docker inside the workspace (DinD on K8s adds complexity)

### Multi-Org Overhead

- Each client organization needs its own provisioner infrastructure
- Templates cannot be shared or moved between orgs
- IdP sync integration required for proper user management
- Premium license required — cost unknown without sales conversation

### Network Dependencies

- WireGuard UDP may be blocked by corporate firewalls (DERP relay fallback adds latency)
- Workspace proxies (Premium) help with geographic distribution but add infrastructure
- If self-hosting on cloud: internet dependency for developer connectivity (same as Codespaces)
- If self-hosting on-prem with VPN: truly offline-capable, but VPN management is its own complexity

---

## Consulting-Specific Fit Assessment

### Strengths for Consulting

1. **Data sovereignty solved**: Client code stays on infrastructure you control. This eliminates the #1 objection to Codespaces for enterprise clients.

2. **Multi-client isolation**: Organizations feature provides genuine tenant isolation — separate provisioners, credentials, templates, and admin roles per client.

3. **Infrastructure flexibility**: Each client org can target different cloud providers or on-prem infrastructure. Client-A on AWS, Client-B on GCP, Client-C on-prem — all from one Coder instance.

4. **Template reuse with customization**: Base templates can be forked per client org with client-specific modifications. New engagement setup is "create org, deploy provisioner, push template."

5. **Cost control**: Auto-stop, dormancy, and quotas prevent runaway spending. You pay infra costs directly (can use reserved instances, spot pricing) rather than per-hour Codespaces pricing.

6. **Air-gap capable**: For government or defense clients requiring air-gapped environments, Coder is one of the few CDE platforms that fully supports this.

7. **IDE freedom**: Developers can use their preferred IDE (VS Code, JetBrains, Cursor, etc.) — no friction in onboarding developers with different tool preferences.

### Weaknesses for Consulting

1. **Operational overhead is high**: Running Coder is a platform engineering commitment. A 10-person consulting firm needs someone maintaining Kubernetes, PostgreSQL, Coder updates, and per-client templates.

2. **Premium license required for consulting-critical features**: Multi-org isolation, audit logging, and RBAC customization all require Premium. The free Community Edition works for a single team but lacks the isolation features consulting demands. Pricing is opaque.

3. **Terraform expertise required**: Creating and maintaining templates requires real Terraform knowledge. This is a specialized skill that not all consulting firms have in-house.

4. **Onboarding friction**: Compared to Codespaces (click a button) or DevPod (install CLI), Coder requires infrastructure provisioning before the first workspace can be created.

5. **Single point of failure risk**: One Coder instance serving multiple clients means a platform outage affects all client work simultaneously.

### When Coder Makes Sense for Consulting

- Firm has 20+ developers and existing cloud infrastructure (K8s cluster)
- Multiple clients with strict data sovereignty requirements
- Firm has Terraform expertise in-house
- Engagements are long enough (months+) to amortize setup cost
- Clients in regulated industries (finance, defense, healthcare)

### When Coder is Overkill for Consulting

- Small firm (< 10 developers) without dedicated platform engineering
- Short engagements where setup time exceeds engagement duration
- Clients that already provide their own dev environments
- Teams comfortable with local Dev Containers + direnv/Nix devShells

---

## Sources

All source documents saved to `docs/`:

| File | Content |
|------|---------|
| `docs/coder-github-readme.md` | Repository overview and key statistics |
| `docs/coder-readme-raw.md` | README content from raw GitHub |
| `docs/coder-templates-docs.md` | Template system: concepts, customization, best practices |
| `docs/coder-devcontainer-integration-docs.md` | Dev container integration: architecture, discovery, sub-agents, limitations |
| `docs/coder-workspace-lifecycle-docs.md` | Workspace states, provisioning process, cost control |
| `docs/coder-workspace-scheduling-docs.md` | Auto-start/stop, activity detection, dormancy |
| `docs/coder-secrets-management-docs.md` | Secrets: SSH keys, dynamic secrets, Vault integration |
| `docs/coder-multi-tenancy-rfc-discussion.md` | Multi-tenancy RFC: isolation design, provisioner separation, credential management |
| `docs/coder-organizations-docs.md` | Organizations: isolation, provisioner requirements, user management |
| `docs/coder-organizations-best-practices.md` | When to use orgs, ideal use cases, security isolation, scale management |
| `docs/coder-template-permissions-docs.md` | RBAC: Use vs Admin roles, access control |
| `docs/coder-user-management-roles-docs.md` | User roles: Owner, Member, Template Admin, User Admin, Auditor |
| `docs/coder-web-search-findings.md` | Consolidated web search findings: architecture, pricing, adoption, air-gap |

### Key External Sources

- [Coder GitHub Repository](https://github.com/coder/coder)
- [Coder Documentation](https://coder.com/docs)
- [Coder Pricing](https://coder.com/pricing)
- [Coder Premium vs Community](https://coder.com/cde/premium)
- [Coder Organizations Docs](https://coder.com/docs/admin/users/organizations)
- [Coder Organizations Best Practices](https://coder.com/docs/tutorials/best-practices/organizations)
- [Coder Architecture](https://coder.com/docs/admin/infrastructure/architecture)
- [Coder Dev Containers](https://coder.com/docs/user-guides/devcontainers)
- [Coder Secrets](https://coder.com/docs/admin/security/secrets)
- [Coder Air-Gapped Deployments](https://coder.com/docs/install/airgap)
- [Multi-Tenancy RFC Discussion #7638](https://github.com/coder/coder/discussions/7638)
- [Coder vs Codespaces Blog](https://coder.com/blog/coder-the-github-codespaces-alternative)
- [BayTech Coder Value Proposition 2025](https://www.baytechconsulting.com/blog/coder-com-platform-value-proposition-2025)
- [vCluster Comparison: Coder vs Codespaces vs Gitpod vs DevPod](https://www.vcluster.com/blog/comparing-coder-vs-codespaces-vs-gitpod-vs-devpod)
