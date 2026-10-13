package validation

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/pathmatch"
)

// TestCheckBoundaryReadPath checks each rule against a fixed home directory,
// so the result does not depend on the machine running the test.
func TestCheckBoundaryReadPath(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		path    string
		wantErr string
	}{
		{"absolute", "/opt/sdk/src", ""},
		{"home relative", "~/.m2/repository", ""},
		{"trailing slash", "/opt/sdk/", ""},
		{"windows drive", `C:\tools\sdk`, ""},
		{"windows forward slash", "D:/sdk", ""},
		{"empty", "", "empty"},
		{"relative", "vendor/src", "absolute"},
		{"dot relative", "./x", "absolute"},
		{"unknown user tilde", "~bob/src", "absolute"},
		{"variable", "$HOME/src", "absolute"},
		{"root", "/", "root"},
		{"double slash root", "//", "root"},
		{"home", "~", "root"},
		{"home slash", "~/", "root"},
		{"windows drive root", `C:\`, "root"},
		{"dotdot", "/opt/../etc", `".."`},
		{"home dotdot", "~/../other", `".."`},
		{"comma", "/opt/a,/etc", "comma"},
		{"newline", "/opt/a\n/etc", "control"},
		{"nul", "/opt/a\u0000", "control"},
		{"leading space", " /opt/a", "whitespace"},
		{"too long", "/" + strings.Repeat("a", maxReadPathLen), "too long"},
		{"home absolute", "/home/u", "home directory"},
		{"home parent", "/home", "home directory"},
		{"home sibling", "/home/other/sdk", ""},
		{"inside home", "/home/u/sdks/android", ""},
		{"home credential store", "~/.ssh", "credential store"},
		{"inside home credential store", "~/.aws/sso", "credential store"},
		{"contains home credential store", "~/.config", "credential store"},
		{"absolute home credential store", "/home/u/.gnupg", "credential store"},
		{"absolute contains credential store", "/home/u/.docker", "credential store"},
		{"system credential store", "/etc", "credential store"},
		{"system credential file", "/etc/sudoers.d/x", "credential store"},
		{"credential lookalike", "~/.sshfoo", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := checkBoundaryReadPath(tc.path, "/home/u", pathmatch.Options{})
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("checkBoundaryReadPath(%q) = %v, want nil", tc.path, err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("checkBoundaryReadPath(%q) = %v, want error containing %q", tc.path, err, tc.wantErr)
			}
		})
	}
}

// TestCheckBoundaryReadPath_FilesystemAliases checks that on a case-folding
// filesystem (macOS, Windows), and one that ignores Windows name aliases,
// other spellings of the home directory or a credential store are rejected
// too, while an exact-name filesystem still allows them.
func TestCheckBoundaryReadPath_FilesystemAliases(t *testing.T) {
	t.Parallel()
	fold := pathmatch.Options{FoldCase: true}
	windows := pathmatch.Options{FoldCase: true, WindowsAliases: true}
	cases := []struct {
		name    string
		path    string
		opts    pathmatch.Options
		wantErr string
	}{
		{"case-folded home ancestor", "/USERS/me", fold, "home directory"},
		{"case-folded home", "/Users/ME", fold, "home directory"},
		{"case-folded absolute store", "/Users/Me/.ssh", fold, "credential store"},
		{"case-folded relative store", "~/.SSH", fold, "credential store"},
		{"case-folded store file", "~/.Ssh/config", fold, "credential store"},
		{"trailing-dot store", "~/.ssh./", windows, "credential store"},
		{"stream-suffixed store", "~/.ssh::$DATA", windows, "credential store"},
		{"windows home ancestor", `C:\USERS`, windows, "home directory"},
		// Deliberately rejected even on an exact-name host: the committed
		// .qsdev.yaml is shared across hosts, and on macOS or Windows "~/.SSH"
		// is the credential store "~/.ssh". (Allowed before U19-WS5.)
		{"exact-name filesystem", "~/.SSH", pathmatch.Options{}, "credential store"},
		{"case-folded lookalike", "~/.SSHfoo", fold, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			home := "/Users/me"
			if strings.HasPrefix(tc.path, "C:") {
				home = "C:/Users/me"
			}
			err := checkBoundaryReadPath(tc.path, home, tc.opts)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("checkBoundaryReadPath(%q) = %v, want nil", tc.path, err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("checkBoundaryReadPath(%q) = %v, want error containing %q", tc.path, err, tc.wantErr)
			}
		})
	}
}

// matchOptionSets is every way a host filesystem may compare names. The
// credential overlap check must not depend on it, since the committed
// .qsdev.yaml is read on every host.
var matchOptionSets = map[string]pathmatch.Options{
	"exact":    {},
	"platform": pathmatch.Platform,
	"fold":     {FoldCase: true},
	"windows":  {FoldCase: true, WindowsAliases: true},
}

// TestCheckBoundaryReadPath_RejectsCredentialVariants checks that every
// spelling of a credential store, and every path that contains or lies inside
// one, is rejected whatever the host's filesystem semantics.
func TestCheckBoundaryReadPath_RejectsCredentialVariants(t *testing.T) {
	t.Parallel()
	paths := []string{
		"~/.SSH",
		"~/.ssh.",
		"~/.ssh::$DATA",
		"~/.npmrc",
		"~/.config/gh",
		"~/.config/gh/hosts.yml",
		"~/.git-credentials",
		"~/.claude",
		"~/.m2",
		"~/.pypirc",
		"/home/u/.NPMRC",
		"/home/u/.config/GH/hosts.yml",
		"/ETC/shadow",
	}
	for optName, opts := range matchOptionSets {
		for _, p := range paths {
			t.Run(optName+"/"+p, func(t *testing.T) {
				t.Parallel()
				err := checkBoundaryReadPath(p, "/home/u", opts)
				if err == nil || !strings.Contains(err.Error(), "credential store") {
					t.Fatalf("checkBoundaryReadPath(%q) = %v, want a credential store error", p, err)
				}
			})
		}
	}
}

// TestCheckBoundaryReadPath_CredentialMessageUsesTilde checks that the error
// names the store as the user wrote it: home-relative as "~/<rel>", absolute
// under the home directory as "<home>/<rel>".
func TestCheckBoundaryReadPath_CredentialMessageUsesTilde(t *testing.T) {
	t.Parallel()
	cases := []struct {
		path string
		want string
	}{
		{"~/.config/gh", "overlaps the credential store ~/.config/gh"},
		{"~/.config/gh/hosts.yml", "overlaps the credential store ~/.config/gh"},
		{"~/.npmrc", "overlaps the credential store ~/.npmrc"},
		{"/home/u/.config/gh", "overlaps the credential store /home/u/.config/gh"},
		{"/etc/sudoers.d/x", "overlaps the credential store /etc/sudoers.d"},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()
			err := checkBoundaryReadPath(tc.path, "/home/u", pathmatch.Options{})
			if err == nil || err.Error() != "path "+tc.want {
				t.Fatalf("checkBoundaryReadPath(%q) = %v, want %q", tc.path, err, tc.want)
			}
		})
	}
}

// TestCheckBoundaryReadPath_StillAllowed pins the paths that stay valid after
// the credential canon grew and the overlap check became cross-host: tool
// caches beside a credential file, paths outside home, and home-relative
// paths when home is /root (a system deny entry), while an absolute path
// under /root is still rejected for another user.
func TestCheckBoundaryReadPath_StillAllowed(t *testing.T) {
	t.Parallel()
	allowed := []string{
		"~/sdks/android",
		"/opt/sdk",
		"~/.m2/repository",
		"~/.gradle/caches",
		"/home/other/sdk",
		"~/.sshfoo",
	}
	for _, home := range []string{"/root", "/home/u"} {
		for optName, opts := range matchOptionSets {
			for _, p := range allowed {
				t.Run(home+"/"+optName+"/"+p, func(t *testing.T) {
					t.Parallel()
					if err := checkBoundaryReadPath(p, home, opts); err != nil {
						t.Fatalf("checkBoundaryReadPath(%q) with home %s = %v, want nil", p, home, err)
					}
				})
			}
		}
	}
	if err := checkBoundaryReadPath("/root/sdks", "/home/u", pathmatch.Options{}); err == nil {
		t.Fatal(`checkBoundaryReadPath("/root/sdks") with home /home/u = nil, want an error`)
	}
}
