# Protecting Against Indirect Prompt Injection Attacks in MCP

- **Source**: https://developer.microsoft.com/blog/protecting-against-indirect-injection-attacks-mcp
- **Retrieved**: 2026-05-14

## Defense Techniques

The blog identifies two primary mitigation approaches:

**1. AI Prompt Shields**
Microsoft's solution employs multiple protective mechanisms:
- "Detection and Filtering" using machine learning and natural language processing to identify malicious instructions in external content
- "Spotlighting" transforms input text to make it more distinguishable from system instructions
- "Delimiters and Datamarking" using special markers to separate trusted from untrusted data sources
- Continuous monitoring and updates against emerging threats

**2. Supply Chain Security**
- Verify all components before integration, including foundation models and embedding services
- Implement secure deployment pipelines with continuous monitoring
- Use GitHub Advanced Security features (secret, dependency, and CodeQL scanning)

## Content Sanitization & Structural Approaches

The article emphasizes delimiter strategies: "Including delimiters in the system message explicitly outlines the location of the input text, helping the AI system recognize and separate user inputs from potentially harmful external content."

Datamarking extends this by highlighting boundaries between data sources.

## Specific Attack Vector: Tool Poisoning

Tool metadata becomes vulnerable when descriptions contain embedded malicious instructions. The blog warns about "rug pull" scenarios where tool definitions change after initial user approval, potentially enabling "data exfiltration or manipulation."

## Security Fundamentals

The Microsoft Digital Defence Report indicates "98% of reported breaches would be prevented by robust security hygiene" through MFA, least privilege access, and timely updates—foundational practices that apply to AI systems.

## Limitations

The article provides conceptual guidance rather than code examples or detailed architectural implementations.
