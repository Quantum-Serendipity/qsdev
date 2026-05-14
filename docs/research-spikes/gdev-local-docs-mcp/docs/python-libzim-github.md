# python-libzim (openzim/python-libzim)
> Source: https://github.com/openzim/python-libzim
> Retrieved: 2026-05-14

## Description
Python-libzim is a Python binding for the C++ libzim library, enabling users to read and write ZIM files -- offline content archives. The project serves as a foundational tool for openZIM scrapers like sotoki and youtube2zim.

## Key Features
- **Reading**: Access ZIM file contents with archive browsing capabilities
- **Writing**: Create ZIM files programmatically with metadata and illustrations
- **Full-text search**: Query indexed content with the Query and Searcher classes
- **Suggestions**: Access suggestion search functionality via SuggestionSearcher
- **Thread support**: Reading is mostly thread-safe; searching and creation require synchronization

## Installation & Platform Support
Available via PyPI (`pip install libzim`). Pre-built wheels support:
- **macOS**: x86_64, arm64
- **Linux**: x86_64, armhf, aarch64 (glibc and musl variants)
- **Windows**: x64

Limited to CPython; source distribution available for other platforms.

## Core API Overview

### Reading
```python
Archive("file.zim")  # open archive, access entries by path
Query().set_query(text)  # full-text search
SuggestionSearcher  # autocomplete suggestions
```

### Writing
Subclass `Item` with content providers (StringProvider/FileProvider), then use `Creator` to build archives with metadata and indexing options.

## Metadata
- **Version**: 3.9.0 (latest, March 2026)
- **Stars**: 104
- **Forks**: 29
- **License**: GPLv3
- **Languages**: Python (50.5%), Cython (38.7%), C++ (10.8%)
- **Latest release**: 21+ releases available

## Notable Implementation Details
The library "disables the GIL on most of C++ libzim calls," requiring manual thread synchronization via locks for concurrent operations. It wraps the C++ libzim via Cython bindings, so it includes compiled native code in the wheel.
