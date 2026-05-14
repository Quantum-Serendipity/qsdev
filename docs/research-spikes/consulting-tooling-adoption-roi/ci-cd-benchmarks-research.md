# CI/CD Build Time Benchmarks: Nix vs. Conventional Approaches

## Executive Summary

The claim that Nix achieves "50-75% CI build time reduction" is **not supported by published benchmarks**. No controlled study comparing Nix-based CI pipelines to conventional approaches (Docker, apt-get, brew) with measured before/after data was found. The claim appears to be an extrapolation from a single anecdotal data point (Ryan Rasti's 4-person team) combined with general caching intuition from Nix ecosystem companies. Conventional CI caching strategies (Docker layer caching, GitHub Actions cache) claim comparable percentage improvements (40-80%) without Nix's complexity overhead. The honest framing is: Nix's caching model has theoretical advantages for incremental builds, but the magnitude of real-world CI speedup is project-dependent, and no rigorous comparison to well-optimized conventional pipelines exists.

---

## 1. Origin of the "50-75% CI Build Time Reduction" Claim

### Where It Appears

The claim appears in `research-spikes/nix-consulting-environments/real-world-patterns-research.md` line 175: "Typical CI speedup: 50-75% reduction in build times." It is attributed to advice "consistent across Tweag, Determinate Systems, and community advice" but **no specific source document contains this figure with supporting data**.

### The Evidence Behind It

The gap analysis (`synthesized-reports/working/sq2-implicit-assumptions.md`) previously identified this as "weakly supported — one anecdotal data point extrapolated to a general claim." This research confirms that assessment.

The single concrete data point is **Ryan Rasti's 3-year production story**:
- 4-person team, full-stack Elixir/React
- Early CI: 3 minutes reduced to under 1 minute (~67% reduction)
- At scale: builds stayed under 15 minutes vs. estimated 30+ minutes without Nix (~50% reduction)
- Source: `docs/ryan-rasti-why-nix-will-win-3-year-production.md`

**Limitations of this data point:**
1. Single team, single project, single tech stack
2. Self-hosted GitHub Actions runners (improvements partially from infrastructure, not just Nix)
3. The "30+ minutes" baseline is an estimate of what builds *would* take without Nix, not a measured pre-Nix build time
4. 4 engineers is far from consulting firm or enterprise scale

### Verdict

**The "50-75%" range is unsubstantiated.** It cannot be traced to any published benchmark, controlled experiment, or multi-company study. It appears to be a rough generalization from the Rasti anecdote and general Nix caching principles. It should be flagged as such in any presentation or report.

---

## 2. Published Benchmark Data (Nix-to-Nix Comparisons)

The only rigorous benchmarks found compare **different Nix CI platforms against each other**, not Nix vs. conventional approaches.

### 2.1 Garnix CI Benchmarks

- **Source**: `docs/garnix-nix-ci-benchmarks.md`
- Compared: GitHub Actions (serial/parallel), magic-nix-cache, Cachix, nixbuild.net, Garnix
- Projects: agda/agda, crytic/echidna, helix-editor/helix (10 commits each)
- **Findings**: Garnix fastest, Cachix showed improvement, magic-nix-cache showed "no apparent benefit"
- **Limitation**: Created by Garnix staff (acknowledged bias). Interactive dashboard data failed to load — no numerical timing data extractable.
- **Critical gap**: Compares Nix CIs to each other, not to non-Nix alternatives.

### 2.2 Japanese Binary Cache Tools Comparison

- **Source**: `docs/nix-binary-cache-tools-comparison-github-actions.md`
- Compared cachix-action, cache-nix-action, magic-nix-cache-action in GitHub Actions
- **Best results** (cache-nix-action, warm cache, without official binary cache):
  - Job time: 45% of uncached baseline (55% reduction)
  - Build time: 17% of uncached baseline (83% reduction)
- **Worst results** (magic-nix-cache-action):
  - Job time: 100-148% of uncached baseline (no improvement or worse)
  - Cache generation overhead: up to 1174% of baseline
- **Critical context**: These compare **cached Nix builds to uncached Nix builds**, not Nix to Docker/conventional.

### 2.3 Determinate Systems Magic Nix Cache Claims

- **Source**: `docs/magic-nix-cache-determinate-systems.md`
- Claimed "30-50% reduction in Nix-related build times in Actions"
- **No benchmark data provided** — framed as a "confidence statement"
- **Contradicted** by both the Garnix benchmarks and the Japanese comparison, which found magic-nix-cache provided little to no improvement

---

## 3. Case Studies: Companies Using Nix for CI

### 3.1 Ryan Rasti / 4-Person Team (Elixir/React)

- **Source**: `docs/ryan-rasti-why-nix-will-win-3-year-production.md`
- Early: 3 min → under 1 min. At scale: under 15 min vs. estimated 30+ min
- Self-hosted runners with Nix caching
- **Tradeoffs**: Cross-platform Mac-to-Linux builds problematic, steep learning curve, custom tooling forks needed

### 3.2 Channable (Python/Haskell)

- **Source**: Web search results (full page fetch failed with 429)
- Database generation took ~30 minutes, impacting CI
- After Nix cache: "every target is built only once and developers and other CI runs can reuse the output from the cache"
- **No before/after CI pipeline times published** — only the database generation improvement is mentioned

### 3.3 Pinterest (iOS CI)

- **Source**: Web search results (Medium fetch failed with 403)
- Adopted Nix and Buildkite for iOS CI
- Replaced setup_environment.sh with Nix installation
- Claims: "saving CI capacity, and increasing developer productivity"
- **No specific build time measurements published**

### 3.4 Shopify (Rails Monorepo)

- **Source**: `docs/shopify-faster-ci-engineering-blog.md`
- Achieved p95 CI reduction from 45 min → 18 min (60% reduction)
- **This was achieved WITHOUT Nix** — used Docker optimization, MD5 hash caching, test selection
- Their separate Nix adoption (via devenv) is for developer environments, not CI pipeline optimization
- **Important comparator**: Shows conventional optimization achieving 60% CI reduction

---

## 4. Conventional CI Caching: Comparable Claims

This is critical context. Conventional caching strategies claim similar or greater percentage improvements:

### 4.1 Docker Layer Caching

- **Source**: `docs/docker-layer-caching-ci-netdata.md` and web search results
- Claims: "70% or more" build time reduction
- AWS CodeBuild: reported 98% reduction (24 min → 16 sec) with ECR remote cache
- Well-optimized Dockerfile: 7-8 min → 30 seconds
- Cache mount after layer invalidation: 8 min → 1 min 30 sec

### 4.2 GitHub Actions Cache (Non-Nix)

- Web search results (not separately saved — conventional caching guide data)
- Claims: "40-80%" reduction with proper caching
- Node.js example: 10-15 min → 60-90 sec with dependency and build caching
- Rust CI: 55% reduction in subsequent builds
- One case: 14 min → 1.75 min (87% reduction)

### 4.3 Implication

**Conventional CI optimization with proper caching achieves the same 40-80% range** that Nix ecosystem companies claim for Nix. The comparison should not be "Nix with cache vs. unoptimized conventional CI" but "Nix with cache vs. well-optimized conventional CI with proper caching." No benchmark makes this comparison.

---

## 5. Where Nix Has Theoretical CI Advantages

Despite the lack of benchmarks, Nix's caching model has structural properties that *should* produce better incremental build performance in certain scenarios:

### 5.1 Content-Addressable Storage

- Every build output stored at a hash-based path in `/nix/store`
- Changes to one dependency only trigger rebuilds of packages that actually depend on it
- Content-addressed derivations enable "early cutoff" — skipping rebuilds when output would be identical
- **Source**: Tweag blog on CA derivations (fetch failed, data from web search)

### 5.2 Granular Caching

- Nix caches at the package/derivation level, not the layer level
- Docker layer caching invalidates all subsequent layers when one changes
- Nix's DAG-based caching only rebuilds the exact subtree that changed
- **Theoretical advantage**: Significant for large projects with many independent dependencies

### 5.3 Cross-Project Cache Sharing

- Nix store deduplicates identical packages across projects
- Cachix/self-hosted binary cache serves as a shared artifact repository
- Multiple CI pipelines sharing dependencies benefit from a single build
- **Source**: `docs/adopting-nix-denny-britz.md`, `docs/jetify-nix-package-caches-intro.md`

### 5.4 Docker Image Optimization

- Nix can automatically split Docker images into optimal layers
- Unrelated images (PHP, MySQL) automatically share layers
- Image push/fetch times improved "by an order of magnitude" (Graham Christensen — no baseline data)
- **Source**: `docs/nix-layered-docker-images-grahamc.md`

---

## 6. Where Nix Makes CI SLOWER

### 6.1 Cold Build / No Cache

- Without binary cache, Nix builds everything from source
- MongoDB alone: up to 30 minutes from source
- Full closure from source: can take hours
- **Source**: `docs/jetify-nix-package-caches-intro.md`

### 6.2 Evaluation Overhead

- NixOS/nixpkgs evaluation: 0.4s (2015) → 3s (2025) — 7.5x slowdown over 10 years
- Large flakes: evaluation alone can take seconds to minutes before building starts
- `nix develop` in CI: >2 minutes overhead reported
- **Sources**: `docs/nixos-nixpkgs-evaluation-times-discourse.md`, `docs/why-avoid-nix-docker-images-mccurdyc.md`

### 6.3 Nix Store Locking

- Running `nix develop --command` in parallel fails due to SQLite database busy errors
- Prevents concurrent task execution in CI
- **Source**: `docs/why-avoid-nix-docker-images-mccurdyc.md`

### 6.4 Cache Generation Overhead

- First-time cache population can take 4-11x longer than uncached builds
- magic-nix-cache-action cache generation: 428s vs. 36s baseline (1174% overhead)
- **Source**: `docs/nix-binary-cache-tools-comparison-github-actions.md`

### 6.5 Large Closure Transfer

- CI setups using remote builders copy the entire transitive closure back
- Can be wasteful when you only need pass/fail results
- **Source**: NixOS Discourse on remote builders (web search data)

### 6.6 Complexity Overhead

- Learning curve for CI configuration
- Maintaining Nix expressions alongside conventional build files
- Debugging Nix build failures requires Nix expertise
- Organizations not fully committed to Nix face dual-maintenance burden
- **Source**: `docs/why-avoid-nix-docker-images-mccurdyc.md`

---

## 7. Cost Comparison

### 7.1 CI Compute Costs

- GitHub Actions: ~40x more expensive than Garnix or nixbuild.net for closed-source repos (Garnix benchmark author's estimate)
- This is a comparison of Nix CI platforms to GitHub Actions pricing, not a Nix vs. non-Nix comparison
- **Source**: `docs/garnix-benchmarks-discourse-discussion.md`

### 7.2 Hardware Impact

- More powerful hardware reduces build times regardless of Nix vs. Docker
- GitHub runner (4 min 20s) vs. Actuated runner (2 min 15s) for same Nix build — 50% faster on better hardware
- ARM QEMU emulation (55 min) vs. native ARM (3 min 29s) — 16x faster
- **Source**: `docs/actuated-faster-nix-builds-github-actions.md`
- **Caveat**: These improvements are hardware-driven, not Nix-specific

### 7.3 Complexity Cost

- No published data on Nix CI maintenance overhead in engineering hours
- Anecdotal: self-hosted runners required "setup and ongoing maintenance" (Rasti)
- Organizations need at least 1-2 Nix experts to maintain CI configurations
- Complexity cost may offset CI time savings for small teams or short-term projects

---

## 8. Evidence Gap Analysis

### What We Searched For vs. What We Found

| Evidence Type | Searched | Found |
|---|---|---|
| Controlled Nix vs. Docker CI benchmark | Yes | **None** |
| Multi-company Nix CI adoption study | Yes | **None** |
| Published before/after CI metrics | Yes | **1 anecdote** (Rasti) |
| Nix ecosystem company benchmarks | Yes | **Claims without data** |
| Nix-to-Nix CI platform comparisons | Yes | 2 studies (Garnix, Japanese comparison) |
| Conventional caching benchmarks | Yes | Multiple (40-80% claims) |
| Cases where Nix slows CI | Yes | Multiple documented issues |
| Conference talks with measured data | Yes | **None with published metrics** |

### Why This Gap Exists

1. **Nix ecosystem companies** (Cachix, Determinate Systems, Numtide, Flox) market benefits qualitatively or compare Nix tools to each other, not to conventional alternatives
2. **Companies using Nix** report satisfaction anecdotally but don't publish controlled benchmarks
3. **The comparison is genuinely hard** — "Nix CI" and "conventional CI" are not single configurations; performance depends on project type, caching strategy, infrastructure, and optimization effort
4. **Selection bias** — companies writing about Nix CI are those for whom it worked; failures are underreported

---

## 9. Conclusions

### 9.1 The Claim Is Unsubstantiated

The "50-75% CI build time reduction" claim cannot be traced to any published benchmark, controlled experiment, or multi-company study. It should be presented as an **extrapolation from limited anecdotal data**, not as a validated finding.

### 9.2 The Mechanism Is Plausible

Nix's content-addressable, DAG-based caching model has structural properties that should produce better incremental build caching than Docker's layer-based approach for certain project types (large dependency trees, many independent packages, multi-project organizations). The theoretical advantage is real but unquantified.

### 9.3 Conventional Alternatives Achieve Similar Results

Docker layer caching, GitHub Actions cache, and other conventional CI optimization strategies claim comparable percentage improvements (40-80%). Without a controlled comparison, there is no basis for claiming Nix is materially better than well-optimized conventional CI.

### 9.4 Nix Adds CI Overhead That Must Be Amortized

Cold builds, evaluation overhead, cache generation costs, and Nix expertise requirements create upfront and ongoing costs. These must be amortized over enough builds and enough projects to produce net savings. For short-term consulting engagements with small teams, the payback period may exceed the project duration.

### 9.5 Recommended Honest Framing

Instead of "50-75% CI build time reduction," the defensible claim is:

> "Nix's caching model can significantly reduce incremental CI build times for projects with large dependency trees, particularly when builds are shared across multiple projects via a binary cache. One team reported builds staying under 15 minutes for a project that would otherwise exceed 30 minutes. However, no controlled benchmark comparing Nix CI to well-optimized conventional CI exists, and conventional caching strategies claim comparable improvements. Initial setup and cold build times may be significantly longer with Nix."

---

## Sources

All source documents saved to `docs/`:

| File | Source | Key Data |
|---|---|---|
| `garnix-nix-ci-benchmarks.md` | garnix-io.github.io | Nix CI platform comparison (no numerical data extracted) |
| `garnix-benchmarks-discourse-discussion.md` | discourse.nixos.org | Community discussion, cost comparison |
| `nix-binary-cache-tools-comparison-github-actions.md` | zenn.dev | Detailed cache tool benchmark with specific numbers |
| `nix-based-continuous-integration-compilersaysno.md` | compilersaysno.com | Architectural guide, 10s savings |
| `magic-nix-cache-determinate-systems.md` | determinate.systems | 30-50% claim without data, contradicted by other benchmarks |
| `adopting-nix-denny-britz.md` | dennybritz.com | Docker image build qualitative comparison |
| `hacker-news-nix-ci-pipeline-discussion.md` | news.ycombinator.com | No quantitative data in discussion |
| `ryan-rasti-why-nix-will-win-3-year-production.md` | ryanrasti.com | Primary anecdotal data: 3min→1min, 30+min→15min |
| `fast-ci-build-with-nix-quentin-dufour.md` | quentin.dufour.io | Architectural analysis, no Nix-specific benchmarks |
| `jetify-nix-package-caches-intro.md` | jetify.com | Cold build costs: MongoDB 30min, full closure hours |
| `actuated-faster-nix-builds-github-actions.md` | actuated.com | Hardware comparison data (not Nix-specific advantage) |
| `nixos-nixpkgs-evaluation-times-discourse.md` | discourse.nixos.org | Evaluation overhead: 0.4s→3s over 10 years |
| `shopify-faster-ci-engineering-blog.md` | shopify.engineering | 60% CI reduction WITHOUT Nix |
| `nix-layered-docker-images-grahamc.md` | grahamc.com | Docker layer optimization, "order of magnitude" claim |
| `numtide-nix-docker-or-both.md` | numtide.com | No performance data |
| `why-avoid-nix-docker-images-mccurdyc.md` | mccurdyc.dev | Counter-argument: >2min overhead, parallelization issues |
| `docker-layer-caching-ci-netdata.md` | netdata.cloud | Docker caching claims 70%+ reduction |
