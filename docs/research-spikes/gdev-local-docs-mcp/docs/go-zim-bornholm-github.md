# go-zim (Bornholm/go-zim)
> Source: https://github.com/Bornholm/go-zim
> Retrieved: 2026-05-14

## Description
"A Golang library to read and serve ZIM archives" -- inspired by github.com/tim-st/go-zim.

## Key Features
- ZIM file reading: Open and parse ZIM archives with metadata access
- HTTP server integration: Serve ZIM archives through a web server using http.FileServer
- Entry access: Query entry counts and retrieve main page information
- Compression support: Handles multiple compression formats (uncompressed, compressed, XZ, Zstandard)
- Filesystem abstraction: Includes an fs package providing http.FS compatibility

## API Highlights
- `zim.Open(path)` - opens an archive
- `reader.EntryCount()` - retrieves entry quantity
- `reader.MainPage()` - accesses primary page
- `zimFS.New(reader)` - creates filesystem wrapper

## Technical Details
- **Language**: 100% Go (pure Go implementation)
- **CGo requirement**: No C library dependencies mentioned; appears to be pure Go
- **Search capability**: No full-text search functionality documented
- **Stars**: 3
- **Forks**: 1
- **License**: MIT
- **Commits**: 5 total on master branch
