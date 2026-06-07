package mcpregistry

// LanguageToDevDocsSlugs maps detected language/ecosystem names to the
// DevDocs slug identifiers used by devdocs.io for documentation sets.
var LanguageToDevDocsSlugs = map[string][]string{
	"go":         {"go"},
	"javascript": {"javascript", "node~22_lts"},
	"python":     {"python~3.12"},
	"rust":       {"rust"},
	"java":       {"openjdk~21"},
	"dotnet":     {"dotnet~8"},
	"terraform":  {"terraform"},
}
