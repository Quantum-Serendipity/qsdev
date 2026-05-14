<!-- Source: https://michael.stapelberg.ch/posts/2025-09-21-nixcon-2025-trip-report/ -->
<!-- Retrieved: 2026-03-20 -->

# NixCon 2025 Trip Report - Michael Stapelberg

## Shopify's Nix Adoption Story

Josh Heinrichs from Shopify presented "Nix-based development environments at Shopify (reprise)" on Friday. Stapelberg characterized this as a "real-world enterprise adoption story" that proved particularly interesting.

### Key Points from Shopify's Journey

**Initial Context (2016):**
Shopify had developed a `dev` command offering declarative configuration that dispatched to `apt` (Linux) or `homebrew` (macOS).

**First Attempt:**
The initial Nix migration "didn't reach stable footing" with some team members unable to use it. A company-wide shift to cloud development environments made the easier solution preferable at that time.

**Renewed Adoption:**
Years later, CEO Tobias Lütke discovered devenv (https://devenv.sh), which closely resembled Shopify's original `dev` tool. His adoption and support became catalytic.

### Critical Success Factors

The second attempt succeeded through:
- Incremental adoption strategies
- Stakeholder engagement and buy-in
- Extended rollout planning

### Main Takeaway

"One specific, well-supported use-case can be the adoption driver." Once development environments use Nix-based solutions, broader ecosystem adoption becomes more feasible.

Recording: 19 minutes on media.ccc.de
