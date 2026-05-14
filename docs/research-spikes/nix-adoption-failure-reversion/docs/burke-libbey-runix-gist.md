<!-- Source: https://gist.github.com/burke/72ca46c80e57a25907a75611ee5eb66d -->
<!-- Retrieved: 2026-03-20 -->

# ru.nix - Runix Project Configuration Example (Burke Libbey, Shopify)

**Created:** June 8, 2020

## Overview

This is an example of Shopify's "Runix" system — a Nix-module-based approach for declaring projects abstractly. Each project gets a `ru.nix` file that describes its dependencies, services, and commands in a declarative format.

## File Content: ru.nix for "shopify-pay"

**Project Metadata:**
- Project name: "shopify-pay"
- Repository name: "pay"
- Framework: Rails (enabled)

**Development Dependencies (pkgs):**
geolite2, ngrok, mysqlClient57, overmind, watchman, toxiproxy, v8

**Server Configuration:**
- Hostname: pay.myshopify.io
- Port: 40018

**Environment Variables:**
- KARAFKA_BOOT_FILE set to "./shopify_pay_karafka/config/application.rb"

**Integrations:**
shop-accounts, cardsink-copy, cardserver-copy, hosted-fields-copy

**Custom Commands Defined:**
- `server`: Launches overmind with Procfile.dev
- `karafka`: Runs karafka process
- `sidekiq`: Manages queues (payment_create, payment_poll, mailers, default, maintenance, kafka)
- `ngrok`: Initiates ngrok tunnel from config/ngrok.yml
- `staging`: Pushes branch to staging and opens deployment URL

**Git Hooks:**
Pre-commit hook configured at ./bin/git/pre-commit.sh

## Significance

This demonstrates Shopify's first-attempt approach to Nix adoption: a custom "Runix" module system that provided a high-level declarative interface over raw Nix. Projects defined their needs in `ru.nix` files, and the Runix system resolved them into Nix derivations. This is conceptually very similar to what devenv.sh later provided — both are abstraction layers over Nix for developer environments.

The parallel is notable: devenv succeeded where Runix didn't gain traction, likely because devenv had a broader community, better documentation, and didn't require Shopify's internal team to maintain the entire abstraction layer themselves.
