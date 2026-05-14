# Phase 24: Kubernetes Ecosystem Modules

## Goal

Add Kubernetes tooling in three tiers: core operational tools (kubectl, kubectx, k9s, stern, kustomize), K8s development workflow tools (Skaffold, Tilt, DevSpace, Telepresence), and security scanning (kubescape, kube-linter). Cloud-auth plugins (gke-gcloud-auth-plugin, kubelogin) are coordinated with Phase 23's cloud modules — they activate only when both the relevant cloud provider and K8s are co-detected. Per-project `KUBECONFIG` isolation (never `~/.kube/config`) is the central safety invariant of this phase.

## Dependencies

Phase 1 complete (EcosystemModule interface, detection engine, template engine, generation pipeline). Phase 2 units 2.7 (Docker) and 2.8 (Terraform) complete. Phase 23 complete — cloud co-detection for gke-gcloud-auth-plugin and kubelogin depends on the GCP and Azure module detection results from Phase 23.

## Phase Outputs

- `EcosystemModule` for K8s Core Tools: kubectl, kubectx, k9s, stern, kustomize (Unit 24.1)
- `EcosystemModule` for K8s Development Tools: Skaffold, Tilt, DevSpace, opt-in Telepresence (Unit 24.2)
- `EcosystemModule` for K8s Security Tools: kubescape, kube-linter, optional kube-bench and polaris (Unit 24.3)
- `EcosystemModule` for Helm Ecosystem: helm via `wrapHelm` pattern, helm-secrets, helm-diff, helmfile (Unit 24.4)
- Cloud-auth plugin coordinator that wires Phase 23 cloud detections to K8s auth package selection (Unit 24.5)
- `KubeconfigTemplate` helper enforcing per-project KUBECONFIG isolation (Unit 24.6)
- Generated CLAUDE.md K8s section with context-aware commands, deny rules for destructive operations (Unit 24.7)

---

### Unit 24.1: K8s Core Tools Module (Tier 1)

**Description:** Implement the `EcosystemModule` for core Kubernetes tooling. Detect K8s usage from directory structure, manifest files, and tool config files. Generate a devenv.nix fragment with kubectl (version-pinnable), kubectx/kubens, k9s, stern, and conditional kustomize. Enforce per-project `KUBECONFIG` isolation.

**Context:** kubectl is nearly universal for K8s-using projects. The critical challenge is kubeconfig management: the default `~/.kube/config` merges all cluster credentials, so consulting engineers working across multiple clients risk operating on the wrong cluster. gdev's solution is per-project `KUBECONFIG` in devenv.nix pointing to a client-specific file path. kubectl has a version skew policy — the client must be within +/-1 minor version of the cluster API server — so devenv.nix pins a specific version. k9s and stern are included at Tier 1 because their consulting daily-use value is very high: k9s replaces kubectl for interactive exploration, and stern provides multi-pod log tailing essential for debugging across replicas.

**Desired Outcome:** When K8s indicator files are detected, `gdev init` generates a devenv.nix fragment with kubectl (version-pinnable per wizard input), kubectx, k9s, stern, kustomize (when `kustomization.yaml` present), and per-project `KUBECONFIG` that does not point to `~/.kube/config`.

**Steps:**
1. Implement `Detect(projectRoot string) (DetectionResult, error)`:
   - Check for `k8s/` or `kubernetes/` directories
   - Check for `kustomization.yaml` or `kustomization.yml`
   - Check for `Chart.yaml` (Helm — implies K8s)
   - Check for `helmfile.yaml`
   - Check for `skaffold.yaml`
   - Check for `Tiltfile`
   - Check for `devspace.yaml`
   - Check for `*.yaml` files containing `apiVersion:` with K8s API group patterns (e.g., `apps/v1`, `v1`, `networking.k8s.io/v1`, `batch/v1`) — scan top-level and one level deep only to keep detection fast
   - Return `Detected: true` on first match; `DetectionResult.SubFeatures` includes `"kustomize"` when `kustomization.yaml` found
2. Implement `DevenvNixFragment() string`:
   - Add kubectl with version pin comment: `pkgs.kubectl  # pin to match cluster version: pkgs.kubectl_1_28 for K8s 1.28`
   - Add `pkgs.kubectx` (includes `kubens`)
   - Add `pkgs.k9s`
   - Add `pkgs.stern`
   - Add `pkgs.kustomize` only when `kustomization.yaml` detected
   - Set `env.KUBECONFIG` using `KubeconfigTemplate` (Unit 24.6): `env.KUBECONFIG = "$HOME/.kube/TODO-client-name-config";`
   - Add shell hook: `echo "K8s context: $(kubectl config current-context 2>/dev/null || echo 'none configured')"`
3. Implement `SecurityConfigs() []GeneratedFile`:
   - Generate `.env.example` snippet: `KUBECONFIG=$HOME/.kube/TODO-client-name-config`
   - Include comment: `# NEVER use ~/.kube/config directly — use per-client kubeconfig files`
   - Include naming convention note: `# Convention: $HOME/.kube/<client>-<env>-<region>-config`
4. Implement `DoctorChecks() []DoctorCheck` via the `DoctorCheckRegistry` from Unit 23.5:
   - `kubectl version --client` — verifies kubectl is installed and reports client version (instant, no network)
   - `kubectl cluster-info` — verifies cluster connectivity (5s timeout; likely to fail if VPN disconnected)
   - Check that `KUBECONFIG` is set and non-empty
   - Check that `KUBECONFIG` does NOT equal `~/.kube/config` — this is a warning-level finding, not a failure
5. Implement wizard integration:
   - Wizard asks: "kubectl version to pin (match your cluster version, e.g., 1.28)?" with text input, default: latest stable
   - When user provides version X.Y: devenv.nix uses `pkgs.kubectl_1_XX` (Nixpkgs provides version-specific kubectl packages)
   - Wizard asks: "Kubeconfig path for this project?" with text input, default `"$HOME/.kube/TODO-client-name-config"`
6. Implement `PreCommitHooks() []PreCommitHook`: return nil (YAML linting in language modules; K8s manifest security in Unit 24.3)
7. Implement `DenyRules() []DenyRule`: return nil (populated by Unit 24.7)
8. Write unit tests:
   - Detection from `k8s/` directory
   - Detection from `kustomization.yaml`
   - Detection from `Chart.yaml`
   - Detection from `skaffold.yaml`
   - Detection from YAML file with `apiVersion: apps/v1`
   - No detection from YAML files without K8s API patterns (e.g., a GitHub Actions workflow YAML)
   - kustomize included in fragment when `kustomization.yaml` detected, absent otherwise
   - `KUBECONFIG` set to non-default path in fragment
   - `KUBECONFIG` value does not equal `~/.kube/config`

**Acceptance Criteria:**
- [ ] Detects K8s from directory structure (`k8s/`, `kubernetes/`), kustomization files, `Chart.yaml`, `helmfile.yaml`, `skaffold.yaml`, `Tiltfile`, `devspace.yaml`, and YAML files with K8s `apiVersion:` patterns
- [ ] devenv.nix fragment always includes kubectl, kubectx, k9s, and stern
- [ ] kustomize included conditionally only when `kustomization.yaml` detected
- [ ] Per-project `KUBECONFIG` set with a TODO placeholder path — never `~/.kube/config`
- [ ] kubectl version pin comment present (and version-specific package used when wizard provides version)
- [ ] `gdev doctor` warns when `KUBECONFIG` equals `~/.kube/config`
- [ ] `gdev doctor` cluster connectivity check has 5-second timeout

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md` § 2.1 kubectl — version skew policy, kubeconfig management, auth plugins, consulting best practices
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md` § 2.2 Context & Namespace Switching — kubectx/kubens consulting value
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md` § 2.4 Kustomize — standalone vs kubectl-integrated
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md` § 2.6 K8s Observability Tools — k9s and stern daily-use value
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md` § 4.1 Multi-Client Credential Isolation — K8s isolation pattern with per-client kubeconfig
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-module-design.md` Unit 2.14 — full step-by-step design for this module

**Status:** Not Started

---

### Unit 24.2: K8s Development Tools Module (Tier 2)

**Description:** Implement the `EcosystemModule` for K8s development workflow tools. Detect Skaffold, Tilt, and DevSpace from their config files. Install only the detected tool(s). Telepresence is opt-in only — no config file detection.

**Context:** K8s development tools automate the build-push-deploy inner loop for local K8s development. They are project-specific and opinionated: a project uses Skaffold OR Tilt OR DevSpace, rarely more than one. Detection is straightforward (each has a unique config file) but these tools must never be default-installed since they impose workflow opinions. Telepresence is different: it intercepts traffic from a remote cluster to a local machine for debugging and requires a cluster agent to be installed — it must never be auto-detected. Garden is excluded because it is not in Nixpkgs. This is Tier 2: install when the specific config file is detected.

**Desired Outcome:** When a K8s development tool config file is present, `gdev init` includes exactly that tool's Nixpkgs package in the devenv.nix fragment, with no cross-installation of competing tools. Telepresence is available only when explicitly requested in the wizard or `.gdev.yaml`.

**Steps:**
1. Implement `Detect(projectRoot string) (DetectionResult, error)`:
   - **Skaffold**: check for `skaffold.yaml`
   - **Tilt**: check for `Tiltfile`
   - **DevSpace**: check for `devspace.yaml`
   - **Telepresence**: never auto-detected from config files — Telepresence requires a cluster agent and has no meaningful local config indicator
   - If multiple dev tool config files are present (unusual but possible), detect all of them independently
   - `DetectionResult.SubFeatures` lists which tools triggered: `["skaffold", "tilt"]`
2. Implement `DevenvNixFragment() string` — per-detected-tool:
   - Skaffold: `pkgs.skaffold`
   - Tilt: `pkgs.tilt`
   - DevSpace: `pkgs.devspace`
   - Telepresence (opt-in): `pkgs.telepresence2`
   - When Skaffold detected and `Chart.yaml` also present: add comment `# Skaffold can deploy via Helm — see skaffold.yaml helm deployer config`
3. Implement wizard integration for Telepresence:
   - Wizard offers Telepresence as an optional addition when K8s is detected: "Add Telepresence for remote cluster debugging? (requires cluster agent installation)"
   - Default: no
   - When selected: add `pkgs.telepresence2` to devenv.nix and add CLAUDE.md note about cluster agent requirement
4. Implement `DoctorChecks() []DoctorCheck` — per-detected-tool, each a binary version check (no network needed):
   - Skaffold: `skaffold version`
   - Tilt: `tilt version`
   - DevSpace: `devspace --version`
   - Telepresence: `telepresence version`
5. Implement `SecurityConfigs() []GeneratedFile`: return nil
6. Implement `PreCommitHooks() []PreCommitHook`: return nil
7. Implement `DenyRules() []DenyRule`: return nil
8. Write unit tests:
   - `skaffold.yaml` present → Skaffold detected, Tilt and DevSpace not included
   - `Tiltfile` present → Tilt detected only
   - Both `skaffold.yaml` and `Tiltfile` present → both detected and both packages included
   - No dev tool config file → module not triggered
   - Telepresence absent from fragment when not opt-in; present when wizard selects it
   - `skaffold.yaml` + `Chart.yaml` → Helm integration comment added

**Acceptance Criteria:**
- [ ] Skaffold, Tilt, and DevSpace detected independently from their respective config files
- [ ] Telepresence is opt-in only (never auto-detected from config files)
- [ ] Only detected tools are included in devenv.nix — no cross-installation
- [ ] Multiple dev tools can coexist if multiple config files present
- [ ] Each detected tool has a version doctor check (no network required)
- [ ] Skaffold+Helm co-detection produces a helpful integration comment in devenv.nix

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md` § 2.5 K8s Development Tools — tool comparison (Skaffold vs Tilt vs DevSpace vs Telepresence), detection files, Nixpkgs availability
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md` § 5 Recommended Tiering — Tier 2 classification for Skaffold, Tilt, DevSpace, Telepresence
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-module-design.md` Unit 2.15 — full step-by-step design for this module

**Status:** Not Started

---

### Unit 24.3: K8s Security Tools Module (Tier 3)

**Description:** Provide optional K8s security scanning tools activated when explicitly enabled or when a security-focused profile is active. Default set: kubescape + kube-linter. Extended set (CIS compliance): kube-bench + polaris. Includes pre-commit hook for manifest linting.

**Context:** K8s security tools are valuable but niche — most projects do not run them locally. kubescape is the most comprehensive single tool (260+ controls spanning NSA-CISA, MITRE ATT&CK, and CIS frameworks; CNCF Incubating status). kube-linter is fast static analysis designed for pre-commit/CI gates on K8s YAML and Helm charts. kube-bench specifically targets CIS Kubernetes Benchmark compliance by inspecting cluster component configuration (requires cluster access). polaris validates resource configs against best practices. The recommended default is kubescape (breadth) + kube-linter (speed for CI gates). This module does NOT auto-detect — it activates via explicit enablement, a security profile, or the presence of kubescape/kube-linter config files.

**Desired Outcome:** When K8s security scanning is enabled, `gdev init` includes kubescape and kube-linter in devenv.nix with a pre-commit hook that lints K8s YAML on change. CI scanning commands are documented in CLAUDE.md. kube-bench and polaris are available as an extended set for CIS compliance scenarios.

**Steps:**
1. Implement `Detect(projectRoot string) (DetectionResult, error)`:
   - This module does NOT auto-detect from standard K8s files — it activates only via:
     - Explicit wizard selection during `gdev init`
     - `.gdev.yaml` setting: `kubernetes.security_scanning = true`
     - Profile activation: consulting security profile or compliance level `"strict"` (Phase 13, Unit 13.7)
     - Exception: auto-activate when `.kubescape/` directory or `.kube-linter.yaml` file already exists
   - Return `Detected: false` unless one of the above conditions is met
2. Implement `DevenvNixFragment() string`:
   - Default set: `pkgs.kubescape`, `pkgs.kube-linter`
   - Extended set (when CIS compliance enabled in `.gdev.yaml` or wizard selects it): add `pkgs.kube-bench`, `pkgs.polaris`
   - Add comment: `# kubescape and kube-bench require cluster access for runtime scanning (kubescape scan, kube-bench run)`
3. Implement `SecurityConfigs() []GeneratedFile`:
   - Generate `.kube-linter.yaml`:
     ```yaml
     # Generated by gdev — K8s manifest linting configuration
     checks:
       default: true  # enable all default kube-linter checks
       add:
         - no-read-only-root-fs
         - require-non-root-group
       ignore: []
     ```
   - Generate `kubescape-config.yaml` documenting the default scanning framework (NSA-CISA recommended)
4. Implement `PreCommitHooks() []PreCommitHook`:
   - `kube-linter lint` pre-commit hook on `*.yaml` files
   - File pattern: `^(k8s|kubernetes|charts|manifests)/.*\.yaml$` — restrict to K8s paths, not all YAML
   - Hook only active when kube-linter package is in devenv.nix
5. Implement `CICommands() []CICommand` (helper for CLAUDE.md generation, Unit 24.7):
   - Static analysis (no cluster): `kube-linter lint k8s/`
   - Comprehensive scan (no cluster, against manifest files): `kubescape scan framework nsa --local k8s/`
   - Full cluster scan (requires running cluster): `kubescape scan framework nsa --exclude-namespaces kube-system`
   - CIS compliance (cluster required): `kube-bench run` — note cluster access requirement in output
6. Implement `DoctorChecks() []DoctorCheck`:
   - `kubescape version` — binary available
   - `kube-linter version` — binary available
   - No cluster connectivity check (that is in Unit 24.1)
7. Write unit tests:
   - Module does NOT activate from K8s YAML files alone
   - Module DOES activate when `.kube-linter.yaml` already exists
   - Module DOES activate when consulting-strict profile is active
   - Default set: only kubescape and kube-linter in fragment
   - Extended set: all four tools when CIS compliance selected
   - Pre-commit hook uses path pattern restricting to K8s manifest directories
   - Pre-commit hook absent when module not activated

**Acceptance Criteria:**
- [ ] Module does NOT auto-detect from standard K8s files — requires explicit enablement
- [ ] Exception: auto-activates when `.kubescape/` directory or `.kube-linter.yaml` present
- [ ] Default set is kubescape + kube-linter only (not all four tools)
- [ ] Extended set (kube-bench + polaris) requires explicit CIS compliance opt-in
- [ ] `.kube-linter.yaml` generated with sensible defaults (prevent privileged containers, resource limits)
- [ ] Pre-commit hook restricts to K8s manifest paths only (not all YAML)
- [ ] CI commands documented with correct flags and cluster-access requirements noted
- [ ] kube-bench and polaris absent from fragment when CIS compliance not enabled

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md` § 2.7 K8s Security Tools — tool comparison, CNCF status, 260+ controls, selection guide
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md` § 5 Recommended Tiering — Tier 2/3 classification for security tools
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-module-design.md` Unit 2.16 — full step-by-step design for this module

**Status:** Not Started

---

### Unit 24.4: Helm Ecosystem Module

**Description:** Implement the `EcosystemModule` for Helm, using the `wrapHelm` Nix pattern for plugin management. Include helm-secrets and helm-diff as default plugins. Add helmfile when `helmfile.yaml` detected.

**Context:** Helm is a K8s package manager and the most common K8s deployment tool in consulting. The key Nix-specific challenge is Helm plugin management: installing Helm plugins via `helm plugin install` would download binaries at runtime, breaking the reproducibility guarantee. The Nix-idiomatic solution is `pkgs.wrapHelm pkgs.kubernetes-helm { plugins = [ ... ]; }`, which bundles plugins into the Nix store alongside Helm. helm-secrets (secret management via Helm hooks) and helm-diff (preview `helm upgrade` changes) are recommended defaults for consulting projects. helmfile is a separate binary detected by its own config file.

**Desired Outcome:** When Helm indicator files are detected, `gdev init` generates a devenv.nix fragment that installs Helm via `wrapHelm` with helm-secrets and helm-diff as Nix-managed plugins, plus helmfile when `helmfile.yaml` is present. No `helm plugin install` calls appear in any generated file.

**Steps:**
1. Implement `Detect(projectRoot string) (DetectionResult, error)`:
   - Check for `Chart.yaml` (Helm chart in repo root or subdirectory)
   - Check for `charts/` directory
   - Check for `helmfile.yaml` (helmfile deployment tool)
   - Check for `*.helmignore` files
   - K8s co-detection: Helm implies K8s — when Helm is detected, also ensure the K8s Core Module (Unit 24.1) is triggered
   - `DetectionResult.SubFeatures` includes `"helmfile"` when `helmfile.yaml` found
2. Implement `DevenvNixFragment() string`:
   ```nix
   # Helm with Nix-managed plugins (wrapHelm pattern — avoids runtime plugin downloads)
   (pkgs.wrapHelm pkgs.kubernetes-helm {
     plugins = with pkgs.kubernetes-helmPlugins; [
       helm-secrets   # secret management via helm hooks
       helm-diff      # preview helm upgrade changes before applying
     ];
   })
   ```
   - Add `pkgs.helmfile` when helmfile detected
   - Do NOT use bare `pkgs.kubernetes-helm` — always use `wrapHelm` to ensure plugins are reproducible
3. Implement `SecurityConfigs() []GeneratedFile`:
   - When helm-secrets is included: generate a `.helmfile.yaml` snippet showing the vault backend options (`sops`, `vault`, `aws`, `gcpkms`)
   - Include comment: `# Never commit decrypted secrets — use helm-secrets encrypt before committing`
4. Implement `DoctorChecks() []DoctorCheck`:
   - `helm version` — binary available
   - `helm plugin list` — lists installed plugins; verify helm-secrets and helm-diff are present
   - `helmfile --version` — when helmfile detected
5. Implement `PreCommitHooks() []PreCommitHook`:
   - `helm lint` on directories containing `Chart.yaml`
   - Pre-commit hook pattern: run `helm lint charts/` or `helm lint .` depending on chart location
6. Implement `DenyRules() []DenyRule`: return nil (helm is operational; helm-specific deny rules in Unit 24.7)
7. Write unit tests:
   - Detection from `Chart.yaml`
   - Detection from `charts/` directory
   - Detection from `helmfile.yaml`
   - Fragment uses `wrapHelm` pattern, not bare `kubernetes-helm`
   - `helm-secrets` and `helm-diff` in plugins array
   - `helmfile` package present when helmfile detected, absent otherwise
   - No `helm plugin install` command in any generated output

**Acceptance Criteria:**
- [ ] Detects Helm from `Chart.yaml`, `charts/` directory, `helmfile.yaml`, and `*.helmignore` files
- [ ] devenv.nix uses `wrapHelm` pattern with `helm-secrets` and `helm-diff` as Nix-managed plugins
- [ ] No `helm plugin install` command appears in any generated file (reproducibility guarantee)
- [ ] `helmfile` package included when `helmfile.yaml` detected, absent otherwise
- [ ] `helm lint` pre-commit hook generated for chart directories
- [ ] `gdev doctor` verifies helm-secrets and helm-diff are present in `helm plugin list`

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md` § 2.3 Helm v3 — wrapHelm pattern, plugin management, helmfile
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md` § 3.1 How devenv.sh Users Currently Handle Cloud CLIs — Pattern D (Helm with plugins via wrapHelm)
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-module-design.md` Unit 2.17 — Helm expansion within K8s shared infrastructure

**Status:** Not Started

---

### Unit 24.5: Cloud-Auth Plugin Coordination

**Description:** Implement the detection coordination logic that adds cloud-provider K8s authentication plugins when both a cloud provider AND K8s are co-detected. `gke-gcloud-auth-plugin` is added to the GCP devenv.nix fragment; `kubelogin` is added to the Azure devenv.nix fragment. AWS EKS auth is handled by `awscli2` v2 natively.

**Context:** K8s cluster authentication for managed cloud K8s services (GKE, AKS, EKS) requires cloud-provider-specific auth plugins. Without these plugins, `kubectl` cannot authenticate to the cluster even when the correct kubeconfig is present. The coordination must happen at module compose time, not at individual module detection time, because it requires knowing the state of both the cloud module AND the K8s module simultaneously. This coordination runs after both sets of modules have reported their detection results, before the devenv.nix fragment is assembled.

The three cases are distinct:
- **GKE (GCP + K8s)**: `gke-gcloud-auth-plugin` must be bundled with `google-cloud-sdk` via `withExtraComponents` in the GCP module's fragment (not as a separate package) — this is the Nix-correct pattern
- **AKS (Azure + K8s)**: `kubelogin` is a standalone package added to Azure module's fragment
- **EKS (AWS + K8s)**: `awscli2` v2 bundles the EKS auth plugin natively — no extra package needed

**Desired Outcome:** After all module detections run, the coordinator checks for cloud+K8s co-detection and updates the respective cloud module's devenv.nix fragment to include the auth plugin. The final devenv.nix fragment has the correct auth plugin for the detected cloud provider without any manual configuration.

**Steps:**
1. Implement `CloudK8sCoordinator` in `internal/cloud/coordinator.go`:
   ```go
   type CoordinationResult struct {
       // GCPAuthPlugin: true if gke-gcloud-auth-plugin should be added to GCP fragment
       GCPAuthPlugin bool
       // AzureAuthPlugin: true if kubelogin should be added to Azure fragment
       AzureAuthPlugin bool
       // AWSAuthPlugin: always false (awscli2 v2 handles EKS natively)
       AWSAuthPlugin bool
       // Notes: human-readable notes about auth configuration for CLAUDE.md
       Notes []string
   }

   // Coordinate checks whether both cloud and K8s modules are active and returns
   // the set of auth plugins that should be injected.
   func Coordinate(cloudDetections map[string]DetectionResult, k8sDetected bool) CoordinationResult
   ```
2. Implement coordination logic:
   ```go
   func Coordinate(cloudDetections map[string]DetectionResult, k8sDetected bool) CoordinationResult {
       result := CoordinationResult{}
       if !k8sDetected {
           return result // no K8s → no auth plugins needed
       }

       if cloudDetections["gcp"].Detected {
           result.GCPAuthPlugin = true
           result.Notes = append(result.Notes,
               "GKE auth: gke-gcloud-auth-plugin bundled with google-cloud-sdk via withExtraComponents")
       }
       if cloudDetections["azure"].Detected {
           result.AzureAuthPlugin = true
           result.Notes = append(result.Notes,
               "AKS auth: kubelogin required for Azure AD authentication — run: az aks get-credentials")
       }
       if cloudDetections["aws"].Detected {
           result.Notes = append(result.Notes,
               "EKS auth: handled by awscli2 v2 natively — run: aws eks update-kubeconfig --name <cluster>")
       }
       return result
   }
   ```
3. Wire into the GCP module (Unit 23.2):
   - When `CoordinationResult.GCPAuthPlugin == true`: use `google-cloud-sdk.withExtraComponents [ gke-gcloud-auth-plugin ]` in the devenv.nix fragment instead of bare `google-cloud-sdk`
   - The GCP module checks the coordinator result at fragment generation time
4. Wire into the Azure module (Unit 23.3):
   - When `CoordinationResult.AzureAuthPlugin == true`: add `pkgs.kubelogin` to the Azure module's devenv.nix fragment
5. Wire into the K8s Core module (Unit 24.1) shell hook:
   - Add auth setup notes to the shell hook based on `CoordinationResult.Notes`
   - e.g., for AWS+K8s: `# Run: aws eks update-kubeconfig --name <cluster-name> --region $AWS_DEFAULT_REGION`
6. Implement coordination timing:
   - Run coordination AFTER all module `Detect()` calls complete, BEFORE any `DevenvNixFragment()` call
   - Pass `CoordinationResult` into affected modules via a dependency injection pattern or module context struct
7. Write unit tests:
   - GCP detected + K8s detected → `GCPAuthPlugin: true`
   - GCP detected + K8s NOT detected → `GCPAuthPlugin: false`
   - Azure detected + K8s detected → `AzureAuthPlugin: true`
   - Azure detected + K8s NOT detected → `AzureAuthPlugin: false`
   - AWS detected + K8s detected → `AWSAuthPlugin: false` (no extra plugin) + EKS note present
   - All three clouds + K8s → all three coordination results active
   - No clouds + K8s → empty CoordinationResult

**Acceptance Criteria:**
- [ ] `gke-gcloud-auth-plugin` added to GCP devenv.nix fragment when GCP + K8s co-detected
- [ ] `kubelogin` added to Azure devenv.nix fragment when Azure + K8s co-detected
- [ ] AWS + K8s co-detection produces a note about `aws eks update-kubeconfig` — no extra package
- [ ] Coordination runs after all `Detect()` calls and before any `DevenvNixFragment()` calls
- [ ] GCP fragment uses `withExtraComponents` pattern (not a separate `gke-gcloud-auth-plugin` package entry)
- [ ] No auth plugins added when K8s is not detected, even if cloud providers are detected

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md` § 2.1 kubectl — auth plugins section, EKS/GKE/AKS specific requirements
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md` § 3.2 devenv.nix Patterns for gdev — cloud+K8s coordination pattern
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-module-design.md` Unit 2.17 — cloud-provider auth coordinator design

**Status:** Not Started

---

### Unit 24.6: KUBECONFIG Isolation & Doctor Checks

**Description:** Implement `KubeconfigTemplate`, the helper that generates per-project `KUBECONFIG` paths and validates that the shared `~/.kube/config` is never used. Extend `gdev doctor` with K8s-specific checks: KUBECONFIG path validation, current-context match, and cluster connectivity.

**Context:** The `~/.kube/config` file is the default kubeconfig path and merges credentials for every cluster a user has ever accessed. In consulting, an engineer's `~/.kube/config` may contain credentials for a dozen different client clusters. Running `kubectl apply -f manifests/` against the wrong context is a serious operational risk. The gdev solution is per-project `KUBECONFIG` isolation: devenv.nix sets `KUBECONFIG` to a project-specific path (`$HOME/.kube/client-name-config`), so direnv ensures the correct kubeconfig is always active inside the project shell. Doctor checks verify this invariant is maintained.

**Desired Outcome:** All K8s module devenv.nix fragments use `KubeconfigTemplate` for KUBECONFIG generation. `gdev doctor --category k8s` warns on shared kubeconfig usage, validates the expected context name is active, and checks cluster connectivity with a 5-second timeout.

**Steps:**
1. Implement `KubeconfigTemplate` in `internal/k8s/kubeconfig.go`:
   ```go
   // KubeconfigTemplate generates a per-project KUBECONFIG env var for devenv.nix.
   type KubeconfigTemplate struct {
       // ProjectName is used to construct the default path suggestion.
       ProjectName string
       // ExpectedContext is the K8s context name for this project (optional).
       ExpectedContext string
       // ExplicitPath overrides the generated path (from wizard input).
       ExplicitPath string
   }

   // Render returns the devenv.nix `env.KUBECONFIG` line.
   func (t *KubeconfigTemplate) Render() string

   // Validate checks that the path is not the shared default.
   func (t *KubeconfigTemplate) Validate() error
   ```
2. Implement `Validate()`:
   - Return error if path equals `~/.kube/config` or `$HOME/.kube/config` (shared path)
   - Return error if path is empty
   - Return warning (non-error) if path contains TODO placeholder (not yet filled in by engineer)
3. Implement multi-kubeconfig support:
   - Allow colon-separated paths: `env.KUBECONFIG = "$HOME/.kube/client-dev-config:$HOME/.kube/client-prod-config";`
   - Validate each path in the colon-separated list independently
   - Wizard offers this pattern when project has multiple cluster tiers (dev/staging/prod)
4. Implement naming convention helper:
   ```go
   // SuggestKubeconfigPath returns a suggested per-project kubeconfig path.
   // Convention: $HOME/.kube/<project-name>-<env>-config
   func SuggestKubeconfigPath(projectName string) string
   ```
5. Implement K8s doctor checks in `internal/k8s/doctor.go`:
   - `kubeconfig-not-shared`: check `KUBECONFIG` env var does not equal `~/.kube/config` — warning-level, instant
   - `kubeconfig-file-exists`: check `KUBECONFIG` path points to an existing file — instant
   - `kubectl-context`: run `kubectl config current-context` — instant, no network
   - `context-match`: when `.gdev.yaml` has `kubernetes.expected_context` set, compare against current context — instant
   - `cluster-connectivity`: `kubectl cluster-info` with 5s timeout — network-dependent
6. Integrate with `gdev doctor --category k8s` (extend Phase 15 doctor infrastructure):
   - Register all five K8s checks under category `"k8s"`
   - K8s checks only run when K8s module is active for the current project
7. Write unit tests:
   - `Validate()` rejects `~/.kube/config` and `$HOME/.kube/config`
   - `Validate()` accepts project-specific paths
   - `SuggestKubeconfigPath("acme")` returns `"$HOME/.kube/acme-config"` or similar
   - Multi-kubeconfig: colon-separated path validated element-by-element
   - Doctor check `kubeconfig-not-shared` fires warning when `KUBECONFIG` is the shared default
   - Doctor check `context-match` passes when `.gdev.yaml` expected_context matches actual context
   - Doctor check `cluster-connectivity` returns timeout after 5s

**Acceptance Criteria:**
- [ ] `KubeconfigTemplate.Validate()` rejects `~/.kube/config` and empty paths with clear error messages
- [ ] `KubeconfigTemplate.Validate()` accepts project-specific paths
- [ ] Multi-kubeconfig (colon-separated paths) supported and validated per-path
- [ ] `SuggestKubeconfigPath` produces a project-name-derived default path suggestion
- [ ] `gdev doctor --category k8s` includes five checks: not-shared, file-exists, context, context-match, connectivity
- [ ] `kubeconfig-not-shared` is a warning-level finding (not a blocking failure)
- [ ] Cluster connectivity check enforces 5-second timeout

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md` § 4.1 Multi-Client Credential Isolation — K8s isolation pattern, per-client kubeconfig naming convention
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md` § 2.1 kubectl — kubeconfig management, multi-cluster consulting patterns
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-module-design.md` Unit 2.17 — KUBECONFIG template design

**Status:** Not Started

---

### Unit 24.7: K8s Ecosystem CLAUDE.md & Skills

**Description:** Generate a K8s-specific CLAUDE.md section with context-aware commands for all detected K8s tools. Add a `devenv task k8s:status` task definition. Add K8s-specific deny rules blocking destructive cluster operations.

**Context:** The CLAUDE.md generation framework (Phase 4) produces a project-level CLAUDE.md with sections for each active ecosystem module. The K8s section should be practical and context-specific: it should show the current cluster context and namespace, list the available K8s tools, and provide common commands adapted to what is actually installed (kubectl, helm, kustomize, skaffold, etc.). The deny rules are the critical safety addition: `kubectl delete namespace`, `kubectl drain`, and `helm uninstall` in production contexts are irreversible operations that Claude Code must not execute autonomously. These rules complement the existing settings.json deny rule infrastructure from Phase 5.

**Desired Outcome:** The generated CLAUDE.md K8s section includes context-aware commands for detected tools, the active K8s context and expected namespace, and a `devenv task k8s:status` that shows cluster connectivity. Deny rules in settings.json prevent Claude Code from executing destructive K8s operations in production contexts.

**Steps:**
1. Implement K8s CLAUDE.md section generator in the Phase 4 CLAUDE.md generation pipeline:
   - Section activated when K8s Core Module (Unit 24.1) is active
   - Section template:
     ```markdown
     ## Kubernetes

     **Cluster:** TODO-cluster-name (context: TODO-context-name)
     **Namespace:** TODO-namespace
     **KUBECONFIG:** $HOME/.kube/TODO-client-name-config

     ### Common Commands

     ```bash
     # Check cluster status
     kubectl cluster-info

     # View all pods in current namespace
     kubectl get pods -n TODO-namespace

     # Follow logs from a pod
     kubectl logs -f <pod-name> -n TODO-namespace

     # Open k9s TUI
     k9s --namespace TODO-namespace
     ```

     <!-- [HELM] (included when Helm detected) -->
     ### Helm Commands
     ```bash
     helm list -A           # list all releases
     helm diff upgrade ...  # preview changes before applying
     ```
     <!-- [/HELM] -->

     <!-- [SKAFFOLD] (included when Skaffold detected) -->
     ### Skaffold Commands
     ```bash
     skaffold dev    # continuous build+deploy to local cluster
     skaffold run    # one-shot build+deploy
     ```
     <!-- [/SKAFFOLD] -->

     ### Security Scanning
     ```bash
     kube-linter lint k8s/                                    # fast static analysis
     kubescape scan framework nsa --local k8s/               # NSA-CISA framework
     ```
     ```
   - Replace TODO values with `.gdev.yaml` settings when available (`kubernetes.cluster_name`, `kubernetes.default_namespace`, `kubernetes.expected_context`)
2. Implement `devenv task k8s:status` task:
   - Add to the `devenv.tasks` block in devenv.nix:
     ```nix
     devenv.tasks = {
       "k8s:status" = {
         description = "Show K8s cluster connectivity and context";
         exec = ''
           echo "=== K8s Status ==="
           echo "KUBECONFIG: $KUBECONFIG"
           echo "Context: $(kubectl config current-context 2>/dev/null || echo 'none')"
           echo "Namespace: ${TODO-namespace}"
           kubectl cluster-info 2>/dev/null || echo "Cluster unreachable (VPN connected?)"
         '';
       };
     };
     ```
3. Implement K8s deny rules for settings.json (extending Phase 5 deny rule infrastructure):
   - Deny rule format follows the `permissions.deny` array pattern from Phase 5
   - Generate deny rules:
     ```json
     "Bash(kubectl delete namespace*)",
     "Bash(kubectl drain*)",
     "Bash(kubectl cordon*)",
     "Bash(kubectl delete pv*)",
     "Bash(kubectl delete pvc*)",
     "Bash(helm uninstall*)",
     "Bash(helm rollback*)"
     ```
   - Add a comment block in CLAUDE.md explaining the deny rules: "Claude Code will not execute destructive K8s operations (delete namespace, drain, helm uninstall) autonomously. Instruct the engineer to run these manually."
4. Implement production context detection for deny rules:
   - Read `.gdev.yaml` for `kubernetes.production_context_patterns` (list of glob patterns, default: `["*-prod-*", "*-production-*", "*-prd-*"]`)
   - When a deny rule is context-specific (only block in production), generate a wrapper that checks `kubectl config current-context` against the patterns
   - For simplicity in Phase 23 scope: apply deny rules unconditionally (always deny destructive ops); context-aware rules are a future enhancement
5. Implement per-tool documentation sections in CLAUDE.md:
   - When kubescape active: add `kubescape scan framework nsa` command with correct `--local` flag for offline use
   - When Telepresence active: add note about cluster agent requirement and `telepresence connect` command
   - When helmfile active: add `helmfile sync` and `helmfile diff` commands
6. Write unit tests:
   - CLAUDE.md section generated when K8s Core Module active
   - Helm subsection present when Helm detected, absent when not
   - Skaffold subsection present when Skaffold detected, absent when not
   - `devenv tasks k8s:status` task in devenv.nix when K8s active
   - Deny rules include `kubectl delete namespace*` and `helm uninstall*`
   - Deny rules absent when K8s not detected

**Acceptance Criteria:**
- [ ] K8s CLAUDE.md section generated with cluster, context, namespace, and KUBECONFIG shown
- [ ] Commands in CLAUDE.md are tool-specific (only tools present in devenv.nix appear)
- [ ] `devenv task k8s:status` added to devenv.nix when K8s active
- [ ] `k8s:status` task shows KUBECONFIG path, active context, and cluster connectivity
- [ ] Deny rules added to settings.json for `kubectl delete namespace`, `kubectl drain`, `kubectl cordon`, `kubectl delete pv/pvc`, `helm uninstall`, `helm rollback`
- [ ] Deny rules absent when K8s module is not active
- [ ] CLAUDE.md explains why deny rules are in place

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md` § 4.1 Multi-Client Credential Isolation — context awareness for safety
- `phases/04-claude-code-addon-core-generation.md` — CLAUDE.md generation framework this unit extends
- `phases/05-security-infrastructure-integration.md` — deny rule infrastructure this unit adds rules to

**Status:** Not Started

---

## Code-Grounded Implementation Notes

### Interface to Implement

All units implement the `EcosystemModule` interface from Phase 1. Units 24.1, 24.2, 24.3, and 24.4 are independent modules that compose over the shared K8s detection infrastructure in Unit 24.6.

### K8s File Detection Sharing

Units 24.1, 24.2, 24.3, and 24.4 all need to check for K8s indicator files. Implement a single `K8sFileDetector` (per the design in `cloud-k8s-module-design.md` Unit 2.17) that all modules call, returning a `K8sDetectionResult` with granular sub-feature flags rather than scanning the filesystem four times.

### New Packages (All in Nixpkgs)

| Package | Nixpkgs Attribute | Used By |
|---------|-------------------|---------|
| `kubectl` | `pkgs.kubectl` | Unit 24.1 |
| `kubectx` | `pkgs.kubectx` | Unit 24.1 |
| `k9s` | `pkgs.k9s` | Unit 24.1 |
| `stern` | `pkgs.stern` | Unit 24.1 |
| `kustomize` | `pkgs.kustomize` | Unit 24.1 (conditional) |
| `skaffold` | `pkgs.skaffold` | Unit 24.2 |
| `tilt` | `pkgs.tilt` | Unit 24.2 |
| `devspace` | `pkgs.devspace` | Unit 24.2 |
| `telepresence2` | `pkgs.telepresence2` | Unit 24.2 (opt-in) |
| `kubescape` | `pkgs.kubescape` | Unit 24.3 |
| `kube-linter` | `pkgs.kube-linter` | Unit 24.3 |
| `kube-bench` | `pkgs.kube-bench` | Unit 24.3 (CIS only) |
| `polaris` | `pkgs.polaris` | Unit 24.3 (CIS only) |
| `kubernetes-helm` via `wrapHelm` | `pkgs.wrapHelm pkgs.kubernetes-helm { ... }` | Unit 24.4 |
| `helmfile` | `pkgs.helmfile` | Unit 24.4 (conditional) |
| `helm-secrets` | `pkgs.kubernetes-helmPlugins.helm-secrets` | Unit 24.4 |
| `helm-diff` | `pkgs.kubernetes-helmPlugins.helm-diff` | Unit 24.4 |

### KUBECONFIG Invariant

The central safety invariant of this phase is stricter than the credential invariant in Phase 23: `KUBECONFIG` must never be set to `~/.kube/config` in any generated devenv.nix fragment. Every K8s module must use `KubeconfigTemplate` from Unit 24.6 to generate the `env.KUBECONFIG` line.

### Phase 23 Dependency

Unit 24.5 (Cloud-Auth Plugin Coordination) has a hard dependency on Phase 23 completion. Units 24.1 through 24.4 can be implemented before Phase 23 is complete, but the coordination unit (and thus the complete devenv.nix fragment for GCP+K8s and Azure+K8s combinations) requires Phase 23's cloud module detection results.

---

## Phase Completion Criteria

- [ ] All seven units pass acceptance criteria
- [ ] K8s Core Module detects all documented K8s indicator file patterns
- [ ] KUBECONFIG invariant: `grep -r '~/.kube/config' generated/` returns no matches
- [ ] Dev tools module: each of Skaffold, Tilt, DevSpace detected and installed only when their config file present
- [ ] Telepresence never auto-installed (wizard opt-in only)
- [ ] K8s security module does not auto-activate from standard K8s files
- [ ] Helm uses `wrapHelm` pattern in all generated fragments — no `helm plugin install` in any output
- [ ] helmfile included when `helmfile.yaml` detected, absent otherwise
- [ ] Cloud-auth coordination: GKE context shows `withExtraComponents [ gke-gcloud-auth-plugin ]` when GCP+K8s active
- [ ] Cloud-auth coordination: `kubelogin` present in Azure fragment when Azure+K8s active
- [ ] Deny rules for destructive K8s operations present in settings.json when K8s active
- [ ] `devenv task k8s:status` runs without error when cluster is reachable
- [ ] `gdev doctor --category k8s` completes within 30 seconds even when cluster unreachable (timeouts enforced)
