package devinit

import "github.com/Quantum-Serendipity/qsdev/internal/catalog"

// GoWeb is a project-type profile for Go web services.
var GoWeb = catalogProfile("go-web")

// TSFullstack is a project-type profile for TypeScript full-stack applications.
var TSFullstack = catalogProfile("ts-fullstack")

// PythonData is a project-type profile for Python data science projects.
var PythonData = catalogProfile("python-data")

// RustCLI is a project-type profile for Rust command-line tools.
var RustCLI = catalogProfile("rust-cli")

// JavaWeb is a project-type profile for Java web services.
var JavaWeb = catalogProfile("java-web")

// PythonWeb is a project-type profile for Python web services.
var PythonWeb = catalogProfile("python-web")

// TSBackend is a project-type profile for TypeScript backend services.
var TSBackend = catalogProfile("ts-backend")

// ElixirWeb is a project-type profile for Elixir web services.
var ElixirWeb = catalogProfile("elixir-web")

// RustWeb is a project-type profile for Rust web services.
var RustWeb = catalogProfile("rust-web")

// DotnetWeb is a project-type profile for .NET web services.
var DotnetWeb = catalogProfile("dotnet-web")

// projectProfileOrder defines the canonical registration order for project profiles.
var projectProfileOrder = []string{
	"go-web", "ts-fullstack", "ts-backend", "python-data", "python-web",
	"rust-cli", "rust-web", "java-web", "elixir-web", "dotnet-web",
}

// DefaultProjectProfileRegistry returns a ProjectProfileRegistry pre-loaded
// with the built-in project-type profiles.
func DefaultProjectProfileRegistry() *ProjectProfileRegistry {
	r := NewProjectProfileRegistry()
	for _, name := range projectProfileOrder {
		_ = r.Register(name, catalogProfile(name))
	}
	return r
}

func catalogProfile(name string) Profile {
	cat := catalog.MustDefault()
	def, ok := cat.ProjectProfile(name)
	if !ok {
		return Profile{}
	}

	langs := make([]LanguageSpec, len(def.Languages))
	for i, l := range def.Languages {
		langs[i] = LanguageSpec{
			Name:           l.Name,
			Version:        l.Version,
			PackageManager: l.PackageManager,
		}
	}

	return Profile{
		Description:     def.Description,
		Languages:       langs,
		Services:        def.Services,
		Direnv:          def.Direnv,
		ClaudeCode:      def.ClaudeCode,
		PermissionLevel: def.PermissionLevel,
		Tier:            def.Tier,
		Skills:          def.Skills,
		Hooks:           def.Hooks,
	}
}
