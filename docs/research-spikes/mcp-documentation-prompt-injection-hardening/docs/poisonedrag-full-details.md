<!-- Source: https://arxiv.org/html/2402.07867v3 -->
<!-- Retrieved: 2026-05-14 -->

# PoisonedRAG: Knowledge Corruption Attacks to Retrieval-Augmented Generation — Full Details

## Two Attack Conditions

The paper identifies two necessary conditions for effective knowledge corruption attacks:

1. **Retrieval Condition**: The malicious text's embedding must be similar to the target question's embedding so the retriever ranks it among the top-k results retrieved for that question.

2. **Generation Condition**: When the malicious text alone serves as context, the LLM should generate the target answer. This ensures the LLM produces the desired response when the malicious text appears in retrieved results.

## Attack Success Rates

### Black-Box Setting (5 malicious texts per question):
- **NQ Dataset**: 97% ASR (PaLM 2), 92% (GPT-3.5), 97% (GPT-4)
- **HotpotQA**: 99% (PaLM 2), 98% (GPT-3.5), 93% (GPT-4)
- **MS-MARCO**: 91% (PaLM 2), 89% (GPT-3.5), 92% (GPT-4)

### White-Box Setting:
- **NQ**: 97% ASR (PaLM 2), 99% (GPT-4)
- **HotpotQA**: 94% (PaLM 2), 99% (GPT-4)
- **MS-MARCO**: 90% (PaLM 2), 91% (GPT-4)

## Defenses Evaluated

The paper tested four defense mechanisms:

1. **Paraphrasing**: Using LLMs to rephrase retrieved texts before presenting to the generation model
2. **Perplexity-based Detection**: Identifying texts with anomalously low perplexity scores
3. **Duplicate Text Filtering**: Removing near-duplicate texts from the knowledge base
4. **Knowledge Expansion**: Augmenting knowledge bases with additional verified information (increasing k up to 50)

The authors concluded these defenses "are insufficient to defend against PoisonedRAG." Even knowledge expansion with k=50 still allowed 41-43% ASR.

## Threat Models

**Black-Box Threat Model**: Attacker cannot access retriever parameters or query it. The attack sets S=Q (malicious text = target question concatenated with generated text I). This proves highly effective because the target question itself is maximally similar to the retrieval query.

**White-Box Threat Model**: Attacker has full access to retriever parameters. PoisonedRAG optimizes S using gradient descent to maximize similarity between the malicious text and target question embedding, achieving marginally better results in most cases.

## Conclusions and Future Work

The authors emphasize that knowledge databases represent a "new and practical attack surface" for RAG systems. Their findings demonstrate RAG's vulnerability despite using millions of clean texts. The paper advocates for developing robust defenses, noting that existing approaches fail to adequately protect against such knowledge corruption attacks.
