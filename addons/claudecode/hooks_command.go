package claudecode

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// projectDirPrefix is how hook commands reference scripts inside the project.
const projectDirPrefix = `"${CLAUDE_PROJECT_DIR}"/`

// Deployment states reported for a hook.
const (
	deployStatusDeployed      = "deployed"
	deployStatusNotDeployed   = "not deployed"
	deployStatusScriptMissing = "script missing"
	deployStatusScriptChanged = "script modified"
)

// HookStatus describes a single hook's configuration and deployment state for
// display.
type HookStatus struct {
	Name    string `json:"name"`
	Event   string `json:"event"`
	Matcher string `json:"matcher"`
	// Configured reports whether the saved answers enable the hook.
	Configured bool `json:"configured"`
	// Policy is "none" when the hook is configured but has no policy to
	// enforce (it restricts nothing), and empty otherwise.
	Policy string `json:"policy,omitempty"`
	// Deployment reports what is actually on disk: whether .claude/settings.json
	// wires the hook and, for script hooks, whether the script is intact.
	Deployment string `json:"deployment"`
}

// policyNone is the HookStatus.Policy of a configured hook without a policy.
const policyNone = "none"

func hooksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hooks",
		Short: "Manage Claude Code hooks",
	}

	cmd.AddCommand(listHooksCmd())
	return cmd
}

func listHooksCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all registered hooks with configured and deployed status",
		Long: `List every registered hook, whether the saved answers configure it, and
whether it is actually deployed: wired into .claude/settings.json with its
hook script present and unmodified. A hook configured without the policy it
enforces (tool-gates with no .qsdev.yaml hooks.tool_gates lists) is shown as
"yes (no policy)": it runs but restricts nothing.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			projectRoot, err := cmdutil.ProjectRoot()
			if err != nil {
				return err
			}

			answers, err := loadHookAnswers(projectRoot)
			if err != nil {
				return err
			}
			deployed, err := loadDeployedHooks(projectRoot)
			if err != nil {
				return err
			}
			scripts, err := GenerateHookFiles(answers)
			if err != nil {
				return fmt.Errorf("generating expected hook scripts: %w", err)
			}

			registry := defaultHookRegistry()
			statuses := buildHookStatuses(registry, answers)
			applyDeployment(statuses, registry, answers, deployed, scripts, projectRoot)

			if jsonOutput {
				return writeHookStatusesJSON(cmd, statuses)
			}
			writeHookStatusesTable(cmd, statuses)
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	return cmd
}

// loadHookAnswers loads the saved answers, treating a missing answers file as
// empty answers. Any other failure (unreadable or corrupt file) is returned
// rather than silently reported as "nothing configured". The addon's LSP
// enforcement override is reconciled exactly as Generate does, so the listing
// agrees with what generation deploys.
func loadHookAnswers(projectRoot string) (types.WizardAnswers, error) {
	var answers types.WizardAnswers
	if _, err := os.Stat(answersPath(projectRoot)); err == nil {
		if answers, err = loadAnswers(projectRoot); err != nil {
			return types.WizardAnswers{}, err
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return types.WizardAnswers{}, fmt.Errorf("checking saved answers: %w", err)
	}
	if addon.Config.LSPEnforcement != "" {
		answers.LSP.Enforcement = addon.Config.LSPEnforcement
	}
	return answers, nil
}

// loadDeployedHooks reads the hooks map from .claude/settings.json. A missing
// file means nothing is deployed.
func loadDeployedHooks(projectRoot string) (map[string][]HookMatcher, error) {
	data, err := os.ReadFile(filepath.Join(projectRoot, ".claude", "settings.json"))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading .claude/settings.json: %w", err)
	}
	var settings struct {
		Hooks map[string][]HookMatcher `json:"hooks"`
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("parsing .claude/settings.json: %w", err)
	}
	return settings.Hooks, nil
}

func buildHookStatuses(registry *HookRegistry, answers types.WizardAnswers) []HookStatus {
	var statuses []HookStatus
	for _, h := range registry.Definitions() {
		configured := h.EnabledFunc == nil || h.EnabledFunc(answers)
		var policy string
		if h.lacksPolicy(answers) {
			policy = policyNone
		}
		statuses = append(statuses, HookStatus{
			Name:       h.Owner,
			Event:      h.Event,
			Matcher:    h.Matcher,
			Configured: configured,
			Policy:     policy,
			Deployment: deployStatusNotDeployed,
		})
	}
	return statuses
}

// applyDeployment fills in each status's Deployment from the deployed
// settings.json hooks and the on-disk hook scripts, compared against the
// scripts generation would write.
func applyDeployment(statuses []HookStatus, registry *HookRegistry, answers types.WizardAnswers, deployed map[string][]HookMatcher, scripts []types.GeneratedFile, projectRoot string) {
	expected := make(map[string][]byte, len(scripts))
	for _, f := range scripts {
		expected[filepath.ToSlash(f.Path)] = f.Content
	}
	for i, def := range registry.Definitions() {
		if !hookWired(def, def.commandFor(answers), deployed[def.Event]) {
			statuses[i].Deployment = deployStatusNotDeployed
			continue
		}
		statuses[i].Deployment = scriptState(def.Command, expected, projectRoot)
	}
}

// hookWired reports whether def's emitted command (command, as generation
// writes it for the current answers) is wired under its matcher, either
// directly or exactly as the sandbox wraps it. Any other wrapper may not run
// the hook as generated, so it does not count as deployed.
func hookWired(def HookDefinition, command string, matchers []HookMatcher) bool {
	sandboxed := sandboxHookCommand(branding.Get().AppName, def.SandboxCategory, command)
	for _, m := range matchers {
		if m.Matcher != def.Matcher {
			continue
		}
		for _, h := range m.Hooks {
			if h.Command == command || h.Command == sandboxed {
				return true
			}
		}
	}
	return false
}

// scriptState checks the project script a hook command runs, if any. A script
// is intact when it matches the generated content; commands that run a binary
// rather than a project script are deployed once wired.
func scriptState(command string, expected map[string][]byte, projectRoot string) string {
	rest, ok := strings.CutPrefix(command, projectDirPrefix)
	if !ok {
		return deployStatusDeployed
	}
	rel, _, _ := strings.Cut(rest, " ")
	onDisk, err := os.ReadFile(filepath.Join(projectRoot, filepath.FromSlash(rel)))
	if err != nil {
		return deployStatusScriptMissing
	}
	if want, ok := expected[rel]; ok && !bytes.Equal(onDisk, want) {
		return deployStatusScriptChanged
	}
	return deployStatusDeployed
}

func writeHookStatusesJSON(cmd *cobra.Command, statuses []HookStatus) error {
	data, err := json.MarshalIndent(statuses, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling hook statuses: %w", err)
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return nil
}

func writeHookStatusesTable(cmd *cobra.Command, statuses []HookStatus) {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "Hook\tEvent\tMatcher\tConfigured\tDeployment")
	_, _ = fmt.Fprintln(w, "----\t-----\t-------\t----------\t----------")
	for _, s := range statuses {
		configured := "no"
		if s.Configured {
			configured = "yes"
		}
		if s.Policy == policyNone {
			configured += " (no policy)"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", s.Name, s.Event, s.Matcher, configured, s.Deployment)
	}
	_ = w.Flush()
}
