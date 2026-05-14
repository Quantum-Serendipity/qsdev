# GitHub Codespaces Prebuilds
- **Source**: https://docs.github.com/en/codespaces/prebuilding-your-codespaces/about-github-codespaces-prebuilds
- **Retrieved**: 2026-03-20
- **Note**: Content synthesized from web search results (WebFetch unavailable)

## How Prebuilds Work

A prebuild assembles the main components of a codespace for a particular combination of:
- Repository
- Branch
- devcontainer.json configuration file

### Build Process

1. A GitHub Actions workflow is triggered (on config change, push, or schedule)
2. GitHub creates a temporary codespace
3. Setup operations execute up through `onCreateCommand` and `updateContentCommand`
4. A container snapshot is saved to storage
5. When a user creates a codespace from a prebuild, GitHub:
   - Downloads the existing container snapshot
   - Deploys it on a fresh VM
   - Runs remaining commands (postCreateCommand, postStartCommand, postAttachCommand)

### Result

Creating a codespace from a prebuild can be substantially quicker — startup times are often reduced from many minutes to under one minute, regardless of repository size or complexity.

## Configuration Options

- **Trigger on config change**: Rebuild when devcontainer.json files change
- **Trigger on push**: Rebuild on every push to the prebuild-enabled branch
- **Scheduled**: Rebuild on a cron schedule only
- **Branch selection**: Choose which branches get prebuilds
- **Region selection**: Choose which regions store prebuild snapshots

Multiple devcontainer.json files supported for monorepo configurations.

## Key Limitations

### Repository Size
- Prebuilds are NOT available for 2-core and 4-core machine types if the repository is greater than 32 GB (storage limit)

### Concurrency
- Only one prebuild workflow can run at a time per configuration
- If changes are queued while a prebuild is running, the next run will handle them

### Build Times
- Prebuild creation times vary significantly:
  - Minimal projects: ~30 minutes
  - Complex projects (Rust with clippy, tests, release builds): ~180 minutes
- Prebuilds consume GitHub Actions minutes

### Storage Costs
- Each prebuild snapshot consumes storage at $0.07/GiB/mo
- Multiple branches x multiple regions = multiplied storage costs
- Stale prebuilds should be cleaned up

### Staleness
- If prebuilds aren't updated frequently, codespaces may still need to run setup steps
- Trade-off between freshness (more Actions minutes) and startup speed

## Multi-Repo and Monorepo Support

- Multiple devcontainer.json files supported: `.devcontainer/${DIR}/devcontainer.json`
- Prebuilding supported for multi-repository and monorepo project configurations
- Users can select the ideal devcontainer, machine type, and region during codespace creation

## Additional Sources

- https://docs.github.com/en/codespaces/prebuilding-your-codespaces/configuring-prebuilds
- https://docs.github.com/en/codespaces/prebuilding-your-codespaces/managing-prebuilds
- https://www.sitepoint.com/github-codespaces-prebuilds-ci-cd-optimization/
- https://github.blog/news-insights/product-news/codespaces-largest-repositories-faster/
