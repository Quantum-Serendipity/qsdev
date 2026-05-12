<!-- Source: https://github.com/sirwart/ripsecrets -->
<!-- Retrieved: 2026-05-12 -->

# Ripsecrets: Performance Benchmarks and Configuration

## Performance Benchmarks (Sentry repo, M1 MacBook Air)

| Tool | Runtime | Performance Ratio |
|------|---------|------------------|
| ripsecrets | 0.32s | 1x (baseline) |
| trufflehog | 31.2s | 95x slower |
| detect-secrets | 73.5s | 226x slower |

"Most of the time, your pre-commit will be running on a small number of files," so these benchmarks represent worst-case scenarios rather than typical usage patterns.

## Configuration Options

**CLI Flags:**
- `--install-pre-commit`: Automatically installs as a git pre-commit hook
- `--strict-ignore`: Respects `.secretsignore` file entries when running as pre-commit
- `--additional-pattern`: Allows detection of custom secrets via regex patterns

**Configuration Files:**
- `.secretsignore`: Supports gitignore-style syntax for excluding files; includes a `[secrets]` section for allowlisting specific detected secrets
- `.pre-commit-config.yaml`: Compatible with the pre-commit framework

**Inline Allowlisting:**
Supports "detect-secrets style allowlist comments" on the same line as detected secrets using `# pragma: allowlist secret`

## Supported Secret Patterns

1. **Known Pattern Matches**: Services with identifiable prefixes (Stripe, Slack, etc.)
2. **Random String Detection**: Variables assigned with keywords like "token," "secret," or "password" that contain statistically improbable character combinations (probability threshold: less than 1 in 10,000 of occurring randomly)

## Limitations

"While local-only tools are always going to have more false positives than one that verifies secrets" -- ripsecrets accepts higher false positive rates in exchange for never transmitting code externally. No API verification capability.
