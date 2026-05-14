# GitHub Codespaces Pricing and Billing
- **Source**: https://docs.github.com/billing/managing-billing-for-github-codespaces/about-billing-for-github-codespaces
- **Retrieved**: 2026-03-20
- **Note**: Content synthesized from web search results (WebFetch unavailable)

## Compute Pricing (per active hour, billed per-minute)

| Machine Type | Cores | RAM | Storage | Per Hour |
|---|---|---|---|---|
| Basic | 2 | 8 GB | 32 GB | $0.18 |
| Standard | 4 | 16 GB | 32 GB | $0.36 |
| Large | 8 | 32 GB | 64 GB | $0.72 |
| XL | 16 | 64 GB | 128 GB | $1.44 |
| XXL | 32 | 128 GB | 128 GB | $2.88 |

Compute cost is directly proportional to core count — the 16-core machine costs 8x the 2-core machine.

**Important**: You are NOT billed for compute when a codespace is stopped/suspended. Only active (running) codespaces incur compute charges.

## Storage Pricing

- **Rate**: $0.07 USD per GiB per month
- Charged for actual used space, not max allocated
- Includes: Docker image layers, repo source, dependencies, any files created
- Storage charges accrue even when codespace is stopped
- Prebuild snapshots also consume storage and incur charges

## Free Tier (Personal Accounts)

All personal GitHub accounts include monthly free usage:
- **120 core-hours** of compute (= 60 hours on 2-core, 30 hours on 4-core, etc.)
- **15 GB** of storage
- Resets monthly
- No free tier for organization/enterprise accounts

## Organization Billing

### Billing Models
1. **User-billed** (default): Each user who creates a codespace pays from their own included usage / spending limit
2. **Organization-billed**: Organization pays for members' and collaborators' Codespaces usage

### Spending Limits
- Default spending limit: $0 USD (must be explicitly increased)
- Can set specific dollar amounts or unlimited
- When limit is reached, no new codespaces can be created and existing ones cannot resume
- Enterprise accounts can set spending limits that cascade to organizations

### Cost Controls Available
- Restrict available machine types (e.g., only allow 2-core and 4-core)
- Set maximum idle timeout (reduce wasted compute)
- Set maximum retention period (reduce storage costs)
- Limit number of codespaces per user
- Restrict which repos can use org-billed codespaces

## Prebuild Costs

- Prebuilds run as GitHub Actions workflows — Actions minutes are consumed
- Prebuild storage snapshots incur storage charges at $0.07/GiB/mo
- Multiple prebuilds (per branch, per region) multiply storage costs
- Prebuilds must be rebuilt when dev container config changes

## Cost Example

A developer using a 4-core machine for 8 hours/day, 22 days/month:
- Compute: 8h x 22d x $0.36/hr = $63.36/mo
- Storage (assuming 20 GB used): 20 x $0.07 = $1.40/mo
- **Total: ~$64.76/mo per developer**

A team of 10 developers on 4-core machines: ~$648/mo

## Additional Sources

- https://github.com/pricing
- https://github.com/pricing/calculator
- https://docs.github.com/en/codespaces/managing-codespaces-for-your-organization/managing-the-cost-of-github-codespaces-in-your-organization
- https://medium.com/@udtc.us/understanding-the-cost-of-github-codespaces-a-deep-dive-into-2-core-instances-913a110eefb3
