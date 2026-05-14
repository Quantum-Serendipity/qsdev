<!-- Source: https://github.com/freeCodeCamp/devdocs/issues/133 -->
<!-- Retrieved: 2026-05-14 -->

# DevDocs API / Programmatic Access (GitHub Issue #133)

## Current State (as of issue, 2014)
The issue identifies a limitation: "It's not possible to query with CURL-like libs using the OpenSearch URL as JavaScript is not supported."

## Requested Feature
Implementing programmatic access through alternative formats — "an output raw format like pure HTML, XML or JSON" potentially "implemented thru a JSON API."

## Key Findings
- **No formal REST API** exists for DevDocs
- OpenSearch URL exists but requires JavaScript execution
- Incompatible with standard HTTP clients (curl, libraries)
- Issue represents a long-standing feature request (2014)
- Never implemented as a formal API

## Implication for MCP Integration
Since DevDocs has no REST API, MCP servers must either:
1. Access the static JSON/HTML files directly from the filesystem
2. Run a DevDocs instance and scrape/parse its web interface
3. Use the generated index.json and db.json files directly
