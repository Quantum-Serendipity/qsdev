<!-- Source: https://arxiv.org/html/2403.14720v1 -->
<!-- Retrieved: 2026-05-14 -->

# Defending Against Indirect Prompt Injection Attacks With Spotlighting

## Overview

This research presents "spotlighting," a family of prompt engineering techniques designed to help large language models (LLMs) distinguish between trusted system instructions and potentially malicious content embedded in external data sources.

## The Problem: Indirect Prompt Injection (XPIA)

The core vulnerability exists because LLMs process concatenated text inputs without distinguishing their sources. Attackers exploit this by embedding adversarial instructions in untrusted data (websites, emails, documents) that the model then treats as legitimate commands. This differs from direct prompt injection attacks because "the user of the LM system is an innocent bystander who is often the victim."

## Three Spotlighting Methods

### 1. Delimiting
Uses special tokens (like `<<` and `>>`) to mark the boundaries of untrusted input text. The system prompt instructs the model to ignore any instructions within these delimiters.

**Results**: Reduced attack success rate (ASR) by approximately 50% for GPT-3.5-Turbo (from ~60% to ~30%), but remained ineffective enough that researchers don't recommend this approach alone.

### 2. Datamarking
Interleaves special characters (like `^`) throughout the input text, replacing whitespace to create a continuous visual marker of external data.

Example transformation: "In this manner" becomes "In^this^manner"

**Results**:
- GPT-3.5-Turbo: ASR reduced from ~50% to below 3%
- Text-Davinci-003: ASR reduced from ~40% to 0.0%
- GPT-4: Consistent strong reduction across tasks

**Impact on NLP tasks**: "Across all benchmarks, the presence of datamarking...does not have any detrimental impact on task performance." Testing included SQuAD Q&A, SuperGLUE Word-In-Context, SuperGLUE BoolQ, and IMDB Sentiment analysis.

### 3. Encoding
Transforms input text using encoding algorithms (base64, ROT13, binary) that LLMs can natively decode.

**Results**:
- Achieves 0.0% ASR across summarization and Q&A tasks with GPT-3.5-Turbo
- Q&A task: 1.8% ASR reduction

**Trade-off**: Performance varies significantly by model:
- **GPT-4**: Maintains strong task performance with encoded text
- **GPT-3.5-Turbo**: Shows "very detrimental impact" on task performance when decoding

**Recommendation**: Only use with high-capacity models like GPT-4.

## Experimental Methodology

**Models tested**: Text-Davinci-003, GPT-3.5-Turbo (June 2023), GPT-4 (June 2023)

**Attack measurement**: Researchers created a synthetic corpus of 1,000 documents containing simple keyword-based payload attacks. Success was measured by whether the model output the attack keyword instead of completing the legitimate task.

**Baseline ASR without defense**:
- GPT-3.5-Turbo: ~60% in summarization tasks
- Text-Davinci-003: ~40% in summarization tasks
- ASR varied significantly by task type

## Key Quantitative Results

| Method | GPT-3.5-Turbo ASR | Text-003 ASR | GPT-4 ASR |
|--------|------------------|-------------|-----------|
| Baseline | ~60% | ~40% | Lower baseline |
| Delimiting only | ~30% | ~25% | - |
| Datamarking | <3% | 0.0% | ~1-8% |
| Encoding | 0.0%-1.8% | 0.0% | 0.0% |

## Downstream Task Performance

- **Datamarking**: Zero detrimental impact on tested benchmarks
- **Encoding (GPT-4)**: No meaningful performance degradation
- **Encoding (GPT-3.5-Turbo)**: Significant performance loss due to decoding errors

## Recommendations

1. **Minimum approach**: Implement datamarking, which provides robust protection with minimal task performance impact
2. **Best case**: Use encoding with GPT-4 or equivalent high-capacity models for maximum security
3. **Avoid**: Delimiting alone, as attackers who discover the system prompt can easily bypass it

## Additional Considerations

**Dynamic marking tokens**: Researchers suggest randomizing or frequently changing the special characters used for marking to prevent attackers who obtain the system prompt from crafting workarounds. With character set size N and k-gram length k, adversaries face 1/N^k probability of guessing the token correctly.

**Adversarial resilience**: Datamarking becomes more robust when marking tokens are interleaved at randomized locations rather than just at whitespace boundaries, preventing attacks that contain no spaces.

## Broader Context

The authors draw an analogy to telecommunications history, comparing the problem to "in-band signaling" challenges where control signals and user data coexist in the same channel. They suggest future work might involve "out-of-band signaling" approaches — using truly separate channels for instructions versus data — though current LLM architectures don't easily support this.

## Limitations

The few-shot learning approach (including attack examples in prompts) showed promise but raises concerns: it generalizes only to known attack patterns and risks "leaking labels" during evaluation. The spotlighting approach is preferred because it targets structural vulnerabilities that should generalize better.
