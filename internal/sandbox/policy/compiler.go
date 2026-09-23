package policy

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"time"

	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

// evalTimeout bounds a single `nix eval` of a policy file.
const evalTimeout = 10 * time.Second

// cacheFormat versions the cache key, so a change to how policies are compiled
// or keyed invalidates every existing entry.
const cacheFormat = "qsdev-sandbox-policy-v1"

// CompilePolicy reads and evaluates a Nix policy file, returning the resulting
// PolicySpec. If policyPath does not exist, it returns DefaultPolicy. A policy
// file that exists but cannot be evaluated or parsed is an error; callers must
// not substitute the defaults, which could be weaker than the user's policy.
//
// `sandbox exec` compiles the policy on every hook invocation, and a `nix eval`
// can take longer than a hook's whole time budget, so the evaluated JSON is
// cached in the user cache directory. The cache is content-addressed: its key
// covers the policy file's absolute path and every *.nix file in the policy's
// directory, so editing the policy (or a sibling it imports) re-evaluates it.
// Files imported from outside that directory are not part of the key.
func CompilePolicy(ctx context.Context, policyPath string) (*PolicySpec, error) {
	c := compiler{eval: nixEval}
	if dir, err := os.UserCacheDir(); err == nil {
		c.cacheDir = filepath.Join(dir, "qsdev", "sandbox-policy")
	}
	return c.compile(ctx, policyPath)
}

// compiler evaluates policy files, caching results under cacheDir when it is
// set.
type compiler struct {
	cacheDir string
	eval     func(ctx context.Context, policyPath string) ([]byte, error)
}

func (c compiler) compile(ctx context.Context, policyPath string) (*PolicySpec, error) {
	if _, err := os.Stat(policyPath); errors.Is(err, fs.ErrNotExist) {
		return DefaultPolicy(), nil
	}

	cachePath := c.cachePath(policyPath)
	if cachePath != "" {
		if spec, err := parsePolicy(cachePath); err == nil {
			return spec, nil
		}
	}

	out, err := c.eval(ctx, policyPath)
	if err != nil {
		return nil, fmt.Errorf("evaluating policy %s: %w", policyPath, err)
	}

	var spec PolicySpec
	if err := json.Unmarshal(out, &spec); err != nil {
		return nil, fmt.Errorf("parsing policy output from %s: %w", policyPath, err)
	}

	if cachePath != "" {
		if err := fileutil.WriteFileAtomic(cachePath, out, 0o600); err != nil {
			slog.Debug("caching compiled sandbox policy failed", "path", cachePath, "error", err)
		}
	}
	return &spec, nil
}

// cachePath returns the cache file for the policy's current content, or "" when
// caching is disabled or the key cannot be computed.
func (c compiler) cachePath(policyPath string) string {
	if c.cacheDir == "" {
		return ""
	}
	key, err := cacheKey(policyPath)
	if err != nil {
		slog.Debug("sandbox policy cache key unavailable", "path", policyPath, "error", err)
		return ""
	}
	return filepath.Join(c.cacheDir, key+".json")
}

// cacheKey hashes the policy file's absolute path and the name and content of
// every *.nix file in its directory (which includes the policy file itself).
func cacheKey(policyPath string) (string, error) {
	abs, err := filepath.Abs(policyPath)
	if err != nil {
		return "", fmt.Errorf("resolving %s: %w", policyPath, err)
	}
	siblings, err := filepath.Glob(filepath.Join(filepath.Dir(abs), "*.nix"))
	if err != nil {
		return "", fmt.Errorf("listing policy directory: %w", err)
	}
	if !slices.Contains(siblings, abs) {
		siblings = append(siblings, abs)
	}
	slices.Sort(siblings)

	h := sha256.New()
	writeField := func(b []byte) {
		_, _ = fmt.Fprintf(h, "%d:", len(b))
		_, _ = h.Write(b)
	}
	writeField([]byte(cacheFormat))
	writeField([]byte(abs))
	for _, p := range siblings {
		data, err := os.ReadFile(p) //nolint:gosec // p is the policy file or a sibling in its directory
		if err != nil {
			return "", fmt.Errorf("reading %s: %w", p, err)
		}
		writeField([]byte(p))
		writeField(data)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// parsePolicy reads a cached policy evaluation.
func parsePolicy(path string) (*PolicySpec, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path is derived from the cache dir and a hex digest
	if err != nil {
		return nil, fmt.Errorf("reading cached policy: %w", err)
	}
	var spec PolicySpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("parsing cached policy %s: %w", path, err)
	}
	return &spec, nil
}

// nixEval evaluates policyPath with `nix eval --json`.
func nixEval(ctx context.Context, policyPath string) ([]byte, error) {
	evalCtx, cancel := context.WithTimeout(ctx, evalTimeout)
	defer cancel()

	out, err := exec.CommandContext(evalCtx, "nix", "eval", "--json", "-f", policyPath).Output()
	if err != nil {
		// Output captures nix's stderr in the ExitError; surface it, since it
		// says what is wrong with the policy.
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && len(bytes.TrimSpace(exitErr.Stderr)) > 0 {
			return nil, fmt.Errorf("nix eval: %w: %s", err, bytes.TrimSpace(exitErr.Stderr))
		}
		return nil, fmt.Errorf("nix eval: %w", err)
	}
	return out, nil
}
