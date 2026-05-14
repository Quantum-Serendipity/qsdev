# GitHub Codespaces Organization Management and Policies
- **Source**: https://docs.github.com/en/codespaces/managing-codespaces-for-your-organization
- **Retrieved**: 2026-03-20
- **Note**: Content synthesized from web search results (WebFetch unavailable)

## Organization-Level Policies

### Available Policy Controls

1. **Machine Type Restrictions**
   - Restrict which machine types are available (e.g., only 2-core and 4-core)
   - Prevents excessive compute costs from large VMs
   - Can be set globally or per-repository

2. **Idle Timeout Constraints**
   - Default idle timeout: 30 minutes
   - Org admins can set maximum idle timeout
   - Reduces wasted compute from forgotten sessions

3. **Retention Period Constraints**
   - Default retention: 30 days for stopped codespaces
   - Org admins can set maximum retention period
   - Can disable "Keep codespace" (indefinite retention) option
   - Automatic deletion happens regardless of unpushed changes

4. **Codespace Count Limits**
   - Restrict number of org-billed codespaces per user
   - Helps control overall spending

5. **Port Visibility Restrictions**
   - Can restrict forwarded ports to private-only or org-visible
   - Prevents accidental public exposure of development services

6. **Base Image Restrictions**
   - Can restrict which base images codespaces can use
   - Useful for compliance and security standardization

### Policy Scope
- Policies can apply to all repositories or specific repositories
- Multiple policies can coexist (most restrictive wins in overlap)

## Billing and Ownership Models

### Organization-Owned Codespaces
- Organization pays for usage
- Org chooses which members/collaborators can create codespaces at org expense
- Available for GitHub Team and GitHub Enterprise plans
- Codespace is billed when created from org-owned repo (public or private)

### User-Owned Codespaces
- User pays from their own free tier or spending limit
- Default behavior for personal accounts
- Not supported with data residency (Enterprise Cloud)

### Spending Limits
- Default: $0 USD (must be explicitly set to enable billing)
- Can set specific dollar amount or unlimited
- When limit reached: no new codespaces, existing ones cannot resume
- Enterprise-level limits cascade to organizations

## Access Control

### Who Can Create Codespaces
- Organization owners control access for private/internal repos
- Options: all members, selected members, no one
- Can include or exclude outside collaborators
- Public repos: anyone with access can create a codespace (using their own billing)

### Enterprise-Level Enforcement
- Enterprise admins can enforce policies across all organizations
- Can completely disable Codespaces for an enterprise
- Can limit which organizations enable Codespaces

## Cost Monitoring

- Usage reports available showing compute and storage by user
- Can export billing data for analysis
- Alerts when approaching spending limits
- GitHub Billing platform provides per-product cost breakdowns

## Additional Sources

- https://docs.github.com/en/codespaces/managing-codespaces-for-your-organization/choosing-who-owns-and-pays-for-codespaces-in-your-organization
- https://docs.github.com/en/codespaces/managing-codespaces-for-your-organization/restricting-the-idle-timeout-period
- https://docs.github.com/en/codespaces/managing-codespaces-for-your-organization/restricting-access-to-machine-types
- https://docs.github.com/en/codespaces/managing-codespaces-for-your-organization/restricting-the-retention-period-for-codespaces
- https://docs.github.com/en/enterprise-cloud@latest/admin/enforcing-policies/enforcing-policies-for-your-enterprise/enforcing-policies-for-github-codespaces-in-your-enterprise
