# Aider Repository Map with Tree-Sitter

- **Source URLs**:
  - https://aider.chat/2023/10/22/repomap.html
  - https://aider.chat/docs/repomap.html
  - https://deepwiki.com/Aider-AI/aider/4.1-repository-mapping
- **Retrieved**: 2026-03-15
- **Note**: Content synthesized from web search results.

## Overview

Aider sends the LLM a concise map of the whole git repository that includes the most important classes and functions along with their types and call signatures. This map is built automatically using tree-sitter to extract symbol definitions from source files.

## How Tree-Sitter Is Used

Tree-sitter parses source code into an Abstract Syntax Tree (AST) based on the syntax of the programming language. Using the AST, Aider:
1. Identifies where functions, classes, variables, types and other definitions occur
2. Identifies where else in the code these things are used or referenced
3. Builds a dependency graph between files

## Graph Ranking Algorithm

The RepoMap class:
1. Extracts code definitions and references using tree-sitter parsers
2. Builds a NetworkX MultiDiGraph of file relationships (files = nodes, dependencies = edges)
3. Ranks nodes using PageRank with personalization (personalized toward files the user is actively editing)
4. Formats the top-ranked definitions into a token-limited context string

## Token Budget Optimization

- User configures budget via `--map-tokens` switch (default: 1,024 tokens)
- The `get_ranked_tags_map()` method uses binary search to find the maximum number of tags fitting the budget
- Targets output within 15% of max_map_tokens
- Only includes the most important identifiers — those most often referenced by other code
- A function called by 20 other functions gets higher priority than a private helper called once

## Key Insight
The repo map gives the LLM a "bird's eye view" of the entire codebase in a very token-efficient way, enabling the model to understand code structure and navigate to relevant files without consuming excessive context.
