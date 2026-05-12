<!-- Source: https://raw.githubusercontent.com/NixOS/nix-security-tracker/main/CONTRIBUTING.md -->
<!-- Retrieved: 2026-05-12 -->

# Nix Security Tracker - Contributing / Architecture

## Architecture Overview
- **Global configuration**: `src/project/`
- **Core logic & data models**: `src/shared/` application
- **Frontend**: `src/webview/` application
- High-level system design documented in `docs/README.md`
- Visual architecture in `docs/architecture.mermaid`

## Technology Stack
- **Language**: Python with Django framework
- **Database**: PostgreSQL (only supported option)
- **Frontend**: Plain CSS with utility-class approach
- **Async messaging**: `django-pgpubsub` for database change notifications
- **Build/Deploy**: Nix

## Running Locally
```
nix-shell
manage runserver
manage fetch_all_channels
manage run_evaluation <commit>
manage listen -v3 --recover
manage ingest_bulk_cve --from DATE --to DATE
```

## CVE Matching Process
Matching CVEs against Nixpkgs metadata is triggered by `pgpubsub` notifications internally as CVEs are ingested. Listeners defined with `@pgpubsub.post_insert_listener` decorators react to database changes asynchronously.

The process generates "untriaged matches" requiring human review before publication as GitHub issues.

## API Status
No public REST API documented. The tracker is a Django web application with HTML views, not an API-first service. Programmatic access would require scraping or direct database access. No GraphQL endpoint mentioned.
