# ProtectAI DeBERTa-v3-base Prompt Injection v2
- **Source**: https://huggingface.co/protectai/deberta-v3-base-prompt-injection-v2
- **Retrieved**: 2026-05-14

## Model Training
- Fine-tuned from Microsoft's DeBERTa-v3-base
- Over 20 configurations tested to optimize detection
- Focused on hyperparameters, training regimens, and dataset compositions
- Purpose: Detect and classify prompt injection attacks in language model inputs

## Datasets Used
Training dataset assembled from multiple public sources:
- Apache License 2.0: 5 datasets (chatbot_instruction_prompts, grok-conversation-harmless, Salad-Data, jailbreak-classification)
- MIT License: 8 datasets
- CC-BY-4.0: 1 dataset (xstest-v2-copy)
- CC-BY-3.0: 1 dataset (open-instruct)
- CC0 1.0 Universal: 1 dataset
- No License: 6 datasets

Sources included academic research, security competitions, and LLM Guard community feedback.

## Performance Metrics

**Training Dataset Results:**
- Loss: 0.0036, Accuracy: 99.93%, Precision: 99.92%, Recall: 99.94%, F1: 99.93%

**Post-Training Evaluation (20,000 untrained prompts):**
- Accuracy: 95.25%, Precision: 91.59%, Recall: 99.74%, F1: 95.49%

## Limitations

1. **English-only**: Doesn't handle non-English prompts
2. **Jailbreak attacks**: Does not detect advanced jailbreak attacks
3. **System prompts**: Not recommended for system prompt scanning due to false positives
4. **Limited scope**: Focuses solely on prompt injections, not other security threats

## Production Usage

Binary classification output:
- `0` = Benign (safe)
- `1` = Injection detected (suspicious)

ONNX-optimized deployment available for production. Max length: 512 tokens.

## Known Failure Modes

1. System prompt scanning: High false positive rate
2. Non-English text: Unreliable results
3. Sophisticated jailbreaks: May bypass detection
4. High recall, potential false positives: 99.74% recall means some benign inputs flagged

## Integration Options
- Langchain (official docs available)
- LLM Guard (integrated as input scanner)
- 18 community spaces already in production
