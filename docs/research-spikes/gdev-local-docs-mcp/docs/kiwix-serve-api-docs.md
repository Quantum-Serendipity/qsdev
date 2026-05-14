# kiwix-serve API Documentation
> Source: https://kiwix-tools.readthedocs.io/en/latest/kiwix-serve.html
> Retrieved: 2026-05-14

## Overview
kiwix-serve is an HTTP server for delivering ZIM file content (offline Wikipedia and similar resources). It supports single or multiple ZIM files with search, filtering, and viewing capabilities.

## Command-Line Options
- `kiwix-serve --library LIBRARY_FILE_PATH` - Serve files from XML library
- `kiwix-serve ZIM_FILE_PATH [...]` - Serve individual ZIM files
- `-i ADDR, --address` - Bind to specific IP (values: all, ipv4, ipv6)
- `-p PORT, --port` - HTTP listening port (default: 80)
- `-r ROOT, --urlRootLocation` - URL prefix for content access
- `-d, --daemon` - Run as background service
- `-M, --monitorLibrary` - Auto-reload library on file changes
- `-t N, --threads` - Parallel worker threads (default: 4)
- `-s N, --searchLimit` - Max ZIM files per full-text search
- `-L N, --ipConnectionLimit` - Concurrent connections per IP

## HTTP API Endpoints

### OPDS Catalog (v2)
- `/catalog/v2/root.xml` - Catalog root linking to feeds
- `/catalog/v2/categories` - ZIM file categories as OPDS navigation feed
- `/catalog/v2/languages` - Available languages
- `/catalog/v2/entries` - Filtered, paginated ZIM file listing
- `/catalog/v2/partial_entries` - ZIM listings with partial entry data
- `/catalog/v2/entry/ZIMID` - Full metadata for specific UUID
- `/catalog/v2/illustration/ZIMID?size=N` - Cover art

### Content Access
- `/raw/ZIMNAME/content/PATH` - Unprocessed ZIM entry data
- `/raw/ZIMNAME/meta/METADATAID` - ZIM file metadata

### Search
- `/search` - Full-text search with HTML/XML results
- `/search/searchdescription.xml` - OpenSearch descriptor

### Private/Frontend Endpoints
- `/content/ZIMNAME/PATH` - Processed ZIM content with viewer
- `/viewer#ZIMNAME/PATH` - Interactive ZIM file viewer
- `/suggest?content=ZIMNAME&term=QUERY` - Title-based autocomplete
- `/random?content=ZIMNAME` - Random article redirect

## Search API Details

**ZIM Selection:**
- `content=ZIMNAME` - Single ZIM search (legacy)
- `books.id=UUID` - Search by UUID (repeatable for multi-ZIM)
- `books.name=ZIMNAME` - Search by name (repeatable)
- `books.filter.{criteria}` - Filter using library criteria

**Query Parameters:**
- `pattern` - Search text
- `pageLength` - Results per page (default: 25, max: 140)
- `start` - Pagination offset
- `format` - Response format: html or xml (default: html)

## Library Filtering
- `lang=CODE[,CODE]` - Language codes
- `category=CAT[,CAT]` - Categories
- `tag=TAG[;TAG]` - Tags (AND logic)
- `maxsize=BYTES` - Maximum file size
- `q=TEXT` - Title/description keyword search
- `name=BOOKNAME` - Exact book name match
