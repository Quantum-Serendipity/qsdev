package devinit

// GoWeb is a project-type profile for Go web services.
// Includes PostgreSQL and Redis, direnv, standard Claude Code permissions,
// deploy + security-review skills, and safety-block + pre-commit hooks.
var GoWeb = Profile{
	Description: "Go web service: Go 1.24, PostgreSQL, Redis, direnv, Claude Code (standard), deploy + security-review skills",
	Languages: []LanguageSpec{
		{Name: "go", Version: "1.24"},
	},
	Services:        []string{"postgres", "redis"},
	Direnv:          true,
	ClaudeCode:      true,
	PermissionLevel: "standard",
	Tier:            "full",
	Skills:          []string{"deploy", "security-review"},
	Hooks:           []string{"safety-block", "pre-commit"},
}

// TSFullstack is a project-type profile for TypeScript full-stack applications.
// Includes JavaScript with pnpm, PostgreSQL and Redis, direnv, standard Claude Code
// permissions, deploy + security-review skills, and auto-format + safety-block + pre-commit hooks.
var TSFullstack = Profile{
	Description: "TypeScript full-stack: JavaScript (pnpm), PostgreSQL, Redis, direnv, Claude Code (standard), deploy + security-review skills",
	Languages: []LanguageSpec{
		{Name: "javascript", PackageManager: "pnpm"},
	},
	Services:        []string{"postgres", "redis"},
	Direnv:          true,
	ClaudeCode:      true,
	PermissionLevel: "standard",
	Tier:            "full",
	Skills:          []string{"deploy", "security-review"},
	Hooks:           []string{"auto-format", "safety-block", "pre-commit"},
}

// PythonData is a project-type profile for Python data science projects.
// Includes Python 3.12 with uv, no services, direnv, minimal Claude Code
// permissions, security-review skill, and safety-block hook.
var PythonData = Profile{
	Description: "Python data science: Python 3.12 (uv), no services, direnv, Claude Code (minimal), security-review skill",
	Languages: []LanguageSpec{
		{Name: "python", Version: "3.12", PackageManager: "uv"},
	},
	Services:        nil,
	Direnv:          true,
	ClaudeCode:      true,
	PermissionLevel: "minimal",
	Tier:            "full",
	Skills:          []string{"security-review"},
	Hooks:           []string{"safety-block", "pre-commit"},
}

// RustCLI is a project-type profile for Rust command-line tools.
// Includes Rust, no services, direnv, minimal Claude Code permissions,
// security-review skill, and safety-block + pre-commit hooks.
var RustCLI = Profile{
	Description: "Rust CLI tool: Rust, no services, direnv, Claude Code (minimal), security-review skill",
	Languages: []LanguageSpec{
		{Name: "rust"},
	},
	Services:        nil,
	Direnv:          true,
	ClaudeCode:      true,
	PermissionLevel: "minimal",
	Tier:            "full",
	Skills:          []string{"security-review"},
	Hooks:           []string{"safety-block", "pre-commit"},
}

// JavaWeb is a project-type profile for Java web services.
// Includes Java 21 with Gradle, PostgreSQL, Redis, direnv, standard Claude Code
// permissions, deploy + security-review skills, and safety-block + pre-commit hooks.
var JavaWeb = Profile{
	Description: "Java web service: Java 21 (Gradle), PostgreSQL, Redis, direnv, Claude Code (standard)",
	Languages: []LanguageSpec{
		{Name: "java", Version: "21", PackageManager: "gradle"},
	},
	Services:        []string{"postgres", "redis"},
	Direnv:          true,
	ClaudeCode:      true,
	PermissionLevel: "standard",
	Tier:            "full",
	Skills:          []string{"deploy", "security-review"},
	Hooks:           []string{"safety-block", "pre-commit"},
}

// PythonWeb is a project-type profile for Python web services.
// Includes Python 3.12 with uv, PostgreSQL, Redis, direnv, standard Claude Code
// permissions, deploy + security-review skills, and safety-block + pre-commit hooks.
var PythonWeb = Profile{
	Description: "Python web service: Python 3.12 (uv), PostgreSQL, Redis, direnv, Claude Code (standard)",
	Languages: []LanguageSpec{
		{Name: "python", Version: "3.12", PackageManager: "uv"},
	},
	Services:        []string{"postgres", "redis"},
	Direnv:          true,
	ClaudeCode:      true,
	PermissionLevel: "standard",
	Tier:            "full",
	Skills:          []string{"deploy", "security-review"},
	Hooks:           []string{"safety-block", "pre-commit"},
}

// TSBackend is a project-type profile for TypeScript backend services.
// Includes JavaScript with pnpm, PostgreSQL, Redis, direnv, standard Claude Code
// permissions, deploy + security-review skills, and safety-block + pre-commit hooks.
var TSBackend = Profile{
	Description: "TypeScript backend: JavaScript (pnpm), PostgreSQL, Redis, direnv, Claude Code (standard)",
	Languages: []LanguageSpec{
		{Name: "javascript", PackageManager: "pnpm"},
	},
	Services:        []string{"postgres", "redis"},
	Direnv:          true,
	ClaudeCode:      true,
	PermissionLevel: "standard",
	Tier:            "full",
	Skills:          []string{"deploy", "security-review"},
	Hooks:           []string{"safety-block", "pre-commit"},
}

// ElixirWeb is a project-type profile for Elixir web services.
// Includes Elixir, PostgreSQL, Redis, direnv, standard Claude Code permissions,
// deploy + security-review skills, and safety-block + pre-commit hooks.
var ElixirWeb = Profile{
	Description: "Elixir web service: Elixir, PostgreSQL, Redis, direnv, Claude Code (standard)",
	Languages: []LanguageSpec{
		{Name: "elixir"},
	},
	Services:        []string{"postgres", "redis"},
	Direnv:          true,
	ClaudeCode:      true,
	PermissionLevel: "standard",
	Tier:            "full",
	Skills:          []string{"deploy", "security-review"},
	Hooks:           []string{"safety-block", "pre-commit"},
}

// RustWeb is a project-type profile for Rust web services.
// Includes Rust, PostgreSQL, Redis, direnv, standard Claude Code permissions,
// deploy + security-review skills, and safety-block + pre-commit hooks.
var RustWeb = Profile{
	Description: "Rust web service: Rust, PostgreSQL, Redis, direnv, Claude Code (standard)",
	Languages: []LanguageSpec{
		{Name: "rust"},
	},
	Services:        []string{"postgres", "redis"},
	Direnv:          true,
	ClaudeCode:      true,
	PermissionLevel: "standard",
	Tier:            "full",
	Skills:          []string{"deploy", "security-review"},
	Hooks:           []string{"safety-block", "pre-commit"},
}

// DotnetWeb is a project-type profile for .NET web services.
// Includes .NET, PostgreSQL, Redis, direnv, standard Claude Code permissions,
// deploy + security-review skills, and safety-block + pre-commit hooks.
var DotnetWeb = Profile{
	Description: ".NET web service: .NET, PostgreSQL, Redis, direnv, Claude Code (standard)",
	Languages: []LanguageSpec{
		{Name: "dotnet"},
	},
	Services:        []string{"postgres", "redis"},
	Direnv:          true,
	ClaudeCode:      true,
	PermissionLevel: "standard",
	Tier:            "full",
	Skills:          []string{"deploy", "security-review"},
	Hooks:           []string{"safety-block", "pre-commit"},
}

// DefaultProjectProfileRegistry returns a ProjectProfileRegistry pre-loaded
// with the built-in project-type profiles.
func DefaultProjectProfileRegistry() *ProjectProfileRegistry {
	r := NewProjectProfileRegistry()
	_ = r.Register("go-web", GoWeb)
	_ = r.Register("ts-fullstack", TSFullstack)
	_ = r.Register("ts-backend", TSBackend)
	_ = r.Register("python-data", PythonData)
	_ = r.Register("python-web", PythonWeb)
	_ = r.Register("rust-cli", RustCLI)
	_ = r.Register("rust-web", RustWeb)
	_ = r.Register("java-web", JavaWeb)
	_ = r.Register("elixir-web", ElixirWeb)
	_ = r.Register("dotnet-web", DotnetWeb)
	return r
}
