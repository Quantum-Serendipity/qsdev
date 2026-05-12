<!-- Source: https://docs.socket.dev/docs/faq -->
<!-- Retrieved: 2026-05-12 -->

# Socket Security Platform: Core Details

## How Socket Works

Socket employs three complementary analysis techniques:

**Static Analysis**: Socket examines source code without execution to identify suspicious patterns. As their documentation states, they look for "new install scripts, network requests, environment variable access, telemetry, suspicious strings, obfuscated code" and similar indicators. They use a custom in-house static analysis engine across the npm ecosystem and plan to extend this to PyPI, Go, Maven, and other ecosystems.

**Package Metadata Analysis**: This involves examining how packages load and distribute code. Socket detects when packages "load code from a remote git repository or a remote HTTP server, totally bypassing your package lockfile," enabling attackers to serve different code to different users.

**Maintainer Behavior Analysis**: Socket monitors "who is maintaining the package and their activity history," flagging unmaintained packages, trivial packages, and those with "major refactors" or "unstable ownership" (new maintainer permissions granted).

## What Socket Detects

Socket identifies 70+ security signals across multiple categories:

**Supply Chain Attacks**:
- Known malware
- Typosquatting (using Levenshtein distance and download count comparisons)
- AI-detected malware and security risks
- Git/GitHub/HTTP dependencies
- Obfuscated code
- Protestware/unwanted behavior

**Behavioral Red Flags**:
- Network access via fetch(), Node's net/dgram/dns/http/https modules
- Shell access
- Filesystem access
- Environment variable access
- Dynamic require/eval usage
- Native code execution

**Quality & Maintenance Issues**:
- Deprecated or unmaintained packages
- Minified code
- Unpopular packages
- New authors

**License Concerns**:
- Copyleft licenses
- Unlicensed items
- Non-permissive licenses

## Supported Ecosystems

Socket currently supports "JavaScript, Python, Java, Ruby, .NET, Go, Rust, Scala, Kotlin and more," with PHP, Swift, and Objective-C in active development.

## Integration Points

**GitHub App**: Automatically evaluates package.json changes in pull requests and comments on dependency security risks.

**CLI**: Available via PyPI package installation for command-line integration.

**CI/CD Platforms**: Direct integrations for GitHub Actions, GitLab Pipeline, Bitbucket Pipeline, Jenkins, and Azure DevOps.

**Additional Tools**: REST API, JavaScript SDK, Python SDK, VS Code extension, Chrome extension, Slack integration, Jira integration, and webhooks.

## Pricing Model

- Package search and Health Scores: Free for all users
- Open source repositories: Free indefinitely
- Private repositories: First repository free, then paid tier required

## Important Limitations

Socket explicitly states it "does not analyze, upload, or share customer source code." They only examine publicly available open source dependencies and manifest files (package.json, requirements.txt), not proprietary code. All communication uses HTTPS encryption.
