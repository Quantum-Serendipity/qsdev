package cmdutil

import (
	"bytes"
	"encoding/binary"
	"errors"
	"go/ast"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/archtest"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/canon"
	"github.com/Quantum-Serendipity/qsdev/internal/testutil/humantest"
)

// fakeProcs is a process table: each pid's parent, starting environment and
// argv (argv[0] is also the executable). It counts the reads the gate makes.
type fakeProcs struct {
	parent      map[int]int
	env         map[int][]string
	argv        map[int][]string
	parentReads int
	envReads    int
}

func (p *fakeProcs) programOf(pid int) (string, []string, error) {
	argv, ok := p.argv[pid]
	if !ok {
		return "", nil, os.ErrPermission
	}
	return argv[0], argv, nil
}

func (p *fakeProcs) parentOf(pid int) (int, error) {
	p.parentReads++
	ppid, ok := p.parent[pid]
	if !ok {
		return 0, errors.New("no such process")
	}
	return ppid, nil
}

func (p *fakeProcs) environOf(pid int) ([]string, error) {
	p.envReads++
	env, ok := p.env[pid]
	if !ok {
		return nil, os.ErrPermission
	}
	return env, nil
}

// fakeTTY is a controlling terminal: what the gate writes lands in out, and
// it reads the human's reply from in.
type fakeTTY struct {
	in     io.Reader
	out    bytes.Buffer
	closed bool
}

func (t *fakeTTY) Read(p []byte) (int, error)  { return t.in.Read(p) }
func (t *fakeTTY) Write(p []byte) (int, error) { return t.out.Write(p) }
func (t *fakeTTY) Close() error                { t.closed = true; return nil }

const testCode = "042917"

// testGate returns a gate over procs with stdin a terminal and no marker in
// its own environment, whose controlling terminal is tty (nil: none).
func testGate(procs *fakeProcs, tty *fakeTTY) humanGate {
	return humanGate{
		getenv:     func(string) string { return "" },
		isTerminal: func(io.Reader) bool { return true },
		self:       func() int { return 4242 },
		parentOf:   procs.parentOf,
		environOf:  procs.environOf,
		programOf:  procs.programOf,
		openTTY: func() (io.ReadWriteCloser, error) {
			if tty == nil {
				return nil, errors.New("no controlling terminal")
			}
			return tty, nil
		},
		newCode: func() (string, error) { return testCode, nil },
	}
}

// humanChain is a human's process tree: qsdev (4242) under a login shell
// (300) under a terminal emulator (200) under a session manager (100).
func humanChain() *fakeProcs {
	return &fakeProcs{
		parent: map[int]int{4242: 300, 300: 200, 200: 100, 100: 1},
		env: map[int][]string{
			300: {"HOME=/home/u", "TERM=xterm"},
			200: {"DISPLAY=:0", "CLAUDE_CODE_SSE_PORT=1234"},
			100: {"CLAUDECODE="},
		},
		argv: map[int][]string{
			300: {"/bin/zsh", "-l"},
			200: {"/usr/bin/kitty"},
			100: {"/usr/lib/systemd/systemd", "--user"},
		},
	}
}

// TestHumanGate_ClaudeAncestorRefuses pins the walk's program check: the
// Claude Code CLI starts with none of the markers it sets for its shells,
// so an agent shell that replaced itself with a marker-free qsdev (`exec
// env -u ... qsdev`) leaves only the CLI in the ancestry, which still
// refuses.
func TestHumanGate_ClaudeAncestorRefuses(t *testing.T) {
	t.Parallel()
	for _, argv := range [][]string{
		{"/home/u/.local/share/claude/versions/2.1.0", "--resume"},
		{"/nix/store/abc-claude-code/bin/.claude-wrapped"},
		{"claude"},
		{"/usr/bin/node", "--no-warnings", "/usr/lib/node_modules/@anthropic-ai/claude-code/cli.js"},
	} {
		t.Run(argv[0], func(t *testing.T) {
			t.Parallel()
			procs := humanChain()
			procs.argv[200] = argv // clean environment, as the CLI starts
			tty := &fakeTTY{in: strings.NewReader(testCode + "\n")}
			err := testGate(procs, tty).verify(stdinFile, "'qsdev defaults pin'")
			if err == nil || !strings.Contains(err.Error(), "Claude Code CLI runs as ancestor process 200") {
				t.Fatalf("verify = %v, want a refusal naming the Claude Code ancestor 200", err)
			}
			if tty.out.Len() != 0 {
				t.Errorf("the gate challenged a known agent: %q", tty.out.String())
			}
		})
	}
	// A human's tree passes the walk to the challenge.
	tty := &fakeTTY{in: strings.NewReader(testCode + "\n")}
	if err := testGate(humanChain(), tty).verify(stdinFile, "'qsdev defaults pin'"); err != nil {
		t.Errorf("human chain: verify = %v, want a pass", err)
	}
}

// stdinFile is stdin that is not a Terminal double, as the shipped binary
// sees it; testGate's isTerminal decides whether it is a terminal.
var stdinFile io.Reader = strings.NewReader("")

func TestHumanGate_AncestorMarkerRefuses(t *testing.T) {
	t.Parallel()
	markers := canon.AgentEnvMarkers()
	if len(markers) != 4 {
		t.Fatalf("canon.AgentEnvMarkers() = %q, want the four verified markers", markers)
	}
	for _, marker := range markers {
		t.Run(marker, func(t *testing.T) {
			t.Parallel()
			procs := humanChain()
			// An agent's shell two levels up, stripped before qsdev ran.
			procs.env[200] = append(procs.env[200], marker+"=1")
			tty := &fakeTTY{in: strings.NewReader(testCode + "\n")}
			err := testGate(procs, tty).verify(stdinFile, "'qsdev defaults pin'")
			if err == nil || !strings.Contains(err.Error(), "requires a human") ||
				!strings.Contains(err.Error(), marker) || !strings.Contains(err.Error(), "ancestor process 200") {
				t.Fatalf("verify = %v, want a refusal naming %s in ancestor 200", err, marker)
			}
			if tty.out.Len() != 0 {
				t.Errorf("the gate challenged a known agent: %q", tty.out.String())
			}
		})
	}
}

func TestHumanGate_OwnMarkerRefuses(t *testing.T) {
	t.Parallel()
	for _, marker := range canon.AgentEnvMarkers() {
		t.Run(marker, func(t *testing.T) {
			t.Parallel()
			gate := testGate(humanChain(), &fakeTTY{in: strings.NewReader(testCode + "\n")})
			gate.getenv = func(name string) string {
				if name == marker {
					return "1"
				}
				return ""
			}
			// A Terminal double does not exempt a run inside a session.
			in := humantest.HumanTerminal{Reader: strings.NewReader(""), TTY: true}
			if err := gate.verify(in, "'qsdev teardown'"); err == nil || !strings.Contains(err.Error(), marker+" is set") {
				t.Errorf("%s set: verify = %v, want an agent-session refusal", marker, err)
			}
			if err := gate.verify(stdinFile, "'qsdev teardown'"); err == nil || !strings.Contains(err.Error(), marker+" is set") {
				t.Errorf("%s set, real stdin: verify = %v, want an agent-session refusal", marker, err)
			}
		})
	}
}

func TestHumanGate_NoControllingTerminalRefuses(t *testing.T) {
	t.Parallel()
	err := testGate(humanChain(), nil).verify(stdinFile, "'qsdev teardown'")
	if err == nil || !strings.Contains(err.Error(), "requires a human at an interactive terminal") {
		t.Fatalf("verify = %v, want a refusal for the missing controlling terminal", err)
	}
	gate := testGate(humanChain(), &fakeTTY{in: strings.NewReader(testCode + "\n")})
	gate.isTerminal = func(io.Reader) bool { return false }
	if err := gate.verify(stdinFile, "'qsdev teardown'"); err == nil || !strings.Contains(err.Error(), "interactive terminal") {
		t.Fatalf("stdin not a terminal: verify = %v, want a refusal", err)
	}
}

func TestHumanGate_ChallengeMismatchAndEOFRefuse(t *testing.T) {
	t.Parallel()
	for name, reply := range map[string]string{
		"eof":            "",
		"partial":        testCode,
		"wrong":          "123456\n",
		"prefix":         testCode[:5] + "\n",
		"extra":          testCode + "0\n",
		"yes":            "y\ny\ny\n",
		"another line":   "\n" + testCode + "\n",
		"over the limit": strings.Repeat(" ", challengeLineLimit) + testCode + "\n",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tty := &fakeTTY{in: strings.NewReader(reply)}
			err := testGate(humanChain(), tty).verify(stdinFile, "'qsdev defaults reset'")
			if err == nil || !strings.Contains(err.Error(), "requires a human") {
				t.Fatalf("reply %q: verify = %v, want a refusal", reply, err)
			}
			if !tty.closed {
				t.Error("the controlling terminal was left open")
			}
		})
	}
}

func TestHumanGate_ChallengeExactCodePasses(t *testing.T) {
	t.Parallel()
	for _, reply := range []string{testCode + "\n", testCode + "\r\n", "  " + testCode + " \n"} {
		tty := &fakeTTY{in: strings.NewReader(reply)}
		if err := testGate(humanChain(), tty).verify(stdinFile, "'qsdev defaults reset'"); err != nil {
			t.Errorf("reply %q: verify = %v, want a pass", reply, err)
		}
		if !strings.Contains(tty.out.String(), testCode) || !strings.Contains(tty.out.String(), "'qsdev defaults reset'") {
			t.Errorf("challenge = %q, want it to name the command and show the code", tty.out.String())
		}
		if !tty.closed {
			t.Error("the controlling terminal was left open")
		}
	}
}

// TestHumanGate_TerminalDoubleAnswersForHuman pins the test seam: a Terminal
// double decides the terminal question without the ancestry or challenge,
// even under an agent's process tree.
func TestHumanGate_TerminalDoubleAnswersForHuman(t *testing.T) {
	t.Parallel()
	procs := humanChain()
	procs.env[300] = []string{"CLAUDECODE=1", "AI_AGENT=claude"}
	gate := testGate(procs, nil)
	if err := gate.verify(humantest.HumanTerminal{Reader: strings.NewReader(""), TTY: true}, "'qsdev x'"); err != nil {
		t.Errorf("Terminal double at a terminal: verify = %v, want a pass", err)
	}
	if err := gate.verify(humantest.HumanTerminal{Reader: strings.NewReader("")}, "'qsdev x'"); err == nil {
		t.Error("Terminal double without a terminal: verify passed, want a refusal")
	}
	if procs.parentReads+procs.envReads != 0 {
		t.Errorf("the gate read the ancestry %d times for a Terminal double", procs.parentReads+procs.envReads)
	}
}

// TestHumanGate_AncestorWalkBounded pins the walk's cost on every OS through
// the injected reader: at most maxAncestorDepth ancestors on a deep tree,
// never pid 1, and an unreadable environment passed over, not trusted.
func TestHumanGate_AncestorWalkBounded(t *testing.T) {
	t.Parallel()
	deep := &fakeProcs{parent: map[int]int{}, env: map[int][]string{}}
	for pid := 10_000; pid > 2; pid-- {
		deep.parent[pid] = pid - 1
		deep.env[pid] = []string{"PATH=/bin"}
	}
	deep.env[2] = []string{"CLAUDECODE=1"} // beyond the cap
	gate := testGate(deep, &fakeTTY{in: strings.NewReader(testCode + "\n")})
	gate.self = func() int { return 10_000 }
	if pid, marker := gate.agentAncestor(); marker != "" {
		t.Errorf("agentAncestor = %d %s, want nothing within the cap", pid, marker)
	}
	if deep.envReads != maxAncestorDepth || deep.parentReads != maxAncestorDepth {
		t.Errorf("walk read %d environments and %d parents, want %d each", deep.envReads, deep.parentReads, maxAncestorDepth)
	}

	short := humanChain()
	short.env[1] = []string{"CLAUDECODE=1"} // init's environment is never read
	delete(short.env, 200)                  // another user's process
	short.env[100] = []string{"AI_AGENT=claude"}
	if pid, marker := testGate(short, nil).agentAncestor(); pid != 100 || !strings.HasPrefix(marker, "AI_AGENT ") {
		t.Errorf("agentAncestor = %d %q, want 100 AI_AGENT past the unreadable 200", pid, marker)
	}
	if short.envReads != 3 {
		t.Errorf("walk read %d environments, want 3 (300, 200, 100)", short.envReads)
	}

	initOnly := humanChain()
	initOnly.env[1] = []string{"CLAUDECODE=1"}
	if _, marker := testGate(initOnly, nil).agentAncestor(); marker != "" || initOnly.envReads != 3 {
		t.Errorf("walk to init: marker %q after %d environment reads, want none after 3", marker, initOnly.envReads)
	}

	unknown := &fakeProcs{}
	if _, marker := testGate(unknown, nil).agentAncestor(); marker != "" || unknown.parentReads != 1 || unknown.envReads != 0 {
		t.Errorf("unknown ancestry: marker %q after %d/%d reads, want none after one parent read",
			marker, unknown.parentReads, unknown.envReads)
	}
}

// TestHumanGate_ProductionReadsAncestry checks the real readers on the
// platforms that have them: the test's own parent is found, and its
// environment is the one it started with.
func TestHumanGate_ProductionReadsAncestry(t *testing.T) {
	t.Parallel()
	ppid, err := parentPID(os.Getpid())
	switch runtime.GOOS {
	case "linux", "darwin":
		if err != nil || ppid != os.Getppid() {
			t.Fatalf("parentPID(self) = %d, %v; want %d", ppid, err, os.Getppid())
		}
		env, err := processEnviron(os.Getpid())
		if err != nil || len(env) == 0 {
			t.Fatalf("processEnviron(self) = %d entries, %v", len(env), err)
		}
		if environLookup(env)("PATH") != os.Getenv("PATH") {
			t.Errorf("processEnviron(self) PATH differs from the process's")
		}
		exe, argv, err := processProgram(os.Getpid())
		if err != nil || len(argv) == 0 || argv[0] != os.Args[0] || exe == "" {
			t.Errorf("processProgram(self) = %q, %q, %v; want the test binary and argv %q", exe, argv, err, os.Args)
		}
	default:
		if !errors.Is(err, errAncestryUnknown) {
			t.Fatalf("parentPID on %s = %d, %v; want errAncestryUnknown", runtime.GOOS, ppid, err)
		}
	}
	if code, err := challengeCode(); err != nil || len(code) != 6 || strings.Trim(code, "0123456789") != "" {
		t.Errorf("challengeCode = %q, %v; want six digits", code, err)
	}
}

// TestNoProductionTerminalDouble pins that shipped code cannot answer the
// human gate: no non-test type has the IsTerminal() bool method that makes
// it a Terminal, except the test helper humantest.HumanTerminal, which only
// test files may import.
func TestNoProductionTerminalDouble(t *testing.T) {
	t.Parallel()
	repo, err := archtest.Load(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	const helperPkg = "internal/testutil/humantest"
	helperImport := repo.Module + "/" + helperPkg
	for _, f := range repo.Files {
		if f.IsTest {
			continue
		}
		for _, imp := range f.AST.Imports {
			if strings.Trim(imp.Path.Value, `"`) == helperImport {
				t.Errorf("%s: non-test file imports %s", f.Path, helperPkg)
			}
		}
		if f.Pkg == helperPkg {
			continue
		}
		for _, decl := range f.AST.Decls {
			if isTerminalMethod(decl) {
				t.Errorf("%s: non-test type implements cmdutil.Terminal (IsTerminal() bool); only tests may", f.Path)
			}
		}
	}
}

// isTerminalMethod reports whether decl is a method IsTerminal() bool.
func isTerminalMethod(decl ast.Decl) bool {
	fn, ok := decl.(*ast.FuncDecl)
	if !ok || fn.Recv == nil || fn.Name.Name != "IsTerminal" || fn.Type.Params.NumFields() != 0 {
		return false
	}
	res := fn.Type.Results
	if res.NumFields() != 1 {
		return false
	}
	id, ok := res.List[0].Type.(*ast.Ident)
	return ok && id.Name == "bool"
}

// TestParseStatPPID pins the /proc/<pid>/stat layout: the ppid is the second
// field after the command name, which is parenthesised and may itself hold
// spaces and parentheses.
func TestParseStatPPID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, stat string
		want       int
	}{
		{"plain", "4242 (qsdev) S 300 4242 300 34816", 300},
		{"spaces", "4242 (tmux: server) S 1234 1 1 0", 1234},
		{"close paren", "4242 (a) b) R 77 4242 1 0", 77},
		{"paren then open", "4242 (x) (y) S 88 4242 1 0", 88},
		{"trailing newline", "9 (sh) S 5 9 5 0\n", 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseStatPPID([]byte(tt.stat))
			if err != nil || got != tt.want {
				t.Errorf("parseStatPPID(%q) = %d, %v; want %d", tt.stat, got, err, tt.want)
			}
		})
	}
	for _, bad := range []string{"", "4242 qsdev S 300", "4242 (qsdev) S", "4242 (qsdev) S x"} {
		if got, err := parseStatPPID([]byte(bad)); err == nil {
			t.Errorf("parseStatPPID(%q) = %d, want an error", bad, got)
		}
	}
}

// TestParseProcArgs2 pins the macOS kern.procargs2 layout: argc, the
// executable path and its NUL padding, argc arguments (an empty one
// included), then the environment up to an empty string.
func TestParseProcArgs2(t *testing.T) {
	t.Parallel()
	buf := func(argc uint32, rest string) []byte {
		return append(binary.NativeEndian.AppendUint32(nil, argc), rest...)
	}
	tests := []struct {
		name      string
		data      []byte
		exe       string
		argv, env []string
	}{
		{"args then env", buf(3, "/bin/zsh\x00\x00\x00\x00-zsh\x00\x00-l\x00CLAUDECODE=1\x00HOME=/u\x00\x00junk\x00"),
			"/bin/zsh", []string{"-zsh", "", "-l"}, []string{"CLAUDECODE=1", "HOME=/u"}},
		{"no args", buf(0, "/bin/sh\x00\x00AI_AGENT=x\x00"), "/bin/sh", nil, []string{"AI_AGENT=x"}},
		{"no env", buf(1, "/bin/sh\x00sh\x00"), "/bin/sh", []string{"sh"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p, err := parseProcArgs2(tt.data)
			if err != nil || p.exe != tt.exe || !slices.Equal(p.argv, tt.argv) || !slices.Equal(p.env, tt.env) {
				t.Errorf("parseProcArgs2 = %+v, %v; want exe %q argv %q env %q", p, err, tt.exe, tt.argv, tt.env)
			}
		})
	}
	for name, data := range map[string][]byte{
		"short":         {1, 2},
		"no exe end":    buf(0, "/bin/zsh"),
		"missing args":  buf(4, "/bin/zsh\x00a\x00b\x00"),
		"negative argc": buf(0xffffffff, "/bin/zsh\x00"),
	} {
		if p, err := parseProcArgs2(data); err == nil {
			t.Errorf("%s: parseProcArgs2 = %+v, want an error", name, p)
		}
	}
}
