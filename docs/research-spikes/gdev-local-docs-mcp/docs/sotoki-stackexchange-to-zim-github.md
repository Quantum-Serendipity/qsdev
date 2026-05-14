# Sotoki - StackExchange to ZIM Scraper
> Source: https://github.com/openzim/sotoki
> Retrieved: 2026-05-14

## What It Does
Sotoki converts Stack Exchange websites (like Stack Overflow) into offline ZIM files for use with Kiwix. It processes Stack Exchange's Data Dumps hosted by The Internet Archive.

## Usage
Requires:
- Mirror URL (e.g., `https://archive.org/download/stackexchange_20240829`)
- Domain (e.g., `sports.stackexchange.com`)
- Title (under 30 characters)
- Description (under 80 characters)

## Technical Details
- **Language**: Python 3 (79.9%)
- **Additional**: HTML (16.3%), JavaScript (1.8%), Dockerfile (1.2%), CSS (0.8%)
- **License**: GPL v3
- **Stars**: 241
- **Forks**: 31
- **Releases**: 12
- **Latest Release**: v3.0.2 (December 22, 2025)
- **Commits**: 976 total

## Installation
- Docker: `ghcr.io/openzim/sotoki`
- PyPI: `pip install sotoki`

## Pre-built ZIM Files
Available at library.kiwix.org for all Stack Exchange sites, eliminating the need to build your own.
