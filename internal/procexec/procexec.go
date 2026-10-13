// Package procexec is the one place qsdev starts child processes.
//
// Command and CommandContext mirror os/exec, so routing a call site through
// the package is a one-line change. Under ForbidExecEnv every command that
// is not one of the declared localProbes panics with its argv: tests set it
// to prove that a read-only command starts nothing that fetches, executes
// project code or mutates state. VersionProbe is the one name-agnostic
// entry point, for data-driven tool detection.
package procexec

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

// ForbidExecEnv, when non-empty, makes every non-allowlisted exec panic.
const ForbidExecEnv = "QSDEV_TEST_FORBID_EXEC"

// localProbe is an argv shape of a named host binary that only queries the
// local host: it neither contacts the network nor runs project code nor
// writes anything. Every entry names its binary; an argv shape that is safe
// for any binary does not exist (`npx version` fetches, `./gradlew
// --version` downloads, `make version` runs project rules).
type localProbe struct {
	// Name is the normalised base name of the binary.
	Name string
	// Args must equal the arguments exactly, or start them when Prefix is set.
	Args   []string
	Prefix bool
	// Kind is what the probe asks the binary.
	Kind ProbeKind
}

// ProbeKind is what a local probe asks its binary.
type ProbeKind int

const (
	// ProbeVersion asks for the binary's version.
	ProbeVersion ProbeKind = iota + 1
	// ProbeSyntax checks the syntax of its input: the exit status says
	// whether it is valid.
	ProbeSyntax
	// ProbeQuery asks the host a question whose answer the caller parses:
	// a user entry, a process name, a container runtime's mode, a
	// repository's state.
	ProbeQuery
)

// localProbes is the single allowlist of execs a read-only command may run.
var localProbes = []localProbe{
	// A query, not only a version: container detection reads whether the
	// answer names Podman, whose Docker alias it is.
	{Name: "docker", Args: []string{"--version"}, Kind: ProbeQuery},
	{Name: "docker", Args: []string{"compose", "version"}, Kind: ProbeVersion},
	{Name: "podman", Args: []string{"info", "--format"}, Prefix: true, Kind: ProbeQuery},
	{Name: "podman", Args: []string{"version", "--format"}, Prefix: true, Kind: ProbeQuery},
	{Name: "git", Args: []string{"rev-parse"}, Prefix: true, Kind: ProbeQuery},
	// internal/gitrepo's diff, stash show and log: no lock, fsmonitor,
	// external diff, textconv or signature check, and every later argument is
	// a revision or pathspec.
	{Name: "git", Args: []string{
		"--no-optional-locks", "-c", "core.fsmonitor=false", "diff",
		"--name-only", "-z", "--no-relative", "--no-renames", "--no-ext-diff", "--no-textconv", "--end-of-options",
	}, Prefix: true, Kind: ProbeQuery},
	{Name: "git", Args: []string{
		"--no-optional-locks", "stash", "show", "--name-only", "-z", "--no-renames", "--include-untracked", "--end-of-options",
	}, Prefix: true, Kind: ProbeQuery},
	{Name: "git", Args: []string{
		"--no-optional-locks", "log", "--format=", "--name-only", "-z", "--no-relative", "--no-renames",
		"--no-show-signature", "--no-walk", "--diff-merges=separate", "--end-of-options",
	}, Prefix: true, Kind: ProbeQuery},
	// internal/gitrepo's patch listing: --numstat turns applying off, and
	// the rest is -R, -p<n>, --directory=<d>, "--" and the patch files.
	{Name: "git", Args: []string{"--no-optional-locks", "apply", "--numstat", "-z"}, Prefix: true, Kind: ProbeQuery},
	// internal/gitrepo's index listing, exactly: no pathspec, no options.
	{Name: "git", Args: []string{"--no-optional-locks", "-c", "core.fsmonitor=false", "ls-files", "-z"}, Kind: ProbeQuery},
	{Name: "getent", Args: []string{"passwd"}, Prefix: true, Kind: ProbeQuery},
	{Name: "nix-instantiate", Args: []string{"--parse", "-"}, Kind: ProbeSyntax},
	{Name: "bash", Args: []string{"-n"}, Kind: ProbeSyntax},
	{Name: "ps", Args: []string{"-p"}, Prefix: true, Kind: ProbeQuery},
	{Name: "cmd", Args: []string{"/c", "ver"}, Kind: ProbeQuery},
	{Name: "xcode-select", Args: []string{"-p"}, Kind: ProbeQuery},
	{Name: "ghc", Args: []string{"--numeric-version"}, Kind: ProbeVersion},
	{Name: "sw_vers", Args: []string{"-productVersion"}, Kind: ProbeQuery},
	{Name: "sw_vers", Args: []string{"-productName"}, Kind: ProbeQuery},
	{Name: "uname", Args: []string{"-r"}, Kind: ProbeQuery},
	{Name: "sysctl", Args: []string{"-n", "sysctl.proc_translated"}, Kind: ProbeQuery},
}

// LocalProbeNames returns the binaries the forbid-exec guard lets a command
// run by name (see localProbes) whose every allowed probe is of one of
// kinds, or every such binary when kinds is empty; sorted, without repeats.
// A hermetic test can stand an empty program in for a binary probed only for
// its version or for syntax, which reads as a tool of unknown version or
// valid input, but not for one it queries, whose empty answer the caller
// would parse as a different host.
func LocalProbeNames(kinds ...ProbeKind) []string {
	names := make([]string, 0, len(localProbes))
	for _, p := range localProbes {
		names = append(names, p.Name)
	}
	slices.Sort(names)
	return slices.DeleteFunc(slices.Compact(names), func(name string) bool {
		return len(kinds) > 0 && slices.ContainsFunc(localProbes, func(p localProbe) bool {
			return p.Name == name && !slices.Contains(kinds, p.Kind)
		})
	})
}

// Command is exec.Command behind the forbid-exec guard.
func Command(name string, args ...string) *exec.Cmd {
	guard(name, args)
	return exec.Command(name, args...) //nolint:gosec // argv is an explicit array; no shell interpolation
}

// CommandContext is exec.CommandContext behind the forbid-exec guard.
func CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	guard(name, args)
	return exec.CommandContext(ctx, name, args...) //nolint:gosec // argv is an explicit array; no shell interpolation
}

// LookPath is exec.LookPath: it resolves a bare name through PATH and checks
// a name containing a path separator directly. It starts nothing, so the
// forbid-exec guard does not apply.
func LookPath(file string) (string, error) {
	path, err := exec.LookPath(file)
	if err != nil {
		return "", fmt.Errorf("procexec: %w", err)
	}
	return path, nil
}

// ErrInsideProject marks a version probe VersionProbe refused because the
// binary lies inside the project.
var ErrInsideProject = errors.New("binary is inside the project; it would run project code")

// versionFlags are the arguments VersionProbe accepts.
var versionFlags = []string{"--version", "version", "-v"}

// NeutralDir is the working directory for probes that query the host, not
// the project. Run from the project, a toolchain shim or manager reads the
// repository's pins and may download and run what they name: `go version`
// under GOTOOLCHAIN=auto follows go.mod's go/toolchain line, and mise or
// asdf shims follow .tool-versions. On Windows a probe grandchild that
// outlives a timeout would also hold the project directory open.
func NeutralDir() string {
	return os.TempDir()
}

// VersionProbe returns a command that runs the binary at absPath with the
// single argument flag to print its version. It serves data-driven tool
// detection, where the binary comes from a catalog: absPath must be an
// absolute path (as exec.LookPath returns) to an existing file, flag one of
// --version, version or -v, and the binary must lie outside project, the
// directory whose content belongs to the project the caller works in ("" for
// none). Anything else is never started: the command carries the error in
// Err, which Start and Run return. The command runs in NeutralDir, never in
// the caller's working directory, so repository content cannot steer what
// the probe fetches or runs. It is the only name-agnostic exec the
// forbid-exec guard allows.
func VersionProbe(ctx context.Context, project, absPath, flag string) *exec.Cmd {
	if err := checkVersionProbe(project, absPath, flag); err != nil {
		cmd := &exec.Cmd{Path: absPath, Args: []string{absPath, flag}}
		cmd.Err = err
		return cmd
	}
	cmd := exec.CommandContext(ctx, absPath, flag) //nolint:gosec // absolute catalog binary with a fixed version flag
	cmd.Dir = NeutralDir()
	return cmd
}

// IsVersionFlag reports whether VersionProbe accepts flag.
func IsVersionFlag(flag string) bool {
	return slices.Contains(versionFlags, flag)
}

func checkVersionProbe(project, absPath, flag string) error {
	if !filepath.IsAbs(absPath) {
		return fmt.Errorf("procexec: version probe needs an absolute binary path, got %q", absPath)
	}
	if !IsVersionFlag(flag) {
		return fmt.Errorf("procexec: version probe flag %q is not one of %q", flag, versionFlags)
	}
	return checkOutsideProject(project, absPath)
}

// checkOutsideProject refuses a binary that lies inside project by its own
// path or by the path it resolves to. PATH can name such a binary
// (node_modules/.bin, a virtualenv, a committed bin/), and running it is
// running project code wherever the probe's cwd is. Both forms are checked:
// the project chooses the lexical one (a project symlink can give any host
// binary a catalog name) and the resolved one is what runs. A binary that
// cannot be resolved is refused.
func checkOutsideProject(project, absPath string) error {
	resolved, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		return fmt.Errorf("procexec: version probe binary %q: %w", absPath, err)
	}
	if project == "" {
		return nil
	}
	projectInfo, err := os.Stat(project)
	if err != nil {
		return fmt.Errorf("procexec: version probe project: %w", err)
	}
	for _, p := range []string{absPath, resolved} {
		if isUnder(projectInfo, filepath.Dir(p)) {
			return fmt.Errorf("procexec: version probe binary %q under %s: %w", absPath, project, ErrInsideProject)
		}
	}
	return nil
}

// isUnder reports whether dir, or any directory above it, is the directory
// root describes. Comparing file identity (os.SameFile) rather than strings
// holds across symlinked ancestors and case-insensitive filesystems.
func isUnder(root os.FileInfo, dir string) bool {
	for dir = filepath.Clean(dir); ; {
		if info, err := os.Stat(dir); err == nil && os.SameFile(info, root) {
			return true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return false
		}
		dir = parent
	}
}

// guard panics when ForbidExecEnv is set and argv is not a local probe. A
// forbidden exec is a programmer error that an error return could hide:
// several callers treat a failed probe as "tool absent".
func guard(name string, args []string) {
	if os.Getenv(ForbidExecEnv) == "" || isLocalProbe(name, args) {
		return
	}
	argv := strings.Join(append([]string{name}, args...), " ")
	panic(fmt.Sprintf("procexec: exec forbidden under %s: %s", ForbidExecEnv, argv))
}

func isLocalProbe(name string, args []string) bool {
	base := baseName(name)
	if base == "git" && len(args) >= 2 && args[0] == "-C" {
		args = args[2:]
	}
	return slices.ContainsFunc(localProbes, func(p localProbe) bool {
		if p.Name != base {
			return false
		}
		if p.Prefix {
			return len(args) >= len(p.Args) && slices.Equal(args[:len(p.Args)], p.Args)
		}
		return slices.Equal(args, p.Args)
	})
}

// baseName strips any directory (either separator, whatever the host OS),
// lowercases and drops a Windows ".exe" suffix.
func baseName(name string) string {
	base := strings.ToLower(name[strings.LastIndexAny(name, `/\`)+1:])
	return strings.TrimSuffix(base, ".exe")
}
