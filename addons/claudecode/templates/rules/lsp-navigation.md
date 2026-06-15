# LSP Navigation

This project has LSP servers configured. Use the `LSP` tool — not Grep — for
anything to do with code symbols. It is far faster and uses far fewer tokens
than grepping, and it understands the code instead of matching text.

## Use the `LSP` tool for

- `goToDefinition` — jump to where a symbol is defined (do this before editing
  unfamiliar code).
- `findReferences` — find every use of a symbol (do this before renaming or
  refactoring).
- `workspaceSymbol` — locate a symbol by name across the whole project.
- `hover` — inspect a symbol's type, signature, or doc comment.
- `goToImplementation` — find concrete implementations of an interface or
  abstract method.
- call hierarchy — trace incoming/outgoing calls of a function or method.

## Use Grep / Glob instead for

- String literals, log messages, comments, and TODO/FIXME markers.
- Config files, data files, and anything not source code.
- File-pattern / filename matching (use Glob).
- Languages with no configured LSP server.

## Pre-edit workflow (mandatory)

1. Before editing unfamiliar code, run `LSP` `goToDefinition` on the symbols
   involved to understand them.
2. Before renaming or refactoring, run `LSP` `findReferences` to see every
   call site.
3. After every edit, rely on the automatic diagnostics — they surface errors
   without a manual lint/build step.
