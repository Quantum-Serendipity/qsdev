# OpenHands Software Agent SDK V1 Architecture

- **Source**: https://arxiv.org/html/2511.03690v1
- **Retrieved**: 2026-03-15

## Core Architecture

OpenHands V1 refactors from monolithic design to modular SDK with four independently deployable Python packages:

1. **openhands.sdk**: Core abstractions (Agent, Conversation, LLM, Tool, MCP) and event systems
2. **openhands.tools**: Concrete tool implementations extending SDK abstractions
3. **openhands.workspace**: Execution environments (Docker, hosted APIs) implementing base classes
4. **openhands.agent_server**: REST/WebSocket API server enabling remote execution

### Design Principles

1. **Optional Sandboxing**: Agents run locally by default but can switch to sandboxed environment when additional safety or resource control is required. V0 assumed all tool calls should run inside sandboxed Docker containers, introducing friction with divergent states between agent and sandbox processes.

2. **Immutable Components with Single State Source**: All agents, tools, and LLMs are immutable, serializable Pydantic models. ConversationState is the exclusive mutable entity.

3. **Strict Separation of Concerns**: SDK decoupled from applications (CLI, Web UI, GitHub App) to use as shared library.

4. **Two-Layer Composability**: Independent deployment packages combine flexibly with typed components.

## Event-Sourced State Management

All interactions treated as immutable events appended to append-only log.

**Event Hierarchy**:
- **Event**: Base immutable structure with ID, timestamp, type-safe serialization
- **LLMConvertibleEvent**: Adds conversion to LLM message format (MessageEvent, ActionEvent, SystemPromptEvent, ObservationBaseEvent)
- **Internal Events**: State updates and control flow without LLM exposure

**ConversationState** maintains mutable metadata fields plus append-only EventLog. FIFO lock ensures thread-safe updates. Persistence: metadata to base_state.json, events as individual JSON files. Resume by reloading state and replaying events.

## Tool System: Action-Execution-Observation Pattern

- **Action**: Specifies input schema as Pydantic models, validates LLM-generated arguments before execution
- **Execution**: ToolExecutor implements actual logic with validated Actions
- **Observation**: Captures execution output in structured form, converts to LLM-compatible format

MCP tools are first-class: JSON Schemas translate automatically into Action models. MCPToolDefinition extends ToolDefinition, MCPToolExecutor delegates to FastMCP's MCPClient.

### Distributed Execution

Registry-based mechanism decouples tool specifications from implementations. Tools serialize as lightweight specs with registered names and JSON parameters. At runtime, resolver reconstructs full definitions including executors based on conversation context.

## Agent Execution Model

Agents are stateless, immutable specifications. Event-driven loop: agents emit structured events through callbacks rather than returning results directly. This enables:
- Security interleaving
- Incremental execution with pause/resume
- Event streaming for real-time UI updates

**Sub-Agent Delegation**: Hierarchical coordination through delegation tools. Sub-agents operate as independent conversations inheriting parent configuration and workspace context.

## Context Management

**Condenser** system maintains conversation length within LLM context limits by dropping events and replacing with summaries. LLMSummarizingCondenser reduces API costs by up to 2x with no degradation.

## Security and Confirmation

**SecurityAnalyzer** rates each tool call as low, medium, high, or unknown risk. **ConfirmationPolicy** determines whether user approval required. Agent pauses in WAITING_FOR_CONFIRMATION state. Dynamic trust adaptation during sessions.

## Error Handling

Event-sourced design enables deterministic replay. **Stuck Detection**: Automatic detection of infinite loops (repeated same action) and redundant tool calls (querying same information repeatedly). System can automatically terminate to prevent resource waste.

## Workspace Abstraction

- **LocalWorkspace**: Executes in-process against host filesystem/shell
- **RemoteWorkspace**: Delegates over HTTP, with DockerWorkspace and APIRemoteWorkspace implementations
- Both share identical API for seamless migration

## Performance

SWE-Bench Verified: 72.8% with Claude Sonnet 4.5. GAIA: 67.9% accuracy.
