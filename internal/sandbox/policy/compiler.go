package policy

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"
)

// CompilePolicy reads and evaluates a Nix policy file, returning the resulting
// PolicySpec. If policyPath does not exist, it returns DefaultPolicy.
func CompilePolicy(policyPath string) (*PolicySpec, error) {
	if _, err := os.Stat(policyPath); os.IsNotExist(err) {
		return DefaultPolicy(), nil
	}

	out, err := exec.Command("nix", "eval", "--json", "-f", policyPath).Output()
	if err != nil {
		return nil, fmt.Errorf("evaluating policy %s: %w", policyPath, err)
	}

	var spec PolicySpec
	if err := json.Unmarshal(out, &spec); err != nil {
		return nil, fmt.Errorf("parsing policy output from %s: %w", policyPath, err)
	}

	return &spec, nil
}

// cachedPolicy caches a compiled PolicySpec, re-evaluating only when the
// underlying file's modification time changes.
type cachedPolicy struct {
	mu      sync.Mutex
	path    string
	modTime time.Time
	spec    *PolicySpec
}

// NewCachedPolicy creates a new cached policy reader for the given path.
func NewCachedPolicy(path string) *cachedPolicy {
	return &cachedPolicy{path: path}
}

// Get returns the cached PolicySpec, recompiling if the file has been modified
// since the last call. If the file does not exist, DefaultPolicy is returned
// and cached until a file appears.
func (c *cachedPolicy) Get() (*PolicySpec, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	info, err := os.Stat(c.path)
	if os.IsNotExist(err) {
		if c.spec == nil {
			c.spec = DefaultPolicy()
			c.modTime = time.Time{}
		}
		return c.spec, nil
	}
	if err != nil {
		return nil, fmt.Errorf("stating policy file %s: %w", c.path, err)
	}

	if c.spec != nil && info.ModTime().Equal(c.modTime) {
		return c.spec, nil
	}

	spec, err := CompilePolicy(c.path)
	if err != nil {
		return nil, err
	}

	c.spec = spec
	c.modTime = info.ModTime()

	return c.spec, nil
}
