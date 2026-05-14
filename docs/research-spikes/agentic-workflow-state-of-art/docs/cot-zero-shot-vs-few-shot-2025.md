# Chain-of-Thought Prompting: Zero-Shot vs Few-Shot (2025)

- **Source URLs**:
  - https://arxiv.org/abs/2506.14641 (Revisiting Chain-of-Thought Prompting: Zero-shot Can Be Stronger than Few-shot)
  - https://www.promptingguide.ai/techniques/cot
  - https://arxiv.org/abs/2205.11916 (Large Language Models are Zero-Shot Reasoners)
- **Retrieved**: 2026-03-15
- **Note**: Compiled from web search results on CoT prompting research.

---

## Key 2025 Finding: Zero-Shot CoT Often Beats Few-Shot CoT

### Evidence
- Qwen2.5-72B: 81.2% on GSM8K (zero-shot CoT) vs 79.0% (8-shot CoT)
- MATH: 55.3% (zero-shot) vs 53.8% (8-shot)
- For sufficiently strong LLMs, few-shot CoT exemplars primarily enforce output format, not increase reasoning

### Why Zero-Shot Can Win
- Model generates its own logical path without being constrained by potentially unrepresentative examples
- Examples can anchor the model to specific reasoning patterns that may not be optimal
- Modern models have internalized reasoning patterns through training

## When CoT Helps
- **Mathematical reasoning**: Significant improvement over direct answering
- **Logical reasoning**: Multi-step deduction benefits
- **Complex multi-step tasks**: Where intermediate steps prevent error accumulation
- **Difficult problems**: Benefits increase with task difficulty

## When CoT Hurts or Doesn't Help
- **Simple tasks**: Classification, sentiment analysis — CoT adds overhead without benefit
- **Tasks with clear patterns**: Where the answer is obvious, CoT can introduce errors through overthinking
- **With strong models**: Modern frontier models may reason internally without explicit CoT

## CoT Faithfulness Concerns
- CoT is frequently unfaithful to actual reasoning (see Anthropic's research)
- Can exhibit post-hoc rationalization
- Longer CoTs correlate with less faithful reasoning
- Should not be relied upon as sole transparency mechanism

## Practical Guidance for Claude Code
1. For reasoning tasks: "Think step by step" or adaptive thinking mode
2. For formatting tasks: Few-shot examples more valuable than CoT
3. Prefer adaptive thinking over manual CoT in Claude 4.6
4. "Think thoroughly" often produces better reasoning than prescriptive step-by-step plans
5. Use `<thinking>` tags in few-shot examples to demonstrate reasoning patterns
