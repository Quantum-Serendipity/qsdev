package claudecode_test

import (
	"path/filepath"
	"testing"
)

// TestHookLibExpandHome pins how the hooks expand a leading ~: ~ and
// ~<current user> resolve to the home directory, another user's ~name is
// unresolved (nil), and no case asks the user database about a name. That
// lookup blocked file-boundary for 8 s on a macOS runner for ~nobody-qsdev
// and outlasted the hook's deadline.
func TestHookLibExpandHome(t *testing.T) {
	t.Parallel()
	var got struct {
		Lookups []string           `json:"lookups"`
		Home    string             `json:"home"`
		Me      string             `json:"me"`
		Out     map[string]*string `json:"out"`
	}
	hookLibEval(t, `
lookups = []
try:
    import pwd
    def _lookup(name):
        lookups.append(name)
        raise KeyError(name)
    pwd.getpwnam = _lookup
except ImportError:  # Windows: no user database module
    pass
me = lib._current_user()
paths = ["~", "~/x/y", "~nobody-qsdev", "~nobody-qsdev/x", "src/a.txt", "/abs/~x", "$HOME/x"]
if me:
    paths += ["~" + me, "~" + me + "/x"]
result = {"lookups": lookups, "home": os.path.expanduser("~"), "me": me,
          "out": {p: lib.expand_home(p) for p in paths}}
`, &got)

	if len(got.Lookups) > 0 {
		t.Errorf("expand_home looked up users %q in the user database", got.Lookups)
	}
	str := func(s string) *string { return &s }
	want := map[string]*string{
		"~":               str(got.Home),
		"~/x/y":           str(got.Home + "/x/y"),
		"~nobody-qsdev":   nil,
		"~nobody-qsdev/x": nil,
		"src/a.txt":       str("src/a.txt"),
		"/abs/~x":         str("/abs/~x"),
		"$HOME/x":         str("$HOME/x"),
	}
	if got.Me != "" {
		want["~"+got.Me] = str(got.Home)
		want["~"+got.Me+"/x"] = str(got.Home + "/x")
	}
	for path, w := range want {
		g, ok := got.Out[path]
		switch {
		case !ok:
			t.Errorf("expand_home(%q) not evaluated", path)
		case w == nil && g != nil:
			t.Errorf("expand_home(%q) = %q, want None (another user's home is unresolved)", path, *g)
		case w != nil && (g == nil || filepath.ToSlash(*g) != filepath.ToSlash(*w)):
			t.Errorf("expand_home(%q) = %v, want %q", path, g, *w)
		}
	}
}
