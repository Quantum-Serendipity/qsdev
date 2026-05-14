---
source: https://github.com/openzim/sotoki
retrieved: 2026-05-14
---

# Sotoki - StackExchange to ZIM Scraper

Sotoki (Stack Overflow to Kiwix) is an openZIM scraper to create offline versions of Stack Exchange websites. It processes Stack Exchange data dumps hosted by The Internet Archive.

## Usage
- --mirror: URL to the data dump source (e.g., https://archive.org/download/stackexchange_20240829)
- --domain: Specific Stack Exchange site domain (e.g., sports.stackexchange.com)
- --title: ZIM file title (max 30 characters)
- --description: ZIM file description (max 80 characters)

## Key Points
- Processes raw XML data dump files
- Outputs ZIM format for offline reading (Kiwix)
- Does NOT appear to support tag-based filtering from README alone
- Would need source code examination to understand internal pipeline
- Source code in /src/sotoki directory
