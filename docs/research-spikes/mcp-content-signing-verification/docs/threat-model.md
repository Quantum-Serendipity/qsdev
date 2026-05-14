# Threat Model: Content Tampering in MCP Documentation Pipeline

- **Scope**: gdev's MCP-served documentation (ZIM files, DevDocs JSON bundles) consumed by Claude Code
- **Date**: 2026-05-14
- **Spike**: mcp-content-signing-verification
- **Task**: P1-T5

## 1. System Description

gdev provisions local developer documentation via MCP (Model Context Protocol) servers. Two content formats are served:

- **ZIM files**: Offline documentation archives (MDN, Wikipedia, Stack Overflow) produced by Kiwix/openZIM. Integrity mechanisms: MD5 checksum embedded in-file (footer), SHA-256 checksum sidecar files at download.kiwix.org. No cryptographic signatures.
- **DevDocs JSON bundles**: Scraped and normalized documentation from devdocs.io, maintained by freeCodeCamp. Integrity mechanisms: none. No checksums, no signatures.

Both content types are:
1. Downloaded (typically over HTTPS) during gdev setup or update
2. Hash-pinned via Nix SRI hashes (post-recording integrity)
3. Stored on the local filesystem
4. Served to Claude Code via a locally-running MCP server
5. Consumed by Claude Code as reference material when generating code

The consumer (Claude Code) treats documentation content as authoritative reference material. Tampered content that reaches Claude Code could influence code generation, introduce vulnerabilities, or execute prompt injection attacks.

## 2. Trust Boundaries

```
┌─────────────────────────────────────────────────────────────┐
│  Upstream Content Production                                │
│  (MDN editors, DevDocs scrapers, Kiwix builders)            │
│  Trust: HIGH but not verified cryptographically             │
└──────────────────────┬──────────────────────────────────────┘
                       │ Content creation
                       ▼
┌─────────────────────────────────────────────────────────────┐
│  Distribution Infrastructure                                │
│  (download.kiwix.org, devdocs.io CDN, mirrors)              │
│  Trust: MEDIUM — HTTPS provides transport security only     │
└──────────────────────┬──────────────────────────────────────┘
                       │ HTTPS download
                       ▼
┌─────────────────────────────────────────────────────────────┐
│  Nix Packaging / gdev                                       │
│  (fetchurl with SRI hash, local Nix store)                  │
│  Trust: HIGH — hash pinning locks content post-recording    │
└──────────────────────┬──────────────────────────────────────┘
                       │ File read from Nix store
                       ▼
┌─────────────────────────────────────────────────────────────┐
│  Local MCP Server                                           │
│  (kiwix-serve, custom DevDocs server)                       │
│  Trust: HIGH — runs locally, no network exposure            │
└──────────────────────┬──────────────────────────────────────┘
                       │ MCP protocol (local IPC)
                       ▼
┌─────────────────────────────────────────────────────────────┐
│  Claude Code (AI Consumer)                                  │
│  Trust: N/A — this is the asset we're protecting            │
└─────────────────────────────────────────────────────────────┘
```

Key trust boundary crossings:
- **TB1**: Content creation to distribution (upstream to CDN)
- **TB2**: Distribution to local machine (download)
- **TB3**: Nix store to MCP server (local filesystem read)
- **TB4**: MCP server to Claude Code (MCP protocol)

## 3. Attack Vectors

### AV-1: Compromised Upstream Build Infrastructure

**Description**: An attacker compromises the Kiwix build farm or DevDocs scraper infrastructure. Tampered content is produced at the source, meaning all integrity mechanisms downstream (checksums, hashes) faithfully verify the malicious content.

**Trust boundary**: TB1 (content production)

**Prerequisites**: Access to Kiwix CI/CD, openZIM build systems, or freeCodeCamp's DevDocs infrastructure (GitHub Actions, scraper configs).

**Likelihood**: LOW — Kiwix is a well-known open-source project with standard GitHub security. DevDocs is simpler but maintained by freeCodeCamp. Neither is a high-value target compared to package registries. However, CI compromise is increasingly common in supply chain attacks.

**Detection difficulty**: VERY HIGH — all downstream integrity checks pass because the malicious content was produced by the legitimate pipeline. Only manual content review or behavioral anomaly detection would catch this.

**Signing effectiveness**: PARTIAL — signing by the build infrastructure would not help if the infrastructure itself is compromised (the attacker signs with the legitimate key). Reproducible builds plus multi-party verification would be needed. Sigstore's transparency log provides some forensic value (immutable record of what was signed when).

### AV-2: Compromised Source Documentation

**Description**: A malicious edit to upstream documentation (e.g., MDN, a library's official docs) gets scraped into a ZIM file or DevDocs bundle. The content is "legitimately" wrong because the source was poisoned.

**Trust boundary**: Pre-TB1 (before content even enters the pipeline)

**Prerequisites**: Ability to make edits to MDN (wiki-style contributions), library documentation repos, or other scraped sources. Many documentation sources accept community contributions.

**Likelihood**: MEDIUM — MDN and major documentation sites have editorial review, but volume of changes makes comprehensive review difficult. Less popular library documentation may have minimal review. This attack has been demonstrated in practice against package READMEs and wiki documentation.

**Detection difficulty**: HIGH — content passes all integrity and provenance checks because it was legitimately present in the source. Only content-level analysis (semantic review, comparison to known-good versions) could detect it.

**Signing effectiveness**: NONE — signing verifies that the content came from the expected producer. If the producer faithfully scraped poisoned source material, the signature is valid over malicious content.

### AV-3: MITM on Download

**Description**: An attacker intercepts the download of ZIM files or DevDocs bundles and substitutes tampered content.

**Trust boundary**: TB2 (distribution to local machine)

**Subtypes**:
- **AV-3a: HTTP downgrade** — If any download URL falls back to HTTP (e.g., mirror misconfiguration, redirect chain), content is trivially modifiable in transit.
- **AV-3b: DNS hijack** — Attacker controls DNS resolution, pointing download.kiwix.org or devdocs.io to an attacker-controlled server serving tampered files.
- **AV-3c: CDN compromise** — Attacker compromises the CDN or hosting infrastructure (e.g., gaining write access to the S3 bucket or CDN edge node).
- **AV-3d: TLS certificate compromise** — Attacker obtains a valid certificate for the download domain (CA compromise, domain validation exploit).

**Likelihood**: LOW for targeted attacks on these specific domains. MEDIUM if the user is on an untrusted network (public WiFi, corporate proxy with TLS inspection).

**Detection difficulty**: LOW-MEDIUM — Nix hash pinning catches this for subsequent downloads (hash mismatch). But the first download that establishes the pinned hash is vulnerable. SHA-256 sidecar verification at download time also helps for ZIM files.

**Signing effectiveness**: HIGH — cryptographic signatures from the content producer would detect all MITM variants. The signature is created before content enters the distribution channel, so any in-transit modification invalidates it.

### AV-4: Tampered Content on Unofficial Mirrors / Redistribution

**Description**: ZIM files are distributed on unofficial mirrors, torrent networks, or shared directly. DevDocs bundles could be redistributed via npm packages, GitHub releases, or file sharing. Tampered versions could be served from these channels.

**Trust boundary**: TB2 (alternative distribution path)

**Prerequisites**: Attacker publishes or modifies content on an unofficial channel. User (or gdev configuration) points to that channel.

**Likelihood**: LOW for gdev specifically (Nix fetches from pinned URLs, not arbitrary mirrors). MEDIUM for general Kiwix usage where users might download from mirrors.

**Detection difficulty**: MEDIUM — Nix hash pinning would catch tampering if hashes are pinned to known-good values. Without hash pinning, only signature verification or manual checksum comparison would detect it.

**Signing effectiveness**: HIGH — signatures from the original producer would be verifiable regardless of the distribution channel. This is the canonical use case for content signing.

### AV-5: Supply Chain Compromise of the Nix Package

**Description**: An attacker compromises the Nix expression that defines how ZIM files or DevDocs bundles are fetched. They modify the URL to point to a malicious source and update the SRI hash to match the tampered content.

**Trust boundary**: TB2 (Nix packaging layer)

**Subtypes**:
- **AV-5a: Compromised gdev repository** — Attacker gains commit access and modifies fetch URLs and hashes.
- **AV-5b: Malicious PR merged** — A pull request modifying content URLs/hashes is merged without adequate review. Hash changes in Nix files are routine (version bumps), making malicious changes hard to distinguish.
- **AV-5c: Dependency confusion** — If content fetching uses any overlay or flake input, an attacker could substitute a malicious input.

**Likelihood**: LOW-MEDIUM — this is the standard software supply chain attack pattern. gdev is a smaller project (fewer eyeballs on PRs), but the attack requires gaining merge access.

**Detection difficulty**: MEDIUM — code review could catch URL changes, but hash changes look identical whether legitimate or malicious. Git commit signing helps attribute changes.

**Signing effectiveness**: HIGH — if gdev verifies content signatures independent of the Nix hash, then modifying the URL and hash isn't sufficient. The attacker would also need to produce content signed by the legitimate producer's key.

### AV-6: Local Filesystem Tampering After Download

**Description**: An attacker with access to the local filesystem modifies documentation content after it has been downloaded and verified.

**Trust boundary**: TB3 (Nix store to MCP server)

**Subtypes**:
- **AV-6a: Nix store modification** — Content in /nix/store is read-only by default (mounted read-only on NixOS), but root can modify it. A compromised system process or malware with root access could tamper with stored files.
- **AV-6b: MCP server data directory** — If the MCP server reads content from a mutable location (outside /nix/store), modification is straightforward for any process with write access.
- **AV-6c: Symlink/mount attacks** — Replacing the content path with a symlink or bind mount pointing to attacker-controlled content.

**Likelihood**: LOW — requires local root or write access to the data directory. If the attacker has this level of access, documentation tampering is likely not their primary objective (they could directly modify code, inject backdoors, etc.).

**Detection difficulty**: LOW (for Nix store, which is immutable) to HIGH (for mutable data directories, where no integrity checks occur post-download).

**Signing effectiveness**: LOW — signing verifies authenticity at download time but does not protect against post-download modification. Runtime integrity checking (re-verifying hashes on read) would be needed.

### AV-7: MCP Server Compromise / Malicious MCP Server

**Description**: The MCP server itself is compromised or replaced with a malicious version that serves tampered content regardless of what's on disk.

**Trust boundary**: TB4 (MCP server to Claude Code)

**Prerequisites**: Ability to modify the MCP server binary/script or its configuration.

**Likelihood**: VERY LOW — the MCP server is locally installed via Nix. Compromising it requires the same level of access as AV-6.

**Detection difficulty**: HIGH — Claude Code receives content via MCP protocol and has no mechanism to verify that the MCP server is unmodified.

**Signing effectiveness**: NONE — signing content at the distribution layer doesn't protect against a compromised serving layer. This requires code signing / binary integrity verification of the MCP server itself, which is orthogonal to content signing.

### AV-8: Prompt Injection via Legitimate Documentation Content

**Description**: Documentation content — whether tampered or naturally occurring — contains text patterns that function as prompt injection against Claude Code. This could be:
- Hidden instructions in code comments or examples ("ignore previous instructions and...")
- Crafted function signatures that cause Claude Code to generate vulnerable code
- Unicode tricks or formatting that is interpreted differently by the AI than by a human reader
- Documentation that appears normal to humans but contains steganographic instructions for the model

**Trust boundary**: TB4 (content interpretation by AI)

**Prerequisites**: Ability to get crafted content into the documentation (via AV-1, AV-2, or even through legitimate documentation contributions). The content must survive the scraping/normalization pipeline.

**Likelihood**: MEDIUM — prompt injection is an active area of research and attack. Documentation content (especially code examples) is a plausible injection vector. The normalization pipeline (HTML-to-markdown for ZIM, JSON structure for DevDocs) may strip some attack patterns but could preserve others.

**Detection difficulty**: HIGH — prompt injection payloads can be subtle and context-dependent. No reliable automated detection exists today.

**Signing effectiveness**: NONE — signing verifies the producer, not the safety of the content. Legitimately signed documentation can contain prompt injection payloads (whether planted via AV-2 or naturally occurring).

## 4. Impact Analysis

### I-1: Prompt Injection Execution

**Severity**: CRITICAL

If tampered content contains prompt injection payloads that Claude Code acts on, the attacker could:
- Override Claude Code's system instructions
- Cause Claude Code to execute arbitrary commands (if tool use is enabled)
- Exfiltrate context (other code in the project, environment variables) via crafted tool calls
- Generate code that phones home or contains backdoors

Claude Code does have prompt injection defenses (content is treated as tool output, not user instructions). The effectiveness of these defenses against sophisticated payloads embedded in documentation content is an open question, but Claude models are generally robust against obvious injection in tool results.

### I-2: Vulnerable Code Generation

**Severity**: HIGH

Tampered documentation that presents insecure patterns as correct (e.g., SQL query examples without parameterization, crypto examples using deprecated algorithms, auth examples with token validation disabled) would cause Claude Code to generate vulnerable code. This is particularly dangerous because:
- The developer trusts Claude Code's output because it's "from the official docs"
- The vulnerable pattern may look syntactically correct and pass tests
- The vulnerability may only manifest under specific conditions (making it hard to catch in review)

### I-3: Subtle Misinformation

**Severity**: MEDIUM

Incorrect but plausible documentation (wrong function signatures, deprecated patterns presented as current, incorrect default values) leads to:
- Code that compiles but behaves incorrectly
- Subtle bugs that are difficult to trace back to documentation errors
- Wasted developer time debugging issues caused by bad reference material

This is the most likely impact from AV-2 (compromised source documentation) because the tampering doesn't need to be sophisticated — just wrong enough to cause problems.

### I-4: Data Exfiltration via Crafted MCP Responses

**Severity**: HIGH (if achievable)

If prompt injection causes Claude Code to exfiltrate data (e.g., by generating code that sends environment variables to an attacker-controlled server, or by using MCP tool calls to leak context), the impact extends beyond the current coding session. However, this requires:
- Successful prompt injection (I-1)
- Claude Code having access to sensitive data
- The developer not noticing the exfiltration in generated code
- Claude Code's safety filters not catching the pattern

The multiple prerequisite chain makes this lower likelihood despite high theoretical severity.

### I-5: Reputation and Trust Damage

**Severity**: MEDIUM

If gdev-served documentation is discovered to contain tampered content, it undermines trust in the entire local-documentation-via-MCP approach. This is a project-level risk rather than a technical one, but it motivates defense-in-depth even for lower-likelihood attack vectors.

## 5. Existing Mitigations

### M-1: Nix SRI Hash Pinning

**What it protects**: After a hash is recorded for a content artifact, any modification to that artifact will cause a hash mismatch and Nix will reject the download. This covers:
- MITM attacks on subsequent downloads (AV-3, after first pin)
- Mirror/redistribution tampering (AV-4, if hashes are pinned)
- Accidental corruption during download or storage

**What it does NOT protect**:
- First-download provenance (the initial hash recording trusts the download source)
- Supply chain compromise of the Nix package itself (AV-5 — attacker updates the hash)
- Compromised upstream infrastructure (AV-1 — hash faithfully reflects malicious content)
- Post-download local tampering outside the Nix store (AV-6b, AV-6c)

### M-2: HTTPS Transport Security

**What it protects**: TLS encryption and certificate validation prevent passive eavesdropping and basic MITM attacks during download.

**What it does NOT protect**:
- CDN compromise (AV-3c — attacker controls the server, valid TLS)
- Certificate authority compromise (AV-3d)
- HTTP downgrade if misconfigured (AV-3a)
- Anything upstream of the download (AV-1, AV-2)

### M-3: Claude Code Prompt Injection Defenses

**What it protects**: Claude models are trained to distinguish tool output (including MCP results) from user instructions. Content returned via MCP is treated as data, not instructions. This provides baseline defense against naive prompt injection.

**What it does NOT protect**:
- Sophisticated prompt injection that exploits model-specific behaviors
- Indirect prompt injection via plausible-looking code examples that happen to be insecure
- Content that is technically correct but misleading in context

### M-4: Local MCP Server (No Network Exposure)

**What it protects**: The MCP server runs locally and communicates via stdio/Unix socket. There is no network-accessible attack surface for the MCP server itself.

**What it does NOT protect**: Anything about the content served by the MCP server. Local execution is an isolation benefit, not a content integrity mechanism.

### M-5: Well-Known Content Sources

**What it protects**: Content comes from established, widely-used sources (MDN, official library documentation). These sources have editorial processes, large user bases that catch errors, and reputational incentives for accuracy.

**What it does NOT protect**:
- Determined targeted attacks that get malicious edits through review
- Less popular documentation sources with weaker editorial oversight
- The gap between "mostly accurate" and "cryptographically verified"

### M-6: ZIM Checksums (MD5 In-File, SHA-256 Sidecar)

**What it protects**: Detects accidental corruption of ZIM files after download (MD5 in-file) or during download (SHA-256 sidecar verification).

**What it does NOT protect**:
- Intentional tampering (attacker recalculates checksums for tampered content)
- No authentication of the checksum source (the sidecar file could also be tampered)
- The libzim #614 write-corruption gap (checksums generated after write, missing write-time corruption)

### M-7: Nix Store Immutability

**What it protects**: On NixOS, /nix/store is mounted read-only. Content stored there cannot be modified by unprivileged processes. This prevents casual local filesystem tampering.

**What it does NOT protect**:
- Root-level access (can remount read-write)
- Content read from locations outside /nix/store
- Content that was malicious before entering the store

## 6. Mitigation Gap Analysis

| Attack Vector | Signing Addresses? | Nix Hash Helps? | Other Control Needed |
|---|---|---|---|
| AV-1: Compromised upstream | Partially (transparency log) | No (hash matches malicious content) | Reproducible builds, multi-party verification |
| AV-2: Compromised source docs | No | No | Content-level analysis, version diffing |
| AV-3: MITM on download | Yes (strong) | Partially (after first pin) | HTTPS enforcement, certificate pinning |
| AV-4: Unofficial mirrors | Yes (strong) | Yes (if pinned) | Restrict to official sources |
| AV-5: Nix package compromise | Yes (independent verification) | No (hash is changed by attacker) | Code review, commit signing, CODEOWNERS |
| AV-6: Local filesystem tampering | No (post-download) | Partially (Nix store immutability) | Runtime integrity checks, filesystem permissions |
| AV-7: Compromised MCP server | No | No | Code signing on server binary, reproducible builds |
| AV-8: Prompt injection in content | No | No | Content sanitization, AI-side defenses, output review |

### What Signing Addresses

Signing provides strong protection for the **distribution integrity** problem (AV-3, AV-4, AV-5). It binds content to a known producer identity and is verifiable regardless of distribution channel. Specifically:

- **Mirror/redistribution integrity**: Any third party can verify that content was produced by the legitimate upstream without trusting the distribution channel.
- **First-download provenance**: Unlike Nix hash pinning, signing provides provenance on the very first download (assuming the signing key/identity is pre-trusted).
- **Nix package supply chain**: Even if an attacker modifies the Nix expression to point to different content, signature verification catches the mismatch.

### What Signing Does NOT Address

- **Compromised source material** (AV-2): The single largest residual risk. Signing proves who produced the content, not whether the content is safe or correct.
- **Compromised build infrastructure** (AV-1): If the signer is compromised, signatures are valid over malicious content. Transparency logs (Sigstore Rekor) provide forensic evidence but not prevention.
- **Prompt injection** (AV-8): Orthogonal to content authenticity. Legitimately signed documentation can contain injection payloads.
- **Post-download tampering** (AV-6): Signing is a point-in-time verification. Runtime integrity requires re-verification on each read.
- **MCP server integrity** (AV-7): Content signing doesn't protect the serving layer.

## 7. Risk Prioritization

Risk = Likelihood x Impact, adjusted for existing mitigations and the specific context (local-first, Nix-managed, documentation content, AI consumer).

### Tier 1: Highest Priority

**AV-2: Compromised Source Documentation** — MEDIUM-HIGH risk
- Likelihood: MEDIUM (demonstrated in practice, large attack surface for wiki-style docs)
- Impact: HIGH (I-2 vulnerable code generation, I-3 subtle misinformation)
- Existing mitigations: Weak (M-5 editorial review is probabilistic, not deterministic)
- Signing helps: No
- Priority rationale: This is the most likely attack vector AND the one that no proposed mitigation fully addresses. It's the residual risk that persists even with perfect signing.

**AV-8: Prompt Injection via Documentation** — MEDIUM-HIGH risk
- Likelihood: MEDIUM (active research area, documentation is a plausible vector)
- Impact: CRITICAL (I-1 prompt injection execution could lead to I-4 data exfiltration)
- Existing mitigations: Moderate (M-3 Claude Code defenses, but effectiveness varies)
- Signing helps: No
- Priority rationale: The AI consumer is uniquely vulnerable to this class of attack. Traditional documentation integrity focuses on human readers; prompt injection adds a qualitatively new risk dimension.

### Tier 2: Moderate Priority

**AV-5: Nix Package Supply Chain Compromise** — MEDIUM risk
- Likelihood: LOW-MEDIUM (standard supply chain attack, smaller project = fewer reviewers)
- Impact: HIGH (complete content substitution, all downstream checks pass)
- Existing mitigations: Moderate (code review, but hash changes are routine)
- Signing helps: Yes (strong — independent verification layer)
- Priority rationale: This is the vector where signing provides the most value. Without signing, a Nix package compromise is undetectable. With signing, it requires additionally compromising the upstream signing key.

**AV-1: Compromised Upstream Infrastructure** — MEDIUM risk
- Likelihood: LOW (well-maintained open-source projects, but CI compromise is increasing)
- Impact: CRITICAL (undetectable by any downstream mechanism except transparency logs)
- Existing mitigations: Weak (none that would detect this)
- Signing helps: Partially (transparency log provides forensic evidence)
- Priority rationale: Low likelihood but catastrophic impact and very limited detection capability.

### Tier 3: Lower Priority (Given Context)

**AV-3: MITM on Download** — LOW-MEDIUM risk
- Likelihood: LOW (HTTPS + Nix hashes cover most scenarios)
- Impact: HIGH (complete content substitution)
- Existing mitigations: Strong (M-1 Nix hashes after first pin, M-2 HTTPS)
- Signing helps: Yes (closes the first-download gap)
- Priority rationale: Already substantially mitigated by existing mechanisms. Signing would close the remaining first-download gap.

**AV-4: Unofficial Mirrors** — LOW risk
- Likelihood: LOW (gdev controls fetch URLs via Nix, unlikely to use unofficial mirrors)
- Impact: HIGH (if it occurs)
- Existing mitigations: Strong (M-1 Nix hash pinning, controlled URLs)
- Signing helps: Yes (strong)
- Priority rationale: gdev's architecture (Nix-managed URLs) largely prevents this.

**AV-6: Local Filesystem Tampering** — LOW risk
- Likelihood: LOW (requires elevated access; attacker has better options)
- Impact: MEDIUM-HIGH
- Existing mitigations: Strong (M-7 Nix store immutability on NixOS)
- Signing helps: No
- Priority rationale: If an attacker has root on your machine, documentation tampering is the least of your problems.

**AV-7: Compromised MCP Server** — VERY LOW risk
- Likelihood: VERY LOW (locally installed via Nix, same access requirements as AV-6)
- Impact: HIGH
- Existing mitigations: Moderate (M-4 local only, M-7 Nix store)
- Signing helps: No
- Priority rationale: Same "root on your machine" argument as AV-6.

## 8. Recommended Defense-in-Depth Layers

Based on the risk prioritization, a comprehensive defense strategy should include:

### Layer 1: Content Signing Verification (Addresses AV-3, AV-4, AV-5)
- Verify cryptographic signatures on ZIM files and DevDocs bundles at download time
- Use Sigstore/cosign for keyless verification where possible
- Fall back to GPG signature verification where Sigstore isn't available
- This is the primary gap that the parent research spike investigates

### Layer 2: Content Provenance Metadata in MCP Responses (Addresses AV-1, AV-2 partially)
- MCP server includes metadata (content source, content date, verification status) in responses
- Claude Code (or the user) can make trust decisions based on provenance
- Enables differential trust: verified-signed MDN > unsigned community wiki

### Layer 3: Content Diffing and Anomaly Detection (Addresses AV-2, AV-8)
- Compare new content versions to previous versions at update time
- Flag large unexpected changes or suspicious patterns (e.g., unusual instructions in code examples)
- This is the only mitigation for compromised source documentation
- Lightweight implementation: store content hashes per-page, flag pages with >N% change

### Layer 4: Prompt Injection Hardening (Addresses AV-8)
- Sanitize documentation content before serving via MCP (strip unusual Unicode, control characters)
- Structure MCP responses to clearly delineate documentation content from MCP metadata
- Rely on and stay current with Claude Code's built-in defenses
- Consider content-type tagging (code example vs prose vs configuration) to help Claude Code contextualize

### Layer 5: Nix Package Integrity (Addresses AV-5)
- Require commit signing on gdev repository
- CODEOWNERS rules requiring multiple reviewers for content hash changes
- Separate content hash updates into dedicated PRs that are easy to audit
- CI verification that fetched content matches declared signatures

### Layer 6: Runtime Integrity (Addresses AV-6)
- Verify content hashes at MCP server startup (compare on-disk content to expected hashes)
- Serve content directly from /nix/store paths (immutable on NixOS)
- Log content access patterns for forensic analysis

## 9. Residual Risk Assessment

Even with all recommended layers implemented:

1. **Compromised source documentation (AV-2) remains the largest residual risk.** No proposed mitigation fully addresses malicious content that was legitimately present in the source documentation. Content diffing can catch large-scale changes but not subtle, targeted modifications.

2. **Prompt injection (AV-8) is an evolving threat with no complete solution.** Defenses will improve over time (both in Claude Code and in content sanitization), but this is fundamentally an arms race. The risk is unique to AI-consumer documentation pipelines and has no analog in traditional documentation distribution.

3. **Upstream infrastructure compromise (AV-1) is detectable but not preventable from the consumer side.** Sigstore transparency logs provide forensic evidence, and reproducible builds would help, but neither is within gdev's control to implement.

The signing gap — the primary focus of this spike — is a **meaningful hardening measure** that substantially reduces risk from the distribution integrity attack vectors (AV-3, AV-4, AV-5). However, it does not address the highest-priority risks (AV-2, AV-8), which require content-level rather than provenance-level mitigations.
