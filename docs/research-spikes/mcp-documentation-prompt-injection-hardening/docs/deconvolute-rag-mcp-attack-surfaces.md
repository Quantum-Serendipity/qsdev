<!-- Source: https://deconvoluteai.com/blog/attack-surfaces-rag -->
<!-- Retrieved: 2026-05-14 -->

# AI Security - The Hidden Attack Surfaces of RAG and MCP

## Core Attack Surfaces

**Front Door (Synchronous Attacks)**
Direct Prompt Injection occurs when attackers craft malicious user input to override developer instructions. These attacks are transient, affecting single interactions, and require real-time attacker presence. Example: instructing a system to "ignore retrieved context" and answer from general knowledge instead.

**Back Door (Asynchronous Attacks)**
Malicious content is planted in knowledge bases ahead of time, remaining dormant until retrieval activates it. The article notes: "a single poisoned document can influence responses for many users without further attacker involvement."

## Attack Methodologies

**Indirect Prompt Injection**
Malicious instructions embedded in documents reach the language model when retrieved as context, since models cannot reliably distinguish instruction origins.

**Corpus Poisoning**
Attackers optimize document text using gradient-based methods to manipulate which items the retriever surfaces. These "Vector Magnets" exploit dense vector embeddings' continuous nature, positioning malicious documents near target query locations in semantic space.

**Confused Deputy Problem (MCP Context)**
When agents access external tools, retrieved poisoned content can influence tool selection and execution, escalating from text manipulation to unauthorized actions.

## Defense Recommendations

- Treat retrieved content as untrusted until inspection
- Evaluate systems under adversarial conditions across all attack surfaces
- Measure both attack success rates and legitimate performance
- Inspect retrieved data before it reaches models or influences tool execution

The fundamental security principle: "securing modern RAG and MCP-based Agents requires layered defenses across ingestion, retrieval, generation, and tool execution."
