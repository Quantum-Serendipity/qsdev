<!-- Source: https://raw.githubusercontent.com/freeCodeCamp/devdocs/main/lib/docs/core/doc.rb -->
<!-- Retrieved: 2026-05-14 -->

# DevDocs Doc Class Structure

## Overview
The `Docs::Doc` class is an abstract base for documentation scrapers. It manages metadata, versioning, and storage of documentation pages and indexes.

## Key Metadata Fields

Primary attributes via `attr_accessor`:
- **name**: Documentation title
- **slug**: URL-friendly identifier
- **type**: Documentation category
- **release**: Version information
- **abstract**: Boolean flag (prevents instantiation if true)
- **links**: Related resource URLs

## File Structure

Three JSON files per documentation set:
- `index.json`: Entry index with searchable content
- `db.json`: Page database with rendered output
- `meta.json`: Metadata including modification time and database size

## Versioning System

Documentation supports multiple versions through the `version()` method. When called with a block, it creates a versioned subclass. The `slug` incorporates version information: `"#{slug}~#{version_slug}"` for versioned docs.

The `version_slug` sanitizes version strings by:
- Converting to lowercase
- Replacing `+` with `p`, `#` with `s`
- Removing special characters except underscores and periods

## JSON Output Structure

The `as_json` method returns: `{ name, slug, type, [links], [version], [release] }`

## Storage Operations

- `store_page()`: Stores individual pages with instrumentation
- `store_pages()`: Batch stores all pages, creating index, database, and metadata files

## Index Structure

Each entry in `index.json` has:
- **name**: Display name of the entry
- **path**: Relative path to the HTML file (can include #fragment)
- **type**: Category for sidebar grouping

The `db.json` maps paths to HTML content strings.
