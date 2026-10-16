package cmdscan

import (
	"reflect"
	"slices"
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
				{Name: "c", Piped: true, GroupArg: true},
				{Name: "d", Args: []string{"3"}, Nesting: 1},
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
			want: []PSCommand{{Name: "ri", Args: []string{"x"}}, {Name: "gc", Args: []string{"$(ri x)"}, Subexpression: true}}},
		{name: "subexpression with nested quotes and parens", text: `echo "a $(Set-Content .npmrc "x" (1)) b"; gc y`,
			want: []PSCommand{
				{Name: "set-content", Args: []string{".npmrc", "x"}, GroupArg: true},
				{Name: "1", Nesting: 1},
				{Name: "echo", Args: []string{`a $(Set-Content .npmrc "x" (1)) b`}, Subexpression: true},
				{Name: "gc", Args: []string{"y"}},
			}},
		{name: "nested subexpressions", text: `echo "$(Write-Host "$(git commit --no-verify)")"`,
			want: []PSCommand{
				{Name: "git", Args: []string{"commit", "--no-verify"}},
				{Name: "write-host", Args: []string{"$(git commit --no-verify)"}, Subexpression: true},
				{Name: "echo", Args: []string{`$(Write-Host "$(git commit --no-verify)")`}, Subexpression: true},
			}},
		{name: "unclosed subexpression", text: `echo "$(ri x`,
			want: []PSCommand{{Name: "ri", Args: []string{"x"}}, {Name: "echo", Args: []string{"$(ri x"}, Subexpression: true}}},
		{name: "expandable here-string subexpressions", text: "Write-Host @\"\na $(ri x) b $(gc y)\n\"@",
			want: []PSCommand{
				{Name: "ri", Args: []string{"x"}},
				{Name: "gc", Args: []string{"y"}},
				{Name: "write-host", Args: []string{"a $(ri x) b $(gc y)"}, Subexpression: true},
			}},
		{name: "spaced assignment", text: `$x = git commit --no-verify`,
			want: []PSCommand{{Name: "$x", Args: []string{"="}}, {Name: "git", Args: []string{"commit", "--no-verify"}}}},
		{name: "compact assignment", text: `$x=git config core.hooksPath /tmp`,
			want: []PSCommand{{Name: "$x="}, {Name: "git", Args: []string{"config", "core.hooksPath", "/tmp"}}}},
		{name: "assignment of a call", text: `$out += & git commit -n`,
			want: []PSCommand{{Name: "$out", Args: []string{"+="}}, {Name: "git", Args: []string{"commit", "-n"}}}},
		{name: "assignment of a value", text: `$x = 1`,
			want: []PSCommand{{Name: "$x", Args: []string{"="}}, {Name: "1"}}},
		{name: "comparison is not an assignment", text: `$x -eq 1`,
			want: []PSCommand{{Name: "$x", Args: []string{"-eq", "1"}}}},
		{name: "escaped subexpression is literal", text: "gc \"`$(ri x)\"",
			want: []PSCommand{{Name: "gc", Args: []string{"$(ri x)"}}}},
		{name: "bare subexpression is its own command", text: `gc $(ri x)`,
			want: []PSCommand{{Name: "gc", Args: []string{"$"}, GroupArg: true}, {Name: "ri", Args: []string{"x"}, Nesting: 1}}},
		// A pipeline's or a group's value reaches a command through no word
		// of its own (U18-WS3 review).
		{name: "pipeline input", text: "gci .npmrc | ri",
			want: []PSCommand{{Name: "gci", Args: []string{".npmrc"}}, {Name: "ri", Piped: true}}},
		{name: "pipeline continued on the next line", text: "gci .npmrc |\n  ri",
			want: []PSCommand{{Name: "gci", Args: []string{".npmrc"}}, {Name: "ri", Piped: true}}},
		{name: "or operator is not a pipeline", text: "gc a || ri b",
			want: []PSCommand{{Name: "gc", Args: []string{"a"}}, {Name: "ri", Args: []string{"b"}}}},
		{name: "group argument", text: "Set-Content ('.npmrc') x",
			want: []PSCommand{{Name: "set-content", GroupArg: true}, {Name: ".npmrc", Nesting: 1}, {Name: "x"}}},
		{name: "method call", text: "$f.Delete()",
			want: []PSCommand{{Name: "$f.delete", GroupArg: true}}},
		{name: "keyword condition is not an argument", text: "if ((Test-Path a) -and (b)) { gc a }",
			want: []PSCommand{
				{Name: "if"}, {Name: "test-path", Args: []string{"a"}, Nesting: 2}, {Name: "-and", Nesting: 1},
				{Name: "b", Nesting: 2}, {Name: "gc", Args: []string{"a"}},
			}},
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
		{"gc \"$(gc x)\"", false},
		{"> f", false},
	}
	for _, tc := range cases {
		// A subexpression's commands come first; the command holding it is
		// the last one.
		cmds := ParsePowerShell(tc.text)
		if want := 1 + strings.Count(tc.text, "$("); len(cmds) != want {
			t.Fatalf("ParsePowerShell(%q) = %d commands, want %d", tc.text, len(cmds), want)
		}
		if got := cmds[len(cmds)-1].IsRead(); got != tc.want {
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

// psUnicodeSpaces are whitespace runes PowerShell's tokenizer separates
// words on beyond the ASCII blanks: NEL, NBSP and Unicode Zs, Zl and Zp
// separators.
var psUnicodeSpaces = []rune{0x85, 0xa0, 0x1680, 0x2003, 0x2009, 0x2028, 0x2029, 0x202f, 0x3000}

func TestIsPowerShellSpace(t *testing.T) {
	t.Parallel()
	for _, r := range append([]rune{' ', '\t', '\v', '\f', 0x2000, 0x200a, 0x205f}, psUnicodeSpaces...) {
		if !IsPowerShellSpace(r) {
			t.Errorf("IsPowerShellSpace(%U) = false, want true", r)
		}
	}
	// Newlines end a statement rather than a word, and zero-width or
	// format characters are not whitespace to PowerShell.
	for _, r := range []rune{'\n', '\r', 'a', '-', 0, 0x180e, 0x200b, 0x2060, 0xfeff} {
		if IsPowerShellSpace(r) {
			t.Errorf("IsPowerShellSpace(%U) = true, want false", r)
		}
	}
}

// TestParsePowerShell_UnicodeWhitespace pins that a word ends at every rune
// PowerShell reads as whitespace (W1bN-16), so git, NBSP, reset is git with
// a reset argument, while the same rune inside quotes stays in the word.
func TestParsePowerShell_UnicodeWhitespace(t *testing.T) {
	t.Parallel()
	for _, sep := range psUnicodeSpaces {
		s := string(sep)
		text := s + "Remove-Item" + s + `.claude\settings.json` + s + "'a" + s + "b'" + s + `"c` + s + `d"` + s + s
		want := []PSCommand{{Name: "remove-item", Args: []string{`.claude\settings.json`, "a" + s + "b", "c" + s + "d"}}}
		if got := ParsePowerShell(text); !reflect.DeepEqual(got, want) {
			t.Errorf("ParsePowerShell(%q)\n got %#v\nwant %#v", text, got, want)
		}
	}
	// A zero-width space is part of the word, as it is to PowerShell.
	zwsp := "git" + string(rune(0x200b)) + "reset"
	text := zwsp + " --hard"
	if got := ParsePowerShell(text); len(got) != 1 || got[0].Name != zwsp {
		t.Errorf("ParsePowerShell(%q) = %#v, want one command named git\\u200breset", text, got)
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
		// Every PowerShell whitespace rune reads as an ASCII space.
		{"$env:X\u00a0=\u3000'.cl'\u2003+\u0085'aude'", "$env:x = '.cl' + 'aude'", "$env:x = .claude"},
		{"ri\u2028.claude\u000b", "ri .claude ", "ri .claude "},
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
		{`Start-Process qsdev -ArgumentList 'a','b'`, "start-process qsdev -ArgumentList a b\nqsdev a b\n"},
		{`git status`, "git status\n"},
	}
	for _, tt := range tests {
		if got := PowerShellAsPOSIX(ParsePowerShell(tt.line)); got != tt.want {
			t.Errorf("PowerShellAsPOSIX(%q) = %q, want %q", tt.line, got, tt.want)
		}
	}
}

// TestPowerShellAsPOSIX_StartProcessAnyOrder pins that a Start-Process line
// is also rendered as the command line it starts, FilePath then the
// ArgumentList elements, whatever order or spelling the parameters take:
// named in either order, positional, abbreviated, with a colon-attached
// value, -PSPath/-Path or -Args, around switches that take no value, and an
// array written with blanks after its commas.
func TestPowerShellAsPOSIX_StartProcessAnyOrder(t *testing.T) {
	t.Parallel()
	tests := []struct{ line, want string }{
		{`Start-Process -ArgumentList 'teardown','--force' -FilePath qsdev`, "qsdev teardown --force"},
		{`Start-Process -Wait -ArgumentList 'teardown','--force' -FilePath qsdev`, "qsdev teardown --force"},
		{`Start-Process -ArgumentList 'teardown','--force' -FilePath qsdev -Wait`, "qsdev teardown --force"},
		{`Start-Process -ArgumentList teardown,--force qsdev`, "qsdev teardown --force"},
		{`Start-Process -ArgumentList 'teardown --force' -FilePath qsdev`, "qsdev teardown --force"},
		{`Start-Process -ArgumentList 'self-update','--no-strict' -FilePath 'qsdev'`, "qsdev self-update --no-strict"},
		{`Start-Process qsdev teardown,--force`, "qsdev teardown --force"},
		{`saps -a teardown -f qsdev`, "qsdev teardown"},
		{`start -Arg teardown -File qsdev`, "qsdev teardown"},
		{`Start-Process -FilePath:qsdev -ArgumentList:teardown`, "qsdev teardown"},
		{`Start-Process -FilePath:C:\bin\qsdev.exe -ArgumentList teardown`, `C:\bin\qsdev.exe teardown`},
		{`Start-Process -Path qsdev -Args teardown`, "qsdev teardown"},
		{`Start-Process -PSPath qsdev -Args teardown`, "qsdev teardown"},
		{`Start-Process -NoNewWindow -PassThru qsdev teardown`, "qsdev teardown"},
		{`Start-Process -UseNewEnvironment -LoadUserProfile qsdev teardown`, "qsdev teardown"},
		{`Start-Process -WorkingDirectory C:\x -ArgumentList teardown qsdev`, "qsdev teardown"},
		{`Start-Process -ArgumentList 'teardown', '--force' -FilePath qsdev`, "qsdev teardown --force"},
		{`Start-Process -ArgumentList '/c','dir' -FilePath cmd`, "cmd /c dir"},
		// An ArgumentList or FilePath the line computes is the dynamic word
		// "$_": an array literal, a splat, a parenthesised expression or a
		// subexpression.
		{`Start-Process qsdev -ArgumentList @('teardown','--force')`, "qsdev $_"},
		{`Start-Process -FilePath qsdev -ArgumentList @args`, "qsdev $_"},
		{`Start-Process -FilePath qsdev -ArgumentList ('teardown','--force')`, "qsdev $_"},
		{`Start-Process -FilePath qsdev -ArgumentList ("teardown")`, "qsdev $_"},
		{`Start-Process -FilePath (Get-Command qsdev).Source -ArgumentList teardown`, "$_ teardown"},
		{`Start-Process -FilePath $(Get-Command qsdev) -ArgumentList teardown`, "$_ teardown"},
		{`Start-Process -FilePath ((gcm qsdev).Source) -ArgumentList teardown`, "$_ teardown"},
		{`Start-Process -FilePath (gcm qsdev).Source -ArgumentList (Get-Args)`, "$_ $_"},
	}
	for _, tt := range tests {
		got := PowerShellAsPOSIX(ParsePowerShell(tt.line))
		if !slices.Contains(strings.Split(got, "\n"), tt.want) {
			t.Errorf("PowerShellAsPOSIX(%q) = %q, want a line %q", tt.line, got, tt.want)
		}
	}
}

// TestParsePowerShellDeepSubexpressions verifies subexpressions nested past
// maxPSSubexpressionDepth still report a command no read check accepts, and
// that the inner command of a shallower nesting is found.
func TestParsePowerShellDeepSubexpressions(t *testing.T) {
	t.Parallel()
	for _, depth := range []int{maxPSSubexpressionDepth, 4 * maxPSSubexpressionDepth} {
		text := strings.Repeat(`echo "$(`, depth) + "ri .npmrc" + strings.Repeat(`)"`, depth)
		cmds := ParsePowerShell(text)
		found := slices.ContainsFunc(cmds, func(c PSCommand) bool { return c.Name == "ri" })
		opaque := slices.ContainsFunc(cmds, func(c PSCommand) bool { return c.Name == "$(" && !c.IsRead() })
		if !found && !opaque {
			t.Errorf("depth %d: ParsePowerShell found neither ri nor an opaque command: %+v", depth, cmds)
		}
		if depth == maxPSSubexpressionDepth && !found {
			t.Errorf("depth %d: ri not found", depth)
		}
	}
}

// TestPSCommandMayRedefine pins the commands that may give a command name
// another meaning: an alias, a function or filter, or the function: drive.
func TestPSCommandMayRedefine(t *testing.T) {
	t.Parallel()
	tests := map[string]bool{
		"Set-Alias gh Remove-Item":            true,
		"sal -Name gh -Value ri":              true,
		"New-Alias gh ri":                     true,
		"nal gh ri":                           true,
		"function gh":                         true,
		"filter gh":                           true,
		"$function:gh = 1":                    true,
		"Set-Item Function:gh -Value x":       true,
		"New-Item -Path function:\\gh -Value": true,
		"Get-Content .npmrc":                  false,
		"gh pr create --title function":       false,
	}
	for text, want := range tests {
		if got := ParsePowerShell(text)[0].MayRedefine(); got != want {
			t.Errorf("MayRedefine(%q) = %v, want %v", text, got, want)
		}
	}
}
