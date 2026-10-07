package devinit

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
	"github.com/Quantum-Serendipity/qsdev/internal/sandbox/bwrap"
	"github.com/Quantum-Serendipity/qsdev/internal/sandbox/shim"
	"github.com/Quantum-Serendipity/qsdev/internal/shebang"
)

// sandboxStoreDir is mounted read-only into every bubblewrap sandbox.
const sandboxStoreDir = sandbox.NixStoreDir

// maxSymlinkHops bounds symlink-chain resolution, matching the kernel's limit.
const maxSymlinkHops = 40

// namespaceHookCommand returns cfg.HookCommand rewritten so that every file the
// kernel opens to start it is reachable inside a bubblewrap sandbox, which
// exposes only /nix/store, the project directory and the policy's mounts:
//
//   - A command found on the host PATH, or reached through a symlink outside
//     the sandbox (e.g. /run/current-system/sw/bin/qsdev), is replaced by the
//     first path in its symlink chain that is visible inside. The final path
//     component is kept, so multi-call binaries still see their own name.
//   - A script whose interpreter is not visible (typically
//     `#!/usr/bin/env python3`) is started through its interpreter explicitly.
//   - qsdev itself (the same file as the running binary, however the hook
//     names it) is replaced by shim.SandboxPath(), where the backend mounts
//     the running binary, so a self-invoked guard runs wherever qsdev is
//     installed.
//
// A command that cannot be made reachable is an error: bwrap would otherwise
// fail with a non-blocking exit status and the wrapped guard would fail open.
func namespaceHookCommand(cfg *sandbox.SandboxConfig) ([]string, error) {
	visible := sandboxVisibility(cfg)

	exe, err := hostExecutable(cfg.HookCommand[0])
	if err != nil {
		return nil, err
	}
	if isSelf(exe) {
		return append([]string{shim.SandboxPath()}, cfg.HookCommand[1:]...), nil
	}
	exe, err = visiblePath(exe, visible)
	if err != nil {
		return nil, err
	}

	interp, err := scriptInterpreter(exe, visible)
	if err != nil {
		return nil, err
	}

	argv := make([]string, 0, len(interp)+len(cfg.HookCommand))
	argv = append(argv, interp...)
	argv = append(argv, exe)
	argv = append(argv, cfg.HookCommand[1:]...)
	return argv, nil
}

// isSelf reports whether exe is the running qsdev binary, the file the
// bubblewrap backend mounts at shim.SandboxPath(). Comparing files rather
// than paths covers PATH names, symlinks and hard links alike.
func isSelf(exe string) bool {
	self, err := shim.HostExecutable()
	if err != nil {
		return false
	}
	selfInfo, err := os.Stat(self)
	if err != nil {
		return false
	}
	exeInfo, err := os.Stat(exe)
	return err == nil && os.SameFile(selfInfo, exeInfo)
}

// sandboxVisibility reports whether a host path is visible at the same path
// inside the sandbox described by cfg. Deny-list masks are not visible, and a
// mount whose target differs from its source does not expose the host path.
func sandboxVisibility(cfg *sandbox.SandboxConfig) func(string) bool {
	roots := []string{sandboxStoreDir}
	if cfg.ProjectDir != "" {
		roots = append(roots, filepath.Clean(cfg.ProjectDir))
	}
	for _, m := range cfg.Mounts {
		src, dst := filepath.Clean(m.Source), filepath.Clean(m.Target)
		if src != dst || bwrap.IsDenyPath(src) {
			continue
		}
		roots = append(roots, dst)
	}
	deny := make([]string, 0, len(cfg.Deny))
	for _, d := range cfg.Deny {
		deny = append(deny, filepath.Clean(d))
	}
	return func(p string) bool {
		for _, d := range deny {
			if pathWithin(p, d) {
				return false // masked by a policy deny entry
			}
		}
		for _, root := range roots {
			if pathWithin(p, root) {
				return true
			}
		}
		return false
	}
}

// pathWithin reports whether p is root or lies beneath it.
func pathWithin(p, root string) bool {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// hostExecutable locates name the way a shell would (PATH lookup for a bare
// name) and returns its absolute host path.
func hostExecutable(name string) (string, error) {
	path := name
	if !strings.ContainsRune(name, filepath.Separator) {
		found, err := exec.LookPath(name)
		if err != nil {
			return "", fmt.Errorf("locating %q: %w", name, err)
		}
		path = found
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolving %q: %w", name, err)
	}
	if _, err := os.Stat(abs); err != nil {
		return "", fmt.Errorf("checking %s: %w", abs, err)
	}
	return abs, nil
}

// visiblePath follows p's symlink chain until it reaches a path that resolves
// inside the sandbox. Only as many links as needed are followed, so the returned
// path keeps the most specific name (e.g. .../coreutils/bin/cat, not
// .../coreutils).
func visiblePath(p string, visible func(string) bool) (string, error) {
	for range maxSymlinkHops {
		if resolvesInside(p, visible) {
			return p, nil
		}
		dir, err := filepath.EvalSymlinks(filepath.Dir(p))
		if err != nil {
			return "", fmt.Errorf("resolving %s: %w", p, err)
		}
		target, err := os.Readlink(p)
		if err != nil {
			// p is not a symlink; only a directory above it may have been.
			if q := filepath.Join(dir, filepath.Base(p)); resolvesInside(q, visible) {
				return q, nil
			}
			return "", fmt.Errorf("%s is not visible inside the sandbox (only %s, the project directory and the policy's mounts are)",
				p, sandboxStoreDir)
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(dir, target)
		}
		p = filepath.Clean(target)
	}
	return "", fmt.Errorf("resolving %s: too many levels of symbolic links", p)
}

// resolvesInside reports whether p is visible inside the sandbox and so is
// every link of its symlink chain. A visible path that links out of the
// sandbox (e.g. a project .venv/bin/python pointing at /usr/bin/python3) would
// dangle inside it, and bwrap would fail with a non-blocking exit status.
func resolvesInside(p string, visible func(string) bool) bool {
	for range maxSymlinkHops {
		if !visible(p) {
			return false
		}
		target, err := os.Readlink(p)
		if err != nil {
			return true // not a symlink: the chain ends inside the sandbox
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(p), target)
		}
		p = filepath.Clean(target)
	}
	return false
}

// scriptInterpreter returns the argv prefix that must run exe explicitly when
// exe is a script whose interpreter is not visible inside the sandbox. It
// returns nil for binaries and for scripts the kernel can start as-is.
func scriptInterpreter(exe string, visible func(string) bool) ([]string, error) {
	line, err := shebang.Read(exe)
	if err != nil || line.Interpreter == "" {
		return nil, err
	}
	interp, arg := line.Interpreter, line.Arg
	if resolvesInside(interp, visible) {
		return nil, nil
	}

	// `#!/usr/bin/env prog` looks prog up on PATH; do that lookup on the host so
	// the sandbox needs neither /usr/bin/env nor the host's PATH directories.
	// Only a lone program word is replaced: options and assignments are left
	// for env itself to apply.
	if prog, ok := line.EnvProgram(runtime.GOOS); ok && prog == arg {
		prog, err := hostExecutable(prog)
		if err != nil {
			return nil, fmt.Errorf("%s: interpreter: %w", exe, err)
		}
		prog, err = visiblePath(prog, visible)
		if err != nil {
			return nil, fmt.Errorf("%s: interpreter: %w", exe, err)
		}
		return []string{prog}, nil
	}

	resolved, err := visiblePath(interp, visible)
	if err != nil {
		return nil, fmt.Errorf("%s: interpreter: %w", exe, err)
	}
	if arg == "" {
		return []string{resolved}, nil
	}
	return []string{resolved, arg}, nil
}
