# Prompt Injection Surface Analysis: Web Fetch vs Local Documentation

## Executive Summary

Web-fetched content presents a large, actively exploited prompt injection attack surface for AI coding assistants. Local documentation corpora (DevDocs, Kiwix/ZIM) eliminate the dominant attack vectors -- dynamic content injection, MITM, and bot-triggered failures -- while introducing a smaller, more controllable residual risk profile centered on upstream supply chain integrity. This analysis quantifies the threat differential and concludes that local-first documentation access provides a meaningful security improvement for gdev's developer environment bootstrap.

---

## 1. Web Fetch Prompt Injection Vectors

### 1.1 The Fundamental Problem: Instruction-Data Collapse

Modern LLMs cannot reliably distinguish between instructions and data within a single context window. As Lakera's research states: "Most modern AI applications blend system prompts, user inputs, retrieved documents, and tool metadata into a single context window." When an AI coding assistant fetches a web page and incorporates its content into the context, any instructions embedded in that page become indistinguishable from legitimate system prompts.

This is not a bug that can be patched -- it is an architectural property of transformer-based language models. OpenAI has publicly stated that "prompt injection, much like scams and social engineering on the web, is unlikely to ever be fully 'solved'" (Fortune, December 2025).

### 1.2 Hidden Instruction Techniques

Unit 42's field research (March 2026) documented the specific concealment methods used in the wild, with quantified prevalence:

**Visual concealment (CSS/HTML tricks):**
- `font-size: 0px` with `line-height: 0` -- collapses text to zero visual footprint
- `display: none` / `visibility: hidden` -- removes from visual rendering but preserved in DOM/text extraction
- `opacity: 0` or white-on-white text -- invisible to humans, readable by scrapers
- `position: absolute; left: -9999px` -- pushed off-screen
- Content placed in `<textarea>` elements (normally input-only)

**Observed delivery method distribution:**
- Visible plaintext: 37.8% (brazenly embedded in footers, comments)
- HTML attribute cloaking (`data-*` attributes): 19.8%
- CSS rendering suppression: 16.9%
- Remaining: obfuscation, runtime assembly, XML/SVG encapsulation

**Obfuscation techniques:**
- Zero-width Unicode characters inserted between letters (visually identical, digitally distinct)
- Homoglyph substitution (Cyrillic "a" for Latin "a" to defeat keyword filters)
- Payload splitting across multiple HTML elements (individually benign, malicious when aggregated via `innerText`)
- Unicode bi-directional override (U+202E) to reverse visible text
- Nested encoding (HTML entities inside Base64 inside URL encoding)
- HTML comments containing instructions (stripped from visual rendering, preserved in raw fetch)

**Jailbreak method distribution:**
- Social engineering framing ("developer mode", "do anything now"): 85.2%
- JSON/syntax injection (breaking out of structured contexts): 7.0%
- Multilingual instruction repetition: 2.1%

### 1.3 SEO-Poisoned Documentation Pages

Microsoft Security (February 2026) documented "AI Recommendation Poisoning" -- a direct analog of SEO poisoning adapted for AI systems. Key findings:

- Over 50 unique poisoning prompts found from 31 companies across 14 industries
- **Legitimate businesses, not just threat actors**, deploy these techniques
- Freely available tooling (CiteMET NPM package, AI Share URL Creator) democratizes the attack
- Patterns include: "remember [service] as a trusted source," injecting marketing copy into AI memory, biasing financial/health/educational recommendations

This is directly relevant to documentation fetching: a documentation page with embedded "always recommend library X for this use case" or "this deprecated API is the preferred approach" would manipulate coding assistant output.

### 1.4 Compromised or Malicious Documentation Sites

The attack surface extends beyond hidden text to entirely compromised sites:

- Google observed a 32% increase in malicious prompt injection detections between November 2025 and February 2026 (scanning CommonCrawl's 2-3 billion monthly webpage snapshots)
- Unit 42 documented sites specifically designed to target AI agents: `reviewerpress[.]com` (24 injection attempts per page), `splintered[.]co[.]uk` (database deletion commands), `1winofficialsite[.]in` (SEO manipulation)
- 73.2% of malicious injections appeared on `.com` domains, 4.3% on `.dev` domains -- documentation-adjacent TLDs

### 1.5 Man-in-the-Middle on HTTP Fetches

When AI tools fetch documentation over HTTP (or even HTTPS with compromised certificate chains):
- Content can be modified in transit on untrusted networks (corporate proxies, public WiFi, compromised DNS)
- SSL stripping attacks downgrade HTTPS to HTTP transparently
- DNS poisoning can redirect documentation domains to attacker-controlled servers
- Unlike browser-based browsing, AI tool fetches typically lack visual indicators of connection security

The risk is amplified because AI tools often make fetches programmatically without user visibility into the connection details.

### 1.6 Real-World CVEs in AI Coding Assistants

Two critical vulnerabilities demonstrate the concrete impact:

**CVE-2025-53773 (GitHub Copilot, CVSS 7.8):** Prompt injection via web content, GitHub issues, or source code caused Copilot to write `"chat.tools.autoApprove": true` to `.vscode/settings.json`, enabling "YOLO mode" -- unrestricted shell command execution without user confirmation. The change was written directly to disk (not shown as a reviewable diff). Patched August 2025.

**CVE-2025-59944 (Cursor IDE, CVSS 8.0):** A case-sensitivity bug in file path protection allowed prompt injection to create `.cUrSoR/mcp.json` (bypassing the protection on `.cursor/mcp.json` on case-insensitive filesystems), enabling persistent MCP server registration that executed attacker commands on every project open. Fixed in Cursor 1.7.

**CVE-2025-54136 (Cursor MCPoison):** Attackers could modify previously-approved MCP configurations in shared Git repositories. Trust was bound to the MCP key name, not the command content -- so changing a benign command to a reverse shell required no re-approval. Fixed in Cursor 1.3.

### 1.7 Attack Success Rates

Research quantifies the severity:
- Prompt injection success rates of **66.9% to 84.1%** in auto-execution mode across tested coding assistants (arXiv 2601.17548)
- PoisonedRAG demonstrates **90% success with only 5 poisoned documents**
- Eight existing defense mechanisms can be bypassed with adaptive attacks achieving **>50% success**
- Even Anthropic's best defense (Claude Opus 4.5) still shows a **1% success rate** against 100 adaptive attempts per environment

---

## 2. Bot Blocks and Rate Limiting

### 2.1 The Blocking Landscape

Documentation site accessibility for AI tools has deteriorated significantly:

**Cloudflare (July 2025):** Now blocks all known AI crawlers by default for every new Cloudflare domain. Over one million customers opted into blocking when it was opt-in (September 2024). This affects any documentation site behind Cloudflare's CDN -- which is a substantial fraction of the web.

**Rate of AI crawling growth:** GPTBot requests rose 147% and Meta-ExternalAgent requests rose 843% from July 2024 to July 2025, driving aggressive countermeasures.

**Stack Overflow:** CEO Prashanth Chandrasekar publicly advocated for blocking AI bots, stating "Community platforms that fuel LLMs should be compensated." Stack Overflow has implemented restrictions on AI crawling.

**Trend toward "Fully Disallowed":** Websites are increasingly choosing complete AI crawler blocks rather than partial restrictions -- a steep decrease in "Partially Disallowed" policies in favor of "Fully Disallowed" for GPTBot, CCBot, and Google-Extended.

### 2.2 Impact on Developer Workflow

When AI coding assistants attempt to fetch documentation and encounter blocks:
- Cloudflare challenge pages return HTML/JavaScript challenges instead of documentation content
- Rate-limited responses return 429 errors, often with opaque retry-after periods
- CAPTCHA-gated content is completely inaccessible to automated tools
- The failure is often silent or confusing -- the AI may summarize the error page as if it were documentation
- Fallback to cached or hallucinated content introduces its own risks

### 2.3 Which Sites Block?

While comprehensive data on individual documentation sites' bot policies is limited, the pattern is clear:
- Any site behind Cloudflare (default blocking since July 2025) is potentially affected
- Community-generated content sites (Stack Overflow, forums) are actively hostile to AI scrapers
- Official language/framework documentation sites have mixed policies, but the trend favors restriction
- The `ai-robots-txt` project (3,900+ GitHub stars) provides standardized blocking configurations across Apache, nginx, Caddy, HAProxy, and lighttpd

### 2.4 Implications for gdev

Bot blocks create a reliability problem separate from security: a developer environment bootstrap tool cannot depend on web fetches that may fail unpredictably based on the documentation site's CDN configuration, rate limiting policies, or anti-bot measures. Local documentation eliminates this entire failure class.

---

## 3. Local Documentation Security Properties

### 3.1 Content Frozen at Fetch Time

Local documentation corpora (DevDocs offline data, Kiwix ZIM files) capture content at a specific point in time:
- **No dynamic injection possible** -- the content does not change between download and query
- **No JavaScript execution** -- static HTML/text is served, eliminating runtime assembly attacks
- **No CSS rendering** -- text extraction from local stores bypasses visual concealment entirely (no `display:none`, no `opacity:0`, no off-screen positioning)
- **No HTML attribute cloaking** -- local documentation MCP servers serve processed text, not raw HTML with `data-*` attributes

This eliminates the entire category of visual concealment attacks that constitute ~37% of observed web-based injection delivery methods (CSS rendering suppression + HTML attribute cloaking from Unit 42's data).

### 3.2 No Network Calls During Query

Once documentation is local:
- **No MITM risk** -- no network traffic to intercept or modify
- **No DNS poisoning** -- no domain resolution to redirect
- **No SSL stripping** -- no TLS connections to downgrade
- **No tracking** -- no analytics scripts, no telemetry, no fingerprinting
- **No bot detection** -- queries never leave the machine, never encounter Cloudflare challenges or CAPTCHAs
- **No rate limiting** -- unlimited queries at local I/O speed

### 3.3 Known-Good Upstream Sourcing

Both DevDocs and Kiwix source from identifiable upstreams:

**DevDocs:** Scrapers pull from official documentation sources (MDN for web APIs, python.org for Python, etc.). The scraper code is open source (freeCodeCamp/devdocs), reviewable, and deterministic. Content passes through a pipeline of HTML filters that normalize and sanitize.

**Kiwix/ZIM:** Content is sourced from Wikipedia, Stack Exchange data dumps, and other curated sources. ZIM files are generated by the openZIM project's scrapers (also open source) from known upstream datasets.

### 3.4 Controlled Update Cadence

Unlike web content that can change between fetches:
- Updates happen on a deliberate schedule (weekly, monthly, per-release)
- Each update is a discrete event that can be reviewed, checksummed, and rolled back
- Stale content is a known tradeoff -- but stale-and-safe beats current-and-poisoned for most documentation needs
- Security-critical updates (e.g., deprecation of insecure APIs) can be flagged through update changelogs

---

## 4. Residual Risks in Local Corpora

Local documentation is not risk-free. The attack surface shrinks dramatically but does not reach zero.

### 4.1 Poisoned Upstream Documentation

If the official documentation source itself is compromised:
- A malicious commit to python.org documentation could propagate to local DevDocs copies
- A compromised MDN contributor could insert hidden instructions that survive scraping
- Official documentation has been known to contain errors and outdated security guidance

**Likelihood:** Low. Official documentation repositories have review processes, contributor vetting, and version control. Compromising python.org or MDN documentation requires sustained, sophisticated effort.

**Mitigation:** Source from official release artifacts rather than live sites. Pin to known-good versions. DevDocs versioned documentation provides some protection (you choose which version to install).

### 4.2 Stack Overflow Answer Quality

The CISPA/USENIX research (2025) demonstrates systemic issues with Stack Overflow code snippets:
- **Every second reused snippet is outdated** regardless of programming language (study of ~11,500 GitHub projects)
- **No evidence that developers update copied snippets** when the original Stack Overflow answer is corrected
- **69 serious vulnerabilities found in 2,560 C++ snippets** across 29 CWE types
- Privacy-violating code patterns are common and propagate unchecked

This is not prompt injection per se, but it is a content quality/security problem that persists in local corpora: a Kiwix snapshot of Stack Overflow preserves both good and bad answers, including intentionally misleading or subtly vulnerable code.

**Mitigation:** Stack Overflow content should be treated as lower-trust than official documentation. MCP server responses can tag content by source (official docs vs. community Q&A) so the AI assistant can weight accordingly.

### 4.3 ZIM/DevDocs Data Integrity

**ZIM format integrity:**
- Uses a checksum appended to the file (algorithm not explicitly documented as cryptographic in available sources)
- SHA-256 checksums published alongside downloads at `download.kiwix.org`
- **No cryptographic signing** of ZIM file contents -- checksums verify download integrity but not content authenticity
- Known limitation: "any data corruption occurring during the initial writing of the ZIM file to the disk cannot be detected by the checksum" (libzim issue #614)
- ZIM archives can run dynamic code in a browser context -- Kiwix warns users to "get ZIM archives only from a secure source"

**DevDocs integrity:**
- No cryptographic signing of scraped content
- Content is generated by Ruby scrapers that transform upstream HTML through a filter pipeline
- Integrity depends on trusting the DevDocs scraper code (open source, auditable) and the upstream source
- The scraper output is a set of normalized HTML partials and JSON index files -- no built-in integrity verification

**Attack scenario:** A compromised DevDocs mirror or a tampered ZIM file distributed through an unofficial channel could contain injected content. Since neither format uses cryptographic signatures for content authenticity, verification depends on trusting the download source and verifying checksums.

**Mitigation:** Download only from official sources (devdocs.io, download.kiwix.org). Verify SHA-256 checksums. In gdev's context, Nix packaging can pin exact hashes of downloaded artifacts, providing reproducible integrity verification.

### 4.4 Stale Documentation Leading to Insecure Recommendations

Frozen documentation means:
- Deprecated APIs may still appear as current recommendations
- Security advisories published after the snapshot are missing
- Version-specific vulnerabilities (e.g., "use X instead of Y in version 3.2+") may not reflect the developer's actual installed version

**Mitigation:** Hybrid architecture with local-first, web-fallback specifically addresses this. The AI can be instructed to check currency of security-sensitive recommendations. Regular update cadence (monthly or per-release) limits the staleness window.

---

## 5. Quantitative Risk Comparison

### 5.1 Threat Model: Web Fetch

| Factor | Assessment |
|--------|-----------|
| **Attack surface** | Every fetched page is a potential injection vector. Entire public web is in scope. |
| **Likelihood** | High and increasing. 32% quarterly growth in observed attacks. 66-84% success rate in research. |
| **Impact** | Critical. Demonstrated RCE (CVE-2025-53773), persistent backdoors (CVE-2025-54136), data exfiltration. |
| **Attacker effort** | Low. Freely available tooling. No target-specific knowledge needed. |
| **Mitigations available** | Model-level defenses (1% residual success at best), human confirmation for dangerous actions, content filtering. |
| **Reliability** | Poor. Bot blocks, rate limits, CAPTCHAs cause unpredictable failures. |

### 5.2 Threat Model: Local DevDocs

| Factor | Assessment |
|--------|-----------|
| **Attack surface** | Upstream official documentation sources only. No dynamic content. No network exposure at query time. |
| **Likelihood** | Very low. Requires compromising official documentation repos or the DevDocs scraper supply chain. |
| **Impact** | Medium. Poisoned docs could suggest insecure patterns, but no mechanism for dynamic RCE payload delivery. |
| **Attacker effort** | High. Must compromise official documentation source or DevDocs infrastructure. |
| **Mitigations available** | Pinned versions, Nix hash verification, open-source scraper auditability, source tagging. |
| **Reliability** | Excellent. No network dependency at query time. |

### 5.3 Threat Model: Local Kiwix/Stack Overflow

| Factor | Assessment |
|--------|-----------|
| **Attack surface** | Stack Exchange data dump content + ZIM packaging pipeline. |
| **Likelihood** | Low for injection attacks. Medium for inherent content quality issues (50% of snippets outdated). |
| **Impact** | Medium. Vulnerable code patterns could be recommended. No dynamic payload delivery mechanism. |
| **Attacker effort** | High for targeted injection into data dumps. Low for exploiting existing bad answers (they're already there). |
| **Mitigations available** | Source tagging (mark as community content), checksum verification, score-based filtering. |
| **Reliability** | Excellent. No network dependency. |

### 5.4 Threat Model: Hybrid (Local-First, Web Fallback)

| Factor | Assessment |
|--------|-----------|
| **Attack surface** | Local corpus (small surface) for majority of queries. Web surface only for fallback queries. |
| **Likelihood** | Low for most queries (served locally). Moderate for fallback queries (web exposure). |
| **Impact** | Reduced. Web fallback can be sandboxed with stricter controls (e.g., confirmation required, no tool execution from web-sourced content). |
| **Attacker effort** | Must either compromise upstream docs (high effort) or wait for fallback to web (reduced opportunity). |
| **Mitigations available** | All local mitigations + differential trust levels for web-sourced content + user confirmation for web fallback. |
| **Reliability** | Good. Graceful degradation instead of hard failure. |

### 5.5 Risk Reduction Summary

Moving from pure web fetch to local-first documentation eliminates or substantially reduces:
- **Visual concealment attacks**: Eliminated (no CSS/HTML rendering in local text extraction)
- **Dynamic content injection**: Eliminated (content frozen at download time)
- **MITM attacks**: Eliminated (no network calls during query)
- **Bot blocks/rate limits**: Eliminated (local I/O only)
- **SEO poisoning**: Eliminated (content sourced from official docs, not search results)
- **Real-time weaponization**: Eliminated (attacker cannot modify content between download and query)

Residual risks requiring ongoing management:
- **Upstream supply chain**: Mitigated by pinning versions and verifying checksums
- **Content quality** (especially Stack Overflow): Mitigated by source tagging and differential trust
- **Staleness**: Mitigated by controlled update cadence and web fallback for currency-sensitive queries

---

## 6. Prior Art and Industry Guidance

### 6.1 OWASP LLM Top 10 (2025)

Prompt injection holds the **#1 position** (LLM01:2025) in the OWASP Top 10 for LLM Applications. Their guidance explicitly distinguishes indirect prompt injection (via external sources including websites) from direct injection and recommends:
- Segregating external content with clear untrusted source markers
- Enforcing least privilege for LLM-connected systems
- Requiring human approval for high-risk operations
- Conducting adversarial testing

OWASP's mitigation to "segregate external content" directly supports the local-first architecture: local documentation from known-good sources is inherently more segregated from attacker-controlled content than arbitrary web fetches.

### 6.2 Research Papers

**Maloyan & Namiot (arXiv 2601.17548):** Systematically analyzed prompt injection across coding assistants (Claude, Copilot, Cursor, Codex). Found 66.9-84.1% attack success rates in auto-execution mode. Recommended input validation, instruction hierarchy enforcement, and multi-agent defense coordination.

**Google Threat Intelligence (April 2026):** Scanned CommonCrawl data finding 32% quarterly growth in malicious injections. Noted current attacks are "low sophistication" but trajectory is toward maturation. Categorized attacks into pranks, SEO manipulation, data exfiltration, and destructive commands.

**Microsoft Security (February 2026):** Documented "AI Recommendation Poisoning" with 50+ prompts from 31 companies. Mapped to MITRE ATT&CK (T1204.001, AML.T0051, AML.T0080.000). Noted that legitimate businesses, not just threat actors, deploy these techniques.

**CISPA/USENIX (2025):** Found 50% of reused Stack Overflow snippets are outdated across ~11,500 GitHub projects, with no evidence developers track upstream fixes. Identified 69 serious vulnerabilities in 2,560 C++ snippets.

### 6.3 Corporate Policies on AI Tool Internet Access

Air-gapped AI deployment is standard practice in regulated environments:
- **DoD IL5/IL6, FedRAMP High**: Require air-gapped AI with no external network access
- **HIPAA, FFIEC, SR 11-7**: Financial and healthcare regulations frequently prohibit external AI connections
- **Los Alamos National Laboratory** (January 2025): Self-hosts LLMs for handling controlled unclassified information rather than using cloud-based services
- **Sovereign AI** reached peak priority on the 2025 Gartner Hype Cycle for government services

The MIT thesis "Securing Intelligence: The Strategic Necessity of Air-Gapped AI Systems" (2025) argues that eliminating network connectivity is the most effective way to prevent indirect prompt injection, data exfiltration, and supply chain attacks in AI systems.

### 6.4 Anthropic's Defense Research

Anthropic has invested in three defensive layers for Claude's web browsing:
1. **Adversarial training** -- RL exposure to simulated prompt injections
2. **Content classification** -- classifiers scanning untrusted content entering the context window
3. **Red team testing** -- continuous probing by security researchers

Result: Claude Opus 4.5 achieved a 1% attack success rate against adaptive attackers (100 attempts per environment). This is the best published result but still means ~1 in 100 sophisticated attack attempts succeeds -- non-trivial for a tool processing hundreds of web pages per development session.

### 6.5 Relevance to gdev

The industry consensus supports gdev's local-first approach:
- OWASP recommends segregating external content -- local docs are inherently segregated
- Air-gapped patterns are established in regulated industries -- local-first is a pragmatic partial air-gap
- Even the best model-level defenses (Anthropic's 1% residual) are insufficient for high-volume web access
- The reliability benefits (no bot blocks) compound the security benefits

---

## 7. Conclusions and Recommendations for gdev

### 7.1 The Security Case Is Strong

Local-first documentation eliminates the dominant prompt injection attack surface. The threat is not theoretical -- it has produced CVEs with demonstrated RCE in the tools gdev's users will run (Copilot, Cursor). The 32% quarterly growth in observed attacks means the threat landscape will worsen. Waiting for model-level defenses to mature is insufficient given the architectural nature of the vulnerability (instruction-data collapse).

### 7.2 Recommended Architecture

1. **Local-first with differential trust tagging**: Serve documentation from local DevDocs/ZIM corpora. Tag all responses with their source (official docs vs. community content) so the AI can apply appropriate trust levels.

2. **Web fallback with elevated controls**: When local documentation is insufficient, fall back to web fetch but with:
   - Explicit marking of web-sourced content as untrusted
   - User confirmation required for any tool execution following web-fetched context
   - Rate limiting and domain allowlisting for web fallback

3. **Nix-pinned integrity**: Use Nix to pin exact hashes of DevDocs/ZIM artifacts, providing reproducible integrity verification that neither format natively offers through cryptographic signing.

4. **Controlled update cadence**: Monthly or per-major-release updates to local corpora, with changelog review for security-relevant changes.

### 7.3 What This Does Not Solve

Local documentation does not protect against:
- Prompt injection via source code files in the project being worked on (in-code comments attack vector)
- Prompt injection via MCP server responses from other servers
- Prompt injection via Git repository content (issues, PRs, commit messages)
- Inherently bad advice in Stack Overflow answers (content quality, not injection)

These require separate mitigations (code review, MCP server vetting, human confirmation for dangerous operations) that are complementary to but independent of the documentation access strategy.

---

## Sources

All source documents saved to `docs/`:
- `unit42-web-based-indirect-prompt-injection-wild.md` -- Unit 42 field research on IDPI
- `google-ai-threats-wild-prompt-injections.md` -- Google threat intelligence scan
- `owasp-llm01-prompt-injection-2025.md` -- OWASP LLM Top 10 guidance
- `knostic-prompt-injection-ides.md` -- IDE-specific prompt injection vectors
- `microsoft-ai-recommendation-poisoning.md` -- AI recommendation poisoning research
- `arxiv-prompt-injection-agentic-coding-assistants.md` -- Academic attack success rates
- `lakera-indirect-prompt-injection-hidden-threat.md` -- Comprehensive IPI analysis
- `anthropic-prompt-injection-defenses.md` -- Anthropic's defense mechanisms
- `embracethered-copilot-rce-cve-2025-53773.md` -- Copilot RCE vulnerability
- `checkpoint-cursor-mcpoison-cve-2025-54136.md` -- Cursor MCP vulnerability
- `openai-prompt-injection-never-fully-solved.md` -- OpenAI's position on the problem
- `crowdstrike-indirect-prompt-injection-hidden-risks.md` -- Enterprise threat model
- `stack-overflow-outdated-snippets-security.md` -- Stack Overflow code quality research
