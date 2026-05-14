# Blog Posts, Articles, and Conference Talks: Nix Adoption Failures and Pain Points

## Overview

This report catalogs and analyzes published first-person accounts of Nix adoption challenges, partial abandonments, and full reversions. It draws on 18+ blog posts, Discourse threads, vendor acknowledgments, and journalism pieces to identify recurring failure patterns. The emphasis is on concrete, first-person narratives with specific technical details -- not generic complaints.

The sources divide into four categories:
1. **Full abandonment narratives** -- authors who tried Nix/NixOS and left entirely
2. **Partial reversion narratives** -- authors who scaled back from NixOS to selective Nix use
3. **"Cursed but staying" narratives** -- authors who document severe pain but continue despite it
4. **Vendor/ecosystem acknowledgments** -- companies and vendors that publicly recognize adoption barriers

---

## 1. Full Abandonment Narratives

### 1.1 Karl Voit: "Good bye NixOS, Hello Debian (Again)!" (Aug 2025)
- **Source:** `docs/goodbye-nixos-hello-debian-karl-voit.md`
- **Background:** Nearly 30 years of GNU/Linux experience. Used NixOS for ~2 years.
- **Pain points:** Demanded becoming a "Nix wizard" (multi-month/lifetime commitment). Python scripts that were 4 lines on Debian required 50+ lines of Nix configuration. Xfce settings via xfconf.settings were inconsistent and many produced no effect. A firmware update via fwupdmgr created an unrecoverable boot loop that defeated NixOS's rollback. 30GB base installation vs 10GB for Debian.
- **Switched to:** Debian 13 Trixie
- **Verdict:** "One of my worst IT ideas so far." NixOS "provided me solutions to problems I never had."

### 1.2 Sleeyax: "Why I Stopped Using NixOS and Went Back to Arch Linux" (Mar 2025)
- **Source:** `docs/stopped-using-nixos-back-to-arch-sleeyax.md`
- **Background:** Used NixOS for ~1 year on laptop (Arch on desktop). Used nixos-unstable with flakes, home-manager, cachix.
- **Pain points:** Endless "rebuild > fix > rebuild" cycle. Random breakage of audio, Bluetooth, Electron apps after reboots. Cryptic error messages burying actionable info under stack traces. 4-5+ hour compilation times for routine updates. Binary caches frequently missed packages. "Better answers on the Arch Wiki than on the NixOS Wiki itself."
- **Breaking point:** Compilation times made the system impractical for daily use.
- **Switched to:** Arch Linux
- **Alternatives cited:** BTRFS snapshots for generations, aconfmgr for declarative packages, Docker/Podman for dev environments.

### 1.3 Carlos Becker: "Moving on from Nix" (Jun 2025)
- **Source:** `docs/bye-nix-carlos-becker.md`
- **Background:** Used Nix for 2+ years for dotfile management on macOS.
- **Pain points:** Required "nixifying everything" -- couldn't use Lazy, Mason, tpm plugin managers due to read-only folders. macOS Spotlight ignored Nix Apps symlinks, forcing Homebrew anyway. nix-unstable lagged significantly behind upstream. Configuration changes took several minutes to apply. Nix store consumed 60GB+ vs 6GB for Homebrew. nixpkgs repo was 4.7GB to clone.
- **Switched to:** Shell script for dotfiles + Homebrew
- **Nuance:** Acknowledges Nix has value for genuine reproducibility needs, just not for personal laptops. Changed laptops only 4 times in a decade -- reproducibility ROI was negative.

### 1.4 Ugur Erdem Seyfi (rugu.dev): "Why I'm Leaving NixOS After a Year" (Aug 2025)
- **Source:** `docs/leaving-nixos-after-a-year-rugu.md`
- **Background:** Software developer. Wrote an earlier post about switching from Arch to NixOS.
- **Pain points:** Three-option dilemma when programs don't work: debug NixOS module, create manual systemd units, or use containers. "NixOS hates pre-compiled programs" (nix-ld, buildFhsEnv). Leaky abstractions -- "when things go wrong, you now have an additional layer to worry about." Tasks that take minutes on traditional distros take significantly longer.
- **Key insight:** Predicted ROI would improve with more usage; the opposite happened.
- **Switched to:** Arch Linux

### 1.5 MR Lemon (Yulqen): "Quietly moving on from NixOS?" (Apr 2023)
- **Source:** `docs/quietly-moving-on-from-nixos-yulqen.md`
- **Background:** Used NixOS for ~6 months.
- **Pain points:** Insufficient depth to troubleshoot outside standard configs. FHS absence made simple scripts complex. Overlays required significant mental effort. Hardware failures on 2 of 3 laptops eliminated the multi-machine reproducibility benefit.
- **Switched to:** Debian
- **Verdict:** "I'm not clever enough and life is too short."

### 1.6 Jonathan (jonsdocs): "Moving from NixOS to Ubuntu" (Nov 2020)
- **Source:** `docs/moving-from-nixos-to-ubuntu-jonathan.md`
- **Background:** Used NixOS for several years.
- **Pain points:** Persistent app compatibility issues: Brasero, TeamViewer, Zoom, Steam all broken. Nixpkgs overwhelmed with 4,400+ issues and insufficient maintainers. Niche status made finding solutions difficult.
- **Switched to:** Ubuntu (applications "worked straight away")

### 1.7 Railway: "Why We're Moving on From Nix" (Mar 2025)
- **Source:** `docs/railway-moving-on-from-nix.md`
- **Background:** Company. Built 14 million applications with Nixpacks over 3 years. ~200K users encountered limitations.
- **Pain points:** Nix's commit-based versioning meant only latest major version accessible. Updating one commit hash cascaded all package versions, breaking previously-working builds. Single-layer /nix/store produced massive Docker images (500MB+ for Python). Minimal control over layer caching.
- **Switched to:** Railpack (custom Go tool using BuildKit). 38% smaller Node images, 77% smaller Python images.
- **Significance:** This is the most prominent company-level "moved away from Nix" story. Nixpacks is now deprecated.

---

## 2. Partial Reversion Narratives

### 2.1 Jono: "Nix - Death by a Thousand Cuts" (Jan 2025)
- **Source:** `docs/nix-death-by-a-thousand-cuts-jono.md`
- **Background:** Decades of software engineering and DevOps experience. Linux daily since the 1990s. Attended NixCon US, donated to NixOS Foundation. 2 years as primary desktop OS.
- **Pain points (10 categories):**
  1. Multiple competing approaches with no clear guidance (home-manager vs nix-darwin vs flakes vs legacy)
  2. Daily updates downloading 500MB; builds grinding 16-core/64GB machines to a halt
  3. Unstable packages lagging years behind upstream; inconsistent quality across packages
  4. Nix language barriers; no comprehensive language server or autocomplete
  5. Documentation fragmented and assumes expertise (unfavorably compared to Arch Wiki)
  6. Cryptic error messages; poor debugging tools
  7. Desktop integration failures (XDG associations, display managers)
  8. Dev environment friction (Python/conda requiring 3-level nesting; IDE launch issues)
  9. Configuration cruft accumulating (commented-out attempts, symlink workarounds)
  10. Package search misrepresenting availability across architectures
- **Concrete failures:** ZFS encryption setup depending on disappeared GitHub account. Firefox extension management never working. Flatpak integration hanging for 10 minutes. npm link rejected by Nix.
- **What works:** NixOS on home servers (simple services). Ephemeral shells. Declarative config for multi-machine swaps. Home-manager for dotfiles.
- **Decision:** Keep NixOS on servers, migrate workstations to nix-the-package-manager + home-manager only.
- **Quote:** "NixOS is shit. The problem is, all other OS are even worse."
- **Significance:** This is the single most comprehensive and technically detailed criticism piece. Author is experienced, sympathetic to Nix, and specific about failures.

### 2.2 Daniel Sidhion: "NixOS is a good server OS, except when it isn't" (Mar 2024)
- **Source:** `docs/nixos-server-issues-daniel-sidhion.md`
- **Background:** Investigated NixOS for minimal server/microVM deployments.
- **Pain points:** Minimal NixOS was 900MB vs Alpine's 210MB. Reducing it required disabling security wrappers, removing Perl/Python, overlaying packages -- hitting infinite recursion errors, circular dependencies, and hardcoded binaries. Concluded the architecture assumes interactive OS defaults.
- **Decision:** Abandoned NixOS for ultra-minimal servers. Kept it for standard server use.
- **Verdict:** "Trying to mold NixOS into the shape I wanted just isn't the way to go."

---

## 3. "Cursed But Staying" Narratives

### 3.1 Wesley Aptekar-Cassels: "The Curse of NixOS" (Jan 2022)
- **Source:** `docs/curse-of-nixos-wesley-aptekar-cassels.md`
- **Background:** 3 years as sole OS. One of the most-cited Nix criticism pieces.
- **Core metaphor:** NixOS is a "curse" -- it shows you superior package management, making alternatives unbearable, but implements it through "extremely complicated constantly changing software" in a "poorly-designed homegrown language."
- **Key criticism:** Most users copy-paste configs until they need customization, then are "completely high and dry." Dependencies identified by grepping for /nix/store/ rather than static analysis. Standard shebangs fail.
- **Decision:** Continuing despite frustrations. Hopes for a successor with Nix's philosophy but better implementation.

### 3.2 Pierre Zemb: "Three Years of Nix and NixOS" (Jul 2025)
- **Source:** `docs/three-years-nixos-good-bad-ugly-pierre-zemb.md`
- **Background:** 3 years of use. Distributed systems engineer.
- **The bad:** No quick edits (even aliases require rebuilds). Steep learning curve where existing knowledge doesn't help. Pre-compiled binaries fail. Hardcoded build paths in crypto libraries.
- **The ugly:** The Nix language -- "simple things can be hard to figure out." Notes LLMs have significantly eased learning.
- **Decision:** Stays. "I wouldn't switch away from NixOS." Trades short-term convenience for long-term stability.
- **Recommendation:** Start with nix-the-package-manager before committing to NixOS.

### 3.3 Simon Gutgesell: "Nix, the Good and the Bad" (2024)
- **Source:** `docs/nix-good-and-bad-simon-gutgesell.md`
- **Pain points:** Build evaluation takes 10+ minutes before building begins. Error messages fill half a terminal buffer. "All or nothing" approach to dynamically linked libraries.
- **PR backlog:** 5,000+ open PRs making contributions difficult.
- **Decision:** Continues. "Pretty bad, but it's the best that there is."

### 3.4 Julia Evans: "Some notes on NixOS" (Jan 2024)
- **Source:** `docs/some-notes-on-nixos-julia-evans.md`
- **Background:** Well-known technical blogger. Used NixOS for a server replacing chaotic Ansible setup.
- **Pain points:** "I still don't really understand the nix language syntax that well." Cryptic errors requiring Discord help. Planned to copy-paste templates indefinitely.
- **Decision:** Cautiously optimistic after one week. Found it more reliable than Ansible.

---

## 4. Vendor/Ecosystem Acknowledgments of Adoption Barriers

### 4.1 Determinate Systems: "We Want to Make Nix Better"
- **Source:** `docs/determinate-systems-make-nix-better.md`
- **Key admission:** Even experienced developers question whether benefits justify effort. Existing tools are "good enough" so organizations see Nix adoption as risky. Solution requires writing Nix code -- a barrier for non-Nix users.
- **Strategy:** Make Nix work invisibly in the background without requiring users to learn the language.

### 4.2 Flox: "Enterprise Nix: It's Time to Bring Nix to Work"
- **Source:** `docs/flox-enterprise-nix-adoption-barriers.md`
- **Key admission:** The "wizard" dilemma -- organizations develop silos where only a few Nix experts emerge. "Unfortunately people are often not willing or able to learn Nix...even if it would make them vastly more productive." Nix lacks enterprise features: security auditing, sharing, collaboration tools.
- **Strategy:** Wrap Nix with familiar CLI commands that don't require learning the language.

### 4.3 Grayson Head: "Is Nix Worth The Hype?" (Sep 2023)
- **Source:** `docs/is-nix-worth-the-hype-grayson-head.md`
- **Team adoption insight:** Organizational adoption requires "organic interest within the engineering organization." Without existing developer enthusiasm, convincing multi-team enterprises is difficult. Documentation reads "like an RFC" without narrative guidance. Minimal viable examples are as complex as full package builds.

### 4.4 Denny Britz: "Adopting Nix" (Jan 2023)
- **Source:** `docs/adopting-nix-denny-britz.md`
- **Context:** Refactored CI pipelines from Makefiles/Dockerfiles to Nix.
- **Key criticism:** Nix language is "like taking Haskell, removing its type system, and mixing in Javascript." Official docs outdated and scattered. Flakes documentation split between flake and non-flake approaches. macOS sandboxing significantly weaker. Package manager tools (crane, pip2nix) early-stage with unresolved bugs.

---

## 5. Community and Governance Dimension

### 5.1 The 2024 Governance Crisis
- **Source:** `docs/nix-forked-politics-the-register.md`
- The Nix project experienced a governance crisis in 2024: the Anduril military contractor sponsorship controversy, an anonymous open letter, the forced resignation of founder Eelco Dolstra, mass resignations from the Foundation board and moderation team, and the emergence of community forks (Lix, Auxolotl).
- **Adoption impact:** The crisis fragmented an already-small community, drove away core contributors, and introduced uncertainty about the project's long-term direction. For organizations evaluating Nix adoption, governance instability is a legitimate risk factor.

### 5.2 NixOS Discourse: Community Self-Assessment
- **Source:** `docs/discourse-nixos-pain-points-newbie-intermediate.md`
- Even within the community, intermediate users report: legacy code layers (the "lava layer antipattern"), documentation gaps requiring reading nixpkgs source, pinning complexity, limited customization hooks forcing fork-and-maintain workflows, and search discoverability problems.

---

## 6. Cross-Cutting Pattern Analysis

### 6.1 Pain Points by Frequency (across all sources)

| Pain Point | Sources Citing It | Severity |
|---|---|---|
| Nix language difficulty / poor design | 14/18 | High |
| Documentation fragmented / outdated | 13/18 | High |
| Steep learning curve / time investment | 12/18 | High |
| Pre-compiled binary incompatibility (FHS) | 8/18 | High |
| Cryptic error messages | 7/18 | High |
| Build/compilation times | 6/18 | Medium-High |
| Disk space / bandwidth consumption | 6/18 | Medium |
| Package currency (lagging upstream) | 5/18 | Medium |
| Desktop/application integration issues | 5/18 | Medium |
| Multiple competing approaches (no "one way") | 4/18 | Medium |
| macOS-specific friction | 4/18 | Medium |
| Governance/community instability | 3/18 | Medium |

### 6.2 Abandonment vs. Partial Reversion

Of the accounts studied:
- **Full abandonment (7):** Karl Voit, Sleeyax, Carlos Becker, Ugur Erdem Seyfi, MR Lemon, Jonathan, Railway
- **Partial reversion (2):** Jono (servers only), Daniel Sidhion (standard servers only)
- **Staying despite pain (4):** Wesley Aptekar-Cassels, Pierre Zemb, Simon Gutgesell, Julia Evans
- **Pragmatic adoption with reservations (2):** Denny Britz, Grayson Head

### 6.3 What They Switched To

| Alternative | Count | Context |
|---|---|---|
| Arch Linux | 2 | Desktop users who wanted control without abstraction |
| Debian | 3 | Users seeking stability and simplicity |
| Ubuntu | 1 | User seeking "works out of the box" |
| Homebrew + shell scripts | 1 | macOS dotfile management |
| Custom tooling (Railpack) | 1 | Company replacing Nix in production infrastructure |
| BTRFS snapshots + Docker | 1 | Cited as alternatives to Nix features (not direct switch) |

### 6.4 Duration Before Abandonment

| Duration | Count | Pattern |
|---|---|---|
| < 6 months | 1 | Gave up during learning curve |
| 6-12 months | 3 | ROI never materialized |
| 1-2 years | 4 | Invested heavily, accumulated frustration over time |
| 2+ years | 3 | Deepest investment; most nuanced criticism |
| 3+ years (company) | 1 | Railway -- outgrew Nix's limitations at scale |

### 6.5 Survivorship Bias Assessment

The existence of these narratives is significant precisely because they represent the visible minority. For every person who writes a blog post about leaving Nix, many more quietly abandon it. Several patterns suggest the published accounts undercount abandonment:

1. **Selection for articulateness:** Blog authors tend to be experienced developers who can name their frustrations. Less technical users likely abandon earlier and silently.
2. **The "quietly" pattern:** MR Lemon's post is literally titled "Quietly moving on from NixOS?" -- suggesting abandonment without public commentary is the default.
3. **Vendor acknowledgment:** Both Determinate Systems and Flox built entire businesses around the premise that Nix is too hard for normal adoption. Their existence is evidence of widespread silent failure.
4. **The "wizard" dependency:** Flox explicitly identifies that enterprise adoption depends on a single champion. When that person leaves, the adoption collapses -- but nobody writes a blog post about it.
5. **Community awareness:** The NixOS Discourse thread "Talk me out of (or into?) quitting NixOS" shows this is a recognized pattern within the community itself.

---

## 7. Notable Quotes

> "I spent so much time cobbling together hacky configs that when I step back and look at the house of cards I have built, it is apparent Nix has created more problems then solutions for me." -- Jono

> "NixOS provided me solutions to problems I never had." -- Karl Voit

> "I'm not clever enough and life is too short." -- MR Lemon

> "NixOS is shit. The problem is, all other OS are even worse." -- Jono

> "Pretty bad, but it's the best that there is." -- Simon Gutgesell

> "I'm going to keep using it, since I can't stand anything else after having a taste of NixOS." -- Wesley Aptekar-Cassels

> "Unfortunately people are often not willing or able to learn Nix...even if it would make them vastly more productive." -- Flox

> "We feel bad when users can't access the latest packages, but feel worse when previously functional builds suddenly fail." -- Railway

> "Tasks that would take very little time on a traditional FHS-based distro can take significantly longer on NixOS." -- Ugur Erdem Seyfi

---

## 8. Implications for Nix Adoption Strategy

### What the failure stories tell us about safe adoption paths:

1. **NixOS-the-OS is the highest-risk entry point.** Every full abandonment involved NixOS as desktop or full system. No one who used only nix-the-package-manager or devshells wrote an abandonment post.

2. **Server use has the best retention.** Even partial reverters (Jono, Sidhion) kept NixOS on servers. The pain concentrates in desktop/interactive use, not service management.

3. **The "wizard" problem is real and documented.** Flox and Determinate Systems both identify single-champion dependency as the primary enterprise failure mode. When the champion leaves, adoption collapses.

4. **macOS is a secondary pain multiplier.** Sandboxing differences, Spotlight integration failures, and slower builds on macOS amplify existing friction.

5. **Python and data science are particularly painful.** Karl Voit, Jono, and Denny Britz all cite Python/conda/pip integration as a major source of friction. C library dependencies (NumPy) create especially intractable problems.

6. **The ROI curve disappoints.** Multiple authors (Ugur Erdem Seyfi, Karl Voit) explicitly note that increased investment did not yield proportional returns. The expectation of "it gets easier" was falsified.

7. **Gradual adoption is the only strategy that avoids the worst failure modes.** Pierre Zemb and Grayson Head both recommend starting with nix-the-package-manager before NixOS. The Discourse community itself recommends Nix -> Home-Manager -> NixOS progression. No one recommends jumping straight to NixOS.
