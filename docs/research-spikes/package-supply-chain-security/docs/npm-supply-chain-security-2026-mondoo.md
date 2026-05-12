<!-- Source: https://mondoo.com/blog/npm-supply-chain-security-package-manager-defenses-2026 -->
<!-- Retrieved: 2026-05-12 -->

# npm Supply Chain Security in 2026: Protections and Gaps

## Current Threat Landscape

The JavaScript ecosystem faces unprecedented supply chain attacks. The Shai-Hulud worm compromised 796 packages with 132 million monthly downloads, followed by Glassworm demonstrating this represents "the beginning of a new attack paradigm" rather than isolated incidents.

## Publisher-Side Defenses (npm Registry)

### Trusted Publishing & Provenance Attestations

These complementary features work within the SLSA framework. Trusted publishing requires packages to be built through CI/CD environments (via GitHub Actions OIDC tokens) rather than developer laptops. This makes "stolen passwords and leaked tokens far less useful because attackers cannot replicate the trusted build environment."

Provenance attestations create "a verifiable, cryptographically signed link...between a published package version, the exact source commit that produced it, and the build system that ran the publish step." npm packages published this way achieve roughly SLSA Build Level 2.

**Critical limitation**: Provenance proves *where* code was built and *from which source*, but not "what the code does." Malicious maintainers can publish harmful code through fully trusted pipelines with valid attestations passing every check.

### Granular Access Tokens

Replacing all-or-nothing tokens with scoped, time-limited, IP-restricted credentials significantly reduces blast radius when credentials leak. Tokens can be limited to single packages or marked read-only.

### Enhanced Account Security

Mandatory 2FA for high-impact packages and improved account recovery procedures prevent social engineering takeovers.

## Consumer-Side Gap

Despite publisher improvements, a critical vulnerability remains: "When you run `npm install`, every package you pull down can execute arbitrary code on your machine through lifecycle scripts." These scripts operate with full user privileges, accessing credentials, SSH keys, and CI/CD secrets.

Shai-Hulud exploited this precisely: preinstall scripts harvested npm tokens and GitHub credentials, enabling exponential worm propagation. Publisher-side improvements "do not prevent your machine from running malicious code that already made it through."

## Consumer-Side Defenses (Package Manager Level)

### pnpm v11's Three-Layer Defense

**Layer 1: Lifecycle Script Blocking**
The `strictDepBuilds` setting (enabled by default since late 2025) blocks all preinstall and postinstall scripts unless explicitly allowlisted through `allowBuilds`. This represents "a fundamental shift" from trusting all dependencies by default.

**Layer 2: Release Cooldown**
The `minimumReleaseAge` setting (defaulting to 1440 minutes/one day) prevents resolving packages published within a time window. Most attacks are detected and removed within hours -- Shai-Hulud within 12 hours, the debug/chalk attack within 2.5 hours. This "trades bleeding-edge freshness for safety."

**Layer 3: Trust Policy**
The `trustPolicy: no-downgrade` setting (opt-in) detects credential compromise by blocking installations where authentication strength decreases between versions. It verifies provenance attestations through npm's attestation API, assigning trust levels from Trusted Publisher (strongest) to No Trust Evidence (weakest).

### Comparative Landscape

**pnpm v11**: Script blocking (default), per-package allowlists, 1-day cooldown (default), trust policy (opt-in), exotic subdependency blocking

**Yarn Berry v4**: Script blocking (default via `enableScripts: false`), 3-day cooldown (default), hardened mode for lockfile validation

**Bun**: Script blocking (default), optional release cooldown

**npm CLI**: No consumer-side protections -- scripts run by default with only a blunt `--ignore-scripts` flag; no per-package allowlisting; no release cooldown; no trust policy enforcement

## Defense-in-Depth Model

Neither layer independently provides complete protection. Publisher controls cannot prevent malicious code already in the registry from executing. Consumer controls cannot prevent attackers from compromising the registry itself. The combination significantly "raise[s] the bar for attackers," requiring simultaneous defeat of multiple defense layers.

## Actionable Recommendations

- **pnpm v11 users**: Review `allowBuilds` allowlists and consider enabling `trustPolicy: no-downgrade`
- **Yarn Berry users**: Verify `enableHardenedMode` is active
- **Bun users**: Enable `minimumReleaseAge`
- **npm CLI users**: Evaluate switching to package managers with stronger defaults; the npm CLI provides "the fewest consumer-side protections available"
- **All teams**: Document security control exceptions, creating audit trails and forcing conscious trust decisions rather than silent defaults
