package rules

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/canon"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/cmdscan"
)

// Claude Code runs every hook command through a shell with its own PATH, so a
// hook named by program (`qsdev selfprotect`, a script whose shebang is
// `#!/usr/bin/env python3`) runs whichever file that name resolves to. A file
// of that name placed in an earlier PATH directory, or a rewrite of the file it
// resolves to, replaces the hook with the agent's own program. The hook
// scripts themselves and the qsdev binary are protected by path (SP-001,
// SP-010, INT-001); SP-011 protects the rest of each hook's command path.

// hookEnv describes where the running session's hooks are registered and how
// their programs resolve.
type hookEnv struct {
	// projectDir is the project the session runs in (CLAUDE_PROJECT_DIR),
	// which hook commands name as ${CLAUDE_PROJECT_DIR} and run in.
	projectDir string
	// settingsFiles are the Claude Code settings files that register hooks.
	settingsFiles []string
	// pathDirs is the PATH hook commands resolve program names through.
	pathDirs []string
	// pathExts are the executable extensions tried for a bare name (Windows
	// PATHEXT); empty elsewhere.
	pathExts []string
}

// hookEnvFromProcess describes the running session. The self-protection hook
// is itself spawned by Claude Code, so its environment (PATH,
// CLAUDE_PROJECT_DIR, CLAUDE_CONFIG_DIR) is the one every other hook runs
// with. cwd is the session directory, used when CLAUDE_PROJECT_DIR is unset.
func hookEnvFromProcess(cwd string) hookEnv {
	project := os.Getenv("CLAUDE_PROJECT_DIR")
	if project == "" {
		project = cwd
	}
	env := hookEnv{projectDir: project, pathDirs: filepath.SplitList(os.Getenv("PATH"))}
	if project != "" {
		env.settingsFiles = append(env.settingsFiles,
			filepath.Join(project, ".claude", "settings.json"),
			filepath.Join(project, ".claude", "settings.local.json"))
	}
	if dir, err := canon.ClaudeConfigDir(); err == nil {
		env.settingsFiles = append(env.settingsFiles, filepath.Join(dir, "settings.json"))
	}
	env.settingsFiles = append(env.settingsFiles, canon.ManagedSettingsFiles()...)
	if runtime.GOOS == "windows" {
		env.pathExts = strings.Split(strings.ToLower(os.Getenv("PATHEXT")), ";")
	}
	return env
}

// hookTargets is the set of files the registered hook commands execute, keyed
// by canon.PathKey, with the directories that hold them for copy-into-directory
// checks.
type hookTargets struct {
	files map[string]bool
	// names maps each PATH directory key to the program names that resolve
	// through it.
	names map[string][]string
	// forms are the spellings of each target that an inline program may use
	// (absolute, ~/..., $HOME/...), for programs passed as one word.
	forms []string
	// home is the user's home directory ("" when unknown), for forms.
	home string
	// dirKeys memoizes the keys of each directory's spellings.
	dirKeys map[string][]string
}

// hookTargetsFor returns the hook command targets, computed on first use and
// memoized on ctx. Tests set ctx.hookEnv; otherwise it describes the running
// session.
func (ctx *EvalContext) hookTargetsFor() *hookTargets {
	if ctx.hookTargets == nil {
		env := hookEnvFromProcess(ctx.CWD)
		if ctx.hookEnv != nil {
			env = *ctx.hookEnv
		}
		ctx.hookTargets = env.targets()
	}
	return ctx.hookTargets
}

// targets resolves every hook command registered in the settings files.
func (e hookEnv) targets() *hookTargets {
	t := &hookTargets{files: make(map[string]bool), names: make(map[string][]string), dirKeys: make(map[string][]string)}
	if home, err := os.UserHomeDir(); err == nil {
		t.home = home
	}
	expand := strings.NewReplacer("${CLAUDE_PROJECT_DIR}", e.projectDir, "$CLAUDE_PROJECT_DIR", e.projectDir)
	seen := make(map[string]bool)
	for _, f := range e.settingsFiles {
		for _, command := range hookCommands(f) {
			cmds, err := cmdscan.Parse(expand.Replace(command))
			if err != nil {
				continue
			}
			for _, c := range cmds {
				for _, prog := range commandPrograms(c) {
					if !seen[prog] {
						seen[prog] = true
						e.addProgram(t, prog, true)
					}
				}
			}
		}
	}
	return t
}

// commandPrograms returns the programs a hook's simple command runs: its
// command word and, for an interpreter, the script file it is given.
func commandPrograms(c cmdscan.Command) []string {
	if c.Name == "" {
		return nil
	}
	progs := []string{c.Name}
	if interpreterVerbs[path.Base(c.Name)] && len(c.Args) > 0 && !isFlag(c.Args[0]) {
		progs = append(progs, c.Args[0])
	}
	return progs
}

// addProgram records the files a program runs from: a path names one file (and
// its shebang interpreter), a bare name every PATH candidate up to the one it
// resolves to. followShebang limits interpreter resolution to one level.
func (e hookEnv) addProgram(t *hookTargets, prog string, followShebang bool) {
	if !strings.ContainsAny(prog, `/\`) {
		e.addPathLookup(t, prog, followShebang)
		return
	}
	if !filepath.IsAbs(prog) && e.projectDir != "" {
		prog = filepath.Join(e.projectDir, prog)
	}
	t.add(prog)
	if followShebang {
		if interp := shebangProgram(prog); interp != "" {
			e.addProgram(t, interp, false)
		}
	}
}

// addPathLookup records dir/name for each PATH directory until the first one
// holding the program: a file there shadows the real program, and the real one
// may be rewritten.
func (e hookEnv) addPathLookup(t *hookTargets, name string, followShebang bool) {
	names := []string{name}
	for _, ext := range e.pathExts {
		if ext != "" {
			names = append(names, name+ext)
		}
	}
	for _, dir := range e.pathDirs {
		if dir == "" {
			dir = "."
		}
		if !filepath.IsAbs(dir) && e.projectDir != "" {
			dir = filepath.Join(e.projectDir, dir)
		}
		found := ""
		for _, n := range names {
			p := filepath.Join(dir, n)
			t.addInDir(dir, n)
			if info, err := os.Stat(p); err == nil && !info.IsDir() && found == "" {
				found = p
				if resolved, err := canon.Canonicalize(p); err == nil {
					t.files[canon.PathKey(resolved)] = true // the file a symlinked program runs
				}
			}
		}
		if found != "" {
			if followShebang {
				if interp := shebangProgram(found); interp != "" {
					e.addProgram(t, interp, false)
				}
			}
			return
		}
	}
}

// add records file p under its written and symlink-resolved spellings.
func (t *hookTargets) add(p string) {
	for _, s := range pathSpellings(p) {
		t.files[canon.PathKey(s)] = true
	}
	t.addForms(p)
}

// addInDir records the file name in directory dir, which need not exist yet,
// under every spelling of dir, and records that name resolves through dir.
func (t *hookTargets) addInDir(dir, name string) {
	keys, ok := t.dirKeys[dir]
	if !ok {
		for _, d := range pathSpellings(dir) {
			keys = append(keys, canon.PathKey(d))
		}
		t.dirKeys[dir] = keys
	}
	for _, key := range keys {
		t.files[path.Join(key, canon.PathKey(name))] = true
		if !slices.Contains(t.names[key], name) {
			t.names[key] = append(t.names[key], name)
		}
	}
	t.addForms(filepath.Join(dir, name))
}

// addForms records the spellings of p an inline program may use.
func (t *hookTargets) addForms(p string) {
	p = filepath.Clean(p)
	t.forms = append(t.forms, filepath.ToSlash(p))
	if t.home == "" {
		return
	}
	if rel, err := filepath.Rel(t.home, p); err == nil && !strings.HasPrefix(rel, "..") {
		rel = filepath.ToSlash(rel)
		t.forms = append(t.forms, "~/"+rel, "$HOME/"+rel, "${HOME}/"+rel)
	}
}

// has reports whether path p (absolute) is a hook target.
func (t *hookTargets) has(p string) bool {
	for _, s := range pathSpellings(p) {
		if t.files[canon.PathKey(s)] {
			return true
		}
	}
	return false
}

// pathSpellings returns p cleaned and, when it resolves, symlink-resolved.
func pathSpellings(p string) []string {
	p = filepath.Clean(p)
	out := []string{p}
	if resolved, err := canon.Canonicalize(p); err == nil && resolved != p {
		out = append(out, resolved)
	}
	return out
}

// hookCommands returns the command strings of the hooks a settings file
// registers. A missing or unreadable file registers none (SP-001 protects the
// file itself).
func hookCommands(settingsFile string) []string {
	data, err := os.ReadFile(settingsFile)
	if err != nil {
		return nil
	}
	var settings struct {
		Hooks map[string][]struct {
			Hooks []struct {
				Type    string `json:"type"`
				Command string `json:"command"`
			} `json:"hooks"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil
	}
	var commands []string
	for _, matchers := range settings.Hooks {
		for _, m := range matchers {
			for _, h := range m.Hooks {
				if h.Command != "" && (h.Type == "" || h.Type == "command") {
					commands = append(commands, h.Command)
				}
			}
		}
	}
	return commands
}

// maxShebangLine bounds how much of a file shebangProgram reads.
const maxShebangLine = 256

// shebangProgram returns the interpreter a script's #! line runs: the program
// name `/usr/bin/env` looks up (skipping its options and assignments), or the
// interpreter path. It returns "" for a file without a #! line.
func shebangProgram(file string) string {
	f, err := os.Open(file)
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()
	line, err := bufio.NewReader(io.LimitReader(f, maxShebangLine)).ReadString('\n')
	if err != nil && line == "" {
		return ""
	}
	rest, ok := strings.CutPrefix(line, "#!")
	if !ok {
		return ""
	}
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return ""
	}
	if path.Base(filepath.ToSlash(fields[0])) != "env" {
		return fields[0]
	}
	for _, f := range fields[1:] {
		if !isFlag(f) && !strings.Contains(f, "=") {
			return f
		}
	}
	return ""
}

// writesHookTarget reports the hook target a shell command creates, rewrites,
// moves or links, or "". A line that writes nothing is cleared without
// resolving the hook targets.
func writesHookTarget(ctx *EvalContext) string {
	scs, err := ctx.scannedCommands()
	if err == nil && !slices.ContainsFunc(scs, mayWrite) {
		return ""
	}
	t := ctx.hookTargetsFor()
	if len(t.files) == 0 {
		return ""
	}
	if err != nil {
		return t.formIn(looseText(ctx.Command))
	}
	return t.writesIn(scs, 0)
}

// mayWrite reports whether sc can write a file: through a redirect, or as a
// command that is not proven read-only.
func mayWrite(sc scannedCommand) bool {
	return len(sc.WriteRedirects) > 0 || isMutating(sc)
}

// writesIn returns the hook target that one of scs writes, following shell
// scripts (`sh -c`, eval) up to maxSettingsScriptDepth.
func (t *hookTargets) writesIn(scs []scannedCommand, depth int) string {
	for _, sc := range scs {
		if p := t.commandWrites(sc); p != "" {
			return p
		}
		script, ok := shellScript(append([]string{sc.Name}, sc.Args...))
		if !ok {
			continue
		}
		sub, err := cmdscan.Parse(script)
		if err != nil || depth >= maxSettingsScriptDepth {
			if p := t.formIn(script); p != "" {
				return p
			}
			continue
		}
		nested := make([]scannedCommand, len(sub))
		for i, c := range sub {
			nested[i] = scannedCommand{Command: c, cwd: sc.cwd, cwdUnknown: sc.cwdUnknown, inProtectedDir: sc.inProtectedDir}
		}
		if p := t.writesIn(nested, depth+1); p != "" {
			return p
		}
	}
	return ""
}

// commandWrites returns the hook target that command sc writes, or "".
func (t *hookTargets) commandWrites(sc scannedCommand) string {
	for _, w := range slices.Concat(mutationTargets(sc), sc.WriteRedirects) {
		if isFlag(w) && w != "-" {
			w = flagValue(w)
		} else if name, value, ok := strings.Cut(w, "="); ok && name != "" && !strings.ContainsAny(name, `/\`) {
			w = value // an operand assignment (dd of=FILE)
		}
		if p := t.wordNames(sc, w); p != "" {
			return p
		}
	}
	if p := t.copyIntoDir(sc); p != "" {
		return p
	}
	// An inline program (`python3 -c "open('/home/u/.local/bin/qsdev','w')"`)
	// names the target inside the word; shell scripts are parsed by writesIn.
	if name := path.Base(sc.Name); interpreterVerbs[name] && !scriptShells[name] {
		for _, a := range sc.Args {
			if p := t.formIn(a); p != "" {
				return p
			}
		}
	}
	return ""
}

// wordNames returns the hook target that word w of sc names, or "": as written,
// after brace or glob expansion, resolved against sc's directory, or, when a
// shell expansion could supply its directory, by its program name alone.
func (t *hookTargets) wordNames(sc scannedCommand, w string) string {
	if w == "" {
		return ""
	}
	variants, ok := expandBraces(w)
	if !ok {
		variants = []string{w}
	}
	for _, v := range variants {
		p := expandTilde(v)
		if isRelativePath(p) {
			resolved, known := resolveWord(sc, p)
			if !known {
				continue
			}
			p = resolved
		}
		if hasGlobMeta(p) {
			if m := t.globMatch(p); m != "" {
				return m
			}
			continue
		}
		if t.has(p) {
			return p
		}
		if sc.HasExpansion && t.isProgramName(path.Base(filepath.ToSlash(p))) {
			return p
		}
	}
	return ""
}

// copyIntoDir returns the hook target a copy, move, link or install creates by
// placing a file of a hook program's name into a PATH directory given as the
// destination (`cp evil/python3 ~/.local/bin/`).
func (t *hookTargets) copyIntoDir(sc scannedCommand) string {
	if !copyVerbs[sc.Name] && !linkVerbs[sc.Name] && sc.Name != "install" {
		return ""
	}
	var sources []string
	dest := ""
	if targetDirVerbs[sc.Name] {
		sources, dest = targetDirOperands(sc.Args)
	}
	if dest == "" {
		ops := nonFlagArgs(sc.Args)
		if len(ops) < 2 {
			return ""
		}
		sources, dest = ops[:len(ops)-1], ops[len(ops)-1]
	}
	dest = expandTilde(dest)
	if isRelativePath(dest) {
		resolved, known := resolveWord(sc, dest)
		if !known {
			return ""
		}
		dest = resolved
	}
	var names []string
	for _, d := range pathSpellings(dest) {
		names = append(names, t.names[canon.PathKey(d)]...)
	}
	for _, src := range sources {
		if base := path.Base(filepath.ToSlash(src)); slices.Contains(names, base) {
			return filepath.Join(dest, base)
		}
	}
	return ""
}

// targetDirVerbs are the copy-family commands whose -t/--target-directory
// option names the destination directory (rsync's -t preserves times, tar's
// lists an archive).
var targetDirVerbs = map[string]bool{"cp": true, "mv": true, "ln": true, "install": true}

// targetDirOperands returns the sources and destination directory of a
// targetDirVerbs command given its destination with -t/--target-directory
// (`cp -t ~/.local/bin evil/qsdev`), or a "" destination when it has none.
func targetDirOperands(args []string) (sources []string, dest string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			sources = append(sources, args[i+1:]...)
			return sources, dest
		case a == "-t" || a == "--target-directory":
			if i+1 < len(args) {
				dest = args[i+1]
				i++
			}
		case strings.HasPrefix(a, "--target-directory="):
			dest = strings.TrimPrefix(a, "--target-directory=")
		case strings.HasPrefix(a, "-t") && !strings.HasPrefix(a, "--"):
			dest = a[2:]
		case !isFlag(a):
			sources = append(sources, a)
		}
	}
	return sources, dest
}

// globMatch returns a hook target that glob pattern p can expand to, or "".
func (t *hookTargets) globMatch(p string) string {
	pattern := canon.PathKey(p)
	for f := range t.files {
		if ok, _ := path.Match(pattern, f); ok {
			return f
		}
	}
	return ""
}

// isProgramName reports whether name is the file name of a PATH hook target.
func (t *hookTargets) isProgramName(name string) bool {
	for _, names := range t.names {
		if slices.Contains(names, name) {
			return true
		}
	}
	return false
}

// formIn returns the first target spelling that text contains, or "".
func (t *hookTargets) formIn(text string) string {
	text = filepath.ToSlash(text)
	for _, f := range t.forms {
		if strings.Contains(text, f) {
			return f
		}
	}
	return ""
}
