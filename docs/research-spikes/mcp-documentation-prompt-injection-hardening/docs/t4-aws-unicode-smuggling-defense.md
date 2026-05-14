# Defending LLM Applications Against Unicode Character Smuggling

- **Source**: https://aws.amazon.com/blogs/security/defending-llm-applications-against-unicode-character-smuggling/
- **Retrieved**: 2026-05-14

## Unicode Ranges Addressed

The article specifically focuses on **Unicode tag blocks** spanning `U+E0000` to `U+E007F`. These were originally designed as invisible language markers but can be exploited for prompt injection attacks against LLMs.

The content does not discuss zero-width characters, RTL overrides, or homoglyphs—only tag blocks and their UTF-16 surrogate pair representations (`\uDB40\uXXXX` ranges in Java).

## Normalization Techniques

The article does not recommend Unicode normalization forms (NFC, NFD, NFKC, NFKD). Instead, it advocates direct character removal.

## Sanitization Code Examples

**Java Implementation (Recursive):**
The authors provide a recursive solution because single-pass sanitization can inadvertently create new tag characters from orphaned surrogate pairs:

```java
public static String removeHiddenCharacters(String input) {
    String previous;
    do {
        previous = input;
        StringBuilder result = new StringBuilder();
        previous.codePoints().forEach(cp -> {
            if ((cp < 0xE0000 || cp > 0xE007F) && 
                (!Character.isSurrogate((char)cp))) {
                result.appendCodePoint(cp);
            }
        });
        input = result.toString();
    } while (!input.equals(previous));
    return input;
}
```

**Python Implementation:**
Python's UTF-8 representation avoids surrogate pair issues, allowing single-pass filtering:

```python
def removeHiddenCharacters(input):
    return ''.join(
        ch for ch in input
        if not (0xE0000 <= ord(ch) <= 0xE007F or 0xD800 <= ord(ch) <= 0xDFFF)
    )
```

## Detection & Prevention Approaches

1. **AWS Lambda Functions**: Deploy sanitization handlers to filter user inputs before LLM processing
2. **Amazon Bedrock Guardrails**: Use denied topics to block prompts containing tag block characters
3. **Recursive Validation**: Essential for Java applications to prevent character reconstruction attacks

## Key Vulnerability

The article emphasizes that "single sanitization pass can inadvertently create new tag block characters" when processing malformed surrogate pairs, requiring iterative validation.
