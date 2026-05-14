<!-- Source: https://developer.microsoft.com/blog/protecting-against-indirect-injection-attacks-mcp -->
<!-- Retrieved: 2026-05-14 -->

# Protecting Against Indirect Prompt Injection Attacks in MCP — Microsoft

## Key Vulnerability Overview

The blog post defines **Indirect Prompt Injection** (also called cross-domain prompt injection or XPIA) as "a security exploit targeting generative AI systems where malicious instructions are embedded in external content." A specific variant, **Tool Poisoning**, occurs when attackers embed malicious instructions in MCP tool descriptions.

## Two Primary Defense Approaches

### 1. AI Prompt Shields

Microsoft recommends implementing Prompt Shields through:

- **Detection and Filtering**: Uses machine learning to identify harmful instructions in external content
- **Spotlighting**: Transforms input text to help models distinguish trusted system instructions from untrusted external data
- **Delimiters and Datamarking**: Employs special markers to highlight boundaries between trusted and untrusted data sources
- **Continuous Monitoring**: Keeps defenses current against evolving attack patterns

### 2. Supply Chain Security

The post emphasizes traditional security principles applied to AI components:
- Verify all dependencies before integration (models, packages, embeddings services)
- Use tools like GitHub Advanced Security for scanning vulnerabilities
- Maintain secure deployment pipelines with continuous monitoring

## Critical Foundation

The authors stress that "98% of reported breaches would be prevented by robust security hygiene," including multi-factor authentication, least privilege access, and keeping systems updated. They position AI security within broader organizational security practices rather than as an isolated concern.
