# Phase 23: Cloud CLI & Credential Isolation Modules

## Goal

Add cloud platform CLI ecosystem modules using the existing `EcosystemModule` interface. Installation is entirely solved by Nixpkgs; the core value gdev adds is per-project environment variable isolation — preventing cross-client credential leakage that is the single largest consulting safety risk. This phase delivers AWS, GCP, Azure, and Tier 3 platform CLIs, plus shared infrastructure for Terraform provider detection, credential isolation, and doctor checks.

## Dependencies

Phase 1 complete (shared types, EcosystemModule interface, detection engine, template engine, generation pipeline). Phase 2 units 2.7 (Docker) and 2.8 (Terraform) complete — the Terraform module's `*.tf` parser is extended here for `required_providers` block inspection.

## Phase Outputs

- `EcosystemModule` implementations for AWS, GCP, and Azure (Tier 1)
- `EcosystemModule` implementation for Tier 3 cloud platform CLIs (Cloudflare, DigitalOcean, Fly.io, Vercel, Netlify)
- Shared `CloudEnvVarTemplate` helper that refuses to emit secret-bearing variable names
- Shared `TerraformProviderDetector` extending the Phase 2 Terraform parser
- Shared `DoctorCheckRegistry` with 5-second timeouts and graceful degradation
- `.envrc` additions that warn when ambient cloud credentials detected from outside devenv scope
- `gdev doctor` cloud checks: auth status, profile/config match, region/project match

---

### Unit 23.1: AWS CLI Module

**Description:** Implement the `EcosystemModule` for AWS. Detect AWS usage from Terraform provider blocks, CDK, Serverless, SAM, and CodeBuild indicator files. Generate a devenv.nix fragment with `awscli2`, optional credential helpers, and per-project `AWS_PROFILE` isolation.

**Context:** AWS is the most common cloud provider in enterprise consulting. The CLI (`awscli2` in Nixpkgs) is trivial to install; the real value is per-project `AWS_PROFILE` in devenv.nix, which prevents accidental cross-client operations such as `terraform apply` running against the wrong account. Credential helpers (aws-vault, saml2aws) store credentials in OS keychains rather than plaintext `~/.aws/credentials`. On NixOS/Linux, aws-vault uses the `pass` or `file` backend since there is no macOS Keychain. Static credentials (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`) must never appear in generated devenv.nix — the wizard warns about this explicitly. Detection reuses the Terraform provider parser introduced in Unit 23.6 rather than duplicating `.tf` file scanning.

**Desired Outcome:** When AWS usage is detected in a project, `gdev init` generates a devenv.nix fragment that includes `awscli2`, optional credential helpers, a per-project `AWS_PROFILE` env var with a TODO placeholder, and a `gdev doctor` check that runs `aws sts get-caller-identity` with a 5-second timeout. No credential values appear in any generated file.

**Steps:**
1. Implement `Detect(projectRoot string) (DetectionResult, error)` for the AWS module:
   - Call `TerraformProviderDetector` (Unit 23.6) and check if `"aws"` or `"hashicorp/aws"` is in the returned provider set
   - Check for `cdk.json` (AWS CDK)
   - Check for `serverless.yml` or `serverless.yaml` (Serverless Framework)
   - Check for `samconfig.toml` or a `template.yaml` containing `AWS::` resource type strings
   - Check for `buildspec.yml` (AWS CodeBuild) and `appspec.yml` (AWS CodeDeploy)
   - Check for `.aws-sam/` directory
   - Return `Detected: true` on first match; include which indicator triggered detection in `DetectionResult.Reason`
2. Implement `DevenvNixFragment() string`:
   - Always include `pkgs.awscli2`
   - Include `pkgs.aws-vault` when the consulting profile is active or when wizard selects credential helper (default: yes for consulting profile)
   - Include `pkgs.saml2aws` only when SAML-related keywords found in project docs (Okta, ADFS, OneLogin referenced in README or infra docs)
   - Include `pkgs.aws-sso-cli` only when explicit wizard opt-in
   - Set `env.AWS_PROFILE` with a commented TODO: `env.AWS_PROFILE = "TODO-set-client-profile-name";`
   - Set `env.AWS_DEFAULT_REGION` with detected region (from `samconfig.toml`, CDK context, or wizard input) or `"us-east-1"` default
   - Add shell hook comment: `# Run: aws sso login --profile $AWS_PROFILE`
   - Add `AWS_VAULT_BACKEND=pass` comment for NixOS/Linux when aws-vault is included
3. Implement `SecurityConfigs() []GeneratedFile`:
   - Generate `.env.example` snippet with `AWS_PROFILE=TODO` and `AWS_DEFAULT_REGION=TODO` entries
   - Include inline comment: `# Never set AWS_ACCESS_KEY_ID or AWS_SECRET_ACCESS_KEY here — use AWS_PROFILE + aws sso login`
   - Do not generate any file that sets credential values
4. Implement `DoctorChecks() []DoctorCheck` using the `DoctorCheckRegistry` from Unit 23.5:
   - `aws sts get-caller-identity` with 5s timeout — pass: identity printed, fail: actionable message with `aws sso login --profile $AWS_PROFILE` instruction
   - `aws-vault list` (when aws-vault package included) — verifies vault has entries
   - Check that `AWS_PROFILE` is set and non-empty in current environment
5. Implement `PreCommitHooks() []PreCommitHook`: return nil (gitleaks from Phase 5 already covers AWS key patterns)
6. Implement `DenyRules() []DenyRule`: return nil (AWS CLI is an operational tool, not a package manager)
7. Implement wizard integration:
   - When AWS detected, wizard asks: "AWS profile name for this project?" with text input, default `"TODO-client-name"`
   - Wizard asks: "Default region?" with text input, default `"us-east-1"`
   - Wizard explicitly warns: "Static credentials (AWS_ACCESS_KEY_ID, etc.) must never be set in devenv.nix. Use AWS profiles."
8. Write unit tests:
   - Detection from Terraform provider block (both `provider "aws"` and `source = "hashicorp/aws"`)
   - Detection from `cdk.json`
   - Detection from `serverless.yml`
   - Detection from `template.yaml` with `AWS::S3::Bucket` resource type
   - No detection when none of the indicator files exist
   - Generated fragment includes `awscli2` and `AWS_PROFILE` placeholder
   - Generated fragment never contains `AWS_ACCESS_KEY_ID` or `AWS_SECRET_ACCESS_KEY`
   - Doctor check registered with 5s timeout

**Acceptance Criteria:**
- [ ] Detects AWS usage from Terraform provider blocks, `cdk.json`, `serverless.yml`, SAM templates, `buildspec.yml`, and `.aws-sam/`
- [ ] devenv.nix fragment always includes `awscli2` and per-project `AWS_PROFILE = "TODO-..."` placeholder
- [ ] `aws-vault` included by default for consulting profile with `AWS_VAULT_BACKEND=pass` Linux note
- [ ] `saml2aws` included only when SAML IdP indicators detected in project docs
- [ ] Wizard collects profile name and region, warns against static credentials in generated files
- [ ] `gdev doctor` check runs `aws sts get-caller-identity` with 5-second timeout
- [ ] `gdev doctor` warns when `AWS_PROFILE` is unset
- [ ] No credential env vars (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN`) appear in any generated file

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md` § 1.1 AWS CLI v2 — detection heuristics, auth patterns, multi-account strategy
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md` § 1.2 AWS Credential Helpers — aws-vault, saml2aws, aws-sso-cli capabilities and Linux backend notes
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md` § 3.4 Should gdev Configure Credentials or Just Install Tools? — install+scaffold, never manage credentials
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md` § 4.1 Multi-Client Credential Isolation — per-project AWS_PROFILE pattern
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-module-design.md` Unit 2.9 — full step-by-step design for this module

**Status:** Not Started

---

### Unit 23.2: GCP CLI Module

**Description:** Implement the `EcosystemModule` for Google Cloud Platform. Detect GCP usage from Terraform provider blocks, App Engine, Cloud Build, and Firebase indicator files. Generate a devenv.nix fragment with `google-cloud-sdk`, per-project `CLOUDSDK_ACTIVE_CONFIG_NAME` isolation, and a `gdev doctor` auth check.

**Context:** GCP uses named configurations (`gcloud config configurations`) for multi-project isolation. The `CLOUDSDK_ACTIVE_CONFIG_NAME` environment variable selects the active configuration, analogous to `AWS_PROFILE`. The `google-cloud-sdk` Nixpkgs package includes `gcloud`, `gsutil`, and `bq`. Additional components like `gke-gcloud-auth-plugin` (required for GKE cluster auth) are added via `google-cloud-sdk.withExtraComponents` in Nix — this should trigger when both GCP and K8s are co-detected (coordinated with Phase 24). GCP uses Application Default Credentials (ADC) via `gcloud auth application-default login`; `GOOGLE_APPLICATION_CREDENTIALS` (service account key files, a legacy pattern) must NOT be set in generated devenv.nix.

**Desired Outcome:** When GCP usage is detected, `gdev init` generates a devenv.nix fragment with `google-cloud-sdk` (augmented with the GKE auth plugin when K8s co-detected), per-project `CLOUDSDK_ACTIVE_CONFIG_NAME`, and a `gdev doctor` check that verifies active auth via `gcloud auth print-access-token`.

**Steps:**
1. Implement `Detect(projectRoot string) (DetectionResult, error)`:
   - Call `TerraformProviderDetector` (Unit 23.6) and check for `"google"` or `"google-beta"` providers
   - Check for `app.yaml` containing App Engine runtime indicators (`runtime:` with GCP-specific values)
   - Check for `cloudbuild.yaml` (Cloud Build)
   - Check for `.gcloudignore`
   - Check for `firebase.json`
2. Implement `DevenvNixFragment() string`:
   - When K8s is NOT co-detected: `pkgs.google-cloud-sdk`
   - When K8s IS co-detected: `pkgs.google-cloud-sdk.withExtraComponents [ pkgs.google-cloud-sdk.components.gke-gcloud-auth-plugin ]`
   - K8s co-detection check: call the Phase 24 K8s detector or check for K8s indicator files directly (same list as Unit 24.1)
   - Set `env.CLOUDSDK_ACTIVE_CONFIG_NAME` with TODO placeholder
   - Set `env.CLOUDSDK_CORE_PROJECT` with TODO placeholder
   - Set `env.GOOGLE_CLOUD_PROJECT` with TODO placeholder (used by client libraries via ADC)
   - Add shell hook comment: `# Run: gcloud auth login && gcloud auth application-default login`
   - Add comment: `# GOOGLE_APPLICATION_CREDENTIALS is NOT set — use ADC via gcloud auth application-default login`
3. Implement `SecurityConfigs() []GeneratedFile`:
   - Generate `.env.example` with `CLOUDSDK_ACTIVE_CONFIG_NAME=TODO` and `GOOGLE_CLOUD_PROJECT=TODO`
   - Include note: `# GOOGLE_APPLICATION_CREDENTIALS must never be set here (service account key file — legacy pattern)`
4. Implement `DoctorChecks() []DoctorCheck`:
   - `gcloud auth print-access-token` with 5s timeout — non-zero exit means not authenticated
   - `gcloud config get-value project` — verifies a project is set in the active configuration
   - Check that `CLOUDSDK_ACTIVE_CONFIG_NAME` is set and non-empty
5. Implement `PreCommitHooks() []PreCommitHook`: return nil
6. Implement `DenyRules() []DenyRule`: return nil
7. Implement wizard integration:
   - Wizard asks: "GCP configuration name for this project?" with text input, default `"TODO-client-name"`
   - Wizard asks: "GCP project ID?" with text input, default `"TODO-project-id"`
8. Write unit tests:
   - Detection from Terraform provider `google` and `google-beta`
   - Detection from `app.yaml`
   - Detection from `cloudbuild.yaml`
   - Detection from `firebase.json`
   - No detection without indicator files
   - Fragment uses bare `google-cloud-sdk` when K8s not detected
   - Fragment uses `withExtraComponents [ gke-gcloud-auth-plugin ]` when K8s co-detected
   - `GOOGLE_APPLICATION_CREDENTIALS` absent from all generated output

**Acceptance Criteria:**
- [ ] Detects GCP from Terraform provider blocks (`google`, `google-beta`), `app.yaml`, `cloudbuild.yaml`, `.gcloudignore`, and `firebase.json`
- [ ] devenv.nix fragment includes bare `google-cloud-sdk` by default
- [ ] `gke-gcloud-auth-plugin` added via `withExtraComponents` when K8s co-detected (coordinated with Phase 24)
- [ ] Per-project `CLOUDSDK_ACTIVE_CONFIG_NAME`, `CLOUDSDK_CORE_PROJECT`, and `GOOGLE_CLOUD_PROJECT` env vars set with TODO placeholders
- [ ] `GOOGLE_APPLICATION_CREDENTIALS` explicitly NOT set in any generated file
- [ ] `gdev doctor` check runs `gcloud auth print-access-token` with 5-second timeout
- [ ] ADC pattern documented in generated comments

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md` § 1.3 GCP CLI (Google Cloud SDK) — auth patterns, ADC chain, named configurations, Nixpkgs withExtraComponents
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md` § 4.1 Multi-Client Credential Isolation — GCP isolation pattern with CLOUDSDK_ACTIVE_CONFIG_NAME
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-module-design.md` Unit 2.10 — full step-by-step design for this module

**Status:** Not Started

---

### Unit 23.3: Azure CLI Module

**Description:** Implement the `EcosystemModule` for Azure. Detect Azure usage from Terraform provider blocks, Azure Pipelines, Bicep, and Azure Developer CLI indicator files. Generate a devenv.nix fragment with `azure-cli`, per-project `ARM_SUBSCRIPTION_ID` and `AZURE_CONFIG_DIR` isolation, and a `gdev doctor` auth check.

**Context:** Azure uses subscription-based isolation; `ARM_SUBSCRIPTION_ID` and `ARM_TENANT_ID` are the Terraform-standard env vars for per-project scoping. Unlike AWS and GCP, Azure has no named-configuration system in the CLI — subscription selection is via `az account set`. Setting `AZURE_CONFIG_DIR` to a project-local path (e.g., `.azure-config/`) isolates the Azure CLI state (active subscription, cached tokens) from the global `~/.azure/` state, preventing cross-client contamination. For AKS authentication, `kubelogin` must be present — it should activate when both Azure and K8s are co-detected (coordinated with Phase 24). `ARM_CLIENT_SECRET` must never appear in any generated file.

**Desired Outcome:** When Azure usage is detected, `gdev init` generates a devenv.nix fragment with `azure-cli`, per-project ARM env vars, project-local `AZURE_CONFIG_DIR`, optional `kubelogin` when K8s co-detected, and a `gdev doctor` check that verifies login via `az account show`.

**Steps:**
1. Implement `Detect(projectRoot string) (DetectionResult, error)`:
   - Call `TerraformProviderDetector` (Unit 23.6) and check for `"azurerm"` or `"azuread"` providers
   - Check for `azure-pipelines.yml` (Azure DevOps)
   - Check for any `*.bicep` files in the project tree (recursive, up to 3 levels)
   - Check for `azure.yaml` (Azure Developer CLI project)
   - Check for `.azure/` directory
2. Implement `DevenvNixFragment() string`:
   - Always include `pkgs.azure-cli`
   - Include `pkgs.kubelogin` when Azure + K8s co-detected (AKS AAD auth requirement)
   - Set `env.ARM_SUBSCRIPTION_ID` with TODO placeholder
   - Set `env.ARM_TENANT_ID` with TODO placeholder
   - Set `env.AZURE_CONFIG_DIR` to a project-local path: `env.AZURE_CONFIG_DIR = "${config.devenv.root}/.azure-config";`
   - Add shell hook comment: `# Run: az login --tenant $ARM_TENANT_ID`
   - When `azure.yaml` detected: add comment `# Azure Developer CLI (azd) available — run: azd auth login`
3. Implement `SecurityConfigs() []GeneratedFile`:
   - Generate `.env.example` with `ARM_SUBSCRIPTION_ID=TODO` and `ARM_TENANT_ID=TODO`
   - Include explicit warning: `# ARM_CLIENT_SECRET and ARM_CLIENT_ID must never be set here — use az login`
   - Add `.azure-config/` to project `.gitignore` (generated file contains cached tokens)
4. Implement `DoctorChecks() []DoctorCheck`:
   - `az account show` with 5s timeout — verifies login status and shows active subscription
   - `az account show --query id -o tsv` output compared against `ARM_SUBSCRIPTION_ID` env var — warn if mismatch
   - Check that `ARM_SUBSCRIPTION_ID` is set and non-empty
5. Implement `PreCommitHooks() []PreCommitHook`: return nil
6. Implement `DenyRules() []DenyRule`: return nil
7. Implement wizard integration:
   - Wizard asks: "Azure subscription ID for this project?" with text input, default `"TODO-subscription-id"`
   - Wizard asks: "Azure tenant ID?" with text input, default `"TODO-tenant-id"`
8. Write unit tests:
   - Detection from Terraform `azurerm` and `azuread` provider blocks
   - Detection from `azure-pipelines.yml`
   - Detection from `*.bicep` file anywhere in project tree
   - Detection from `.azure/` directory
   - Fragment includes `azure-cli` + `ARM_SUBSCRIPTION_ID` placeholder
   - `kubelogin` added when K8s co-detected
   - `ARM_CLIENT_SECRET` absent from all generated output
   - `.azure-config/` added to `.gitignore`

**Acceptance Criteria:**
- [ ] Detects Azure from Terraform provider blocks (`azurerm`, `azuread`), `azure-pipelines.yml`, `*.bicep` files, `azure.yaml`, and `.azure/`
- [ ] devenv.nix fragment includes `azure-cli` with per-project `ARM_SUBSCRIPTION_ID`, `ARM_TENANT_ID`, and `AZURE_CONFIG_DIR` env vars
- [ ] `AZURE_CONFIG_DIR` set to project-local path to isolate Azure CLI state from `~/.azure/`
- [ ] `.azure-config/` added to `.gitignore` in `SecurityConfigs`
- [ ] `kubelogin` included when Azure + K8s co-detected (AKS)
- [ ] `gdev doctor` checks login via `az account show` and warns on subscription mismatch
- [ ] `ARM_CLIENT_SECRET` and `ARM_CLIENT_ID` explicitly excluded from all generated files

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md` § 1.4 Azure CLI — two CLIs, auth patterns, multi-subscription management
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md` § 4.1 Multi-Client Credential Isolation — Azure isolation pattern with ARM_SUBSCRIPTION_ID
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-module-design.md` Unit 2.11 — full step-by-step design for this module

**Status:** Not Started

---

### Unit 23.4: Tier 3 Cloud Platform CLIs Module

**Description:** Implement a single `EcosystemModule` that covers five Tier 3 cloud platform CLIs — Cloudflare Workers (`wrangler`), DigitalOcean (`doctl`), Fly.io (`flyctl`), Vercel, and Netlify. Each platform is detected by its own indicator file and added independently; multiple platforms can coexist in one project.

**Context:** These platforms are project-specific rather than universally needed. Unlike Tier 1 cloud providers, they are typically used by a single project and do not require the multi-client credential isolation patterns of AWS/GCP/Azure. Each has a simple detection heuristic (one config file), a single CLI package in Nixpkgs, and a single `gdev doctor` auth check. This is a single module with internal sub-detection rather than five separate modules, because they share the same Tier 3 pattern: auto-detect, no wizard questions, no generated config files beyond the devenv.nix fragment.

**Desired Outcome:** When any Tier 3 platform indicator file is present, `gdev init` includes exactly the matching CLI(s) in the devenv.nix packages list with auth reminder comments. No cross-installation occurs — detecting `wrangler.toml` adds `wrangler` but not the other four platforms.

**Steps:**
1. Implement `Detect(projectRoot string) (DetectionResult, error)` with per-platform sub-heuristics:
   - **Cloudflare**: `wrangler.toml` or `wrangler.jsonc`
   - **DigitalOcean**: `.do/app.yaml` or `.do/` directory
   - **Fly.io**: `fly.toml`
   - **Vercel**: `vercel.json` or `.vercel/` directory
   - **Netlify**: `netlify.toml` or `.netlify/` directory
   - `DetectionResult` includes a `DetectedPlatforms []string` field listing which platforms triggered
   - Return `Detected: true` if any platform matches; `Detected: false` only if none match
2. Implement `DevenvNixFragment() string` — emit packages for all detected platforms:
   - Cloudflare: `pkgs.wrangler` + shell hook comment `# Run: wrangler login`
   - DigitalOcean: `pkgs.doctl` + comment `# Run: doctl auth init`
   - Fly.io: `pkgs.flyctl` + comment `# Run: fly auth login`
   - Vercel: `pkgs.nodePackages.vercel` + comment `# Run: vercel login`
   - Netlify: `pkgs.netlify-cli` + comment `# Run: netlify login`
3. Implement `SecurityConfigs() []GeneratedFile`: return nil (no config files for these platforms)
4. Implement `DoctorChecks() []DoctorCheck` — one check per detected platform, each with 5s timeout:
   - Cloudflare: `wrangler whoami`
   - DigitalOcean: `doctl account get`
   - Fly.io: `fly auth whoami`
   - Vercel: `vercel whoami`
   - Netlify: `netlify status`
5. Implement `PreCommitHooks() []PreCommitHook`: return nil
6. Implement `DenyRules() []DenyRule`: return nil
7. Write unit tests:
   - Cloudflare detected from `wrangler.toml`, not detected without it
   - Fly.io detected from `fly.toml`, not detected without it
   - Two platforms detected simultaneously (e.g., `wrangler.toml` + `fly.toml` → both CLIs included)
   - Doctor checks registered only for detected platforms (not all five)
   - Fragment is empty when no platform indicator files exist

**Acceptance Criteria:**
- [ ] Detects each of the five platforms independently from its indicator file(s)
- [ ] Only detected platforms are included in devenv.nix (no blanket installation)
- [ ] Multiple platforms can be detected simultaneously and all are included
- [ ] Each detected platform has exactly one `gdev doctor` auth check with 5-second timeout
- [ ] Auth reminder comments are platform-specific (correct login command per platform)
- [ ] No config files generated (Tier 3: install only)

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md` § 1.5 Other Cloud CLIs — package names, versions, detection heuristics, consulting relevance
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md` § 5 Recommended Tiering — Tier 3/4 classification for project-specific cloud platforms
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-module-design.md` Unit 2.12 — full step-by-step design for this module

**Status:** Not Started

---

### Unit 23.5: Cloud Credential Isolation Engine

**Description:** Implement the shared `CloudEnvVarTemplate` helper, `DoctorCheckRegistry` infrastructure, and an `.envrc` ambient-credential warning generator that all cloud modules (23.1-23.4) depend on.

**Context:** All cloud provider modules share three cross-cutting concerns. First, credential isolation: the `CloudEnvVarTemplate` must refuse to emit any env var name matching secret-bearing patterns (`*_SECRET*`, `*_KEY*`, `*_TOKEN*`, `*_PASSWORD*`) — this is the hard guarantee that no credential can leak into generated files. Second, doctor check infrastructure: all cloud checks need 5-second timeouts and graceful degradation when CLI binaries are absent (report "not installed" rather than a crash). Third, direnv ambient credential detection: gdev should warn when ambient cloud credentials (sourced from outside devenv scope, e.g., a shell-level `AWS_ACCESS_KEY_ID`) are present at environment entry time, since they can shadow the per-project profile settings.

**Desired Outcome:** A shared engine that `CloudEnvVarTemplate` uses to block credential variable names, a `DoctorCheckRegistry` with timeout and graceful-degradation semantics, and `.envrc` additions that emit a visible warning when ambient cloud credentials are detected on `direnv allow`.

**Steps:**
1. Implement `CloudEnvVarTemplate` in `internal/cloud/envvar.go`:
   ```go
   // CredentialPatterns lists env var name patterns that are never allowed in generated files.
   var CredentialPatterns = []string{
       "*_SECRET*", "*_KEY*", "*_TOKEN*", "*_PASSWORD*",
       "*_ACCESS_KEY*", "*_PRIVATE_KEY*", "*_CLIENT_SECRET*",
   }

   // SetEnvVar adds an env var to the template, returning an error if the name
   // matches a credential pattern.
   func (t *CloudEnvVarTemplate) SetEnvVar(name, value string) error {
       for _, pattern := range CredentialPatterns {
           if matchGlob(pattern, name) {
               return fmt.Errorf(
                   "env var %q matches credential pattern %q and cannot be set in devenv.nix; "+
                   "use per-engineer credential storage (aws-vault, gcloud auth, az login)",
                   name, pattern,
               )
           }
       }
       t.vars[name] = value
       return nil
   }

   // Render returns a devenv.nix-compatible `env = { ... }` block string.
   func (t *CloudEnvVarTemplate) Render() string
   ```
2. Implement per-project isolation validation:
   - `ValidateProjectIsolation(vars map[string]string) []IsolationWarning` — checks that cloud env vars in one project's devenv.nix do not shadow values that would be used by another project's devenv.nix running simultaneously in the same shell session (direnv scoping provides this, but validate the KUBECONFIG pattern specifically: per-project path rather than `~/.kube/config`)
3. Implement `DoctorCheckRegistry` in `internal/cloud/doctor.go`:
   ```go
   type DoctorCheck struct {
       Name        string
       Command     []string      // argv, not shell string
       Timeout     time.Duration // default 5s
       PassMessage string
       FailMessage string        // includes actionable fix instruction
       FixCommand  string        // suggested command to run on failure
   }

   // RunCheck executes the check with timeout and returns pass/fail/skip.
   func RunCheck(check DoctorCheck) DoctorResult

   type DoctorResult struct {
       Name    string
       Status  string // "pass", "fail", "skip" (binary not found), "timeout"
       Message string
       Fix     string
   }
   ```
4. Implement graceful degradation in `RunCheck`:
   - If the CLI binary is not on PATH: return `Status: "skip"` with message `"<binary> not installed — skipping check"`
   - If the command times out: return `Status: "timeout"` with message `"check timed out after 5s (VPN connected? Network reachable?)"` and fix instruction
   - If exit code is non-zero: return `Status: "fail"` with `FailMessage` and `FixCommand`
5. Implement `.envrc` ambient credential warning generator:
   - Generate a shell function that runs on `direnv allow`:
     ```bash
     # gdev: warn if ambient cloud credentials detected outside devenv scope
     if [ -n "${AWS_ACCESS_KEY_ID:-}" ]; then
       echo "gdev WARNING: AWS_ACCESS_KEY_ID set in ambient shell — may shadow devenv per-project AWS_PROFILE"
     fi
     if [ -n "${GOOGLE_APPLICATION_CREDENTIALS:-}" ]; then
       echo "gdev WARNING: GOOGLE_APPLICATION_CREDENTIALS set in ambient shell — prefer ADC via gcloud auth application-default login"
     fi
     ```
   - Warnings are informational (no exit code change)
   - `.envrc` section managed by gdev section markers (per Phase 3 shared-file surgery patterns)
6. Write unit tests:
   - `SetEnvVar` accepts `AWS_PROFILE` (not a credential)
   - `SetEnvVar` rejects `AWS_SECRET_ACCESS_KEY` (matches `*_SECRET*` and `*_KEY*`)
   - `SetEnvVar` rejects `GOOGLE_API_KEY` (matches `*_KEY*`)
   - `SetEnvVar` rejects `ARM_CLIENT_SECRET` (matches `*_SECRET*`)
   - `RunCheck` returns `skip` when binary not on PATH
   - `RunCheck` returns `timeout` when command hangs past 5s
   - Ambient credential warnings generated for AWS and GCP patterns

**Acceptance Criteria:**
- [ ] `CloudEnvVarTemplate.SetEnvVar` rejects names matching `*_SECRET*`, `*_KEY*`, `*_TOKEN*`, `*_PASSWORD*`, `*_ACCESS_KEY*`, `*_PRIVATE_KEY*`, `*_CLIENT_SECRET*`
- [ ] Template accepts non-credential env vars like `AWS_PROFILE`, `CLOUDSDK_ACTIVE_CONFIG_NAME`, `ARM_SUBSCRIPTION_ID`
- [ ] `DoctorCheckRegistry` executes checks with 5-second default timeout
- [ ] Doctor checks return `skip` when CLI binary is absent (no crash)
- [ ] Doctor checks return `timeout` with network-failure hint message
- [ ] `.envrc` ambient credential warnings generated for `AWS_ACCESS_KEY_ID`, `GOOGLE_APPLICATION_CREDENTIALS`, and `ARM_CLIENT_ID`
- [ ] `.envrc` section managed by gdev section markers (does not overwrite user content)

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md` § 3.4 Should gdev Configure Credentials or Just Install Tools? — the 6 things gdev SHOULD do and 4 it should NOT do
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md` § 4.1 Multi-Client Credential Isolation — env var isolation patterns and ambient credential problem
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-module-design.md` Unit 2.13 — shared cloud module infrastructure design

**Status:** Not Started

---

### Unit 23.6: Cloud Provider Detection from Terraform

**Description:** Implement `TerraformProviderDetector`, a shared parser that reads `*.tf` files and extracts all declared cloud providers from `required_providers` blocks and bare `provider` blocks. This is the primary detection mechanism for AWS, GCP, and Azure modules and feeds pre-populated wizard defaults.

**Context:** The Terraform ecosystem module (Phase 2, Unit 2.8) already parses `*.tf` files for Terraform usage. This unit extends that parser to extract cloud provider identities from two syntactic forms: bare `provider "aws" {}` blocks and the modern `required_providers { aws = { source = "hashicorp/aws" } }` form. Multi-cloud projects (Terraform spanning AWS and GCP, for example) are handled by detecting all providers and returning a set — the calling modules decide which of their providers are present. The parser must be shared (not duplicated) across all three cloud modules.

**Desired Outcome:** A `TerraformProviderDetector` that parses all `*.tf` files in a project, returns the set of detected cloud providers, and feeds that set to cloud module `Detect()` calls. Multi-cloud projects correctly activate all matching cloud modules simultaneously.

**Steps:**
1. Implement `TerraformProviderDetector` in `internal/cloud/terraform_detect.go`:
   ```go
   // ProviderSet is the set of Terraform provider source identifiers found in *.tf files.
   type ProviderSet map[string]struct{} // e.g., {"hashicorp/aws": {}, "hashicorp/google": {}}

   // DetectTerraformProviders scans all *.tf files in projectRoot (non-recursive
   // into .git, .devenv) and returns the complete set of provider source identifiers.
   func DetectTerraformProviders(projectRoot string) (ProviderSet, error)
   ```
2. Implement parser for two syntactic forms:
   - **Form 1** — bare `provider` block:
     ```hcl
     provider "aws" {
       region = "us-east-1"
     }
     ```
     Extracts: `"aws"` (short name, normalized to `"hashicorp/aws"`)
   - **Form 2** — `required_providers` source string:
     ```hcl
     terraform {
       required_providers {
         aws = {
           source  = "hashicorp/aws"
           version = "~> 5.0"
         }
       }
     }
     ```
     Extracts: `"hashicorp/aws"` (full source address)
   - Normalization table: `"aws"` → `"hashicorp/aws"`, `"google"` → `"hashicorp/google"`, `"azurerm"` → `"hashicorp/azurerm"`, `"azuread"` → `"hashicorp/azuread"`, `"cloudflare"` → `"cloudflare/cloudflare"`, `"digitalocean"` → `"digitalocean/digitalocean"`
   - Use `strings.Contains` and simple line scanning rather than a full HCL parser (avoid adding a heavy dependency; `.tf` provider declarations are structurally predictable)
3. Implement `ProviderSet` helper methods:
   ```go
   func (p ProviderSet) HasAWS() bool     { return p["hashicorp/aws"] != (struct{}{}) || ... }
   func (p ProviderSet) HasGCP() bool     { ... }
   func (p ProviderSet) HasAzure() bool   { ... }
   func (p ProviderSet) HasCloudflare() bool { ... }
   ```
4. Implement result caching: run the scan once per `gdev init` invocation, cache in a module-level variable so all cloud modules share a single parse pass
5. Wire into cloud module `Detect()` calls: Units 23.1, 23.2, 23.3, and 23.4 all call `DetectTerraformProviders(projectRoot)` and query the returned `ProviderSet`
6. Write unit tests:
   - Bare `provider "aws"` block detected → `HasAWS() == true`
   - `required_providers` with `source = "hashicorp/aws"` detected
   - `required_providers` with `source = "hashicorp/google"` detected → `HasGCP() == true`
   - Multi-cloud: file with both `azurerm` and `aws` providers detected → both `HasAzure()` and `HasAWS()` true
   - Empty project (no `.tf` files) → empty ProviderSet, no error
   - `.tf` files with non-cloud providers (e.g., `hashicorp/kubernetes`) do not trigger cloud modules

**Acceptance Criteria:**
- [ ] Parses both bare `provider "X"` blocks and `required_providers { X = { source = "..." } }` forms
- [ ] Short names (`aws`, `google`, `azurerm`) normalized to full source addresses (`hashicorp/aws`, etc.)
- [ ] Multi-cloud: all providers in a single project detected simultaneously
- [ ] Result is cached per `gdev init` invocation — `.tf` files scanned only once
- [ ] Non-cloud providers (kubernetes, helm, vault) do not trigger cloud module detection
- [ ] Empty or non-Terraform projects return empty ProviderSet without error
- [ ] Shared by all three Tier 1 cloud modules (no per-module duplication)

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md` § 3.3 Detection Heuristics — Terraform provider parsing strategy, two syntactic forms
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-module-design.md` Unit 2.13 — shared cloud module infrastructure design including Terraform provider parser

**Status:** Not Started

---

### Unit 23.7: Cloud Credential Doctor Checks

**Description:** Implement the full set of `gdev doctor` cloud checks: credential validity, profile/config name match, region/project match, and `KUBECONFIG` shared-path warning. Checks run with 5-second timeouts and a no-network fallback mode.

**Context:** `gdev doctor` already performs machine and tool health checks (Phase 15). This unit extends it with a cloud credential health category. The checks must be non-destructive and safe to run in CI. Network-dependent commands (AWS, GCP, Azure auth checks) must time out cleanly — consulting engineers frequently work from restricted networks or disconnected laptops. The no-network mode (`--offline`) checks only env var presence and config file existence, which is always fast.

**Desired Outcome:** `gdev doctor --category cloud` runs all registered cloud checks, reports credential validity for each detected cloud provider, warns on profile/context mismatches, and completes in under 30 seconds even when cloud API endpoints are unreachable.

**Steps:**
1. Register cloud checks with the `gdev doctor` check runner (Phase 15 infrastructure):
   - Category: `"cloud"` (new category alongside existing `system`, `tools`, `project`)
   - Each check registered with: name, description, required packages, network-dependent flag
2. Implement per-provider checks:

   **AWS checks:**
   - `aws-profile-set`: `AWS_PROFILE` env var is non-empty — no network, instant
   - `aws-auth`: `aws sts get-caller-identity` — 5s timeout, network-dependent
   - `aws-region-set`: `AWS_DEFAULT_REGION` or `AWS_REGION` is non-empty — no network, instant

   **GCP checks:**
   - `gcp-config-set`: `CLOUDSDK_ACTIVE_CONFIG_NAME` env var is non-empty — instant
   - `gcp-auth`: `gcloud auth print-access-token` — 5s timeout, network-dependent
   - `gcp-project-set`: `gcloud config get-value project` output matches `GOOGLE_CLOUD_PROJECT` env var — 5s timeout

   **Azure checks:**
   - `azure-sub-set`: `ARM_SUBSCRIPTION_ID` env var is non-empty — instant
   - `azure-auth`: `az account show` — 5s timeout, network-dependent
   - `azure-sub-match`: compare `az account show --query id -o tsv` output against `ARM_SUBSCRIPTION_ID` — 5s timeout

   **K8s check (registered here for cross-cloud concern):**
   - `kubeconfig-not-shared`: warn if `KUBECONFIG` equals `~/.kube/config` or is unset (shared-path risk)
3. Implement `--offline` / `--no-network` mode:
   - When `--offline` flag passed: skip all checks marked `network-dependent: true`
   - Run env var presence checks and config file existence checks only
   - Report skipped checks as `skip (offline mode)` rather than failures
4. Implement check output formatting consistent with existing `gdev doctor` output:
   - Pass: green checkmark with one-line summary
   - Fail: red X with remediation command
   - Skip: yellow dash with reason
   - Timeout: orange clock with network hint
5. Add cloud category to `gdev doctor --help` output and the generated CLAUDE.md doctor section
6. Write integration tests:
   - With `AWS_PROFILE` set and mock `aws sts get-caller-identity` returning success: `aws-auth` passes
   - With `AWS_PROFILE` unset: `aws-profile-set` fails with message to set `env.AWS_PROFILE` in devenv.nix
   - With `KUBECONFIG=~/.kube/config`: `kubeconfig-not-shared` warns about shared-state risk
   - `--offline` mode: network-dependent checks skipped, env var checks run
   - Timeout: mock command that sleeps 10s → returns `timeout` after 5s

**Acceptance Criteria:**
- [ ] `gdev doctor --category cloud` runs all registered cloud credential checks
- [ ] Checks for all three Tier 1 providers: AWS (profile-set, auth, region), GCP (config-set, auth, project-match), Azure (sub-set, auth, sub-match)
- [ ] `KUBECONFIG` shared-path warning fires when `KUBECONFIG` equals `~/.kube/config`
- [ ] All network-dependent checks enforce 5-second timeout
- [ ] `--offline` mode skips network-dependent checks and reports `skip (offline mode)`
- [ ] Each failing check includes a specific remediation command
- [ ] Total `gdev doctor --category cloud` run time under 30 seconds even when all network checks timeout

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md` § 4.1 Multi-Client Credential Isolation — per-provider isolation patterns and what to verify
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-module-design.md` Units 2.9-2.14 — per-provider doctor check specifications
- `phases/15-health-status-compliance-reporting.md` — `gdev doctor` infrastructure this unit extends

**Status:** Not Started

---

## Code-Grounded Implementation Notes

### Interface to Implement

All units in this phase implement the `EcosystemModule` interface established in Phase 1 and first used in Phase 2 (`phases/02-ecosystem-modules-tier1.md`). No interface changes are needed.

### Reuse from Phase 2

The Terraform module (Phase 2, Unit 2.8) already parses `*.tf` files. Unit 23.6 extends rather than duplicates that work. The exact integration point must be confirmed by reading `addons/devinit/` module code before implementation.

### New Packages (All in Nixpkgs)

| Package | Nixpkgs Attribute | Used By |
|---------|-------------------|---------|
| `awscli2` | `pkgs.awscli2` | Unit 23.1 |
| `aws-vault` | `pkgs.aws-vault` | Unit 23.1 |
| `saml2aws` | `pkgs.saml2aws` | Unit 23.1 (conditional) |
| `google-cloud-sdk` | `pkgs.google-cloud-sdk` | Unit 23.2 |
| `azure-cli` | `pkgs.azure-cli` | Unit 23.3 |
| `kubelogin` | `pkgs.kubelogin` | Unit 23.3 (K8s co-detected) |
| `wrangler` | `pkgs.wrangler` | Unit 23.4 |
| `doctl` | `pkgs.doctl` | Unit 23.4 |
| `flyctl` | `pkgs.flyctl` | Unit 23.4 |
| `netlify-cli` | `pkgs.netlify-cli` | Unit 23.4 |

### Credential Safety Invariant

The credential safety invariant enforced by Unit 23.5 applies globally across all units in this phase: no generated file (devenv.nix, .envrc, .env.example, CLAUDE.md) may contain an env var name matching `*_SECRET*`, `*_KEY*`, `*_TOKEN*`, or `*_PASSWORD*`. This invariant must be tested in every unit.

---

## Phase Completion Criteria

- [ ] All seven units pass acceptance criteria
- [ ] AWS, GCP, and Azure modules correctly detect their respective indicator files and are independent (detecting AWS does not trigger GCP)
- [ ] Multi-cloud projects: project with both AWS and GCP Terraform providers activates both modules
- [ ] Credential safety invariant: `grep` of all generated test output confirms no `*_SECRET*`, `*_KEY*`, `*_TOKEN*`, or `*_PASSWORD*` env vars in any generated file
- [ ] Tier 3 platform module: all five platforms independently detectable and only detected platforms appear in devenv.nix
- [ ] All cloud doctor checks have 5-second timeouts; `--offline` mode skips network checks
- [ ] `.envrc` ambient credential warnings fire when `AWS_ACCESS_KEY_ID` or `GOOGLE_APPLICATION_CREDENTIALS` detected in ambient shell
- [ ] Terraform provider parser correctly handles both syntactic forms and multi-provider files
- [ ] Phase 24 K8s co-detection integration points defined: GCP+K8s adds `gke-gcloud-auth-plugin`, Azure+K8s adds `kubelogin`
