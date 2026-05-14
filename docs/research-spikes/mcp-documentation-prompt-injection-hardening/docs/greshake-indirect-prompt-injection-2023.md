<!-- Source: https://arxiv.org/abs/2302.12173 -->
<!-- Retrieved: 2026-05-14 -->
<!-- Note: Abstract and summary from arXiv page -->

# Not what you've signed up for: Compromising Real-World LLM-Integrated Applications with Indirect Prompt Injection

## Authors
Kai Greshake, Sahar Abdelnabi, Shailesh Mishra, Christoph Endres, Thorsten Holz, Mario Fritz

## Published
AISec@CCS 2023 (ACM Workshop on Artificial Intelligence and Security)

## Overview

This February 2023 paper introduces a novel security threat to LLM-integrated systems. The researchers demonstrate how the boundary between data and instructions blurs in applications using large language models.

## Core Vulnerability

The paper's central argument concerns "Indirect Prompt Injection" attacks. Rather than users directly interacting with an LLM, adversaries can "remotely exploit LLM-integrated applications by strategically injecting prompts into data likely to be retrieved." This represents a significant departure from traditional prompt injection research focused on direct user-LLM interactions.

## Attack Taxonomy

The taxonomy developed in this research encompasses several categories of security risks:
- Data theft
- Worming capabilities
- Information ecosystem contamination
- Arbitrary code execution effects
- Manipulation of application functionality
- Control over API calls

## Real-World Demonstrations

The researchers validated their attacks against practical systems, including Bing's GPT-4 powered chat and code-completion engines, plus synthetic GPT-4 applications. These demonstrations showed that retrieved prompts can execute arbitrary operations and modify system behavior.

## Security Gap

The paper emphasizes that "effective mitigations of these emerging threats are currently lacking," highlighting an urgent need for defensive mechanisms as LLM integration accelerates across applications.
