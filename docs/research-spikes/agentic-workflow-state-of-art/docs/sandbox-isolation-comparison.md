# Sandbox and Isolation Technologies for AI Agents

- **Sources**:
  - https://northflank.com/blog/best-code-execution-sandbox-for-ai-agents
  - https://northflank.com/blog/how-to-sandbox-ai-agents
  - https://www.codeant.ai/blogs/agentic-rag-shell-sandboxing
  - https://e2b.dev/docs
  - https://modal.com/blog/top-code-agent-sandbox-products
- **Retrieved**: 2026-03-15

## Isolation Technologies

### Firecracker MicroVMs
- Boot in ~125ms with <5 MiB memory overhead
- Hardware-level isolation — each workload gets its own kernel
- Developed by AWS for Lambda and Fargate
- Widely recognized as gold standard for running untrusted AI code
- Used by E2B for their sandbox infrastructure

### gVisor
- Application kernel written in Go (memory-safe)
- Intercepts syscalls in user space — sits between containers and VMs
- 10-30% overhead on I/O-heavy workloads, minimal on compute
- Developed and extensively used by Google
- Used by Modal for their sandbox product

### Docker/OCI Containers
- Share host kernel — kernel vulnerability can allow escape
- Fastest startup, lowest overhead
- Insufficient for truly untrusted code as of 2026 consensus
- Still common for development and trusted workloads

### Kata Containers
- Lightweight VMs with container-like experience
- Stronger isolation than standard containers
- More overhead than Firecracker

## Managed Platforms

### E2B
- Open-source, Firecracker-based microVM isolation
- Boots sandbox in <1 second
- Session-scoped sandboxes defined via custom templates
- Python and JS/TS SDKs
- Used by ~50% of Fortune 500 (as of 2025)
- Millions of sandboxes per week
- Sandbox run times went up >10x from 2024 to 2025 (driven by long-running agents like Manus)

### Modal
- gVisor-based isolation within broader platform (inference, training, batch)
- Dynamically defined at runtime
- Scales from zero to 50,000+ concurrent instances in seconds
- Part of larger AI infrastructure platform

### Daytona
- Used by Open SWE (LangChain's coding agent)
- Workspace-focused sandboxing

## 2026 Consensus

Shared-kernel container isolation (Docker/runc) is no longer adequate for executing untrusted AI agent code. LLM-generated or user-supplied code should be treated as hostile. MicroVMs or gVisor represent the minimum viable isolation.

## Practical Considerations for Coding Agents

- **OpenHands V0**: Docker containers for all sandboxing (caused friction with state divergence)
- **OpenHands V1**: Opt-in sandboxing — local by default, containerize when isolation needed
- **Devin**: Isolated VMs per instance, multiple parallel instances
- **Open SWE**: Daytona sandboxes, supports Modal, Runloop, LangSmith
- **Claude Code**: Runs locally by default (local terminal), cloud VMs available for offload
