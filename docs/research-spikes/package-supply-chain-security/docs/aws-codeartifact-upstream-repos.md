# AWS CodeArtifact: Upstream Repositories and Package Origin Controls

- **Source URLs**:
  - https://docs.aws.amazon.com/codeartifact/latest/ug/repos-upstream.html
  - https://github.com/aws/codeartifact-origin-control-toolkit/blob/main/README.md
- **Retrieved**: 2026-05-12

## Upstream Repository Architecture

A repository can have other CodeArtifact repositories as upstream repositories, enabling single-endpoint access to multiple package sources. Up to 10 upstream repositories per CodeArtifact repository. Only one external connection (to a public registry) per repository.

If an upstream repository has an external connection to a public repository, downstream repositories can pull packages from that public repository transitively.

## Supported Package Formats

npm, PyPI, Maven, NuGet, Swift, Ruby, Cargo, generic packages. Repositories are polyglot — a single repository can contain packages of any supported type.

## Package Origin Controls

Security feature managing how packages are sourced and published:
- **Publish**: Controls whether new package versions can be created (ALLOW or BLOCK)
- **Upstream**: Controls whether packages can be obtained from upstream repositories (ALLOW or BLOCK)

Defends against dependency confusion attacks by blocking upstream access to internally-maintained packages.

**Critical caveat:** Packages created before the feature release are NOT protected without explicit configuration. New packages default to publish=ALLOW, upstream=ALLOW.

The origin control toolkit provides Python scripts for bulk application of policies across existing packages.

## Configuration for Developers

Package managers are configured to point at the CodeArtifact repository endpoint. Authentication uses short-lived tokens obtained via AWS CLI (`aws codeartifact login`).

## Pricing

Pay-as-you-go: $0.05/GB-month storage, $0.05 per 10,000 requests. Same-region AWS data transfer free. Always-free tier: 2 GB storage + 100,000 requests/month.

## Limitations

- Only one external connection per repository
- Package Origin Controls not retroactively applied to existing packages
- Tightly coupled to AWS ecosystem
- No built-in vulnerability scanning or malware detection
- No age-gating or quarantine features
