# libzim C++ API Usage Documentation
> Source: https://libzim.readthedocs.io/en/latest/usage.html
> Retrieved: 2026-05-14

## Core Architecture
All classes reside in the `zim` namespace, use exception handling via std::exception-derived errors. Reading is generally thread-safe; creation requires serialized access.

## Opening Archives
```cpp
zim::Archive archive("wikipedia.zim");
```
Archives use reference semantics (copies reference the same file).

## Iterating Entries
```cpp
for (auto entry: archive.iterByPath()) {
  std::cout << entry.getPath() << " " << entry.getTitle() << std::endl;
}
```

## Retrieving Entries
- `getEntryByPath(const std::string&)` - by path (throws EntryNotFound)
- `getEntryByTitle(const std::string&)` - by title (throws EntryNotFound)
- Entries may be redirects or items: `entry.isRedirect()`, then `entry.getRedirectEntry()` or `entry.getItem()`

## Prefix Search
- `findByPath(std::string)` -> EntryRange
- `findByTitle(std::string)` -> EntryRange

## Full-Text Search
```cpp
zim::Searcher searcher(archive);
zim::Query query;
query.setQuery("bar");
zim::Search search = searcher.search(query);
zim::SearchResultSet results = search.getResults(10, 20);
for(auto entry: results) { ... }
```

## Suggestions API
```cpp
zim::SuggestionSearcher searcher(archive);
zim::SuggestionSearch search = searcher.search("bar");
zim::SuggestionResultSet results = search.getResults(10, 20);
```

## Key Classes
| Class | Purpose |
|-------|---------|
| Archive | Archive access point |
| Entry | Entry reference (redirect or item) |
| Item | Actual content container |
| Searcher | Full-text search engine |
| SuggestionSearcher | Suggestion lookup |
| EntryRange | Iterable entry collection |
