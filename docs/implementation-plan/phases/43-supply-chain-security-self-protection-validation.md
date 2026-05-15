# Phase 43: Supply Chain, Security & Self-Protection Validation

## Goal

Validate all supply chain attestation (Phase 38), native security patterns (Phase 39), and agent self-protection (Phases 40-42) features through E2E testing using the testscript framework and CI pipeline from Phase 17. Each unit exercises the feature's key behavioral contracts -- security properties that must hold under adversarial conditions, performance targets that must be met for developer experience, and correctness invariants that distinguish working from broken behavior. All tests use safe test fixtures and simulated attack vectors -- no real malware, no real credential exposure.

## Dependencies

Phase 17 complete (test infrastructure framework -- testscript E2E framework, custom commands like `json_path`, `yaml_has`, golden file infrastructure, CI pipeline). Phase 38 complete (SBOM generation & supply chain attestation -- Syft dual-format SBOMs, cosign keyless signing, govulncheck OpenVEX, GitHub attestations, Nix distribution integrity). Phase 39 complete (native security pattern library -- policy engine core, package risk assessment, security analysis pipeline, MCP trust scoring). Phase 40 complete (agent self-protection Tier 1 -- rule definition schema, path canonicalization, fail-closed harness, 18 Tier 1 rules, PreToolUse Bash hook). Phase 41 complete (agent self-protection bypass & monitor -- three-tier bypass system, per-rule monitor mode, Tier 2 rules, session state, PreToolUse Write/Edit hook). Phase 42 complete (Go binary consolidation -- unified `gdev-hook` binary, rule consolidation, performance optimization, Agent/MCP rules, migration).

## Phase Outputs

- SBOM pipeline E2E test suite (GoReleaser snapshot, dual-format validation, cosign round-trip, OpenVEX generation, embedded SBOM display, Nix SRI verification)
- Policy engine and risk assessment test suite (built-in rulesets, custom YAML rules, severity-tiered evaluation, package risk scoring, SARIF output validation, MCP trust scoring)
- Tier 1 self-protection test suite (18 rules with positive + negative controls, path canonicalization evasion tests, fail-closed harness error handling, performance benchmarks)
- Bypass and monitor mode test suite (enforce_always non-bypassability, session bypass lifecycle, command bypass, monitor mode logging, audit trail completeness, Tier 2 rule coverage)
- Go binary consolidation test suite (unified binary subcommands, Phase 32 regression, performance benchmarks, migration and rollback, cross-phase coordination)
- Cross-phase integration test suite (cosign identity consistency, legitimate operation passthrough, health reporting integration, audit trail format compatibility, aggregated security status)

---

### Unit 43.1: SBOM Pipeline E2E Validation

**Description:** Validate the complete SBOM generation and supply chain attestation pipeline from Phase 38: GoReleaser snapshot builds produce dual-format SBOMs, SBOM content accurately reflects the Go module dependency graph, cosign keyless signing completes a sign-verify-tamper-verify cycle, govulncheck generates OpenVEX documents with reachability-based suppression, `gdev version --sbom` displays embedded SBOM data, `gdev self-update` verifies SBOM signatures, release artifacts are complete, and Nix flake builds verify SRI hashes against signed releases.

**Context:** Phase 38 adds five units of supply chain infrastructure: Syft SBOM generation integrated into GoReleaser (38.1), cosign keyless signing with SLSA attestation (38.2), govulncheck with OpenVEX (38.3), consumer verification tooling including `gdev version --sbom` and `gdev self-update` verification (38.4), and Nix distribution with SRI hash integrity (38.5). The validation strategy exercises the end-to-end pipeline rather than individual components -- a GoReleaser snapshot build is the primary test vehicle because it produces real artifacts without requiring a git tag or GitHub release. The cosign round-trip test is the critical security property: if signing verifies but a tampered artifact also verifies, the entire supply chain is compromised.

**Desired Outcome:** A test suite proving that every release artifact is produced, correctly formatted, cryptographically signed, and verifiable -- and that tampering is reliably detected. The embedded SBOM matches the build's dependency graph and is accessible via CLI.

**Steps:**
1. Write `sbom-goreleaser-snapshot.txtar` -- run `goreleaser build --snapshot --clean` and verify both SBOM formats are produced:
   ```
   # GoReleaser snapshot produces dual-format SBOMs
   exec goreleaser build --snapshot --clean
   exists dist/
   # SPDX 2.3 format produced
   exec find dist -name '*.spdx.json' -type f
   stdout '\.spdx\.json'
   # CycloneDX 1.5 format produced
   exec find dist -name '*.cdx.json' -type f
   stdout '\.cdx\.json'
   ```
2. Write `sbom-content-accuracy.txtar` -- parse a generated CycloneDX SBOM and verify its component list matches `go.sum` entries. Extract component names from the SBOM JSON and compare against `go list -m all` output. Key packages (cobra, viper, sops, etc.) must appear in the SBOM.
3. Write `cosign-sign-verify-roundtrip.txtar` -- generate a test artifact, sign it with cosign keyless (or a local key pair for CI), verify the signature succeeds, tamper with the artifact (append a byte), and verify the tampered artifact fails verification:
   ```
   # Cosign round-trip: sign -> verify -> tamper -> verify-fail
   exec sh -c 'echo "test-artifact-content" > artifact.bin'
   exec cosign sign-blob --key cosign.key --output-signature artifact.sig artifact.bin
   exec cosign verify-blob --key cosign.pub --signature artifact.sig artifact.bin
   # Tamper with the artifact
   exec sh -c 'echo "x" >> artifact.bin'
   ! exec cosign verify-blob --key cosign.pub --signature artifact.sig artifact.bin
   stderr 'invalid signature\|verification failed\|FAIL'
   ```
4. Write `openvex-generation.txtar` -- run govulncheck against a fixture with a known-reachable vulnerability and a known-unreachable vulnerability. Verify the generated OpenVEX document marks the unreachable vulnerability as `not_affected` with justification `vulnerable_code_not_in_execute_path`, while the reachable vulnerability retains its `affected` status.
5. Write `gdev-version-sbom.txtar` -- verify `gdev version --sbom` outputs valid CycloneDX JSON to stdout:
   ```
   # gdev version --sbom displays embedded SBOM
   exec gdev version --sbom
   stdout '"bomFormat"'
   stdout '"CycloneDX"'
   # Pipe through jq to validate JSON structure
   exec sh -c 'gdev version --sbom | jq .bomFormat'
   stdout 'CycloneDX'
   ```
6. Write `self-update-sbom-verify.txtar` -- simulate `gdev self-update` against a local HTTP server serving release artifacts with valid signatures (positive case) and with a tampered checksum file (negative case). Verify that valid signatures proceed and tampered signatures abort the update.
7. Write `release-artifact-completeness.txtar` -- after a GoReleaser snapshot, verify the full artifact set is present: binary archives, SPDX SBOMs, CycloneDX SBOMs, cosign signature bundles, OpenVEX documents, and checksum file.
8. Write `nix-sri-hash-verification.txtar` -- verify that the Nix flake's `fetchurl` SRI hash matches the SHA256 of the actual release binary. Compute the SRI hash from a snapshot binary and compare against what `nix flake show` or the flake.nix specifies.

**Acceptance Criteria:**
- [ ] GoReleaser snapshot produces both SPDX 2.3 and CycloneDX 1.5 SBOM files
- [ ] CycloneDX SBOM component list matches `go list -m all` output for key dependencies
- [ ] Cosign sign-verify succeeds on unmodified artifact
- [ ] Cosign verify fails on tampered artifact (appended byte)
- [ ] govulncheck OpenVEX marks unreachable vulnerabilities as `not_affected` with correct justification
- [ ] `gdev version --sbom` outputs valid CycloneDX JSON parseable by jq
- [ ] `gdev self-update` proceeds with valid SBOM signatures, aborts with tampered signatures
- [ ] Release snapshot contains: binaries + SBOMs (both formats) + signatures + VEX + checksums
- [ ] Nix SRI hash computed from release binary matches flake.nix specification

**Research Citations:**
- `phases/38-sbom-supply-chain-attestation.md § Unit 38.1` -- Syft SBOM generation, dual-format configuration
- `phases/38-sbom-supply-chain-attestation.md § Unit 38.2` -- cosign keyless signing, SLSA attestation
- `phases/38-sbom-supply-chain-attestation.md § Unit 38.3` -- govulncheck OpenVEX generation, reachability-based suppression
- `phases/38-sbom-supply-chain-attestation.md § Unit 38.4` -- `gdev version --sbom`, `gdev self-update` verification, consumer toolchain
- `phases/38-sbom-supply-chain-attestation.md § Unit 38.5` -- Nix distribution with SRI hash integrity

**Status:** Not Started

---

### Unit 43.2: Policy Engine & Risk Assessment Validation

**Description:** Validate Phase 39's native security pattern library: the policy engine evaluates built-in rulesets correctly (default, strict, permissive), custom YAML rules override built-in rules, severity-tiered evaluation produces correct verdicts, package risk assessment scores known-malicious patterns high, publication age checks flag recent packages, SARIF output validates against the schema, integration with `gdev status --security` produces correct output, and MCP trust scoring rates high-permission servers lower than read-only servers.

**Context:** Phase 39 implements five units of security primitives: a YAML-based policy engine (39.1), package risk assessment (39.2), a security analysis pipeline (39.3), MCP server trust assessment (39.4), and optional tool integration points (39.5). The policy engine is the most critical component -- it replaces hardcoded shell patterns from Phase 32 with data-driven rules that users can customize. The key correctness invariant is the escalation model: when multiple rules match, deny overrides ask overrides allow. The package risk assessment validates that metadata signals (publication age, maintainer count, download volume) produce meaningful risk differentiation between known-good and known-bad packages. SARIF output enables integration with GitHub code scanning and other SAST consumers.

**Desired Outcome:** A test suite proving the policy engine evaluates rules correctly across all three built-in rulesets, that custom rules extend and override built-ins, that the escalation model holds under multi-rule matches, that package risk scoring produces distinguishable scores for safe vs. suspicious packages, and that all machine-readable outputs (SARIF, JSON) conform to their schemas.

**Steps:**
1. Write `policy-engine-default-ruleset.txtar` -- initialize a project with default security policy, evaluate a set of operations, and verify correct verdicts:
   ```
   # Default policy: credential access denied, normal operations allowed
   exec gdev init --non-interactive --yes
   exec gdev policy check --operation 'Bash: cat ~/.aws/credentials' --format json > check.json
   json_path check.json '.verdict' 'deny'
   exec gdev policy check --operation 'Bash: go build ./...' --format json > check2.json
   json_path check2.json '.verdict' 'allow'
   ```
2. Write `policy-engine-strict-ruleset.txtar` -- set `security.policy: strict` in `.gdev.yaml`, verify that additional rules (deny network operations, deny dotfile writes, deny MCP config modifications) are active beyond the default set.
3. Write `policy-engine-permissive-ruleset.txtar` -- set `security.policy: permissive`, verify that only credential exfiltration and destructive infrastructure commands are denied while other operations are allowed.
4. Write `policy-engine-custom-rules.txtar` -- create a custom YAML rule file that adds a new rule and overrides a built-in rule's severity, verify the custom rule is evaluated and the override takes effect:
   ```
   # Custom rules extend and override built-in rules
   exec gdev init --non-interactive --yes
   cp custom-rules.yaml .gdev-security-rules.yaml
   exec gdev policy list --format json > rules.json
   # Custom rule appears in active rule list
   json_path rules.json '.rules' 'some' '.id=="CUSTOM-001"'
   # Overridden rule has new severity
   json_path rules.json '.rules[]|select(.id=="SPR-001")' '.severity' 'high'

   -- custom-rules.yaml --
   rules:
     - id: CUSTOM-001
       name: "Block access to production database"
       severity: critical
       verdict: deny
       match:
         patterns: ["DATABASE_URL.*prod"]
     - id: SPR-001
       severity: high  # Override built-in severity from critical to high
   ```
5. Write `policy-engine-escalation.txtar` -- create a scenario where multiple rules match with different verdicts (one allow, one deny), verify that deny wins:
   ```
   # Escalation: deny overrides allow when multiple rules match
   exec gdev policy check --operation 'Write: ~/.config/special-file' --format json > check.json
   json_path check.json '.verdict' 'deny'
   json_path check.json '.matched_rules' 'length' '2'
   ```
6. Write `package-risk-scoring.txtar` -- evaluate packages using test fixtures representing known-malicious package characteristics (low download count, single maintainer, very recent publication) and known-good packages (high downloads, many maintainers, years old). Verify malicious-profile packages score significantly higher risk than established packages.
7. Write `publication-age-check.txtar` -- create a test fixture with a package metadata response showing a publication date less than 7 days ago. Verify the age check flags it as a risk factor. Verify a package published years ago is not flagged.
8. Write `sarif-output-validation.txtar` -- run `gdev policy check` with `--format sarif`, capture the output, and validate it against the SARIF 2.1.0 schema using a JSON Schema validator:
   ```
   # SARIF output validates against SARIF 2.1.0 schema
   exec gdev policy check --lockfile go.sum --format sarif > results.sarif
   # Validate required SARIF fields
   exec sh -c 'jq -e ".version == \"2.1.0\"" results.sarif'
   exec sh -c 'jq -e ".runs | length > 0" results.sarif'
   exec sh -c 'jq -e ".runs[0].tool.driver.name" results.sarif'
   ```
9. Write `gdev-status-security-integration.txtar` -- run `gdev status --security --format json`, verify the output includes policy engine evaluation results with rule counts and verdict summary.
10. Write `mcp-trust-scoring.txtar` -- configure two MCP servers in the test fixture: one read-only server (e.g., filesystem read) and one high-permission server (e.g., full shell access). Verify the high-permission server receives a lower trust score than the read-only server:
    ```
    # MCP trust scoring: high-permission servers score lower
    exec gdev mcp trust-score filesystem-read --format json > read.json
    exec gdev mcp trust-score shell-executor --format json > shell.json
    # Read-only server should have higher trust score
    exec sh -c 'read_score=$(jq .trust_score read.json); shell_score=$(jq .trust_score shell.json); test "$read_score" -gt "$shell_score"'
    ```

**Acceptance Criteria:**
- [ ] Default policy ruleset: credential access denied, normal build operations allowed
- [ ] Strict policy ruleset: network operations, dotfile writes, and MCP config modifications additionally denied
- [ ] Permissive policy ruleset: only credential exfiltration and destructive commands denied
- [ ] Custom YAML rules appear in active rule list and override built-in rule properties
- [ ] Escalation model: deny overrides allow when multiple rules match the same operation
- [ ] Package risk scoring: malicious-profile packages score measurably higher than established packages
- [ ] Publication age check: packages published < 7 days ago flagged; packages published years ago not flagged
- [ ] SARIF output validates: `version == "2.1.0"`, `runs` array non-empty, `tool.driver.name` present
- [ ] `gdev status --security` includes policy engine evaluation results (rule counts, verdict summary)
- [ ] MCP trust scoring: high-permission servers receive lower trust scores than read-only servers

**Research Citations:**
- `phases/39-native-security-pattern-library.md § Unit 39.1` -- policy engine core, YAML rules, three built-in rulesets, escalation model, shadow mode
- `phases/39-native-security-pattern-library.md § Unit 39.2` -- package risk assessment, publication age signal, metadata-based scoring
- `phases/39-native-security-pattern-library.md § Unit 39.3` -- security analysis pipeline, SARIF 2.1.0 output
- `phases/39-native-security-pattern-library.md § Unit 39.4` -- MCP server trust assessment, permission-weighted scoring
- `phases/39-native-security-pattern-library.md § Unit 39.5` -- optional tool integration points

**Status:** Not Started

---

### Unit 43.3: Self-Protection Tier 1 Rules Validation

**Description:** Validate all 18 Tier 1 self-protection rules from Phase 40: each rule triggers on its documented attack vector, path canonicalization defeats evasion attempts (symlinks, `../` traversal, `/proc/self/fd/` indirection), the fail-closed harness blocks on internal errors, exit code 2 is used exclusively for denials, negative controls confirm legitimate operations are not blocked, and all Tier 1 rules evaluate within the 50ms performance budget.

**Context:** Phase 40 deploys the foundational self-protection layer: 18 rules with `enforce_always: true` that protect the root of trust -- settings.json hook registrations, hook scripts, audit trail, bypass mechanisms, and the gdev CLI itself. The rules are organized into 3 logical groups across 2 physical matchers: 13 Bash matcher rules (SP-01, SP-02, SP-06 through SP-13, MCP-03, MCP-04) and 6 Write/Edit matcher rules (SP-03, SP-04, SP-05, SP-14, MCP-01, MCP-02; note SP-04 covers Write/Edit while SP-06 covers the Bash vector for the same files). The critical property is that path canonicalization closes the evasion vectors: an agent cannot bypass SP-03 by using `~/../home/user/.qsdev/` or `/proc/self/fd/<fd>/.qsdev/` because all paths are resolved to their canonical form before matching.

**Desired Outcome:** Every Tier 1 rule has at least one positive control (blocks the attack vector) and one negative control (allows the legitimate operation). Path evasion attempts via symlinks, traversal, and `/proc/self/fd/` are all defeated. Internal hook errors produce exit code 2 (fail-closed), never exit code 0.

**Steps:**
1. Write `tier1-sp03-deny-writes-qsdev.txtar` -- verify SP-03 blocks Write/Edit to paths under `~/.qsdev/`:
   ```
   # SP-03: blocks writes to gdev install directory
   env HOME=$WORK/home
   mkdir -p $WORK/home/.qsdev/hooks
   ! exec gdev-hook pre-write <<EOF
   {"tool_name":"Write","tool_input":{"file_path":"$WORK/home/.qsdev/hooks/my-hook.sh","content":"#!/bin/bash\necho pwned"}}
   EOF
   stderr 'SP-03'
   stderr 'gdev installation directory'
   ```
2. Write `tier1-sp04-deny-writes-settings.txtar` -- verify SP-04 blocks Write/Edit to `~/.claude/settings.json` and `settings.local.json` but allows project-level `.claude/settings.json`.
3. Write `tier1-sp06-deny-bash-settings.txtar` -- verify SP-06 blocks Bash commands that write to settings.json via various methods (sed -i, cat >, tee, cp, mv, echo >, perl -pi, python -c, awk >, dd of=):
   ```
   # SP-06: blocks Bash writes to settings.json via sed
   ! exec gdev-hook pre-bash <<EOF
   {"tool_name":"Bash","tool_input":{"command":"sed -i 's/hooks//' ~/.claude/settings.json"}}
   EOF
   stderr 'SP-06'
   ```
4. Write `tier1-sp08-deny-bypass-export.txtar` -- verify SP-08 blocks `export GDEV_HOOK_BYPASS`, `export GDEV_BYPASS_*`, and `export GDEV_SELF_PROTECTION`.
5. Write `tier1-sp09-deny-bypass-next.txtar` -- verify SP-09 blocks `gdev hook bypass-next` when invoked via the Bash tool.
6. Write `tier1-sp12-deny-obfuscation.txtar` -- verify SP-12 blocks `base64 -d | bash`, `base64 --decode | sh`, and similar pipe-to-interpreter patterns.
7. Write `tier1-mcp01-deny-temp-dir.txtar` -- verify MCP-01 blocks `.mcp.json` writes containing `/tmp/`, `/dev/shm/`, or `/var/tmp/` paths.
8. Write `tier1-mcp04-deny-ioc-domain.txtar` -- verify MCP-04 blocks `npm install`/`pip install` commands referencing IOC domains (pastebin.com, transfer.sh, etc.).
9. Write `tier1-remaining-rules.txtar` -- batch test for SP-01, SP-02, SP-05, SP-07, SP-10, SP-11, SP-13, SP-14, MCP-02, MCP-03, each with one positive and one negative control.
10. Write `path-canonicalization-evasion.txtar` -- attempt to bypass SP-03 and SP-04 using:
    - Symlink: create symlink to `~/.qsdev/` and write via the symlink path
    - Traversal: use `~/../home/<user>/.qsdev/` as the file path
    - `/proc/self/fd/`: open a file descriptor to a protected directory and reference via `/proc/self/fd/<fd>/`
    - Verify all three evasion attempts are blocked.
11. Write `fail-closed-harness.txtar` -- feed malformed JSON to `gdev-hook pre-bash` and verify it returns exit code 2 (block), not exit code 0 (allow):
    ```
    # Fail-closed: malformed input -> exit code 2
    ! exec gdev-hook pre-bash <<EOF
    {invalid json
    EOF
    # Exit code should be 2 (block), not 1 (error) or 0 (allow)
    ```
12. Write `exit-code-2-exclusive.txtar` -- verify that all denial responses use exit code 2 and never produce a JSON `permissionDecision` response on stdout (which has open bugs #39344, #52822).
13. Write `tier1-negative-controls.txtar` -- verify legitimate operations are NOT blocked:
    - `rm -rf ./build` (relative path, not protected)
    - `cat ~/.bashrc` (read, not write to protected path)
    - `gdev init` / `gdev enable hooks` (gdev commands via legitimate invocation)
    - `Write: file_path="./src/main.go"` (project file, not protected path)
    - `Edit: file_path=".claude/settings.json"` (project-level settings, not user-level)
14. Write `tier1-performance-benchmark.txtar` -- time the evaluation of all 18 Tier 1 rules against a single input and verify total evaluation completes in under 50ms:
    ```
    # Performance: all Tier 1 rules evaluate in < 50ms
    exec sh -c 'start=$(date +%s%N); for i in $(seq 1 100); do echo "{\"tool_name\":\"Bash\",\"tool_input\":{\"command\":\"ls -la\"}}" | gdev-hook pre-bash 2>/dev/null; done; end=$(date +%s%N); elapsed=$(( (end - start) / 100 / 1000000 )); test $elapsed -lt 50'
    ```

**Acceptance Criteria:**
- [ ] All 18 Tier 1 rules trigger (exit 2) on their documented attack vectors
- [ ] SP-03: blocks Write/Edit to any path under `~/.qsdev/` including via symlinks
- [ ] SP-04: blocks Write/Edit to `~/.claude/settings.json` but allows project `.claude/settings.json`
- [ ] SP-06: blocks Bash writes to settings.json via sed, cat, tee, cp, mv, echo, perl, python, awk, dd
- [ ] SP-12: blocks base64-decode-to-interpreter obfuscation patterns
- [ ] Path canonicalization defeats symlink evasion, `../` traversal, and `/proc/self/fd/` indirection
- [ ] Fail-closed harness: malformed JSON input produces exit code 2 (block)
- [ ] All denials use exit code 2 exclusively; no JSON permissionDecision output on stdout
- [ ] Negative controls: `rm -rf ./build`, `cat ~/.bashrc`, project-level `.claude/settings.json` writes all pass (exit 0)
- [ ] All 18 Tier 1 rules evaluate in < 50ms per invocation (averaged over 100 iterations)

**Research Citations:**
- `phases/40-agent-self-protection-tier1-harness.md § Unit 40.1` -- rule definition schema, YAML configuration
- `phases/40-agent-self-protection-tier1-harness.md § Unit 40.2` -- path canonicalization, symlink resolution, evasion vectors
- `phases/40-agent-self-protection-tier1-harness.md § Unit 40.3` -- fail-closed hook harness, exit code 2 exclusive use
- `phases/40-agent-self-protection-tier1-harness.md § Unit 40.4` -- 18 Tier 1 rules, rule-by-rule matching logic
- `phases/40-agent-self-protection-tier1-harness.md § Unit 40.5` -- PreToolUse Bash hook, command parsing

**Status:** Not Started

---

### Unit 43.4: Self-Protection Bypass & Monitor Mode Validation

**Description:** Validate Phase 41's interactive bypass system, per-rule monitor mode, Tier 2 security rules, and session state management: `enforce_always` rules cannot be bypassed, session bypasses persist for the session duration and expire after, command-level bypasses work for Tier 2/3 rules, monitor mode logs violations without blocking, `enforce_always` rules cannot be set to monitor mode, all bypass events are logged to the audit JSONL, and Tier 2 rules cover their documented scenarios.

**Context:** Phase 41 implements the escape hatch layer that makes self-protection practical for daily use. Without bypasses, Tier 2 rules would block legitimate operations like editing `devenv.nix` or modifying `.pre-commit-config.yaml`. The bypass system is designed to be agent-resistant: `gdev hook bypass-next` is itself blocked by Tier 1 rule SP-09, forcing the developer to run it in their own terminal. Session bypasses persist via a session token file that the hook reads but cannot create. Monitor mode allows calibration of new rules before enforcement. The critical security invariant is that `enforce_always` rules (all 18 Tier 1 rules) are immune to bypass and cannot be placed in monitor mode.

**Desired Outcome:** A test suite proving that the bypass hierarchy is airtight (Tier 1 cannot be bypassed), that bypass state management works correctly (session scope, expiry), that monitor mode enables non-blocking rule calibration, and that all bypass events produce audit trail entries for SOC 2 evidence.

**Steps:**
1. Write `enforce-always-no-bypass.txtar` -- attempt to bypass a Tier 1 rule via the session bypass mechanism and verify it is still blocked, with a warning logged:
   ```
   # enforce_always: Tier 1 rules cannot be bypassed
   env HOME=$WORK/home
   mkdir -p $WORK/home/.qsdev/state
   # Simulate a bypass token for SP-03
   exec sh -c 'echo "SP-03" > $HOME/.qsdev/state/bypass-session.token'
   # SP-03 should still block despite bypass token
   ! exec gdev-hook pre-write <<EOF
   {"tool_name":"Write","tool_input":{"file_path":"$WORK/home/.qsdev/hooks/evil.sh","content":"pwned"}}
   EOF
   stderr 'SP-03'
   stderr 'cannot be bypassed\|enforce_always'
   ```
2. Write `session-bypass-lifecycle.txtar` -- create a session bypass for a Tier 2 rule, verify the operation is allowed, then expire the session (delete or invalidate the token) and verify the operation is blocked again:
   ```
   # Session bypass: persists for session, expires after
   env HOME=$WORK/home
   exec gdev hook bypass-next --rule SP-16 --session
   # Operation now allowed (Tier 2 rule bypassed for session)
   exec gdev-hook pre-write <<EOF
   {"tool_name":"Write","tool_input":{"file_path":"devenv.nix","content":"{ pkgs, ... }: { }"}}
   EOF
   # Expire the session
   exec gdev hook bypass-clear
   # Operation now blocked again
   ! exec gdev-hook pre-write <<EOF
   {"tool_name":"Write","tool_input":{"file_path":"devenv.nix","content":"{ pkgs, ... }: { }"}}
   EOF
   stderr 'SP-16'
   ```
3. Write `command-bypass.txtar` -- use a `# gdev-allow-<rule-id>` comment in a command to bypass a Tier 2/3 rule for a single invocation:
   ```
   # Command bypass: inline override for Tier 2/3 rules
   exec gdev-hook pre-bash <<EOF
   {"tool_name":"Bash","tool_input":{"command":"# gdev-allow-SP-16\ncat devenv.nix"}}
   EOF
   # Same command without the override should be evaluated normally
   ```
4. Write `monitor-mode-logging.txtar` -- set a rule to monitor mode, trigger it, and verify the operation is allowed (exit 0) but the violation is logged:
   ```
   # Monitor mode: violation logged, operation allowed
   env HOME=$WORK/home
   env GDEV_RULE_MODE_SP_16=monitor
   exec gdev-hook pre-write <<EOF
   {"tool_name":"Write","tool_input":{"file_path":"devenv.nix","content":"{ pkgs, ... }: { }"}}
   EOF
   # Check audit log for the monitored violation
   exec grep 'SP-16' $HOME/.qsdev/audit/hook-events.jsonl
   stdout 'monitor'
   ```
5. Write `enforce-always-no-monitor.txtar` -- attempt to set a Tier 1 (`enforce_always`) rule to monitor mode and verify it remains in enforce mode:
   ```
   # enforce_always rules cannot be set to monitor mode
   env GDEV_RULE_MODE_SP_03=monitor
   ! exec gdev-hook pre-write <<EOF
   {"tool_name":"Write","tool_input":{"file_path":"$WORK/home/.qsdev/hooks/evil.sh","content":"pwned"}}
   EOF
   stderr 'SP-03'
   ```
6. Write `bypass-audit-trail.txtar` -- exercise both session and command bypasses, then verify the audit JSONL contains entries for each bypass event with the rule ID, bypass type, timestamp, and outcome:
   ```
   # Bypass audit trail: all bypass events logged
   env HOME=$WORK/home
   exec gdev hook bypass-next --rule SP-16 --session
   exec gdev-hook pre-write <<EOF
   {"tool_name":"Write","tool_input":{"file_path":"devenv.nix","content":"{ }"}}
   EOF
   exec sh -c 'jq -e "select(.rule_id==\"SP-16\" and .bypass_type==\"session\")" $HOME/.qsdev/audit/hook-events.jsonl'
   ```
7. Write `tier2-rule-coverage.txtar` -- test Tier 2 rules for their documented scenarios: large file writes (> configurable threshold), unusual port bindings in Docker configurations, privileged Docker container operations, and sensitive file path reads. Each rule gets one positive control (triggers the rule) and one negative control (legitimate variant passes).

**Acceptance Criteria:**
- [ ] `enforce_always` rules (Tier 1): bypass attempt still blocked, warning logged about bypass attempt
- [ ] Session bypass: Tier 2 rule bypassed during session, blocked again after session expiry/clear
- [ ] Command bypass: `# gdev-allow-<rule-id>` inline comment overrides Tier 2/3 rules for single invocation
- [ ] Monitor mode: rule violation logged to audit JSONL but operation allowed (exit 0)
- [ ] `enforce_always` rules cannot be placed in monitor mode (environment variable override ignored)
- [ ] Bypass audit trail: every bypass event logged with rule ID, bypass type, timestamp, and outcome
- [ ] Tier 2 rules: large file writes, unusual ports, privileged Docker, sensitive path reads all tested with positive and negative controls

**Research Citations:**
- `phases/41-agent-self-protection-bypass-monitor.md § Unit 41.1` -- three-tier bypass system, agent-resistant design
- `phases/41-agent-self-protection-bypass-monitor.md § Unit 41.2` -- per-rule monitor mode, shadow-mode calibration
- `phases/41-agent-self-protection-bypass-monitor.md § Unit 41.3` -- Tier 2 security rules, documented scenarios
- `phases/41-agent-self-protection-bypass-monitor.md § Unit 41.4` -- session state management, token lifecycle
- `phases/41-agent-self-protection-bypass-monitor.md § Unit 41.5` -- PreToolUse Write/Edit hook integration

**Status:** Not Started

---

### Unit 43.5: Go Binary Consolidation Validation

**Description:** Validate Phase 42's unified `gdev-hook` binary: it handles all three subcommands (`pre-bash`, `pre-write`, `pre-agent`), all Phase 32 consulting hook patterns are preserved after consolidation (no regression), all 32+ rules evaluate within the 50ms performance budget, `gdev upgrade hooks` migrates from Phase 32 scripts to the unified binary, rollback restores Phase 32 scripts, self-protection rules do not duplicate Phase 32 credential patterns, and Agent/MCP rules check Phase 28 registry trust scores.

**Context:** Phase 42 consolidates all hook scripts into a single compiled Go binary, eliminating shell/Python interpreter dependencies and enabling tree-sitter-bash AST analysis. The critical correctness invariant is zero regression: every Phase 32 hook pattern (destructive command prevention, credential scanning, cost alerting, SOC 2 logging, test enforcement, client isolation) must produce identical verdicts after consolidation. The performance target is aggressive -- all rules (Phase 32 patterns + Phase 40-41 self-protection rules, 32+ total) must evaluate in under 50ms to avoid perceptible latency in the AI agent's tool-use loop.

**Desired Outcome:** A test suite proving the unified binary is a drop-in replacement for Phase 32 scripts with identical behavior, that performance meets the 50ms target, that migration and rollback work reliably, and that cross-phase rule coordination avoids duplicate detection.

**Steps:**
1. Write `unified-binary-subcommands.txtar` -- verify `gdev-hook` accepts all three subcommands and processes stdin JSON correctly:
   ```
   # gdev-hook handles all three subcommands
   exec gdev-hook pre-bash <<EOF
   {"tool_name":"Bash","tool_input":{"command":"ls -la"}}
   EOF
   exec gdev-hook pre-write <<EOF
   {"tool_name":"Write","tool_input":{"file_path":"./src/main.go","content":"package main"}}
   EOF
   exec gdev-hook pre-agent <<EOF
   {"tool_name":"Task","tool_input":{"prompt":"Help me understand the codebase"}}
   EOF
   ```
2. Write `phase32-regression-destructive.txtar` -- verify Phase 32 destructive command prevention patterns are preserved: `rm -rf /` blocked, `rm -rf ./build` allowed, `chmod 777 /etc/shadow` blocked, `chmod 644 ./README.md` allowed.
3. Write `phase32-regression-credentials.txtar` -- verify Phase 32 credential scanning patterns are preserved: file writes containing AWS access key patterns blocked, file writes containing normal code allowed.
4. Write `phase32-regression-cost.txtar` -- verify Phase 32 cost alerting patterns are preserved for commands that could incur unexpected cloud costs.
5. Write `phase32-regression-soc2-logging.txtar` -- verify that hook evaluations produce SOC 2 compatible JSONL audit entries with the Phase 32 schema (no field exceeds 256 characters).
6. Write `consolidation-performance.txtar` -- benchmark all 32+ rules evaluating against a single input and verify total time is under 50ms:
   ```
   # Performance: all 32+ rules evaluate in < 50ms
   exec sh -c 'start=$(date +%s%N); for i in $(seq 1 100); do echo "{\"tool_name\":\"Bash\",\"tool_input\":{\"command\":\"npm install suspicious-package\"}}" | gdev-hook pre-bash 2>/dev/null; done; end=$(date +%s%N); avg=$(( (end - start) / 100 / 1000000 )); echo "Average: ${avg}ms"; test $avg -lt 50'
   ```
7. Write `migration-upgrade.txtar` -- start with Phase 32 shell script hooks registered in `settings.json`, run `gdev upgrade hooks`, verify settings.json now points to the unified binary and old scripts are preserved (not deleted):
   ```
   # Migration: Phase 32 scripts -> unified binary
   env HOME=$WORK/home
   # Set up Phase 32 hook configuration
   mkdir -p $WORK/home/.claude
   cp phase32-settings.json $WORK/home/.claude/settings.json
   mkdir -p $WORK/home/.qsdev/hooks
   cp phase32-hooks/* $WORK/home/.qsdev/hooks/
   exec gdev upgrade hooks
   # settings.json now references gdev-hook binary
   exec grep 'gdev-hook' $WORK/home/.claude/settings.json
   # Old scripts preserved
   exists $WORK/home/.qsdev/hooks/destructive-prevention.sh
   ```
8. Write `migration-rollback.txtar` -- run `gdev upgrade hooks` then `gdev upgrade hooks --rollback`, verify settings.json is restored to Phase 32 script references:
   ```
   # Rollback: restores Phase 32 scripts
   exec gdev upgrade hooks
   exec gdev upgrade hooks --rollback
   # settings.json restored to Phase 32 script references
   ! exec grep 'gdev-hook' $WORK/home/.claude/settings.json
   exec grep 'destructive-prevention.sh' $WORK/home/.claude/settings.json
   ```
9. Write `no-duplicate-credential-patterns.txtar` -- verify that the consolidated rule set does not double-fire on credential patterns (Phase 32 credential scanning and Phase 40 self-protection should not both trigger for the same input, or if they do, the output is deduplicated):
   ```
   # No duplicate detections: credential patterns fire once
   exec sh -c 'echo "{\"tool_name\":\"Write\",\"tool_input\":{\"file_path\":\"config.yaml\",\"content\":\"aws_secret_access_key: AKIAIOSFODNN7EXAMPLE\"}}" | gdev-hook pre-write 2>stderr.txt; true'
   # Count rule violations -- should not have duplicates for same pattern
   exec sh -c 'grep -c "BLOCKED" stderr.txt'
   stdout '^1$'
   ```
10. Write `mcp-trust-integration.txtar` -- verify Agent/MCP rules check Phase 28 registry trust scores: an unregistered MCP server is blocked, a registered server with sufficient trust is allowed, a registered server with low trust is blocked:
    ```
    # MCP trust integration: registry + trust score enforcement
    ! exec gdev-hook pre-agent <<EOF
    {"tool_name":"mcp__unknown_server__some_tool","tool_input":{"query":"test"}}
    EOF
    stderr 'not registered\|MCP-UNREGISTERED'
    ```

**Acceptance Criteria:**
- [ ] `gdev-hook` accepts `pre-bash`, `pre-write`, and `pre-agent` subcommands with correct stdin JSON processing
- [ ] Phase 32 patterns preserved: `rm -rf /` blocked, `rm -rf ./build` allowed (destructive command prevention)
- [ ] Phase 32 patterns preserved: AWS key pattern in file write blocked, normal code allowed (credential scanning)
- [ ] Phase 32 patterns preserved: SOC 2 JSONL entries produced with no field exceeding 256 characters
- [ ] All 32+ rules evaluate in < 50ms per invocation (averaged over 100 iterations)
- [ ] `gdev upgrade hooks` migrates settings.json to unified binary; old scripts preserved
- [ ] `gdev upgrade hooks --rollback` restores settings.json to Phase 32 script references
- [ ] Credential patterns do not double-fire between Phase 32 and Phase 40 rule sets
- [ ] Agent/MCP rules validate against Phase 28 registry (unregistered servers blocked, low-trust servers blocked)

**Research Citations:**
- `phases/42-agent-self-protection-go-consolidation.md § Unit 42.1` -- unified hook binary architecture, three subcommands
- `phases/42-agent-self-protection-go-consolidation.md § Unit 42.2` -- rule consolidation, deduplication
- `phases/42-agent-self-protection-go-consolidation.md § Unit 42.3` -- performance optimization, 50ms target, pre-compiled regex
- `phases/42-agent-self-protection-go-consolidation.md § Unit 42.4` -- Agent/MCP rules, Phase 28 registry integration, trust scores
- `phases/42-agent-self-protection-go-consolidation.md § Unit 42.5` -- migration, backward compatibility, rollback
- `phases/32-managed-hook-policy-consulting-enforcement.md` -- Phase 32 hook patterns (destructive command, credentials, cost, SOC 2)

**Status:** Not Started

---

### Unit 43.6: Cross-Phase Integration Validation

**Description:** Validate that supply chain, security, and self-protection features integrate correctly with existing gdev infrastructure: SBOM signing uses the same cosign identity as Phase 10's release pipeline, self-protection rules do not block legitimate gdev operations, policy engine findings feed into Phase 15 health/status reporting, self-protection audit events conform to Phase 32's SOC 2 audit trail format, and `gdev status --security` aggregates data from all three subsystems.

**Context:** Phases 38-42 add significant new infrastructure that must interoperate with existing gdev features. The highest-risk integration point is self-protection blocking legitimate gdev operations -- a false positive on `gdev init` or `gdev enable hooks` would break the tool's core functionality. The second-highest risk is audit trail format divergence: Phase 32 established a JSONL schema for SOC 2 compliance, and Phase 40-41's self-protection events must conform to the same schema to avoid breaking compliance tooling. The aggregated `gdev status --security` view must coherently present data from three separate subsystems (policy engine score, self-protection status, SBOM state) without contradictions.

**Desired Outcome:** A test suite proving that all cross-phase integration points work correctly: no false positives on legitimate gdev commands, consistent cosign identity across pipelines, audit trail schema compatibility, and a coherent aggregated security status view.

**Steps:**
1. Write `cosign-identity-consistency.txtar` -- verify that SBOM signing (Phase 38) and release artifact signing (Phase 10) use the same cosign certificate identity and OIDC issuer. Extract the identity from both signing configurations and compare:
   ```
   # Cosign identity: SBOM signing matches release pipeline
   exec grep -r 'certificate-identity\|certificate-oidc-issuer' .goreleaser.yaml > goreleaser-identity.txt
   exec grep -r 'certificate-identity\|certificate-oidc-issuer' .github/workflows/release.yml > release-identity.txt
   # Both should reference the same GitHub Actions workflow identity
   exec diff <(sort goreleaser-identity.txt) <(sort release-identity.txt)
   ```
2. Write `legitimate-gdev-operations.txtar` -- run a comprehensive set of legitimate gdev operations with self-protection hooks active and verify none are blocked:
   ```
   # Self-protection does not block legitimate gdev operations
   exec gdev init --non-interactive --yes
   exec gdev enable hooks
   exec gdev enable pre-commit
   exec gdev status
   exec gdev status --security
   exec gdev policy list
   exec gdev version
   exec gdev version --sbom
   # All commands should complete without self-protection blocks
   ```
3. Write `policy-engine-health-integration.txtar` -- run the policy engine against a project with known findings, then verify `gdev status --health` (Phase 15) includes the policy engine score and finding count:
   ```
   # Policy findings feed into health reporting
   exec gdev init --non-interactive --yes
   exec gdev status --health --format json > health.json
   json_path health.json '.security.policy_engine' 'exists'
   json_path health.json '.security.policy_engine.rule_count' 'gt' '0'
   ```
4. Write `audit-trail-format-compatibility.txtar` -- trigger a self-protection rule violation and verify the resulting audit JSONL entry conforms to Phase 32's SOC 2 schema: required fields present (timestamp, event_type, rule_id, tool_name, verdict, session_id), no field exceeds 256 characters, timestamp is ISO 8601:
   ```
   # Audit trail: self-protection events match Phase 32 SOC 2 schema
   env HOME=$WORK/home
   ! exec gdev-hook pre-bash <<EOF
   {"tool_name":"Bash","tool_input":{"command":"rm -rf /"}}
   EOF
   exec sh -c 'tail -1 $HOME/.qsdev/audit/hook-events.jsonl | jq -e ".timestamp and .event_type and .rule_id and .tool_name and .verdict and .session_id"'
   # No field exceeds 256 characters
   exec sh -c 'tail -1 $HOME/.qsdev/audit/hook-events.jsonl | jq -r "to_entries[].value | tostring | length" | while read len; do test "$len" -le 256; done'
   ```
5. Write `aggregated-security-status.txtar` -- run `gdev status --security --format json` and verify the output aggregates all three subsystems without contradictions:
   ```
   # Aggregated security status: policy engine + self-protection + SBOM
   exec gdev init --non-interactive --yes
   exec gdev status --security --format json > security.json
   # Policy engine section present
   json_path security.json '.policy_engine' 'exists'
   json_path security.json '.policy_engine.active_ruleset' 'not_empty'
   # Self-protection section present
   json_path security.json '.self_protection' 'exists'
   json_path security.json '.self_protection.tier1_rules_active' 'eq' '18'
   json_path security.json '.self_protection.mode' 'enforce'
   # SBOM section present
   json_path security.json '.sbom' 'exists'
   json_path security.json '.sbom.format' 'CycloneDX'
   ```
6. Write `self-protection-audit-soc2-compatible.txtar` -- generate multiple self-protection events (block, bypass, monitor), then validate the complete audit log can be parsed by the Phase 32 SOC 2 reporting pipeline without errors. Verify event_type values are consistent with Phase 32's taxonomy.

**Acceptance Criteria:**
- [ ] Cosign certificate identity and OIDC issuer identical between SBOM signing (Phase 38) and release signing (Phase 10)
- [ ] Self-protection rules do not block: `gdev init`, `gdev enable hooks`, `gdev enable pre-commit`, `gdev status`, `gdev policy list`, `gdev version`, `gdev version --sbom`
- [ ] Policy engine findings appear in `gdev status --health` output (Phase 15 integration)
- [ ] Self-protection audit JSONL entries contain all Phase 32 SOC 2 required fields (timestamp, event_type, rule_id, tool_name, verdict, session_id)
- [ ] No audit trail field exceeds 256 characters (Phase 32 SOC 2 constraint)
- [ ] `gdev status --security` aggregates: policy engine (active ruleset, finding count) + self-protection (Tier 1 rule count, mode) + SBOM (format, signature status)
- [ ] Self-protection audit events parseable by Phase 32 SOC 2 reporting pipeline without errors

**Research Citations:**
- `phases/38-sbom-supply-chain-attestation.md § Unit 38.2` -- cosign keyless signing, certificate identity
- `phases/10-distribution-self-bootstrapping.md` -- release pipeline, signing configuration
- `phases/15-health-status-compliance-reporting.md` -- health/status reporting, security posture model
- `phases/32-managed-hook-policy-consulting-enforcement.md` -- SOC 2 JSONL audit trail schema, 256-char field limit
- `phases/40-agent-self-protection-tier1-harness.md § Unit 40.4` -- Tier 1 rule list, exit code 2
- `phases/41-agent-self-protection-bypass-monitor.md § Unit 41.4` -- session state, audit logging

**Status:** Not Started

---

## Phase Completion Criteria

- [ ] All six units pass acceptance criteria
- [ ] SBOM pipeline: GoReleaser snapshot produces dual-format SBOMs; cosign sign-verify succeeds and tamper detection works
- [ ] SBOM pipeline: `gdev version --sbom` outputs valid CycloneDX JSON; `gdev self-update` verifies signatures
- [ ] Policy engine: all three built-in rulesets evaluate correctly; custom YAML rules override built-ins; deny > ask > allow escalation holds
- [ ] Policy engine: package risk scoring differentiates safe from suspicious packages; SARIF 2.1.0 output validates
- [ ] Self-protection Tier 1: all 18 rules trigger on attack vectors and pass negative controls
- [ ] Self-protection Tier 1: path canonicalization defeats symlink, traversal, and `/proc/self/fd/` evasion
- [ ] Self-protection Tier 1: fail-closed harness returns exit code 2 on malformed input; all denials use exit code 2 exclusively
- [ ] Bypass & monitor: `enforce_always` rules immune to bypass and cannot be placed in monitor mode
- [ ] Bypass & monitor: session and command bypasses work for Tier 2/3 rules; all bypass events logged to audit JSONL
- [ ] Consolidation: `gdev-hook` unified binary handles all three subcommands; all Phase 32 patterns preserved (zero regression)
- [ ] Consolidation: all 32+ rules evaluate in < 50ms; `gdev upgrade hooks` migration and rollback both work
- [ ] Cross-phase: self-protection does not block legitimate gdev operations (init, enable, status, version)
- [ ] Cross-phase: audit events conform to Phase 32 SOC 2 schema; `gdev status --security` aggregates all three subsystems
- [ ] All tests run successfully in the Phase 17 CI pipeline (quick-validation and nightly matrix)
