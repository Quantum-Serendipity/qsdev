<!-- Source: https://shopify.github.io/shadowenv/ -->
<!-- Retrieved: 2026-03-20 -->

# Shadowenv - Shopify

**Creator:** Shopify (copyright 2019)

## Core Purpose
Shadowenv performs "project-local environment variable shadowing" — automatic variable configuration upon entering a project directory and restoration upon exit.

## Documentation Sections
1. **Getting Started** — installation and initial setup
2. **Shadowlisp API** — programming language for writing Shadowenv programs
3. **Best Practices** — recommended usage patterns
4. **Trust** — security system preventing unauthorized use
5. **Integration** — editor integrations and custom integration development

## Relevance to Shopify's Nix Story
Shadowenv is a key piece of Shopify's developer tooling infrastructure. It was created in 2019 as part of Burke Libbey's effort to build Nix-based development environments. The `dev` tool + shadowenv + Nix formed the core stack:
- `dev up` would enforce a specific nixpkgs revision
- shadowenv would manage environment variables per-project
- Nix would provide the actual packages

The NixCon 2019 code gist includes `setup-hook-to-shadowenv`, a script converting Nix package setup-hooks into shadowenv format — demonstrating how tightly integrated these tools were.
