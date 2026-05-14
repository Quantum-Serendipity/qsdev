# Understanding and Mitigating Unicode Tag Prompt Injection

- **Source**: https://blogs.cisco.com/ai/understanding-and-mitigating-unicode-tag-prompt-injection
- **Retrieved**: 2026-05-14

## Unicode Tag Ranges

The attack exploits Unicode characters reserved for flag emojis:
- **UTF-8 Begin:** `\xf3\xa0\x80\x80`
- **UTF-8 End:** `\xf3\xa0\x81\xbf`
- **CodePoint Begin:** U+E0000
- **CodePoint End:** U+E007F

## How It Works

The technique relies on tokenizer behavior. According to the article, "the invisible text payloads are a sequence of tag + char. When an LLM receives a prompt obfuscated with this technique, the tokenizer splits the text back into the tag characters and original characters, and the LLM essentially re-builds the payload for you as it only regards the meaningful characters."

Each ASCII character becomes invisible by prefixing it with a tag character. For example, "Hello" converts to: tag + H + tag + e + tag + l + tag + l + tag + o.

## Vulnerable Models

Confirmed vulnerable systems include:
- ChatGPT
- Twitter's Grok
- Other LLMs (not specifically enumerated)

## Mitigation Techniques

**Python-based filtering:**
```python
def remove(input_string):
    output_string = ''.join(ch for ch in input_string 
                           if not (0xE0000 <= ord(ch) <= 0xE007F))
    return output_string
```

**YARA rule detection:**
```
rule UnicodeTags { 
    strings:
      $pattern1 = { F3 A0 [0-2] ?? }
    condition:
      #pattern1 > 10 
}
```

The YARA condition flags 10+ tag occurrences, accounting for realistic payload lengths.

## Attack Vectors

- Indirect prompt injection from connected systems
- Malicious prompts in shared repositories
- Training data poisoning
- Human-in-the-loop exploitation where users unknowingly copy hidden instructions
