package claudecode

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"path"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// SkillDefinition describes a skill and the tool operations it requires.
type SkillDefinition struct {
	Name         string
	AllowedTools []string
}

// ExpectedConflicts returns the map of known expected conflicts.
// These are conflicts that exist by design — where deny rules intentionally
// block operations that a skill would otherwise need.
// Key format: "skillName:denyRule"
//
// Note: Package install operations (npm install, pip install, etc.) are no
// longer in deny — they are in the ask list, gated by the PreToolUse
// package-guard hook. This means upgrade-dep no longer conflicts with deny rules.
func ExpectedConflicts() map[string]string {
	return map[string]string{}
}

// skillTemplateDirs are the embedded template directories holding the skills
// and subagents qsdev deploys.
var skillTemplateDirs = []string{"templates/skills", "templates/agents"}

// frontmatterRenderers render the templates whose frontmatter is itself
// templated, keyed by skill name. Each renders the widest tool set the skill
// can be deployed with, so every operation it may request is checked.
var frontmatterRenderers = map[string]func() ([]byte, error){
	"lookup-docs": renderLookupDocsAllSources,
}

// BuiltinSkillDefinitions returns the tool operations requested by every skill
// and subagent template qsdev deploys. They are parsed from the templates'
// frontmatter (`allowed-tools` for skills, `tools` for subagents) at runtime,
// so `qsdev check` validates exactly what the deployed files request.
func BuiltinSkillDefinitions() []SkillDefinition {
	defs, err := loadBuiltinSkillDefinitions()
	if err != nil {
		// The templates are embedded and covered by tests; return what parsed
		// rather than skipping the conflict check entirely.
		slog.Warn("parsing built-in skill frontmatter", "error", err)
	}
	return defs
}

// loadBuiltinSkillDefinitions walks the embedded skill and subagent templates
// and returns the tool operations each one declares, sorted by name.
// Templates without frontmatter or without a tool list declare no operations
// and are omitted.
func loadBuiltinSkillDefinitions() ([]SkillDefinition, error) {
	var defs []SkillDefinition
	var errs []error
	for _, dir := range skillTemplateDirs {
		err := fs.WalkDir(templateFS, dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !isMarkdownTemplate(p) {
				return nil
			}
			def, ok, err := skillDefinitionFromTemplate(p)
			if err != nil {
				errs = append(errs, err)
				return nil
			}
			if ok {
				defs = append(defs, def)
			}
			return nil
		})
		if err != nil {
			errs = append(errs, fmt.Errorf("walking %s: %w", dir, err))
		}
	}
	sort.Slice(defs, func(i, j int) bool { return defs[i].Name < defs[j].Name })
	return defs, errors.Join(errs...)
}

// isMarkdownTemplate reports whether p is a Markdown skill or agent template.
func isMarkdownTemplate(p string) bool {
	return strings.HasSuffix(p, ".md") || strings.HasSuffix(p, ".md.tmpl")
}

// templateSkillName derives a skill's name from its template path:
// skills/<name>/SKILL.md[.tmpl] or <dir>/<name>.md[.tmpl].
func templateSkillName(p string) string {
	base := strings.TrimSuffix(strings.TrimSuffix(path.Base(p), ".tmpl"), ".md")
	if base == "SKILL" {
		return path.Base(path.Dir(p))
	}
	return base
}

// skillFrontmatter holds the frontmatter keys that declare tool operations.
type skillFrontmatter struct {
	Name         string `yaml:"name"`
	AllowedTools any    `yaml:"allowed-tools"`
	Tools        any    `yaml:"tools"`
}

// skillDefinitionFromTemplate parses the tool operations declared by one
// template. ok is false when the template declares none.
func skillDefinitionFromTemplate(p string) (SkillDefinition, bool, error) {
	content, err := templateFS.ReadFile(p)
	if err != nil {
		return SkillDefinition{}, false, fmt.Errorf("reading %s: %w", p, err)
	}
	name := templateSkillName(p)

	fm, ok := frontmatterBlock(content)
	if !ok {
		return SkillDefinition{}, false, nil
	}
	if bytes.Contains(fm, []byte("{{")) {
		render, found := frontmatterRenderers[name]
		if !found {
			return SkillDefinition{}, false, fmt.Errorf("%s: templated frontmatter has no renderer", p)
		}
		rendered, err := render()
		if err != nil {
			return SkillDefinition{}, false, fmt.Errorf("rendering %s: %w", p, err)
		}
		if fm, ok = frontmatterBlock(rendered); !ok {
			return SkillDefinition{}, false, fmt.Errorf("%s: rendered template has no frontmatter", p)
		}
	}

	var meta skillFrontmatter
	if err := yaml.Unmarshal(fm, &meta); err != nil {
		return SkillDefinition{}, false, fmt.Errorf("parsing frontmatter of %s: %w", p, err)
	}
	if meta.Name != "" {
		name = meta.Name
	}

	var tools []string
	for _, field := range []any{meta.AllowedTools, meta.Tools} {
		parsed, err := toolListValue(field)
		if err != nil {
			return SkillDefinition{}, false, fmt.Errorf("%s: %w", p, err)
		}
		tools = append(tools, parsed...)
	}
	if len(tools) == 0 {
		return SkillDefinition{}, false, nil
	}
	return SkillDefinition{Name: name, AllowedTools: tools}, true, nil
}

// frontmatterBlock returns the YAML between a leading "---" line and the next
// "---" line, and whether content has such a block.
func frontmatterBlock(content []byte) ([]byte, bool) {
	content = bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n"))
	rest, found := bytes.CutPrefix(content, []byte("---\n"))
	if !found {
		return nil, false
	}
	block, _, found := bytes.Cut(rest, []byte("\n---"))
	if !found {
		return nil, false
	}
	return block, true
}

// toolListValue converts a frontmatter tool field (a string or a YAML list)
// into individual tool patterns.
func toolListValue(v any) ([]string, error) {
	switch val := v.(type) {
	case nil:
		return nil, nil
	case string:
		return splitToolList(val), nil
	case []any:
		var tools []string
		for _, item := range val {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("tool list entry %v is not a string", item)
			}
			tools = append(tools, splitToolList(s)...)
		}
		return tools, nil
	default:
		return nil, fmt.Errorf("tool list has unsupported type %T", v)
	}
}

// splitToolList splits a tool list on whitespace and commas outside
// parentheses, so "Bash(git log *) Read, Grep" yields
// ["Bash(git log *)", "Read", "Grep"].
func splitToolList(s string) []string {
	var tools []string
	var cur strings.Builder
	depth := 0
	flush := func() {
		if cur.Len() > 0 {
			tools = append(tools, cur.String())
			cur.Reset()
		}
	}
	for _, r := range s {
		switch {
		case r == '(':
			depth++
		case r == ')' && depth > 0:
			depth--
		case depth == 0 && (r == ',' || r == ' ' || r == '\t' || r == '\n'):
			flush()
			continue
		}
		cur.WriteRune(r)
	}
	flush()
	return tools
}

// renderLookupDocsAllSources renders the lookup-docs skill with every
// documentation source available, its widest possible tool set.
func renderLookupDocsAllSources() ([]byte, error) {
	var answers types.WizardAnswers
	for _, src := range docSourceDefs {
		answers.MCPServers = append(answers.MCPServers, src.ServerName)
	}
	f, err := generateLookupDocsSkill(answers)
	if err != nil {
		return nil, err
	}
	return f.Content, nil
}
