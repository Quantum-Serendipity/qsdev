<!-- Source: https://www.lakera.ai/blog/indirect-prompt-injection -->
<!-- Retrieved: 2026-05-14 -->

# Indirect Prompt Injection: The Hidden Threat Breaking Modern AI Systems

## How IPI Works Through Documentation Sources

Documentation becomes dangerous when AI systems ingest it without distinguishing trusted from untrusted content. "AI systems combine system prompts, user inputs, retrieved documents, tool metadata, memory entries, and code snippets in a single context window." This unified processing means malicious instructions embedded in technical docs operate identically to legitimate system guidance.

## Web-Fetched Content Attack Vectors

The Perplexity Comet incident demonstrates the real threat. Attackers embedded invisible instructions in a public Reddit post; "When Comet fetched the page, the AI summarizer read the hidden instructions, leaked the user's one time password, and sent it to an attacker-controlled server." This pattern extends beyond webpages — any content-retrieval surface becomes an ingestion channel.

## Curated Documentation as Risk Reduction

Local documentation reduces exposure by narrowing ingestion surfaces. Teams practicing "zero trust" toward external content inherently protect systems relying on internal, curated sources rather than dynamic web fetching.

## Code and Configuration Examples

Real incidents include: "a poisoned code rules file pushes a harmful dependency" and "a small case sensitivity bug in a protected file path allowed an attacker to influence Cursor's agentic behavior." Comments in repositories and configuration files become executable attack surfaces.

## Documentation-Focused Mitigations

Key strategies include: separate trusted from untrusted inputs with "clear delimiters around external content" and "labels that identify source and reliability." Additionally, treating "all external data as untrusted" forces explicit validation before documentation influences model decisions.
