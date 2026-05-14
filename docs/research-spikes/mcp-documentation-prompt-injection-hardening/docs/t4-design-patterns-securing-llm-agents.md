# Design Patterns for Securing LLM Agents Against Prompt Injections

- **Source**: https://arxiv.org/html/2506.08837v2
- **Retrieved**: 2026-05-14

## Core Architectural Patterns

### 1. Action-Selector Pattern
The agent translates natural language requests into predefined tool calls without processing feedback from those actions. "The agent acts merely as an action selector," preventing any consequential actions influenced by untrusted data.

### 2. Plan-Then-Execute Pattern
The agent commits to a fixed sequence of actions before processing untrusted data. Provides "control flow integrity" protection -- tool outputs cannot influence which actions execute, only their parameters.

### 3. LLM Map-Reduce Pattern
Dispatches isolated LLM agents to process individual data items independently. Map operation processes discrete data units with constrained outputs; reduce operation either avoids LLMs entirely or enforces safety constraints (e.g., regex validation ensuring outputs are numeric).

"A malicious file is now restricted to tricking the map LLM into marking...that file" without affecting other documents' processing.

### 4. Dual LLM Pattern
Two LLM tiers separate privileges:
- **Privileged LLM:** Receives instructions, plans actions, accesses tools
- **Quarantined LLM:** Processes untrusted data only; no tool access

Results stored in restricted variables (e.g., `$VAR`) that privileged instances cannot dereference.

### 5. Code-Then-Execute Pattern
The LLM generates formal computer programs calling available tools and spawning unprivileged LLMs for untrusted text processing. Generalizes plan-then-execute by replacing planning with explicit program generation.

### 6. Context-Minimization Pattern
Unnecessary context -- particularly user prompts -- is removed after informing initial actions but before processing results. Prevents subsequent prompt injections during response generation or result aggregation.

## Information Flow Control Mechanisms

### Trust Boundary Enforcement
"Once an LLM agent has ingested untrusted input, it must be constrained so that it is impossible for that input to trigger any consequential actions."

### Output Constraint Validation
Isolated LLMs return constrained types (booleans, predetermined categories, or structured formats) that cannot propagate injections downstream.

### Symbolic Variable Handling
Symbolic references rather than dereferencing values within LLM contexts. Traditional software components substitute actual values only during function execution.

## Tool Result Handling Strategies

1. **Action-Selector:** Results never fed back to decision logic
2. **Plan-Then-Execute:** Results modify action parameters only, not action selection
3. **Map-Reduce:** Individual results constrained; aggregation uses non-LLM operations
4. **Dual LLM:** Results maintained as symbolic references inaccessible to privileged instances
5. **Code-Then-Execute:** Code variables receive tool outputs but cannot be dereferenced within subsequent LLM prompts
6. **Context-Minimization:** Results processed but source context removed before final generation

## Security-Utility Trade-offs

- **Action-Selector:** Maximum security but severely limited flexibility
- **Plan-Then-Execute:** Moderate security; requires predefined action sequences
- **Map-Reduce/Dual LLM:** Higher utility retention through controlled decomposition
- **Context-Minimization:** Minimal utility loss when information naturally flows unidirectionally

The authors recommend "use a combination of design patterns" rather than relying on single approaches.
