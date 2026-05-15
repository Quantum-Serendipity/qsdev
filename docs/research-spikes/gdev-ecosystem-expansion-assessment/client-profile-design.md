# Client Profile System — Implementation Unit Design

## Overview

This document defines implementation units for the client profile system in gdev. The system enables consulting engineers to maintain encrypted per-client configuration profiles (`~/.qsdev/clients/<name>.yaml`), select a profile during `qsdev init` to pre-populate cloud config, git identity, registry endpoints, and secret references, and manage profiles via `qsdev profile` CRUD commands.

Units amend two existing phases:
- **Phase 6** (Wizard & Orchestration) — units 6.7-6.10: age key management, profile CRUD, init-time profile selection, non-interactive profile mode
- **Phase 13** (Project Configuration & Team Standards) — units 13.8-13.11: profile schema & sops encryption, SecretSpec integration, baked config propagation, profile-aware compliance enforcement

## Design Decisions

**Init-time only, no runtime switching.** Profiles are applied at `qsdev init` time. Non-secret values are baked into `.qsdev.yaml`. Secret values generate `secretspec.toml` entries for devenv runtime resolution. To change profiles, re-run `qsdev init --profile <name>`. This was decided in the addon architecture fit assessment: init-time profile selection is orchestration, which is devinit's job.

**Two-layer secret handling.** Profile YAML is sops+age encrypted at rest in `~/.qsdev/clients/`. At `qsdev init` time, the profile is decrypted in memory. Non-secret values (aws_profile name, git email, registry URLs) are written to `.qsdev.yaml`. Secret values (API keys, tokens) are never written to any checked-in file; instead, they generate `secretspec.toml` entries that resolve at devenv shell entry via the configured provider (keyring, 1password, dotenv, env).

**sops+age, not GPG.** Age is simpler than GPG (no keyring daemon, no key server, no trust model). sops supports age natively. The age keypair lives at `~/.qsdev/keys/age.key` and is generated during `qsdev setup` if not present.

---

## Phase 6 Units (Wizard & Orchestration)

### Unit 6.7: Age Key Management in gdev setup

**Description:** Generate and manage an age keypair in `~/.qsdev/keys/` as part of the `qsdev setup` bootstrap flow, with prerequisite checking for sops and age binaries.

**Context:** Client profiles are sops+age encrypted at rest. Before any profile operations can work, the engineer needs an age keypair. This fits naturally into `qsdev setup` (Phase 9 bootstrap steps), which already handles system-level tool installation. The keypair is generated once per machine and reused across all profile operations. The public key is printed on generation so the engineer can share it with team leads who manage shared profile templates. If sops or age binaries are not found, the bootstrap step warns with installation instructions rather than silently failing.

**Desired Outcome:** After `qsdev setup`, `~/.qsdev/keys/age.key` exists containing a valid age keypair, and the public key is displayed for the engineer to record.

**Steps:**
1. Add a new bootstrap step `SetupAgeKeyStep()` registered in the devinit addon's bootstrap step list.
2. Check for `age` and `sops` binary availability using `toolcheck.Detect()`. If missing, print installation instructions (`nix profile install nixpkgs#age nixpkgs#sops`) and skip the step with a warning.
3. Check if `~/.qsdev/keys/age.key` already exists. If it does, print the public key and skip generation (idempotent).
4. If no keypair exists, run `age-keygen -o ~/.qsdev/keys/age.key` to generate a new keypair. Set file permissions to `0600`.
5. Create `~/.qsdev/keys/` directory with `0700` permissions if it does not exist.
6. Extract and display the public key from the generated file (first line comment contains `# public key: age1...`).
7. Create `~/.qsdev/.sops.yaml` with a default creation rule pointing to the local age key, so sops auto-discovers the key for encrypt/decrypt operations:
   ```yaml
   creation_rules:
     - age: >-
         <public-key>
   ```
8. Print guidance: "Save your public key. Team leads need it to share encrypted profile templates with you."

**Acceptance Criteria:**
- [ ] `qsdev setup` generates `~/.qsdev/keys/age.key` with `0600` permissions when no key exists
- [ ] `~/.qsdev/keys/` directory created with `0700` permissions
- [ ] Step is idempotent: re-running when key exists skips generation, prints existing public key
- [ ] Missing `age` binary produces clear installation instructions and skips step without error
- [ ] Missing `sops` binary produces clear installation instructions and skips step without error
- [ ] `~/.qsdev/.sops.yaml` created with correct age public key reference
- [ ] Public key printed to stdout on both generation and re-run

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/rejected-features-consulting-ops-research.md § 2.2 Client Environment Isolation` — client profiles as strongest consulting-specific differentiator
- `research-spikes/gdev-ecosystem-expansion-assessment/addon-architecture-fit-research.md § devinit Addon Expansions` — sops+age encrypted profiles in `~/.qsdev/clients/`
- `phases/06-wizard-orchestration.md § Unit 5.6` — bootstrap step registration pattern (SkipInContainer, headless mode)

**Status:** Not Started

---

### Unit 6.8: Profile CRUD Commands

**Description:** Implement `qsdev profile create|list|edit|delete` commands for managing sops+age encrypted client profile YAML files in `~/.qsdev/clients/`.

**Context:** Engineers need to create, inspect, modify, and remove client profiles without manually running sops commands. `qsdev profile create <name>` opens `$EDITOR` with a YAML template pre-populated with the profile schema, then sops-encrypts the result on save. `qsdev profile edit <name>` runs `sops edit` to decrypt-in-editor-encrypt in one operation. `qsdev profile list` shows available profiles with summary metadata (client name, compliance level, cloud provider). `qsdev profile delete <name>` removes the encrypted file after confirmation. All commands check for sops/age prerequisites before proceeding.

**Desired Outcome:** Engineers can manage the full lifecycle of client profiles through `qsdev profile` subcommands without directly invoking sops.

**Steps:**
1. Register `qsdev profile` command group with subcommands: `create`, `list`, `edit`, `delete`.
2. Implement prerequisite gate: all profile commands check for `sops` and `age` binaries and `~/.qsdev/keys/age.key` existence. If missing, print actionable message ("Run `qsdev setup` to generate your age keypair") and exit with error.
3. Implement `qsdev profile create <name>`:
   - Validate `<name>` is kebab-case, alphanumeric with hyphens only.
   - Check `~/.qsdev/clients/<name>.yaml` does not already exist. If it does, error with "Profile '<name>' already exists. Use `qsdev profile edit <name>` to modify."
   - Write a temporary plaintext YAML file with the full profile template (all fields with comments explaining each):
     ```yaml
     # Client Profile: <name>
     # All fields are optional. Remove or leave blank for fields that don't apply.

     # Cloud provider configuration
     cloud:
       aws_profile: ""        # AWS CLI named profile (from ~/.aws/config)
       gcp_project: ""        # GCP project ID
       azure_subscription: "" # Azure subscription ID

     # Git identity for this client's repositories
     git:
       user_email: ""         # e.g., colin@highspring.com
       signing_key: ""        # Path to signing key or key ID

     # Package registry endpoints
     registries:
       npm: ""                # Private npm registry URL
       docker: ""             # Private Docker registry URL
       nix_cache: ""          # Private Nix binary cache URL

     # Security compliance level: baseline, enhanced, or strict
     compliance: enhanced

     # Secret references (resolved at devenv shell entry via SecretSpec)
     # Format: name -> provider config
     # Supported providers: keyring, 1password, dotenv, env
     secrets: {}
       # Example:
       # npm_token:
       #   provider: keyring
       #   service: npm-registry
       #   account: acme-corp
       # aws_session:
       #   provider: 1password
       #   vault: Engineering
       #   item: "AWS Acme Corp"
       #   field: credential
     ```
   - Open the temp file in `$EDITOR` (fall back to `vi`). Wait for editor to close.
   - Validate the edited YAML parses correctly. If invalid, offer to re-open editor or abort.
   - Encrypt the validated YAML with sops: `sops encrypt --age <public-key> <temp-file> > ~/.qsdev/clients/<name>.yaml`.
   - Remove the temporary plaintext file.
   - Print confirmation: "Profile '<name>' created and encrypted."
4. Implement `qsdev profile list`:
   - Glob `~/.qsdev/clients/*.yaml`.
   - For each file, decrypt in memory via `sops decrypt`, parse YAML, extract summary fields (client name from filename, compliance level, cloud providers configured, git email).
   - Display as a table:
     ```
     NAME          COMPLIANCE  CLOUD       GIT IDENTITY
     acme-corp     enhanced    aws, gcp    colin@highspring.com
     initech       strict      aws         colin@highspring.com
     personal      baseline    -           colin@gmail.com
     ```
   - If no profiles exist, print "No client profiles found. Create one with `qsdev profile create <name>`."
5. Implement `qsdev profile edit <name>`:
   - Verify `~/.qsdev/clients/<name>.yaml` exists. If not, error with suggestion to create.
   - Run `sops edit ~/.qsdev/clients/<name>.yaml` which handles decrypt-edit-encrypt atomically.
   - After sops exits, decrypt and validate the updated YAML. If validation fails, warn but do not block (sops already saved the encrypted file).
6. Implement `qsdev profile delete <name>`:
   - Verify the file exists.
   - Prompt for confirmation: "Delete client profile '<name>'? This cannot be undone. [y/N]"
   - With `--yes` flag, skip confirmation.
   - Remove `~/.qsdev/clients/<name>.yaml`.
   - Print confirmation.
7. Handle edge cases:
   - `~/.qsdev/clients/` directory does not exist: create it on first `qsdev profile create`.
   - Editor exits with non-zero: treat as abort, do not encrypt.
   - sops decrypt fails (wrong key, corrupted file): print clear error with troubleshooting steps.
   - Multiple age recipients: support `--recipient <age-pubkey>` flag on create for shared profiles.

**Acceptance Criteria:**
- [ ] `qsdev profile create acme-corp` opens editor with full template, encrypts on save
- [ ] `qsdev profile create` fails gracefully when sops/age not installed, with `qsdev setup` hint
- [ ] `qsdev profile create` fails gracefully when age key not generated, with `qsdev setup` hint
- [ ] `qsdev profile create` rejects non-kebab-case names with clear error
- [ ] `qsdev profile create` for an existing profile name errors with edit suggestion
- [ ] `qsdev profile list` shows table of all profiles with summary metadata
- [ ] `qsdev profile list` with no profiles shows helpful create hint
- [ ] `qsdev profile edit acme-corp` invokes `sops edit` for atomic decrypt-edit-encrypt
- [ ] `qsdev profile delete acme-corp` requires confirmation (skippable with `--yes`)
- [ ] Temporary plaintext file removed after encryption (never left on disk)
- [ ] Editor abort (non-zero exit) cancels profile creation without leaving artifacts
- [ ] Invalid YAML after editor save offers re-edit or abort

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/rejected-features-consulting-ops-research.md § 2.2 Client Environment Isolation` — profile YAML schema, `qsdev switch` concept (evolved to init-time selection)
- `research-spikes/gdev-ecosystem-expansion-assessment/addon-architecture-fit-research.md § devinit Addon Expansions` — sops+age encrypted profiles, profile CRUD in devinit

**Status:** Not Started

---

### Unit 6.9: Init-Time Profile Selection in Wizard

**Description:** Extend the `qsdev init` wizard (Unit 5.2) with a profile selection step that lists available client profiles from `~/.qsdev/clients/` and integrates the selected profile's values into the wizard answers.

**Context:** During `qsdev init`, before the ecosystem detection and language selection steps, the wizard should offer client profile selection. This is a new wizard group inserted between the quick-path question (Group 1) and the languages group (Group 2). If profiles exist, the wizard shows them as selectable options plus "None (skip)" and "Create new profile". If no profiles exist, this group is hidden entirely. The selected profile pre-populates downstream wizard answers: cloud config, git identity, registry endpoints, compliance level, and secret references. The engineer can still customize these in subsequent wizard groups. Profile selection happens before detection so that profile-specific registries and compliance levels influence the generation pipeline.

**Desired Outcome:** `qsdev init` in a project shows available client profiles, and selecting one pre-populates wizard answers with the profile's non-secret values and queues secret references for SecretSpec generation.

**Steps:**
1. Add a new wizard form group (Group 1.5, between quick selection and languages) using `huh.NewGroup()`:
   - Title: "Client Profile"
   - Show only when `~/.qsdev/clients/` contains at least one `.yaml` file.
   - Hide via `WithHideFunc` when quick path is selected AND a `--profile` flag was provided.
   - Options: list of profile names from `qsdev profile list` logic + "None (skip client profile)" + "Create new (opens profile editor)".
2. On profile selection, decrypt the selected profile in memory via `sops decrypt`.
3. Map profile fields to `WizardAnswers`:
   - `cloud.aws_profile` -> set `AWS_PROFILE` in devenv env vars
   - `cloud.gcp_project` -> set `CLOUDSDK_CORE_PROJECT` in devenv env vars
   - `cloud.azure_subscription` -> set `AZURE_SUBSCRIPTION_ID` in devenv env vars
   - `git.user_email` -> set git user.email in devenv git config
   - `git.signing_key` -> set git user.signingkey in devenv git config
   - `registries.npm` -> set npm registry in `.npmrc` generation
   - `registries.docker` -> set docker registry in devenv config
   - `registries.nix_cache` -> set substituters in devenv.nix
   - `compliance` -> set `security.level` in resolved config
   - `secrets` -> queue for SecretSpec generation (Unit 13.9)
4. Store the selected profile name in `WizardAnswers` so it persists to `.qsdev.yaml` as `client.name`.
5. If "Create new" is selected, invoke `qsdev profile create` flow inline (open editor, encrypt, then re-read the new profile).
6. If "None" is selected, skip all profile-related pre-population. Downstream wizard groups show their normal defaults.
7. Pre-populated values from the profile appear as defaults in subsequent wizard groups. The engineer can override any value.
8. Display a summary of what the profile provides before proceeding to customization groups:
   ```
   Client profile: acme-corp
     Cloud: AWS (profile: acme-prod), GCP (project: acme-web)
     Git: colin@highspring.com (signing: ~/.ssh/acme-signing)
     Registry: npm.acme-internal.com
     Compliance: enhanced
     Secrets: 2 references (will generate SecretSpec entries)
   ```

**Acceptance Criteria:**
- [ ] Profile selection group appears in wizard when profiles exist in `~/.qsdev/clients/`
- [ ] Profile selection group hidden when no profiles exist
- [ ] Profile selection group hidden on quick path with `--profile` flag
- [ ] Selected profile's values pre-populate downstream wizard answers
- [ ] Cloud config maps to correct environment variables in devenv generation
- [ ] Git identity maps to devenv git configuration
- [ ] Registry endpoints map to ecosystem-specific config file generation
- [ ] Compliance level from profile sets security floor for the project
- [ ] Secret references queued for SecretSpec generation (not written to any checked-in file)
- [ ] "Create new" option launches inline profile creation flow
- [ ] "None" option skips all profile pre-population
- [ ] Profile summary displayed before proceeding to customization
- [ ] Pre-populated values are overridable in subsequent wizard groups

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/addon-architecture-fit-research.md § Why No Fourth Addon?` — init-time profile selection is devinit orchestration
- `research-spikes/gdev-ecosystem-expansion-assessment/addon-architecture-fit-research.md § devinit Addon Expansions` — non-secret values baked into project config, secret values generate SecretSpec entries
- `phases/06-wizard-orchestration.md § Unit 5.2` — huh wizard form group structure, `WithHideFunc` pattern

**Status:** Not Started

---

### Unit 6.10: Non-Interactive Profile Mode

**Description:** Implement `qsdev init --profile <client-name> --yes` for fully non-interactive project initialization using a client profile, supporting CI and scripted setup workflows.

**Context:** The existing non-interactive mode (Unit 5.5) maps wizard fields to CLI flags. Client profiles add a new dimension: `--profile <name>` selects a client profile that provides cloud config, git identity, registries, compliance level, and secret references. Combined with `--yes`, this enables zero-question project setup: `qsdev init --profile acme-corp --lang go --service postgres --yes`. The `--profile` flag is distinct from the existing `--profile` for project type profiles (go-web, ts-fullstack); the client profile flag uses `--client-profile` or reuses `--profile` with disambiguation logic. For clarity, this unit uses `--client-profile` to avoid collision with the existing project-type `--profile` flag.

**Desired Outcome:** `qsdev init --client-profile acme-corp --yes` initializes a project with all values from the acme-corp client profile and defaults for any unspecified fields, with zero interactive prompts.

**Steps:**
1. Add `--client-profile <name>` flag to `qsdev init` command.
2. When `--client-profile` is specified:
   - Verify the profile exists in `~/.qsdev/clients/`.
   - Decrypt and parse the profile.
   - Map profile values to `WizardAnswers` (same mapping as Unit 6.9).
   - Combine with any other flags (`--lang`, `--service`, etc.) — explicit flags override profile values.
3. When `--client-profile` combined with `--yes`:
   - Skip wizard entirely.
   - Fill all unspecified fields from detection results, then from compiled defaults.
   - Generate all files and report results.
4. When `--client-profile` without `--yes`:
   - Pre-populate wizard with profile values.
   - Run wizard for confirmation/customization of remaining fields.
5. Error handling:
   - `--client-profile nonexistent` with `--yes`: exit with error and list available profiles.
   - `--client-profile` when sops/age not available: exit with error and `qsdev setup` hint.
   - `--client-profile` when age key not present: exit with error and `qsdev setup` hint.
6. Support combining both profile types: `qsdev init --profile go-web --client-profile acme-corp --yes` uses go-web as project type template and acme-corp for client-specific config.
7. Log the profile used in `.devinit/.qsdev-init-answers.yaml` so `qsdev init` in Join mode knows which profile was originally selected.

**Acceptance Criteria:**
- [ ] `qsdev init --client-profile acme-corp --yes` produces complete project config with zero prompts
- [ ] `qsdev init --client-profile acme-corp` pre-populates wizard with profile values
- [ ] `--client-profile` combined with `--lang`/`--service` flags: explicit flags override profile values
- [ ] `--client-profile` combined with `--profile` (project type): both applied, client profile for identity/cloud, project profile for ecosystem
- [ ] Nonexistent profile name with `--yes` exits with error listing available profiles
- [ ] Missing sops/age prerequisites produce actionable error with `qsdev setup` hint
- [ ] Profile selection recorded in internal state for Join mode awareness
- [ ] All generated files respect the profile's compliance level as security floor

**Research Citations:**
- `phases/06-wizard-orchestration.md § Unit 5.5` — non-interactive flag mapping pattern, `answersFromFlags` approach
- `research-spikes/gdev-ecosystem-expansion-assessment/addon-architecture-fit-research.md § Client profiles` — `qsdev init --profile <client>` design
- `research-spikes/gdev-ecosystem-expansion-assessment/rejected-features-consulting-ops-research.md § 2.2` — bundled client switching concept

**Status:** Not Started

---

## Phase 13 Units (Project Configuration & Team Standards)

### Unit 13.8: Client Profile Schema & sops Encryption Layer

**Description:** Define the canonical YAML schema for client profile files (`~/.qsdev/clients/<name>.yaml`), the Go struct types with YAML and validation tags, and the sops+age encryption/decryption wrapper functions.

**Context:** Client profiles are the bridge between a consulting engineer's per-client credentials and gdev's project configuration system. The profile schema must cover five domains: cloud provider config, git identity, registry endpoints, compliance level, and secret references. The schema is versioned independently of `.qsdev.yaml` (profiles evolve on a different cadence than project config). All profiles are sops+age encrypted at rest; the encryption layer provides `Encrypt(plaintext, recipients) -> ciphertext` and `Decrypt(path) -> plaintext` functions that wrap sops CLI invocations. The decryption function uses the age key at `~/.qsdev/keys/age.key` (or `SOPS_AGE_KEY_FILE` env var).

**Desired Outcome:** A fully typed, validated profile schema with encryption/decryption wrappers that other units can import, and clear separation between secret and non-secret fields in the type system.

**Steps:**
1. Define the `ClientProfile` struct in `pkg/types/client_profile.go`:
   ```go
   type ClientProfile struct {
       // Schema version for profile format (independent of .qsdev.yaml version).
       Version int `yaml:"version" validate:"required,min=1"`

       // Cloud provider configuration (non-secret: profile/project names only).
       Cloud CloudConfig `yaml:"cloud,omitempty"`

       // Git identity for this client's repositories (non-secret).
       Git GitIdentityConfig `yaml:"git,omitempty"`

       // Package registry endpoints (non-secret: URLs only).
       Registries RegistryConfig `yaml:"registries,omitempty"`

       // Security compliance level applied as floor.
       Compliance string `yaml:"compliance,omitempty" validate:"omitempty,oneof=baseline enhanced strict"`

       // Secret references mapped to SecretSpec provider configurations.
       // These are NOT the secrets themselves — they are instructions for
       // SecretSpec on where to find the secrets at devenv shell entry.
       Secrets map[string]SecretRef `yaml:"secrets,omitempty"`
   }

   type CloudConfig struct {
       AWSProfile        string `yaml:"aws_profile,omitempty"`
       GCPProject        string `yaml:"gcp_project,omitempty"`
       AzureSubscription string `yaml:"azure_subscription,omitempty"`
   }

   type GitIdentityConfig struct {
       UserEmail  string `yaml:"user_email,omitempty" validate:"omitempty,email"`
       SigningKey string `yaml:"signing_key,omitempty"`
   }

   type RegistryConfig struct {
       NPM      string `yaml:"npm,omitempty" validate:"omitempty,url"`
       Docker   string `yaml:"docker,omitempty" validate:"omitempty,url"`
       NixCache string `yaml:"nix_cache,omitempty" validate:"omitempty,url"`
   }

   type SecretRef struct {
       Provider string            `yaml:"provider" validate:"required,oneof=keyring 1password dotenv env"`
       Config   map[string]string `yaml:"config,omitempty"`
       // Provider-specific config keys:
       // keyring: service, account
       // 1password: vault, item, field
       // dotenv: file, key
       // env: var
   }
   ```
2. Implement `ParseClientProfile(data []byte) (*ClientProfile, error)`:
   - Unmarshal YAML, validate struct tags.
   - Check `version` field. If missing or unsupported, error with actionable message.
   - Validate secret reference provider configs have required keys per provider.
3. Implement `ValidateClientProfile(p *ClientProfile) []ValidationError`:
   - Each cloud field, if set, must be non-empty string.
   - Git email, if set, must be valid email format.
   - Registry URLs, if set, must be valid URLs.
   - Compliance, if set, must be one of baseline/enhanced/strict.
   - Each secret ref must have a valid provider with required config keys.
4. Implement the sops encryption wrapper in `internal/sops/sops.go`:
   ```go
   // Decrypt reads a sops-encrypted file and returns plaintext bytes.
   // Uses the age key at ~/.qsdev/keys/age.key (or SOPS_AGE_KEY_FILE).
   func Decrypt(path string) ([]byte, error)

   // Encrypt takes plaintext bytes and writes a sops-encrypted file.
   // Recipients are age public keys.
   func Encrypt(plaintext []byte, outputPath string, recipients []string) error

   // Edit opens a sops-encrypted file in $EDITOR for atomic edit.
   // Equivalent to `sops edit <path>`.
   func Edit(path string) error

   // EnsurePrerequisites checks that sops and age are installed
   // and an age key exists. Returns a user-facing error if not.
   func EnsurePrerequisites() error
   ```
5. Implement `Decrypt`:
   - Set `SOPS_AGE_KEY_FILE` env var to `~/.qsdev/keys/age.key` if not already set.
   - Execute `sops decrypt <path>` and capture stdout.
   - If sops exits non-zero, parse error message and return user-facing error:
     - "could not decrypt" -> "Cannot decrypt profile. Your age key may not be a recipient. Ask the profile creator to re-encrypt with your public key."
     - "file not found" -> "Profile file not found at <path>."
6. Implement `Encrypt`:
   - Write plaintext to a temp file.
   - Execute `sops encrypt --age <recipients-comma-separated> <temp-file>`.
   - Write encrypted output to `outputPath`.
   - Remove temp file (with defer to ensure cleanup on error).
   - Set file permissions on output to `0600`.
7. Define profile schema version constants:
   ```go
   const (
       ProfileVersionMin     = 1
       ProfileVersionMax     = 1
       ProfileVersionCurrent = 1
   )
   ```
8. Write unit tests:
   - Valid profile parses all fields correctly.
   - Missing version field produces clear error.
   - Invalid compliance level produces validation error.
   - Invalid email format in git.user_email produces validation error.
   - Secret ref with unknown provider produces validation error.
   - Secret ref missing required config keys produces validation error (e.g., keyring without service).
   - Encryption round-trip: encrypt plaintext, decrypt, compare.

**Acceptance Criteria:**
- [ ] `ClientProfile` struct covers all five domains: cloud, git, registries, compliance, secrets
- [ ] Schema version field enables future profile format evolution independent of `.qsdev.yaml`
- [ ] `ParseClientProfile` validates all fields with clear error messages
- [ ] Secret references typed by provider with per-provider config validation
- [ ] `Decrypt` wrapper uses `~/.qsdev/keys/age.key` and provides user-facing error messages
- [ ] `Encrypt` wrapper supports multiple age recipients for shared profiles
- [ ] Temporary plaintext files cleaned up on both success and error paths
- [ ] Encrypted output files have `0600` permissions
- [ ] `EnsurePrerequisites` checks sops binary, age binary, and age key existence
- [ ] Profile schema version independent of `.qsdev.yaml` config version

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/rejected-features-consulting-ops-research.md § 2.2 Client Environment Isolation` — profile fields: aws_profile, git identity, ssh_key, env_vars
- `research-spikes/gdev-ecosystem-expansion-assessment/addon-architecture-fit-research.md § devinit Addon Expansions` — sops+age encryption, two-layer (sops at rest, SecretSpec at runtime)
- `phases/13-project-configuration-team-standards.md § Unit 13.1` — `ClientConfig` struct in `.qsdev.yaml` (profile data flows into this)

**Status:** Not Started

---

### Unit 13.9: SecretSpec Integration & Generation

**Description:** Implement the mapping from client profile secret references to `secretspec.toml` entries, generating a per-project SecretSpec configuration file that resolves secrets at devenv shell entry time.

**Context:** Client profiles contain secret references (not secrets) — instructions for SecretSpec on where to find each secret at runtime. During `qsdev init`, when a profile with secrets is selected, gdev generates a `secretspec.toml` file in the project root (gitignored) that maps each secret name to a SecretSpec provider entry. At devenv shell entry, SecretSpec reads `secretspec.toml` and resolves each entry from the configured provider (keyring, 1password, dotenv, or env). This means the secret value only exists in memory during the devenv session — it never touches disk in plaintext, never appears in `.qsdev.yaml`, `devenv.nix`, or any committed file. The `secretspec.toml` itself contains no secrets — only provider references (e.g., "look up 'npm-token' in the system keyring under service 'npm-registry'").

**Desired Outcome:** `qsdev init` with a client profile that has secret references produces a `secretspec.toml` that SecretSpec can resolve, and `devenv shell` makes the secrets available as environment variables.

**Steps:**
1. Define the SecretSpec TOML generation types in `internal/secretspec/generate.go`:
   ```go
   type SecretSpecConfig struct {
       Secrets map[string]SecretSpecEntry `toml:"secrets"`
   }

   type SecretSpecEntry struct {
       EnvVar   string `toml:"env_var"`            // Environment variable name to set
       Provider string `toml:"provider"`            // keyring, 1password, dotenv, env
       // Provider-specific fields
       Service  string `toml:"service,omitempty"`   // keyring
       Account  string `toml:"account,omitempty"`   // keyring
       Vault    string `toml:"vault,omitempty"`     // 1password
       Item     string `toml:"item,omitempty"`      // 1password
       Field    string `toml:"field,omitempty"`     // 1password
       File     string `toml:"file,omitempty"`      // dotenv
       Key      string `toml:"key,omitempty"`       // dotenv
       Var      string `toml:"var,omitempty"`       // env (source env var name)
   }
   ```
2. Implement `GenerateSecretSpec(secrets map[string]SecretRef, envMapping map[string]string) (*SecretSpecConfig, error)`:
   - For each secret reference in the profile, create a `SecretSpecEntry`.
   - Map the profile secret name to an environment variable name using `envMapping` (e.g., `npm_token` -> `NPM_TOKEN`, `aws_session` -> `AWS_SESSION_TOKEN`).
   - Copy provider-specific config fields from the `SecretRef.Config` map to the typed `SecretSpecEntry` fields.
   - Validate that all required provider-specific fields are present.
3. Implement `WriteSecretSpec(config *SecretSpecConfig, projectRoot string) error`:
   - Marshal to TOML format.
   - Write to `<projectRoot>/secretspec.toml`.
   - Add a header comment:
     ```toml
     # Generated by qsdev init from client profile.
     # This file is gitignored. It contains NO secrets — only references
     # to where SecretSpec should find them at devenv shell entry.
     # Re-generate with: qsdev init --client-profile <name>
     ```
4. Implement `MergeSecretSpec(existing *SecretSpecConfig, new *SecretSpecConfig) *SecretSpecConfig`:
   - If `secretspec.toml` already exists, merge rather than overwrite.
   - New entries from the profile are added. Existing entries not from the profile are preserved.
   - Conflicting entries (same name, different provider): new profile value wins with a warning.
5. Ensure `secretspec.toml` is added to `.gitignore`:
   - Use the same gitignore management pattern from Phase 13 Unit 13.2 (section markers).
   - Entry: `secretspec.toml` under a `# gdev-managed` section.
6. Implement default environment variable mapping for common secret names:
   ```go
   var DefaultEnvMapping = map[string]string{
       "npm_token":         "NPM_TOKEN",
       "aws_access_key":    "AWS_ACCESS_KEY_ID",
       "aws_secret_key":    "AWS_SECRET_ACCESS_KEY",
       "aws_session":       "AWS_SESSION_TOKEN",
       "docker_password":   "DOCKER_PASSWORD",
       "gcp_credentials":   "GOOGLE_APPLICATION_CREDENTIALS",
       "azure_client_secret": "AZURE_CLIENT_SECRET",
   }
   ```
   - Custom mappings can be specified in the profile's secret config with an `env_var` key.
7. Wire into the `qsdev init` pipeline:
   - After profile selection (Unit 6.9) and wizard completion, if the resolved profile has secrets, call `GenerateSecretSpec`.
   - Include `secretspec.toml` in the plan preview (Unit 5.2 Group 6) so the engineer sees what will be generated.
   - In the post-generation summary, list generated SecretSpec entries and remind the engineer to populate their secrets in the configured provider.
8. Write unit tests:
   - Profile with keyring secrets generates correct TOML.
   - Profile with 1password secrets generates correct TOML.
   - Profile with mixed providers generates correct TOML.
   - Profile with no secrets generates no `secretspec.toml`.
   - Merge with existing `secretspec.toml` preserves non-profile entries.
   - Default env mapping applies for known secret names.
   - Custom env_var in secret config overrides default mapping.

**Acceptance Criteria:**
- [ ] Profile secret references generate valid `secretspec.toml` with correct provider entries
- [ ] Four providers supported: keyring, 1password, dotenv, env
- [ ] `secretspec.toml` contains NO secret values — only provider references
- [ ] `secretspec.toml` automatically added to `.gitignore`
- [ ] Default environment variable mapping for common secret names (NPM_TOKEN, AWS_ACCESS_KEY_ID, etc.)
- [ ] Custom env_var mapping via secret config overrides defaults
- [ ] Merge with existing `secretspec.toml` preserves non-profile entries
- [ ] Plan preview shows SecretSpec entries that will be generated
- [ ] Post-generation summary reminds engineer to populate secrets in their provider
- [ ] Profile with no secrets does not generate `secretspec.toml`
- [ ] Generated TOML includes header comment explaining the file's purpose and regeneration command

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/addon-architecture-fit-research.md § devinit Addon Expansions` — "Secret values generate SecretSpec entries resolved at devenv runtime via provider (keyring/1Password/env)"
- `research-spikes/gdev-ecosystem-expansion-assessment/rejected-features-consulting-ops-research.md § 2.2` — credential tools integration (aws-vault, Granted) as provider model
- `phases/13-project-configuration-team-standards.md § Unit 13.1` — `ClientConfig` schema, `.qsdev.yaml` client block

**Status:** Not Started

---

### Unit 13.10: Baked Config Propagation to .qsdev.yaml

**Description:** Implement the logic that extracts non-secret values from a selected client profile and writes them into `.qsdev.yaml` as the `client` block, propagating cloud config, git identity, registry endpoints, and compliance level into the committed project configuration.

**Context:** When a client profile is selected during `qsdev init`, its non-secret values become part of the project's `.qsdev.yaml` so that other team members who `qsdev init` (Join mode) get the same cloud provider, git identity, registry, and compliance settings without needing the original encrypted profile. This is the "baking" step: profile data is flattened into `.qsdev.yaml` fields. Secret references are NOT baked — they go to `secretspec.toml` (Unit 13.9). The baked values use the existing `ClientConfig` struct from Unit 13.1. After baking, the profile name is recorded so `qsdev init --update` can detect when the profile has changed and offer to re-bake.

**Desired Outcome:** Selecting a client profile during `qsdev init` writes non-secret values to `.qsdev.yaml` `client` block, and team members in Join mode receive these values without needing the encrypted profile.

**Steps:**
1. Implement `BakeProfileToConfig(profile *ClientProfile, profileName string) *ClientConfig`:
   ```go
   func BakeProfileToConfig(profile *ClientProfile, profileName string) *ClientConfig {
       cfg := &ClientConfig{
           Name:          profileName,
           SecurityLevel: profile.Compliance,
       }

       // Bake registry endpoints
       if profile.Registries.NPM != "" {
           cfg.RegistryProxy = profile.Registries.NPM
       }
       if profile.Registries.NixCache != "" {
           cfg.NixCache = profile.Registries.NixCache
       }

       // Map compliance to compliance frameworks
       switch profile.Compliance {
       case "strict":
           cfg.Compliance = []string{"soc2", "hipaa"}
       case "enhanced":
           cfg.Compliance = []string{"soc2"}
       case "baseline":
           cfg.Compliance = []string{}
       }

       return cfg
   }
   ```
2. Implement `BakeProfileToEnvVars(profile *ClientProfile) map[string]string`:
   - Map cloud config to environment variables:
     - `aws_profile` -> `AWS_PROFILE`
     - `gcp_project` -> `CLOUDSDK_CORE_PROJECT`
     - `azure_subscription` -> `AZURE_SUBSCRIPTION_ID`
   - These env vars are written to devenv.nix `env` block via the generation pipeline.
3. Implement `BakeProfileToGitConfig(profile *ClientProfile) *GitConfig`:
   - Map git identity to devenv git configuration:
     - `user_email` -> `git.user.email` in devenv enterShell hook
     - `signing_key` -> `git.user.signingkey` in devenv enterShell hook
   - Generate the enterShell git config lines only if values are non-empty.
4. Wire baking into the `qsdev init` generation pipeline:
   - After wizard completion, if a client profile was selected:
     a. Call `BakeProfileToConfig` -> set `GdevConfig.Client`.
     b. Call `BakeProfileToEnvVars` -> merge into `WizardAnswers.ExtraEnvVars`.
     c. Call `BakeProfileToGitConfig` -> merge into devenv generation context.
   - The baked `ClientConfig` is included when writing `.qsdev.yaml`.
5. Implement profile change detection for Update mode:
   - Store the profile name and a hash of the baked values in `.devinit/.qsdev-init-state.yaml`.
   - On `qsdev init` in Update mode, if the profile name matches but the hash differs (profile was edited), offer to re-bake.
   - On `qsdev init` in Update mode, if a different profile is specified via `--client-profile`, confirm the switch and re-bake.
6. Handle partial profiles:
   - A profile with only `cloud.aws_profile` and no other fields produces a `ClientConfig` with just the name, plus `AWS_PROFILE` env var. No registry, git, or compliance values are baked.
   - Missing fields produce no output (not empty strings in `.qsdev.yaml`).
7. Write unit tests:
   - Full profile bakes all fields to `ClientConfig`, env vars, and git config.
   - Partial profile (cloud only) bakes only cloud env vars.
   - Empty profile produces `ClientConfig` with only the name.
   - Re-bake after profile edit detects hash change.
   - Profile switch (different name) triggers re-bake with confirmation.
   - Baked values in `.qsdev.yaml` are readable by Join mode without the encrypted profile.

**Acceptance Criteria:**
- [ ] Non-secret profile values written to `.qsdev.yaml` `client` block
- [ ] Cloud config baked as environment variables in devenv.nix generation
- [ ] Git identity baked as enterShell git config hooks
- [ ] Registry endpoints baked as `ClientConfig` fields for downstream generation
- [ ] Compliance level baked as `SecurityLevel` in `ClientConfig`
- [ ] Secret references excluded from `.qsdev.yaml` (only in `secretspec.toml`)
- [ ] Profile name and baked-values hash stored in state for change detection
- [ ] Update mode detects profile edits via hash comparison and offers re-bake
- [ ] Profile switch (`--client-profile` with different name) triggers re-bake with confirmation
- [ ] Partial profiles bake only the fields that are set (no empty strings in output)
- [ ] Join mode works with baked values without requiring the encrypted profile

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/addon-architecture-fit-research.md § devinit Addon Expansions` — "Non-secret values (aws_profile name, git email, registry URLs) baked into project config"
- `phases/13-project-configuration-team-standards.md § Unit 13.1` — `ClientConfig` struct definition, `.qsdev.yaml` schema
- `phases/13-project-configuration-team-standards.md § Unit 13.3` — onboarding mode detection, Update mode re-bake trigger

**Status:** Not Started

---

### Unit 13.11: Profile-Aware Compliance Enforcement in qsdev check

**Description:** Extend `qsdev check` (Unit 13.6) to validate that the project's configuration satisfies the compliance requirements declared by the baked client profile, including security level floor checks, required tool presence, and MCP server policy enforcement.

**Context:** When a client profile specifies `compliance: strict`, the baked `ClientConfig.SecurityLevel` in `.qsdev.yaml` becomes the compliance floor for `qsdev check`. Unit 13.7 already defines compliance level mappings (baseline/enhanced/strict) to concrete settings (age-gating threshold, required hooks, SBOM policy, etc.). This unit adds profile-specific checks on top: verifying that the baked compliance level matches the profile's declared level (detecting tampering), that profile-specific registry endpoints are configured, and that cloud provider env vars are set. These checks belong in the "config integrity" and "security hardening" categories of `qsdev check`.

**Desired Outcome:** `qsdev check` on a project with a baked client profile verifies that the project meets the client's compliance requirements, catching manual downgrades of security level or removal of required configurations.

**Steps:**
1. Add profile-aware checks to the "config integrity" category:
   ```go
   func checkProfileIntegrity(cfg *GdevConfig) []CheckResult {
       var results []CheckResult

       if cfg.Client == nil {
           return results // No client profile, skip profile checks
       }

       // Check that client name is non-empty
       if cfg.Client.Name == "" {
           results = append(results, CheckResult{
               Category:    CategoryConfigIntegrity,
               Name:        "client_profile_name",
               Status:      "fail",
               Severity:    SeverityHigh,
               Message:     "Client block present but name is empty",
               Remediation: "Set client.name in .qsdev.yaml or re-run qsdev init --client-profile <name>",
           })
       }

       // Check compliance level is valid
       if cfg.Client.SecurityLevel != "" {
           if _, ok := ComplianceLevels[cfg.Client.SecurityLevel]; !ok {
               results = append(results, CheckResult{
                   Category:    CategoryConfigIntegrity,
                   Name:        "client_compliance_valid",
                   Status:      "fail",
                   Severity:    SeverityCritical,
                   Message:     fmt.Sprintf("Unknown compliance level: %s", cfg.Client.SecurityLevel),
                   Remediation: "Valid levels: baseline, enhanced, strict",
               })
           }
       }

       return results
   }
   ```
2. Add profile-aware checks to the "security hardening" category:
   - Verify the resolved security level meets or exceeds the client's declared level.
   - Verify required pre-commit hooks for the compliance level are present.
   - Verify SBOM policy matches the compliance level's requirement.
   - Verify Claude Code permission level does not exceed the compliance level's maximum.
   - If client specifies `blocked_mcp_servers`, verify none of the blocked servers are configured.
   - If client specifies `allowed_mcp_servers`, verify no servers outside the allowlist are configured.
3. Add registry consistency checks:
   - If `client.registry_proxy` is set, verify `.npmrc` (or equivalent) points to it.
   - If `client.nix_cache` is set, verify devenv.nix substituters include it.
4. Add environment variable checks:
   - Verify that cloud provider env vars baked from the profile are present in the devenv.nix env block.
   - Verify git identity configuration matches the baked values.
   - These are "warn" severity (not "fail") because the baked values might have been intentionally overridden.
5. Implement `--check-profile` flag for targeted profile compliance check:
   - `qsdev check --check-profile` runs only profile-related checks (useful for quick validation after profile edit/re-bake).
   - Without the flag, profile checks run as part of the normal `qsdev check` suite.
6. Add remediation suggestions specific to profile issues:
   - "Security level lowered below client requirement" -> "Re-run `qsdev init --client-profile <name>` to re-apply profile compliance level"
   - "Required pre-commit hook missing" -> "Run `qsdev enable <hook>` or add to tools.enabled in .qsdev.yaml"
   - "MCP server not in client allowlist" -> "Remove the server from claude_code.mcp_servers or update the client profile"
7. Wire profile checks into the existing `qsdev check` pipeline (Unit 13.6):
   - Profile checks run after security hardening checks.
   - Profile check failures at "critical" severity block the pipeline regardless of `--audit-level`.
8. Write unit tests:
   - Project with strict client profile and enhanced security level fails check.
   - Project with enhanced client profile and enhanced security level passes check.
   - Missing required hook for compliance level detected.
   - Blocked MCP server configured in project detected.
   - Registry mismatch between client profile and .npmrc detected.
   - `--check-profile` runs only profile checks.
   - Project with no client block skips all profile checks.

**Acceptance Criteria:**
- [ ] `qsdev check` validates client compliance level has not been manually lowered
- [ ] `qsdev check` validates required pre-commit hooks for the compliance level are present
- [ ] `qsdev check` validates SBOM policy matches compliance level requirement
- [ ] `qsdev check` validates Claude Code permission level does not exceed compliance maximum
- [ ] `qsdev check` validates blocked MCP servers are not configured
- [ ] `qsdev check` validates allowed MCP server policy is respected
- [ ] `qsdev check` warns (not fails) when baked env vars or git config differ from expected
- [ ] `qsdev check` validates registry endpoints match baked profile values
- [ ] `--check-profile` flag runs only profile-related checks
- [ ] Remediation suggestions reference `qsdev init --client-profile` for re-bake
- [ ] Project with no client block skips all profile checks cleanly
- [ ] Profile compliance violations at critical severity are not suppressible by `--audit-level`

**Research Citations:**
- `phases/13-project-configuration-team-standards.md § Unit 13.6` — `qsdev check` structure, check categories, output formats
- `phases/13-project-configuration-team-standards.md § Unit 13.7` — compliance level mappings, security floor enforcement
- `research-spikes/gdev-ecosystem-expansion-assessment/addon-architecture-fit-research.md § Risk Assessment` — "Client profiles become an identity management system" risk mitigation via strict scope

**Status:** Not Started

---

## Cross-Unit Dependency Graph

```
Phase 6:
  6.7 (Age Key Management)
    └─> 6.8 (Profile CRUD) — requires age key
        └─> 6.9 (Init-Time Selection) — requires profile CRUD
            └─> 6.10 (Non-Interactive Mode) — requires wizard integration

Phase 13:
  13.8 (Profile Schema & sops Layer) — independent, can parallel with 6.7
    ├─> 13.9 (SecretSpec Integration) — requires schema types
    └─> 13.10 (Baked Config Propagation) — requires schema types
        └─> 13.11 (Profile Compliance Enforcement) — requires baked config in .qsdev.yaml

Cross-phase:
  6.8 (Profile CRUD) depends on 13.8 (Profile Schema)
  6.9 (Init-Time Selection) depends on 13.8 (Profile Schema) + 13.9 (SecretSpec)
  6.10 (Non-Interactive Mode) depends on 13.10 (Baked Config)
```

## Security Invariants

These hold across all units:

1. **Profile YAML is always sops-encrypted on disk.** Plaintext only exists in memory during `qsdev init` decryption and in `$EDITOR` during `qsdev profile create/edit` (sops handles the editor lifecycle).
2. **No plaintext secrets in committed files.** `.qsdev.yaml`, `devenv.nix`, and all generated files contain only non-secret values. Secret references in `secretspec.toml` point to providers, not values.
3. **Secret values exist in memory only during devenv shell sessions.** SecretSpec resolves from provider at shell entry, sets env vars for the session, and they are gone when the shell exits.
4. **Age private key never leaves `~/.qsdev/keys/`.** File permissions `0600`, directory permissions `0700`.
5. **Temporary plaintext files are always cleaned up.** Profile creation uses defer-based cleanup. Editor abort does not leave plaintext artifacts.
