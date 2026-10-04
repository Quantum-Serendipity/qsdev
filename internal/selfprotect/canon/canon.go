package canon

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"syscall"

	"github.com/Quantum-Serendipity/qsdev/internal/userhome"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

var (
	protectedPrefixes []protectedEntry
	protectedSuffixes []protectedEntry
	protectedHomes    []string
	initOnce          sync.Once
	initErr           error

	// userHomeDir resolves the current user's home directory. It is a package
	// variable (defaulting to os.UserHomeDir) so tests can simulate a
	// home-resolution failure and verify the fail-closed behavior of IsProtected.
	userHomeDir = os.UserHomeDir

	// accountHomeDir resolves the home directory the user database records
	// for the account, where the CLI reads the org overlay from (a variable
	// for tests).
	accountHomeDir = userhome.Account

	// namedHomeDir resolves the home directory of a named account, which
	// `~name` expands to (a variable for tests).
	namedHomeDir = userhome.Named

	// executablePath resolves the running qsdev binary (a variable for tests).
	executablePath = os.Executable
)

type protectedEntry struct {
	path     string
	category string
}

func ensureInit() error {
	initOnce.Do(func() {
		home, err := userHomeDir()
		if err != nil {
			initErr = fmt.Errorf("resolving home directory: %w", err)
			return
		}

		// Home- and system-anchored locations. A directory entry ends in a
		// separator and protects everything below it; a file entry matches
		// exactly. Locations protected wherever they live (home config or
		// project checkout) are in the segment tables (pathTables) instead.
		protectedPrefixes = installedBinaryEntries(runtime.GOOS, home, os.Getenv("LOCALAPPDATA"), runningExecutable())
		protectedPrefixes = append(protectedPrefixes, []protectedEntry{
			{filepath.Join(home, ".claude", "managed-settings.json"), "claude-settings"},
			// ~/.claude.json holds the user-scoped MCP servers, per-project
			// permission grants and trust decisions.
			{filepath.Join(home, ".claude.json"), "claude-settings"},
			{"/etc/gdev/", "system-config"},
		}...)
		for _, dir := range managedSettingsDirs(runtime.GOOS, os.Getenv) {
			protectedPrefixes = append(protectedPrefixes, protectedEntry{dir + string(filepath.Separator), "system-config"})
		}
		protectedPrefixes = append(protectedPrefixes, claudeConfigDirEntries(os.Getenv(ClaudeConfigDirEnv))...)
		// The CLI reads the overlay below the account's home directory
		// (catalog.OrgConfigPath); the one below HOME is protected too.
		protectedHomes = []string{home}
		if account, err := accountHomeDir(); err == nil && account != home {
			protectedHomes = append(protectedHomes, account)
		}
		protectedPrefixes = append(protectedPrefixes, orgOverlayEntries(branding.Get(), protectedHomes, os.Getenv)...)

		protectedSuffixes = []protectedEntry{
			{string(filepath.Separator) + ".mcp.json", "mcp-config"},
		}
	})
	return initErr
}

// ClaudeConfigDirEnv names the environment variable that relocates Claude
// Code's user configuration directory (default ~/.claude). Claude Code passes
// its environment to every hook, so the value a hook sees is the directory the
// running session loads its user settings from.
const ClaudeConfigDirEnv = "CLAUDE_CONFIG_DIR"

// claudeConfigFiles are the entries of a Claude Code configuration directory
// that register or steer enforcement, mirroring the .claude entries of
// staticSegments (a directory entry ends in "/"), plus .claude.json, which
// Claude Code keeps inside a relocated configuration directory.
var claudeConfigFiles = []string{
	"settings.json", "settings.local.json", ".claude.json",
	"hooks/", "agents/", "commands/", "skills/",
}

// claudeConfigDirEntries returns the protected entries for a Claude Code
// configuration directory named by CLAUDE_CONFIG_DIR (dir, "" when unset). The
// .claude segment entries only cover a directory named .claude, so a relocated
// one (~/.config/claude) would otherwise leave the user settings, which can
// disable or replace every hook, writable. The directory is protected under
// its absolute spelling and, when it exists, its symlink-resolved one, since
// rules compare canonical paths.
func claudeConfigDirEntries(dir string) []protectedEntry {
	if dir == "" {
		return nil
	}
	sep := string(filepath.Separator)
	var entries []protectedEntry
	for _, root := range spellings(dir) {
		for _, f := range claudeConfigFiles {
			p := filepath.Join(root, strings.TrimSuffix(f, "/"))
			if strings.HasSuffix(f, "/") {
				p += sep
			}
			entries = append(entries, protectedEntry{p, "claude-settings"})
		}
	}
	return entries
}

// spellings returns p (tilde-expanded) made absolute and, when it differs,
// its canonical form with every existing symlink resolved, including a
// dangling final one. Rules compare canonical paths, so a location configured
// through a symlink must be protected under both. It returns nil when p is
// empty or cannot be made absolute.
func spellings(p string) []string {
	if p == "" {
		return nil
	}
	abs, err := filepath.Abs(expandTildeOrSelf(p))
	if err != nil {
		return nil
	}
	out := []string{abs}
	if resolved, err := Canonicalize(abs); err == nil && resolved != abs {
		out = append(out, resolved)
	}
	return out
}

// orgConfigEnv names the environment variable that points the org overlay at
// a file other than ~/.config/<app>/defaults.yaml (catalog.OrgConfigPath).
func orgConfigEnv(cfg branding.Config) string {
	return cfg.EnvPrefix + "ORG_CONFIG"
}

// ShellPathVars returns the shell variables that name a home-anchored
// protected location, with the values a command the agent runs sees: HOME
// (the home directory), USERPROFILE (the Windows profile directory, which
// Git Bash also exports) and <EnvPrefix>ORG_CONFIG when they are set. A
// command scan renders them (cmdscan.ParseWithVars) so
// `$HOME/.config/<app>/x`, `cd $HOME/.config` and `"$QSDEV_ORG_CONFIG"` are
// checked as the paths they expand to.
func ShellPathVars() map[string]string {
	vars := make(map[string]string, 3)
	if home, err := userHomeDir(); err == nil {
		vars["HOME"] = home
	}
	if profile := os.Getenv("USERPROFILE"); profile != "" {
		vars["USERPROFILE"] = profile
	}
	env := orgConfigEnv(branding.Get())
	if v := os.Getenv(env); v != "" {
		vars[env] = v
	}
	return vars
}

// orgOverlayEntries returns the protected entries for the user-level org
// overlay the catalog merges into every generation: the ~/.config/<app>/
// directory below each of homes, and the file named by <EnvPrefix>ORG_CONFIG
// when getenv sets it. All are protected under their absolute and
// symlink-resolved spellings.
func orgOverlayEntries(cfg branding.Config, homes []string, getenv func(string) string) []protectedEntry {
	sep := string(filepath.Separator)
	var entries []protectedEntry
	for _, home := range homes {
		for _, dir := range spellings(cfg.OrgConfigDir(home)) {
			entries = append(entries, protectedEntry{dir + sep, "config"})
		}
	}
	for _, file := range spellings(getenv(orgConfigEnv(cfg))) {
		entries = append(entries, protectedEntry{file, "config"})
	}
	return entries
}

// managedSettingsDirs returns the directories Claude Code reads its managed
// (policy) settings from on goos: /etc/claude-code on Linux (also protected
// elsewhere, where it is harmless), /Library/Application Support/ClaudeCode on
// macOS, and ClaudeCode under Program Files (current) and ProgramData (older
// releases) on Windows. getenv resolves the Windows folder variables.
func managedSettingsDirs(goos string, getenv func(string) string) []string {
	dirs := []string{"/etc/claude-code"}
	switch goos {
	case "darwin":
		dirs = append(dirs, "/Library/Application Support/ClaudeCode")
	case "windows":
		for _, v := range []struct{ env, fallback string }{
			{"ProgramFiles", `C:\Program Files`},
			{"ProgramData", `C:\ProgramData`},
		} {
			root := getenv(v.env)
			if root == "" {
				root = v.fallback
			}
			dirs = append(dirs, filepath.Join(root, "ClaudeCode"))
		}
	}
	return dirs
}

// ClaudeConfigDir returns the Claude Code user configuration directory the
// running session uses: CLAUDE_CONFIG_DIR when set, else ~/.claude.
func ClaudeConfigDir() (string, error) {
	if dir := os.Getenv(ClaudeConfigDirEnv); dir != "" {
		return expandTildeOrSelf(dir), nil
	}
	home, err := userHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	return filepath.Join(home, ".claude"), nil
}

// ManagedSettingsFiles returns the managed-settings.json paths Claude Code
// reads on this platform.
func ManagedSettingsFiles() []string {
	dirs := managedSettingsDirs(runtime.GOOS, os.Getenv)
	files := make([]string, len(dirs))
	for i, d := range dirs {
		files[i] = filepath.Join(d, "managed-settings.json")
	}
	return files
}

// expandTildeOrSelf is ExpandTilde that returns path unchanged when the home
// directory cannot be resolved.
func expandTildeOrSelf(path string) string {
	if expanded, err := ExpandTilde(path); err == nil {
		return expanded
	}
	return path
}

// PathKey returns p in the form protected-path comparisons use on this
// platform: slash-separated, case-folded on case-insensitive filesystems, and
// with Windows name aliases removed. Two spellings of one file have equal keys.
func PathKey(p string) string {
	return platformMatch.key(p)
}

// installedBinaryEntries returns the protected entries for the qsdev binary,
// which runs every guard hook: each installer's default install directory,
// and the running executable itself (exe, "" when unknown) wherever it was
// installed (--install-dir, QSDEV_INSTALL_DIR, a package manager). The
// executable is protected as a file, not by directory, since it may sit in a
// shared directory such as /usr/local/bin or a project checkout.
func installedBinaryEntries(goos, home, localAppData, exe string) []protectedEntry {
	sep := string(filepath.Separator)
	entries := []protectedEntry{{filepath.Join(home, ".qsdev", "bin") + sep, "binary"}}
	// install.ps1 installs to %LOCALAPPDATA%\qsdev\bin by default.
	if goos == "windows" && localAppData != "" {
		entries = append(entries, protectedEntry{filepath.Join(localAppData, "qsdev", "bin") + sep, "binary"})
	}
	if exe != "" {
		entries = append(entries, protectedEntry{exe, "binary"})
	}
	return entries
}

// runningExecutable returns the canonical path of the running binary, or ""
// when it cannot be resolved (protection then rests on the default install
// directories).
func runningExecutable() string {
	exe, err := executablePath()
	if err != nil {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	abs, err := filepath.Abs(exe)
	if err != nil {
		return ""
	}
	return abs
}

// ExpandTilde replaces a leading ~ with the user's home directory, and a
// leading ~name with the home directory the user database records for the
// account name, as the shell does. A ~name whose account cannot be resolved
// (or a ~+, ~- or ~N directory-stack form) is returned unchanged, so callers
// that need a resolved path see that it is not rooted.
func ExpandTilde(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := userHomeDir()
		if err != nil {
			return "", fmt.Errorf("expanding tilde: %w", err)
		}
		return filepath.Join(home, path[1:]), nil
	}
	if name, rest, ok := tildeUser(path); ok {
		if home, err := namedHomeDir(name); err == nil {
			return filepath.Join(home, rest), nil
		}
	}
	return path, nil
}

// tildeUser splits a ~name or ~name/rest path into the account name and the
// rest. Bash takes the name up to the first slash and passes it to getpwnam
// whatever it holds, so a directory-service name such as alice@corp.com is
// an account too; only the directory-stack forms are not (see
// isTildeAccountName).
func tildeUser(path string) (name, rest string, ok bool) {
	after, found := strings.CutPrefix(path, "~")
	if !found {
		return "", "", false
	}
	name, rest, _ = strings.Cut(after, "/")
	if !isTildeAccountName(name) {
		return "", "", false
	}
	return name, rest, true
}

// isTildeAccountName reports whether name, the text between ~ and the first
// slash, is looked up as an account: anything but the directory-stack forms
// (~+, ~-, ~N, ~+N, ~-N). A name starting with + or - is not one either: no
// account is called that (the user database reserves those for NIS compat
// lines) and getent would read it as an option.
func isTildeAccountName(name string) bool {
	return name != "" && name[0] != '+' && name[0] != '-' && strings.Trim(name, "0123456789") != ""
}

// ProtectedHomes returns the home directories protected locations are
// anchored below: the hook's HOME and, when it differs, the account's home
// directory from the user database, where the CLI reads the org overlay. It
// returns nil when the table cannot be built.
func ProtectedHomes() []string {
	if ensureInit() != nil {
		return nil
	}
	return slices.Clone(protectedHomes)
}

// Canonicalize resolves a path to its canonical form.
// Tier 1: filepath.EvalSymlinks + filepath.Abs for paths that exist.
// Tier 2: for a path that does not exist (yet), resolveMissing walks it
// component by component the way the kernel does, following every existing
// symlink — including a dangling final one, which a write follows to create
// its target.
func Canonicalize(path string) (string, error) {
	return canonicalize(nil, path)
}

// canonicalize is Canonicalize making its filesystem lookups through r.
//
// A non-nil r walks the path first. A walk that ends below a missing
// component is the answer Canonicalize gives for that path: EvalSymlinks
// Lstat-s the same components in the same order, so it fails at the same
// missing one (not with a permission or loop error, which would have failed
// the walk first) and Canonicalize falls through to resolveMissing, this
// same walk. Taking it directly lets every path of r share the Lstat of each
// directory instead of each EvalSymlinks re-reading its whole prefix. Any
// other path (one that exists, or whose walk fails) takes the tiers below,
// so their answers and errors are unchanged.
func canonicalize(r *Resolver, path string) (string, error) {
	expanded, err := ExpandTilde(path)
	if err != nil {
		return "", fmt.Errorf("canonicalizing path: %w", err)
	}
	if r != nil {
		if canonical, missing, err := r.walk(expanded); err == nil && missing {
			return canonical, nil
		}
	}

	// Tier 1: the full path exists (including through symlinks).
	resolved, err := r.EvalSymlinks(expanded)
	if err == nil {
		abs, absErr := filepath.Abs(resolved)
		if absErr != nil {
			return "", fmt.Errorf("resolving absolute path: %w", absErr)
		}
		return abs, nil
	}

	// Detect symlink loops or permission errors — don't attempt the walk.
	if os.IsPermission(err) || isSymlinkLoop(err) {
		return "", fmt.Errorf("canonicalizing path %q: %w", path, err)
	}

	// Tier 2: resolve the existing part and append the missing tail.
	resolved, _, err = r.walk(expanded)
	return resolved, err
}

// maxSymlinkHops bounds symlink expansion in resolveMissing (Linux's
// MAXSYMLINKS), so a symlink cycle fails instead of looping forever.
const maxSymlinkHops = 40

// errTooManySymlinks reports a symlink chain longer than maxSymlinkHops.
var errTooManySymlinks = errors.New("too many levels of symbolic links")

// resolveMissing canonicalizes a path whose final target does not exist. It
// resolves components left to right: each existing component is Lstat-ed and,
// if it is a symlink, replaced by its target (relative targets resolve against
// the symlink's directory), and ".." is applied only after the component
// before it has been resolved. Unlike a lexical Clean, this sees that
// <dir>/lnk/../x is <lnk target's parent>/x and that a dangling symlink names
// its target, exactly as a write through the path would. Once a component is
// missing nothing below it exists, so the rest of the path is collected into a
// missing tail (a ".." pops it) and joined once at the end: joining component
// by component re-cleans the whole path each time, O(depth^2).
func resolveMissing(p string) (string, error) {
	resolved, _, err := (*Resolver)(nil).walk(p)
	return resolved, err
}

// walk is resolveMissing making its filesystem lookups through r. It also
// reports whether the path ends below a missing component.
func (r *Resolver) walk(p string) (canonical string, missing bool, err error) {
	start, err := absWithoutClean(p)
	if err != nil {
		return "", false, fmt.Errorf("resolving absolute path: %w", err)
	}

	vol := filepath.VolumeName(start)
	resolved := vol + string(filepath.Separator)
	pending := splitPath(start[len(vol):])
	// tail holds the components from the first missing one on; resolved is
	// the existing directory they hang below.
	var tail []string
	hops := 0

	for len(pending) > 0 {
		comp := pending[0]
		pending = pending[1:]
		switch {
		case comp == ".":
			continue
		case comp == ".." && len(tail) > 0:
			// A kernel walk fails at the missing component, so the path only
			// reaches a file when the consumer cleans it lexically first
			// (<dir>/missing/../lnk/x -> <dir>/lnk/x); once the tail is
			// popped empty, resume resolving so a symlink after the ".." is
			// still followed.
			tail = tail[:len(tail)-1]
			continue
		case comp == "..":
			resolved = filepath.Dir(resolved)
			continue
		case len(tail) > 0:
			tail = append(tail, comp)
			continue
		}

		next := filepath.Join(resolved, comp)
		info, err := r.lstat(next)
		switch {
		case errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR):
			resolved = normalizeExisting(r, resolved)
			tail = append(tail, comp)
			continue
		case err != nil:
			return "", false, fmt.Errorf("canonicalizing path %q: %w", p, err)
		case info.Mode()&fs.ModeSymlink == 0:
			resolved = next
			continue
		}

		hops++
		if hops > maxSymlinkHops {
			return "", false, fmt.Errorf("canonicalizing path %q: %w", p, errTooManySymlinks)
		}
		target, err := r.readlink(next)
		if err != nil {
			return "", false, fmt.Errorf("canonicalizing path %q: %w", p, err)
		}
		if isRooted(target) {
			// An absolute target restarts resolution at its root (the current
			// volume when a Windows target is rooted without a drive).
			targetVol := filepath.VolumeName(target)
			target = target[len(targetVol):]
			if targetVol == "" {
				targetVol = vol
			}
			resolved = targetVol + string(filepath.Separator)
		}
		pending = append(splitPath(target), pending...)
	}
	return filepath.Join(append([]string{resolved}, tail...)...), len(tail) > 0, nil
}

// normalizeExisting returns the platform's canonical spelling of dir, an
// existing directory whose symlinks resolveMissing has already followed. On
// Windows filepath.EvalSymlinks also expands 8.3 short names (CLAUDE~1 ->
// .claude) and fixes the case of each component, which a component-wise
// Lstat walk does not; elsewhere it returns dir unchanged. On error dir is
// kept as-is.
func normalizeExisting(r *Resolver, dir string) string {
	if norm, err := r.EvalSymlinks(dir); err == nil {
		return norm
	}
	return dir
}

// absWithoutClean makes p absolute WITHOUT lexically cleaning it, so ".."
// components survive for resolveMissing to apply after symlink resolution
// (filepath.Abs would collapse lnk/.. before lnk is resolved).
func absWithoutClean(p string) (string, error) {
	if filepath.IsAbs(p) {
		return p, nil
	}
	if filepath.VolumeName(p) != "" {
		// Windows drive-relative path ("C:foo"): only Abs knows that drive's
		// working directory.
		return filepath.Abs(p)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}
	if isRooted(p) {
		return filepath.VolumeName(cwd) + p, nil
	}
	return cwd + string(filepath.Separator) + p, nil
}

// isRooted reports whether p is absolute or starts at a root separator.
func isRooted(p string) bool {
	return filepath.IsAbs(p) || (p != "" && os.IsPathSeparator(p[0]))
}

// splitPath splits p into its non-empty components, accepting every separator
// the platform does.
func splitPath(p string) []string {
	return strings.FieldsFunc(p, func(r rune) bool {
		return r < 0x80 && os.IsPathSeparator(uint8(r))
	})
}

func isSymlinkLoop(err error) bool {
	if errors.Is(err, errTooManySymlinks) {
		return true
	}
	if pathErr, ok := errors.AsType[*os.PathError](err); ok {
		return errors.Is(pathErr.Err, errors.ErrUnsupported) ||
			strings.Contains(pathErr.Err.Error(), "too many levels of symbolic links")
	}
	return false
}

// IsProtected checks whether a canonical path falls under any protected prefix.
// Returns (true, category) if protected, (false, "") otherwise.
//
// On case-insensitive filesystems (the macOS and Windows defaults) the match
// ignores case, so ~/.CLAUDE/settings.json cannot slip past a rule that names
// ~/.claude/settings.json while opening the same file. On Windows it also
// ignores the name aliases the filesystem strips (trailing dots and spaces, and
// an alternate-data-stream suffix such as "::$DATA").
func IsProtected(canonicalPath string) (bool, string) {
	return isProtected(canonicalPath, platformMatch)
}

func isProtected(canonicalPath string, opts matchOptions) (bool, string) {
	if err := ensureInit(); err != nil {
		// Fail closed. If the home directory cannot be resolved we cannot build
		// the home-anchored protected-prefix table, so we cannot prove that a
		// path is UNprotected. A self-protection control must never silently drop
		// protection because of an environment error (the phase-28 fail-closed
		// mandate), so treat every path as protected and let the rules deny the
		// operation. This is deliberately conservative and only triggers when
		// os.UserHomeDir fails (e.g. HOME/USERPROFILE unset), which is rare. The
		// "config" category makes SP-001 — and, via deny-overrides, the whole
		// Tier-1 rule set — block the operation.
		return true, "config"
	}

	key := opts.key(canonicalPath)

	// Check home- and system-anchored paths.
	for _, entry := range protectedPrefixes {
		entryKey := opts.key(entry.path)
		if strings.HasSuffix(entryKey, "/") {
			if strings.HasPrefix(key, entryKey) {
				return true, entry.category
			}
		} else if key == entryKey {
			return true, entry.category
		}
	}

	// Check locations protected wherever they live. A project-relative .claude/
	// or .qsdev/ canonicalizes OUTSIDE $HOME, so an anchored prefix cannot catch
	// it; the segment match does, so SP-001/SP-013 guard Write/Edit to them in
	// both the home config and a project checkout.
	if category := brandedTables().segmentCategory(key, opts.foldCase); category != "" {
		return true, category
	}

	// Check suffix-based protected paths.
	for _, entry := range protectedSuffixes {
		entryKey := opts.key(entry.path)
		if strings.HasSuffix(key, entryKey) {
			return true, entry.category
		}
		// Also match when the path is exactly ".mcp.json" (no directory prefix).
		base := strings.TrimPrefix(entryKey, "/")
		if key == base || path.Base(key) == base {
			return true, entry.category
		}
	}

	return false, ""
}

// segmentEntry is a protected location matched wherever it appears in a path.
// A segment ending in "/" is a directory (it and everything below it are
// protected); otherwise it names a single file.
type segmentEntry struct {
	segment  string
	category string
}

// staticSegments are the qsdev and Claude Code control files protected in
// any location — home config or project checkout. More specific entries come
// first so the first match yields the most precise category.
//
// The .claude entries are the files that register or steer enforcement: the
// settings that register the PreToolUse hooks and deny rules, the hook scripts,
// and the agent, command and skill definitions that can drive tools. Prose
// instruction files (CLAUDE.md, .claude/rules/) are legitimate edit targets and
// stay with gatedodge's content checks rather than being blocked outright.
// Other .claude content (e.g. Claude Code's own .claude/worktrees/ checkouts)
// stays writable.
//
// Every segment starts with one of staticSubstringPatterns, so the
// raw-command check (ContainsProtectedPath) covers every location this table
// protects; TestProtectedSegmentsCoveredByCommandScan enforces that. The
// branding-derived entries are added by newPathTables.
var staticSegments = []segmentEntry{
	{".qsdev/audit/", "audit"},
	// Hook audit logs and the SOC 2 session trail (~/.claude/audit).
	{".claude/logs/", "audit"},
	{".claude/audit/", "audit"},
	{".claude/hook-audit.log", "audit"},
	{".claude/hook-audit.log.1", "audit"},
	{".qsdev/", "config"},
	{".gdev/", "config"},
	{".claude/settings.json", "claude-settings"},
	{".claude/settings.local.json", "claude-settings"},
	{".claude/hooks/", "claude-settings"},
	{".claude/agents/", "claude-settings"},
	{".claude/commands/", "claude-settings"},
	{".claude/skills/", "claude-settings"},
}

// hasPathSegment reports whether the slash-separated path key contains seg as
// whole path components: a directory segment ("x/y/") matches the directory
// itself or anything below it, a file segment ("x/y") matches only at the end.
// Component boundaries are required, so "foo.claude/hooks/x" does not match
// ".claude/hooks/". It runs for every segment on every path a rule checks, so
// it scans key in place rather than building bounded copies of it.
func hasPathSegment(key, seg string) bool {
	name, dir := strings.CutSuffix(seg, "/")
	for from := 0; from <= len(key); {
		i := strings.Index(key[from:], name)
		if i < 0 {
			return false
		}
		start, end := from+i, from+i+len(name)
		if (start == 0 || key[start-1] == '/') && (end == len(key) || dir && key[end] == '/') {
			return true
		}
		from = start + 1
	}
	return false
}

// matchOptions describe how the host filesystem compares names.
type matchOptions struct {
	// foldCase compares names case-insensitively (macOS and Windows defaults).
	foldCase bool
	// windowsAliases strips the name aliases Windows ignores when it opens a
	// file: trailing dots and spaces, and an alternate-data-stream suffix.
	windowsAliases bool
}

var platformMatch = matchOptions{
	foldCase:       runtime.GOOS == "darwin" || runtime.GOOS == "windows",
	windowsAliases: runtime.GOOS == "windows",
}

// key returns p in the form protected-path comparisons use: slash-separated,
// with the platform's filesystem name aliases normalized away.
func (o matchOptions) key(p string) string {
	s := filepath.ToSlash(p)
	if o.windowsAliases {
		s = stripWindowsAliases(s)
	}
	if o.foldCase {
		s = strings.ToLower(s)
	}
	return s
}

// stripWindowsAliases removes, from each component of a slash-separated path,
// an alternate-data-stream suffix ("settings.json::$DATA" -> "settings.json")
// and trailing dots and spaces (".claude." -> ".claude"), which Windows
// discards when it opens the file. A drive component ("C:") and "."/".." are
// left alone.
func stripWindowsAliases(s string) string {
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

// staticSubstringPatterns are path fragments used by ContainsProtectedPath
// to detect protected path references in raw command strings. This is the
// union of all patterns previously in evasion.containsProtectedPath and
// rules.containsProtectedPathStr. Each carries a trailing separator, so a
// protected path FOLLOWED by more path (`.claude/settings.json`) is matched.
// The branding-derived patterns are added by newPathTables.
var staticSubstringPatterns = []string{
	".claude/",
	".qsdev/",
	".gdev/",
	"/etc/gdev/",
	"/etc/claude-code/",
}

// staticFileTokens are protected file names that sit directly in a
// directory other code may legitimately touch (the home directory), so no
// directory fragment above covers them. ContainsProtectedPath matches them as
// complete path segments, so `~/.claude.json.bak` and `my.claude.json` do not
// match.
var staticFileTokens = []string{
	".claude.json",
}

// staticProbeMembers are names of files qsdev and Claude Code keep inside the
// protected directories. A check that cannot see the filesystem (a find
// expression's name patterns) tries them as representative contents of every
// protected directory; anything below such a directory is protected, so
// pairing any member with any directory is sound. The branding-derived names
// are added by newPathTables.
var staticProbeMembers = []string{
	"config.yaml", "defaults.yaml", "package-guard.py", "audit-log.sh",
	"agent.md", "audit.jsonl", "events.log",
}

// pathTables are the location-independent protection tables: the static
// entries above plus those derived from the branding.
type pathTables struct {
	// segments are matched as whole path components by IsProtected.
	segments []segmentEntry
	// substrings are matched anywhere in a raw command.
	substrings []string
	// tokens are the bare protected directory names (substrings without the
	// trailing separator) and the protected file names, matched in a raw
	// command only as complete path segments, so a whole-directory operation
	// like `rm -rf .claude` is caught while `my.claude.bak` is not.
	tokens []string
	// envVars are the variables that name a protected location, matched in a
	// raw command as complete words like tokens (`"$QSDEV_ORG_CONFIG"`), but
	// never file names (ProtectedNames).
	envVars []string
	// probeMembers are representative contents of a protected directory
	// (see staticProbeMembers), relative to it.
	probeMembers []string
}

// brandedTables returns the tables for the active branding. They are built on
// first use rather than at package initialization, because a white-label
// build sets its branding in main, after every package has initialized.
var brandedTables = sync.OnceValue(func() *pathTables {
	return newPathTables(branding.Get(), os.Getenv)
})

// newPathTables builds the tables for cfg. The branding-derived entries are
// the inputs the Claude settings generator trusts:
//   - the state directory (StateDir), which holds the answers and the state
//     manifest that teardown and drift detection rely on;
//   - the devenv addon's mirror of the answers, .devenv/.<app>-answers.yaml,
//     protected as a file because the rest of .devenv/ is devenv's runtime
//     directory;
//   - .envrc, which direnv runs on every shell entry.
//
// Each segment is covered by a substring or a token, so the raw-command scan
// sees every location IsProtected guards. The org overlay is home-anchored
// (orgOverlayEntries), so it has no segment; its raw-command fragments are
// added here so a spelling the scan cannot resolve statically (`U=~; echo x >
// $U/.config/<app>/f`, `sh -c "..."`) still counts as a mention: the
// home-relative directory .config/<app>/, the <EnvPrefix>ORG_CONFIG variable
// name, and the file getenv says that variable names.
func newPathTables(cfg branding.Config, getenv func(string) string) *pathTables {
	stateDir := strings.Trim(filepath.ToSlash(cfg.StateDir), "/")
	devenvCopy := "." + cfg.AppName + "-answers.yaml"
	t := &pathTables{
		segments: append(slices.Clone(staticSegments),
			segmentEntry{stateDir + "/", "answers"},
			segmentEntry{".devenv/" + devenvCopy, "answers"},
			segmentEntry{".envrc", "config"},
		),
		substrings: append(slices.Clone(staticSubstringPatterns), stateDir+"/", ".config/"+cfg.AppName+"/"),
	}
	for _, p := range t.substrings {
		t.tokens = append(t.tokens, strings.TrimSuffix(p, "/"))
	}
	t.tokens = append(t.tokens, staticFileTokens...)
	t.tokens = append(t.tokens, devenvCopy, ".envrc")
	t.envVars = []string{orgConfigEnv(cfg)}
	t.substrings = append(t.substrings, commandSpellings(getenv(orgConfigEnv(cfg)))...)
	t.probeMembers = append(slices.Clone(staticProbeMembers),
		"."+cfg.AppName+"-init-answers.yaml", path.Join("bin", cfg.AppName))
	return t
}

// commandSpellings returns the slash-separated forms a command may use to name
// the file p: its absolute and symlink-resolved spellings (see spellings) and,
// when rooted or home-relative, p as written (a Windows `\srv\x` has no drive
// in its written form). A relative p is not included as written, since a bare
// name would match unrelated text.
func commandSpellings(p string) []string {
	var out []string
	for _, s := range spellings(p) {
		out = append(out, filepath.ToSlash(s))
	}
	if isRooted(p) || strings.HasPrefix(p, "~") {
		if raw := filepath.ToSlash(p); !slices.Contains(out, raw) {
			out = append(out, raw)
		}
	}
	return out
}

// ProtectedEnvVars returns the environment variables that relocate a
// protected generator input (<EnvPrefix>ORG_CONFIG). Protection covers the
// location the hook process sees, so a command that sets one can point a
// regeneration at an unprotected file; the Bash rules deny setting them.
func ProtectedEnvVars() []string {
	return slices.Clone(brandedTables().envVars)
}

// envSourceFiles are the lower-cased base names of the files that set the
// environment a later CLI run inherits, as the generated environment and the
// shell load them: devenv's configuration (devenv.yaml and devenv.local.yaml,
// whose imports devenv loads, devenv.nix, and devenv.local.nix, the
// documented place for local env), and the startup files of bash, zsh, ksh,
// fish and PowerShell and pam_environment. The generated environment loads no
// other file: devenv.nix disables dotenv (so .env is not read) and .envrc runs
// only devenv's own direnvrc (so .envrc.local is not read); a test in the
// devenv addon pins that. What these files import or source (another .nix
// file, a ~/.bashrc.d fragment) is not listed: see catalog.OrgConfigPin for
// the control that does not depend on it.
var envSourceFiles = []string{
	"devenv.yaml", "devenv.local.yaml", "devenv.nix", "devenv.local.nix",
	".profile", ".bashrc", ".bash_profile", ".bash_login",
	".zshenv", ".zprofile", ".zshrc", ".zlogin",
	".kshrc", ".mkshrc", "config.fish", ".pam_environment",
	"profile.ps1", "microsoft.powershell_profile.ps1",
}

// EnvSourceFiles returns the lower-cased base names of the files that set the
// environment a later CLI run inherits (see envSourceFiles). A change that
// adds, removes or alters a ProtectedEnvVars assignment in one of them
// relocates the org overlay as surely as a command line that sets it.
func EnvSourceFiles() []string {
	return slices.Clone(envSourceFiles)
}

// homeEnvVars are the variables os.UserHomeDir reads the home directory from:
// HOME on Unix (and plan9's home, the same name case-folded), USERPROFILE on
// Windows.
var homeEnvVars = []string{"HOME", "USERPROFILE"}

// HomeEnvVars returns the environment variables that name the home
// directory (os.UserHomeDir). The CLI reads the org overlay below the
// account's home directory from the user database, and not at all when that
// cannot be resolved (catalog.OrgConfigPath), so they no longer move it; the
// Bash rules still deny a line that runs the CLI and sets or clears one, as
// defense in depth. Unlike ProtectedEnvVars they are set for ordinary programs
// (`HOME=$(mktemp -d) go test`), so only a line that runs the CLI may not
// change them.
func HomeEnvVars() []string {
	return slices.Clone(homeEnvVars)
}

// ProtectedNames returns the single-component names protected wherever they
// appear (.claude, the state directory, .envrc, ...), for a check that must
// decide whether a glob segment can expand to one of them.
func ProtectedNames() []string {
	var names []string
	for _, tok := range brandedTables().tokens {
		if !strings.Contains(tok, "/") {
			names = append(names, tok)
		}
	}
	return names
}

// ProtectedLocations returns the home- and system-anchored protected
// locations IsProtected checks, a directory ending in a separator, for a check
// that must decide whether an absolute glob can expand to one of them. It
// returns nil when the table cannot be built; IsProtected then fails closed.
func ProtectedLocations() []string {
	if ensureInit() != nil {
		return nil
	}
	locs := make([]string, len(protectedPrefixes))
	for i, e := range protectedPrefixes {
		locs[i] = e.path
	}
	return locs
}

// FindProbes returns representative protected paths, slash-separated, for a
// check that must decide whether a name or path pattern can select a
// protected file without looking at the filesystem (a find expression).
// relative holds the locations protected wherever they live, relative to the
// directory holding them: every segment with its ancestors (a find can
// select .claude itself), and below each directory segment every probe
// member. absolute holds the home- and system-anchored locations
// (ProtectedLocations) with their ancestors and, below each directory, every
// probe member. absolute is nil when that table cannot be built; IsProtected
// then fails closed.
func FindProbes() (relative, absolute []string) {
	t := brandedTables()
	for _, seg := range t.segments {
		relative = appendProbes(relative, seg.segment, t.probeMembers)
	}
	for _, loc := range ProtectedLocations() {
		absolute = appendProbes(absolute, filepath.ToSlash(loc), t.probeMembers)
	}
	slices.Sort(relative)
	slices.Sort(absolute)
	return slices.Compact(relative), slices.Compact(absolute)
}

// appendProbes appends entry (slash-separated; a directory ends in "/") and
// each of its ancestors to probes, and below a directory each member.
func appendProbes(probes []string, entry string, members []string) []string {
	dir := strings.HasSuffix(entry, "/")
	p := strings.TrimSuffix(entry, "/")
	if dir {
		for _, m := range members {
			probes = append(probes, p+"/"+m)
		}
	}
	for ; p != "" && p != "." && p != "/"; p = path.Dir(p) {
		probes = append(probes, p)
	}
	return probes
}

// segmentCategory returns the category of the first segment the
// slash-separated path key (already case-folded when foldCase is set)
// contains, or "" when none does.
func (t *pathTables) segmentCategory(key string, foldCase bool) string {
	for _, seg := range t.segments {
		if hasPathSegment(key, foldIf(seg.segment, foldCase)) {
			return seg.category
		}
	}
	return ""
}

// ContainsProtectedPath reports whether s contains any protected path
// fragment. Unlike IsProtected (which checks a canonical path against known
// prefixes/suffixes), this performs a substring search on raw text such as
// shell commands where the path may appear anywhere in the string. It matches
// both a protected path with trailing path (`.claude/settings.json`) and a bare
// protected directory name at a path-token boundary (`rm -rf .claude`). Like
// IsProtected, it ignores case on case-insensitive filesystems.
func ContainsProtectedPath(s string) bool {
	return containsProtectedPath(s, platformMatch.foldCase)
}

func containsProtectedPath(s string, foldCase bool) bool {
	return brandedTables().containsProtectedPath(s, foldCase)
}

func (t *pathTables) containsProtectedPath(s string, foldCase bool) bool {
	normalized := filepath.ToSlash(s)
	if foldCase {
		normalized = strings.ToLower(normalized)
	}
	for _, p := range t.substrings {
		if strings.Contains(normalized, foldIf(p, foldCase)) {
			return true
		}
	}
	for _, tok := range slices.Concat(t.tokens, t.envVars) {
		if containsSegment(normalized, foldIf(tok, foldCase)) {
			return true
		}
	}
	return false
}

// foldIf lower-cases s when foldCase is set, so a branding-derived entry with
// capitals still matches a folded command.
func foldIf(s string, foldCase bool) string {
	if foldCase {
		return strings.ToLower(s)
	}
	return s
}

// containsSegment reports whether tok appears in s as a complete path segment:
// bounded on both sides by the start/end of the string or a character that is not
// part of a path token (a separator, whitespace, a shell metacharacter, a quote).
// This matches a bare protected directory name (`.claude`) while rejecting a
// longer name that merely embeds it (`my.claude.bak`, `foo.claudex`, `my.claude`).
func containsSegment(s, tok string) bool {
	for from := 0; ; {
		i := strings.Index(s[from:], tok)
		if i < 0 {
			return false
		}
		start := from + i
		end := start + len(tok)
		if isTokenBoundary(s, start-1) && isTokenBoundary(s, end) {
			return true
		}
		from = start + 1
	}
}

// isTokenBoundary reports whether index i marks a path-token boundary in s: an
// out-of-range index (the string start or end) or a byte that cannot appear
// inside a single path token.
func isTokenBoundary(s string, i int) bool {
	if i < 0 || i >= len(s) {
		return true
	}
	return !isPathTokenByte(s[i])
}

// isPathTokenByte reports whether b can appear inside one path token: an ASCII
// letter or digit, or the filename punctuation '.', '-', '_'. Every other byte
// (separators, whitespace, shell metacharacters, quotes) is a token boundary.
func isPathTokenByte(b byte) bool {
	switch {
	case b >= 'a' && b <= 'z', b >= 'A' && b <= 'Z', b >= '0' && b <= '9':
		return true
	case b == '.', b == '-', b == '_':
		return true
	default:
		return false
	}
}
