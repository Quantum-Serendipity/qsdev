<!-- Source: https://code.claude.com/docs/en/permission-modes -->
<!-- Retrieved: 2026-05-12 -->

# Choose a permission mode - Claude Code Official Documentation

> Control whether Claude asks before editing files or running commands.

When Claude wants to edit a file, run a shell command, or make a network request, it pauses and asks you to approve the action. Permission modes control how often that pause happens.

## Available modes

| Mode                | What runs without asking                                                               | Best for                                |
| :------------------ | :------------------------------------------------------------------------------------- | :-------------------------------------- |
| `default`           | Reads only                                                                             | Getting started, sensitive work         |
| `acceptEdits`       | Reads, file edits, and common filesystem commands (`mkdir`, `touch`, `mv`, `cp`, etc.) | Iterating on code you're reviewing      |
| `plan`              | Reads only                                                                             | Exploring a codebase before changing it |
| `auto`              | Everything, with background safety checks                                              | Long tasks, reducing prompt fatigue     |
| `dontAsk`           | Only pre-approved tools                                                                | Locked-down CI and scripts              |
| `bypassPermissions` | Everything                                                                             | Isolated containers and VMs only        |

In every mode except `bypassPermissions`, writes to protected paths are never auto-approved.

Modes set the baseline. Layer permission rules on top to pre-approve or block specific tools in any mode except `bypassPermissions`, which skips the permission layer entirely.

## dontAsk mode

`dontAsk` mode auto-denies every tool call that would otherwise prompt. Only actions matching your `permissions.allow` rules and read-only Bash commands can execute; explicit `ask` rules are denied rather than prompting. This makes the mode fully non-interactive for CI pipelines or restricted environments where you pre-define exactly what Claude may do.

## auto mode

Auto mode lets Claude execute without permission prompts. A separate classifier model reviews actions before they run, blocking anything that escalates beyond your request, targets unrecognized infrastructure, or appears driven by hostile content Claude read.

### What the classifier blocks by default

**Blocked by default:**
* Downloading and executing code, like `curl | bash`
* Sending sensitive data to external endpoints
* Production deploys and migrations
* Mass deletion on cloud storage
* Granting IAM or repo permissions
* Modifying shared infrastructure
* Irreversibly destroying files that existed before the session
* Force push, or pushing directly to `main`

**Allowed by default:**
* Local file operations in your working directory
* Installing dependencies declared in your lock files or manifests
* Reading `.env` and sending credentials to their matching API
* Read-only HTTP requests
* Pushing to the branch you started on or one Claude created

### Auto mode drops broad allow rules on entry

On entering auto mode, broad allow rules that grant arbitrary code execution are dropped:
* Blanket `Bash(*)` or `PowerShell(*)`
* Wildcarded interpreters like `Bash(python*)`
* Package-manager run commands
* `Agent` allow rules

Narrow rules like `Bash(npm test)` carry over. Dropped rules are restored when you leave auto mode.

## bypassPermissions mode

`bypassPermissions` mode disables permission prompts and safety checks so tool calls execute immediately. Only `rm -rf /` and `rm -rf ~` still prompt as a circuit breaker.

Administrators can prevent this mode by setting `permissions.disableBypassPermissionsMode` to `"disable"` in managed settings.

## Protected paths

Writes to a small set of paths are never auto-approved, in every mode except `bypassPermissions`:

Protected directories: `.git`, `.vscode`, `.idea`, `.husky`, `.claude` (except `.claude/commands`, `.claude/agents`, `.claude/skills`, `.claude/worktrees`)

Protected files: `.gitconfig`, `.gitmodules`, `.bashrc`, `.bash_profile`, `.zshrc`, `.zprofile`, `.profile`, `.ripgreprc`, `.mcp.json`, `.claude.json`

## Switching modes

During a session: press `Shift+Tab` to cycle `default` -> `acceptEdits` -> `plan`.

At startup: `claude --permission-mode plan`

As a default in settings:
```json
{
  "permissions": {
    "defaultMode": "acceptEdits"
  }
}
```
