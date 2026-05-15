# Cloud Provider CLIs, Credential Management & Kubernetes Tooling for gdev

## Executive Summary

gdev currently has **zero coverage** of cloud provider CLIs and Kubernetes tooling (beyond Helm as a language ecosystem module). This is the single largest gap for a consulting-oriented developer environment tool. Consulting engineers routinely work across multiple cloud providers and Kubernetes clusters simultaneously, often switching between clients weekly or daily. This report catalogs every relevant tool, assesses its Nixpkgs availability, configuration complexity, detection heuristics, and consulting-specific considerations, then recommends a tiered integration strategy.

**Key finding**: Every major tool in this space is available in Nixpkgs and can be installed declaratively via devenv.sh. The hard problem is not installation but **credential management and multi-client isolation** -- areas where gdev can provide significant value beyond what raw `nix develop` offers.

---

## 1. Cloud Provider CLIs

### 1.1 AWS CLI v2

| Attribute | Value |
|-----------|-------|
| **Nixpkgs package** | `awscli2` (v2.31.39) |
| **Config complexity** | Medium -- needs profiles, regions, SSO config |
| **Detection heuristic** | `*.tf` with `provider "aws"`, `serverless.yml`, `sam.yaml`, `cdk.json`, `.aws/` dir, `AWS_*` env vars, `buildspec.yml` |
| **Consulting impact** | Critical -- most common cloud provider in enterprise consulting |

**What it provides**: `aws` CLI for all AWS service interactions. Standard auth via `~/.aws/credentials` and `~/.aws/config` with named profiles.

**Authentication patterns**:
- **IAM Access Keys** (legacy): Static credentials stored in `~/.aws/credentials`. Simple but insecure for long-term use.
- **AWS SSO / IAM Identity Center** (modern standard): `aws sso login --profile <name>`. Short-lived tokens, browser-based auth. Requires `sso_start_url`, `sso_region`, `sso_account_id`, `sso_role_name` in profile config.
- **AssumeRole with MFA**: Role chaining from a base profile. Supports MFA device serial and session token caching.
- **Environment variables**: `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN`, `AWS_PROFILE`, `AWS_DEFAULT_REGION`, `AWS_REGION`.

**Multi-account strategy**: Named profiles in `~/.aws/config` with `[profile client-a-dev]`, `[profile client-a-prod]`, `[profile client-b-staging]` etc. Each profile can have independent SSO config, role ARN, region, and output format. The `AWS_PROFILE` environment variable selects the active profile.

**gdev recommendation**: Install `awscli2`. Do NOT manage credentials/profiles (that's per-engineer, per-client). Optionally detect AWS usage and remind engineers to configure profiles.

### 1.2 AWS Credential Helpers

These tools layer on top of the AWS CLI to solve credential security and ergonomics problems. They are especially valuable for consulting where engineers manage credentials for multiple client AWS accounts.

#### aws-vault

| Attribute | Value |
|-----------|-------|
| **Nixpkgs package** | `aws-vault` (v7.7.10) |
| **Config complexity** | Low -- wraps existing AWS config |
| **What it solves** | Credentials never touch disk in plaintext; stored in OS keychain |

- Executes commands with temporary credentials: `aws-vault exec client-a -- terraform plan`
- Supports SSO: `aws-vault login client-a` opens browser for SSO auth
- MFA session caching eliminates repeated MFA prompts across profiles sharing the same MFA serial
- Backend storage: macOS Keychain, Pass (Linux), encrypted file, KWallet
- **Linux note**: On NixOS/Linux, the `pass` or `file` backend is typical since there's no macOS Keychain. `AWS_VAULT_BACKEND=pass` or `AWS_VAULT_BACKEND=file`.
- Server modes: ECS metadata server for Docker containers, EC2 metadata server for desktop apps
- **Consulting pattern**: One vault entry per client, `aws-vault exec client-x -- <command>` ensures credential isolation

#### saml2aws

| Attribute | Value |
|-----------|-------|
| **Nixpkgs package** | `saml2aws` (v2.36.19) |
| **Config complexity** | Medium -- needs IdP configuration |
| **What it solves** | SAML-based SSO for orgs using Okta, ADFS, OneLogin, etc. |

- Automates SAML login flow for enterprise IdPs
- Generates temporary AWS credentials from SAML assertions
- Supports 15+ IdP types: Okta, ADFS, Azure AD, OneLogin, Ping, KeyCloak, Google Apps
- **Consulting pattern**: Clients using enterprise SAML often require saml2aws; engineers may need different IdP configs per client

#### aws-sso-cli

| Attribute | Value |
|-----------|-------|
| **Nixpkgs package** | `aws-sso-cli` (v2.1.0) |
| **Config complexity** | Low-Medium |
| **What it solves** | Enhanced AWS SSO experience beyond stock CLI |

- Can authenticate to the same SSO instance as two different users simultaneously
- Better profile management and credential caching than stock `aws sso`
- Complements or replaces `aws-sso-util`

#### Leapp

| Attribute | Value |
|-----------|-------|
| **Nixpkgs package** | NOT in nixpkgs (Electron app) |
| **Config complexity** | GUI-based |
| **What it solves** | Visual credential manager for AWS/Azure with auto-rotation |

- Rotates credentials every 20 minutes via STS
- Never writes long-term credentials to `~/.aws/credentials`
- GUI application -- not well suited for CLI-first devenv integration
- **gdev recommendation**: Document as optional, do not integrate

### 1.3 GCP CLI (Google Cloud SDK)

| Attribute | Value |
|-----------|-------|
| **Nixpkgs package** | `google-cloud-sdk` (v537.0.0) |
| **Config complexity** | Medium -- needs project, auth, and component config |
| **Detection heuristic** | `*.tf` with `provider "google"`, `app.yaml` (App Engine), `cloudbuild.yaml`, `.gcloudignore`, `GOOGLE_*` env vars |
| **Consulting impact** | High -- second most common cloud provider |

**Included tools**: `gcloud`, `gsutil` (Cloud Storage), `bq` (BigQuery). Additional components installable via `gcloud components install`.

**Authentication patterns**:
- **User credentials** (local dev): `gcloud auth login` opens browser for Google account auth
- **Application Default Credentials (ADC)**: `gcloud auth application-default login` -- the standard for local development. Client libraries use ADC automatically.
  - Discovery chain: `GOOGLE_APPLICATION_CREDENTIALS` env var → `gcloud auth application-default` user creds → compute metadata (GCE/GKE)
- **Service account impersonation**: `gcloud auth application-default login --impersonate-service-account=SA_EMAIL` -- preferred over downloading service account keys
- **Workload Identity Federation**: For external workloads authenticating to GCP without service account keys. Uses OIDC/SAML tokens from external IdPs.
- **GKE auth**: `gke-gcloud-auth-plugin` for IAM-based Kubernetes auth (required for GKE clusters)

**Key environment variables**: `GOOGLE_APPLICATION_CREDENTIALS`, `GOOGLE_CLOUD_PROJECT`, `CLOUDSDK_CORE_PROJECT`, `CLOUDSDK_COMPUTE_REGION`

**Multi-project strategy**: `gcloud config configurations` -- named configurations with different project, account, region settings. `gcloud config configurations activate <name>` switches context.

**Nixpkgs note**: The `google-cloud-sdk` package includes gcloud, gsutil, and bq. Additional components (like `gke-gcloud-auth-plugin`) need to be added via `google-cloud-sdk.withExtraComponents` in Nix or installed separately.

**gdev recommendation**: Install `google-cloud-sdk`. Consider including `gke-gcloud-auth-plugin` when K8s + GCP detected.

### 1.4 Azure CLI

| Attribute | Value |
|-----------|-------|
| **Nixpkgs package** | `azure-cli` (v2.79.0) |
| **Config complexity** | Medium |
| **Detection heuristic** | `*.tf` with `provider "azurerm"`, `azure-pipelines.yml`, `.azure/`, `ARM_*` env vars, `bicep` files |
| **Consulting impact** | High -- especially in enterprise/government consulting |

**Two CLIs**:
- **`az`** (Azure CLI): General-purpose Azure management. `az login` for browser-based auth.
- **`azd`** (Azure Developer CLI): Higher-level developer workflow tool. `azd auth login`, `azd up` for full provisioning + deploy. NOT in nixpkgs as a separate package (ships with azure-cli or installed separately).

**Authentication patterns**:
- **Interactive login**: `az login` opens browser
- **Device code**: `az login --use-device-code` for headless systems
- **Service principal**: `az login --service-principal -u CLIENT_ID -p CLIENT_SECRET --tenant TENANT_ID`
- **Managed identity**: `az login --identity` (on Azure VMs only)
- **Environment variables**: `ARM_CLIENT_ID`, `ARM_CLIENT_SECRET`, `ARM_SUBSCRIPTION_ID`, `ARM_TENANT_ID` (Terraform pattern)

**Multi-subscription**: `az account set --subscription <name-or-id>`. `az account list` shows all accessible subscriptions.

**gdev recommendation**: Install `azure-cli`. The `azd` CLI is a separate concern -- install if `azure.yaml` detected in project.

### 1.5 Other Cloud CLIs

| Tool | Nixpkgs Package | Version | Detection Heuristic | Consulting Relevance |
|------|----------------|---------|--------------------|--------------------|
| Cloudflare Wrangler | `wrangler` | v4.62.0 | `wrangler.toml`, `wrangler.jsonc` | Medium -- Workers/Pages projects |
| DigitalOcean doctl | `doctl` | v1.147.0 | `.do/` dir, `DIGITALOCEAN_TOKEN` | Low-Medium -- startup clients |
| Fly.io flyctl | `flyctl` | v0.3.209 | `fly.toml` | Low-Medium -- indie/startup |
| Vercel CLI | `vercel` (nodePackages) | v41.4.1 | `vercel.json`, `.vercel/` | Medium -- Next.js projects |
| Netlify CLI | `netlify-cli` | v23.9.2 | `netlify.toml`, `.netlify/` | Medium -- Jamstack projects |

**gdev recommendation**: These are project-specific. Install only when detection heuristic matches. Do NOT include in default toolset. Treat as ecosystem modules similar to language toolchains.

---

## 2. Kubernetes & Container Tools

### 2.1 kubectl

| Attribute | Value |
|-----------|-------|
| **Nixpkgs package** | `kubectl` (v1.34.3) |
| **Config complexity** | Medium -- kubeconfig management is the challenge |
| **Detection heuristic** | `k8s/`, `kubernetes/`, `*.yaml` with `apiVersion:`, `Dockerfile` + `deployment.yaml`, Helm charts, `kustomization.yaml` |
| **Consulting impact** | Critical -- nearly universal for K8s-using clients |

**Version management challenge**: kubectl has a version skew policy -- client must be within +/-1 minor version of the cluster API server. Consulting engineers may need different kubectl versions for different client clusters. Devenv.sh solves this by pinning kubectl version per-project in `devenv.nix`.

**kubeconfig management**:
- Default file: `~/.kube/config`
- Multiple files: `KUBECONFIG` env var supports colon-separated paths (e.g., `KUBECONFIG=~/.kube/client-a:~/.kube/client-b`)
- Contexts: Named bindings of cluster + user + namespace. `kubectl config use-context <name>` switches.
- **Best practice for consulting**: Separate kubeconfig files per client (credential isolation), merged via `KUBECONFIG` env var. Use `--context` flag for one-off commands to avoid accidental context switches.
- Naming convention: `<client>-<env>-<region>` (e.g., `acme-prod-us-east-1`)

**Authentication plugins**:
- `kubelogin` (nixpkgs: `kubelogin`, v0.2.13): Azure AD / AAD auth for AKS clusters
- `kubelogin-oidc` (nixpkgs: `kubelogin-oidc`, v1.34.2): Generic OIDC auth
- `gke-gcloud-auth-plugin`: GKE IAM auth (part of google-cloud-sdk)
- `aws-iam-authenticator`: EKS IAM auth (typically bundled with awscli2)

**gdev recommendation**: Install `kubectl`. This is a tier-1 must-ship tool for any K8s-aware project. Consider version pinning guidance in devenv.nix.

### 2.2 Context & Namespace Switching

| Tool | Nixpkgs Package | Version | Purpose |
|------|----------------|---------|---------|
| kubectx | `kubectx` | v0.9.5 | Fast context switching (`kubectx <name>`) + fuzzy selection |
| kubens | (included in `kubectx`) | v0.9.5 | Fast namespace switching (`kubens <name>`) |

**Why these matter for consulting**: Engineers switching between client clusters multiple times per day need fast, safe context switching. `kubectx` provides:
- Interactive fuzzy selection (with fzf)
- `kubectx -` to switch to previous context (like `cd -`)
- Clear display of current context
- kubens does the same for namespaces

**gdev recommendation**: Install alongside kubectl. Minimal footprint, high daily-use value.

### 2.3 Helm v3 (as K8s deployment tool)

| Attribute | Value |
|-----------|-------|
| **Nixpkgs package** | `kubernetes-helm` (v3.19.1) |
| **Config complexity** | Medium -- repos, plugins, secrets |
| **Detection heuristic** | `Chart.yaml`, `charts/`, `helmfile.yaml`, `values.yaml` |

devenv.sh already has a `languages.helm` module providing:
- Helm binary (`pkgs.kubernetes-helm`)
- Helm Language Server (`pkgs.helm-ls`)
- Plugin management via `languages.helm.plugins` (symlinked, exposed via `HELM_PLUGINS`)
- Supported plugins: `helm-secrets`, `helm-diff`, `helm-unittest`

**Additional Helm ecosystem tools**:

| Tool | Nixpkgs Package | Version | Purpose |
|------|----------------|---------|---------|
| helmfile | `helmfile` | v1.1.9 | Declarative Helm release management |
| helm-secrets | (via kubernetes-helmPlugins) | -- | SOPS/age/GPG encrypted values |
| helm-diff | (via kubernetes-helmPlugins) | -- | Preview changes before apply |

**gdev recommendation**: Expand the existing Helm language module to also function as a K8s deployment tool context. Include `helmfile` when `helmfile.yaml` detected.

### 2.4 Kustomize

| Attribute | Value |
|-----------|-------|
| **Nixpkgs package** | `kustomize` (v5.8.0) |
| **Config complexity** | Low -- standalone binary |
| **Detection heuristic** | `kustomization.yaml`, `kustomization.yml`, `kustomize/` dir |

**Standalone vs kubectl-integrated**: `kubectl` bundles an older kustomize (`kubectl apply -k`), but the standalone binary is typically newer and has more features. Recommend installing standalone when `kustomization.yaml` detected.

**gdev recommendation**: Install when detected. Low-cost addition.

### 2.5 K8s Development Tools

These tools automate the build-push-deploy cycle for local K8s development. They are project-specific rather than universally needed.

| Tool | Nixpkgs | Version | Config File | Best For |
|------|---------|---------|-------------|----------|
| Skaffold | `skaffold` | v2.16.1 | `skaffold.yaml` | Google ecosystem, CI/CD integration, mature community |
| Tilt | `tilt` | v0.35.2 | `Tiltfile` | Multi-service visualization, less K8s experience on team |
| DevSpace | `devspace` | v6.3.18 | `devspace.yaml` | Fast feedback loops, senior K8s teams |
| Telepresence | `telepresence2` | v2.25.1 | N/A (CLI flags) | Intercepting remote cluster traffic locally |
| Garden | NOT in nixpkgs | -- | `garden.yml` | Complex microservice dependency graphs |

**Comparison summary** (from research):
- **Skaffold**: CLI-only, YAML config, mature Google-backed project. Best for distributed teams, CI/CD pipeline integration. Supports Helm, kustomize, kubectl deployments.
- **Tilt**: Browser-based dashboard, Starlark (Python-like) config. Best observability of build/deploy status. Steeper learning curve due to Tiltfile language.
- **DevSpace**: Easiest setup, standard YAML config. Strong file sync and hot-reload. Best for experienced K8s teams wanting fast inner-loop.
- **Telepresence**: Different category -- intercepts traffic from a remote cluster to local machine. Useful for debugging against staging/prod data without full local cluster.
- **Garden**: Not in nixpkgs, complex, best for large microservice architectures. Skip for gdev.

**gdev recommendation**: Install based on config file detection only. These are too opinionated to default-install. Skaffold and Tilt are the most common; DevSpace is rising.

### 2.6 K8s Observability Tools

| Tool | Nixpkgs | Version | Purpose | Daily-Use Value |
|------|---------|---------|---------|----------------|
| k9s | `k9s` | v0.50.16 | Terminal UI for cluster management | **Very high** -- the kubectl replacement for daily work |
| stern | `stern` | v1.33.0 | Multi-pod log tailing with color coding | High -- essential for debugging |
| kubetail | `kubetail` | v1.6.22 | Log tailing with web dashboard option | Medium -- alternative to stern |

**k9s** is the standout tool here. It provides:
- Real-time cluster visualization in terminal
- Navigate namespaces, view logs, describe resources, edit manifests
- Port-forwarding, exec into pods, delete resources
- Custom resource views and filtering
- **Consulting value**: Quick cluster exploration when onboarding to a new client's infrastructure

**stern** provides:
- Regex-based pod filtering across multiple pods
- Automatic tracking of pod creation/deletion
- Color-coded output per pod/container
- Works in shell scripts and piped workflows

**gdev recommendation**: k9s and stern are tier-1 (install with kubectl). kubetail is tier-3 (nice-to-have).

### 2.7 K8s Security Tools

| Tool | Nixpkgs | Version | Focus | CNCF Status |
|------|---------|---------|-------|-------------|
| kubescape | `kubescape` | v3.0.45 | Comprehensive posture management | Incubating |
| kube-bench | `kube-bench` | v0.14.0 | CIS Benchmark compliance | -- |
| trivy | `trivy` | v0.66.0 | All-in-one vulnerability scanner | -- |
| polaris | `polaris` | v0.14.3 | Best-practice validation | -- |
| kube-linter | `kube-linter` | v0.7.6 | Static analysis of K8s YAML/Helm | -- |
| checkov | `checkov` | v3.2.495 | IaC scanner (K8s, Helm, Terraform) | -- |

**Tool selection guide**:
- **kubescape**: Most comprehensive -- covers CI/CD scanning through runtime detection. 260+ controls, NSA-CISA/MITRE ATT&CK/CIS frameworks. Runtime reachability analysis reduces CVE noise by ~90%.
- **kube-bench**: Focused CIS Kubernetes Benchmark auditing. Checks API server flags, etcd settings, file permissions. Deeper on cluster component config than other tools.
- **trivy**: Already in gdev plan (as Grype alternative). Covers images, filesystems, Git repos, K8s clusters, IaC. Generates SBOMs.
- **polaris**: Validates K8s resource configs against best practices. Good for CI/CD gates.
- **kube-linter**: Fast static analysis for K8s YAML and Helm charts. Designed for PR checks.
- **checkov**: Broader IaC scanner. Overlaps with Semgrep's IaC scanning.

**gdev recommendation**: kubescape is the best single tool for K8s security posture. kube-linter for pre-commit/CI. trivy already covers vulnerability scanning. Tier-2 for all -- install when K8s manifests detected.

### 2.8 Service Mesh CLIs

| Tool | Nixpkgs | Version | Purpose |
|------|---------|---------|---------|
| istioctl | `istioctl` | v1.28.0 | Istio service mesh management |
| linkerd | `linkerd` | stable-2.14.9 | Linkerd service mesh management |

**Assessment**: Service mesh CLIs are niche. Most consulting projects don't use service meshes, and those that do are already deeply committed to one. Linkerd is simpler and lighter; Istio is more feature-complete but complex.

**gdev recommendation**: Tier-3 at best. Install only when `istio` or `linkerd` references detected in project manifests. Not worth default-installing.

### 2.9 Container Tools (Beyond Docker)

| Tool | Nixpkgs | Version | Purpose |
|------|---------|---------|---------|
| podman | `podman` | v5.7.0 | Daemonless, rootless container engine |
| buildah | `buildah` (wrapper) | v1.42.1 | OCI image building without daemon |
| skopeo | `skopeo` | v1.20.0 | Image inspection and copying between registries |
| crane | `go-containerregistry` | v0.20.6 | Fast image manipulation CLI |
| dive | `dive` | v0.13.1 | Image layer analysis and optimization |
| hadolint | `hadolint` | v2.14.0 | Dockerfile linter (already in gdev plan) |

**Podman vs Docker**: Podman is daemonless and rootless, making it more suitable for environments where Docker daemon is not wanted or available. On NixOS specifically, Podman is well-supported via `virtualisation.podman`. For consulting, Docker is still the standard, but Podman provides compatibility (`alias docker=podman` works for most workflows).

**gdev recommendation**: hadolint already covered. dive is high-value for any containerized project (install when Dockerfile detected). Podman/buildah/skopeo are alternatives to Docker -- install based on project preference detection. crane is specialized (tier-3).

---

## 3. Integration Patterns

### 3.1 How devenv.sh Users Currently Handle Cloud CLIs

Based on research (the "Ship Your Toolchain" article and devenv-k8s project):

**Pattern A: Package list in devenv.nix**
```nix
{ pkgs, ... }: {
  packages = [
    pkgs.awscli2
    pkgs.google-cloud-sdk
    pkgs.kubectl
    pkgs.k9s
    pkgs.kubectx
    pkgs.stern
  ];
}
```

**Pattern B: Custom modules** (as demonstrated by maxdaten.io)
```
platform-env/
├── devenv.nix
├── devenv.lock
└── modules/
    ├── google-cloud.nix   # Configurable GCP options
    └── scripts/gcp-costs.sh
```
Modules offer configurable options (e.g., `google-cloud.enable`, `google-cloud.kubernetesNamespace`) and set environment variables on shell activation.

**Pattern C: Reusable flake modules** (LCOGT/devenv-k8s)
Import a shared K8s toolchain as a flake input:
```nix
inputs = {
  devenv-k8s.url = "github:LCOGT/devenv-k8s/v1";
};
```

**Pattern D: Helm with plugins** (NixOS Wiki pattern)
```nix
packages = [
  (pkgs.wrapHelm pkgs.kubernetes-helm {
    plugins = with pkgs.kubernetes-helmPlugins; [
      helm-secrets
      helm-diff
      helm-s3
    ];
  })
];
```

### 3.2 devenv.nix Patterns for gdev

gdev should provide devenv.sh modules (not just package lists) for cloud/K8s tooling. The module approach allows:
- Conditional inclusion based on detection
- Version pinning per project
- Environment variable setup (AWS_PROFILE, KUBECONFIG, etc.)
- Shell hooks for auth reminders or context display

**Proposed module structure**:
```nix
# gdev cloud module
cloud.aws.enable = true;       # Installs awscli2, aws-vault
cloud.aws.credentialHelper = "aws-vault";  # or "saml2aws", "aws-sso-cli"
cloud.gcp.enable = true;       # Installs google-cloud-sdk
cloud.gcp.gkeAuth = true;      # Adds gke-gcloud-auth-plugin
cloud.azure.enable = true;     # Installs azure-cli

# gdev kubernetes module  
kubernetes.enable = true;       # kubectl, kubectx, kubens
kubernetes.observability = true; # k9s, stern
kubernetes.security = true;     # kubescape, kube-linter
kubernetes.helm.enable = true;  # Extends existing helm module
kubernetes.helm.plugins = ["helm-secrets" "helm-diff"];
kubernetes.development = "skaffold";  # or "tilt", "devspace", null
```

### 3.3 Detection Heuristics

gdev should detect which cloud/K8s tools a project needs by scanning for indicator files.

**Cloud provider detection**:

| Provider | Detection Files |
|----------|----------------|
| AWS | `*.tf` containing `provider "aws"`, `serverless.yml`, `samconfig.toml`, `cdk.json`, `buildspec.yml`, `appspec.yml`, `.aws-sam/`, `template.yaml` (SAM) |
| GCP | `*.tf` containing `provider "google"`, `app.yaml`, `cloudbuild.yaml`, `.gcloudignore`, `firebase.json` |
| Azure | `*.tf` containing `provider "azurerm"`, `azure-pipelines.yml`, `*.bicep`, `azure.yaml` (azd), `.azure/` |
| Cloudflare | `wrangler.toml`, `wrangler.jsonc` |
| Fly.io | `fly.toml` |
| Vercel | `vercel.json`, `.vercel/` |
| Netlify | `netlify.toml`, `.netlify/` |
| DigitalOcean | `.do/app.yaml` |

**Kubernetes detection**:

| Tool Need | Detection Files |
|-----------|----------------|
| kubectl (core) | `k8s/`, `kubernetes/`, `**/deployment.yaml` with `apiVersion:`, `kustomization.yaml`, `Chart.yaml`, `skaffold.yaml`, `Tiltfile`, `devspace.yaml` |
| Helm | `Chart.yaml`, `charts/`, `helmfile.yaml` |
| Kustomize | `kustomization.yaml`, `kustomization.yml` |
| Skaffold | `skaffold.yaml` |
| Tilt | `Tiltfile` |
| DevSpace | `devspace.yaml` |

**Terraform provider parsing**: The most reliable detection is parsing `*.tf` files for `required_providers` blocks:
```hcl
terraform {
  required_providers {
    aws = { source = "hashicorp/aws" }
    google = { source = "hashicorp/google" }
    azurerm = { source = "hashicorp/azurerm" }
  }
}
```

### 3.4 Should gdev Configure Credentials or Just Install Tools?

**Recommendation: Install tools + provide scaffolding, but do NOT manage credentials.**

Rationale:
- Credentials are per-engineer, per-client, and often governed by client security policies
- SSO configurations contain client-specific URLs and account IDs
- Credential misconfiguration is a security risk -- better to fail safe
- aws-vault backend choice depends on OS and user preference

**What gdev SHOULD do**:
1. Install the CLI tools declaratively
2. Install credential helpers (aws-vault, saml2aws) based on project detection
3. Provide a `qsdev doctor` check that verifies auth is configured (e.g., `aws sts get-caller-identity` succeeds)
4. Provide a `qsdev init` template that generates skeleton AWS/GCP/Azure profile configs with TODO comments
5. Set environment variables that don't contain secrets (e.g., `AWS_DEFAULT_REGION`, `KUBECONFIG` path)
6. Display active cloud context in shell prompt (which AWS profile, which K8s context)

**What gdev should NOT do**:
1. Store or manage credentials
2. Auto-configure SSO endpoints
3. Run `aws configure` or `gcloud auth login` on behalf of the user
4. Set `AWS_ACCESS_KEY_ID` or similar credential env vars

---

## 4. Consulting-Specific Considerations

### 4.1 Multi-Client Credential Isolation

The core challenge for consulting engineers: switching between Client A's AWS account, Client B's GCP project, and Client C's Azure subscription -- often within the same day.

**AWS isolation pattern**:
- Named profiles: `[profile clienta-dev]`, `[profile clienta-prod]`, `[profile clientb-staging]`
- aws-vault per-client: `aws-vault exec clienta-dev -- terraform plan`
- Environment variable override: `AWS_PROFILE=clienta-dev` in devenv.nix per project

**GCP isolation pattern**:
- Named configurations: `gcloud config configurations create clienta && gcloud config set project clienta-project-id`
- `gcloud config configurations activate clienta` to switch
- `CLOUDSDK_ACTIVE_CONFIG_NAME=clienta` in devenv.nix per project

**Azure isolation pattern**:
- Subscription selection: `az account set --subscription clienta-sub-id`
- Tenant switching: `az login --tenant clienta-tenant-id`
- `ARM_SUBSCRIPTION_ID` in devenv.nix per project

**K8s isolation pattern**:
- Separate kubeconfig files per client: `~/.kube/clienta-config`, `~/.kube/clientb-config`
- Per-project KUBECONFIG: `KUBECONFIG=~/.kube/clienta-config` in devenv.nix
- Context naming: `clienta-prod-us-east-1`, `clientb-staging-eu-west-1`

**gdev value-add**: Per-project devenv.nix can set `KUBECONFIG`, `AWS_PROFILE`, `CLOUDSDK_ACTIVE_CONFIG_NAME`, and `ARM_SUBSCRIPTION_ID` to the correct client values. This is the single biggest consulting productivity win -- eliminating accidental cross-client operations.

### 4.2 Client VPN and Network Considerations

Many client K8s clusters and cloud accounts require VPN access. gdev should be aware that:
- kubectl commands may fail if VPN is not connected
- `qsdev doctor` could check connectivity to known cluster API endpoints
- Cloud CLI auth flows may require specific DNS resolution (client SSO portals)

### 4.3 Onboarding Velocity

For consulting, onboarding speed to a new client project is paramount. The ideal flow:
1. `git clone client-project`
2. `cd client-project && gdev setup` (devenv activates, all tools installed)
3. `qsdev doctor` reports what auth is still needed
4. Engineer configures auth manually (one-time)
5. All subsequent `cd client-project` activations have correct tools and context

### 4.4 HashiCorp Vault Integration

Some enterprise clients use HashiCorp Vault for credential management:
- Vault generates short-lived dynamic credentials for AWS/GCP/Azure
- Each credential has a TTL and is automatically revoked
- Developer authenticates to Vault once, then Vault issues cloud-specific credentials
- **gdev consideration**: If a project uses Vault, gdev should install the `vault` CLI. Detection: `VAULT_ADDR` env var, `.vault-token`, Vault-related Terraform resources.

---

## 5. Recommended Tiering for gdev

### Tier 1: Must-Ship (install by default when category detected)

| Tool | Package | Detection Trigger |
|------|---------|-------------------|
| AWS CLI v2 | `awscli2` | Any AWS indicator file |
| Google Cloud SDK | `google-cloud-sdk` | Any GCP indicator file |
| Azure CLI | `azure-cli` | Any Azure indicator file |
| kubectl | `kubectl` | Any K8s indicator file |
| kubectx/kubens | `kubectx` | Installed with kubectl |
| k9s | `k9s` | Installed with kubectl |
| stern | `stern` | Installed with kubectl |
| Helm v3 | `kubernetes-helm` | `Chart.yaml`, `helmfile.yaml` |
| kustomize | `kustomize` | `kustomization.yaml` |

### Tier 2: Should-Ship (install when specific config detected)

| Tool | Package | Detection Trigger |
|------|---------|-------------------|
| aws-vault | `aws-vault` | AWS detected + `aws-vault` in scripts/docs |
| saml2aws | `saml2aws` | SAML IdP references in project |
| aws-sso-cli | `aws-sso-cli` | AWS SSO config detected |
| helmfile | `helmfile` | `helmfile.yaml` |
| Skaffold | `skaffold` | `skaffold.yaml` |
| Tilt | `tilt` | `Tiltfile` |
| DevSpace | `devspace` | `devspace.yaml` |
| kubescape | `kubescape` | K8s manifests present |
| kube-linter | `kube-linter` | K8s manifests or Helm charts present |
| dive | `dive` | `Dockerfile` present |
| Telepresence | `telepresence2` | K8s development context detected |

### Tier 3: Nice-to-Have (available but not auto-detected)

| Tool | Package | Use Case |
|------|---------|----------|
| kubetail | `kubetail` | Alternative to stern |
| polaris | `polaris` | K8s best-practice validation |
| kube-bench | `kube-bench` | CIS compliance auditing |
| checkov | `checkov` | Broad IaC scanning |
| istioctl | `istioctl` | Istio service mesh |
| linkerd | `linkerd` | Linkerd service mesh |
| podman | `podman` | Docker alternative |
| buildah | `buildah` | OCI image building |
| skopeo | `skopeo` | Image registry operations |
| crane | `go-containerregistry` | Image manipulation |
| argocd | `argocd` | GitOps CD |

### Tier 4: Project-Specific Cloud Platforms (install on detection)

| Tool | Package | Detection Trigger |
|------|---------|-------------------|
| Wrangler | `wrangler` | `wrangler.toml` |
| doctl | `doctl` | `.do/app.yaml` |
| flyctl | `flyctl` | `fly.toml` |
| Vercel CLI | `vercel` | `vercel.json` |
| Netlify CLI | `netlify-cli` | `netlify.toml` |

---

## 6. Implementation Considerations

### 6.1 Module Architecture

Cloud/K8s tooling should be implemented as **two new qsdev devenv module categories**:

1. **`cloud` module**: AWS, GCP, Azure, platform CLIs + credential helpers
2. **`kubernetes` module**: kubectl, context tools, development tools, security tools, observability

These are distinct from the existing `languages.helm` module because they serve infrastructure/operations concerns rather than language development.

### 6.2 Shell Integration

gdev should enhance the shell prompt to show:
- Active AWS profile (`AWS_PROFILE`)
- Active GCP configuration/project
- Active K8s context and namespace (kubectx/kubens already do this with most prompt tools)
- Visual warning when no auth is configured

### 6.3 Version Pinning Strategy

For kubectl specifically, version should match the target cluster version (within +/-1 minor). gdev could:
- Allow explicit version pin in devenv.nix: `kubernetes.kubectlVersion = "1.28"`
- Default to latest stable if not pinned
- Warn on version skew via `qsdev doctor`

### 6.4 Credential Security Hooks

Extend gdev's existing pre-commit security scanning:
- Detect hardcoded AWS access keys in committed files (gitleaks already covers this)
- Warn if `~/.aws/credentials` contains long-term credentials instead of using aws-vault/SSO
- Check that `.env` files containing cloud credentials are in `.gitignore`

---

## 7. Open Questions

1. **Should gdev manage kubeconfig merging?** Setting `KUBECONFIG` per-project is straightforward, but some engineers prefer a single merged kubeconfig. Which default?
2. **Helm module expansion vs new K8s module?** The existing `languages.helm` module could be expanded, or K8s could be a separate top-level module that includes Helm. Recommend: separate module, Helm as a sub-option.
3. **How to handle tool version conflicts?** If Client A needs kubectl 1.28 and Client B needs 1.30, devenv.sh handles this per-project. But should gdev provide version override UX?
4. **Cloud CLI plugin management**: gcloud has `components install`, az has `extensions add`. Should gdev manage these, or leave to the engineer?
5. **Offline/air-gapped scenarios**: Some government consulting clients restrict internet access. Should gdev support pre-cached cloud CLI installations?
