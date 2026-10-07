# Security Policy

## Reporting Security Vulnerabilities

**Do NOT open a public issue for security vulnerabilities.**

Please report security issues through [GitHub's private vulnerability reporting](https://github.com/Quantum-Serendipity/qsdev/security/advisories/new).

Include:
- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

We will acknowledge receipt within 48 hours and provide a resolution timeline within 7 days.

## Supported Versions

| Version | Supported          |
|---------|--------------------|
| latest  | Yes                |
| < latest | No (upgrade)      |

## Security Practices

- Releases carry [SLSA](https://slsa.dev/) Build L2 provenance (GitHub artifact attestations)
- All release artifacts are signed with Cosign (Sigstore)
- The gateway container image (`ghcr.io/quantum-serendipity/qsdev`) is published for linux/amd64 and linux/arm64, cosign keyless-signed by the `release.yml` workflow and carries build provenance; the release verifies the signature against that workflow's exact identity before publishing binaries
- Dependencies are monitored with Dependabot and govulncheck
- Code is scanned with CodeQL on every PR
- Branch protection enforces peer review on `main`
- All CI actions are pinned to commit SHAs

## Guardrail Invariants

Every command that generates `.claude/settings.json` registers two hooks,
whatever the saved answers say:

- **Self-protection** (`qsdev selfprotect`) is always registered. It has no
  opt-out; the `hooks.self_protection` answer is kept only for schema
  compatibility and is rewritten to `true`.
- **package-guard** (`.claude/hooks/package-guard.py`) is registered unless
  the project opts out with `qsdev disable attach-guard --force`, which
  records the opt-out in the committed `.qsdev.yaml`. Editing the answers
  file, or passing `--claude-hooks`, does not remove it. A project without a
  committed `.qsdev.yaml` (set up with only `qsdev claude init`) cannot opt
  out: `disable --force` refuses there until `qsdev init` creates the file.

CI enforces both for every generating command (`cmd/qsdev/guardrail_invariants_test.go`).
