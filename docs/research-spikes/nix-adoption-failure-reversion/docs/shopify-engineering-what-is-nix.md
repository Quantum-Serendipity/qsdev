<!-- Source: https://shopify.engineering/what-is-nix -->
<!-- Retrieved: 2026-03-20 -->

# What Is Nix - Shopify Engineering Blog

## Author
Burke Libbey, Shopify

## Core Concept

Burke Libbey explains that "Everything on your computer implicitly depends on a whole bunch of other things on your computer." Nix's innovation is making this typically invisible dependency graph explicit and manageable.

## Four Building Blocks

**1. The Nix Store**
The `/nix/store` directory functions as a graph database where each entry is immutable after creation. Entries follow the pattern `<hash>-<name>`, with the hash derived from their contents. Dependencies are detected by scanning for literal path references within nodes — a surprisingly reliable method in practice.

**2. Derivations**
These special store nodes contain build instructions. A derivation specifies inputs, outputs, builder programs, arguments, and environment variables needed to construct other store paths. When evaluated, they create `.drv` files as a side effect, the only function in Nix with side effects.

**3. Sandboxing**
Builds operate in restricted environments with access only to explicitly declared dependencies. Nix patches compilers and linkers to prevent default system library lookups, ensuring "artifacts in the Nix Store essentially can't depend on anything outside of the Nix Store."

**4. The Nix Language**
This domain-specific language features lazy evaluation and functional purity. It cannot perform I/O operations, networking, or file writing — its only effect is calling the `derivation` function to create build recipes.

## Practical Implementation

The article demonstrates querying store relationships using `nix-store --query` commands, showing how to inspect direct references and transitive closures of dependencies. A Ruby installation example visualizes the complete dependency graph complexity.

## Shopify's Adoption

Libbey mentions Shopify has been "progressively rebuilding parts of our developer tooling with Nix" and hints at future expansion ("spoiler: everything?"), though specific implementation details remain reserved for follow-up content.
