<!-- Source: https://changelog.com/posts/nixos-fatal-flaw -->
<!-- Retrieved: 2026-03-20 -->

# NixOS Has One Fatal Flaw - Tammer Saleh (Changelog)

## Core Argument

Tammer Saleh identifies NixOS's fatal flaw as **the usability of Nix itself** — specifically its steep learning curve and poor user experience that prevents adoption at scale.

## Shopify Case Study

The article references Shopify as a prominent example of enterprise adoption difficulty:

> "Shopify was touted as a place that was going to use Nix holistically, throughout their entire developer experience... And they tried."

However, according to engineers at Shopify:

> "No, it was too hard to understand, especially for our new engineers."

## Key Technical Points

**Docker vs. Nix Comparison:**
- Docker solves three problems: secure container execution, dependency packaging, and distribution
- Nix solves two: dependency packaging and distribution, but lacks runtime isolation/security features
- Docker dominates due to momentum and ubiquity (Docker Hub ecosystem)

**Scaling Limitations:**
Saleh argues Nix works well in isolated environments but "doesn't scale to any larger environment" due to its learning curve.

## Conclusion

While acknowledging Nix's technical merit, Saleh contends it remains "fantastic" for small teams but "not for the masses" — Docker's territory.

## Relevance to Shopify Story

This article captures the EXTERNAL perception of Shopify's first Nix attempt: that it simply failed due to complexity. The NixCon 2025 talk and subsequent podcast reveal the story is more nuanced — the first attempt didn't completely fail, it "didn't reach stable footing" before being superseded by a company-wide pivot to cloud development (Spin). The devenv revival later proved that Nix COULD work at Shopify when wrapped in better UX.
