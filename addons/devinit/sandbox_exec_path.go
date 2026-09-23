package devinit

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
	"github.com/Quantum-Serendipity/qsdev/internal/sandbox/bwrap"
)

// sandboxStoreDir is mounted read-only into every bubblewrap sandbox.
const sandboxStoreDir = "/nix/store"

// maxSymlinkHops bounds symlink-chain resolution, matching the kernel's limit.
const maxSymlinkHops = 40

// shebangBufSize is the number of leading bytes the kernel reads to parse an
// interpreter line (BINPRM_BUF_SIZE).
const shebangBufSize = 256

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
//
// A command that cannot be made reachable is an error: bwrap would otherwise
// fail with a non-blocking exit status and the wrapped guard would fail open.
func namespaceHookCommand(cfg *sandbox.SandboxConfig) ([]string, error) {
	visible := sandboxVisibility(cfg)

	exe, err := hostExecutable(cfg.HookCommand[0])
	if err != nil {
		return nil, err
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
	return func(p string) bool {
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
	line, err := readShebang(exe)
	if err != nil || line == "" {
		return nil, err
	}

	// Linux semantics: the interpreter ends at the first blank, and everything
	// after it (trimmed) is passed as one optional argument.
	interp, arg := line, ""
	if i := strings.IndexAny(line, " \t"); i >= 0 {
		interp, arg = line[:i], strings.TrimSpace(line[i+1:])
	}
	if interp == "" {
		return nil, fmt.Errorf("%s: empty interpreter line", exe)
	}
	if resolvesInside(interp, visible) {
		return nil, nil
	}

	// `#!/usr/bin/env prog` looks prog up on PATH; do that lookup on the host so
	// the sandbox needs neither /usr/bin/env nor the host's PATH directories.
	if filepath.Base(interp) == "env" && arg != "" && !strings.HasPrefix(arg, "-") && !strings.ContainsAny(arg, " \t") {
		prog, err := hostExecutable(arg)
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

// readShebang returns the interpreter line of a script (without the leading
// "#!"), or "" when the file does not start with one.
func readShebang(path string) (string, error) {
	f, err := os.Open(path) //nolint:gosec // path is the hook command the caller asked to run
	if err != nil {
		return "", fmt.Errorf("opening %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	buf := make([]byte, shebangBufSize)
	n, err := io.ReadFull(f, buf)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("reading %s: %w", path, err)
	}
	head := buf[:n]
	if !bytes.HasPrefix(head, []byte("#!")) {
		return "", nil
	}
	line, _, _ := bytes.Cut(head[2:], []byte("\n"))
	return strings.TrimSpace(string(line)), nil
}
