<!-- Source: https://graphite.com/guides/git-commit--no-verify -->
<!-- Retrieved: 2026-05-15 -->

# Git Commit --no-verify: Functionality and Considerations

## What It Does

The `--no-verify` flag bypasses pre-commit and pre-push hooks when executing a git commit. As the guide explains, "Git typically runs pre-commit hooks—scripts that inspect your changes before the commit is allowed" and this option "bypasses these hooks, letting you commit changes without the checks usually performed."

## When It's Appropriate

The guide identifies three legitimate scenarios:

1. **Urgent fixes** - When quick rollbacks or patches are needed and hook delays are problematic
2. **Tool malfunctions** - Temporary issues with configured scripts or linting tools
3. **Work-in-progress saves** - Intermediate commits that preserve state without triggering validation checks

## Security and Quality Concerns

The guide emphasizes restraint, noting that this capability "should be used sparingly and judiciously. Regularly skipping hooks can defeat the purpose of having quality checks in place."

The resource stresses that "final commits, especially those merged into main development branches or deployed, undergo full checks to maintain code quality."

## Better Approaches

Rather than habitual bypassing, the guide recommends:

- **Team communication** - Explain bypass reasons to collaborators, especially for shared branches
- **Fixing root causes** - Address broken hooks rather than circumvent them
- **Selective use** - Reserve `--no-verify` for genuine exceptions, not routine workflow

The underlying message: bypassing verification mechanisms should remain exceptional, not normalized.
