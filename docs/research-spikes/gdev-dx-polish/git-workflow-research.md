# Git Workflow Automation Research

## Research Question

Beyond pre-commit hooks (already planned), commitlint (already planned), and git-cliff (already planned), what git workflow automation would streamline the developer experience for a consulting firm?

## Already Planned in gdev

Before identifying gaps, acknowledge what's covered:

1. **Pre-commit hooks** (Phase 5) -- linting, formatting, security scanning via prek/pre-commit
2. **Commitlint** (Phase 12) -- Conventional Commits enforcement
3. **git-cliff** (Phase 12) -- Changelog generation from commit history
4. **Gitleaks** (Phase 12) -- Secret detection in commits
5. **Branch protection** -- Implicitly via CI workflow generation

## Gap Analysis: What's Missing

### 1. Branch Naming Convention Enforcement

**Problem**: Consulting teams need consistent branch naming for client project tracking, time tracking integration, and PR automation. Without enforcement, branches end up as `fix-stuff`, `test`, `johns-branch`.

**Solution**: A `prepare-commit-msg` or custom git hook that validates branch names against a configurable pattern. Common patterns:
- `<type>/<ticket>-<description>` (e.g., `feat/ACME-123-add-login`)
- `<type>/<description>` (e.g., `fix/null-pointer-crash`)

**Implementation**: This is a natural extension of the pre-commit hook infrastructure. A simple regex check in a `pre-push` or even a custom `post-checkout` hook that warns (not blocks) on non-conforming names.

**Recommendation: Include.** Low cost, high signal. Generate a configurable branch naming pattern in devenv.nix git-hooks. Default pattern should be loose enough to not annoy (`^(feat|fix|chore|docs|refactor|test|ci)/[a-z0-9-]+$`) but strict enough to prevent garbage.

### 2. PR Template Generation

**Problem**: Every project needs a PR template but nobody creates one from scratch. The content should be ecosystem-aware (security checklist for security-hardened projects, test coverage for tested projects).

**Solution**: `qsdev init` generates `.github/pull_request_template.md` with sections:
- Summary (what changed and why)
- Type of change (feature/fix/refactor/etc.)
- Testing checklist (auto-populated based on detected test frameworks)
- Security checklist (when security hardening is enabled)
- Reviewer notes

**Recommendation: Include.** This is a static file generated once during `qsdev init`. Zero ongoing maintenance cost, immediate value. Already fits naturally into the file generation pipeline.

### 3. Commit Message Ticket Extraction

**Problem**: Teams want commit messages to reference ticket numbers (JIRA, Linear, GitHub Issues). Manual entry is error-prone.

**Solution**: A `prepare-commit-msg` hook that extracts the ticket number from the branch name and prepends it to the commit message. If branch is `feat/ACME-123-add-login`, the commit message gets `[ACME-123] ` prefixed automatically.

**Recommendation: Include as opt-in.** When branch naming enforcement is enabled and a ticket pattern is configured, auto-extract and prepend. This is 10 lines of shell script in a git hook.

### 4. Automated PR Labels

**Problem**: PRs need labels for categorization, priority triage, and changelog generation. Manual labeling is inconsistent.

**Solution**: A GitHub Action (or GitLab CI equivalent) that labels PRs based on:
- Changed file paths (e.g., `docs/**` -> `documentation` label)
- Commit types (e.g., `feat:` -> `enhancement` label)
- PR size (lines changed -> `size/S`, `size/M`, `size/L`)

**Recommendation: Include in CI workflow generation.** GitHub has `actions/labeler` which reads a `.github/labeler.yml` config. gdev can generate both the config and the workflow. Low cost, high utility for changelog and triage.

### 5. Merge Queue / Auto-Merge Configuration

**Problem**: Teams waste time manually merging PRs that have passed all checks.

**Solution**: Generate GitHub branch protection rules and merge queue configuration as part of CI setup.

**Recommendation: Do NOT include.** This is repository settings, not file generation. gdev should not manage GitHub API settings -- it generates files. Branch protection is best configured via Terraform/Pulumi (which gdev already supports as an ecosystem) or the GitHub UI.

### 6. Release Automation

**Problem**: Creating releases, tagging, bumping versions, generating changelogs -- the full release cycle.

**Solution**: git-cliff handles changelog. But version bumping (e.g., `npm version`, `cargo release`, Go tags) and GitHub Release creation are separate.

**Recommendation: Do NOT include now.** git-cliff + commitlint cover the hard parts. Full release automation (semantic-release, release-please) adds Node.js dependencies and significant complexity. The plan already rejected these tools (Phase 12 research finding #32). If needed later, it's a separate spike.

## What's Genuinely Missing (Summary)

| Feature | Value | Cost | Recommendation |
|---------|-------|------|----------------|
| Branch naming enforcement | High for teams | Low (regex hook) | **Include** |
| PR template generation | Medium-high | Very low (static file) | **Include** |
| Commit ticket extraction | Medium | Low (prepare-commit-msg hook) | **Include (opt-in)** |
| Automated PR labels | Medium | Low (labeler.yml + workflow) | **Include** |
| Merge queue config | Low (for gdev) | Medium (API, not files) | **Exclude** |
| Release automation | Medium | High (complexity) | **Exclude** |

## Depth Checklist

- [x] Underlying mechanism explained -- git hook types, CI workflow generation, PR templates
- [x] Key tradeoffs -- automation value vs configuration burden, file generation vs API management
- [x] Compared to alternatives -- each feature evaluated against manual workflow and existing tools
- [x] Failure modes -- branch naming too strict (blocks developers), ticket extraction wrong pattern, PR labels noisy
- [x] Concrete examples -- regex patterns, labeler.yml, prepare-commit-msg hook logic
- [x] Standalone-readable -- yes
