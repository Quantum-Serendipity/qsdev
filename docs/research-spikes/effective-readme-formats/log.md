# Research Log: Effective README Formats for Developer Tools

## 2026-05-15 — Spike Created
- **Type**: decision
- **Status**: success
- **Depth**: surface
- **Summary**: Spike initialized. Research question: What makes a README effective at communicating what a developer tool is, why it's useful, how to install it, and how to get started — with a focus on strategies that capture attention and drive adoption? Scope includes ecosystem analysis of popular devex tool READMEs and a meta-analysis of best practices from literature and community wisdom.
- **Next**: Define research question and create Phase 1 tasks.

## 2026-05-15 — CLI Tools Ecosystem Analysis Complete
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**: 10 raw READMEs fetched from GitHub (ripgrep, fzf, bat, eza, fd, zoxide, starship, lazygit, jq, delta) → `docs/*-readme.md`
- **Summary**: Analyzed 10 popular CLI developer tool READMEs across 7 dimensions (structure, value proposition, visual strategy, installation, quick-start, attention capture, tone). Identified cross-cutting structural patterns, winning strategies (concrete comparison, feature gallery, numbered setup flow, emotional hook, configuration-as-quickstart, benefit bullets, try-before-install), anti-patterns (sponsor banners above fold, deferred installation, missing quick-start, wall-of-text features, no comparison to alternatives), and synthesized an optimal structural template. bat ranked highest overall; fd best at persuasion-through-comparison; starship most polished product identity; lazygit best copywriting.
- **Next**: Integrate findings into Phase 3 synthesis with other ecosystem analyses and literature review.

## 2026-05-15 14:00 — Literature Review: Documentation UX & Adoption Psychology
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**: 17 sources fetched and saved to `docs/` (see `documentation-ux-adoption-research.md` Sources section for full list)
- **Summary**: Completed deep literature review across four domains: developer documentation UX (cognitive load, progressive disclosure, learning styles), first-impression psychology (F-pattern scanning, 50ms judgments, above-the-fold attention), open-source marketing (education-first model, discovery funnel, documentation as product), and anti-patterns (wall of text, feature-first framing, assumed context, stale content). Synthesized into README Conversion Framework mapping landing page principles to README structure. Key quantitative findings: 15-minute TTV window, 71% peer recommendation rate, 50ms aesthetic judgment, 4-5x productivity difference from documentation quality. Academic sources include Head et al. 2018, Meng et al. 2019, Shen & Sood 2025 on social proof. 17 sources saved to docs/.
- **Next**: Phase 2 task complete. Remaining Phase 2 tasks (build/runtime ecosystem analysis, literature review of best practices guides) may proceed independently. Phase 3 synthesis can begin once all Phase 2 tasks are done.

## 2026-05-15 15:30 — Literature Review: README Best Practices & Community Guides
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**: 20 sources fetched and saved to `docs/` — community guides (Make a README, Awesome README, Standard Readme, Art of README), platform docs (GitHub, npm, PyPI, crates.io), practitioner essays (Preston-Werner, Hearth/thoughtbot, Bugayenko, Burazin/Daytona), blog posts (dev.to x2, freeCodeCamp, Changelog), academic papers (Venigalla & Chimalakonda 2022, Wang et al. 2023, Prana et al. 2019), and GFM features guide
- **Summary**: Completed comprehensive literature review synthesized into `best-practices-literature-research.md`. Key findings: (1) Universal consensus on 6 core sections (name, description, visual demo, installation, usage examples, license). (2) Strong consensus on cognitive funneling — broadest info first, specifics later. (3) Empirical research confirms README quality (structure, images, lists, freshness) correlates with repository popularity. (4) Prana et al. found systematic gap — "What" and "Why" sections are frequently missing despite being critical for evaluation. (5) Cross-platform rendering differences matter — GitHub-specific features (Mermaid, alerts) don't render on PyPI or crates.io. (6) No A/B testing data exists; all evidence is correlational or practitioner-reported. (7) Identified 8 specific actionable patterns and 11 documented anti-patterns.
- **Next**: Phase 2 literature review tasks now complete. Ready for Phase 3 synthesis across all four Phase 2 deliverables.

## 2026-05-15 16:30 — Build/Runtime/Framework Ecosystem Analysis Complete
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**: 10 raw READMEs fetched from GitHub (bun, deno, uv, ruff, biome, turborepo, mise, nushell, pnpm, esbuild) -> `docs/*-readme.md`
- **Summary**: Analyzed 10 popular build tools, runtimes, and package manager READMEs across 7 dimensions (structure, value proposition, visual strategy, installation, quick-start, attention capture, tone). uv ranked highest overall for its one-liner + benchmark chart + "replaces 7 tools" + console timing pattern. Ruff best at social proof (testimonials from FastAPI/isort/GraphQL creators). esbuild best at problem-framing ("Why?" before "What?"). mise best quickstart (progressive complexity with realistic terraform example). Identified 3 information hierarchy strategies (Full Pitch, Teaser, Technical Showcase), 7 anti-patterns (README as sitemap, README as redirect, sponsors dominating content, no install commands, burying the differentiator, promoting other projects at top, unexplained flags), and 8 specific techniques worth adopting. Compared build/runtime patterns to CLI tool patterns from companion analysis.
- **Next**: All Phase 2 tasks now complete. Ready for Phase 3 pattern extraction and synthesis.

## 2026-05-15 17:00 — Phase 3: Pattern Synthesis Complete
- **Type**: analysis
- **Status**: success
- **Depth**: deep
- **Summary**: Wrote cross-cutting pattern synthesis report (pattern-synthesis-research.md, 612 lines) unifying all Phase 2 findings. Covers: README as conversion funnel (with cognitive science backing), optimal section ordering with evidence rationale, 12 winning patterns (one-liner value prop, visual proof before text, concrete comparison, 30-second quick-start, platform-aware installation, benefit-framed bullets, calibrated social proof, emotional hook, ecosystem positioning, configuration-as-quickstart, try-before-install, honest positioning), 11 anti-patterns across 3 severity tiers, effectiveness spectrum ranking all 20 tools into 4 tiers, actionable copy-pasteable README template with usage notes, platform rendering comparison table, and evidence quality assessment (empirical vs practitioner vs opinion). Updated research.md with full conclusions.
- **Next**: Depth checklist review.

## 2026-05-15 17:30 — Phase 3: Depth Checklist Review Complete
- **Type**: analysis
- **Status**: success
- **Depth**: deep
- **Summary**: Ran depth checklist (6 items) against all 5 reports. Results: pattern-synthesis-research.md passes 6/6. Two ecosystem reports pass 5/6 each (partial on anti-pattern behavioral consequences — structural failures identified but not always connected to user behavior outcomes). Literature review passes 4/6 (partial on mechanisms behind consensus points, fewer concrete named examples). UX report passes 4/6 (partial on alternative strategy comparison, fewer concrete examples). All gaps are compensated by the synthesis report which integrates mechanisms, examples, and alternatives across all sources. A reader using pattern-synthesis-research.md alone gets the complete, actionable picture.
- **Next**: Spike ready for completion. Optional gap-filling available for individual sub-reports but non-blocking.

## 2026-05-15 18:00 — Spike Completed
- **Type**: analysis
- **Status**: success
- **Depth**: deep
- **Summary**: Spike finalized with 5 research reports (2,195 lines total), 57+ source documents in docs/, and comprehensive conclusions in research.md. Core finding: a README is a conversion funnel governed by the same mechanisms as landing page optimization — 50ms visual judgment, F-pattern scanning, 15-minute TTV abandonment. 12 winning patterns identified and ranked by impact (one-liner value prop, visual proof, concrete comparison, 30-second quick-start, progressive disclosure are the top 5). 11 anti-patterns catalogued across 3 severity tiers. Actionable copy-pasteable README template provided in pattern-synthesis-research.md. 20 developer tools analyzed across 7 dimensions each with a 4-tier effectiveness ranking. Evidence base includes 3 academic studies, 11 community guides, and industry research from Stack Overflow, NNGroup, Google UX, and DX. Open questions are all academic research gaps (no A/B testing, mobile patterns, AI trust, cultural variation, causality) — none warrant follow-on spikes. No follow-on candidates flushed to proposed-spikes.md.
