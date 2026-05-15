# Crates.io README Rendering and Rust Package Documentation
- **Source**: Multiple search results (crates.io, GitHub PRs, cargo-readme/cargo-rdme crate pages)
- **Retrieved**: 2026-05-15

## README Display on crates.io

On the crate/version route, the README contents is displayed in place of the package description. This means the README is the primary content users see when visiting a crate page.

## Rendering and Security

- crates.io renders README markdown to HTML
- The generated HTML is sanitized using Ammonia, which removes all tags and attributes that are not whitelisted
- This means some GitHub-specific markdown features (Mermaid diagrams, alerts/admonitions, math expressions) may not render on crates.io even though they work on GitHub

## Documentation Generation Tools (Rust-Specific)

1. **cargo-readme**: Populates README.md with contents of doc comments from lib.rs or main.rs — keeps README in sync with code docs
2. **cargo-rdme**: Inserts crate documentation into README; can verify README is up to date via `cargo rdme --check`
3. **cargo-sync-readme**: Another synchronization tool for keeping README aligned with rustdoc comments

## Cross-Platform Implications

Because crates.io uses its own markdown renderer (not GitHub's API like npm does), README authors need to be aware that:
- GitHub-specific extensions may not render
- Stick to standard/CommonMark markdown for maximum compatibility
- Test rendering on crates.io after publishing, not just on GitHub
