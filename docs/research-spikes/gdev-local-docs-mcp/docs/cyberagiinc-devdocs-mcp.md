<!-- Source: https://github.com/cyberagiinc/DevDocs -->
<!-- Retrieved: 2026-05-14 -->

# cyberagiinc/DevDocs - UI-Based Tech Documentation MCP Server

## Core Purpose
DevDocs is a free, private, UI-based technical documentation Model Context Protocol (MCP) server designed for developers. It transforms lengthy documentation research into rapid implementation by intelligently crawling, extracting, and organizing technical content for LLM integration.

## What It Does
**Primary Function**: "Turn Weeks of Documentation Research into Hours of Productive Development" by pointing the system at documentation URLs, discovering related pages, extracting meaningful content, and presenting it in searchable, LLM-ready formats.

## Architecture & Components

**Multi-Service Docker Stack**:
- Frontend UI (port 3001)
- Backend API (port 24125)
- Crawl4AI Service (port 11235)
- MCP Server

**Technology Stack**:
- **Frontend**: TypeScript (48.1%), Next.js with Tailwind CSS
- **Backend**: Python (35.7%)
- **DevOps**: Shell (11.9%), Docker, Docker Compose
- **Additional**: JavaScript, Batchfile, PowerShell

## How It Works

**Intelligent Crawling Process**:
1. Discovers all related pages across documentation sites
2. Extracts content without extraneous elements
3. Organizes information logically
4. Exports to Markdown, JSON, or MCP-server-ready formats

**Key Capabilities**:
- Smart depth control (1-5 crawl levels)
- Automatic child URL detection up to level 5
- Parallel processing for multiple simultaneous page crawls
- Lazy loading support for modern web applications
- Rate limiting for respectful server interactions
- Smart caching to prevent duplicate processing

## Tools & Resources Exposed

The system integrates with AI code editors through MCP protocol, enabling:
- **Table of Contents Tool**: Returns filtered documentation topics
- **Section Access Tool**: Retrieves detailed content from specific sections
- Claude Desktop App integration
- Cursor, Windsurf, Cline, and Roo Code compatibility

## Installation & Deployment

**Prerequisites**: Docker and Git

**Quick Start**:
```bash
git clone https://github.com/cyberagiinc/DevDocs.git
cd DevDocs
cp .env.template .env
./docker-start.sh  # Mac/Linux
docker-start.bat   # Windows
```

## Limitations & Status

**Current Status**: "Not publicly maintained. Enhanced internal version at CyberAGI — public release coming soon."

**Known Constraints**:
- Windows support remains experimental
- Permission issues may require manual configuration
- Depends on upstream Crawl4AI service stability

## Differentiation from Other Tools

**vs. FireCrawl**:
- Free tier with unlimited pages (FireCrawl: none)
- Crawl speed: 1000/min vs. FireCrawl's 20/min
- Self-hosted option at no cost
- Native MCP server integration

**License**: Not specified
