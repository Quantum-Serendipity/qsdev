# Reflexion: Language Agents with Verbal Reinforcement Learning
- **Source**: https://arxiv.org/abs/2303.11366
- **Retrieved**: 2026-03-15
- **Note**: AI-extracted summary from arxiv abstract page, not full paper text

## Content

The Reflexion approach operates through linguistic feedback rather than weight updates. Agents "verbally reflect on task feedback signals, then maintain their own reflective text in an episodic memory buffer to induce better decision-making in subsequent trials."

The system accepts diverse feedback sources—both "scalar values or free-form language" and feedback from "external or internally simulated" origins. This flexibility allows agents to learn across varied problem domains without traditional fine-tuning.

On HumanEval coding tasks, Reflexion achieved "91% pass@1 accuracy," surpassing "previous state-of-the-art GPT-4 that achieves 80%." The paper demonstrates effectiveness across multiple problem categories: sequential decision-making, code generation, and language reasoning tasks.

Rather than updating model weights through standard reinforcement learning—which requires "extensive training samples and expensive model fine-tuning"—Reflexion uses the language model's own capability for self-assessment. Agents store reflections in episodic memory, creating a learning mechanism grounded in natural language understanding rather than gradient updates.

The authors conducted "ablation and analysis studies using different feedback signals, feedback incorporation methods, and agent types" to investigate which components drive performance gains.
