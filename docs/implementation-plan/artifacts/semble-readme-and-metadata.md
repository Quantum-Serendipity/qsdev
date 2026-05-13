# Semble - Repository README and Metadata

- **Source**: https://github.com/MinishLab/semble
- **Retrieved**: 2026-05-12

## Repository Metadata

- **Stars**: 798
- **Forks**: 61
- **Language**: Python (99.7%), Makefile (0.3%)
- **License**: MIT
- **Latest Release**: v0.1.7 (May 12, 2026)
- **Topics**: retrieval, mcp, embeddings, code-search, agents, model-context-protocol, mcp-server
- **Requires Python**: >=3.10
- **Authors**: Thomas van Dongen, Stephan Tulkens (MinishLab)
- **Development Status**: Beta (4 - Beta)

## README Summary

Semble is a code search library designed for AI agents. It returns the exact code snippets they need instantly, using ~98% fewer tokens than grep+read. It performs comprehensive codebase indexing and searching in under one second using CPU-only processing, with no external APIs or GPU requirements.

### Key Features

- **Performance**: Indexes repositories in ~250ms and answers queries in ~1.5ms on CPU
- **Accuracy**: Achieves 0.854 NDCG@10 score, comparable to larger transformer models
- **Token Efficiency**: Uses approximately 98% fewer tokens than traditional grep-and-read approaches
- **No External Dependencies**: Operates entirely on CPU without API keys or GPU
- **Multi-Agent Support**: Functions as an MCP server compatible with Claude Code, Cursor, Codex, and OpenCode

### Core Capabilities

Two primary MCP tools:
1. **search**: Natural language or code queries across codebases (local directories or git URLs)
2. **find_related**: Discovers semantically similar code chunks given a file path and line number

### Technical Approach

Combines multiple retrieval mechanisms:
- Model2Vec static embeddings using the potion-code-16M model for semantic matching
- BM25 algorithm for lexical identifier matching
- Reciprocal Rank Fusion to combine score lists
- Code-aware reranking signals including definition boosts and file coherence weighting

### Installation

```bash
pip install semble
# or
uv tool install semble
```

For MCP server use:
```bash
pip install 'semble[mcp]'
```

## Release History

- v0.1.7 (May 12, 2026) - Fixed savings aggregation issues
- v0.1.6 (May 11, 2026) - Allowlist-style .gitignore fix
- v0.1.5 (May 11, 2026) - Pinned tree-sitter-language-pack
- v0.1.4 (May 11, 2026) - Added savings command, bash integration docs, bounded git transports, chunking module replacing chonkie, Python 3.14 CI
- v0.1.3 (May 5, 2026) - Improved readme flow, pinned versions
- v0.1.2 (May 4, 2026) - Auto-reindexing for local paths, CONTRIBUTING.md, release workflow
- v0.1.1 (April 30, 2026) - CLI functionality, token efficiency benchmarks
- v0.1.0 (April 26, 2026) - Initial release
