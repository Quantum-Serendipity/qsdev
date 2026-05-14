# Constitutional AI: Harmlessness from AI Feedback

- **Source URL**: https://arxiv.org/abs/2212.08073
- **Retrieved**: 2026-03-15
- **Authors**: Yuntao Bai, Saurav Kadavath, et al. (Anthropic)
- **Published**: 2022

## Overview

Constitutional AI trains a harmless AI assistant through self-improvement using a list of principles (a "constitution") instead of human-labeled harmful outputs.

## Two-Phase Training

### Supervised Phase (Critique + Revision)
1. Sample from initial model
2. Generate self-critiques based on constitutional principles
3. Generate revisions addressing the critiques
4. Fine-tune original model on revised responses

### RL Phase (RLAIF)
1. Sample from fine-tuned model
2. Use a model to evaluate which of two samples better follows principles
3. Train preference model from AI-generated preferences
4. Train with RL using preference model as reward signal

## Core Principles

Examples: "Choose the response that is most helpful, honest, and harmless"; "Choose the response that answers in a more friendly, conscientious, and socially acceptable manner."

## Key Insight for Quality

The self-critique → revision pattern is directly applicable to quality enhancement: define explicit criteria (a "constitution") and have the model evaluate its own output against those criteria, then revise. This grounds self-reflection in concrete, enumerable principles rather than vague "make it better" instructions.

## Significance for Agentic Quality

Constitutional AI demonstrates that explicit, enumerable quality criteria dramatically improve self-correction reliability compared to open-ended self-reflection. This principle extends beyond safety to any quality dimension: code correctness, completeness, adherence to requirements, etc.
