package validation

import (
	"strings"
	"testing"
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
			err := checkBoundaryReadPath(tc.path, "/home/u")
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
