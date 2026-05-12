# Google Artifact Registry: Remote Repositories Overview

- **Source URL**: https://docs.cloud.google.com/artifact-registry/docs/repositories/remote-overview
- **Retrieved**: 2026-05-12

## How Caching Works

Remote repositories function as proxies that download and cache packages upon first request. Subsequently, Artifact Registry serves the cached copy for identical package versions.

## Supported Upstream Sources

**Preset Options:**

| Format | Upstream | Preset Name |
|--------|----------|-------------|
| Docker | `https://registry-1.docker.io` | DOCKER-HUB |
| Go | `https://proxy.golang.org` | (URL-based) |
| Maven | `https://repo.maven.apache.org/maven2` | MAVEN-CENTRAL |
| npm | `https://registry.npmjs.org` | NPMJS |
| Python | `https://pypi.io` | PYPI |

Custom upstream support for Docker, npm, Maven, and Python (connecting to GitHub Container Registry, AWS ECR, Nexus instances, etc.).

## Supported Formats

Docker containers, Go modules, Maven packages, npm packages, Python packages, OS packages (Apt/Yum in preview).

## Security Features

**Dependency Confusion Mitigation:** Virtual repositories can prioritize private repositories over remote repositories.

**VPC Service Controls:** If Artifact Registry is in a VPC Service Controls perimeter, access to upstream sources outside the perimeter is denied by default.

**Vulnerability Scanning:** Container Scanning API enables automatic scanning of images in standard and remote Docker repositories.

## Metadata Update Policy

Maven metadata: 5 minutes. npm manifests: 5 minutes. Python indexes: hourly. Docker tag caches: hourly.

## Limitations

- Upstream sources must be internet-accessible (no on-premise/VPC sources without public IP)
- Maven repositories cannot use snapshot or release version policies
- Scanning primarily focused on container images

## Pricing

Pay-as-you-go based on storage and network egress. Free tier available across billing account. Network egress charges can accumulate significantly in high-traffic scenarios.
