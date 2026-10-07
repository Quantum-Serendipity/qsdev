package devenv

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"gopkg.in/yaml.v3"
)

// devenvLockFile is the lock file devenv writes next to devenv.yaml.
const devenvLockFile = "devenv.lock"

// UnlockedInputs returns, sorted, the flake inputs the devenv.yaml content
// yamlContent declares that projectRoot's devenv.lock does not pin yet. devenv
// resolves such an input to whatever its branch holds on the next shell
// entry, so the lock change must be reviewed and committed like any other.
// A project without a devenv.lock yet has nothing to compare and yields nil.
func UnlockedInputs(projectRoot string, yamlContent []byte) ([]string, error) {
	data, err := os.ReadFile(filepath.Join(projectRoot, devenvLockFile)) //nolint:gosec // fixed name under the project root
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", devenvLockFile, err)
	}
	var lock struct {
		Root  string `json:"root"`
		Nodes map[string]struct {
			Inputs map[string]any `json:"inputs"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal(data, &lock); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", devenvLockFile, err)
	}
	var declared struct {
		Inputs map[string]any `yaml:"inputs"`
	}
	if err := yaml.Unmarshal(yamlContent, &declared); err != nil {
		return nil, fmt.Errorf("parsing devenv.yaml: %w", err)
	}
	locked := lock.Nodes[lock.Root].Inputs
	var unlocked []string
	for _, name := range slices.Sorted(maps.Keys(declared.Inputs)) {
		if _, ok := locked[name]; !ok {
			unlocked = append(unlocked, name)
		}
	}
	return unlocked, nil
}
