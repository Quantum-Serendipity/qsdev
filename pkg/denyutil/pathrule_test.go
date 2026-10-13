package denyutil

import "testing"

// TestMatchesPathRule pins MatchesPathRule to the documented Read and Edit
// rule semantics (code.claude.com/docs/en/permissions, "Read and Edit").
func TestMatchesPathRule(t *testing.T) {
	t.Parallel()

	const (
		home = "/home/me"
		root = "/work/proj"
	)
	deny := PathRuleContext{Home: home, ProjectRoot: root, Cwd: root, Deny: true}
	allow := PathRuleContext{Home: home, ProjectRoot: root, Cwd: root}
	sub := PathRuleContext{Home: home, ProjectRoot: root, Cwd: root + "/pkg", Deny: true}
	win := PathRuleContext{Home: `C:\Users\a`, ProjectRoot: `C:\work\proj`, Cwd: `C:\work\proj`, Deny: true}

	tests := []struct {
		name string
		rule string
		path string
		ctx  PathRuleContext
		want bool
	}{
		// ~/ anchors at the home directory.
		{"home file", "Read(~/.zshrc)", home + "/.zshrc", deny, true},
		{"home file not nested", "Read(~/.zshrc)", home + "/x/.zshrc", deny, false},
		{"home dir contents", "Read(~/.ssh/**)", home + "/.ssh/id_ed25519", deny, true},
		{"home dir deep contents", "Read(~/.ssh/**)", home + "/.ssh/a/b/c", deny, true},
		{"home dir other dir", "Read(~/.ssh/**)", home + "/.sshx/id", deny, false},
		{"home star within segment", "Read(~/Documents/*.pdf)", home + "/Documents/a.pdf", deny, true},
		{"home star not across segments", "Read(~/Documents/*.pdf)", home + "/Documents/a/b.pdf", deny, false},
		{"home rule outside home", "Read(~/.zshrc)", "/root/.zshrc", deny, false},

		// // anchors at the filesystem root.
		{"absolute any depth", "Read(//**/.env)", "/x/y/.env", deny, true},
		{"absolute at root", "Read(//**/.env)", "/.env", deny, true},
		{"absolute exact", "Edit(//tmp/scratch.txt)", "/tmp/scratch.txt", deny, true},
		{"absolute exact not nested", "Edit(//tmp/scratch.txt)", "/var/tmp/scratch.txt", deny, false},

		// / anchors at the settings source (the project root here).
		{"slash anchored allow", "Read(/src/**)", root + "/src/x", allow, true},
		{"slash anchored not nested", "Read(/src/**)", root + "/a/src/x", allow, false},
		{"slash anchored deny not nested", "Read(/src/**)", root + "/a/src/x", deny, false},
		{"slash is not filesystem root", "Read(/etc/passwd)", "/etc/passwd", deny, false},
		{"slash with cwd below root", "Edit(/docs/**)", root + "/docs/a.md", sub, true},

		// Bare and ./ patterns are relative to the current directory.
		{"bare filename at cwd", "Read(.env)", root + "/.env", deny, true},
		{"bare filename any depth", "Read(.env)", root + "/a/b/.env", deny, true},
		{"bare filename not in parent", "Read(.env)", "/work/.env", deny, false},
		{"bare filename allow any depth", "Read(.env)", root + "/a/b/.env", allow, true},
		{"dot slash filename any depth", "Read(./.env)", root + "/a/.env", deny, true},
		{"bare star any depth", "Read(*.env)", root + "/a/prod.env", deny, true},
		{"double star prefix", "Read(**/.env)", root + "/a/.env", deny, true},
		{"bare relative to cwd not root", "Read(.env)", root + "/.env", sub, false},
		{"bare relative to cwd below", "Read(.env)", root + "/pkg/a/.env", sub, true},
		{"multi segment anchored", "Edit(src/components/**)", root + "/src/components/a.tsx", deny, true},
		{"multi segment not nested", "Edit(src/components/**)", root + "/x/src/components/a.tsx", deny, false},

		// A single-segment directory pattern: any depth for deny and ask,
		// only at the anchor for allow.
		{"single segment deny nested", "Read(secrets/**)", root + "/internal/secrets/known_vars.go", deny, true},
		{"single segment allow nested", "Read(secrets/**)", root + "/internal/secrets/known_vars.go", allow, false},
		{"single segment allow at anchor", "Read(secrets/**)", root + "/secrets/a", allow, true},
		{"dot slash single segment deny nested", "Read(./secrets/**)", root + "/internal/secrets/x.go", deny, true},
		{"dot slash single segment allow nested", "Read(./secrets/**)", root + "/internal/secrets/x.go", allow, false},
		{"double star dir allow nested", "Edit(**/src/**)", root + "/vendor/pkg/src/lib.js", allow, true},
		{"dir star star excludes dir itself", "Read(secrets/**)", root + "/secrets", deny, false},

		// Windows paths are normalised to POSIX /c/... before matching.
		{"windows drive pattern", "Read(//c/**/.env)", `C:\Users\a\.env`, win, true},
		{"windows any drive", "Read(//**/.env)", `D:\x\.env`, win, true},
		{"windows other drive", "Read(//c/**/.env)", `D:\x\.env`, win, false},
		{"windows home", "Read(~/.ssh/**)", `C:\Users\a\.ssh\id_rsa`, win, true},
		{"windows cwd bare", "Read(.env)", `C:\work\proj\a\.env`, win, true},
		{"windows forward slashes", "Read(~/.ssh/**)", "C:/Users/a/.ssh/config", win, true},

		// Rules this oracle does not match.
		{"negation out of scope", "Read(!sample.env)", root + "/sample.env", deny, false},
		{"bash rule", "Bash(cat .env)", root + "/.env", deny, false},
		{"write rule never consulted", "Write(.env)", root + "/.env", deny, false},
		{"bare tool no path", "Read", root + "/.env", deny, false},
		{"relative path argument", "Read(.env)", ".env", deny, false},
		{"unusable pattern guards exact path on deny", "Read(//a/[b)", "/a/[b", deny, true},
		{"unusable pattern approves nothing", "Read(//a/[b)", "/a/[b", allow, false},
		{"cleaned path", "Read(~/.ssh/**)", home + "/x/../.ssh/id", deny, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := MatchesPathRule(tt.rule, tt.path, tt.ctx); got != tt.want {
				t.Errorf("MatchesPathRule(%q, %q, %+v) = %v, want %v", tt.rule, tt.path, tt.ctx, got, tt.want)
			}
		})
	}
}
