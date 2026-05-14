<!-- Source: https://raw.githubusercontent.com/obsessiondb/rudel/main/docs/self-hosting.md -->
<!-- Retrieved: 2026-03-26 -->

# Rudel Self-Hosting Documentation

## Overview

Rudel is a Bun server requiring Postgres and ClickHouse. The guide demonstrates deployment using ObsessionDB (ClickHouse), Neon (Postgres), and Fly.io, though "any Postgres instance, any ClickHouse instance...and any platform that can run a Bun/Node.js server will work."

## Services Table

| Component | Provider | Free Tier | Purpose |
|-----------|----------|-----------|---------|
| ClickHouse | ObsessionDB | Yes | Session transcript storage and analytics |
| Postgres | Neon | Yes (0.5 GB) | Authentication (users, sessions, accounts) |
| App Server | Fly.io | Yes (3 shared VMs) | HTTP API + static frontend serving |

## ClickHouse Setup (ObsessionDB)

1. Create account at obsessiondb.com
2. Create ClickHouse instance
3. Note connection details (Host, Username, Password)
4. Apply schema migration:

```bash
CLICKHOUSE_URL=https://your-instance.obsessiondb.com \
CLICKHOUSE_USER=your-username \
CLICKHOUSE_PASSWORD=your-password \
CLICKHOUSE_DB=default \
  bun --bun --cwd packages/ch-schema chkit migrate --apply
```

The migration creates the `rudel` database, `claude_sessions` and `session_analytics` tables, plus a materialized view for analytics. Uses `SharedReplacingMergeTree` for cloud deployment.

**Note**: `CLICKHOUSE_DB` must equal `default` initially since the `rudel` database doesn't exist yet.

## Postgres Setup (Neon)

1. Create account at neon.tech
2. Create project with database named `rudel`
3. Copy connection string
4. Run Drizzle migrations:

```bash
PG_CONNECTION_STRING="postgres://user:pass@host/rudel?sslmode=require" \
  bun run --cwd packages/sql-schema migrate
```

Creates authentication tables (users, sessions, accounts, verification tokens) for `better-auth`.

## Fly.io Deployment

1. Install Fly CLI
2. Authenticate: `fly auth login`
3. Create app: `fly launch --name your-app-name --no-deploy`
4. Set secrets (note: API uses `CLICKHOUSE_USERNAME`, not `CLICKHOUSE_USER`):

```bash
fly secrets set \
  PG_CONNECTION_STRING="postgres://user:pass@host/rudel?sslmode=require" \
  BETTER_AUTH_SECRET="$(openssl rand -base64 32)" \
  CLICKHOUSE_URL="https://your-instance.obsessiondb.com" \
  CLICKHOUSE_USERNAME="your-username" \
  CLICKHOUSE_PASSWORD="your-password" \
  APP_URL="https://your-app-name.fly.dev" \
  ALLOWED_ORIGIN="https://your-app-name.fly.dev"
```

5. Deploy: `fly deploy` (Dockerfile builds frontend and runs API)
6. Verify: `curl https://your-app-name.fly.dev/health`

## Social Login (Optional)

GitHub or Google OAuth supported. Email/password works without OAuth credentials.

## Environment Variables Reference

| Variable | Required | Description |
|----------|----------|-------------|
| `PG_CONNECTION_STRING` | Yes | Postgres connection string |
| `BETTER_AUTH_SECRET` | Yes | Auth secret |
| `CLICKHOUSE_URL` | Yes | ClickHouse HTTPS endpoint |
| `CLICKHOUSE_USERNAME` | Yes | ClickHouse username |
| `CLICKHOUSE_PASSWORD` | Yes | ClickHouse password |
| `APP_URL` | Yes | Public URL of the deployed app |
| `ALLOWED_ORIGIN` | Yes | CORS origin |
| `GITHUB_CLIENT_ID` | No | GitHub OAuth client ID |
| `GITHUB_CLIENT_SECRET` | No | GitHub OAuth client secret |
| `GOOGLE_CLIENT_ID` | No | Google OAuth client ID |
| `GOOGLE_CLIENT_SECRET` | No | Google OAuth client secret |
