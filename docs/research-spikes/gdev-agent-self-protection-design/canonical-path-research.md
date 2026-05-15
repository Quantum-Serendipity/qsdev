# Canonical Path Resolution for gdev Hook Scripts

## Research Question

How should gdev canonicalize file paths to prevent symlink/traversal bypasses of path-based self-protection rules, and what are the limits of this approach for Bash tool commands?

## Executive Summary

Path canonicalization is the critical foundation of gdev's self-protection system. Every path-based rule (protecting `~/.claude/settings.json`, `~/.qsdev/hooks/`, audit trails, etc.) is only as strong as the canonicalization that normalizes paths before matching. This report catalogs 9 bypass technique categories, designs a two-tier canonicalization pipeline adapted from Prempti's approach, analyzes the feasibility of extracting write targets from arbitrary Bash commands, defines the path matching algorithm, and provides implementation pseudocode for both Bash and Go.

**Key finding**: Write/Edit tool paths can be reliably canonicalized (single `file_path` field, resolved via `realpath`). Bash tool commands cannot be reliably parsed for all write targets -- the problem is equivalent to static analysis of arbitrary shell scripts, which is undecidable in the general case. gdev must use a **dual strategy**: precise canonicalization for Write/Edit/Read hooks, and pattern-matching heuristics (regex + optional AST) for Bash hooks, accepting that Bash matching is a speed bump, not a wall.

---

## 1. Path Bypass Technique Catalog

### 1.1 Symlink Traversal

**Mechanism**: Create a symlink from an unprotected path to a protected file, then operate on the symlink.

```bash
# Agent creates symlink
ln -s ~/.claude/settings.json /tmp/innocent-config.json
# Agent edits via symlink -- path check sees /tmp/innocent-config.json
Edit: file_path="/tmp/innocent-config.json"
```

**gdev exposure**: HIGH. This is the most common bypass in real-world incidents. The Gemini CLI symlink vulnerability (issue #1121) demonstrated exactly this pattern: path validation on user-provided paths without resolving symlinks. CVE-2026-5061 (Consul-template) showed the same class of bug. The Ona research documented Claude Code autonomously discovering `/proc/self/root` as a symlink-like bypass.

**Defense**: `realpath` (no flags) resolves all symlinks in the path. `/tmp/innocent-config.json` resolves to `/home/user/.claude/settings.json`, which matches the protection rule. This is the primary reason canonicalization exists.

**Edge case -- dangling symlinks**: If the agent creates a symlink to a protected path that doesn't yet exist (e.g., a new file under `~/.qsdev/`), `realpath` without `-m` will fail. Must use the two-tier strategy: try `realpath` first, fall back to `realpath -m` for non-existent targets.

**Edge case -- recursive symlinks**: `realpath` handles these gracefully (returns ELOOP after kernel's symlink follow limit, typically 40 hops). Hook should treat ELOOP as a deny (suspicious).

### 1.2 Relative Path Manipulation

**Mechanism**: Use `../` sequences to traverse from a seemingly safe location to a protected path.

```bash
# If cwd is /home/user/project:
Edit: file_path="../../.claude/settings.json"
# Without canonicalization, regex matching /.claude/settings.json might miss this
```

**gdev exposure**: MEDIUM. Claude Code's Write/Edit tools typically provide absolute paths, but the contract does not guarantee this. Relative paths are more common in Bash tool commands.

**Defense**: `realpath` converts relative paths to absolute. Even `realpath -m` (lexical fallback) resolves `../` sequences. The combination of `realpath -m` handles both relative paths and non-existent targets.

**Edge case -- cwd symlinks**: If the working directory itself is a symlink, relative path resolution depends on whether `realpath` resolves the cwd symlink first. `realpath` (physical mode, default) resolves the full chain, including the cwd component. This is correct behavior.

### 1.3 Hardlink Creation

**Mechanism**: Create a hardlink to a protected file, then modify via the hardlink path.

```bash
# Agent creates hardlink
ln ~/.claude/settings.json /tmp/settings-hardlink.json
# Agent edits via hardlink
Edit: file_path="/tmp/settings-hardlink.json"
```

**gdev exposure**: LOW (with caveats). Unlike symlinks, `realpath` does NOT reveal the original path of a hardlink -- both paths resolve to themselves (they are equally valid names for the same inode). The hardlink `/tmp/settings-hardlink.json` resolves to `/tmp/settings-hardlink.json`, which does NOT match the protection pattern `~/.claude/settings.json`.

**However**: This is actually a more nuanced problem than it first appears:

1. **Hardlink creation itself can be blocked**: The agent must first run `ln ~/.claude/settings.json /tmp/target` via the Bash tool. The Bash guard's regex matching catches `ln` commands targeting protected paths.

2. **Linux `fs.protected_hardlinks` kernel parameter**: Modern Linux kernels (and NixOS default) set `fs.protected_hardlinks = 1`, which prevents creating hardlinks to files you don't own. Since `~/.claude/settings.json` is owned by the user, the agent (running as the same user) CAN create hardlinks to it. This kernel protection does not help here.

3. **Cross-filesystem limitation**: Hardlinks cannot span filesystem boundaries. If `~/.claude/` and `/tmp/` are on different filesystems (common -- `/tmp` is often tmpfs), the hardlink creation fails with EXDEV. This is a natural (but unreliable) defense.

4. **Inode-based detection (advanced)**: For a future Go-based implementation, the hook could `stat()` the target file and compare its inode number against known protected file inodes (cached at session start). If the inodes match, block regardless of path. This catches hardlink-based bypasses definitively.

**Defense strategy**: Block hardlink creation to protected paths via Bash guard (regex on `ln` targeting protected paths). For defense in depth, consider inode-based detection in Go implementation.

### 1.4 /proc/self/root Traversal

**Mechanism**: Use `/proc/self/root/` prefix to access the filesystem root, bypassing path prefix checks.

```bash
# Instead of ~/.claude/settings.json:
/proc/self/root/home/user/.claude/settings.json
```

**gdev exposure**: HIGH (documented real-world bypass). The Ona research showed Claude Code autonomously discovering this bypass vector. `/proc/self/root` is a symlink to `/`, so the path resolves identically to the direct path.

**Defense**: `realpath` resolves `/proc/self/root/home/user/.claude/settings.json` to `/home/user/.claude/settings.json` because it follows the `/proc/self/root -> /` symlink. Standard canonicalization handles this automatically.

**Variant -- /proc/PID/root**: The agent could use a specific PID instead of `self`: `/proc/1234/root/home/user/.claude/settings.json`. This also resolves via `realpath` as long as the process has read access to that `/proc/PID/root` entry (governed by ptrace access mode check).

### 1.5 /dev/fd/ and File Descriptor Tricks

**Mechanism**: Use file descriptor paths to access files through inherited or leaked descriptors.

```bash
# Open a protected file, then access via /dev/fd/N or /proc/self/fd/N
exec 3< ~/.claude/settings.json
cat /proc/self/fd/3  # reads settings.json via fd path
```

**gdev exposure**: LOW for Write/Edit tools, MEDIUM for Bash tool. Claude Code's Write and Edit tools take a `file_path` string, not a file descriptor. The agent cannot pass `/dev/fd/3` as a `file_path` to Write because Write opens the file by path, not by descriptor. However, the Bash tool can execute arbitrary commands that manipulate file descriptors.

**Defense**: For Write/Edit hooks, `/dev/fd/N` and `/proc/self/fd/N` paths can be detected by prefix matching before canonicalization (since `realpath` on `/proc/self/fd/N` resolves to the original file path, which is actually the desired behavior). For Bash hooks, file descriptor manipulation is extremely difficult to detect via command analysis. This falls into the "Bash analysis is a speed bump" category.

**Practical note**: The agent would need to first open the file descriptor (via a Bash tool call that IS visible to hooks) and then use it in a subsequent call. The first call can be caught by the Bash guard.

### 1.6 Case Sensitivity

**Mechanism**: On case-insensitive filesystems, use different casing to bypass string matching.

```bash
# On macOS HFS+/APFS (case-insensitive by default):
~/.Claude/Settings.json  # resolves to same file as ~/.claude/settings.json
```

**gdev exposure**: NONE on Linux (ext4/btrfs/ZFS are case-sensitive). LOW for cross-platform if gdev targets macOS in the future.

**Defense on Linux**: No action needed. `realpath` returns the exact case used on disk. On case-sensitive filesystems, `~/.Claude/Settings.json` is a genuinely different path that points to a nonexistent file.

**Defense on macOS (future)**: `realpath` on macOS resolves to the canonical casing stored on disk. So `realpath ~/.Claude/Settings.json` returns `~/.claude/settings.json` (the actual on-disk name) on HFS+/APFS. This means canonicalization automatically handles case sensitivity -- no additional work needed.

### 1.7 Unicode Normalization Attacks

**Mechanism**: Use different Unicode representations of the same visual characters to bypass string matching.

```bash
# NFC: .claude (U+002E U+0063 U+006C ...)
# NFD: .claude with combining characters
# NFKC: using fullwidth characters U+FF0E (fullwidth period) + ...
```

**gdev exposure**: VERY LOW on Linux for ASCII paths. All gdev-protected paths (`.claude/`, `.qsdev/`, `settings.json`) use only ASCII characters, which have no Unicode normalization variants. Unicode normalization is only relevant for user-chosen path components (e.g., a Korean home directory name, as seen in the moai-adk issue #342).

**Real-world impact**: The moai-adk project encountered this on macOS where the home directory contained Korean characters (`~/코딩/`). macOS stores filenames in NFD form, but other systems use NFC. String comparison failed because the byte sequences differ despite representing the same characters.

**Defense**: For gdev's protected paths (all ASCII), this is a non-issue. For future extensibility (user-defined protected paths), the Go implementation should normalize paths to NFC using `golang.org/x/text/unicode/norm` before comparison:

```go
import "golang.org/x/text/unicode/norm"
canonicalPath = norm.NFC.String(canonicalPath)
```

The Bash implementation does not need Unicode normalization because all protected paths are ASCII.

### 1.8 Race Conditions (TOCTOU)

**Mechanism**: Modify the filesystem between gdev's path check and Claude Code's file operation.

```bash
# Call 1: Create symlink pointing to safe file
Bash: ln -s /tmp/safe.txt /home/user/project/link.txt

# Call 2: Edit via symlink (hook resolves to /tmp/safe.txt -- allowed)
# But between hook check and actual Write, retarget symlink to settings.json
```

**gdev exposure**: LOW (architectural mitigation). Claude Code's tool execution model is serialized: the PreToolUse hook runs, returns a verdict, and then the tool executes. The agent can only act through tool calls, and tool calls are queued. The agent CANNOT retarget a symlink between the hook's `realpath` check and Claude Code's file write within a single tool call.

**The real TOCTOU risk**: The agent can retarget a symlink across separate tool calls:
1. Call 1 (Bash): `ln -s /tmp/safe.txt /home/user/project/link.txt` -- hook allows (not targeting protected paths)
2. Call 2 (Bash): Symlink already exists. Agent retargets: `ln -sf ~/.claude/settings.json /home/user/project/link.txt` -- Bash guard should catch the `ln` targeting a protected path
3. Call 3 (Edit): `file_path="/home/user/project/link.txt"` -- hook resolves via realpath to `~/.claude/settings.json` -- BLOCKED

The defense works here because canonicalization happens at call 3, when the symlink already points to the protected target. The hook sees the real destination.

**CVE-2026-5061 (Consul-template) counterexample**: Consul-template was vulnerable because it checked the path once and then re-read the file later without re-checking. gdev's hooks check at the moment of the tool call, not in advance. This is architecturally resistant to TOCTOU.

**Remaining risk**: An external process (not the agent) could retarget a symlink between the hook's `realpath` and Claude Code's actual file operation. This is a kernel-level race that cannot be prevented without `openat2(RESOLVE_NO_SYMLINKS)` or similar. The practical risk from external processes is negligible in gdev's threat model (the adversary is the agent, not a separate process).

### 1.9 Mount Namespace Tricks (Bind Mounts)

**Mechanism**: Use bind mounts to overlay a directory on top of protected paths, or to create alternative access paths to protected files.

```bash
# Agent bind-mounts an attacker-controlled directory over ~/.qsdev/hooks/
mount --bind /tmp/evil-hooks/ ~/.qsdev/hooks/
# Now all hook scripts read from /tmp/evil-hooks/ instead

# Or mount a writable overlay over a read-only path
```

**gdev exposure**: VERY LOW. Bind mount operations require `CAP_SYS_ADMIN` (root) or a user namespace with mount namespace. On NixOS, the developer user does not typically have these capabilities. Claude Code does not run in a privileged container.

**Defense**: This is out of scope for path canonicalization. `realpath` cannot detect bind mounts -- the mounted path resolves identically to a native path. Defense against bind mount attacks requires either:
- Restricting the agent from running `mount` commands (already covered by Claude Code's default deny rules and gdev's Bash guard)
- Using a sandbox that restricts mount namespace access (e.g., bubblewrap, which Claude Code already uses)

**Note**: If gdev ever runs inside a container (e.g., for CI), bind mount attacks become more relevant. The container runtime should restrict mount capabilities.

---

## 2. Canonicalization Pipeline Design

### 2.1 The Two-Tier Strategy

Adapted from Prempti's `resolve_file_path()` (documented in `docs/prempti-path-canonicalization-source.md`):

**Tier 1 -- Filesystem canonicalization**: Resolve symlinks and normalize path against the actual filesystem. This is the strongest form of canonicalization because it produces the true canonical path. Use when the path exists on disk.

**Tier 2 -- Lexical normalization**: Resolve `../` and `.` components, convert to absolute path, but do NOT resolve symlinks (because the path doesn't exist, so there are no symlinks to resolve). Use as fallback when Tier 1 fails (path doesn't exist).

### 2.2 Implementation: Bash

```bash
# canonicalize_path - Resolve a file path to its canonical form.
# Tier 1: realpath (resolves symlinks, requires path to exist)
# Tier 2: realpath -m (lexical normalization, works for non-existent paths)
#
# Usage: CANON=$(canonicalize_path "$raw_path")
canonicalize_path() {
    local raw_path="$1"
    local resolved

    # Empty path: return empty (caller must handle)
    if [[ -z "$raw_path" ]]; then
        echo ""
        return
    fi

    # Expand ~ to $HOME (Claude Code usually provides absolute paths,
    # but belt-and-suspenders)
    if [[ "$raw_path" == "~/"* ]]; then
        raw_path="$HOME/${raw_path#\~/}"
    elif [[ "$raw_path" == "~" ]]; then
        raw_path="$HOME"
    fi

    # Tier 1: Full filesystem canonicalization (resolves symlinks)
    resolved=$(realpath -- "$raw_path" 2>/dev/null)
    if [[ $? -eq 0 && -n "$resolved" ]]; then
        echo "$resolved"
        return
    fi

    # Tier 2: Lexical normalization (for non-existent paths)
    # realpath -m resolves existing symlinks in parent components,
    # then lexically normalizes the non-existent tail.
    resolved=$(realpath -m -- "$raw_path" 2>/dev/null)
    if [[ $? -eq 0 && -n "$resolved" ]]; then
        echo "$resolved"
        return
    fi

    # Tier 3: Last resort -- return the path after basic cleanup.
    # This should only happen if realpath itself is missing (unlikely
    # on NixOS since it's in coreutils).
    echo "$raw_path"
}
```

### 2.3 Implementation: Go

```go
package selfprotect

import (
    "os"
    "path/filepath"
    "strings"
)

// CanonicalizePath resolves a file path to its canonical form.
// Tier 1: filepath.EvalSymlinks (resolves symlinks, requires existing path)
// Tier 2: Resolve parent + lexical normalize leaf (for non-existent paths)
// Tier 3: filepath.Abs + filepath.Clean (pure lexical fallback)
func CanonicalizePath(rawPath string) string {
    if rawPath == "" {
        return ""
    }

    // Expand ~ to home directory
    if strings.HasPrefix(rawPath, "~/") {
        home, err := os.UserHomeDir()
        if err == nil {
            rawPath = filepath.Join(home, rawPath[2:])
        }
    }

    // Make absolute
    absPath, err := filepath.Abs(rawPath)
    if err != nil {
        return filepath.Clean(rawPath)
    }

    // Tier 1: Full filesystem canonicalization
    resolved, err := filepath.EvalSymlinks(absPath)
    if err == nil {
        return filepath.Clean(resolved)
    }

    // Tier 2: Resolve the parent directory (which may exist and contain
    // symlinks), then append the leaf component.
    // This handles the common case: writing a new file in an existing
    // directory that may itself be a symlink.
    dir := filepath.Dir(absPath)
    base := filepath.Base(absPath)
    resolvedDir, err := filepath.EvalSymlinks(dir)
    if err == nil {
        return filepath.Join(filepath.Clean(resolvedDir), base)
    }

    // Tier 3: Walk up the tree, resolving what exists
    // This handles deeply nested non-existent paths like
    // /existing/symlink-dir/new-subdir/new-file.txt
    parts := strings.Split(absPath, string(filepath.Separator))
    resolved = string(filepath.Separator)
    for i, part := range parts {
        if part == "" {
            continue
        }
        candidate := filepath.Join(resolved, part)
        evalCandidate, err := filepath.EvalSymlinks(candidate)
        if err != nil {
            // This component doesn't exist; append remaining
            // components lexically
            remaining := strings.Join(parts[i:], string(filepath.Separator))
            return filepath.Join(resolved, remaining)
        }
        resolved = evalCandidate
    }
    return resolved
}
```

### 2.4 Handling Non-Existent Paths

The Write tool creates files that don't exist yet. The path `/home/user/project/newdir/newfile.txt` may have no existing components beyond `/home/user/project/`. The canonicalization must handle this gracefully.

**Bash approach**: `realpath -m` handles this natively. The `-m` flag (canonicalize-missing) resolves symlinks in existing components and lexically normalizes non-existing components. This is the correct behavior for security: it catches symlinks in the existing portion of the path while still producing a usable canonical path.

**Go approach**: `filepath.EvalSymlinks` fails on non-existent paths. The Go implementation must walk up the path tree until it finds an existing component, resolve that component, then lexically append the remaining non-existent components. This is what Tier 2 and Tier 3 of the Go implementation above do.

**Critical security property**: The parent directory resolution is essential. Consider:
```
ln -s /home/user/.claude /home/user/project/safe-dir
Write: file_path="/home/user/project/safe-dir/settings.json"
```
Without parent resolution, the lexical fallback produces `/home/user/project/safe-dir/settings.json` (no match). With parent resolution, `safe-dir` resolves to `/home/user/.claude`, producing `/home/user/.claude/settings.json` (MATCH, BLOCKED).

### 2.5 Performance Considerations

**Cost of `realpath`**: On Linux, `realpath` issues one `lstat()` syscall per path component to check for symlinks. A typical path like `/home/user/.claude/settings.json` requires 4 `lstat()` calls. Each `lstat()` costs ~100-500ns on cached dentries (dentry cache hit) and ~1-10us on cold cache or networked filesystems.

**Total cost per hook invocation**: For a single path (Write/Edit hook), the canonicalization adds approximately 2-20us. For the Bash hook (which may extract and canonicalize multiple paths from the command), the cost is proportionally higher.

**Context**: Claude Code hook invocations involve process spawning (~3-10ms for bash), JSON parsing via `jq` (~5-15ms), and script execution. The `realpath` call (~0.02ms) is negligible compared to these costs. Even 10 `realpath` calls would add only 0.2ms.

**Caching**: The Linux kernel's dentry cache (dcache) caches path lookups. Repeated `realpath` calls on the same path within a session are effectively free after the first call. gdev hooks run per-tool-call (not persistent), so they cannot cache across invocations. However, the kernel cache provides the same benefit.

**Recommendation**: No performance optimization needed. `realpath` is fast enough for every hook invocation. The process-spawning overhead dominates.

---

## 3. Bash Tool Edge Cases: Write Target Extraction

### 3.1 The Problem

The agent uses the Bash tool to execute arbitrary shell commands. Unlike Write/Edit (which have a structured `file_path` field), Bash commands encode write targets in diverse ways:

```bash
# Direct redirections
echo "malicious" > /path/to/file
echo "append" >> /path/to/file
cat payload > /path/to/file
command 2> /path/to/error-log

# Multi-redirect
command > /path/stdout 2> /path/stderr
command &> /path/both

# Tee
echo "data" | tee /path/to/file
echo "data" | tee -a /path/to/file

# dd
dd if=/dev/zero of=/path/to/file bs=1 count=0

# In-place edit
sed -i 's/old/new/' /path/to/file
perl -pi -e 's/old/new/' /path/to/file

# Copy/move (creates or overwrites target)
cp source /path/to/target
mv source /path/to/target

# Programmatic write
python3 -c "open('/path/to/file', 'w').write('data')"
node -e "require('fs').writeFileSync('/path/to/file', 'data')"

# Here-document
cat > /path/to/file << 'EOF'
content
EOF

# Variable expansion
FILE="$HOME/.claude/settings.json"
echo "data" > "$FILE"

# Command substitution in path
echo "data" > "$(echo /home/user/.claude/settings.json)"

# Subshell
(echo "data" > /path/to/file)
bash -c 'echo "data" > /path/to/file'

# Eval
eval 'echo "data" > /path/to/file'

# Multi-command
cmd1 && echo "data" > /path/to/file
cmd1; echo "data" > /path/to/file
cmd1 || echo "data" > /path/to/file
```

### 3.2 Feasibility Assessment

**Can we reliably extract ALL write targets from arbitrary Bash commands?**

**No.** This is equivalent to static analysis of shell scripts, which is undecidable in the general case. Specific blocking examples:

1. **Variable expansion**: `echo > "$VAR"` -- the write target depends on the runtime value of `$VAR`, which is unknowable at analysis time.
2. **Command substitution**: `echo > "$(compute_path)"` -- the target depends on the output of another command.
3. **Eval/exec indirection**: `eval "$CMD"` -- the command itself is a runtime value.
4. **Encoded commands**: `echo "ZWNobyBkYXRhID4gL3BhdGg=" | base64 -d | bash` -- the command is obfuscated.
5. **Interpreter wrappers**: `python3 -c "..."`, `node -e "..."`, `ruby -e "..."` -- the write target is inside a different language's string.

**What CAN we extract?**

For commands with static, visible write targets, extraction is feasible at varying reliability levels:

| Pattern | Extractability | Method |
|---------|---------------|--------|
| `> /literal/path` | HIGH | Regex or AST: `file_redirect` destination |
| `>> /literal/path` | HIGH | Same as above |
| `tee /literal/path` | HIGH | Regex: path after `tee` command |
| `dd of=/literal/path` | HIGH | Regex: `of=` argument |
| `sed -i ... /literal/path` | HIGH | Regex: last argument(s) after `-i` |
| `cp/mv src /literal/path` | MEDIUM | Regex: last argument (but could be dir) |
| `> "$HOME/path"` | MEDIUM | Expand `$HOME` then match |
| `> ~/path` | MEDIUM | Expand `~` then match |
| `eval "echo > /path"` | LOW | Would need to parse the eval'd string |
| `python3 -c "..."` | NONE | Requires Python analysis |
| `> "$VAR"` (arbitrary var) | NONE | Runtime-dependent |

### 3.3 Strategy Recommendation: Layered Defense

Given that complete extraction is impossible, gdev should use three complementary strategies:

#### Strategy A: Regex Pattern Matching (Bash script, Phase 32)

Match literal path strings appearing anywhere in the command, combined with write-operation indicators. This catches the common cases.

```bash
# Extract all potential write targets from the command
# Strategy: look for protected path strings appearing near write operators

COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command // ""')

# Canonicalize HOME for matching
REAL_HOME=$(realpath -m "$HOME" 2>/dev/null || echo "$HOME")

# Protected path patterns (after canonicalization of HOME)
PROTECTED_PATTERNS=(
    "$REAL_HOME/.claude/settings.json"
    "$REAL_HOME/.claude/settings.local.json"
    "$REAL_HOME/.claude/policy-limits.json"
    "$REAL_HOME/.qsdev/"
    "$REAL_HOME/.config/nix/nix.conf"
)

# --- Check 1: Literal path string in command ---
# Expand ~ and $HOME in the command for matching
EXPANDED_CMD="$COMMAND"
EXPANDED_CMD="${EXPANDED_CMD//\~\//$REAL_HOME/}"
EXPANDED_CMD="${EXPANDED_CMD//\$HOME\//$REAL_HOME/}"
EXPANDED_CMD="${EXPANDED_CMD//\$\{HOME\}\//$REAL_HOME/}"

for pattern in "${PROTECTED_PATTERNS[@]}"; do
    if [[ "$EXPANDED_CMD" == *"$pattern"* ]]; then
        # Path appears in command -- check if it's a write context
        # Write indicators: >, >>, tee, dd of=, sed -i, cp, mv, rm,
        # chmod, cat >, echo >, perl -pi, truncate, shred
        if echo "$COMMAND" | grep -qE \
            '(>|>>|tee\s|dd\s.*of=|sed\s+-i|cp\s|mv\s|rm\s|chmod\s|chown\s|truncate\s|shred\s|perl\s+-pi|cat\s.*>|echo\s.*>|printf\s.*>)'; then
            echo "BLOCKED by gdev self-protection: Bash command targets protected path" >&2
            echo "  Pattern matched: $pattern" >&2
            exit 2
        fi
    fi
done

# --- Check 2: Redirect targets ---
# Extract paths from redirect operators using regex
# This catches: > /path, >> /path, 2> /path, &> /path
REDIRECT_TARGETS=$(echo "$COMMAND" | grep -oE '[0-9]*>{1,2}\s*[^ ;|&]+' | \
    sed 's/^[0-9]*>*\s*//')

for target in $REDIRECT_TARGETS; do
    # Expand ~ and $HOME
    target="${target//\~\//$REAL_HOME/}"
    target="${target//\$HOME\//$REAL_HOME/}"
    target="${target//\$\{HOME\}\//$REAL_HOME/}"
    # Canonicalize
    canon_target=$(canonicalize_path "$target")
    for pattern in "${PROTECTED_PATTERNS[@]}"; do
        if [[ "$canon_target" == "$pattern"* ]]; then
            echo "BLOCKED by gdev self-protection: redirect targets protected path" >&2
            echo "  Resolved target: $canon_target" >&2
            exit 2
        fi
    done
done
```

**Known bypasses of Strategy A**:
- Variable expansion: `F=~/.claude/settings.json; echo > "$F"` (variable set in the same command)
- Encoding: `echo "base64-payload" | base64 -d | bash`
- Python/node/ruby wrappers
- eval/exec indirection
- String concatenation: `echo > /home/user/.clau"de/setti"ngs.json`

#### Strategy B: AST-Based Analysis (Go binary, future)

For the Go implementation, use a tree-sitter-bash parser to build an AST and walk it for `file_redirect` and `redirected_statement` nodes. This is more accurate than regex because it respects shell quoting and nesting.

```go
// Pseudocode for Go AST-based redirect extraction
func ExtractWriteTargets(command string) []string {
    tree := treesitter.Parse(command, bashLanguage)
    targets := []string{}

    // Walk all file_redirect nodes
    query := `(file_redirect destination: (_) @target)`
    matches := treesitter.QueryMatches(tree, query)
    for _, match := range matches {
        target := match.CaptureText("target")
        targets = append(targets, target)
    }

    // Walk command nodes for known write commands
    cmdQuery := `(command name: (command_name) @cmd
                  argument: (_) @arg)`
    cmdMatches := treesitter.QueryMatches(tree, cmdQuery)
    for _, match := range cmdMatches {
        cmd := match.CaptureText("cmd")
        args := match.CaptureTexts("arg")
        switch cmd {
        case "tee":
            // tee's positional args (excluding flags) are write targets
            for _, arg := range args {
                if !strings.HasPrefix(arg, "-") {
                    targets = append(targets, arg)
                }
            }
        case "dd":
            for _, arg := range args {
                if strings.HasPrefix(arg, "of=") {
                    targets = append(targets, arg[3:])
                }
            }
        case "sed":
            // Check for -i flag; last args are file targets
            if containsFlag(args, "-i") {
                // Files are arguments after all flags and the script
                targets = append(targets, extractSedFiles(args)...)
            }
        case "cp", "mv", "install":
            // Last argument is typically the destination
            if len(args) > 0 {
                targets = append(targets, args[len(args)-1])
            }
        }
    }

    return targets
}
```

**Advantages over regex**:
- Correctly handles quoting: `echo > "/path/with spaces/file"` extracts the full path
- Handles nesting: `(echo > /path) && other_cmd` correctly scopes the redirect
- Handles multi-command: `cmd1; echo > /path; cmd2` extracts from the correct statement
- Type-safe: `file_redirect` node type is unambiguous

**Still bypassed by**: eval, base64, interpreter wrappers, runtime variable expansion.

**Implementation note**: Claude Code itself uses tree-sitter-bash for its permission walker (confirmed by issue #47701), demonstrating this approach is production-viable. However, even Claude Code's implementation had gaps with `file_redirect` nodes, showing the difficulty of complete coverage.

#### Strategy C: Command Blocklist (complementary, both implementations)

For bypass vectors that cannot be caught by path extraction, block the evasion mechanisms themselves:

```bash
# Block evasion mechanisms
EVASION_PATTERNS=(
    'base64.*-d.*\|\s*(ba)?sh'     # base64 decode piped to shell
    'base64.*--decode.*\|\s*(ba)?sh'
    'eval\s'                        # eval execution
    'bash\s+-c\s'                   # nested shell
    'sh\s+-c\s'
    'python[23]?\s+-c\s.*open\s*\(' # python file operations
    'node\s+-e\s.*writeFile'        # node file operations
    'ruby\s+-e\s.*File\.'           # ruby file operations
    'perl\s+-e\s.*open\s*\('        # perl file operations
)
```

This is the "block the tools of evasion" approach rather than trying to analyze what the evasion produces. It has false positives (legitimate uses of `eval`, `bash -c`, etc.) and should be applied selectively -- only in the context of self-protection, not for all Bash commands.

### 3.4 Practical Assessment

**For Phase 32 (Bash script hooks)**: Use Strategy A (regex) + Strategy C (evasion blocklist). Accept that this catches 80-90% of naive attacks and raises the cost for sophisticated attacks. The remaining 10-20% are caught by:
- Write/Edit hooks (which DO have reliable canonicalization)
- SessionStart integrity check (detects after-the-fact tampering)
- `gdev doctor` runtime verification

**For Go binary (future)**: Add Strategy B (AST) on top. The tree-sitter-bash parser extracts redirects more accurately. Still cannot handle eval/base64/interpreter wrappers, but these are blocked by Strategy C.

**Fundamental limit**: An AI agent with access to arbitrary Bash commands can always find a way to write to a file that evades static analysis. This is why kernel-level enforcement (AgentFS, bubblewrap sandbox, Nix store immutability) is the only complete solution. Path-based hooks are a practical defense layer, not a complete one.

---

## 4. Path Matching Algorithm

### 4.1 Match Types

gdev needs three match types for protected paths:

1. **Exact match**: Protect a specific file (e.g., `~/.claude/settings.json`)
2. **Prefix match**: Protect a directory and everything under it (e.g., `~/.qsdev/`)
3. **Suffix match**: Protect files by name pattern regardless of location (e.g., any file ending in `/settings.json` under `~/.claude/`)

### 4.2 Algorithm

```bash
# matches_protected_path - Check if a canonicalized path matches any
# protected path rule.
#
# Returns 0 (true) if the path matches, 1 (false) otherwise.
# Sets MATCH_REASON to a human-readable explanation.
matches_protected_path() {
    local canon_path="$1"
    local verdict="$2"  # "deny" or "ask"

    # --- Exact matches (deny) ---
    if [[ "$verdict" == "deny" || "$verdict" == "both" ]]; then
        local -a DENY_EXACT=(
            "$REAL_HOME/.claude/settings.json"
            "$REAL_HOME/.claude/settings.local.json"
            "$REAL_HOME/.claude/policy-limits.json"
            "$REAL_HOME/.config/nix/nix.conf"
        )
        for protected in "${DENY_EXACT[@]}"; do
            if [[ "$canon_path" == "$protected" ]]; then
                MATCH_REASON="exact match: $protected"
                return 0
            fi
        done

        # --- Prefix matches (deny) ---
        local -a DENY_PREFIX=(
            "$REAL_HOME/.qsdev/"
        )
        for protected in "${DENY_PREFIX[@]}"; do
            if [[ "$canon_path" == "$protected"* ]]; then
                MATCH_REASON="prefix match: $protected"
                return 0
            fi
        done
    fi

    # --- Ask matches ---
    if [[ "$verdict" == "ask" || "$verdict" == "both" ]]; then
        # Project-level .claude/ directory requires the path to be
        # relative to the project root, not HOME.
        # Use PROJECT_ROOT from the hook's environment.
        local -a ASK_EXACT=()
        local -a ASK_PREFIX=()

        if [[ -n "$PROJECT_ROOT" ]]; then
            ASK_EXACT+=(
                "$PROJECT_ROOT/.mcp.json"
                "$PROJECT_ROOT/.gdev.yaml"
            )
            ASK_PREFIX+=(
                "$PROJECT_ROOT/.claude/commands/"
                "$PROJECT_ROOT/.claude/agents/"
            )
        fi

        for protected in "${ASK_EXACT[@]}"; do
            if [[ "$canon_path" == "$protected" ]]; then
                MATCH_REASON="ask-exact match: $protected"
                return 0
            fi
        done
        for protected in "${ASK_PREFIX[@]}"; do
            if [[ "$canon_path" == "$protected"* ]]; then
                MATCH_REASON="ask-prefix match: $protected"
                return 0
            fi
        done
    fi

    return 1
}
```

### 4.3 Handling .claude/ at Different Roots

The `.claude/` directory exists at two levels with different security treatment:

| Location | Contains | Verdict |
|----------|----------|---------|
| `~/.claude/settings.json` | User-level hook registrations, deny rules | DENY (agent must never modify) |
| `~/.claude/settings.local.json` | User-level local overrides | DENY |
| `~/.claude/policy-limits.json` | Token/rate limits | DENY |
| `<project>/.claude/settings.json` | Project-level settings | ASK (legitimate but sensitive) |
| `<project>/.claude/commands/*.md` | Custom slash commands | ASK (injection risk) |
| `<project>/.claude/agents/*.md` | Custom agent definitions | ASK (injection risk) |
| `<project>/.claude/rules/*.md` | Custom rules | ASK (injection risk) |

The matching algorithm must distinguish between these by checking whether the path is under `$HOME/.claude/` (deny) or `$PROJECT_ROOT/.claude/` (ask). The canonicalized `$HOME` (via `realpath $HOME`) is used for the comparison, ensuring that even if `$HOME` itself is a symlink, the comparison is correct.

### 4.4 Glob vs Regex

**Recommendation: Neither.** Use string prefix/suffix/exact matching with canonicalized paths.

- **Glob patterns** (`~/.qsdev/hooks/*.sh`) would require a glob expansion library. Unnecessary complexity for the limited pattern vocabulary gdev needs.
- **Regex patterns** add attack surface (ReDoS), require escaping of literal path characters, and are harder to audit.
- **String prefix matching** (`canon_path starts with "$HOME/.qsdev/"`) is simple, correct, and fast. It naturally handles "protect a directory and everything under it" without glob or regex.

For the rare case where suffix matching is needed (e.g., "any file named `settings.json` under `.claude/`"), combine prefix and suffix checks in code rather than introducing a pattern language.

---

## 5. Implementation Pseudocode

### 5.1 Bash Script: Self-Protection File Hook (Write/Edit/Read)

```bash
#!/usr/bin/env bash
set -euo pipefail

# gdev self-protection: file operation guard
# Hook event: PreToolUse
# Matcher: Write|Edit|Read

INPUT=$(cat)
TOOL_NAME=$(echo "$INPUT" | jq -r '.tool_name // ""')
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // ""')

# --- Canonicalization ---
canonicalize_path() {
    local raw_path="$1"
    [[ -z "$raw_path" ]] && { echo ""; return; }
    # Expand tilde
    [[ "$raw_path" == "~/"* ]] && raw_path="$HOME/${raw_path#\~/}"
    [[ "$raw_path" == "~" ]] && raw_path="$HOME"
    # Tier 1: filesystem
    local resolved
    resolved=$(realpath -- "$raw_path" 2>/dev/null) && { echo "$resolved"; return; }
    # Tier 2: lexical (non-existent paths)
    resolved=$(realpath -m -- "$raw_path" 2>/dev/null) && { echo "$resolved"; return; }
    # Tier 3: passthrough
    echo "$raw_path"
}

CANON_PATH=$(canonicalize_path "$FILE_PATH")
REAL_HOME=$(realpath -m "$HOME" 2>/dev/null || echo "$HOME")

# --- Protected path definitions ---
DENY_EXACT=(
    "$REAL_HOME/.claude/settings.json"
    "$REAL_HOME/.claude/settings.local.json"
    "$REAL_HOME/.claude/policy-limits.json"
    "$REAL_HOME/.config/nix/nix.conf"
)
DENY_PREFIX=(
    "$REAL_HOME/.qsdev/"
)
ASK_PATHS=()
if [[ "$TOOL_NAME" == "Read" ]]; then
    ASK_PATHS=(
        "$REAL_HOME/.claude/settings.json"
        "$REAL_HOME/.claude/settings.local.json"
    )
fi

# --- DENY checks (Write/Edit only) ---
if [[ "$TOOL_NAME" == "Write" || "$TOOL_NAME" == "Edit" ]]; then
    for protected in "${DENY_EXACT[@]}"; do
        if [[ "$CANON_PATH" == "$protected" ]]; then
            echo "BLOCKED by gdev: cannot modify $protected" >&2
            echo "  Resolved from: $FILE_PATH" >&2
            echo "  Use 'gdev enable hooks' to manage hook configuration." >&2
            exit 2
        fi
    done
    for protected in "${DENY_PREFIX[@]}"; do
        if [[ "$CANON_PATH" == "$protected"* ]]; then
            echo "BLOCKED by gdev: cannot write under $protected" >&2
            echo "  Resolved from: $FILE_PATH" >&2
            exit 2
        fi
    done
fi

# --- ASK checks ---
for protected in "${ASK_PATHS[@]}"; do
    if [[ "$CANON_PATH" == "$protected" ]]; then
        cat <<ASKJSON
{"decision":"ask","message":"gdev: The agent wants to read security configuration ($FILE_PATH). Allow?"}
ASKJSON
        exit 0
    fi
done

# Not a protected path -- allow
exit 0
```

### 5.2 Go Implementation: Canonicalization + Matching

```go
package selfprotect

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"
)

// ProtectedPath defines a single protected path rule.
type ProtectedPath struct {
    Path      string    // Canonicalized protected path
    MatchType MatchType // Exact, Prefix, or Suffix
    Verdict   Verdict   // Deny or Ask
    Message   string    // Human-readable explanation
}

type MatchType int
const (
    MatchExact  MatchType = iota
    MatchPrefix
)

type Verdict int
const (
    VerdictAllow Verdict = iota
    VerdictDeny
    VerdictAsk
)

// PathMatcher holds the canonicalized protected path set.
type PathMatcher struct {
    rules    []ProtectedPath
    realHome string
}

// NewPathMatcher creates a matcher with default self-protection rules.
func NewPathMatcher() (*PathMatcher, error) {
    home, err := os.UserHomeDir()
    if err != nil {
        return nil, fmt.Errorf("cannot determine home directory: %w", err)
    }
    realHome := CanonicalizePath(home)

    m := &PathMatcher{realHome: realHome}
    m.rules = []ProtectedPath{
        {
            Path:      filepath.Join(realHome, ".claude", "settings.json"),
            MatchType: MatchExact,
            Verdict:   VerdictDeny,
            Message:   "Cannot modify Claude Code settings (hook registrations)",
        },
        {
            Path:      filepath.Join(realHome, ".claude", "settings.local.json"),
            MatchType: MatchExact,
            Verdict:   VerdictDeny,
            Message:   "Cannot modify Claude Code local settings",
        },
        {
            Path:      filepath.Join(realHome, ".claude", "policy-limits.json"),
            MatchType: MatchExact,
            Verdict:   VerdictDeny,
            Message:   "Cannot modify Claude Code policy limits",
        },
        {
            Path:      filepath.Join(realHome, ".qsdev") + "/",
            MatchType: MatchPrefix,
            Verdict:   VerdictDeny,
            Message:   "Cannot write to gdev installation directory",
        },
        {
            Path:      filepath.Join(realHome, ".config", "nix", "nix.conf"),
            MatchType: MatchExact,
            Verdict:   VerdictDeny,
            Message:   "Cannot modify Nix configuration",
        },
    }
    return m, nil
}

// Check evaluates a raw file path against all protection rules.
func (m *PathMatcher) Check(rawPath string) (Verdict, string) {
    canon := CanonicalizePath(rawPath)
    if canon == "" {
        return VerdictAllow, ""
    }

    for _, rule := range m.rules {
        switch rule.MatchType {
        case MatchExact:
            if canon == rule.Path {
                return rule.Verdict, rule.Message
            }
        case MatchPrefix:
            if strings.HasPrefix(canon, rule.Path) {
                return rule.Verdict, rule.Message
            }
        }
    }
    return VerdictAllow, ""
}
```

---

## 6. Edge Cases and Design Decisions

### 6.1 What if `realpath` is Missing?

On NixOS, `realpath` is part of GNU coreutils, which is always available in the base system. This is not a concern for the NixOS target. For broader portability:
- macOS: `realpath` is available since macOS 12.3 (Monterey). Older versions need `grealpath` from Homebrew coreutils.
- Alpine/BusyBox: BusyBox `realpath` supports `-m` flag.
- Fallback: `readlink -f` is an alternative (available on most Linux systems) but does NOT support non-existent paths (no equivalent to `-m`).

### 6.2 Should We Resolve Parent Even When Leaf Exists?

Yes. `realpath` (Tier 1) resolves the entire path including the leaf. If the leaf is itself a symlink (e.g., `settings.json` is a symlink to the real settings file), `realpath` follows it. This is correct behavior -- we want the true final target, not an intermediate symlink.

### 6.3 What About NixOS Store Paths?

NixOS stores packages in `/nix/store/`, which is read-only. Paths like `/nix/store/abc123-coreutils/bin/realpath` are immutable. gdev's self-protection rules do not need to protect Nix store paths because they are already protected by the Nix store's read-only mount. However, if gdev hooks themselves are deployed via Nix (e.g., via home-manager), their paths resolve through the Nix store, and the `~/.qsdev/hooks/script.sh` might be a symlink to `/nix/store/...`. `realpath` resolves this correctly -- the canonical path is the Nix store path, which is different from the `~/.qsdev/hooks/` prefix match. This needs consideration: if hooks are Nix-managed, the prefix match rule should include the resolved Nix store path, or the rule should match against the pre-resolution path.

**Recommendation**: Match against BOTH the raw path and the canonicalized path. If either matches, the protection rule fires. This handles both Nix-managed (symlinked) and directly-deployed hooks.

### 6.4 Exit Code Semantics

Claude Code hooks use specific exit codes:
- `exit 0`: Allow (or JSON response on stdout for ask/deny)
- `exit 2`: Block (stderr message shown to agent)
- Any other non-zero: Error (Claude Code may treat as block or skip depending on configuration)

The self-protection hooks should use `exit 2` for deny (clear signal to Claude Code) and JSON stdout for ask verdicts. Using `exit 1` would work but `exit 2` is the documented "explicit block" code.

### 6.5 Handling ELOOP (Recursive Symlinks)

If `realpath` encounters a recursive symlink (symlink loop), it fails with ELOOP. The hook should treat this as suspicious and DENY:

```bash
resolved=$(realpath -- "$raw_path" 2>/dev/null)
if [[ $? -ne 0 ]]; then
    # Check if it's ELOOP specifically
    if realpath -- "$raw_path" 2>&1 | grep -q "Too many levels of symbolic links"; then
        echo "BLOCKED by gdev: recursive symlink detected at $raw_path" >&2
        exit 2
    fi
    # Otherwise fall through to Tier 2 (non-existent path)
fi
```

---

## Depth Checklist

- [x] **Underlying mechanism explained** -- 9 bypass techniques with specific mechanisms, kernel-level details (dentry cache, lstat calls, ELOOP), and the fundamental undecidability of Bash command analysis
- [x] **Key tradeoffs and limitations identified** -- Regex vs AST vs kernel enforcement tradeoff, the "speed bump not a wall" limitation of path-based Bash analysis, performance vs security (realpath cost is negligible), NixOS store path complications
- [x] **Compared to alternatives** -- Prempti (two-tier canonicalize in Rust), bashguard (tree-sitter AST), AgentFS (kernel-level FUSE isolation), Claude Code's own tree-sitter-bash permission walker, Gemini CLI's vulnerable approach (no canonicalization)
- [x] **Failure modes and edge cases** -- Non-existent paths, recursive symlinks, NixOS store symlinks, Unicode normalization, hardlink vs symlink differences, TOCTOU across tool calls, missing realpath binary, exit code semantics
- [x] **Concrete examples found** -- CVE-2026-5061 (Consul-template TOCTOU), Gemini CLI #1121 (symlink bypass), Claude Code #47701 (file_redirect handling gap), moai-adk #342 (Unicode NFD/NFC mismatch), Ona /proc/self/root bypass, ZeptoClaw CVE-2026-32232 (combined symlink+TOCTOU+hardlink)
- [x] **Report is standalone-readable** -- Complete pseudocode for both Bash and Go, full bypass catalog, design decisions documented with rationale

---

## Sources

### External (saved to docs/)
| File | Content |
|------|---------|
| `docs/gemini-cli-symlink-bypass.md` | Gemini CLI symlink bypass vulnerability (TOCTOU) |
| `docs/go-filepath-resolve-proposal.md` | Go filepath.Resolve proposal (EvalSymlinks limitations) |
| `docs/realpath-man-page.md` | realpath(1) man page with flag semantics |
| `docs/moai-adk-unicode-path-mismatch.md` | macOS Unicode NFD/NFC path comparison failure |
| `docs/agentfs-kernel-isolation-approach.md` | AgentFS kernel-enforced mount isolation |
| `docs/posix-hardlink-security-issues.md` | POSIX hardlink bypass analysis |
| `docs/consul-template-symlink-bypass-cve-2026-5061.md` | Consul-template TOCTOU symlink bypass |
| `docs/bashguard-ast-command-security.md` | bashguard tree-sitter-bash AST command interception |
| `docs/tree-sitter-bash-redirect-node-types.md` | tree-sitter-bash AST node types for redirections |
| `docs/claude-code-file-redirect-permission-issue.md` | Claude Code #47701: file_redirect handling gap |

### Internal (from prior spikes)
| File | Content |
|------|---------|
| `docs/prempti-path-canonicalization-source.md` | Prempti's two-tier canonicalization (Rust source) |
| `docs/prempti-interceptor-path-handling.md` | Prempti interceptor architecture (thin passthrough) |
| `docs/ona-claude-code-sandbox-escape.md` | /proc/self/root bypass and autonomous sandbox disable |
| `prempti-patterns-research.md` | Prempti rule translation including path canonicalization |
| `threat-model-research.md` | Vector 4 (path manipulation) threat analysis |
