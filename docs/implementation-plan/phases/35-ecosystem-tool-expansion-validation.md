# Phase 35: Ecosystem & Tool Expansion Validation

## Goal

Validate all ecosystem expansion phases using the testscript E2E framework from Phase 17: cloud CLI modules (Phase 23), Kubernetes ecosystem (Phase 24), service templates (Phase 25), non-language tool detection (Phase 26), and IDE/shell/workstation configuration (Phase 27). Each unit exercises both the detection engine (correct signals trigger correct modules) and the generation output (correct devenv.nix fragments, env vars, and config files are produced), using `--answers-file` for non-interactive testing throughout.

## Dependencies

Phase 17 complete (test infrastructure framework — testscript E2E framework, custom commands like `yaml_has`, `json_path`, `nix_valid`, golden file infrastructure, CI pipeline). Phase 23 complete (cloud CLI modules, AWS/GCP/Azure, `TerraformProviderDetector`, `DoctorCheckRegistry`). Phase 24 complete (Kubernetes ecosystem module, kubectl version pinning, K8s security tools, Helm). Phase 25 complete (service template expansion — Kafka, MinIO, Mailpit, Keycloak, NATS, tiered detection engine). Phase 26 complete (non-language tool detection — git platform tools, documentation generators, API tools, database migration tools). Phase 27 complete (IDE/shell/workstation configuration — EditorConfig, VS Code extensions.json, shell fragments, Starship).

## Phase Outputs

- Cloud CLI module validation test suite (AWS/GCP/Azure detection, credential isolation, doctor check mocks)
- Kubernetes ecosystem validation test suite (kubectl version pinning, K8s security tools, Helm, cloud-auth coordination, KUBECONFIG isolation)
- Service template validation test suite (Kafka KRaft, MinIO S3 env vars, Mailpit ports, Keycloak dev mode, NATS JetStream, tiering correctness)
- Non-language tool detection validation test suite (git platform, docs, API, database migration, mixed projects, module ordering)
- IDE/shell/workstation configuration validation test suite (EditorConfig, VS Code extensions, shell fragments, Starship, personal tools, shell aliases)

---

### Unit 35.1: Cloud CLI Module Validation

**Description:** Validate detection accuracy and generation output for all cloud CLI modules: AWS, GCP, Azure (Tier 1), and Tier 3 platform CLIs (Cloudflare/wrangler, Fly.io). Verify that per-project credential env vars are generated correctly, that NO secret values appear in any generated file, that multi-cloud projects detect all providers, and that `qsdev doctor` cloud checks work correctly against mocked CLI responses.

**Context:** Phase 23 introduced the `TerraformProviderDetector` and per-cloud `EcosystemModule` implementations. The single highest consulting safety risk is cross-client credential leakage — an `AWS_PROFILE` bound at project scope in devenv.nix prevents accidentally running Terraform against the wrong client account. The inverse risk is equally dangerous: static credentials (`AWS_SECRET_ACCESS_KEY`, `ARM_CLIENT_SECRET`, `GCP_CREDENTIALS`) appearing in generated files would constitute a credential leak on every `git push`. The `CloudEnvVarTemplate` helper enforces this by refusing to emit secret-bearing variable names. The `DoctorCheckRegistry` runs auth-check commands with 5-second timeouts and degrades gracefully when the CLI is not authenticated, so tests must mock CLI responses rather than requiring real cloud credentials.

**Desired Outcome:** A test suite proving that cloud detection is triggered by the correct Terraform provider blocks, that generated devenv.nix fragments contain only safe env vars, that no secret names appear in any generated output, that multi-cloud projects detect all three providers, that Tier 3 CLIs are detected from their native config files, and that `qsdev doctor` cloud checks produce structured pass/fail output against mocked CLI responses.

**Steps:**
1. Create `e2e/testdata/script/cloud/` directory for cloud module test scripts.
2. Write `aws-detection-terraform.txtar` — verify AWS detection from Terraform provider block:
   ```
   # AWS detected from terraform required_providers aws block
   exec qsdev init --non-interactive --answers-file answers.yaml
   exec gdev detect --format json > detect.json
   json_path detect.json '.modules' 'some' '.name=="aws"'
   json_path detect.json '.modules[]|select(.name=="aws")' '.reason' 'contains' 'hashicorp/aws'
   exec grep 'awscli2' devenv.nix
   exec grep 'AWS_PROFILE' devenv.nix
   ! exec grep 'AWS_SECRET_ACCESS_KEY' devenv.nix
   ! exec grep 'AWS_ACCESS_KEY_ID' devenv.nix

   -- answers.yaml --
   quick_mode: true

   -- main.tf --
   terraform {
     required_providers {
       aws = {
         source  = "hashicorp/aws"
         version = "~> 5.0"
       }
     }
   }
   ```
3. Write `aws-detection-cdk.txtar` — verify AWS detection from CDK indicator:
   ```
   # AWS detected from cdk.json presence
   exec qsdev init --non-interactive --answers-file answers.yaml
   exec gdev detect --format json > detect.json
   json_path detect.json '.modules[]|select(.name=="aws")' '.reason' 'contains' 'cdk.json'

   -- answers.yaml --
   quick_mode: true

   -- cdk.json --
   {"app": "npx ts-node --prefer-ts-exts bin/app.ts"}
   ```
4. Write `gcp-detection-terraform.txtar` — verify GCP detection from google provider block:
   ```
   # GCP detected from terraform required_providers google block
   exec qsdev init --non-interactive --answers-file answers.yaml
   exec grep 'google-cloud-sdk' devenv.nix
   exec grep 'CLOUDSDK_CORE_PROJECT' devenv.nix
   ! exec grep 'GOOGLE_APPLICATION_CREDENTIALS' devenv.nix

   -- answers.yaml --
   quick_mode: true

   -- main.tf --
   terraform {
     required_providers {
       google = {
         source  = "hashicorp/google"
         version = "~> 5.0"
       }
     }
   }
   ```
5. Write `azure-detection-terraform.txtar` — verify Azure detection from azurerm provider block:
   ```
   # Azure detected from terraform required_providers azurerm block
   exec qsdev init --non-interactive --answers-file answers.yaml
   exec grep 'azure-cli' devenv.nix
   exec grep 'ARM_SUBSCRIPTION_ID' devenv.nix
   ! exec grep 'ARM_CLIENT_SECRET' devenv.nix
   ! exec grep 'ARM_CLIENT_ID' devenv.nix

   -- answers.yaml --
   quick_mode: true

   -- main.tf --
   terraform {
     required_providers {
       azurerm = {
         source  = "hashicorp/azurerm"
         version = "~> 3.0"
       }
     }
   }
   ```
6. Write `multi-cloud-detection.txtar` — verify all three providers detected from one Terraform file:
   ```
   # Multi-cloud: aws + google + azurerm all detected from single tf file
   exec qsdev init --non-interactive --answers-file answers.yaml
   exec gdev detect --format json > detect.json
   json_path detect.json '.modules' 'some' '.name=="aws"'
   json_path detect.json '.modules' 'some' '.name=="gcp"'
   json_path detect.json '.modules' 'some' '.name=="azure"'
   exec grep 'awscli2' devenv.nix
   exec grep 'google-cloud-sdk' devenv.nix
   exec grep 'azure-cli' devenv.nix

   -- answers.yaml --
   quick_mode: true

   -- main.tf --
   terraform {
     required_providers {
       aws    = { source = "hashicorp/aws",    version = "~> 5.0" }
       google = { source = "hashicorp/google", version = "~> 5.0" }
       azurerm = { source = "hashicorp/azurerm", version = "~> 3.0" }
     }
   }
   ```
7. Write `no-secret-names-in-output.txtar` — exhaustive negative test for all known secret env var names:
   ```
   # No credential secret names appear in any generated file
   exec qsdev init --non-interactive --answers-file answers.yaml
   ! exec grep -r 'AWS_SECRET_ACCESS_KEY' devenv.nix devenv.yaml .envrc
   ! exec grep -r 'AWS_SESSION_TOKEN' devenv.nix devenv.yaml .envrc
   ! exec grep -r 'ARM_CLIENT_SECRET' devenv.nix devenv.yaml .envrc
   ! exec grep -r 'ARM_CLIENT_CERTIFICATE' devenv.nix devenv.yaml .envrc
   ! exec grep -r 'GOOGLE_APPLICATION_CREDENTIALS' devenv.nix devenv.yaml .envrc
   ! exec grep -r 'GCP_CREDENTIALS' devenv.nix devenv.yaml .envrc
   ! exec grep -r 'CLOUDFLARE_API_TOKEN' devenv.nix devenv.yaml .envrc
   ! exec grep -r 'FLY_API_TOKEN' devenv.nix devenv.yaml .envrc

   -- answers.yaml --
   quick_mode: true

   -- main.tf --
   terraform {
     required_providers {
       aws    = { source = "hashicorp/aws" }
       google = { source = "hashicorp/google" }
       azurerm = { source = "hashicorp/azurerm" }
     }
   }
   ```
8. Write `wrangler-toml-detection.txtar` — verify Cloudflare wrangler detected from wrangler.toml:
   ```
   # Tier 3: wrangler.toml -> wrangler CLI detected
   exec qsdev init --non-interactive --answers-file answers.yaml
   exec gdev detect --format json > detect.json
   json_path detect.json '.modules[]|select(.name=="cloudflare")' '.reason' 'contains' 'wrangler.toml'
   exec grep 'wrangler' devenv.nix

   -- answers.yaml --
   quick_mode: true

   -- wrangler.toml --
   name = "my-worker"
   compatibility_date = "2024-01-01"
   ```
9. Write `fly-toml-detection.txtar` — verify Fly.io flyctl detected from fly.toml:
   ```
   # Tier 3: fly.toml -> flyctl detected
   exec qsdev init --non-interactive --answers-file answers.yaml
   exec gdev detect --format json > detect.json
   json_path detect.json '.modules[]|select(.name=="fly")' '.reason' 'contains' 'fly.toml'
   exec grep 'flyctl' devenv.nix

   -- answers.yaml --
   quick_mode: true

   -- fly.toml --
   app = "my-app"
   primary_region = "iad"
   ```
10. Write `doctor-cloud-checks-pass.txtar` — verify `qsdev doctor` cloud checks with mocked passing CLI responses:
    ```
    # qsdev doctor: cloud checks pass against mocked CLI responses
    # Use PATH override to inject mock aws/gcloud/az scripts
    env PATH=$WORK/mock-bin:$PATH
    exec qsdev init --non-interactive --answers-file answers.yaml
    exec qsdev doctor --check cloud --format json > doctor.json
    json_path doctor.json '.checks[]|select(.name=="aws-auth")' '.status' 'pass'

    -- answers.yaml --
    quick_mode: true

    -- main.tf --
    terraform { required_providers { aws = { source = "hashicorp/aws" } } }

    -- mock-bin/aws --
    #!/bin/sh
    if [ "$1" = "sts" ] && [ "$2" = "get-caller-identity" ]; then
      echo '{"UserId": "AIDATEST", "Account": "123456789012", "Arn": "arn:aws:iam::123456789012:user/test"}'
      exit 0
    fi
    exit 1
    ```
11. Write `doctor-cloud-checks-fail.txtar` — verify `qsdev doctor` cloud checks fail gracefully with mocked failing CLI responses:
    ```
    # qsdev doctor: cloud checks fail gracefully with actionable message
    env PATH=$WORK/mock-bin:$PATH
    exec qsdev init --non-interactive --answers-file answers.yaml
    exec qsdev doctor --check cloud --format json > doctor.json
    json_path doctor.json '.checks[]|select(.name=="aws-auth")' '.status' 'fail'
    json_path doctor.json '.checks[]|select(.name=="aws-auth")' '.fix' 'contains' 'aws sso login'

    -- answers.yaml --
    quick_mode: true

    -- main.tf --
    terraform { required_providers { aws = { source = "hashicorp/aws" } } }

    -- mock-bin/aws --
    #!/bin/sh
    echo "Error: Unable to locate credentials" >&2
    exit 255
    ```

**Acceptance Criteria:**
- [ ] AWS detected from Terraform `hashicorp/aws` provider block; `awscli2` added to devenv.nix
- [ ] AWS detected from `cdk.json` presence; detection reason recorded
- [ ] GCP detected from Terraform `hashicorp/google` provider block; `google-cloud-sdk` added; `CLOUDSDK_CORE_PROJECT` env var generated
- [ ] Azure detected from Terraform `hashicorp/azurerm` provider block; `azure-cli` added; `ARM_SUBSCRIPTION_ID` env var generated
- [ ] Multi-cloud project: all three providers detected from single `.tf` file; all three CLIs added to devenv.nix
- [ ] No secret env var names appear in any generated file (`AWS_SECRET_ACCESS_KEY`, `ARM_CLIENT_SECRET`, `GOOGLE_APPLICATION_CREDENTIALS`, etc.)
- [ ] `wrangler.toml` triggers `wrangler` CLI detection
- [ ] `fly.toml` triggers `flyctl` detection
- [ ] `qsdev doctor --check cloud` reports `pass` when mocked CLI returns valid identity response
- [ ] `qsdev doctor --check cloud` reports `fail` with actionable fix message when mocked CLI returns credential error
- [ ] Doctor checks complete within 5-second timeout when CLI hangs (requires mock-bin script that sleeps >5s)

**Research Citations:**
- `phases/23-cloud-cli-credential-isolation.md § Unit 23.1` — AWS module implementation, `AWS_PROFILE` isolation, doctor checks
- `phases/23-cloud-cli-credential-isolation.md § Unit 23.2` — GCP module, `CLOUDSDK_CORE_PROJECT`
- `phases/23-cloud-cli-credential-isolation.md § Unit 23.3` — Azure module, `ARM_SUBSCRIPTION_ID`
- `phases/23-cloud-cli-credential-isolation.md § Unit 23.5` — `DoctorCheckRegistry`, 5-second timeout, graceful degradation
- `phases/23-cloud-cli-credential-isolation.md § Unit 23.6` — `TerraformProviderDetector`, `required_providers` parsing
- `phases/23-cloud-cli-credential-isolation.md § Unit 23.7` — `CloudEnvVarTemplate`, secret name blocklist, Tier 3 CLIs

**Status:** Not Started

---

### Unit 35.2: Kubernetes Ecosystem Validation

**Description:** Validate the Kubernetes ecosystem module: detection from YAML `apiVersion` and kustomization files, kubectl version pinning to the correct `kubectl_1_XX` Nixpkgs variant, K8s security tools (kubescape, kube-linter, kube-bench), Helm detection from Chart.yaml, cloud-auth coordination (GKE/AKS/EKS plugins), KUBECONFIG isolation to a project-local path, and K8s-specific deny rules.

**Context:** Phase 24 implements the Kubernetes module on top of the Phase 23 cloud infrastructure. The key correctness properties are version pinning (kubectl must be within one minor version of the cluster — using the wrong version causes API skew errors) and KUBECONFIG isolation (pointing kubectl at `~/.kube/config` causes cross-project cluster operations). Cloud-auth coordination is conditional on which cloud CLIs Phase 23 detected: GCP presence adds `gke-gcloud-auth-plugin`, Azure adds `kubelogin`, AWS adds nothing because aws-cli handles EKS auth natively via `aws eks update-kubeconfig`. The deny rules must block `kubectl delete namespace` (irreversible) while allowing `kubectl get pods` and other read operations.

**Desired Outcome:** A test suite verifying K8s module detection accuracy, kubectl version pinning, correct security tool selection based on profile, cloud-auth plugin coordination without duplication, KUBECONFIG path isolation, and K8s deny rule precision (blocks destructive commands, allows read operations).

**Steps:**
1. Create `e2e/testdata/script/kubernetes/` directory for K8s validation scripts.
2. Write `k8s-detection-apiversion.txtar` — verify detection from Kubernetes YAML manifest:
   ```
   # K8s detected from apiVersion: apps/v1 in yaml file
   exec qsdev init --non-interactive --answers-file answers.yaml
   exec gdev detect --format json > detect.json
   json_path detect.json '.modules' 'some' '.name=="kubernetes"'
   exec grep 'kubectl' devenv.nix

   -- answers.yaml --
   quick_mode: true

   -- k8s/deployment.yaml --
   apiVersion: apps/v1
   kind: Deployment
   metadata:
     name: my-app
   ```
3. Write `k8s-detection-kustomization.txtar` — verify kustomize added when kustomization.yaml detected:
   ```
   # kustomization.yaml -> kustomize package added
   exec qsdev init --non-interactive --answers-file answers.yaml
   exec grep 'kustomize' devenv.nix

   -- answers.yaml --
   quick_mode: true

   -- kustomization.yaml --
   apiVersion: kustomize.config.k8s.io/v1beta1
   kind: Kustomization
   resources:
     - deployment.yaml
   ```
4. Write `kubectl-version-pinning.txtar` — verify kubectl version matches cluster version spec:
   ```
   # kubectl version pinning: cluster version 1.29 -> kubectl_1_29 variant
   exec qsdev init --non-interactive --answers-file answers.yaml
   exec grep 'kubectl_1_29' devenv.nix

   -- answers.yaml --
   quick_mode: true
   kubernetes_version: "1.29"

   -- k8s/deployment.yaml --
   apiVersion: apps/v1
   kind: Deployment
   metadata:
     name: my-app
   ```
5. Write `k8s-security-tools-default.txtar` — verify kubescape and kube-linter in default set:
   ```
   # K8s security tools: kubescape + kube-linter in default set
   exec qsdev init --non-interactive --answers-file answers.yaml
   exec grep 'kubescape' devenv.nix
   exec grep 'kube-linter' devenv.nix

   -- answers.yaml --
   quick_mode: true

   -- k8s/deployment.yaml --
   apiVersion: apps/v1
   kind: Deployment
   metadata:
     name: my-app
   ```
6. Write `k8s-bench-cis-profile.txtar` — verify kube-bench only added when CIS profile selected:
   ```
   # kube-bench only when CIS hardening profile selected
   exec qsdev init --non-interactive --answers-file answers-cis.yaml
   exec grep 'kube-bench' devenv.nix

   exec qsdev init --non-interactive --answers-file answers-default.yaml
   ! exec grep 'kube-bench' devenv.nix

   -- answers-cis.yaml --
   quick_mode: true
   security_profile: cis-hardening

   -- answers-default.yaml --
   quick_mode: true
   security_profile: consulting-default

   -- k8s/deployment.yaml --
   apiVersion: apps/v1
   kind: Deployment
   metadata:
     name: my-app
   ```
7. Write `helm-detection.txtar` — verify Helm detected from Chart.yaml:
   ```
   # Chart.yaml -> helm with wrapHelm plugin pattern
   exec qsdev init --non-interactive --answers-file answers.yaml
   exec grep 'helm' devenv.nix
   exec grep 'wrapHelm' devenv.nix

   -- answers.yaml --
   quick_mode: true

   -- charts/myapp/Chart.yaml --
   apiVersion: v2
   name: myapp
   version: 0.1.0
   ```
8. Write `k8s-gcp-auth-plugin.txtar` — verify GCP+K8s adds gke-gcloud-auth-plugin:
   ```
   # GCP + K8s -> gke-gcloud-auth-plugin added
   exec qsdev init --non-interactive --answers-file answers.yaml
   exec grep 'gke-gcloud-auth-plugin' devenv.nix

   -- answers.yaml --
   quick_mode: true

   -- main.tf --
   terraform { required_providers { google = { source = "hashicorp/google" } } }

   -- k8s/deployment.yaml --
   apiVersion: apps/v1
   kind: Deployment
   metadata:
     name: my-app
   ```
9. Write `k8s-azure-auth-plugin.txtar` — verify Azure+K8s adds kubelogin:
   ```
   # Azure + K8s -> kubelogin added
   exec qsdev init --non-interactive --answers-file answers.yaml
   exec grep 'kubelogin' devenv.nix
   # Should not also add gke plugin
   ! exec grep 'gke-gcloud-auth-plugin' devenv.nix

   -- answers.yaml --
   quick_mode: true

   -- main.tf --
   terraform { required_providers { azurerm = { source = "hashicorp/azurerm" } } }

   -- k8s/deployment.yaml --
   apiVersion: apps/v1
   kind: Deployment
   metadata:
     name: my-app
   ```
10. Write `k8s-aws-no-extra-plugin.txtar` — verify AWS+K8s does NOT add extra auth plugin:
    ```
    # AWS + K8s -> no extra auth plugin (aws-cli handles EKS auth natively)
    exec qsdev init --non-interactive --answers-file answers.yaml
    ! exec grep 'gke-gcloud-auth-plugin' devenv.nix
    ! exec grep 'kubelogin' devenv.nix

    -- answers.yaml --
    quick_mode: true

    -- main.tf --
    terraform { required_providers { aws = { source = "hashicorp/aws" } } }

    -- k8s/deployment.yaml --
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: my-app
    ```
11. Write `kubeconfig-isolation.txtar` — verify KUBECONFIG points to project-local path:
    ```
    # KUBECONFIG isolation: generated path is project-local, not ~/.kube/config
    exec qsdev init --non-interactive --answers-file answers.yaml
    exec grep 'KUBECONFIG' devenv.nix
    ! exec grep '\.kube/config' devenv.nix
    exec grep '\.gdev/kubeconfig\|\.kube-gdev' devenv.nix

    -- answers.yaml --
    quick_mode: true

    -- k8s/deployment.yaml --
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: my-app
    ```
12. Write `k8s-deny-rules.txtar` — verify K8s deny rules block destructive, allow read operations:
    ```
    # K8s deny rules: block delete namespace, allow get pods
    exec qsdev init --non-interactive --answers-file answers.yaml
    exec qsdev check --deny-rules --format json > rules.json
    json_path rules.json '.denyRules' 'some' 'contains("kubectl delete namespace")'
    ! json_path rules.json '.denyRules' 'some' 'contains("kubectl get")'

    -- answers.yaml --
    quick_mode: true

    -- k8s/deployment.yaml --
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: my-app
    ```

**Acceptance Criteria:**
- [ ] K8s detected from YAML file with `apiVersion: apps/v1`; `kubectl` added to devenv.nix
- [ ] `kustomization.yaml` presence adds `kustomize` package
- [ ] kubectl version pinning: `kubernetes_version: "1.29"` in answers file generates `kubectl_1_29` Nixpkgs variant
- [ ] `kubescape` and `kube-linter` present in default security tool set
- [ ] `kube-bench` added only when CIS hardening profile selected, absent with default profile
- [ ] `Chart.yaml` triggers helm detection with `wrapHelm` plugin pattern
- [ ] GCP + K8s combination adds `gke-gcloud-auth-plugin`
- [ ] Azure + K8s combination adds `kubelogin`
- [ ] AWS + K8s combination adds neither GKE plugin nor kubelogin
- [ ] `KUBECONFIG` env var in devenv.nix points to project-local path, not `~/.kube/config`
- [ ] `kubectl delete namespace` blocked by deny rules; `kubectl get pods` not blocked

**Research Citations:**
- `phases/24-kubernetes-ecosystem-module.md § Unit 24.1` — K8s detection signals, kubectl version pinning variants
- `phases/24-kubernetes-ecosystem-module.md § Unit 24.2` — K8s security tools, kubescape/kube-linter default, kube-bench CIS gating
- `phases/24-kubernetes-ecosystem-module.md § Unit 24.3` — Helm wrapHelm pattern
- `phases/24-kubernetes-ecosystem-module.md § Unit 24.4` — cloud-auth coordination, GKE/AKS/EKS plugin decision matrix
- `phases/24-kubernetes-ecosystem-module.md § Unit 24.5` — KUBECONFIG isolation, project-local path
- `phases/24-kubernetes-ecosystem-module.md § Unit 24.6` — K8s deny rules, destructive vs read-only operation classification

**Status:** Not Started

---

### Unit 35.3: Service Template Validation

**Description:** Validate all five new service modules from Phase 25: Kafka (KRaft mode, no Zookeeper, `KAFKA_BOOTSTRAP_SERVERS`), MinIO (S3-compatible env vars pointing to localhost), Mailpit (SMTP + web UI ports), Keycloak (dev-file DB mode), and NATS (JetStream toggle). Validate the Tier 1/Tier 2 tiering classification in the wizard path, and verify that mixed-signal projects correctly tier services.

**Context:** Phase 25 expanded the service template catalog from 6 to 11 services and introduced Tier 1 (quick wizard path) / Tier 2 (customize path) classification. Kafka's most common correctness failure is accidental Zookeeper dependency — gdev must generate KRaft-only config since Kafka 4.0 removed Zookeeper support entirely. MinIO's value is S3 API compatibility: the env vars `AWS_S3_ENDPOINT_URL`, `AWS_DEFAULT_REGION=us-east-1`, and `MINIO_ROOT_USER`/`MINIO_ROOT_PASSWORD` must be set correctly for S3 SDKs to connect without code changes. The tiering test verifies that Kafka (Tier 1) appears in the quick wizard path while MinIO (Tier 2) requires the customize path — mixing them up would either overwhelm the quick path or hide critical services.

**Desired Outcome:** A test suite verifying that each new service generates correct devenv.nix config, that no Zookeeper dependency appears in Kafka output, that MinIO env vars enable S3 SDK compatibility, that JetStream appears in NATS output only when code signals are present, and that the wizard tiering routes services correctly.

**Steps:**
1. Create `e2e/testdata/script/services/` directory for service validation scripts.
2. Write `kafka-kraft-mode.txtar` — verify Kafka uses KRaft, no Zookeeper:
   ```
   # Kafka: KRaft mode generated, no Zookeeper dependency
   exec qsdev init --non-interactive --answers-file answers.yaml
   exec grep 'services.kafka.enable' devenv.nix
   ! exec grep -i 'zookeeper' devenv.nix
   exec grep 'KAFKA_BOOTSTRAP_SERVERS' devenv.nix

   -- answers.yaml --
   quick_mode: true

   -- package.json --
   {"name": "test", "dependencies": {"kafkajs": "^2.2.4"}}
   ```
3. Write `kafka-memory-caps.txtar` — verify JVM memory options are set:
   ```
   # Kafka: JVM memory options cap heap at developer-friendly size
   exec qsdev init --non-interactive --answers-file answers.yaml
   exec grep -E 'Xmx|jvmOptions' devenv.nix

   -- answers.yaml --
   quick_mode: true

   -- package.json --
   {"name": "test", "dependencies": {"kafkajs": "^2.2.4"}}
   ```
4. Write `minio-s3-envvars.txtar` — verify MinIO env vars enable S3 SDK compatibility:
   ```
   # MinIO: S3-compatible env vars pointing to localhost
   exec qsdev init --non-interactive --answers-file answers.yaml
   exec grep 'services.minio' devenv.nix
   exec grep 'MINIO_ROOT_USER\|AWS_ACCESS_KEY_ID' devenv.nix
   exec grep 'localhost\|127.0.0.1' devenv.nix

   -- answers.yaml --
   quick_mode: true

   -- package.json --
   {"name": "test", "dependencies": {"@aws-sdk/client-s3": "^3.0.0"}}
   ```
5. Write `mailpit-ports.txtar` — verify Mailpit SMTP and web UI ports:
   ```
   # Mailpit: SMTP port 1025 + web UI port 8025
   exec qsdev init --non-interactive --answers-file answers.yaml
   exec grep 'services.mailpit' devenv.nix
   exec grep '1025\|SMTP_PORT\|SMTP_HOST' devenv.nix
   exec grep '8025\|MAILPIT_UI' devenv.nix

   -- answers.yaml --
   quick_mode: true
   services: [mailpit]
   ```
6. Write `keycloak-dev-file-mode.txtar` — verify Keycloak uses dev-file DB mode:
   ```
   # Keycloak: dev-file DB mode (no external DB dependency required)
   exec qsdev init --non-interactive --answers-file answers.yaml
   exec grep 'services.keycloak' devenv.nix
   ! exec grep 'keycloak.*postgres\|keycloak.*mysql' devenv.nix

   -- answers.yaml --
   quick_mode: true
   services: [keycloak]
   ```
7. Write `nats-jetstream-detected.txtar` — verify JetStream toggle based on code detection:
   ```
   # NATS: JetStream enabled when JetStream-specific API usage detected
   exec qsdev init --non-interactive --answers-file answers-jetstream.yaml
   exec grep 'jetstream\|JetStream' devenv.nix

   exec qsdev init --non-interactive --answers-file answers-basic.yaml
   ! exec grep 'jetstream\|JetStream' devenv.nix

   -- answers-jetstream.yaml --
   quick_mode: true

   -- src/messaging.go --
   package main

   import "github.com/nats-io/nats.go"

   func main() {
     js, _ := nats.Connect("nats://localhost:4222")
     _ = js.JetStream()
   }

   -- answers-basic.yaml --
   quick_mode: true

   -- src/messaging.go --
   package main

   import "github.com/nats-io/nats.go"

   func main() {
     nats.Connect("nats://localhost:4222")
   }
   ```
8. Write `service-tiering-wizard.txtar` — verify Tier 1 in quick path, Tier 2 in customize path:
   ```
   # Service tiering: Kafka (Tier 1) in quick path, MinIO (Tier 2) in customize path
   exec qsdev init --non-interactive --answers-file answers.yaml
   exec gdev detect --format json > detect.json
   json_path detect.json '.services[]|select(.name=="kafka")' '.tier' '1'
   json_path detect.json '.services[]|select(.name=="minio")' '.tier' '2'

   -- answers.yaml --
   quick_mode: true

   -- package.json --
   {"name": "test", "dependencies": {"kafkajs": "^2.2.4", "@aws-sdk/client-s3": "^3.0.0"}}
   ```
9. Write `mixed-service-detection.txtar` — verify mixed signals produce correct combined output:
   ```
   # Mixed signals: kafkajs + @aws-sdk/client-s3 -> Kafka Tier 1 + MinIO Tier 2
   exec qsdev init --non-interactive --answers-file answers.yaml
   exec grep 'services.kafka.enable' devenv.nix
   exec grep 'services.minio' devenv.nix
   ! exec grep -i 'zookeeper' devenv.nix

   -- answers.yaml --
   quick_mode: true

   -- package.json --
   {"name": "test", "dependencies": {"kafkajs": "^2.2.4", "@aws-sdk/client-s3": "^3.0.0"}}
   ```

**Acceptance Criteria:**
- [ ] Kafka module generates `services.kafka.enable = true` with KRaft mode; no Zookeeper reference in output
- [ ] `KAFKA_BOOTSTRAP_SERVERS` env var generated pointing to localhost
- [ ] Kafka JVM memory options set to developer-laptop-friendly values (`-Xmx256m -Xms256m` or equivalent)
- [ ] MinIO module generates S3-compatible env vars (`MINIO_ROOT_USER`/`MINIO_ROOT_PASSWORD`) pointing to localhost
- [ ] Mailpit generates SMTP port 1025 and web UI port 8025 env vars
- [ ] Keycloak generates dev-file DB mode config; no external database dependency required
- [ ] NATS with JetStream API usage in code generates JetStream-enabled config; basic NATS usage does not
- [ ] Kafka classified as Tier 1 (quick wizard path); MinIO classified as Tier 2 (customize path)
- [ ] Mixed project (kafkajs + `@aws-sdk/client-s3`) correctly detects both services with correct tiers

**Research Citations:**
- `phases/25-service-template-expansion.md § Unit 25.1` — Kafka KRaft mode, JVM options, detection signals
- `phases/25-service-template-expansion.md § Unit 25.2` — MinIO, S3-compatible env vars
- `phases/25-service-template-expansion.md § Unit 25.3` — Mailpit ports
- `phases/25-service-template-expansion.md § Unit 25.4` — Keycloak dev-file mode
- `phases/25-service-template-expansion.md § Unit 25.5` — NATS JetStream toggle
- `phases/25-service-template-expansion.md § Unit 25.6` — Tier 1/Tier 2 classification engine, wizard quick/customize path routing

**Status:** Not Started

---

### Unit 35.4: Non-Language Tool Detection Validation

**Description:** Validate detection accuracy and module generation for all four non-language tool categories from Phase 26: git platform tools (gh, glab, git-lfs), documentation generators (mkdocs+material, mdbook, d2, adr-tools), API tools (grpcurl+buf, openapi-generator+redocly, bruno), and database migration tools (prisma, Alembic CLAUDE.md-only). Verify that mixed projects detect all modules without conflicts, and that non-language modules run after language modules in the generation pipeline.

**Context:** Phase 26 extended gdev's detection engine beyond language runtimes to the broader developer toolchain. The primary correctness risk is false positives — detecting a tool when it is not actually used creates noise in devenv.nix. Each detection signal is therefore indicator-specific: `gh` requires `.github/` directory (not just any GitHub URL), `glab` requires `.gitlab-ci.yml` (not just any `.git` remote). Database migration tools present a split pattern: Prisma is installed as a devenv package (it needs a binary in PATH), while Alembic is installed in the Python virtualenv so gdev adds only a CLAUDE.md note. Module ordering matters because language modules must run before non-language modules so that tool mappings like "this project uses Python → include Alembic note" resolve correctly.

**Desired Outcome:** A test suite verifying that each non-language detection signal triggers the correct module, that mixed projects detect all applicable modules without conflicts, and that the generation pipeline respects module ordering.

**Steps:**
1. Create `e2e/testdata/script/nonlanguage/` directory for non-language tool test scripts.
2. Write `git-platform-gh.txtar` — verify GitHub CLI detected from `.github/` directory:
   ```
   # .github/ directory -> gh CLI detected
   exec qsdev init --non-interactive --answers-file answers.yaml
   exec grep 'gh\b' devenv.nix

   -- answers.yaml --
   quick_mode: true

   -- .github/workflows/ci.yaml --
   name: CI
   on: [push]
   jobs:
     test:
       runs-on: ubuntu-latest
   ```
3. Write `git-platform-glab.txtar` — verify GitLab CLI detected from `.gitlab-ci.yml`:
   ```
   # .gitlab-ci.yml -> glab CLI detected
   exec qsdev init --non-interactive --answers-file answers.yaml
   exec grep 'glab' devenv.nix

   -- answers.yaml --
   quick_mode: true

   -- .gitlab-ci.yml --
   stages:
     - test
   test:
     script: go test ./...
   ```
4. Write `git-lfs-detection.txtar` — verify git-lfs detected from `.gitattributes` filter:
   ```
   # .gitattributes with filter=lfs -> git-lfs detected
   exec qsdev init --non-interactive --answers-file answers.yaml
   exec grep 'git-lfs' devenv.nix

   -- answers.yaml --
   quick_mode: true

   -- .gitattributes --
   *.psd filter=lfs diff=lfs merge=lfs -text
   *.png filter=lfs diff=lfs merge=lfs -text
   ```
5. Write `docs-mkdocs.txtar` — verify mkdocs+material detected from mkdocs.yml:
   ```
   # mkdocs.yml -> mkdocs + mkdocs-material detected
   exec qsdev init --non-interactive --answers-file answers.yaml
   exec grep 'mkdocs' devenv.nix

   -- answers.yaml --
   quick_mode: true

   -- mkdocs.yml --
   site_name: My Docs
   theme:
     name: material
   ```
6. Write `docs-mdbook.txtar` — verify mdbook detected from book.toml:
   ```
   # book.toml -> mdbook detected
   exec qsdev init --non-interactive --answers-file answers.yaml
   exec grep 'mdbook' devenv.nix

   -- answers.yaml --
   quick_mode: true

   -- book.toml --
   [book]
   title = "My Book"
   src = "src"
   ```
7. Write `docs-d2.txtar` — verify d2 diagram tool detected from .d2 files:
   ```
   # *.d2 file -> d2 detected
   exec qsdev init --non-interactive --answers-file answers.yaml
   exec grep '\bd2\b' devenv.nix

   -- answers.yaml --
   quick_mode: true

   -- architecture.d2 --
   web -> db: queries
   ```
8. Write `docs-adr-tools.txtar` — verify adr-tools detected from docs/adr/ directory:
   ```
   # docs/adr/ directory -> adr-tools detected
   exec qsdev init --non-interactive --answers-file answers.yaml
   exec grep 'adr-tools' devenv.nix

   -- answers.yaml --
   quick_mode: true

   -- docs/adr/0001-use-postgresql.md --
   # 1. Use PostgreSQL
   Date: 2024-01-01
   Status: Accepted
   ```
9. Write `api-grpc.txtar` — verify grpcurl+buf detected from .proto files:
   ```
   # *.proto file -> grpcurl + buf detected
   exec qsdev init --non-interactive --answers-file answers.yaml
   exec grep 'grpcurl' devenv.nix
   exec grep 'buf\b' devenv.nix

   -- answers.yaml --
   quick_mode: true

   -- proto/user.proto --
   syntax = "proto3";
   package user;
   service UserService {
     rpc GetUser (GetUserRequest) returns (User);
   }
   ```
10. Write `api-openapi.txtar` — verify openapi-generator+redocly detected from openapi.yaml:
    ```
    # openapi.yaml -> openapi-generator + redocly detected
    exec qsdev init --non-interactive --answers-file answers.yaml
    exec grep 'openapi-generator\|redocly' devenv.nix

    -- answers.yaml --
    quick_mode: true

    -- openapi.yaml --
    openapi: "3.0.0"
    info:
      title: My API
      version: "1.0.0"
    paths: {}
    ```
11. Write `api-bruno.txtar` — verify bruno detected from .bru files:
    ```
    # *.bru file -> bruno detected
    exec qsdev init --non-interactive --answers-file answers.yaml
    exec grep 'bruno' devenv.nix

    -- answers.yaml --
    quick_mode: true

    -- requests/get-users.bru --
    meta {
      name: Get Users
      type: http
    }
    get {
      url: {{baseUrl}}/users
    }
    ```
12. Write `db-migration-prisma.txtar` — verify prisma added as devenv package:
    ```
    # prisma/schema.prisma -> prisma added as devenv package
    exec qsdev init --non-interactive --answers-file answers.yaml
    exec grep 'prisma' devenv.nix

    -- answers.yaml --
    quick_mode: true

    -- prisma/schema.prisma --
    datasource db {
      provider = "postgresql"
      url      = env("DATABASE_URL")
    }
    ```
13. Write `db-migration-alembic-claudemd.txtar` — verify Alembic adds CLAUDE.md note, not devenv package:
    ```
    # alembic/ directory -> CLAUDE.md note only, not a devenv.nix package
    exec qsdev init --non-interactive --answers-file answers.yaml
    exec grep -i 'alembic' CLAUDE.md
    ! exec grep 'alembic' devenv.nix

    -- answers.yaml --
    quick_mode: true

    -- alembic/env.py --
    from alembic import context
    ```
14. Write `mixed-nonlanguage-detection.txtar` — verify mixed project detects all four categories without conflict:
    ```
    # Mixed: proto + openapi + mkdocs + .github/ -> all 4 modules detected, no conflicts
    exec qsdev init --non-interactive --answers-file answers.yaml
    exec gdev detect --format json > detect.json
    json_path detect.json '.modules' 'some' '.name=="gh"'
    json_path detect.json '.modules' 'some' '.name=="grpc"'
    json_path detect.json '.modules' 'some' '.name=="openapi"'
    json_path detect.json '.modules' 'some' '.name=="mkdocs"'
    exec qsdev check --deny-rules --format json > rules.json
    json_path rules.json '.unexpectedConflicts' '0'

    -- answers.yaml --
    quick_mode: true

    -- proto/service.proto --
    syntax = "proto3";
    package svc;

    -- openapi.yaml --
    openapi: "3.0.0"
    info:
      title: API
      version: "1.0"
    paths: {}

    -- mkdocs.yml --
    site_name: Docs

    -- .github/workflows/ci.yaml --
    name: CI
    on: [push]
    jobs:
      test:
        runs-on: ubuntu-latest
    ```
15. Write `module-ordering.txtar` — verify non-language modules run after language modules:
    ```
    # Module ordering: language modules before non-language modules in generation pipeline
    exec qsdev init --non-interactive --answers-file answers.yaml
    exec gdev detect --format json > detect.json
    # Non-language modules appear after language modules in ordered list
    exec sh -c 'node -e "const d=require('"'"'./detect.json'"'"'); const lang=d.modules.findIndex(m=>m.type=='"'"'language'"'"'); const nonlang=d.modules.findIndex(m=>m.type=='"'"'nonlanguage'"'"'); process.exit(lang<nonlang?0:1)"'

    -- answers.yaml --
    quick_mode: true

    -- go.mod --
    module example.com/test
    go 1.22

    -- proto/service.proto --
    syntax = "proto3";
    package svc;
    ```

**Acceptance Criteria:**
- [ ] `.github/` directory triggers `gh` CLI detection
- [ ] `.gitlab-ci.yml` triggers `glab` CLI detection
- [ ] `.gitattributes` with `filter=lfs` triggers `git-lfs` detection
- [ ] `mkdocs.yml` triggers `mkdocs` + `mkdocs-material` detection
- [ ] `book.toml` triggers `mdbook` detection
- [ ] `*.d2` file triggers `d2` detection
- [ ] `docs/adr/` directory triggers `adr-tools` detection
- [ ] `*.proto` file triggers `grpcurl` + `buf` detection
- [ ] `openapi.yaml` triggers `openapi-generator` + `redocly` detection
- [ ] `*.bru` file triggers `bruno` detection
- [ ] `prisma/schema.prisma` adds prisma as a devenv package
- [ ] `alembic/` directory adds CLAUDE.md note only; no `alembic` package in devenv.nix
- [ ] Mixed project (proto + openapi + mkdocs + .github/) detects all four categories with zero deny rule conflicts
- [ ] Non-language modules run after language modules in the generation pipeline (ordering verified)

**Research Citations:**
- `phases/26-non-language-tool-detection.md § Unit 26.1` — git platform tool detection signals (gh, glab, git-lfs)
- `phases/26-non-language-tool-detection.md § Unit 26.2` — documentation tool detection (mkdocs, mdbook, d2, adr-tools)
- `phases/26-non-language-tool-detection.md § Unit 26.3` — API tool detection (grpcurl+buf, openapi-generator+redocly, bruno)
- `phases/26-non-language-tool-detection.md § Unit 26.4` — database migration tool detection (prisma as devenv package, Alembic as CLAUDE.md note)
- `phases/26-non-language-tool-detection.md § Unit 26.5` — module ordering, language-before-nonlanguage pipeline invariant

**Status:** Not Started

---

### Unit 35.5: IDE & Shell Configuration Validation

**Description:** Validate all Phase 27 IDE and shell configuration features: EditorConfig generation with correct ecosystem-aware indent rules, VS Code extensions.json creation/removal via `qsdev enable/disable vscode`, extension ecosystem mappings, shell fragment generation (does not modify rc files), personal tool installation via nix profile (mocked), Starship config generation (does not overwrite existing), and conditional shell alias generation based on installed tools.

**Context:** Phase 27 sits at the boundary between project-scope and user-scope configuration. The central invariant is that gdev never modifies shell rc files — it generates fragments and prints instructions. Violating this invariant would corrupt user shell configuration silently. EditorConfig rules are project-scope and tracked by the Phase 8 hash system. VS Code `extensions.json` is opt-in (not generated by default) to avoid forcing tooling decisions on developers who use other editors. The Starship integration must not overwrite an existing `starship.toml` — it generates a gdev-specific include file. Shell aliases are conditional on whether the target tool is actually installed, since generating an alias for `bat` when `bat` is not installed creates confusion.

**Desired Outcome:** A test suite verifying ecosystem-aware EditorConfig correctness, VS Code extension lifecycle, shell fragment safety (no rc file modification), Starship non-destructive integration, and conditional alias generation.

**Steps:**
1. Create `e2e/testdata/script/ide-shell/` directory for IDE and shell configuration test scripts.
2. Write `editorconfig-go.txtar` — verify Go project gets tab indent rules:
   ```
   # Go project: EditorConfig uses tabs (consistent with gofmt)
   exec qsdev init --non-interactive --answers-file answers.yaml
   exists .editorconfig
   exec grep 'indent_style = tab' .editorconfig
   exec grep '\[.*\.go\]' .editorconfig

   -- answers.yaml --
   quick_mode: true

   -- go.mod --
   module example.com/test
   go 1.22
   ```
3. Write `editorconfig-python.txtar` — verify Python project gets 4-space indent:
   ```
   # Python project: EditorConfig uses 4-space indent (PEP 8)
   exec qsdev init --non-interactive --answers-file answers.yaml
   exists .editorconfig
   exec grep 'indent_style = space' .editorconfig
   exec grep 'indent_size = 4' .editorconfig

   -- answers.yaml --
   quick_mode: true

   -- pyproject.toml --
   [project]
   name = "test"
   version = "0.1.0"
   ```
4. Write `editorconfig-js.txtar` — verify JavaScript/TypeScript project gets 2-space indent:
   ```
   # JS/TS project: EditorConfig uses 2-space indent (community convention)
   exec qsdev init --non-interactive --answers-file answers.yaml
   exists .editorconfig
   exec grep 'indent_style = space' .editorconfig
   exec grep 'indent_size = 2' .editorconfig

   -- answers.yaml --
   quick_mode: true

   -- package.json --
   {"name": "test", "version": "1.0.0"}
   ```
5. Write `editorconfig-multi-ecosystem.txtar` — verify multi-ecosystem project gets per-section rules:
   ```
   # Multi-ecosystem: correct per-section rules for each file type
   exec qsdev init --non-interactive --answers-file answers.yaml
   exists .editorconfig
   exec grep '\[.*\.go\]' .editorconfig
   exec grep 'indent_style = tab' .editorconfig
   exec grep '\[.*\.ts\]' .editorconfig
   exec grep 'indent_size = 2' .editorconfig

   -- answers.yaml --
   quick_mode: true

   -- go.mod --
   module example.com/test
   go 1.22

   -- package.json --
   {"name": "test"}
   ```
6. Write `vscode-enable-disable.txtar` — verify VS Code extensions.json lifecycle:
   ```
   # qsdev enable vscode: file created; qsdev disable vscode: file removed cleanly
   exec qsdev init --non-interactive --answers-file answers.yaml
   ! exists .vscode/extensions.json

   exec qsdev enable vscode
   exists .vscode/extensions.json

   exec qsdev disable vscode
   ! exists .vscode/extensions.json

   -- answers.yaml --
   quick_mode: true

   -- go.mod --
   module example.com/test
   go 1.22
   ```
7. Write `vscode-go-extension.txtar` — verify Go extension mapping:
   ```
   # Go detected -> golang.go in extensions.json
   exec qsdev init --non-interactive --answers-file answers.yaml
   exec qsdev enable vscode
   exec grep 'golang.go' .vscode/extensions.json

   -- answers.yaml --
   quick_mode: true

   -- go.mod --
   module example.com/test
   go 1.22
   ```
8. Write `vscode-claudecode-extension.txtar` — verify Claude Code extension added when claudecode active:
   ```
   # claudecode active -> anthropics.claude-code in extensions.json
   exec qsdev init --non-interactive --answers-file answers.yaml
   exec qsdev enable vscode
   exec grep 'anthropics.claude-code' .vscode/extensions.json

   -- answers.yaml --
   quick_mode: true
   claude_code: true

   -- go.mod --
   module example.com/test
   go 1.22
   ```
9. Write `shell-fragments-no-rc-modification.txtar` — verify shell fragments generated without touching rc files:
   ```
   # gdev setup --shell: generates fragments, does NOT modify rc files
   env HOME=$WORK/home
   mkdir $WORK/home
   cp fake-zshrc $WORK/home/.zshrc
   exec cp $WORK/home/.zshrc $WORK/home/.zshrc.before

   exec gdev setup --shell --shell-type zsh --yes
   exists $WORK/home/.qsdev/shell/gdev.zsh
   cmp $WORK/home/.zshrc $WORK/home/.zshrc.before

   -- fake-zshrc --
   # User's existing zshrc
   export PATH="$HOME/.local/bin:$PATH"
   ```
10. Write `starship-no-overwrite.txtar` — verify existing starship.toml is not overwritten:
    ```
    # Existing starship.toml NOT overwritten; gdev generates separate include file
    exec qsdev init --non-interactive --answers-file answers.yaml
    exec gdev setup --shell --shell-type zsh --yes
    cmp starship.toml starship.toml.before
    exists .starship-gdev.toml
    exec grep 'custom.gdev' .starship-gdev.toml

    -- answers.yaml --
    quick_mode: true

    -- starship.toml --
    [username]
    show_always = true

    -- starship.toml.before --
    [username]
    show_always = true
    ```
11. Write `starship-generated.txtar` — verify Starship module generated when no existing config:
    ```
    # No existing starship.toml -> gdev-specific module generated
    exec qsdev init --non-interactive --answers-file answers.yaml
    exec gdev setup --shell --shell-type zsh --yes
    exists .starship-gdev.toml
    exec grep 'custom.gdev' .starship-gdev.toml
    exec grep 'QSDEV_PROJECT_NAME\|gdev' .starship-gdev.toml

    -- answers.yaml --
    quick_mode: true
    ```
12. Write `shell-aliases-conditional.txtar` — verify aliases only generated for installed tools:
    ```
    # Shell aliases: only generated for installed tools
    env PATH=$WORK/mock-bin:$PATH
    exec qsdev init --non-interactive --answers-file answers.yaml
    exec gdev setup --shell --shell-type zsh --yes
    # bat is in PATH (mock installed)
    exec grep 'cat.*bat\|alias cat' $WORK/home/.qsdev/shell/gdev.zsh

    # Remove bat from mock PATH and regenerate
    rm $WORK/mock-bin/bat
    exec gdev setup --shell --shell-type zsh --yes --force
    ! exec grep 'cat.*bat' $WORK/home/.qsdev/shell/gdev.zsh

    -- answers.yaml --
    quick_mode: true

    -- mock-bin/bat --
    #!/bin/sh
    echo "bat version"
    ```

**Acceptance Criteria:**
- [ ] Go project generates `.editorconfig` with `indent_style = tab` for `*.go` files
- [ ] Python project generates `.editorconfig` with `indent_style = space` and `indent_size = 4`
- [ ] JavaScript/TypeScript project generates `.editorconfig` with `indent_size = 2`
- [ ] Multi-ecosystem project generates per-section rules correct for each file type
- [ ] `qsdev enable vscode` creates `.vscode/extensions.json`; `qsdev disable vscode` removes it cleanly
- [ ] Go detected → `golang.go` extension in `extensions.json`
- [ ] Claude Code active → `anthropics.claude-code` extension in `extensions.json`
- [ ] `qsdev setup --shell` generates shell fragment files; existing rc files are NOT modified
- [ ] Existing `starship.toml` is not overwritten; gdev generates separate include file with `[custom.gdev]` module
- [ ] `bat` alias for `cat` only generated when `bat` is present in PATH; absent when `bat` is not installed

**Research Citations:**
- `phases/27-ide-shell-workstation-configuration.md § Unit 27.1` — EditorConfig generation, ecosystem-to-rule mapping, hash tracking
- `phases/27-ide-shell-workstation-configuration.md § Unit 27.2` — VS Code extensions.json, `qsdev enable/disable vscode`, ecosystem extension mapping
- `phases/27-ide-shell-workstation-configuration.md § Unit 27.3` — shell fragment system, rc file invariant, fragment directory
- `phases/27-ide-shell-workstation-configuration.md § Unit 27.4` — personal CLI tools, `nix profile install`, conditional alias generation
- `phases/27-ide-shell-workstation-configuration.md § Unit 27.5` — Starship integration, non-destructive `[custom.gdev]` module

**Status:** Not Started

---

## Phase Completion Criteria

- [ ] All five units pass acceptance criteria
- [ ] Cloud CLI: AWS/GCP/Azure each detected from Terraform provider blocks; no secret env var names in any generated file
- [ ] Cloud CLI: Multi-cloud project detects all three providers; Tier 3 CLIs detected from native config files
- [ ] Cloud CLI: `qsdev doctor` cloud checks produce structured pass/fail output against mocked responses
- [ ] K8s: kubectl version pinning generates correct `kubectl_1_XX` Nixpkgs variant
- [ ] K8s: Cloud-auth plugins correctly coordinated (GKE/AKS add plugins, EKS does not)
- [ ] K8s: KUBECONFIG points to project-local path, not `~/.kube/config`
- [ ] Services: Kafka generates KRaft config with no Zookeeper reference
- [ ] Services: MinIO generates S3-compatible env vars; NATS JetStream toggle based on code signals
- [ ] Services: Tier 1/Tier 2 classification routes services to correct wizard path
- [ ] Non-language: all 12 detection signals trigger correct modules; Alembic adds CLAUDE.md note only (not devenv package)
- [ ] Non-language: mixed project (proto + openapi + mkdocs + .github/) detects all modules with zero deny rule conflicts
- [ ] IDE/shell: EditorConfig rules correct for Go (tabs), Python (4-space), JS/TS (2-space)
- [ ] IDE/shell: `qsdev setup --shell` never modifies existing rc files
- [ ] IDE/shell: Starship integration never overwrites existing `starship.toml`
- [ ] All tests run successfully in the Phase 17 CI pipeline (quick-validation and nightly matrix)
