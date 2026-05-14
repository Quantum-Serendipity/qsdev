# mcp.hosting Self-Hosted Deploy README
- **Source**: https://github.com/YawLabs/mcp-hosting-deploy (via GitHub API)
- **Retrieved**: 2026-05-14

## Overview

Self-hosted instance of mcp.hosting for teams needing data-sovereignty, compliance, or contract reasons. Requires active Team subscription ($15/seat/mo).

## What You Get

- mcph orchestrator + dashboard + team sign-up
- Unlimited MCP servers per user
- Opt-in usage analytics (30-day retention)
- Compliance test runner
- Shared servers, admin controls, centralized billing
- Priority support

## Prerequisites

- Active Team subscription at mcp.hosting
- License key (mcph_sh_<hex>) + GHCR pull token from dashboard
- Linux server (Ubuntu 22.04+) or Kubernetes cluster
- Domain name with A record
- Docker Engine 24+ / Docker Compose v2+ OR kubectl + Helm 3+
- AWS SES sender identity for magic-link auth

## Deployment Options

### Docker Compose
```bash
git clone https://github.com/yawlabs/mcp-hosting-deploy.git
cd mcp-hosting-deploy/docker-compose
cp .env.example .env
# Configure: DOMAIN, POSTGRES_PASSWORD, COOKIE_SECRET, LICENSE_KEY, REDIS_AUTH_TOKEN, 
#            GITHUB_CLIENT_ID/SECRET, EMAIL_FROM, AWS SES vars
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```
Caddy handles TLS via Let's Encrypt automatically.

### Helm
```bash
helm install mcp-hosting oci://ghcr.io/yawlabs/charts/mcp-hosting \
  --version 0.2.2 \
  --namespace mcp-hosting --create-namespace \
  --set domain=mcp.example.com \
  --set licenseKey=mcph_sh_...
```

## Client Onboarding

Team members run:
```bash
npx -y @yawlabs/mcph install claude-code --token mcp_pat_...
```
Then add `"apiBase": "https://mcp.example.com"` to `~/.mcph.json`.

## Pricing

- Free tier: hosted-only at mcp.hosting, no self-host
- Team plan: $15/seat/month, includes self-host license + GHCR pull token
