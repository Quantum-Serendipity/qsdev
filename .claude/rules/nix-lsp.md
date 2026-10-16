---
paths: ["**/*.nix", "**/flake.nix", "**/flake.lock"]
---

# Nix LSP (nixd)

Use the `LSP` tool, not Grep, to navigate Nix symbols here. `nixd` is configured
for this project and is always available in qsdev environments.

- `goToDefinition` / `findReferences` resolve `let` bindings, function
  parameters, and `with`-scoped names.
- Attribute-path navigation lets `goToDefinition` follow `a.b.c` selections to
  the attribute that defines them, which Grep cannot resolve.
- NixOS / Home Manager / devenv option types resolve via `hover`, so you can see
  the expected type and documentation of an option without leaving the editor.
- `workspaceSymbol` locates top-level bindings across `.nix` files.

Reach for Grep only for string literals and comments inside Nix expressions.
