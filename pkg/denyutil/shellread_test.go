package denyutil

import (
	"slices"
	"strings"
	"testing"
)

func TestHomeShellReadRules(t *testing.T) {
	t.Parallel()
	const spaced = "Library/Application Support/pip/pip.conf"
	tests := []struct {
		name     string
		rel      string
		dir      bool
		perShape int // rules per reader and prefix
		want     []string
		denied   []string
		allowed  []string
	}{
		{
			name:     "file store",
			rel:      ".npmrc",
			perShape: 1,
			want:     []string{"Bash(cat *~/.npmrc*)", "Bash(env *cat *~/.npmrc*)", "Bash(xxd *~/.npmrc*)"},
			denied: []string{
				"cat ~/.npmrc", "cat -n ~/.npmrc", "cp ~/.npmrc /tmp/x",
				"env cat ~/.npmrc", "env FOO=1 head ~/.npmrc", "base64 ~/.npmrc",
			},
			allowed: []string{"cat .npmrc", "cat ./.npmrc", "ls ~/.npmrc", "npm whoami"},
		},
		{
			name:     "dir store",
			rel:      ".ssh",
			dir:      true,
			perShape: 3,
			want: []string{
				"Bash(cat *~/.ssh/*)", "Bash(cat *~/.ssh)", "Bash(cat *~/.ssh *)",
				"Bash(env *cp *~/.ssh/*)", "Bash(env *cp *~/.ssh)", "Bash(env *cp *~/.ssh *)",
			},
			denied: []string{
				"cat ~/.ssh/id_ed25519", "cp -r ~/.ssh /tmp/x", "cp -r ~/.ssh/ /tmp/x",
				"tail ~/.ssh", "env FOO=1 head ~/.ssh/id_ed25519",
			},
			allowed: []string{
				"ls ~/.ssh", "cp -r ~/.sshfoo /tmp/x", "cat ~/.sshfoo/x",
				"ssh -i ~/.ssh/id_ed25519 host",
			},
		},
		{
			name:     "nested dir store",
			rel:      ".config/gh",
			dir:      true,
			perShape: 3,
			want:     []string{"Bash(cat *~/.config/gh/*)", "Bash(cat *~/.config/gh)"},
			denied:   []string{"cat ~/.config/gh/hosts.yml", "cp -R ~/.config/gh /tmp/g"},
			allowed:  []string{"cat ~/.config/ghq", "cat ~/.config/ghq/x"},
		},
		{
			name:     "space in path",
			rel:      spaced,
			perShape: 2, // escaped and double-quoted spellings
			want: []string{
				`Bash(cat *~/Library/Application\ Support/pip/pip.conf*)`,
				`Bash(cat *~/"Library/Application Support/pip/pip.conf"*)`,
			},
			denied: []string{
				`cat ~/Library/Application\ Support/pip/pip.conf`,
				`cat ~/"Library/Application Support/pip/pip.conf"`,
				`env cat ~/"Library/Application Support/pip/pip.conf"`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rules := HomeShellReadRules(tt.rel, tt.dir)
			readers := ShellReaders()
			if want := len(readers) * len(CommandPrefixes("x")) * tt.perShape; len(rules) != want {
				t.Fatalf("got %d rules, want %d: %v", len(rules), want, rules)
			}
			for _, r := range rules {
				if err := Validate(r); err != nil {
					t.Errorf("Validate: %v", err)
				}
				if strings.Contains(r, "Application Support") && !strings.Contains(r, `"Library/Application Support`) {
					t.Errorf("rule %s has the raw unescaped-space spelling", r)
				}
				if n := countOf(rules, r); n != 1 {
					t.Errorf("rule %s emitted %d times", r, n)
				}
			}
			for _, w := range tt.want {
				if !slices.Contains(rules, w) {
					t.Errorf("rules lack %s", w)
				}
			}
			for _, reader := range readers {
				for _, prefix := range CommandPrefixes(reader) {
					if !slices.ContainsFunc(rules, func(r string) bool { return strings.HasPrefix(r, "Bash("+prefix+" *~/") }) {
						t.Errorf("no rule for prefix %q", prefix)
					}
				}
			}
			for _, cmd := range tt.denied {
				if _, ok := FirstMatchingBashRule(rules, cmd); !ok {
					t.Errorf("no rule denies %q", cmd)
				}
			}
			for _, cmd := range tt.allowed {
				if r, ok := FirstMatchingBashRule(rules, cmd); ok {
					t.Errorf("rule %s over-blocks %q", r, cmd)
				}
			}
		})
	}
}

func TestShellReaders(t *testing.T) {
	t.Parallel()
	want := []string{"cat", "less", "head", "tail", "cp", "base64", "xxd"}
	got := ShellReaders()
	if !slices.Equal(got, want) {
		t.Fatalf("ShellReaders() = %v, want %v", got, want)
	}
	got[0] = "mutated"
	if ShellReaders()[0] != "cat" {
		t.Fatal("ShellReaders returns its backing slice")
	}
}

func TestShellReadCovered(t *testing.T) {
	t.Parallel()
	file := HomeShellReadRules(".aws/credentials", false)
	aws := HomeShellReadRules(".aws", true)
	cache := HomeShellReadRules(".aws/sso/cache", true)
	unrelated := []string{"Bash(rm -rf /)", "Read(~/.aws/**)"}
	join := func(sets ...[]string) []string { return slices.Concat(sets...) }
	tests := []struct {
		name string
		deny []string
		path string
		want bool
	}{
		{"file store full set", join(unrelated, file), "~/.aws/credentials", true},
		{"ancestor dir full set", join(unrelated, aws), "~/.aws/credentials", true},
		{"dir itself", aws, "~/.aws", true},
		{"glob path, ancestor dir", aws, "~/.aws/sso/cache/*", true},
		{"glob path, globbed dir itself", cache, "~/.aws/sso/cache/*", true},
		{"double-star glob", aws, "~/.aws/legacy/**", true},
		{"glob path not covered by file rules", HomeShellReadRules(".aws/sso/cache/*", false), "~/.aws/sso/cache/*", false},
		{"file rules do not cover a sibling", file, "~/.aws/config", false},
		{"sibling dir does not cover", HomeShellReadRules(".azure", true), "~/.aws/credentials", false},
		{"dir rules of a descendant do not cover", cache, "~/.aws/credentials", false},
		{"empty deny", nil, "~/.aws/credentials", false},
		{"not home-relative", join(file, aws), ".aws/credentials", false},
		{"bare glob", aws, "~/*", false},
	}
	for i := range file {
		tests = append(tests, struct {
			name string
			deny []string
			path string
			want bool
		}{"file set missing " + file[i], slices.Delete(slices.Clone(file), i, i+1), "~/.aws/credentials", false})
	}
	for i := range aws {
		tests = append(tests, struct {
			name string
			deny []string
			path string
			want bool
		}{"dir set missing " + aws[i], slices.Delete(slices.Clone(aws), i, i+1), "~/.aws/sso/cache/*", false})
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ShellReadCovered(tt.deny, tt.path); got != tt.want {
				t.Errorf("ShellReadCovered(%s) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

// countOf returns how many times s occurs in list.
func countOf(list []string, s string) int {
	n := 0
	for _, v := range list {
		if v == s {
			n++
		}
	}
	return n
}
