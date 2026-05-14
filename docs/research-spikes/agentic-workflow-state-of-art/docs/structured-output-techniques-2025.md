# Structured Output Techniques for LLMs (2025)

- **Source URLs**:
  - https://platform.claude.com/docs/en/build-with-claude/structured-outputs
  - https://medium.com/@emrekaratas-ai/structured-output-generation-in-llms-json-schema-and-grammar-based-decoding-6a5c58b698a6
  - https://github.com/guidance-ai/llguidance
  - https://docs.vllm.ai/en/v0.8.4/features/structured_outputs.html
- **Retrieved**: 2026-03-15
- **Note**: Compiled from web search results on structured output approaches.

---

## Two Main Approaches

### 1. JSON Schema Enforcement (API-Level)
- Specify JSON Schema with prompt
- Model/intermediate layer ensures conformance
- OpenAI: function calling + structured outputs
- Claude: `output_config.format` for JSON, `strict: true` for tool use
- Compiles schema into grammar, constrains token generation during inference

### 2. Grammar-Guided Generation (Decoding-Level)
- Constrains decoding using formal grammar rules
- Masks invalid tokens during generation
- Only tokens complying with constraints remain candidates for sampling
- Frameworks: Guidance, Outlines, XGrammar, llama.cpp grammar module

## Claude-Specific Structured Output

### Available Features
1. **JSON outputs** (`output_config.format`): Get response in specific JSON format
2. **Strict tool use** (`strict: true`): Guarantee schema validation on tool names/inputs

### Performance Characteristics
- First request latency: Additional time for grammar compilation
- Compiled grammars cached 24 hours
- More complex schemas = larger grammars = longer compilation
- May slightly reduce output quality vs unconstrained (research debate ongoing)

### Without Structured Outputs
Claude can generate malformed JSON or invalid tool inputs. Even careful prompting has schema violations requiring error handling and retries.

## State of the Art (2025)

### Near-Zero Overhead
Open-source engines XGrammar and llguidance achieved near-zero overhead constrained decoding, enabling structured output at production scale. In May 2025, OpenAI credited llguidance for foundational work.

### vLLM Support
vLLM 0.8.5+ supports wide range of output constraints, from simple choice lists to full JSON schemas, with minimal overhead.

### Quality Concern
Some research suggests constrained decoding may slightly harm model performance and quality. The constraint can prevent the model from generating optimal intermediate reasoning tokens.

## Practical Recommendations for Claude Code
1. Use structured outputs for any machine-consumed output (tool results, parsed data)
2. For simple formatting, XML tags in prompts often sufficient and lower overhead
3. For complex schemas, prefer tool calling with `strict: true`
4. Test structured vs unconstrained output quality for your specific use case
5. Consider retries with validation as alternative to constrained decoding
