# Claude Code Environment Variables - Official Documentation
- **Source**: https://code.claude.com/docs/en/env-vars
- **Retrieved**: 2026-05-14

## Configuration & Settings Paths

| Variable | Description | Default | Controls |
|----------|-------------|---------|----------|
| `CLAUDE_CODE_PLUGIN_CACHE_DIR` | Override plugins root directory | `~/.claude/plugins` | Parent directory for marketplaces and plugin cache subdirectories |
| `CLAUDE_CODE_DEBUG_LOGS_DIR` | Override debug log file path | `~/.claude/debug/<session-id>.txt` | Debug logging output location (file path, not directory) |

## Memory & Context Files

| Variable | Description | Default | Controls |
|----------|-------------|---------|----------|
| `CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD` | Load memory files from `--add-dir` directories | Disabled | Loads `CLAUDE.md`, `.claude/CLAUDE.md`, `.claude/rules/*.md`, `CLAUDE.local.md` from additional directories |
| `CLAUDE_CODE_DISABLE_CLAUDE_MDS` | Prevent loading CLAUDE.md files | Disabled | Blocks all CLAUDE.md memory files (user, project, auto-memory) |
| `CLAUDE_CODE_DISABLE_AUTO_MEMORY` | Disable auto memory system | Disabled | Controls creation/loading of automatic memory files |

## Project Context & Code Discovery

| Variable | Description | Default | Controls |
|----------|-------------|---------|----------|
| `CLAUDE_CODE_GLOB_HIDDEN` | Include dotfiles in Glob results | `true` | Whether Glob tool includes hidden files |
| `CLAUDE_CODE_GLOB_NO_IGNORE` | Glob respects `.gitignore` patterns | `false` | `.gitignore` pattern filtering in Glob tool |
| `CLAUDE_CODE_GLOB_TIMEOUT_SECONDS` | Glob tool timeout | 20s (60s on WSL) | File discovery timeout duration |
| `CLAUDE_CODE_MAX_CONTEXT_TOKENS` | Override context window size | Model default | Context capacity assumption for auto-compaction |

## Hooks & Automation

| Variable | Description | Default | Controls |
|----------|-------------|---------|----------|
| `CLAUDE_CODE_SESSIONEND_HOOKS_TIMEOUT_MS` | SessionEnd hooks time budget | 1.5s (up to 60s) | Timeout for exit/`/clear`/session-switch hooks |
| `CLAUDECODE` | Detection flag in spawned shells | Not set in hooks/statusline | Set to `1` in Bash/tmux sessions spawned by Claude Code |

## Git & Workflow Integration

| Variable | Description | Default | Controls |
|----------|-------------|---------|----------|
| `CLAUDE_CODE_DISABLE_GIT_INSTRUCTIONS` | Remove git workflow instructions | Disabled | Removes commit/PR instructions and git status snapshot from system prompt |
| `CLAUDE_CODE_PERFORCE_MODE` | Enable Perforce-aware protection | Disabled | Prevents writes to files without owner-write bit |

## Skills & Plugin Management

| Variable | Description | Default | Controls |
|----------|-------------|---------|----------|
| `CLAUDE_CODE_DISABLE_POLICY_SKILLS` | Skip system-wide managed skills | Disabled | Prevents loading operator-provisioned skills from managed directory |
| `CLAUDE_CODE_PLUGIN_SEED_DIR` | Pre-populated plugin directories | None | Read-only plugin seed directories (`:` Unix, `;` Windows separated) |
| `CLAUDE_CODE_PLUGIN_GIT_TIMEOUT_MS` | Git timeout for plugin operations | 120000ms | Clone/update timeout for plugin repositories |
| `CLAUDE_CODE_PLUGIN_PREFER_HTTPS` | Clone GitHub plugins via HTTPS | Disabled | Uses HTTPS instead of SSH for GitHub plugin sources |
| `CLAUDE_CODE_PLUGIN_KEEP_MARKETPLACE_ON_FAILURE` | Retain cache on git failure | Disabled | Preserves existing marketplace cache when `git pull` fails |
| `CLAUDE_CODE_DISABLE_OFFICIAL_MARKETPLACE_AUTOINSTALL` | Skip auto-add official marketplace | Disabled | Prevents automatic addition of official plugin marketplace |

## Project Directory & Working Directory

| Variable | Description | Default | Controls |
|----------|-------------|---------|----------|
| `CLAUDE_CODE_BASH_MAINTAIN_PROJECT_WORKING_DIR` | Return to original directory after commands | Disabled | Returns to original working directory after each Bash command |
| `CLAUDE_CODE_HIDE_CWD` | Hide working directory in startup logo | Disabled | Obscures path in startup output |

## Managed Settings & Configuration

| Variable | Description | Default | Controls |
|----------|-------------|---------|----------|
| `CLAUDE_CODE_PROVIDER_MANAGED_BY_HOST` | Host manages provider routing | Not set | Ignores provider/endpoint/auth variables in settings files |

## Dynamic Behavior & Performance

| Variable | Description | Default | Controls |
|----------|-------------|---------|----------|
| `CLAUDE_CODE_AUTO_COMPACT_WINDOW` | Context capacity for auto-compaction | Model default | Decouples compaction threshold from full context window |
| `CLAUDE_CODE_AUTOCOMPACT_PCT_OVERRIDE` | Auto-compaction trigger percentage | ~95% | When context reaches this % of capacity, auto-compaction triggers |
| `CLAUDE_CODE_MAX_OUTPUT_TOKENS` | Maximum output tokens | Model default | Caps output tokens |
| `CLAUDE_CODE_EFFORT_LEVEL` | Effort level for supported models | Model default | `low`, `medium`, `high`, `xhigh`, `max`, or `auto` |

## Session & Environment Detection

| Variable | Description | Default | Controls |
|----------|-------------|---------|----------|
| `CLAUDE_CODE_REMOTE` | Cloud session detection flag | Not set in local | Auto-set to `true` in cloud sessions |
| `CLAUDE_CODE_REMOTE_SESSION_ID` | Cloud session ID | Not set in local | Auto-set in cloud sessions |
| `CLAUDE_CODE_SESSION_ID` | Current session ID | Auto-set | Available in Bash subprocesses |

## Remote Memory & Settings

| Variable | Description | Default | Controls |
|----------|-------------|---------|----------|
| `CLAUDE_CODE_REMOTE_SETTINGS_PATH` | Path to remote settings file | None | Settings file for remote/cloud contexts |
| `CLAUDE_CODE_REMOTE_MEMORY_DIR` | Shared/remote memory directory | None | Shared memory directory for remote contexts |

## Key Finding: No CLAUDE_HOME Override

There is NO environment variable to override the base config directory (`~/.claude/`). Configuration is scope-based:
- User scope: always `~/.claude/`
- Project scope: always `.claude/` in repository root
- Managed: system-level paths (platform-specific)
