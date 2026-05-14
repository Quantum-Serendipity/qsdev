---
source: Multiple web searches (flox.dev/pricing, search results)
retrieved: 2026-03-20
---

# Flox Enterprise Features and Pricing

## Pricing Tiers

### Personal (Free)
- For developers who want to build and share portable, reproducible dev environments using a large catalog of open source software.

### Team ($40/seat/month)
- For teams who need a single way to build, publish, and manage all of the software they use across local dev, CI, and production.

### Enterprise (Custom pricing)
- For orgs who need to curate their software supply chain, work within firm compliance requirements, or manage complex builds.
- Custom deployment options so your software never "leaves the building"
- Custom base catalog to curate the open source software you use.

## Enterprise Positioning

Flox runs on an open-core model, charging for services rendered in the cloud:
- The catalog includes a paid option for storing private packages
- The factory charges per build time
- The manager is available as a paid service
- Premium features such as support for generating Software Bill of Materials (SBOMs)

## Environment Composition and Layering

Flox environments aren't isolated; they layer upon one another, allowing you to combine them in endless ways while keeping your toolsets discrete. Your workspace can be in one environment, a copy of podman layered over that in another, and project data in yet another — all interacting on a single machine.

## Catalog

The Flox Catalog contains over 3 years of history for each of its 100k packages, totaling over a million building blocks. Flox offers SBOMs for every package tracked by the Flox Catalog: more than 190,000 distinct Nixpkgs packages spanning millions of historical package-version combinations.

## SBOM Features

- Flox Catalog SBOMs: available free for all users (190,000+ packages)
- Environment SBOMs: scoped to specific Flox environments showing exact dependency versions, available to commercial users

## Funding

Flox raised $25 million in a Series B round in September 2025 led by Addition, with participation from NEA, Hetz Ventures, Illuminate Financial, and D. E. Shaw.

## Adoption Examples

- PostHog: reduced local dev guide from 16 steps with 14 caveats to a single `flox activate` command
