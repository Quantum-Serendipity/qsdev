# Developer Onboarding Workflow

## Research Question

What happens when a new engineer joins a project that already has gdev configuration? How do we minimize the clone-to-productive gap while maintaining security compliance?

## The Clone-to-Productive Gap

The "clone-to-productive" gap is the time between `git clone` and a developer being able to build, test, and run the project. Industry benchmarks:

- **Best in class:** Under 30 minutes (dev containers, automated provisioning)
- **Good:** 2-3 hours (documented setup scripts)
- **Typical:** 1-3 days (manual wiki-based setup)
- **Consulting firm pain point:** Multiplied by frequent project rotation

For a consulting firm where engineers rotate across client projects, this gap is experienced repeatedly -- not just at hiring. Every project transition is an onboarding event.

## Scenario Analysis: `gdev init` on an Existing Project

### Scenario 1: New Engineer, Existing Project with `.gdev.yaml`

The optimal flow for minimum friction:

```
1. git clone <project-url>             # ~10 seconds
2. cd <project>
3. gdev init                           # Detects .gdev.yaml, enters "join" mode
   -> Reads .gdev.yaml
   -> Checks gdev binary version compatibility
   -> Detects what's already configured (devenv.nix, .claude/, etc.)
   -> Identifies what needs local setup:
      - Install devenv.sh if missing
      - Install direnv if missing  
      - Install claude CLI if missing
   -> Offers to install missing tools via gdev setup
   -> Generates local-only files (.gdev.local.yaml template)
   -> Activates devenv shell
4. devenv shell                        # Working environment
```

**Target: 3 commands, under 2 minutes of hands-on time.**

### Key Detection Logic

When `gdev init` runs in a directory with existing config, it must distinguish:

| What to Detect | How | Action |
|---|---|---|
| `.gdev.yaml` exists | File existence check | Enter "join" mode (not "create" mode) |
| Generated files present (devenv.nix, .claude/) | File existence + hash check against `.gdev.yaml` config | Skip generation, verify state |
| Generated files match expected state | SHA256 comparison against what `.gdev.yaml` would produce | Report drift if mismatched |
| Local tools installed (devenv, direnv, claude) | `which` / PATH check | Offer `gdev setup` for missing |
| devenv shell activated | `DEVENV_ROOT` env var check | Skip activation prompt |
| Trust not established (mise model) | Check if project path is trusted | Prompt for trust on first run |

### Scenario 2: Existing Engineer, New gdev Version

Engineer pulls latest gdev binary, runs on an existing project:

```
$ gdev init
  gdev v0.16.0 | project config version: 1 | compatible: yes
  
  Checking project state...
  ✓ .gdev.yaml present (profile: go-web-service)
  ✓ devenv.nix present (matches expected state)
  ✓ .claude/settings.json present (3 user modifications detected)
  ✓ All tools installed
  
  Updates available:
  - 2 new pre-commit hooks added in v0.16.0 (ripsecrets, semgrep)
  - Claude Code deny rules updated (12 new patterns)
  - skills/security-review.md updated to v2.1
  
  Run `gdev init --update` to apply updates.
```

### Scenario 3: Existing Engineer, Outdated gdev Version

```
$ gdev init
  ⚠ gdev version mismatch
  Your version:    v0.14.2
  Required:        >= 0.15.0 (from .gdev.yaml gdev_version)
  
  This project requires a newer gdev. Update with:
    gdev self-update
  
  Or override with --skip-version-check (not recommended)
```

### Scenario 4: Brownfield Project (No .gdev.yaml Yet)

An existing project without gdev configuration. This is the standard `gdev init` wizard flow:

```
$ gdev init
  No .gdev.yaml found. Let's set up this project.
  
  Detected: Go project (go.mod), TypeScript (package.json)
  Existing configs: .eslintrc.js, tsconfig.json, Dockerfile
  
  ◆ Set up with recommended defaults?
    ● Yes -- Go 1.22, TypeScript 22, devenv.sh, Claude Code
    ○ No, let me customize
```

After the wizard, `gdev init` generates `.gdev.yaml` plus all output files. The `.gdev.yaml` is committed to git so the next team member gets Scenario 1.

## Machine-Specific vs Project-Specific Setup

Critical distinction for the onboarding flow:

### Project-Specific (Already in Git)
These files are generated once by the first engineer, committed, and shared:
- `.gdev.yaml` -- project configuration
- `devenv.yaml` -- devenv inputs and settings
- `devenv.nix` -- development environment definition
- `.envrc` -- direnv activation
- `CLAUDE.md` -- AI assistant instructions
- `.claude/settings.json` -- Claude Code settings
- `.claude/skills/*.md` -- team skills
- `.claude/rules/*.md` -- team rules
- `.mcp.json` -- MCP server config
- `.gitignore` -- ignore patterns
- `.pre-commit-config.yaml` -- pre-commit hooks

### Machine-Specific (Local Setup)
These must happen on each developer's machine:
- Install gdev binary itself
- Install devenv.sh
- Install direnv + shell hook
- Install claude CLI
- Run `devenv shell` to download Nix derivations
- Trust the project directory (mise-style trust model)
- Generate `.gdev.local.yaml` from template
- Authenticate to any MCP servers (GitHub token, etc.)

### The `gdev setup` Command

For machine-specific setup, `gdev setup` (run once per machine, not per project) handles:

```
$ gdev setup
  Checking system prerequisites...
  
  ✓ Nix package manager (v2.28.0)
  ✓ devenv.sh (v2.1.0)  
  ✓ direnv (v2.35.0)
  ✗ claude CLI (not found)
  ✗ pre-commit (not found)
  
  Install missing tools? [Y/n] y
  
  Installing claude CLI... done
  Installing pre-commit... done
  
  Shell integration:
  ✓ direnv hook in .zshrc
  ✗ mise activation not found
  
  Add mise activation to .zshrc? [Y/n] y
  
  Setup complete. You may need to restart your shell.
```

## UX Audit: Commands from Clone to Working Environment

### Ideal Flow (gdev already installed)

| Step | Command | Time | Notes |
|------|---------|------|-------|
| 1 | `git clone <url> && cd <project>` | 10-30s | Network dependent |
| 2 | `gdev init` | 5-10s | Detects .gdev.yaml, verifies state, reports any drift |
| 3 | `devenv shell` | 30-120s | First run downloads Nix derivations (cached after) |
| **Total** | **3 commands** | **~2 minutes** | Subsequent runs: <10s |

### First-Time Flow (fresh machine)

| Step | Command | Time | Notes |
|------|---------|------|-------|
| 1 | `curl -fsSL https://get.myxdev.dev \| sh` | 30-60s | Installs gdev binary |
| 2 | `gdev setup` | 2-5 min | Installs devenv, direnv, claude, shell hooks |
| 3 | `git clone <url> && cd <project>` | 10-30s | |
| 4 | `gdev init` | 5-10s | |
| 5 | `devenv shell` | 30-120s | First Nix download |
| **Total** | **5 commands** | **~5 minutes** | One-time machine setup |

### Comparison to Alternatives

| Tool | Commands from clone | Approximate time | Notes |
|------|-------------------|-----------------|-------|
| gdev (proposed) | 3 (post-setup) | ~2 min | Nix download dominates |
| Dev Containers | 1 (open in container) | 2-10 min | Container build time |
| mise | 2 (trust + install) | 30s-2 min | No env isolation |
| Manual wiki | 10-30 | 1-3 days | Error-prone |

## Detection Engine Design

The detection engine is the core of smart onboarding. It reads existing project state to make intelligent decisions:

```go
type ProjectState struct {
    // Config state
    HasGdevYaml       bool
    GdevYaml          *GdevConfig  // parsed if present
    GdevVersionCompat bool         // binary meets version constraint
    
    // Generated file state
    GeneratedFiles    map[string]FileState  // path -> state
    // FileState: Missing | MatchesExpected | UserModified | Drifted
    
    // Tool detection
    DetectedLanguages []Language   // from go.mod, package.json, etc.
    DetectedServices  []Service    // from docker-compose, etc.
    ExistingConfigs   []string     // .eslintrc, tsconfig, etc.
    
    // Machine state
    InstalledTools    map[string]ToolState  // tool -> installed/version/missing
}

type OnboardingMode int
const (
    ModeCreate  OnboardingMode = iota  // No .gdev.yaml, run full wizard
    ModeJoin                           // .gdev.yaml exists, verify + local setup
    ModeUpdate                         // .gdev.yaml exists, gdev version newer
    ModeRepair                         // .gdev.yaml exists, generated files drifted
)
```

The detection engine determines the mode, then the UI adapts:

- **Create mode:** Full wizard (quick path or customize)
- **Join mode:** Minimal prompts -- just confirm and do local setup
- **Update mode:** Show what changed, offer `--update`
- **Repair mode:** Show drifted files, offer to fix

## Edge Cases and Failure Modes

1. **Nix download fails (no internet):** If Nix derivations are not cached, `devenv shell` fails. Mitigation: `gdev init` warns if Nix cache is empty and suggests pre-populating.

2. **Trust prompt fatigue:** If every project requires trusting, engineers may blindly trust everything. Mitigation: `gdev setup` can pre-trust the company's project directory (like mise's `trusted_config_paths`).

3. **Version skew in team:** One engineer has gdev v0.16, another v0.14. The `.gdev.yaml` `gdev_version` constraint catches this, but the error must be actionable (include `gdev self-update` command).

4. **Partial state:** Engineer starts `gdev init`, interrupts mid-generation. Mitigation: Atomic write pipeline (from existing config-template-engine design) ensures no partial files. Either all files are written or none.

5. **Conflicting existing config:** Project has a hand-written `devenv.nix` that predates gdev. Detection engine must not clobber it. Mitigation: `gdev init` in create mode shows the plan preview and lists conflicts before writing anything.

6. **Multiple gdev versions on PATH:** Engineers on different projects needing different gdev versions. Mitigation: Project-level `.gdev.yaml` `gdev_version` constraint detects this. Unlike mise/proto (which manage their own versions), gdev relies on `gdev self-update` being backward-compatible.

## Depth Checklist

- [x] Underlying mechanism explained -- detection engine, four onboarding modes, three-layer config
- [x] Key tradeoffs and limitations identified -- Nix download time as bottleneck, trust prompt fatigue
- [x] Compared to alternatives -- dev containers, mise, manual setup
- [x] Failure modes and edge cases described -- six scenarios covered
- [x] Concrete examples -- command sequences, config examples, time estimates
- [x] Report is standalone-readable
