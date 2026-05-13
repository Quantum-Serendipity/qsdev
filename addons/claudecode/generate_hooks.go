package claudecode

import (
	"fmt"

	"fastcat.org/go/gdev-secure-devenv-bootstrap/pkg/types"
)

// GenerateHookFiles returns GeneratedFile entries for all enabled hook presets.
func GenerateHookFiles(answers types.WizardAnswers) ([]types.GeneratedFile, error) {
	var files []types.GeneratedFile

	if answers.Hooks.SafetyBlock {
		content, err := templateFS.ReadFile("templates/hooks/package-guard.py")
		if err != nil {
			return nil, fmt.Errorf("reading package-guard hook: %w", err)
		}
		files = append(files, types.GeneratedFile{
			Path:     ".claude/hooks/package-guard.py",
			Content:  content,
			Mode:     0o755,
			Strategy: types.Overwrite,
		})
	}

	if answers.Hooks.AuditLog {
		content, err := templateFS.ReadFile("templates/hooks/audit-log.sh")
		if err != nil {
			return nil, fmt.Errorf("reading audit-log hook: %w", err)
		}
		files = append(files, types.GeneratedFile{
			Path:     ".claude/hooks/audit-log.sh",
			Content:  content,
			Mode:     0o755,
			Strategy: types.Overwrite,
		})
	}

	return files, nil
}

// GeneratePackageGuardHook returns a GeneratedFile for the PreToolUse package
// install guardrail hook when safety blocking is enabled, or nil when it is
// disabled or not requested.
func GeneratePackageGuardHook(answers types.WizardAnswers) (*types.GeneratedFile, error) {
	if !answers.Hooks.SafetyBlock {
		return nil, nil
	}

	content, err := templateFS.ReadFile("templates/hooks/package-guard.py")
	if err != nil {
		return nil, fmt.Errorf("reading package-guard hook: %w", err)
	}

	return &types.GeneratedFile{
		Path:     ".claude/hooks/package-guard.py",
		Content:  content,
		Mode:     0o755,
		Strategy: types.Overwrite,
	}, nil
}
