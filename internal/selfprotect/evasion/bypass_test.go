package evasion

import "testing"

// TestCheck_HardlinkVariants covers the hard-link forms other than `ln`: a hard
// link gives a protected file an unprotected alias that later writes go
// through, so `link`, `cp -l`/`cp --link` and `rsync --link-dest` must be
// blocked just like a plain `ln`.
func TestCheck_HardlinkVariants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		command string
		blocked bool
	}{
		{"cp -l", "cp -l ~/.claude/settings.json ./x", true},
		{"cp --link", "cp --link ~/.claude/settings.json ./x", true},
		{"cp --link abbreviated", "cp --li ~/.claude/settings.json ./x", true},
		{"cp -al cluster", "cp -al ~/.claude/ ./x", true},
		{"cp -l onto protected dest", "cp -l ./evil .claude/hooks/pre.sh", true},
		{"link verb", "link ~/.claude/settings.json ./x", true},
		{"link behind sudo", "sudo link ~/.claude/settings.json ./x", true},
		{"link behind sh -c", "sh -c 'link ~/.claude/settings.json ./x'", true},
		{"link script piped to a shell", "echo 'link ~/.claude/settings.json ./x' | sh", true},
		{"link in unparseable command", "link ~/.claude/settings.json './x", true},
		{"rsync --link-dest=", "rsync -a --link-dest=/home/u/.claude/ src/ ./x", true},
		{"rsync --link-dest value", "rsync -a --link-dest /home/u/.claude src/ ./x", true},

		{"plain cp backup is left to the copy rule", "cp ~/.claude/settings.json ./backup", false},
		{"cp -l of unprotected files", "cp -l /tmp/a /tmp/b", false},
		{"link of unprotected files", "link /tmp/a /tmp/b", false},
		{"readlink is not link", "readlink .claude/hooks/pre.sh", false},
		{"link as a read-only search term", "git log --grep link -- .claude/settings.json", false},
		{"rsync without link-dest", "rsync -a src/ ./x && cat .claude/settings.json", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			blocked, category, reason := Check("Bash", tt.command, "")
			if blocked != tt.blocked {
				t.Errorf("Check(%q) blocked = %v, want %v (category %q, reason %q)",
					tt.command, blocked, tt.blocked, category, reason)
			}
			if tt.blocked && category != "hardlink" {
				t.Errorf("Check(%q) category = %q, want hardlink", tt.command, category)
			}
		})
	}
}

// TestCheck_RunTimeComputedCode verifies that code computed at run time is
// caught: eval of a backtick command substitution (it contains no `$`, so the
// old `eval.*\$` gate never consulted the parse and the decoded payload ran),
// and a shell -c script built from an expansion.
func TestCheck_RunTimeComputedCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		command string
		blocked bool
	}{
		{"backtick substitution", "eval `printf rm` x", true},
		{"backtick base64 payload", "eval `echo cm0gLi5jbGF1ZGUvc2V0dGluZ3MuanNvbg== | base64 -d`", true},
		{"backtick inside double quotes", "eval \"`cat /tmp/payload`\"", true},
		{"unparseable backtick eval", "eval `printf rm", true},
		{"backtick text in a read-only grep", "grep 'eval `x`' notes.md", false},
		{"eval text behind a mutating git", `git rm x && grep 'eval "$("' f`, true},
		{"eval text behind a read-only git", `git diff && grep 'eval "$("' f`, false},

		// A shell -c script computed at run time hides its code like eval does.
		{"bash -c of a decoded payload", `bash -c "$(echo cm0gLi5jbGF1ZGUvc2V0dGluZ3MuanNvbg== | base64 -d)"`, true},
		{"sh -c of a backtick payload", "sh -c \"`cat /tmp/payload`\"", true},
		{"clustered -ec flag", `bash -ec "$X"`, true},
		{"shell named by path", `/bin/bash -c "$(echo cm0= | base64 -d)"`, true},
		{"shell behind env", `env bash -c "$(echo cm0= | base64 -d)"`, true},
		{"shell behind sudo with options", `sudo -u root sh -c "$X"`, true},
		{"eval text behind a computed command word", `E=ev; $E 'eval "$X"'`, true},
		{"literal sh -c script", `sh -c 'go test ./...'`, false},
		{"shell script with an expanded argument", `bash ./build.sh "$TARGET"`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			blocked, category, reason := Check("Bash", tt.command, "")
			if blocked != tt.blocked {
				t.Errorf("Check(%q) blocked = %v, want %v (category %q, reason %q)",
					tt.command, blocked, tt.blocked, category, reason)
			}
			if tt.blocked && category != "obfuscation" {
				t.Errorf("Check(%q) category = %q, want obfuscation", tt.command, category)
			}
		})
	}
}
