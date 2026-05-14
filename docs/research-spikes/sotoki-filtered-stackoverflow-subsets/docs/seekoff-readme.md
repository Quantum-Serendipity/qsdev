---
source: https://github.com/Caspia/seekoff
retrieved: 2026-05-14
type: github-readme
---

# Seekoff: Offline Stack Exchange Search Tool with Tag Filtering

## Overview

Seekoff is an application designed to provide searchable access to Stack Exchange data in offline environments without internet connectivity. The primary use case targets computer science students in settings where internet access is unavailable and certain security-related topics should be restricted.

**ARCHIVED: December 15, 2022 (read-only)**

## Core Architecture

Two distinct phases:

1. **Indexing Phase**: Processes raw Stack Exchange XML files and populates an Elasticsearch database. Users can access this through either an Electron desktop application or command-line scripts.

2. **Search Phase**: A Node.js server with web interface provides search and display capabilities, packaged as Docker containers (Elasticsearch + web server).

## Key Features

- **Tag-based Filtering**: Supports inclusion and exclusion of posts based on tags, allowing administrators to restrict sensitive topics
- **Offline-first Design**: No internet required after initial indexing
- **Multiple Interface Options**: Electron app for indexing, web interface for searching
- **Docker Deployment**: Production deployment uses containerized components

## Data Processing

Processes Stack Exchange data dumps from archive.org, requiring:
- Posts.xml
- Users.xml
- Votes.xml
- PostLinks.json

Note: "stackoverflow files are huge, and an indexing run typically takes hours or days."

## Deployment Architecture

**Production**: Two-tier deployment model:
- Online indexing server (processes raw XML files)
- Offline webapp server (serves indexed data to users)

Both use Linux Docker installations.

## System Requirements

- Linux-based Docker for production
- Elasticsearch (requires vm.max_map_count=262144)
- Node.js v8+
- Docker containers run as user 1000
- Storage: /srv/elasticsearch and /srv/sedata

## Limitations

- Repository archived (no longer maintained)
- Configuration documentation incomplete
- Large dataset indexing is time-intensive
- Does NOT produce ZIM files -- it's a completely separate reader/server

## 93.1% JavaScript, 101 commits
