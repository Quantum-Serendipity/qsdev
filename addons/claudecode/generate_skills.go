package claudecode

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/sliceutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// SkillManifest holds the list of available skills parsed from manifest.yaml.
type SkillManifest struct {
	Skills []SkillEntry `yaml:"skills"`
}

// SkillEntry describes a single skill in the manifest.
type SkillEntry struct {
	Name                string   `yaml:"name"`
	Description         string   `yaml:"description"`
	Tags                []string `yaml:"tags"`
	ApplicableLanguages []string `yaml:"applicable_languages"`
}

// loadManifest reads and parses the skill manifest from the embedded filesystem.
func loadManifest() (*SkillManifest, error) {
	return loadYAMLManifest[SkillManifest]("templates/skills/manifest.yaml")
}

// legacySkillAliases maps renamed skill names to their current manifest name so
// a pre-rename .qsdev.yaml (which persists the old name) still resolves instead
// of hard-erroring on regeneration. security-review was renamed to
// security-review-owasp to avoid colliding with Claude Code's built-in
// /security-review command.
var legacySkillAliases = map[string]string{
	"security-review": "security-review-owasp",
}

// canonicalSkillName resolves a possibly-legacy persisted skill name to its
// current manifest name.
func canonicalSkillName(name string) string {
	if current, ok := legacySkillAliases[name]; ok {
		return current
	}
	return name
}

// legacyFlatSkillPath maps a current-layout skill path
// (.claude/skills/<name>/SKILL.md) to its pre-migration flat path
// (.claude/skills/<name>.md), reporting false for any other path. Used to remove
// the stale flat file left behind by the layout change.
func legacyFlatSkillPath(p string) (string, bool) {
	const prefix = ".claude/skills/"
	const suffix = "/SKILL.md"
	s := filepath.ToSlash(p)
	if !strings.HasPrefix(s, prefix) || !strings.HasSuffix(s, suffix) {
		return "", false
	}
	name := s[len(prefix) : len(s)-len(suffix)]
	if name == "" || strings.Contains(name, "/") {
		return "", false
	}
	return prefix + name + ".md", true
}

// deploySkills reads the selected skill files from the embedded filesystem and
// returns GeneratedFile entries for each. It validates that every requested
// skill name exists in the manifest, tolerating legacy (renamed) names so
// existing project configs keep regenerating.
func deploySkills(answers types.WizardAnswers) ([]types.GeneratedFile, error) {
	if len(answers.Skills) == 0 {
		return nil, nil
	}

	manifest, err := loadManifest()
	if err != nil {
		return nil, err
	}

	// Index by name for validation and front-matter synthesis.
	entryByName := make(map[string]SkillEntry, len(manifest.Skills))
	for _, s := range manifest.Skills {
		entryByName[s.Name] = s
	}

	var files []types.GeneratedFile
	seen := make(map[string]bool, len(answers.Skills))
	for _, requested := range answers.Skills {
		name := canonicalSkillName(requested)
		if seen[name] {
			continue // legacy and current name both listed, or a duplicate
		}
		seen[name] = true

		entry, ok := entryByName[name]
		if !ok {
			return nil, fmt.Errorf("unknown skill %q: not found in manifest", requested)
		}

		body, err := templateFS.ReadFile("templates/skills/" + name + ".md")
		if err != nil {
			return nil, fmt.Errorf("reading skill file %q: %w", name, err)
		}

		// Claude Code loads skills only from <name>/SKILL.md with YAML
		// front-matter. Synthesize name+description from the manifest (the
		// single source of truth) and write the directory layout.
		content, err := prependSkillFrontMatter(entry.Name, entry.Description, body)
		if err != nil {
			return nil, err
		}

		files = append(files, types.GeneratedFile{
			Path:     ".claude/skills/" + name + "/SKILL.md",
			Content:  content,
			Mode:     fileutil.ModeReadWrite,
			Strategy: types.LibraryManaged,
			Owner:    name,
		})
	}

	return files, nil
}

// AvailableSkillNames returns the names of all skills from the embedded manifest.
func AvailableSkillNames() []string {
	manifest, err := loadManifest()
	if err != nil {
		return nil
	}
	names := make([]string, len(manifest.Skills))
	for i, s := range manifest.Skills {
		names[i] = s.Name
	}
	return names
}

// languageToRules maps ecosystem canonical names to their convention rule files.
var languageToRules = map[string][]string{
	"go":         {"go-conventions.md"},
	"javascript": {"typescript-conventions.md"},
	"python":     {"python-conventions.md"},
	"rust":       {"rust-conventions.md"},
	"java":       {"java-conventions.md"},
	"dotnet":     {"dotnet-conventions.md"},
	"container":  {"docker-conventions.md"},
	"terraform":  {"terraform-conventions.md"},
}

// languageToLSPRule maps ecosystem canonical names to their per-language LSP
// guidance rule. Each of these rules carries `paths:` frontmatter so it loads
// lazily only when Claude reads a matching file. Languages without an entry
// here rely on the universal lsp-navigation.md rule alone. The path globs in
// each rule must match the corresponding pkg/lsp RuleGlobs.
var languageToLSPRule = map[string]string{
	"go":         "go-lsp.md",
	"javascript": "typescript-lsp.md",
	"rust":       "rust-lsp.md",
	"python":     "python-lsp.md",
	"java":       "java-lsp.md",
}

// lspRulesOwner tags the LSP rule files for teardown tracking. Convention rules
// set no Owner; only the LSP rules are owned so they can be cleaned up
// independently.
const lspRulesOwner = "lsp-rules"

// alwaysOnLSPRules are deployed for every project regardless of detected
// languages: the universal navigation rule, and nixd (always available — every
// qsdev project has .nix files).
var alwaysOnLSPRules = []string{"lsp-navigation.md", "nix-lsp.md"}

// deployRules selects convention rule files based on the project's languages
// and always includes the security rules. It returns GeneratedFile entries
// for each selected rule.
func deployRules(answers types.WizardAnswers) ([]types.GeneratedFile, error) {
	var ruleNames []string

	for _, lang := range answers.Languages {
		if rules, ok := languageToRules[lang.Name]; ok {
			ruleNames = append(ruleNames, rules...)
		}
	}

	// Always include security rules.
	ruleNames = append(ruleNames, "security-rules.md")
	ruleNames = sliceutil.Dedup(ruleNames)

	var files []types.GeneratedFile
	for _, name := range ruleNames {
		content, err := templateFS.ReadFile("templates/rules/" + name)
		if err != nil {
			return nil, fmt.Errorf("reading rule file %q: %w", name, err)
		}

		files = append(files, types.GeneratedFile{
			Path:     ".claude/rules/" + name,
			Content:  content,
			Mode:     fileutil.ModeReadWrite,
			Strategy: types.LibraryManaged,
		})
	}

	// Deploy LSP navigation rules. The universal rule and nixd are always
	// present (LSP is always available); per-language rules deploy for each
	// detected ecosystem that has one.
	lspFiles, err := deployLSPRules(answers)
	if err != nil {
		return nil, err
	}
	files = append(files, lspFiles...)

	return files, nil
}

// deployLSPRules returns the LSP guidance rule files. lsp-navigation.md and
// nix-lsp.md are always deployed; a per-language LSP rule is added for each
// detected ecosystem that maps to one. All returned files carry Owner
// lspRulesOwner so they can be torn down independently of convention rules.
func deployLSPRules(answers types.WizardAnswers) ([]types.GeneratedFile, error) {
	ruleNames := append([]string(nil), alwaysOnLSPRules...)

	for _, lang := range answers.Languages {
		if rule, ok := languageToLSPRule[lang.Name]; ok {
			ruleNames = append(ruleNames, rule)
		}
	}

	ruleNames = sliceutil.Dedup(ruleNames)

	var files []types.GeneratedFile
	for _, name := range ruleNames {
		content, err := templateFS.ReadFile("templates/rules/" + name)
		if err != nil {
			return nil, fmt.Errorf("reading LSP rule file %q: %w", name, err)
		}

		files = append(files, types.GeneratedFile{
			Path:     ".claude/rules/" + name,
			Content:  content,
			Mode:     fileutil.ModeReadWrite,
			Strategy: types.LibraryManaged,
			Owner:    lspRulesOwner,
		})
	}

	return files, nil
}
