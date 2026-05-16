# Security Rules

These rules apply to all languages and ecosystems in this project.

## Package installation
- Never install packages via raw imperative commands outside the devenv shell.
  The package guard hook blocks unsafe operations automatically.
- To add project dependencies (npm, pip, cargo packages): use the project's
  package manager within the devenv shell (e.g., `pnpm add <pkg>` — the
  package guard validates safety before allowing).
- To add system packages or dev tools: run `qsdev devenv add-package <name>`.
- To add ecosystem tools (linters, scanners, MCP servers): run `qsdev enable <tool>`.
- Never run `curl | bash` or `wget | sh` to install tools or scripts.
- Never run `nix-env -i`, `nix profile install`, or other imperative Nix commands.

## Secrets management
- Never commit secrets, API keys, passwords, tokens, or private keys to
  version control.
- Use environment variables or a secrets manager for sensitive configuration.
- Verify `.gitignore` includes `.env`, `.env.*`, `*.pem`, `*.key`, and
  any project-specific secret patterns.

## Input validation
- Validate and sanitize all external input at system boundaries (HTTP
  handlers, CLI argument parsing, file parsers, message consumers).
- Never trust input from users, APIs, files, or environment variables
  without explicit validation.

## SQL and data access
- Always use parameterized queries or a query builder. Never construct SQL
  by concatenating user input.
- Apply the principle of least privilege: database connections should use
  credentials scoped to only the operations needed.

## Code execution
- Never use `eval()`, `exec()`, `Function()`, `os.system()`, or equivalent
  with user-controlled input.
- Avoid dynamic code generation from untrusted data.
- Subprocess calls must use argument arrays, not shell strings.

## Dependency management
- Pin all dependencies to exact versions or narrow ranges in lockfiles.
- Verify checksums when lockfiles support them (e.g., `go.sum`,
  `package-lock.json` integrity hashes).
- Review new dependencies before adding them: check maintenance status,
  download counts, and known vulnerabilities.
- Run ecosystem audit tools regularly (`npm audit`, `pip-audit`,
  `cargo audit`, `govulncheck`).

## File operations
- Use restrictive file permissions: 0o644 for regular files, 0o755 for
  executables. Never use 0o777.
- Create temporary files in designated temp directories with restricted
  permissions and random names.
- Validate file paths to prevent path traversal attacks. Never join
  user-supplied paths without sanitization.

## Cryptography
- Use well-maintained cryptographic libraries. Do not implement custom
  crypto algorithms.
- Avoid deprecated algorithms: no MD5, SHA1, DES, or RC4 for security
  purposes.
- Use cryptographically secure random number generators for tokens,
  keys, and nonces.
