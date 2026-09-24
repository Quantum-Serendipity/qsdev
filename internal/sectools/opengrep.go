package sectools

import (
	"fmt"
	"path"

	"github.com/Quantum-Serendipity/qsdev/nix"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
	"github.com/Quantum-Serendipity/qsdev/rules"
)

// opengrepNixPath is the project-relative path of the OpenGrep package
// derivation. The opengrep catalog entry's nix_expr imports its directory
// (`pkgs.callPackage ./.opengrep/nix {}`), so devenv.nix only references a file
// qsdev itself writes into the project. It lives under the opengrep-owned
// .opengrep/ directory rather than a top-level nix/ so it cannot collide with a
// project's own Nix sources.
const opengrepNixPath = ".opengrep/nix/default.nix"

// GenerateOpengrepFiles produces the full set of files delivered when the
// opengrep tool is enabled: the pinned OpenGrep package derivation at
// opengrepNixPath (which the devenv.nix package list imports) and the embedded
// core taint-rule library, written under rules.ProjectCoreDir, the directory
// the security-scan task passes to `opengrep scan --config`.
//
// No config file is generated: OpenGrep (like Semgrep, whose parser it shares)
// has no project config file, so scan settings live on the task's command line.
func GenerateOpengrepFiles(_ types.WizardAnswers) ([]types.GeneratedFile, error) {
	files := []types.GeneratedFile{{
		Path:     opengrepNixPath,
		Content:  nix.OpengrepDerivation(),
		Mode:     fileutil.ModeReadWrite,
		Strategy: types.Overwrite,
		Owner:    "opengrep",
	}}

	ruleFiles, err := rules.CoreRuleFiles()
	if err != nil {
		return nil, fmt.Errorf("loading embedded opengrep rules: %w", err)
	}
	for _, rf := range ruleFiles {
		files = append(files, types.GeneratedFile{
			Path:     path.Join(rules.ProjectCoreDir, rf.RelPath),
			Content:  rf.Content,
			Mode:     fileutil.ModeReadWrite,
			Strategy: types.Overwrite,
			Owner:    "opengrep",
		})
	}
	return files, nil
}
