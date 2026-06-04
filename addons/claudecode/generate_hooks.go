package claudecode

import (
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// GenerateHookFiles returns GeneratedFile entries for all enabled hook presets.
func GenerateHookFiles(answers types.WizardAnswers) ([]types.GeneratedFile, error) {
	specs := []hookFileSpec{
		{
			enabled:      answers.Hooks.SafetyBlock,
			templatePath: "templates/hooks/package-guard.py",
			outputPath:   ".claude/hooks/package-guard.py",
			mode:         0o755,
			strategy:     types.Overwrite,
			owner:        "attach-guard",
		},
		{
			enabled:      answers.Hooks.CredentialScan,
			templatePath: "templates/hooks/scan-secrets.py",
			outputPath:   ".claude/hooks/scan-secrets.py",
			mode:         0o755,
			strategy:     types.Overwrite,
			owner:        "credential-scan",
		},
		{
			enabled:      answers.Hooks.DestructivePrevention,
			templatePath: "templates/hooks/block-destructive.py",
			outputPath:   ".claude/hooks/block-destructive.py",
			mode:         0o755,
			strategy:     types.Overwrite,
			owner:        "destructive-prevention",
		},
		{
			enabled:      answers.Hooks.FileBoundary,
			templatePath: "templates/hooks/file-boundary.py",
			outputPath:   ".claude/hooks/file-boundary.py",
			mode:         0o755,
			strategy:     types.Overwrite,
			owner:        "file-boundary",
		},
		{
			enabled:      answers.Hooks.ToolGates,
			templatePath: "templates/hooks/tool-gates.py",
			outputPath:   ".claude/hooks/tool-gates.py",
			mode:         0o755,
			strategy:     types.Overwrite,
			owner:        "tool-gates",
		},
		{
			enabled:      answers.Hooks.SOC2Audit,
			templatePath: "templates/hooks/soc2-audit-log.py",
			outputPath:   ".claude/hooks/soc2-audit-log.py",
			mode:         0o755,
			strategy:     types.Overwrite,
			owner:        "soc2-audit",
		},
		{
			enabled:      answers.Hooks.AuditLog && !answers.Hooks.SOC2Audit,
			templatePath: "templates/hooks/audit-log.sh",
			outputPath:   ".claude/hooks/audit-log.sh",
			mode:         0o755,
			strategy:     types.Overwrite,
		},
	}

	var files []types.GeneratedFile
	for _, spec := range specs {
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
