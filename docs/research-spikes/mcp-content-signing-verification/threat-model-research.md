# Threat Model Research: MCP Documentation Content Tampering

## Overview

This report synthesizes the threat model for content tampering in gdev's MCP documentation pipeline. The analysis covers eight distinct attack vectors, maps them against existing and proposed mitigations, and prioritizes risk in the context of a local-first, Nix-managed tool serving documentation to an AI coding assistant.

## Highest-Priority Attack Vectors

### 1. Compromised Source Documentation (Risk: MEDIUM-HIGH)

The most likely and hardest-to-mitigate vector. An attacker edits upstream documentation (MDN wiki pages, library docs on GitHub, community-contributed content) and the malicious content gets scraped into ZIM files or DevDocs bundles through normal channels. Every integrity check passes because the content was legitimately present in the source. This has real-world precedent — package README poisoning and wiki vandalism are established attack patterns. The attack surface is large: MDN alone accepts community contributions, and many library documentation repos have low review thresholds for docs-only PRs.

### 2. Prompt Injection via Documentation Content (Risk: MEDIUM-HIGH)

Unique to the AI-consumer context. Documentation content — code examples, API descriptions, configuration snippets — could contain text patterns that function as prompt injection against Claude Code. This vector can be combined with source documentation poisoning (vector 1) or could arise from content tampered at any point in the pipeline. Claude Code treats MCP results as tool output rather than user instructions, providing baseline defense, but the effectiveness against sophisticated payloads embedded in plausible-looking documentation is not fully characterized. The impact ceiling is critical: successful injection could lead to arbitrary command execution or data exfiltration.

### 3. Nix Package Supply Chain Compromise (Risk: MEDIUM)

An attacker modifies the gdev Nix expression to point to malicious content and updates the SRI hash to match. This is undetectable by Nix itself (the hash is correct for the new content) and routine hash changes during version bumps provide camouflage. This is the vector where content signing provides the strongest value — it adds an independent verification layer that a Nix-package-only attacker cannot satisfy.

### 4. Compromised Upstream Build Infrastructure (Risk: MEDIUM)

If the Kiwix build farm or DevDocs scraper infrastructure is compromised, malicious content is produced with valid provenance. Low likelihood (these are standard open-source projects, not high-value targets), but catastrophic impact because no downstream check can detect it. Sigstore transparency logs would provide forensic evidence but not prevention.

### 5. MITM on First Download (Risk: LOW-MEDIUM)

HTTPS and Nix hash pinning substantially mitigate this, but the first download — the one that establishes the pinned hash — relies solely on HTTPS. Content signing would close this gap, but given that HTTPS is already in place, the incremental risk reduction is modest.

## What Signing Solves vs What It Does Not

### Signing addresses distribution integrity

Content signing provides strong protection against three vectors: MITM attacks on download (including the first-download gap that Nix hashes miss), tampered content on unofficial mirrors or redistribution channels, and Nix package supply chain compromise (because the attacker cannot forge the upstream signature). These are real risks, and signing is the correct and well-understood mitigation.

### Signing does NOT address content-level threats

The two highest-priority vectors — compromised source documentation and prompt injection — are completely unaffected by signing. Signing verifies *who produced* the content, not *whether the content is safe*. A perfectly signed ZIM file can contain malicious edits that were present in the upstream source. A perfectly signed DevDocs bundle can contain code examples that function as prompt injection. This is a fundamental limitation: signing is a provenance mechanism, not a content safety mechanism.

### Signing partially addresses upstream compromise

If the upstream build infrastructure is compromised, the attacker signs with the legitimate key. Sigstore's transparency log provides an immutable record (useful for forensic analysis and detecting key misuse patterns), but does not prevent the attack. Reproducible builds and multi-party verification would be needed, neither of which is within gdev's control.

## Recommended Defense-in-Depth Layers

Signing alone is insufficient. A robust defense requires layered controls:

1. **Content signing verification at download time** — Verify Sigstore/cosign or GPG signatures on ZIM and DevDocs artifacts. Primary defense against distribution integrity vectors. Requires upstream cooperation (Kiwix and DevDocs would need to sign releases).

2. **Content diffing at update time** — Compare new content versions against previous versions, flagging unexpected large changes or suspicious patterns. This is the only mitigation that partially addresses compromised source documentation. Implementation: per-page content hashing with delta thresholds.

3. **MCP response provenance metadata** — Include content source, verification status, and content date in MCP responses. Enables differential trust decisions (verified MDN vs unsigned community content). Does not prevent attacks but improves the AI consumer's ability to weight sources.

4. **Prompt injection hardening** — Sanitize documentation content before serving (strip unusual Unicode, control characters). Structure MCP responses to clearly delineate content from metadata. Stay current with Claude Code's evolving defenses. This is an arms race with no complete solution.

5. **Nix package integrity practices** — Commit signing on the gdev repo, CODEOWNERS requiring multi-reviewer approval for content hash changes, dedicated PRs for hash updates. Defense against supply chain compromise of the packaging layer.

6. **Runtime integrity verification** — Verify content hashes at MCP server startup. Serve from immutable /nix/store paths. Defense against post-download local tampering (low-priority vector but trivial to implement on NixOS).

## Assessment: Critical Risk or Hardening Measure?

The signing gap is a **meaningful hardening measure, not a critical risk**. Here is the reasoning:

The vectors that signing addresses (MITM, mirror tampering, Nix package compromise) are already partially mitigated by HTTPS and Nix hash pinning. Adding signing closes real gaps — particularly the first-download provenance gap and the Nix package compromise vector — but these are not the highest-likelihood attack paths.

The highest-priority risks (compromised source documentation, prompt injection) are not addressed by signing at all. These require content-level mitigations that don't exist in mature form today.

The attack context also matters: gdev is a local developer tool, not a public-facing service. The content sources (MDN, official library docs) are well-maintained. The consumer (Claude Code) has built-in prompt injection defenses. The distribution channel (HTTPS + Nix) is already reasonably secure.

Signing should be pursued as part of a defense-in-depth strategy, prioritized below content diffing and prompt injection hardening in terms of risk reduction per effort. The strongest argument for signing is not the immediate threat level but the principle of defense-in-depth: it closes a class of attack that is currently possible even if not yet observed, and it provides a foundation for provenance-aware trust decisions in MCP responses.

If upstream projects (Kiwix, DevDocs) adopt signing — particularly via Sigstore, which has low adoption friction — gdev should verify signatures. If they do not, gdev could sign content itself after first-download verification, providing "gdev attests this matched the upstream at download time" rather than "the upstream produced this." This is weaker but still valuable as a tamper-detection mechanism for redistribution and subsequent downloads.
