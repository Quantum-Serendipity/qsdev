# Oasis Security: Claude.ai Prompt Injection Data Exfiltration Vulnerability

- **Source URL**: https://www.oasis.security/blog/claude-ai-prompt-injection-data-exfiltration-vulnerability
- **Retrieved**: 2026-05-14

## Attack Mechanism ("Claudy Day")

The vulnerability chains three separate issues:

### Invisible Prompt Injection
Claude.ai's URL parameter feature (`claude.ai/new?q=...`) allows pre-filling prompts. Researchers embedded "certain HTML tags" that remained invisible in the text box but were "fully processed by Claude when the user hit Enter," enabling hidden command injection.

### Data Exfiltration Method
Despite sandbox restrictions on outbound network access, Claude can connect to `api.anthropic.com`. Attackers embedded their API key in hidden prompts to "instruct Claude to search the user's conversation history for sensitive information, write it to a file, and upload it to the attacker's Anthropic account via the Files API."

### Delivery Mechanism
An open redirect vulnerability on claude.com combined with Google Ads allowed attackers to create "search ads displaying a trusted claude.com URL that, when clicked, silently redirected the victim to the injection URL."

## Accessible Data

In default sessions: conversation history and memory, potentially including business strategies, financial information, health concerns.

With integrations enabled: blast radius expands to "read files, send messages, and interact with any connected service."

## Anthropic's Response

The prompt injection vulnerability has been fixed. Remaining issues are "currently being addressed." Oasis Security worked through Anthropic's responsible disclosure program.
