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
// matched.
func SubcommandRules(cli string, subs ...string) []string {
	var rules []string
	for _, prefix := range []string{cli, "env *" + cli} {
		for _, sub := range subs {
			plain, global := prefix+" "+sub, prefix+" * "+sub
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
// Unlike MatchesDenyRule, which deliberately widens a trailing " *" to a
// token-boundary prefix, this matcher answers what Claude Code itself would
// match, so tests can assert both what a rule blocks and what it leaves
// allowed.
func MatchesBashRule(rule, command string) bool {
	tool, pattern := ParseToolPattern(rule)
	if tool != "Bash" {
		return false
	}
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
