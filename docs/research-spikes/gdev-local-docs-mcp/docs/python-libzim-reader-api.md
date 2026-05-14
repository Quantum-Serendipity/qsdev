# python-libzim Reader API Reference
> Source: https://python-libzim.readthedocs.io/en/latest/api_reference/libzim.reader/
> Retrieved: 2026-05-14

## Archive Class

**Constructor:** `Archive(filename: Path)`

**Key Properties:**
- `all_entry_count` -> int: Total entries including internal/metadata
- `article_count` -> int: Number of articles
- `entry_count` -> int: User entries only
- `media_count` -> int: Media items
- `filename` -> Path, `filesize` -> int
- `uuid` -> UUID, `checksum` -> str
- `has_fulltext_index` -> bool
- `has_title_index` -> bool
- `has_main_entry` -> bool
- `has_new_namespace_scheme` -> bool
- `main_entry` -> Entry
- `metadata_keys` -> list[str]

**Methods:**
- `check()` -> bool: Validates checksum
- `get_entry_by_path(path)` -> Entry
- `get_entry_by_title(title)` -> Entry
- `has_entry_by_path(path)` -> bool
- `has_entry_by_title(title)` -> bool
- `get_random_entry()` -> Entry
- `get_metadata(name)` -> bytes
- `get_metadata_item(name)` -> Item

## Entry Class
- `path` -> str, `title` -> str
- `is_redirect` -> bool
- `get_item()` -> Item
- `get_redirect_entry()` -> Entry

## Item Class
- `content` -> memoryview (raw data bytes)
- `size` -> int, `path` -> str, `title` -> str
- `mimetype` -> str

## Usage Example
```python
with Archive(fpath) as zim:
    entry = zim.get_entry_by_path(zim.main_entry.path)
    print(f"Article {entry.title} at {entry.path} is "
          f"{entry.get_item().content.nbytes}b")
```

## Search (separate module)
Full-text search via Query/Searcher classes. Suggestion search via SuggestionSearcher.
