package claudecode

import (
	"context"

	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func (a *Adapter) SupportedTransports() []aiframework.MCPTransport {
	return []aiframework.MCPTransport{aiframework.TransportStdio}
}

func (a *Adapter) ToolCeiling() int { return 0 }

func (a *Adapter) GenerateMCPConfig(_ context.Context, servers []aiframework.MCPServerSpec, _ map[string]string) ([]types.GeneratedFile, error) {
	answers := types.WizardAnswers{}
	cfg := a.cfg

	for _, s := range servers {
		cfg.MCPServers = append(cfg.MCPServers, MCPServerConfig{
			Name:    s.Name,
			Command: s.Command,
			Args:    s.Args,
			Env:     s.Env,
		})
	}

	gf, err := GenerateMcpJson(answers, cfg)
	if err != nil {
		return nil, err
	}
	if gf == nil {
		return nil, nil
	}
	return []types.GeneratedFile{*gf}, nil
}

func (a *Adapter) FilterServers(servers []aiframework.MCPServerSpec) []aiframework.MCPServerSpec {
	var filtered []aiframework.MCPServerSpec
	for _, s := range servers {
		if s.Transport == aiframework.TransportStdio {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

func (a *Adapter) ValidateServers(_ context.Context, _ []aiframework.MCPServerSpec) []aiframework.ValidationIssue {
	return nil
}
