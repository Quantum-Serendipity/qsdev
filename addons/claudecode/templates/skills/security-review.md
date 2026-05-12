# Security Review

Perform a security-focused code review informed by the OWASP Top 10 and supply
chain security best practices.

## Scope

Review all code changes on the current branch relative to the base branch:
`git diff main...HEAD`. Also review any new dependencies introduced.

## Injection (OWASP A03)

- Search for string concatenation in SQL queries. All queries must use
  parameterized statements or a query builder.
- Check for template injection: user input must not be interpolated directly
  into HTML, shell commands, or log format strings.
- Look for command injection via `os/exec`, `subprocess`, `child_process`,
  or equivalent. Arguments must be passed as arrays, never shell strings.

## Broken authentication and access control (OWASP A01, A07)

- Verify that authentication checks are present on all new endpoints.
- Ensure authorization is enforced at the resource level, not just the route.
- Check that session tokens, API keys, and JWTs are validated correctly.
- Look for insecure direct object references (IDOR).

## Sensitive data exposure (OWASP A02)

- Search the diff for hardcoded secrets, API keys, passwords, or tokens.
  Run: `git diff main...HEAD | grep -iE '(password|secret|api_key|token|private_key)\s*[:=]'`
- Verify that sensitive fields are excluded from logs and API responses.
- Check that PII is encrypted at rest and in transit.

## Dependency vulnerabilities

Run the ecosystem-appropriate audit tool:
- Go: `govulncheck ./...`
- Node.js: `npm audit --audit-level=high`
- Python: `pip-audit` or `safety check`
- Rust: `cargo audit`

Report any vulnerabilities with their severity and remediation path.

## Secrets in version control

- Check for `.env` files, private keys, or credential files in the diff.
- Verify that `.gitignore` includes common secret patterns.
- If secrets are detected, flag as **critical** and recommend immediate rotation.

## File and process permissions

- New files should not have overly broad permissions (no 0o777, avoid 0o666).
- Executables should be explicitly marked (0o755) only when necessary.
- Check that temporary files are created securely (restricted permissions, random names).

## Rate limiting and resource exhaustion

- New API endpoints should have rate limiting or be behind a rate-limited gateway.
- Check for unbounded loops, unbounded allocations, or missing pagination.
- Verify that file uploads have size limits.

## Cryptography

- Verify use of standard, well-maintained cryptographic libraries.
- Check for deprecated algorithms (MD5, SHA1, DES, RC4).
- Ensure random values for security use come from cryptographic RNGs
  (`crypto/rand`, `secrets`, not `math/rand`).

## Output format

```
## Security Review Summary

### Critical
- **[file:line]** <finding>

### High
- **[file:line]** <finding>

### Medium
- **[file:line]** <finding>

### Low / Informational
- **[file:line]** <finding>

### Dependency audit
<output from audit tool>

### Recommendation
<PASS / FAIL — with required actions>
```
