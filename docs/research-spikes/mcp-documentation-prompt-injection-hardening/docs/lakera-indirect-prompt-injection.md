# Indirect Prompt Injection: The Hidden Threat Breaking Modern AI Systems
- **Source**: https://www.lakera.ai/blog/indirect-prompt-injection
- **Retrieved**: 2026-05-14
- **Note**: AI-summarized content

## Definition and Core Concept

Indirect prompt injection (IPI) embeds malicious instructions in data sources rather than direct user input. The attacker "poisons the data the model will later read: a webpage, a PDF, an MCP tool description, an email, a memory entry, or a configuration file."

The fundamental vulnerability: "Modern AI applications blend system prompts, user inputs, retrieved documents, and tool metadata into a single context window. This is where the vulnerability lives."

## Attack Categories and Surfaces

### Primary Ingestion Surfaces

1. Webpages - HTML content, blogs, hidden text
2. Documents - PDFs, reports, scanned files
3. Emails & Metadata - Body content, headers
4. MCP Tool Descriptions - Schemas, capability text
5. RAG Corpora - Knowledge bases, documents
6. Memory Stores - Conversation history
7. Code Repositories - Configuration files, comments
8. Internal Knowledge Bases - Wikis, CRM notes

### Four-Step Attack Lifecycle

1. **Poison the Source** - Attacker embeds hidden instructions in content
2. **AI Ingestion** - System retrieves poisoned material during operations
3. **Activation** - Model interprets malicious text as instructions
4. **Unintended Behavior** - Data leaks, output manipulation, or harmful tool calls

## Real-World Demonstrations

### Perplexity Comet Incident
Researchers hid invisible text in a Reddit post. When Comet's browser summarizer processed the page, it extracted hidden instructions that leaked the user's one-time password to attacker infrastructure.

### Zero-Click RCE in MCP-Based IDEs
"A seemingly harmless Google Docs file triggered an agent inside an IDE to fetch attacker-authored instructions from an MCP server. The agent executed a Python payload, harvested secrets."

### CVE-2025-59944
A case sensitivity bug in Cursor's protected file path allowed agents to read unintended configuration files, leading to remote code execution.

### Agent Breaker Scenarios (Lakera testing platform)
- Trippy Planner - Travel blog injects phishing links into itineraries
- OmniChat Desktop - Compromised MCP tool descriptions leak email addresses
- PortfolioIQ Advisor - Due diligence PDFs alter risk assessments
- Curs-ed CodeReview - Poisoned code rules inject harmful dependencies
- MindfulChat - Single memory entry shapes behavior across sessions

## Direct vs. Indirect Comparison

| Factor | Direct | Indirect |
|--------|--------|----------|
| Visibility | Visible to user | Invisible to user |
| Attack Vector | Prompt interface | External content |
| Success Rate | Lower (more scrutiny) | Higher (unmonitored surfaces) |
| Detection Difficulty | Obvious patterns | Natural language camouflage |

## Core Vulnerabilities Enabling IPI

1. **Blended Context Streams** - One continuous token stream, no clear boundaries
2. **Instruction-Following Design** - Models cannot reliably distinguish source of instructions
3. **Silent Attack Surfaces** - Non-interactive channels avoid monitoring
4. **Amplification Through Autonomy** - Agentic AI increases blast radius
5. **Memory Persistence** - Poisoned entries influence behavior across sessions
6. **Ineffective Sanitization** - Standard filters miss natural language attacks

## Defense Strategies

1. System prompt hardening (limited effectiveness alone)
2. Trust boundary separation with clear delimiters
3. Tool call validation against strict schemas
4. Output verification layers (secondary LLMs, rules, validators)
5. Least-privilege architecture
6. Zero trust for all external data
7. Behavioral monitoring and logging
8. Architectural risk assessment (question whether autonomy is necessary)

## Key Insight
"The security perimeter is no longer the model. It is everything around it."
