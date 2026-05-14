# The Invisible Threat: How Zero-Width Unicode Characters Can Silently Backdoor Your AI-Generated Code
- **Source**: https://www.promptfoo.dev/blog/invisible-unicode-threats/
- **Retrieved**: 2026-05-14
- **Note**: AI-summarized content

## Encoding Mechanism

Binary encoding system using invisible Unicode characters:

**Start/End Markers:**
- Zero Width Space (U+200B) initiates the message
- Zero Width Joiner (U+200D) terminates it

**Binary Representation:**
Characters convert through ASCII codes to 8-bit binary, where:
- Zero Width Non-Joiner (U+200C) represents '0' bits
- Invisible Separator (U+2063) represents '1' bits

"LLMs process text at the Unicode character level. While these characters are invisible to humans, LLMs see them as distinct, valid Unicode characters."

## Attack Vector in Development Tools

Demonstration shows how malicious instructions can be embedded in configuration files (like `.mdc` files for Cursor) that appear benign to human reviewers but contain hidden directives to AI coding assistants. These could instruct systems to inject backdoors, leak credentials, or bypass security protocols -- all imperceptible to code reviewers.

## Defense Strategies

1. **Input Validation** -- Filter Unicode characters through whitelisting and sanitization
2. **File Review** -- Use tools displaying hidden characters and examine raw file contents
3. **Detection Tools** -- Scan `.txt`, `.md`, and `.mdc` files for suspicious invisible characters

The core vulnerability exploits the disconnect between human-visible text and machine-readable Unicode sequences.
