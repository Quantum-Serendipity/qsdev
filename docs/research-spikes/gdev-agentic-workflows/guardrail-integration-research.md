# Guardrail-Workflow Integration: Security Without Blocking Legitimate Operations

## Executive Summary

The core tension: security guardrails (deny rules, hooks, sandbox restrictions) must not block legitimate agentic workflows. A skill that runs `npm test` must not be blocked by deny rules targeting `npm install`. An agent that reads files for security review must not be blocked by read-deny rules on sensitive directories. The solution is precision-scoped guardrails with explicit carve-outs for known workflow tools, combined with the architectural insight that skills can *grant* permissions via `allowed-tools` while hooks and deny rules *restrict* them -- and **the most restrictive answer always wins**.

This report builds directly on the completed `claude-code-agent-package-guardrails` spike's five-layer defense model, focusing specifically on how that model interacts with gdev's generated workflow skills and agents.

## 1. The Permission Interaction Model

### How Permissions Layer

```
Agent/Skill allowed-tools → Grants without prompting
settings.json allow rules → Grants without prompting
settings.json deny rules  → Blocks unconditionally
PreToolUse hooks          → Can block (exit 2) or allow (exit 0)
OS sandbox                → Restricts regardless of above
```

Critical rules:
1. **Deny always wins over allow**: A deny rule blocks even if `allowed-tools` would grant
2. **Hooks can block what allows permit**: Hook exit 2 blocks even with explicit allow
3. **Neither can override the other's deny**: Both are one-way valves
4. **allowed-tools only skips prompting**: It doesn't override deny rules

### Implication for gdev

gdev must ensure its deny rules don't accidentally block operations that its own skills need. This requires explicit coordination between generated deny rules and generated skill `allowed-tools`.

## 2. Common Conflict Points

### Conflict 1: Package Manager Deny Rules vs Build/Test Skills

**Deny rules** (from guardrails spike): Block `npm install *`, `pip install *`, etc.
**Skills**: `/refactor-safe` needs `Bash(npm test *)`, `/add-tests` needs `Bash(npm test *)`

**Resolution**: Deny rules use glob patterns. `Bash(npm install *)` does NOT match `Bash(npm test *)` because the pattern matches literally. No conflict exists for `npm test`, `npm run *`, or `npm audit`.

However, `Bash(npm *)` as a deny rule WOULD block test execution. gdev must never generate overly broad package manager deny rules.

**Safe deny rules that don't conflict with workflows**:
```json
{
  "deny": [
    "Bash(npm install *)",
    "Bash(npm uninstall *)",
    "Bash(npx *)",
    "Bash(yarn add *)",
    "Bash(pip install *)",
    "Bash(cargo install *)",
    "Bash(go install *)"
  ]
}
```

**Unsafe deny rules that would break workflows**:
```json
{
  "deny": [
    "Bash(npm *)",      // Blocks npm test, npm run, npm audit
    "Bash(pip *)",      // Blocks pip list, pip show
    "Bash(cargo *)"     // Blocks cargo test, cargo build, cargo clippy
  ]
}
```

### Conflict 2: Read-Deny Rules vs Security Review Agent

**Deny rules**: Block reading SSH keys, cloud credentials, etc.
**Agent**: `security-reviewer` needs to check if secrets are committed to the repo

**Resolution**: The security-reviewer agent should check for the *presence* of secret-like patterns in code files, not read actual secret files. Read-deny rules on `~/.ssh/**` etc. protect the *user's* secrets, not the *codebase's*.

For checking secrets in code, the agent uses `Grep` to find patterns:
```bash
grep -r "AWS_SECRET\|PRIVATE_KEY\|password\s*=" --include="*.{ts,js,py,go}" .
```

This searches file contents (allowed by default) without needing to read the user's actual SSH keys (blocked by deny rules). No conflict.

### Conflict 3: Sandbox vs Skill Scripts

**Sandbox**: Restricts writes to current directory
**Skills**: Scripts in `${CLAUDE_SKILL_DIR}/scripts/` may need to write temp files

**Resolution**: `${CLAUDE_SKILL_DIR}` resolves to within the project's `.claude/skills/` directory or `~/.claude/skills/`, both within the sandbox's write-allow zone (project root). Skill scripts should write output to the project directory, not to `/tmp` or other system locations.

### Conflict 4: Hooks vs Skill-Granted Permissions

**Hooks**: PreToolUse hooks that validate package installs
**Skills**: `/upgrade-dep` needs to install packages as part of its workflow

**Resolution**: The hook should recognize when the skill's workflow is executing. Two approaches:

**Approach A: Skill-scoped hooks**
The skill defines its own hooks in SKILL.md frontmatter that override the session hook during skill execution:

```yaml
---
name: upgrade-dep
hooks:
  PreToolUse:
    - matcher: "Bash"
      hooks:
        - type: command
          command: "${CLAUDE_SKILL_DIR}/scripts/validate-upgrade.sh"
---
```

This doesn't override session-level hooks -- both fire. But the skill-scoped hook can do additional validation specific to the upgrade workflow.

**Approach B: Hook recognizes workflow context**
The session-level package guardrail hook receives the full tool input as JSON. If the hook can detect that an upgrade workflow is in progress (e.g., by checking for a lock file or environment variable), it can apply different rules:

```bash
# In package-guardrail hook
if [ -f ".claude/upgrade-in-progress" ]; then
    # Allow installs during upgrade workflow
    exit 0
fi
# Normal validation...
```

**Approach C (Recommended): Don't bypass, enhance**
The `/upgrade-dep` skill doesn't bypass the guardrail hook. Instead, the hook validates the package being upgraded (checks for vulnerabilities, age, etc.) and either allows or blocks. The skill works *with* the guardrail, not around it:

```
/upgrade-dep lodash 4.18.0
→ Skill determines target package and version
→ Skill runs npm install lodash@4.18.0
→ PreToolUse hook fires, checks lodash@4.18.0 against OSV.dev
→ Hook allows (no vulnerabilities found)
→ Install proceeds
```

If the target version has a known vulnerability, the hook blocks the install and the skill reports the finding. This is the desired behavior.

## 3. gdev's Guardrail Generation Strategy

### Principle: Precision Over Coverage

Generate deny rules that target specific dangerous operations, not broad tool categories.

```go
// Good: precise deny rules that don't block workflow tools
var precisionDenyRules = map[string][]string{
    "npm": {
        "Bash(npm install *)",
        "Bash(npm uninstall *)",
        "Bash(npm link *)",
        "Bash(npm publish *)",
        "Bash(npx *)",
    },
    "pip": {
        "Bash(pip install *)",
        "Bash(pip uninstall *)",
    },
    "general": {
        "Bash(rm -rf /)",
        "Bash(rm -rf ~)",
        "Bash(chmod 777 *)",
        "Bash(curl * | sh)",
        "Bash(curl * | bash)",
        "Bash(wget * | sh)",
    },
}

// Bad: overly broad rules that block workflow tools
var broadDenyRules = []string{
    "Bash(npm *)",    // Blocks npm test!
    "Bash(pip *)",    // Blocks pip list!
    "Bash(rm *)",     // Blocks rm of test fixtures!
}
```

### Principle: Allow Rules for Known Workflow Operations

Generate explicit allow rules for operations that workflow skills need:

```json
{
  "allow": [
    "Bash(npm test *)",
    "Bash(npm run *)",
    "Bash(npm audit *)",
    "Bash(go test *)",
    "Bash(go build *)",
    "Bash(go vet *)",
    "Bash(pytest *)",
    "Bash(cargo test *)",
    "Bash(cargo build *)",
    "Bash(cargo clippy *)",
    "Bash(git *)",
    "Bash(gh *)",
    "Bash(make *)"
  ]
}
```

### Principle: Hooks for Nuanced Validation

Where deny rules are too blunt, use hooks:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "${CLAUDE_PROJECT_DIR}/.claude/hooks/validate-command.sh",
            "timeout": 10
          }
        ]
      }
    ]
  }
}
```

The hook can parse the command, extract the operation type, and make nuanced decisions:
- `npm install lodash` → validate against vulnerability DB → allow/block
- `npm test` → always allow
- `rm -rf node_modules` → allow (safe cleanup)
- `rm -rf /` → block

### Principle: Agent Tool Restrictions as Guardrails

Agent tool restrictions are a form of guardrail that's naturally compatible with workflows:

```yaml
---
name: security-reviewer
tools: Read, Grep, Glob, Bash
disallowedTools: Write, Edit
---
```

The security-reviewer agent literally cannot edit files. This is a stronger guarantee than a deny rule because it's enforced at the tool access level, not the command pattern level.

## 4. Testing the Integration

gdev should include a self-test that validates guardrail-workflow compatibility:

```go
func TestGuardrailWorkflowCompatibility(config Config) []Conflict {
    var conflicts []Conflict
    
    for _, skill := range config.EnabledSkills {
        for _, allowedTool := range skill.AllowedTools {
            for _, denyRule := range config.DenyPatterns {
                if globMatch(denyRule, allowedTool) {
                    conflicts = append(conflicts, Conflict{
                        Skill:    skill.Name,
                        Tool:     allowedTool,
                        DenyRule: denyRule,
                        Message:  fmt.Sprintf("Skill %q needs %q but deny rule %q would block it",
                            skill.Name, allowedTool, denyRule),
                    })
                }
            }
        }
    }
    
    return conflicts
}
```

This validation runs during `gdev init` and `gdev update` to catch conflicts before they cause runtime failures.

## 5. Managed Settings for Enterprise

For consulting firms deploying gdev across client projects, managed settings provide non-overridable guardrails:

```json
// /etc/claude-code/managed-settings.json (Linux)
{
  "permissions": {
    "deny": [
      "Bash(npm publish *)",
      "Bash(git push --force *)",
      "Bash(docker push *)"
    ]
  },
  "allowManagedPermissionRulesOnly": false,  // Allow project-level rules too
  "allowManagedHooksOnly": false,            // Allow project-level hooks too
  "disableBypassPermissionsMode": "disable"  // Block --dangerously-skip-permissions
}
```

This ensures firm-wide safety rules while allowing per-project customization via gdev.

## Depth Checklist

- [x] Underlying mechanism explained (permission layering, deny-wins-over-allow, hook interaction)
- [x] Key tradeoffs identified (precision vs coverage, broad deny vs targeted deny, hook bypass approaches)
- [x] Compared to alternatives (deny rules vs hooks vs agent tool restrictions vs OS sandbox)
- [x] Failure modes described (overly broad deny rules blocking test commands, sandbox blocking skill scripts)
- [x] Concrete examples found (specific deny rule patterns, conflict test code, managed settings example)
- [x] Standalone-readable
