# Shopify's Nix Adoption Journey: The Stalled First Attempt and devenv Revival

## Executive Summary

Shopify's Nix adoption is a two-act story spanning seven years. In **Act 1** (2018-2020), a small developer infrastructure team led by Burke Libbey built an ambitious Nix-based development environment for ~1,000 macOS developers. The effort was technically sophisticated but failed to reach broad stable footing before a company-wide pivot to cloud development (Spin) made it moot. In **Act 2** (2023-2025), CEO Tobias Lutke personally discovered devenv.sh, adopted it for a service, and catalyzed a renewed Nix adoption — this time with incremental rollout, stakeholder buy-in, and a much higher-level abstraction that shielded developers from raw Nix complexity. Today, the majority of Shopify development happens inside Nix-based environments.

This case study is the most detailed publicly documented example of enterprise Nix adoption, stall, and revival.

---

## 1. Timeline

### Phase 0: Pre-Nix Foundation (~2016)

Shopify built a proprietary `dev` CLI tool that provided declarative configuration for development environments. Running `dev up` would dispatch to `apt` (Linux) or `homebrew` (macOS) to install project dependencies. This tool established the pattern of declarative dev environments long before Nix entered the picture.

**Shadowenv** (open-sourced 2019, github.com/Shopify/shadowenv) was created to manage project-local environment variable shadowing — automatically setting variables when entering a project directory and restoring them on exit.

### Phase 1: Burke Libbey's Nix Initiative (Late 2018 - 2020)

**Key person:** Burke Libbey, Staff Production Engineer / "Nixologist" at Shopify.

**December 2018:** Burke publishes his personal "Learning Nix" guide, documenting his seven-step approach to mastering Nix — suggesting he was ramping up personally around this time.

**October 2019 (NixCon 2019):** Burke presents "Nix-based development environments at Shopify" (27 minutes). Key facts from this era:
- Shopify had ~1,000 developers ("on the order of 1e3"), all on macOS laptops as primary dev environments
- The `dev` tool was modified to enforce a specific nixpkgs revision on each `dev up` invocation
- Custom infrastructure built around Nix included:
  - Binary cache backed by Google Cloud Storage via MinIO + Nginx proxy
  - Custom Ruby-based build output parser with real-time progress UI
  - `setup-hook-to-shadowenv` converter (bridging Nix package hooks into shadowenv)
  - Post-build hooks for cache population
  - Upload daemon for sharing derivations across the org
- By mid-2020, Burke claimed "thousands of R&D folks running Nix on their MacBook Pros already"

**May 2020:** Two Shopify engineering blog posts published:
- "What Is Nix" — Burke's technical explainer of Nix fundamentals
- "ShipIt! Presents: How Shopify Uses Nix" — demonstration of practical tooling

**June 2020:** Burke posts on NixOS Discourse hiring for the Nix infrastructure team. The post describes:
- **Runix**: A Nix-module-based system for declaring projects abstractly. Each project gets a `ru.nix` file specifying its framework, packages, services, commands, and integrations. Runix inferred defaults for common project types and treated divergences as improvement opportunities.
- Goal: Make Nix the "common substrate" for dev environments, CI, and production
- Plans to expand from dev environments into CI/CD (targeted for late 2020)

**June 2020:** Burke also publishes his "Nixology" YouTube screencast series — originally recorded for internal Shopify developer training — publicly.

### Phase 2: The Stall and Pivot to Cloud Dev (2020-2022)

**Why it stalled — multiple reinforcing factors:**

1. **Complexity barrier for non-specialists:** According to the Changelog interview with engineers: "It was too hard to understand, especially for our new engineers." Raw Nix expressions, flakes, and the Nix language presented a steep learning curve for a large, diverse engineering organization.

2. **Tooling mismatch:** Burke himself identified that Nix's "tooling is in general really optimized for 'build' workflows, not development workflows." The bundleEnv/bundleApp workflows didn't map to how Shopify developers actually wanted to work. This required extensive custom solution engineering.

3. **macOS-specific friction:** The Nix ecosystem on macOS had rough edges. Burke documented debugging C99 compilation failures with Ruby gems (gctrack) caused by impure system dependencies — header files missing despite runtime library access. For a Ruby-on-Rails company running thousands of macOS laptops, this was a significant pain point.

4. **Bandwidth/infrastructure issues:** The shift to remote work (2020) exposed bandwidth constraints — large Nix store downloads were problematic over residential internet vs. office infrastructure.

5. **Company-wide pivot to cloud development:** The critical blow was organizational, not technical. As Shopify's monolith grew more complex, laptops were "spinning fans and spooling swapfiles." The company invested heavily in **Spin** — cloud development environments. The pitch was compelling: scalable infrastructure, no more melting laptops, easier onboarding.

**The Spin era (2020-2022):**
- Shopify tried GCE VMs, then Kubernetes pods, then finally **Isospin** (systemd-based "laptop in the cloud")
- Cloud dev made the Nix argument moot: "the easier solution was to just use Ubuntu"
- Nix's promise of reproducible local environments was less relevant when everyone ran identical cloud VMs
- The first Nix effort "didn't reach stable footing" — some teams couldn't use it yet — and then cloud dev provided a seemingly easier path

**What happened to the Nix infrastructure team is unclear.** Shopify went through significant layoffs in 2022-2023 (headcount dropped from ~11,600 to ~8,300). Whether the Nix team was specifically affected or simply deprioritized isn't publicly documented.

### Phase 3: The devenv Revival (2023-2025)

**The catalyst:** CEO Tobias Lutke — himself a technical founder and active coder — personally discovered [devenv.sh](https://devenv.sh). He noticed its remarkable similarity to Shopify's original `dev` tool concept. Lutke adopted devenv for one of Shopify's services and became an active advocate for using Nix again.

**Why this time was different:**

1. **Executive sponsorship:** Having the CEO personally use and advocate for the tool eliminated the "bottom-up adoption stalls at middle management" failure mode. This is described as "catalytic" in conference reports.

2. **devenv as abstraction layer:** devenv provided what Shopify's custom Runix had tried to build — a high-level declarative interface over Nix — but maintained by an external community with proper documentation, broad language support, and ongoing development. Developers didn't need to understand Nix to use devenv.

3. **Incremental adoption strategy:** Unlike the first attempt's ambitious "Nix as common substrate" vision, the second approach was explicitly incremental. They migrated projects one at a time rather than attempting organization-wide rollout.

4. **Stakeholder engagement:** The team "spent much more time on a successful rollout within the organization, meaning incremental adoption and getting all stakeholders on board."

5. **Dissatisfaction with cloud dev:** Over time, developers became unhappy with cloud development environments. The latency, complexity, and constraints of cloud dev created demand for better local development — exactly what Nix excels at.

6. **Simultaneous monorepo transition:** Shopify moved from multi-repo to monorepo structure around the same time. devenv's monorepo support (later formalized in devenv 1.10) aligned well with this transition.

**Current state (as of NixCon 2025 / November 2025):**
- "The majority of development is now being done inside Nix-based environments"
- Hundreds of projects incrementally migrated
- Moved from cloud development to local development, multirepo to monorepo, Homebrew/Apt to Nix
- Burke Libbey still at Shopify, appeared on Full Time Nix podcast (Episode 67, November 2025) alongside Ashley Williams and Thomas Bereknyei
- Shopify actively hiring "Software Engineer - Monorepo Systems (Rust & Nix)"
- Josh Heinrichs presented the updated story at NixCon 2025 (September 5, 2025)

---

## 2. Scale

| Metric | First Attempt (2019-2020) | Current State (2025) |
|--------|--------------------------|---------------------|
| Developers | ~1,000 (NixCon 2019) | Thousands (exact number unpublished; Shopify total headcount ~8,300 as of 2023) |
| Platform | macOS laptops | Local development (macOS + Linux) |
| Projects | Multi-repo | Monorepo with hundreds of projects |
| Tooling | Custom: dev + shadowenv + Runix + binary cache | devenv.sh + existing infrastructure |
| Nix exposure | Direct (developers saw Nix expressions) | Abstracted (developers use devenv config) |
| Champion | Burke Libbey (Staff Engineer) | Tobias Lutke (CEO) + infrastructure team |
| CI integration | Planned, not shipped | In progress / active |

---

## 3. What Stalled Concretely

The first attempt didn't cleanly "fail" — it was a partial adoption that lost momentum. Specifically:

1. **Not all developers could use it:** Some team members were unable to get Nix working, likely due to macOS edge cases with native gem compilation and the complexity of debugging Nix build failures.

2. **Maintenance burden too high:** Shopify had to build and maintain extensive custom infrastructure (binary cache, build UI, Runix module system, shadowenv bridges) — essentially becoming a Nix tooling company within Shopify.

3. **Organizational priority shifted:** When cloud development became the company direction, resources and attention moved to Spin. The Nix effort was deprioritized rather than explicitly killed.

4. **Learning curve blocked expansion:** The engineers who could use Nix successfully were specialists. Expanding to general developers required either massive training investment or better abstractions — neither materialized before the cloud pivot.

---

## 4. What They Would Do Differently

Based on statements across the NixCon 2025 talk, trip report, and podcast:

1. **Start with a high-level abstraction** (like devenv) rather than raw Nix. The first attempt asked developers to understand too much Nix.

2. **Adopt incrementally** rather than attempting organization-wide rollout. "One specific, well-supported use-case can be the adoption driver."

3. **Get executive buy-in early.** Having CEO support eliminated organizational friction that doomed the first attempt.

4. **Use community-maintained tools** rather than building custom abstraction layers. Runix was conceptually similar to devenv but couldn't match a community project's documentation, breadth, and ongoing development.

5. **Prioritize developer experience over Nix purity.** The NixCon 2025 talk's central message is that DX matters more than technical elegance for adoption at scale.

---

## 5. Key People

- **Burke Libbey** — Staff Production Engineer / "Nixologist." Led the first Nix attempt (2018-2020). Created shadowenv, the Nixology series, Runix. Still at Shopify as of November 2025, participated in the Full Time Nix podcast discussing current Nix usage.

- **Josh Heinrichs** — Developer tooling engineer based in Saskatoon, Canada. Presented the NixCon 2025 talk. Works on the current devenv-based Nix adoption.

- **Tobias Lutke** — CEO and co-founder. Technical founder who personally discovered devenv and catalyzed the second adoption wave. Previously advised the Spin team that "developers shouldn't need to understand infrastructure implementation details."

- **Ashley Williams** and **Thomas Bereknyei** — Appeared alongside Burke on the Full Time Nix podcast (November 2025). Roles not specified in available sources.

---

## 6. Comparison: First Attempt vs. Second Attempt

| Dimension | First Attempt (2019-2020) | Second Attempt (2023-2025) |
|-----------|--------------------------|---------------------------|
| **Abstraction level** | Low — developers saw Nix expressions, `ru.nix` files, custom tooling | High — devenv provides declarative config without Nix language exposure |
| **Rollout strategy** | Ambitious: "Nix as common substrate for everything" | Incremental: one project at a time, stakeholder buy-in |
| **Sponsorship** | Bottom-up (Staff Engineer champion) | Top-down (CEO + infrastructure team) |
| **Custom tooling** | Heavy — binary cache, build UI, Runix, shadowenv bridges | Lighter — leveraging community devenv project |
| **Training approach** | Internal screencasts (Nixology), blog posts | Abstraction reduces training need; devenv documentation |
| **Competing paradigm** | Cloud development was gaining momentum | Cloud development had proven disappointing |
| **Outcome** | Stalled, superseded by Spin | Succeeded, now majority of development |

---

## 7. Lessons for Other Organizations

1. **Abstraction layers are not optional at scale.** Raw Nix works for small, motivated teams. For organizations with hundreds of developers at varying skill levels, a tool like devenv that hides Nix complexity is necessary.

2. **Executive sponsorship changes the game.** Burke Libbey's first attempt had deep technical merit but lacked organizational power. Tobi Lutke's personal advocacy removed friction that no amount of engineering could overcome.

3. **Timing and context matter.** The first attempt failed partly because cloud dev provided an apparently easier alternative. The second succeeded partly because cloud dev had proven disappointing. The right solution at the wrong time still fails.

4. **Don't build what the community will build.** Runix was a prescient design (strikingly similar to devenv), but maintaining a bespoke Nix abstraction layer internally is unsustainable. Community-maintained tools with active development provide better long-term support.

5. **Incremental adoption over big-bang migration.** "One specific, well-supported use-case can be the adoption driver." Once development environments use Nix, broader ecosystem adoption becomes feasible.

6. **Developer experience trumps technical purity.** This is Shopify's single most-repeated lesson across all sources.

---

## 8. Sources

| Source | Type | Date | File |
|--------|------|------|------|
| NixCon 2025 talk abstract (Josh Heinrichs) | Conference talk | 2025-09-05 | `docs/nixcon-2025-shopify-talk-abstract.md` |
| NixCon 2025 trip report (Michael Stapelberg) | Blog post | 2025-09-21 | `docs/nixcon-2025-trip-report-stapelberg.md` |
| Full Time Nix podcast E67 (Burke Libbey et al.) | Podcast | 2025-11-12 | `docs/fulltimenix-podcast-nix-at-shopify.md` |
| "How Shopify Uses Nix" (Burke Libbey, ShipIt!) | Blog post | 2020-05-25 | `docs/shopify-engineering-how-shopify-uses-nix.md` |
| "What Is Nix" (Burke Libbey) | Blog post | 2020-05 | `docs/shopify-engineering-what-is-nix.md` |
| NixOS Discourse hiring post (Burke) | Forum post | 2020-06-08 | `docs/nixos-discourse-shopify-nix-hiring.md` |
| NixCon 2019 code gist (Burke) | Source code | 2019-10 | `docs/burke-libbey-nixcon-2019-code-gist.md` |
| Runix example gist (Burke) | Source code | 2020-06-08 | `docs/burke-libbey-runix-gist.md` |
| Shopify Spin cloud dev journey | Blog post | 2022-06-09 | `docs/shopify-engineering-spin-cloud-dev-journey.md` |
| Shopify Isospin tooling | Blog post | 2022 | `docs/shopify-isospin-cloud-dev-tooling.md` |
| "NixOS Has One Fatal Flaw" (Tammer Saleh) | Blog post | Undated | `docs/changelog-nixos-fatal-flaw.md` |
| Debugging Nix gem build (Burke) | Blog post | ~2019 | `docs/burke-libbey-debugging-nix-gem.md` |
| Learning Nix (Burke) | Notes | 2018-12-17 | `docs/burke-libbey-learning-nix.md` |
| Shadowenv documentation | Docs | 2019 | `docs/shopify-shadowenv-docs.md` |
| HN discussion threads | Forum | 2020-05 | `docs/hn-shopify-nix-discussion.md` |

---

## Depth Checklist

- [x] **Underlying mechanism explained**: Full timeline of how Nix was integrated, what custom tooling was built, how devenv changed the approach
- [x] **Key tradeoffs and limitations identified**: Raw Nix complexity vs. abstraction, custom tooling maintenance burden, organizational priority competition with cloud dev
- [x] **Compared to alternative approaches**: First attempt (raw Nix + Runix) vs. second attempt (devenv); Nix vs. cloud dev (Spin)
- [x] **Failure modes and edge cases described**: macOS native gem compilation, bandwidth issues, learning curve barrier, champion-without-executive-support pattern
- [x] **Concrete examples found**: Runix config files, NixCon 2019 code gist (binary cache, build UI, shadowenv bridges), specific debugging examples
- [x] **Standalone-readable**: Yes — sufficient for understanding Shopify's full Nix journey without consulting original sources
