<!-- Source: https://socket.dev/blog/malicious-package-exploits-go-module-proxy-caching-for-persistence -->
<!-- Retrieved: 2026-05-12 -->

# Go Module Supply Chain Attack: BoltDB Typosquatting Analysis

## Attack Overview

Socket researchers discovered a malicious typosquat package `github.com/boltdb-go/bolt` impersonating the legitimate BoltDB database, which is "widely adopted within the Go ecosystem, with 8,367 other packages depending on it." The attack exploited Go Module Proxy caching to maintain persistence for over three years.

## Exploitation Mechanism

The threat actor employed a sophisticated technique leveraging Go's immutable module design:

1. **Initial Distribution**: Published malicious package version v1.3.1, which the Go Module Proxy cached indefinitely upon first access.

2. **Cover Tracks**: After ensuring proxy caching, the attacker rewrote GitHub tags to point to clean code, making manual repository audits reveal no malicious content.

3. **Persistent Delivery**: Developers using the `go` CLI continued receiving "the cached malicious version from the Go Module Proxy, rather than the updated, benign version."

## Technical Backdoor

The `boltdb-go/bolt` package contained a covert remote access capability embedding a persistent TCP connection to `49.12.198[.]231:20022`. The backdoor:

- Activates when developers call `Open()`
- Receives and executes arbitrary shell commands without validation
- Returns command output to threat actors
- Auto-restarts if connection fails, maintaining "continuous access to the compromised system"

Obfuscation techniques included string manipulation transforming constants into the hidden IP address, evading static analysis detection.

## Go Module Vulnerability

This incident reveals a critical architectural tension: Go's intentional immutability prevents silent code changes but simultaneously enables malicious actors to "persistently distribute malicious code despite subsequent changes to the repository." Once a malicious version is cached, no removal mechanism exists -- it remains permanently available through the proxy.

## Recommendations

Organizations should implement layered defenses including dependency analysis tools, real-time pull request monitoring, and security scanning of actually installed package contents rather than relying solely on repository audits.
