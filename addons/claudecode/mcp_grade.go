package claudecode

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

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

// criterionMatchesRegistry names the extra criterion reported for every
// configured server: whether its entry is a definition qsdev vouches for. The
// name predates the trust set and is kept for JSON consumers.
const criterionMatchesRegistry = "matches-registry-definition"

// gradeTarget is one server to grade. def is the definition that is graded:
// the .mcp.json entry when the project configures the server, otherwise the
// registry definition (graded only on request and reported as not configured).
type gradeTarget struct {
	name       string
	def        *mcpregistry.McpServerDefinition
	configured bool
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
environment actually configured there. For every configured server, the extra
matches-registry-definition criterion reports whether the entry is exactly a
definition qsdev trusts (a catalog launcher, its hardened form with any valid
release cutoff, or its installed binary, an organization overlay server, or a
server built into qsdev), comparing the command, arguments, URL, headers and
environment, and otherwise names the first field that differs. It is the same
check mcp status uses before probing, and it does not change the grade level.
Registry servers the project does not configure are graded only with --all or
when named explicitly, and are marked as not configured.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pc, err := cmdutil.Project(cmd)
			if err != nil {
				return err
			}
			projectRoot := pc.Root

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

			printCodeKeyringMigrationHint(cmd.ErrOrStderr())
			trusted := mcpregistry.TrustedDefinitions(configuredServerSpecs())
			results := make([]mcpregistry.GradeResult, len(targets))
			for i, t := range targets {
				results[i] = gradeTargetResult(cmd.Context(), t, trusted[t.name])
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

	return cmdutil.MarkReadOnly(cmd, "")
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
		defCopy := cfgDef
		targets = append(targets, gradeTarget{name: name, def: &defCopy, configured: true})
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

// gradeTargetResult grades the target's definition (a configured one through
// mcpregistry.GradeConfigured, the grade `qsdev status` reports too) and, for
// a configured server, appends whether its entry is one of trusted, the
// definitions qsdev trusts for its name (TrustedDefinitions). The appended
// criterion is informational: the level is already set. ctx bounds the
// attestation check.
func gradeTargetResult(ctx context.Context, t gradeTarget, trusted []mcpregistry.LaunchSpec) mcpregistry.GradeResult {
	if !t.configured {
		return mcpregistry.GradeServer(ctx, t.def)
	}
	gr := mcpregistry.GradeConfigured(ctx, *t.def)
	gr.Criteria = append(gr.Criteria, registryMatchCriterion(t.def, trusted))
	return gr
}

// registryMatchCriterion reports whether a configured server is exactly one of
// the trusted definitions of its name (mcpregistry.TrustMismatch, the check the
// probe gate applies), naming the first differing field when it is not.
func registryMatchCriterion(configured *mcpregistry.McpServerDefinition, trusted []mcpregistry.LaunchSpec) mcpregistry.CriterionResult {
	if mismatch := mcpregistry.TrustMismatch(configured.ServerConfig(), trusted); mismatch != "" {
		return mcpregistry.CriterionResult{Name: criterionMatchesRegistry, Detail: mismatch}
	}
	return mcpregistry.CriterionResult{
		Name:   criterionMatchesRegistry,
		Passed: true,
		Detail: "configured entry matches a trusted definition",
	}
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
