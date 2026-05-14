<!-- Source: GitHub API reads from freeCodeCamp/devdocs repository (lib/docs/core/) -->
<!-- Retrieved: 2026-05-14 -->

# DevDocs Data Format — From Source Code Analysis

## File Structure Per Documentation Set

Each documentation set (e.g., `python~3.12/`) contains three files:

### 1. `index.json`
Structure: `{ entries: [...], types: [...] }`

Each **entry** has:
- `name` (String): Display name, stripped of whitespace
- `path` (String): Relative path to HTML file, can include #fragment
- `type` (String): Category name for sidebar grouping

Each **type** has:
- `name` (String): Type display name
- `slug` (String): Parameterized name (for URLs)
- `count` (Integer): Number of entries of this type

Entries are sorted alphabetically (case-insensitive) with semantic version sorting for numeric prefixes.

### 2. `db.json`
Structure: `{ "path": "html_content", ... }`

A simple hash mapping paths to HTML content strings. Each key is a page path, each value is the rendered HTML partial for that page.

### 3. `meta.json`
Structure: `{ name, slug, type, [links], [version], [release], mtime, db_size }`

- `name`: Documentation title
- `slug`: URL-friendly identifier (e.g., "python~3.12")
- `type`: Documentation category
- `links`: Related resource URLs (optional)
- `version`: Version string (optional)
- `release`: Software version (optional)
- `mtime`: Unix timestamp of last update
- `db_size`: Size of db.json in bytes

## Global Manifest

### `docs.json`
Array of meta.json objects for all available documentation sets, plus:
- `attribution`: HTML string with copyright/license info
- `alias`: Short alias for the documentation

## Entry Model (entry.rb)

```ruby
class Entry
  attr_accessor :name, :type, :path
  # name: required, stripped
  # path: required (unless root)
  # type: required (unless root)
  # Root entry has path == 'index'
end
```

## Versioning

Version slugs are sanitized: lowercase, `+` → `p`, `#` → `s`, non-alphanumeric → `_`.
Documentation slug format: `"#{base_slug}~#{version_slug}"` (e.g., `python~3.12`).

## PageDb (page_db.rb)

Simple hash-based storage:
```ruby
class PageDb
  def add(path, content)
    @pages[path] = content
  end
  def as_json
    @pages  # { path_string => html_content_string }
  end
end
```

## Key Insight for MCP Integration

The data format is simple and well-suited for direct file access:
- `index.json` provides a searchable index with name/path/type triples
- `db.json` provides the actual HTML content keyed by path
- No database required — pure JSON files
- No server required for read access — just filesystem access to the generated files
