package cmdscan

import (
	"reflect"
	"strings"
	"testing"
)

func TestToolDialect(t *testing.T) {
	t.Parallel()
	cases := map[string]Dialect{
		"Bash":       POSIX,
		"Monitor":    POSIX,
		"PowerShell": PowerShell,
		"powershell": POSIX, // tool names are exact
		"Write":      POSIX,
		"":           POSIX,
	}
	for tool, want := range cases {
		if got := ToolDialect(tool); got != want {
			t.Errorf("ToolDialect(%q) = %v, want %v", tool, got, want)
		}
	}
	// Every shell tool has a dialect, and exactly one is PowerShell.
	ps := 0
	for _, tool := range ShellTools {
		if ToolDialect(tool) == PowerShell {
			ps++
		}
	}
	if ps != 1 {
		t.Errorf("%d ShellTools map to the PowerShell dialect, want 1", ps)
	}
}

func TestParsePowerShell(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		text string
		want []PSCommand
	}{
		{name: "simple", text: `Remove-Item .claude\settings.json`,
			want: []PSCommand{{Name: "remove-item", Args: []string{`.claude\settings.json`}}}},
		{name: "case folded name only", text: `REMOVE-ITEM .CLAUDE\X`,
			want: []PSCommand{{Name: "remove-item", Args: []string{`.CLAUDE\X`}}}},
		{name: "single quotes with doubled quote", text: `Set-Content -Path 'a b''c' -Value '{}'`,
			want: []PSCommand{{Name: "set-content", Args: []string{"-Path", "a b'c", "-Value", "{}"}}}},
		{name: "double quotes with backtick and doubled quote", text: "Write-Host \"a`\"b\"\"c; d\"",
			want: []PSCommand{{Name: "write-host", Args: []string{`a"b"c; d`}}}},
		{name: "backtick escape outside quotes", text: "gc a`;b",
			want: []PSCommand{{Name: "gc", Args: []string{"a;b"}}}},
		{name: "backtick line continuation", text: "gc `\n x",
			want: []PSCommand{{Name: "gc", Args: []string{"x"}}}},
		{name: "concatenated quote spellings", text: `ri .cl''aude".x"`,
			want: []PSCommand{{Name: "ri", Args: []string{".claude.x"}}}},
		{name: "literal here-string", text: "Set-Content x @'\nri .claude; $(y)\n'@\ngc z",
			want: []PSCommand{
				{Name: "set-content", Args: []string{"x", "ri .claude; $(y)"}},
				{Name: "gc", Args: []string{"z"}},
			}},
		{name: "expandable here-string", text: "Write-Host @\"\nhi\n\"@",
			want: []PSCommand{{Name: "write-host", Args: []string{"hi"}}}},
		{name: "separators", text: "a 1; b 2 | c (d 3) {e} && f || g\nh",
			want: []PSCommand{
				{Name: "a", Args: []string{"1"}},
				{Name: "b", Args: []string{"2"}},
				{Name: "c"},
				{Name: "d", Args: []string{"3"}},
				{Name: "e"},
				{Name: "f"},
				{Name: "g"},
				{Name: "h"},
			}},
		{name: "call operators", text: `& 'C:\Program Files\Git\bin\git.EXE' status; . .\setup.ps1`,
			want: []PSCommand{
				{Name: "git", Args: []string{"status"}},
				{Name: "setup.ps1"},
			}},
		{name: "path qualified exe", text: `C:/tools/QSDEV.exe x`,
			want: []PSCommand{{Name: "qsdev", Args: []string{"x"}}}},
		{name: "redirect separate target", text: `gc x > .claude\settings.json`,
			want: []PSCommand{{Name: "gc", Args: []string{"x"}, WriteRedirects: []string{`.claude\settings.json`}}}},
		{name: "redirect attached target", text: `gc x>>y 2>z *>w`,
			want: []PSCommand{{Name: "gc", Args: []string{"x"}, WriteRedirects: []string{"y", "z", "w"}}}},
		{name: "null and stream merges are not writes", text: `gc x 2>$null 2>&1 *>$NULL 3>&2`,
			want: []PSCommand{{Name: "gc", Args: []string{"x"}}}},
		{name: "quoted redirect target", text: `gc x > 'a b'`,
			want: []PSCommand{{Name: "gc", Args: []string{"x"}, WriteRedirects: []string{"a b"}}}},
		{name: "redirect only", text: `> f`,
			want: []PSCommand{{WriteRedirects: []string{"f"}}}},
		{name: "subexpression in double quotes runs code", text: `gc "$(ri x)"`,
			want: []PSCommand{{Name: "gc", Args: []string{"$(ri x)"}, Subexpression: true}}},
		{name: "escaped subexpression is literal", text: "gc \"`$(ri x)\"",
			want: []PSCommand{{Name: "gc", Args: []string{"$(ri x)"}}}},
		{name: "bare subexpression is its own command", text: `gc $(ri x)`,
			want: []PSCommand{{Name: "gc", Args: []string{"$"}}, {Name: "ri", Args: []string{"x"}}}},
		{name: "line comment", text: "gc x # it's ; ri y\nri z",
			want: []PSCommand{{Name: "gc", Args: []string{"x"}}, {Name: "ri", Args: []string{"z"}}}},
		{name: "block comment", text: "gc x <# ' ri y #> z",
			want: []PSCommand{{Name: "gc", Args: []string{"x", "z"}}}},
		{name: "hash inside a word", text: "gc a#b",
			want: []PSCommand{{Name: "gc", Args: []string{"a#b"}}}},
		{name: "smart quotes close a string", text: "gc \"x\u201d; ri .claude",
			want: []PSCommand{{Name: "gc", Args: []string{"x"}}, {Name: "ri", Args: []string{".claude"}}}},
		{name: "smart single quotes", text: "gc \u2018a\u2019\u2019b\u2019",
			want: []PSCommand{{Name: "gc", Args: []string{"a\u2019b"}}}},
		{name: "empty", text: " ;; \n ", want: nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := ParsePowerShell(tc.text); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("ParsePowerShell(%q)\n got %#v\nwant %#v", tc.text, got, tc.want)
			}
		})
	}
}

func TestPSCommandIsRead(t *testing.T) {
	t.Parallel()
	cases := []struct {
		text string
		want bool
	}{
		{"Get-Content .claude\\settings.json", true},
		{"gc x 2>$null", true},
		{"Get-ChildItem .claude", true},
		{"Select-String -Path x -Pattern y", true},
		{"Test-Path x", true},
		{"Get-Content x > y", false},
		{"Set-Content x y", false},
		{"Remove-Item x", false},
		{"git status", true},
		{"git diff .claude\\settings.json", true},
		{"git log -- .claude", true},
		{"git diff --output=x", false},
		{"git status > x", false},
		{"git commit -m x", false},
		{"gc \"$(ri x)\"", false},
		{"> f", false},
	}
	for _, tc := range cases {
		cmds := ParsePowerShell(tc.text)
		if len(cmds) != 1 {
			t.Fatalf("ParsePowerShell(%q) = %d commands, want 1", tc.text, len(cmds))
		}
		if got := cmds[0].IsRead(); got != tc.want {
			t.Errorf("IsRead(%q) = %v, want %v", tc.text, got, tc.want)
		}
	}
}

func TestPSParamName(t *testing.T) {
	t.Parallel()
	cases := []struct {
		word, name string
		ok         bool
	}{
		{"-EncodedCommand", "encodedcommand", true},
		{"--enc", "enc", true},
		{"/EC", "ec", true},
		{"-Verb:RunAs", "verb", true},
		{"\u2013enc", "enc", true},
		{"\u2014Verb", "verb", true},
		{"\u2015ec", "ec", true},
		{"-", "", false},
		{"enc", "", false},
	}
	for _, tc := range cases {
		if name, ok := PSParamName(tc.word); name != tc.name || ok != tc.ok {
			t.Errorf("PSParamName(%q) = (%q, %v), want (%q, %v)", tc.word, name, ok, tc.name, tc.ok)
		}
	}
	if !IsPSParamPrefix("-enc", "encodedcommand") || IsPSParamPrefix("-Verbose", "verb") {
		t.Error("IsPSParamPrefix: want -enc to abbreviate encodedcommand and -Verbose not to abbreviate verb")
	}
}

func TestPowerShellText(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, norm, loose string
	}{
		{`Remove-Item .CLAUDE\Settings.json`, `remove-item .claude/settings.json`, `remove-item .claude/settings.json`},
		{`$p='.cl'+'aude'; ri "$p\settings.json"`, `$p='.cl'+'aude'; ri "$p/settings.json"`, `$p=.claude; ri $p/settings.json`},
		{`$p = '.cl' + 'aude'`, `$p = '.cl' + 'aude'`, `$p = .claude`},
		{"ri .cl`aude", "ri .cl`aude", "ri .claude"},
		{"ri \u2018.cl\u2019\u201c\u201daude", "ri \u2018.cl\u2019\u201c\u201daude", "ri .claude"},
	}
	for _, tc := range cases {
		norm, loose := PowerShellText(tc.in)
		if norm != tc.norm || loose != tc.loose {
			t.Errorf("PowerShellText(%q) = (%q, %q), want (%q, %q)", tc.in, norm, loose, tc.norm, tc.loose)
		}
	}
}

// TestParsePowerShellAllocs bounds the tokenizer's allocations per token on
// every OS: it is single pass, so a command line costs a constant number of
// allocations per word plus the slices that hold them.
func TestParsePowerShellAllocs(t *testing.T) {
	text := `Get-Content .claude\settings.json 2>$null | Set-Content -Path 'x.json' -Value "{}"; ri a > b`
	words := len(strings.Fields(text))
	got := testing.AllocsPerRun(100, func() { ParsePowerShell(text) })
	if limit := float64(2 * words); got > limit {
		t.Errorf("ParsePowerShell allocated %.0f times for %d words, want at most %.0f", got, words, limit)
	}
	if got := testing.AllocsPerRun(100, func() { PowerShellText(text) }); got > 2 {
		t.Errorf("PowerShellText allocated %.0f times, want at most 2", got)
	}
}

func TestPowerShellAsPOSIX(t *testing.T) {
	t.Parallel()
	tests := []struct{ line, want string }{
		{`qsdev 'teardown','--force'`, "qsdev teardown --force\n"},
		{`& qsdev teardown,--force`, "qsdev teardown --force\n"},
		{`& qsdev @a`, "qsdev $_\n"},
		{`& qsdev @('teardown','--force')`, "qsdev $_\nteardown --force\n"},
		{`[Diagnostics.Process]::Start('qsdev','teardown --force')`, "[diagnostics.process]::start\nqsdev teardown --force\n"},
		{`Start-Process qsdev -ArgumentList 'a','b'`, "start-process qsdev -ArgumentList a b\n"},
		{`git status`, "git status\n"},
	}
	for _, tt := range tests {
		if got := PowerShellAsPOSIX(ParsePowerShell(tt.line)); got != tt.want {
			t.Errorf("PowerShellAsPOSIX(%q) = %q, want %q", tt.line, got, tt.want)
		}
	}
}
