package denyutil

import (
	"regexp"
	"strings"
)

// SubcommandRules returns Bash deny rules that block `<cli> <sub>` however
// Claude usually writes it: plain (`terraform apply`), with global options
// between the CLI and the subcommand (`terraform -chdir=infra apply`,
// `helm --kube-context prod install`), and behind an `env` prefix
// (`env TF_LOG=debug terraform apply`). Claude Code matches Bash rules against
// the command text as written and strips wrappers such as timeout, nice and
// command before matching, but not env, so a rule anchored right after the CLI
// name misses both the global-option and the env form.
//
// A sub ending in "*" is a prefix (for example "show *-json*"); any other sub
// is a whole word sequence that matches bare or followed by arguments. For a
// whole-word sub the rules keep a space on both sides, so an argument that
// merely starts with the word (`-var initial_count=1` for "init") is not
// matched. A sub starting with "*" (an argument anywhere on the command line,
// such as "*--privileged*") already matches after global options, so it gets
// no separate global-option rule.
func SubcommandRules(cli string, subs ...string) []string {
	return subcommandRules(cli, " * ", subs)
}

// DashedOptionSubcommandRules is SubcommandRules for a CLI whose global
// options all start with "-" (docker, podman): the global-option form is
// anchored on that dash (`docker -* pull`), so a later word equal to the sub
// (`docker exec web git pull`) is not matched, while `docker --context prod
// pull` and `docker -H tcp://x pull` still are. A global option followed by
// another subcommand whose arguments contain the sub (`docker --context prod
// exec web git pull`) is still matched.
func DashedOptionSubcommandRules(cli string, subs ...string) []string {
	return subcommandRules(cli, " -* ", subs)
}

// subcommandRules builds SubcommandRules with gap, the pattern between the
// CLI and a whole-word or prefix sub in the global-option form.
func subcommandRules(cli, gap string, subs []string) []string {
	var rules []string
	for _, prefix := range CommandPrefixes(cli) {
		for _, sub := range subs {
			plain, global := prefix+" "+sub, prefix+gap+sub
			if strings.HasPrefix(sub, "*") {
				// "<cli> *x" strictly subsumes "<cli> * *x", bare or with
				// arguments.
				rules = append(rules, "Bash("+plain+")")
				if !strings.HasSuffix(sub, "*") {
					rules = append(rules, "Bash("+plain+" *)")
				}
				continue
			}
			if strings.HasSuffix(sub, "*") {
				rules = append(rules, "Bash("+plain+")", "Bash("+global+")")
				continue
			}
			// A trailing " *" matches the bare command only when it is the
			// rule's only wildcard, so every form with another wildcard (the
			// global-option form, the env prefix) needs a bare rule of its own.
			if strings.Contains(prefix, "*") {
				rules = append(rules, "Bash("+plain+")")
			}
			rules = append(rules,
				"Bash("+plain+" *)",
				"Bash("+global+")",
				"Bash("+global+" *)",
			)
		}
	}
	return rules
}

// CommandPrefixes returns the spellings that start a cli invocation in a
// Bash deny rule: the plain name and the `env` prefix (`env X=1 <cli>`),
// which Claude Code does not strip before matching. Rule builders outside
// this package use it rather than repeating the list.
func CommandPrefixes(cli string) []string {
	return []string{cli, "env *" + cli}
}

// InterspersedOptionRules returns Bash deny rules for CLIs that accept global
// options anywhere on the command line (aws, gcloud, az): each word of an
// operation may be preceded by options, so `aws sts --profile p
// get-session-token` and `gcloud auth --quiet print-access-token` match as
// well as `aws --profile p sts get-session-token`. The rules are also emitted
// behind an `env` prefix (`env AWS_PROFILE=x aws ...`): Claude Code strips
// wrappers such as timeout, nice and command before matching, but not env.
//
// Operation words are joined with " *", which keeps a word boundary after each
// word but not before the next, so operations should be distinctive names
// (get-session-token, print-access-token) rather than short common words. An
// explicit " * " inside an operation is a gap that keeps the word boundary on
// both sides, for short flags such as "aks get-credentials * -a". An operation
// ending in "*" is a prefix; any other operation matches bare or followed by
// arguments.
func InterspersedOptionRules(cli string, ops ...string) []string {
	var rules []string
	for _, prefix := range []string{cli + " *", "env *" + cli + " *"} {
		for _, op := range ops {
			for _, pattern := range interspersedPatterns(prefix, op) {
				if strings.HasSuffix(pattern, "*") {
					rules = append(rules, "Bash("+pattern+")")
					continue
				}
				// The pattern already has wildcards, so a trailing " *" would
				// not match the bare operation: emit it separately.
				rules = append(rules, "Bash("+pattern+")", "Bash("+pattern+" *)")
			}
		}
	}
	return rules
}

// interspersedPatterns returns the patterns for one InterspersedOptionRules
// operation. Words are joined with " *"; an explicit " * " gap matches either
// a single space or options between two spaces, so the operation is emitted in
// both forms (a gap of zero options would otherwise need two spaces).
func interspersedPatterns(prefix, op string) []string {
	segments := strings.Split(op, " * ")
	for i, seg := range segments {
		segments[i] = strings.Join(strings.Fields(seg), " *")
	}
	patterns := []string{prefix + segments[0]}
	for _, seg := range segments[1:] {
		var next []string
		for _, p := range patterns {
			next = append(next, p+" "+seg, p+" * "+seg)
		}
		patterns = next
	}
	return patterns
}

// MatchesBashRule reports whether a Claude Code Bash permission rule such as
// "Bash(git log *)" matches command, following Claude Code's documented
// wildcard semantics (code.claude.com/docs/en/permissions, "Wildcard
// patterns"):
//
//   - `*` matches any text, including spaces, at any position;
//   - a rule without `*` matches one exact command;
//   - a trailing " *" (or ":*") that is the rule's only wildcard also matches
//     the bare command;
//   - the space before a trailing `*` is literal, so "ls *" does not match
//     "lsof".
//
// Unlike Shadows, which deliberately widens a trailing " *" to a
// token-boundary prefix, this matcher answers what Claude Code itself would
// match, so tests can assert both what a rule blocks and what it leaves
// allowed.
func MatchesBashRule(rule, command string) bool {
	tool, pattern := ParseToolPattern(rule)
	return tool == "Bash" && matchesCommandPattern(pattern, command)
}

// MatchesPowerShellRule is MatchesBashRule for Claude Code's PowerShell tool:
// it reports whether a "PowerShell(...)" rule matches command under the same
// wildcard semantics, and never matches a Bash rule. As the docs state
// ("Matching is case-insensitive"), pattern and command are compared with
// case folded. Claude Code also canonicalizes common aliases (gci, ls, dir
// for Get-ChildItem) before matching; that is not modelled, so callers must
// write the cmdlet name in both the rule and the command.
func MatchesPowerShellRule(rule, command string) bool {
	tool, pattern := ParseToolPattern(rule)
	return tool == "PowerShell" && matchesCommandPattern(strings.ToLower(pattern), strings.ToLower(command))
}

// FirstMatch returns the first of rules that matches op, a tool call written
// as "Bash(<command>)" or "PowerShell(<command>)", using MatchesBashRule or
// MatchesPowerShellRule for the op's tool. A Bash op never matches a
// PowerShell rule and the other way round; any other tool reports no match.
// Tests use the returned rule to name what blocked an operation that should
// have been allowed.
func FirstMatch(rules []string, op string) (rule string, ok bool) {
	tool, command := ParseToolPattern(op)
	var match func(rule, command string) bool
	switch tool {
	case "Bash":
		match = MatchesBashRule
	case "PowerShell":
		match = MatchesPowerShellRule
	default:
		return "", false
	}
	for _, r := range rules {
		if match(r, command) {
			return r, true
		}
	}
	return "", false
}

// matchesCommandPattern applies the documented wildcard semantics to one
// rule pattern (the text inside the parentheses).
func matchesCommandPattern(pattern, command string) bool {
	if base, ok := strings.CutSuffix(pattern, ":*"); ok {
		pattern = base + " *"
	}
	if !strings.Contains(pattern, "*") {
		return pattern == command
	}
	if base, ok := strings.CutSuffix(pattern, " *"); ok && !strings.Contains(base, "*") && command == base {
		return true
	}
	parts := strings.Split(pattern, "*")
	for i, p := range parts {
		parts[i] = regexp.QuoteMeta(p)
	}
	return regexp.MustCompile(`(?s)^` + strings.Join(parts, `.*`) + `$`).MatchString(command)
}

// sampleToken stands in for each wildcard in a SampleCommand.
const sampleToken = "x"

// SampleCommand returns a concrete command that the Bash permission rule
// matches, with each `*` (and a trailing ":*") replaced by a sample argument:
// "Bash(npm -* install *)" yields "npm -x install x". It reports false for a
// non-Bash rule or a Bash rule without a command. Checking another rule with
// MatchesBashRule against the sample tells whether that rule also covers
// commands the first one gates.
func SampleCommand(rule string) (string, bool) {
	tool, pattern := ParseToolPattern(rule)
	if tool != "Bash" || pattern == "" {
		return "", false
	}
	if base, ok := strings.CutSuffix(pattern, ":*"); ok {
		pattern = base + " *"
	}
	return strings.ReplaceAll(pattern, "*", sampleToken), true
}
