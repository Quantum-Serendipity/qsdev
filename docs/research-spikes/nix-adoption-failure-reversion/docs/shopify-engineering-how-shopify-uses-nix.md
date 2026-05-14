<!-- Source: https://shopify.engineering/shipit-presents-how-shopify-uses-nix -->
<!-- Retrieved: 2026-03-20 -->

# ShipIt! Presents: How Shopify Uses Nix

## Presenter & Event
Burke Libbey presented "How Shopify Uses Nix" at ShipIt!, Shopify's monthly engineering event series, on May 25, 2020.

## Developer Tooling Implementation
Shopify rebuilt their developer tooling infrastructure using Nix. The presentation demonstrated practical tools the company uses daily, building on Libbey's earlier "What is Nix" post.

## Key Technical Details

**The `dev` Tool:**
The company created a development environment tool that enforces a "specific nixpkgs revision" each time developers run `dev up`. This approach addresses consistency across the team.

**shadowenv Strategy:**
Shopify implemented a custom approach distinct from alternatives like lorri. They were open to compatibility improvements but noted their strategy differed fundamentally.

**Gem Dependency Management:**
They initially prioritized managing Ruby gem dependencies with Nix because gems "populate a global cache by default." Node modules management was planned but required substantial engineering effort.

## Challenges Encountered

Libbey identified that Nix's "tooling is in general really optimized for 'build' workflows, not development workflows." This misalignment required custom solutions for bundleEnv/bundleApp workflows to match Shopify's actual usage patterns.

**Remote Work Considerations:**
The shift to work-from-home revealed bandwidth constraints when updating dependencies, as large downloads became problematic over residential internet versus office infrastructure.

## Future Plans
CI/CD integration with Nix was targeted for late 2020, though not yet implemented at presentation time.
