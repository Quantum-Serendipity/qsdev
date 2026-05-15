# gdev Claude Code Integration: Skills, Commands, and Wrapper Patterns

## 1. Claude Code Skill File Format

### 1.1 Format Overview

Skills are the current recommended format (replacing the legacy `.claude/commands/` directory). A skill is a directory containing a `SKILL.md` file:

```
.claude/skills/<skill-name>/
├── SKILL.md           # Main instructions (required)
├── template.md        # Template for Claude to fill in (optional)
├── examples/          # Example outputs (optional)
│   └── sample.md
└── scripts/           # Scripts Claude can execute (optional)
    └── validate.sh
```

### 1.2 SKILL.md Anatomy

Two parts: YAML frontmatter between `---` markers, and markdown body with instructions.

```yaml
---
name: my-skill              # Display name (optional, defaults to directory name)
description: What it does   # Recommended -- Claude uses this for auto-invocation
argument-hint: [args]       # Shown during autocomplete
arguments: [arg1, arg2]     # Named positional args for $arg1/$arg2 substitution
disable-model-invocation: true  # User-only invocation (for side effects)
allowed-tools: Bash(gdev *) Read Grep  # Pre-approved tools
model: opus                 # Model override
effort: high                # Effort level override
context: fork               # Run in isolated subagent
agent: Explore              # Subagent type when forked
---

Markdown instructions here. $ARGUMENTS gets replaced with user input.
$0 is first arg, $1 is second, $arg1 is named arg.
```

### 1.3 Dynamic Context Injection

The `` !`command` `` syntax is a preprocessor -- it runs shell commands *before* the skill content reaches Claude. The output replaces the placeholder.

```markdown
## Current system state
!`qsdev devenv doctor --json 2>/dev/null || echo '{"error": "gdev not installed"}'`

## Instructions
Based on the system state above, recommend actions...
```

Multi-line variant uses fenced code blocks opened with `` ```! ``:

````markdown
## Environment
```!
qsdev devenv doctor --json
qsdev status --json
git status --short
```
````

This is critical for gdev integration: the skill can gather system state *before* Claude starts reasoning, so Claude receives the actual state rather than needing to run discovery commands.

### 1.4 String Substitutions

| Variable | Description |
|:---------|:------------|
| `$ARGUMENTS` | All arguments passed when invoking |
| `$ARGUMENTS[N]` or `$N` | Specific argument by 0-based index |
| `$name` | Named argument from `arguments` frontmatter |
| `${CLAUDE_SESSION_ID}` | Current session ID |
| `${CLAUDE_EFFORT}` | Current effort level |
| `${CLAUDE_SKILL_DIR}` | Directory containing SKILL.md |

Shell-style quoting: `/my-skill "hello world" second` makes `$0` = "hello world", `$1` = "second".

### 1.5 Invocation Control

| Config | User invokes | Claude invokes | Context cost |
|:-------|:-------------|:---------------|:-------------|
| (default) | Yes | Yes | Description always loaded |
| `disable-model-invocation: true` | Yes | No | Zero until invoked |
| `user-invocable: false` | No | Yes | Description always loaded |

Key insight: `disable-model-invocation: true` removes the skill from Claude's context entirely until the user types the command. This is the right choice for gdev operations with side effects (init, setup, enable/disable).

### 1.6 allowed-tools

Grants permission for listed tools *while the skill is active*. Does not restrict other tools. Permission settings still govern unlisted tools.

Pattern for CLI wrappers:
```yaml
allowed-tools: Bash(gdev *) Bash(git status *) Read Grep Glob
```

This means Claude can run any `gdev` subcommand without per-use approval when the skill is active.

### 1.7 Skill Content Lifecycle

Once invoked, SKILL.md content enters the conversation as a single message and stays for the rest of the session. Auto-compaction re-attaches up to 5,000 tokens per skill, with a combined 25,000-token budget across all active skills. Most recently invoked skills get priority.

Implication: Keep SKILL.md bodies concise. Move detailed reference material to supporting files that Claude reads on demand.

### 1.8 Supporting Files

Reference from SKILL.md so Claude knows what to load when needed:

```markdown
## Reference
- For ecosystem-specific configs, see [ecosystem-reference.md](ecosystem-reference.md)
- For security hardening options, see [security-options.md](security-options.md)
```

Use `${CLAUDE_SKILL_DIR}` in shell commands to reference bundled scripts:
```markdown
Run the validation script:
```bash
bash ${CLAUDE_SKILL_DIR}/scripts/validate.sh $ARGUMENTS
```

---

## 2. Legacy Command Format (.claude/commands/)

### 2.1 Format

Commands are single markdown files in `.claude/commands/`. The filename (minus `.md`) becomes the command name.

```markdown
---
allowed-tools: Read, Grep, Glob
description: Run security vulnerability scan
---

Analyze the codebase for security vulnerabilities...
```

### 2.2 Relationship to Skills

Commands and skills are functionally merged. A file at `.claude/commands/deploy.md` and a skill at `.claude/skills/deploy/SKILL.md` both create `/deploy`. If both exist, the skill takes precedence. The CLI supports both formats, but skills are recommended because they support:

- Directory for supporting files (scripts, templates, examples)
- All frontmatter fields (context: fork, agent, hooks, paths, etc.)
- Autonomous invocation by Claude (when `disable-model-invocation` is not set)
- Live change detection during sessions

### 2.3 Recommendation for gdev

**Use skills, not commands.** gdev skills benefit from supporting files (ecosystem reference docs, validation scripts, template files). The directory structure also matches gdev's embed.FS pattern -- skills can be embedded in the gdev binary and extracted to `.claude/skills/` during `qsdev init`.

---

## 3. Recommended gdev Operations to Expose

### 3.1 Operation Mapping

| Operation | Skill Name | Side Effects | Invocation | Rationale |
|:----------|:-----------|:-------------|:-----------|:----------|
| Initialize project | `/gdev-init` | Creates files | User-only | Major side effect; user should control when new configs are generated |
| Onboard existing project | `/gdev-onboard` | Creates/modifies files | User-only | Detects existing state, fills gaps -- needs user confirmation |
| Run health check | `/gdev-doctor` | None (read-only) | Both | Safe read-only operation; Claude can check health autonomously when troubleshooting |
| Install prerequisites | `/gdev-setup` | Installs packages | User-only | System-level changes need explicit user intent |
| Enable tool | `/gdev-enable` | Modifies configs | User-only | Adds security tooling; user should choose when |
| Disable tool | `/gdev-disable` | Modifies configs | User-only | Removes tooling; user should choose when |
| Check tool status | `/gdev-status` | None (read-only) | Both | Safe; Claude can check what's enabled autonomously |
| List available tools | `/gdev-tools` | None (read-only) | Both | Safe; Claude can discover what's available |
| Generate compliance report | `/gdev-compliance` | Creates report file | User-only | Generates artifacts; user should control when |
| Update configs after changes | `/gdev-update` | Modifies configs | User-only | Modifies existing config files; needs user confirmation |

### 3.2 Detailed Skill Designs

#### `/gdev-init` -- Initialize New Project

```yaml
---
name: gdev-init
description: Initialize a new project with gdev. Detects ecosystem, generates security-hardened devenv configs, Claude Code settings, and pre-commit hooks. Use when setting up a new repo for development.
disable-model-invocation: true
allowed-tools: Bash(gdev *) Read Grep Glob
argument-hint: [--profile <name>] [--yes]
---

## Current project state
!`qsdev devenv doctor --json 2>/dev/null || echo '{"installed": false}'`

## Current directory
!`ls -la`

## Detected ecosystems
!`qsdev detect --json 2>/dev/null || echo '{"ecosystems": []}'`

## Instructions

Initialize this project with gdev. Based on the detected state above:

1. If gdev is not installed, tell the user to install it first and stop.
2. If ecosystems are detected, confirm the detection with the user.
3. Run `qsdev init` with appropriate flags:
   - If $ARGUMENTS includes --yes or --profile, pass them through
   - Otherwise, use `qsdev init --non-interactive` with detected ecosystems
   - If the user specified a profile (e.g., "set up for Python with full security"), map that to the right flags
4. After init completes, run `qsdev devenv doctor` to verify everything is healthy.
5. Summarize what was created and any manual steps needed.

If the user says something like "set up this repo for Python development with full security":
- Run: `qsdev init --ecosystem python --profile security-full --non-interactive`

If they say "initialize for a TypeScript/Node project":
- Run: `qsdev init --ecosystem javascript-typescript --non-interactive`
```

#### `/gdev-doctor` -- Health Check

```yaml
---
name: gdev-doctor
description: Run gdev health checks on the current project. Reports missing prerequisites, configuration issues, and security gaps. Use when diagnosing development environment problems or verifying setup.
allowed-tools: Bash(gdev *) Read
---

## System health
!`qsdev devenv doctor --json 2>/dev/null || echo '{"error": "gdev not found"}'`

## Instructions

Analyze the health check results above and:

1. Report any FAIL or WARN items clearly
2. For each issue, explain what it means and how to fix it
3. If gdev is not installed, provide installation instructions
4. If all checks pass, confirm the environment is healthy
5. Suggest next actions if there are unresolved issues

For specific diagnostic requests ($ARGUMENTS), focus on those areas.
```

#### `/gdev-enable` -- Enable a Tool

```yaml
---
name: gdev-enable
description: Enable a security or development tool in the current gdev-managed project. Adds tool configuration, updates shared config files, and verifies the tool is working.
disable-model-invocation: true
allowed-tools: Bash(gdev *) Read Grep
argument-hint: <tool-name>
---

## Currently enabled tools
!`qsdev status --json 2>/dev/null || echo '{"tools": []}'`

## Available tools
!`qsdev list --json 2>/dev/null || echo '{"available": []}'`

## Instructions

Enable the tool: $ARGUMENTS

1. Check if the tool is already enabled (from status above)
2. Check if it's in the available list
3. If valid, run: `qsdev enable $ARGUMENTS`
4. Verify with: `qsdev status $ARGUMENTS`
5. Report what changed (files modified, configs updated)
6. Note any additional setup steps the user needs to take

If $ARGUMENTS is empty, show the available tools list and ask what to enable.
```

#### `/gdev-disable` -- Disable a Tool

```yaml
---
name: gdev-disable
description: Disable and cleanly remove a tool from the current gdev-managed project. Removes configuration, cleans up shared config files, and verifies removal.
disable-model-invocation: true
allowed-tools: Bash(gdev *) Read Grep
argument-hint: <tool-name>
---

## Currently enabled tools
!`qsdev status --json 2>/dev/null || echo '{"tools": []}'`

## Instructions

Disable the tool: $ARGUMENTS

1. Check if the tool is currently enabled
2. If enabled, run: `qsdev disable $ARGUMENTS`
3. Verify with: `qsdev status`
4. Report what changed (files modified, configs cleaned up)
5. Note if any other tools depended on the disabled tool

If $ARGUMENTS is empty, show currently enabled tools and ask what to disable.
```

#### `/gdev-onboard` -- Onboard Existing Project

```yaml
---
name: gdev-onboard
description: Onboard an existing project to gdev management. Detects what's already configured, identifies gaps in security hardening, and fills them without overwriting existing customizations.
disable-model-invocation: true
allowed-tools: Bash(gdev *) Read Grep Glob
---

## Project analysis
!`qsdev devenv doctor --json 2>/dev/null || echo '{"installed": false}'`

## Existing configuration files
```!
ls -la .envrc devenv.yaml devenv.nix .pre-commit-config.yaml .claude/settings.json CLAUDE.md 2>/dev/null || echo "No existing configs found"
```

## Detected ecosystems
!`qsdev detect --json 2>/dev/null || echo '{"ecosystems": []}'`

## Instructions

Onboard this existing project to gdev management:

1. Analyze what's already configured vs what's missing
2. Present a gap analysis to the user:
   - What security hardening is already in place
   - What's missing or could be improved
   - What gdev would add/modify
3. Ask the user to confirm before making changes
4. Run: `qsdev init --merge` to fill gaps without overwriting
5. Run `qsdev devenv doctor` to verify the result
6. Summarize what was added and what was preserved
```

#### `/gdev-status` -- Check Tool Status (Model-Invocable)

```yaml
---
name: gdev-status
description: Show the current qsdev configuration status. Lists enabled tools, detected ecosystems, security posture, and configuration health. Use when the user asks about their development environment state or when troubleshooting.
allowed-tools: Bash(gdev *) Read
---

## Current status
!`qsdev status --json 2>/dev/null || echo '{"error": "gdev not found"}'`

## Instructions

Present the qsdev status clearly:
- Enabled tools and their health
- Detected ecosystems
- Security posture summary
- Any warnings or issues

If $ARGUMENTS specifies a particular tool or area, focus on that.
```

#### `/gdev-compliance` -- Generate Compliance Report

```yaml
---
name: gdev-compliance
description: Generate a security compliance report for the current project. Checks all security hardening layers and produces a structured report.
disable-model-invocation: true
allowed-tools: Bash(gdev *) Read Write
argument-hint: [--format json|markdown|sarif]
---

## Current security posture
!`qsdev devenv doctor --json 2>/dev/null`

## Instructions

Generate a compliance report:

1. Run: `qsdev report $ARGUMENTS`
2. Read the generated report
3. Summarize key findings:
   - Overall compliance score
   - Passing checks
   - Failing checks with remediation steps
   - Recommendations for improvement
4. If specific format requested via $ARGUMENTS, ensure it's passed through
```

#### `/gdev-update` -- Update Configs After Changes

```yaml
---
name: gdev-update
description: Update gdev-managed configuration files after project changes. Re-detects ecosystems, updates templates, and merges changes while preserving user customizations.
disable-model-invocation: true
allowed-tools: Bash(gdev *) Read Grep
---

## Current state
!`qsdev status --json 2>/dev/null`

## Recent changes
!`git diff --name-only HEAD~5 2>/dev/null || echo "Not a git repo or no recent commits"`

## Instructions

Update qsdev configuration:

1. Show what has changed since last qsdev update
2. Run: `qsdev init --update`
3. Report what was updated:
   - Files regenerated
   - Files preserved (user-modified)
   - New ecosystems detected
   - New tools available
4. Run `qsdev devenv doctor` to verify health after update
```

### 3.3 Design Rationale

**Why `disable-model-invocation: true` for most operations**: gdev operations modify configuration files, install packages, or generate artifacts. These are side effects that the user should explicitly request. Claude should not autonomously run `qsdev init` because a conversation topic seems related to project setup.

**Why allow Claude to invoke `gdev-doctor` and `gdev-status`**: These are read-only diagnostic operations. When Claude is troubleshooting a build failure or answering questions about the environment, it should be able to check health autonomously without the user needing to type `/gdev-doctor`.

**Why dynamic context injection (`!`command``)**: By running `qsdev devenv doctor --json` and `qsdev status --json` *before* Claude sees the skill content, Claude receives the actual project state as structured data. This is dramatically more efficient than having Claude run discovery commands one by one, and it ensures Claude has the full picture before it starts reasoning.

**Why JSON output from gdev**: Structured output (via `--json` flags) gives Claude parseable data rather than human-formatted tables. This is more reliable and more token-efficient. All gdev commands should support `--json` output.

---

## 4. CLI Wrapper Patterns from the Ecosystem

### 4.1 Pattern: Terraform Skill (Knowledge + CLI)

The Terraform skill (`antonbabenko/terraform-skill`) represents the "knowledge-heavy" pattern:
- Embeds Terraform best practices, testing frameworks, module patterns, CI/CD workflows
- Describes *how* to use the CLI correctly rather than wrapping specific commands
- Lets Claude figure out the exact commands based on the knowledge

This works for complex tools where the challenge is knowing *what* to do, not *how* to invoke the CLI.

### 4.2 Pattern: DevOps Generator/Validator Pairs

The devops-claude-skills pattern uses paired skills:
- **Generator**: Creates configurations (Dockerfile, Kubernetes manifests, Terraform modules)
- **Validator**: Runs a linter/checker on the output (hadolint, kubeval, tflint)
- Generator automatically invokes validator on its output

This is relevant for gdev because `qsdev init` generates configs and `qsdev devenv doctor` validates them -- a natural generator/validator pair.

### 4.3 Pattern: Dynamic State Injection

The PR summary skill from official docs demonstrates the strongest CLI wrapper pattern:

```yaml
---
name: pr-summary
context: fork
agent: Explore
allowed-tools: Bash(gh *)
---

## Pull request context
- PR diff: !`gh pr diff`
- PR comments: !`gh pr view --comments`
- Changed files: !`gh pr diff --name-only`

## Your task
Summarize this pull request...
```

This pattern -- inject live CLI state before Claude reasons -- is exactly what gdev skills should use.

### 4.4 Pattern: Fix-Issue Workflow

```yaml
---
name: fix-issue
description: Fix a GitHub issue
disable-model-invocation: true
---

Fix GitHub issue $ARGUMENTS following our coding standards.
1. Read the issue description
2. Understand the requirements
3. Implement the fix
4. Write tests
5. Create a commit
6. Push and create a PR
```

This pattern shows how a skill orchestrates a multi-step workflow where the CLI tool (`gh`) is one step among many. gdev's `/gdev-init` follows this pattern: detect, configure, verify, report.

### 4.5 Pattern: Codebase Visualizer (Script Bundling)

The official codebase-visualizer skill bundles a Python script:

```yaml
---
allowed-tools: Bash(python3 *)
---

Run the visualization script:
```bash
python3 ${CLAUDE_SKILL_DIR}/scripts/visualize.py .
```

This is relevant for gdev skills that need helper scripts -- validation logic, report formatting, or state inspection scripts could be bundled and referenced via `${CLAUDE_SKILL_DIR}`.

### 4.6 Anti-Pattern: Knowledge-Only Skills for CLI Tools

Skills that only describe a CLI tool's capabilities without injecting live state or providing specific commands are less effective. Claude already has general knowledge of most popular CLI tools. The value of a skill is in:
1. **Dynamic state** -- inject current project/system state
2. **Workflow orchestration** -- multi-step sequences with error handling
3. **Permission pre-approval** -- `allowed-tools` so Claude can execute without prompts
4. **Organization-specific knowledge** -- profiles, conventions, infrastructure details

---

## 5. CLAUDE.md Integration Design

### 5.1 What Goes in CLAUDE.md vs Skills

| Content | Location | Rationale |
|:--------|:---------|:----------|
| gdev is available, what it does | CLAUDE.md | Always-loaded context so Claude knows gdev exists |
| Available gdev commands | CLAUDE.md | Quick reference for Claude to know what's possible |
| Detailed workflow instructions | Skills | Loaded on demand, avoids bloating every-session context |
| Ecosystem-specific configs | Skill supporting files | Loaded only when relevant |
| Security policies | CLAUDE.md | Must be enforced in every session |
| Tool enable/disable procedures | Skills | Detailed steps only needed when performing the action |

### 5.2 Recommended CLAUDE.md Section

```markdown
## Development Environment (gdev)

This project is managed by gdev. Run `qsdev devenv doctor` to check environment health.

### Available commands
- `qsdev init` — Initialize or re-initialize project configuration
- `qsdev devenv doctor` — Check system and project health
- `qsdev devenv setup` — Install missing prerequisites
- `qsdev enable <tool>` — Enable a security/development tool
- `qsdev disable <tool>` — Disable a tool
- `qsdev status` — Show current configuration state
- `qsdev list` — Show available tools

### Skills available
- `/gdev-init` — Full project initialization workflow
- `/gdev-doctor` — Health check with analysis
- `/gdev-onboard` — Onboard existing project
- `/gdev-enable <tool>` — Enable a tool with verification
- `/gdev-disable <tool>` — Disable a tool cleanly
- `/gdev-status` — Environment status check
- `/gdev-compliance` — Generate compliance report
- `/gdev-update` — Update configs after changes

### Security policy
- Always use `qsdev enable` to add security tools, never configure them manually
- Run `qsdev devenv doctor` after any configuration changes
- Package installations go through gdev's security pipeline (age-gating, vuln scanning)
```

### 5.3 Section Markers for Safe Updates

gdev should own a specific section in CLAUDE.md using markers, so `qsdev init --update` can update it without clobbering user content:

```markdown
<!-- gdev:start -->
## Development Environment (gdev)
...content generated by gdev...
<!-- gdev:end -->
```

This matches gdev's existing migration strategy design (SHA256 hash tracking + section markers).

### 5.4 @-Import Pattern

For larger gdev documentation, use CLAUDE.md imports:

```markdown
## Development Environment
@.claude/gdev-reference.md
```

Where `.claude/gdev-reference.md` is a gdev-generated file with full command documentation, current tool status, and project-specific information. This keeps CLAUDE.md lean while making full docs available.

---

## 6. Safety Considerations

### 6.1 Operation Classification

| Category | Operations | Risk | Model | Autonomous? |
|:---------|:-----------|:-----|:------|:------------|
| Read-only diagnostics | doctor, status, list, detect | None | Always safe | Yes |
| Config generation (new) | init, setup | Medium -- creates new files | User-confirmed | No |
| Config modification | enable, disable, update | Medium -- modifies existing files | User-confirmed | No |
| System installation | setup (installing packages) | High -- system-level changes | User-confirmed | No |
| Report generation | compliance, report | Low -- creates output files | User-confirmed | No (artifacts) |
| Destructive operations | (none in gdev currently) | N/A | N/A | N/A |

### 6.2 Permission Model

**Skills with `disable-model-invocation: true`**:
- User must explicitly type `/gdev-init`, `/gdev-enable`, etc.
- Claude cannot autonomously decide to run these
- This is the primary safety gate for side-effect operations

**Skills with `allowed-tools: Bash(gdev *)`**:
- When the skill is active (invoked by the user), Claude can run gdev commands without per-use permission prompts
- This is safe because the user explicitly invoked the skill
- The `Bash(gdev *)` pattern is scoped -- Claude can't run arbitrary bash commands, only gdev subcommands

**Read-only skills (doctor, status)**:
- Enabled for Claude auto-invocation
- Claude can check health when troubleshooting without user intervention
- These have no side effects, so autonomous invocation is safe

### 6.3 Layered Safety Architecture

1. **Skill-level**: `disable-model-invocation: true` prevents autonomous triggering of side-effect operations
2. **Tool-level**: `allowed-tools: Bash(gdev *)` scopes what commands Claude can run
3. **gdev-level**: gdev's own `--non-interactive` and `--dry-run` flags provide a safety net within the tool itself
4. **Permission-level**: Claude Code's permission system (deny rules, auto mode classifier) provides a backstop
5. **Hooks-level**: PreToolUse hooks can intercept and validate gdev commands before execution (for enterprise deployments via managed settings)

### 6.4 Specific Safety Patterns

**Dry-run before execute**: For init and update operations, skills should first run with `--dry-run` and show the user what would change before executing:

```markdown
1. Run: `qsdev init --dry-run --json`
2. Show the user what files would be created/modified
3. Ask for confirmation
4. Run: `qsdev init` (actual execution)
```

**Idempotency**: gdev operations should be idempotent. Running `qsdev init` on an already-initialized project should be safe (detect existing state, skip what's already done). This reduces the risk of accidental re-runs.

**Reversibility**: `qsdev enable/disable` should be clean round-trips. Enabling and then disabling a tool should leave the project in its original state. This makes experimentation safe.

**Explicit confirmation for system-level changes**: `qsdev devenv setup` installs system packages. Even though the user invoked `/gdev-setup`, the skill should confirm before installing:

```markdown
1. Run: `qsdev devenv setup --dry-run`
2. Show: "The following packages will be installed: ..."
3. Ask: "Proceed? (This will use sudo to install system packages)"
4. If confirmed: `qsdev devenv setup`
```

### 6.5 Enterprise Safety via Managed Settings

For organizations deploying gdev at scale, managed settings can enforce additional guardrails:

```json
{
  "hooks": {
    "PreToolUse": [{
      "matcher": "Bash",
      "hooks": [{
        "type": "command",
        "command": "gdev hook-validate \"$TOOL_INPUT\"",
        "timeout": 5
      }]
    }]
  }
}
```

This hooks into Claude Code's PreToolUse system to validate all shell commands against gdev's security policy before execution -- independent of whether the user invoked a gdev skill.

### 6.6 What Should Never Be Autonomous

- Installing system packages (`qsdev devenv setup`)
- First-time initialization (`qsdev init` on a fresh project)
- Disabling security tools (`qsdev disable` for security-related tools)
- Generating compliance reports (creates artifacts with potential audit implications)
- Modifying managed/enterprise settings

### 6.7 What Can Safely Be Autonomous

- Health checks (`qsdev devenv doctor`)
- Status queries (`qsdev status`, `qsdev list`)
- Ecosystem detection (`qsdev detect`)
- Reading existing configuration files
- Suggesting remediation steps (without executing them)

---

## 7. Implementation Recommendations

### 7.1 Embedding Skills in gdev Binary

gdev should embed skill files via Go's `embed.FS` and extract them during `qsdev init`:

```go
//go:embed skills/*
var skillFiles embed.FS

func deploySkills(projectDir string) error {
    skillsDir := filepath.Join(projectDir, ".claude", "skills")
    return extractEmbed(skillFiles, "skills", skillsDir)
}
```

This ensures skills are versioned with gdev and updated when gdev is updated.

### 7.2 Skill Directory Layout in gdev Source

```
internal/claudecode/skills/
├── gdev-init/
│   ├── SKILL.md
│   └── ecosystem-reference.md
├── gdev-doctor/
│   └── SKILL.md
├── gdev-onboard/
│   └── SKILL.md
├── gdev-enable/
│   └── SKILL.md
├── gdev-disable/
│   └── SKILL.md
├── gdev-status/
│   └── SKILL.md
├── gdev-compliance/
│   └── SKILL.md
├── gdev-update/
│   └── SKILL.md
└── gdev-tools/
    └── SKILL.md
```

### 7.3 JSON Output Contract

Every gdev command should support `--json` for machine-readable output. This is critical for dynamic context injection:

```json
{
  "command": "doctor",
  "timestamp": "2026-05-12T14:00:00Z",
  "overall": "warn",
  "checks": [
    {"name": "devenv-installed", "status": "pass", "message": "devenv 1.5.2"},
    {"name": "nix-hardening", "status": "warn", "message": "sandbox not enabled"},
    {"name": "precommit-hooks", "status": "fail", "message": "no .pre-commit-config.yaml"}
  ]
}
```

### 7.4 Non-Interactive Mode

All gdev commands must support non-interactive execution for use within skills:
- `--non-interactive` or `--yes` to skip confirmation prompts
- `--answers-file` for complex wizard inputs
- `--profile` for pre-configured option sets
- Standard exit codes (0 success, 1 failure, 2 warning)

### 7.5 Integration with gdev's Existing Addon Architecture

From the implementation plan, gdev uses a three-addon model: `devenv`, `claudecode`, `devinit`. The skill deployment is a natural fit for the `claudecode` addon:

- `claudecode` addon's `Init()` method deploys skills to `.claude/skills/`
- `claudecode` addon's `Update()` method updates skills to match current gdev version
- `claudecode` addon manages the CLAUDE.md gdev section (with section markers)
- Skills are embedded in the gdev binary alongside other claudecode addon assets

### 7.6 Testing Skills

Skills can be tested by:
1. **Direct invocation**: `claude -p "/gdev-doctor"` in a test project
2. **Content rendering**: Verify dynamic context injection outputs are parseable
3. **Permission verification**: Ensure `allowed-tools` patterns match actual gdev commands
4. **Idempotency**: Run skills multiple times, verify no unintended changes
5. **Error handling**: Test with gdev not installed, broken configs, missing prerequisites

---

## 8. Comparison: Skills vs MCP Server vs Hooks

gdev could integrate with Claude Code through multiple mechanisms. Here's why skills are the right primary choice:

| Mechanism | Pros | Cons | gdev Fit |
|:----------|:-----|:-----|:---------|
| **Skills** | User-visible workflow, pre-approved tools, dynamic context, supports arguments, zero runtime dependency | Advisory (not deterministic), context cost | Primary interface for all gdev operations |
| **MCP Server** | Structured tool API, rich input/output schemas, persistent server process | Requires running server, more complex setup, less visible to users | Overkill for CLI wrapping; better for persistent services (databases, APIs) |
| **Hooks** | Deterministic, fires before/after every tool use, cannot be ignored | Cannot be invoked by user, no arguments, narrow trigger model | Security enforcement (validate commands), not workflow orchestration |
| **CLAUDE.md** | Always loaded, zero invocation cost | Advisory, bloats context if too long | Quick reference to gdev capabilities |
| **Deny rules** | Fast, deterministic glob matching | Cannot reason about context, binary allow/deny | Block dangerous commands (not gdev's job) |

**Recommended combination for gdev:**
1. **Skills** for all user-facing gdev operations (init, doctor, enable, etc.)
2. **CLAUDE.md section** for quick reference to available skills and security policy
3. **Hooks** (via managed settings) for enterprise enforcement of security policies
4. **Deny rules** for blocking dangerous package manager commands (already in gdev's claudecode addon)

---

## 9. Failure Modes and Edge Cases

### 9.1 gdev Not Installed

Skills using `!`qsdev ...`` for dynamic context injection will fail if gdev isn't installed. The pattern handles this:

```markdown
!`qsdev devenv doctor --json 2>/dev/null || echo '{"error": "gdev not installed"}'`
```

The `|| echo` fallback ensures Claude always gets parseable output and can handle the missing-tool case gracefully.

### 9.2 Stale Dynamic Context

Dynamic context is injected once when the skill is invoked. If the user makes changes during the conversation, the injected state is stale. Skills should instruct Claude to re-run commands when needed:

```markdown
If the user has made changes since the status was captured above, 
re-run `qsdev status --json` to get current state.
```

### 9.3 Long gdev Output Filling Context

Some gdev commands (e.g., `qsdev report`) may produce large outputs. For dynamic context injection, use `--json` with `jq` to extract only what's needed:

```markdown
!`qsdev devenv doctor --json 2>/dev/null | jq '{overall, checks: [.checks[] | select(.status != "pass")]}'`
```

### 9.4 Skill Description Budget Overflow

If the gdev project has many skills plus other third-party skills, description budgets may overflow. Keep descriptions concise and front-load the key use case:

```yaml
description: Run gdev health checks. Use when diagnosing dev environment problems.
```

Not:

```yaml
description: This skill runs the qsdev devenv doctor command to perform a comprehensive health check of your development environment, including checking for missing prerequisites, configuration issues, security hardening gaps, and more.
```

### 9.5 Compaction Dropping Skills

After context compaction, Claude retains only the 5,000 most recent tokens per skill, and the total budget is 25,000 tokens. If the user has invoked many skills, older gdev skills may be dropped. This is fine for task-specific skills (/gdev-init) but matters for reference skills. Keep reference content in CLAUDE.md (which survives compaction differently).

### 9.6 Permission Conflicts

If a user has deny rules that block `Bash(gdev *)` or `Bash` generally, gdev skills will fail. The `allowed-tools` in skills takes effect after workspace trust dialog acceptance, but cannot override explicit deny rules. Document this in gdev's troubleshooting guide.
