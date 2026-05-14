# LLM Guard - Prompt Injection Scanner Details
- **Source**: https://protectai.github.io/llm-guard/input_scanners/prompt_injection/
- **Retrieved**: 2026-05-14

## Models Used
The system employs "ProtectAI/deberta-v3-base-prompt-injection-v2," which is a fine-tuned variant of Microsoft's DeBERTa v3 base model trained on multiple prompt injection datasets.

## Detection Thresholds
The scanner operates with a configurable threshold parameter, with a default value of 0.5. The classification system uses binary outputs: `0` indicates no injection detected, while `1` signals potential injection.

## Scanner Configuration
Implementation involves importing the PromptInjection class and specifying parameters including threshold and match_type options. The documentation notes that "Switching the match type might help with improving the accuracy, especially for longer prompts."

## Performance Characteristics
Testing across multiple AWS and Azure instances shows variable latency results:
- CPU-based execution ranges from 81-421ms average latency
- ONNX optimization significantly improves speeds, achieving sub-8ms latency on GPU instances
- Throughput varies from 911 to 50,216 queries per second depending on hardware and optimization

## Known Limitations
The developers explicitly state: "We don't recommend using this scanner for system prompts. It's designed to work with user inputs." The model is also noted as still undergoing testing phases.
