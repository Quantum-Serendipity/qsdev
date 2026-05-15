# Persuasion, Adoption & Demo Psychology Research

> **Task**: P2-T2 — Meta-analysis of persuasion & adoption literature
> **Status**: Complete
> **Sources**: 14 documents saved to `docs/`

---

## 1. Technology Adoption Models

### 1.1 Diffusion of Innovations (Rogers, 1962)

Rogers identified five adopter categories with distinct motivations that map directly to pitch strategy:

| Category | % | What they need to hear | gdev implication |
|----------|---|------------------------|------------------|
| **Innovators** (2.5%) | Risk-seeking, want novelty | "Here's something new and powerful" | These folks will find gdev on GitHub and try it unprompted. No pitch needed — just make it discoverable. |
| **Early Adopters** (13.5%) | Vision-driven, opinion leaders | "This gives you a strategic advantage" | Target Staff+ engineers and platform leads. Pitch the vision: unified devex + security-by-default. Show the 6-layer defense model. |
| **Early Majority** (34%) | Pragmatic, want proven results | "Others like you use this successfully" | Need references, case studies, and the "whole product." Lead with onboarding speed and posture scoring — measurable outcomes. |
| **Late Majority** (34%) | Skeptical, risk-averse | "This is now the standard" | Won't adopt until gdev is already org-standard. Policy-driven adoption via `gdev check` in CI. |
| **Laggards** (16%) | Traditional, change-resistant | "You have to" | Compliance mandates. `gdev check --fail-on degraded` in CI pipeline. |

Rogers also identified five innovation characteristics that predict adoption variation (explaining 49-87% of adoption variation):

1. **Relative advantage** — gdev's 60-second setup vs. 30-90 minutes manual (strong)
2. **Compatibility** — works with existing devenv.sh, generates standard files (strong)
3. **Complexity** — single binary, one command `gdev init` (low complexity = good)
4. **Trialability** — zero-cost, reversible with `gdev teardown` (strong)
5. **Observability** — posture scores, status dashboard, team reports (strong)

**Key insight for gdev**: gdev scores well on all five Rogers characteristics. The pitch should explicitly highlight each one, especially trialability and reversibility, which reduce adoption risk.

### 1.2 Crossing the Chasm (Moore, 1991)

The "chasm" is the gap between early adopters (visionaries) and the early majority (pragmatists). These groups have fundamentally different buying criteria:

- **Early adopters** buy on vision and potential — tolerate bugs, incomplete products
- **Early majority** buys on proven results and references — expects a "whole product" that works out of the box

**Crossing strategy for gdev:**

1. **Target a beachhead segment** — Small-to-mid consulting firms doing multi-client work (Highspring's profile). The consulting lifecycle features (`gdev init --profile client-healthcare`, `gdev teardown --compliance`) are a "whole product" for this niche.

2. **Deliver the whole product** — Not just the CLI binary but: documentation, migration guides, profile templates, CI integration examples, team dashboards. The early majority won't piece together a solution.

3. **Create pragmatist references** — Pragmatists trust other pragmatists, not visionaries. Internal case studies showing measurable outcomes (onboarding time reduced from X to Y, posture score improved from C to A) matter more than feature lists.

4. **Bowling alley strategy** — Win consulting firms first, then use that reference to expand to mid-size engineering orgs, then larger enterprises.

**Limitation**: Not all innovations face a chasm — only discontinuous ones. gdev straddles this: for teams already using devenv.sh, it's a continuous improvement (incremental, lower risk). For teams not using Nix at all, it's discontinuous (requires paradigm shift). The pitch must account for which audience you're addressing.

### 1.3 Technology Acceptance Model (Davis, 1989)

TAM identifies two primary drivers of technology adoption:

1. **Perceived Usefulness (PU)** — "Does this enhance my job performance?"
2. **Perceived Ease of Use (PEOU)** — "Is this free from effort?"

Critical finding: **usefulness dominates**. Davis found that "users are often willing to cope with some difficulty of use in a system that provides critically needed functionality." For developer tools specifically, a study found: training → ease of use → usefulness → usage. This means that for gdev:

- Lead the pitch with **usefulness** (what it does for you), not ease of use (how simple it is)
- But ease of use is a **gateway** — if the first experience is confusing, they never discover usefulness
- The `gdev init` experience must be frictionless because ease of use is the on-ramp to perceived usefulness

**Practical implication**: The demo should show the *outcome* first (working environment, posture score, compliance evidence), then reveal how easy it was to get there. Usefulness creates desire; ease of use removes objections.

### 1.4 Jobs-to-Be-Done (Christensen)

JTBD reframes adoption: customers don't buy products, they "hire" them to make progress in specific circumstances. The question is not "what features does gdev have?" but "what job is the user hiring gdev to do?"

**gdev's jobs-to-be-done by persona:**

| Persona | Job to be done | Competing "hires" |
|---------|---------------|-------------------|
| Individual dev | "Get me to a working devenv without the yak-shaving" | Manual setup, copying from another project, asking a colleague |
| Platform lead | "Keep all projects at a consistent security baseline without policing developers" | Manual audits, custom scripts, policy documents nobody reads |
| CTO/VP Eng | "Show me compliance evidence without manual effort" | Manual audit prep, spreadsheets, dedicated compliance team |
| Consulting lead | "Clean start/end for client engagements with evidence" | Manual teardown checklists, tribal knowledge |
| New hire | "Be productive on day one without asking for help" | README.md, pair programming with senior, trial and error |

**Key insight**: Each persona is hiring gdev for a different job. The pitch must match the job, not list features. "gdev generates devenv.nix" is a feature. "You're productive in 60 seconds instead of 90 minutes" is a job completed.

### 1.5 Developer-Specific Adoption Patterns

Research from multiple sources reveals how developer adoption differs from general technology adoption:

**Discovery channels** (survey of 202 open-source developers):
- 43.5% discover through tech social platforms (HN, Reddit, dev.to)
- 29% through peer recommendations
- Jump into quickstarts (49.3%) before reading anything else
- Trust good documentation (39.1%) as primary credibility signal

**Evaluation behavior**:
- Developers evaluate by trying, not by reading marketing materials
- Free trial / open-source access is the most common evaluation method
- 68% abandon tools due to lengthy setup times
- Developers making first successful use within 10 minutes are 3-4x more likely to convert
- Average 14 interactions before purchase decision
- 83% of developer ad conversions happen without direct clicks (delayed recall)

**Trust signals developers respect**:
- Working code examples and documentation
- Active GitHub presence
- Honest trade-off discussions
- Transparency about limitations
- Peer recommendations over corporate messaging

**Trust destroyers**:
- Buzzword-heavy language ("revolutionary," "game-changing")
- Gated content behind sign-up forms
- Overselling / broken promises
- 96% of developers assume companies are lying before reading the first sentence

**The 70-20-10 rule for developer content**:
- 70% pure technical value without product mentions
- 20% transparency about challenges and lessons learned
- 10% direct product promotion

---

## 2. Persuasion Frameworks

### 2.1 AIDA (Attention, Interest, Desire, Action)

The classic four-stage persuasion model (Lewis, 1898), still relevant for structuring any pitch:

1. **Attention** — Break through noise with a bold hook
2. **Interest** — Provide context and relevance — why this matters to them
3. **Desire** — Build emotional connection through benefits and transformation
4. **Action** — Clear call to action

**Applied to gdev elevator pitch:**
1. Attention: "Every new project starts with the same 90 minutes of yak-shaving — devenv.nix, .envrc, Claude Code config, pre-commit hooks..."
2. Interest: "gdev detects your project and generates all of it in one command"
3. Desire: "That's 60 seconds to a security-hardened, AI-configured environment. Your new hire is productive before their first coffee gets cold."
4. Action: "Try `gdev init` on any project — it's reversible with `gdev teardown`"

### 2.2 PAS (Problem, Agitate, Solution)

A pain-first framework that works when the audience already feels the problem:

1. **Problem** — Name the specific pain
2. **Agitate** — Make the pain vivid, show consequences
3. **Solution** — Present the resolution

**Applied to gdev (security angle for leadership):**
1. Problem: "Every project has a different security posture. Some have pre-commit hooks, some don't. Some scan dependencies, some don't."
2. Agitate: "When the auditor asks for evidence, you're scrambling across 30 repos manually checking. And 92% of PyPI malware packages are less than 24 hours old — are your projects even checking package age?"
3. Solution: "gdev enforces a security floor across every project. `gdev status` gives you a posture score in 100ms. `gdev evidence` maps every defense layer to SOC2/HIPAA controls with SHA256-hashed artifacts."

**When to use PAS over AIDA**: PAS works best when the audience has experienced the pain and just needs it named and agitated. AIDA works better when the audience doesn't yet know they have the problem.

### 2.3 Before/After/Bridge (BAB)

A transformation narrative framework:

1. **Before** — Paint the current painful state
2. **After** — Show the ideal state
3. **Bridge** — Reveal how to get from Before to After

**Applied to gdev (onboarding angle):**
1. Before: "New developer joins. Day 1: clone the repo. Day 1-3: figure out which Nix packages, formatters, linters, and Claude Code settings they need. Ask three different people. Break something. Fix it. Finally productive on Day 4."
2. After: "New developer joins. `git clone && gdev init --mode join`. Productive in 2 minutes. Security hardened. AI configured. Pre-commit hooks running."
3. Bridge: "gdev reads the project's `.gdev.yaml`, detects the ecosystem, and generates everything. The bridge is one command."

**BAB is particularly effective for developer tool pitches** because developers can immediately visualize both states from their own experience. The transformation is concrete and measurable (days vs. minutes), not abstract.

### 2.4 Elaboration Likelihood Model (Petty & Cacioppo, 1986)

The ELM describes two routes to persuasion:

**Central route** — Deep, thoughtful processing of argument quality. Produces durable, behavior-predictive attitude change. Requires motivation + ability.

**Peripheral route** — Surface-level processing using heuristics (speaker credibility, social proof, emotional cues). Produces temporary, easily-displaced attitudes.

**Critical insight for developer pitches**: Developers are high-need-for-cognition individuals who default to central route processing. This means:

- **Strong arguments with evidence win.** Feature lists with benchmarks, architecture diagrams, and working code create durable adoption decisions.
- **Peripheral cues still matter for attention** — but they open the door to central processing rather than replacing it. A polished README, clean CLI output, professional error messages signal "this is worth evaluating deeply."
- **Weak arguments backfire.** When a motivated, analytical audience encounters weak arguments, they generate counterarguments that produce *more negative* attitudes than no pitch at all. Overclaiming ("10x productivity!") actively hurts.
- **Cognitive load degrades central processing.** If a presentation is confusing, overloaded, or poorly structured, even motivated developers will fall back to peripheral processing — judging by surface impressions rather than substance. Keep presentations clean and focused.
- **Durability difference matters.** Central-route attitude change predicts behavior (actually adopting the tool). Peripheral-route change doesn't. A flashy demo that doesn't survive technical scrutiny creates temporary enthusiasm that evaporates.

**Practical implications:**
- Lead with strong, verifiable claims (60-second setup, 6 defense layers, 92% malware catch rate)
- Show real terminal output, real generated files, real posture scores
- Never make claims you can't demonstrate live
- Use peripheral cues (professional design, clean output) to earn the right to central-route evaluation
- Respect cognitive load — don't overwhelm in a 15-minute format

### 2.5 Loss Aversion in Framing

Loss aversion (Kahneman & Tversky): the pain of losing is psychologically about twice as powerful as the pleasure of gaining. Framing the same information as avoiding a loss vs. achieving a gain significantly changes behavior.

**Gain frame**: "gdev gives you consistent security across projects"
**Loss frame**: "Without gdev, you're one dependency away from a supply chain attack that your current setup won't catch"

**Developer-specific application**:
- Developers are somewhat resistant to emotional manipulation, so loss framing must be grounded in technical reality
- Most effective when tied to concrete, credible risks: "92% of PyPI malware packages are <24h old — age-gating catches them. Without it, `pip install` is a gamble."
- Works especially well for security audiences and leadership (who think in risk terms)
- Less effective for individual devs (who think in productivity terms — use gain framing)

**Persona-appropriate framing:**

| Persona | Better frame | Example |
|---------|-------------|---------|
| Individual dev | Gain | "Get to productive in 60 seconds" |
| Platform lead | Loss | "Without consistent baselines, every project is a drift risk" |
| CTO/VP Eng | Loss | "The next audit will ask for evidence you don't have" |
| Security lead | Loss | "Your supply chain defenses have gaps you can't see" |

### 2.6 Social Proof Mechanics

Social proof works differently for developers vs. management:

**What works for developers:**
- GitHub stars, contributor activity, issue resolution times (open-source health metrics)
- Internal champions: peer-led demos and coding sessions shift sentiment from skepticism to enthusiasm
- "Impact stories" with structure: problem → what was done → measurable outcome
- Community size and engagement (Stack Overflow activity, Discord activity)
- Companies using the tool (but only if technically respected companies)

**What works for management:**
- Fortune 500 / industry adoption numbers
- ROI metrics and case studies with hard numbers
- Analyst reports and compliance certifications
- Risk reduction evidence (incidents prevented, audit time saved)

**What doesn't work for anyone:**
- Vague testimonials ("Great tool! — Happy Customer")
- Logos without context
- Self-reported satisfaction scores

**Key finding**: Different adoption stages need different proof. Early adopters and innovators don't need social proof — they want technical novelty. The early majority needs reference customers in their segment. Late majority needs it to be "standard."

---

## 3. Demo & Presentation Psychology

### 3.1 Peak-End Rule (Kahneman & Fredrickson, 1993)

People judge experiences primarily by two moments: the **most intense moment** (peak) and the **final moment** (end). The duration and average quality of the experience barely matter.

**Implications for gdev demos:**

- **Design a deliberate peak moment.** The peak should be the "aha" — the moment where value clicks. For gdev, this is likely: running `gdev init` and watching it detect the ecosystem, generate files, and produce a working environment in real-time. Or: running `gdev status` after init and seeing a posture score jump from "no data" to "A (92/100)."

- **Design a strong ending.** Don't let the demo trail off into Q&A. End with the most impressive capability: `gdev evidence --framework soc2` producing a compliance report, or `gdev teardown --compliance` creating an audit trail. The ending should leave them thinking "I want that."

- **Manage negative peaks.** If something breaks during a live demo, that becomes the peak memory. Have fallback plans (pre-recorded segments, screenshots). If a failure happens, handle it gracefully — acknowledge it, explain why, move on. An unflappable presenter turns a negative peak into a positive one (trust signal: "they're honest about failures").

- **Structure by intensity, not chronology.** Don't present features in phase order (1→2→3). Present in intensity order: boring setup first, then building intensity, peaking at the "wow" moment, then ending strong.

### 3.2 Cognitive Load Theory (Sweller, 1988)

Working memory has strict limits. Three types of load compete for capacity:

1. **Intrinsic load** — inherent complexity of the content (can't be reduced, only managed)
2. **Extraneous load** — presentation method overhead (should be minimized)
3. **Germane load** — effort spent building mental models (should be maximized)

**Practical limits:**
- Working memory holds ~4 chunks of new information
- Visual information increases retention by 42% over text-only
- Simultaneous text + speech from different sources overloads the visual channel

**Implications for gdev presentations:**

| Format | Time | Max new concepts | Strategy |
|--------|------|-----------------|----------|
| Elevator pitch | 60 sec | 1-2 | One job, one proof point |
| 15-min demo | 15 min | 4-5 | One narrative arc, 3-4 demo beats |
| Deep-dive | 45-60 min | 8-12 | Progressive complexity, breaks between sections |

**Reducing extraneous load:**
- Clean terminal with large font, minimal prompt
- One concept per slide/demo beat
- Narrate what you're typing and why (dual coding: visual + auditory)
- Don't show config files line-by-line; show the before/after diff
- Use progressive disclosure: show the simple case first, then reveal depth

**Increasing germane load (good):**
- Connect each feature to a problem the audience recognizes
- Use analogies: "gdev doctor is like `brew doctor` but for your whole devenv"
- Build a mental model: the 6-layer security model as a visual, then demo each layer

### 3.3 The "Aha Moment"

The aha moment is the user's first emotional realization that the product will be valuable — distinct from activation (first hands-on experience of value). It's the spark that turns curiosity into commitment.

**Research findings:**
- The aha moment and activation are different events — aha is cognitive/emotional, activation is behavioral
- Knowing your aha moment is the key to successful onboarding and retention
- Products that engineer a fast path to the aha moment see dramatically higher conversion

**gdev's aha moment candidates (by persona):**

| Persona | Likely aha moment |
|---------|------------------|
| Individual dev | Running `gdev init` and seeing a complete, working environment generated in seconds |
| Platform lead | Running `gdev status` across multiple projects and seeing unified posture scores |
| Security lead | Running the test fixtures and seeing each defense layer provably working |
| CTO/VP Eng | Seeing `gdev evidence --framework soc2` produce audit-ready compliance reports |
| New hire | `git clone && gdev init --mode join` → productive in 2 minutes |

**Demo implication**: The demo must reach the aha moment as fast as possible. Everything before the aha is cost; everything after is reinforcement. For a 15-minute demo, the aha should hit by minute 3-4, not minute 12.

### 3.4 Primacy and Recency Effects

The serial position effect: people remember the **first** items (primacy) and **last** items (recency) in a sequence better than middle items. Primacy items enter long-term memory through focused attention; recency items are still in working memory.

**Presentation structure implications:**

| Position | Effect | What to put here |
|----------|--------|-----------------|
| Opening | Primacy (long-term memory) | The single most important value proposition — the "why should I care" |
| Middle | Weak recall | Supporting details, configuration options, edge cases |
| Closing | Recency (working memory) | The most impressive capability + clear call to action |

**For gdev specifically:**
- **Open** with the transformation: "90 minutes of manual setup becomes 60 seconds"
- **Middle**: walk through the features that support this claim
- **Close** with the most impressive feature the audience hasn't seen yet (compliance evidence, team dashboards, or the security test fixtures)

**Combining with peak-end rule**: The peak can be anywhere but should be designed. The recency-weighted ending must be strong. This means: don't end with Q&A as the last thing — end with a strong closing demo beat, *then* take questions.

### 3.5 Narrative Transportation

Narrative transportation is the experience of being immersed in a story, reducing counterarguing and lowering resistance to persuasion (Green & Brock, 2000). When people are "transported" into a narrative, they:

- Generate fewer counterarguments
- Experience stronger emotional engagement
- Are more persuaded by story-embedded claims
- Remember story-central elements better over time

**Application to technical presentations:**

Stories bypass the evaluative regions of the brain and activate experiential/emotional regions. This is critical because developers' default mode is evaluative (central route processing). A story creates a window where even analytical audiences process with less resistance.

**Effective story structure for technical talks:**
1. Introduce a relatable problem you encountered
2. Highlight consequences and impact
3. Present your solution with clear reasoning
4. Show the improvement
5. Acknowledge imperfections (builds trust)
6. Provide actionable advice

**For gdev**: The most effective narrative is a real scenario — "Last month, a new developer joined our team. Here's what happened..." Walk through the actual experience of onboarding with and without gdev. The story makes abstract benefits concrete and emotionally resonant.

**Caution**: Narrative transportation works but must be authentic. Developers detect manufactured stories quickly. Use real experiences, real terminal output, real problems. Acknowledge what gdev doesn't do well alongside what it does.

---

## 4. Developer-Specific Considerations

### 4.1 Developer Skepticism and Trust

The core challenge: **96% of developers assume a company is lying before reading the first sentence** of marketing content. Developers have developed "collective immunity to fluff."

**Trust is built on a specific ladder:**
1. Working code (evaluate the actual tool)
2. Documentation quality (can I figure this out myself?)
3. Community presence (do real people use this?)
4. Brand messaging (only matters after 1-3 are established)

Skipping any rung destroys credibility. You cannot market a tool that doesn't have excellent documentation. You cannot documentation-sell a tool that doesn't work when tried.

**Trust-building timeline**: Research indicates developers need approximately **14 interactions** before a purchase/adoption decision. This means a single demo rarely converts — it starts a relationship. The demo must be good enough that they try the tool themselves (interaction 2), read the docs (interaction 3), tell a colleague (interaction 4), etc.

### 4.2 The "10x Claim" Problem

Hyperbolic claims trigger active resistance in developer audiences. There's a documented tipping point where additional hype hurts rather than helps — it reduces credibility and perceived value.

**What goes wrong:**
- "10x productivity" → developer mentally argues: "In what conditions? For whom? Measured how?"
- "Revolutionary" → "Every tool says this. None of them are."
- "Game-changing" → "I've heard this 50 times this year"

**What works instead:**
- Specific, verifiable claims: "60 seconds from clone to working devenv shell" (can be timed)
- Bounded claims: "catches 92% of PyPI malware published <24h old" (specific, sourced, bounded)
- Honest comparisons: "gdev adds X over plain devenv.sh, but doesn't replace Y"
- Acknowledge limitations: "gdev's detection covers 27 ecosystems; if yours isn't one of them, you'll need manual config"

**The specificity principle**: "Reduces API response time from 200ms to 50ms" beats "lightning-fast performance." "50ms p99 latency" beats "fast API." Precision signals engineering credibility. Vagueness signals marketing.

### 4.3 Open Source Trust Dynamics

For developer tools, open-source availability fundamentally changes the trust equation:

- **Try-before-buy is table stakes.** 68% of developers abandon tools with lengthy setup. gdev's zero-cost, zero-prerequisite static binary is a strong trust signal.
- **Evaluation happens without permission.** Developers will `gdev init` on a side project before bringing it to their team. The first-run experience is the pitch.
- **Community metrics are trust proxies.** GitHub stars, fork count, contributor activity, issue response times signal project health.
- **Open source → enterprise pipeline works.** HashiCorp grew to 100M+ Terraform downloads and $14B IPO valuation through this exact model: free core that solves real problems → internal champions → enterprise features for governance/compliance.

**gdev's position**: As MIT-licensed open source, gdev can leverage the full PLG funnel: discover → try → adopt → expand → procure. The pitch for individual developers is "try it" (zero risk). The pitch for organizations is "standardize on it" (governance features).

### 4.4 The Role of Try-Before-Buy

The "10-minute rule" from PLG research: developers who achieve first value within 10 minutes are 3-4x more likely to convert. Best-in-class examples:

- **Stripe**: Working API call with 3 lines of code using test keys
- **Vercel**: Deploy project with `npx vercel` in under 60 seconds
- **Supabase**: Launch fully configured backend in under 1 minute

**gdev's 10-minute positioning**: `gdev init` targets <60 seconds to working environment. This is exceptionally strong — faster than Stripe's benchmark. The pitch should emphasize this speed as a differentiator and prove it live.

### 4.5 Developer Advocacy vs. Traditional Marketing

Developer relations succeeds by being the opposite of marketing:

| Traditional Marketing | Developer Advocacy |
|----------------------|-------------------|
| Persuade | Educate |
| Generate leads | Build trust |
| Gated content | Open documentation |
| Sales-driven messaging | Technical, educational |
| Feature-focused | Problem-focused |
| Short-term metrics (MQLs) | Long-term metrics (MAD, activation) |

**The moment developers sense they're being marketed to, trust evaporates.** Conference talks that are thinly-veiled product pitches get negative reception. The effective approach: teach something genuinely useful (developer environment best practices, supply chain security patterns), demonstrate it with gdev as the concrete example, and let the audience draw their own conclusions.

**Real-world examples of what works:**
- **Stripe**: Documentation so good it's used as a teaching resource. Features aren't shipped until docs are written, reviewed, and published.
- **Vercel/Next.js**: Open-sourced Next.js so developers could inspect technical expertise. Community trust preceded commercial trust.
- **HashiCorp**: Free hands-on tutorials via HashiCorp Learn. Education → adoption → advocacy → more adoption (virtuous cycle).
- **Cloudflare**: Technical blog posts tackling complex infrastructure challenges with deep architectural insights, not feature promotion.

### 4.6 Internal Champion Cultivation

Developer tools typically follow a bottom-up adoption path:

```
Individual discovery → Personal use → Team champion → Leadership pitch → Org adoption
```

Case study evidence shows that **peer-led tool rollouts** (where experienced team members act as advocates, hosting live demos and coding sessions) shift developer sentiment from skepticism to enthusiasm more effectively than top-down mandates.

**Internal champion characteristics:**
- Well-respected team member (credibility from technical competence, not title)
- Has used the tool on real projects (authentic experience)
- Can speak to problems the team actually has (relevance)
- Willing to help others get started (reduces adoption friction)

**Implication for gdev**: The pitch materials need to serve champions, not just decision-makers. Champions need:
- Quick-start guide they can share
- Talking points for their leadership pitch
- Before/after metrics from their own usage
- A demo they can reproduce (not just watch)

---

## 5. Synthesis: Framework Selection by Context

### Which framework to use when

| Context | Best framework | Why |
|---------|---------------|-----|
| Hallway conversation | BAB | Quick transformation story in 30 seconds |
| Elevator pitch to dev | AIDA + gain frame | Attention → interest through productivity benefits |
| Elevator pitch to leader | PAS + loss frame | Name the pain (inconsistency, compliance gaps), agitate (audit risk, supply chain attacks), solve |
| 15-min demo | Peak-end rule + AIDA | Structure around peak moment (aha), end strong, AIDA for narrative arc |
| 45-min deep-dive | Narrative transportation + progressive disclosure | Story-driven opening, then systematic depth with cognitive load management |
| Written README | BAB + social proof | Before/after transformation + metrics + who uses it |
| Internal champion pitch | JTBD + social proof | "Here's the job I was trying to do, here's how gdev did it, here are my numbers" |

### Universal principles across all formats

1. **Show, don't tell.** Live demos beat slides. Real output beats mockups. Specific numbers beat adjectives.
2. **Specificity signals credibility.** "60 seconds" beats "fast." "6 defense layers" beats "secure." "27 ecosystems" beats "broad support."
3. **Acknowledge limitations honestly.** Developers trust you more after hearing what gdev *doesn't* do. It lowers defenses for what it does do.
4. **Match the frame to the persona.** Gain for devs (productivity), loss for leaders (risk), both for security (defense + consequences of gaps).
5. **Reach the aha moment fast.** Everything before the aha is cost. In any format, minimize the time to "oh, I want that."
6. **End strong.** Peak-end rule + recency effect both say: the last thing they see shapes the entire memory of your presentation.
7. **Respect cognitive load.** One concept at a time. Progressive disclosure. Clean visuals. Narrate what you're doing.
8. **Build for the champion, not just the buyer.** The person who tries gdev first is rarely the person who approves org-wide adoption. Give champions the materials to pitch internally.

---

## Sources

All source documents saved to `docs/`:

1. `evil-martians-six-things-devtools-trust-adoption.md` — Evil Martians on 6 trust/adoption principles for devtools (2026)
2. `daily-dev-technical-marketing-to-developers.md` — daily.dev guide on technical marketing to developers
3. `daily-dev-earn-trust-not-impressions.md` — daily.dev on trust-based developer engagement
4. `crossing-the-chasm-summary.md` — High Tech Strategies summary of Crossing the Chasm
5. `cognitive-load-theory-presentations.md` — Ethos3 on cognitive load theory for presentations
6. `nngroup-peak-end-rule.md` — Nielsen Norman Group on the peak-end rule
7. `daily-dev-plg-developer-tools.md` — daily.dev on product-led growth for developer tools
8. `hashicorp-open-source-strategy.md` — HashiCorp's open source to enterprise growth strategy
9. `simply-psychology-elaboration-likelihood-model.md` — Simply Psychology on the Elaboration Likelihood Model
10. `developer-focused-sales-funnels.md` — Developer-focused sales funnel hybrid approach
11. `stripe-developer-platform-insights.md` — Stripe's developer platform experience insights
12. `luke-lowrey-product-demo-tips.md` — Product demo presentation tips
13. `daily-dev-five-case-studies-tool-adoption.md` — 5 case studies on developer tool adoption
14. `freecodecamp-tech-conference-talks.md` — freeCodeCamp guide to tech conference talks
