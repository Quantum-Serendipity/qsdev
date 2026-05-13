package devinit

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"

	"fastcat.org/go/gdev-secure-devenv-bootstrap/addons/claudecode"
	"fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem"
	"fastcat.org/go/gdev-secure-devenv-bootstrap/pkg/types"
)

// formState holds intermediate variables that huh form fields bind to.
type formState struct {
	quickChoice string // "yes", "customize"

	selectedLanguages []string
	goVersion         string
	jsVersion         string
	pythonVersion     string

	selectedServices []string

	direnv       bool
	gitHooks     []string
	extraPackages string // comma-separated input

	claudeCode      bool
	permissionLevel string
	skills          []string
	autoFormat      bool
	safetyBlock     bool
	mcpServers      []string

	confirmed bool
}

// RunWizard runs the interactive huh form, collecting user choices.
// It pre-populates defaults from detection and any partial flag answers.
// Returns the fully populated WizardAnswers.
func RunWizard(projectRoot string, detected types.DetectedProject, partial types.WizardAnswers, flagSet *FlagSet) (types.WizardAnswers, error) {
	defaults := MapDetectionToDefaults(detected, projectRoot)
	projectName := defaults.ProjectName

	// Seed formState from detection defaults.
	fs := &formState{
		quickChoice:       "yes",
		selectedLanguages: PreSelectedLanguages(detected),
		direnv:            true,
		claudeCode:        true,
		permissionLevel:   "standard",
		safetyBlock:       true,
	}

	// Extract version defaults from detection.
	for _, lang := range defaults.Languages {
		switch lang.Name {
		case "go":
			fs.goVersion = lang.Version
		case "javascript":
			fs.jsVersion = lang.Version
		case "python":
			fs.pythonVersion = lang.Version
		}
	}

	// Override with any partial flag values.
	if len(partial.Languages) > 0 {
		names := make([]string, len(partial.Languages))
		for i, l := range partial.Languages {
			names[i] = l.Name
		}
		fs.selectedLanguages = names
	}
	if partial.Direnv {
		fs.direnv = partial.Direnv
	}
	if partial.PermissionLevel != "" {
		fs.permissionLevel = partial.PermissionLevel
	}
	if len(partial.Skills) > 0 {
		fs.skills = partial.Skills
	}
	if len(partial.MCPServers) > 0 {
		fs.mcpServers = partial.MCPServers
	}
	if len(partial.GitHooks) > 0 {
		fs.gitHooks = partial.GitHooks
	}
	if len(partial.ExtraPackages) > 0 {
		fs.extraPackages = strings.Join(partial.ExtraPackages, ", ")
	}
	fs.claudeCode = partial.ClaudeCode
	fs.autoFormat = partial.Hooks.AutoFormat
	fs.safetyBlock = partial.Hooks.SafetyBlock

	form := buildWizardForm(detected, fs, flagSet)
	if err := form.Run(); err != nil {
		if err == huh.ErrUserAborted {
			return types.WizardAnswers{Confirmed: false}, nil
		}
		return types.WizardAnswers{}, fmt.Errorf("wizard form: %w", err)
	}

	return mapFormToAnswers(fs, projectRoot, projectName, detected), nil
}

// buildWizardForm constructs the huh form with 6 groups.
func buildWizardForm(detected types.DetectedProject, fs *formState, flagSet *FlagSet) *huh.Form {
	defaults := MapDetectionToDefaults(detected, "")
	summary := QuickPathSummary(defaults)

	anyFlagExplicit := flagSetHasAny(flagSet)

	// --- Group 1: Quick Selection ---
	quickGroup := huh.NewGroup(
		huh.NewSelect[string]().
			Title("Quick setup detected your project").
			Description("We detected your project configuration. Would you like to use these defaults?").
			Options(
				huh.NewOption("Yes — "+summary, "yes"),
				huh.NewOption("No, let me customize", "customize"),
			).
			Value(&fs.quickChoice),
	).WithHideFunc(func() bool { return anyFlagExplicit })

	// --- Group 2: Languages & Runtimes ---
	langOptions := BuildLanguageOptions(detected)
	langOpts := make([]huh.Option[string], len(langOptions))
	for i, lo := range langOptions {
		langOpts[i] = huh.NewOption(lo.Label, lo.Value)
	}

	langGroup := huh.NewGroup(
		huh.NewMultiSelect[string]().
			Title("Languages & Runtimes").
			Description("Select the languages and platforms for this project.").
			Options(langOpts...).
			Value(&fs.selectedLanguages),
	).WithHideFunc(func() bool { return fs.quickChoice == "yes" })

	// Version input groups — each hidden unless the corresponding language is selected.
	goVersionGroup := huh.NewGroup(
		huh.NewInput().
			Title("Go version").
			Placeholder("e.g. 1.24").
			Value(&fs.goVersion),
	).WithHideFunc(func() bool {
		return fs.quickChoice == "yes" || !ecosystem.ContainsStr(fs.selectedLanguages, "go")
	})

	jsVersionGroup := huh.NewGroup(
		huh.NewInput().
			Title("Node.js version").
			Placeholder("e.g. 22").
			Value(&fs.jsVersion),
	).WithHideFunc(func() bool {
		return fs.quickChoice == "yes" || !ecosystem.ContainsStr(fs.selectedLanguages, "javascript")
	})

	pythonVersionGroup := huh.NewGroup(
		huh.NewInput().
			Title("Python version").
			Placeholder("e.g. 3.12").
			Value(&fs.pythonVersion),
	).WithHideFunc(func() bool {
		return fs.quickChoice == "yes" || !ecosystem.ContainsStr(fs.selectedLanguages, "python")
	})

	// --- Group 3: Services ---
	serviceOpts := []huh.Option[string]{
		huh.NewOption("PostgreSQL", "postgres"),
		huh.NewOption("Redis", "redis"),
		huh.NewOption("MySQL", "mysql"),
		huh.NewOption("MongoDB", "mongodb"),
		huh.NewOption("Elasticsearch", "elasticsearch"),
		huh.NewOption("RabbitMQ", "rabbitmq"),
	}

	servicesGroup := huh.NewGroup(
		huh.NewMultiSelect[string]().
			Title("Services").
			Description("Select development services to include.").
			Options(serviceOpts...).
			Value(&fs.selectedServices),
	).WithHideFunc(func() bool { return fs.quickChoice == "yes" })

	// --- Group 4: Dev Environment ---
	hookOpts := []huh.Option[string]{
		huh.NewOption("pre-commit", "pre-commit"),
		huh.NewOption("pre-push", "pre-push"),
		huh.NewOption("commit-msg", "commit-msg"),
	}

	devEnvGroup := huh.NewGroup(
		huh.NewConfirm().
			Title("Enable direnv integration?").
			Description("Automatically activates the dev environment when entering the project directory.").
			Affirmative("Yes").
			Negative("No").
			Value(&fs.direnv),
		huh.NewMultiSelect[string]().
			Title("Git hooks").
			Description("Select git hooks to configure.").
			Options(hookOpts...).
			Value(&fs.gitHooks),
		huh.NewInput().
			Title("Extra Nix packages").
			Description("Comma-separated list of additional packages to include.").
			Placeholder("e.g. jq, ripgrep, fd").
			Value(&fs.extraPackages),
	).WithHideFunc(func() bool { return fs.quickChoice == "yes" })

	// --- Group 5: Claude Code ---
	claudeEnableGroup := huh.NewGroup(
		huh.NewConfirm().
			Title("Enable Claude Code?").
			Description("Generates .claude/settings.json, CLAUDE.md, hooks, and skills.").
			Affirmative("Yes").
			Negative("No").
			Value(&fs.claudeCode),
	).WithHideFunc(func() bool { return fs.quickChoice == "yes" })

	permOpts := []huh.Option[string]{
		huh.NewOption("Minimal — read-only tools only", "minimal"),
		huh.NewOption("Standard — common dev tools allowed", "standard"),
		huh.NewOption("Permissive — broad tool access", "permissive"),
		huh.NewOption("Custom — fine-grained control", "custom"),
	}

	skillNames := claudecode.AvailableSkillNames()
	skillOpts := make([]huh.Option[string], len(skillNames))
	for i, name := range skillNames {
		skillOpts[i] = huh.NewOption(name, name)
	}

	mcpOpts := []huh.Option[string]{
		huh.NewOption("GitHub", "github"),
		huh.NewOption("Filesystem", "filesystem"),
		huh.NewOption("PostgreSQL", "postgres"),
		huh.NewOption("Fetch", "fetch"),
		huh.NewOption("Socket", "socket"),
	}

	claudeDetailFields := []huh.Field{
		huh.NewSelect[string]().
			Title("Permission level").
			Description("Controls which tools Claude Code is allowed to use.").
			Options(permOpts...).
			Value(&fs.permissionLevel),
		huh.NewConfirm().
			Title("Enable auto-format hook?").
			Description("Automatically formats code after Claude edits files.").
			Affirmative("Yes").
			Negative("No").
			Value(&fs.autoFormat),
		huh.NewConfirm().
			Title("Enable safety-block hook?").
			Description("Blocks potentially dangerous operations.").
			Affirmative("Yes").
			Negative("No").
			Value(&fs.safetyBlock),
	}

	if len(skillOpts) > 0 {
		claudeDetailFields = append(claudeDetailFields,
			huh.NewMultiSelect[string]().
				Title("Skills").
				Description("Select skills to install for Claude Code.").
				Options(skillOpts...).
				Value(&fs.skills),
		)
	}

	claudeDetailFields = append(claudeDetailFields,
		huh.NewMultiSelect[string]().
			Title("MCP servers").
			Description("Select Model Context Protocol servers to configure.").
			Options(mcpOpts...).
			Value(&fs.mcpServers),
	)

	claudeDetailGroup := huh.NewGroup(claudeDetailFields...).
		WithHideFunc(func() bool { return fs.quickChoice == "yes" || !fs.claudeCode })

	// --- Group 6: Preview & Confirm ---
	confirmGroup := huh.NewGroup(
		huh.NewNote().
			Title("Plan Preview").
			Description(buildPlanPreview(fs)),
		huh.NewConfirm().
			Title("Proceed with this configuration?").
			Affirmative("Yes, generate files").
			Negative("No, cancel").
			Value(&fs.confirmed),
	)

	form := huh.NewForm(
		quickGroup,
		langGroup,
		goVersionGroup,
		jsVersionGroup,
		pythonVersionGroup,
		servicesGroup,
		devEnvGroup,
		claudeEnableGroup,
		claudeDetailGroup,
		confirmGroup,
	).WithTheme(huh.ThemeDracula()).
		WithAccessible(isAccessible())

	return form
}

// mapFormToAnswers converts formState into WizardAnswers.
func mapFormToAnswers(fs *formState, projectRoot, projectName string, detected types.DetectedProject) types.WizardAnswers {
	answers := types.WizardAnswers{
		ProjectName:     projectName,
		ProjectRoot:     projectRoot,
		Detected:        detected,
		Direnv:          fs.direnv,
		ClaudeCode:      fs.claudeCode,
		PermissionLevel: fs.permissionLevel,
		Confirmed:       fs.confirmed,
		QuickChoice:     fs.quickChoice,
	}

	// On quick path, fill from detection defaults.
	if fs.quickChoice == "yes" {
		answers.FillDefaults(detected)
		answers.Confirmed = fs.confirmed
		return answers
	}

	// Map selected languages.
	for _, name := range fs.selectedLanguages {
		lc := types.LanguageChoice{Name: name}
		switch name {
		case "go":
			lc.Version = fs.goVersion
		case "javascript":
			lc.Version = fs.jsVersion
		case "python":
			lc.Version = fs.pythonVersion
		}
		answers.Languages = append(answers.Languages, lc)
	}

	// Map selected services.
	for _, name := range fs.selectedServices {
		answers.Services = append(answers.Services, types.ServiceChoice{Name: name})
	}

	// Git hooks.
	if len(fs.gitHooks) > 0 {
		answers.GitHooks = fs.gitHooks
	}

	// Extra packages: parse comma-separated input.
	answers.ExtraPackages = parseExtraPackages(fs.extraPackages)

	// Hooks.
	answers.Hooks = types.HookChoices{
		AutoFormat:  fs.autoFormat,
		SafetyBlock: fs.safetyBlock,
	}

	// Claude Code details (only when enabled).
	if fs.claudeCode {
		answers.Skills = fs.skills
		answers.MCPServers = fs.mcpServers
	} else {
		answers.Skills = nil
		answers.MCPServers = nil
	}

	return answers
}

// parseExtraPackages splits a comma-separated string into trimmed package names,
// filtering out empty entries.
func parseExtraPackages(input string) []string {
	if strings.TrimSpace(input) == "" {
		return nil
	}
	parts := strings.Split(input, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// isAccessible returns true when ACCESSIBLE or NO_COLOR env var is set.
func isAccessible() bool {
	if os.Getenv("ACCESSIBLE") != "" {
		return true
	}
	if os.Getenv("NO_COLOR") != "" {
		return true
	}
	return false
}

// flagSetHasAny returns true when any relevant flag was explicitly set.
func flagSetHasAny(fs *FlagSet) bool {
	relevantFlags := []string{
		"lang", "service", "direnv", "claude-code", "claude-permissions",
		"claude-skills", "claude-hooks", "git-hooks", "packages", "mcp",
		"go-version", "node-version", "node-pkg-mgr", "python-version",
		"python-pkg-mgr", "rust-channel", "java-version", "java-build-tool",
		"infra-profile",
	}
	for _, name := range relevantFlags {
		if fs.IsSet(name) {
			return true
		}
	}
	return false
}

// languageLabel returns a display label for a language name.
func languageLabel(name string) string {
	labels := map[string]string{
		"go":         "Go",
		"javascript": "JavaScript/TypeScript",
		"python":     "Python",
		"rust":       "Rust",
		"java":       "Java/Kotlin",
		"dotnet":     "C#/.NET",
		"docker":     "Docker",
		"terraform":  "Terraform/OpenTofu",
	}
	if l, ok := labels[name]; ok {
		return l
	}
	return name
}

// serviceLabel returns a display label for a service name.
func serviceLabel(name string) string {
	labels := map[string]string{
		"postgres":      "PostgreSQL",
		"redis":         "Redis",
		"mysql":         "MySQL",
		"mongodb":       "MongoDB",
		"elasticsearch": "Elasticsearch",
		"rabbitmq":      "RabbitMQ",
	}
	if l, ok := labels[name]; ok {
		return l
	}
	return name
}
