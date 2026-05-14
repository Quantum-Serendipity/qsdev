# Presentation Design: Claude Code Tools CoP Talk

## The Core Question

What is the "one thing" (Carmen Simon) for this talk, and how should the 15-minute talk be structured to maximize actionable impact for Highspring Digital engineers?

---

## 1. Evaluating the Four "One Thing" Candidates

### Evaluation Framework

Each candidate is scored against the six design principles from the Nix CoP talk (`implementation-plans/nix-consulting-cop-talk/plan.md`), plus two criteria from the presentation skills research (visceral audience reaction, Carmen Simon memorability).

### Candidate 1: Live Session Replay of a Real (Redacted) Consulting Session

**What it looks like**: Run `claude-replay` on a real session from a Highspring engagement. The audience watches a compressed playback of an actual AI-assisted task — the prompts, tool calls, thinking, and output scrolling by. Token counters tick upward. Redacted client details appear as `[REDACTED]`.

| Criterion | Score | Rationale |
|-----------|-------|-----------|
| Pain-first setup | Medium | Pain is "you have no idea what happened in your AI sessions." Works but isn't visceral — it's curiosity, not dread. |
| 40% terminal time | High | The replay IS a terminal demo. Natural screen time. |
| Low-commitment on-ramp | Low | claude-replay requires installation and session selection. Not a 5-minute post-talk action. |
| Capability framing | Medium | "We review our AI-assisted development sessions for quality" — decent but abstract. |
| Visceral reaction | Medium | Interesting to watch but voyeuristic rather than actionable. The audience thinks "cool" not "I need this." |
| Carmen Simon memorability | Medium | Visual and novel, but what do they remember 48 hours later? "I watched someone's Claude session." Not a clear takeaway. |

**Verdict**: Entertaining but passive. The audience watches something happen to someone else. The Nix talk's `cd` demo worked because the audience could imagine doing it themselves immediately. A replay doesn't have that self-projection quality.

### Candidate 2: Cost Dashboard Reveal (ccusage Output Across a Week)

**What it looks like**: Run `npx ccusage` and show a week of real usage data. A table appears with daily costs, token counts, and a cost spike on one particular day. "See that spike on Wednesday? That was a session that went sideways for 45 minutes. Cost me $47 in tokens. I didn't notice until I ran this command."

| Criterion | Score | Rationale |
|-----------|-------|-----------|
| Pain-first setup | **High** | "You burned $400 on Claude this month and you have no idea where it went." Universally relatable. Money is concrete. Consultants think in cost. |
| 40% terminal time | High | `npx ccusage` is a natural terminal demo. The output IS the revelation. |
| Low-commitment on-ramp | **Highest** | "Run `npx ccusage` right now. Takes 3 seconds. No install." This is the most frictionless possible on-ramp. |
| Capability framing | High | "We track and optimize our AI-assisted development costs per engagement." CFO-friendly. |
| Visceral reaction | **High** | Seeing your own money disappear into a cost table produces an immediate gut response. The spike day makes people think of their own worst sessions. |
| Carmen Simon memorability | High | "Run `npx ccusage` and see your spend." This is a single sentence, a single action, a single tool. Clear, concrete, repeatable. |

**Verdict**: Strong. The on-ramp is the strongest of any candidate — literally zero friction. The pain point (wasted money) is universal and consulting-specific (it maps to client delivery costs). But the "one thing" is just a cost table. It's informative, not transformative. The audience learns something about their past; they don't gain a new capability for their future.

### Candidate 3: Hook-Enforced Quality Gate (Stop Hook Catches Issue Live)

**What it looks like**: Configure a Claude Code Stop hook that enforces a quality gate — for example, a hook that runs after every `Bash` command and blocks the session if tests fail, or a hook that prevents commits without test coverage. During the talk, trigger the hook by having Claude Code attempt something that violates the gate. The hook fires, Claude is blocked, the audience sees enforcement happen in real-time.

| Criterion | Score | Rationale |
|-----------|-------|-----------|
| Pain-first setup | High | "Claude Code committed untested code to your client's repo at 2 AM while you were asleep." Consulting-specific nightmare. |
| 40% terminal time | High | Live demo of configuring and triggering a hook. Natural terminal flow. |
| Low-commitment on-ramp | **Medium** | Adding a hook requires editing `.claude/hooks/` or `CLAUDE.md`. It's a 5-minute task, not a 3-second one. Slightly higher barrier. |
| Capability framing | **Highest** | "We enforce automated quality gates on all AI-generated code — every commit passes the same bar as human-written code." This is the strongest capability statement for client-facing contexts. |
| Visceral reaction | **High** | Watching enforcement happen live — Claude tries something, gets blocked, adjusts — is dramatic. The audience sees the safety net catch something. |
| Carmen Simon memorability | High | "Claude Code can't ship code that doesn't pass your tests" is a clear, memorable statement. But the mechanism (hooks) is more complex than `npx ccusage`. |

**Verdict**: The strongest capability story and the most dramatic demo. But the on-ramp friction is higher than ccusage, and hooks are still relatively new — the hooks-in-practice research gap (GAP 3 in gap-analysis-research.md) means we lack empirical evidence that this works reliably in consulting contexts. The demo risks being "cool but will I actually set this up?" unless the on-ramp is specifically designed.

### Candidate 4: Before/After Session Comparison (Unstructured vs. RPI with Observability)

**What it looks like**: Two session analyses side by side — one from an unstructured "just ask Claude to do it" session, one from an RPI-structured session with observability. Show token usage, turn count, tool calls, and outcome for both. "Same task, same developer. One cost $12 and shipped clean code. The other cost $47 and the code had to be rewritten."

| Criterion | Score | Rationale |
|-----------|-------|-----------|
| Pain-first setup | Medium | Compelling as data but requires setup time to explain both approaches. The pain is diluted by the comparison structure. |
| 40% terminal time | Medium | Some terminal time showing the analysis tools, but the comparison requires slides or split-screen. Less naturally terminal-native. |
| Low-commitment on-ramp | Low | The takeaway is "use RPI and observability tools" — two things, not one. Violates the "one thing" principle. |
| Capability framing | Medium | "We use structured AI workflows with outcome measurement." Good but wordy. |
| Visceral reaction | Medium | Interesting comparison but academic. The audience is evaluating evidence, not having an emotional response. |
| Carmen Simon memorability | Low | What do they remember? "One session was better than the other." The mechanism and the takeaway are both complex. |

**Verdict**: This is a 30-minute talk crammed into 15 minutes. The comparison structure requires explaining two approaches, showing two analyses, and drawing conclusions — that's at least three major content blocks plus setup. It violates the 15-minute format's most important constraint: one idea, not two.

### Consolidated Scoring

| Criterion | Replay | Cost Reveal | Hook Gate | Before/After |
|-----------|--------|-------------|-----------|--------------|
| Pain-first setup | 2 | **4** | **4** | 2 |
| 40% terminal time | 4 | **4** | **4** | 2 |
| Low-commitment on-ramp | 1 | **5** | 3 | 1 |
| Capability framing | 2 | 3 | **5** | 2 |
| Visceral reaction | 2 | **4** | **4** | 2 |
| Carmen Simon memorability | 2 | **4** | 3 | 1 |
| **Total** | **13** | **24** | **23** | **10** |

### The Recommendation: Cost Reveal as the "One Thing," Hooks as the "One More Thing"

The evaluation reveals two strong candidates that serve different purposes:

**ccusage cost reveal** wins as the "one thing" because:
- It has the lowest on-ramp friction of any candidate (literally `npx ccusage`)
- The pain point (wasted money) is the most universally relatable for consulting
- It's a single tool, a single command, a single action — clean Carmen Simon compliance
- It mirrors the Nix talk's `cd` demo: one command, immediate visible result, "you can do this right now"

**Hook-enforced quality gate** is the second-best candidate and serves as the "capability escalation" — the reason to keep going after ccusage. "You saw your cost. Now here's how you guarantee quality." Hooks answer the question ccusage raises: "I'm spending a lot on Claude. Is the code any good?"

The talk structure should use ccusage as the core demo and hooks as the "where this goes next" teaser — mirroring how the Nix talk used `cd` switching as the demo and devenv.sh as the "where to go from here."

**Why not hooks as the one thing?** The gap analysis (GAP 3) identifies that hooks-in-practice research doesn't yet exist. The hook demo is theoretically strong but empirically unproven — we don't have case studies of teams using hooks reliably for quality enforcement. More importantly, the on-ramp for hooks (edit a config file, write a shell script, test it) is meaningfully more complex than `npx ccusage`. The Nix talk's power came from "you can do this in 5 minutes"; hooks require more like 30 minutes of setup. For a CoP audience of busy consultants, the 3-second on-ramp wins.

---

## 2. Talk Structure: 15-Minute Format

### Structure: Problem/Demo/Takeaway (Structure A from 15-minute format research)

This is the default structure for practitioner talks. Time allocation: Problem (2-3 min) -> Approach (2-3 min) -> Demo (5-6 min) -> Capability Extension (2 min) -> On-Ramp Close (1-2 min).

Total planned content: ~13 minutes (leaving 2 minutes for buffer/Q&A per the 15-minute format research recommendation of planning 12-13 minutes).

---

### Minute-by-Minute Structure

#### Opening Hook: "The Invisible Bill" (0:00-2:30)

**0:00-0:30 — The cold open.**
"Last month, one of our teams burned through $1,200 in Claude Code tokens in a single week. They didn't know until the invoice arrived. The sessions that cost the most? Not the productive ones — the ones where Claude went in circles for 40 minutes trying an approach that was never going to work."

**0:30-1:30 — Make it personal.**
"Here's the thing about Claude Code — and this applies to Cursor, Copilot, all of them. You have zero visibility into what's happening. You type a prompt, Claude does... something... for 3 minutes, and at the end you either have working code or you don't. But you have no idea how many tokens it burned, how many files it read, whether it tried 4 approaches and threw away 3, or whether half your context window is being consumed by a CLAUDE.md you wrote six months ago and forgot about."

"Who here has had a Claude Code session that felt like it was just... spinning? Going in circles, trying the same thing over and over?" [Show of hands — expect most hands up.]

**1:30-2:30 — The consulting-specific twist.**
"Now here's why this matters specifically for us. We're consultants. That token spend is part of our delivery cost. When a session burns $50 in tokens on a wrong approach, that's not coming out of a product budget that amortizes over millions of users — that's delivery margin on a specific engagement. And right now, we can't even tell a client which project the spend was for, let alone whether the money produced working code."

"Last quarter we talked about the AI coding evidence — the research showing a perception gap between how productive developers *feel* when using AI tools versus what the data shows. Today I want to show you something different. Not what the *research* says — what your *own sessions* say."

#### The Approach: "Your Sessions Are Already Being Recorded" (2:30-4:30)

**2:30-3:30 — The reveal.**
"Here's something most people don't know: Claude Code already records everything. Every session, every prompt, every tool call, every token — it's all sitting in JSONL files on your machine right now. `~/.claude/projects/` — go look after this talk. It's all there."

"The problem isn't data. The problem is that nobody reads those files. They're raw JSON, hundreds of megabytes, organized by encoded paths that are barely human-readable. So the data exists, but it's invisible."

**3:30-4:30 — The ecosystem in 60 seconds.**
"A small ecosystem of open-source tools has emerged to make this data visible. I've evaluated about 50 of them. Most are toys. A handful are genuinely useful. Today I'm going to show you one that you can run in the next 3 seconds — no install, no config, nothing leaves your machine — and see exactly where your tokens are going."

[Transition: switch to terminal.]

#### Core Demo: ccusage Live Run (4:30-8:30)

**4:30-5:30 — The 3-second demo.**
[Terminal, already in a project directory.]

"Ready? Here it is."

```
npx ccusage
```

[Output appears: a table showing daily token usage and estimated cost for the past week/month. There should be at least one visible cost spike.]

"That's it. Three seconds. No install. Everything is local — nothing left my machine. And now I can see exactly what happened."

**5:30-6:30 — Reading the output.**
"See this spike on [day]? That was a session where I was [describe a real scenario — e.g., 'debugging a CI pipeline that turned out to be a permissions issue']. Claude tried 6 different approaches before I realized I'd forgotten to tell it about the deployment target. Forty minutes. [$X] in tokens. If I'd seen this data in real-time, I would have stopped and reprompted after approach #2."

"And this is per-project. See the project paths? Each one maps to a client engagement. I can tell you exactly how much Claude cost on [Project A] versus [Project B] this month. That's cost attribution — the thing we need for delivery margins and we currently have zero visibility on."

**6:30-7:30 — The gateway question.**
"But ccusage answers 'how much' — it doesn't answer 'why.' Why did Wednesday cost so much? What happened in that session?"

[Run claude-history — show a quick fuzzy search, find the expensive session.]

"claude-history. Same idea — reads the same local files, gives you a searchable index of every session you've ever had. I found Wednesday's session in 2 seconds. And if I want to see what actually happened, token by token..."

[Show Claude DevTools briefly — open the session, show the token attribution pie chart.]

"Claude DevTools. This shows me that 15% of every turn's context was my CLAUDE.md file. Another 40% was tool outputs — Claude was reading the same 6 files over and over because it kept losing context. That's the 'why.' I trimmed my CLAUDE.md, added a rule about not re-reading files, and my next session on the same task cost half as much."

**7:30-8:30 — The optimization loop.**
"This creates a feedback loop: ccusage shows you the cost, claude-history finds the session, DevTools shows you why it was expensive, you optimize your CLAUDE.md or your prompting approach, and ccusage confirms it worked. Each tool costs you 3 seconds to run. Total workflow: under 5 minutes."

"But notice what all three of these tools have in common: they're *after the fact*. You find out about the problem after it's already cost you money. What if you could prevent the waste in the first place?"

#### Capability Extension: Hooks Preview (8:30-10:30)

**8:30-9:30 — Hooks as quality gates.**
"Claude Code has a hooks system — event handlers that fire when Claude does things. Before a tool call. After a command. Before a commit. And the powerful one: Stop hooks — they can block Claude from proceeding if a condition isn't met."

[Show a simple hook configuration — either a `.claude/hooks.json` or a brief CLAUDE.md rule.]

"This is a 5-line shell script that runs after every Bash command. If the command was a test run and the tests failed, it blocks Claude from continuing until the tests pass. Claude can't ship untested code. Not 'Claude, please run the tests' — Claude *cannot proceed* without passing tests."

**9:30-10:30 — Why this matters for consulting.**
"Think about what this means for client delivery. You can tell a client: 'Every piece of AI-generated code on your project passes the same automated quality gates as human-written code. Not because we ask the AI nicely — because the AI literally cannot proceed without passing them.'"

"That's a capability statement: 'We enforce automated quality assurance on all AI-assisted development.' That's not 'we use Claude Code.' That's a delivery guarantee."

[Brief callback to the Nix talk's capability framing principle.]

#### Honest Costs (10:30-11:30)

**10:30-11:30 — What this doesn't solve.**
"Honest moment. These tools are pre-1.0. Most are solo-developer projects. ccusage has 12k GitHub stars, so it's probably not going anywhere. The others are smaller. The file format Claude Code uses is undocumented — Anthropic could change it tomorrow and break everything."

"There's also no team-level analytics tool that's safe for consulting. Rudel exists, but it uploads your full session transcripts to a server — including client source code, API keys, everything. That's a non-starter for client work. The team analytics layer doesn't exist yet. For now, this is individual-developer tooling."

"And hooks are new. I've tested them; they work. But I don't have 6 months of evidence from a team using them in production. That's the frontier — and it's where we can contribute back to the community."

#### On-Ramp Close (11:30-13:00)

**11:30-12:00 — The one thing.**
"Here's the one thing I want you to take from this talk. After we're done — today, right now — open your terminal and run:"

```
npx ccusage
```

"Three seconds. No install. See your Claude Code spend for the last week. If the number surprises you — and for most people it will — that's your starting point."

**12:00-12:30 — The next steps (for those who want more).**
"If you want to go further:
- `brew install tani/tap/claude-history` or `cargo install claude-history` — searchable session history.
- `brew install --cask claude-devtools` — visual token attribution.
- Ask me about hooks after the talk — I'll share the quality gate configuration."

"All of these are fully local. Nothing leaves your machine. No client data at risk. No security review needed."

**12:30-13:00 — Year 2 continuity callback.**
"Earlier this year, [MC name] showed you QubesOS — how to isolate your environments so that a compromise in one can't reach another. That was adversarial isolation: keeping bad actors out."

"This talk is the other side: operational visibility. Not 'is someone attacking me?' but 'is my own tool working well?' QubesOS keeps your environments isolated. These tools keep your AI usage *visible*. Different threat models, same principle: you can't manage what you can't see."

"Run `npx ccusage`. See what you find. I'll be around after."

---

## 3. Pain Point Opener: "The Invisible Bill"

### Why Cost, Not Security

The gap analysis identified four "one thing" candidates and recommended hooks. But the evaluation above shifts the recommendation to ccusage based on the on-ramp principle. The pain-point opener must match.

Considered and rejected pain-point openers:

**Option A: "Claude committed client credentials to a shared replay."**
- Problem: This hasn't happened (that we know of). Fabricating a horror story feels manipulative. The audience will wonder "did that really happen?" and if the answer is no, credibility drops.
- Also: claude-replay isn't something most people use yet. The audience can't relate to a pain they haven't experienced.

**Option B: "A junior developer's Claude Code session went in circles for 2 hours and nobody noticed."**
- Problem: This is real, but it sounds like a management problem ("why wasn't someone reviewing their work?"), not a tools problem. It shifts blame rather than creating shared identification.

**Option C: "You got a $2,400 Claude Code bill and you have no idea which client engagement it's from."**
- Problem: Good, but only hits engineering managers. Individual developers don't see the aggregate bill. Needs to be personal.

**Selected opener: "The Invisible Bill" (hybrid of B and C)**

The most effective opener combines personal experience with consulting-specific stakes:

> "Last month, one of our teams burned through $1,200 in Claude Code tokens in a single week. They didn't know until the invoice arrived. The sessions that cost the most? Not the productive ones — the ones where Claude went in circles for 40 minutes trying an approach that was never going to work."

This works because:
1. **It's plausibly real.** $1,200/week for a team is within the range documented in the agentic-workflow-state-of-art spike (~$12k/month/team).
2. **It's a cost problem.** Consultants are trained to think about delivery costs and margins.
3. **It implies both waste and blindness.** The team didn't know — that's the problem this talk solves.
4. **The "wrong sessions were most expensive" twist** creates a curiosity gap (Loewenstein). The audience wants to know: which of *my* sessions were the expensive ones?

### The Cold-Sweat Equivalence

In the Nix talk, the cold-sweat moment was "deploying to the wrong client's AWS account." The equivalent here:

**Primary**: "You burned $400 on a Claude Code session that went in circles for 45 minutes and you didn't notice until the monthly bill came."

**Escalated** (for later in the talk): "Your CLAUDE.md file — the one you wrote 6 months ago and forgot about — is consuming 15% of every single turn's context window. Every session, every prompt, 15% wasted before Claude even starts thinking about your actual question."

The escalated version is more visceral for technical audiences because it's not just money — it's *their own configuration* actively making things worse, and they had no way to know.

---

## 4. The On-Ramp: What Can Someone Do in 5 Minutes After the Talk?

### The "Start with direnv" Equivalent

The Nix talk closed with "start with direnv in one project, stop whenever you want." The equivalent:

**Primary on-ramp**: "Run `npx ccusage` on your current project. Takes 3 seconds."

This is actually a *better* on-ramp than the Nix talk's because:
- `npx ccusage` requires zero installation (npx downloads and runs temporarily)
- It produces visible results in 3 seconds (vs. direnv requiring a `.envrc` file)
- It works on any project where Claude Code has been used (no setup)
- The output is immediately meaningful (cost data)

### Tiered On-Ramp (for the slide/follow-up)

| Tier | Action | Time | What You Get |
|------|--------|------|-------------|
| **Now** (during the talk) | `npx ccusage` | 3 seconds | See your Claude Code spend |
| **Tonight** | `brew install tani/tap/claude-history` | 2 minutes | Searchable session history |
| **This week** | `brew install --cask claude-devtools` | 5 minutes | Visual token attribution per session |
| **This month** | Add a Stop hook to your project | 30 minutes | Automated quality gate on AI-generated code |

### Post-Talk Resource

Prepare a single page (internal wiki or Slack post) with:
1. The four commands above
2. A sample Stop hook configuration (5 lines of bash)
3. Links to each tool's GitHub repo
4. A note: "All tools are fully local. Nothing leaves your machine. No security review needed for tiers 1-3."

This mirrors the Nix talk's devenv.sh addendum — a concrete follow-up resource for people who want to go further.

---

## 5. The MC Bridge from QubesOS

### The 2-Sentence Callback

QubesOS was about **adversarial isolation** — keeping untrusted code in separate VMs so a compromise can't spread. This talk is about **operational visibility** — seeing what your own AI tools are actually doing so waste and quality problems can't hide.

**MC bridge script:**

> "Last time, [speaker] showed us QubesOS — how to build walls between your environments so that if one is compromised, the damage can't spread. That was about threats from the outside. Today, [speaker] is going to show you something about threats from the *inside* — not malicious ones, but the invisible costs and quality gaps hiding in the AI tools you're already using every day."

### Why This Bridge Works

1. **It reframes "inside" vs "outside" without being heavy-handed.** QubesOS = external threats. Claude Code tools = internal blind spots. Same principle (you can't manage what you can't see), different application.

2. **It doesn't require the audience to remember QubesOS details.** The bridge works even if they missed the QubesOS talk entirely — "threats from outside vs. threats from inside" is self-explanatory.

3. **It maintains the Year 2 arc.** The implicit narrative: Year 1 gave you evidence and workflow. Year 2 gives you control — first over your environment (QubesOS), now over your tools (observability).

---

## 6. Talk Design Summary

### The One Sentence

**"Run `npx ccusage` — see where your Claude Code money is going in 3 seconds."**

This is the Carmen Simon "one thing." It is:
- A single command
- A single tool
- A single action
- Produces an immediate, visible result
- Zero friction (no install, no config, fully local)
- Consulting-relevant (cost = delivery margin)

### The Arc in One Paragraph

Open with the invisible bill problem (you're spending money on AI and you can't see where it goes). Reveal that the data already exists on your machine. Show ccusage — 3 seconds, see your spend. Show the investigation chain (claude-history to find sessions, DevTools to see token attribution). Demonstrate the optimization loop (find waste, fix CLAUDE.md, verify improvement). Preview hooks as the next frontier (prevent waste, not just detect it). Be honest about limitations (pre-1.0 tools, no team analytics yet). Close with the on-ramp: run `npx ccusage` right now.

### Comparison to Nix Talk

| Element | Nix Talk | Claude Tools Talk |
|---------|----------|-------------------|
| Pain point | Deployed to wrong AWS account | Burned $1,200 in Claude tokens with no visibility |
| One thing | `cd` and everything switches | `npx ccusage` and you see your spend |
| Demo | cd into client dir, watch env/prompt/creds switch | Run ccusage, find the cost spike, trace to session |
| Capability frame | "We onboard in 1 hour with identical environments" | "We track and optimize AI-assisted delivery costs per engagement" |
| Honest costs | Learning curve, disk usage, error messages | Pre-1.0 tools, no team analytics, format undocumented |
| On-ramp | "Start with direnv in one project" | "Run `npx ccusage` on your current project" |
| Year continuity | Sets up QubesOS (accidental → adversarial) | Callbacks QubesOS (adversarial → operational) |

### What Makes This Talk Different from a Generic "AI Tools" Talk

1. **Consulting-specific framing throughout.** Every pain point ties to delivery margins, client engagements, and professional accountability — not individual developer curiosity.
2. **The on-ramp is genuinely zero-friction.** Most tool talks end with "go read the docs and set it up." This talk ends with "type this command right now."
3. **The Year 1 foundation does the heavy lifting.** Because Q3 (AI Evidence) established the perception gap and Q4 (RPI) established structured AI workflow, this talk doesn't need to justify *whether* to use AI tools or *how* to structure AI work. It only needs to answer: "how do you know it's working?" That narrow focus is what makes 15 minutes sufficient.
4. **Honest about what doesn't exist yet.** The team analytics gap, the pre-1.0 risk, the undocumented format — these are stated plainly, building credibility for the things that are recommended.

---

## Depth Checklist

- [x] **Underlying mechanism explained**: Evaluated 4 candidates against 6 criteria with scored rationale. Talk structure designed minute-by-minute with specific content at each beat.
- [x] **Key tradeoffs identified**: ccusage vs hooks as "one thing" (on-ramp friction vs capability story), cost-first vs security-first opener, how much to show vs how much to tease.
- [x] **Compared to alternatives**: All 4 candidates evaluated head-to-head. Talk structure compared to Nix talk element-by-element.
- [x] **Failure modes described**: What happens if hooks are chosen but lack empirical evidence (GAP 3). What happens if before/after is chosen (scope explosion in 15 min). What happens if replay is chosen (passive, not actionable).
- [x] **Concrete examples**: Specific terminal commands, specific cost numbers, specific CLAUDE.md percentages, specific hook configuration, specific MC bridge script.
- [x] **Standalone-readable**: Sufficient to build the talk's implementation plan without re-reading source material.

## Sources

### On-Disk Research
- `implementation-plans/nix-consulting-cop-talk/plan.md` — Design principles, structure model, on-ramp pattern
- `research-spikes/practitioner-presentation-skills/research.md` — Carmen Simon "one thing," 15-minute format, audience psychology
- `research-spikes/practitioner-presentation-skills/engagement-tactics-research.md` — Live demo engagement, "who here has..." technique, storytelling beats
- `research-spikes/practitioner-presentation-skills/fifteen-minute-format-research.md` — Structure A (Problem/Demo/Takeaway), timing allocations, scope discipline
- `research-spikes/claude-tools-consulting-adoption/gap-analysis-research.md` — "One thing" candidates, hooks-in-practice gap (GAP 3), Year 2 positioning
- `research-spikes/claude-tools-consulting-adoption/adoption-strategy-research.md` — ccusage as on-ramp, 4-step progression, champion model, resistance patterns
- `research-spikes/claude-tools-consulting-adoption/consulting-tool-selection-research.md` — Tool-by-tool consulting analysis, privacy hierarchy
- `research-spikes/claude-tools-consulting-adoption/privacy-compliance-research.md` — Data classification, /feedback risk, Rudel transcript upload risk
