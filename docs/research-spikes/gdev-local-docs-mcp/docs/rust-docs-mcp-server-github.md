<!-- Source: https://github.com/Govcraft/rust-docs-mcp-server -->
<!-- Retrieved: 2026-05-14 -->

# Rust Docs MCP Server Analysis

## What It Does
This project provides an MCP (Model Context Protocol) server that prevents AI coding assistants from generating outdated Rust code. It delivers current crate documentation to LLMs through semantic search and embeddings, enabling assistants like Cursor and Cline to query live API information before writing code.

## Architecture
The system operates through these stages:

1. **Documentation Generation**: Creates a temporary Rust project and executes `cargo doc` with optional specified features to generate HTML documentation
2. **Content Extraction**: Parses HTML files using the `scraper` crate to isolate text from main content sections
3. **Embedding Creation**: Generates semantic embeddings via OpenAI's `text-embedding-3-small` model with token counting via `tiktoken-rs`
4. **Caching Layer**: Stores documentation and embeddings in XDG data directories using `bincode` serialization, segregating by crate, version, and features
5. **Query Processing**: Computes cosine similarity between user questions and cached embeddings, then sends top matches to `gpt-4o-mini-2024-07-18` for contextual answering

## Documentation Fetching
The server downloads Rust crate documentation by creating a temporary project with the target crate as a dependency, then running `cargo doc` with the Cargo library API. It dynamically locates output in `target/doc` by searching for `index.html`, then extracts content from HTML files rather than querying docs.rs or relying on pre-built sources.

## Tools & Resources Provided
- **`query_rust_docs` Tool**: Accepts a question string, returns context-grounded answers prefixed with "From <crate_name> docs:"
- **`crate://<crate_name>` Resource**: Provides the configured crate name as plain text

## Repository Metrics
- **Stars**: 275
- **Language**: Rust (89.2%), Nix (10.8%)
- **License**: MIT
- **Latest Release**: v1.3.1 (May 8, 2025)
- **Total Releases**: 14

## Caching Behavior
Documentation and embeddings cache in `~/.local/share/rustdocs-mcp-server/<crate_name>/<sanitized_version>/<features_hash>/`. The cache automatically regenerates if missing or corrupted, with distinct entries per feature combination to prevent conflicts.

## Security & Notable Properties
- Requires `OPENAI_API_KEY` environment variable
- Needs internet access for API calls and crate downloads
- Runs over stdio as a standard MCP server
- Sends informational logs via `logging/message` notifications
- Single-developer project (Govcraft) with sponsorship support option
