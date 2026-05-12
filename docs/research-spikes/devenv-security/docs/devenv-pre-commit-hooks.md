# devenv.sh Pre-commit Hooks Documentation
- **Source**: https://devenv.sh/pre-commit-hooks/
- **Retrieved**: 2026-05-12

## Explicitly Listed Hooks

1. **shellcheck** - "lint shell scripts"
2. **mdsh** - "execute example shell from Markdown files"
3. **black** - "format Python code"
4. **ormolu** - Haskell code formatter (with package override capability)
5. **clippy** - Rust linter with settings support

## Important Limitation

The webpage states: "See the list of all available hooks" with a reference to "../reference/options/#git-hooks" but does not include the complete list on this page. The documentation excerpt focuses on setup and custom hook creation rather than a comprehensive hook inventory.

## Security-Related Hooks

The content does not mention any hooks specifically designed for:
- Security scanning
- Secrets detection
- Dependency auditing
- License checking
- SAST tools

## Custom Hook Capability

The documentation shows how to define custom hooks with properties like `entry`, `files`, `types`, and `excludes`, suggesting users can implement their own security-focused hooks, but no pre-built security hooks are detailed here.

To find the complete hook list, visit the referenced options reference page at `/reference/options/#git-hooks`.
