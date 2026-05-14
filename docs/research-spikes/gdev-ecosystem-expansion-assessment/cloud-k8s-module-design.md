# Cloud & Kubernetes Module Design — Phase 2 Amendment

## Overview

This document defines implementation units for two new gdev ecosystem module categories: **Cloud Provider CLIs** and **Kubernetes Tooling**. These units extend Phase 2 (Ecosystem Modules — Tier 1) from unit 2.9 onward, following the same `EcosystemModule` interface pattern as units 2.1-2.8.

The cloud and kubernetes modules differ from language ecosystem modules in one critical way: the hard problem is not installation but **credential management and multi-client isolation**. Every tool listed here is available in Nixpkgs. The value gdev adds is per-project environment variable isolation that prevents cross-client credential leakage — the single biggest consulting productivity and safety win.

**Design constraint**: gdev installs tools and provides scaffolding but does NOT manage credentials. Credentials are per-engineer, per-client, and governed by client security policies. gdev sets non-secret environment variables (`AWS_PROFILE`, `KUBECONFIG`, `CLOUDSDK_ACTIVE_CONFIG_NAME`) and provides `gdev doctor` checks to verify auth status.

## Dependencies

- Phase 1 complete (module interface, detection engine, template engine, generation pipeline)
- Phase 2 units 2.7 (Docker) and 2.8 (Terraform) provide patterns for infrastructure-oriented modules
- Terraform module (2.8) already handles `*.tf` file parsing — cloud provider detection reuses that engine to inspect `required_providers` blocks

---

## Cloud Provider Modules

### Unit 2.9: AWS Module (Tier 1)

**Description:** Detect AWS usage, install `awscli2` and optional credential helpers (aws-vault, saml2aws), set per-project `AWS_PROFILE` isolation, and provide `gdev doctor` auth verification.

**Context:** AWS is the most common cloud provider in enterprise consulting. The CLI itself is straightforward (`awscli2` in Nixpkgs), but consulting engineers routinely manage credentials for multiple client AWS accounts. The critical gdev value-add is per-project `AWS_PROFILE` in devenv.nix, which prevents accidental cross-client operations (e.g., running `terraform apply` against the wrong account). Credential helpers like aws-vault store credentials in OS keychains rather than plaintext `~/.aws/credentials` — on NixOS/Linux, the `pass` or `file` backend is used since there is no macOS Keychain. Detection must handle both direct AWS file indicators and Terraform provider blocks.

**Desired Outcome:** When AWS usage is detected in a project, `gdev init` generates a devenv.nix fragment with `awscli2`, optional credential helper packages, per-project `AWS_PROFILE` environment variable, and a `gdev doctor` check that runs `aws sts get-caller-identity`.

**Steps:**
1. Implement `Detect`: check for indicator files in priority order:
   - `*.tf` containing `provider "aws"` or `source = "hashicorp/aws"` in `required_providers` (reuse Terraform module's provider parser)
   - `cdk.json` (AWS CDK)
   - `serverless.yml` or `serverless.yaml` (Serverless Framework)
   - `samconfig.toml` or `template.yaml` with `AWS::` resource types (AWS SAM)
   - `buildspec.yml` (AWS CodeBuild)
   - `appspec.yml` (AWS CodeDeploy)
   - `.aws-sam/` directory
2. Implement `DevenvNixFragment`:
   - Add `pkgs.awscli2` to packages
   - Conditionally add `pkgs.aws-vault` (default credential helper for consulting profile) and/or `pkgs.saml2aws` (when SAML IdP references detected in project docs)
   - Conditionally add `pkgs.aws-sso-cli` (when AWS SSO config patterns detected)
   - Set `env.AWS_PROFILE` with a TODO placeholder: `env.AWS_PROFILE = "TODO-set-client-profile-name";`
   - Set `env.AWS_DEFAULT_REGION` with project-detected region or placeholder
   - Add shell hook comment: `# Run: aws sso login --profile $AWS_PROFILE`
3. Implement `SecurityConfigs`:
   - No config file generation (credentials are per-engineer)
   - Generate `.env.example` snippet documenting expected AWS env vars with TODO comments
   - Include `AWS_VAULT_BACKEND=pass` note for NixOS/Linux in generated comments
4. Implement `DoctorChecks`:
   - `aws sts get-caller-identity` — verifies credentials are configured and valid
   - `aws-vault list` (when aws-vault installed) — verifies vault has entries
   - Check `AWS_PROFILE` is set and non-empty
5. Implement `PreCommitHooks`: none specific to AWS (gitleaks already covers AWS key detection via Phase 12 integration)
6. Implement `DenyRules`: none specific to AWS CLI (it is a read/write operational tool, not a package manager)

**Acceptance Criteria:**
- [ ] Detects AWS from Terraform provider blocks, CDK, Serverless, SAM, CodeBuild, and CodeDeploy files
- [ ] devenv.nix fragment includes `awscli2` and per-project `AWS_PROFILE` placeholder
- [ ] aws-vault included conditionally with `pass` backend note for Linux
- [ ] saml2aws included only when SAML indicators detected
- [ ] `gdev doctor` check runs `aws sts get-caller-identity` and reports pass/fail
- [ ] No credential values are ever written to generated files

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md § 1.1 AWS CLI v2` — detection heuristics, auth patterns, multi-account strategy
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md § 1.2 AWS Credential Helpers` — aws-vault, saml2aws, aws-sso-cli capabilities and Linux backend notes
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md § 3.4 Should gdev Configure Credentials or Just Install Tools?` — install+scaffold, never manage credentials
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md § 4.1 Multi-Client Credential Isolation` — per-project AWS_PROFILE pattern

**Status:** Not Started

---

### Unit 2.10: GCP Module (Tier 1)

**Description:** Detect GCP usage, install `google-cloud-sdk` with optional `gke-gcloud-auth-plugin`, set per-project `CLOUDSDK_ACTIVE_CONFIG_NAME` isolation, and provide `gdev doctor` auth verification.

**Context:** GCP is the second most common cloud provider in enterprise consulting. The `google-cloud-sdk` Nixpkgs package includes `gcloud`, `gsutil`, and `bq`. Additional components like `gke-gcloud-auth-plugin` (required for GKE cluster auth) need to be added via `google-cloud-sdk.withExtraComponents` in Nix or installed separately. GCP uses named configurations (`gcloud config configurations`) for multi-project isolation — the `CLOUDSDK_ACTIVE_CONFIG_NAME` environment variable selects the active configuration, analogous to `AWS_PROFILE`. No third-party credential helpers are needed; `gcloud auth login` and `gcloud auth application-default login` handle all auth patterns natively.

**Desired Outcome:** When GCP usage is detected in a project, `gdev init` generates a devenv.nix fragment with `google-cloud-sdk` (plus GKE auth plugin when K8s co-detected), per-project `CLOUDSDK_ACTIVE_CONFIG_NAME`, and a `gdev doctor` check that verifies active auth.

**Steps:**
1. Implement `Detect`: check for indicator files:
   - `*.tf` containing `provider "google"` or `source = "hashicorp/google"` in `required_providers`
   - `app.yaml` with App Engine runtime indicator
   - `cloudbuild.yaml` (Cloud Build)
   - `.gcloudignore`
   - `firebase.json` (Firebase, uses GCP underneath)
2. Implement `DevenvNixFragment`:
   - Add `pkgs.google-cloud-sdk` to packages (base)
   - When K8s + GCP co-detected: use `pkgs.google-cloud-sdk.withExtraComponents [ pkgs.google-cloud-sdk.components.gke-gcloud-auth-plugin ]` instead of bare package
   - Set `env.CLOUDSDK_ACTIVE_CONFIG_NAME` with TODO placeholder
   - Set `env.CLOUDSDK_CORE_PROJECT` with TODO placeholder
   - Set `env.GOOGLE_CLOUD_PROJECT` with TODO placeholder (used by client libraries via ADC)
   - Add shell hook comment: `# Run: gcloud auth login && gcloud auth application-default login`
3. Implement `SecurityConfigs`:
   - Generate `.env.example` snippet documenting expected GCP env vars
   - Note: `GOOGLE_APPLICATION_CREDENTIALS` should NOT be set in devenv.nix (it points to service account key files, which is the legacy pattern — ADC via `gcloud auth application-default login` is preferred)
4. Implement `DoctorChecks`:
   - `gcloud auth print-access-token` — verifies user auth is active (non-zero exit = not authenticated)
   - `gcloud config get-value project` — verifies a project is set
   - Check `CLOUDSDK_ACTIVE_CONFIG_NAME` is set and non-empty
5. Implement `PreCommitHooks`: none specific to GCP
6. Implement `DenyRules`: none specific to GCP CLI

**Acceptance Criteria:**
- [ ] Detects GCP from Terraform provider blocks, App Engine, Cloud Build, and Firebase files
- [ ] devenv.nix fragment includes `google-cloud-sdk` with GKE auth plugin when K8s co-detected
- [ ] Per-project `CLOUDSDK_ACTIVE_CONFIG_NAME` set in devenv.nix env
- [ ] `gdev doctor` check verifies auth via `gcloud auth print-access-token`
- [ ] ADC pattern documented; `GOOGLE_APPLICATION_CREDENTIALS` explicitly NOT set

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md § 1.3 GCP CLI (Google Cloud SDK)` — auth patterns, ADC chain, named configurations, Nixpkgs notes on withExtraComponents
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md § 4.1 Multi-Client Credential Isolation` — GCP isolation pattern with CLOUDSDK_ACTIVE_CONFIG_NAME

**Status:** Not Started

---

### Unit 2.11: Azure Module (Tier 1)

**Description:** Detect Azure usage, install `azure-cli` (and optionally `azd`), set per-project `ARM_SUBSCRIPTION_ID` isolation, and provide `gdev doctor` auth verification.

**Context:** Azure is the third Tier 1 cloud provider, especially common in enterprise and government consulting. Two CLIs exist: `az` (general-purpose, `azure-cli` in Nixpkgs) and `azd` (Azure Developer CLI, higher-level workflow tool — not separately packaged in Nixpkgs but ships with azure-cli or installed separately). Azure uses subscription-based isolation; `ARM_SUBSCRIPTION_ID` and `ARM_TENANT_ID` are the Terraform-standard env vars for per-project scoping. Unlike AWS and GCP, Azure has no named-configuration system in the CLI — subscription selection is via `az account set`. For K8s (AKS), the `kubelogin` package provides Azure AD auth.

**Desired Outcome:** When Azure usage is detected in a project, `gdev init` generates a devenv.nix fragment with `azure-cli`, per-project ARM env vars, and a `gdev doctor` check that verifies login status.

**Steps:**
1. Implement `Detect`: check for indicator files:
   - `*.tf` containing `provider "azurerm"` or `source = "hashicorp/azurerm"` in `required_providers`
   - `azure-pipelines.yml` (Azure DevOps)
   - `*.bicep` files (Azure Bicep IaC)
   - `azure.yaml` (Azure Developer CLI project)
   - `.azure/` directory
2. Implement `DevenvNixFragment`:
   - Add `pkgs.azure-cli` to packages
   - Conditionally add `pkgs.kubelogin` when K8s + Azure co-detected (AKS auth)
   - Set `env.ARM_SUBSCRIPTION_ID` with TODO placeholder
   - Set `env.ARM_TENANT_ID` with TODO placeholder
   - Add shell hook comment: `# Run: az login --tenant $ARM_TENANT_ID`
   - When `azure.yaml` detected, note `azd` availability in comments
3. Implement `SecurityConfigs`:
   - Generate `.env.example` snippet documenting expected Azure env vars
   - Note: `ARM_CLIENT_SECRET` must NEVER appear in devenv.nix or committed files
4. Implement `DoctorChecks`:
   - `az account show` — verifies login status and shows active subscription
   - `az account show --query id -o tsv` compared against `ARM_SUBSCRIPTION_ID` — verifies correct subscription is active
   - Check `ARM_SUBSCRIPTION_ID` is set and non-empty
5. Implement `PreCommitHooks`: none specific to Azure
6. Implement `DenyRules`: none specific to Azure CLI

**Acceptance Criteria:**
- [ ] Detects Azure from Terraform provider blocks, Azure Pipelines, Bicep, and azd files
- [ ] devenv.nix fragment includes `azure-cli` and per-project `ARM_SUBSCRIPTION_ID`
- [ ] kubelogin included when K8s + Azure co-detected (AKS)
- [ ] `gdev doctor` check verifies login via `az account show` and validates active subscription
- [ ] `ARM_CLIENT_SECRET` explicitly excluded from all generated files

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md § 1.4 Azure CLI` — two CLIs, auth patterns, multi-subscription management
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md § 4.1 Multi-Client Credential Isolation` — Azure isolation pattern with ARM_SUBSCRIPTION_ID

**Status:** Not Started

---

### Unit 2.12: Cloud Platform CLIs Module (Tier 3)

**Description:** Detect project-specific cloud platforms (Cloudflare, DigitalOcean, Vercel, Fly.io, Netlify) and install their CLIs when indicator files are present.

**Context:** These platforms are project-specific rather than universally needed. Unlike Tier 1 cloud providers, they are typically used by a single project and do not require multi-client credential isolation patterns. Each has a simple detection heuristic (one or two config files) and a single CLI package. They are Tier 3 because they appear on a minority of consulting engagements — mostly startup and indie clients. All packages are available in Nixpkgs. This unit is a single module that handles multiple platforms via a sub-detection pattern, rather than five separate modules.

**Desired Outcome:** When any Tier 3 platform indicator file is detected, `gdev init` includes the corresponding CLI in the devenv.nix fragment with appropriate auth reminder comments.

**Steps:**
1. Implement `Detect` as a multi-platform detector with sub-heuristics:
   - **Cloudflare**: `wrangler.toml` or `wrangler.jsonc`
   - **DigitalOcean**: `.do/app.yaml` or `DIGITALOCEAN_TOKEN` referenced in project files
   - **Fly.io**: `fly.toml`
   - **Vercel**: `vercel.json` or `.vercel/` directory
   - **Netlify**: `netlify.toml` or `.netlify/` directory
2. Implement `DevenvNixFragment` — per-detected-platform additions:
   - Cloudflare: `pkgs.wrangler` + comment `# Run: wrangler login`
   - DigitalOcean: `pkgs.doctl` + comment `# Run: doctl auth init`
   - Fly.io: `pkgs.flyctl` + comment `# Run: fly auth login`
   - Vercel: `pkgs.nodePackages.vercel` + comment `# Run: vercel login`
   - Netlify: `pkgs.netlify-cli` + comment `# Run: netlify login`
3. Implement `DoctorChecks` — per-detected-platform:
   - Cloudflare: `wrangler whoami`
   - DigitalOcean: `doctl account get`
   - Fly.io: `fly auth whoami`
   - Vercel: `vercel whoami`
   - Netlify: `netlify status`
4. No security configs, pre-commit hooks, or deny rules for these platforms

**Acceptance Criteria:**
- [ ] Detects each platform from its indicator files independently
- [ ] Only detected platforms are included in devenv.nix (no blanket installation)
- [ ] Each platform has a `gdev doctor` auth check
- [ ] Multiple platforms can be detected simultaneously in the same project
- [ ] Auth reminder comments are platform-specific

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md § 1.5 Other Cloud CLIs` — package names, versions, detection heuristics, consulting relevance
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md § 5 Recommended Tiering` — Tier 4 classification (project-specific cloud platforms)

**Status:** Not Started

---

### Unit 2.13: Cloud Module Shared Infrastructure

**Description:** Implement shared detection, environment variable isolation, and doctor check infrastructure that all cloud provider modules (2.9-2.12) depend on.

**Context:** All cloud provider modules share common patterns: Terraform provider block parsing for detection, per-project environment variable isolation in devenv.nix, and `gdev doctor` auth verification. Rather than duplicating this logic across four modules, this unit extracts shared infrastructure. The Terraform module (2.8) already parses `*.tf` files — the cloud modules need to hook into that parser to extract `required_providers` blocks. Environment variable isolation is the single biggest consulting value-add: setting `AWS_PROFILE`, `KUBECONFIG`, `CLOUDSDK_ACTIVE_CONFIG_NAME` per-project in devenv.nix prevents accidental cross-client operations. The doctor check infrastructure provides a standardized pattern for "run command, check exit code, report pass/fail with actionable fix message."

**Desired Outcome:** A shared cloud module infrastructure package exists that cloud provider modules compose over, providing Terraform provider detection, env var template helpers, and doctor check registration.

**Steps:**
1. Implement `TerraformProviderDetector`:
   - Parse `*.tf` files for `required_providers` blocks
   - Return set of detected providers: `aws`, `google`, `azurerm`, `cloudflare`, `digitalocean`, etc.
   - Reuse or extend the file parser from the Terraform module (2.8) — do not duplicate
   - Handle both `provider "aws" {}` blocks and `required_providers` source strings
2. Implement `CloudEnvVarTemplate` helper:
   - Generates devenv.nix `env = { ... }` blocks with TODO placeholders
   - Includes inline comments explaining isolation purpose
   - Validates that no secret-bearing env var names (`*_SECRET*`, `*_KEY*`, `*_TOKEN*`, `*_PASSWORD*`) are included
3. Implement `DoctorCheckRegistry` for cloud checks:
   - Standardized check format: command, expected exit code, pass message, fail message with fix instructions
   - Timeout handling (cloud CLI commands can hang if network is unavailable or VPN disconnected)
   - Graceful degradation: if CLI binary not found, report "not installed" rather than error
4. Implement shell prompt integration helpers:
   - Generate starship/oh-my-posh config snippets showing active cloud context
   - Display `AWS_PROFILE`, GCP project, K8s context in prompt

**Acceptance Criteria:**
- [ ] Terraform provider parser extracts cloud providers from `required_providers` blocks
- [ ] Provider parser handles both HCL syntax forms (`provider "aws"` and `source = "hashicorp/aws"`)
- [ ] Env var template helper refuses to emit secret-bearing variable names
- [ ] Doctor checks have configurable timeout (default 5s) for network-dependent commands
- [ ] Doctor checks report "not installed" gracefully when CLI binary is absent
- [ ] Shell prompt helpers generate valid starship config snippets

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md § 3.3 Detection Heuristics` — Terraform provider parsing strategy
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md § 3.4 Should gdev Configure Credentials or Just Install Tools?` — the 6 things gdev SHOULD do and 4 things it should NOT do
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md § 4.1 Multi-Client Credential Isolation` — all four provider isolation patterns
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md § 6.2 Shell Integration` — prompt context display

**Status:** Not Started

---

## Kubernetes Modules

### Unit 2.14: Kubernetes Core Module (Tier 1)

**Description:** Detect Kubernetes usage, install core tooling (`kubectl`, `kubectx`/`kubens`, `k9s`, `stern`, `kustomize`), enforce per-project `KUBECONFIG` isolation, and provide `gdev doctor` cluster connectivity checks.

**Context:** kubectl is nearly universal for K8s-using clients. The critical challenge is kubeconfig management: the default `~/.kube/config` file merges all cluster credentials, meaning consulting engineers working with multiple clients risk running commands against the wrong cluster. The gdev solution is per-project `KUBECONFIG` in devenv.nix, pointing to a client-specific kubeconfig file (e.g., `~/.kube/clienta-config`). kubectl has a version skew policy — client must be within +/-1 minor version of the cluster API server — so devenv.nix pins a specific version. k9s and stern are included at Tier 1 because they provide very high daily-use value: k9s replaces kubectl for interactive cluster exploration, and stern provides multi-pod log tailing essential for debugging.

**Desired Outcome:** When K8s indicator files are detected, `gdev init` generates a devenv.nix fragment with kubectl (version-pinnable), kubectx, k9s, stern, kustomize (when kustomization.yaml present), per-project `KUBECONFIG`, and `gdev doctor` checks for cluster connectivity.

**Steps:**
1. Implement `Detect`: check for indicator files (any match triggers the module):
   - `k8s/` or `kubernetes/` directories
   - `kustomization.yaml` or `kustomization.yml`
   - `Chart.yaml` (Helm — implies K8s)
   - `helmfile.yaml`
   - `skaffold.yaml`
   - `Tiltfile`
   - `devspace.yaml`
   - `*.yaml` files containing `apiVersion:` with K8s API group patterns (e.g., `apps/v1`, `v1`, `networking.k8s.io/v1`)
2. Implement `DevenvNixFragment`:
   - Add `pkgs.kubectl` to packages (with version pin comment: `# Pin to match cluster version; kubectl supports +/-1 minor version skew`)
   - Add `pkgs.kubectx` (includes kubens)
   - Add `pkgs.k9s`
   - Add `pkgs.stern`
   - Conditionally add `pkgs.kustomize` when `kustomization.yaml` detected
   - Set `env.KUBECONFIG` with per-project path: `env.KUBECONFIG = "$HOME/.kube/TODO-client-name-config";`
   - Add shell hook displaying current context: `echo "K8s context: $(kubectl config current-context 2>/dev/null || echo 'none')"`
3. Implement `SecurityConfigs`:
   - Generate `.env.example` snippet documenting `KUBECONFIG` isolation pattern
   - Include inline comment: `# NEVER use ~/.kube/config directly — use per-client kubeconfig files`
   - Document naming convention: `<client>-<env>-<region>` (e.g., `acme-prod-us-east-1`)
4. Implement `DoctorChecks`:
   - `kubectl cluster-info` — verifies cluster connectivity (with 5s timeout for VPN-dependent clusters)
   - `kubectl version --client` — verifies kubectl is installed and reports version
   - Check `KUBECONFIG` is set, non-empty, and points to an existing file
   - Check `KUBECONFIG` does NOT equal `~/.kube/config` (warn about shared kubeconfig)
5. Implement cloud-provider K8s auth integration:
   - When AWS co-detected: note that `aws eks update-kubeconfig` generates client-specific kubeconfig
   - When GCP co-detected: add `gke-gcloud-auth-plugin` (via Unit 2.10)
   - When Azure co-detected: add `kubelogin` (via Unit 2.11)
6. Implement `PreCommitHooks`: none specific to core K8s (YAML linting handled by language modules; K8s-specific linting in Unit 2.16)
7. Implement `DenyRules`: none for kubectl (operational tool, not a package manager)

**Acceptance Criteria:**
- [ ] Detects K8s from directory structure, kustomization, Helm, Skaffold, Tilt, and DevSpace files
- [ ] devenv.nix fragment includes kubectl, kubectx, k9s, and stern
- [ ] kustomize included conditionally only when kustomization.yaml detected
- [ ] Per-project `KUBECONFIG` set with TODO placeholder path (not `~/.kube/config`)
- [ ] `gdev doctor` warns when `KUBECONFIG` points to shared `~/.kube/config`
- [ ] `gdev doctor` cluster connectivity check has 5s timeout
- [ ] Version pin comment present for kubectl
- [ ] Cloud-provider auth plugins noted when co-detected with AWS/GCP/Azure

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md § 2.1 kubectl` — version skew policy, kubeconfig management, auth plugins, best practices for consulting
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md § 2.2 Context & Namespace Switching` — kubectx/kubens consulting value
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md § 2.4 Kustomize` — standalone vs kubectl-integrated
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md § 2.6 K8s Observability Tools` — k9s and stern daily-use value
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md § 4.1 Multi-Client Credential Isolation` — K8s isolation pattern with per-client kubeconfig

**Status:** Not Started

---

### Unit 2.15: Kubernetes Development Module (Tier 2)

**Description:** Detect K8s development tools (Skaffold, Tilt, DevSpace, Telepresence) from their config files and install the matching tool — detect-and-offer, not default-install.

**Context:** K8s development tools automate the build-push-deploy inner loop for local K8s development. They are project-specific and opinionated — a project uses Skaffold OR Tilt OR DevSpace, rarely more than one. Detection is straightforward (each has a unique config file), but these tools should never be default-installed since they impose workflow opinions. Telepresence is a different category entirely: it intercepts traffic from a remote cluster to a local machine for debugging. Garden is excluded because it is not in Nixpkgs. This module is Tier 2: install when specific config file detected.

**Desired Outcome:** When a K8s development tool config file is detected, `gdev init` includes exactly that tool in the devenv.nix fragment, with no cross-installation of competing tools.

**Steps:**
1. Implement `Detect` with mutually exclusive sub-heuristics:
   - **Skaffold**: `skaffold.yaml`
   - **Tilt**: `Tiltfile`
   - **DevSpace**: `devspace.yaml`
   - **Telepresence**: no config file detection — include only when explicitly enabled in wizard or `.gdev.yaml`
   - If multiple config files exist (unlikely but possible), include all detected tools without conflict
2. Implement `DevenvNixFragment` — per-detected-tool:
   - Skaffold: `pkgs.skaffold`
   - Tilt: `pkgs.tilt`
   - DevSpace: `pkgs.devspace`
   - Telepresence: `pkgs.telepresence2`
3. Implement `DoctorChecks` — per-detected-tool:
   - Skaffold: `skaffold version` (binary present and functional)
   - Tilt: `tilt version`
   - DevSpace: `devspace --version`
   - Telepresence: `telepresence version`
4. No security configs, pre-commit hooks, or deny rules for development tools
5. Implement Helm integration awareness:
   - When Skaffold detected and `Chart.yaml` present: note in comments that Skaffold can deploy via Helm
   - When helmfile detected alongside any dev tool: include `pkgs.helmfile` in fragment

**Acceptance Criteria:**
- [ ] Detects Skaffold, Tilt, and DevSpace from their respective config files
- [ ] Telepresence is opt-in only (not auto-detected)
- [ ] Only detected tools are included — no cross-installation
- [ ] Each tool has a version doctor check
- [ ] Multiple dev tools can coexist if multiple config files are present

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md § 2.5 K8s Development Tools` — tool comparison (Skaffold vs Tilt vs DevSpace vs Telepresence), detection files, Nixpkgs availability
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md § 5 Recommended Tiering` — Tier 2 classification for Skaffold, Tilt, DevSpace, Telepresence

**Status:** Not Started

---

### Unit 2.16: Kubernetes Security Module (Tier 3)

**Description:** Provide optional K8s security scanning tools (kubescape, kube-linter, kube-bench, polaris) that activate when explicitly enabled or when a cloud security profile is active.

**Context:** K8s security tools are valuable but niche — most projects do not run them locally. kubescape is the most comprehensive single tool (260+ controls, NSA-CISA/MITRE ATT&CK/CIS frameworks, CNCF Incubating). kube-linter is fast static analysis designed for pre-commit/CI gates on K8s YAML and Helm charts. kube-bench focuses specifically on CIS Kubernetes Benchmark compliance (cluster component config). polaris validates resource configs against best practices. These tools overlap with each other and with the Terraform/IaC security scanning from Unit 2.8 — the recommended default is kubescape (breadth) + kube-linter (speed for CI). This module is Tier 3: available when explicitly enabled via wizard, `.gdev.yaml`, or a security-focused profile.

**Desired Outcome:** When K8s security scanning is enabled (not auto-detected), `gdev init` includes kubescape and kube-linter in devenv.nix with pre-commit hooks for manifest validation and CI scanning commands.

**Steps:**
1. Implement `Detect`: this module does NOT auto-detect — it activates via:
   - Explicit wizard selection during `gdev init`
   - `.gdev.yaml` setting: `kubernetes.security = true`
   - Profile activation: consulting security profile or compliance profile
   - Exception: if `kubescape` or `kube-linter` config files detected (`.kubescape/`, `.kube-linter.yaml`), auto-enable
2. Implement `DevenvNixFragment`:
   - Default set: `pkgs.kubescape`, `pkgs.kube-linter`
   - Extended set (when CIS compliance needed): add `pkgs.kube-bench`, `pkgs.polaris`
   - Note in comments: `# Requires cluster access for runtime scanning (kubescape, kube-bench)`
3. Implement `SecurityConfigs`:
   - Generate `.kube-linter.yaml` with recommended checks enabled (prevent privileged containers, require resource limits, require readiness probes)
   - Generate `kubescape` config snippet documenting framework selection (NSA-CISA recommended as default)
4. Implement `PreCommitHooks`:
   - `kube-linter lint` on `*.yaml` files in `k8s/`, `kubernetes/`, `charts/` directories
   - File pattern filtering to avoid linting non-K8s YAML
5. Implement `CICommands`:
   - `kube-linter lint k8s/` (fast, static analysis)
   - `kubescape scan framework nsa --exclude-namespaces kube-system` (comprehensive, requires cluster or manifest files)
   - `kube-bench run` (CIS compliance, requires cluster access — note this in comments)
6. Implement `DoctorChecks`:
   - `kubescape version` and `kube-linter version` — binary availability
   - No cluster connectivity check (that is in Unit 2.14)

**Acceptance Criteria:**
- [ ] Module does NOT auto-detect — requires explicit enablement
- [ ] Exception: auto-enables when kubescape/kube-linter config files found
- [ ] Default set is kubescape + kube-linter (not all four tools)
- [ ] `.kube-linter.yaml` generated with recommended security checks
- [ ] Pre-commit hooks filter to K8s YAML paths only (not all YAML)
- [ ] CI commands distinguish static analysis (no cluster) from runtime scanning (cluster needed)
- [ ] kube-bench and polaris are extended-set only (CIS compliance profile)

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md § 2.7 K8s Security Tools` — tool comparison, CNCF status, selection guide
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md § 5 Recommended Tiering` — Tier 2/3 classification for security tools

**Status:** Not Started

---

### Unit 2.17: Kubernetes Module Shared Infrastructure

**Description:** Implement shared detection, KUBECONFIG isolation, and Helm integration infrastructure that all Kubernetes modules (2.14-2.16) depend on.

**Context:** The three Kubernetes modules share detection patterns (all triggered by K8s indicator files), KUBECONFIG management (the core isolation mechanism), and Helm integration (Helm is both a language ecosystem module from Unit 2.8's domain and a K8s deployment tool). This unit extracts shared infrastructure to avoid duplication and ensure consistent behavior. The Helm module already exists in devenv.sh as `languages.helm` — gdev should extend rather than replace it. The key design decision: Helm is a sub-option of the K8s module (not a standalone infrastructure module), consistent with the research recommendation.

**Desired Outcome:** A shared Kubernetes module infrastructure package exists that K8s modules compose over, providing unified K8s file detection, KUBECONFIG template generation, Helm expansion hooks, and cloud-provider auth plugin coordination.

**Steps:**
1. Implement `K8sFileDetector`:
   - Unified detection across all K8s indicator files (directories, config files, YAML with K8s apiVersion patterns)
   - Return detection results with granularity: `core` (any K8s file), `helm` (Chart.yaml, helmfile.yaml), `kustomize` (kustomization.yaml), `dev-tool` (skaffold.yaml, Tiltfile, devspace.yaml), `security-config` (.kubescape/, .kube-linter.yaml)
   - Share results across all three K8s modules to avoid redundant file scanning
2. Implement `KubeconfigTemplate` helper:
   - Generate `env.KUBECONFIG` with per-project path pattern
   - Validate KUBECONFIG path is not `~/.kube/config` (the shared default)
   - Support colon-separated KUBECONFIG paths for projects needing multiple cluster configs
   - Include naming convention documentation in generated comments
3. Implement Helm expansion:
   - When K8s core detected and Helm files present, extend the existing `languages.helm` devenv.sh module
   - Add `pkgs.helmfile` when `helmfile.yaml` detected
   - Configure Helm plugins via `wrapHelm` pattern: `helm-secrets`, `helm-diff` as recommended defaults
   - Coordinate with Unit 2.8 (Terraform) when both Helm and Terraform detected (common IaC pattern)
4. Implement cloud-provider auth coordinator:
   - Query Units 2.9-2.11 detection results
   - When AWS + K8s: note `aws-iam-authenticator` (bundled with awscli2) and `aws eks update-kubeconfig` pattern
   - When GCP + K8s: ensure `gke-gcloud-auth-plugin` is in google-cloud-sdk components
   - When Azure + K8s: ensure `kubelogin` is in packages (AKS AAD auth)
5. Implement version matching helper:
   - Pattern for kubectl version pinning in devenv.nix: `pkgs.kubectl_1_28` (Nixpkgs provides version-specific packages)
   - Document the version skew policy (+/-1 minor) in generated comments
   - `gdev doctor` check: compare `kubectl version --client` against `kubectl version --short` server version when cluster is reachable

**Acceptance Criteria:**
- [ ] Unified K8s file detector returns granular detection categories
- [ ] Detection results are shared across K8s modules (no redundant scanning)
- [ ] KUBECONFIG template validates against shared `~/.kube/config`
- [ ] Helm expansion uses `wrapHelm` pattern for plugin management
- [ ] Cloud-provider auth coordinator queries cloud module detection results
- [ ] kubectl version matching helper documents skew policy

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md § 2.1 kubectl` — version skew, kubeconfig management, auth plugins
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md § 2.3 Helm v3` — wrapHelm pattern, plugin management, helmfile
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md § 3.1 How devenv.sh Users Currently Handle Cloud CLIs` — Pattern D (Helm with plugins via wrapHelm)
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md § 3.2 devenv.nix Patterns for gdev` — proposed module structure
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md § 7 Open Questions` — Helm module expansion vs new K8s module, version conflict handling

**Status:** Not Started

---

## Phase 2 Completion Criteria Amendment

The following criteria extend the existing Phase 2 completion criteria to cover cloud and K8s modules:

- [ ] All 3 Tier 1 cloud modules (AWS, GCP, Azure) detect their respective indicator files
- [ ] All 3 Tier 1 cloud modules generate devenv.nix fragments with correct packages and env var isolation
- [ ] Terraform provider block parsing shared between Terraform module (2.8) and cloud modules (2.9-2.11)
- [ ] K8s core module detects all K8s indicator file patterns
- [ ] Per-project `KUBECONFIG` isolation enforced — never defaults to `~/.kube/config`
- [ ] `gdev doctor` checks cover auth status for all 3 Tier 1 cloud providers + K8s cluster connectivity
- [ ] Doctor checks have 5s timeout and graceful degradation when CLI binaries absent
- [ ] K8s development tools (Skaffold, Tilt, DevSpace) installed only on config file detection
- [ ] K8s security tools (kubescape, kube-linter) installed only on explicit enablement
- [ ] Helm expansion uses `wrapHelm` for plugin management
- [ ] Cloud-provider K8s auth plugins (gke-gcloud-auth-plugin, kubelogin) coordinate across modules
- [ ] No credential values or secret-bearing environment variables appear in any generated file
- [ ] Shell prompt integration helpers generate valid starship config for cloud/K8s context display

## Unit Summary

| Unit | Title | Tier | Category | Key Packages |
|------|-------|------|----------|-------------|
| 2.9 | AWS Module | 1 | Cloud | `awscli2`, `aws-vault`, `saml2aws` |
| 2.10 | GCP Module | 1 | Cloud | `google-cloud-sdk`, `gke-gcloud-auth-plugin` |
| 2.11 | Azure Module | 1 | Cloud | `azure-cli`, `kubelogin` |
| 2.12 | Cloud Platform CLIs Module | 3 | Cloud | `wrangler`, `doctl`, `flyctl`, `vercel`, `netlify-cli` |
| 2.13 | Cloud Module Shared Infrastructure | -- | Cloud | (shared library, no packages) |
| 2.14 | Kubernetes Core Module | 1 | Kubernetes | `kubectl`, `kubectx`, `k9s`, `stern`, `kustomize` |
| 2.15 | Kubernetes Development Module | 2 | Kubernetes | `skaffold`, `tilt`, `devspace`, `telepresence2` |
| 2.16 | Kubernetes Security Module | 3 | Kubernetes | `kubescape`, `kube-linter`, `kube-bench`, `polaris` |
| 2.17 | Kubernetes Module Shared Infrastructure | -- | Kubernetes | (shared library, no packages) |
