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
	"strings"
	"time"

	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

// evalTimeout bounds a single `nix eval` of a policy file.
const evalTimeout = 10 * time.Second

// cacheFormat versions the cache key, so a change to how policies are compiled
// or keyed invalidates every existing entry.
const cacheFormat = "qsdev-sandbox-policy-v2"

// CompilePolicy reads and evaluates a Nix policy file, returning the resulting
// PolicySpec. If policyPath does not exist, it returns DefaultPolicy. A policy
// file that exists but is not approved, or cannot be evaluated or parsed, is
// an error; callers must not substitute the defaults, which could be weaker
// than the user's policy.
//
// The policy comes from the repository the sandbox contains, so it is treated
// as privileged configuration: its snapshot (see Snapshot) must match the
// digest the user approved with `sandbox approve`, and it is evaluated with
// Nix's restricted evaluation confined to a private copy of that snapshot, so
// it cannot read other files, the environment or the network.
//
// `sandbox exec` compiles the policy on every hook invocation, and a `nix eval`
// can take longer than a hook's whole time budget, so the evaluated JSON is
// cached, keyed by the policy's path and snapshot digest (see policyCacheDir).
func CompilePolicy(ctx context.Context, policyPath string) (*PolicySpec, error) {
	c := compiler{eval: nixEval, approved: checkDefaultApproval}
	if store, err := DefaultApprovalStore(); err == nil {
		c.cacheDir = policyCacheDir(store)
	}
	return c.compile(ctx, policyPath)
}

// policyCacheDir returns the evaluation cache directory, beside the approval
// store under ~/.<app>/. A cache hit stands in for evaluating the approved
// policy, so the cache needs the same protection as the approvals: in a
// location an agent can write (such as the XDG cache directory) it could plant
// a weaker policy under the approved digest's key. Self-protection
// write-protects ~/.<app>/.
func policyCacheDir(store *ApprovalStore) string {
	return filepath.Join(filepath.Dir(store.Path()), "cache", "sandbox-policy")
}

// EvaluateSnapshot evaluates a policy snapshot exactly as CompilePolicy would,
// without an approval check or the cache. `sandbox approve` uses it to show
// the user what they are approving, and so that a policy that cannot be
// compiled is never approved; approving that same snapshot afterwards leaves
// no window for the files to change in between.
func EvaluateSnapshot(ctx context.Context, snap *Snapshot) (*PolicySpec, error) {
	return compiler{eval: nixEval}.evaluate(ctx, snap)
}

// checkDefaultApproval checks snap against the user's DefaultApprovalStore.
func checkDefaultApproval(snap *Snapshot) error {
	store, err := DefaultApprovalStore()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrPolicyNotApproved, err)
	}
	return store.Check(snap)
}

// compiler evaluates policy files, caching results under cacheDir when it is
// set.
type compiler struct {
	cacheDir string
	// eval evaluates the policy file at the given path, which is the entry of
	// a private snapshot copy; nixEval confines the evaluation to its
	// directory.
	eval func(ctx context.Context, policyPath string) ([]byte, error)
	// approved returns nil when the snapshot may be evaluated.
	approved func(snap *Snapshot) error
}

func (c compiler) compile(ctx context.Context, policyPath string) (*PolicySpec, error) {
	if _, err := os.Stat(policyPath); errors.Is(err, fs.ErrNotExist) {
		return DefaultPolicy(), nil
	}

	snap, err := ReadSnapshot(policyPath)
	if err != nil {
		return nil, err
	}
	if err := c.approved(snap); err != nil {
		return nil, err
	}

	cachePath := c.cachePath(snap)
	if cachePath != "" {
		if spec, err := parsePolicy(cachePath); err == nil {
			return spec, nil
		}
	}

	spec, err := c.evaluate(ctx, snap)
	if err != nil {
		return nil, err
	}

	if cachePath != "" {
		if out, err := json.Marshal(spec); err == nil {
			if err := fileutil.WriteFileAtomic(cachePath, out, 0o600); err != nil {
				slog.Debug("caching compiled sandbox policy failed", "path", cachePath, "error", err)
			}
		}
	}
	return spec, nil
}

// evaluate writes snap to a private temporary directory and evaluates its
// entry there.
func (c compiler) evaluate(ctx context.Context, snap *Snapshot) (*PolicySpec, error) {
	dir, err := os.MkdirTemp("", "qsdev-sandbox-policy-*")
	if err != nil {
		return nil, fmt.Errorf("creating policy snapshot directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	entry, err := snap.writeTo(dir)
	if err != nil {
		return nil, err
	}
	out, err := c.eval(ctx, entry)
	if err != nil {
		return nil, fmt.Errorf("evaluating policy %s: %w", snap.Path, err)
	}

	var spec PolicySpec
	if err := json.Unmarshal(out, &spec); err != nil {
		return nil, fmt.Errorf("parsing policy output from %s: %w", snap.Path, err)
	}
	return &spec, nil
}

// cachePath returns the cache file for the snapshot, or "" when caching is
// disabled.
func (c compiler) cachePath(snap *Snapshot) string {
	if c.cacheDir == "" {
		return ""
	}
	return filepath.Join(c.cacheDir, cacheKey(snap)+".json")
}

// cacheKey hashes the cache format, the policy's path and its snapshot digest.
func cacheKey(snap *Snapshot) string {
	h := sha256.New()
	for _, field := range []string{cacheFormat, snap.Path, snap.Digest} {
		_, _ = fmt.Fprintf(h, "%d:%s", len(field), field)
	}
	return hex.EncodeToString(h.Sum(nil))
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

// nixEvalArgs returns the `nix eval` arguments for the snapshot entry at
// policyPath. The --file form cannot be combined with pure evaluation, so the
// evaluation is restricted instead: restrict-eval confines file access to the
// search path, which is emptied and then set to the snapshot's own directory,
// allowed-uris is emptied so no fetcher reaches the network, getEnv returns ""
// in restricted mode, and import-from-derivation (which would build) is off.
// builtins.exec is pinned off too: nix.conf or NIX_CONFIG can enable it, and
// restricted evaluation does not disable it.
func nixEvalArgs(policyPath string) []string {
	return []string{
		"eval", "--json",
		"--extra-experimental-features", "nix-command",
		"--option", "restrict-eval", "true",
		"--option", "allowed-uris", "",
		"--option", "nix-path", "",
		"--option", "allow-import-from-derivation", "false",
		"--option", "allow-unsafe-native-code-during-evaluation", "false",
		"-I", "policy=" + filepath.Dir(policyPath),
		"--file", policyPath,
	}
}

// nixEvalEnv returns the environment for `nix eval`: the current one without
// NIX_PATH, whose entries restricted evaluation would otherwise allow.
func nixEvalEnv() []string {
	env := os.Environ()
	out := env[:0:0]
	for _, kv := range env {
		if !strings.HasPrefix(kv, "NIX_PATH=") {
			out = append(out, kv)
		}
	}
	return out
}

// nixEval evaluates the snapshot entry at policyPath with restricted
// evaluation (see nixEvalArgs).
func nixEval(ctx context.Context, policyPath string) ([]byte, error) {
	evalCtx, cancel := context.WithTimeout(ctx, evalTimeout)
	defer cancel()

	cmd := exec.CommandContext(evalCtx, "nix", nixEvalArgs(policyPath)...)
	cmd.Env = nixEvalEnv()
	out, err := cmd.Output()
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
