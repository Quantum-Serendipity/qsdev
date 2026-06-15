---
paths: ["**/*.go", "**/go.mod", "**/go.sum"]
---

# Go LSP (gopls)

Use the `LSP` tool, not Grep, to navigate Go symbols here. `gopls` is configured
for this project.

- `goToDefinition` / `findReferences` / `workspaceSymbol` resolve Go symbols
  precisely across packages and modules.
- `goToImplementation` navigates between interfaces and their implementing
  types — Grep cannot do this reliably.
- `nilness` analysis flags guaranteed nil dereferences and impossible nil
  comparisons; it surfaces automatically in diagnostics after edits.
- `govulncheck` diagnostics report known vulnerabilities in imported packages.
- `hover` shows inferred types, signatures, and doc comments.

Reach for Grep only for string literals, struct tags, build tags, or generated
files that gopls does not index.
