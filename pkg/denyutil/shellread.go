package denyutil

import "strings"

// shellReaders are the commands a shell-read tripwire denies over a home
// path: the ones that print or copy a file's bytes. Like the cloud modules'
// denied operations, this is rule vocabulary, not a list of tools to skip.
var shellReaders = []string{"cat", "less", "head", "tail", "cp", "base64", "xxd"}

// ShellReaders returns the reader commands HomeShellReadRules denies.
func ShellReaders() []string {
	return append([]string(nil), shellReaders...)
}

// shellSpecial holds the characters that make a path need quoting in a shell
// command. A path without any of them has one spelling, ~/<rel>.
const shellSpecial = " \t\n'\"\\$`!*?[](){}<>|&;#"

// HomeShellReadRules returns the Bash deny rules of a shell-read tripwire over
// the home path ~/<rel>: for each reader of ShellReaders, behind each of its
// CommandPrefixes (plain and `env X=1 <reader>`), and for each shell spelling
// of the path. The '*' between the reader and the path admits options
// (`cat -n ~/.npmrc`) and matches the empty string for the bare form.
//
// A file (dir false) gets "<p> *<path>*". A directory gets "<p> *<path>/*"
// for anything inside it, and "<p> *<path>" and "<p> *<path> *" for the
// directory itself (`cp -r ~/.ssh /tmp/x`); none of them matches a sibling
// that merely starts with the name (~/.sshfoo) or a command that is not a
// reader (`ls ~/.ssh`).
//
// It is a tripwire for the spellings an agent usually writes, not a boundary:
// $HOME and absolute spellings, other quoting, globs, ~// and ~/./, absolute
// reader paths, stdin, other readers, input redirection and PowerShell are
// not matched.
func HomeShellReadRules(rel string, dir bool) []string {
	var rules []string
	for _, reader := range shellReaders {
		for _, prefix := range CommandPrefixes(reader) {
			for _, target := range homeShellSpellings(rel) {
				head := "Bash(" + prefix + " *" + target
				if !dir {
					rules = append(rules, head+"*)")
					continue
				}
				rules = append(rules, head+"/*)", head+")", head+" *)")
			}
		}
	}
	return rules
}

// homeShellSpellings returns the shell spellings of ~/<rel>: ~/<rel> itself
// when rel has no shell-special character, otherwise the backslash-escaped
// form and the double-quoted suffix form ~/"<rel>". The raw form would split
// into several arguments and never name the file, so it is not emitted.
func homeShellSpellings(rel string) []string {
	if !strings.ContainsAny(rel, shellSpecial) {
		return []string{"~/" + rel}
	}
	var escaped, quoted strings.Builder
	for _, r := range rel {
		if strings.ContainsRune(shellSpecial, r) {
			escaped.WriteByte('\\')
		}
		escaped.WriteRune(r)
		if strings.ContainsRune("\"\\$`", r) {
			quoted.WriteByte('\\')
		}
		quoted.WriteRune(r)
	}
	return []string{"~/" + escaped.String(), `~/"` + quoted.String() + `"`}
}

// ShellReadCovered reports whether deny holds every HomeShellReadRules rule
// for homePath, a "~/"-prefixed path, either as a file or for one of its
// directories under ~/ (the path itself or an ancestor). A path segment with
// a glob character ends the path: ~/.aws/sso/cache/* is covered by the
// directory rules of ~/.aws/sso/cache or of an ancestor such as ~/.aws.
// It is a set lookup, so checking a large deny list stays cheap.
func ShellReadCovered(deny []string, homePath string) bool {
	rel, ok := strings.CutPrefix(homePath, "~/")
	if !ok {
		return false
	}
	segments := strings.Split(rel, "/")
	globbed := false
	for i, seg := range segments {
		if strings.ContainsAny(seg, "*?[") {
			segments, globbed = segments[:i], true
			break
		}
	}
	if len(segments) == 0 || segments[0] == "" {
		return false
	}
	set := make(map[string]struct{}, len(deny))
	for _, r := range deny {
		set[r] = struct{}{}
	}
	if !globbed && containsAll(set, HomeShellReadRules(strings.Join(segments, "/"), false)) {
		return true
	}
	for n := len(segments); n > 0; n-- {
		if containsAll(set, HomeShellReadRules(strings.Join(segments[:n], "/"), true)) {
			return true
		}
	}
	return false
}

// containsAll reports whether set holds every rule of rules.
func containsAll(set map[string]struct{}, rules []string) bool {
	for _, r := range rules {
		if _, ok := set[r]; !ok {
			return false
		}
	}
	return true
}
