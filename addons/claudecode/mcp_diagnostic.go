package claudecode

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	"github.com/Quantum-Serendipity/qsdev/internal/mcphealth"
)

func mcpStatusCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show health status of configured MCP servers",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectRoot, err := cmdutil.ProjectRoot()
			if err != nil {
				return err
			}

			servers, err := loadMCPServers(projectRoot)
			if err != nil {
				return err
			}

			if len(servers) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No MCP servers configured.")
				return nil
			}

			report := mcphealth.CheckAll(servers, 10*time.Second)

			if jsonOutput {
				data, err := json.MarshalIndent(report, "", "  ")
				if err != nil {
					return fmt.Errorf("marshaling report: %w", err)
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(data))
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "MCP Server Status (%d servers)\n", report.TotalCount)
			fmt.Fprintln(cmd.OutOrStdout(), "----------------------------------------")
			for _, s := range report.Servers {
				status := s.Status
				fmt.Fprintf(cmd.OutOrStdout(), "  %-20s  %-14s  tools: %d  %dms\n",
					s.Name, status, s.ToolCount, s.ResponseMs)
				if s.Error != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "    error: %s\n", s.Error)
				}
				for _, p := range s.Prerequisites {
					if !p.Met {
						fmt.Fprintf(cmd.OutOrStdout(), "    prerequisite: %s (%s) — %s\n", p.Name, p.Type, p.Detail)
					}
				}
			}
			fmt.Fprintf(cmd.OutOrStdout(), "\n%d/%d healthy\n", report.HealthyCount, report.TotalCount)

			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")

	return cmd
}

func mcpListCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List configured MCP servers without health-checking",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectRoot, err := cmdutil.ProjectRoot()
			if err != nil {
				return err
			}

			servers, err := loadMCPServers(projectRoot)
			if err != nil {
				return err
			}

			if len(servers) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No MCP servers configured.")
				return nil
			}

			if jsonOutput {
				data, err := json.MarshalIndent(servers, "", "  ")
				if err != nil {
					return fmt.Errorf("marshaling servers: %w", err)
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(data))
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Configured MCP Servers (%d)\n", len(servers))
			fmt.Fprintln(cmd.OutOrStdout(), "----------------------------------------")
			for name, cfg := range servers {
				fmt.Fprintf(cmd.OutOrStdout(), "  %-20s  %s %v\n", name, cfg.Command, cfg.Args)
				if len(cfg.RequiredEnv) > 0 {
					fmt.Fprintf(cmd.OutOrStdout(), "    required env: %v\n", cfg.RequiredEnv)
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")

	return cmd
}

func loadMCPServers(projectRoot string) (map[string]mcphealth.ServerConfig, error) {
	mcpPath := filepath.Join(projectRoot, ".mcp.json")
	data, err := os.ReadFile(mcpPath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading .mcp.json: %w", err)
	}

	var mcp McpJSON
	if err := json.Unmarshal(data, &mcp); err != nil {
		return nil, fmt.Errorf("parsing .mcp.json: %w", err)
	}

	servers := make(map[string]mcphealth.ServerConfig, len(mcp.MCPServers))
	for name, entry := range mcp.MCPServers {
		cfg := mcphealth.ServerConfig{
			Name:    name,
			Command: entry.Command,
			Args:    entry.Args,
			Env:     entry.Env,
		}
		if known, ok := knownMCPServers[name]; ok {
			cfg.RequiredEnv = known.RequiredEnv
		}
		servers[name] = cfg
	}

	return servers, nil
}
