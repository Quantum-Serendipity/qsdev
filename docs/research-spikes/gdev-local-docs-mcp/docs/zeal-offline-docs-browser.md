<!-- Source: https://github.com/zealdocs/zeal -->
<!-- Retrieved: 2026-05-14 -->

# Zeal: Offline Documentation Browser

## Project Overview
Zeal is an open-source desktop application that functions as "a simple offline documentation browser inspired by Dash." It enables developers to access API documentation without internet connectivity.

## Technology Stack
- **Primary Language:** C++ (94.6% of codebase)
- **Build System:** CMake
- **UI Framework:** Qt 6.4.2 or later, specifically Qt WebEngine Widgets
- **Dependencies:** libarchive, SQLite
- **Platform Requirements:** Linux/BSD need extra-cmake-modules; X11 platforms require libxkbcommon and xcb-util-keysyms

## Docset Format & Storage
Zeal adopts the Dash docset standard. Docsets are stored locally using SQLite databases. The Dash docset format consists of:
- A directory with `.docset` extension
- Contains `Contents/Resources/docSet.dsidx` (SQLite index)
- Contains `Contents/Resources/Documents/` (HTML files)
- An `Info.plist` metadata file

Users manage docsets through the `Tools->Docsets` menu.

## Search & Query Capabilities
- Supports command-line queries: `zeal python:pprint`
- Docset-scoped searching using colon notation (e.g., `java:BaseDAO`)
- Comma-separated queries for multi-docset search
- No programmatic API or server mode

## Platform Support
Binary builds for Windows and Linux. Source code supports Linux, BSD, and X11-based systems.

## Relationship to DevDocs
- Uses Dash docset format (different from DevDocs JSON/HTML format)
- DevDocs has its own scraper system; Zeal uses Dash-compatible docsets
- Different ecosystems but overlapping documentation coverage
- doc2dash can convert Sphinx documentation to Dash docsets

## Notable Characteristics
- 12.6k GitHub stars, 828 forks
- License: GPLv3
- Purely desktop application — no server/API mode
- No MCP integration exists
