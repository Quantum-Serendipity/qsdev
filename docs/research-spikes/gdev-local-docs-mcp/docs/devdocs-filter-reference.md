<!-- Source: https://raw.githubusercontent.com/freeCodeCamp/devdocs/main/docs/filter-reference.md -->
<!-- Retrieved: 2026-05-14 -->

# DevDocs Filter Reference

## Core Architecture

Filters extend `Docs::Filter` and require a `call` method. They're divided into two types:

**HTML Filters:** Manipulate the Nokogiri node object (`doc`). Must return `doc`.

**Text Filters:** Manipulate string representation (`html`). Must return `html`.

The pipeline executes HTML filters first, then text filters, avoiding redundant document parsing.

## Key Instance Methods

- `doc` — Nokogiri node representation
- `html` — String representation  
- `context` — Frozen hash with scraper options and URL data
- `result` — Stores metadata and page information
- `css()`, `xpath()` — DOM query shortcuts
- `slug` — "The `subpath` removed of any leading slash or `.html` extension"

## EntriesFilter: The Index Generator

This abstract filter extracts page metadata for the documentation index. Each scraper must implement it by overriding four methods:

**`get_name`** — Extracts the page's primary entry name (usually derived from HTML headings or the slug)

**`get_type`** — Determines the entry category for sidebar organization

**`include_default_entry?`** — Controls whether the page itself appears in the index

**`additional_entries`** — Returns an array of secondary entries: `[['Name', 'fragment-id', 'type']]`

The fragment ID links to element IDs (typically headings), creating searchable subsections within pages.
