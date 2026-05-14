# DSPy: Programmatic Prompt Optimization Framework

- **Source URLs**:
  - https://dspy.ai/
  - https://dspy.ai/learn/optimization/optimizers/
  - https://dspy.ai/learn/programming/signatures/
  - https://github.com/stanfordnlp/dspy
  - https://www.ibm.com/think/topics/dspy
- **Retrieved**: 2026-03-15
- **Note**: Content compiled from multiple search results.

## Overview

DSPy (Declarative Self-improving Python) is a framework from Stanford NLP for programming language models rather than prompting them. Allows iterating fast on building modular AI systems, offering algorithms for optimizing prompts and weights. Think of DSPy as a higher-level language for AI programming — like the shift from assembly to C.

## Core Architecture: Three Abstractions

### 1. Signatures
Natural-language typed function declarations. A concise specification describing what a text transformation should achieve, not how a specific LM should be prompted.

Example: `"question -> answer"` or `"context, question -> answer, reasoning"`

Signatures define semantic roles for inputs/outputs with optional types.

### 2. Modules
For every AI component, specify input/output behavior as a signature and select a module to assign a strategy for invoking the LM.

Key modules:
- `dspy.Predict` — basic prediction
- `dspy.ChainOfThought` — step-by-step reasoning
- `dspy.ReAct` — reasoning + acting with tools
- `dspy.ProgramOfThought` — generate and execute code

DSPy expands signatures into prompts and parses typed outputs, enabling composition of different modules into optimizable AI systems.

### 3. Teleprompters/Optimizers
Fine-tune the compiled program for the specific language model. In older versions called "teleprompters."

## How Optimization Works

1. Takes your program and runs it many times across different inputs
2. Collects traces of input/output behavior for each module
3. Filters traces to keep only those in trajectories scored highly by your metric
4. Uses these to generate optimized prompts or few-shot examples

### Key Optimizers

**BootstrapFewShot**: Uses a teacher module to generate complete demonstrations for every stage. Parameters include max_labeled_demos and max_bootstrapped_demos. Validates demonstrations via metric.

**MIPROv2**: Creates both few-shot examples and new instructions for each predictor, then searches over these using Bayesian Optimization to find the best combination.

**SIMBA**: Uses stochastic mini-batch sampling to identify challenging examples, then applies LLM to introspectively analyze failures and generate self-reflective improvement rules.

**GEPA**: Reflective optimizer that adaptively evolves textual components of arbitrary systems.

## Compilation Process

1. Assembles modules and signatures into a graph of LLM calls
2. Executes pipeline on data, collects results
3. Applies optimizers if requested
4. Produces optimized Python code or model weights

## Key Properties

- **Model-agnostic**: Programs optimized for one model transfer to others with minimal degradation
- **Declarative**: Specify what, not how
- **Composable**: Modules combine naturally
- **Optimizable**: Systematic rather than manual prompt engineering

## Strengths
- Eliminates brittle hand-crafted prompts
- Systematic optimization via classical ML concepts (training data, metrics, optimization)
- Model-agnostic — switch models with minimal impact
- Composable programs that can be iteratively improved
- Research-backed approach from Stanford NLP

## Weaknesses
- **Black box**: Optimization operates without access to model internals
- **Metrics are critical**: Optimization quality fundamentally limited by metric quality
- **Cost**: Optimization can be expensive in token usage
- **Portability issues**: Optimized prompts may not work well outside DSPy's internal behavior
- **Requires high-quality data**: Needs good examples for optimization
- **Learning curve**: Using DSPy well requires ML expertise
- **Gaps**: Observability, experimental tracking, cost management, deployment are current gaps
