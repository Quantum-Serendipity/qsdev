# LLM-as-Judge: Reliability, Methodology, and Limitations
- **Sources**:
  - https://arxiv.org/abs/2306.05685 (original MT-Bench/Chatbot Arena paper)
  - https://arxiv.org/html/2506.09443v1 (LLMs Cannot Reliably Judge Yet)
  - https://arxiv.org/html/2508.06225v2 (Overconfidence in LLM-as-Judge)
  - https://skywork.ai/blog/chatbot-arena-lmsys-review-2025/
- **Retrieved**: 2026-03-15
- **Note**: Composite summary from multiple papers and reviews

## Core Methodology

### MT-Bench
- 80 multi-turn questions across 8 categories
- GPT-4 as judge, scoring 1-10
- Tests instruction following, reasoning, math, coding, creativity, knowledge

### Chatbot Arena
- Anonymous, randomized head-to-head battles
- Crowdsourced human preferences
- Bradley-Terry model fitted to pairwise preferences
- Reports Elo-like scores with uncertainty intervals
- Multiple arenas: text, vision, text-to-video

### AlpacaEval 2.0
- Length-controlled win rates
- Spearman correlation of 0.98 with Chatbot Arena (up from 0.94 without length control)
- Runs in <3 minutes, costs <$10
- Uses GPT-4 Turbo as auto-annotator

## Reliability Findings

### Agreement rates
- Strong LLM judges (GPT-4) achieve >80% agreement with human preferences
- This matches inter-human agreement rates

### Serious robustness concerns (2025)
- A null model (always outputting constant, irrelevant response) can secure high win rates on AlpacaEval 2.0 and MT-Bench
- Appending short, task-agnostic adversarial phrases can dramatically inflate scores
- LLM judges are overconfident — minimal variation in confidence between correct and incorrect assessments

### Known biases
- Length bias: longer responses preferred (addressed by length-controlled AlpacaEval)
- Position bias: preference for response in certain position
- Self-enhancement bias: models may prefer their own outputs
- Casual users penalize abstention more than subtle inaccuracy

## Self-Evaluation Limitations

### Overconfidence
- LLMs tend to be overconfident, presenting answers in articulate manner regardless of correctness
- Training optimizes for fluency and coherence, not truthful reasoning
- Models not trained to say "I don't know" — admitting uncertainty penalized during training

### Calibration Issues
- SFT yields well-calibrated confidence via maximum-likelihood estimation
- RLHF (PPO, GRPO) and DPO induce overconfidence via reward exploitation
- Black-box self-critique approaches are heuristic and still return overconfident responses

### Practical Implications
- Self-evaluation useful for catching obvious errors
- Unreliable for assessing correctness of novel or complex reasoning
- External verification (test execution, tool-based checks) much more reliable than self-assessment
