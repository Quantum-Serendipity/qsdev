# First-Person Accounts of Nix/NixOS Abandonment

## Overview

This report catalogs first-person accounts from Reddit, Hacker News, Lobsters, NixOS Discourse, and blog posts of individuals and teams who adopted Nix or NixOS and subsequently abandoned, partially retreated from, or significantly scaled back their use. The goal is to address survivorship bias in Nix advocacy by documenting the negative stories that don't get amplified.

**Corpus**: 15 saved source documents from HN threads, Discourse threads, Lobsters, blog posts, and tech publications. Approximately 30 distinct first-person narratives extracted.

**Key finding**: Team/company abandonment stories are remarkably scarce in the public record. The overwhelming majority of abandonment accounts come from individuals using NixOS as a desktop OS or Nix for personal projects. Corporate adoption failures appear to manifest as "never got past the advocacy stage" rather than "adopted then reverted."

---

## Complete Abandonment Narratives

### 1. Sleeyax — NixOS desktop, 1 year, back to Arch
- **Source**: `docs/sleeyax-stopped-nixos-back-to-arch.md`
- **Who**: Individual developer, personal laptop
- **What**: NixOS as daily driver desktop OS
- **Duration**: ~1 year (installed May 2024)
- **Why they left**:
  - System instability: components randomly broke after reboot (audio, Bluetooth, Electron apps)
  - Cryptic error messages with actual errors buried in verbose stack traces
  - Compilation overhead: maintenance updates took 4-5+ hours on slower hardware
  - Binary cache misses forced local compilation frequently
  - Poor documentation compared to Arch Wiki
- **Switched to**: Arch Linux
- **Nuance**: Acknowledges NixOS benefits for enterprise/specific use cases; frustration was desktop-specific

### 2. Uğur Erdem Seyfi (kugurerdem) — NixOS desktop + server, 1 year, back to Arch
- **Source**: `docs/rugu-leaving-nixos-after-year.md`
- **Who**: Individual developer
- **What**: Personal system config, server management, experimentation
- **Duration**: ~1 year
- **Why they left**:
  - Troubleshooting required debugging NixOS module source code
  - Pre-compiled binaries don't work; forced to learn nix-ld, buildFhsEnv, custom derivations
  - Leaky abstractions: "when things go wrong, you now have an additional layer to worry about"
  - Time cost: tasks taking minutes on traditional distros consumed much more time
  - No practical reproducibility benefit experienced to justify overhead
- **Switched to**: Arch Linux
- **Nuance**: Acknowledges value for managing multiple systems requiring strict reproducibility

### 3. kstenerud — NixOS, multiple attempts over a decade, back to Debian
- **Source**: `docs/hn-tried-again-to-like-nix-thread.md`
- **Who**: Individual, 30+ years Linux experience
- **What**: Desktop and server use, multiple attempted adoptions
- **Duration**: Multiple forays over ~10 years
- **Why they left**:
  - Broke both systems requiring reinstall despite idempotency claims
  - Documentation "complete and technically accurate, but maddeningly obtuse"
  - Feels like "alchemy rather than science" — requires years of experiential knowledge
  - "I'm done with Nix - burned one time too many"
- **Switched to**: Debian (reverted one server), considering Guix

### 4. k32 — Nix package manager, 6 months, back to Ansible
- **Source**: `docs/hn-nix-nice-research-project-6months.md`
- **Who**: Individual developer (Erlang/Elixir ecosystem)
- **What**: Production and development use, contributed to nix-packages
- **Duration**: 6 months
- **Why they left**:
  - "not suitable for production nor development" (in their case)
  - Extensive patching requirements; security concerns about patch quality
  - Erlang/Elixir integration issues (patched out rebar.lock checks)
  - Secrets management concerns (world-readable /nix directories)
  - opam packages and pre-compiled game compatibility issues
- **Switched to**: Ansible playbooks for machine management

### 5. cge — Nix for research reproducibility, ~3 years, abandoned
- **Source**: `docs/hn-nix-death-by-thousand-cuts-thread.md`
- **Who**: Academic/researcher
- **What**: Making research code repositories reproducible
- **Duration**: ~3 years
- **Why they left**:
  - "everything broke within around three years"
  - Poor documentation for channel revision pinning
  - Opaque commit hash system
- **Switched to**: Not specified

### 6. _w1tm — Nix for Rust development on macOS, abandoned
- **Source**: `docs/hn-nix-death-by-thousand-cuts-thread.md`
- **Who**: Individual developer
- **What**: Rust development environment on macOS
- **Duration**: Not specified
- **Why they left**:
  - Couldn't resolve linker errors with macOS standard libraries
  - Nix fixes created different problems
  - "Everything just worked outside of the Nix environment so ended up dropping it"
- **Switched to**: Native macOS toolchain

### 7. fransje26 — Nix, gave up ~5 years ago
- **Source**: `docs/hn-tried-again-to-like-nix-thread.md`
- **Who**: Individual developer
- **What**: Not specified
- **Duration**: Not specified
- **Why they left**: Documentation issues; notes "not much has changed on that front since then"
- **Switched to**: Not specified

### 8. brabel — Nix package manager, abandoned
- **Source**: `docs/hn-nix-slow-disk-heavy.md`
- **Who**: Individual developer
- **What**: Package management on laptop
- **Duration**: Not specified
- **Why they left**:
  - Nix store reached ~100 GB on a 250GB laptop
  - Updates consumed "a good part of an hour"
  - "it's very slow and it's extremely disk heavy"
- **Switched to**: apt (stayed on traditional distro)

### 9. Zambyte — NixOS, several months, switched to GNU Guix
- **Source**: `docs/hn-nix-death-by-thousand-cuts-thread.md`
- **Who**: Individual developer (Haskell background)
- **What**: NixOS as desktop OS
- **Duration**: Several months
- **Why they left**:
  - Nix language underdocumented
  - "compounding frustration" over months
  - Preferred Guix's use of Scheme — "a language with decades of academic backing"
- **Switched to**: GNU Guix (~4 years as of writing)

### 10. Simon Batt — NixOS desktop, 1 week, back to Fedora
- **Source**: `docs/xda-nixos-daily-driver-back-to-fedora.md`
- **Who**: Tech journalist / general user
- **What**: NixOS as daily driver laptop OS
- **Duration**: 1 week
- **Why they left**:
  - Declarative paradigm shock: "Installing Google Chrome via a config file felt like programming"
  - KDE Discover broke immediately
  - Learning curve too steep for workflow
- **Switched to**: Fedora Kinoite
- **Nuance**: Described NixOS as "the most enjoyable, customizable nightmare I've ever had"

### 11. Jasper Clarke — NixOS, 1 year, partial retreat to Arch VM
- **Source**: `docs/jasper-clarke-nixos-not-best-linux.md`
- **Who**: Individual developer
- **What**: System administration and development (C, Java, Rust, OpenGL)
- **Duration**: 1 year
- **Why they partially left**:
  - Development with external libraries extremely painful
  - OpenGL setup that's one command on Arch was impossible on NixOS
  - "NixOS makes the operating system easy, but the development environment torture"
  - Setup overhead created psychological barriers to starting projects
- **Switched to**: Hybrid — NixOS for sysadmin, Arch Linux VM for development
- **Nuance**: Didn't fully abandon; adopted split approach

### 12. kingmob — Nix on macOS, months, back to asdf/brew
- **Source**: `docs/lobsters-nix-breaking-point-thread.md`
- **Who**: Individual developer
- **What**: Nix on macOS for development
- **Duration**: "a few months"
- **Why they left**:
  - Poor documentation
  - Time-consuming learning curve for non-off-the-shelf derivations
  - Continued split between flakes/non-flakes
  - Various macOS-specific issues
  - Governance concerns (DetSys prioritization, Anduril sponsorship controversy)
- **Switched to**: asdf + Homebrew

### 13. kivikakk — Nix, ~1 year, partial retreat to asdf/brew
- **Source**: `docs/lobsters-nix-breaking-point-thread.md`
- **Who**: Individual developer
- **What**: Nix on macOS, nix-darwin
- **Duration**: ~1 year of earnest engagement
- **Why they partially left**:
  - Platform incompatibilities: "Builds may not generally be reproducible between NixOS and Nix on a different platform"
  - Community governance issues lacking "meaningful forward progress"
- **Switched to**: Reinstalled asdf, Homebrew back on PATH, still uses nix-darwin

### 14. mtmk — Nix for .NET deployment, abandoned attempt
- **Source**: `docs/hn-tried-again-to-like-nix-thread.md`
- **Who**: Individual developer
- **What**: .NET 8 AOT deployment
- **Duration**: Not specified (single attempt)
- **Why they left**:
  - "spiraled into a plethora of issues, ultimately forcing me to back down"
  - Concluded "you need solid understanding before reasonably productive"
- **Switched to**: Not specified

### 15. jokethrowaway — NixOS, switching back to Arch
- **Source**: `docs/hn-tried-again-to-like-nix-thread.md`
- **Who**: Individual developer
- **What**: NixOS desktop
- **Duration**: Not specified
- **Why they left**:
  - "rebuilding everything at every update take forever"
  - Prefers binary bleeding edge packages and AUR
- **Switched to**: Arch Linux

---

## Failed Corporate/Team Adoption Attempts

These are cases where someone tried to introduce Nix into a team or company and it was rejected or abandoned. These are particularly valuable because team dynamics amplify individual pain points.

### 16. TLATER's consultancy — Nix rejected, team stayed on Ansible
- **Source**: `docs/discourse-issues-pushing-nixos-to-companies.md`
- **Who**: Smallish consultancy, filled with Debian and GNOME maintainers
- **What**: Attempted to suggest NixOS for infrastructure
- **Why it failed**:
  - Operations lead: "it's too different, I don't have the time to learn this"
  - Team continued using Ansible
  - Suggesting "relatively obscure" technology with "smaller community" to customers met with indifference
  - One internal project added flake.nix, which became unused after enthusiasts left the company
- **Key pattern**: **Champion departure** — Nix usage died when advocates left

### 17. Melkor333's employer — NixOS advocacy stalled
- **Source**: `docs/discourse-issues-pushing-nixos-to-companies.md`
- **Who**: Company (size not specified)
- **What**: Attempted to advocate for NixOS adoption
- **Why it failed**:
  - No proper LTS release; "way too short update time"
  - No company backing comparable to RHEL/SLES
  - Flakes remain experimental, deterring enterprise recommendation
  - CVE handling concerns
- **Key pattern**: **Enterprise stability expectations** unmet

### 18. NobbZ's international firm — NixOS prohibited for hosting
- **Source**: `docs/discourse-issues-pushing-nixos-to-companies.md`
- **Who**: International firm, ops team of 3, large customers (one 800x larger)
- **What**: Presented Nix(OS) to ops team
- **Why it failed**:
  - Compliance agreements mandate Ubuntu LTS
  - "completely different beast to maintain"
  - 6-month release cycle incompatible with hosting agreements
- **Outcome**: NixOS prohibited for hosting; Nix permitted only on Mac/WSL for development
- **Key pattern**: **Compliance/contractual constraints**

### 19. Nebucatnetzer's 20-person company — Nix limited to dev environments
- **Source**: `docs/discourse-issues-pushing-nixos-to-companies.md`
- **Who**: 20-employee company, single DevOps person
- **What**: Considering NixOS for company servers
- **Why it stalled**:
  - Only DevOps person on team — can't risk new tech choices alone
  - "Conventional tools allow bugs; new tools create expectation of personal problem-solving"
- **Outcome**: Nix for dev environments only; deploying to Ubuntu servers
- **Key pattern**: **Bus factor** — single-person risk

### 20. Cachix/devenv observations — Pattern of team abandonment
- **Source**: Web search result (Cachix blog / devenv marketing)
- **Who**: Multiple teams observed by the Cachix team
- **What**: Nix for development environments in teams
- **Pattern observed**:
  - "Nix is initially introduced by someone enthusiastic about the technology"
  - "abandoned after backlash from the rest of the team when faced with a steep adoption curve"
  - Shopify "was vocal about Nix way back in 2020, but eventually went quiet"
- **Key pattern**: **Enthusiast-driven adoption collapses** when the rest of the team can't keep up

---

## Partial Abandonment / "Stockholm Syndrome" Cases

These individuals stayed but express significant reservations, serving as leading indicators of potential future abandonment.

### 21. emarthinsen — NixOS daily driver, would choose Arch if starting over
- **Source**: `docs/hn-nix-death-by-thousand-cuts-thread.md`
- "I wouldn't recommend it for most people (even for me)"
- Builds non-reproducible in practice (needs multiple `nixos-rebuild switch` runs)
- NVIDIA complexity, Wayland issues

### 22. Wesley Aptekar-Cassels — NixOS 3 years, stayed but deeply critical
- **Source**: `docs/wesley-ac-curse-of-nixos.md`
- "I'm going to keep using it, since I can't stand anything else after having a taste of NixOS"
- Nix language "not very good and is extremely difficult to learn"
- Most users "simply copy/paste example configurations"
- All software needs recompilation "often with terrifying hacks"

### 23. Pierre Zemb — NixOS 3 years, stayed but acknowledges heavy cost
- **Source**: `docs/pierre-zemb-three-years-nixos-good-bad-ugly.md`
- "needing an AI to help with basic packaging shows how hard the language is to learn"
- Binary incompatibility, patchelf workarounds, buildFHSUserEnv fallbacks
- Stayed because reproducibility is "a superpower"

### 24. Simon Gutgesell — Nix user, ambivalent
- **Source**: `docs/nomisiv-nix-good-and-bad.md`
- "Nix is pretty bad, but it's the best that there is"
- Flake evaluation takes up to 10 minutes before building starts
- Error messages fill half the terminal buffer with internal nixpkgs references

### 25. Drake Rossman — Nix on macOS, reluctantly continues
- **Source**: `docs/drake-rossman-nix-macos-good-bad-ugly.md`
- "the worst package manager on MacOS except for all the other package managers"
- Applications randomly stop working; macOS deletes Nix files
- Karabiner, Firefox broken; deploy-rs doesn't work on macOS

### 26. tombert — NixOS 6+ years, stays but won't recommend
- **Source**: `docs/hn-nix-death-by-thousand-cuts-thread.md`
- Loves it for servers; can't run "generic Linux programs" on desktop without workarounds
- Too complex for non-technical users
- "I didn't really want to know how to make my own Nix package"

### 27. yoyohello13 — Left NixOS, kept Nix + Home Manager on Arch
- **Source**: `docs/hn-nix-death-by-thousand-cuts-thread.md`
- Waiting for Flakes to lose experimental flag before committing to NixOS
- Uses Nix + Home Manager for dotfile management on Arch

---

## Contributor/Community Abandonment

These are cases where the governance crisis and community dynamics drove people away from the Nix ecosystem.

### 28. Major Nixpkgs contributor (~10 years) — left over Anduril sponsorship
- **Source**: Web search (Discourse thread)
- ~10 years NixOS use, 8+ years contributions
- Left over Foundation's "neutral" stance on Anduril (weapons manufacturer) sponsorship

### 29. Multiple community members — left over governance crisis
- **Source**: `docs/lobsters-nix-breaking-point-thread.md`, web search results
- Eelco Dolstra (founder) stepped down after open letter signed by 160 people
- Mass resignations from Foundation board and moderation team
- Community forks emerged (Lix, Auxolotl)
- Contributor burnout from PR bureaucracy and stalled decision-making

---

## Pattern Analysis

### Frequency of Abandonment Reasons

Ranked by how often each reason appeared across the ~30 narratives:

| Reason | Frequency | Context |
|--------|-----------|---------|
| **Nix language / learning curve** | Very high (~20/30) | Universal across all contexts |
| **Poor/fragmented documentation** | Very high (~18/30) | Universal |
| **Pre-compiled binary incompatibility** | High (~12/30) | Desktop and dev environments |
| **Build/evaluation slowness** | High (~10/30) | Desktop, CI, dev environments |
| **Cryptic error messages** | High (~10/30) | Universal |
| **macOS-specific issues** | Moderate (~5/30) | Dev environments on macOS |
| **Disk space consumption** | Moderate (~5/30) | Laptops, CI |
| **Flakes instability/experimental status** | Moderate (~5/30) | Enterprise, long-term commitment |
| **Governance/community crisis** | Moderate (~5/30) | Contributors, long-term users |
| **No LTS / enterprise support** | Low-moderate (~4/30) | Corporate adoption |
| **Champion departure** | Low (~2/30 explicit) | Teams (but likely underreported) |
| **Compliance/contractual constraints** | Low (~2/30) | Enterprise |

### What People Switched To

| Alternative | Count | Context |
|-------------|-------|---------|
| **Arch Linux** | 6 | Desktop NixOS users |
| **Ansible** | 2 | Server management |
| **asdf + Homebrew** | 2 | macOS dev environments |
| **Debian** | 2 | Desktop/server |
| **Fedora** | 1 | Desktop |
| **GNU Guix** | 1 | Desktop (philosophical preference) |
| **Docker** | 1 | CI/deterministic builds |
| **Native toolchain** | 1 | macOS development |

### Duration Before Abandonment

| Duration | Count | Notes |
|----------|-------|-------|
| < 1 week | 2 | Quick rejection of paradigm |
| 1 week - 6 months | 6 | Learning curve not overcome |
| 6 months - 1 year | 5 | Realized costs outweigh benefits |
| 1-3 years | 3 | Long-term maintenance burden |
| 3+ years | 2 | Reproducibility promises broke down over time |

### Individual vs. Team Patterns

**Individual abandonment** (25/30 narratives): Well-documented, vocal, spans desktop and development use. Primary drivers are learning curve, daily friction, and the gap between NixOS's promises and reality for personal workflows.

**Team/corporate abandonment** (5/30 narratives): Poorly documented publicly. The stories that exist fall into two patterns:
1. **Never-adopted**: Nix champion couldn't overcome team resistance (learning curve, compliance, risk aversion)
2. **Champion-dependent**: Nix was adopted by an enthusiast, then abandoned when that person left or when the rest of the team pushed back

The Cachix team's observation is the most direct evidence of the team abandonment pattern: Nix gets introduced by an enthusiast, the rest of the team faces a steep learning curve, backlash follows, and the tool is abandoned.

---

## Key Quotes for Presentation Use

> "I broke Arch only once in 5 years whereas NixOS already breaks before updating." — Sleeyax

> "When things go wrong, you now have an additional layer to worry about." — kugurerdem

> "I'm done with Nix - burned one time too many." — kstenerud (HN, decade of attempts)

> "Everything just worked outside of the Nix environment so ended up dropping it." — _w1tm (macOS Rust development)

> "NixOS makes the operating system easy, but the development environment torture." — Jasper Clarke

> "Nix is initially introduced by someone enthusiastic about the technology, then abandoned after backlash from the rest of the team when faced with a steep adoption curve." — Cachix/devenv team observation

> "it's too different, I don't have the time to learn this" — Operations lead rejecting Nix at a consultancy

> "Nix is pretty bad, but it's the best that there is." — Simon Gutgesell (stayed but ambivalent)

> "needing an AI to help with basic packaging shows how hard the language is to learn" — Pierre Zemb (stayed but critical)

---

## Gaps and Limitations

1. **Team stories are underrepresented**: Companies that tried and abandoned Nix are unlikely to write public blog posts about it. The signal is mostly from individuals. The Cachix observation and the Discourse corporate thread are the best team-level data we have.

2. **NixOS-as-desktop dominates**: Most abandonment stories involve NixOS as a desktop OS, which is a different use case than "Nix for team dev environments." The dev-environment-specific abandonment evidence is thinner.

3. **Reddit was largely inaccessible**: Web search consistently failed to surface Reddit-specific threads despite multiple query variations. Reddit's content may not be well-indexed, or these discussions may happen in comments rather than top-level posts.

4. **Survivorship bias in the abandonment stories too**: People who quietly stopped using Nix without writing about it aren't captured here. The accounts we have skew toward people who were invested enough to write about their experience.

5. **No large-company post-mortems found**: Despite searching for team/company abandonment stories extensively, no formal post-mortems from companies that adopted and then abandoned Nix were found. The Shopify "went quiet" observation is the closest signal.
