# gdev Deep-Dive Presentation: Complete Presenter's Playbook

> **Format**: 55 minutes content + 5 minutes Q&A
> **Audience**: Mixed developer + leadership (adjustable per Section 6)
> **Last updated**: 2026-05-15

---

## 1. Presentation Overview

### Title Options

| Context | Title |
|---------|-------|
| **Internal engineering all-hands** | *One Command to a Secure Dev Environment: How gdev Changes Your First Hour on Every Project* |
| **Conference talk** | *Defense in Depth by Default: Building Security-Hardened Dev Environments That Developers Actually Use* |
| **Leadership decision meeting** | *From 90 Minutes to 60 Seconds: Automated Security Compliance for Every Project* |

### Duration

55 minutes of structured content across 6 modules, followed by 5 minutes of Q&A. Each module is approximately 10 minutes, with an emotional attention reset between every module per Medina's Brain Rules research.

### Audience Assumptions

The default outline targets a mixed audience: individual developers, Staff+ engineers, platform leads, and one or more engineering leaders (VP Eng / CTO). Developers outnumber leaders roughly 4:1. Audience familiarity with Nix/devenv is helpful but not assumed. Section 6 provides adjustment notes for pure-developer, pure-leadership, and conference variants.

### Key Objectives

After this presentation, the audience should:

- **Know**: What gdev does (one-command generation), how the 6-layer defense model works, and what the adoption path looks like
- **Feel**: Excited about the productivity gain, confident that security is handled, and reassured that adoption is reversible
- **Do**: Try `gdev init` on a real project within 48 hours, or (for leaders) approve a pilot program

---

## 2. Module-by-Module Outline

### Module 1: The Problem (Minutes 0:00 - 10:00)

**Title**: *The First Hour on Every Project*

**Duration**: 10 minutes

**Opening Hook** (Narrative transportation -- bypass evaluative resistance):

> "Let me tell you about last month. A new engineer joined our team -- experienced, senior, ready to go. They cloned the repo at 9 AM. By noon, they were still configuring their environment. devenv.nix, .envrc, Claude Code settings, pre-commit hooks, security tools. Three hours of copying configs from another project, asking colleagues what settings to use, and debugging Nix errors. Meanwhile, Claude Code was running with zero guardrails -- no deny rules, no package age checks, nothing. That's not an onboarding failure. That's Tuesday."

**Core Content**:

| Time | Content | Technique |
|------|---------|-----------|
| 0:00 - 1:30 | Opening story. Deliver it conversationally, not from slides. Make eye contact. Let the details land -- "three hours," "zero guardrails," "that's Tuesday." | Narrative transportation |
| 1:30 - 2:30 | Audience check-in. "Raise your hand if this has happened on your team in the last six months." Pause. Let hands go up. "Keep your hand up if it's happened more than once." | Participation -- establishes shared experience |
| 2:30 - 4:30 | The manual setup tax. Show a slide listing the 15+ files a developer creates manually for each project: devenv.nix, devenv.yaml, .envrc, .claude/settings.json, CLAUDE.md, .pre-commit-config.yaml, .github/workflows/security.yml, package manager security configs, .gitignore additions, .mcp.json, editor settings, and more. "This is what 'just set up the project' actually means." | Visual evidence -- make the invisible visible |
| 4:30 - 6:30 | The security gap. Transition with: "But the time cost isn't even the dangerous part." Briefly present the supply chain attack landscape. Not a scare piece -- factual, concise. "SolarWinds: 18,000 organizations through a routine update. $90 million in recovery. The CISO was charged with fraud. ua-parser-js: 7 million weekly downloads, hijacked maintainer account, 4 hours to detection. The attack vector in every case was the same: routine package installation." Then the pivot: "92% of malicious PyPI packages are published and removed within 24 hours. Raise your hand if your projects check package age." Pause. | Loss framing -- factual, not fear-mongering |
| 6:30 - 8:30 | The cost quantified. "Let's do the math. 30 to 90 minutes per project per developer. You have 50 developers and 20 active projects. That's 1,000 to 3,000 hours per year -- just on setup. At $75 an hour, that's $75,000 to $225,000 in engineering time that produces zero features. And none of those setups are consistent. None of them are auditable. None of them check package age." | Quantified impact -- gives leaders a number to anchor on |
| 8:30 - 10:00 | Transition hook. Lower your voice slightly. "What if there was one command that did all of this? One command that generated every file, configured every security layer, and got your developer to productive in 60 seconds instead of 60 minutes?" Pause two beats. "Let me show you." | Curiosity gap -- the bridge to Module 2 |

**Key Talking Points** (actual sentences to deliver):

- "This isn't about any one tool being hard. It's that the combination of 15 files across 6 directories with 47 configuration decisions is what makes every project's first hour identical yak-shaving."
- "The security problem isn't that developers don't care. It's that nobody has time to research which PyPI packages are safe when they're already three hours into setup."
- "Manual configuration doesn't scale. The tenth project gets worse security than the first because people are tired of the process."

**Slides Needed**:

1. **Slide 1** -- Title slide. Presentation title, your name (small), date. No company history, no bio, no agenda.
2. **Slide 2** -- The 15 files. A single visual: a directory tree showing every file that gets created manually. Each filename is labeled with estimated time. Total at the bottom: "30-90 minutes." This slide stays up during the manual setup discussion.
3. **Slide 3** -- Supply chain attack timeline. A simple horizontal timeline showing SolarWinds (2020, 14 months undetected), Codecov (2021, 2+ months), ua-parser-js (2021, 4 hours), event-stream (2018, months). Not a wall of text -- one line per attack with the key number.
4. **Slide 4** -- The cost math. Three numbers stacked large: "50 developers x 20 projects x 60 min = **1,000 hours/year**" and the dollar figure below. Visual, not a spreadsheet.

**Transition to Module 2**: "Let me show you." Walk to the terminal. (Physical movement signals modality change and recaptures attention.)

---

### Module 2: One Command (Minutes 10:00 - 20:00)

**Title**: *The 60-Second Environment*

**Duration**: 10 minutes

**Opening Hook** (Live demo -- the aha moment):

Open a terminal. The audience sees a real project directory. No slides. The terminal IS the presentation.

> "This is a real project. Go backend, TypeScript frontend, Docker config, Terraform modules. No gdev artifacts. No devenv.nix. No Claude Code configuration. Nothing. Watch what happens."

**Core Content**:

| Time | Content | Technique |
|------|---------|-----------|
| 10:00 - 11:00 | Set the stage. Terminal is visible, large font (28pt+), simplified prompt. "This is a real project directory. Let me show you what's here." Quick `ls` or `tree` showing existing project files -- no gdev artifacts. | Context establishment |
| 11:00 - 14:00 | **LIVE DEMO: `gdev init`**. Type the command live. Narrate as output scrolls: "It's detecting the project... Go module found... TypeScript package.json... Docker... Terraform... Now it's generating devenv.nix... Claude Code settings with deny rules... pre-commit hooks... CI security workflows..." Let the output finish. If using `--profile go-web --yes`, the whole thing runs unattended. Otherwise, walk through 2-3 wizard prompts conversationally. The timer should show roughly 60 seconds. | The aha moment -- must be live for credibility |
| 14:00 - 16:00 | Reveal the generated files. `ls` the directory again. Highlight the new files: devenv.nix, devenv.yaml, .envrc, .claude/settings.json, CLAUDE.md, .pre-commit-config.yaml, etc. Open one or two briefly -- not a line-by-line walkthrough. Show the devenv.nix: "Look -- correct Go packages, TypeScript toolchain, formatters, linters. These are the same packages you'd choose yourself." Show settings.json: "48 deny rules. PreToolUse hooks. Context budget configuration." Pause. "All of this, in 60 seconds." | Progressive disclosure -- outcomes first, then peek at mechanism |
| 16:00 - 17:30 | Enter the environment. `devenv shell`. Show that tools are available. Run a quick command (`go version`, `node --version`). Then: `gdev status`. Show the posture score: A (92/100). "This project just went from 'no security posture data' to 'A grade, 92 out of 100' in one minute. Every defense layer is active. Every tool is configured." | Making the invisible visible -- posture score is the leadership aha |
| 17:30 - 18:30 | The "what if I don't like it?" beat. Briefly: "And if you decide this isn't for you -- `gdev teardown`. Everything gdev generated is removed. Standard files you can maintain yourself in the meantime. You're never locked in." This is a trust-building beat, not a feature walkthrough. | Objection pre-emption -- reversibility reduces adoption risk |
| 18:30 - 20:00 | Checkpoint Q&A. "Before we go deeper into how this works -- any questions about what you just saw?" Take 1-2 questions if they come. If not, bridge directly. | Engagement checkpoint |

**Key Talking Points**:

- (Before typing `gdev init`): "I'm going to type one command. Everything you just saw on that list of 15 files -- devenv.nix, Claude Code config, pre-commit hooks, security hardening, CI workflows -- will be generated in about 60 seconds."
- (While output scrolls): "Notice it's detecting each ecosystem independently. Go module, TypeScript, Docker, Terraform -- each one gets its own security configuration. That's 27 ecosystems it can detect."
- (After posture score): "That A grade isn't a vanity metric. It means 6 independent defense layers are active. Package age-gating, install script blocking, lock file enforcement, vulnerability scanning, AI guardrails, and hardened Nix evaluation. Each one is working right now."
- (Reversibility): "gdev generates standard files -- devenv.nix, settings.json, YAML configs. If gdev disappeared tomorrow, your project still works. These are files you'd maintain yourself. gdev just wrote them correctly and consistently."

**Slides Needed**:

5. **Slide 5** -- "One Command" in large text. Show below it: `gdev init`. This slide appears for exactly 3 seconds before switching to the terminal. It's a section header, not content.
6. **Slide 6** -- (Post-demo, optional) Side-by-side: "Before" directory listing vs "After" directory listing. Use this only if the live demo fails and you need to show a static comparison.

**Transition to Module 3**:

> "So that's what gdev does. Now let me show you WHY it works -- what's happening under the hood. This is the part that matters when you're evaluating whether to trust generated configuration."

---

### Module 3: How It Works (Minutes 20:00 - 30:00)

**Title**: *Architecture for Skeptics*

**Duration**: 10 minutes

**Opening Hook** (Split-level announcement -- set expectations for mixed audience):

> "We're going to go under the hood now. If you're a platform engineer or Staff+ engineer, this is for you -- this is how you evaluate whether gdev's generated artifacts are trustworthy. If you're in a leadership role, the key takeaway from this section is one thing: gdev generates standard files, doesn't own your config, and tracks every change it makes. I'll point out when we hit the business-relevant parts."

**Core Content**:

| Time | Content | Technique |
|------|---------|-----------|
| 20:00 - 21:00 | The split-level announcement (above). Explicitly set expectations so non-technical attendees know they can relax and re-engage when you signal. | Audience management |
| 21:00 - 23:30 | Detection engine. "When you run `gdev init`, the first thing that happens is detection. gdev scans your project in under 100 milliseconds -- no network calls, purely local -- and identifies which ecosystems are present. It looks for go.mod, package.json, Cargo.toml, requirements.txt, and 23 other markers." Show a slide with the detection flow: project files on the left, an arrow to the detection engine in the center, and a list of detected ecosystems on the right with confidence scores. "Each detection is scored. If gdev is uncertain, it asks. If it's confident, it proceeds. This is why the wizard can be so fast -- it already knows your project." | Zoom level 2 -- how detection works |
| 23:30 - 26:00 | Generation pipeline. "Once detection is done, the template engine generates files for each ecosystem. Each template produces standard, idiomatic configuration -- the devenv.nix a senior Nix engineer would write. The key technical detail: every generated file is tracked with a SHA256 hash. When you run `gdev init --update` later, gdev knows exactly which files it generated, which ones you've modified, and which ones need a three-way merge to update safely." Show the architecture slide: 3-addon model (devenv, claudecode, devinit). Keep it simple -- three boxes with arrows. | Zoom level 3 -- generation pipeline |
| 26:00 - 28:00 | The trust question. "Here's what matters for evaluation: gdev generates files, it doesn't own them. Your devenv.nix is a real devenv.nix. Your settings.json is a real Claude Code settings file. You can edit them. You can version-control them. If you stop using gdev, nothing breaks -- you just maintain those files yourself. The SHA256 tracking means gdev never silently overwrites your changes." (Leadership flag): "This is the lock-in answer. There is none. `gdev teardown` removes everything cleanly. The generated files are the industry-standard formats." | Technical trust building |
| 28:00 - 30:00 | Quick `gdev info` demo. Run `gdev info` to show project context in <100ms: detected ecosystems, active tools, compliance level, last update. "This is what gdev knows about your project. Under 100 milliseconds, all local, no network calls." | Brief demo reinforcing speed |

**Key Talking Points**:

- "Detection runs in under 100 milliseconds because it's entirely local. No API calls, no cloud dependency. Your project structure never leaves your machine."
- "The SHA256 hash tracking is what separates gdev from a scaffolding tool. Scaffolding tools generate once and walk away. gdev tracks what it generated, detects what you changed, and updates safely."
- "The 3-addon architecture -- devenv, claudecode, devinit -- means each concern is independently maintained. The devenv addon handles Nix generation. The claudecode addon handles AI agent configuration. The devinit addon handles project scaffolding. They compose but don't depend on each other."

**Slides Needed**:

7. **Slide 7** -- Section header: "Under the Hood" with a subtitle "How gdev generates trustworthy configuration"
8. **Slide 8** -- Detection flow diagram. Left: project files (go.mod, package.json, etc.). Center: detection engine with "<100ms" label. Right: detected ecosystems with confidence scores (Go: 95%, TypeScript: 90%, Docker: 85%).
9. **Slide 9** -- 3-addon architecture. Three boxes (devenv, claudecode, devinit) with arrows showing their outputs: devenv.nix/.envrc, settings.json/CLAUDE.md, and project scaffold files respectively.
10. **Slide 10** -- SHA256 tracking visual. Shows a generated file, its hash, a user edit (hash changes), and the three-way merge on update. Simple three-panel diagram.
11. **Slide 11** -- "No Lock-In" summary. Three bullet points max: (1) Standard file formats, (2) SHA256 change tracking, (3) `gdev teardown` for clean exit.

**Transition to Module 4**:

> "So that's how gdev generates your environment. Now let me show you what makes it different from every other environment generator: the security model. This is the part that keeps your CISO happy -- and the part that blocks real attacks."

---

### Module 4: Defense in Depth (Minutes 30:00 - 40:00)

**Title**: *Six Layers, Each One Independent*

**Duration**: 10 minutes

**Opening Hook** (Attack narrative -- the valley counter-strategy. This module is deliberately placed at minute 30, the attention valley, and uses the highest-contrast storytelling technique to reset energy):

> "It's a Tuesday morning. A developer on your team opens their terminal and runs `pip install` on a package a colleague recommended in Slack. The package was published 6 hours ago. It has a perfectly legitimate name -- similar to a popular library, one letter different. The install script executes arbitrary code during installation. Within seconds, credentials are exfiltrated. Tokens, SSH keys, environment variables -- gone. Your CI/CD secrets, your cloud credentials, your database access. And nobody notices. Not for hours. Maybe not for days."
>
> Pause. Let the silence sit for 3 full seconds.
>
> "Now let me show you what happens when that same developer is working in a gdev-managed environment."

**Core Content**:

| Time | Content | Technique |
|------|---------|-----------|
| 30:00 - 32:30 | Attack narrative Pass 1 (above). Tell the story slowly. Use present tense ("The package IS published 6 hours ago"). Make it visceral. This is not a CVE number -- it's a Tuesday morning. Reference the real attacks briefly: "This is exactly what happened with ua-parser-js. 7 million weekly downloads. The attacker hijacked the maintainer's npm account. Detected in 4 hours, but the downstream damage was unknown." | Narrative transportation -- two-pass attack pattern |
| 32:30 - 36:00 | **LIVE DEMO: Attack narrative Pass 2**. "Same Tuesday. Same developer. Same terminal. But they're in a gdev environment." Run the safe test fixtures. Show age-gating blocking a fresh package: "This package is 6 hours old. Age-gating blocks it. gdev's threshold is 72 hours for warnings, 24 hours for blocks." Show install script blocking: "Even if the developer overrides the age gate, install script blocking prevents arbitrary code execution." Show lock file enforcement: "Even if they somehow bypass both, the lock file catches the unapproved dependency." Show vulnerability scanning: "And even if the package has a known CVE, scanning catches it." "Four independent layers. Each one would have stopped this attack alone. Together, they create defense in depth." | Live demo -- making invisible defenses visible |
| 36:00 - 38:00 | The 6-layer model. Show the defense-in-depth visual. Walk through all 6 layers briefly, using business language for each. Use the castle/supply chain analogy for leadership audience members: age-gating = "new supplier quarantine"; install script blocking = "supplier code of conduct"; lock file enforcement = "approved vendor list"; vulnerability scanning = "quality inspection"; PreToolUse hooks = "AI safety rails"; hardened Nix evaluation = "clean room assembly." "Each layer works independently. If any single layer fails, the others still protect you. This is genuine defense-in-depth, not security theater." | Progressive diagram build -- reveal one layer at a time |
| 38:00 - 39:00 | The AI guardrails beat. Brief but pointed: "48 deny rules for Claude Code. PreToolUse hooks that check package age and CVEs before Claude can install anything. This is the part most teams are missing entirely -- AI agents running with zero guardrails." | Connects to Module 1's hook |
| 39:00 - 40:00 | Checkpoint Q&A. "Security questions are best asked while the topic is fresh. Anything on the defense model?" | Engagement checkpoint |

**Key Talking Points**:

- "The test fixtures I just ran are the EICAR equivalents for supply chain security. They're safe -- they don't install anything malicious. But they prove each defense layer is active and working. You can run these in CI."
- "92% of malicious PyPI packages are less than 24 hours old. Age-gating alone -- one layer -- catches nearly all of them. The other five layers are backup."
- "I want to be honest about what gdev doesn't protect against. A sophisticated, targeted attack by a nation-state actor who is willing to wait 72 hours and avoid known CVE patterns could bypass age-gating and vulnerability scanning. That's why there are 6 layers, not 1. Defense in depth means no single layer is the whole story." (Honest limitation -- builds trust per ELM research)

**Slides Needed**:

12. **Slide 12** -- Section header: "Defense in Depth" with subtitle "Six independent layers, each provably working"
13. **Slide 13** -- The attack timeline. A horizontal flow: "Developer runs pip install" -> "Package is 6 hours old" -> "Install script executes" -> "Credentials exfiltrated" -> "Hours before detection." Simple, dramatic, single visual.
14. **Slide 14** -- The 6-layer defense diagram. Build animation: reveal one layer at a time as you discuss each one. Each layer has a technical label and a business analogy. Use a castle/walls visual or concentric rings -- whichever your slide tool handles better.
15. **Slide 15** -- The "each layer would have stopped it" visual. Show the attack flow from slide 13 again, but now with 4 red "BLOCKED" markers at different points. The message: redundancy, not single-point protection.
16. **Slide 16** -- AI guardrails summary. "48+ deny rules. PreToolUse hooks. Package age + CVE checks before every AI-suggested install." Short. Punchy.

**Transition to Module 5**:

> "So gdev generates your environment in 60 seconds and hardens it with 6 defense layers. The question you're probably asking now is: what does this look like at team scale? How does a whole organization adopt this?"

---

### Module 5: Team Scale and Adoption (Minutes 40:00 - 50:00)

**Title**: *From One Developer to the Whole Org*

**Duration**: 10 minutes

**Opening Hook** (Before/after transformation -- progressive wow):

> "Everything I've shown you so far is one developer, one project. Now let me show you what happens when you roll this out to a team."

**Core Content**:

| Time | Content | Technique |
|------|---------|-----------|
| 40:00 - 42:00 | Team configuration. "A team lead creates a `.gdev.yaml` file in the repo. It specifies the compliance level, required tools, team profile, and any custom overrides. When a new developer clones the repo and runs `gdev init --mode join`, they get the exact same environment as everyone else. No tribal knowledge. No 'ask Sarah for the settings.' Two minutes from `git clone` to productive." Show a brief `.gdev.yaml` example on a slide -- not a full file walkthrough, just the key fields. | Before/after transformation |
| 42:00 - 44:00 | **LIVE DEMO: Compliance evidence**. `gdev evidence --framework soc2`. Show the output: defense layers mapped to specific SOC2 control IDs with SHA256-hashed artifacts. "This is the report your auditor asks for. Currently, it takes your team weeks to produce this manually. gdev generates it from the actual state of your project in seconds. Every artifact is hashed. Every control mapping is based on what's actually configured, not what a policy document says should be configured." | Live demo -- the leadership peak moment |
| 44:00 - 46:00 | Posture scoring and drift detection. "Every project gets a 0-100 posture score with A-through-F letter grades. `gdev status` runs in under 100 milliseconds, entirely local. It checks 6 categories of drift: missing files, outdated configs, disabled tools, weakened permissions, stale dependencies, and CI pipeline gaps. When something degrades, you know immediately -- not at the next audit." Briefly mention team aggregation: "`gdev team-report` aggregates posture across all your projects. Trend tracking over 90 days. Auto-generated GitHub issues when a project degrades." | Making the invisible visible at org scale |
| 46:00 - 47:30 | The reversibility story. "I want to address the elephant in the room. Adopting a new tool is a risk. What if it doesn't work? What if it generates bad config? What if you want to stop using it?" Answer each directly: "`gdev enable semgrep` adds Semgrep to your project -- surgically, into all the right config files. `gdev disable semgrep` removes it just as cleanly. `gdev teardown` removes everything gdev generated. The configs are standard files -- if gdev disappeared tomorrow, your project still works." | Direct objection handling |
| 47:30 - 48:30 | The pilot path. "Here's what adoption actually looks like. Week 1: a champion picks one or two projects and runs `gdev init`. Week 2: they measure before and after -- onboarding time, posture score, developer satisfaction. Week 4: the champion's team adopts. Week 6: the champion presents results to leadership. Not us presenting -- YOUR champion, with YOUR metrics, on YOUR projects. Then you decide whether to expand." | Concrete next steps -- the bowling pin strategy |
| 48:30 - 50:00 | The consulting lifecycle (if audience-relevant; otherwise, expand the pilot path). "For consulting organizations: a profile per client. `gdev init --profile client-healthcare --yes`. The healthcare profile sets strict compliance, HIPAA-aligned controls. At engagement end, `gdev teardown --compliance` creates an evidence archive -- everything your client needs for their audit. Clean exit. Evidence preserved." If not relevant, use this time for a brief `gdev doctor` / `gdev repair` demo showing self-healing. | Persona-specific value |

**Key Talking Points**:

- "Join mode is the answer to the onboarding problem from the opening story. New developer: `git clone`, `gdev init --mode join`. Productive in under 2 minutes. No asking colleagues. No copying configs. No tribal knowledge."
- "The compliance evidence isn't aspirational -- it's generated from the actual state of your project. If a defense layer is disabled, the evidence report says so. It's honest, and that's what makes it trustworthy for auditors."
- "The pilot program is designed to prove value before you commit. 2-3 projects, 6 weeks, clear pass/fail criteria. If the onboarding time doesn't drop and the posture scores don't improve, you stop. No sunk cost."
- "$0 per month. MIT-licensed. The entire infrastructure stack -- Grype, Semgrep, OSV Scanner, Renovate -- runs on free tiers. The total cost of adoption is the time your champion spends running the pilot."

**Slides Needed**:

17. **Slide 17** -- Section header: "Team Scale" with subtitle "Configuration, compliance, and the adoption path"
18. **Slide 18** -- Join mode visual. Left side: old flow (clone -> ask colleague -> find old configs -> copy -> debug -> 2 hours). Right side: new flow (clone -> `gdev init --mode join` -> 2 minutes). Simple two-column comparison.
19. **Slide 19** -- .gdev.yaml example. Show 10-15 lines of a real team config file. Highlight: compliance level, required tools, team profile name. Not a full file.
20. **Slide 20** -- Compliance evidence screenshot. Show actual `gdev evidence` output with SOC2 control IDs mapped to defense layers. If possible, a real terminal screenshot.
21. **Slide 21** -- Posture score dashboard. Show 3-4 projects with letter grades and trend arrows. One project degraded (C, down arrow). Visual, not a table.
22. **Slide 22** -- Pilot program timeline. A horizontal timeline: Week 1 (champion selects projects) -> Week 2-4 (core validation) -> Week 4-6 (depth testing + measurement) -> Week 6 (champion presents results -> decision). Simple visual.
23. **Slide 23** -- "$0/month" slide. Large zero. Below it: "MIT-licensed. Static binary. Zero prerequisites. Free-tier infrastructure." This is the cost objection killer.

**Transition to Module 6**:

> "Let me bring this back to where we started."

---

### Module 6: Close and Call to Action (Minutes 50:00 - 55:00)

**Title**: *What Happens Next*

**Duration**: 5 minutes

**Opening Hook** (Full-circle narrative -- return to the opening story. The peak-end rule demands this ending be as strong as Module 2's aha moment):

> "Remember the engineer from the opening? Three hours to get productive. Zero security guardrails. No consistency across projects. No audit evidence."

Pause.

> "Here's what happens now."

**Core Content**:

| Time | Content | Technique |
|------|---------|-----------|
| 50:00 - 51:30 | The full-circle close. Return to the opening story. "That engineer joins today. They clone the repo. They run `gdev init --mode join`. In 2 minutes, they have a working environment with 6 defense layers, AI guardrails, pre-commit hooks, and CI workflows. They write their first commit before lunch." If time permits and energy is right, do a quick live demo of Join mode on a second terminal to make this concrete. Otherwise, let the verbal callback be the close. | Narrative closure -- full circle |
| 51:30 - 53:00 | The three vertebrae. "If you remember three things from this talk:" (hold up fingers) "ONE: one command, 60 seconds, from clone to a working, secure environment." "TWO: 6 independent defense layers, each provably working -- genuine defense in depth, not security theater." "THREE: fully reversible. Try it on one project. If you don't like it, `gdev teardown` removes everything. You lose nothing." These are the three concepts the audience will carry out of the room. Deliver them with confidence. Pause between each one. | Primacy/recency reinforcement -- the three things they'll remember |
| 53:00 - 54:00 | Single call to action. One thing. Not three things. ONE. Show the install command on screen: `curl -fsSL https://get.gdev.dev | sh`. "Try it tonight. Pick any project -- Go, TypeScript, Python, Rust, 27 ecosystems. Run `gdev init`. See the posture score. If it's not an improvement, `gdev teardown` and you've lost 60 seconds." Show a QR code linking to the leave-behind package. | Single CTA -- the conversion moment |
| 54:00 - 55:00 | Vision close. Brief, forward-looking. "Today, gdev handles 27 ecosystems with 6 defense layers. The roadmap includes multi-team policy federation, automated dependency upgrade flows, and expanded compliance framework support. This is the foundation -- and it's ready now." End with: "I'd love to take your questions." | Forward momentum -- the talk ends before Q&A begins |

**Key Talking Points**:

- "The cost of trying is 60 seconds and one command. The cost of not trying is the status quo -- manual setup, inconsistent security, no compliance evidence, and AI agents running unguarded."
- "I don't need you to adopt gdev today. I need you to try `gdev init` on one project and see the output. The generated files will speak for themselves."
- "Every piece of configuration gdev generates is a standard file you'd write yourself. The difference is: gdev writes all of them, correctly, consistently, in 60 seconds."

**Slides Needed**:

24. **Slide 24** -- The full-circle visual. Two timelines, labeled "Before gdev" and "With gdev." Before: Day 1 -> Day 3 (productive). With: Minute 0 -> Minute 2 (productive). Large, clean, no clutter.
25. **Slide 25** -- The three vertebrae. Three lines, large text: "1. One command. 60 seconds." / "2. Six defense layers. Each provably working." / "3. Fully reversible. Zero risk to try." This is the slide the audience photographs.
26. **Slide 26** -- Call to action. The install command in large monospace font: `curl -fsSL https://get.gdev.dev | sh`. Below it: a QR code linking to the quick-start guide. This slide stays up during Q&A.

---

### Q&A (Minutes 55:00 - 60:00)

**Structure**: Keep slide 26 (install command + QR code) visible throughout Q&A. Address any parked questions first. Use the PAUSE method for difficult questions (Pause 2-3 seconds, Clarify the question, Respond, Bridge back to core message). After 2-3 questions:

> "I'll take one more question, and then I'll be around afterward for individual conversations."

End Q&A on your terms. Do not let it trail off.

---

## 3. Live Demo Segments

### Demo 1: `gdev init` (Module 2, minutes 11:00 - 14:00)

**Exact commands**:
```
ls                          # Show existing project files, no gdev artifacts
gdev init                   # Or: gdev init --profile go-web --yes (for zero-prompt version)
ls                          # Show generated files
bat devenv.nix              # Or: cat devenv.nix | head -40 (show quality of generated Nix)
bat .claude/settings.json   # Flash deny rules and PreToolUse config
devenv shell                # Enter the environment
go version && node --version  # Prove tools are available
gdev status                 # Show posture score: A (92/100)
```

**What to highlight in the output**:
- Ecosystem detection lines ("Detected: Go 1.22, TypeScript 5.x, Docker, Terraform")
- File generation summary ("Generated 12 files across 6 directories")
- Posture score and letter grade

**Fallback if it fails**:
- If `gdev init` hangs or errors within 15 seconds: try once more. If still failing at 30 seconds: switch to pre-baked terminal tab with completed state. "Let me show you the result from the completed environment." Continue from `ls` showing generated files.
- If `devenv shell` is slow (Nix downloads): "In the interest of time, let me switch to an environment I prepared earlier." Switch to pre-baked tab.
- Nuclear fallback: play asciinema recording of the full sequence with live narration.

**Time budget**: 3 minutes for the demo itself. 2 minutes for walkthrough and posture score. Total: 5 minutes.

### Demo 2: Security Test Fixtures (Module 4, minutes 32:30 - 36:00)

**Exact commands**:
```
# Age-gating demo (using safe Verdaccio test fixture)
gdev test-fixture age-gate  # Or the equivalent command that triggers the age-gating layer
# Show the block message: "Package published 6 hours ago. Blocked by age-gating policy."

# Install script blocking demo
gdev test-fixture script-block  # Trigger install script blocking
# Show: "Install script execution blocked by policy."

# Vulnerability scanning demo
gdev test-fixture vuln-scan     # Trigger known-CVE manifest
# Show: "CVE-XXXX-XXXXX detected in dependency X. Blocked."
```

**What to highlight in the output**:
- The specific reason each layer blocks: age, script execution, known CVE
- That each layer operates independently
- The clear, actionable error messages (not cryptic failures)

**Fallback if it fails**:
- Pre-prepared screenshots of each test fixture output. "Let me show you what this looks like when it runs." Walk through the screenshots with the same narration.
- If only one fixture fails, skip it and demonstrate the others. "That fixture needs [specific condition]. Let me show you the other two, which demonstrate the same principle."

**Time budget**: 3.5 minutes total. About 1 minute per fixture demonstrated. Show 2-3 fixtures, not all 6 layers.

### Demo 3: Compliance Evidence (Module 5, minutes 42:00 - 44:00)

**Exact commands**:
```
gdev evidence --framework soc2
# Show the compliance report with control ID mappings and SHA256 artifact hashes
```

**What to highlight in the output**:
- SOC2 control IDs mapped to specific defense layers
- SHA256 hashes on each artifact (proves integrity)
- The fact that this report is generated from actual project state, not a template

**Fallback if it fails**:
- Pre-prepared screenshot of evidence output. "Here's the compliance report from this morning's run."

**Time budget**: 2 minutes. Quick and impactful. The output speaks for itself.

---

## 4. Slide Inventory

Target: 35-38 slides for 55 minutes. Approximately 1 slide per 1.5 minutes during slide-based sections. Demo sections have fewer slides (the terminal is the visual).

| # | Module | Slide Description | Type |
|---|--------|-------------------|------|
| 1 | M1 | Title slide: presentation title, presenter name (small), date | Title |
| 2 | M1 | The 15 files: directory tree showing manual setup work with time estimates | Visual diagram |
| 3 | M1 | Supply chain attack timeline: 4 attacks on a horizontal line with key numbers | Timeline visual |
| 4 | M1 | The cost math: developers x projects x time = hours/year and dollar figure | Large numbers |
| 5 | M2 | Section header: "One Command" with `gdev init` below | Transition |
| 6 | M2 | Before/after directory comparison (backup for failed demo) | Comparison |
| 7 | M3 | Section header: "Under the Hood" | Transition |
| 8 | M3 | Detection flow diagram: project files -> engine -> detected ecosystems | Architecture |
| 9 | M3 | 3-addon architecture: devenv + claudecode + devinit boxes with outputs | Architecture |
| 10 | M3 | SHA256 tracking: generate -> track -> edit -> three-way merge | Process diagram |
| 11 | M3 | "No Lock-In": three bullet points on reversibility | Key message |
| 12 | M4 | Section header: "Defense in Depth" | Transition |
| 13 | M4 | Attack timeline: pip install -> script executes -> credentials exfiltrated | Story visual |
| 14 | M4 | 6-layer defense diagram with build animation (reveal one layer at a time) | Architecture |
| 15 | M4 | "Each layer would have blocked it": attack flow with 4 BLOCKED markers | Contrast visual |
| 16 | M4 | AI guardrails: "48+ deny rules. PreToolUse hooks." | Key message |
| 17 | M5 | Section header: "Team Scale" | Transition |
| 18 | M5 | Join mode before/after: 2 hours vs 2 minutes | Comparison |
| 19 | M5 | .gdev.yaml example: 10-15 lines of team config | Code snippet |
| 20 | M5 | Compliance evidence: screenshot of `gdev evidence` output | Screenshot |
| 21 | M5 | Posture dashboard: 3-4 projects with grades and trends | Dashboard visual |
| 22 | M5 | Pilot timeline: 6-week visual timeline | Timeline |
| 23 | M5 | "$0/month" with supporting points | Key message |
| 24 | M6 | Full-circle: "Before" timeline (3 days) vs "With gdev" (2 minutes) | Comparison |
| 25 | M6 | Three vertebrae in large text | Key message (the photo slide) |
| 26 | M6 | CTA: install command + QR code (stays up during Q&A) | Call to action |
| 27 | Spare | "What gdev generates" checklist: every file type in a single visual | Reference |
| 28 | Spare | "Supported ecosystems" grid: 27 ecosystem logos/names | Reference |
| 29 | Spare | Compliance framework support: SOC2, HIPAA, OWASP ASVS | Reference |
| 30 | Spare | `gdev doctor` / `gdev repair` self-healing flow | Process diagram |
| 31 | Spare | The 3 compliance levels: baseline / enhanced / strict | Comparison |
| 32 | Spare | Team aggregation: multi-project posture summary | Dashboard visual |
| 33 | Spare | Consulting lifecycle: init with profile -> develop -> evidence -> teardown | Process flow |
| 34 | Spare | The adoption path: individual -> team -> org with timeline | Flow diagram |
| 35 | Spare | "How gdev compares" -- vs plain devenv.sh, vs mise, vs Docker | Comparison table |

**Total: 35 slides.** Slides 27-35 are spare slides at the end of the deck, available for Q&A deep-dives or audience-specific emphasis. They are never shown in the main flow unless you choose to pull one in.

**Slide design principles**:
- Maximum 6 words per bullet point. Most slides should have zero bullet points -- use visuals, diagrams, and large numbers instead.
- One concept per slide. If you need to explain the slide, it has too much on it.
- Dark background with high-contrast text for conference projectors. Test from the back of the room.
- No company logos on content slides (one logo on title slide and CTA slide only).
- No slide numbers visible to audience (they create "are we almost done?" anxiety).

---

## 5. Q&A Preparation Guide

### Developer Questions

| Question | Prepared Response |
|----------|-------------------|
| "I can set this up myself in 20 minutes." | "Absolutely. For one project. The question is: can you do it consistently for 10 or 50 projects across 27 ecosystems, with the same security baseline every time, and produce audit evidence? gdev does the thing you'd do yourself -- it just does it for every project, every time, correctly." |
| "Generated config is always garbage. How do I know the devenv.nix is good?" | "Let me show you." (Pull up slide 27 or switch to terminal and show a generated devenv.nix.) "This is idiomatic Nix. The packages are correct for the detected ecosystem. The formatters match the language. If the generated config doesn't match what you'd write yourself, file an issue -- that's a bug." |
| "This is too opinionated. What if I disagree with the defaults?" | "Three compliance levels -- baseline, enhanced, and strict. Per-ecosystem customization in .gdev.yaml. Personal overrides in .gdev.local.yaml that never get committed. And every tool is opt-in/opt-out: `gdev enable semgrep`, `gdev disable semgrep`. The opinions are defaults, not mandates." |
| "Another tool to maintain. What happens when it breaks?" | "`gdev doctor` diagnoses issues. `gdev repair` fixes them. `gdev update` keeps everything current. It's a single static binary with zero runtime dependencies -- self-updating with rollback. And if you decide to stop: `gdev teardown`. You keep the standard files." |
| "Does this work with [specific ecosystem]?" | Check if it's in the 27. If yes: "Yes, [ecosystem] is fully supported." If no: "Not yet -- gdev currently covers 27 ecosystems. Yours isn't one of them today, but the detection engine is extensible. The generated devenv.nix is standard Nix, so you can add custom packages manually." (Honest limitation builds trust.) |

### Leadership Questions

| Question | Prepared Response |
|----------|-------------------|
| "What's the ROI?" | "The direct productivity ROI: 50 developers saving 1 hour per project setup across 20 projects is 1,000 hours per year. At $75/hour, that's $75,000. The security ROI is harder to quantify but larger: IBM's 2025 data shows the average breach costs $4.44 million globally, $10.22 million in the US. gdev's 6 defense layers reduce your supply chain attack surface measurably -- age-gating alone catches 92% of PyPI malware." |
| "How long to deploy org-wide?" | "The pilot is 6 weeks: 2-3 projects, 8-12 participants. Decision gate at week 6. If it passes, the first expansion wave adds 3-5 more teams over 4 weeks. Full org adoption with CI enforcement typically happens around week 16-20. But you see value from Day 1 on the pilot projects." |
| "What if this tool gets abandoned?" | "MIT-licensed, pure Go, static binary. All generated files are standard formats: devenv.nix, Claude Code settings.json, pre-commit YAML, GitHub Actions workflows. If gdev disappears, you maintain those files yourself -- the same files you'd have written without gdev. There's no proprietary format, no cloud dependency, no API key." |
| "We already have security tools." | "gdev doesn't replace your security tools. It curates and configures them. If you use Grype for vulnerability scanning, gdev generates the correct Grype configuration for each project. If you use Semgrep, `gdev enable semgrep` adds it with the right rulesets. gdev is the orchestration layer that makes your existing tools work consistently across every project." |
| "What does this cost?" | "$0 per month. MIT-licensed. Static binary. The entire security infrastructure stack runs on free tiers: OSV Scanner, Grype, Semgrep Community, Renovate. The only cost is the time your team spends on the pilot -- which is offset by the time they save from Day 1." |

### Security Questions

| Question | Prepared Response |
|----------|-------------------|
| "How do I know the defenses actually work?" | "Every defense layer has an EICAR-equivalent test fixture. Age-gating: publish a fresh package to a local Verdaccio, attempt to install, watch it block. Install script blocking: run the @lavamoat canary package, watch it block. Vulnerability scanning: install a known-CVE manifest, watch it detect. These are runnable in CI. I'll share the test fixture documentation afterward." |
| "What about false positives? What if age-gating blocks a legitimate new package?" | "The thresholds are configurable per-project. The default is: warn at 72 hours, block at 24 hours. Your team can adjust these in .gdev.yaml. Overrides are logged, so you have an audit trail of when and why a developer bypassed a gate. The compliance evidence report includes these overrides." |
| "What about Trivy? We use Trivy." | "Trivy was compromised in March 2026 -- the maintainer org was acquired and the supply chain integrity is under question. gdev explicitly replaced Trivy with Grype for vulnerability scanning and Checkov for infrastructure scanning. Both are actively maintained and have clean supply chains. The decision and rationale are documented." |
| "What's NOT protected?" | "gdev protects the development environment setup and package installation paths. It does not protect runtime behavior, network access, or application-layer vulnerabilities. It does not replace a WAF, a SIEM, or a penetration test. It secures the supply chain -- the path between 'developer adds a dependency' and 'dependency runs in the environment.' That specific path has 6 layers. Everything else is out of scope." |
| "Can you prove this meets SOC2 / HIPAA requirements?" | "`gdev evidence --framework soc2` maps each defense layer to specific SOC2 control IDs. `gdev evidence --framework hipaa` does the same for HIPAA Technical Safeguards. Each mapping references real artifacts with SHA256 hashes. This accelerates audit readiness but does not replace your auditor's judgment -- it provides the evidence they need to make their assessment." |

### Hostile Questions and Bridge Techniques

| Hostile Question | Bridge Technique | Response |
|------------------|-----------------|----------|
| "This is just another tool that will be abandoned in a year." | Acknowledge and Reframe | "That's a legitimate concern, and it's why everything gdev generates is standard files. devenv.nix, settings.json, YAML configs -- these are the industry standard formats. If gdev disappears, your projects still work. The lock-in risk is genuinely zero. But let me flip the question: what's the cost of continuing to configure these files manually for another year?" |
| "Our team doesn't need training wheels." | Acknowledge and Pivot | "I agree -- your team is capable of writing great Nix configs. gdev isn't training wheels. It's automation. The same way nobody hand-writes CI pipeline YAML from scratch when a generator exists. The question isn't whether your team CAN do it -- it's whether their time is best spent doing it 20 times across 20 projects." |
| "You're just generating boilerplate. Any script could do this." | The Evidence Response | "A fair challenge. Three things a script doesn't do: first, ecosystem detection with confidence scoring across 27 ecosystems. Second, SHA256 tracking with three-way merge so updates don't destroy your changes. Third, 6-layer defense-in-depth with provable test fixtures. If your script does all three, you don't need gdev." |
| "How do we know YOU haven't been compromised?" | Honest Answer | "The binary is built with GoReleaser from a public repo. It's MIT-licensed -- you can read every line. The SHA-pinned CI actions are auditable. The generated configs are standard files you can inspect. If you need to build from source, you can. That said, this is a fair concern for any tool in your supply chain, and I respect the question." |

**Parking lot strategy**: Keep a visible notepad or slide for questions that deserve thorough answers but would derail the current module. Say: "That's an important question. I want to give it the time it deserves. Let me park it here and we'll come back to it in Q&A." Address parked questions first in the Q&A section.

---

## 6. Audience-Variant Adjustments

### Internal Engineering All-Hands

**What to adjust**:
- **Module 1**: Compress the problem framing. They already know the pain. Skip the supply chain attack case studies and spend 30 seconds: "You all know the setup problem. Here's the security problem you might not know about." Go to the 92% statistic directly.
- **Module 2**: Expand. This is where they live. Spend extra time on the generated files. Open more config files. Show the devenv.nix quality in detail. This audience will judge gdev by the output quality.
- **Module 3**: Expand. Platform engineers want the architecture. Show the SHA256 tracking in detail. Explain the three-way merge. This builds trust.
- **Module 4**: Keep as-is. The attack narrative works for everyone.
- **Module 5**: Emphasize Join mode heavily. "This is what changes for you on Monday." Emphasize reversibility: "Try it on one project. If you hate it, `gdev teardown`."
- **Module 6**: CTA is "try it on your project this week" not "approve a pilot."
- **Tone**: Casual, technical, collegial. No selling. "Here's what I built and why I think it helps."

### Conference Talk

**What to adjust**:
- **Module 1**: Expand the problem framing. Conference audiences are diverse -- not everyone shares the same pain. Build the context carefully. The supply chain attack stories get more time because they're universally relevant.
- **Module 2**: Keep the live demo but prepare for hostile WiFi. Pre-populate all caches. Have asciinema ready. The conference demo must be bulletproof.
- **Module 3**: Architecture gets full treatment. Conference attendees chose this session for depth. Emphasize the design decisions: why 3 addons? Why SHA256 tracking? Why section markers for CLAUDE.md?
- **Module 4**: This is the star module for a conference. The attack narrative is the most engaging content. Expand the test fixture demo. Show 3-4 defense layers, not 2.
- **Module 5**: Compress the organizational content. Conference audiences care about technical merit, not pilot programs. Replace the pilot timeline with a community contribution call.
- **Module 6**: CTA is "try it on a side project tonight" and "star the repo / contribute."
- **Tone**: Educational, authoritative, generous with knowledge. Teach them something they can use regardless of whether they adopt gdev.

### Leadership Decision Meeting

**What to adjust**:
- **Module 1**: Lead with the cost math. "We're spending $X per year on manual environment setup. Here's the security risk on top of that." The supply chain case studies get full treatment -- CTOs know SolarWinds.
- **Module 2**: The demo is shorter and narrated in business terms. Don't open config files. Focus on: "One command. 60 seconds. Every project at the same security baseline." The posture score is the key visual.
- **Module 3**: Compress to 5 minutes. Leadership doesn't need the architecture. Hit two points: "Standard files, no lock-in" and "SHA256 tracking means safe updates." Skip the 3-addon diagram.
- **Module 4**: The attack narrative is powerful for leadership. Use business analogies for every layer. "New supplier quarantine. Approved vendor list. Quality inspection." Skip the live test fixtures; use the visual diagram instead.
- **Module 5**: This is the star module for leadership. Expand compliance evidence, team dashboards, and the pilot program. Show the ROI calculation. Present the pilot as a low-risk decision: "6 weeks, 2-3 projects, clear pass/fail criteria."
- **Module 6**: CTA is "approve the pilot" with a specific proposal: "I'd like to run this on [Project A] and [Project B] for 6 weeks with [Champion Name] leading. We'll measure onboarding time and posture scores before and after. Decision gate at week 6."
- **Tone**: Concise, metric-driven, confident. No selling -- presenting a business case with evidence.

---

## 7. Leave-Behind Materials Inventory

### Executive One-Pager

**What it covers**: A single-page decision-support document (not a brochure) with 6 sections:
1. The Problem -- lead with pain: developer hours wasted, inconsistent security, no compliance evidence
2. The Solution -- one sentence: "gdev generates security-hardened development environments in one command"
3. How It Works -- three steps: install (curl | sh), run `gdev init`, enter `devenv shell`
4. Outcomes -- four metrics: 60 seconds vs 90 minutes, 6 defense layers, 0-100 posture scores, $0/month
5. Proof -- pilot results, before/after data (leave blank template if pre-traction)
6. Next Step -- "Run the 6-week pilot on 2-3 projects"

**Design**: 60% text / 40% white space. Skimmable in 30 seconds. Logo once in corner. Large bold headlines.

### Quick-Start Guide

**What it covers**: The document a developer uses to try gdev within 5 minutes of the talk ending:
1. One-line install: `curl -fsSL https://get.gdev.dev | sh`
2. Three commands: `gdev init`, `gdev status`, `devenv shell`
3. "What just happened" -- brief explanation of generated files
4. "What's next" -- `gdev enable`, `gdev doctor`, `gdev update`
5. Link to full documentation

**Critical requirement**: First value in under 5 minutes. If the guide requires reading more than one page before running a command, it's too long.

### Architecture Reference

**What diagrams it includes**:
- Detection engine flow (same as slide 8, with annotations)
- 3-addon architecture (same as slide 9, with technical details per addon)
- 6-layer defense model with technical specifications per layer
- File generation pipeline with SHA256 tracking and three-way merge detail
- Extension points: .gdev.yaml, .gdev.local.yaml, custom profiles

**Purpose**: Survives technical scrutiny from Staff+ engineers evaluating whether to trust generated artifacts.

### ROI Calculator Template

**Inputs**: Number of developers, number of active projects, average onboarding time (minutes), average security configuration time (minutes), hourly developer cost, number of audits per year, average audit prep time (hours)

**Outputs**: Annual hours saved on setup, annual hours saved on audit prep, dollar value of time savings, risk reduction estimate (based on defense layer coverage), break-even timeline (typically Day 1, since cost is $0)

**Format**: Editable spreadsheet. Pre-filled with industry benchmarks.

### Champion Slide Deck Template

**What it covers**: A 15-20 slide subset of the full presentation, optimized for a team lead to present to their leadership:
1. The problem (3 slides -- cost and risk)
2. The demo results (3 slides -- before/after from their pilot)
3. The security model (2 slides -- 6-layer visual + evidence)
4. The pilot proposal (3 slides -- timeline, metrics, decision criteria)
5. Blank "Our Results" slides for the champion to fill with their data
6. Speaker notes with talking points for each slide
7. The demo script so the champion can reproduce the live demo

---

## 8. Emotional Arc Map

```
Energy/Engagement
 High ||  *           *                *                 *      **
      ||  |\         /|               /|                /|     /  |
      ||  | \       / |              / |               / |    /   |
      ||  |  \     /  |             /  |              /  |   /    |
 Med  ||  |   \   /   |            /   |             /   |  /     |
      ||  |    \ /    |    *      /    |            /    | /      |
      ||  |     X     |   /|    /     |           /     |/       |
 Low  ||  |    / \    |  / |  /      |    *     /      |        |
      ||  |   /   \   | /  |/       |   /|   /       |        |
      ||__|__/     \__|/   |________|__/ |__/________|________|
      0    5   10  15  20  25   30  35  40  45  50  55  60 min

Key:                                        Designed peaks:
      M1        M2        M3        M4        M5        M6
   Problem   Live Demo  Arch.   Security   Team     Close
```

**Minute-by-minute emotional map**:

- **0:00 - 1:30** HIGH: Opening story creates immediate engagement through narrative transportation. The audience is with you because the story is relatable.
- **1:30 - 2:30** HIGH: Audience participation (hand raise) validates shared experience. Energy stays high because they're physically involved.
- **2:30 - 4:30** MEDIUM-HIGH: The 15-file visual creates mild anxiety. "That's a lot of work."
- **4:30 - 6:30** MEDIUM: Supply chain attack facts. Not fear -- factual gravity. Energy dips slightly as content becomes information-heavy.
- **6:30 - 8:30** MEDIUM: Cost math. Necessary but not exciting. Keep it brief.
- **8:30 - 10:00** RISING: Transition hook creates curiosity gap. "What if one command..."
- **10:00 - 14:00** PEAK 1: Live demo of `gdev init`. This is the first major peak. The audience is watching real-time transformation. Terminal output scrolling is inherently engaging. The "60 seconds" moment is the aha.
- **14:00 - 17:30** MEDIUM-HIGH: File walkthrough and posture score. Reinforcement. Still interesting but explanatory.
- **17:30 - 20:00** MEDIUM: Reversibility and checkpoint Q&A. Necessary but lower energy.
- **20:00 - 25:00** DECLINING: Architecture section. This is the natural post-demo energy dip. The split-level announcement manages expectations. Platform engineers stay engaged; others may drift.
- **25:00 - 28:00** LOW POINT: SHA256 tracking and three-way merge. The technical valley. Keep it concise. The most at-risk moment for losing non-technical attendees.
- **28:00 - 30:00** RISING: Transition to security. "This is the part that keeps your CISO happy." Voice change, modality change (back to storytelling).
- **30:00 - 32:30** PEAK 2: Attack narrative Pass 1. Storytelling resets attention hard. Present tense, visceral detail, three-second silence. This is the valley counter-strategy -- deliberately placed at the energy low point.
- **32:30 - 36:00** HIGH: Live security demo. Test fixtures triggering. Each block is a small victory. The audience is watching defenses work in real time.
- **36:00 - 40:00** MEDIUM-HIGH: 6-layer model and AI guardrails. Explanatory but anchored by the demo they just saw.
- **40:00 - 44:00** MEDIUM-HIGH: Team configuration and compliance evidence demo. Steady energy. New content keeps attention.
- **44:00 - 47:30** MEDIUM: Posture scoring and reversibility. Informational. Necessary for the adoption conversation.
- **47:30 - 50:00** MEDIUM: Pilot path and consulting lifecycle. Forward-looking. Energy slightly rising as the audience anticipates the close.
- **50:00 - 51:30** RISING: Full-circle callback to the opening story. Narrative closure creates emotional engagement.
- **51:30 - 53:00** PEAK 3: The three vertebrae. Delivered with conviction and pauses. This is the recency-weighted memory -- the last substantive content.
- **53:00 - 55:00** HIGH: CTA and vision close. Forward momentum. QR code on screen. The audience knows what to do next.
- **55:00 - 60:00** DECLINING: Q&A. Natural energy decline but acceptable -- the designed ending already happened.

**Key design decisions in this arc**:
- Peak 1 (demo) is at minute 12 -- early enough to sustain attention through the architecture valley
- Peak 2 (attack narrative) is at minute 31 -- deliberately placed at the energy low point to reset attention
- Peak 3 (three vertebrae) is at minute 52 -- the recency-weighted memory per peak-end rule
- No module exceeds 10 minutes without an attention reset (per Medina's research)
- The architecture section (minutes 20-28) is the planned valley -- the split-level announcement manages expectations so it doesn't feel like a failure

---

## 9. Pre-Talk Checklist

### One Week Before

- [ ] Script every demo command and narration beat (use this document)
- [ ] Record asciinema fallback of the complete demo sequence (all 3 demos)
- [ ] Prepare `prepare-for-demo.sh` script that resets the demo environment to a known state
- [ ] Pre-populate Nix store and devenv caches so `devenv shell` activates without network downloads
- [ ] Build the slide deck from the Slide Inventory (Section 4)
- [ ] Prepare leave-behind materials (one-pager, quick-start guide) and generate QR code
- [ ] Research the specific audience: what ecosystems do they use? What compliance requirements? What security incidents have they experienced?
- [ ] Identify which audience variant (Section 6) applies and mark module adjustments

### Three Days Before

- [ ] Full dry run with a colleague playing the skeptic role
- [ ] Time the full run end-to-end. If over 55 minutes, cut content (never rush)
- [ ] Practice the three module transitions out loud -- they should feel natural, not scripted
- [ ] Practice the attack narrative in Module 4 -- it must be told, not read
- [ ] Verify all three demo fallbacks work (pre-baked terminal, asciinema, screenshots)
- [ ] Review Q&A preparation guide -- practice the hostile question responses

### One Day Before

- [ ] Run the full demo on the exact machine you'll present from
- [ ] Test font size, color scheme, and terminal width from the expected viewing distance
- [ ] Test screen sharing / projector connection if applicable
- [ ] Verify QR code works and links to the correct page
- [ ] Prepare backup terminal tab with completed gdev state (post-init environment)
- [ ] Prepare second terminal for Join mode demo in Module 6 (if using live close)
- [ ] Charge laptop fully; have power adapter accessible

### 30 Minutes Before

- [ ] Run `prepare-for-demo.sh` to reset the demo environment
- [ ] Run the full demo sequence once end-to-end (silent, fast -- just verify it works)
- [ ] Open all necessary terminal tabs/windows in the correct state
- [ ] Close all unnecessary applications; disable notifications (Slack, email, calendar)
- [ ] Set terminal font to 28pt+ and verify contrast
- [ ] Open the asciinema fallback recording, cued and ready to play
- [ ] Open the slide deck in presenter mode with notes visible on your screen
- [ ] Open this document on your laptop screen for reference during transitions

### During the Presentation

- [ ] Start with the story, not with "about me" (Module 1 hook)
- [ ] Narrate BEFORE typing each demo command
- [ ] Pause 3 seconds after terminal output appears before explaining
- [ ] If a demo breaks: try once (15s), acknowledge (15s), switch to fallback (15s)
- [ ] Watch for disengagement signals at minutes 22-28 (architecture valley) -- if losing them, compress and move to Module 4 early
- [ ] Deliver the three vertebrae with pauses between each one (Module 6)
- [ ] End with the CTA slide visible, not with "any questions?"
- [ ] Keep the install command + QR code visible throughout Q&A
