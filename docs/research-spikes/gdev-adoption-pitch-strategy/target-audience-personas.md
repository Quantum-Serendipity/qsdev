# Target Audience Personas for gdev Adoption

## Primary Decision Makers

### Persona 1: VP of Engineering / CTO ("The Business Case Buyer")

**Role:** Owns engineering org strategy, tooling budget (even $0 matters — opportunity cost of adoption), security compliance obligations.

**What they care about:**
- Risk reduction (supply chain attacks, compliance gaps, audit readiness)
- Developer productivity and onboarding speed
- Consistency across teams and projects
- Measurable outcomes (not just "it's better")
- Total cost of ownership (time, training, migration risk)

**Objections they'll raise:**
- "We already have security tools" → gdev curates and configures existing tools, doesn't replace them
- "How much disruption during adoption?" → Opt-in per project, reversible, profile-driven
- "What if this tool gets abandoned?" → Pure Go, MIT-licensed, all generated configs are standard files
- "What's the ROI?" → 2-minute onboarding vs hours, automated compliance evidence vs manual audit prep
- "Does this work with our existing stack?" → 27 ecosystems, works alongside existing CI

**Decision triggers:**
- Failed audit or security incident
- Slow onboarding hurting project delivery
- Inconsistent security posture across projects
- Client requiring compliance evidence

**Pitch angle:** Risk reduction + measurable compliance + developer velocity. Lead with posture scoring and compliance evidence, not features.

---

### Persona 2: Staff+ Engineer / Platform Team Lead ("The Technical Evaluator")

**Role:** Evaluates technical merit, influences adoption decisions, owns developer platform and tooling standards.

**What they care about:**
- Technical correctness (does it actually work? is the generated Nix valid?)
- Composability (does it play well with existing tools or fight them?)
- Escape hatches (can I override/customize/extend?)
- Maintenance burden (how much will this cost me in ongoing support?)
- Quality of generated artifacts (is the devenv.nix something I'd write myself?)

**Objections they'll raise:**
- "I can set this up myself" → Yes, for one project. Can you do it for 10-50 projects consistently?
- "Generated config is always garbage" → SHA256 tracking, three-way merge, section markers, devenv.nix never auto-overwritten
- "This is too opinionated" → 3 permission presets, per-ecosystem customization, .gdev.local.yaml for personal overrides
- "Another tool to maintain" → Self-updating static binary, lifecycle management for every tool it deploys

**Decision triggers:**
- Third time manually setting up the same devenv.nix pattern
- New team member taking days instead of hours to get productive
- Security tool configuration diverging across projects
- Claude Code generating poor/unsafe code due to missing context

**Pitch angle:** Technical depth + generated artifact quality + escape hatches. Show the actual generated devenv.nix and settings.json. Demonstrate the migration/update story.

---

### Persona 3: Security Engineer / AppSec Lead ("The Defense Validator")

**Role:** Owns security tooling strategy, validates compliance, responds to incidents, defines security policies.

**What they care about:**
- Defense depth (not just one layer)
- Verifiability (can I prove each defense works?)
- False positive/negative rates
- Coverage gaps (what ISN'T protected?)
- Incident response readiness

**Objections they'll raise:**
- "Security tools need active management" → gdev status runs in <100ms, drift detection is continuous
- "How do I know the defenses actually work?" → Safe test fixtures for every layer (EICAR equivalents)
- "What about the tools gdev itself depends on?" → SHA-pinned CI actions, compromised tools explicitly replaced (Trivy, KICS), static binary with zero runtime deps
- "This is security theater" → 6 independent layers, each provably working, with known limitations documented

**Decision triggers:**
- Supply chain incident (or near-miss)
- Audit finding about inconsistent security controls
- Need for automated compliance evidence
- Claude Code adoption raising AI-related security concerns

**Pitch angle:** Defense-in-depth architecture + provable testing + honest limitations. Show the 6-layer model, demonstrate test fixtures, present the compliance evidence output.

---

## Secondary Audiences

### Persona 4: Individual Developer ("The Daily User")

**Role:** Uses gdev day-to-day, needs it to not get in the way.

**What they care about:** Speed, not breaking their flow, not fighting the tool, easy escape when it's wrong.

**Key message:** "One command, then forget about it. Everything just works. If something breaks, `gdev repair` fixes it."

**Objections:** "Will this slow me down?" → Pre-commit hooks <10 seconds, drift detection <100ms, no network calls for baseline operations.

---

### Persona 5: Consulting Engagement Lead ("The Client-Facing Adopter")

**Role:** Manages client projects, needs consistent environments across engagements, clean handoff.

**What they care about:** Fast project setup, compliance evidence for clients, clean teardown between engagements, team consistency.

**Key message:** "Profile per client. `gdev init --profile client-healthcare --yes`. Evidence report at engagement end. Clean teardown."

---

## Adoption Path Model

```
Individual discovery → Personal use → Team champion → Leadership pitch → Org adoption
```

1. **Discovery**: Developer finds gdev, tries `gdev init` on a personal project
2. **Personal adoption**: Uses it on 2-3 projects, sees value in consistency
3. **Team evangelism**: Shows team lead the before/after, demonstrates onboarding speed
4. **Leadership pitch**: Champion presents posture scoring, compliance evidence, onboarding metrics
5. **Org adoption**: Profile standardization, CI enforcement via `gdev check`, team dashboards

The pitch materials need to serve both the **bottom-up** path (individual developer excitement → viral adoption) and the **top-down** path (leadership mandate → org-wide rollout). Different delivery formats serve different stages:

- **Elevator pitch**: Discovery and hallway conversations (bottom-up)
- **15-minute demo**: Team evangelism and leadership introduction (bridge)
- **Deep-dive presentation**: Leadership decision-making and security review (top-down)
