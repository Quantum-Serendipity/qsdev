package testutil

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

// testEvents renders a `go test -json` stream in which test runs, writes
// output and ends with action.
func testEvents(t *testing.T, test, action string, output ...string) string {
	t.Helper()
	type event struct {
		Action  string
		Package string
		Test    string `json:",omitempty"`
		Output  string `json:",omitempty"`
	}
	const pkg = "example.com/p"
	events := []event{{Action: "start", Package: pkg}, {Action: "run", Package: pkg, Test: test}}
	for _, o := range output {
		events = append(events, event{Action: "output", Package: pkg, Test: test, Output: o})
	}
	events = append(events,
		event{Action: action, Package: pkg, Test: test},
		event{Action: "output", Package: pkg, Output: "ok\n"},
		event{Action: "pass", Package: pkg},
	)
	var b strings.Builder
	for _, e := range events {
		line, err := json.Marshal(e)
		if err != nil {
			t.Fatalf("encoding event: %v", err)
		}
		b.Write(line)
		b.WriteByte('\n')
	}
	return b.String()
}

func TestSkipReport(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		test     string
		action   string
		output   []string
		required []Switch
		want     []string // "<test>: <tool>"
	}{
		{
			name:     "raw skip naming a governed tool fails",
			test:     "TestEval",
			action:   "skip",
			output:   []string{"    eval_test.go:12: nix-instantiate not available\n", "--- SKIP: TestEval (0.00s)\n"},
			required: []Switch{RequireNix},
			want:     []string{"TestEval: nix-instantiate"},
		},
		{
			name:     "quoted tool in an exec error fails",
			test:     "TestSandbox/e3",
			action:   "skip",
			output:   []string{`    b_test.go:9: exec: "bwrap": executable file not found in $PATH` + "\n"},
			required: []Switch{RequireNix, RequireE3},
			want:     []string{"TestSandbox/e3: bwrap"},
		},
		{
			name:     "tool at the end of a sentence fails",
			test:     "TestScan",
			action:   "skip",
			output:   []string{"    s_test.go:3: requires gitleaks.\n"},
			required: []Switch{RequireSecTools},
			want:     []string{"TestScan: gitleaks"},
		},
		{
			name:     "shared tool is reported once",
			test:     "TestAttrs",
			action:   "skip",
			output:   []string{"    a_test.go:3: nix not available\n"},
			required: []Switch{RequireNix, CheckNixAttrs},
			want:     []string{"TestAttrs: nix"},
		},
		{
			name:     "capitalised tool name fails",
			test:     "TestT",
			action:   "skip",
			output:   []string{"    t_test.go:1: Nix not installed\n"},
			required: []Switch{RequireNix},
			want:     []string{"TestT: nix"},
		},
		{
			name:     "upper-case tool name fails",
			test:     "TestT",
			action:   "skip",
			output:   []string{"    t_test.go:1: NIX-INSTANTIATE missing\n"},
			required: []Switch{RequireNix},
			want:     []string{"TestT: nix-instantiate"},
		},
		{
			name:     "alias bubblewrap fails",
			test:     "TestT",
			action:   "skip",
			output:   []string{"    t_test.go:1: bubblewrap backend not available on this host\n"},
			required: []Switch{RequireE3},
			want:     []string{"TestT: bwrap"},
		},
		{
			name:     "alias Landlock fails",
			test:     "TestT",
			action:   "skip",
			output:   []string{"    t_test.go:1: Landlock not supported here\n"},
			required: []Switch{RequireE3},
			want:     []string{"TestT: ll-restrict"},
		},
		{
			name:     "absolute path to the tool fails",
			test:     "TestT",
			action:   "skip",
			output:   []string{"    t_test.go:1: /usr/bin/bwrap not found\n"},
			required: []Switch{RequireE3},
			want:     []string{"TestT: bwrap"},
		},
		{
			name:     "man-page section fails",
			test:     "TestT",
			action:   "skip",
			output:   []string{"    t_test.go:1: tool nix-instantiate(1) absent\n"},
			required: []Switch{RequireNix},
			want:     []string{"TestT: nix-instantiate"},
		},
		{
			name:     "alternatives separated by a slash fail for each tool",
			test:     "TestT",
			action:   "skip",
			output:   []string{"    t_test.go:1: need nix/nix-instantiate\n"},
			required: []Switch{RequireNix},
			want:     []string{"TestT: nix", "TestT: nix-instantiate"},
		},
		{
			name:   "switch unset passes",
			test:   "TestEval",
			action: "skip",
			output: []string{"    eval_test.go:12: nix-instantiate not available\n"},
		},
		{
			name:     "other switch's tool passes",
			test:     "TestRules",
			action:   "skip",
			output:   []string{"    r_test.go:7: opengrep not available\n"},
			required: []Switch{RequireNix, RequireE3},
		},
		{
			name:     "unrelated skip passes",
			test:     "TestKill",
			action:   "skip",
			output:   []string{"    k_test.go:5: systemd-run not installed\n"},
			required: Switches(),
		},
		{
			name:   "word boundary: unix, nix paths, nix files and framing pass",
			test:   "TestPerms/nix-parse",
			action: "skip",
			output: []string{
				"=== RUN   TestPerms/nix-parse\n",
				"    p_test.go:5: unix permissions; wrote devenv.nix and flake.nix: under /nix/store/abc-x; nixpkgs pinned; nix-parse\n",
				"    --- SKIP: TestPerms/nix-parse (0.00s)\n",
			},
			required: Switches(),
		},
		{
			name:     "framing line naming a governed subtest passes",
			test:     "TestE2E/bwrap",
			action:   "skip",
			output:   []string{"=== RUN   TestE2E/bwrap\n", "    e_test.go:5: kernel too old\n", "--- SKIP: TestE2E/bwrap (0.00s)\n"},
			required: Switches(),
		},
		{
			name:     "a passing test naming a tool passes",
			test:     "TestEval",
			action:   "pass",
			output:   []string{"    eval_test.go:12: ran nix-instantiate\n"},
			required: Switches(),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			skips, err := FindGovernedSkips(strings.NewReader(testEvents(t, tc.test, tc.action, tc.output...)), tc.required)
			if err != nil {
				t.Fatalf("FindGovernedSkips() error = %v", err)
			}
			var got []string
			for _, s := range skips {
				got = append(got, s.Test+": "+s.Tool)
			}
			if !slices.Equal(got, tc.want) {
				t.Errorf("FindGovernedSkips() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSkipReport_RejectsBadStreams(t *testing.T) {
	t.Parallel()

	for name, in := range map[string]string{
		"empty":         "",
		"not json":      "ok  example.com/p 0.1s\n",
		"package only":  `{"Action":"skip","Package":"example.com/p"}` + "\n",
		"truncated obj": `{"Action":"run","Package":"example.com/p","Test":"T"`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := FindGovernedSkips(strings.NewReader(in), Switches()); err == nil {
				t.Error("FindGovernedSkips() error = nil, want an error")
			}
		})
	}
}

func TestRequiredSwitches(t *testing.T) {
	for _, s := range Switches() {
		t.Setenv(string(s), "")
	}
	t.Setenv(string(RequireE3), "1")
	t.Setenv(string(RequireNix), "true") // only "1" turns a switch on

	if got, want := RequiredSwitches(), []Switch{RequireE3}; !slices.Equal(got, want) {
		t.Errorf("RequiredSwitches() = %q, want %q", got, want)
	}
}
