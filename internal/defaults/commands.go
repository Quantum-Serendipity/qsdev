package defaults

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"reflect"
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

	if err := fileutil.WriteFileAtomic(path, content, fileutil.ModeReadWrite); err != nil {
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
		Short: "Show effective defaults (embedded + project + user overrides)",
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
		Short: "Validate the user and project defaults files",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runValidate(cmd)
		},
	}
}

func runValidate(cmd *cobra.Command) error {
	orgFile := catalog.OrgConfigFile()
	projFile := catalog.ProjectConfigFile(catalog.ProjectRoot())
	if orgFile == "" && projFile == "" {
		path := catalog.OrgConfigPath()
		fmt.Fprintf(cmd.OutOrStdout(), "No defaults file found at %s. Using embedded defaults.\n", path)
		return nil
	}

	_, err := loadFresh()
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Validation errors: %v\n", err)
		return fmt.Errorf("defaults file is invalid")
	}

	if projFile != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Project defaults file %s is valid.\n", projFile)
	}
	if orgFile != "" {
		fmt.Fprintln(cmd.OutOrStdout(), "Defaults file is valid.")
	}
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
		if err := fileutil.WriteFileAtomic(path, content, fileutil.ModeReadWrite); err != nil {
			return fmt.Errorf("writing defaults file: %w", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Created %s\n", path)
	}

	editorBin, editorArgs := editorCommand(os.Getenv("EDITOR"), path)
	if _, err := exec.LookPath(editorBin); err != nil {
		return fmt.Errorf("editor %q not found in PATH: %w", editorBin, err)
	}

	editorCmd := exec.Command(editorBin, editorArgs...)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	if err := editorCmd.Run(); err != nil {
		return fmt.Errorf("running editor: %w", err)
	}

	return runValidate(cmd)
}

// editorCommand splits an $EDITOR value into the editor binary and its
// arguments, with path appended. An unset, empty, or whitespace-only value
// falls back to vi.
func editorCommand(editorEnv, path string) (string, []string) {
	parts := strings.Fields(editorEnv)
	if len(parts) == 0 {
		parts = []string{"vi"}
	}
	args := append(append([]string{}, parts[1:]...), path)
	return parts[0], args
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

// loadFresh loads the catalog, with the same project and user defaults files
// as Default, without using the cached Default() singleton.
func loadFresh() (*catalog.Catalog, error) {
	var opts []catalog.LoadOption

	if projFile := catalog.ProjectConfigFile(catalog.ProjectRoot()); projFile != "" {
		opts = append(opts, catalog.WithProjectConfigFile(projFile))
	}

	if orgFile := catalog.OrgConfigFile(); orgFile != "" {
		opts = append(opts, catalog.WithOrgConfigFile(orgFile))
	}

	return catalog.Load(opts...)
}

// sectionFromUnified extracts a named section from UnifiedDefaults. The
// section is selected by the field's yaml tag, so every section listed by
// catalog.SectionNames resolves without a hand-maintained switch.
func sectionFromUnified(u *catalog.UnifiedDefaults, name string) (any, error) {
	key := strings.ToLower(name)
	if key != "" {
		v := reflect.ValueOf(u).Elem()
		t := v.Type()
		for i := range t.NumField() {
			tag, _, _ := strings.Cut(t.Field(i).Tag.Get("yaml"), ",")
			if tag == key {
				return v.Field(i).Interface(), nil
			}
		}
	}
	return nil, fmt.Errorf("unknown section %q; valid sections: %s",
		name, strings.Join(catalog.SectionNames(), ", "))
}
