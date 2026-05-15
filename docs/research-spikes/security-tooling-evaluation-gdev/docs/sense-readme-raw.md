# Sense README.md (Raw)
- **Source**: https://raw.githubusercontent.com/luuuc/sense/main/README.md
- **Retrieved**: 2026-05-15

---

# Sense: Codebase Understanding for AI

**Sense** is an MCP server that provides structural code understanding to AI assistants like Claude Code, Cursor, and Codex CLI. Rather than a tool users operate directly, it runs locally to give AI agents semantic comprehension of codebases.

## Key Capabilities

The platform offers four primary tools:

- **sense_graph**: Maps symbol relationships, callers, inheritance, and dead code
- **sense_search**: Hybrid semantic and keyword search with text fallback
- **sense_blast**: Calculates blast radius and affected code
- **sense_conventions**: Identifies project coding patterns automatically

## Performance Impact

Testing across seven real-world projects showed measurable improvements:

> "Tool calls per task: -47%, Tokens per task: -32%, Cost per task: -26%"

The system particularly excels at structural tasks like identifying call chains and dead code detection, where dependency graphs outperform manual searching.

## Installation & Setup

Users install a single binary via `curl` or download from GitHub releases. Two commands initialize it:

```
sense scan          # Indexes the codebase
sense setup         # Configures AI tools
```

The index updates automatically as code changes, requiring no ongoing maintenance.

## Technical Details

Sense parses code using tree-sitter, extracts symbols and relationships, embeds them with a bundled ONNX model, and stores everything in a local SQLite database at `.sense/`.

It supports 13 languages across two tiers—full support for Ruby, TypeScript, Python, Go, and Rust; standard support for Java, Kotlin, C#, C++, C, PHP, and Scala.

**Resource requirements**: ~60 MB for the binary; 100-200 MB for the index, depending on project size.

## What It Isn't

The creators explicitly define boundaries: not a code editor, not primarily a token optimizer, not a general search engine, and completely independent with zero external dependencies or API keys.
