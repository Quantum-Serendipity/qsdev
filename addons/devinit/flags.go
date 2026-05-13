package devinit

import (
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

// FlagSet tracks which CLI flags were explicitly set by the user.
type FlagSet struct {
	changed map[string]bool
}

// NewFlagSet inspects the command's flags after parsing and records which
// ones were explicitly provided on the command line.
func NewFlagSet(cmd *cobra.Command) *FlagSet {
	changed := make(map[string]bool)
	cmd.Flags().Visit(func(f *pflag.Flag) {
		changed[f.Name] = true
	})
	return &FlagSet{changed: changed}
}

// IsSet reports whether a flag was explicitly set by the user.
func (f *FlagSet) IsSet(name string) bool {
	return f.changed[name]
}

// InitOptions holds the raw flag values before conversion to WizardAnswers.
type InitOptions struct {
	// Core
	Langs       []string
	Services    []string
	Yes         bool
	Force       bool
	DryRun      bool
	Update      bool
	DevenvOnly  bool
	ClaudeOnly  bool
	ProfileName string

	// Language-specific
	GoVersion     string
	NodeVersion   string
	NodePkgMgr    string
	PythonVersion string
	PythonPkgMgr  string
	RustChannel   string
	JavaVersion   string
	JavaBuildTool string

	// Dev environment
	Direnv            bool
	GitHooks          []string
	Packages          []string
	Env               []string
	NixHardeningGuide bool
	InfraProfile      string

	// Claude Code
	ClaudeCode        bool
	ClaudePermissions string
	ClaudeSkills      []string
	ClaudeHooks       []string
	MCPServers        []string
	ListProfiles      bool
}

// RegisterInitFlags registers all flags for the gdev init command.
func RegisterInitFlags(cmd *cobra.Command, opts *InitOptions) {
	// Core flags.
	cmd.Flags().StringSliceVar(&opts.Langs, "lang", nil, "Languages to configure (e.g. go,javascript,python)")
	cmd.Flags().StringSliceVar(&opts.Services, "service", nil, "Services to configure (e.g. postgres,redis)")
	cmd.Flags().BoolVarP(&opts.Yes, "yes", "y", false, "Accept all defaults, skip confirmation prompts")
	cmd.Flags().BoolVar(&opts.Force, "force", false, "Overwrite existing configuration files")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Preview changes without writing files")
	cmd.Flags().BoolVar(&opts.Update, "update", false, "Regenerate files from saved config, preserving user modifications")
	cmd.Flags().BoolVar(&opts.DevenvOnly, "devenv-only", false, "Only generate devenv configuration (skip Claude Code)")
	cmd.Flags().BoolVar(&opts.ClaudeOnly, "claude-only", false, "Only generate Claude Code configuration (skip devenv)")
	cmd.Flags().StringVar(&opts.ProfileName, "profile", "", "Project-type profile name (e.g. go-web, ts-fullstack)")

	// Language-specific flags.
	cmd.Flags().StringVar(&opts.GoVersion, "go-version", "", "Go version (e.g. 1.24)")
	cmd.Flags().StringVar(&opts.NodeVersion, "node-version", "", "Node.js version (e.g. 22)")
	cmd.Flags().StringVar(&opts.NodePkgMgr, "node-pkg-mgr", "", "Node package manager (npm, pnpm, yarn, bun)")
	cmd.Flags().StringVar(&opts.PythonVersion, "python-version", "", "Python version (e.g. 3.12)")
	cmd.Flags().StringVar(&opts.PythonPkgMgr, "python-pkg-mgr", "", "Python package manager (pip, uv, poetry)")
	cmd.Flags().StringVar(&opts.RustChannel, "rust-channel", "", "Rust channel (stable, beta, nightly)")
	cmd.Flags().StringVar(&opts.JavaVersion, "java-version", "", "Java version (e.g. 21)")
	cmd.Flags().StringVar(&opts.JavaBuildTool, "java-build-tool", "", "Java build tool (maven, gradle)")

	// Dev environment flags.
	cmd.Flags().BoolVar(&opts.Direnv, "direnv", true, "Enable direnv integration")
	cmd.Flags().StringSliceVar(&opts.GitHooks, "git-hooks", nil, "Git hooks to configure (e.g. pre-commit,pre-push)")
	cmd.Flags().StringSliceVar(&opts.Packages, "packages", nil, "Extra Nix packages to include (e.g. jq,ripgrep)")
	cmd.Flags().StringSliceVar(&opts.Env, "env", nil, "Environment variables as KEY=VALUE pairs")
	cmd.Flags().BoolVar(&opts.NixHardeningGuide, "nix-hardening-guide", false, "Generate Nix security hardening guide")
	cmd.Flags().StringVar(&opts.InfraProfile, "infra-profile", "", "Infrastructure profile name (e.g. consulting-default)")

	// Claude Code flags.
	cmd.Flags().BoolVar(&opts.ClaudeCode, "claude-code", true, "Enable Claude Code configuration")
	cmd.Flags().StringVar(&opts.ClaudePermissions, "claude-permissions", "standard", "Permission preset (minimal, standard, permissive, custom)")
	cmd.Flags().StringSliceVar(&opts.ClaudeSkills, "claude-skills", nil, "Skills to install (e.g. deploy,review-pr)")
	cmd.Flags().StringSliceVar(&opts.ClaudeHooks, "claude-hooks", nil, "Hook presets to enable (e.g. safety-block,auto-format)")
	cmd.Flags().StringSliceVar(&opts.MCPServers, "mcp", nil, "MCP servers to configure (e.g. github,filesystem)")
	cmd.Flags().BoolVar(&opts.ListProfiles, "list-profiles", false, "List available project-type profiles and exit")

	// Mark mutually exclusive flags.
	cmd.MarkFlagsMutuallyExclusive("devenv-only", "claude-only")
	cmd.MarkFlagsMutuallyExclusive("update", "lang")
	cmd.MarkFlagsMutuallyExclusive("update", "service")
	cmd.MarkFlagsMutuallyExclusive("update", "profile")
}

// AnswersFromFlags converts flag values into WizardAnswers.
// Language-specific version flags implicitly add their language if it is
// not already present in the --lang list.
func AnswersFromFlags(opts InitOptions, projectRoot string) types.WizardAnswers {
	answers := types.WizardAnswers{
		ProjectRoot:       projectRoot,
		ProjectName:       filepath.Base(projectRoot),
		Direnv:            opts.Direnv,
		NixHardeningGuide: opts.NixHardeningGuide,
		ProfileName:       opts.InfraProfile,
		ClaudeCode:        opts.ClaudeCode,
		PermissionLevel:   opts.ClaudePermissions,
		Skills:            opts.ClaudeSkills,
		MCPServers:        opts.MCPServers,
		GitHooks:          opts.GitHooks,
		ExtraPackages:     opts.Packages,
		Confirmed:         opts.Yes,
	}

	// Build language set from explicit --lang flags.
	// We use an index map instead of pointer map because slice append may
	// reallocate the backing array and invalidate pointers.
	langIdx := make(map[string]int) // name -> index in answers.Languages
	for _, name := range opts.Langs {
		langIdx[name] = len(answers.Languages)
		answers.Languages = append(answers.Languages, types.LanguageChoice{Name: name})
	}

	// applyLang either updates an existing language entry or appends a new one.
	applyLang := func(name, version, pkgMgr string) {
		if version == "" && pkgMgr == "" {
			return
		}
		if idx, ok := langIdx[name]; ok {
			if version != "" {
				answers.Languages[idx].Version = version
			}
			if pkgMgr != "" {
				answers.Languages[idx].PackageManager = pkgMgr
			}
		} else {
			langIdx[name] = len(answers.Languages)
			answers.Languages = append(answers.Languages, types.LanguageChoice{
				Name: name, Version: version, PackageManager: pkgMgr,
			})
		}
	}

	// Language-specific version flags implicitly add the language.
	applyLang("go", opts.GoVersion, "")
	applyLang("javascript", opts.NodeVersion, opts.NodePkgMgr)
	applyLang("python", opts.PythonVersion, opts.PythonPkgMgr)
	applyLang("rust", opts.RustChannel, "")
	applyLang("java", opts.JavaVersion, opts.JavaBuildTool)

	// Only apply node-pkg-mgr alone when no version was given and javascript
	// was not yet added.
	if opts.NodePkgMgr != "" && opts.NodeVersion == "" {
		if _, ok := langIdx["javascript"]; !ok {
			langIdx["javascript"] = len(answers.Languages)
			answers.Languages = append(answers.Languages, types.LanguageChoice{
				Name: "javascript", PackageManager: opts.NodePkgMgr,
			})
		}
	}

	// Only apply python-pkg-mgr alone when no version was given and python
	// was not yet added.
	if opts.PythonPkgMgr != "" && opts.PythonVersion == "" {
		if _, ok := langIdx["python"]; !ok {
			langIdx["python"] = len(answers.Languages)
			answers.Languages = append(answers.Languages, types.LanguageChoice{
				Name: "python", PackageManager: opts.PythonPkgMgr,
			})
		}
	}

	// Convert services.
	for _, name := range opts.Services {
		answers.Services = append(answers.Services, types.ServiceChoice{Name: name})
	}

	// Parse --env KEY=VALUE pairs.
	if len(opts.Env) > 0 {
		answers.EnvVars = make(map[string]string, len(opts.Env))
		for _, kv := range opts.Env {
			if idx := strings.IndexByte(kv, '='); idx >= 0 {
				answers.EnvVars[kv[:idx]] = kv[idx+1:]
			}
		}
	}

	// Convert --claude-hooks to HookChoices.
	if len(opts.ClaudeHooks) > 0 {
		answers.Hooks = hooksFromStrings(opts.ClaudeHooks)
	}

	// --devenv-only disables Claude Code.
	if opts.DevenvOnly {
		answers.ClaudeCode = false
	}

	// --profile is the project-type profile name.
	if opts.ProfileName != "" {
		answers.ProjectTypeProfile = opts.ProfileName
	}

	return answers
}
