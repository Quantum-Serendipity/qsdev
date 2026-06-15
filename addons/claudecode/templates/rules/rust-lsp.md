---
paths: ["**/*.rs", "**/Cargo.toml", "**/Cargo.lock"]
---

# Rust LSP (rust-analyzer)

Use the `LSP` tool, not Grep, to navigate Rust symbols here. `rust-analyzer` is
configured for this project.

- `goToDefinition` / `findReferences` / `workspaceSymbol` resolve symbols across
  crates and modules.
- `hover` performs macro expansion, so you can see what a macro invocation
  actually generates — invaluable for `derive` and declarative macros that Grep
  cannot expand.
- `goToImplementation` navigates between traits and their `impl` blocks.
- Borrow-checker and type diagnostics surface automatically after edits.
- Clippy lints are integrated via `check.command = "clippy"` and appear in
  diagnostics.

Reach for Grep only for string literals, `Cargo.toml` keys, and build scripts
that rust-analyzer does not index.
