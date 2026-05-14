# Web-Based Indirect Prompt Injection Observed in the Wild

- **Source**: https://unit42.paloaltonetworks.com/ai-agent-prompt-injection/
- **Retrieved**: 2026-05-14
- **Publisher**: Palo Alto Networks Unit 42

---

## Core Concept

Web-based indirect prompt injection (IDPI) occurs when attackers embed hidden instructions in webpage content that LLMs later interpret as executable commands. Unlike direct injection where attackers explicitly submit malicious input, IDPI exploits the routine content consumption of AI-powered tools—summarizers, content analyzers, and automated agents—that process untrusted web data at scale.

## Attack Surface and Threat Model

The threat escalates with agentic AI adoption. As noted in the research, "browsers, search engines, developer tools, customer-support bots, security scanners, agentic crawlers and autonomous agents routinely fetch, parse and reason over web content at scale." A single malicious webpage can influence multiple users or systems, with impact scaling alongside AI privileges.

## Attacker Intent Taxonomy

### Low Severity
- Nonsensical output injection (forcing irrelevant responses)
- Minor resource exhaustion (repeated text bloat)
- Benign anti-scraping measures

### Medium Severity
- Recruitment manipulation (misrepresenting candidate qualifications)
- Review manipulation (suppressing negative feedback)
- AI access restriction (triggering safety filters)

### High Severity
- Content moderation bypass (approving scam advertisements)
- SEO poisoning (promoting phishing sites)
- Unauthorized transactions (fraudulent purchases)

### Critical Severity
- Data destruction (database deletion attempts)
- Sensitive information leakage
- System prompt exposure
- Denial of service attacks

## Payload Delivery Methods

### Visual Concealment Techniques

**Zero-sizing strategies:**
- Setting `font-size: 0px` and `line-height: 0` to collapse text
- Using `height: 0` with `overflow: hidden`

**Off-screen positioning:**
- Applying `position: absolute` with extreme negative coordinates (`left: -9999px`)

**CSS rendering suppression:**
- `display: none` for complete removal from visual flow
- `visibility: hidden` for invisibility while maintaining DOM presence
- Placing content in `<textarea>` elements (normally used for input)

**Transparency methods:**
- `opacity: 0` for invisible text
- Color camouflage (white text on white background)

### Obfuscation Approaches

**XML/SVG encapsulation:**
Embedding prompts in CDATA sections tells parsers to ignore content, but LLMs process raw text regardless.

**HTML attribute cloaking:**
Placing malicious instructions as values in `data-*` attributes or other HTML attributes.

**Runtime assembly:**
Using Base64 encoding with runtime decoding, timed delays, and canvas-based rendering to bypass static analysis tools.

## Jailbreak Methods

### Instruction Obfuscation

**Invisible character insertion:**
Zero-width Unicode characters inserted between letters create strings appearing normal to humans but distinct digitally.

**Homoglyph substitution:**
Replacing Latin characters with visually identical characters from other alphabets (Cyrillic "a" for Latin "a") defeats keyword filters.

**Payload splitting:**
Breaking commands across multiple HTML elements; while individual elements appear benign, the aggregated innerText reconstructs malicious sentences.

**Unicode bi-directional override:**
Using U+202E right-to-left override reverses visible text while preserving semantic meaning in raw content.

**Encoding schemes:**
- HTML entity encoding (&#73; for "I")
- Base64 encoding
- URL encoding (%49 for "I")
- Nested encoding (encoding the encoding)

### Semantic Tricks

**Multilingual commands:**
Repeating malicious instructions in multiple languages targets multilingual AI capabilities, bypassing English-specific filters.

**JSON/syntax injection:**
Using syntax characters to break out of legitimate JSON structures and inject fraudulent key-value pairs.

**Social engineering:**
Framing malicious requests as legitimate updates, testing procedures, or authority directives (god mode, developer mode, "do anything now").

## Real-World Case Studies

### AI Ad Review Bypass (First Observed)
A malicious site (reviewerpress[.]com) embedded 24 injection attempts using multiple concealment methods simultaneously. The attacker combined visual suppression, obfuscation, runtime assembly, and jailbreak techniques to bypass ad-checking agents reviewing a fraudulent military glasses advertisement.

### SEO Poisoning Attack
Site 1winofficialsite[.]in placed plaintext injection in the footer impersonating "1win," a popular betting platform, to manipulate LLM-based search recommendations.

### Data Destruction Attempt
Site splintered[.]co[.]uk used CSS rendering suppression to hide an instruction commanding database deletion—a critical severity attack targeting privileged agents.

### Unauthorized Financial Transactions
Multiple sites attempted forced subscriptions, donations, and purchases by directing AI agents to payment platforms with attacker-controlled accounts.

## Distribution Patterns

**Attacker intents observed:**
- Irrelevant output: 28.6%
- Data destruction: 14.2%
- Content moderation bypass: 9.5%

**Delivery methods:**
- Visible plaintext: 37.8%
- HTML attribute cloaking: 19.8%
- CSS rendering suppression: 16.9%

**Jailbreak techniques:**
- Social engineering: 85.2%
- JSON/syntax injection: 7.0%
- Multilingual instructions: 2.1%

**Domain distribution:**
- .com domains: 73.2%
- .dev domains: 4.3%
- .org domains: 4.0%

**Injection density:**
75.8% of pages contained single injections; remaining pages had multiple injections per page.

## Defense Implications

Current defenses remain insufficient against web-scale IDPI. The research emphasizes that "LLMs cannot distinguish instructions from data inside a single context stream." Emerging mitigation approaches include:

- **Spotlighting**: Separating untrusted web content from trusted system instructions
- **Instruction hierarchy**: Establishing priority levels for different instruction sources
- **Adversarial training**: Hardening models against known injection patterns
- **Design-level defenses**: Implementing architectural protections beyond prompt engineering

Effective defense requires moving beyond signature matching toward intent analysis, visibility assessment, and behavioral correlation across multiple data sources.
