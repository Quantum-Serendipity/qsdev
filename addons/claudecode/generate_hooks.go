package claudecode

import (
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// PackageGuardPath is the project-relative path of the package guard.
const PackageGuardPath = ".claude/hooks/package-guard.py"

// packageGuardTemplate is the embedded template written to PackageGuardPath.
const packageGuardTemplate = "templates/hooks/package-guard.py"

// PackageGuardContent returns the content the generator writes to
// PackageGuardPath: the embedded template, copied verbatim. The guard on
// disk is judged against it (posture.AssessOptions.PackageGuard).
func PackageGuardContent() []byte {
	return HookScriptContents()[PackageGuardPath]
}

// HookScriptContents maps the project-relative path of every hook script the
// generator can write, whatever the answers enable, to the content it writes
// there: the embedded template, copied verbatim. A hook script on disk is
// judged against it, so the judgement needs neither the project's answers nor
// a successful generation run, either of which a committed config change can
// take away.
func HookScriptContents() map[string][]byte {
	specs := hookFileSpecs(types.WizardAnswers{})
	contents := make(map[string][]byte, len(specs))
	for _, spec := range specs {
		content, err := templateFS.ReadFile(spec.templatePath)
		if err != nil {
			continue // embedded at build time; a missing one is left unverified
		}
		contents[spec.outputPath] = content
	}
	return contents
}

// GenerateHookFiles returns GeneratedFile entries for all enabled hook presets.
func GenerateHookFiles(answers types.WizardAnswers) ([]types.GeneratedFile, error) {
	var files []types.GeneratedFile
	for _, spec := range hookFileSpecs(answers) {
		f, err := generateHookFile(spec)
		if err != nil {
			return nil, err
		}
		if f != nil {
			files = append(files, *f)
		}
	}

	return files, nil
}

// hookFileSpecs lists every hook script the generator can write, each enabled
// as answers decide.
func hookFileSpecs(answers types.WizardAnswers) []hookFileSpec {
	return []hookFileSpec{
		{
			enabled:      packageGuardEnabled(answers),
			templatePath: packageGuardTemplate,
			outputPath:   PackageGuardPath,
			mode:         fileutil.ModeExecutable,
			strategy:     types.Overwrite,
			owner:        "attach-guard",
		},
		{
			enabled:      answers.Hooks.CredentialScan,
			templatePath: "templates/hooks/scan-secrets.py",
			outputPath:   ".claude/hooks/scan-secrets.py",
			mode:         fileutil.ModeExecutable,
			strategy:     types.Overwrite,
			owner:        "credential-scan",
		},
		{
			enabled:      answers.Hooks.DestructivePrevention,
			templatePath: "templates/hooks/block-destructive.py",
			outputPath:   ".claude/hooks/block-destructive.py",
			mode:         fileutil.ModeExecutable,
			strategy:     types.Overwrite,
			owner:        "destructive-prevention",
		},
		{
			enabled:      answers.Hooks.FileBoundary,
			templatePath: "templates/hooks/file-boundary.py",
			outputPath:   ".claude/hooks/file-boundary.py",
			mode:         fileutil.ModeExecutable,
			strategy:     types.Overwrite,
			owner:        "file-boundary",
		},
		{
			enabled:      answers.Hooks.ToolGates,
			templatePath: "templates/hooks/tool-gates.py",
			outputPath:   ".claude/hooks/tool-gates.py",
			mode:         fileutil.ModeExecutable,
			strategy:     types.Overwrite,
			owner:        "tool-gates",
		},
		{
			enabled:      answers.Hooks.SOC2Audit,
			templatePath: "templates/hooks/soc2-audit-log.py",
			outputPath:   ".claude/hooks/soc2-audit-log.py",
			mode:         fileutil.ModeExecutable,
			strategy:     types.Overwrite,
			owner:        "soc2-audit",
		},
		{
			enabled:      lspGuardEnabled(answers),
			templatePath: "templates/hooks/lsp-first-guard.sh",
			outputPath:   ".claude/hooks/lsp-first-guard.sh",
			mode:         fileutil.ModeExecutable,
			strategy:     types.Overwrite,
			owner:        "lsp-guard",
		},
		{
			enabled:      answers.AgentTools.SembleEnabled,
			templatePath: "templates/hooks/semble-analytics.sh",
			outputPath:   ".claude/hooks/semble-analytics.sh",
			mode:         fileutil.ModeExecutable,
			strategy:     types.Overwrite,
			owner:        "semble",
		},
		{
			enabled:      answers.Hooks.AuditLog && !answers.Hooks.SOC2Audit,
			templatePath: "templates/hooks/audit-log.sh",
			outputPath:   ".claude/hooks/audit-log.sh",
			mode:         fileutil.ModeExecutable,
			strategy:     types.Overwrite,
		},
	}
}
