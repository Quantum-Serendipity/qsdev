# Defense Against Indirect Prompt Injection via Tool Result Parsing

- **Source**: https://arxiv.org/html/2601.04795
- **Retrieved**: 2026-05-14

## Proposed Defense Mechanism

The paper introduces a novel prompt-based defense centered on two modules: **ParseData** and **CheckTool**. Rather than detecting injections directly, the approach extracts only necessary data from tool outputs while filtering malicious content through format validation and logical constraints.

## Core Observations

The defense relies on three key insights:

1. Tool results contain excessive data beyond what agents actually need, providing space for injection payloads
2. Legitimate data conforms to specific formats (email addresses, dates, numerical ranges)
3. Injection instructions cannot satisfy these strict formatting and logical requirements

## Implementation Approach

**ParseData Module**: Prompts the LLM to anticipate expected data structure before tool execution, then uses those specifications to parse results. The LLM identifies what data format and constraints should apply, then extracts only conforming information. A variant, ParseFull, incorporates full conversation history for enhanced context.

**CheckTool Module**: Handles scenarios requiring large text chunks. It monitors whether tool outputs trigger subsequent tool calls. If triggering occurs, content is flagged as potentially malicious, then the LLM sanitizes it by removing segments that caused activation.

**Combinations**: Methods can be stacked -- ParseData+CheckTool or CheckTool+ParseData -- depending on risk tolerance and utility requirements.

## Testing and Models

Evaluation used three LLMs: gpt-oss-120b, llama-3.1-70b, and qwen3-32b, tested against the AgentDojo benchmark across banking, slack, travel, and workspace domains.

## Effectiveness Metrics

Key performance indicators measured:

- **Benign Utility (BU)**: Task completion without attacks
- **Utility under Attack (UA)**: Task completion during attacks
- **Attack Success Rate (ASR)**: Proportion of successful injections
- **Risk**: ASR/UA ratio indicating attack frequency per successful task

## Results

ParseData+CheckTool and CheckTool+ParseData achieved approximately **0.2%-1% Average Risk** compared to Tool Filter's 3%-6%, representing roughly 1/10th the risk. ASRs dropped to 0.1%-0.5% versus 5%+ for competing defenses.

## Limitations

The primary limitation involves **parameter hijacking attacks**, where injections redirect values rather than invoke unauthorized tools. The current approach doesn't address scenarios where attackers manipulate data like email addresses rather than triggering new actions. The authors note their evaluation emphasizes unauthorized tool invocation, leaving parameter-based attacks for future research.
