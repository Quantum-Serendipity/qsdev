# Maven Lockfile Plugin

- **Source**: https://github.com/chains-project/maven-lockfile
- **Retrieved**: 2026-05-12

## Purpose and Core Features

Maven Lockfile is a Maven plugin designed to enhance build integrity and security. It generates lockfiles containing checksums of all artifacts and dependencies, enabling validation of build environments and supporting reproducible rebuilds of past Java releases.

## How It Works

### Generating Lockfiles

Running `mvn io.github.chains-project:maven-lockfile:generate` creates a `lockfile.json` file per module capturing the complete dependency tree with transitive dependencies.

### Validation Process

The validate command `mvn io.github.chains-project:maven-lockfile:validate` checks that "all dependencies defined are still the same as when the lock file was generated."

### Freezing Dependencies

The freeze command generates a `pom.lockfile.xml` file where all dependency versions match the lockfile, with transitive dependencies added to the `dependencyManagement` section.

## Lockfile Format

The JSON lockfile includes:
- Project metadata (groupId, artifactId, version)
- POM checksum with algorithm specification
- Dependency tree with SHA-256 checksums (default)
- Resolved artifact URLs and repository information
- Maven plugin checksums (optional)
- Environment metadata (OS, Maven version, Java version)
- Configuration settings used during generation

## Configuration Options

Key flags:

| Flag | Purpose | Default |
|------|---------|---------|
| `reduced` | Minimize lockfile post-conflict resolution | false |
| `includeMavenPlugins` | Include Maven plugins in lockfile | false |
| `allowValidationFailure` | Warn instead of fail on mismatches | false |
| `includeEnvironment` | Capture environment metadata | false |
| `checksumAlgorithm` | Hash algorithm for verification | SHA-256 |

## CI/CD Integration via GitHub Actions

```yaml
- uses: chains-project/maven-lockfile@2d2ed1462246005ae3aafaf2d0bc619f521eadf6
  with:
    github-token: ${{ secrets.JRELEASER_GITHUB_TOKEN }}
    include-maven-plugins: true
```

When `commit-lockfile: false`, the action validates lockfile correctness and FAILS if dependencies have changed since lockfile generation.

## Important Warnings

- Action results can be platform-dependent (some artifacts have platform-dependent checksums)
- Pull requests from forks cannot commit lockfile updates without personal access tokens
- Maven has no native lockfile support; this is a third-party plugin
