# gdev 15-Minute Demo: Presenter's Playbook

> **Format**: 15-minute live demonstration with hybrid execution
> **Audience**: Mixed (developers + engineering leadership)
> **Goal**: Drive same-day trial of `gdev init` on a real project
> **Core message**: One command, 60 seconds, security-hardened dev environment

---

## 1. Pre-Demo Setup Checklist

### One Week Before

- [ ] Script every command and rehearse end-to-end at least 3 times
- [ ] Record an asciinema fallback of the full demo sequence (text-based, ~50KB, renders at native resolution)
- [ ] Create and test `prepare-for-demo.sh` reset script (see below)
- [ ] Pre-populate Nix store so `devenv shell` requires zero network downloads
- [ ] Prepare 3 "parking lot" answers for likely questions: "Does it work with Docker?", "What if I need to customize?", "How do we roll this back?"
- [ ] Identify which closing beat to use based on confirmed audience composition

### One Day Before

- [ ] Full dry run with a colleague playing skeptic — they should interrupt, ask hard questions
- [ ] Time the full run — if over 13 minutes, cut something from the Explore segment (minutes 6:00-8:00)
- [ ] Verify all commands work on the exact machine and display you will use
- [ ] Record asciinema backup on the actual demo machine (not just your dev laptop)
- [ ] Prepare backup terminal tab with a fully completed gdev environment ready to switch to

### 10 Minutes Before

- [ ] Run `prepare-for-demo.sh` to reset demo directory to clean state
- [ ] Run the full demo sequence once, end-to-end, confirming every command works
- [ ] Close all applications: Slack, email, notifications, browser tabs
- [ ] Terminal settings:
  - Font size: 28pt minimum (32pt for conference projectors)
  - Color scheme: high-contrast dark theme (deep navy/gray, not pure black)
  - Prompt: `export PS1=$'gdev-demo\n> '` (remove path, git info, timestamps)
  - Window: share only the terminal window, not the full desktop
  - Width: verify no line wrapping on your longest command
- [ ] Open three terminal tabs:
  1. **Main**: Clean demo directory, ready for `gdev init`
  2. **Backup**: Completed environment (pre-baked, in case live init fails)
  3. **Asciinema**: Recording loaded and ready to play
- [ ] If presenting to a known audience, adjust narration track (see Section 3)

### The `prepare-for-demo.sh` Script

```bash
#!/bin/bash
# Reset demo environment to a known, clean state
set -euo pipefail

DEMO_DIR="/tmp/gdev-demo"
rm -rf "$DEMO_DIR"
git clone --depth 1 <your-realistic-project-repo> "$DEMO_DIR"
cd "$DEMO_DIR"

# Remove any existing gdev/devenv artifacts
rm -f devenv.nix devenv.yaml .envrc
rm -rf .claude .devenv .pre-commit-config.yaml
rm -f .gdev.yaml .mcp.json

# Verify gdev binary is available
gdev --version

echo "Demo environment ready at $DEMO_DIR"
```

### Audience-Specific Pre-Adjustments

| Audience | Adjust | Why |
|----------|--------|-----|
| Developer-heavy | Skip `gdev evidence` in closing; end with `devenv shell` proving tools work | Developers care about the generated code, not compliance reports |
| Leadership-heavy | Shorten file exploration (6:00-8:00); extend the Quantify segment | Leadership cares about metrics and risk, not devenv.nix internals |
| Mixed (default) | Run the script as written — dual-track narration handles both | The default path is designed for this |

---

## 2. Minute-by-Minute Script

### 0:00-2:00 — Name the Pain (Hook)

**Beat**: Pain
**Energy level**: Calm authority, building tension

**What to show**: Nothing yet. You are standing in front of the room (or on camera) with no terminal visible. Optionally, a single slide showing the number "90" in large type.

**What to say**:

> "We counted. Fifteen files. Forty-seven configuration decisions. Ninety minutes. That's what every new project costs before anyone writes a line of code."
>
> [Pause 2 seconds. Let the number land.]
>
> "devenv.nix. An .envrc. devenv.yaml. Claude Code settings with deny rules. A CLAUDE.md for project context. MCP server config. Pre-commit hooks. Security configs for your package manager. CI workflows. Each one hand-written, each one slightly different from the last project, each one a chance to miss something."
>
> "And here's the part that keeps engineering leads up at night: most teams are running Claude Code right now with zero guardrails. No deny rules. No package age-checking. No install script blocking. Every AI-suggested `npm install` is completely unguarded."

**Presenter notes**:
- Do NOT open with your name, title, or company. Jump straight into the number. Your intro can come during Q&A if someone asks.
- The "we counted" framing borrows credibility from measurement. It signals evidence, not opinion.
- Watch the room after "ninety minutes." If heads nod, you have them. If faces are blank, they may not manage multiple projects — shift emphasis toward the security angle ("zero guardrails") which is more universal.
- Pace: slow and deliberate. You are naming a problem, not delivering information. Speak as if confiding something the audience already suspects but hasn't quantified.

**Transition to next segment**:

> "Let me show you what that actually looks like."

---

### 2:00-3:30 — Show the Old Way

**Beat**: Pain (continued)
**Energy level**: Deliberate frustration — let the tedium be felt

**What to show**: Terminal. Switch to a clean project directory.

**Commands**:
```
> cd /tmp/gdev-demo
> ls
README.md  go.mod  go.sum  main.go  internal/  cmd/
```

**What to say**:

> "Here's a real Go web project. Fresh clone. To get this to a working, secure dev environment, I would need to create..."
>
> [Count on your fingers or gesture at the screen as you list each one]
>
> "A devenv.nix with the right Go version, formatters, linters. An .envrc to hook it into direnv. A devenv.yaml for devenv configuration. A settings.json for Claude Code with deny rules — and if you want real guardrails, that's 48 rules minimum. A CLAUDE.md so the AI understands the project. An .mcp.json for MCP server integration. Pre-commit hooks for formatting, linting, secrets scanning. Security configs for the Go module proxy. And CI workflows that actually enforce all of this."
>
> "That's 9 or more files, each requiring knowledge of both the ecosystem and the security tooling. On a good day, with experience, about 90 minutes. For a new team member? Could be a full day."

**Presenter notes**:
- Do NOT actually create the files. The point is the overwhelming list, not a walkthrough of manual setup.
- Keep this segment under 90 seconds. The danger is lingering in the pain too long — the audience gets it quickly. If you see engagement dropping, cut to the transition immediately.
- The finger-counting is a physical anchor that helps the audience track the scope. It also makes the number visceral rather than abstract.

**Transition**:

> "What if all of this was one command?"
>
> [Pause 2 full seconds. Let the question hang.]

---

### 3:30-6:00 — The Shift: gdev init (THE AHA MOMENT)

**Beat**: Shift
**Energy level**: Rising excitement — this is the peak of the entire demo

**What to show**: Live terminal. Type the command yourself. This must be live — it is the credibility moment.

**What to say (before typing)**:

> "I'm going to run one command. Watch what happens."

**Command**:
```
> gdev init --profile go-web --yes
```

**What to say (as output scrolls)**:

> "gdev just detected this is a Go project — it found the go.mod, figured out we're building a web service. Now it's generating everything."
>
> [Point at the terminal as each file appears]
>
> "devenv.nix with the right Go version, gopls, golangci-lint. Claude Code settings with 48 deny rules. Pre-commit hooks. Security configurations. MCP integration."
>
> [When it completes — pause 3 full seconds before speaking]
>
> "That's it. Everything we listed. One command."

**If the init takes more than 5 seconds of dead air**, fill with:

> "While this runs — gdev is doing ecosystem detection, resolving the right Nix packages for this Go version, generating security configs tuned to Go's module system, and wiring up Claude Code with guardrails specific to this project type."

**Presenter notes**:
- Use `--profile go-web --yes` for the demo. Zero-question mode means no interactive wizard prompts that could slow the flow or introduce uncertainty.
- Pre-type `gdev init` into your clipboard or shell history. Type `gdev` live for authenticity, then paste or ctrl-r the rest. Fewer keystrokes = fewer typos under pressure.
- THE 3-SECOND PAUSE AFTER COMPLETION IS CRITICAL. It feels like an eternity to you. To the audience, it is the moment the value lands. Do not rush past it. Let them read the output. Let the contrast with the "old way" settle.
- This is the designed emotional peak of the entire 15 minutes. Your energy should be noticeably higher here than anywhere else.

**If it fails**: You have exactly 15 seconds. Try the command once more. If it fails again, say: "Let me show you from the prepared environment" and switch to the backup tab. Continue narration as if nothing happened. Do NOT apologize or explain at length.

**Transition**:

> "Now let's look at what was actually generated."

---

### 6:00-8:00 — Explore What Was Generated

**Beat**: Shift (reinforcement)
**Energy level**: Confident, measured — proving the quality of what was just created

**What to show**: Open 2-3 generated files. No more than 3 — cognitive load limits apply.

**Command sequence**:
```
> ls -la
> cat devenv.nix
```

**What to say**:

> "Look at this devenv.nix. This isn't boilerplate — gdev detected our Go version from go.mod and pinned the matching Nix package. gopls for the language server, golangci-lint for linting, delve for debugging. The formatters match what the Go community actually uses."

```
> cat .claude/settings.json | head -30
```

> "Here are the Claude Code settings. See these deny rules? Forty-eight of them. They prevent the AI from running unvetted package installs, executing arbitrary scripts, modifying security configs. Each one is a specific guardrail, not a blanket restriction."
>
> [Point at a specific deny rule on screen]
>
> "This one blocks `go install` for packages gdev hasn't verified. This one prevents modifying the pre-commit config. The AI can still help you write code — it just can't bypass your security controls."

```
> devenv shell
```

> "And this is the proof. We're now inside the dev environment. Every tool is available. Go compiler, LSP, linters, formatters — all project-scoped, all reproducible."

**Presenter notes**:
- Use `cat` or `bat` (if available with syntax highlighting) — NOT an IDE. Opening VS Code introduces lag, window management complexity, and takes the audience out of the terminal flow.
- Show the deny rules in settings.json only briefly. Point at 2-3 specific ones. Do NOT read through the list. The point is "there are 48 and they're specific," not "here's what each one does."
- `devenv shell` should activate near-instantly because you pre-populated the Nix store. If it starts downloading packages, something went wrong with your prep — switch to the backup tab.
- For developer audiences, linger here an extra 30 seconds. They want to see the generated code is good. For leadership audiences, move through faster — they trust the tool if the demo is smooth; they don't need to read Nix.

**Transition**:

> "Speed is one thing. But there's a reason I kept mentioning security. Let me show you what's running underneath."

---

### 8:00-10:00 — The Security Story

**Beat**: Shift (deepening) + Quantify (beginning)
**Energy level**: Serious, authoritative — this is where leadership leans in

**What to show**: `gdev status` output, and optionally a defense layer firing.

**What to say (brief threat setup — 10 seconds max)**:

> "Here's the reality: 92% of malicious PyPI packages are published and removed within 24 hours. Standard vulnerability scanners never flag them because the CVE doesn't exist yet. The attack window is the install moment itself."

**Command**:
```
> gdev status
```

**What to say**:

> "This is your project's security posture right now. Score: 92 out of 100, grade A."
>
> [Point at the category breakdown on screen]
>
> "Six categories, each scored independently. Package management — that's the age-gating and install script blocking. AI guardrails — that's the 48 deny rules and the PreToolUse hooks. Environment integrity — that's the lock files and reproducible builds. Each one works even if the others fail. That's real defense-in-depth, not a single point of failure."
>
> [For leadership audiences, add:]
>
> "This is the number your auditor can verify. It's computed locally in under 100 milliseconds — no network calls, no SaaS dependency."

**Optional dramatic beat (if time permits, ~45 seconds)**: Show a defense layer catching something.

> "Let me show you what happens when something suspicious comes through."

Show a PreToolUse hook blocking a package install attempt, or reference the test fixture concept:

> "gdev ships with safe test fixtures — think of them as fire drills for your security layers. Each one proves a specific defense is working. The age-gate test publishes a fresh package to a local registry and confirms it gets blocked. Every layer has one."

**Presenter notes**:
- Do NOT walk through all 6 layers individually. That is feature dumping. Show the aggregate score, name the layers by count, and demonstrate one in detail if time allows.
- The "92% of malicious PyPI packages" statistic is your strongest security hook because it is specific, surprising, and verifiable. Use it even for Go projects — it illustrates the general principle of supply chain risk.
- If the audience is visibly engaged, include the defense layer demo. If they seem ready to move on, skip it and use the extra 45 seconds in the closing.
- The fear-to-enablement pivot is critical here. Spend no more than 10-15 seconds on the threat. Spend the rest on gdev's response. The narrative is empowerment, not fear.

**Transition**:

> "So we've got speed and security. Let me show you what this looks like at team scale."

---

### 10:00-12:00 — Quantify and Scale

**Beat**: Quantify
**Energy level**: Building toward the close — confident, forward-leaning

**What to show**: `gdev check` (CI enforcement) and `gdev evidence` (compliance).

**What to say**:

> "Everything I just showed you works for one developer on one project. Here's how it scales."

**Command**:
```
> gdev check --format human
```

> "This is what runs in CI. Pass or fail. Every pull request gets checked against the security baseline. If someone disables a pre-commit hook or removes a deny rule, the build fails. You don't have to police it — CI enforces it."
>
> [Brief pause, then the leadership-targeted beat:]
>
> "And for the conversation with your auditor..."

**Command**:
```
> gdev evidence --framework soc2
```

> "This maps every active defense layer to specific SOC2 control IDs. Each entry has a SHA256 hash proving the configuration existed at this point in time. This is the compliance evidence that currently takes teams weeks to produce manually — generated in seconds, always current, never stale."

**What to say (the three-value-driver summary)**:

> "Let me put numbers on this. One: developer productivity. Sixty seconds versus ninety minutes per project setup — multiply that by your team size and project count. Two: security posture. Six defense layers working independently, every one provably active, posture score visible across every project. Three: compliance cost. Audit evidence generated on demand instead of manually assembled over weeks. And the infrastructure cost for all of this is zero dollars a month — every component uses free-tier tooling."

**Presenter notes**:
- The three-value-driver rule from leadership adoption research: always have at least three because leadership will mentally discount the first one. Deliver all three in quick succession.
- The `gdev evidence` output is often the single most impactful moment for leadership audiences. If you had to cut the demo to 10 minutes, keep this and cut the file exploration (6:00-8:00) instead.
- Speak the dollar impact out loud even though it is not on screen: "Fifty developers, ten projects a year — that's 750 hours of setup time eliminated." Leadership thinks in headcount-hours, not seconds-per-command.
- If the audience is developer-heavy, keep this segment to 90 seconds and move to the close. Developers do not need to be convinced by ROI math.

**Transition**:

> "One more thing."

[This is a deliberate beat. The phrase signals something important is coming. Pause after saying it.]

---

### 12:00-13:30 — Close with One Action (Peak-End)

**Beat**: Action
**Energy level**: Peak intensity for the final command, then calm authority for the CTA

Choose ONE of the following closing beats based on audience composition. Do not try to do more than one.

#### Option A: The Compliance Evidence Reveal (for leadership-heavy audiences)

*Use this if `gdev evidence` was not already shown in the Quantify segment. If it was, use Option B or C instead.*

#### Option B: The Reversibility Proof (for mixed or skeptical audiences)

**What to say**:

> "I know what some of you are thinking: what happens if we adopt this and it doesn't work out? What's the exit?"

**Command**:
```
> gdev teardown --compliance
```

> "Every generated file — removed. But notice what it just did: it created an evidence archive first. A record of what was configured, what was active, and when it was removed. Clean exit with an audit trail."
>
> "gdev is fully reversible. The configs it generates are standard files — devenv.nix, settings.json, pre-commit configs. You can maintain them yourself if you ever want to stop using gdev. Nothing is locked in."

#### Option C: The 2-Minute Onboarding (for developer-heavy audiences)

**What to say**:

> "I showed you project setup. Let me show you what happens when a new developer joins an existing project."

Open a second terminal tab pointing at a project that already has a `.gdev.yaml`:

**Command**:
```
> gdev init --mode join
```

> "Join mode reads the team's configuration, verifies prerequisites, generates local files. No tribal knowledge. No asking three people what settings to use. New developer, productive in under two minutes."

---

**Regardless of which closing option you chose, end with this CTA**:

> "Try it tonight. On any project with a go.mod, package.json, Cargo.toml — any of 27 ecosystems."
>
> [Show the install command on screen, leave it visible:]

```
curl -sSfL https://get.gdev.dev | sh
```

> "Ten-second install, then `gdev init`. It's open source, MIT-licensed, and fully reversible. If you don't like it, `gdev teardown` removes everything."

**Presenter notes**:
- The closing beat must be as strong as the peak (gdev init). This is the peak-end rule in action. If the closing feels weak, switch to a different option.
- Leave the install command on screen during Q&A. It is the last visual impression.
- Do NOT end with "So... any questions?" as the final beat. The final beat is the CTA. THEN say "I'd love to hear your questions" as a separate moment.
- Single CTA only. Not "check out the docs, join our Discord, star us on GitHub, and try the tool." One thing: try `gdev init` on a real project.

---

### 13:30-15:00 — Buffer / Q&A

**What to say**:

> "I'd love to hear your questions."

**Handling questions**:
- If someone asks "can it do X?" and the answer is yes: "Yes — I can show you after. For now, the key thing is that one-command setup you just saw."
- If someone asks "can it do X?" and the answer is no: "That's not something gdev handles today. It focuses on initial generation and security hardening. The configs it generates are standard files you can extend however you need."
- If a senior stakeholder asks a question mid-demo: answer in 15-20 seconds, then return to the script. Do not defer a VP of Engineering.
- NEVER start typing unplanned commands during Q&A. If someone wants to see something specific, offer to show them after the session.

**If no questions come**: Have one ready for yourself. "One thing I didn't get to show you — gdev supports 27 ecosystems. TypeScript, Python, Rust, Java, .NET, Terraform, Docker — it detects the project type automatically. The setup you just saw works the same way for any of them."

---

## 3. Audience-Specific Narration Tracks

The demo flow is identical regardless of audience. What changes is the narration layer. Here are the emphasis shifts for each audience type.

### Developer Audience

**Emphasize**: Speed, generated code quality, escape hatches, reversibility.

| Segment | Developer-Specific Narration |
|---------|------------------------------|
| Hook (0:00-2:00) | Lean harder on the "90 minutes of yak-shaving" angle. Developers feel this in their bones. |
| Old Way (2:00-3:30) | Mention that the generated devenv.nix is "the config you would write yourself." Quality matters to this audience. |
| gdev init (3:30-6:00) | After completion, immediately open devenv.nix. Developers want to judge the output. |
| Explore (6:00-8:00) | Spend extra time here. Show that packages are pinned correctly, formatters match community conventions. Show `devenv shell` and run a tool to prove it works. |
| Security (8:00-10:00) | Frame as "security that doesn't slow you down." Pre-commit hooks under 10 seconds. Drift detection under 100ms. No network calls for baseline operations. |
| Close (12:00-13:30) | Use Option C (2-minute onboarding). Developers care about helping teammates get productive. |

**Avoid**: ROI calculations, compliance framework names, headcount multiplier math. Developers disengage from business language.

### Leadership Audience

**Emphasize**: Risk reduction, measurable compliance, team-wide visibility, cost.

| Segment | Leadership-Specific Narration |
|---------|-------------------------------|
| Hook (0:00-2:00) | Lean harder on the "zero guardrails on Claude Code" angle. Frame as organizational risk. |
| Old Way (2:00-3:30) | Mention inconsistency: "Every project has a different security posture. Some have pre-commit hooks, some don't." |
| gdev init (3:30-6:00) | Narrate in business terms: "Every new project now starts at the same security baseline, automatically." |
| Explore (6:00-8:00) | Move through quickly. Show settings.json deny rules briefly as evidence of guardrails, then move on. |
| Security (8:00-10:00) | This is where you earn their attention. The posture score is the number they will remember. Say: "This is the metric your auditor can verify." |
| Quantify (10:00-12:00) | Extend this section. Speak the dollar figures: "Fifty developers saving 90 minutes each, ten times a year — that's over 750 hours." |
| Close (12:00-13:30) | Use Option B (reversibility) — leadership's biggest fear is lock-in. Or use the `gdev evidence` close if it was not shown earlier. |

**Avoid**: Nix-specific terminology, implementation details, file-level walkthrough of configs.

### Mixed Audience (Default)

The default script is designed for this. Run the demo through a developer lens (real terminal, real commands, real output) but narrate in dual-track language. At each major beat, deliver one sentence for developers and one for leadership:

- After `gdev init`: "That just generated your devenv.nix, Claude Code config, and pre-commit hooks" [developers] + "Every new project now starts at the same security baseline, automatically" [leadership].
- After `gdev status`: "A grade means all 6 defense layers are active" [developers] + "This is the metric your auditor can verify — no manual documentation needed" [leadership].
- After `gdev evidence`: "Maps every defense to specific SOC2 control IDs with SHA256 hashes" [developers] + "This is the compliance evidence that currently takes your team weeks to produce manually" [leadership].

---

## 4. Error Recovery Playbook

### The 45-Second Rule

If any error is not resolved within 45 seconds, switch to your fallback. The audience's patience for troubleshooting is exactly this long. After 45 seconds, the failure becomes the demo.

**The fallback cascade** (same for every segment):
1. **0-15 seconds**: Try the command once more, or fix the obvious issue
2. **15-30 seconds**: Acknowledge and pivot: "Let me show you from the prepared environment"
3. **30-45 seconds**: Switch to backup terminal tab with completed state
4. **If backup also fails**: Switch to asciinema recording: "Let me show you a recording of exactly this sequence"

### Segment-by-Segment Recovery

**0:00-2:00 — Hook (no terminal, no risk)**

| What could go wrong | Recovery |
|---------------------|----------|
| Projector/screen sharing fails | Talk through the hook without visuals. It's all narration anyway. Fix the display during this segment. |
| You blank on the opening | Have the numbers written on a card you can glance at: "15 files, 47 decisions, 90 minutes." |

**2:00-3:30 — Show the Old Way**

| What could go wrong | Recovery |
|---------------------|----------|
| Demo directory is not clean (artifacts from a previous run) | Run `prepare-for-demo.sh` live: "Let me reset this to a fresh state — this is actually what you would be starting with." |
| Terminal font is too small on projector | Increase font size immediately. Acknowledge it: "Let me make that readable from the back." |

**3:30-6:00 — gdev init (HIGHEST RISK SEGMENT)**

| What could go wrong | Recovery |
|---------------------|----------|
| `gdev init` fails with an error | Try once more. If it fails again: "Interesting — let me show you from a completed environment." Switch to backup tab. |
| `gdev init` hangs or takes > 45 seconds | "In the interest of time, let me show you the completed result." Switch to backup tab. Continue narration normally. |
| Typo in the command | Laugh it off: "Live typing." Retype. This is humanizing, not damaging. |
| Network error during init | "The demo gods have opinions about WiFi. Let me switch to the offline version." Move to backup tab. |

**What to say if you switch to backup**:

> "This is exactly what you would have seen — same project, same profile, same output. The only difference is I ran it earlier instead of just now."

Do NOT say: "This worked five minutes ago, I promise." Do NOT apologize repeatedly.

**6:00-8:00 — Explore Generated Files**

| What could go wrong | Recovery |
|---------------------|----------|
| `devenv shell` starts downloading packages | "Looks like the cache needs warming — in normal use this is instant. Let me show you the tools that are available." Switch to backup tab where the shell is already active. |
| Generated file looks wrong or unexpected | Acknowledge it calmly: "That's interesting — let me check the profile." If not quickly fixable, move to the next segment. |

**8:00-10:00 — Security Story**

| What could go wrong | Recovery |
|---------------------|----------|
| `gdev status` shows a low score or errors | This can actually be turned into a feature: "See? It caught something. This is drift detection working in real time." |
| Defense layer demo does not trigger as expected | Skip it: "The test fixtures for each layer are in the repo — I'll share the link. Let me show you the team-scale features." |

**10:00-12:00 — Quantify**

| What could go wrong | Recovery |
|---------------------|----------|
| `gdev check` or `gdev evidence` fails | These are lower-risk since you can describe the output verbally. "The evidence report maps each defense to SOC2 control IDs with cryptographic hashes. Let me share an example output after the session." |

**12:00-13:30 — Close**

| What could go wrong | Recovery |
|---------------------|----------|
| `gdev teardown` fails | Simply describe what it does: "Teardown removes every generated file and optionally creates an evidence archive. The point is: nothing is locked in." |
| Second terminal for join mode is not ready | Skip to the CTA directly. The join mode demo is nice-to-have, not essential. |

### Universal Recovery Phrases

Keep these in your back pocket:

- "This is what happens in real life too — and that's actually why gdev has `gdev repair`."
- "The demo gods are not cooperating, but let me show you exactly what you would have seen."
- "Let me switch to a recording of this sequence — I'll narrate as it plays."

### What NEVER to Say

- "Oh no, this is broken" (creates a negative peak memory)
- "This worked five minutes ago, I promise" (desperation)
- "Let me just try one more thing..." (the unplanned detour — the single most common demo killer)
- [Silence while troubleshooting] (the audience has no idea what is happening and assumes the worst)

---

## 5. Emotional Arc Map

```
Audience
Energy
  ^
  |
  |                          * gdev init completes
  |                         /  (THE PEAK - minute 5)
  |                        /
  |                       /          * gdev evidence /
  |                      /          /  teardown close
  |    * "90 min" hook  /          /   (STRONG END - minute 13)
  |   /                /    *     /
  |  /                /    / \   /
  | /    * old way   /    /   \ /
  |/      tension   /    /  (valley:
  |       builds   /    /   quantify
  |               /    /    numbers)
  |              /    /
  |     * "what if  * gdev status
  |      one cmd?"   posture score
  |      (anticipation)
  +-----|-----|-----|-----|-----|-----|-----|----> Time
       1     3     5     7     9    11    13   (minutes)

Key moments:
  0:00  Hook with specific number — grabs attention
  2:30  "What if one command?" — builds anticipation
  5:00  gdev init completes — EMOTIONAL PEAK (design everything to protect this moment)
  6:30  File exploration — steady engagement, proving quality
  8:30  Posture score — second rise (leadership audience peaks here)
  9:30  Brief valley during metrics/numbers — this is expected and fine
 12:00  Closing beat — MUST match peak intensity (peak-end rule)
 13:00  CTA + install command — calm, confident, actionable
```

### Managing the Arc

**Protecting the peak (minute 5)**: Everything before this is setup cost. If you are running long, cut from the Old Way segment (2:00-3:30), never from the init demo. The peak must land between minutes 4-6.

**The valley (minutes 9-10)**: The Quantify segment is necessarily less visceral than live demos. This is fine — the audience needs a moment to process. Keep the energy up through vocal delivery and the three-value-driver structure, but do not worry if this segment feels less electric.

**The closing beat**: Must match the peak in emotional impact. This is the peak-end rule — the audience will judge the entire demo by the peak moment and the final moment. If your closing beat feels flat in rehearsal, switch to a different option from Section 2.

**If you are losing the room** (crossed arms, phone-checking, avoiding eye contact):
- Speed up the current segment and jump to the next live demo beat
- Ask a direct question: "How many of you have spent more than an hour setting up a dev environment for a new project?" (hand-raise creates physical engagement)
- If you are still in the Pain section, skip to `gdev init` immediately — action recaptures attention

---

## 6. Demo Variant: The 5-Minute Lightning Version

For lightning talks, tight meeting slots, or when your time gets cut. Hits only beats 1, 3, and 5 (pain, shift, action).

### 0:00-0:45 — Hook (Pain)

> "Every new project starts with the same 90 minutes of manual setup: devenv.nix, Claude Code config, pre-commit hooks, security hardening — 9 or more files, hand-written every time."

No slide. No terminal. Pure narration.

### 0:45-3:00 — The Shift (gdev init)

> "One command."

Switch to terminal:
```
> cd /tmp/gdev-demo && ls
README.md  go.mod  go.sum  main.go  internal/  cmd/

> gdev init --profile go-web --yes
```

Narrate while output scrolls:

> "Ecosystem detection. Nix package generation. Claude Code guardrails — 48 deny rules. Pre-commit hooks. Security configs. Done."

[Pause 3 seconds]

> "Sixty seconds. Everything you need, security-hardened by default."

Quick file peek:
```
> ls
devenv.nix  devenv.yaml  .envrc  .claude/  .pre-commit-config.yaml  ...
```

### 3:00-3:30 — One Proof Point

Choose one based on audience:
- **Developers**: `devenv shell` and run a Go tool to prove it works
- **Leadership**: `gdev status` to show the posture score (A, 92/100)
- **Security**: Reference the 6 defense layers and test fixtures

### 3:30-4:30 — Close (Action)

> "Open source. MIT-licensed. Fully reversible — `gdev teardown` removes everything."
>
> "Twenty-seven ecosystems. Six security layers. Zero dollars a month."
>
> "Try it tonight."

Show install command:
```
curl -sSfL https://get.gdev.dev | sh
```

### 4:30-5:00 — Buffer

One question, or graceful close: "I'm happy to show you more after the session."

### Lightning Version Notes

- No "old way" segment — there is no time. The "90 minutes" number does the work.
- gdev init MUST work. There is no time for fallback narration. If you are not 100% confident in the live demo, use the asciinema recording for the lightning version.
- One concept: "one command replaces 90 minutes." Do not try to also explain security, compliance, team features, or AI integration. Those are follow-up conversations.
- The 3-second pause after init completes is even more important in 5 minutes. It is the only moment the audience has to absorb the transformation.
