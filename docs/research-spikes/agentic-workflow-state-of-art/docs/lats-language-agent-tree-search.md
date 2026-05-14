# Language Agent Tree Search (LATS): Complete Technical Overview
- **Source**: https://arxiv.org/html/2310.04406v3
- **Retrieved**: 2026-03-15
- **Note**: AI-extracted content from arxiv HTML page

## Core Mechanism

LATS adapts Monte Carlo Tree Search (MCTS) to language models, enabling principled exploration of reasoning and decision-making spaces. Six sequential operations:

**Selection & Expansion**: Starting from root node (initial state), traverses tree using UCT formula to balance exploration-exploitation. Upon reaching a leaf, samples n actions from the language model and receives environmental observations, creating new child nodes.

**Evaluation**: Each new node receives a composite value score: V(s) = λ*LM(s) + (1−λ)*SC(s) where λ is a hyperparameter tuned per domain.

**Simulation & Backpropagation**: Expands selected nodes until reaching terminal states. Upon failure, backpropagation updates node values along the trajectory path.

**Reflection**: Failed trajectories trigger self-reflection prompts generating verbal summaries of errors. These become additional context for future iterations.

## Benchmark Results

### HotPotQA (Multi-hop QA)
- LATS (CoT + ReAct): 71% exact match
- ReAct baseline: 32%
- Reflexion: 51%
- RAP: 60%
- Tree-of-Thought (ReAct): 39%

### Programming (HumanEval)
- LATS with GPT-4: 92.7% Pass@1 (state-of-the-art)
- GPT-4 baseline: 80.1%
- Reflexion with GPT-4: 91.0%
- LATS with GPT-3.5: 83.8%
- ReAct with GPT-3.5: 56.9%

### WebShop
- LATS average score: 75.9
- Reflexion: 64.2
- ReAct best-of-k: 59.1

### Game of 24
- LATS: 44%
- RAP: 40%
- Tree-of-Thought: 20%
- Reflexion: 12%

## Key Advantages Over Baselines

vs. ReAct: Expands multiple candidates using principled search rather than greedy decoding. "Doubles the performance" on HotPotQA.

vs. ToT/RAP: Incorporates external environmental feedback. "Simple combination of existing methods is inadequate" for interactive tasks.

vs. Reflexion: Adds systematic search beyond reflection alone. Reflection alone contributes modest gains (0.05 EM on HotPotQA).

## Limitations

1. Computational overhead: Higher cost than reflexive methods
2. State reversion requirement: Assumes ability to reset to earlier decision points
3. Generic reflections: "Generated reflections are often generic and do not provide useful feedback" in complex environments
