# gdev Elevator Pitch Playbook

> Open this 5 minutes before any conversation about gdev. Pick the audience, pick the length, deliver.

---

## 1. Core Pitch DNA

Every pitch variant -- hallway, HN post, leadership briefing -- must contain these vertebrae. Drop one and the pitch collapses.

| # | Vertebra | What It Sounds Like | Why It's Load-Bearing |
|---|----------|---------------------|-----------------------|
| 1 | **One command, 60 seconds** | "60 seconds from clone to working devenv shell" | The transformation claim. If the listener remembers one thing, it's this. Replaces 30-90 minutes of manual work -- the contrast sells itself. |
| 2 | **Security by default, not by opt-in** | "6 defense layers configured automatically" | The differentiator. No other environment generator ships security hardening. This is what separates gdev from "I could script this myself." |
| 3 | **Fully reversible** | "`gdev teardown` removes everything" | The risk eliminator. Developers and leadership both fear lock-in. Reversibility converts skeptics into trial users. |
| 4 | **Provable, not promissory** | "Every defense has a test fixture" / "Posture score: A, 92/100" | The credibility anchor. Specific, verifiable claims survive the "96% assume you're lying" filter. Use this vertebra for security and leadership audiences; optional for hallway dev chats. |

**Rule**: Every pitch must hit at least vertebrae 1-3. Add #4 for security engineers and CTOs.

---

## 2. One-Liner Variants

Target: 15 words. Usable without modification.

### GitHub Repo Description

1. **"One command to a security-hardened dev environment with AI guardrails"** (11 words)
   Best. Outcome-led, no jargon, immediately communicates scope. Follows the Pattern A (What + Differentiator) that uv/esbuild use successfully.

2. **"Replaces 30-90 minutes of manual devenv setup with one secure, AI-configured command"** (12 words)
   Strong. The replacement pattern (uv's "replaces pip, pip-tools...") works because it maps to known pain. Use if the audience is devenv.sh-familiar.

3. **"Generate devenv.nix, Claude Code config, pre-commit hooks, and 6 security layers in 60 seconds"** (14 words)
   Technical audience only. Replacement list pattern with specifics. Too dense for general discovery but strong for Nix-adjacent communities.

### Slack / Discord Introduction

1. **"Tired of hand-configuring devenvs? `gdev init` detects your project and generates everything -- security hardened, AI configured, 60 seconds."** (18 words)
   Best. Opens with shared pain, names the command, delivers the outcome. Conversational tone works in chat.

2. **"One CLI to set up your whole dev environment -- devenv.nix, Claude Code rules, pre-commit hooks, security layers. 60 seconds, fully reversible."** (21 words)
   Good. Replacement list pattern. Slightly longer but covers more scope.

3. **"I built a thing that automates the 90 minutes of boilerplate at the start of every project. Security included. `gdev init`."** (20 words)
   Good for developer Discords. "I built a thing" is the highest-trust opener in open source communities.

### Tweet / Social Media

1. **"Every new project: same 90 minutes of setup. `gdev init`: 60 seconds to a security-hardened devenv. Open source."** (17 words)
   Best. Contrast hook (90 min vs 60 sec), names the command, "open source" is the trust signal.

2. **"Stop configuring dev environments by hand. One command. 60 seconds. 6 security layers. Fully reversible. github.com/..."** (15 words + URL)
   Good. Staccato rhythm works on social. Each phrase is a vertebra.

3. **"Your team is running Claude Code with zero deny rules. `gdev init` fixes that -- plus devenv, hooks, and security -- in 60 seconds."** (22 words)
   Use for security-aware audiences. The conditional threat hook ("zero deny rules") creates urgency.

---

## 3. 30-Second Pitches (~75 words)

### Developer (Hallway at Meetup)

> **Structure**: Before-After-Bridge | **Frame**: Gain (productivity) | **Tone**: Casual, peer-to-peer

How long does it take you to set up a new project? For me it used to be 90 minutes every time -- devenv.nix, Claude Code config, pre-commit hooks, security tools. Same boilerplate, different repo.

I built a CLI that detects your project and generates all of it in one command. 60 seconds to a working, security-hardened dev shell.

It's open source and fully reversible -- `gdev teardown` removes everything. Try it on any project.

*Why this works*: Question hook invites dialogue instead of pitching at someone. "I built" establishes authenticity. One transformation claim (90 min to 60 sec). Closes with reversibility (risk reducer) and a clear next step.

### Team Lead (Meeting Intro)

> **Structure**: Problem-Solution-Benefit | **Frame**: Consistency + drift risk | **Tone**: Technical, outcome-oriented

How consistent are your projects' security configs right now? devenv.nix, pre-commit hooks, Claude Code permissions -- do they all match?

gdev enforces a baseline across every project. One command generates everything; `gdev check` validates it in CI. When something drifts, you know in 100ms.

I can run it on one of your repos right now -- takes 60 seconds, and `gdev teardown` removes it cleanly if you don't like what you see.

*Why this works*: Opens with the consistency question team leads already worry about. "Drift" is their language. Offers a concrete demo (low commitment) and names the exit path.

### CTO (Leadership Briefing)

> **Structure**: Problem-Agitate-Solve | **Frame**: Loss (risk, compliance) | **Tone**: Business impact

Your developers are using Claude Code across every project. How many deny rules do they have configured? For most teams, it's zero.

That means every AI-suggested package install runs unguarded -- no age-gating, no vulnerability checking. And when auditors ask for evidence of your security controls, you're assembling it manually from 30 repos.

gdev configures 48 deny rules, 6 independent defense layers, and produces audit-ready compliance evidence -- one command, 60 seconds. We can pilot it on 2-3 projects this week. It's fully reversible.

*Why this works*: Loss frame with a specific gap (zero deny rules). Agitate step makes the consequence visceral (unguarded installs + manual audit scramble). Two value drivers from the 3-value-driver rule: risk reduction and compliance automation. Closes with a bounded pilot proposal and reversibility.

### Security Engineer (Technical Evaluation)

> **Structure**: Hook-Mechanism-Proof | **Frame**: Defense terminology | **Tone**: Precise, no marketing

How do you currently prove each of your supply chain defenses actually works?

gdev deploys 6 independent defense layers: package age-gating, install script blocking, lock file enforcement, vulnerability scanning, AI agent guardrails, and hardened Nix evaluation. Each layer has an EICAR-equivalent test fixture.

Run the test suite: it publishes a fresh package to a local Verdaccio, attempts to install it, and proves age-gating blocks it. Same pattern for every layer. I can walk you through the fixtures.

*Why this works*: Opens with provability -- the thing security engineers care about most. "EICAR-equivalent" is precise terminology that signals you understand their world. Proof step is concrete and auditable. Closes with an offer to go deeper, not a sales push.

---

## 4. 60-Second Pitches (~150 words)

### Developer (Conference Hallway)

> **Structure**: Before-After-Bridge (expanded) | **Frame**: Gain | **Tone**: Casual

Think about the last time you started a new project. How long before you actually wrote code? You probably spent an hour configuring devenv.nix, setting up Claude Code permissions, wiring pre-commit hooks, bolting on security tools -- the same ritual you've done a dozen times.

Now picture this: you clone the repo, run one command, and 60 seconds later you're in a working dev shell. devenv.nix generated for your ecosystem. Claude Code configured with 48 deny rules. Pre-commit hooks, vulnerability scanning, the whole stack -- detected from your project files, not copied from some other repo.

That's what gdev does. It detects your project type -- Go, TypeScript, Python, 27 ecosystems -- and generates everything. Security hardened by default. And if you don't like any of it, `gdev teardown` removes every generated file. No lock-in.

Try `gdev init` on any project tonight. It takes less time than reading a README.

### Team Lead (Planning Meeting)

> **Structure**: Problem-Solution-Benefit (expanded) | **Frame**: Consistency + velocity | **Tone**: Outcome-oriented

Here's a question: across your 10 or 20 projects, how many have the same security configuration? Same pre-commit hooks? Same Claude Code deny rules? In my experience, it's close to zero -- every project is a snowflake with different gaps.

gdev fixes this. One command -- `gdev init` -- detects your project ecosystem and generates a consistent baseline: devenv.nix, Claude Code configuration with 48 deny rules, pre-commit hooks, CI security workflows. 60 seconds per project. For returning developers, `gdev init --mode join` reads the team config and gets them productive in 2 minutes flat.

But consistency is only half the story. `gdev check` runs in CI and catches drift before it reaches main. `gdev status` gives you a posture score -- 0 to 100, A through F -- across every project. When something degrades, you see it in 100ms, not at the next audit.

I'd suggest trying it on one project this sprint. It's fully reversible -- `gdev teardown` cleans up everything.

### CTO (Executive Briefing)

> **Structure**: Problem-Agitate-Solve (expanded) | **Frame**: Loss + business impact | **Tone**: Risk, metrics

Every project in your org has a different security posture. Some have pre-commit hooks. Some scan dependencies. Some have Claude Code guardrails -- most don't. When the next audit lands, you'll be pulling evidence manually from every repo. And 92% of malicious PyPI packages are less than 24 hours old -- if your projects aren't checking package age, that's an open door.

gdev makes this a solved problem. One command configures 6 independent defense layers, 48 Claude Code deny rules, and generates audit-ready compliance evidence mapped to SOC2 and HIPAA controls -- with SHA256-hashed artifacts. 60 seconds per project. `gdev status` gives every project a posture score your team leads can track. `gdev evidence` produces the report your auditors actually want.

The cost of manual security configuration across 20 projects is weeks of engineer time. gdev makes it zero. We can pilot on 2-3 projects in a week -- it's MIT-licensed, zero infrastructure cost, and fully reversible with `gdev teardown`. If it doesn't prove its value in the pilot, removing it takes one command.

### Security Engineer (Deep Technical)

> **Structure**: Hook-Mechanism-Proof (expanded) | **Frame**: Defense depth + provability | **Tone**: Precise

Here's a question I've never gotten a satisfying answer to: how do you prove your supply chain defenses actually work -- not that they're configured, but that they're functioning?

gdev takes a defense-in-depth approach with 6 independent layers. Package age-gating blocks anything published in the last 72 hours -- that alone catches 92% of PyPI malware. Install script blocking prevents arbitrary code execution during `npm install`. Lock file enforcement stops unapproved dependency changes. Vulnerability scanning via Grype catches known CVEs. PreToolUse hooks gate every AI-suggested package install through OSV checks. And hardened Nix evaluation ensures reproducible, isolated builds.

Here's what makes it different: every layer has an EICAR-equivalent safe test fixture. The age-gating test publishes a fresh package to a local Verdaccio instance, attempts to install it, and proves the gate blocks it. The install script test uses a @lavamoat canary package. Each defense is provably working or provably broken -- no trust required.

I'll also tell you what it doesn't cover: runtime behavior monitoring, network egress controls, and secrets in environment variables are out of scope. Want to see the test fixtures?

---

## 5. Show HN Launch Post

**Title**: Show HN: gdev -- One command to a security-hardened dev environment with AI guardrails

---

I got tired of the same ritual at the start of every project: write devenv.nix from scratch, configure Claude Code deny rules, set up pre-commit hooks, bolt on security scanning, wire up CI workflows. 90 minutes of boilerplate that looked almost identical every time but was never quite reusable.

So I built gdev. It's a Go CLI that detects your project (Go, TypeScript, Python -- 27 ecosystems), then generates everything in one command: devenv.nix, Claude Code configuration with 48 deny rules, pre-commit hooks, CI security workflows, and 6 independent security layers including package age-gating and install script blocking.

The security piece isn't a checkbox. Each defense layer has an EICAR-equivalent test fixture -- you can prove age-gating works by watching it block a fresh package published to a local Verdaccio. Same pattern for every layer. I wanted "provably secure" to mean something.

Some honest limitations: gdev generates configs for devenv.sh -- it doesn't replace it. If your ecosystem isn't in the 27 supported, you'll need manual config. And the security layers are supply-chain focused; runtime monitoring is out of scope.

It's MIT-licensed, zero dependencies (static binary), and fully reversible (`gdev teardown` removes everything). I've been using it on my own projects for the past few months and it's genuinely changed how I start new work.

Would love feedback on the detection logic and security model. Repo: [link]

---

## 6. Context-Specific Cheat Sheet

Open this, find your row, deliver.

| Context | Structure | Lead With | Hook Type | Length | Close With |
|---------|-----------|-----------|-----------|--------|------------|
| **GitHub repo description** | Noun + differentiator | Outcome | -- | 15 words | -- |
| **Tweet / social** | Contrast + CTA | Time savings | Specific number | 15-20 words | Repo link |
| **Slack / Discord intro** | Problem-outcome | Shared frustration | Question or "I built" | 20-25 words | Command to try |
| **Hallway at meetup** | Before-After-Bridge | Shared pain | Question: "How long does X take you?" | 30 sec (~75 words) | Question: "Have you dealt with this?" |
| **Team meeting intro** | Problem-Solution-Benefit | Consistency gap | Consistency question | 45-60 sec (~100 words) | Offer: "Want me to try it on your repo?" |
| **Leadership briefing** | Problem-Agitate-Solve | Risk / compliance gap | Conditional threat: "zero deny rules" | 60 sec (~150 words) | Pilot proposal: "2-3 projects this week" |
| **Security review** | Hook-Mechanism-Proof | Defense provability gap | "How do you prove...?" | 45-60 sec (~120 words) | Offer: "Walk through test fixtures?" |
| **HN Show launch** | Personal + problem + mechanism | Origin story: "I got tired of..." | "I built" | 200-250 words | Invite feedback + repo link |
| **Internal champion pitch** | JTBD + before/after metrics | "Here's the job I needed done" | Personal experience | 60 sec + metrics | "Try it on your project this sprint" |

### Quick-Reference: Audience to Frame

| Audience | Frame | Why |
|----------|-------|-----|
| Individual developer | **Gain** (productivity) | Devs think in "what do I get" |
| Team lead / staff eng | **Gain** (consistency) + light **loss** (drift) | They manage multiple projects |
| CTO / VP Eng | **Loss** (risk, compliance) | Leadership thinks in "what could go wrong" |
| Security engineer | **Loss** (defense gaps) + **gain** (provability) | Security is fundamentally about risk |

---

## 7. Pre-Delivery Checklist

Run through this before any pitch. If you can't check every box, revise.

- [ ] **Starts with pain, not product.** First sentence is a question or problem statement, not "gdev is..."
- [ ] **No Nix jargon.** Someone who has never used Nix can follow every sentence. (Exception: pitching to known Nix users.)
- [ ] **One transformation claim.** The pitch communicates exactly one "before to after." Not three features -- one change.
- [ ] **Numbers are specific and verifiable.** "60 seconds" not "fast." "6 defense layers" not "secure." "48 deny rules" not "comprehensive guardrails." Use 1-2 numbers per pitch, no more.
- [ ] **Clear next step.** The pitch ends with one action: try a command, see a demo, look at test fixtures, approve a pilot. One, not three.
- [ ] **Fits the time budget.** You've timed yourself and it lands within the target (30 sec or 60 sec). Not 25, not 40.
- [ ] **Survives the skeptic test.** A developer who assumes you're lying would still find the claims credible -- because they're bounded, specific, and verifiable.
- [ ] **Uses replacement, not analogy.** You list what gdev replaces (manual devenv.nix, Claude Code config, pre-commit hooks, security tools), not "it's like X for Y."
- [ ] **Includes the exit.** You've mentioned `gdev teardown` or reversibility. No one commits to a tool they can't leave.
