# Tasks: Devenv.sh Security Boilerplate

## Phase 1: Scoping & Initial Research

### Pending

### Active

### Completed
- [x] **P1-T1: Devenv.sh architecture & internals** — How devenv.sh works under the hood: relationship to Nix, flakes, nixpkgs, cachix; what it controls at install/build/run time
  - Priority: high
  - Estimate: medium
  - Started: 2026-05-12
  - Completed: 2026-05-12
  - Outcome: success
  - Notes: Deep architectural analysis covering all 10 research questions. Report at `architecture-research.md`. 15+ primary sources saved to docs/.

- [x] **P1-T2: Devenv.sh security surface** — What attack vectors exist in a devenv.sh-managed environment: package sources, pre/post-install hooks, binary caches, plugin system, shell hooks
  - Priority: high
  - Estimate: medium
  - Started: 2026-05-12
  - Completed: 2026-05-12
  - Outcome: success
  - Notes: Complete threat model mapping 10 attack vectors with 25+ sub-vectors. Report at `security-surface-research.md`. 20 primary sources saved to docs/.

- [x] **P1-T3: Existing Nix security mechanisms** — What Nix already provides: sandboxed builds, content-addressed derivations, signature verification, restricted eval mode, and how devenv.sh interacts with these
  - Priority: high
  - Estimate: medium
  - Started: 2026-05-12
  - Completed: 2026-05-12
  - Outcome: success
  - Notes: 9 Nix security mechanisms analyzed with devenv interaction for each. Report at `nix-security-mechanisms-research.md`. 13 primary sources saved to docs/.

- [x] **P1-T4: Devenv.sh configuration options inventory** — Complete survey of devenv.yaml/nix configuration knobs relevant to security
  - Priority: high
  - Estimate: medium
  - Started: 2026-05-12
  - Completed: 2026-05-12
  - Outcome: success
  - Notes: Comprehensive survey across devenv.nix (13 groups), devenv.yaml (8 groups), Nix daemon (12 settings). Report at `config-options-research.md`. Includes hardened boilerplate example.

- [x] **P1-T5: Prior art & community practices** — How others have hardened devenv.sh or Nix-based dev environments
  - Priority: medium
  - Estimate: medium
  - Started: 2026-05-12
  - Completed: 2026-05-12
  - Outcome: success
  - Notes: No hardened devenv boilerplate exists. Report at `prior-art-research.md`. 22 sources saved. Identified 6 immediate, 4 near-term, 3 systemic hardening measures.

- [x] **P1-T6: Cross-reference with package-supply-chain-security** — Pull relevant findings from the sibling spike
  - Priority: medium
  - Estimate: small
  - Started: 2026-05-12
  - Completed: 2026-05-12
  - Outcome: success
  - Notes: Mapped 6 applicable areas, identified 7 devenv-specific gaps. Report at `supply-chain-cross-ref-research.md`.

## Phase 2: Research & Investigation

### Pending

### Active

### Completed
- [x] **P2-T1: Hardened devenv.nix + devenv.yaml boilerplate** — Assemble the production-ready security-hardened template from Phase 1 findings.
  - Priority: high
  - Estimate: large
  - Started: 2026-05-12
  - Completed: 2026-05-12
  - Outcome: success
  - Notes: Complete boilerplate at `boilerplate-research.md` (974 lines). Four files: devenv.yaml, devenv.nix, devenv.local.nix.example, .envrc. Settings classified MUST-HAVE/RECOMMENDED/OPTIONAL with threat model mapping.

- [x] **P2-T2: System-level nix.conf hardening guide** — Document the Nix daemon settings devenv can't control.
  - Priority: high
  - Estimate: medium
  - Started: 2026-05-12
  - Completed: 2026-05-12
  - Outcome: success
  - Notes: 10 settings covered with 3 config formats (NixOS module, standalone nix.conf, per-user). Report at `nix-conf-hardening-research.md` (760 lines).

- [x] **P2-T3: Pre-commit security hook suite** — Configure ripsecrets + gitleaks + custom lock-file audit + flake-checker + semgrep/trufflehog.
  - Priority: high
  - Estimate: medium
  - Started: 2026-05-12
  - Completed: 2026-05-12
  - Outcome: success
  - Notes: 17 hooks classified into commit-time/pre-push/CI-only tiers. Custom lock-file-audit hook designed. Report at `precommit-hooks-research.md` (~1000 lines).

- [x] **P2-T4: Trust model documentation** — Team-facing document explaining what devenv trusts and how to verify.
  - Priority: medium
  - Estimate: medium
  - Started: 2026-05-12
  - Completed: 2026-05-12
  - Outcome: success
  - Notes: 8 trust dependencies, verification matrix, code review checklist, 3-tier red flag table. Report at `trust-model-research.md`.

- [x] **P2-T5: Vulnerability scanning integration** — Research vulnix, sbomnix, flake-checker, Trivy/Grype as CI/pre-push gates.
  - Priority: medium
  - Estimate: medium
  - Started: 2026-05-12
  - Completed: 2026-05-12
  - Outcome: success
  - Notes: 6 tools evaluated. Recommended stack: flake-checker (commit-time), vulnxscan/sbomnix (CI), nightly deep scans. Trivy doesn't support Nix and was itself supply-chain-attacked in March 2026. Report at `vuln-scanning-research.md`. 16 new sources saved.

- [x] **P2-T6: Runtime isolation assessment** — Deep dive into runtime isolation options.
  - Priority: medium
  - Estimate: medium
  - Started: 2026-05-12
  - Completed: 2026-05-12
  - Outcome: success
  - Notes: 7 approaches evaluated. Both native PRs stalled. landrun (Landlock CLI) is most promising lightweight option. systemd --user sandboxing nearly useless. Report at `runtime-isolation-research.md`. 13 new sources saved.

## Phase 3: Synthesis & Review

### Pending

### Active

### Completed
- [x] **P3-T1: Depth checklist review of all reports** — Run revision cycle across all 12 research reports.
  - Priority: high
  - Estimate: medium
  - Started: 2026-05-12
  - Completed: 2026-05-12
  - Outcome: success
  - Notes: All 12 reports pass. 7 full pass, 5 with minor partials. Zero contradictions. Cross-report consistency verified. Report at `depth-review-research.md`.

- [x] **P3-T2: Write final conclusions in research.md** — Synthesize all findings into actionable conclusions.
  - Priority: high
  - Estimate: medium
  - Started: 2026-05-12
  - Completed: 2026-05-12
  - Outcome: success
  - Notes: Conclusions written with: answer to research question, security model summary, 6-layer boilerplate description, 6 explicit non-protections, deployment strategy table, key metrics.
