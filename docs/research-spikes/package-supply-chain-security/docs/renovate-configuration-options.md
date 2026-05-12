<!-- Source: https://docs.renovatebot.com/configuration-options/ -->
<!-- Retrieved: 2026-05-12 -->

# Renovate Configuration for Security-First "Set and Forget"

## Key Security-Focused Options

**Stability & Release Vetting:**
The documentation describes `minimumReleaseAge` as enabling Renovate to "suppress branch/PR creation for X days" or "await X time duration before automerging." This allows teams to let new releases stabilize before adoption, reducing the risk of early-stage bugs.

**Selective Automerge Strategy:**
Rather than automerging all updates, Renovate supports granular control through `packageRules`. A practical example from the docs shows:
- Automerge only patch/minor updates: `"matchUpdateTypes": ["minor", "patch"]`
- Exclude major versions from automation
- Apply rules by dependency type: `"matchDepTypes": ["devDependencies"]`

**Confidence-Based Filtering:**
The `packageRules` section includes `matchConfidence` criteria, letting organizations accept only high-confidence updates while requiring review for uncertain ones.

**Scheduling & Rate Limiting:**
- `schedule`: Controls when Renovate creates branches/PRs
- `prHourlyLimit` and `commitHourlyLimit`: Prevent CI/CD overwhelm by capping automation velocity
- `automergeSchedule`: Restricts automatic merging to specific windows (e.g., business hours)

**Merge Strategy Control:**
`automergeStrategy` options (`auto`, `squash`, `rebase`) let teams enforce organizational code review standards even during automation.

## Supported Ecosystems

The `constraints` table spans 40+ package managers (npm, Python, Go, Ruby, Java, etc.), with self-hosted Renovate supporting all platforms (GitHub, GitLab, Azure, Bitbucket).
