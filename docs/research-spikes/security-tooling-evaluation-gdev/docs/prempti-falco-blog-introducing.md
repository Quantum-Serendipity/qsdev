<!-- Source: https://falco.org/blog/introducing-prempti/ -->
<!-- Retrieved: 2026-05-15 -->
<!-- Author: Leonardo Grasso, Published: May 12, 2026 -->

# Prempti: Falco Meets AI Coding Agents - Blog Post

## Introduction

The article announces Prempti, an experimental project that extends Falco's runtime security capabilities to AI coding agents. The core premise addresses a critical gap: developers using tools like Claude Code lack visibility into what these agents do at runtime -- file access, command execution, and credential handling all occur within user sessions with minimal oversight.

## The Problem: Agents as Black Boxes

The piece illustrates a fundamental concern: when coding agents operate within user sessions, they access files, run commands, and interact with credentials using user permissions. A developer might ask an agent to refactor code, but lacks structured visibility into whether the agent subsequently attempts to read `~/.ssh/known_hosts` or write to `~/.aws/` directories.

A demonstration shows an agent attempting both read and write operations to restricted areas, with both being blocked and the agent receiving "structured message explaining why."

## How Prempti Functions

The system operates as a lightweight user-space service requiring no root access, kernel modules, or containers. When an agent makes a tool call -- file writes, shell commands, or file reads -- Prempti intercepts the action **before execution**, evaluates it against Falco rules, and returns a verdict:

- **Allow**: The tool call proceeds normally
- **Deny**: The tool call is blocked with explanation
- **Ask**: User receives an interactive prompt

The architecture involves five steps: the hook fires before each tool call; an interceptor sends the event via Unix socket; Falco's rule engine evaluates against policies; matching rules produce verdicts; the interceptor delivers the result back to the agent.

Prempti leverages Falco's plugin system to define a new event source (`coding_agent`) with specialized fields including `tool.name`, `tool.input_command`, `tool.file_path`, and `agent.cwd`.

## Two Operational Modes

**Monitor mode** logs all tool calls against rules without enforcing actions -- useful for initial observation and rule tuning. **Guardrails mode** actively enforces verdicts through blocking and prompting.

Users can switch modes via command line:
```
premptictl mode monitor      # observe only
premptictl mode guardrails   # enforce verdicts
premptictl logs              # watch live events
```

## Rule Writing and Policy Definition

Falco users will find rule syntax familiar. An example rule blocks piping content to shell interpreters:

```yaml
- rule: Deny pipe to shell
  desc: Block piping content to shell interpreters
  condition: >
    tool.name = "Bash"
    and (tool.input_command contains "| sh"
         or tool.input_command contains "| bash"
         or tool.input_command contains "| zsh")
  output: >
    Falco blocked piping to a shell interpreter (%tool.input_command)
  priority: CRITICAL
  source: coding_agent
  tags: [coding_agent_deny]
```

The output field is designed to be LLM-friendly for agent-user communication.

## Default Ruleset Coverage

The default policies address six areas:

1. **Working-directory boundary**: Monitor and confirm file access outside the project directory
2. **Sensitive paths**: Deny reads/writes to `/etc/`, `~/.ssh/`, `~/.aws/`, cloud credentials, and `.env` files
3. **Sandbox disable**: Detect attempts to disable agent sandbox configuration
4. **Threats**: Credential access, destructive commands, pipe-to-shell attacks, encoded payloads, exfiltration, IMDS access, reverse shells, and supply-chain installs
5. **MCP and skill content**: MCP server config poisoning and slash-command file injection
6. **Persistence vectors**: Hook injection, git hooks, package-registry redirects, AI API base-URL overrides, and API key leaks

Custom rules can be added to `~/.prempti/rules/user/` and persist across upgrades.

## Claude Code Integration for Rule Authoring

The project includes a Claude Code skill for interactive Falco rule creation. Users can install it from the marketplace and request rules like "Block the agent from running git push" or "Deny any read outside the working directory." The skill guides users through writing, placing, and validating rules.

## Acknowledged Limitations

The authors are transparent about constraints. Prempti intercepts declared tool calls, not the system calls resulting from those actions. If an agent compiles and executes a binary, Falco sees the commands (`gcc main.c -o main` and `./main`) but not what the binary does internally. For deep syscall-level visibility on Linux, Falco's kernel instrumentation (eBPF/kmod) remains necessary.

Additionally, Prempti is not a sandbox and cannot prevent determined agents from circumventing the hook mechanism. It functions as a policy layer at the agent level -- complementary to, not replacing, sandboxing and system hardening.

## Installation

macOS, Linux, Windows installers provided. Verification via `premptictl status` and `premptictl hook status`.

## Call for Community Input

The authors invite feedback regarding rules developed, agents needing support, and unexpected behaviors. Released under Apache License 2.0, currently supports Claude Code on Linux (x86_64, aarch64), macOS (Apple Silicon, Intel), and Windows (x86_64, ARM64), with Codex integration on the roadmap.
