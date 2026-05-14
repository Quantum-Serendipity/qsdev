# DSPy MIPROv2 Optimizer and Automated Prompt Optimization

- **Source URLs**:
  - https://dspy.ai/learn/optimization/optimizers/
  - https://dspy.ai/api/optimizers/MIPROv2/
  - https://github.com/stanfordnlp/dspy
- **Retrieved**: 2026-03-15
- **Note**: Compiled from web search results across multiple DSPy documentation pages.

---

## DSPy Framework Overview
DSPy is a framework for programming (not prompting) language models. It allows iterating fast on building modular AI systems and offers algorithms for optimizing their prompts and weights. Key paradigm shift: systematic, programmatic approach to prompt creation without manually crafting complex prompts.

## MIPROv2 (Multiprompt Instruction PRoposal Optimizer v2)

### How It Works
MIPROv2 optimizes both instructions and few-shot examples jointly via three phases:

#### Phase 1: Bootstrap Few-Shot Examples
- Randomly samples from training set
- Runs through LM program
- Keeps examples where output is correct
- Creates `num_candidates` sets of bootstrapped examples

#### Phase 2: Propose Instruction Candidates
Instruction proposer includes:
1. Generated summary of training dataset properties
2. Generated summary of LM program code and specific predictor
3. Previously bootstrapped few-shot examples as reference
4. Randomly sampled tip for generation

Instruction generation is **data-aware** and **demonstration-aware**.

#### Phase 3: Bayesian Optimization Search
- Uses Bayesian Optimization to choose best combinations
- Runs `num_trials` trials evaluating new prompt sets over validation data
- Can optimize few-shot + instructions jointly, or just instructions for 0-shot

## OPRO (Optimization by PROmpting) — Google DeepMind

### Mechanism
- Starts with meta-prompt containing task description + previous solutions with scores
- LLM generates new candidate solutions each step
- Candidates evaluated and added to prompt for next step
- Iterative improvement without gradients or parameters

### Results
- Outperforms human-designed prompts by up to 8% on GSM8K
- Up to 50% improvement on Big-Bench Hard tasks
- Optimized prompts transfer to other benchmarks in same domain

### Limitations
- Requires capable LLMs — small-scale models (LLaMa-2 family, Mistral 7B) lack self-optimization ability
- Only effective with frontier models

## EvoPrompt — Evolutionary Prompt Optimization

### Mechanism
- Borrows evolutionary algorithms (EAs) for discrete prompt optimization
- Starts from population of prompts
- Iteratively generates new prompts using LLM + evolutionary operators
- Population improved based on development set evaluation

### Results
- Up to 25% improvement on BBH over human-engineered prompts
- Tested on 31 datasets (language understanding, generation, BBH)
- Extended to vision-language models in 2025

## Promptomatix (2025)
- Automatic prompt optimization framework
- Supports lightweight meta-prompt optimizer AND DSPy-powered compiler
- Analyzes user intent, generates synthetic training data
- Refines prompts using cost-aware objectives
