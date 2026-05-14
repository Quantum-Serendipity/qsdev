<!-- Source: https://arxiv.org/abs/2402.07867 -->
<!-- Retrieved: 2026-05-14 -->
<!-- Note: Only abstract available from arXiv page; full PDF not fetched -->

# PoisonedRAG: Knowledge Corruption Attacks to Retrieval-Augmented Generation of Large Language Models

## Authors
Wei Zou, Runpeng Geng, Binghui Wang, Jinyuan Jia

## Abstract

The research identifies a critical vulnerability in Retrieval-Augmented Generation systems. As stated: "The knowledge database in a RAG system introduces a new and practical attack surface."

**Attack Mechanism:**
The authors propose an approach where attackers could "inject a few malicious texts into the knowledge database of a RAG system to induce an LLM to generate an attacker-chosen target answer for an attacker-chosen target question."

**Performance Metrics:**
The attack achieved "a 90% attack success rate when injecting five malicious texts for each target question into a knowledge database with millions of texts."

**Defense Analysis:**
The study evaluated defensive measures, finding that "they are insufficient to defend against PoisonedRAG, highlighting the need for new defenses."

## Publication
Published at USENIX Security 2025. arXiv preprint February 2024.
