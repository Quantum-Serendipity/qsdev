<!-- Source: https://simonwillison.net/2023/Apr/25/dual-llm-pattern/ -->
<!-- Retrieved: 2026-05-14 -->

# The Dual LLM Pattern for Building AI Assistants That Can Resist Prompt Injection — Simon Willison (2023)

## Core Problem

Simon Willison identifies prompt injection as a critical vulnerability in AI assistants. The danger lies in systems that blend untrusted data (emails, web content) with trusted decision-making capabilities. An attacker can embed malicious instructions in external content that tricks the LLM into executing harmful actions like deleting emails or exfiltrating data.

## Architectural Solution

Willison proposes a two-model system:

**Privileged LLM**: The primary assistant that accepts only trusted user input. It has full tool access -- sending emails, modifying calendars, executing state-changing operations. This model never encounters potentially compromised content.

**Quarantined LLM**: Processes untrusted external content exclusively. It has no tool access and is treated as "potentially radioactive" at all times. Critically, its output never flows directly back to the Privileged LLM.

**Controller**: Standard software (not an LLM) mediates between the two models. It executes actions, manages variable tokens, and ensures tainted content stays isolated.

## How It Works in Practice

When a user requests an email summary:
- The Privileged LLM initiates the request
- The Controller fetches the email and assigns it a variable token ($VAR1)
- The Controller passes this to the Quarantined LLM for summarization
- Results return as a separate token ($VAR2)
- The Privileged LLM only receives variable references, never raw untrusted text
- The Controller displays final output to the user

## Key Vulnerabilities Remain

Willison acknowledges significant gaps in this approach. Social engineering attacks can still manipulate users into copying-pasting obfuscated data. He notes that "chaining" LLM outputs creates additional injection vectors, requiring zealous protection of the interface between models.

## Honest Assessment

The author concludes this solution is "pretty bad" -- it adds substantial implementation complexity and degrades user experience. However, he frames it as the safest available option until better defenses emerge.
