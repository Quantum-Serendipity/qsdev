# Deno Security and Permissions

- **Source**: https://docs.deno.com/runtime/fundamentals/security/
- **Retrieved**: 2026-05-12

## Core Security Model

Deno implements a "secure by default" approach. "A program run with Deno has no access to sensitive APIs...unless you specifically enable it." Developers must explicitly grant permissions through command-line flags or runtime prompts.

## Key Principles

1. **No default I/O access**: Code cannot read files, make network requests, access environment variables, or spawn subprocesses without explicit permission.
2. **Equal privilege execution**: All code on the same thread operates at identical privilege levels—modules cannot have different security restrictions within a single thread.
3. **No privilege escalation without consent**: Code requires explicit user approval (via flags or interactive prompts) to escalate permissions.
4. **Static imports unrestricted**: Files imported statically in the initial module graph load without restrictions, though dynamic imports require explicit permissions.
5. **Shared data across invocations**: Multiple instances of the same application can share data through caching and KV storage APIs, but different applications cannot access each other's data.
6. **Unrestricted code execution at same level**: JavaScript eval, new Function, dynamic imports, and web workers execute with the caller's privilege level.

## Permission System

### File System Access
Reading requires `--allow-read` (or `-R`); writing requires `--allow-write` (or `-W`). Permissions can target specific paths:
```
deno run --allow-read=./config.json script.ts
```

### Network Access
`--allow-net` grants network permissions. Hostnames don't allow subdomains unless explicitly specified with wildcards:
```
deno run --allow-net="*.example.com" script.ts
```

Default trusted hosts for imports: deno.land, jsr.io, esm.sh, raw.githubusercontent.com, gist.githubusercontent.com.

### Environment Variables
`--allow-env` grants access. Since Deno v2.1, suffix wildcards enable scoped access:
```
deno run --allow-env="AWS_*" script.ts
```

### Subprocess Execution
`--allow-run` enables subprocess spawning. Child processes run independently with unrestricted system access, effectively bypassing sandboxing.

### Foreign Function Interface
`--allow-ffi` permits loading native libraries via Deno.dlopen or Node-API addons. These execute outside the sandbox.

## Deny Flags

All permission categories support `--deny-*` counterparts that take precedence over allow flags:
```
deno run --allow-read=/etc --deny-read=/etc/hosts script.ts
```

## npm Package Scripts

Post-install scripts for npm packages don't execute by default. The `--allow-scripts` flag must be provided to permit subprocess execution during installation.

## Executing Untrusted Code

Recommended defense-in-depth approach:
- Run with minimal permissions and use `--frozen` lockfiles
- Employ OS-level sandboxing (chroot, cgroups, seccomp)
- Isolate execution in VMs or lightweight containers

## Permission Broker

`DENO_PERMISSION_BROKER_PATH` delegates all permission decisions to an external process via Unix sockets (Linux/macOS) or named pipes (Windows). When active, CLI flags and interactive prompts are disabled.

## Monitoring Permissions

- `DENO_TRACE_PERMISSIONS=1`: Generates stack traces for permission requests
- `DENO_AUDIT_PERMISSIONS=/path`: Writes JSONL-formatted audit logs of all permission accesses

## Configuration Files

Permissions can be stored in `deno.json` or `deno.jsonc` configuration files.
