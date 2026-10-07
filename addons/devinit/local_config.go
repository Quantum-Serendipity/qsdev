package devinit

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// GenerateLocalConfigTemplate returns a commented YAML template for
// .qsdev.local.yaml. The template includes sections relevant to the
// answers (e.g., language version overrides, Claude Code settings).
// All keys are commented out so the file has no active configuration
// by default.
func GenerateLocalConfigTemplate(answers types.WizardAnswers, detected types.DetectedProject) []byte {
	var b strings.Builder

	br := branding.Get()
	fmt.Fprintf(&b, "# %s — Local developer overrides (gitignored)\n", br.LocalConfig)
	b.WriteString("# Uncomment and modify lines below to customize your local environment.\n")
	fmt.Fprintf(&b, "# These settings add to %s for your checkout only. They can add\n", br.ConfigFile)
	b.WriteString("# packages, languages, services, tools and MCP servers or tighten security,\n")
	b.WriteString("# never loosen it: weaker settings are ignored with a warning.\n")
	b.WriteString("#\n")

	// Extra packages section.
	b.WriteString("# extra_packages:\n")
	b.WriteString("#   - neovim\n")
	b.WriteString("#   - lazygit\n")
	b.WriteString("#   - ripgrep\n")

	// Language version overrides.
	if len(answers.Languages) > 0 {
		b.WriteString("#\n")
		b.WriteString("# languages:\n")
		for _, lang := range answers.Languages {
			exampleVersion := exampleVersionForLanguage(lang.Name, lang.Version)
			if exampleVersion != "" {
				fmt.Fprintf(&b, "#   - name: %s\n", lang.Name)
				fmt.Fprintf(&b, "#     version: \"%s\"\n", exampleVersion)
			}
		}
	}

	// Claude Code section.
	if answers.ClaudeCode {
		b.WriteString("#\n")
		b.WriteString("# claude_code:\n")
		b.WriteString("#   permission_level: minimal   # only a stricter level than the committed one\n")
	}

	// Tools section.
	b.WriteString("#\n")
	b.WriteString("# tools:\n")
	b.WriteString("#   enabled:\n")
	b.WriteString("#     - changelog\n")

	return []byte(b.String())
}

// exampleVersionForLanguage returns an example version string for the
// given language, based on the current version or a sensible default.
func exampleVersionForLanguage(name, currentVersion string) string {
	if currentVersion != "" {
		return currentVersion
	}
	switch name {
	case "go":
		return "1.24"
	case "javascript":
		return "22"
	case "python":
		return "3.12"
	case "rust":
		return "stable"
	case "java":
		return "21"
	default:
		return ""
	}
}

// writeLocalConfigTemplate creates the developer's local config from the
// template unless they already have one. The file is human-owned and
// gitignored, so it is written outside the generated-file pipeline and never
// recorded in state: editing it is not drift and teardown never removes it.
// The caller gitignores it first, so it is ignored before it exists. A
// symlink that resolves (for example into a dotfiles repository) is the
// developer's config and is kept; a dangling one is reported as an error. A
// created file, or on a dry run one that would be created, is announced on w.
func writeLocalConfigTemplate(projectRoot string, answers types.WizardAnswers, dryRun bool, w io.Writer) error {
	localCfg := branding.Get().LocalConfig
	if dryRun {
		exists, err := localConfigExists(projectRoot, localCfg)
		if err != nil || exists {
			return err
		}
		return announceLocalConfig(w, localCfg)
	}
	content := GenerateLocalConfigTemplate(answers, answers.Detected)
	err := fileutil.WriteNewFileInRoot(projectRoot, localCfg, content, fileutil.ModeReadWrite)
	switch {
	case err == nil:
		return announceLocalConfig(w, localCfg)
	case errors.Is(err, fs.ErrExist):
		return nil
	case errors.Is(err, fileutil.ErrSymlink):
		// Keep a link that resolves; localConfigExists reports a dangling one.
		_, err = localConfigExists(projectRoot, localCfg)
		return err
	default:
		return fmt.Errorf("writing %s template: %w", localCfg, err)
	}
}

// localConfigExists reports whether the developer already has a local config
// at rel under projectRoot, following a symlink as ParseLocalConfig does. A
// dangling symlink, or any error other than the path being absent, is
// returned so neither a dry run nor a real run treats it as fine.
func localConfigExists(projectRoot, rel string) (bool, error) {
	path := filepath.Join(projectRoot, rel)
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("checking %s: %w", rel, err)
	}
	if info.Mode()&fs.ModeSymlink == 0 {
		return true, nil
	}
	if _, err := os.Stat(path); err != nil {
		return false, fmt.Errorf("checking %s: symlink target unreadable: %w", rel, err)
	}
	return true, nil
}

// announceLocalConfig tells the developer about the untracked local config.
func announceLocalConfig(w io.Writer, rel string) error {
	_, err := fmt.Fprintf(w, "+ %s (local, untracked)\n", rel)
	return err
}
