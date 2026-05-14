# Phase 37: Consulting Infrastructure & Analytics Validation

## Goal

Validate all consulting infrastructure phases using the testscript E2E framework from Phase 17: client profile encryption round-trips (Phase 30), profile compliance enforcement (Phase 31), Copier template integration (Phase 32), managed hook policies (Phase 33), observability pipeline and session analytics (Phase 34), and agentic quality and learning features (Phase 34). Each unit exercises the feature's key behavioral contracts — security properties that must hold, performance targets that must be met, and output accuracy requirements that distinguish correct from incorrect behavior.

## Dependencies

Phase 17 complete (test infrastructure framework — testscript E2E framework, custom commands like `json_path`, `yaml_has`, golden file infrastructure, CI pipeline). Phase 30 complete (client profile encryption — sops+age key generation, profile schema, init-time pre-population, SecretSpec references, value baking, non-interactive mode). Phase 31 complete (profile compliance enforcement — security floors, required hooks, blocked MCPs, `gdev check --ci`, non-suppressible critical violations). Phase 32 complete (Copier template integration — `gdev init --from`, Join/Create mode detection, template update + gdev reconciliation). Phase 33 complete (managed hook policies — 3-tier deployment, destructive command prevention, credential scanning, cost alerting, SOC 2 logging, test enforcement, client isolation). Phase 34 complete (observability pipeline and session analytics — OTel sidecar, ccusage, analytics JSONL, team report, learning-opportunities, orient, project clarity, repo map, calm directives, time-to-first-env, pre-edit linter).

## Phase Outputs

- Client profile encryption round-trip test suite (age key generation, schema validation, sops encrypt/decrypt, SecretSpec references, Join mode value baking)
- Profile compliance enforcement test suite (security floor, required hooks, blocked MCPs, CI gate, non-suppressibility)
- Copier template E2E test suite (lifecycle, Join/Create detection, non-interactive mode, template update + gdev reconciliation)
- Hook policy enforcement test suite (destructive command prevention, credential scanning, cost alerting, SOC 2 logging, 3-tier deployment, performance)
- Observability pipeline validation test suite (OTel sidecar, ccusage, analytics metadata-only enforcement, team report)
- Agentic quality and learning validation test suite (learning-opportunities, orient, project clarity, calm directives, time-to-first-env, pre-edit linter)

---

### Unit 37.1: Client Profile Encryption Round-Trip Validation

**Description:** Validate the complete client profile encryption lifecycle: age key generation with correct permissions, profile creation with sops+age encryption, profile editing and re-encryption, profile deletion, init-time pre-population of wizard fields from non-secret values, SecretSpec generation with references (not values), value baking into `.gdev.yaml`, and non-interactive mode with `--client-profile`.

**Context:** Phase 30 introduced encrypted client profiles to solve the consulting firm's configuration management problem: each client engagement has non-secret configuration (AWS account IDs, project names, compliance levels) and secret references (API key names, credential patterns) that must be stored somewhere without committing secrets to git. The sops+age encryption scheme stores encrypted profiles at `~/.qsdev/profiles/<client>.enc.yaml`. The critical correctness invariant for SecretSpec generation is that references (variable names) appear in generated files, never values — a test that verifies `AWS_SECRET_ACCESS_KEY` does not appear as a value in any generated file is the primary control. Value baking moves non-secret values from the encrypted profile into `.gdev.yaml` so Join mode works for teammates without access to the encrypted profile.

**Desired Outcome:** A test suite verifying the complete profile lifecycle, that age key generation produces correctly permissioned files, that sops encryption is verifiable, that non-secret values appear in generated files while secrets appear only as references, and that non-interactive mode completes setup without prompts.

**Steps:**
1. Create `e2e/testdata/script/client-profiles/` directory for profile validation scripts.
2. Write `age-key-generation.txtar` — verify age key created with correct permissions:
   ```
   # gdev setup: creates age key at ~/.qsdev/keys/age.key with 0600 permissions
   env HOME=$WORK/home
   mkdir -p $WORK/home
   exec gdev setup --init-keys --yes
   exists $WORK/home/.qsdev/keys/age.key
   exec sh -c 'stat -c "%a" $HOME/.qsdev/keys/age.key'
   stdout '^600$'
   ```
3. Write `profile-create-encrypt.txtar` — verify profile creation produces encrypted file:
   ```
   # Client profile create: profile encrypted with sops+age
   env HOME=$WORK/home
   exec gdev setup --init-keys --yes
   exec gdev profile create acme-corp --compliance soc2 --aws-account 123456789012 --yes
   exists $WORK/home/.qsdev/profiles/acme-corp.enc.yaml
   # Encrypted file should not be plain text
   ! exec grep -l 'aws_account.*123456789012' $WORK/home/.qsdev/profiles/acme-corp.enc.yaml
   ```
4. Write `profile-schema-validation.txtar` — verify invalid compliance level rejected:
   ```
   # Profile schema validation: invalid compliance level rejected with clear error
   env HOME=$WORK/home
   exec gdev setup --init-keys --yes
   ! exec gdev profile create bad-corp --compliance invalid-level --yes
   stderr 'compliance\|invalid\|allowed\|soc2\|hipaa\|baseline'
   ```
5. Write `profile-edit-reencrypt.txtar` — verify editing a profile re-encrypts correctly:
   ```
   # Profile edit: changes re-encrypted, previous content not accessible
   env HOME=$WORK/home
   exec gdev setup --init-keys --yes
   exec gdev profile create acme-corp --compliance baseline --yes
   exec cp $WORK/home/.qsdev/profiles/acme-corp.enc.yaml $WORK/profile-v1.enc.yaml

   exec gdev profile edit acme-corp --compliance soc2 --yes
   # File should have changed (re-encrypted with updated content)
   ! cmp $WORK/home/.qsdev/profiles/acme-corp.enc.yaml $WORK/profile-v1.enc.yaml
   ```
6. Write `profile-delete.txtar` — verify profile deletion removes file:
   ```
   # Profile delete: removes encrypted profile file
   env HOME=$WORK/home
   exec gdev setup --init-keys --yes
   exec gdev profile create temp-client --compliance baseline --yes
   exists $WORK/home/.qsdev/profiles/temp-client.enc.yaml

   exec gdev profile delete temp-client --yes
   ! exists $WORK/home/.qsdev/profiles/temp-client.enc.yaml
   ```
7. Write `profile-init-prepopulation.txtar` — verify init pre-populates wizard from profile non-secret values:
   ```
   # Init-time pre-population: non-secret profile values pre-populate wizard answers
   env HOME=$WORK/home
   exec gdev setup --init-keys --yes
   exec gdev profile create acme-corp --compliance soc2 --aws-account 123456789012 --yes

   exec gdev init --non-interactive --client-profile acme-corp --yes
   exec cat .gdev.yaml
   exec grep 'acme-corp\|123456789012' .gdev.yaml
   # Compliance level from profile should appear
   exec grep 'soc2\|compliance' .gdev.yaml
   ```
8. Write `profile-secretspec-references.txtar` — verify SecretSpec contains references, not values:
   ```
   # SecretSpec: references appear in generated files, never credential values
   env HOME=$WORK/home
   exec gdev setup --init-keys --yes
   exec gdev profile create acme-corp --compliance soc2 --yes

   exec gdev init --non-interactive --client-profile acme-corp --yes
   # Verify SecretSpec file contains variable name references
   exists .gdev-secrets.yaml
   exec grep 'ref:\|name:\|source:' .gdev-secrets.yaml
   # Verify no actual secret values in any generated file
   ! exec grep -r 'sk-ant-api\|AKIAIO\|secret.*=.*[A-Za-z0-9+/]\{20\}' devenv.nix .gdev.yaml .gdev-secrets.yaml
   ```
9. Write `profile-value-baking.txtar` — verify non-secret values baked into .gdev.yaml for Join mode:
   ```
   # Value baking: non-secret values appear in .gdev.yaml client block for Join mode
   env HOME=$WORK/home
   exec gdev setup --init-keys --yes
   exec gdev profile create acme-corp --compliance soc2 --project-id my-gcp-project --yes

   exec gdev init --non-interactive --client-profile acme-corp --yes
   exec grep 'client:' .gdev.yaml
   exec grep 'project_id\|my-gcp-project' .gdev.yaml
   # Non-secret values baked in; teammate without encrypted profile can still run Join mode
   ```
10. Write `profile-non-interactive.txtar` — verify `--client-profile --yes` completes without prompts:
    ```
    # Non-interactive: gdev init --client-profile --yes completes without interaction
    env HOME=$WORK/home
    exec gdev setup --init-keys --yes
    exec gdev profile create acme-corp --compliance soc2 --yes

    # Should complete entirely without prompts
    exec gdev init --non-interactive --client-profile acme-corp --yes --answers-file answers.yaml
    exists .gdev.yaml
    exists devenv.nix

    -- answers.yaml --
    ecosystems: [go]
    quick_mode: true
    ```

**Acceptance Criteria:**
- [ ] `gdev setup --init-keys` creates age key at `~/.qsdev/keys/age.key` with `0600` permissions
- [ ] Profile creation produces an encrypted file at `~/.qsdev/profiles/<name>.enc.yaml`; plaintext values not readable without decryption key
- [ ] Invalid compliance level rejected with error message naming valid options
- [ ] Profile edit re-encrypts with updated content; resulting file differs from pre-edit version
- [ ] Profile deletion removes encrypted file from `~/.qsdev/profiles/`
- [ ] Init with `--client-profile` pre-populates wizard with non-secret profile values (account IDs, project names, compliance level)
- [ ] Generated `.gdev-secrets.yaml` contains variable name references, not credential values
- [ ] No actual credential values appear in devenv.nix, `.gdev.yaml`, or `.gdev-secrets.yaml`
- [ ] Non-secret profile values baked into `.gdev.yaml` client block, enabling Join mode without encrypted profile
- [ ] `gdev init --client-profile <name> --yes` completes without interactive prompts

**Research Citations:**
- `phases/30-client-profile-encryption.md § Unit 30.1` — age key generation, permissions, `~/.qsdev/keys/`
- `phases/30-client-profile-encryption.md § Unit 30.2` — sops+age encryption, profile create/edit/delete lifecycle
- `phases/30-client-profile-encryption.md § Unit 30.3` — profile schema, compliance level validation
- `phases/30-client-profile-encryption.md § Unit 30.4` — init-time pre-population, wizard integration
- `phases/30-client-profile-encryption.md § Unit 30.5` — SecretSpec generation, references-not-values invariant
- `phases/30-client-profile-encryption.md § Unit 30.6` — value baking, `.gdev.yaml` client block, Join mode compatibility
- `phases/30-client-profile-encryption.md § Unit 30.7` — non-interactive mode, `--client-profile --yes` flags

**Status:** Not Started

---

### Unit 37.2: Profile Compliance Enforcement Validation

**Description:** Validate Phase 31's compliance enforcement mechanisms: that a profile-mandated security floor cannot be manually downgraded, that enhanced profiles require all specified pre-commit hooks, that strict profiles block particular MCPs, that `gdev check --ci` exits non-zero on critical violations, and that critical violations are non-suppressible even with `--audit-level`.

**Context:** Phase 31 implements the compliance enforcement layer that makes client profiles meaningful at runtime rather than just configuration. Without enforcement, a developer could accept an enhanced security profile in `.gdev.yaml` and then manually remove the security tooling. The security floor check prevents downgrade by comparing the resolved security level against the profile's minimum. The required hook check prevents hook deletion by comparing the installed pre-commit hooks against the profile's required set. The blocked MCP check prevents high-risk MCP servers from being enabled on engagements where the client has prohibited them. Critical violations (floor enforcement failures, required hook missing) cannot be suppressed because they represent compliance breaches, not informational warnings.

**Desired Outcome:** A test suite proving that each enforcement mechanism detects and reports violations, that violations cause `gdev check --ci` to exit non-zero, and that critical violations cannot be bypassed regardless of flag combinations.

**Steps:**
1. Create `e2e/testdata/script/compliance-enforcement/` directory for enforcement test scripts.
2. Write `security-floor-violation.txtar` — verify downgrade attempt detected as critical violation:
   ```
   # Security floor: manual downgrade to baseline in .gdev.yaml triggers critical violation
   exec gdev init --non-interactive --answers-file answers-enhanced.yaml
   # Manually downgrade security level
   exec sed -i 's/level: enhanced/level: baseline/' .gdev.yaml

   exec gdev check --format json > check.json
   json_path check.json '.violations' 'some' '.severity=="critical"'
   json_path check.json '.violations[]|select(.severity=="critical")' '.category' 'contains' 'security'

   -- answers-enhanced.yaml --
   quick_mode: true
   security_profile: consulting-default
   ```
3. Write `security-floor-ci-failure.txtar` — verify `gdev check --ci` exits non-zero on floor violation:
   ```
   # gdev check --ci: exits non-zero on critical security floor violation
   env CI=true
   exec gdev init --non-interactive --answers-file answers-enhanced.yaml
   exec sed -i 's/level: enhanced/level: baseline/' .gdev.yaml

   ! exec gdev check --ci --format json
   stderr 'critical\|security.*floor\|violation'

   -- answers-enhanced.yaml --
   quick_mode: true
   security_profile: consulting-default
   ```
4. Write `required-hooks-violation.txtar` — verify missing required hooks trigger critical violation:
   ```
   # Required hooks: enhanced profile missing gitleaks hook triggers critical violation
   exec gdev init --non-interactive --answers-file answers.yaml

   # Remove gitleaks from pre-commit config
   exec sed -i '/gitleaks/d' .pre-commit-config.yaml

   exec gdev check --format json > check.json
   json_path check.json '.violations' 'some' '.severity=="critical"'
   json_path check.json '.violations[]|select(.category=="required-hooks")' '.detail' 'contains' 'gitleaks'

   -- answers.yaml --
   quick_mode: true
   security_profile: consulting-default
   ```
5. Write `blocked-mcp-violation.txtar` — verify blocked MCP server triggers violation:
   ```
   # Blocked MCP: strict profile blocks Slack MCP from being enabled
   env SLACK_BOT_TOKEN=xoxb-test-token
   exec gdev init --non-interactive --answers-file answers-strict.yaml

   # Attempt to enable Slack MCP (blocked by strict profile)
   ! exec gdev mcp enable mcp-slack
   stderr 'blocked\|profile\|strict\|not allowed'

   -- answers-strict.yaml --
   quick_mode: true
   security_profile: enterprise
   ```
6. Write `violation-non-suppressible.txtar` — verify critical violations cannot be bypassed with `--audit-level`:
   ```
   # Non-suppressible: critical violations cannot be bypassed with --audit-level
   exec gdev init --non-interactive --answers-file answers.yaml
   exec sed -i 's/level: enhanced/level: baseline/' .gdev.yaml

   # Even with --audit-level high (suppress only low/medium), critical violations persist
   ! exec gdev check --ci --audit-level high --format json
   stderr 'critical\|cannot.*suppress\|floor'

   # And with --audit-level critical, it should still fail (critical is still critical)
   ! exec gdev check --ci --audit-level critical --format json
   stderr 'violation\|critical'

   -- answers.yaml --
   quick_mode: true
   security_profile: consulting-default
   ```

**Acceptance Criteria:**
- [ ] Manual downgrade of `security.level` from enhanced to baseline detected as critical violation
- [ ] `gdev check --ci` exits non-zero when security floor violation present
- [ ] Removal of gitleaks hook from `.pre-commit-config.yaml` detected as critical violation when profile requires it
- [ ] Strict/enterprise profile blocks `gdev mcp enable mcp-slack` with actionable error message
- [ ] Critical violations remain in `gdev check --ci` output even with `--audit-level` flags set

**Research Citations:**
- `phases/31-profile-compliance-enforcement.md § Unit 31.1` — security floor check, downgrade detection
- `phases/31-profile-compliance-enforcement.md § Unit 31.2` — required hook validation, pre-commit config comparison
- `phases/31-profile-compliance-enforcement.md § Unit 31.3` — blocked MCP enforcement, profile-level MCP allowlists
- `phases/31-profile-compliance-enforcement.md § Unit 31.4` — `gdev check --ci` integration, exit code semantics
- `phases/31-profile-compliance-enforcement.md § Unit 31.5` — non-suppressible critical violations, `--audit-level` bypass prevention

**Status:** Not Started

---

### Unit 37.3: Copier Template E2E Validation

**Description:** Validate the full Copier template integration from Phase 32: add/list/remove template lifecycle, `gdev init --from <template>` runs Copier first then gdev overlays second, Join mode detection when template includes `.gdev.yaml`, Create mode when template does not include `.gdev.yaml`, `gdev update --template` coordination, and non-interactive mode with `--data answers.yaml --yes`.

**Context:** Phase 32 integrates Copier's project templating with gdev's environment setup so consulting projects can use a standard firm template and still get proper devenv configuration. The key sequencing is Copier first, gdev second: Copier generates project structure and may include a `.gdev.yaml`, after which gdev runs in the appropriate mode. If Copier included a `.gdev.yaml`, gdev detects this as Join mode (project config exists, skip wizard). If Copier did not include a `.gdev.yaml`, gdev runs Create mode (full wizard). The `gdev update --template` command coordinates both tools: Copier re-applies template changes and gdev reconciles any conflicts in the generated configs.

**Desired Outcome:** A test suite verifying that the Copier-then-gdev sequencing works correctly, that mode detection is accurate based on template content, that the non-interactive path completes fully, and that update coordination handles the Copier update then gdev reconcile sequence.

**Steps:**
1. Create `e2e/testdata/script/copier/` directory for Copier template test scripts.
2. Write `copier-add-list-remove.txtar` — verify template lifecycle management:
   ```
   # Template add/list/remove lifecycle
   exec gdev template add https://github.com/example/firm-template --alias firm-default --dry-run
   stdout 'firm-default\|added\|registered'

   exec gdev template list --format json > templates.json
   json_path templates.json '.templates' 'length' '>=0'

   exec gdev template remove firm-default --dry-run
   stdout 'removed\|unregistered'
   ```
3. Write `copier-init-no-gdev-yaml.txtar` — verify Create mode when template has no .gdev.yaml:
   ```
   # gdev init --from template without .gdev.yaml: Copier first, gdev Create mode second
   exec gdev init --from $WORK/template-no-config --non-interactive --answers-file answers.yaml

   # Verify Copier ran (creates template-specific files)
   exists src/main.go
   # Verify gdev ran in Create mode (generated devenv.nix without pre-existing .gdev.yaml)
   exists devenv.nix
   exists .gdev.yaml
   stdout 'Create mode\|creating.*config'

   -- answers.yaml --
   ecosystems: [go]
   quick_mode: true
   ```
4. Write `copier-init-with-gdev-yaml.txtar` — verify Join mode when template includes .gdev.yaml:
   ```
   # gdev init --from template with .gdev.yaml: Copier first, gdev Join mode second
   exec gdev init --from $WORK/template-with-config --non-interactive --answers-file answers.yaml

   # Verify Copier ran
   exists src/main.go
   # Verify gdev ran in Join mode (template's .gdev.yaml already existed)
   exists devenv.nix
   stdout 'Join mode\|joining.*existing'

   -- answers.yaml --
   quick_mode: true
   ```
5. Write `copier-update.txtar` — verify `gdev update --template` coordinates Copier update + gdev reconcile:
   ```
   # gdev update --template: Copier update then gdev reconciliation
   exec gdev init --from $WORK/template-v1 --non-interactive --answers-file answers.yaml

   # Update to v2 of template
   exec gdev update --template $WORK/template-v2 --non-interactive --yes
   stdout 'template\|update\|reconcil'
   # devenv.nix should reflect both template changes and preserved project-specific config
   exists devenv.nix

   -- answers.yaml --
   quick_mode: true
   ```
6. Write `copier-non-interactive.txtar` — verify `--data --yes` completes without interaction:
   ```
   # Non-interactive: gdev init --from template --data answers.yaml --yes
   exec gdev init --from $WORK/simple-template --data copier-answers.yaml --non-interactive --answers-file gdev-answers.yaml --yes
   exists devenv.nix
   exists .gdev.yaml

   -- copier-answers.yaml --
   project_name: my-project
   author: Test User

   -- gdev-answers.yaml --
   ecosystems: [go]
   quick_mode: true
   ```

**Acceptance Criteria:**
- [ ] `gdev template add` registers a template; `gdev template list` shows it; `gdev template remove` unregisters it
- [ ] `gdev init --from` without `.gdev.yaml` in template: Copier runs first generating project files, then gdev runs in Create mode (full wizard/answers)
- [ ] `gdev init --from` with `.gdev.yaml` in template: Copier runs first, gdev detects existing config and runs in Join mode (skips redundant questions)
- [ ] `gdev update --template` runs Copier update followed by gdev reconciliation; resulting devenv.nix is valid
- [ ] `gdev init --from <template> --data answers.yaml --yes` completes without interactive prompts

**Research Citations:**
- `phases/32-copier-template-integration.md § Unit 32.1` — template add/list/remove registry management
- `phases/32-copier-template-integration.md § Unit 32.2` — `gdev init --from` sequencing, Copier-first then gdev-second ordering
- `phases/32-copier-template-integration.md § Unit 32.3` — Join mode detection when template includes `.gdev.yaml`
- `phases/32-copier-template-integration.md § Unit 32.4` — Create mode when template omits `.gdev.yaml`
- `phases/32-copier-template-integration.md § Unit 32.5` — `gdev update --template`, Copier update + gdev reconciliation
- `phases/32-copier-template-integration.md § Unit 32.6` — non-interactive mode, `--data` flag, Copier answers file

**Status:** Not Started

---

### Unit 37.4: Hook Policy Enforcement Validation

**Description:** Validate all Phase 33 managed hook policies: destructive command prevention (blocks `rm -rf /`, allows `rm -rf ./build`), credential scanning (blocks file writes with credential patterns, allows normal code), cost alerting on session threshold, SOC 2 logging (metadata JSONL only, no content), test enforcement warning, client isolation check (wrong `AWS_PROFILE`), 3-tier hook deployment, and per-hook performance under 200ms.

**Context:** Phase 33 implements the "managed hooks" tier of gdev's three-tier Claude Code configuration: hooks that the consulting firm ships as policy, distinct from project-level hooks and user-level hooks. The destructive command prevention hook uses path analysis: absolute paths (`/`, `/home`, `/etc`) are blocked; relative paths (`./build`, `../dist`) are allowed. The credential scanning hook pattern-matches PreToolUse Write/Edit tool inputs against known credential formats (AWS key patterns, API key prefixes). The SOC 2 audit log must record session metadata (timestamps, tool types, file categories) but must never record file content — this is validated by checking that no field in the JSONL exceeds a short character limit that content would violate. The 3-tier deployment means managed hooks go in `~/.claude/settings.json`, project hooks go in `.claude/settings.json`, and they must not conflict.

**Desired Outcome:** A test suite proving destructive command prevention works with correct path analysis, credential scanning catches known patterns without blocking normal code, audit logging records only metadata, the 3-tier hierarchy is correctly deployed, and each hook completes within the 200ms performance budget.

**Steps:**
1. Create `e2e/testdata/script/hooks/` directory for hook policy test scripts.
2. Write `destructive-prevention-absolute.txtar` — verify absolute-path destructive commands blocked:
   ```
   # Destructive command prevention: rm -rf with absolute path blocked
   exec gdev init --non-interactive --answers-file answers.yaml
   env PATH=$WORK/mock-bin:$PATH

   # Test the hook script directly with the absolute-path rm command
   exec sh -c 'echo '"'"'{"tool_name":"Bash","tool_input":{"command":"rm -rf /"}}'"'"' | gdev hook --test destructive-prevention'
   stdout 'blocked\|denied\|2'

   -- answers.yaml --
   quick_mode: true
   ```
3. Write `destructive-prevention-relative-allowed.txtar` — verify relative-path commands allowed:
   ```
   # Destructive command prevention: rm -rf with relative path allowed
   exec gdev init --non-interactive --answers-file answers.yaml

   exec sh -c 'echo '"'"'{"tool_name":"Bash","tool_input":{"command":"rm -rf ./build"}}'"'"' | gdev hook --test destructive-prevention'
   stdout 'allowed\|0'

   exec sh -c 'echo '"'"'{"tool_name":"Bash","tool_input":{"command":"rm -rf ../dist"}}'"'"' | gdev hook --test destructive-prevention'
   stdout 'allowed\|0'

   -- answers.yaml --
   quick_mode: true
   ```
4. Write `credential-scanning-blocked.txtar` — verify credential patterns blocked in file writes:
   ```
   # Credential scanning: file write with AKIA AWS key pattern blocked
   exec gdev init --non-interactive --answers-file answers.yaml

   exec sh -c 'echo '"'"'{"tool_name":"Write","tool_input":{"file_path":"config.py","content":"api_key = \"AKIAIOSFODNN7EXAMPLE\""}}'"'"' | gdev hook --test credential-scanning'
   stdout 'blocked\|denied\|credential\|2'

   -- answers.yaml --
   quick_mode: true
   ```
5. Write `credential-scanning-normal-code.txtar` — verify normal code allowed through credential scanning:
   ```
   # Credential scanning: normal code without credential patterns allowed
   exec gdev init --non-interactive --answers-file answers.yaml

   exec sh -c 'echo '"'"'{"tool_name":"Write","tool_input":{"file_path":"main.go","content":"func main() { fmt.Println(\"hello\") }"}}'"'"' | gdev hook --test credential-scanning'
   stdout 'allowed\|0'

   -- answers.yaml --
   quick_mode: true
   ```
6. Write `soc2-logging-metadata-only.txtar` — verify SOC 2 audit JSONL contains no content fields:
   ```
   # SOC 2 logging: session audit JSONL has only metadata, no content
   env HOME=$WORK/home
   mkdir -p $WORK/home
   exec gdev init --non-interactive --answers-file answers.yaml

   # Simulate a session that writes a file (audit log entry created)
   exec sh -c 'echo '"'"'{"tool_name":"Write","tool_input":{"file_path":"test.go","content":"package main\nfunc main(){}"}}'"'"' | gdev hook --test soc2-audit'

   # Find the latest audit JSONL file
   exec sh -c 'find $HOME/.qsdev/audit -name "*.jsonl" -newer /tmp | head -1' > audit-file.txt
   exec sh -c 'cat $(cat audit-file.txt) | head -1' > latest-entry.txt

   # Verify no field exceeds 256 chars (content would be much longer)
   exec sh -c 'node -e "const e=JSON.parse(require(\"fs\").readFileSync(\"latest-entry.txt\",\"utf8\")); const vals=Object.values(e); const maxLen=Math.max(...vals.map(v=>String(v).length)); process.exit(maxLen<=256?0:1)"'

   -- answers.yaml --
   quick_mode: true
   ```
7. Write `test-enforcement-warning.txtar` — verify test failure triggers warning at session end:
   ```
   # Test enforcement: devenv task test failure produces warning at session end
   exec gdev init --non-interactive --answers-file answers.yaml

   # Simulate test failure hook entry
   exec sh -c 'echo '"'"'{"tool_name":"Bash","tool_input":{"command":"devenv task test"},"tool_result":{"exit_code":1}}'"'"' | gdev hook --test test-enforcement'
   stdout 'warning\|test.*failed\|tests'

   -- answers.yaml --
   quick_mode: true
   ```
8. Write `client-isolation-wrong-profile.txtar` — verify wrong AWS_PROFILE triggers warning:
   ```
   # Client isolation: wrong AWS_PROFILE triggers warning
   exec gdev init --non-interactive --answers-file answers.yaml
   exec sed -i 's/AWS_PROFILE = "TODO.*"/AWS_PROFILE = "acme-prod"/' devenv.nix

   # Simulate running with a different profile
   env AWS_PROFILE=other-client
   exec gdev check --client-isolation --format json > isolation.json
   json_path isolation.json '.warnings' 'some' '.category=="client-isolation"'

   -- answers.yaml --
   quick_mode: true
   ```
9. Write `three-tier-deployment.txtar` — verify managed hooks in user settings, project hooks in project settings:
   ```
   # 3-tier deployment: managed hooks in ~/.claude/settings.json, project in .claude/settings.json
   env HOME=$WORK/home
   mkdir -p $WORK/home/.claude
   exec gdev init --non-interactive --answers-file answers.yaml

   # Managed hooks should be in user settings
   exists $WORK/home/.claude/settings.json
   exec grep 'destructive-prevention\|soc2-audit' $WORK/home/.claude/settings.json

   # Project hooks in project settings
   exists .claude/settings.json

   -- answers.yaml --
   quick_mode: true
   ```
10. Write `hook-performance.txtar` — verify each hook completes under 200ms:
    ```
    # Hook performance: each managed hook completes in <200ms
    exec gdev init --non-interactive --answers-file answers.yaml

    # Time the destructive-prevention hook
    exec sh -c 'start=$(date +%s%N); echo '"'"'{"tool_name":"Bash","tool_input":{"command":"ls"}}'"'"' | gdev hook --test destructive-prevention > /dev/null; end=$(date +%s%N); ms=$(( (end - start) / 1000000 )); echo $ms; [ $ms -lt 200 ]'

    # Time the credential-scanning hook
    exec sh -c 'start=$(date +%s%N); echo '"'"'{"tool_name":"Write","tool_input":{"file_path":"f.go","content":"package main"}}'"'"' | gdev hook --test credential-scanning > /dev/null; end=$(date +%s%N); ms=$(( (end - start) / 1000000 )); echo $ms; [ $ms -lt 200 ]'

    -- answers.yaml --
    quick_mode: true
    ```

**Acceptance Criteria:**
- [ ] `rm -rf /` (absolute path) blocked by destructive-prevention hook with exit code 2
- [ ] `rm -rf ./build` (relative path) allowed by destructive-prevention hook with exit code 0
- [ ] File write containing `AKIAIOSFODNN7EXAMPLE` (AWS key pattern) blocked by credential-scanning hook
- [ ] File write containing normal Go/Python/TypeScript code allowed by credential-scanning hook
- [ ] SOC 2 audit JSONL: no field in any entry exceeds 256 characters (content fields absent)
- [ ] Test failure in `devenv task test` triggers a session-end warning message
- [ ] Wrong `AWS_PROFILE` in environment compared to project-expected profile triggers client-isolation warning
- [ ] Managed hooks present in `~/.claude/settings.json`; project hooks in `.claude/settings.json`
- [ ] Each managed hook (destructive-prevention, credential-scanning, soc2-audit) completes in under 200ms

**Research Citations:**
- `phases/33-managed-hook-policies.md § Unit 33.1` — destructive command prevention hook, path analysis logic
- `phases/33-managed-hook-policies.md § Unit 33.2` — credential scanning hook, AWS/API key pattern matching
- `phases/33-managed-hook-policies.md § Unit 33.3` — cost alerting hook, session threshold
- `phases/33-managed-hook-policies.md § Unit 33.4` — SOC 2 audit logging, metadata-only design, JSONL format
- `phases/33-managed-hook-policies.md § Unit 33.5` — test enforcement hook, session-end warning
- `phases/33-managed-hook-policies.md § Unit 33.6` — client isolation check, AWS_PROFILE comparison
- `phases/33-managed-hook-policies.md § Unit 33.7` — 3-tier hook deployment, user/project/managed separation
- `phases/33-managed-hook-policies.md § Unit 33.8` — hook performance budget, <200ms requirement

**Status:** Not Started

---

### Unit 37.5: Observability Pipeline Validation

**Description:** Validate Phase 34's observability and analytics features: OTel sidecar container starts when observability enabled with Grafana accessible on port 3000, OTEL env vars present/absent based on enabled state, `gdev observability up/down/status/logs` subcommands all function, `gdev cost` produces output from local session files, analytics JSONL metadata-only enforcement (no field over 256 chars), and team report CI artifact aggregation produces valid markdown dashboard.

**Context:** Phase 34's observability stack gives consulting teams visibility into their gdev-instrumented environments without requiring cloud accounts. The OTel sidecar runs as a devenv service, collecting traces and metrics and routing them to a local Grafana + Prometheus stack. The analytics system records session-level metadata (tools used, durations, ecosystem types) to a JSONL file that feeds the `gdev cost` and team report commands. The metadata-only invariant is the same as for SOC 2 logging: no content or file contents should appear in analytics records, only structural metadata. The team report aggregates per-session analytics files into a markdown dashboard for COP or engineering review — this is distinct from the Phase 15 per-project posture aggregation.

**Desired Outcome:** A test suite verifying the OTel sidecar lifecycle, correct OTEL env var presence, all `gdev observability` subcommands, cost reporting accuracy, analytics metadata-only enforcement, and team report generation from synthetic session files.

**Steps:**
1. Create `e2e/testdata/script/observability/` directory for observability validation scripts.
2. Write `otel-enable-disable.txtar` — verify OTel sidecar env vars toggled by enable/disable:
   ```
   # OTEL env vars: present when enabled, absent when disabled
   exec gdev init --non-interactive --answers-file answers.yaml

   exec gdev enable observability
   exec grep 'OTEL_EXPORTER_OTLP_ENDPOINT\|OTEL_SERVICE_NAME' devenv.nix

   exec gdev disable observability
   ! exec grep 'OTEL_EXPORTER_OTLP_ENDPOINT' devenv.nix

   -- answers.yaml --
   quick_mode: true
   ```
3. Write `observability-subcommands.txtar` — verify all `gdev observability` subcommands produce output:
   ```
   # gdev observability subcommands: up/down/status/logs all functional
   exec gdev init --non-interactive --answers-file answers.yaml
   exec gdev enable observability

   exec gdev observability status --format json > status.json
   json_path status.json '.enabled' 'true'
   json_path status.json '.services' 'length' '>=1'

   exec gdev observability down --dry-run
   stdout 'stop\|down\|observability'

   -- answers.yaml --
   quick_mode: true
   ```
4. Write `gdev-cost-output.txtar` — verify `gdev cost` produces output from local session files:
   ```
   # gdev cost: produces output from local session JSONL files
   env HOME=$WORK/home
   mkdir -p $WORK/home/.claude/projects
   cp session-log.jsonl $WORK/home/.claude/projects/test-session.jsonl

   exec gdev init --non-interactive --answers-file answers.yaml
   exec gdev cost 2>&1
   stdout 'cost\|session\|tokens\|usage'

   -- answers.yaml --
   quick_mode: true

   -- session-log.jsonl --
   {"type":"session_start","timestamp":"2026-05-14T10:00:00Z","model":"claude-sonnet-4-5"}
   {"type":"tool_use","tool":"Read","duration_ms":45,"timestamp":"2026-05-14T10:00:01Z"}
   {"type":"session_end","timestamp":"2026-05-14T10:00:30Z","total_input_tokens":1500,"total_output_tokens":300}
   ```
5. Write `analytics-metadata-only.txtar` — verify analytics JSONL has no long-form content fields:
   ```
   # Analytics metadata-only: no field in analytics JSONL exceeds 256 chars
   env HOME=$WORK/home
   mkdir -p $WORK/home/.qsdev/analytics
   exec gdev init --non-interactive --answers-file answers.yaml

   # Trigger some analytics events
   exec gdev status > /dev/null
   exec gdev doctor > /dev/null

   # Find latest analytics file
   exec sh -c 'find $HOME/.qsdev/analytics -name "*.jsonl" | head -1' > analytics-file.txt
   exec sh -c 'cat $(cat analytics-file.txt) | head -5' > sample-entries.txt

   # Verify no field exceeds 256 chars
   exec sh -c 'node -e "const fs=require(\"fs\"); const lines=fs.readFileSync(\"sample-entries.txt\",\"utf8\").split(\"\\n\").filter(Boolean); for(const l of lines){ const e=JSON.parse(l); for(const [k,v] of Object.entries(e)){ if(String(v).length>256){ process.stderr.write(k+\"=\"+String(v).length); process.exit(1); } } }"'

   -- answers.yaml --
   quick_mode: true
   ```
6. Write `team-report-generation.txtar` — verify team report aggregates session files into markdown:
   ```
   # Team report: aggregates per-session analytics into markdown dashboard
   exec gdev init --non-interactive --answers-file answers.yaml

   # Create synthetic session analytics files
   mkdir analytics
   cp session-a.jsonl analytics/
   cp session-b.jsonl analytics/

   exec gdev analytics report --input analytics/ --format md > report.md
   exec grep '## Summary' report.md
   exec grep 'Total sessions\|Sessions' report.md
   exec grep 'Average\|Avg' report.md

   -- answers.yaml --
   quick_mode: true

   -- session-a.jsonl --
   {"session_id":"abc123","project":"client-a","ecosystem":"go","duration_s":120,"tools_used":15,"timestamp":"2026-05-14T10:00:00Z"}

   -- session-b.jsonl --
   {"session_id":"def456","project":"client-b","ecosystem":"typescript","duration_s":240,"tools_used":32,"timestamp":"2026-05-14T11:00:00Z"}
   ```

**Acceptance Criteria:**
- [ ] `gdev enable observability` adds OTEL env vars to devenv.nix; `gdev disable observability` removes them
- [ ] `gdev observability status --format json` reports enabled state and service list
- [ ] `gdev observability down --dry-run` produces output describing what would be stopped
- [ ] `gdev cost` produces formatted output showing session usage from local JSONL files
- [ ] Analytics JSONL: no field in any entry exceeds 256 characters (content absent, only metadata)
- [ ] `gdev analytics report --input <dir> --format md` produces markdown with summary section, session count, and averages

**Research Citations:**
- `phases/34-observability-analytics.md § Unit 34.1` — OTel sidecar, `gdev enable/disable observability`, OTEL env var generation
- `phases/34-observability-analytics.md § Unit 34.2` — `gdev observability up/down/status/logs` subcommands
- `phases/34-observability-analytics.md § Unit 34.3` — ccusage integration, `gdev cost` command, local JSONL session files
- `phases/34-observability-analytics.md § Unit 34.4` — analytics JSONL format, metadata-only design, 256-char field limit
- `phases/34-observability-analytics.md § Unit 34.5` — team report generation, analytics aggregation, markdown dashboard

**Status:** Not Started

---

### Unit 37.6: Agentic Quality & Learning Validation

**Description:** Validate Phase 34's agentic quality and learning features: `gdev enable learning-opportunities` deploys skill files to `.claude/skills/`, `gdev enable orient` deploys the orient skill and generates `orientation.md`, CLAUDE.md has a Project Context section populated from Copier questionnaire data, `/repo-map` generates a ~1K token structural overview, all generated CLAUDE.md and rules files contain no ALL-CAPS or aggressive emphasis, `gdev info --timing` displays time-to-first-env breakdown, and the pre-edit linter runs as an advisory warning without blocking.

**Context:** Phase 34's agentic quality features are the final layer of gdev's Claude Code integration: they improve the quality of AI assistance by providing orientation skills, project context, and repository structure information that Claude otherwise lacks. The calm directives requirement addresses a consistent problem with AI-generated CLAUDE.md files: they tend to use ALL-CAPS, exclamation marks, and aggressive emphasis ("NEVER DO THIS", "ALWAYS REQUIRED!") that creates an adversarial tone. All generated instruction files must use calm, declarative language. The pre-edit linter is advisory — it warns when Claude is about to edit a file that appears not to exist or has syntax issues, but does not block the edit, because false blocking is more harmful than false non-blocking.

**Desired Outcome:** A test suite verifying that learning-opportunities and orient skill deployment works, that orientation.md is generated, that CLAUDE.md Project Context is populated from Copier data, that repo-map output is under token budget, that all generated instruction files use calm language, that timing data is recorded and reported, and that the pre-edit linter warns but does not block.

**Steps:**
1. Create `e2e/testdata/script/agentic-quality/` directory for agentic quality test scripts.
2. Write `learning-opportunities-deploy.txtar` — verify skill files deployed on enable:
   ```
   # gdev enable learning-opportunities: skill files deployed to .claude/skills/
   exec gdev init --non-interactive --answers-file answers.yaml
   ! exists .claude/skills/learning-opportunities

   exec gdev enable learning-opportunities
   exists .claude/skills/learning-opportunities/SKILL.md
   exec grep 'learning\|opportunity\|improve' .claude/skills/learning-opportunities/SKILL.md

   -- answers.yaml --
   quick_mode: true
   ```
3. Write `orient-skill-deploy.txtar` — verify orient skill deployed and invocable:
   ```
   # gdev enable orient: orient skill deployed; /orient invocable
   exec gdev init --non-interactive --answers-file answers.yaml
   exec gdev enable orient
   exists .claude/skills/orient/SKILL.md

   # Verify skill can be described (not invoked, just that it parses)
   exec grep 'name:.*orient\|orient' .claude/skills/orient/SKILL.md

   -- answers.yaml --
   quick_mode: true
   ```
4. Write `orient-generates-orientation-md.txtar` — verify orient skill generates orientation.md:
   ```
   # orient skill: generates orientation.md when invoked
   exec gdev init --non-interactive --answers-file answers.yaml
   exec gdev enable orient

   # Simulate orient invocation (non-interactive mode)
   exec gdev orient --generate --yes
   exists orientation.md
   exec grep 'Project\|Setup\|How to\|Getting started' orientation.md

   -- answers.yaml --
   quick_mode: true

   -- go.mod --
   module example.com/test
   go 1.22
   ```
5. Write `claude-md-project-context.txtar` — verify CLAUDE.md has Project Context section:
   ```
   # CLAUDE.md: Project Context section populated
   exec gdev init --non-interactive --answers-file answers.yaml
   exec grep 'Project Context\|## Project\|project_name\|project_description' CLAUDE.md

   -- answers.yaml --
   quick_mode: true
   project_name: my-consulting-project
   project_description: "Client project for acme-corp"
   ```
6. Write `repo-map-token-budget.txtar` — verify repo-map output within ~1K token budget:
   ```
   # /repo-map: generates ~1K token structural overview
   exec gdev init --non-interactive --answers-file answers.yaml
   exec gdev enable repo-map

   exec gdev repo-map --output repo-map.txt
   # 1K tokens ~= 4K characters; verify output is under budget
   exec sh -c 'wc -c < repo-map.txt'
   stdout '^[0-9]\{1,4\}$'  # under 9999 chars

   -- answers.yaml --
   quick_mode: true

   -- go.mod --
   module example.com/test
   go 1.22
   ```
7. Write `calm-directives-audit.txtar` — verify no ALL-CAPS or aggressive emphasis in generated files:
   ```
   # Calm directives: no ALL-CAPS blocks, no exclamation marks, no aggressive emphasis
   exec gdev init --non-interactive --answers-file answers.yaml

   # Check CLAUDE.md and all generated rules files
   exec sh -c 'find . -name "CLAUDE.md" -o -path ".claude/rules/*.md" | xargs grep -l "[A-Z]\{5,\}" || true' > allcaps-files.txt
   exec sh -c '[ ! -s allcaps-files.txt ]'

   exec sh -c 'find . -name "CLAUDE.md" -o -path ".claude/rules/*.md" | xargs grep -l "!" || true' > exclamation-files.txt
   exec sh -c '[ ! -s exclamation-files.txt ]'

   -- answers.yaml --
   ecosystems: [go]
   quick_mode: true
   ```
8. Write `time-to-first-env-timing.txtar` — verify timing recorded and reported by `gdev info --timing`:
   ```
   # Time-to-first-env: gdev info --timing shows breakdown
   exec gdev init --non-interactive --answers-file answers.yaml
   exec gdev info --timing --format json > timing.json
   json_path timing.json '.detection_ms' '>=0'
   json_path timing.json '.generation_ms' '>=0'
   json_path timing.json '.total_ms' '>=0'

   -- answers.yaml --
   quick_mode: true
   ```
9. Write `pre-edit-linter-advisory.txtar` — verify pre-edit linter warns but does not block:
   ```
   # Pre-edit linter: advisory warning, NOT blocking
   exec gdev init --non-interactive --answers-file answers.yaml

   # Linter on a valid file: no warning, exit 0
   exec gdev lint --pre-edit devenv.nix
   # Should succeed (valid Nix file)

   # Linter on a file with syntax issues: warning shown, still exit 0
   exec gdev lint --pre-edit bad-file.go
   # Should still exit 0 (advisory, not blocking)
   stdout 'warning\|syntax\|issue'

   -- answers.yaml --
   quick_mode: true

   -- bad-file.go --
   package main

   func main() {
     fmt.Println("unclosed
   ```

**Acceptance Criteria:**
- [ ] `gdev enable learning-opportunities` deploys skill files to `.claude/skills/learning-opportunities/SKILL.md`
- [ ] `gdev enable orient` deploys orient skill; `SKILL.md` contains orient skill definition
- [ ] `gdev orient --generate --yes` generates `orientation.md` with project setup information
- [ ] CLAUDE.md contains a Project Context section populated with project name/description from init answers
- [ ] `gdev repo-map` output is under 4K characters (~1K tokens) for a simple Go project
- [ ] All generated CLAUDE.md files contain no sequence of 5+ consecutive uppercase letters
- [ ] All generated CLAUDE.md files contain no exclamation marks
- [ ] `gdev info --timing --format json` reports `detection_ms`, `generation_ms`, and `total_ms` fields
- [ ] `gdev lint --pre-edit` on a syntactically invalid file exits 0 (advisory warning, not blocking)

**Research Citations:**
- `phases/34-observability-analytics.md § Unit 34.6` — `gdev enable learning-opportunities`, skill file deployment
- `phases/34-observability-analytics.md § Unit 34.7` — `gdev enable orient`, orient skill, `orientation.md` generation
- `phases/34-observability-analytics.md § Unit 34.8` — CLAUDE.md Project Context section, Copier questionnaire integration
- `phases/34-observability-analytics.md § Unit 34.9` — repo-map generation, ~1K token budget
- `phases/34-observability-analytics.md § Unit 34.10` — calm directives requirement, ALL-CAPS and exclamation prohibition
- `phases/34-observability-analytics.md § Unit 34.11` — time-to-first-env metric, `gdev info --timing` command
- `phases/34-observability-analytics.md § Unit 34.12` — pre-edit linter, advisory-only design, exit code 0

**Status:** Not Started

---

## Phase Completion Criteria

- [ ] All six units pass acceptance criteria
- [ ] Client profiles: age key created at `~/.qsdev/keys/age.key` with 0600 permissions; full encrypt/decrypt round-trip verified
- [ ] Client profiles: no credential secret values appear in any generated file (devenv.nix, `.gdev.yaml`, SecretSpec)
- [ ] Client profiles: non-secret values baked into `.gdev.yaml` for teammate Join mode compatibility
- [ ] Compliance enforcement: security floor downgrade detected as critical violation; `gdev check --ci` exits non-zero
- [ ] Compliance enforcement: critical violations non-suppressible with `--audit-level` flags
- [ ] Copier integration: Create mode when template lacks `.gdev.yaml`; Join mode when template includes it
- [ ] Hooks: `rm -rf /` blocked; `rm -rf ./build` allowed; AWS key pattern in file write blocked; normal code allowed
- [ ] Hooks: SOC 2 JSONL has no field over 256 chars; managed hooks in user settings, project hooks in project settings
- [ ] Hooks: each managed hook completes under 200ms
- [ ] Observability: OTEL env vars present when enabled, absent when disabled
- [ ] Analytics: no field in analytics JSONL exceeds 256 characters
- [ ] Agentic quality: learning-opportunities and orient skills deploy on enable
- [ ] Agentic quality: all generated CLAUDE.md files free of ALL-CAPS sequences and exclamation marks
- [ ] Agentic quality: pre-edit linter exits 0 (advisory) even on files with syntax issues
- [ ] All tests run successfully in the Phase 17 CI pipeline (quick-validation and nightly matrix)
