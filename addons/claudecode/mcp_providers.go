package claudecode

import (
	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserver"
	"github.com/Quantum-Serendipity/qsdev/internal/postmortem"
	"github.com/Quantum-Serendipity/qsdev/internal/vsentinel"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

func newPostmortemProvider() mcpserver.Provider {
	return &postmortem.MCPProvider{
		ChecklistFunc: func() ([]string, error) {
			projectRoot, err := cmdutil.ProjectRoot()
			if err != nil {
				return nil, err
			}
			answers, err := loadAnswers(projectRoot)
			if err != nil {
				return nil, err
			}
			registry := ecosystem.DefaultRegistry()
			return collectVerificationCommands(answers, registry), nil
		},
	}
}

func newVersionSentinelProvider() mcpserver.Provider {
	return &vsentinel.MCPProvider{
		ProjectRootFunc: cmdutil.ProjectRoot,
		ManifestCoverageFunc: func() (any, error) {
			projectRoot, err := cmdutil.ProjectRoot()
			if err != nil {
				return nil, err
			}
			answers, err := loadAnswers(projectRoot)
			if err != nil {
				return nil, err
			}
			registry := ecosystem.DefaultRegistry()
			return collectManifestCoverage(answers, registry), nil
		},
	}
}
