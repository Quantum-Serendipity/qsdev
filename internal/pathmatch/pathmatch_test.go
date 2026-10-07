package pathmatch

import (
	"runtime"
	"strings"
	"testing"
)

// The four Options combinations. Tests drive them explicitly, so the macOS
// and Windows comparisons run on every OS.
var (
	exact   = Options{}
	fold    = Options{FoldCase: true}
	aliases = Options{WindowsAliases: true}
	windows = Options{FoldCase: true, WindowsAliases: true}
)

func TestKey_Options(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts Options
		a, b string
		same bool
	}{
		{"exact keeps case", exact, "/Users/me", "/users/me", false},
		{"exact keeps aliases", exact, "/a/file.txt. ", "/a/file.txt", false},
		{"exact equal", exact, "/a/b", "/a/b", true},
		{"fold home", fold, "/users/me", "/Users/me", true},
		{"fold drive", fold, "c:/users/me", "C:/Users/me", true},
		{"fold keeps aliases", fold, "/a/file.txt::$DATA", "/a/file.txt", false},
		{"aliases keep case", aliases, "/Users/me", "/users/me", false},
		{"aliases trailing dot space", aliases, "/a/file.txt. ", "/a/file.txt", true},
		{"aliases data stream", aliases, "/a/file.txt::$DATA", "/a/file.txt", true},
		{"windows drive", windows, "c:/users/me", "C:/Users/me", true},
		{"windows trailing dot space", windows, "C:/A/File.txt. ", "c:/a/file.txt", true},
		{"windows data stream", windows, "C:/A/File.txt::$DATA", "c:/a/file.txt", true},
		{"windows keeps drive", windows, "C:/a", "D:/a", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ka, kb := tt.opts.Key(tt.a), tt.opts.Key(tt.b)
			if (ka == kb) != tt.same {
				t.Errorf("%+v: Key(%q) = %q, Key(%q) = %q; same = %v, want %v", tt.opts, tt.a, ka, tt.b, kb, ka == kb, tt.same)
			}
		})
	}
}

func TestPlatform(t *testing.T) {
	t.Parallel()
	wantFold := runtime.GOOS == "darwin" || runtime.GOOS == "windows"
	if Platform.FoldCase != wantFold || CaseInsensitiveFS() != wantFold {
		t.Errorf("Platform.FoldCase = %v, CaseInsensitiveFS() = %v on %s, want %v", Platform.FoldCase, CaseInsensitiveFS(), runtime.GOOS, wantFold)
	}
	if Platform.WindowsAliases != (runtime.GOOS == "windows") {
		t.Errorf("Platform.WindowsAliases = %v on %s", Platform.WindowsAliases, runtime.GOOS)
	}
	if Key("/a/B") != Platform.Key("/a/B") {
		t.Error("Key differs from Platform.Key")
	}
}

func TestWithin_CaseFold(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		opts    Options
		p, root string
		want    bool
	}{
		{"fold upper below lower", fold, "/home/me/.SSH", "/home/me/.ssh", true},
		{"fold descendant", fold, "/home/me/.SSH/id_rsa", "/home/me/.ssh", true},
		{"exact upper not within lower", exact, "/home/me/.SSH", "/home/me/.ssh", false},
		{"exact descendant of upper", exact, "/home/me/.SSH/id_rsa", "/home/me/.ssh", false},
		{"fold home case", fold, "/users/me/x", "/Users/me", true},
		{"windows drive case", windows, "c:/users/me/.ssh", "C:/Users/me", true},
		{"windows alias", windows, "C:/Users/me/.ssh./id_rsa::$DATA", "c:/users/me/.ssh", true},
		{"aliases off keep trailing dot", fold, "/home/me/.ssh./id_rsa", "/home/me/.ssh", false},
		{"equal", exact, "/a/b", "/a/b", true},
		{"component boundary", exact, "/a/bc", "/a/b", false},
		{"fold component boundary", fold, "/A/BC", "/a/b", false},
		{"sibling", exact, "/a/c", "/a/b", false},
		{"ancestor is not within", exact, "/a", "/a/b", false},
		{"filesystem root", exact, "/etc/shadow", "/", true},
		{"root itself", exact, "/", "/", true},
		{"drive root", windows, "C:/x", "c:/", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.opts.Within(tt.p, tt.root); got != tt.want {
				t.Errorf("%+v.Within(%q, %q) = %v, want %v", tt.opts, tt.p, tt.root, got, tt.want)
			}
		})
	}
}

func TestStrictlyWithin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		opts    Options
		p, root string
		want    bool
	}{
		{"descendant", exact, "/home/me/.ssh", "/home/me", true},
		{"equal is not strict", exact, "/home/me", "/home/me", false},
		{"fold equal is not strict", fold, "/HOME/me", "/home/ME", false},
		{"fold descendant", fold, "/HOME/me/.SSH", "/home/me", true},
		{"exact differently cased", exact, "/HOME/me/.ssh", "/home/me", false},
		{"root contains everything", exact, "/home", "/", true},
		{"root is not strictly within root", exact, "/", "/", false},
		{"component boundary", exact, "/home/meX", "/home/me", false},
		{"ancestor", exact, "/home", "/home/me", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.opts.StrictlyWithin(tt.p, tt.root); got != tt.want {
				t.Errorf("%+v.StrictlyWithin(%q, %q) = %v, want %v", tt.opts, tt.p, tt.root, got, tt.want)
			}
		})
	}
	if StrictlyWithin("/a", "/a") || !StrictlyWithin("/a/b", "/a") || !Within("/a", "/a") {
		t.Error("package-level Within/StrictlyWithin disagree with Platform")
	}
}

// TestKey_Allocs pins the cost of a key on every OS: none for a path already
// in key form, and a single copy for one that must be folded, so the
// self-protection and sandbox checks that key every path cost the same on
// Windows as on Linux. Not parallel: AllocsPerRun counts every allocation in
// the process.
func TestKey_Allocs(t *testing.T) {
	tests := []struct {
		opts Options
		p    string
		max  float64
	}{
		{exact, "/home/me/Project/Src/Main.go", 0},
		{fold, "/home/me/project/src/main.go", 0},
		{windows, "c:/users/me/project/src/main.go", 0},
		{fold, "/home/me/Project/Src/Main.go", 1},
		{windows, "C:/Users/me/Project/Src/Main.go", 1},
	}
	for _, tt := range tests {
		if got := testing.AllocsPerRun(100, func() { tt.opts.Key(tt.p) }); got > tt.max {
			t.Errorf("%+v.Key(%q) made %v allocations, want at most %v", tt.opts, tt.p, got, tt.max)
		}
	}
}

// TestWithin_Allocs pins that a containment check costs only the keys of its
// two paths: nothing for paths already in key form, and no root+"/" copy. Not
// parallel: AllocsPerRun counts every allocation in the process.
func TestWithin_Allocs(t *testing.T) {
	tests := []struct {
		opts    Options
		p, root string
		max     float64
	}{
		{exact, "/home/me/.ssh/id_rsa", "/home/me/.ssh", 0},
		{windows, "c:/users/me/.ssh/id_rsa", "c:/users/me/.ssh", 0},
		{fold, "/home/me/.SSH/id_rsa", "/home/me/.ssh", 1},
		{windows, "C:/Users/me/.SSH/id_rsa", "C:/Users/me/.ssh", 2},
	}
	for _, tt := range tests {
		if got := testing.AllocsPerRun(100, func() { tt.opts.Within(tt.p, tt.root) }); got > tt.max {
			t.Errorf("%+v.Within(%q, %q) made %v allocations, want at most %v", tt.opts, tt.p, tt.root, got, tt.max)
		}
		if got := testing.AllocsPerRun(100, func() { tt.opts.StrictlyWithin(tt.p, tt.root) }); got > tt.max {
			t.Errorf("%+v.StrictlyWithin(%q, %q) made %v allocations, want at most %v", tt.opts, tt.p, tt.root, got, tt.max)
		}
	}
}

func TestStripWindowsAliases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in, want string
	}{
		{"C:/Users/u/.claude/settings.json::$DATA", "C:/Users/u/.claude/settings.json"},
		{"C:/Users/u/.claude./settings.json. ", "C:/Users/u/.claude/settings.json"},
		{"C:/a/../b/./c", "C:/a/../b/./c"},
		{"//server/share/.claude/hooks/x.sh:stream", "//server/share/.claude/hooks/x.sh"},
		{"", ""},
		{"/", "/"},
		{"a/b/", "a/b/"},
		{"a./b", "a/b"},
		{"a/b.", "a/b"},
		{".../x", "/x"},
		{"file.txt. ", "file.txt"},
		{"file.txt::$DATA", "file.txt"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			if got := stripWindowsAliases(tt.in); got != tt.want {
				t.Errorf("stripWindowsAliases(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestStripWindowsAliases_MatchesReference checks every string of up to six
// characters over the bytes the stripping treats specially against the
// original split-and-join implementation.
func TestStripWindowsAliases_MatchesReference(t *testing.T) {
	t.Parallel()

	const alphabet = "a.: /"
	var walk func(s string)
	walk = func(s string) {
		if got, want := stripWindowsAliases(s), stripWindowsAliasesReference(s); got != want {
			t.Errorf("stripWindowsAliases(%q) = %q, want %q", s, got, want)
		}
		if len(s) == 6 {
			return
		}
		for _, c := range alphabet {
			walk(s + string(c))
		}
	}
	walk("")
}

// stripWindowsAliasesReference is the original stripWindowsAliases, which
// split every path into components and joined them again.
func stripWindowsAliasesReference(s string) string {
	parts := strings.Split(s, "/")
	for i, part := range parts {
		if part == "." || part == ".." || (len(part) == 2 && part[1] == ':') {
			continue
		}
		if j := strings.IndexByte(part, ':'); j >= 0 {
			part = part[:j]
		}
		parts[i] = strings.TrimRight(part, ". ")
	}
	return strings.Join(parts, "/")
}

// TestSepToSlashLower_MatchesTwoStepKey pins that converting separators and
// folding case in one pass gives the key the two separate steps gave, and
// that folding before stripping Windows aliases does not change the key.
func TestSepToSlashLower_MatchesTwoStepKey(t *testing.T) {
	t.Parallel()
	paths := []string{
		"",
		`C:\Users\RUNNER~1\AppData\Local\Temp\Project\Src\Main.go`,
		`C:\Repo\.CLAUDE.\Settings.JSON::$DATA`,
		`\\server\Share\.Claude \hooks\`,
		"/home/Alice/.Config/QSDEV/defaults.yaml",
		"mixed/Sep\\Path/ÄÖÜ/İstanbul",
		"already/lower/case",
	}
	for _, sep := range []rune{'/', '\\'} {
		for _, p := range paths {
			slashed := strings.ReplaceAll(p, string(sep), "/")
			if want, got := strings.ToLower(slashed), sepToSlashLower(p, sep); got != want {
				t.Errorf("sepToSlashLower(%q, %q) = %q, want %q", p, sep, got, want)
			}
			oldOrder := strings.ToLower(stripWindowsAliases(slashed))
			newOrder := stripWindowsAliases(sepToSlashLower(p, sep))
			if oldOrder != newOrder {
				t.Errorf("key(%q, sep %q): strip-then-fold %q, fold-then-strip %q", p, sep, oldOrder, newOrder)
			}
		}
	}
}

// TestSepToSlashLower_OneCopy pins that a Windows path is converted and
// folded in a single allocation, so a key costs the same on Windows as on
// Linux (filepath.ToSlash followed by strings.ToLower copied it twice there).
// Not parallel: AllocsPerRun counts every allocation in the process.
func TestSepToSlashLower_OneCopy(t *testing.T) {
	p := `C:\Users\RUNNER~1\AppData\Local\Temp\Project\Src\Main.go`
	if got := testing.AllocsPerRun(100, func() { sepToSlashLower(p, '\\') }); got > 1 {
		t.Errorf("sepToSlashLower(%q) made %v allocations, want at most 1", p, got)
	}
}

// BenchmarkSepToSlashLower measures folding a long Windows path, the work a
// long cd chain repeats for every command it scans.
func BenchmarkSepToSlashLower(b *testing.B) {
	p := `C:\Users\RUNNER~1\AppData\Local\Temp\` + strings.Repeat(`Missing\Dir\`, 200)
	b.ReportAllocs()
	for b.Loop() {
		sepToSlashLower(p, '\\')
	}
}
