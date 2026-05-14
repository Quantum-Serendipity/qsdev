# Context Engineering for Agents — LangChain

- **Source URLs**:
  - https://blog.langchain.com/context-engineering-for-agents/
  - https://rlancemartin.github.io/2025/06/23/context_engineering/
  - https://blog.langchain.com/context-management-for-deepagents/
  - https://docs.langchain.com/oss/python/langchain/context-engineering
- **Retrieved**: 2026-03-15
- **Note**: Content synthesized from web search results.

## Core Framework: Write, Select, Compress, Isolate

Context engineering is the art and science of filling the context window with just the right information at each step of an agent's trajectory. Lance Martin (LangChain) groups strategies into four buckets:

### Write
Writing context means saving it outside the context window to help an agent perform a task.
- Note-taking via a "scratchpad" persists information while an agent is performing a task
- Saves information outside of the context window so it's available to the agent later
- Examples: scratchpad files, todo lists, structured state files

### Select
Selecting context means pulling it into the context window to help an agent perform a task.
- RAG is a rich topic and can be a central context engineering challenge
- Code agents are some of the best examples of RAG in large-scale production
- Techniques: grep/file search, knowledge graph based retrieval, re-ranking steps

### Compress
Compressing context means retaining only the tokens required to perform a task.
- Claude Code runs "auto-compact" after exceeding ~95% of the context window
- Summarizes the full trajectory of user-agent interactions
- Strategies: recursive or hierarchical summarization, observation masking

### Isolate
Isolating context means splitting it up to help an agent perform a task.
- Sub-agents get their own fresh context window
- Each runs independently with only the context they need
- Only final results flow back to the parent agent
