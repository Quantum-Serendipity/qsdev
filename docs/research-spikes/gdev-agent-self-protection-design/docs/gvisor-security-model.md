# gVisor Security Model

- **Source URL**: https://gvisor.dev/docs/architecture_guide/security/
- **Retrieved**: 2026-05-15

## Defense-in-Depth Principles

The document outlines three core engineering principles:

1. **No direct passthrough**: "No system call is passed through directly to the host. Every supported call has an independent implementation in the Sentry, that is unlikely to suffer from identical vulnerabilities that may appear in the host."

2. **Limited functionality**: Only common, universal functionality is implemented; specialized APIs are excluded.

3. **Minimized host surface**: The host system call surface is "explicitly enumerated and controlled."

## Sentry Protection Measures

Practical restrictions include:
- Unsafe code isolated in files ending with "unsafe.go"
- No CGo allowed; pure Go binary requirement
- Limited external imports in core packages

## Failure Mode Design

gVisor uses seccomp-bpf as a second layer of defense. The Sentry intercepts syscalls from the untrusted application and re-implements them in Go. The failure mode is fundamentally different from standard containers:

- Standard containers: vulnerability in an allowed syscall lets code compromise the host kernel
- gVisor: attacker must first find a bug in gVisor's Go implementation, THEN find a way to escape using only ~70 allowed host syscalls

The default action for seccomp in gVisor is SCMP_ACT_KILL -- the process is killed if an unallowed syscall is attempted. This is fail-closed at the most extreme level: not just deny, but terminate.

## Key Design: No Direct Passthrough

Every syscall goes through the Sentry's independent implementation. This means there is no "fail-open" path -- if the Sentry cannot handle a syscall, the operation fails. The Gofer handles file operations on behalf of the Sentry over a restricted protocol, further limiting the blast radius.
