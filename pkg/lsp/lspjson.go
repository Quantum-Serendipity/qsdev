package lsp

import "sort"

// ExtensionToLanguage maps each handled file extension to this server's
// LanguageID, suitable for the .lsp.json extensionToLanguage object. The result
// is a fresh map; modifying it does not affect the config.
func (c *LSPServerConfig) ExtensionToLanguage() map[string]string {
	m := make(map[string]string, len(c.Extensions))
	for _, ext := range c.Extensions {
		m[ext] = c.LanguageID
	}
	return m
}

// RuleGlobs returns the deterministic, deduplicated set of globs for this
// server's rule-file frontmatter: one "**/*<ext>" per extension followed by the
// RulePatterns verbatim. Extension-derived globs are sorted; RulePatterns
// preserve their declared order. The result is a fresh slice.
func (c *LSPServerConfig) RuleGlobs() []string {
	extGlobs := make([]string, 0, len(c.Extensions))
	for _, ext := range c.Extensions {
		extGlobs = append(extGlobs, "**/*"+ext)
	}
	sort.Strings(extGlobs)

	globs := make([]string, 0, len(extGlobs)+len(c.RulePatterns))
	seen := make(map[string]struct{}, cap(globs))
	add := func(g string) {
		if _, ok := seen[g]; ok {
			return
		}
		seen[g] = struct{}{}
		globs = append(globs, g)
	}
	for _, g := range extGlobs {
		add(g)
	}
	for _, g := range c.RulePatterns {
		add(g)
	}
	return globs
}
