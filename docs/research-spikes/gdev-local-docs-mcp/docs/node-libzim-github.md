# node-libzim: Node.js ZIM File Binding
> Source: https://github.com/openzim/node-libzim
> Retrieved: 2026-05-14

## Description
A Node.js binding to libzim enabling developers to read and write ZIM files in JavaScript.

## Core Features
- Read ZIM files: Access archived content through iterators and entry lookups
- Write ZIM files: Create new ZIM archives with configurable compression and indexing
- Full-text search: Integrated search and suggestion capabilities
- Content compression: Cluster-based compression support

## API Overview

**Writing:**
- `Creator`: Configure and build ZIM archives with options for worker threads, indexing, and cluster size
- `StringItem`: Add text-based content entries with metadata

**Reading:**
- `Archive`: Open and navigate ZIM files
- `Query` / `Searcher`: Execute full-text searches
- `SuggestionSearcher`: Generate auto-complete suggestions
- Entry iteration via `iterByPath()`

## Package Details
- **NPM**: `@openzim/libzim`
- **Stars**: 32
- **License**: GPLv3
- **Latest Release**: v4.1.0 (March 23, 2026)

## Dependencies
Built on node-addon-api/N-API. Requires separate libzim installation for non-Linux/macOS platforms. Auto-downloads libzim binary on Linux/macOS.

## Platform Support
- GNU/Linux (automatic binary download)
- macOS (automatic binary download)
- Other OSes (manual libzim installation required)

## Languages
C++ (78.3%), TypeScript (15.4%), JavaScript (3.4%)
