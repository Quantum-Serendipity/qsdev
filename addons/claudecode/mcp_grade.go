package claudecode

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpregistry"
)

// gradeJSONEntry is the JSON representation of a single server's compliance grade.
type gradeJSONEntry struct {
	Name       string               `json:"name"`
	Grade      string               `json:"grade"`
	Configured bool                 `json:"configured"`
	Criteria   []gradeCriterionJSON `json:"criteria"`
}

// gradeCriterionJSON is the JSON representation of a single compliance criterion.
type gradeCriterionJSON struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
}

// criterionMatchesRegistry names the extra criterion reported for a configured
// server that shares its name with a built-in registry definition.
const criterionMatchesRegistry = "matches-registry-definition"

// gradeTarget is one server to grade. def is the definition that is graded:
// the .mcp.json entry when the project configures the server, otherwise the
// registry definition (graded only on request and reported as not configured).
type gradeTarget struct {
	name       string
	def        *mcpregistry.McpServerDefinition
	configured bool
	// registry is the same-named registry definition of a configured server,
	// or nil when the name is not in the registry.
	registry *mcpregistry.McpServerDefinition
}

func mcpGradeCmd() *cobra.Command {
	var (
		jsonOutput bool
		all        bool
	)

	cmd := &cobra.Command{
		Use:   "grade [server-name]",
		Short: "Show compliance grade for MCP servers",
		Long: `Evaluate MCP servers against the compliance ladder and show per-criterion results.

The compliance levels from lowest to highest are: basic, standard, secure,
verified, attested. Each level requires all criteria from previous levels
plus its own criteria to be satisfied.

Servers configured in .mcp.json are graded from the command, arguments and
environment actually configured there. When a configured server shares its
name with a built-in registry server, the extra matches-registry-definition
criterion reports whether the configured command still matches the registry.
Registry servers the project does not configure are graded only with --all or
when named explicitly, and are marked as not configured.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectRoot, err := cmdutil.ProjectRoot()
			if err != nil {
				return err
			}

			configured, err := mcpregistry.ScanMcpJSON(projectRoot)
			if err != nil {
				return err
			}

			only := ""
			if len(args) == 1 {
				only = args[0]
			}
			targets, err := selectGradeTargets(configured, mcpregistry.DefaultRegistry().All(), only, all)
			if err != nil {
				return err
			}

			if len(targets) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No MCP servers configured in .mcp.json. Use --all to grade the built-in registry servers.")
				return nil
			}

			results := make([]mcpregistry.GradeResult, len(targets))
			for i, t := range targets {
				results[i] = gradeTargetResult(t)
			}

			if jsonOutput {
				return printGradeJSON(cmd, targets, results)
			}
			printGradeText(cmd, targets, results)
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	cmd.Flags().BoolVar(&all, "all", false, "Also grade built-in registry servers the project does not configure")

	return cmd
}

// selectGradeTargets returns the servers to grade, sorted by name. Every
// server configured in .mcp.json is graded from its configured definition;
// a registry definition never replaces it. Registry servers that are not
// configured are included only when all is set or they are named by only.
// When only is non-empty, just that server is returned.
func selectGradeTargets(configured map[string]mcpregistry.McpServerDefinition, known []*mcpregistry.McpServerDefinition, only string, all bool) ([]gradeTarget, error) {
	registry := make(map[string]*mcpregistry.McpServerDefinition, len(known))
	for _, def := range known {
		registry[def.Name] = def
	}

	var targets []gradeTarget
	for name, cfgDef := range configured {
		if only != "" && name != only {
			continue
		}
		defCopy := configuredGradeDef(cfgDef)
		targets = append(targets, gradeTarget{name: name, def: &defCopy, configured: true, registry: registry[name]})
	}
	for name, def := range registry {
		if _, ok := configured[name]; ok {
			continue
		}
		if name == only || (only == "" && all) {
			defCopy := *def
			targets = append(targets, gradeTarget{name: name, def: &defCopy})
		}
	}

	if only != "" && len(targets) == 0 {
		return nil, fmt.Errorf("server %q not found in registry or .mcp.json", only)
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i].name < targets[j].name })
	return targets, nil
}

// transportUnknown marks a configured entry whose transport could not be
// determined, so it never passes the stdio-transport criterion.
const transportUnknown mcpregistry.McpTransport = "unknown"

// configuredGradeDef returns the .mcp.json definition to grade. An entry with
// no command cannot be a stdio server (it is a remote "type"/"url" entry), so
// it is not graded as a local stdio process: its transport is marked unknown,
// which fails stdio-transport and keeps it at basic.
func configuredGradeDef(def mcpregistry.McpServerDefinition) mcpregistry.McpServerDefinition {
	if def.Transport == mcpregistry.TransportStdio && def.Command == "" {
		def.Transport = transportUnknown
	}
	return def
}

// gradeTargetResult grades the target's definition and, for a configured
// server that shadows a registry server, appends whether the configured launch
// command still matches the registry definition.
func gradeTargetResult(t gradeTarget) mcpregistry.GradeResult {
	gr := mcpregistry.GradeServer(t.def)
	if t.configured && t.registry != nil {
		gr.Criteria = append(gr.Criteria, registryMatchCriterion(t.def, t.registry))
	}
	return gr
}

// registryMatchCriterion reports whether a configured server launches the same
// command, arguments and URL as the registry definition of the same name. A
// mismatch means the name no longer identifies the reviewed registry server.
func registryMatchCriterion(configured, registry *mcpregistry.McpServerDefinition) mcpregistry.CriterionResult {
	if configured.Command == registry.Command && slices.Equal(configured.Args, registry.Args) && configured.URL == registry.URL {
		return mcpregistry.CriterionResult{
			Name:   criterionMatchesRegistry,
			Passed: true,
			Detail: "configured command matches the built-in registry definition",
		}
	}
	return mcpregistry.CriterionResult{
		Name: criterionMatchesRegistry,
		Detail: fmt.Sprintf("configured %q differs from the built-in registry definition %q",
			launchSummary(configured), launchSummary(registry)),
	}
}

// launchSummary renders what a server definition launches: its URL for a
// remote server, otherwise its command line.
func launchSummary(def *mcpregistry.McpServerDefinition) string {
	if def.URL != "" {
		return def.URL
	}
	return strings.Join(append([]string{def.Command}, def.Args...), " ")
}

func printGradeJSON(cmd *cobra.Command, targets []gradeTarget, results []mcpregistry.GradeResult) error {
	entries := make([]gradeJSONEntry, len(targets))
	for i, t := range targets {
		entries[i] = gradeJSONEntry{
			Name:       t.name,
			Grade:      results[i].Level.String(),
			Configured: t.configured,
		}
		for _, c := range results[i].Criteria {
			entries[i].Criteria = append(entries[i].Criteria, gradeCriterionJSON{
				Name:   c.Name,
				Passed: c.Passed,
				Detail: c.Detail,
			})
		}
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling grade results: %w", err)
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return nil
}

func printGradeText(cmd *cobra.Command, targets []gradeTarget, results []mcpregistry.GradeResult) {
	w := cmd.OutOrStdout()
	fmt.Fprintln(w, "MCP Server Compliance Grades")
	fmt.Fprintln(w, "----------------------------------------")
	for i, t := range targets {
		label := ""
		if !t.configured {
			label = "  (not configured)"
		}
		fmt.Fprintf(w, "  %-25s %s%s\n", t.name, results[i].Level, label)
		for _, c := range results[i].Criteria {
			tag := "[PASS]"
			if !c.Passed {
				tag = "[FAIL]"
			}
			fmt.Fprintf(w, "    %s %s: %s\n", tag, c.Name, c.Detail)
		}
	}
}
