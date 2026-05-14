# Memory and Context Management in Agentic AI Systems

## Overview

Memory and context management is the foundational challenge of agentic AI. An agent's quality on complex, multi-step tasks is directly limited by its ability to maintain, retrieve, and prioritize the right information at the right time. This report covers 10 specific techniques, examining how they work mechanically, their effectiveness evidence, tradeoffs, state-of-the-art implementations, and applicability to Claude Code.

The central tension is between **context window capacity** (finite, expensive, subject to quality degradation) and **task complexity** (which demands more information than any window can hold). Every technique in this report addresses some aspect of this tension.

---

## 1. Long-Context Window Management

### How It Works

Modern LLMs offer context windows from 128K to 1M+ tokens (Claude: 200K, GPT-5: 400K, Gemini 2.5: 1M). However, raw capacity is not the same as effective capacity. The key challenge is **context rot** — as token count grows, accuracy and recall degrade.

The "Lost in the Middle" phenomenon (Liu et al., 2023) demonstrates that LLMs exhibit a U-shaped attention curve: information at the beginning and end of context receives significantly higher attention than information in the middle. Performance can degrade by **more than 30%** when relevant information shifts from start/end positions to the middle. The root cause is Rotary Position Embedding (RoPE), which introduces long-term decay that de-emphasizes middle content.

### Effective Management Strategies

**Token budgeting as a first-class constraint**: Estimate token count before sending requests and apply policies to drop low-priority items or re-summarize. Add tests that simulate worst-case prompts to prevent overflow surprises.

**Structured, compact tool outputs**: Design tools to return tight, structured payloads instead of large blobs. Return IDs and key fields, then let later steps selectively expand only what is needed. If each tool step dumps full payloads, you quickly accumulate thousands of tokens of intermediate state.

**Strategic information placement**: Place the most important information at the beginning and end of the context window, with less critical content in the middle. For CLAUDE.md files, this means putting the most-violated rules in the first and last 5 lines.

**Multi-layer memory**: Rather than cramming everything into context, use a hierarchy: in-context (always present), retrievable (on demand), and archival (persistent but not loaded). This is the pattern used by MemGPT/Letta, Mem0, and Claude Code's own CLAUDE.md + auto-memory system.

### Evidence of Effectiveness

- Lost in the Middle: >30% accuracy degradation from positional effects
- Claude Code reduced its context buffer from ~45K to ~33K tokens (16.5% of window), gaining ~12K usable tokens, demonstrating that careful buffer management matters
- Auto-compaction triggers at ~83.5% of window capacity in Claude Code as of early 2026

### Tradeoffs

- Larger context = higher latency and cost (quadratic attention in most architectures)
- More context does not mean better quality — noise competes with signal
- Context rot means quality degrades well before the window fills

### State of the Art

- **Claude Code**: 200K token window with auto-compaction at ~83.5%, subagent context isolation, modular system prompt (~110 prompt strings conditionally assembled)
- **Cursor**: Custom embedding model + AST-based chunking for efficient context loading
- **Gemini 2.5**: 1M token window, but effectiveness at full capacity is debatable

### Applicability to Claude Code

This is directly applicable. Claude Code already implements several best practices (auto-compaction, subagent isolation, CLAUDE.md hierarchy). The main opportunity is in **users learning to manage context proactively** — aggressive `/clear` usage, structured CLAUDE.md files, and understanding when to accumulate vs. clear context.

---

## 2. External Memory Systems

### How It Works

External memory systems store knowledge outside the context window in persistent storage that the agent can read from and write to. The fundamental pattern is: **context window = working memory, external storage = long-term memory**.

### MemGPT/Letta: The OS-Inspired Approach

Letta (formerly MemGPT) introduced the most influential external memory architecture, treating the LLM as an operating system that manages its own memory:

**Core Memory (In-Context)**: Labeled, persistent blocks always injected into the prompt. Topics like user info, persona, goals. Editable via tool calls (`core_memory_append`, `core_memory_replace`). Default: 2,000 characters per block.

**Archival Memory (Vector DB)**: Long-running memories and external data too large for context. Searchable via `archival_memory_search`. Stores embeddings for semantic retrieval.

**Recall Memory (Conversation History)**: Complete interaction history. Searchable via `conversation_search`. Auto-persisted to disk.

The critical innovation is **self-editing memory**: the agent decides what to store, retrieve, and update through explicit tool calls. It is not a passive system managed by external infrastructure — the agent actively manages its own memory.

### Mem0: The Memory Layer

Mem0 sits between the application and the LLM, automatically extracting relevant information from conversations:
- Three memory types: episodic (events), semantic (knowledge), procedural (patterns)
- On LOCOMO benchmark: **26% higher accuracy** than OpenAI's built-in memory
- **91% faster** response by selective retrieval vs. full context
- **~90% token reduction** compared to full-context approaches

### File-Based Memory

The simplest and often most effective approach. Claude Code's CLAUDE.md and MEMORY.md are file-based memory. The research spike methodology in this repository (tasks.md, log.md, research.md) is file-based external memory designed for agent context resilience.

Key advantages of file-based memory:
- Human-readable and editable
- Version-controlled
- No infrastructure requirements
- Survives context window compression
- Agents can read/write with standard tools

Letta's own benchmarks showed filesystem-based memory is competitive with more complex approaches, raising the question: "Is a filesystem all you need?"

### Evidence of Effectiveness

- Mem0: 26% accuracy improvement, 91% faster, 90% fewer tokens
- Letta Code: #1 on Terminal-Bench (model-agnostic)
- File-based memory (this repository's spike methodology): enables multi-session, multi-agent research workflows

### Tradeoffs

| Approach | Pros | Cons |
|----------|------|------|
| Vector DB | Semantic search, scales well | Infrastructure complexity, embedding quality matters |
| File-based | Simple, human-readable, version-controlled | No semantic search, manual organization |
| Self-editing (Letta) | Agent autonomy, adaptive | Requires tool overhead, agent may make bad memory decisions |
| Hybrid (Mem0) | Best accuracy, selective retrieval | Additional dependency, latency for extraction |

### State of the Art

- **Letta**: Self-editing hierarchical memory with OS paradigm
- **Mem0**: Universal memory layer with graph-enhanced retrieval
- **Claude Code**: File-based (CLAUDE.md + MEMORY.md) with auto-memory
- **Devin**: Persistent VM with full environment state

### Applicability to Claude Code

Claude Code already uses file-based external memory effectively. The main opportunities are:
1. **Structured memory formats** — the spike methodology in this repo (tasks.md, log.md, research.md) demonstrates how structured files serve as agent-readable external memory
2. **Auto-memory improvements** — Claude Code's auto-memory could become more structured and topic-organized
3. **Explicit memory management tools** — giving the agent explicit "remember this" and "recall that" tool patterns, similar to Letta's approach

---

## 3. Retrieval-Augmented Generation (RAG)

### How It Works

RAG augments the LLM's context with relevant retrieved information at generation time. The basic pipeline: query -> retrieve relevant chunks -> inject into context -> generate response.

### Evolution: Naive to Agentic RAG

**Naive RAG**: Fixed pipeline, single-hop retrieval. Problems: irrelevant context, no quality control, rigid chunking.

**Advanced RAG**: Adds hybrid search (lexical + vector), reranking, query rewriting, HyDE (hypothetical document embeddings).

**Agentic RAG**: LLM-based agents dynamically decide retrieval strategy:
- Plan multiple retrieval steps
- Choose tools (search, code, database)
- Reflect on intermediate results
- Adapt strategy based on quality

**Self-RAG**: Model generates special reflection tokens that trigger on-demand retrieval and self-critique. Decides *when* to retrieve rather than always retrieving.

**CRAG (Corrective RAG)**: Evaluator assesses retrieved document quality:
- Correct → use directly
- Incorrect → trigger additional retrieval
- Ambiguous → seek more context

### Chunking Strategies for Code

- **Recursive character splitting**: 400-512 tokens with 10-20% overlap (best general default)
- **AST-based chunking**: Cursor's approach — traverse AST depth-first, split into sub-trees within token limits, merge sibling nodes. Preserves function/class boundaries.
- **Semantic chunking**: Split based on topic/meaning changes
- **Agentic chunking**: LLM analyzes each document and picks the right chunking method

For code specifically, AST-based chunking dramatically outperforms naive splitting because it preserves semantic units (functions, classes, modules).

### Embedding Models

- **General-purpose**: OpenAI text-embedding-3, Cohere embed, Voyage
- **Code-specific**: Cursor trains its own embedding model on agent sessions — this produces significantly better code retrieval than generic embeddings
- **Cross-encoders for reranking**: Process query+document together for fine-grained interaction scoring (more accurate but slower — ideal for second-stage retrieval)

### When RAG Helps vs. Hurts

**Helps when**: Large knowledge base, factual grounding needed, data changes frequently, citations/transparency required, domain-specific knowledge.

**Hurts when**: Retrieved context is noisy or irrelevant (pollutes reasoning), over-retrieval floods context window, chunking artifacts break cross-reference reasoning, latency budget is tight, the answer is already in the model's training data.

### Evidence of Effectiveness

- Hybrid search + reranking is the "low-risk, high-return" baseline
- Cursor: custom code embeddings + AST chunking → 12.5% better code agent accuracy
- Self-RAG and CRAG show significant accuracy improvements by adding retrieval evaluation
- High-quality preprocessing matters more than retrieval algorithm sophistication

### Tradeoffs

- Latency: every retrieval step adds round-trip time
- Cost: embedding computation + storage + retrieval API calls
- Complexity: chunking, embedding, indexing, retrieval, reranking pipeline
- Quality ceiling: if relevant content isn't chunked/tagged correctly, even the best Agentic RAG fails
- Agentic RAG: higher token usage and latency from multi-step processes

### State of the Art

- **Cursor**: AST-based chunking + custom embeddings + Turbopuffer vector DB
- **Aider**: Tree-sitter repo map + PageRank graph ranking
- **LlamaIndex**: Agentic retrieval framework with auto-routing
- **Zep/Graphiti**: Temporal knowledge graph as retrieval layer

### Applicability to Claude Code

Claude Code uses file search tools (grep, glob, read) as its primary retrieval mechanism — essentially a form of RAG. Opportunities:
1. **Structured repo maps** (Aider's approach) would give Claude Code a token-efficient overview of the entire codebase
2. **Semantic search** over codebase embeddings would find relevant code by meaning, not just text matching
3. **Agentic retrieval** — Claude Code already does this informally when it decides what to search for

---

## 4. Scratchpads and Working Memory

### How It Works

A scratchpad is an external workspace where the agent stores temporary information while working through complex problems. It functions as explicit working memory — a place to write down intermediate results, plans, and state that would otherwise be "lost" as conversation context evolves.

### Mechanical Implementation

**JSON/Structured scratchpad**: Agent reads and writes a structured file (JSON, markdown with sections). Far more reliable than asking the model to "remember" state across reasoning steps. Structured state survives prompt engineering changes; free-text state drifts.

**Scratchpad prompting**: Agent is instructed to show its work before providing final answers. Similar to chain-of-thought but externalized to a file, making intermediate reasoning persistent and verifiable.

**Plan-then-execute with living plan**: Devin's approach — create a plan first, then continuously update the plan's progress during execution. The plan file serves as both scratchpad and progress tracker.

### Implementations

**Cursor/Windsurf scratchpad pattern**: The `devin.cursorrules` project transforms Cursor into a Devin-like agent by requiring a `scratchpad.md` file that is checked and updated before each thinking step.

**OpenSearch Scratchpad Tools**: Dedicated `WriteToScratchPadTool` and `ReadFromScratchPadTool` that enable agents to store and retrieve intermediate thoughts during a single execution session.

**Claude Code's approach**: Uses file-based working memory through tasks.md, log.md, and other structured files. The CLAUDE.md system instructions describe these as "external memory that survives context window compression."

**Iterative merge pattern**: The scratchpad is iteratively merged with earlier versions; once content exceeds a 30K-token threshold, it is compressed into a 15K-token summary by a smaller model, maintaining long-term coherence.

### Evidence of Effectiveness

- "Show Your Work" (Nye et al., 2021): Scratchpads for intermediate computation significantly improve accuracy on multi-step reasoning
- Self-Notes (Lanchantin et al.): Can act as both explicit intermediate reasoning steps and working memory for state-tracking
- Structured state (JSON scratchpad) is more reliable than free-text memory across prompt changes
- The spike methodology in this repository demonstrates that structured scratchpads (tasks.md tracking pending/active/completed) enable reliable multi-session task management

### Tradeoffs

- Extra tool calls for read/write operations (latency)
- Scratchpad must be small enough to fit in context when loaded
- Risk of stale or outdated scratchpad content if not actively maintained
- Agent must be instructed to use the scratchpad consistently

### State of the Art

- **Devin**: Plan file as living scratchpad with persistent VM state
- **Claude Code (this repo's methodology)**: tasks.md + log.md as structured working memory
- **OpenSearch**: Dedicated scratchpad tools
- **Cursor/Windsurf**: scratchpad.md pattern for plan tracking

### Applicability to Claude Code

Directly applicable and already partially implemented in this repository's workflow. Key patterns:
1. **Structured task tracking files** (tasks.md) as working memory for multi-step operations
2. **Log files** (log.md) as append-only memory for decisions and discoveries
3. **The principle**: always write findings to files immediately, never accumulate knowledge only in the conversation

---

## 5. Context Compression and Summarization

### How It Works

Context compression reduces the token count of existing context while preserving the information needed for the current task. Three main approaches:

### LLM Summarization

An LLM rewrites conversation history into condensed natural language summaries.
- **Recursive summarization**: Summarize small segments, then recursively summarize the summaries. Enables long-term dialogue memory.
- **Contextual summarization**: Periodically compress older messages while keeping recent ones verbatim (e.g., summarize everything older than 20 messages, keep last 10 intact).
- **Hierarchical multi-level**: Different memory at different time scales — working memory for current session, episodic memory for important past interactions, semantic memory for general knowledge.

### Observation Masking (JetBrains Research)

Replace old tool outputs with placeholders while keeping tool calls visible. The agent remembers what actions it took without carrying the full output.
- On SWE-bench: **matched LLM summarization quality while using less compute**
- Key insight: **the reasoning trace matters more than the raw data the tools returned**
- An agent typically needs "I searched for X and found it in file Y" rather than the full 500-line file content

### ACON: Learned Compression Guidelines

Agent Context Optimization framework (Kang et al., 2025):
- Given paired trajectories (full context succeeds, compressed context fails), LLMs analyze failure causes
- Compression guidelines optimized in natural language space (gradient-free)
- Applicable to any LLM including API-based closed-source models
- Results: **26-54% memory reduction** while largely preserving task performance
- Distillation: preserves **>95% accuracy** when compressed into smaller models

### Claude's Compaction System

**Auto-compaction** (Claude Code): Triggers at ~83.5% window capacity. Summarizes conversation history, discards verbose tool outputs, preserves critical information.

**Server-side compaction** (API, beta): Available on Opus 4.6 and Sonnet 4.6. Automatically summarizes when approaching configured threshold. Recommended over SDK compaction for less integration complexity.

**Context editing** (API, beta):
- Tool result clearing (`clear_tool_uses_20250919`): Clears tool results when context exceeds threshold
- Thinking block clearing (`clear_thinking_20251015`): Manages thinking blocks, with cache invalidation at clearing point

### Evidence of Effectiveness

- JetBrains: observation masking matches summarization quality at lower compute
- ACON: 26-54% memory reduction with preserved performance
- Claude Code: auto-compaction enables sessions far beyond nominal context limit
- Recursive summarization: enables consistent responses in long-context conversations

### Tradeoffs

| Approach | Compression Ratio | Quality Loss | Compute Cost | Deterministic? |
|----------|-------------------|--------------|-------------|----------------|
| LLM Summarization | High | Moderate (lossy) | High (LLM inference) | No |
| Observation Masking | Moderate | Low | Minimal | Yes |
| Verbatim Compaction | Lower | Minimal | Minimal | Yes |
| ACON | 26-54% | Low | Initial optimization cost, then low | Varies |

### State of the Art

- **Claude Code**: Auto-compaction + server-side compaction API + context editing
- **JetBrains Junie**: Observation masking (matched summarization at lower cost)
- **ACON**: Gradient-free, learned compression guidelines
- **Morph Compact**: 98% verbatim accuracy

### Applicability to Claude Code

Already implemented in Claude Code's core architecture. Key insights for users:
1. **Proactive `/compact` at 70% capacity** rather than waiting for auto-compaction
2. **Write important findings to files before context is compressed** — files survive compaction
3. **Observation masking principle**: when building tools, return compact results by default
4. **Use subagents for context-heavy operations** to keep the main context clean

---

## 6. Codebase Understanding and Mapping

### How It Works

Coding agents need to understand large codebases without loading everything into context. The key technique is building a **structured map** of the codebase that fits in a small token budget.

### Aider's Repo Map (Tree-Sitter + PageRank)

The most well-documented and influential approach:

1. **Tree-sitter parsing**: Parse every source file into an AST. Extract definitions (functions, classes, variables, types) and references (where those symbols are used).

2. **Dependency graph construction**: Build a NetworkX MultiDiGraph. Nodes = source files. Edges = dependencies between files (based on symbol definitions and references).

3. **PageRank ranking**: Rank files using PageRank with personalization toward files the user is actively editing. Files referenced by many other files rank higher — a function called by 20 others is more valuable context than a private helper called once.

4. **Token-budgeted output**: Binary search to find the maximum number of tags fitting within `--map-tokens` budget (default: 1,024 tokens). Output within 15% of target.

Result: A ~1K token "bird's eye view" of an entire repository, highlighting the most structurally important symbols.

### Cursor's Codebase Indexing

Full semantic search pipeline:

1. **AST-based chunking**: Split files into semantic units (~500 token blocks). Traverse AST depth-first, split into sub-trees within token limits, merge sibling nodes.

2. **Custom embedding model**: Trained on real agent sessions for code-specific understanding (outperforms generic embeddings).

3. **Vector storage**: Turbopuffer vector DB optimized for fast search across millions of chunks.

4. **Semantic search**: Query by meaning ("where is authentication handled") finds relevant code even without keyword matches.

### Other Approaches

- **LSP integration**: Language Server Protocol provides symbol resolution, go-to-definition, find-references — what IDEs use for human navigation
- **Dependency graphs**: Package-level dependency analysis for understanding module relationships
- **Call graphs**: Function-level call relationships for understanding execution flow

### Evidence of Effectiveness

- Aider: Repo map dramatically improves the model's ability to make correct edits across large codebases
- Cursor: Custom code embeddings + AST chunking → 12.5% accuracy improvement
- The fundamental insight: giving the model structural understanding of the codebase prevents it from making edits that break dependencies or miss related code

### Tradeoffs

| Approach | Token Cost | Setup Cost | Freshness | Accuracy |
|----------|-----------|-----------|-----------|----------|
| Repo map (Aider) | ~1K tokens | Parsing time | Real-time (re-parses) | Structural only |
| Semantic search (Cursor) | Variable | Indexing time | Needs re-indexing | Semantic understanding |
| LSP | Per-query | Server setup | Real-time | Precise |
| grep/glob | Per-query | None | Real-time | Exact text only |

### State of the Art

- **Aider**: Tree-sitter + PageRank repo map (open source, well-documented)
- **Cursor**: AST chunking + custom embeddings + Turbopuffer
- **RepoMapper MCP**: Standalone repo mapping tool based on Aider's approach, available as MCP server

### Applicability to Claude Code

High opportunity area. Claude Code currently relies on grep/glob/read tools for codebase navigation — effective but not structurally informed. Potential improvements:
1. **Tree-sitter repo map** (Aider-style) could be provided as initial context in CLAUDE.md or via a tool
2. **Semantic search MCP** could enable meaning-based code retrieval
3. **Dependency-aware navigation** would help Claude understand which files are affected by a change

---

## 7. Session Persistence and Resumption

### How It Works

Session persistence allows agents to save their state, shut down, and resume later from where they left off. This is critical for long-running tasks, error recovery, and human-in-the-loop workflows.

### LangGraph Checkpointing

The most mature framework-level implementation:

- **Checkpoint at every super-step**: A snapshot of the graph state is saved at each step boundary
- **Thread-based organization**: Each conversation/task gets a unique thread_id
- **State serialization**: Encoding/decoding protocol with optional encryption and compression
- **Fault-tolerant resumption**: If a node fails, completed nodes' writes are preserved. Resume doesn't re-run successful nodes.
- **Storage backends**: PostgreSQL, DynamoDB, S3, Snowflake, file-based
- **Small checkpoints** (<350 KB): stored directly in DynamoDB
- **Large checkpoints** (>=350 KB): state uploaded to S3 with DynamoDB pointer

### Microsoft Agent Framework

Built-in checkpointing and resuming for workflows:
- FileCheckpointStorage for local persistence
- Session serialization/deserialization for conversation state
- Pause and resume for long-running processes
- Addresses AutoGen's lack of built-in checkpointing

### Devin's Persistent VM Approach

Full environment persistence, not just conversation state:
- Each session runs in an isolated VM
- VM state persists across pauses/resumes
- Running to-do list tracks multi-day task progress
- Bidirectional file sync with <50ms latency

### Claude Code's Approach: File-Based State

Claude Code does not have formal checkpointing. Instead, it relies on **file-based state reconstruction**:
- CLAUDE.md and MEMORY.md provide persistent instructions and memory
- The user's codebase (git state) is persistent
- The spike methodology (tasks.md, log.md, research.md) provides structured state that survives session boundaries
- **Resumption protocol**: "Re-read tasks.md, log.md, and research.md to restore working state — they are the source of truth, not your memory of the conversation"

### Context Serialization Protocol

A structured handoff message pattern:
1. What was in progress
2. What was decided and why
3. What needs attention next
4. What can safely wait
5. Who the agent is waiting on

### Evidence of Effectiveness

- LangGraph: Enables human-in-the-loop, time-travel debugging, fault-tolerant execution
- Devin: Persistent VMs enable multi-day tasks impossible with context-only approaches
- This repository's spike methodology: Successfully enables multi-session, multi-agent research across context boundaries

### Tradeoffs

| Approach | Complexity | Fidelity | Infrastructure | Human Readability |
|----------|-----------|----------|----------------|-------------------|
| Framework checkpointing (LangGraph) | Medium | High (exact state) | Database required | Low |
| Persistent VM (Devin) | High | Complete | Cloud infrastructure | Low |
| File-based state (Claude Code) | Low | Approximate | None (filesystem) | High |
| Context serialization | Low | Moderate | None | High |

### Applicability to Claude Code

The file-based approach is already the primary mechanism for Claude Code session persistence. This repository's methodology is a production example. Key principles:
1. **Write state to files immediately** — don't defer
2. **Structured formats** (tasks.md with status tracking) enable reliable resumption
3. **Append-only logs** (log.md) prevent information loss
4. **Resumption instructions** in CLAUDE.md tell the agent how to restore context

---

## 8. Knowledge Graphs and Structured Memory

### How It Works

Instead of storing memory as flat text chunks, knowledge graphs represent information as entities (nodes) and relationships (edges). This enables relationship-aware retrieval that flat vector search cannot achieve.

### GraphRAG

Leverages semantic relationships between entities:
- Maps entities and relationships into a knowledge graph
- Enables AI to find contextually relevant information through ontologies
- Solves multi-hop reasoning: if A connects to B and B connects to C, a knowledge graph finds the A→C relationship that vector search would miss

### Zep/Graphiti: Temporal Knowledge Graph

The most advanced agent memory implementation:
- **Bi-temporal model**: Tracks when an event occurred AND when it was ingested
- **Validity windows**: Every edge includes explicit start/end times for facts
- **Conflict resolution**: Uses temporal metadata to update/invalidate (but not discard) outdated information
- **Incremental updates**: No batch recomputation needed — integrates updates in real-time
- **Performance**: 94.8% on DMR benchmark; up to 18.5% accuracy improvement on LongMemEval; 90% latency reduction

### Vector DB vs. Knowledge Graph Comparison

| Dimension | Vector Database | Knowledge Graph |
|-----------|----------------|-----------------|
| Best for | Unstructured text, semantic similarity | Structured data, multi-hop relationships |
| Query type | "Find similar content" | "Find connected entities" |
| Multi-hop reasoning | Weak (single-hop similarity) | Strong (follows edges) |
| Setup complexity | Low (easy to spin up) | High (entity extraction, ontology design) |
| Explainability | Opaque similarity scores | Clear, auditable node-edge paths |
| Schema evolution | Flexible | Rigid (ontology must be maintained) |
| Speed | Millisecond retrieval | Depends on query complexity |

### Hybrid Approach (Emerging Best Practice)

Modern systems combine both:
- Vector database for initial retrieval (broad semantic search)
- Knowledge graph for relationship-aware refinement
- Mem0's enhanced variant uses graph-based representations alongside vector retrieval

### Evidence of Effectiveness

- Zep: 94.8% DMR, 18.5% accuracy improvement, 90% latency reduction
- GraphRAG handles multi-hop queries that vector search fails on
- For structured domains (code dependencies, legal, medical), knowledge graphs consistently outperform flat retrieval

### Tradeoffs

- Entity extraction pipelines are complex and error-prone
- Ontology design and maintenance is rigid and labor-intensive
- Overkill for simple retrieval tasks
- Incremental updates (Graphiti) reduce but don't eliminate maintenance burden

### State of the Art

- **Zep/Graphiti**: Temporal knowledge graph for agent memory
- **Microsoft GraphRAG**: Global summarization over knowledge graphs
- **Neo4j**: Graph database with RAG integration
- **Mem0**: Hybrid vector + graph memory

### Applicability to Claude Code

Moderate applicability. Code has inherent graph structure (call graphs, dependency trees, import relationships) that would benefit from graph-based retrieval. However:
- The infrastructure complexity may not be justified for a CLI tool
- Aider's repo map (tree-sitter + PageRank) achieves similar structural understanding with less overhead
- Knowledge graphs would be most valuable for very large, complex codebases where flat file search fails

---

## 9. CLAUDE.md and Instruction File Patterns

### How It Works

CLAUDE.md is a markdown file loaded into Claude Code's system prompt, providing project-specific instructions, conventions, and persistent context. It is the primary mechanism for long-term knowledge persistence across sessions.

### File Hierarchy

**Discovery**: Claude Code walks up the directory tree from the working directory:
- Parent directory CLAUDE.md files: loaded in full at launch
- Subdirectory CLAUDE.md files: loaded on demand when interacting with those files

**Memory levels**:
1. **User level** (`~/.claude/CLAUDE.md`): Global preferences
2. **Project level** (`./CLAUDE.md`): Project-specific instructions (most common)
3. **Subdirectory level**: Per-package or per-module instructions
4. **.claude/rules/**: All markdown files auto-loaded (no imports needed)
5. **Auto-memory** (MEMORY.md): Claude's self-written notes, first 200 lines loaded

**Import system**: `@path/to/file.md` syntax, resolved recursively up to 5 levels.

### What Makes a Good CLAUDE.md

**Content**: Common bash commands, code style guidelines, key architectural patterns, testing instructions, project-specific terminology.

**Size**: Under 200 lines per file. Use .claude/rules/ for modular organization.

**Instruction compliance patterns**:
1. **Positional priority**: Most-violated rules at top (first 5 lines) and bottom (last 5 lines) — exploits "lost in the middle" phenomenon
2. **Positive framing**: "Use ES modules" not "Don't use CommonJS" — cuts violations by ~50%
3. **Specificity**: "Run `npm test` before committing" not "Make sure tests pass"
4. **Living document**: Evolves over time based on what Claude gets wrong

### Broader Pattern: Instruction Files Across Tools

Claude Code is not unique — similar patterns exist:
- **Cursor**: `.cursorrules` file for project-specific instructions
- **Windsurf**: `.windsurfrules` + `scratchpad.md`
- **Junie (JetBrains)**: Guidelines files for project conventions
- **OpenAI Codex**: Setup scripts + memory with workspace-scoped writes

The pattern of a **project-root instruction file** that persists across sessions has become standard across all major coding agents.

### Evidence of Effectiveness

- "Most successful users obsess over context management through CLAUDE.md files" (Claude Code docs)
- Positive framing cuts rule violations by roughly half
- Positional priority exploits well-documented attention bias
- The three most impactful practices: CLAUDE.md configuration, structured prompts, plan mode

### Applicability to Claude Code

This IS Claude Code's primary memory mechanism. Key best practices for users:
1. Keep CLAUDE.md focused and under 200 lines
2. Use .claude/rules/ for modular organization
3. Put most critical rules at top and bottom
4. Frame rules positively
5. Pair with auto-memory for self-improving instructions
6. Use structured CLAUDE.md (like this repository's) for complex workflows

---

## 10. Multi-Session Learning

### How It Works

Multi-session learning enables agents to improve across conversations by persisting and incorporating feedback from past interactions.

### Claude Code's Auto-Memory

The primary multi-session learning mechanism:
- Claude saves notes as it works: build commands, debugging insights, architecture notes, preferences
- Stored in MEMORY.md (plain markdown, human-editable)
- First 200 lines loaded into every new session
- Two complementary systems:
  - CLAUDE.md: human-written persistent instructions
  - Auto-memory: machine-written learnings from corrections and preferences

### Mem0: Structured Memory Extraction

Automatically extracts and consolidates information from conversations:
- User memory persists across all sessions with a specific user
- Example: "prefers morning study sessions" stays available in every future session
- Graph-enhanced variant captures relational structures

### RLHF and Preference Learning (Model Level)

At the model training level:
- RLHF incorporates human preferences into model behavior
- Models like Claude undergo multiple iterative training rounds
- This is not per-user but per-model-version improvement

### Letta's Self-Improving Agents

Agents with explicit memory management tools can improve their own behavior:
- Store successful strategies in archival memory
- Retrieve and apply past solutions to similar problems
- Update core memory blocks based on user corrections

### The Learning Gap

Current agent multi-session "learning" is primarily **memory-based** (storing and retrieving facts) rather than **skill-based** (improving reasoning or strategies). True multi-session learning — where an agent gets better at a type of task through practice — requires metacognitive abilities that current systems lack. The OpenReview paper "Truly Self-Improving Agents Require Intrinsic Metacognitive Learning" argues this is a fundamental missing capability.

### Evidence of Effectiveness

- Claude Code auto-memory: Reduces need to repeat instructions across sessions
- Mem0: 26% accuracy improvement through persistent memory
- File-based memory (this repo): Enables multi-session research workflows with full context preservation

### Tradeoffs

- Auto-memory can accumulate stale or incorrect information
- Human curation needed to prevent memory drift
- True learning (skill improvement) vs. memory (fact storage) is an unsolved problem
- Privacy concerns with persistent preference data

### State of the Art

- **Claude Code**: Auto-memory (MEMORY.md) + human-curated CLAUDE.md
- **Mem0**: Universal memory layer with auto-extraction
- **Letta**: Self-editing memory with agent autonomy
- **OpenAI Codex**: Memory with workspace-scoped writes and guardrails against stale facts

### Applicability to Claude Code

Already implemented via auto-memory. Improvement opportunities:
1. **Structured feedback loops**: Explicit "that worked well" / "that was wrong" signals
2. **Pattern recognition**: Auto-detect recurring correction patterns and update CLAUDE.md
3. **Memory curation tools**: Better visibility and management of what Claude has "learned"
4. **Cross-project learning**: Share learnings between similar projects (currently machine-local)

---

## Cross-Cutting Themes

### The Context Engineering Framework

Lance Martin's (LangChain) framework provides the best organizing principle for all these techniques:

1. **Write**: Save information outside the context window (scratchpads, files, databases)
2. **Select**: Pull the right information into context (RAG, search, repo maps)
3. **Compress**: Retain only needed tokens (summarization, masking, clearing)
4. **Isolate**: Split context into separate spaces (subagents, fresh context per task)

Every technique in this report maps to one or more of these strategies.

### The Fundamental Tradeoff: Simplicity vs. Sophistication

The research consistently shows that **simpler approaches often perform surprisingly well**:
- Letta's own benchmarks: "Is a filesystem all you need?"
- JetBrains: Observation masking matched LLM summarization at lower cost
- Aider: A ~1K token repo map enables effective codebase navigation
- This repository: Structured markdown files enable multi-session, multi-agent research

More sophisticated approaches (knowledge graphs, vector databases, self-editing memory) have clear advantages for specific use cases but add complexity that may not be justified for all scenarios.

### Context Degradation is the Primary Failure Mode

Multiple sources converge on this: **the most successful users obsess over context management**. Context rot, positional bias, and accumulated noise are the main reasons agents fail on complex tasks — not raw capability limitations.

---

## Conclusions and Recommendations for Claude Code

### Already Well-Implemented
1. **Auto-compaction** and server-side compaction
2. **Subagent context isolation**
3. **CLAUDE.md hierarchy** with .claude/rules/ and import support
4. **Auto-memory** (MEMORY.md)
5. **File-based tools** (grep, glob, read) for codebase search

### High-Impact Opportunities
1. **Tree-sitter repo map** (Aider-style): Provide structural codebase understanding in ~1K tokens. This is probably the single highest-impact addition possible.
2. **Observation masking for tool results**: JetBrains showed this matches summarization quality at lower cost. Replace stale tool outputs with placeholders rather than full summarization.
3. **Structured scratchpad patterns**: Formalize the tasks.md/log.md pattern as a first-class feature, not just a user convention.
4. **Positional awareness in CLAUDE.md loading**: Place highest-priority instructions at start and end of system prompt.

### Medium-Impact Opportunities
5. **Semantic code search** via embeddings (Cursor's approach, potentially via MCP)
6. **Explicit memory management tools** (Letta-style "remember this" / "recall that")
7. **Session handoff protocol**: Structured context serialization for long-running tasks
8. **Aggressive `/clear` prompting**: Guide users to clear context more frequently

### Lower Priority (Complex to Implement)
9. **Knowledge graph integration** for complex codebases
10. **Custom code embedding models** trained on agent sessions
11. **ACON-style learned compression** guidelines per domain
12. **Cross-project memory sharing**
