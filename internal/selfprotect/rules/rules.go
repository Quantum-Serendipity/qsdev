package rules

import (
	"regexp"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/canon"
)

var (
	reDeleteCmd     = regexp.MustCompile(`\b(rm|unlink|shred)\b`)
	reSymlinkCmd    = regexp.MustCompile(`\bln\b.*-s`)
	reTraversal     = regexp.MustCompile(`\.\./`)
	reCopyCmd       = regexp.MustCompile(`\b(cp|rsync|mv|tar|dd|tee)\b`)
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
	patterns := []string{
		".qsdev/",
		".gdev/",
		".claude/settings",
		".claude/managed-settings",
		"/etc/gdev/",
		"/etc/claude-code/",
		".claude/hooks/",
	}
	for _, p := range patterns {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}

func isWriteOrEdit(toolName string) bool {
	return toolName == "Write" || toolName == "Edit" || toolName == "MultiEdit"
}

func isMCPConfigPath(path string) bool {
	return strings.HasSuffix(path, ".mcp.json") ||
		strings.HasSuffix(path, ".cursor/mcp.json") ||
		strings.HasSuffix(path, ".vscode/mcp.json")
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
		sensitiveFragments := []string{"/policy/", "/trust.yaml", "/session-state.json", "managed-settings.json"}
		for _, frag := range sensitiveFragments {
			if strings.Contains(ctx.CanonicalPath, frag) {
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
		if reDeleteCmd.MatchString(ctx.Command) && containsProtectedPathStr(ctx.Command) {
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
		if reSymlinkCmd.MatchString(ctx.Command) && containsProtectedPathStr(ctx.Command) {
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
		if reTraversal.MatchString(ctx.Command) && containsProtectedPathStr(ctx.Command) {
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
		if reCopyCmd.MatchString(ctx.Command) && containsProtectedPathStr(ctx.Command) {
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
		modifiesInPlace := reChmodCmd.MatchString(ctx.Command) ||
			reSedInplace.MatchString(ctx.Command) ||
			reAwkInplace.MatchString(ctx.Command)
		if modifiesInPlace && strings.Contains(ctx.Command, ".claude/hooks/") {
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
		if isWriteOrEdit(ctx.ToolName) && (isMCPConfigPath(ctx.CanonicalPath) || isMCPConfigPath(ctx.FilePath)) {
			return Deny, "modification of MCP server configuration"
		}
		if ctx.ToolName == "Bash" && isMCPConfigPath(ctx.Command) {
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
		if ctx.ToolName == "Bash" && reAuditPath.MatchString(ctx.Command) && reAuditModCmd.MatchString(ctx.Command) {
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
