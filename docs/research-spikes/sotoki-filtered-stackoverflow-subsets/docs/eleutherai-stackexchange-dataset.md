---
source: https://github.com/EleutherAI/stackexchange-dataset
retrieved: 2026-05-14
---

# EleutherAI Stack Exchange Dataset Tool

## Core Functionality
Downloads Stack Exchange XML dumps and converts them into a text dataset for Language Models. Processes question-answer pairs from the Stack Exchange archive.

## Processing Details
Filtering by score and responses, with TODO items mentioning: "add flags to change min_score / max_responses args." Default thresholds are applied when processing pairs.

## Relationship Handling
Output is described as "question-answer pair text dataset" but doesn't explicitly detail how comments or other relationships are managed during filtering. Code structure mentions files like `pairer.py`, suggesting deliberate pairing logic.

## Tag Filtering
One TODO item asks: "should we add metadata to the text (i.e name of stackexchange & tags)?" Tags might not be actively filtered or included by default.

## Format
Processes XML files from Stack Exchange dumps and outputs as raw text, with a TODO mentioning potential conversion to "lm dataformat."

Note: Repository lacks detailed documentation on filtering mechanics; examining the actual Python code would be needed for precise processing rules.
