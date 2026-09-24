package rules

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/canon"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/cmdscan"
)

var (
	reDeleteCmd     = regexp.MustCompile(`\b(rm|rmdir|unlink|shred|find|truncate)\b`)
	reLinkCmd       = regexp.MustCompile(`\b(ln|link)\b`)
	reKillCmd       = regexp.MustCompile(`\b(kill|pkill|killall)\b`)
	reProcessTarget = regexp.MustCompile(`\b(qsdev|claude|gdev)\b`)
	reMcpInjection  = regexp.MustCompile(`(?i)(system\s*prompt|ignore\s*previous|you\s+are\s+now|<\s*system\s*>|<\s*/?\s*instructions?\s*>)`)
	reBypassCmd     = regexp.MustCompile(`\bqsdev\s+hook\s+bypass`)
	reCliControl    = regexp.MustCompile(`\bqsdev\s+(disable\s+hooks|enable\s+hooks\s+--force|session\s+allow\b)`)
	reSystemctl     = regexp.MustCompile(`\bsystemctl\s+(stop|disable)\b.*\b(qsdev|gdev)\b`)
	// reProcInfo matches the per-process /proc entries that expose a process's
	// environment, command line, open files, or root: under any pid spelling
	// (self, thread-self, digits, $$, $PPID, ${PID}, or the empty slot a
	// stripped `$(pgrep …)` leaves in a parsed word), and through a
	// /task/<tid>/ thread entry.
	reProcInfo = regexp.MustCompile(`/proc/[^/\s]*(?:/task/[^/\s]+)?/(?:environ|cmdline|fd/|root/)`)
	// reParentSegment matches a `..` path segment within one word.
	reParentSegment = regexp.MustCompile(`(^|/)\.\.(/|$)`)
)

func containsProtectedPathStr(s string) bool {
	return canon.ContainsProtectedPath(s)
}

var (
	deleteVerbs = map[string]bool{"rm": true, "rmdir": true, "unlink": true, "shred": true, "find": true, "truncate": true}
	linkVerbs   = map[string]bool{"ln": true, "link": true}
	copyVerbs   = map[string]bool{"cp": true, "rsync": true, "mv": true, "tar": true, "dd": true, "tee": true, "truncate": true}
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
// path: the line invokes a delete verb (in the raw text, its quote-stripped
// form, or a parsed command word) and mutates a protected path per the shared
// bashMutatesProtected predicate. So `sh -c 'rm .claude/settings.json'`,
// `sudo rm …`, `… | xargs rm`, `V=…; rm "$V"`, `rm ~/.cl""aude/settings.json`
// and `cd .claude && rm settings.json` all deny, while
// `rm -rf ../build && grep x .claude/...` clears.
func deleteTargetsProtected(ctx *EvalContext) bool {
	return hasVerb(ctx, reDeleteCmd, deleteVerbs) && bashMutatesProtected(ctx)
}

// linkTargetsProtected reports whether `ln`/`link` creates a link at or to a
// protected path, with the same shared mutation analysis as SP-003.
func linkTargetsProtected(ctx *EvalContext) bool {
	return hasVerb(ctx, reLinkCmd, linkVerbs) && bashMutatesProtected(ctx)
}

// copyIsDangerous reports whether a Bash command mutates a protected path
// through any command (the shared bashMutatesProtected predicate: redirect
// clobbers, copies onto it, in-place editors, interpreters, wrappers, unknown
// binaries) or exfiltrates one (a protected read copied, redirected, or piped
// out of the repo). Read-only commands and a benign in-repo backup of a
// protected file stay allowed.
func copyIsDangerous(ctx *EvalContext) bool {
	if bashMutatesProtected(ctx) {
		return true
	}
	if !lineMentionsProtected(ctx) {
		return false
	}
	scs, err := ctx.scannedCommands()
	return err == nil && protectedExfil(scs, ctx.CWD)
}

// traversalReachesProtected reports whether a `..`-containing argument or
// redirect target resolves to a protected path. Only a traversal that actually
// points at a protected path denies, so `rm -rf ../build && grep x
// .claude/settings.json` (DEFECT-10) stays allowed. Relative words are
// resolved against the directory each command runs in. Fails closed on a
// parse error when the line mentions a protected path.
func traversalReachesProtected(ctx *EvalContext) bool {
	if !strings.Contains(ctx.Command, "..") {
		return false
	}
	scs, err := ctx.scannedCommands()
	if err != nil {
		return lineMentionsProtected(ctx)
	}
	for _, sc := range scs {
		for _, w := range sc.Args {
			if reParentSegment.MatchString(w) && argRefersProtected(sc, w) {
				return true
			}
		}
		for _, w := range append(append([]string{}, sc.WriteRedirects...), sc.ReadRedirects...) {
			if reParentSegment.MatchString(w) && refersProtected(sc, w) {
				return true
			}
		}
	}
	return false
}

// auditTrailMutated reports whether a shell command writes to, deletes, or
// relocates an audit trail (auditAreas): a mutation of one (or a recursive one
// of its ancestor), or a copy-family command naming it. Reads clear.
func auditTrailMutated(ctx *EvalContext) bool {
	scs, err := ctx.scannedCommands()
	for _, a := range auditAreas {
		if bashMutatesArea(ctx, a) || (err == nil && copiesFromArea(scs, a)) {
			return true
		}
	}
	return false
}

// procInfoInCommand reports whether a Bash command reaches a sensitive /proc
// entry, spelled in the raw text, its quote-stripped form, or any parsed word
// (where an expansion in the pid slot leaves `/proc//environ`).
func procInfoInCommand(ctx *EvalContext) bool {
	if reProcInfo.MatchString(ctx.Command) || reProcInfo.MatchString(looseText(ctx.Command)) {
		return true
	}
	scs, _ := ctx.scannedCommands()
	for _, sc := range scs {
		words := append(append(append([]string{}, sc.Args...), sc.WriteRedirects...), sc.ReadRedirects...)
		for _, w := range words {
			if reProcInfo.MatchString(w) {
				return true
			}
		}
	}
	return false
}

func isWriteOrEdit(toolName string) bool {
	return toolName == "Write" || toolName == "Edit" || toolName == "MultiEdit"
}

// dedicatedWriteCategories are the protected categories whose Write/Edit is
// judged by a dedicated rule rather than SP-001: the audit trail (SP-013) and
// MCP server configs (MCP-001/MCP-005, which allow benign edits).
var dedicatedWriteCategories = map[string]bool{"audit": true, "mcp-config": true}

var sp001 = Rule{
	ID:       "SP-001",
	Name:     "Config file write block",
	Category: "self-protection",
	Evaluate: func(ctx *EvalContext) (Verdict, string) {
		if !isWriteOrEdit(ctx.ToolName) {
			return Allow, ""
		}
		// Every protected category is write-protected unless a dedicated rule
		// owns it. Listing the allowed categories (rather than the denied ones)
		// keeps a newly added category, like the security binary under
		// ~/.qsdev/bin, protected by default.
		protected, category := canon.IsProtected(ctx.CanonicalPath)
		if protected && !dedicatedWriteCategories[category] {
			return Deny, "write to protected " + category + " path: " + ctx.CanonicalPath
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
		if !cmdscan.IsShellTool(ctx.ToolName) {
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
	Name:     "Config link creation block",
	Category: "self-protection",
	Evaluate: func(ctx *EvalContext) (Verdict, string) {
		if !cmdscan.IsShellTool(ctx.ToolName) {
			return Allow, ""
		}
		if linkTargetsProtected(ctx) {
			return Deny, "link creation targeting protected config path"
		}
		return Allow, ""
	},
}

var sp005 = Rule{
	ID:       "SP-005",
	Name:     "Config path traversal block",
	Category: "self-protection",
	Evaluate: func(ctx *EvalContext) (Verdict, string) {
		if !cmdscan.IsShellTool(ctx.ToolName) {
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
		if ctx.ToolName != "Read" && !cmdscan.IsShellTool(ctx.ToolName) {
			return Allow, ""
		}
		if ctx.CanonicalPath != "" && reProcInfo.MatchString(ctx.CanonicalPath) {
			return Deny, "access to sensitive proc filesystem path"
		}
		if cmdscan.IsShellTool(ctx.ToolName) && procInfoInCommand(ctx) {
			return Deny, "access to sensitive proc filesystem path"
		}
		return Allow, ""
	},
}

var sp007 = Rule{
	ID:       "SP-007",
	Name:     "Config mutation and exfiltration block",
	Category: "self-protection",
	Evaluate: func(ctx *EvalContext) (Verdict, string) {
		if !cmdscan.IsShellTool(ctx.ToolName) {
			return Allow, ""
		}
		if copyIsDangerous(ctx) {
			return Deny, "command modifies or exfiltrates a protected config path"
		}
		return Allow, ""
	},
}

// sp008 denies a shell command that changes the settings, and so the hooks,
// of a Claude Code session: a nested session started with a hook-dropping
// option, a settings override or a settings environment override, a
// settings-writing subcommand, or a write through $CLAUDE_CONFIG_DIR (see
// settingsOverride).
var sp008 = Rule{
	ID:       "SP-008",
	Name:     "Claude Code settings override block",
	Category: "self-protection",
	Evaluate: func(ctx *EvalContext) (Verdict, string) {
		if !cmdscan.IsShellTool(ctx.ToolName) {
			return Allow, ""
		}
		if reason := settingsOverride(ctx); reason != "" {
			return Deny, reason
		}
		return Allow, ""
	},
}

var sp009 = Rule{
	ID:       "SP-009",
	Name:     "Process management block",
	Category: "self-protection",
	Evaluate: func(ctx *EvalContext) (Verdict, string) {
		if !cmdscan.IsShellTool(ctx.ToolName) {
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
		if !cmdscan.IsShellTool(ctx.ToolName) {
			return Allow, ""
		}
		if bashMutatesArea(ctx, hooksArea) {
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
		if !isWriteOrEdit(ctx.ToolName) || !writesMCPConfig(ctx) {
			return Allow, ""
		}
		if reMcpInjection.MatchString(ctx.Content) {
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
		if protected, _ := canon.IsProtected(ctx.CanonicalPath); protected {
			return Deny, "MCP tool accessing protected path: " + ctx.CanonicalPath
		}
		if p, ok := mcpToolProtectedPath(ctx); ok {
			return Deny, "MCP tool accessing protected path: " + p
		}
		return Allow, ""
	},
}

var mcp005 = Rule{
	ID:       "MCP-005",
	Name:     "Server config tampering",
	Category: "mcp-poisoning",
	Evaluate: func(ctx *EvalContext) (Verdict, string) {
		// An MCP config edit may reformat, reorder, or remove servers. It may not
		// add a server or change what an existing one runs (its command, args,
		// url, env, ...): a server added by the agent runs with the user's
		// privileges in the next session, so servers are added through
		// `qsdev enable`. Injection content is MCP-001's concern.
		if isWriteOrEdit(ctx.ToolName) && writesMCPConfig(ctx) {
			if reMcpDangerousCmd.MatchString(ctx.Content) {
				return Deny, "MCP server configuration write with dangerous content"
			}
			if reason := mcpServerTampering(ctx); reason != "" {
				return Deny, reason
			}
			return Allow, ""
		}
		// A Bash command that mutates (writes/redirects to) an MCP config is a
		// blind, un-inspectable overwrite; deny it. Reads are allowed.
		if cmdscan.IsShellTool(ctx.ToolName) && bashMutatesMcpConfig(ctx) {
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
		if !cmdscan.IsShellTool(ctx.ToolName) {
			return Allow, ""
		}
		if bashMutatesArea(ctx, binaryArea) {
			return Deny, "modification of security binary"
		}
		return Allow, ""
	},
}

// sp011 denies writing a file a registered hook command executes: a program
// the hook names, resolved through PATH (a new file in an earlier PATH
// directory shadows it), and a hook script's shebang interpreter (see
// hookTargets).
var sp011 = Rule{
	ID:       "SP-011",
	Name:     "Hook command hijack block",
	Category: "self-protection",
	Evaluate: func(ctx *EvalContext) (Verdict, string) {
		if isWriteOrEdit(ctx.ToolName) && ctx.CanonicalPath != "" && ctx.hookTargetsFor().has(ctx.CanonicalPath) {
			return Deny, "write to a program a hook command runs: " + ctx.CanonicalPath
		}
		if cmdscan.IsShellTool(ctx.ToolName) {
			if p := writesHookTarget(ctx); p != "" {
				return Deny, "command writes a program a hook command runs: " + p
			}
		}
		return Allow, ""
	},
}

var sp012 = Rule{
	ID:       "SP-012",
	Name:     "Bypass command block",
	Category: "self-protection",
	Evaluate: func(ctx *EvalContext) (Verdict, string) {
		if !cmdscan.IsShellTool(ctx.ToolName) {
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
		if cmdscan.IsShellTool(ctx.ToolName) && auditTrailMutated(ctx) {
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
		if !cmdscan.IsShellTool(ctx.ToolName) {
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
