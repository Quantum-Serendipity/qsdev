// Package rules embeds the qsdev OpenGrep "core" taint-rule library so it can be
// delivered into user projects when the opengrep tool is enabled.
//
// The rule sources live at repo root under rules/core/ (the same tree qsdev's
// own tests validate). Because //go:embed can only see files at or below the
// embedding package's directory, this package exists at module root purely to
// expose that tree to consumers under internal/ (e.g. internal/sectools), which
// cannot embed a sibling directory.
package rules

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"strings"
)

// coreFS holds the embedded rules/core tree, including its testdata fixtures.
// Consumers should use CoreRuleFiles to obtain only the deliverable rule files.
//
//go:embed core
var coreFS embed.FS

// FS returns the raw embedded filesystem rooted at this package's directory.
// The rule tree is available under the "core" prefix.
func FS() embed.FS { return coreFS }

// RuleFile is a single deliverable OpenGrep rule file: its path relative to the
// core/ directory (POSIX-slash separated) together with its raw YAML content.
type RuleFile struct {
	// RelPath is relative to core/, e.g. "injection/go-sql-injection.yaml".
	RelPath string
	Content []byte
}

// CoreRuleFiles walks the embedded core/ tree and returns every OpenGrep rule
// file (.yaml/.yml). It excludes the testdata/ fixtures (intentionally
// vulnerable sample code that must never be shipped into user projects) and the
// manifest.yaml index (a documentation file, not a loadable rule), matching the
// definition of a "rule file" used by qsdev's own rule validation tests.
func CoreRuleFiles() ([]RuleFile, error) {
	var out []RuleFile
	err := fs.WalkDir(coreFS, "core", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "testdata" {
				return fs.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(path.Ext(p))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}
		if d.Name() == "manifest.yaml" {
			return nil
		}
		data, readErr := coreFS.ReadFile(p)
		if readErr != nil {
			return fmt.Errorf("reading embedded rule %s: %w", p, readErr)
		}
		out = append(out, RuleFile{
			RelPath: strings.TrimPrefix(p, "core/"),
			Content: data,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking embedded opengrep rules: %w", err)
	}
	return out, nil
}
