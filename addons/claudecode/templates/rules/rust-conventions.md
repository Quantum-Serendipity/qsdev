# Rust Conventions

## Error handling
- Use `thiserror` for defining error types in library crates. Each error
  variant should have a descriptive message.
- Use `anyhow` for application-level error propagation where structured
  error types are unnecessary.
- Prefer `?` operator over explicit `match` on `Result` for error propagation.
- Never use `.unwrap()` or `.expect()` in library code. Reserve them for
  cases where the invariant is proven and documented.

## Ownership and borrowing
- Prefer `&str` over `String` in function parameters when the function does
  not need ownership.
- Prefer `&[T]` over `Vec<T>` in function parameters for the same reason.
- Use `Cow<'_, str>` when a function may or may not need to allocate.
- Avoid unnecessary cloning. If you need `clone()`, add a comment explaining why.

## Types and traits
- Derive `Debug` on all public types.
- Derive `Clone`, `PartialEq`, `Eq`, `Hash` where semantically appropriate.
- Use `#[non_exhaustive]` on public enums and structs that may grow.
- Prefer newtype wrappers over type aliases for domain concepts.

## Safety
- Minimize `unsafe` blocks. Every `unsafe` block must have a `// SAFETY:`
  comment explaining why the invariants are upheld.
- Prefer safe abstractions from well-audited crates over hand-written unsafe code.
- Run `cargo clippy -- -W clippy::pedantic` and address all warnings.

## Project structure
- Keep `main.rs` thin: parse args with `clap`, set up logging, call library code.
- Use modules (`mod`) to organize code by responsibility.
- Public API surface should be explicitly chosen with `pub` visibility.

## Testing
- Place unit tests in `#[cfg(test)] mod tests` at the bottom of each file.
- Use `#[test]` functions with descriptive names.
- Use `assert_eq!` for value comparison and `assert!(matches!(...))` for
  pattern matching.
- Integration tests go in the `tests/` directory.

## Dependencies
- Run `cargo audit` regularly to check for known vulnerabilities.
- Pin dependency versions in `Cargo.toml` with exact or tilde requirements.
- Prefer well-maintained crates with active security response policies.
