---
paths: ["**/*.ts", "**/*.tsx", "**/*.js", "**/*.jsx", "**/*.mts", "**/*.cts"]
---

# TypeScript / JavaScript LSP (typescript-language-server)

Use the `LSP` tool, not Grep, to navigate TypeScript/JavaScript symbols here.
`typescript-language-server` is configured for this project.

- `goToDefinition` / `findReferences` / `workspaceSymbol` resolve symbols across
  modules, including through barrel files (`index.ts` re-exports) that Grep
  cannot follow.
- `goToImplementation` navigates from interfaces and abstract members to their
  implementations.
- Strict null-check diagnostics (when `strict` is enabled) surface automatically
  after edits.
- `hover` reveals inferred generic type arguments and resolved conditional/
  mapped types that never appear literally in the source.

Reach for Grep only for string literals, JSON config, and `.d.ts`-free runtime
assets the server does not index.
