package posture

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"maps"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// preCommitConfigFile is the pre-commit framework's config file. devenv's
// git-hooks integration generates it outside qsdev's state tracking, and
// projects may also write it by hand.
const preCommitConfigFile = ".pre-commit-config.yaml"

// observedProtections returns the views that defense and conformance scoring
// assess: the enabled tools plus every tool the project's pre-commit config
// runs as a hook, and the tracked files plus the pre-commit config when it
// declares at least one hook. Both are copies; the inputs are left untouched,
// so config health and drift detection keep seeing only what qsdev tracks.
//
// Crediting is based on what the pre-commit config actually runs, not on the
// file merely existing: a config with no hooks protects nothing.
func observedProtections(projectPath string, enabledTools map[string]bool, genState types.GeneratedState) (map[string]bool, types.GeneratedState) {
	active := maps.Clone(enabledTools)
	if active == nil {
		active = make(map[string]bool)
	}
	present := genState
	present.Files = maps.Clone(genState.Files)
	if present.Files == nil {
		present.Files = make(map[string]types.FileState)
	}

	hookIDs, err := readPreCommitHookIDs(filepath.Join(projectPath, preCommitConfigFile))
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			slog.Warn("posture: pre-commit config unreadable; its hooks are not credited",
				"file", preCommitConfigFile, "error", err)
		}
		return active, present
	}
	for id := range hookIDs {
		active[id] = true
	}
	if _, tracked := present.Files[preCommitConfigFile]; !tracked && len(hookIDs) > 0 {
		present.Files[preCommitConfigFile] = types.FileState{}
	}
	return active, present
}

// readPreCommitHookIDs parses a pre-commit config and returns the set of hook
// ids it declares across all repos.
func readPreCommitHookIDs(path string) (map[string]bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading pre-commit config: %w", err)
	}
	var cfg struct {
		Repos []struct {
			Hooks []struct {
				ID string `yaml:"id"`
			} `yaml:"hooks"`
		} `yaml:"repos"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing pre-commit config %s: %w", path, err)
	}
	ids := make(map[string]bool)
	for _, repo := range cfg.Repos {
		for _, hook := range repo.Hooks {
			if hook.ID != "" {
				ids[hook.ID] = true
			}
		}
	}
	return ids, nil
}
