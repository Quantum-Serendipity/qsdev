<!-- Source: https://github.com/withLinda/claude-JSONL-browser -->
<!-- Retrieved: 2026-03-26 -->

# claude-JSONL-browser

Web-based tool that transforms Claude Code CLI conversation logs from JSONL format into human-readable Markdown. Includes built-in file explorer for managing multiple logs.

**Live Demo:** jsonl.withlinda.dev
**Created by:** Linda

## Key Features

- **Multi-file Management:** Process several conversation logs at once
- **Smart Parsing:** Extracts session metadata, timestamps, and conversation flow
- **Search Functionality:** Find content across all loaded conversations
- **Export Options:** Download individual or combined Markdown files
- **Tool Use Formatting:** Displays when Claude uses tools and their outputs
- **Model Change Tracking:** Highlights switches between Claude models

## Technical Stack

Next.js 15 with TypeScript, Tailwind CSS (Everforest theme). Client-side processing (no server data transmission).

## Installation

```bash
git clone https://github.com/withLinda/claude-JSONL-browser.git
cd claude-JSONL-browser
npm install
npm run dev
```
