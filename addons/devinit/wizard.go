package devinit

import (
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/termutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// formState holds intermediate variables that huh form fields bind to.
type formState struct {
	// partial holds the answers collected before the wizard ran (flags,
	// profile). The wizard overlays its fields onto a copy of it, so settings
	// the form does not ask about (tier, env vars, infra profile, enabled
	// tools, additional hooks) survive.
	partial types.WizardAnswers

	quickChoice string // "yes", "show", "customize"

	selectedLanguages []string
	goVersion         string
	jsVersion         string
	pythonVersion     string

	selectedServices []string

	direnv            bool
	gitHooks          []string
	extraPackages     string // comma-separated input
	nixHardeningGuide bool

	claudeCode      bool
	permissionLevel string
	skills          []string
	autoFormat      bool
	safetyBlock     bool
	mcpServers      []string

	agentPostmortem      bool
	agentVersionSentinel bool
	agentSemble          bool
	agentSembleMode      string
	agentSembleTextFiles bool

	confirmed bool
}

// previewBindings returns pointers to every form field the Plan Preview
// depends on. huh hashes the bindings to decide when to re-render the
// preview; formState's fields are unexported and would be skipped by the
// hash, so the fields themselves must be bound.
func (fs *formState) previewBindings() []any {
	return []any{
		&fs.quickChoice,
		&fs.selectedLanguages, &fs.goVersion, &fs.jsVersion, &fs.pythonVersion,
		&fs.selectedServices,
		&fs.direnv, &fs.gitHooks, &fs.extraPackages, &fs.nixHardeningGuide,
		&fs.claudeCode, &fs.permissionLevel, &fs.skills, &fs.autoFormat, &fs.safetyBlock, &fs.mcpServers,
		&fs.agentPostmortem, &fs.agentVersionSentinel, &fs.agentSemble, &fs.agentSembleMode, &fs.agentSembleTextFiles,
	}
}

// resolveTheme maps a theme name to a huh theme.
func resolveTheme(name string) *huh.Theme {
	switch name {
	case "charm":
		return huh.ThemeCharm()
	case "dracula":
		return huh.ThemeDracula()
	case "catppuccin":
		return huh.ThemeCatppuccin()
	case "base16":
		return huh.ThemeBase16()
	case "default", "":
		return huh.ThemeDracula()
	default:
		return huh.ThemeDracula()
	}
}

// RunWizard runs the interactive huh form, collecting user choices.
// It pre-populates defaults from detection and any partial flag answers.
// Returns the fully populated WizardAnswers.
func RunWizard(projectRoot string, detected types.DetectedProject, partial types.WizardAnswers, flagSet *FlagSet, themeName string) (types.WizardAnswers, error) {
	defaults := MapDetectionToDefaults(detected, projectRoot)
	partial.ProjectRoot = projectRoot
	partial.ProjectName = defaults.ProjectName
	partial.Detected = detected

	fs := newFormState(detected, defaults, partial, flagSet)
	if err := runWizardForm(detected, fs, flagSet, themeName); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return types.WizardAnswers{Confirmed: false}, nil
		}
		return types.WizardAnswers{}, fmt.Errorf("wizard form: %w", err)
	}

	return mapFormToAnswers(fs, projectRoot, defaults.ProjectName, detected), nil
}

// newFormState seeds the form from detection, then applies the values the
// user supplied before the wizard. Flag values only override a seed when the
// flag was explicitly set, because unset flags carry cobra defaults (e.g.
// --claude-hooks is empty) that would otherwise clobber the seeds.
func newFormState(detected types.DetectedProject, defaults, partial types.WizardAnswers, flagSet *FlagSet) *formState {
	// The languages the form starts with: explicit ones, else detected ones.
	seedLangs := defaults.Languages
	if len(partial.Languages) > 0 {
		seedLangs = partial.Languages
	}
	fs := &formState{
		partial:              partial,
		quickChoice:          "yes",
		selectedLanguages:    PreSelectedLanguages(detected),
		direnv:               true,
		claudeCode:           true,
		permissionLevel:      "standard",
		safetyBlock:          true,
		mcpServers:           catalog.MustDefault().DefaultMCPServers(),
		agentPostmortem:      true,
		agentVersionSentinel: hasVSSupportedLanguage(seedLangs),
		agentSemble:          pythonVersionAtLeast(detected.PythonVersion, 3, 10),
		agentSembleMode:      "mcp",
	}

	// Explicit flags already answer part of the quick-setup question, so
	// skip it and start from the customize screens pre-filled with them.
	if flagSetHasAny(flagSet) {
		fs.quickChoice = "customize"
	}

	seedLanguageVersions(fs, defaults.Languages)
	seedFromPartial(fs, partial, flagSet)
	return fs
}

// seedLanguageVersions copies non-empty versions of the languages that have a
// version prompt into the form.
func seedLanguageVersions(fs *formState, langs []types.LanguageChoice) {
	for _, lang := range langs {
		if lang.Version == "" {
			continue
		}
		switch lang.Name {
		case "go":
			fs.goVersion = lang.Version
		case "javascript":
			fs.jsVersion = lang.Version
		case "python":
			fs.pythonVersion = lang.Version
		}
	}
}

// seedFromPartial overrides the detection seeds with the answers supplied
// before the wizard ran.
func seedFromPartial(fs *formState, partial types.WizardAnswers, flagSet *FlagSet) {
	if len(partial.Languages) > 0 {
		fs.selectedLanguages = make([]string, len(partial.Languages))
		for i, l := range partial.Languages {
			fs.selectedLanguages[i] = l.Name
		}
		seedLanguageVersions(fs, partial.Languages)
	}
	if len(partial.Services) > 0 {
		fs.selectedServices = make([]string, len(partial.Services))
		for i, s := range partial.Services {
			fs.selectedServices[i] = s.Name
		}
	}
	if flagSet.IsSet("direnv") {
		fs.direnv = partial.Direnv
	}
	if flagSet.IsSet("claude-code") || flagSet.IsSet("devenv-only") {
		fs.claudeCode = partial.ClaudeCode
	}
	if level := seedPermissionLevel(partial); level != "" {
		fs.permissionLevel = level
	}
	if len(partial.Skills) > 0 {
		fs.skills = slices.Clone(partial.Skills)
	}
	if len(partial.MCPServers) > 0 {
		fs.mcpServers = slices.Clone(partial.MCPServers)
	}
	if len(partial.GitHooks) > 0 {
		fs.gitHooks = slices.Clone(partial.GitHooks)
	}
	if len(partial.ExtraPackages) > 0 {
		fs.extraPackages = strings.Join(partial.ExtraPackages, ", ")
	}
	fs.nixHardeningGuide = partial.NixHardeningGuide
	if flagSet.IsSet("claude-hooks") {
		fs.autoFormat = partial.Hooks.AutoFormat
		fs.safetyBlock = partial.Hooks.SafetyBlock
	}
	seedAgentTools(fs, partial.AgentTools, flagSet)
}

// seedPermissionLevel returns the permission preset the form should start
// from: the explicit level, else the selected tier's default preset, else ""
// (keep the form default).
func seedPermissionLevel(partial types.WizardAnswers) string {
	if partial.PermissionLevel != "" {
		return partial.PermissionLevel
	}
	if partial.Tier == "" {
		return ""
	}
	resolved, err := catalog.MustDefault().ResolveTier(partial.Tier)
	if err != nil {
		return ""
	}
	return resolved.DefaultPermissionPreset
}

// seedAgentTools applies explicitly set --agent-* flags to the form.
func seedAgentTools(fs *formState, tools types.AgentToolsAnswers, flagSet *FlagSet) {
	if flagSet.IsSet("agent-postmortem") {
		fs.agentPostmortem = tools.PostmortemEnabled
	}
	if flagSet.IsSet("agent-version-sentinel") {
		fs.agentVersionSentinel = tools.VersionSentinel
	}
	if flagSet.IsSet("agent-semble") {
		fs.agentSemble = tools.SembleEnabled
	}
	if flagSet.IsSet("agent-semble-mode") && tools.SembleMode != "" {
		fs.agentSembleMode = tools.SembleMode
	}
	if flagSet.IsSet("agent-semble-text-files") {
		fs.agentSembleTextFiles = tools.SembleTextFiles
	}
}

// runWizardForm runs the wizard, as a TUI form or, when the environment asks
// for it, as plain accessible prompts.
func runWizardForm(detected types.DetectedProject, fs *formState, flagSet *FlagSet, themeName string) error {
	if termutil.IsAccessible() {
		steps := buildWizardSteps(detected, fs, flagSet)
		return runAccessibleSteps(steps, resolveTheme(themeName), os.Stdout, os.Stdin)
	}
	return buildWizardForm(detected, fs, flagSet, themeName).Run()
}

// wizardStep is one screen of the wizard. Its fields are built on demand so
// the accessible runner renders each screen (including the Plan Preview)
// from the answers given on earlier screens.
type wizardStep struct {
	fields func() []huh.Field
	hidden func() bool // nil: always shown
}

// buildWizardSteps lists the wizard screens in order.
func buildWizardSteps(detected types.DetectedProject, fs *formState, flagSet *FlagSet) []wizardStep {
	quick := quickPathAnswers(fs.partial, detected)
	anyFlagExplicit := flagSetHasAny(flagSet)

	steps := []wizardStep{
		quickSelectStep(QuickPathSummary(quick), anyFlagExplicit, fs),
		showDefaultsStep(quick, fs),
	}
	steps = append(steps, languageSteps(detected, fs)...)
	steps = append(steps, servicesStep(fs), securityStep(fs))
	steps = append(steps, claudeCodeSteps(fs)...)
	return append(steps, confirmStep(fs))
}

// buildWizardForm constructs the TUI huh form from the wizard steps.
func buildWizardForm(detected types.DetectedProject, fs *formState, flagSet *FlagSet, themeName string) *huh.Form {
	steps := buildWizardSteps(detected, fs, flagSet)
	groups := make([]*huh.Group, len(steps))
	for i, step := range steps {
		groups[i] = huh.NewGroup(step.fields()...)
		if step.hidden != nil {
			groups[i] = groups[i].WithHideFunc(step.hidden)
		}
	}
	return huh.NewForm(groups...).WithTheme(resolveTheme(themeName))
}

// runAccessibleSteps runs the wizard as sequential plain-text prompts.
// huh's own accessible mode ignores group hide funcs (so it would ask every
// question of every branch) and discards field errors, so the wizard drives
// the prompts itself. End of input before a prompt is answered aborts the
// wizard instead of silently accepting defaults.
func runAccessibleSteps(steps []wizardStep, theme *huh.Theme, w io.Writer, r io.Reader) (err error) {
	in := &eofTrackingReader{r: r}
	defer func() {
		if p := recover(); p != nil {
			if in.eof {
				err = huh.ErrUserAborted
				return
			}
			err = fmt.Errorf("accessible prompt failed: %v", p)
		}
	}()

	for _, step := range steps {
		if step.hidden != nil && step.hidden() {
			continue
		}
		for _, field := range step.fields() {
			start := in.n
			field = field.WithTheme(theme)
			_ = field.Init()
			_ = field.Focus()
			if err := field.RunAccessible(w, in); err != nil {
				return fmt.Errorf("accessible prompt: %w", err)
			}
			_, _ = fmt.Fprintln(w)
			if in.eof && in.n == start {
				return huh.ErrUserAborted
			}
		}
	}
	return nil
}

// eofTrackingReader records how many bytes were read and whether the
// underlying reader reached EOF, which huh's accessible prompts swallow.
type eofTrackingReader struct {
	r   io.Reader
	n   int
	eof bool
}

func (e *eofTrackingReader) Read(p []byte) (int, error) {
	n, err := e.r.Read(p)
	e.n += n
	if errors.Is(err, io.EOF) {
		e.eof = true
	}
	return n, err
}

func quickSelectStep(summary string, anyFlagExplicit bool, fs *formState) wizardStep {
	return wizardStep{
		fields: func() []huh.Field {
			return []huh.Field{
				huh.NewSelect[string]().
					Title("Quick setup detected your project").
					Description("We detected your project configuration. Would you like to use these defaults?").
					Options(
						huh.NewOption("Yes — "+summary, "yes"),
						huh.NewOption("Show me what the defaults include", "show"),
						huh.NewOption("No, let me customize", "customize"),
					).
					Value(&fs.quickChoice),
			}
		},
		hidden: func() bool { return anyFlagExplicit },
	}
}

func showDefaultsStep(quick types.WizardAnswers, fs *formState) wizardStep {
	return wizardStep{
		fields: func() []huh.Field {
			return []huh.Field{
				huh.NewNote().
					Title("Default Configuration Details").
					Description(buildDetailedDefaults(quick)),
				huh.NewSelect[string]().
					Title("How would you like to proceed?").
					Options(
						huh.NewOption("Accept these defaults", "yes"),
						huh.NewOption("Customize", "customize"),
					).
					Value(&fs.quickChoice),
			}
		},
		hidden: func() bool { return fs.quickChoice != "show" },
	}
}

func languageSteps(detected types.DetectedProject, fs *formState) []wizardStep {
	onQuickPath := func() bool { return fs.quickChoice == "yes" }

	langStep := wizardStep{
		fields: func() []huh.Field {
			langOptions := BuildLanguageOptions(detected)
			langOpts := make([]huh.Option[string], len(langOptions))
			for i, lo := range langOptions {
				langOpts[i] = huh.NewOption(lo.Label, lo.Value)
			}
			return []huh.Field{
				huh.NewMultiSelect[string]().
					Title("Languages & Runtimes").
					Description("Select the languages and platforms for this project.").
					Options(langOpts...).
					Value(&fs.selectedLanguages),
			}
		},
		hidden: onQuickPath,
	}

	versionStep := func(lang, title, placeholder string, value *string) wizardStep {
		return wizardStep{
			fields: func() []huh.Field {
				return []huh.Field{huh.NewInput().Title(title).Placeholder(placeholder).Value(value)}
			},
			hidden: func() bool {
				return onQuickPath() || !slices.Contains(fs.selectedLanguages, lang)
			},
		}
	}

	return []wizardStep{
		langStep,
		versionStep("go", "Go version", "e.g. 1.24", &fs.goVersion),
		versionStep("javascript", "Node.js version", "e.g. 22", &fs.jsVersion),
		versionStep("python", "Python version", "e.g. 3.12", &fs.pythonVersion),
	}
}

func servicesStep(fs *formState) wizardStep {
	return wizardStep{
		fields: func() []huh.Field {
			return []huh.Field{
				huh.NewMultiSelect[string]().
					Title("Services").
					Description("Select development services to include.").
					Options(serviceOptions()...).
					Value(&fs.selectedServices),
			}
		},
		hidden: func() bool { return fs.quickChoice == "yes" },
	}
}

// serviceOptions lists every service the catalog supports.
func serviceOptions() []huh.Option[string] {
	names := catalog.MustDefault().Services()
	opts := make([]huh.Option[string], len(names))
	for i, name := range names {
		opts[i] = huh.NewOption(serviceLabel(name), name)
	}
	return opts
}

func securityStep(fs *formState) wizardStep {
	hookOpts := []huh.Option[string]{
		huh.NewOption("pre-commit", "pre-commit"),
		huh.NewOption("pre-push", "pre-push"),
		huh.NewOption("commit-msg", "commit-msg"),
	}

	return wizardStep{
		fields: func() []huh.Field {
			return []huh.Field{
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
				huh.NewConfirm().
					Title("Generate Nix hardening guide?").
					Description("Creates nix-hardening.md with security best practices for your Nix configuration.").
					Affirmative("Yes").
					Negative("No").
					Value(&fs.nixHardeningGuide),
			}
		},
		hidden: func() bool { return fs.quickChoice == "yes" },
	}
}

func claudeCodeSteps(fs *formState) []wizardStep {
	enableStep := wizardStep{
		fields: func() []huh.Field {
			return []huh.Field{
				huh.NewConfirm().
					Title("Enable Claude Code?").
					Description("Generates .claude/settings.json, CLAUDE.md, hooks, and skills.").
					Affirmative("Yes").
					Negative("No").
					Value(&fs.claudeCode),
			}
		},
		hidden: func() bool { return fs.quickChoice == "yes" },
	}

	detailStep := wizardStep{
		fields: func() []huh.Field { return claudeDetailFields(fs) },
		hidden: func() bool { return fs.quickChoice == "yes" || !fs.claudeCode },
	}

	sembleStep := wizardStep{
		fields: func() []huh.Field {
			return []huh.Field{
				huh.NewSelect[string]().
					Title("Semble mode").
					Options(
						huh.NewOption("MCP server", "mcp"),
						huh.NewOption("Sub-agent", "subagent"),
						huh.NewOption("Both", "both"),
					).
					Value(&fs.agentSembleMode),
				huh.NewConfirm().
					Title("Include text files in semble index?").
					Description("Enables --include-text-files for infra-heavy repos (YAML/Markdown)").
					Affirmative("Yes").
					Negative("No").
					Value(&fs.agentSembleTextFiles),
			}
		},
		hidden: func() bool {
			return fs.quickChoice == "yes" || !fs.claudeCode || !fs.agentSemble
		},
	}

	return []wizardStep{enableStep, detailStep, sembleStep}
}

// claudeDetailFields builds the Claude Code detail questions.
func claudeDetailFields(fs *formState) []huh.Field {
	fields := []huh.Field{
		huh.NewSelect[string]().
			Title("Permission level").
			Description("Controls which tools Claude Code is allowed to use.").
			Options(permissionOptions()...).
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

	if skillNames := claudecode.AvailableSkillNames(); len(skillNames) > 0 {
		skillOpts := make([]huh.Option[string], len(skillNames))
		for i, name := range skillNames {
			skillOpts[i] = huh.NewOption(name, name)
		}
		fields = append(fields,
			huh.NewMultiSelect[string]().
				Title("Skills").
				Description("Select skills to install for Claude Code.").
				Options(skillOpts...).
				Value(&fs.skills),
		)
	}

	return append(fields,
		huh.NewMultiSelect[string]().
			Title("MCP servers").
			Description("Select Model Context Protocol servers to configure.").
			Options(mcpServerOptions()...).
			Value(&fs.mcpServers),
		huh.NewConfirm().
			Title("Agent-postmortem skill").
			Description("Require evidence-backed verification before claiming tasks done").
			Affirmative("Yes").
			Negative("No").
			Value(&fs.agentPostmortem),
		huh.NewConfirm().
			Title("Version-Sentinel").
			Description("Block dependency changes until versions verified against registry").
			Affirmative("Yes").
			Negative("No").
			Value(&fs.agentVersionSentinel),
		huh.NewConfirm().
			Title("Semble semantic search").
			Description("Semantic code search for AI agents (~98% fewer tokens). Requires Python >=3.10").
			Affirmative("Yes").
			Negative("No").
			Value(&fs.agentSemble),
	)
}

// permissionDescriptions adds a short explanation to known permission presets.
var permissionDescriptions = map[string]string{
	"minimal":           "Minimal — read-only tools only",
	"standard":          "Standard — common dev tools allowed",
	"permissive":        "Permissive — broad tool access",
	"custom":            "Custom — fine-grained control",
	"supply-chain-only": "Supply-chain only — package-install guardrails only",
}

// permissionOptions lists every permission preset the catalog supports.
func permissionOptions() []huh.Option[string] {
	presets := catalog.MustDefault().PermissionPresets()
	opts := make([]huh.Option[string], len(presets))
	for i, name := range presets {
		label := name
		if desc, ok := permissionDescriptions[name]; ok {
			label = desc
		}
		opts[i] = huh.NewOption(label, name)
	}
	return opts
}

// mcpServerOptions lists every MCP server the catalog defines.
func mcpServerOptions() []huh.Option[string] {
	cat := catalog.MustDefault()
	names := cat.MCPServerNames()
	opts := make([]huh.Option[string], len(names))
	for i, name := range names {
		label := name
		if def, ok := cat.MCPServer(name); ok && def.DisplayName != "" {
			label = def.DisplayName
		}
		opts[i] = huh.NewOption(label, name)
	}
	return opts
}

func confirmStep(fs *formState) wizardStep {
	return wizardStep{
		fields: func() []huh.Field {
			return []huh.Field{
				huh.NewNote().
					Title("Plan Preview").
					Description(buildPlanPreview(fs)).
					DescriptionFunc(func() string { return buildPlanPreview(fs) }, fs.previewBindings()),
				huh.NewConfirm().
					Title("Proceed with this configuration?").
					Affirmative("Yes, generate files").
					Negative("No, cancel").
					Value(&fs.confirmed),
			}
		},
	}
}

// quickPathAnswers returns the answers the "use these defaults" choice
// produces: the pre-wizard answers completed from detection and the catalog.
func quickPathAnswers(partial types.WizardAnswers, detected types.DetectedProject) types.WizardAnswers {
	answers := cloneAnswers(partial)
	answers.FillDefaults(detected, catalog.MustDefault())
	enforceAnswerInvariants(&answers)
	return answers
}

// mapFormToAnswers converts formState into WizardAnswers. It starts from the
// quick-path answers so settings the form does not ask about (tier, env vars,
// tier-derived compliance and tools) are kept and completed the same way on
// both paths; the customize path then overlays the form's choices.
func mapFormToAnswers(fs *formState, projectRoot, projectName string, detected types.DetectedProject) types.WizardAnswers {
	answers := quickPathAnswers(fs.partial, detected)
	if fs.quickChoice != "yes" {
		applyFormChoices(&answers, fs, detected)
		enforceAnswerInvariants(&answers)
	}

	answers.ProjectName = projectName
	answers.ProjectRoot = projectRoot
	answers.Detected = detected
	answers.QuickChoice = fs.quickChoice
	answers.Confirmed = fs.confirmed
	return answers
}

// applyFormChoices overlays the customize-path form fields onto answers.
func applyFormChoices(answers *types.WizardAnswers, fs *formState, detected types.DetectedProject) {
	answers.Languages = formLanguages(fs, answers.Languages, detected)
	answers.Services = formServices(fs.selectedServices, answers.Services)
	answers.Direnv = fs.direnv
	answers.GitHooks = slices.Clone(fs.gitHooks)
	answers.ExtraPackages = parseExtraPackages(fs.extraPackages)
	answers.NixHardeningGuide = fs.nixHardeningGuide
	answers.Hooks.AutoFormat = fs.autoFormat
	answers.Hooks.SafetyBlock = fs.safetyBlock
	answers.ClaudeCode = fs.claudeCode
	answers.PermissionLevel = fs.permissionLevel

	if !fs.claudeCode {
		answers.Skills = nil
		answers.MCPServers = nil
		answers.AgentTools = types.AgentToolsAnswers{}
		return
	}

	answers.Skills = slices.Clone(fs.skills)
	answers.MCPServers = slices.Clone(fs.mcpServers)
	answers.AgentTools.PostmortemEnabled = fs.agentPostmortem
	answers.AgentTools.VersionSentinel = fs.agentVersionSentinel
	answers.AgentTools.SembleEnabled = fs.agentSemble
	answers.AgentTools.SembleMode = fs.agentSembleMode
	answers.AgentTools.SembleTextFiles = fs.agentSembleTextFiles
	if answers.AgentTools.VersionSentinelHours == 0 {
		answers.AgentTools.VersionSentinelHours = catalog.MustDefault().DefaultVersionSentinelHours()
	}
}

// formLanguages builds the language list from the selected names. Each entry
// keeps the package manager and extras from the pre-wizard answers or, failing
// that, from detection; the versions the form asks for come from the form.
func formLanguages(fs *formState, base []types.LanguageChoice, detected types.DetectedProject) []types.LanguageChoice {
	known := make(map[string]types.LanguageChoice)
	for _, lc := range MapDetectionToDefaults(detected, "").Languages {
		known[lc.Name] = lc
	}
	for _, lc := range base {
		known[lc.Name] = lc
	}

	var langs []types.LanguageChoice
	for _, name := range fs.selectedLanguages {
		lc, ok := known[name]
		if !ok {
			lc = types.LanguageChoice{Name: name}
		}
		lc.Extras = slices.Clone(lc.Extras)
		switch name {
		case "go":
			lc.Version = fs.goVersion
		case "javascript":
			lc.Version = fs.jsVersion
		case "python":
			lc.Version = fs.pythonVersion
		}
		langs = append(langs, lc)
	}
	return langs
}

// formServices builds the service list from the selected names, keeping the
// version and settings of services that were already configured.
func formServices(selected []string, base []types.ServiceChoice) []types.ServiceChoice {
	var services []types.ServiceChoice
	for _, name := range selected {
		sc := types.ServiceChoice{Name: name}
		if i := slices.IndexFunc(base, func(s types.ServiceChoice) bool { return s.Name == name }); i >= 0 {
			sc = base[i]
		}
		services = append(services, sc)
	}
	return services
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

// flagSetHasAny returns true when any relevant flag was explicitly set.
func flagSetHasAny(fs *FlagSet) bool {
	relevantFlags := []string{
		"lang", "service", "direnv", "claude-code", "claude-permissions",
		"claude-skills", "claude-hooks", "git-hooks", "packages", "mcp",
		"go-version", "node-version", "node-pkg-mgr", "python-version",
		"python-pkg-mgr", "rust-channel", "java-version", "java-build-tool",
		"infra-profile",
		"agent-postmortem", "agent-version-sentinel", "agent-semble",
		"agent-semble-mode", "agent-semble-text-files",
	}
	for _, name := range relevantFlags {
		if fs.IsSet(name) {
			return true
		}
	}
	return false
}

// hasVSSupportedLanguage checks whether any selected language is covered by
// Version-Sentinel (npm, pip, cargo, nuget).
func hasVSSupportedLanguage(langs []types.LanguageChoice) bool {
	for _, l := range langs {
		switch l.Name {
		case "javascript", "python", "rust", "dotnet":
			return true
		}
	}
	return false
}

// pythonVersionAtLeast parses a version string like "3.12" or "3.10.1" and
// returns true when it is at least major.minor.
func pythonVersionAtLeast(version string, major, minor int) bool {
	if version == "" {
		return false
	}
	var parts [2]int
	idx := 0
	n := 0
	for i := 0; i < len(version) && idx < 2; i++ {
		if version[i] == '.' {
			parts[idx] = n
			idx++
			n = 0
		} else if version[i] >= '0' && version[i] <= '9' {
			n = n*10 + int(version[i]-'0')
		} else {
			break
		}
	}
	if idx < 2 {
		parts[idx] = n
	}
	if parts[0] > major {
		return true
	}
	return parts[0] == major && parts[1] >= minor
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
		"kafka":         "Kafka",
		"minio":         "MinIO",
		"mailpit":       "Mailpit",
		"keycloak":      "Keycloak",
		"nats":          "NATS",
	}
	if l, ok := labels[name]; ok {
		return l
	}
	return name
}
