# 15-Minute Live Demo Structure: Best Practices for Developer Tool Presentations

> **Task**: P2-T5 — 15-minute demo structure research
> **Status**: Complete
> **Sources**: 12 new documents saved to `docs/`, plus 8 pre-existing sources from P2-T1/T2/T3

---

## Executive Summary

A 15-minute demo is the hardest format to execute well. It is long enough to require narrative structure but too short for recovery from derailment. The research converges on a clear set of principles: lead with a real pain point (not a feature list), reach the "aha moment" by minute 5-6, show no more than 3-4 capabilities through a single narrative thread, and end with the most impressive beat — not Q&A. The hybrid approach (pre-baked boring parts, live-typed wow moments, pre-recorded fallback ready) is the consensus best practice across all sources. For a dual audience of developers and leadership, the key structural insight is to run the demo through a developer lens (real terminal, real commands) but narrate in business language (time saved, risk reduced, compliance generated).

---

## 1. Narrative Arc: The Demo Story Structure

### 1.1 The Five-Beat Framework (Applied to 15 Minutes)

The universal five-beat structure identified in ecosystem analysis maps directly to a 15-minute format. The time allocations below synthesize findings from storytelling research, daily.dev demo guidance, and the Developer Advocacy Handbook.

| Beat | Time | What Happens | Cognitive Purpose |
|------|------|-------------|-------------------|
| **1. Name the Pain** | 0:00-2:00 | Specific, relatable problem. Not abstract ("security is hard") but concrete ("Last month, a new dev spent 3 hours hand-configuring devenv.nix, missed the pre-commit hook setup entirely, and pushed a commit with a vulnerable dependency") | Establishes relevance; primes the audience to care about the solution |
| **2. Show the Old Way** | 2:00-3:30 | Brief, visceral demonstration of the status quo. Show the 15 files you would need to create manually, or a timer counting the minutes of manual setup. "Let that silence hang" before the reveal. | Creates contrast baseline; without this, the solution has no perceived value |
| **3. The Shift (Live Demo)** | 3:30-10:00 | One clear transformation via `gdev init`. This is the core. Show it working in real time. Narrate what is happening. Reveal generated files. | The "aha moment" — cognitive/emotional realization of value. Must land by minute 5-6 |
| **4. Quantify Impact** | 10:00-12:00 | Specific metrics tied to timelines. "60 seconds vs 90 minutes. 6 defense layers, each provably working. SOC2 evidence in one command." | Converts emotional reaction into rational justification. Critical for leadership audience |
| **5. Close with One Action** | 12:00-13:30 | Single call to action. "Try `gdev init` on your project tonight." End with the most impressive capability — compliance evidence or posture scoring — not with "any questions?" | Peak-end rule: the last impression is disproportionately remembered. Make it count |
| **Buffer/Q&A** | 13:30-15:00 | Brief Q&A or graceful close | Allows for timing variance; prevents running over |

### 1.2 The Critical First 60 Seconds (The Hook)

Research from freeCodeCamp's conference guide, the Developer Advocacy Handbook, and demo psychology all converge: **hook the audience immediately**. The first 60 seconds determine whether the audience gives you their full attention or checks their email.

**What works for a hook:**
- A specific, personal story: "Last Tuesday, our newest engineer joined. Here's what happened..."
- A provocative number: "We counted: 15 files, 47 configuration decisions, 90 minutes. That's what every new project costs before you write a line of code."
- A visible timer: Start a stopwatch on screen. "I'm going to set up a complete, security-hardened dev environment before this timer hits 60 seconds."

**What fails as a hook:**
- "Hi, I'm X from Y, and today I'll be talking about..." (the most forgettable opening possible)
- Architecture diagrams (premature; explain HOW after showing WHAT)
- Feature lists ("gdev does devenv generation, Claude Code configuration, pre-commit hooks, CI workflows..." — the audience forgets all of them)
- Company logos and personal biography (cut these entirely per Developer Advocacy Handbook)

### 1.3 The Aha Moment Placement

The persuasion research established that the aha moment is the cognitive/emotional realization that the product will be valuable — distinct from activation (first hands-on experience). Research from PLG literature suggests it should come fast: developers who achieve first value within 10 minutes are 3-4x more likely to convert.

**For a 15-minute demo, the aha moment should land between minutes 4-6** (roughly 30-40% through). This is when `gdev init` completes and the audience sees the generated files for the first time. Everything before this is cost (establishing pain); everything after is reinforcement and deepening.

The aha moment candidates by audience segment:
- **Developers**: `gdev init` completing → opening the generated devenv.nix and seeing it is correct
- **Leadership**: `gdev status` showing a posture score jumping from nothing to A (92/100)
- **Security**: Running a test fixture and watching a defense layer catch it

For a mixed audience, **show the developer aha first** (the generation), then **layer the leadership aha** (the posture score) as reinforcement. This respects the bottom-up adoption model: leadership sees the individual developer value before they see the organizational value.

### 1.4 Ending Strong (Peak-End Rule)

The peak-end rule (Kahneman & Fredrickson) establishes that people judge experiences by two moments: the most intense moment (peak) and the final moment (end). Duration and average quality barely matter.

**Implications for the demo ending:**
- Never trail off into "so... any questions?" as the last beat
- Never end with configuration details or edge case discussion
- The final live command should be the most impressive capability the audience has not yet seen

**Strong ending candidates for gdev:**
1. `gdev evidence --framework soc2` producing a compliance report (leadership audiences)
2. `gdev teardown --compliance` showing clean exit with audit trail (consulting audiences)
3. Running the security test fixtures and watching defenses trigger (security audiences)
4. `gdev init --mode join` on a second terminal showing 2-minute onboarding (developer audiences)

**Structure the ending as a deliberate beat:** "Now for the part that made our CTO's eyes light up..." + final command + pause for impact + single CTA.

---

## 2. Live Demo Execution

### 2.1 Pre-Recorded vs Fully Live vs Hybrid

Sources unanimously favor the hybrid approach. The Developer Advocacy Handbook goes furthest in recommending pre-recorded screencasts with live commentary, while the daily.dev demo guide and live coding literature favor live execution with fallbacks.

| Approach | Strengths | Weaknesses | When to Use |
|----------|-----------|------------|-------------|
| **Fully live** | Maximum authenticity; proves the tool actually works; audience can see real behavior | High risk of failure; cognitive load of narrating + typing; timing unpredictable | Small team demos, internal presentations, when trust/authenticity is paramount |
| **Fully pre-recorded** | Zero failure risk; perfect pacing; can include post-production polish | Loses authenticity; audience knows it is staged; no adaptation to questions; "too polished" triggers developer skepticism | Website embed, async sharing, backup fallback |
| **Hybrid (consensus best practice)** | Authenticity where it matters; reliability where it does not; recoverable | Requires preparation of both live and recorded paths; transitions need practice | Conference talks, leadership demos, any high-stakes presentation |

**The hybrid formula for gdev:**
1. **Pre-bake** the project directory (have a realistic project already cloned), Nix store pre-populated (so `devenv shell` does not require network downloads during demo)
2. **Live-type** the `gdev init` command and show real output scrolling (this is the wow moment — it must be live)
3. **Pre-bake** a completed environment to switch to if the live init fails (have a second terminal tab with a completed state ready)
4. **Live-open** the generated files (devenv.nix, settings.json) to show quality
5. **Pre-record** an asciinema fallback of the full sequence, ready to play if catastrophic failure occurs

### 2.2 Terminal Presentation Setup

Terminal visibility is the most commonly overlooked preparation item. Sources converge on specific technical requirements.

**Font size:**
- Minimum 24pt for body text in any presentation setting
- 28-32pt recommended for conference venues (gilliard blog, font research)
- On dark backgrounds, add 15-20% to compensate for light dispersion through projection
- Test from the back of the room before presenting

**Color scheme:**
- **Light-on-dark** (the developer default) works well in dim conference rooms but poorly with weak projectors (<2500 lumens)
- **Dark-on-light** (black text on white background) has superior projector visibility and is recommended by gilliard as the safer default
- **Compromise**: Use a high-contrast dark theme (not pure black — deep gray/navy reduces eye strain) with bright syntax highlighting colors
- 78% of developers use dark mode on their machines, so dark-on-light feels "wrong" but reads better at distance

**Terminal prompt:**
- Simplify heavily: `export PS1=$'gdev-demo\n> '` — remove path, git info, timestamps
- Keep visual action in the upper 2/3 of the screen (bottom gets blocked by audience heads in sloped seating)
- Watch for line wrapping: use backslashes to break long commands or pre-configure terminal width

**Window management:**
- Share only the terminal window, not the full desktop
- Use separate workspaces or virtual desktops for slides vs terminal
- Have a `prepare-for-demo.sh` script that resets the environment to a known state
- Keep a second terminal hidden (with completed state) as a hot backup

### 2.3 Narrating While Typing: The Dual-Task Challenge

The cognitive load of simultaneous narration, typing, audience monitoring, and troubleshooting is the primary cause of human error under pressure (srinathmohan article). Multiple sources address this.

**Strategies that work:**
- **Pre-write commands and paste them**: Use clipboard history or a cheat sheet visible on your laptop screen (not the projected display). Type the first few characters live for authenticity, then paste the rest
- **Use `ctrl-r` with tagged history**: Pre-populate shell history with comments (e.g., `# DEMO: init project`) so `ctrl-r demo` retrieves exact commands
- **Use aliases**: `alias gi='gdev init'`, `alias gs='gdev status'` — fewer keystrokes, fewer typos
- **Narrate BEFORE typing**: "Now I'm going to run gdev init, which will detect the project type and generate all the configuration..." THEN type. This gives the audience cognitive scaffolding before the visual input arrives. Narrating simultaneously splits attention.
- **Pause after output appears**: Let the audience read the output before explaining it. The 3-second pause feels eternal to you; it feels necessary to them
- **Use the presenter pair model**: One person narrates and manages slides; the other types and operates the terminal. This eliminates the dual-task problem entirely (measuredco guide, Luke Lowrey tips)

**Strategies that fail:**
- Typing silently and explaining afterward (audience has already lost the thread)
- Reading terminal output aloud verbatim (boring and adds nothing)
- Alt-tabbing between multiple windows (disorienting for audience)
- Using IDE autocomplete or keyboard shortcuts that the audience cannot follow (ycmjason tips)

### 2.4 Dealing with Terminal Output Speed

CLI tools produce output at computer speed, not human reading speed. This creates a specific pacing challenge.

**When output scrolls too fast:**
- Warn the audience: "You'll see a lot of output scroll by — that's gdev detecting your project and generating configs. The important part is what happens at the end."
- Use `--verbose` mode selectively (only for the parts you want them to see)
- Pipe through `less` or `head` to show specific sections after the command completes
- Have a slide ready that shows the key output lines in readable format (screenshot backup)

**When output requires waiting (boring pauses):**
- If `gdev init` takes more than 5 seconds, fill the dead air with narration: "While this runs, let me explain what's happening under the hood..."
- Pre-populate Nix store / devenv caches so shell activation is near-instant
- Kelsey Hightower's technique: acknowledge the wait with humor ("This is what we do as speakers — we wave our hands and say 'did you see that?' knowing you didn't see anything")
- If wait exceeds ~45 seconds, switch to the pre-baked environment: "In the interest of time, let me show you the completed result" (gilliard tip: 45 seconds is "just long enough for the audience to lose focus")

### 2.5 The Clean Room Setup

Starting from a known, reproducible state is critical for reliable demos.

**The `prepare-for-demo.sh` pattern:**
```
#!/bin/bash
# Reset demo environment to known state
rm -rf /tmp/gdev-demo
git clone --depth 1 <repo> /tmp/gdev-demo
cd /tmp/gdev-demo
# Remove any existing gdev artifacts
rm -f devenv.nix devenv.yaml .envrc .claude/settings.json
# Pre-warm Nix store (so devenv shell is fast)
# ... (done in advance, not during script)
```

**Before every demo:**
- Run the reset script 10 minutes before (not an hour — things can change)
- Verify the full demo flow works end-to-end
- Check internet connectivity (or confirm offline mode works)
- Close all unnecessary applications, notifications, Slack, email
- Set terminal font size and color scheme
- Open backup tabs/windows in the correct state

---

## 3. Error Recovery: When the Demo Breaks

### 3.1 The Failure Taxonomy

From srinathmohan's analysis and cross-referencing with other sources, demo failures cluster into four categories:

| Failure Type | Likelihood | Recovery Time | Strategy |
|-------------|-----------|---------------|----------|
| **Environmental** (WiFi, projector, audio) | High at conferences | Variable | Backup connectivity (mobile hotspot); offline demo mode; backup laptop |
| **Tooling** (command fails, unexpected error) | Medium | 15-45 seconds | Acknowledge, try once, switch to backup if not fixed in 45 seconds |
| **Human** (typo, wrong command, forgot step) | High under pressure | 5-15 seconds | Pre-populated history, aliases, cheat sheet; humor and acknowledge |
| **Catastrophic** (machine crash, demo completely broken) | Low | Unrecoverable live | Switch to pre-recorded asciinema or screenshot walkthrough |

### 3.2 The 45-Second Rule

Multiple sources converge on this principle: **if the error is not fixed within 45 seconds, switch to your fallback**. The audience's attention window for troubleshooting is extremely short. After 45 seconds, you have lost them and the demo has become the failure, not the tool.

**The fallback cascade:**
1. **First attempt** (0-15 seconds): Try the command again or fix the obvious typo
2. **Acknowledge and pivot** (15-30 seconds): "Looks like [specific thing] isn't cooperating. Let me show you from the prepared environment."
3. **Switch to backup** (30-45 seconds): Move to the pre-baked terminal tab with the completed state
4. **Nuclear option** (if backup also fails): Switch to asciinema recording or screenshot walkthrough: "Let me show you a recording of exactly what you would have seen"

### 3.3 Composure and Framing

How you handle failure matters more than whether it happens. The srinathmohan article identifies "confident composure" as the key skill.

**What to say:**
- "This is what happens in real life too — let me show you how gdev handles it" (turn the failure into a feature demo)
- "The demo gods are not with us today. Let me switch to a recording of exactly this sequence" (humor + pivot)
- "Interesting — this is actually a network issue, not a gdev issue. Let me show you the offline path" (diagnose briefly, then move on)

**What NOT to say:**
- "Oh no, this is broken!" (creates a negative peak memory)
- "This worked five minutes ago, I promise!" (desperation destroys credibility)
- "Let me just try one more thing..." (the unplanned detour anti-pattern)
- Nothing (silent troubleshooting is the worst — the audience has no idea what is happening)

### 3.4 Planned Failures as Demo Beats

Advanced presenters deliberately trigger failures to demonstrate recovery. For gdev, this could be a powerful technique:

- Run `gdev init` successfully, then deliberately introduce a bad dependency to show that `gdev doctor` catches it
- Attempt to install a package that violates an age-gate, showing the PreToolUse hook blocking it
- Show `gdev status` dropping from A to C after removing a security layer, demonstrating drift detection

**This technique serves dual purposes:** it demonstrates the tool's resilience features AND gives the audience a "controlled failure" experience that normalizes the idea that things break — and gdev catches them.

---

## 4. Showing Security Features Without Boring the Audience

### 4.1 The Invisibility Problem

Security hardening is inherently invisible when it works correctly. "Nothing happened" is the desired outcome — but "nothing happened" is not a demo. This is the fundamental challenge of demonstrating security tools.

### 4.2 The Threat-Defense-Verification Pattern

The most effective technique for making security visible, synthesized across Snyk/Semgrep positioning analysis and security demo research:

1. **Name a specific threat** (not abstract risk): "92% of malicious PyPI packages are published and removed within 24 hours. If you install one during that window, standard vulnerability scanners will never flag it because the CVE hasn't been published yet."

2. **Show the defense**: "gdev configures age-gating: any package published less than 72 hours ago triggers a warning. Any package less than 24 hours old is blocked."

3. **Prove it works (verification)**: Run the safe test fixture against the defense layer. Show the block in real time. "This is the EICAR equivalent for supply chain security — a safe test that proves the defense is active."

This three-beat pattern turns invisible features into visible, dramatic moments. The threat creates stakes; the defense creates reassurance; the verification creates proof.

### 4.3 Making Defense-in-Depth Visual

For the 6-layer security model, do NOT walk through all 6 layers in a 15-minute demo. Cognitive load theory limits new concepts to 3-4 per segment. Instead:

- **Show one layer in detail** (the most dramatic: age-gating with live test fixture)
- **Reference the others by count**: "This is one of 6 independent defense layers. Each one works even if the others fail. Each one has its own test fixture."
- **Show the aggregate**: `gdev status` with the posture score showing all layers active (the visual summary)

For leadership audiences, the single most effective security demo moment is `gdev evidence --framework soc2` — the instant production of audit-ready compliance documentation. This turns invisible security into a visible artifact they can hold.

### 4.4 The Fear-to-Enablement Pivot

Leadership adoption research established that fear-based messaging backfires with sophisticated audiences. The demo should follow this arc:

1. **Brief factual threat** (10-15 seconds): "SolarWinds affected 18,000 organizations through a routine update. The attack vector was the software supply chain — exactly what we're protecting against."
2. **Quick pivot to enablement** (not dwelling on fear): "gdev makes security the default. Your developers don't have to think about it — it's already configured correctly."
3. **Show the positive outcome**: Posture score, evidence report, passing test fixtures — frame these as achievements, not defenses.

---

## 5. Audience Engagement in a 15-Minute Window

### 5.1 Questions: During vs End

For 15 minutes, the consensus across sources is **defer questions to the end**. Reasons:
- Interruptions break narrative flow and can derail timing
- A single "can it do X?" question can consume 2-3 minutes — 15-20% of your total time
- The Developer Advocacy Handbook notes that mid-talk questions distract from the planned arc and create awkward moments for recordings

**How to defer gracefully:**
- State it upfront: "I'll do a quick 13-minute demo and then we'll have time for questions at the end"
- If someone interrupts: "Great question — I'm actually going to show that in about 2 minutes" (if you are) or "Let me come back to that — I want to make sure I show you the key thing first" (if you are not)

**The exception:** If a senior stakeholder (VP Eng, CTO) asks a question during the demo, answer it briefly. The organizational dynamics make deferral feel dismissive. Keep the answer to 15-20 seconds, then smoothly return to the demo.

### 5.2 Handling "Can It Do X?" Interruptions

These are the most dangerous questions because they invite unplanned live exploration — the "let me just..." anti-pattern.

**Response framework (Five-Option Model from presentation research):**
1. **Answer directly** (if the answer is short): "Yes, gdev supports 27 ecosystems including Elixir. Let me show you the detection after the demo."
2. **Defer with specificity**: "That's a great question about custom profiles. I'll show you exactly that after the main demo."
3. **Deflect with context**: "gdev doesn't do X directly — it generates standard devenv.nix files that you can extend however you need. The exit story is important: nothing is locked in."
4. **Redirect to existing content**: "We have a detailed doc on that — I'll share the link after."
5. **Acknowledge limitation honestly**: "That's not something gdev handles today. It focuses on the initial generation and security hardening, not ongoing X." (Honest limitation builds trust per ELM/developer skepticism research)

**Never:**
- Start exploring a feature live that you have not rehearsed
- Say "let me just..." and start typing unplanned commands
- Promise something the tool cannot do

### 5.3 Reading the Room: Engagement vs Disengagement Signals

From Turpin Communication and presentation coaching research:

**Engaged audience:**
- Eye contact with the presenter
- Nodding, especially during problem-statement beats
- Taking notes or photographs of the screen
- Leaning forward
- Asking questions (even if you defer them, questions signal engagement)

**Disengaged audience:**
- Crossed arms, slouching, leaning back
- Looking at phones/laptops
- Avoiding eye contact or staring blankly
- Yawning, fidgeting
- Side conversations

**Recovery tactics when you're losing them:**
- Speed up the narrative (you may be in the "old way" section too long)
- Jump to the live demo earlier than planned (action recaptures attention)
- Ask a direct question: "How many of you have spent more than an hour setting up a dev environment for a new project?" (hand raise creates physical engagement)
- Change your position or volume (physical movement breaks pattern)
- Skip to the most impressive demo beat

### 5.4 The "Try It Yourself" Moment

In a 15-minute format, there is no time for audience hands-on. However, the CTA should make the "try it yourself" moment feel immediate:

- Show the install command on screen: `curl -sSfL https://get.gdev.dev | sh`
- Leave it visible during Q&A
- Say: "This is a 10-second install. Try it on any project tonight — `gdev init` works on any directory with a go.mod, package.json, or Cargo.toml."
- If possible, provide a QR code linking to a quickstart guide

The goal is to convert demo enthusiasm into a same-day trial. PLG research shows the conversion window is narrow: if they do not try it within 48 hours of seeing the demo, they likely never will.

---

## 6. Demo Anti-Patterns

### 6.1 The Deadly Seven (Ranked by Frequency)

These anti-patterns are synthesized from the ecosystem analysis, srinathmohan's failure analysis, the Presentation Patterns book (Ford et al.), daily.dev demo guide, and Developer Advocacy Handbook.

**1. "Let Me Just..." (The Unplanned Detour)**
Triggered by audience questions or presenter curiosity. The presenter starts typing unplanned commands, hits an error, tries to debug, and burns 3-5 minutes. Prevention: pre-plan every command. If asked about something off-script, defer or switch to a prepared screenshot.

**2. Feature Dumping (The Twelve-Feature Tour)**
Showing every feature in 15 minutes instead of deeply demonstrating 2-3 through a narrative. The audience remembers nothing. The "Dead Demo" antipattern from Presentation Patterns confirms: even outstanding demonstrators cover only 60% of the material possible in a pure presentation. Prevention: choose 3 capabilities that tell one story.

**3. Configuration Showcase (Showing YAML Instead of Outcomes)**
Displaying configuration files line-by-line is not a demo. "Here's what the devenv.nix looks like..." for 4 minutes is a sure way to lose the room. Prevention: show the *result* of the configuration (working environment, passing tests, posture score), then briefly flash the config as evidence of quality.

**4. Architecture Before Action (Premature Explanation)**
Opening with "Let me explain how gdev works internally..." and spending 5 minutes on architecture before showing the tool running. Prevention: show the tool working first. Explain the mechanism only after they have seen the outcome and care about understanding it.

**5. The Happy Path Only**
Never showing what happens when things go wrong. Experienced engineers know things break. Showing only the golden path destroys credibility. Prevention: include at least one deliberate failure/recovery moment (a defense layer catching a bad package, `gdev doctor` finding an issue).

**6. Starting with "About Me" / "About My Company"**
Corporate slides, biography, team photo. The Developer Advocacy Handbook explicitly says: cut all of this. The audience came for the demo, not your LinkedIn profile. Prevention: jump straight to the pain point. If introduction is required, limit it to one sentence.

**7. The WiFi-Dependent Demo**
Building a demo that requires network access at a conference where 500 people are on the same WiFi. Prevention: pre-populate all caches, use offline mode, have a mobile hotspot as backup, test the full demo with airplane mode enabled.

### 6.2 The "Works on My Machine" Failure Mode

The most embarrassing demo failure: the tool works perfectly in rehearsal but breaks on the demo machine. Causes:
- Different OS version, shell configuration, or Nix store state
- Missing environment variables or credentials
- Different terminal emulator with different rendering
- Screen resolution or font rendering differences

Prevention: rehearse on the exact machine, in the exact venue, with the exact display configuration you will use for the real demo. If using your own laptop, close and reboot before the demo. If using a provided machine, bring a USB stick with your entire demo environment.

---

## 7. Reference Examples and Patterns

### 7.1 The "Empty Directory to Working App" Pattern

The most effective developer tool demo pattern, used by Vercel, Bun, and mise: start with nothing and end with something working. For CLI tools, the visual is:

```
$ ls
(empty directory)
$ gdev init
[output showing detection, generation, configuration]
$ ls
devenv.nix  devenv.yaml  .envrc  .claude/  .pre-commit-config.yaml  ...
$ devenv shell
[working environment with all tools available]
```

This pattern works because:
- The empty start is the control condition (before)
- The single command is the intervention
- The populated directory is the experimental result (after)
- The working shell is the proof of correctness

YCM Jason's tip to "start with `mkdir your-topic`" maps directly to this: create a project directory live, or clone a repo, then show that gdev transforms it.

### 7.2 The Before/After Split

For contexts where a side-by-side comparison strengthens the case:

- **Two terminal windows side by side**: Left shows manual setup (scrolling list of commands, files to create, configurations to write); Right shows `gdev init` producing the same result in one command
- **Timer comparison**: "Manual setup: 47 minutes. gdev: 58 seconds."
- **File count comparison**: "Manual: create 15 files across 6 directories. gdev: one command."

This is especially effective for the "show the old way" beat — have the manual process visible as a reference while the automated process runs.

### 7.3 Exemplary Developer Tool Demo Patterns

**Kelsey Hightower (HashiCorp/Kubernetes):**
- Famous for live demos at HashiConf
- Pattern: starts with the pain of manual infrastructure, then shows automated solution
- Handles delays with self-aware humor: acknowledges bootstrapping time, fills dead air with commentary
- Once deployed a Nomad cluster using voice recognition from his phone — pushing the boundary of what a "live demo" can be

**esbuild (Evan Wallace):**
- Demo IS the benchmark: a single bar chart showing 0.39s vs 41.21s
- No narration needed — the visual tells the story
- Pattern: let the numbers speak; do not over-explain obvious performance

**Vercel (Guillermo Rauch):**
- Pattern: `git push` to live URL in under 60 seconds
- The deploy-in-seconds demo creates genuine audience surprise
- Minimal narration during the deploy; let the speed be the statement

**Turborepo:**
- Pattern: before/after CI time with specific dollar amounts
- User testimonials with metrics: "saved us $20k", "67 HOURS of CI"
- The number is the pitch — memorable and repeatable

### 7.4 Asciinema as a Safety Net

Asciinema (asciinema.org) records terminal sessions as text-based data, not video. This makes it ideal for demo fallbacks because:

- Recordings are tiny (50KB for 5 minutes vs megabytes for video)
- Text remains copy-pastable from the recording
- Playback speed can be adjusted in real time
- Can be embedded in slides or web pages
- Renders at native resolution on any display (always sharp)

**Recommended workflow:** Record a complete, perfect run of the demo using asciinema. If the live demo fails, switch to the asciinema playback with live commentary: "Let me show you a recording of exactly this sequence — I'll narrate as it plays." The audience gets the same information with slightly less authenticity but zero risk.

---

## 8. Dual-Audience Strategy: Developers + Leadership

### 8.1 The Fundamental Tension

Developers want to see real commands, real terminal output, and real code. Leadership wants to see business impact, risk reduction, and metrics. A 15-minute demo cannot serve both deeply, so the strategy must be:

**Run the demo through a developer lens; narrate in leadership language.**

This means:
- The visual is always the terminal (developers are engaged by authenticity)
- The narration adds business context ("This just generated 6 independent security layers — that's what your next SOC2 auditor will ask about")
- The metrics are spoken, not just shown ("60 seconds instead of 90 minutes — multiply that by 50 developers and 10 projects")

### 8.2 Cognitive Load Management for Mixed Audiences

Cognitive load theory limits new concepts to 3-4 per segment for a 15-minute format. For a mixed audience, the 3 concepts should be:

1. **One-command environment generation** (resonates with both: developers see DX, leadership sees productivity)
2. **Security by default** (resonates with both: developers see it is non-intrusive, leadership sees risk reduction)
3. **Measurable compliance** (resonates primarily with leadership, but developers appreciate that it eliminates manual audit work)

Do NOT try to also show: AI agent integration, consulting lifecycle, self-updating, cross-platform support, pre-commit hooks in detail, CI workflow generation, or the full 27-ecosystem list. Save these for Q&A or a longer format.

### 8.3 Language Bridging

| What You Show | Developer Narration | Leadership Narration |
|--------------|--------------------|--------------------|
| `gdev init` completing | "That just generated your devenv.nix, Claude Code config, and pre-commit hooks" | "Every new project now starts at the same security baseline, automatically" |
| Generated devenv.nix | "Look — correct Nix packages for this Go project, formatters, linters, all configured" | "This eliminates 90 minutes of manual setup per project" |
| `gdev status` posture score | "A grade means all 6 defense layers are active" | "This is the metric your auditor can verify — no manual documentation needed" |
| Security test fixture triggering | "The age-gate just caught that — anything published less than 24 hours ago is blocked" | "92% of PyPI malware is published and removed within 24 hours. This catches it automatically" |
| `gdev evidence` output | "Maps every defense to specific SOC2 control IDs with SHA256 hashes" | "This is the compliance evidence that currently takes your team weeks to produce manually" |

---

## 9. Practical Timing Template

A concrete minute-by-minute template for a 15-minute gdev demo, incorporating all research findings:

| Time | Beat | What Happens | Slide/Terminal | Notes |
|------|------|-------------|---------------|-------|
| 0:00-0:30 | Hook | "We counted: 15 files, 47 decisions, 90 minutes. That's what every project costs before you write code." | Slide: the number "90 minutes" in large type | Primacy effect: the opening is remembered |
| 0:30-1:30 | Pain | Brief story: new dev joins, spends day 1-3 configuring | Slide: 2-3 bullet points listing the manual steps | Keep it relatable; use "we" not "they" |
| 1:30-2:30 | Old Way | Quick flash of the 15 files they would need to create | Terminal: `ls` showing empty dir; briefly list what they need | Do not dwell. 60 seconds max. Create desire for a better way |
| 2:30-3:00 | Transition | "What if this was one command?" Start timer on screen | Terminal: clean prompt, empty directory | Build anticipation |
| 3:00-5:00 | Live Demo: Init | Type `gdev init` live. Narrate as output scrolls: detection, generation, configuration. Timer running. | Terminal: `gdev init` with live output | THE AHA MOMENT. Must be live. If fails, switch to backup by 3:45 |
| 5:00-6:30 | Reveal | Open generated files. Show devenv.nix, settings.json, pre-commit config. "All of this, in 58 seconds." Stop timer. | Terminal: `cat` or `bat` key files | Pause for impact. Let the audience absorb. |
| 6:30-8:00 | Security Layer | "But speed is only half the story." Name one threat. Show one defense. Run test fixture. | Terminal: test fixture triggering age-gate or script blocker | Threat-defense-verification pattern. The dramatic security beat |
| 8:00-9:00 | Posture Score | `gdev status` — show the A grade, 92/100 score. "This is every project, measured consistently." | Terminal: colorized status output | Bridge to leadership value |
| 9:00-10:30 | Quantify | Specific metrics: "60 seconds vs 90 minutes. 6 defense layers. 27 ecosystems. $0/month." | Slide or terminal: key numbers in large type | Use the 3-value-driver rule |
| 10:30-12:00 | Peak Moment | The most impressive capability: `gdev evidence --framework soc2` or onboarding demo (`gdev init --mode join` on second terminal) | Terminal: evidence output or join mode | This is the designed peak. Choose based on audience composition |
| 12:00-13:00 | Close | Single CTA: "Try it tonight. `curl -sSfL https://get.gdev.dev \| sh`. Then `gdev init` on any project." | Slide: install command + QR code to quickstart | Recency effect: last thing they see shapes memory |
| 13:00-15:00 | Q&A | "I'd love to hear your questions." Handle 2-3 questions briefly. | Keep install slide visible | Do not start new demo sequences during Q&A |

---

## 10. Pre-Demo Checklist

A practical checklist synthesized from all sources:

### One Week Before
- [ ] Script the entire demo: every command, every narration beat, every transition
- [ ] Record an asciinema fallback of the complete demo sequence
- [ ] Prepare the `prepare-for-demo.sh` reset script
- [ ] Pre-populate all Nix stores and caches for offline operation
- [ ] Identify and prepare the 3 "parking lot" answers for likely questions

### One Day Before
- [ ] Full dry run with a colleague (they should play skeptic)
- [ ] Test the demo on the actual machine you will use
- [ ] Verify font size, color scheme, and terminal width
- [ ] Time the full run — if over 13 minutes, cut a section
- [ ] Prepare backup terminal tab with completed state

### 10 Minutes Before
- [ ] Run `prepare-for-demo.sh` to reset to known state
- [ ] Run the full demo sequence once (end-to-end verification)
- [ ] Close all unnecessary applications, disable notifications
- [ ] Set terminal font size and open the correct windows
- [ ] Test screen sharing / projector connection
- [ ] Open backup asciinema recording, ready to play
- [ ] Start any background processes needed (Nix daemon, etc.)

### During the Demo
- [ ] Start with the hook, not with "about me"
- [ ] Narrate BEFORE typing each command
- [ ] Pause 3 seconds after output appears
- [ ] If something breaks: try once (15s), acknowledge (15s), switch to backup (15s)
- [ ] Watch for disengagement signals; skip to next beat if losing them
- [ ] End with the strongest beat, not Q&A
- [ ] Leave the install command visible on screen

---

## Sources

### New Sources (saved to `docs/` during this investigation)

1. `docs/dasroot-live-coding-presentations-best-practices.md` — Live coding in presentations: preparation, environment setup, engagement, error handling (dasroot.net, 2026)
2. `docs/measuredco-great-live-demos-guide.md` — How to do great live demos: structure, pair presenting, preparation, question management (dev.to/measuredco)
3. `docs/srinathmohan-why-tech-demos-fail.md` — Why tech demos fail: four failure modes, prevention strategies, notable case studies (Medium, srinathmohan)
4. `docs/developer-advocacy-handbook-talk-delivery.md` — Developer Advocacy Handbook: pacing, live coding risks, question management, content strategy (developer-advocacy.com)
5. `docs/developer-advocacy-handbook-delivering-talks.md` — Developer Advocacy Handbook: delivering talks, authenticity, audience management, honesty (developer-advocacy.com)
6. `docs/craignicol-live-coding-during-presentation.md` — Live coding during presentations: preparation, risk management, handling imperfection (dev.to/craignicol)
7. `docs/ycmjason-5-tips-live-coding-talks.md` — 5 tips for live coding talks: enthusiasm, empty-project start, minimize automation, code elegance (dev.to/ycmjason)
8. `docs/gilliard-live-coding-tips.md` — Comprehensive live coding guide: terminal config, font/color, command efficiency, contingency planning (blog.gilliard.lol)
9. `docs/turpin-reading-the-room-engagement.md` — Reading the room: engagement signals, real-time responsiveness, pausing technique (turpincommunication.com)
10. `docs/sixminutes-presentation-patterns-book-review.md` — Presentation Patterns book review: 60+ patterns, 25+ antipatterns including Dead Demo and Expansion Joints (sixminutes.dlugan.com)

### Pre-Existing Sources (from P2-T1, P2-T2, P2-T3)

11. `docs/developer-focused-demos-daily-dev.md` — Developer-focused demo creation: 5-step process, interactive elements, anti-patterns (daily.dev)
12. `docs/storytelling-for-technical-demos.md` — Five-beat storytelling framework for technical demos (codewithcaptain.com)
13. `docs/luke-lowrey-product-demo-tips.md` — Product demo tips: team approach, pacing, question management (lukelowrey.com)
14. `docs/freecodecamp-tech-conference-talks.md` — Conference talk guide: audience, storytelling, visual design, engagement (freecodecamp.org)
15. `docs/nngroup-peak-end-rule.md` — Peak-end rule for experience design (nngroup.com)
16. `docs/cognitive-load-theory-presentations.md` — Cognitive load theory applied to presentations (ethos3.com)
17. `docs/simply-psychology-elaboration-likelihood-model.md` — Elaboration Likelihood Model: central vs peripheral processing (simplypsychology.org)
18. `docs/evil-martians-100-devtool-landing-pages.md` — 100+ devtool landing page analysis (evilmartians.com)
