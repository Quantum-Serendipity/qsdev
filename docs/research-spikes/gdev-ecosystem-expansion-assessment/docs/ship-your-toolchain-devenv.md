# Ship Your Toolchain, Not Just Infrastructure

- **Source**: https://www.maxdaten.io/2026-01-31-ship-your-toolchain-not-just-infrastructure
- **Retrieved**: 2026-05-14

## Core Concept

Jan-Philip Loos advocates treating platform tooling as version-controlled, declarative software rather than wiki pages and Slack announcements. As he states: "Platform teams deliver Terraform modules via registries...But for their daily work, product teams need CLI tools...Those ship as wiki pages and Slack announcements."

## The Problem: Version Drift

The article highlights real consequences of mismatched tool versions. A concrete example involves OpenSSL certificate generation -- incorrect versions produce incompatible certificates despite validation scripts. Similarly, kubectl clients can fail when version skew with Kubernetes servers exceeds acceptable ranges.

## Devenv Solution

Built atop Nix, devenv provides a simplified approach to reproducible development environments without requiring deep Nix expertise from consumers. Key capabilities include:

- **Tool version locking**: Ensures uniform `kubectl`, `terraform`, and AWS CLI versions across teams
- **Custom script distribution**: Bundles platform-specific automation alongside standard tooling
- **Native shell integration**: Tools appear in PATH without containerization overhead

## Configuration Architecture

The example demonstrates a GCP/Kubernetes platform setup with this structure:

```
platform-env/
├── devenv.nix
├── devenv.lock
└── modules/
    ├── google-cloud.nix
    └── scripts/gcp-costs.sh
```

The `google-cloud.nix` module offers configurable options like `kubernetesNamespace`, automatically setting developer contexts on shell activation. It pins versions for `gke-gcloud-auth-plugin`, `kubectl`, `helm`, and related tools.

## Consumption Models

**Model A** requires zero Nix knowledge -- developers clone the repository and invoke devenv with command-line flags:

```bash
devenv shell \
  --option google-cloud.enable:bool true \
  --option google-cloud.kubernetesNamespace:string "team-namespace"
```

**Model B** allows teams already using devenv to import platform modules into their own configurations, simplifying invocations.

## Trade-offs

Platform maintainers face learning curve challenges with Nix debugging and complexity. Initial developer setup requires installing Nix/devenv, and first-shell entry can consume 1-20+ minutes building dependencies. Mitigation strategies include leveraging binary caches, Cachix services, or organizational S3-compatible storage.

## Extended Value Delivery

Beyond basic tooling, platforms can ship version notifications, security compliance hooks, onboarding automation, and AI coding agent integrations through devenv's extensibility framework.
