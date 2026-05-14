---
source: https://github.com/Caspia/seekoff
retrieved: 2026-05-14
---

# Seekoff - Offline Stack Overflow dump reader with tag filtering

## Core Functionality
Seekoff operates in two phases: an indexing phase that processes Stack Exchange XML dumps into an Elasticsearch database, and a search/display phase that provides web-based access to indexed content.

## Tag Filtering Mechanism
The documentation mentions "inclusion and exclusion of posts by tags" in the project description, but specific filtering implementation details are not documented in the provided content. The README focuses on deployment and setup rather than the technical filtering logic.

## Architecture Overview
- **Indexing**: Raw XML files (Posts, Users, Votes, PostLinks) are processed into Elasticsearch indices via either an Electron GUI or command-line interface
- **Storage**: An Elasticsearch database stores the indexed data across four indices
- **Display**: A Node.js server with web interface queries this database

## Regarding Cascading Data
The documentation does not address whether the system handles related data (answers, comments, users) when filtering by tags. It specifies required XML files but doesn't explain cascade handling logic.

## Output Type
Seekoff produces a filtered Elasticsearch database deployed via Docker containers, rather than exporting filtered dump files. The filtered index is then made available offline through the web interface.
