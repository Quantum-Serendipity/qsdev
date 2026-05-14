# Program-Aided Language Models (PAL) and Code-Augmented Reasoning
- **Sources**:
  - https://arxiv.org/abs/2211.10435
  - https://reasonwithpal.com/
  - https://learnprompting.org/docs/agents/pal
- **Retrieved**: 2026-03-15
- **Note**: Synthesized from search results

## Core Approach

PAL enhances problem-solving by generating code to represent intermediate reasoning steps, in contrast with CoT prompting which uses natural language. PAL writes code to solve a given question and sends it to a programmatic runtime to retrieve the result.

While LLMs are adept at step-by-step decomposition, they often make logical and arithmetic mistakes in the solution part, even when the problem is decomposed correctly. PAL addresses this by using the LLM to read natural language problems and generate programs as intermediate reasoning steps, then offloads the solution step to a Python interpreter.

## Program of Thoughts (PoT)

A related approach: code-augmented prompting that generates hybrid rationales containing both natural language statements and code. The code portion is offloaded to an interpreter for evaluation, decoupling reasoning and computation.

## Performance

- PAL outperforms CoT by 11% on three BIG-Bench Hard reasoning tasks
- On GSM-hard, PAL outperforms CoT by an absolute 40%
- Generating code + Python interpreter leads to more accurate results than much larger models

## Tool-Integrated Reasoning (TIR)

The broader category: LLMs leveraging external tools (search engines, calculators, code interpreters) to enhance reasoning rather than relying solely on parametric knowledge. This represents a fundamental shift in how reasoning accuracy is achieved.
