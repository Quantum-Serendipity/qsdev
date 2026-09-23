package bwrap

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
	"github.com/Quantum-Serendipity/qsdev/internal/sandbox/denylist"
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
	if cfg.ProjectDir != "" {
		if cfg.HookCategory.WorktreeReadOnly() {
			args = append(args, "--ro-bind", cfg.ProjectDir, cfg.ProjectDir)
		} else {
			args = append(args, "--bind", cfg.ProjectDir, cfg.ProjectDir)
		}
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

	// 9. Mask every deny entry. Emitted LAST so the mask always wins over any
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
// at its own location, plus its image under any extra mount whose Source is an
// ancestor of it (a mount of /opt at /mnt/opt re-exposes /opt/secret at
// /mnt/opt/secret). An entry that does not exist on the host has nothing to
// expose and is skipped: masking it would make bwrap create a mount point,
// which fails beneath a read-only bind and would break every hook.
func denyMasks(deny []string, mounts []sandbox.MountSpec) []denyMask {
	var out []denyMask
	seen := make(map[string]bool)
	add := func(host, target string) {
		if !seen[target] {
			seen[target] = true
			out = append(out, denyMask{host: host, target: target})
		}
	}
	for _, d := range deny {
		if _, err := os.Lstat(d); errors.Is(err, fs.ErrNotExist) {
			continue
		}
		add(d, d)
		for _, m := range mounts {
			for _, src := range denylist.CandidatePaths(m.Source) {
				if !denylist.IsStrictAncestor(src, d) {
					continue
				}
				if rel, err := filepath.Rel(src, d); err == nil {
					add(d, filepath.Join(m.Target, rel))
				}
			}
		}
	}
	return out
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
