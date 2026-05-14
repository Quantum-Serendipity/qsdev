# javascript-libzim: WebAssembly ZIM Reader
> Source: https://github.com/openzim/javascript-libzim
> Retrieved: 2026-05-14

## Overview
Compiles the libzim C++ library to WebAssembly (WASM) and ASM.js, enabling ZIM archive reading in web browsers and Node.js environments.

- **Repository:** openzim/javascript-libzim
- **License:** GPL-3.0
- **Stars:** 4
- **Languages:** JavaScript (82.7%), HTML (5.6%), C++ (4.6%), Makefile (4.2%)

## Features
- Full-text search with optional language specification
- Enhanced search with snippets displaying content excerpts with highlighted terms
- Autocomplete/suggestion functionality using SuggestionSearcher class
- Archive management (loading, article access, redirect handling)
- Web Worker support using WORKERFS file system
- Large file handling (multi-gigabyte archives)

## JavaScript API

**Archive Access:**
- `Module.loadArchive(filename)` - Load ZIM file
- `Module.getArticleCount()` - Retrieve article count
- `Module.getEntryByPath(path)` - Access specific entries

**Search Functions:**
- `Module.search(query, maxResults)` - Basic full-text search
- `Module.searchWithSnippets(query, maxResults)` - Search with content excerpts
- `Module.searchWithLanguage(query, maxResults, language)` - Language-specific search

**Suggestions:**
- `Module.suggest(query, maxResults)` - Title-based autocomplete

## Platform Support
- Web browsers (primary target, WASM with WORKERFS)
- Node.js (supported via NODEFS)
- Web Workers required for WORKERFS implementation

## API Stability
API considered unstable until v1.0 release (currently v0.x); breaking changes possible.
