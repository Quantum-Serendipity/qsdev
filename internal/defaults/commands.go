package defaults

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

// Command returns the top-level "defaults" command with all subcommands.
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "defaults",
		Short: "Manage user-level default configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runShow(cmd, "", false)
		},
	}

	cmd.AddCommand(
		initCmd(),
		showCmd(),
		validateCmd(),
		editCmd(),
		pathCmd(),
		resetCmd(),
	)

	return cmd
}

func initCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create a defaults template file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runInit(cmd, force)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing defaults file")

	return cmd
}

func runInit(cmd *cobra.Command, force bool) error {
	path := catalog.OrgConfigPath()

	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("defaults file already exists at %s (use --force to overwrite)", path)
		}
	}

	content, err := catalog.GenerateDefaultsTemplate()
	if err != nil {
		return fmt.Errorf("generating defaults template: %w", err)
	}

	if err := fileutil.WriteFileAtomic(path, content, 0o644); err != nil {
		return fmt.Errorf("writing defaults file: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Created %s\n", path)
	return nil
}

func showCmd() *cobra.Command {
	var (
		section  string
		jsonFlag bool
	)

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show effective defaults (embedded + user overrides)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runShow(cmd, section, jsonFlag)
		},
	}

	cmd.Flags().StringVar(&section, "section", "", "Show only this section (e.g. tiers, tools, compliance)")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON instead of YAML")

	return cmd
}

func runShow(cmd *cobra.Command, section string, jsonFlag bool) error {
	cat, err := loadFresh()
	if err != nil {
		return fmt.Errorf("loading catalog: %w", err)
	}

	unified := cat.ToUnified()

	var target any = unified
	if section != "" {
		s, err := sectionFromUnified(unified, section)
		if err != nil {
			return err
		}
		target = s
	}

	var out []byte
	if jsonFlag {
		out, err = json.MarshalIndent(target, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling to JSON: %w", err)
		}
		out = append(out, '\n')
	} else {
		out, err = yaml.Marshal(target)
		if err != nil {
			return fmt.Errorf("marshaling to YAML: %w", err)
		}
	}

	fmt.Fprint(cmd.OutOrStdout(), string(out))
	return nil
}

func validateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate the user defaults file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runValidate(cmd)
		},
	}
}

func runValidate(cmd *cobra.Command) error {
	orgFile := catalog.OrgConfigFile()
	if orgFile == "" {
		path := catalog.OrgConfigPath()
		fmt.Fprintf(cmd.OutOrStdout(), "No defaults file found at %s. Using embedded defaults.\n", path)
		return nil
	}

	_, err := loadFresh()
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Validation errors: %v\n", err)
		return fmt.Errorf("defaults file is invalid")
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Defaults file is valid.")
	return nil
}

func editCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit",
		Short: "Open the defaults file in $EDITOR",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runEdit(cmd)
		},
	}
}

func runEdit(cmd *cobra.Command) error {
	path := catalog.OrgConfigPath()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		content, err := catalog.GenerateDefaultsTemplate()
		if err != nil {
			return fmt.Errorf("generating defaults template: %w", err)
		}
		if err := fileutil.WriteFileAtomic(path, content, 0o644); err != nil {
			return fmt.Errorf("writing defaults file: %w", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Created %s\n", path)
	}

	editorEnv := os.Getenv("EDITOR")
	if editorEnv == "" {
		editorEnv = "vi"
	}
	editorParts := strings.Fields(editorEnv)
	editorBin := editorParts[0]
	if _, err := exec.LookPath(editorBin); err != nil {
		return fmt.Errorf("editor %q not found in PATH: %w", editorBin, err)
	}
	editorArgs := append(append([]string{}, editorParts[1:]...), path)

	editorCmd := exec.Command(editorBin, editorArgs...)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	if err := editorCmd.Run(); err != nil {
		return fmt.Errorf("running editor: %w", err)
	}

	return runValidate(cmd)
}

func pathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the defaults file path",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runPath(cmd)
		},
	}
}

func runPath(cmd *cobra.Command) error {
	p := catalog.OrgConfigPath()
	if _, err := os.Stat(p); err == nil {
		fmt.Fprintf(cmd.OutOrStdout(), "%s (exists)\n", p)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "%s (not created yet — run 'qsdev defaults init')\n", p)
	}
	return nil
}

func resetCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Remove the user defaults file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runReset(cmd, yes)
		},
	}

	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")

	return cmd
}

func runReset(cmd *cobra.Command, yes bool) error {
	p := catalog.OrgConfigPath()

	if _, err := os.Stat(p); os.IsNotExist(err) {
		fmt.Fprintln(cmd.OutOrStdout(), "No defaults file found.")
		return nil
	}

	if !yes {
		fmt.Fprintf(cmd.OutOrStdout(), "Remove %s? [y/N] ", p)
		reader := bufio.NewReader(os.Stdin)
		answer, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("reading confirmation: %w", err)
		}
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
			return nil
		}
	}

	if err := os.Remove(p); err != nil {
		return fmt.Errorf("removing defaults file: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Defaults reset to built-in values.")
	return nil
}

// loadFresh loads the catalog without using the cached Default() singleton.
func loadFresh() (*catalog.Catalog, error) {
	var opts []catalog.LoadOption

	if orgFile := catalog.OrgConfigFile(); orgFile != "" {
		opts = append(opts, catalog.WithOrgConfigFile(orgFile))
	}

	return catalog.Load(opts...)
}

// sectionFromUnified extracts a named section from UnifiedDefaults.
func sectionFromUnified(u *catalog.UnifiedDefaults, name string) (any, error) {
	switch strings.ToLower(name) {
	case "tiers":
		return u.Tiers, nil
	case "compliance":
		return u.Compliance, nil
	case "profiles":
		return u.Profiles, nil
	case "profile_aliases":
		return u.ProfileAliases, nil
	case "project_profiles":
		return u.ProjectProfiles, nil
	case "tools":
		return u.Tools, nil
	case "security_hooks":
		return u.SecurityHooks, nil
	case "base_packages":
		return u.BasePackages, nil
	case "unset_vars":
		return u.UnsetVars, nil
	case "keep_vars":
		return u.KeepVars, nil
	case "custom_hooks":
		return u.CustomHooks, nil
	case "hook_tier_order":
		return u.HookTierOrder, nil
	case "hook_tiers":
		return u.HookTiers, nil
	case "tier_to_compliance":
		return u.TierToCompliance, nil
	case "tier_to_enabled_tools":
		return u.TierToEnabledTools, nil
	case "default_mcp_servers":
		return u.DefaultMCPServers, nil
	case "default_agent_tools":
		return u.DefaultAgentTools, nil
	case "languages":
		return u.Languages, nil
	case "services":
		return u.Services, nil
	case "permission_presets":
		return u.PermissionPresets, nil
	case "hook_presets":
		return u.HookPresets, nil
	case "security_levels":
		return u.SecurityLevels, nil
	case "data_classifications":
		return u.DataClassifications, nil
	case "package_managers":
		return u.PackageManagers, nil
	case "tool_categories":
		return u.ToolCategories, nil
	default:
		return nil, fmt.Errorf("unknown section %q; valid sections: %s",
			name, strings.Join(sectionNames(), ", "))
	}
}

// sectionNames returns all valid section names for sectionFromUnified.
func sectionNames() []string {
	return []string{
		"tiers",
		"compliance",
		"profiles",
		"profile_aliases",
		"project_profiles",
		"tools",
		"security_hooks",
		"base_packages",
		"unset_vars",
		"keep_vars",
		"custom_hooks",
		"hook_tier_order",
		"hook_tiers",
		"tier_to_compliance",
		"tier_to_enabled_tools",
		"default_mcp_servers",
		"default_agent_tools",
		"languages",
		"services",
		"permission_presets",
		"hook_presets",
		"security_levels",
		"data_classifications",
		"package_managers",
		"tool_categories",
	}
}
