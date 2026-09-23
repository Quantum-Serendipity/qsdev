package claudecode

import (
	"errors"
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpregistry"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// docsEnableCmd creates the "enable" subcommand under "qsdev docs". It enables
// the catalog tools that provide the documentation MCP servers through the
// regular tool lifecycle, exactly as `qsdev enable <tool>` does.
func docsEnableCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enable [server...]",
		Short: "Enable local documentation MCP servers",
		Long: fmt.Sprintf(`Enable local documentation MCP servers in the project configuration.

This enables the tools that provide the documentation servers (DevDocs, ZIM,
man pages, NixOS), or only the named servers, through the same lifecycle as
'%[1]s enable <tool>': .mcp.json and CLAUDE.md are updated and the tools are
recorded in the saved answers. After enabling, run '%[1]s docs download' to
fetch the documentation data.`, branding.Get().AppName),
		RunE: func(cmd *cobra.Command, args []string) error {
			tools, err := docServerTools(args)
			if err != nil {
				return err
			}
			if len(tools) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No documentation servers available in the registry.")
				return nil
			}
			if err := runToolLifecycle(cmd, "enable", tools); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "\nRun '%s docs download' to fetch documentation data.\n", branding.Get().AppName)
			return nil
		},
	}

	return cmd
}

// docsDisableCmd creates the "disable" subcommand under "qsdev docs". It
// disables the documentation server tools through the regular tool lifecycle,
// exactly as `qsdev disable <tool>` does.
func docsDisableCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disable [server...]",
		Short: "Disable local documentation MCP servers",
		Long: fmt.Sprintf(`Disable local documentation MCP servers in the project configuration.

This disables the tools that provide the documentation servers, or only the
named servers, through the same lifecycle as '%[1]s disable <tool>', removing
them from .mcp.json and CLAUDE.md. Downloaded documentation data is preserved;
use '%[1]s docs clean' to remove it.`, branding.Get().AppName),
		RunE: func(cmd *cobra.Command, args []string) error {
			tools, err := docServerTools(args)
			if err != nil {
				return err
			}
			if len(tools) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No documentation servers to disable.")
				return nil
			}
			if err := runToolLifecycle(cmd, "disable", tools); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "\nDownloaded documentation data is preserved. Run '%s docs clean --all' to remove it.\n", branding.Get().AppName)
			return nil
		},
	}

	return cmd
}

// docServerTools returns the catalog tools that provide the registry's
// documentation MCP servers, sorted by name. When servers is non-empty, only
// the tools for those servers are returned, and an unknown name is an error.
func docServerTools(servers []string) ([]string, error) {
	toolByServer := make(map[string]string)
	for _, s := range mcpregistry.DefaultRegistry().ByCategory(mcpregistry.CategoryDocumentation) {
		if s.ToolRegName != "" {
			toolByServer[s.Name] = s.ToolRegName
		}
	}

	var tools []string
	if len(servers) == 0 {
		for _, tool := range toolByServer {
			tools = append(tools, tool)
		}
	} else {
		for _, name := range servers {
			tool, ok := toolByServer[name]
			if !ok {
				return nil, fmt.Errorf("unknown documentation server %q", name)
			}
			tools = append(tools, tool)
		}
	}
	sort.Strings(tools)
	return tools, nil
}

// runToolLifecycle runs the top-level tool lifecycle command named verb
// ("enable" or "disable", provided by the init addon) for each tool, so the
// change is persisted and generated exactly as `qsdev <verb> <tool>` would.
// Every tool is attempted; the failures are returned together.
func runToolLifecycle(cmd *cobra.Command, verb string, tools []string) error {
	lifecycle, _, err := cmd.Root().Find([]string{verb})
	if err != nil || lifecycle.Name() != verb || lifecycle.RunE == nil {
		return fmt.Errorf("the '%s %s' tool lifecycle command is not available", branding.Get().AppName, verb)
	}
	var errs []error
	for _, tool := range tools {
		if err := lifecycle.RunE(lifecycle, []string{tool}); err != nil {
			errs = append(errs, fmt.Errorf("%s %s: %w", verb, tool, err))
		}
	}
	return errors.Join(errs...)
}
