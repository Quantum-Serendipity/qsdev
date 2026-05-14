# The Invisible Threat: How Zero-Width Unicode Characters Can Silently Backdoor Your AI-Generated Code

- **Source**: https://www.promptfoo.dev/blog/invisible-unicode-threats/
- **Retrieved**: 2026-05-14

## Character Types Used

The article identifies specific Unicode characters for binary encoding:

1. **Zero Width Space (U+200B)** - marks message start
2. **Zero Width Non-Joiner (U+200C)** - represents binary '0' bits
3. **Invisible Separator (U+2063)** - represents binary '1' bits
4. **Zero Width Joiner (U+200D)** - marks message end

## Encoding Mechanism

The threat works through a multi-step process:
- Each character converts to ASCII code
- ASCII transforms into 8-bit binary
- Binary digits map to invisible Unicode characters
- The sequence remains valid Unicode, bypassing standard validation

As the article explains: "The encoding is essentially a binary code hidden in plain sight, using invisible characters that are still part of the text's Unicode sequence."

## Why LLMs Are Vulnerable

"LLMs process text at the Unicode character level. While these characters are invisible to humans, LLMs see them as distinct, valid Unicode characters in the input stream."

## Detection & Prevention Methods

**Scanning approaches:**
- Display hidden Unicode character sequences
- Review raw file contents rather than rendered text
- Focus on configuration files (.mdc, .md, .txt)

**Sanitization strategies:**
- Implement strict Unicode character filtering
- Maintain whitelisted acceptable characters
- Validate all text input before processing
