package bwrap

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
	"github.com/Quantum-Serendipity/qsdev/internal/sandbox/denylist"
	"github.com/Quantum-Serendipity/qsdev/internal/sandbox/shim"
)

// BuildArgs constructs the bwrap command-line arguments for the given sandbox
// configuration and degradation tier. The returned slice does NOT include
// "bwrap" itself; the caller prepends that. It validates all mount source paths
// and the project directory before constructing arguments.
func BuildArgs(cfg *sandbox.SandboxConfig, _ sandbox.DegradationTier) ([]string, error) {
	if cfg.ProjectDir != "" {
		if err := ValidateMountPath(cfg.ProjectDir); err != nil {
			return nil, fmt.Errorf("validating project dir: %w", err)
		}
	}

	if err := rejectReservedTargets(cfg); err != nil {
		return nil, err
	}

	deny, err := normalizeDenyPaths(cfg.Deny)
	if err != nil {
		return nil, err
	}

	for _, m := range cfg.Mounts {
		if err := ValidateMountPath(m.Source); err != nil {
			return nil, fmt.Errorf("validating mount source: %w", err)
		}
		if err := ValidateMountPath(m.Target); err != nil {
			return nil, fmt.Errorf("validating mount target: %w", err)
		}
		if d, ok := matchedDenyPath(denylist.CandidatePaths(m.Source), deny); ok {
			return nil, fmt.Errorf("mount source %q is denied by policy: overlaps %q", m.Source, d)
		}
	}

	for _, p := range cfg.NixStorePaths {
		if err := ValidateMountPath(p); err != nil {
			return nil, fmt.Errorf("validating nix store path: %w", err)
		}
	}

	var args []string

	// 1. Namespace flags.
	args = append(args, "--unshare-user", "--unshare-pid", "--unshare-ipc", "--unshare-uts")

	// The network decision comes from the resolved mode alone, so an explicit
	// "deny" always wins over a category that would otherwise get the network.
	networkIsolated := cfg.NetworkIsolated()
	if networkIsolated {
		args = append(args, "--unshare-net")
	}

	// 2. Session control.
	args = append(args, "--die-with-parent", "--new-session")

	// 3. Core filesystem.
	args = append(args, "--dev", "/dev", "--proc", "/proc", "--tmpfs", "/tmp")

	// 4. Nix store (read-only).
	args = append(args, "--ro-bind", "/nix/store", "/nix/store")

	// 5. Project directory. Only bind it when one is configured; binding an
	// empty path emits `--ro-bind "" ""`, which bwrap rejects with "Can't find
	// source path" and would break `sandbox exec` (which sets no ProjectDir).
	// Then pin the directories holding the project's control plane
	// (guardrailPins), with the same access, before any other mount can
	// change what that location shows.
	guardrails := sandbox.GuardrailPaths(cfg.ProjectDir)
	if cfg.ProjectDir != "" {
		args = append(args, projectBind(cfg), cfg.ProjectDir, cfg.ProjectDir)
		args = append(args, guardrailPins(cfg, guardrails)...)
	}

	// 6. System files (always read-only).
	args = append(args,
		"--ro-bind", "/etc/passwd", "/etc/passwd",
		"--ro-bind", "/etc/group", "/etc/group",
		"--ro-bind", "/etc/hosts", "/etc/hosts",
	)
	if !networkIsolated {
		args = append(args, "--ro-bind", "/etc/resolv.conf", "/etc/resolv.conf")
	}

	// 7. Extra mounts.
	for _, m := range cfg.Mounts {
		if m.ReadOnly {
			args = append(args, "--ro-bind", m.Source, m.Target)
		} else {
			args = append(args, "--bind", m.Source, m.Target)
		}
	}

	// 8. Additional Nix store paths.
	for _, p := range cfg.NixStorePaths {
		args = append(args, "--ro-bind", p, p)
	}

	// 9. Guardrail overlays: re-bind the project's control plane read-only
	// over every writable view of it. Emitted after the project bind and the
	// policy mounts, so no mount can re-widen it, and before the deny masks,
	// which still win. Then the directories hooks legitimately write inside
	// them (the hook logs, devenv's state) are made writable again.
	args = append(args, guardrailOverlays(cfg, guardrails)...)
	args = append(args, writableDirBinds(cfg, guardrails)...)

	// 10. Mask every deny entry. Emitted LAST so the mask always wins over any
	// earlier bind: even if a broader mount above exposed an ancestor directory
	// (e.g. $HOME or a policy extra mount), the denied path underneath is
	// replaced with an empty tmpfs (directories) or a read-only /dev/null
	// (files) and can be neither read nor written. The masks are trusted
	// directives, so they bypass ValidateMountPath (which would reject them).
	for _, m := range denyMasks(deny, cfg.Mounts) {
		args = append(args, m.args()...)
	}

	return args, nil
}

// rejectReservedTargets refuses a project dir or mount target on or under
// shim.SandboxRoot(): RunHook mounts qsdev itself there, as the trusted shim,
// and a configured bind must never replace or shadow it.
func rejectReservedTargets(cfg *sandbox.SandboxConfig) error {
	root := shim.SandboxRoot()
	targets := []string{cfg.ProjectDir}
	for _, m := range cfg.Mounts {
		targets = append(targets, m.Target)
	}
	for _, t := range targets {
		if t != "" && denylist.Overlaps(filepath.Clean(t), root) {
			return fmt.Errorf("sandbox path %q overlaps %s, which is reserved for qsdev's own mounts", t, root)
		}
	}
	return nil
}

// normalizeDenyPaths validates the configured deny entries and returns them
// cleaned, together with their symlink-resolved forms, without duplicates. A
// relative entry cannot be located inside the sandbox, so it is rejected
// rather than silently ignored.
func normalizeDenyPaths(deny []string) ([]string, error) {
	var out []string
	seen := make(map[string]bool)
	for _, p := range deny {
		if !filepath.IsAbs(p) {
			return nil, fmt.Errorf("deny path must be absolute: %q", p)
		}
		for _, c := range denylist.CandidatePaths(p) {
			if !seen[c] {
				seen[c] = true
				out = append(out, c)
			}
		}
	}
	return out, nil
}

// denyMask is one masking directive: the host path whose contents must stay
// hidden and the in-sandbox path where they would otherwise appear.
type denyMask struct {
	host   string
	target string
}

// denyMasks returns every masking directive for the deny entries: each entry
// at every location it appears in the sandbox (mountImages). An entry that
// does not exist on the host has nothing to expose and is skipped: masking it
// would make bwrap create a mount point, which fails beneath a read-only bind
// and would break every hook.
func denyMasks(deny []string, mounts []sandbox.MountSpec) []denyMask {
	var out []denyMask
	seen := make(map[string]bool)
	for _, d := range deny {
		if _, err := os.Lstat(d); errors.Is(err, fs.ErrNotExist) {
			continue
		}
		for _, target := range mountImages(d, mounts) {
			if !seen[target] {
				seen[target] = true
				out = append(out, denyMask{host: d, target: target})
			}
		}
	}
	return out
}

// mountImages returns every in-sandbox location where host path p appears:
// p itself, plus its image under each mount whose Source is a strict
// ancestor of it (a mount of /opt at /mnt/opt re-exposes /opt/secret at
// /mnt/opt/secret).
//
// Both paths are compared in every spelling (denylist.CandidatePaths), so a
// mount of the symlink-resolved directory still re-exposes a p spelled
// through the link. The result may repeat a location; callers deduplicate.
func mountImages(p string, mounts []sandbox.MountSpec) []string {
	out := []string{p}
	spellings := denylist.CandidatePaths(p)
	for _, m := range mounts {
		for _, src := range denylist.CandidatePaths(m.Source) {
			for _, hp := range spellings {
				if !denylist.IsStrictAncestor(src, hp) {
					continue
				}
				if rel, err := filepath.Rel(src, hp); err == nil {
					out = append(out, filepath.Join(m.Target, rel))
				}
			}
		}
	}
	return out
}

// guardrailsExposed reports whether the sandbox gives the hook a writable
// view of the project, through the worktree bind or a writable policy mount,
// so the project's guardrail paths (sandbox.GuardrailPaths) need protecting.
func guardrailsExposed(cfg *sandbox.SandboxConfig) bool {
	if cfg.ProjectDir == "" {
		return false
	}
	return !cfg.WorktreeReadOnly() || slices.ContainsFunc(cfg.Mounts, func(m sandbox.MountSpec) bool { return !m.ReadOnly })
}

// projectBind returns the bwrap bind option for the project directory: its
// access is the category's worktree access.
func projectBind(cfg *sandbox.SandboxConfig) string {
	if cfg.WorktreeReadOnly() {
		return "--ro-bind"
	}
	return "--bind"
}

// guardrailPins returns the bwrap arguments that bind every in-project
// directory holding a guardrail (.git for .git/hooks) onto itself, with the
// project bind's access, when the hook gets a writable view of the project.
// A mount point cannot be renamed or removed: rename(2) and rmdir(2) fail
// with EBUSY for a dentry mounted on anywhere in the namespace, so through
// every view of it, a policy mount's included. A hook therefore cannot move
// the directory aside and plant a writable replacement beside the read-only
// overlays. They are emitted straight after the project bind, so each pin
// shows exactly what the project bind did there (a bind does not keep its
// source's read-only flag, so a later pin could widen access), and the
// overlays mount on top of them.
func guardrailPins(cfg *sandbox.SandboxConfig, guardrails []string) []string {
	if !guardrailsExposed(cfg) {
		return nil
	}
	var dirs []string
	for _, g := range guardrails {
		host, ok := projectLocation(cfg.ProjectDir, g)
		if !ok {
			continue
		}
		for d := filepath.Dir(host); denylist.IsStrictAncestor(cfg.ProjectDir, d); d = filepath.Dir(d) {
			dirs = append(dirs, d)
		}
	}
	slices.Sort(dirs) // a parent sorts before its children, so is pinned first
	var args []string
	for _, d := range slices.Compact(dirs) {
		args = append(args, projectBind(cfg), d, d)
	}
	return args
}

// guardrailOverlays returns the bwrap arguments that bind every existing
// guardrail path (guardrails, sandbox.GuardrailPaths) read-only at each
// location it appears in the sandbox. A guardrail absent on the host is
// skipped (binding it would create it on the host through the writable
// project bind); RunHook detects its creation instead. Like the deny masks
// they are trusted directives and bypass ValidateMountPath.
func guardrailOverlays(cfg *sandbox.SandboxConfig, guardrails []string) []string {
	if !guardrailsExposed(cfg) {
		return nil
	}
	var args []string
	seen := make(map[string]bool)
	for _, g := range guardrails {
		host, ok := projectLocation(cfg.ProjectDir, g)
		if !ok {
			continue
		}
		for _, target := range mountImages(host, cfg.Mounts) {
			if !seen[target] {
				seen[target] = true
				args = append(args, "--ro-bind", host, target)
			}
		}
	}
	return args
}

// writableDirBinds returns the bwrap arguments that keep the directories
// inside the guardrails that hooks legitimately write
// (sandbox.WritableGuardrailDirs) writable under the read-only overlays when
// the worktree is writable: the hook log directory inside .claude, where the
// generated audit and analytics hooks append, and devenv's state directory
// inside .devenv, which holds GOPATH and the venv a test runner fills.
// Bwrap cannot tell hooks apart, so each is writable for every hook with a
// writable worktree: any of them can rewrite the logs or plant a symlink
// there, and the logging hooks refuse to append through a symlink for that
// reason. Each is bound only as a real directory (not a symlink) inside the
// project that is no guardrail and holds none, so it can never re-widen the
// control plane. RunHook creates them beforehand (prepareWritableDirs), as a
// hook could not under the overlay.
func writableDirBinds(cfg *sandbox.SandboxConfig, guardrails []string) []string {
	if cfg.ProjectDir == "" || cfg.WorktreeReadOnly() {
		return nil
	}
	var args []string
	for _, dir := range sandbox.WritableGuardrailDirs(cfg.ProjectDir) {
		if host, ok := writableDir(cfg.ProjectDir, dir, guardrails); ok {
			args = append(args, "--bind", host, host)
		}
	}
	return args
}

// writableDir returns where dir lives in the project when it may be bound
// writable (see writableDirBinds).
func writableDir(projectDir, dir string, guardrails []string) (string, bool) {
	if fi, err := os.Lstat(dir); err != nil || !fi.IsDir() {
		return "", false
	}
	host, ok := projectLocation(projectDir, dir)
	if !ok {
		return "", false
	}
	for _, g := range guardrails {
		if loc, ok := projectLocation(projectDir, g); ok && denylist.Overlaps(loc, host) {
			return "", false
		}
	}
	return host, true
}

// prepareWritableDirs creates, for a writable worktree, the directories
// writableDirBinds keeps writable, so a hook can write them under the
// overlays: the hooks' log directory when .claude is a directory inside the
// project, and devenv's state directory (with .devenv) when the project uses
// devenv (sandbox.UsesDevenv) and .devenv, if present, lies inside the
// project. Best effort: without them a write there fails as it does under a
// read-only worktree, and the hook itself still runs.
func prepareWritableDirs(cfg *sandbox.SandboxConfig) {
	if cfg.ProjectDir == "" || cfg.WorktreeReadOnly() {
		return
	}
	logs := sandbox.HookLogDir(cfg.ProjectDir)
	if _, ok := projectLocation(cfg.ProjectDir, filepath.Dir(logs)); ok {
		_ = os.Mkdir(logs, 0o700)
	}
	if !sandbox.UsesDevenv(cfg.ProjectDir) {
		return
	}
	state := sandbox.DevenvStateDir(cfg.ProjectDir)
	dotDir := filepath.Dir(state)
	if _, err := os.Lstat(dotDir); err == nil {
		if _, ok := projectLocation(cfg.ProjectDir, dotDir); !ok {
			return
		}
	}
	_ = os.MkdirAll(state, 0o755)
}

// projectLocation returns where p's content lives inside projectDir, spelled
// under projectDir as the sandbox sees it: p itself, or the in-project file a
// symlink at p resolves to. It reports false when p does not exist or
// resolves outside the project, where the project bind does not expose it
// (binding it would make bwrap fail, as its target is missing in the
// sandbox); a symlink's own replacement is caught by RunHook's snapshot.
func projectLocation(projectDir, p string) (string, bool) {
	real, err := filepath.EvalSymlinks(p)
	if err != nil {
		return "", false
	}
	root, err := filepath.EvalSymlinks(projectDir)
	if err != nil {
		return "", false
	}
	rel, err := filepath.Rel(root, real)
	if err != nil || !filepath.IsLocal(rel) {
		return "", false
	}
	return filepath.Join(projectDir, rel), true
}

// args returns the bwrap arguments that mask one deny path so its contents can
// never be read or written inside the sandbox, even if a broader bind exposed
// one of its ancestor directories. A directory is masked with an empty tmpfs; a
// regular file with a read-only bind of /dev/null. Callers must emit these
// AFTER every bind so the mask wins.
func (m denyMask) args() []string {
	if info, err := os.Stat(m.host); err == nil && !info.IsDir() {
		return []string{"--ro-bind", "/dev/null", m.target}
	}
	return []string{"--tmpfs", m.target}
}
