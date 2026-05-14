<!-- Source: https://docs.mantra.gonewx.com/about/faq -->
<!-- Retrieved: 2026-03-26 -->

# Mantra FAQ — Documentation

## Basic Questions

**What is Mantra?**
Mantra is a local-first AI programming companion tool built around three core pillars: "Replay, Control, Secure." Helps developers review AI programming steps, manage MCP services and Skills, and detect sensitive information in AI-generated content. Currently supports Claude Code, Cursor, and Gemini CLI.

**Which platforms does Mantra support?**
macOS (Intel & Apple Silicon), Windows 10/11, and Linux (Ubuntu 20.04+, Fedora 36+).

**Is Mantra free?**
All local features permanently free without limits — unlimited sessions, projects, MCP Hub, Git time travel, full-text search, local redaction engine. Optional paid: Sync ($4/month), Publish ($8/month per site).

## Data & Privacy

**Where is my data stored?**
All data stored on local device. Paths:
- macOS: `~/Library/Application Support/com.gonewx.mantra/`
- Windows: `%APPDATA%\com.gonewx.mantra\`
- Linux: `~/.config/com.gonewx.mantra/` and `~/.local/share/com.gonewx.mantra/`

**Will Mantra upload my data?**
No code or session content uploaded. Anonymous usage statistics enabled by default but can be disabled.

**Will Mantra upload my conversations or code?**
No. Optional Sync and Publish use end-to-end encryption.

**Where is Sync and Publish data stored?**
Encrypted data stored with chosen provider; company cannot decrypt.

**How do I delete my data?**
Via in-app "clear data" function, direct directory deletion, or during uninstall.

## Features

**What is the "time travel" feature?**
Drag the timeline to replay every moment of an AI programming session while viewing the code state at that time. Helps understand context, review AI reasoning, and learn patterns.

**What sensitive information types are detected?**
API keys (OpenAI, Anthropic, AWS, etc.), database passwords, access tokens (GitHub, GitLab, etc.), private key files, sensitive environment variables.

**How does full-text search work?**
Local indexing supporting cross-project and cross-session searches with real-time preview and highlighted matches.

## Technical Support

**How do I report bugs?**
GitHub Issues (mantra-hq/mantra-releases/issues), email (mantra@gonewx.com), Discord community, or Twitter (@decker502).

**How do I request features?**
GitHub Issues, Discord community, or email.
