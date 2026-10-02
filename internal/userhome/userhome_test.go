package userhome

import (
	"os/user"
	"path/filepath"
	"testing"
)

// TestAccountIgnoresHomeEnvironment pins that the account's home directory
// does not follow HOME or USERPROFILE: a line such as `HOME=/tmp/e qsdev
// claude update` must not move what Account and Dir return.
func TestAccountIgnoresHomeEnvironment(t *testing.T) {
	want, err := Account()
	if err != nil {
		t.Skipf("the current account has no user database entry here: %v", err)
	}
	for _, name := range []string{"HOME", "USERPROFILE"} {
		t.Run(name, func(t *testing.T) {
			fake := filepath.Join(t.TempDir(), "elsewhere")
			t.Setenv(name, fake)
			got, err := Account()
			if err != nil || got != want {
				t.Errorf("Account() with %s=%s = %q, %v; want %q", name, fake, got, err, want)
			}
		})
	}
}

// TestAccountFailsWithoutEntry pins that Account fails, rather than reading
// the environment, when the account has no usable user database entry.
func TestAccountFailsWithoutEntry(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	tests := []struct {
		name   string
		lookup func() (*user.User, error)
	}{
		{"no entry", func() (*user.User, error) { return nil, user.UnknownUserIdError(4242) }},
		{"entry without home", func() (*user.User, error) { return &user.User{Uid: "4242"}, nil }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orig := lookup
			lookup = tt.lookup
			t.Cleanup(func() { lookup = orig })
			if home, err := Account(); err == nil {
				t.Errorf("Account() = %q without a usable entry, want an error", home)
			}
		})
	}
}
