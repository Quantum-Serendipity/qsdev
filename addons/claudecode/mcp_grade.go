package claudecode

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpregistry"
)

// gradeJSONEntry is the JSON representation of a single server's compliance grade.
type gradeJSONEntry struct {
	Name     string               `json:"name"`
	Grade    string               `json:"grade"`
	Criteria []gradeCriterionJSON `json:"criteria"`
}

// gradeCriterionJSON is the JSON representation of a single compliance criterion.
type gradeCriterionJSON struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
}

func mcpGradeCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "grade [server-name]",
		Short: "Show compliance grade for MCP servers",
		Long: `Evaluate MCP servers against the compliance ladder and show per-criterion results.

The compliance levels from lowest to highest are: basic, standard, secure,
verified, attested. Each level requires all criteria from previous levels
plus its own criteria to be satisfied.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectRoot, err := cmdutil.ProjectRoot()
			if err != nil {
				return err
			}

			// Scan .mcp.json for configured servers.
			configured, err := mcpregistry.ScanMcpJSON(projectRoot)
			if err != nil {
				return err
			}

			// Get known servers from the registry.
			registry := mcpregistry.DefaultRegistry()
			knownDefs := registry.List()

			// Build merged map: registry definitions enriched with config data.
			merged := make(map[string]*mcpregistry.McpServerDefinition)
			for _, def := range knownDefs {
				defCopy := *def
				merged[def.Name] = &defCopy
			}
			for name, cfgDef := range configured {
				if _, exists := merged[name]; !exists {
					defCopy := cfgDef
					merged[name] = &defCopy
				}
			}

			// Filter to a single server if requested.
			if len(args) == 1 {
				serverName := args[0]
				def, ok := merged[serverName]
				if !ok {
					return fmt.Errorf("server %q not found in registry or .mcp.json", serverName)
				}
				filtered := map[string]*mcpregistry.McpServerDefinition{serverName: def}
				merged = filtered
			}

			if len(merged) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No MCP servers found.")
				return nil
			}

			// Grade all servers and collect results.
			type namedGrade struct {
				name  string
				grade mcpregistry.GradeResult
			}
			var results []namedGrade
			for name, def := range merged {
				gr := mcpregistry.GradeServer(def)
				results = append(results, namedGrade{name: name, grade: gr})
			}
			sort.Slice(results, func(i, j int) bool {
				return results[i].name < results[j].name
			})

			if jsonOutput {
				var entries []gradeJSONEntry
				for _, r := range results {
					entry := gradeJSONEntry{
						Name:  r.name,
						Grade: r.grade.Level.String(),
					}
					for _, c := range r.grade.Criteria {
						entry.Criteria = append(entry.Criteria, gradeCriterionJSON{
							Name:   c.Name,
							Passed: c.Passed,
							Detail: c.Detail,
						})
					}
					entries = append(entries, entry)
				}
				data, err := json.MarshalIndent(entries, "", "  ")
				if err != nil {
					return fmt.Errorf("marshaling grade results: %w", err)
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(data))
				return nil
			}

			fmt.Fprintln(cmd.OutOrStdout(), "MCP Server Compliance Grades")
			fmt.Fprintln(cmd.OutOrStdout(), "----------------------------------------")
			for _, r := range results {
				fmt.Fprintf(cmd.OutOrStdout(), "  %-25s %s\n", r.name, r.grade.Level)
				for _, c := range r.grade.Criteria {
					tag := "[PASS]"
					if !c.Passed {
						tag = "[FAIL]"
					}
					fmt.Fprintf(cmd.OutOrStdout(), "    %s %s: %s\n", tag, c.Name, c.Detail)
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")

	return cmd
}
