# Contributing to qsdev

Contributions are welcome. If you have ideas for new tools, plugins, ecosystem modules, configuration presets, or better defaults — I'd love to hear them.

## What's useful

- New or improved ecosystem modules (languages, frameworks, package managers)
- Security hardening configs for tools qsdev doesn't cover yet
- Better defaults and best practices for existing ecosystems
- Claude Code skills, hooks, deny rules, and MCP server configs
- devenv.sh service definitions and package sets
- Pre-commit hooks and linting configs
- Performance and ergonomic improvements to the CLI
- Documentation fixes and clarifications

## How to contribute

**Got an idea?** Open an issue. Describe what you'd like to see and why it's useful. Even rough ideas are fine — we can figure out the shape together.

**Want to submit code?** Fork the repo, make your changes on a branch, and open a PR. Keep PRs focused on one thing. Include a short description of what changed and why.

## Development setup

qsdev uses itself for development environment management.

**Prerequisites**: [qsdev](https://github.com/Quantum-Serendipity/qsdev) and
[devenv](https://devenv.sh) must be installed. On NixOS, the qsdev module
provides both.

```bash
git clone https://github.com/Quantum-Serendipity/qsdev.git
cd qsdev
direnv allow   # activates the devenv environment
go build ./...
go test ./...
```

The devenv environment provides Go tooling, pre-commit hooks, and security
scanning. Run `go vet ./...` and `golangci-lint run` before submitting.

Without direnv: `devenv shell` for manual activation.

## Guidelines

- Keep changes focused. One PR, one concern.
- Follow existing code style and patterns.
- Add tests for new functionality.
- Don't introduce copyleft dependencies. Apache-2.0, MIT, BSD, and ISC are fine.
- Security-sensitive changes should note the threat model impact.

## License

By contributing, you agree that your contributions will be licensed under the [Apache-2.0 License](LICENSE).
