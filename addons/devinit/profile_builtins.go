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
	Skills:          []string{"security-review"},
	Hooks:           []string{"safety-block", "pre-commit"},
}

// DefaultProjectProfileRegistry returns a ProjectProfileRegistry pre-loaded
// with the four built-in project-type profiles.
func DefaultProjectProfileRegistry() *ProjectProfileRegistry {
	r := NewProjectProfileRegistry()
	// Errors are impossible here because names are unique constants.
	_ = r.Register("go-web", GoWeb)
	_ = r.Register("ts-fullstack", TSFullstack)
	_ = r.Register("python-data", PythonData)
	_ = r.Register("rust-cli", RustCLI)
	return r
}
