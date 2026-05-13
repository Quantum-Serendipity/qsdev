<!-- Source: https://github.com/trailofbits/skills -->
<!-- Retrieved: 2026-05-12 -->

# Trail of Bits Skills Repository

5.1k stars, 453 forks, CC-BY-SA-4.0 license. 113 commits.

## Plugin Categories

### Smart Contract Security (2)
- building-secure-contracts: Vulnerability scanners for 6 blockchains
- entry-point-analyzer: State-changing function identification

### Code Auditing (14+)
- agentic-actions-auditor: GitHub Actions workflow security review
- audit-context-building: Deep architectural code analysis
- burpsuite-project-parser: Burp Suite data extraction
- c-review: C/C++ security analysis with parallel workers
- differential-review: Security-focused code change review
- dimensional-analysis: Unit mismatch and formula bug detection
- fp-check: False positive verification with review gates
- insecure-defaults: Configuration and credential detection
- semgrep-rule-creator: Custom vulnerability detection rules
- semgrep-rule-variant-creator: Multi-language rule porting
- sharp-edges: Error-prone API and footgun identification
- static-analysis: CodeQL, Semgrep, SARIF integration
- supply-chain-risk-auditor: Dependency threat assessment
- testing-handbook-skills: Fuzzers, sanitizers, coverage tools
- trailmark: Graph analysis, mutation testing, protocol verification
- variant-analysis: Cross-codebase vulnerability pattern matching

### Malware Analysis (1)
- yara-authoring: YARA rule creation with linting

### Verification (5)
- constant-time-analysis: Timing side-channel detection
- mutation-testing: Campaign configuration and optimization
- property-based-testing: Multi-language testing guidance
- spec-to-code-compliance: Blockchain audit verification
- zeroize-audit: Secret zeroization verification

### Reverse Engineering (1)
- dwarf-expert: DWARF debugging format interaction

### Mobile Security (1)
- firebase-apk-scanner: Android APK Firebase misconfiguration scanning

### Development (9)
- ask-questions-if-underspecified: Requirement clarification
- devcontainer-setup: Pre-configured development containers
- gh-cli: GitHub CLI authentication
- git-cleanup: Worktree and branch management
- let-fate-decide: Cryptographic randomness tool
- modern-python: uv, ruff, pytest tooling
- seatbelt-sandboxer: macOS sandbox configurations
- second-opinion: External LLM code reviews
- skill-improver: Iterative refinement loops
- workflow-skill-design: Design patterns for skills

### Team Management (1)
- culture-index: Survey result interpretation

### Tooling (1)
- claude-in-chrome-troubleshooting: MCP extension diagnostics

### Infrastructure (1)
- debug-buttercup: Kubernetes deployment debugging

## Installation
- `/plugin marketplace add trailofbits/skills`
- `/plugin menu` for browsing
- Local dev: Add marketplace from parent directory path

## Notable
- constant-time-analysis skill found a timing side-channel in ML-DSA signing in RustCrypto
- Python 66.3%, Shell 23.4%, YARA 2.5%, CodeQL 1.3%
