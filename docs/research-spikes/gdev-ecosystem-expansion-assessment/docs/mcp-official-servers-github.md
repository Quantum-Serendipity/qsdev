<!-- Source: https://github.com/modelcontextprotocol/servers -->
<!-- Retrieved: 2026-05-14 -->

# MCP Official Servers Repository (GitHub)

## Reference Servers (Maintained by MCP Steering Group)

| Name | Description | Category |
|------|-------------|----------|
| Everything | Reference / test server with prompts, resources, and tools | Testing/Reference |
| Fetch | Web content fetching and conversion for efficient LLM usage | Web/Utilities |
| Filesystem | Secure file operations with configurable access controls | File Management |
| Git | Tools to read, search, and manipulate Git repositories | Version Control |
| Memory | Knowledge graph-based persistent memory system | Data/Knowledge |
| Sequential Thinking | Dynamic and reflective problem-solving through thought sequences | AI/Reasoning |
| Time | Time and timezone conversion capabilities | Utilities |

## Archived Servers (Previously Reference, Now Community-Maintained)

| Name | Description | Category |
|------|-------------|----------|
| AWS KB Retrieval | Retrieval from AWS Knowledge Base using Bedrock Agent Runtime | Cloud/AWS |
| Brave Search | Web and local search using Brave's Search API | Search |
| EverArt | AI image generation using various models | Multimedia |
| GitHub | Repository management, file operations, and GitHub API integration | Version Control |
| GitLab | GitLab API enabling project management | Version Control |
| Google Drive | File access and search capabilities for Google Drive | Cloud/Productivity |
| Google Maps | Location services, directions, and place details | Location/Maps |
| PostgreSQL | Read-only database access with schema inspection | Database |
| Puppeteer | Browser automation and web scraping | Web Automation |
| Redis | Interact with Redis key-value stores | Database |
| Sentry | Retrieving and analyzing issues from Sentry.io | Observability |
| Slack | Channel management and messaging capabilities | Communication |
| SQLite | Database interaction and business intelligence capabilities | Database |

## Key Observations

The repository emphasizes these are reference implementations, not production-ready solutions. Most servers are implemented in TypeScript or Python using official MCP SDKs across multiple languages (C#, Go, Java, Kotlin, PHP, Python, Ruby, Rust, Swift).

The database servers mentioned (PostgreSQL, Redis, SQLite) are archived, while cloud providers (AWS) and communication tools (Slack) have limited official coverage but are addressed through the community ecosystem.
