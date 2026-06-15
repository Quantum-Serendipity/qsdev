---
paths: ["**/*.py", "**/pyproject.toml"]
---

# Python LSP (pyright)

Use the `LSP` tool, not Grep, to navigate Python symbols here. `pyright` is
configured for this project.

- `goToDefinition` / `findReferences` / `workspaceSymbol` resolve symbols across
  modules and packages.
- pyright performs type inference even on untyped code, so `hover` shows
  inferred types where no annotation exists in the source.
- `__init__.py` re-exports are resolved transparently — `goToDefinition` follows
  a re-exported name to its real definition, which Grep cannot do.
- Type-checking diagnostics (`typeCheckingMode = "basic"`) surface automatically
  after edits.

Reach for Grep only for string literals, `pyproject.toml` keys, and dynamic
attribute access pyright cannot statically resolve.
