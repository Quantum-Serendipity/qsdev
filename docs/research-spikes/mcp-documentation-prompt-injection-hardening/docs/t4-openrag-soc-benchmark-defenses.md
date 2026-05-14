# Hidden-in-Plain-Text: OpenRAG-Soc Benchmark for Social-Web Indirect Prompt Injection in RAG

- **Source**: https://arxiv.org/html/2601.10923v2
- **Retrieved**: 2026-05-14

## Defense Mechanisms Tested

The benchmark evaluates three primary defenses:

1. **Sanitization**: "HTML/Markdown sanitization that neutralizes hidden/off-screen carriers and risky attributes" using DOMPurify while preserving visible text.

2. **Unicode Normalization**: Application of NFKC normalization plus control character stripping to address zero-width and homoglyph risks.

3. **Attribution-Gated Prompting**: Quote-and-cite methodology requiring inline citations for all claims, constraining outputs to retrieved spans.

## Attack Success Rate (ASR) Results

Across carriers, vanilla configurations showed highest instruction-following rates:

| Carrier | Vanilla | Sanitized | Normalized | All Defenses |
|---------|---------|-----------|-----------|--------------|
| Hidden spans | 34.0% | 12.3% | 33.1% | 5.0% |
| Off-screen CSS | 30.1% | 9.8% | 29.3% | 4.6% |
| Alt text | 27.8% | 11.1% | 27.0% | 4.8% |
| ARIA | 9.6% | 9.3% | 9.4% | 5.1% |
| Zero-width | 23.2% | 23.0% | 7.8% | 4.2% |
| **Macro average** | **24.9%** | **13.1%** | **21.3%** | **4.7%** |

## Defense-Specific Findings

**Sanitization effectiveness**: Reduced ASR primarily for HTML/Markdown carriers (hidden spans, off-screen CSS, alt text) with minimal impact on ARIA or zero-width attacks.

**Normalization impact**: "Normalized chiefly reduces zero-width attacks," achieving 7.8% ASR on zero-width vectors versus 23.2% vanilla.

**Combined defenses**: "All Defenses is consistently lowest" at 4.7% macro ASR with negligible utility cost.

## Real-Web Stress Testing

Under adaptive prompts (N=2,350 pages):

| Defense | Static | Adaptive | Delta |
|---------|--------|----------|-------|
| Vanilla | 22.7% | 28.9% | +6.2 pp |
| Normalized | 19.1% | 22.2% | +3.1 pp |
| Sanitized | 12.0% | 15.8% | +3.8 pp |
| All Defenses | 4.3% | 5.4% | +1.1 pp |

Defense ordering remained consistent despite adaptive attacks.

## Utility and Performance

- Sanitization adds 3.1% pipeline latency (p95: 7.4%)
- Unicode normalization: <0.5% latency impact
- "Answerability changed by -1.8pp / -2.2pp under Sanitized+Normalized"
- Attribution-gated runs achieved 0.88 token-level citation alignment

## Failure Mode Analysis

Residual attacks post-defense distribute as:
- Visible imperatives (49%): Attribution-gated reduces by 78%
- Confusables in code (31%): Normalization reduces by 66%
- Query-aligned attacks (20%): Combined sanitization+attribution reduces by 61%

Human evaluation validated detector performance (F1=0.90, Cohen's kappa=0.84) with +/-0.8pp macro-ASR uncertainty bounds.
