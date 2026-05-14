# Tasks: Claude Code Tools — Consulting Adoption & CoP Presentation

## Phase 1: Scoping & Initial Research

### Pending

### Active

### Completed
- [x] **Privacy & compliance analysis** — Consulting firms handle client IP. Tools like Rudel upload full transcripts. Map the privacy spectrum from existing comparison against consulting constraints (client NDAs, SOC2, data residency).
  - Priority: high
  - Estimate: medium
  - Started: 2026-03-27
  - Completed: 2026-03-27
  - Outcome: success
  - Output: [privacy-compliance-research.md](privacy-compliance-research.md)
  - Notes: Mapped 7 tools/categories against SOC 2, ISO 27001, GDPR. Classified 5 sensitive data categories in sessions. Recommendation matrix: claude-history/DevTools/ccusage safe; claude-replay safe with config; Mantra requires review; Rudel hosted not recommended. 10-item pre-adoption checklist. Critical finding: /feedback command sends full transcripts with 5-year retention.

- [x] **Gap analysis: missing research spikes** — Identify what research doesn't yet exist that would be needed for a complete CoP talk. Candidates: AI coding evidence longitudinal data, Claude Code hooks/customization, cost optimization practices, team workflow patterns, Claude Code vs. other AI tools comparison.
  - Priority: high
  - Estimate: medium
  - Started: 2026-03-27
  - Completed: 2026-03-27
  - Outcome: success
  - Output: [gap-analysis-research.md](gap-analysis-research.md)
  - Notes: Cross-referenced 11 existing spikes against Nix talk model. 1 blocker spike (consulting-adoption-strategy), 2 high-value items (hooks-in-practice, ROI synthesis), 2 nice-to-haves. Talk positioned Year 2 after QubesOS. Narrative gaps resolved (complementary to Q3 AI Evidence, Q3+Q4 provide conceptual foundation).

- [x] **CoP talk positioning & narrative fit** — Where does this talk fit in the Year 1/Year 2 arc? Could slot into Q3 "AI Evidence" or Year 2. Analyze narrative hooks and dependencies.
  - Priority: medium
  - Estimate: small
  - Started: 2026-03-27
  - Completed: 2026-03-27
  - Outcome: success
  - Output: [gap-analysis-research.md](gap-analysis-research.md) (Dimension 3: Narrative Gaps)
  - Notes: Year 2, after QubesOS. The Q3 AI Evidence talk is "knowing" (what the research says); this talk is "seeing" (what's actually happening in your sessions). Q3+Q4 Year 1 sequence provides the necessary conceptual foundation.

- [x] **Consulting-specific tool selection** — Map the 5 deep-dived tools + ecosystem against consulting-firm needs: multi-client privacy boundaries, team-level visibility, cost attribution per client engagement, onboarding speed. Which tools solve consulting pain vs. general developer pain?
  - Priority: high
  - Estimate: medium
  - Started: 2026-03-27
  - Completed: 2026-03-27
  - Outcome: success
  - Output: [consulting-tool-selection-research.md](consulting-tool-selection-research.md)

- [x] **Adoption strategy & rollout sequencing** — How to introduce these tools at a firm like Highspring. What's the low-commitment on-ramp? What order minimizes friction? Individual tools first vs. team analytics?
  - Priority: high
  - Estimate: medium
  - Started: 2026-03-27
  - Completed: 2026-03-27
  - Outcome: success
  - Output: [adoption-strategy-research.md](adoption-strategy-research.md)
  - Notes: 4-step individual-to-team progression (ccusage → claude-history → Claude DevTools → opt-in Rudel). 5-phase champion activation. "Golden path" voluntary adoption outperforms mandates. Frame cost waste as client delivery problem.

- [x] **Presentation design research** — What's the "one thing" (Carmen Simon) for this talk? What's the demo equivalent of the Nix `cd` switch? Session replay? Live cost dashboard? Team analytics view? Leading candidate from gap analysis: Stop hook quality gate demo.
  - Priority: medium
  - Estimate: small
  - Started: 2026-03-27
  - Completed: 2026-03-27
  - Outcome: success
  - Output: [presentation-design-research.md](presentation-design-research.md)
  - Notes: Evaluated 4 candidates against 6 criteria. ccusage cost reveal wins as "one thing" (lowest friction, strongest pain-first, clearest Carmen Simon compliance). Hooks are the "capability escalation" teaser, not the core. Full minute-by-minute talk structure designed with "Invisible Bill" opener, live demo chain, hooks preview, honest costs, and 3-second on-ramp close. MC bridge from QubesOS designed.
