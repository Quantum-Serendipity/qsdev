package claudecode

import (
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
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
			Owner:    "attach-guard",
		})
	}

	if answers.Hooks.CredentialScan {
		content, err := templateFS.ReadFile("templates/hooks/scan-secrets.py")
		if err != nil {
			return nil, fmt.Errorf("reading credential scan hook: %w", err)
		}
		files = append(files, types.GeneratedFile{
			Path:     ".claude/hooks/scan-secrets.py",
			Content:  content,
			Mode:     0o755,
			Strategy: types.Overwrite,
			Owner:    "credential-scan",
		})
	}

	if answers.Hooks.DestructivePrevention {
		content, err := templateFS.ReadFile("templates/hooks/block-destructive.py")
		if err != nil {
			return nil, fmt.Errorf("reading destructive prevention hook: %w", err)
		}
		files = append(files, types.GeneratedFile{
			Path:     ".claude/hooks/block-destructive.py",
			Content:  content,
			Mode:     0o755,
			Strategy: types.Overwrite,
			Owner:    "destructive-prevention",
		})
	}

	if answers.Hooks.FileBoundary {
		content, err := templateFS.ReadFile("templates/hooks/file-boundary.py")
		if err != nil {
			return nil, fmt.Errorf("reading file boundary hook: %w", err)
		}
		files = append(files, types.GeneratedFile{
			Path:     ".claude/hooks/file-boundary.py",
			Content:  content,
			Mode:     0o755,
			Strategy: types.Overwrite,
			Owner:    "file-boundary",
		})
	}

	if answers.Hooks.ToolGates {
		content, err := templateFS.ReadFile("templates/hooks/tool-gates.py")
		if err != nil {
			return nil, fmt.Errorf("reading tool gates hook: %w", err)
		}
		files = append(files, types.GeneratedFile{
			Path:     ".claude/hooks/tool-gates.py",
			Content:  content,
			Mode:     0o755,
			Strategy: types.Overwrite,
			Owner:    "tool-gates",
		})
	}

	if answers.Hooks.SOC2Audit {
		content, err := templateFS.ReadFile("templates/hooks/soc2-audit-log.py")
		if err != nil {
			return nil, fmt.Errorf("reading SOC 2 audit hook: %w", err)
		}
		files = append(files, types.GeneratedFile{
			Path:     ".claude/hooks/soc2-audit-log.py",
			Content:  content,
			Mode:     0o755,
			Strategy: types.Overwrite,
			Owner:    "soc2-audit",
		})
	}

	if answers.Hooks.AuditLog && !answers.Hooks.SOC2Audit {
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
