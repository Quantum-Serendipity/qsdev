<!-- Source: https://github.com/withLinda/claude-JSONL-browser -->
<!-- Retrieved: 2026-03-26 -->

# claude-JSONL-browser

Web application that transforms Claude Code CLI conversation logs from JSONL format into readable Markdown documents. Handles multiple files simultaneously with integrated file management.

## Core Functionality

- Multi-file processing for numerous conversation logs in a single session
- Metadata extraction: session IDs, Git branches, working directories
- Content organization with timestamp preservation
- Search across loaded conversations
- Export to individual or combined Markdown files
- Special formatting for tool usage instances and model switching events

## Getting Started

**Web-Based Access:** Visit jsonl.withlinda.dev

**Local Installation:**
```
git clone https://github.com/withLinda/claude-JSONL-browser.git
cd ClaudeJSONLbrowser
npm install
npm run dev
```

## Technical Architecture

- **Framework**: Next.js 15 with TypeScript
- **Styling**: Tailwind CSS with Everforest theme
- **Processing**: Client-side only (no server data transmission)

## Supported Data Elements

- User queries and Claude responses
- System summaries and metadata
- Model change commands (`/model` syntax)
- Tool execution records with outputs
- Complete timing information
