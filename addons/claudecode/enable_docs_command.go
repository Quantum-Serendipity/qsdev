package claudecode

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpregistry"
)

// docsEnableCmd creates the "enable" subcommand under "qsdev docs".
// It lists the available documentation MCP servers and prints an advisory
// to run "qsdev docs download" afterward.
func docsEnableCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enable",
		Short: "Enable local documentation MCP servers",
		Long: `Enable local documentation MCP servers in the project configuration.

This adds documentation servers (DevDocs, ZIM, man pages, NixOS) to the
project's MCP server list. After enabling, run 'qsdev docs download' to
fetch the documentation data.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			registry := mcpregistry.DefaultRegistry()
			docServers := registry.ListByCategory(mcpregistry.CategoryDocumentation)

			if len(docServers) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No documentation servers available in the registry.")
				return nil
			}

			names := make([]string, 0, len(docServers))
			for _, s := range docServers {
				names = append(names, s.Name)
			}
			sort.Strings(names)

			fmt.Fprintln(cmd.OutOrStdout(), "Enabling documentation MCP servers:")
			fmt.Fprintln(cmd.OutOrStdout(), "----------------------------------------")
			for _, s := range docServers {
				display := s.DisplayName
				if display == "" {
					display = s.Name
				}
				fmt.Fprintf(cmd.OutOrStdout(), "  %-25s %s\n", s.Name, display)
			}

			fmt.Fprintln(cmd.OutOrStdout())
			fmt.Fprintln(cmd.OutOrStdout(), "Documentation servers will be added to .mcp.json on next 'qsdev init'.")
			fmt.Fprintln(cmd.OutOrStdout(), "After enabling, run 'qsdev docs download' to fetch documentation data.")

			return nil
		},
	}

	return cmd
}

// docsDisableCmd creates the "disable" subcommand under "qsdev docs".
// It lists the documentation servers that would be removed and prints
// an advisory about preserving downloaded data.
func docsDisableCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disable",
		Short: "Disable local documentation MCP servers",
		Long: `Disable local documentation MCP servers in the project configuration.

This removes documentation servers from the project's MCP server list.
Downloaded documentation data is preserved; use 'qsdev docs clean' to
remove it.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			registry := mcpregistry.DefaultRegistry()
			docServers := registry.ListByCategory(mcpregistry.CategoryDocumentation)

			if len(docServers) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No documentation servers to disable.")
				return nil
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Disabling documentation MCP servers:")
			fmt.Fprintln(cmd.OutOrStdout(), "----------------------------------------")
			for _, s := range docServers {
				display := s.DisplayName
				if display == "" {
					display = s.Name
				}
				fmt.Fprintf(cmd.OutOrStdout(), "  %-25s %s\n", s.Name, display)
			}

			fmt.Fprintln(cmd.OutOrStdout())
			fmt.Fprintln(cmd.OutOrStdout(), "Documentation servers will be removed from .mcp.json on next 'qsdev init'.")
			fmt.Fprintln(cmd.OutOrStdout(), "Downloaded documentation data is preserved. Run 'qsdev docs clean --all' to remove it.")

			return nil
		},
	}

	return cmd
}
