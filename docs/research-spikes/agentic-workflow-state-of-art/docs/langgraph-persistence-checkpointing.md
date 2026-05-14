# LangGraph Persistence and Checkpointing

- **Source URLs**:
  - https://docs.langchain.com/oss/python/langgraph/persistence
  - https://aws.amazon.com/blogs/database/build-durable-ai-agents-with-langgraph-and-amazon-dynamodb/
  - https://dev.to/programmingcentral/unlocking-ai-resilience-mastering-state-persistence-with-langgraph-and-postgresql-50h0
- **Retrieved**: 2026-03-15
- **Note**: Content synthesized from web search results.

## Core Architecture

LangGraph has a built-in persistence layer that saves graph state as checkpoints. A snapshot of the graph state is saved at every super-step (a single "tick" where all scheduled nodes execute, potentially in parallel).

## Checkpoints

- Represented by StateSnapshot objects
- Organized into threads (unique IDs for separate conversation/task states)
- Created at each super-step boundary

## Thread Persistence

A thread is a unique ID assigned to each checkpoint saved by a checkpointer. It contains the accumulated state of a sequence of runs. Threads enable checkpointing of multiple different runs — essential for multi-tenant applications.

## State Serialization

- Serialization protocol for encoding/decoding checkpoint data
- EncryptedSerializer available for secure state handling
- Compression option (`enable_checkpoint_compression`) reduces checkpoint size before storage
- Small checkpoints (<350 KB) stored directly in DynamoDB
- Large checkpoints (>=350 KB) uploaded to S3 with DynamoDB storing a reference

## Fault-Tolerant Resumption

When a graph node fails mid-execution:
- LangGraph stores pending checkpoint writes from other nodes that completed successfully
- When you resume execution, you don't re-run successful nodes
- Enables human-in-the-loop workflows, time travel debugging, and fault-tolerant execution

## Checkpointer Interface

Required methods:
- `.put` — Store checkpoint with configuration and metadata
- `.get_tuple` — Fetch checkpoint tuple for given thread_id and checkpoint_id
- `.list` — List checkpoints matching configuration and filter criteria
- `.delete_thread()` — Delete all checkpoints associated with a thread

## Storage Backends

- PostgresSaver, DynamoDB, Snowflake, Couchbase, and file-based options
- Database-backed: best for structured logs and conversation history with fast queries
- File-based: best for long-running agents with large state (GB-scale)
