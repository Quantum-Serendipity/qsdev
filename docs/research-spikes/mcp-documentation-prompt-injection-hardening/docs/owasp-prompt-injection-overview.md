# OWASP Prompt Injection Overview
- **Source**: https://owasp.org/www-community/attacks/PromptInjection
- **Retrieved**: 2026-05-14
- **Note**: AI-summarized content; may not contain full page text

## Core Definition
According to OWASP, "Prompt Injection is a novel security vulnerability that targets Large Language Models" by manipulating behavior through malicious inputs that bypass safety mechanisms.

## Attack Classification

**Delivery Vectors:**
- Direct injection: Attackers append override commands directly into prompts
- Indirect injection: Malicious instructions embedded in content the LLM processes later, sometimes using concealment techniques like invisible text

**Injection Types:**
- Multimodal attacks exploiting image/audio/video files
- Code injection hiding dangerous instructions in programming requests
- Context hijacking manipulating session memory to override safeguards

## Notable Incidents

The Bing Chat "Sydney" incident demonstrated how a Stanford student bypassed Microsoft's safeguards by instructing the system to "ignore prior directives," revealing internal guidelines. The Chevrolet chatbot exploitation showed how prompt injection tricked the AI into recommending competitors.

## Core Vulnerability

The fundamental issue is the "semantic gap" -- both system prompts and user inputs share the same natural-language text format, making them difficult to distinguish.

## Mitigation Strategies

- Input sanitization and length restrictions
- Separate user input from system instructions using templates
- Implement post-generation content filters
- Sanitize training data and embed security policies into model behavior
- Keep prompts confidential and limit response scope

## Testing Approach

Security teams should use known payloads like "Ignore previous instructions" and conduct adversarial testing during development.
