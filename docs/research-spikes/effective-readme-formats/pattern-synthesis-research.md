# Cross-Cutting Pattern Synthesis: The Effective Developer Tool README

## Overview

This report synthesizes findings from four Phase 2 investigations — ecosystem analysis of 20 developer tool READMEs (10 CLI, 10 build/runtime), literature review of 20+ best-practices guides, and deep-dive into documentation UX and adoption psychology research. The goal is a unified, actionable framework for writing developer tool READMEs that capture attention, communicate value, and drive adoption.

Total evidence base: 20 tools analyzed, 57+ sources saved, 3 academic studies, 11 community guides surveyed, industry research from Stack Overflow, NNGroup, Google UX, and DX.

---

## Part 1: The README as Conversion Funnel

### The Core Insight

Every report converges on a single finding: **a README is a landing page, not a manual.** The reader arrives via GitHub trending, a search result, a peer's link, or a Hacker News thread. They have not committed to anything. They are deciding — in seconds — whether to invest further.

The mechanisms behind this are well-documented:

- **50ms aesthetic judgment** (Google UX research) — Visual quality and perceived usability are assessed before a single word is read
- **F-pattern scanning** (NNGroup, replicated 2006-2024) — First horizontal line, shorter second line, then vertical left-side scan. The first line after the title is the single most-read text in the entire README
- **80% of viewing time above the fold** (NNGroup) — Content that requires scrolling gets a fraction of the attention
- **60-second evaluation window** (Bugayenko, Daytona, practitioner consensus) — If the reader can't understand what the tool does and how to try it within ~60 seconds, they leave
- **15-minute time-to-first-value** (daily.dev / TTV research) — Tools that fail to deliver a concrete result within 15 minutes of first use are abandoned

The implication: the README's job is to move a skeptical evaluator through a conversion funnel, not to document the tool comprehensively.

### The Conversion Funnel

```
DISCOVERY (Trending / Search / Word-of-mouth / HN / Reddit)
  ↓
LAND ON REPO (GitHub page loads — 50ms visual judgment)
  ↓
FIRST SCAN (0-5 seconds: title, one-liner, hero visual)
  → Decision: "Is this relevant to me?" → Leave or continue
  ↓
QUICK EVALUATION (5-30 seconds: value prop, comparison, key features)
  → Decision: "Is this better than what I have?" → Leave or continue
  ↓
TRY IT (30 seconds - 5 minutes: install + quick-start)
  → Decision: "Can I get this working easily?" → Leave or continue
  ↓
FIRST VALUE (5-15 minutes: first real use producing visible result)
  → Decision: "Does this actually solve my problem?" → Abandon or adopt
  ↓
RECOMMEND TO PEERS (viral loop — README must be shareable)
```

71% of developers ask colleagues when evaluating tools (Stack Overflow 2023). This means the README must be concise enough to link with confidence that the recipient will quickly understand the value.

---

## Part 2: The Optimal README Structure

### Evidence-Backed Section Ordering

Synthesizing across all four reports, the following structure maximizes conversion at each funnel stage. Items are ordered by the reader's decision sequence, not by information taxonomy.

```
┌─────────────────────────────────────────────────────┐
│  ABOVE THE FOLD — First Screenful (~25-30 lines)    │
│                                                      │
│  1. Logo / Visual Identity (0-2 lines)              │
│  2. One-Liner Value Proposition (1 line)            │
│  3. Badges (3-5 max: build, version, license)       │
│  4. Hero Visual (screenshot / GIF / benchmark)      │
│  5. Key Differentiator (comparison or bold claim)   │
├─────────────────────────────────────────────────────┤
│  THE QUICK WIN — Next 20-30 lines                   │
│                                                      │
│  6. Feature Highlights (3-7 bullets, benefit-framed) │
│  7. Quick-Start (3-5 commands, copy-paste ready)    │
│  8. Installation (primary method first, others in   │
│     collapsible <details> blocks)                   │
├─────────────────────────────────────────────────────┤
│  THE DEPTH LAYER — For committed readers             │
│                                                      │
│  9. Detailed Features / Usage Examples              │
│  10. Configuration                                   │
│  11. Integration with Other Tools                   │
│  12. Comparison / Alternatives                      │
│  13. Documentation Links                            │
├─────────────────────────────────────────────────────┤
│  TRUST & COMMUNITY — Bottom                         │
│                                                      │
│  14. Contributing                                    │
│  15. Sponsors (if any — never above the fold)       │
│  16. License                                         │
└─────────────────────────────────────────────────────┘
```

### Constraint: Items 1-7 Must Fit in Two Screenfuls

This is not arbitrary. The F-pattern research shows scanning drops precipitously after the second horizontal sweep. If a reader has to scroll past two screenfuls to reach the quick-start, the majority will never reach it.

### Why This Ordering (Not the "Logical" One)

The standard guide consensus puts Installation before Usage (10/11 guides). But the ecosystem analysis found the most effective READMEs (bat, fd, uv) show what the tool does *before* asking you to install it. The UX research explains why: **opportunistic learners** (the majority) won't install until they've confirmed the tool is worth trying. Visual proof and a quick-start example create that confirmation.

The resolution: Quick-Start (showing what you'll be able to do) should precede or be interleaved with Installation. The installation section exists for reference; the quick-start creates motivation.

---

## Part 3: The 12 Winning Patterns

These patterns emerged consistently across the ecosystem analyses and are backed by the literature and UX research. Ordered by impact.

### Pattern 1: The One-Liner Value Proposition

**What**: A single sentence (under 15 words) that communicates what the tool does and its primary differentiator.

**Evidence**: 10/11 best-practices guides recommend it. Every top-ranked tool across both ecosystem analyses has one. F-pattern research confirms the first line after the title receives the most attention.

**Examples (ranked by effectiveness)**:
- bat: "A cat(1) clone with syntax highlighting and Git integration" — names what it replaces and what it adds
- uv: "An extremely fast Python package and project manager, written in Rust" — names the domain, the differentiator (speed), and the credibility signal (Rust)
- fd: "A simple, fast and user-friendly alternative to 'find'" — positions against the tool it replaces

**Anti-pattern**: Starting with project history, organizational context, or generic category descriptions ("A powerful next-generation tool for...").

### Pattern 2: Visual Proof Before Text Arguments

**What**: A screenshot, GIF, terminal recording, or benchmark chart placed immediately after the title/one-liner — before any prose explanation.

**Evidence**: 8/11 guides recommend visual demos. bat's "Feature Gallery" (3 screenshots as primary argument) is the #1-ranked CLI README element. uv's benchmark chart is the centerpiece of the #1-ranked build README. NNGroup's 50ms judgment research confirms visual quality is assessed before text is read. Venigalla & Chimalakonda (2022) found images in READMEs correlate with higher repository popularity.

**Best practices by tool type**:
- CLI tools: Terminal recording GIF (15-30s max) — bat, fzf, lazygit, starship, mise
- Build/speed tools: Benchmark bar chart with dark/light mode variants — uv, esbuild, Ruff
- Enhancement tools: Before/after comparison screenshots — delta

**Technical note**: Use `<picture><source media="(prefers-color-scheme: dark)">` for dark/light mode image variants. This is supported on GitHub and provides a polished feel.

### Pattern 3: The Concrete Comparison

**What**: A side-by-side showing the old way vs. the new way — the tool you're replacing vs. this tool.

**Evidence**: The ecosystem analysis called fd's `fd PATTERN` vs `find -iname '*PATTERN*'` comparison "the single most effective attention-capture element across all 20 tools analyzed." Only 3/10 CLI tools and 3/10 build tools explicitly compare — but every one that does ranks in the top half.

**Variants**:
- **Syntax comparison** (fd vs find): Old command vs new command, side by side
- **"Replaces N tools" framing** (uv, Ruff): "A single tool that replaces pip, pip-tools, pipx, poetry, pyenv, twine, virtualenv" — answers "why switch?" and "what can I uninstall?" simultaneously
- **Speed comparison** (ripgrep, esbuild): Concrete numbers (0.082s vs 0.273s) or benchmark charts
- **Qualitative comparison** (bat vs cat): Visual screenshot showing enhanced output vs plain output

**Why it works**: The UX research identifies "Project requirements fit" as the #1 factor in tool selection. A concrete comparison answers "does this solve my specific problem better than what I have?" with zero cognitive effort.

### Pattern 4: The 30-Second Quick-Start

**What**: A minimal working example (3-5 commands) that takes the reader from zero to first visible result in under 30 seconds of terminal time.

**Evidence**: The 15-minute TTV rule means the README must make the path to first value feel achievable. The best examples: fzf's three-keybinding pattern (CTRL-T, CTRL-R, ALT-C — immediate use), Deno's 3-step web server, uv's `uvx pycowsay` with delightful ASCII output, mise's progressive 6-scenario tutorial.

**Requirements**:
- Copy-paste ready — no placeholder values requiring editing
- Show expected output (especially when output is the value proposition)
- Maximum 5 commands
- First command should produce a visible result
- Include timing information if speed is the differentiator (uv embeds millisecond timings in console output)

**Anti-pattern**: "See the documentation for getting started" (Turborepo). A quick-start deferred to external docs has a ~90% drop-off (per funnel conversion research).

### Pattern 5: Platform-Aware Installation with Progressive Disclosure

**What**: Lead with the most common installation method as a single copy-paste command. Put alternative methods in collapsible `<details>` blocks.

**Evidence**: zoxide's 4-step numbered installation with collapsible platform sections was ranked the best installation flow across all 20 tools. The UX research warns that "multiple CTAs create decision fatigue" — five installation methods upfront overwhelms rather than helps. bat's 15+ platform coverage with Repology badge handles the breadth problem.

**Structure**:
```markdown
## Install

```bash
curl -LsSf https://example.com/install.sh | sh
```

<details><summary>Other installation methods</summary>

### Homebrew
```bash
brew install example
```

### Cargo
```bash
cargo install example
```

### From source
...

</details>
```

**Trust-building extras**: Platform-specific gotcha warnings (fd's `fdfind` on Debian, bat's `batcat` naming), Repology badges showing package availability, Linux kernel version requirements (Bun).

### Pattern 6: Benefit-Framed Feature Bullets

**What**: 3-7 features described in terms of user benefit, not technical capability. Bold keyword + one-line explanation.

**Evidence**: fzf's four-bullet pattern (**Portable**, **Fast**, **Versatile**, **All-inclusive**) was called "a template worth copying." starship's six benefit bullets create a scannable value proposition that works even if you read nothing else. The UX research emphasizes problem-solving framing over feature-listing: "Supports X, Y, Z" answers "what does it have?" before "why should I care?"

**Format**:
```markdown
- **Fast** — 10-100x faster than alternatives ([benchmarks](link))
- **Compatible** — Drop-in replacement for existing workflows
- **Cross-platform** — macOS, Linux, Windows out of the box
```

**Anti-pattern**: Undifferentiated feature lists ("supports JSON, YAML, TOML, XML, CSV...") without explaining why any of those matter to the reader.

### Pattern 7: Social Proof That Developers Trust

**What**: Evidence of adoption and quality calibrated to what developers actually find persuasive.

**Evidence**: Shen & Sood (2025) found GitHub stars have **limited persuasive power** on actual adoption when controlling for utility. Developers face tangible consequences from poor tool choices, driving central-route (analytical) processing rather than peripheral-cue (heuristic) processing. Stars serve mainly as a filtering heuristic (below ~500 dismissed, above that diminishing returns).

**Effectiveness ranking (most to least persuasive)**:
1. **Named expert testimonials** — Ruff's quotes from FastAPI, isort, Conda creators. "1000x faster. Literally. Not a typo." from Nick Schrock was called "the most compelling testimonial in any README analyzed"
2. **"Used by" logos** — Imply production validation at known organizations
3. **Download/install counts** — Practical adoption evidence
4. **CI/build badges** — Maintenance commitment signal
5. **Recent release date** — Active development signal
6. **Star count** — Filtering heuristic only; diminishing returns above ~500

**Anti-pattern**: Badge walls (10+ badges), sponsor logos above the value proposition, star-count-as-centerpiece marketing.

### Pattern 8: The Emotional Hook

**What**: An opening that creates an emotional connection — frustration, excitement, or recognition — before presenting the technical solution.

**Evidence**: lazygit's "Elevator Pitch" (a profanity-laden rant about git's UX failures) was the only README that "makes you feel something before showing you anything" and has 55k stars despite significant structural anti-patterns. esbuild's "Our current build tools for the web are 10-100x slower than they could be" frames a problem that creates urgency.

**Four copywriting frameworks** (from UX research):
1. **Problem-Solution**: "Tired of slow grep? ripgrep searches your code 10x faster."
2. **Benefit-Driven**: "Deploy in seconds" (not "Uses containerized microservices")
3. **Question Hook**: "What if your terminal could show syntax-highlighted diffs?"
4. **Bold Claim**: "The last CSS framework you'll need" — works only with credible evidence

**Calibration**: Developer audiences are uniquely skeptical of marketing language and uniquely responsive to working code. The emotional hook must be authentic and quickly followed by concrete evidence. lazygit works because the frustration is universal and genuine; a corporate-sounding "revolutionize your workflow" would not.

### Pattern 9: Integration & Ecosystem Positioning

**What**: A section showing how the tool works with other tools the reader already uses.

**Evidence**: bat's "Integration with other tools" section (fzf, fd, ripgrep, tail, git, man, prettier) signals ecosystem maturity and reduces adoption risk. zoxide's 20+ tool integration table does the same. The "ecosystem play" converts "another tool to learn" into "enhancement for your existing workflow."

**Why it works**: Tech stack compatibility is factor #6 in developer tool selection (Stack Overflow 2023). An integration section proactively answers "will this work with my setup?" without requiring the reader to investigate.

### Pattern 10: Configuration-as-Quickstart

**What**: For tools that enhance existing workflows (git diff enhancers, prompt customizers, shell plugins), showing the configuration change IS the quick-start. No new commands to learn.

**Evidence**: delta's Getting Started section opens with gitconfig changes — the first thing you see is actionable setup. After adding 3 lines to `.gitconfig`, your existing `git diff` is enhanced. Zero new commands to memorize.

**When to use**: When the tool is an enhancement (not a replacement) for something the user already does. The message is: "you don't have to learn anything new — just paste this config."

### Pattern 11: Try-Before-Install

**What**: A way to experience the tool without installing it — online playgrounds, `npx`/`uvx`/`nix run` commands, or web demos.

**Evidence**: Identified as "underutilized across the ecosystem." eza's `nix run github:eza-community/eza` and jq's play.jqlang.org both enable zero-commitment evaluation. Biome and Ruff offer online playgrounds. Stack Overflow 2023 found "starting a free trial" is the most common evaluation method — and try-before-install is the developer tool equivalent.

**The lowest-friction path to first value**. Especially powerful for tools where the value is immediately visible in output (formatters, linters, shell tools).

### Pattern 12: Honest Positioning & Non-Goals

**What**: Explicitly stating what the tool is NOT good at or NOT designed for.

**Evidence**: The UX research found that "honest positioning (acknowledging what tool is NOT good at) builds credibility faster than overclaiming." Nushell's status section admitting instability was noted positively. The benchmark credibility case study (context-router) showed that correcting an inflated claim "earned more credibility than the original."

**Why it works**: Developers process tool claims via central-route (analytical) processing, not peripheral (heuristic). Acknowledging limitations signals that the rest of the claims are trustworthy. A tool that claims to be perfect at everything is trusted less than one that honestly scopes its strengths.

---

## Part 4: The 11 Anti-Patterns

### Critical Anti-Patterns (directly cause reader abandonment)

**A1. No README or near-empty README**
Changelog's "Top Ten Reasons" puts this as #1 reason developers reject a project. Turborepo's one-sentence-plus-redirect README was rated worst in the build/runtime set. Academic research (Venigalla & Chimalakonda 2022) confirms README quality correlates with repository popularity.

**A2. Sponsor banners / avatars above the value proposition**
lazygit, eza, zoxide all place sponsor content above the tool's value proposition, consuming 200-800px of the most valuable real estate. With 80% of viewing time above the fold, this is literally pushing the product below the reader's attention threshold.

**A3. Installation deferred to external files or websites**
eza → INSTALL.md, jq → website, Turborepo → turborepo.dev, esbuild → esbuild.github.io. The README is often the ONLY page a user reads. Every redirect is a funnel drop-off point. Include at minimum the single most common installation method.

**A4. Missing quick-start examples**
jq, eza, Turborepo, pnpm — tools with no working example in the README. Even a single 3-line example is dramatically better than zero. The 15-minute TTV rule means readers must be able to mentally project the path to first value.

### Structural Anti-Patterns (reduce effectiveness)

**A5. README as documentation index / sitemap**
Bun's 150+ categorized links to docs pages turn the README into a table of contents. This creates severe cognitive overload and pushes all meaningful content off-screen. The README is the lobby, not the building's directory.

**A6. Wall-of-text features without visual breaks**
ripgrep relies entirely on text for its value proposition. Lazygit's 1500-line README with 15 GIFs but no collapsible sections. Both represent extremes. The research recommends: headed sections, bullets, code blocks, and `<details>` tags to maintain scannability.

**A7. Feature-first framing instead of problem-first**
"Supports JSON, YAML, TOML, XML, CSV" vs "Parse any config format with one command." The UX research is clear: problem-solving framing captures attention; feature-listing answers a question the reader hasn't asked yet.

### Trust-Damaging Anti-Patterns

**A8. Stale or broken examples**
Wang et al. (2023, empirical) found README update frequency correlates with popularity. The UX report: "Outdated examples are actively harmful — worse than no documentation." Examples become misleading within 6-12 months without validation.

**A9. Inflated or unsubstantiated benchmark claims**
The context-router case study demonstrated that inflated claims, once corrected, damage credibility. Effective benchmarks show methodology, use reproducible scripts, and acknowledge limitations. "If you can't tell whether both systems saw the same input, treat the result as marketing."

**A10. Assumed context and jargon**
Words like "obviously" and "simply" risk making readers feel inadequate. Domain-specific jargon without explanation excludes newcomers. The README must work for someone encountering the tool for the first time.

**A11. Missing license**
Changelog: developers on commercial products "won't even look at projects" without common licenses. 10/11 best-practices guides list license as essential. Standard Readme mandates it as the final section.

---

## Part 5: Cognitive Science Behind the Structure

### Why This Order Works (Not Just "What" But "How")

The recommended structure maps directly to three cognitive science frameworks:

**1. Cognitive Funneling** (Art of README / hackergrrl)
Information flows from broadest (name + one-liner) to most specific (configuration, API). Each level filters the audience — only those still interested proceed. This matches how evaluation actually works: broad fit → specific fit → commitment.

**2. Progressive Disclosure** (NNGroup)
Show "only a few of the most important options" initially, reveal more on request. Three levels:
- **Level 1 (visible)**: One-liner, hero visual, install, quick-start
- **Level 2 (one click)**: Detailed features, config, comparison tables (via `<details>` tags)
- **Level 3 (linked out)**: Full docs site, architecture, contributing

**3. Cognitive Load Theory** (Sweller 1988)
Three types of load: intrinsic (content complexity), extraneous (presentation burden), germane (productive understanding). A well-structured README minimizes extraneous load (clean formatting, scannable structure, visual proof) so cognitive resources go to evaluating the tool itself.

### Three Developer Learning Styles (Meng et al. 2019)

The README must serve all three simultaneously:
- **Systematic**: Comprehensive overview → feature list → installation → detailed usage
- **Opportunistic**: Copy-paste quick-start → working example → explore from there
- **Pragmatic**: Scan headings → jump to relevant section → deep-dive as needed

The recommended structure accommodates all three: systematic readers follow top-to-bottom, opportunistic readers jump to quick-start, pragmatic readers scan headings (which must be self-sufficient and descriptive).

---

## Part 6: The Effectiveness Spectrum

### What Separates Great From Adequate

Across all 20 tools analyzed, a clear spectrum emerges:

**Tier 1 — Exceptional (converts skeptics)**
- uv, bat, fd, esbuild (opening only), Ruff
- Characteristics: One-liner that names what it replaces, visual proof as primary argument, concrete comparison, inline evidence (timings, benchmarks), quick-start producing visible result

**Tier 2 — Strong (converts the curious)**
- fzf, starship, mise, zoxide, delta, Deno
- Characteristics: Clear value prop, good visual elements, working quick-start, but missing one or more Tier 1 elements (usually the concrete comparison or inline evidence)

**Tier 3 — Adequate (converts the already-motivated)**
- lazygit, ripgrep, Bun, Biome, Nushell, pnpm
- Characteristics: The tool's quality carries the README rather than the README selling the tool. Significant structural issues (wall of text, deferred install, buried differentiator, sponsor dominance) that would fail for a lesser-known tool

**Tier 4 — Insufficient (depends entirely on reputation)**
- jq, eza, Turborepo
- Characteristics: Missing fundamental elements (quick-start, installation, features). Survives on brand recognition or category dominance. Would fail entirely for a new or unknown tool

**The key insight**: Tier 3 and 4 tools often have high star counts despite their READMEs, not because of them. jq (31k stars) and Turborepo (Vercel brand) succeed through category dominance and corporate backing. A new tool with a Tier 3 or 4 README will not achieve discovery-driven adoption.

---

## Part 7: Actionable README Template

### The Template

This template encodes all 12 patterns and avoids all 11 anti-patterns. Sections in brackets are conditional.

```markdown
<p align="center">
  <img src="logo.svg" alt="Tool Name" width="200">
</p>

<h1 align="center">Tool Name</h1>

<p align="center">
  <strong>One-liner: what it does + primary differentiator (under 15 words)</strong>
</p>

<p align="center">
  <a href="ci-link"><img src="build-badge" alt="Build"></a>
  <a href="version-link"><img src="version-badge" alt="Version"></a>
  <a href="license-link"><img src="license-badge" alt="License"></a>
</p>

<!-- Hero Visual: screenshot, GIF, or benchmark chart -->
<!-- Use <picture><source> for dark/light mode variants -->
<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="demo-dark.png">
    <img src="demo-light.png" alt="Demo" width="600">
  </picture>
</p>

<!-- [Optional] Concrete comparison: old way vs new way -->
<!-- This is the single most effective persuasion element -->

## Highlights

- **Benefit One** — User-facing outcome, not technical detail
- **Benefit Two** — Concrete claim with number if possible
- **Benefit Three** — Addresses a pain point the reader recognizes
- [**Replaces X, Y, Z**] — If applicable, name what it consolidates

## Quick Start

```bash
# Install (one command, most common method)
curl -LsSf https://example.com/install.sh | sh

# First use — produces visible result
tool-name hello-world
```

Expected output:
```
✓ Hello, world! (completed in 42ms)
```

[## Try Without Installing]

```bash
npx tool-name      # or uvx, nix run, etc.
```

[Online playground: https://playground.example.com]

## Install

<details><summary>macOS</summary>

```bash
brew install tool-name
```
</details>

<details><summary>Linux</summary>

```bash
# Debian/Ubuntu (note: package may be named tool-name-bin)
apt install tool-name

# Arch
pacman -S tool-name
```
</details>

<details><summary>Windows</summary>

```powershell
scoop install tool-name
```
</details>

<details><summary>From source</summary>

```bash
cargo install tool-name
```
</details>

## Usage

<!-- Progressive examples: simple → intermediate → real-world -->
<!-- Show expected output for each -->

### Basic

```bash
tool-name file.txt
```

### With Options

```bash
tool-name --format json file.txt
```

### Real-World Example

```bash
# Solve an actual problem the reader recognizes
tool-name --recursive src/ | grep TODO
```

[## Integrations]

<!-- How it works with tools the reader already uses -->

| Tool | Integration |
|------|-------------|
| fzf  | `tool-name | fzf` for interactive selection |
| git  | Add to `.gitconfig` for enhanced diffs |

[## Comparison]

<!-- Honest comparison to alternatives -->
<!-- Only include if you have genuine differentiators -->

| Feature | tool-name | alternative |
|---------|-----------|-------------|
| Speed   | 42ms      | 1,200ms     |
| Config  | Zero      | .rc file    |

[## Testimonials]

> "Specific quote from a known community figure."
> — **Name**, Creator of Well-Known Project

## Documentation

Full documentation: [docs.example.com](https://docs.example.com)

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[MIT](LICENSE) — or whichever license applies.
```

### Template Usage Notes

1. **Sections in `[brackets]` are conditional** — include only when you have strong content for them. An empty comparison table is worse than no comparison.
2. **The hero visual is the single most impactful element.** If you can only do one thing, do this. A terminal GIF for CLI tools, a benchmark chart for speed-focused tools.
3. **The one-liner must name what the tool replaces or improves.** "A fast X" is weaker than "A faster alternative to Y." Positioning against the status quo is the strongest framing.
4. **Quick-start before Install** — The quick-start includes an install command. The separate Install section is for alternative methods and platform coverage. Don't make people scroll past 15 package managers to find usage examples.
5. **Console output with timing** is more persuasive than external benchmarks because it's what the user will actually experience (uv's pattern).
6. **Three examples minimum** in Usage: basic (proves it works), with options (shows depth), real-world (proves it solves actual problems). mise's terraform example was the only "real production scenario" across 20 READMEs — this is a massively underutilized pattern.
7. **Sponsor sections belong at the bottom**, never above the value proposition.

---

## Part 8: Platform Considerations

### Cross-Platform Rendering

READMEs render differently across platforms. The safe subset:

| Feature | GitHub | npm | PyPI | crates.io |
|---------|--------|-----|------|-----------|
| Tables | Yes | Yes | Yes | Yes |
| Code blocks | Yes | Yes | Yes | Yes |
| Images (URL) | Yes | Yes | Yes | Yes |
| `<details>` | Yes | Partial | Yes | No |
| `<picture>` | Yes | No | No | No |
| Mermaid diagrams | Yes | No | No | No |
| Alert callouts | Yes | No | No | No |
| Footnotes | Yes | No | No | No |

**Strategy**: Use standard CommonMark as the base. GitHub-specific features (`<picture>`, alerts, Mermaid) are bonuses for the primary audience. Keep images as hosted URLs for registry display. Test rendering on your target registries.

### GitHub-Specific Enhancements Worth Using

- `<details><summary>` for collapsible sections (progressive disclosure)
- `<picture><source media="(prefers-color-scheme: dark)">` for dark/light images
- Alert callouts: `> [!NOTE]`, `> [!TIP]`, `> [!WARNING]`, `> [!IMPORTANT]`, `> [!CAUTION]`
- Auto-ToC via GitHub's outline button (reduces need for manual ToC)

---

## Part 9: Evidence Quality Assessment

### What We Know With Confidence (empirical + practitioner convergence)

- README structural quality (lists, images, organization, freshness) correlates with repository popularity (Venigalla & Chimalakonda 2022, Wang et al. 2023 — 6,950 repos total)
- "What" and "Why" sections are systematically underrepresented in READMEs despite being critical for evaluation (Prana et al. 2019, 4,226 sections)
- Developers evaluate tools hands-on first, rely on peer recommendations (71%), and abandon tools that don't demonstrate value within ~15 minutes
- Visual elements receive attention before text (50ms judgment, F-pattern scanning)
- Stars have limited causal effect on adoption when controlling for utility (Shen & Sood 2025)
- Documentation quality affects developer productivity by 4-5x and costs $500K-$2M per 100 engineers annually when poor

### What We Believe But Can't Prove (practitioner consensus, no experiments)

- The specific section ordering recommended above is optimal (no A/B testing exists)
- 500-1,500 words is the ideal length
- The "60-second evaluation window" duration
- Emotional hooks are effective for developer audiences (lazygit is one data point)
- The concrete comparison pattern is the "single most effective" element (strong practitioner signal but no controlled experiment)

### What We Don't Know

- Mobile README consumption patterns (growing but unstudied)
- AI-generated README impact on trust (emerging concern)
- Cultural variation in README preferences (all evidence is English-language)
- Whether the correlation between README quality and popularity is causal or confounded by "teams that write good code also write good READMEs"
- Optimal GIF duration, image size, or badge count (no empirical data)

---

## Sources

This synthesis draws on four detailed reports:

1. **[CLI Tools Ecosystem Analysis](cli-tools-ecosystem-research.md)** — 10 tools, 7 dimensions each
2. **[Build/Runtime Tools Ecosystem Analysis](build-runtime-ecosystem-research.md)** — 10 tools, 7 dimensions each
3. **[Best Practices Literature Review](best-practices-literature-research.md)** — 20+ community guides, 3 academic studies
4. **[Documentation UX & Adoption Psychology](documentation-ux-adoption-research.md)** — 17 sources, academic + industry research

Raw sources (57+ documents) are in the [`docs/`](docs/) directory.
